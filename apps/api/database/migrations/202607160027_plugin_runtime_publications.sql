-- +goose Up
-- P12: plugin runtime convergence is a full-set publication. PostgreSQL is
-- authoritative; NOTIFY only wakes api/worker processes, which must reload the
-- durable revision after reconnecting or missing a notification.
CREATE TABLE plugin_runtime_publications (
  revision BIGSERIAL PRIMARY KEY,
  member_count INTEGER NOT NULL CHECK (member_count >= 0),
  members_digest TEXT NOT NULL CHECK (members_digest ~ '^[0-9a-f]{64}$'),
  reason TEXT NOT NULL
    CHECK (reason IN (
      'enable', 'disable', 'upgrade', 'rollback', 'uninstall',
      'startup_reconcile', 'recovery'
    )),
  actor_user_id BIGINT CHECK (actor_user_id IS NULL OR actor_user_id > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp()
);

CREATE INDEX plugin_runtime_publications_created_idx
  ON plugin_runtime_publications (created_at DESC, revision DESC);

-- The redundant leading id makes the immutable version tuple a legal exact
-- composite FK target without changing existing version uniqueness semantics.
CREATE UNIQUE INDEX extension_versions_plugin_runtime_identity_idx
  ON extension_versions (id, extension_id, version, package_digest);

CREATE TABLE plugin_runtime_publication_members (
  publication_revision BIGINT NOT NULL
    REFERENCES plugin_runtime_publications(revision) ON DELETE RESTRICT,
  extension_id TEXT NOT NULL CHECK (extension_id <> ''),
  extension_version_id BIGINT NOT NULL CHECK (extension_version_id > 0),
  extension_version TEXT NOT NULL CHECK (extension_version <> ''),
  package_digest TEXT NOT NULL CHECK (package_digest ~ '^[0-9a-f]{64}$'),
  PRIMARY KEY (publication_revision, extension_id),
  UNIQUE (
    publication_revision, extension_id, extension_version_id,
    extension_version, package_digest
  ),
  FOREIGN KEY (
    extension_version_id, extension_id, extension_version, package_digest
  ) REFERENCES extension_versions(id, extension_id, version, package_digest)
    ON DELETE RESTRICT
);

CREATE INDEX plugin_runtime_publication_members_version_idx
  ON plugin_runtime_publication_members (
    extension_id, extension_version_id, package_digest, publication_revision DESC
  );

-- Runtime publications are exclusively for plugin subprocesses. The exact
-- version FK is not enough because extension_versions does not carry type.
-- +goose StatementBegin
CREATE FUNCTION enforce_plugin_runtime_member_type() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  stored_type TEXT;
BEGIN
  SELECT e.type INTO stored_type
  FROM extension_versions AS v
  JOIN extensions AS e ON e.id = v.extension_id
  WHERE v.id = NEW.extension_version_id
    AND v.extension_id = NEW.extension_id
    AND v.version = NEW.extension_version
    AND v.package_digest = NEW.package_digest
  FOR NO KEY UPDATE OF e;

  IF stored_type IS DISTINCT FROM 'plugin' THEN
    RAISE EXCEPTION 'plugin runtime publication member must be a plugin';
  END IF;
  RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER plugin_runtime_publication_member_type
BEFORE INSERT ON plugin_runtime_publication_members
FOR EACH ROW EXECUTE FUNCTION enforce_plugin_runtime_member_type();

-- A historical publication keeps its plugin classification. The exact version
-- FK already prevents deleting the extension through its version cascade.
-- +goose StatementBegin
CREATE FUNCTION reject_published_plugin_type_change() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF NEW.type IS DISTINCT FROM OLD.type AND EXISTS (
    SELECT 1 FROM plugin_runtime_publication_members
    WHERE extension_id = OLD.id
  ) THEN
    RAISE EXCEPTION 'published plugin runtime extension type is immutable';
  END IF;
  RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER plugin_runtime_extension_type_immutable
BEFORE UPDATE OF type ON extensions
FOR EACH ROW EXECUTE FUNCTION reject_published_plugin_type_change();

-- Desired revisions and their exact members are append-only. A rollback or
-- repair publishes a new complete set instead of editing historical intent.
-- +goose StatementBegin
CREATE FUNCTION reject_plugin_runtime_desired_mutation() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  RAISE EXCEPTION 'plugin runtime desired publications are append-only';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER plugin_runtime_publication_immutable
BEFORE UPDATE OR DELETE ON plugin_runtime_publications
FOR EACH ROW EXECUTE FUNCTION reject_plugin_runtime_desired_mutation();

CREATE TRIGGER plugin_runtime_publication_no_truncate
BEFORE TRUNCATE ON plugin_runtime_publications
FOR EACH STATEMENT EXECUTE FUNCTION reject_plugin_runtime_desired_mutation();

CREATE TRIGGER plugin_runtime_publication_member_immutable
BEFORE UPDATE OR DELETE ON plugin_runtime_publication_members
FOR EACH ROW EXECUTE FUNCTION reject_plugin_runtime_desired_mutation();

CREATE TRIGGER plugin_runtime_publication_member_no_truncate
BEFORE TRUNCATE ON plugin_runtime_publication_members
FOR EACH STATEMENT EXECUTE FUNCTION reject_plugin_runtime_desired_mutation();

-- A publication and all of its members must be inserted in one transaction.
-- The deferred trigger also prevents appending members to a committed set.
-- +goose StatementBegin
CREATE FUNCTION validate_plugin_runtime_desired_full_set() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  target_revision BIGINT;
  expected_count INTEGER;
  expected_digest TEXT;
  actual_count BIGINT;
  actual_digest TEXT;
BEGIN
  IF TG_TABLE_NAME = 'plugin_runtime_publications' THEN
    target_revision := NEW.revision;
  ELSE
    target_revision := NEW.publication_revision;
  END IF;

  SELECT member_count, members_digest INTO expected_count, expected_digest
  FROM plugin_runtime_publications
  WHERE revision = target_revision;

  -- Canonical SHA-256 input is the C-ordered concatenation of four
  -- UTF-8 byte-length-prefixed identity fields. Length prefixes make
  -- the representation unambiguous without relying on a separator escaping
  -- convention. The empty set hashes the empty byte string.
  SELECT count(*), encode(sha256(convert_to(coalesce(string_agg(
    octet_length(extension_id)::text || ':' || extension_id ||
    octet_length(extension_version_id::text)::text || ':' || extension_version_id::text ||
    octet_length(extension_version)::text || ':' || extension_version ||
    octet_length(package_digest)::text || ':' || package_digest,
    '' ORDER BY extension_id COLLATE "C"
  ), ''), 'UTF8')), 'hex')
  INTO actual_count, actual_digest
  FROM plugin_runtime_publication_members
  WHERE publication_revision = target_revision;

  IF expected_count IS NULL OR actual_count <> expected_count
     OR actual_digest IS DISTINCT FROM expected_digest THEN
    RAISE EXCEPTION
      'plugin runtime publication % has an invalid full set', target_revision;
  END IF;
  RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE CONSTRAINT TRIGGER plugin_runtime_publication_full_set
AFTER INSERT ON plugin_runtime_publications
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_plugin_runtime_desired_full_set();

CREATE CONSTRAINT TRIGGER plugin_runtime_publication_member_full_set
AFTER INSERT ON plugin_runtime_publication_members
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_plugin_runtime_desired_full_set();

-- One lease belongs to one api/worker process boot. A restarted process cannot
-- inherit the previous boot's acknowledgement or runtime-instance evidence.
CREATE TABLE plugin_runtime_nodes (
  node_id TEXT NOT NULL CHECK (octet_length(node_id) BETWEEN 1 AND 128),
  process_role TEXT NOT NULL CHECK (process_role IN ('api', 'worker')),
  boot_id TEXT NOT NULL CHECK (octet_length(boot_id) BETWEEN 1 AND 128),
  last_applied_revision BIGINT NOT NULL DEFAULT 0
    CHECK (last_applied_revision >= 0),
  first_seen_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  lease_expires_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (node_id, process_role, boot_id),
  CHECK (last_seen_at >= first_seen_at),
  CHECK (lease_expires_at > last_seen_at)
);

-- A boot always starts from revision zero. Heartbeats may leave progress
-- unchanged, but no write can move an existing boot backwards.
-- +goose StatementBegin
CREATE FUNCTION enforce_plugin_runtime_node_monotonicity() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF TG_OP = 'INSERT' THEN
    IF NEW.last_applied_revision <> 0 THEN
      RAISE EXCEPTION 'plugin runtime node boot must start at revision zero';
    END IF;
    RETURN NEW;
  END IF;

  IF NEW.node_id IS DISTINCT FROM OLD.node_id
     OR NEW.process_role IS DISTINCT FROM OLD.process_role
     OR NEW.boot_id IS DISTINCT FROM OLD.boot_id THEN
    RAISE EXCEPTION 'plugin runtime node boot identity is immutable';
  END IF;
  IF NEW.first_seen_at IS DISTINCT FROM OLD.first_seen_at THEN
    RAISE EXCEPTION 'plugin runtime node first-seen time is immutable';
  END IF;
  IF NEW.last_seen_at < OLD.last_seen_at THEN
    RAISE EXCEPTION 'plugin runtime node last-seen time cannot move backwards';
  END IF;
  IF NEW.last_applied_revision < OLD.last_applied_revision THEN
    RAISE EXCEPTION 'plugin runtime node revision cannot move backwards';
  END IF;
  RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER plugin_runtime_node_monotonic
BEFORE INSERT OR UPDATE ON plugin_runtime_nodes
FOR EACH ROW EXECUTE FUNCTION enforce_plugin_runtime_node_monotonicity();

CREATE INDEX plugin_runtime_nodes_live_idx
  ON plugin_runtime_nodes (
    lease_expires_at DESC, process_role, node_id, boot_id
  );

CREATE TABLE plugin_runtime_publication_acks (
  publication_revision BIGINT NOT NULL
    REFERENCES plugin_runtime_publications(revision) ON DELETE RESTRICT,
  node_id TEXT NOT NULL,
  process_role TEXT NOT NULL,
  boot_id TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'applying'
    CHECK (status IN ('applying', 'applied', 'failed')),
  applied_member_count INTEGER,
  applied_members_digest TEXT,
  error_reason TEXT NOT NULL DEFAULT ''
    CHECK (octet_length(error_reason) <= 2048),
  attempt_count INTEGER NOT NULL DEFAULT 1 CHECK (attempt_count > 0),
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  started_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  applied_at TIMESTAMPTZ,
  PRIMARY KEY (publication_revision, node_id, process_role, boot_id),
  FOREIGN KEY (node_id, process_role, boot_id)
    REFERENCES plugin_runtime_nodes(node_id, process_role, boot_id)
    ON DELETE RESTRICT,
  CHECK (updated_at >= started_at),
  CHECK (
    (status = 'applied'
      AND applied_member_count IS NOT NULL AND applied_member_count >= 0
      AND applied_members_digest ~ '^[0-9a-f]{64}$'
      AND error_reason = '' AND applied_at IS NOT NULL
      AND applied_at >= started_at)
    OR
    (status = 'failed'
      AND applied_member_count IS NULL AND applied_members_digest IS NULL
      AND error_reason <> '' AND applied_at IS NULL)
    OR
    (status = 'applying'
      AND applied_member_count IS NULL AND applied_members_digest IS NULL
      AND error_reason = '' AND applied_at IS NULL)
  )
);

CREATE INDEX plugin_runtime_publication_acks_status_idx
  ON plugin_runtime_publication_acks (
    status, publication_revision DESC, process_role, node_id, boot_id
  );

-- Actual runtime ids are node-local, but their artifact tuple must equal one
-- exact member of the desired revision acknowledged by that boot.
CREATE TABLE plugin_runtime_applied_members (
  publication_revision BIGINT NOT NULL,
  node_id TEXT NOT NULL,
  process_role TEXT NOT NULL,
  boot_id TEXT NOT NULL,
  extension_id TEXT NOT NULL CHECK (extension_id <> ''),
  extension_version_id BIGINT NOT NULL CHECK (extension_version_id > 0),
  extension_version TEXT NOT NULL CHECK (extension_version <> ''),
  package_digest TEXT NOT NULL CHECK (package_digest ~ '^[0-9a-f]{64}$'),
  runtime_instance_id TEXT NOT NULL
    CHECK (
      octet_length(runtime_instance_id) BETWEEN 1 AND 512
      AND runtime_instance_id = btrim(runtime_instance_id)
    ),
  recorded_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  PRIMARY KEY (
    publication_revision, node_id, process_role, boot_id, extension_id
  ),
  FOREIGN KEY (publication_revision, node_id, process_role, boot_id)
    REFERENCES plugin_runtime_publication_acks(
      publication_revision, node_id, process_role, boot_id
    ) ON DELETE RESTRICT,
  FOREIGN KEY (
    publication_revision, extension_id, extension_version_id,
    extension_version, package_digest
  ) REFERENCES plugin_runtime_publication_members(
    publication_revision, extension_id, extension_version_id,
    extension_version, package_digest
  ) ON DELETE RESTRICT
);

CREATE INDEX plugin_runtime_applied_members_instance_idx
  ON plugin_runtime_applied_members (
    node_id, process_role, boot_id, extension_id, runtime_instance_id
  );

-- Runtime-instance evidence may only be written by the still-live boot that
-- owns the acknowledgement. Commit-time validation repeats this fence.
-- +goose StatementBegin
CREATE FUNCTION enforce_plugin_runtime_applied_member_lease() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM plugin_runtime_nodes
    WHERE node_id = NEW.node_id
      AND process_role = NEW.process_role
      AND boot_id = NEW.boot_id
      AND lease_expires_at > statement_timestamp()
  ) THEN
    RAISE EXCEPTION 'plugin runtime applied member requires a live node lease';
  END IF;
  RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER plugin_runtime_applied_member_lease
BEFORE INSERT ON plugin_runtime_applied_members
FOR EACH ROW EXECUTE FUNCTION enforce_plugin_runtime_applied_member_lease();

-- Ack identity is fixed, every update increments its CAS revision exactly
-- once, and applied is terminal. Failed acknowledgements may only retry by
-- returning to applying with a new attempt.
-- +goose StatementBegin
CREATE FUNCTION enforce_plugin_runtime_ack_cas() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  node_applied_revision BIGINT;
  node_lease_expires_at TIMESTAMPTZ;
BEGIN
  SELECT last_applied_revision, lease_expires_at
    INTO node_applied_revision, node_lease_expires_at
  FROM plugin_runtime_nodes
  WHERE node_id = NEW.node_id
    AND process_role = NEW.process_role
    AND boot_id = NEW.boot_id;

  IF node_applied_revision IS NULL
     OR node_lease_expires_at <= statement_timestamp() THEN
    RAISE EXCEPTION 'plugin runtime acknowledgement requires a live node lease';
  END IF;

  IF TG_OP = 'INSERT' THEN
    IF NEW.status <> 'applying'
       OR NEW.attempt_count <> 1 OR NEW.revision <> 1 THEN
      RAISE EXCEPTION 'plugin runtime acknowledgement must begin at applying revision 1';
    END IF;
    IF NEW.publication_revision <= node_applied_revision THEN
      RAISE EXCEPTION 'plugin runtime acknowledgement revision is not newer than the node';
    END IF;
    RETURN NEW;
  END IF;

  IF NEW.publication_revision IS DISTINCT FROM OLD.publication_revision
     OR NEW.node_id IS DISTINCT FROM OLD.node_id
     OR NEW.process_role IS DISTINCT FROM OLD.process_role
     OR NEW.boot_id IS DISTINCT FROM OLD.boot_id THEN
    RAISE EXCEPTION 'plugin runtime acknowledgement identity is immutable';
  END IF;
  IF NEW.revision <> OLD.revision + 1 THEN
    RAISE EXCEPTION 'plugin runtime acknowledgement revision must increment by one';
  END IF;
  IF NEW.updated_at < OLD.updated_at THEN
    RAISE EXCEPTION 'plugin runtime acknowledgement time cannot move backwards';
  END IF;

  IF OLD.status = 'applying' THEN
    IF NEW.status NOT IN ('applied', 'failed')
       OR NEW.attempt_count <> OLD.attempt_count THEN
      RAISE EXCEPTION 'plugin runtime applying acknowledgement has an invalid transition';
    END IF;
  ELSIF OLD.status = 'failed' THEN
    IF NEW.status <> 'applying'
       OR NEW.attempt_count <> OLD.attempt_count + 1 THEN
      RAISE EXCEPTION 'plugin runtime failed acknowledgement must retry as a new attempt';
    END IF;
    IF NEW.publication_revision <= node_applied_revision THEN
      RAISE EXCEPTION 'plugin runtime acknowledgement revision is not newer than the node';
    END IF;
  ELSE
    RAISE EXCEPTION 'plugin runtime applied acknowledgement is terminal';
  END IF;
  RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER plugin_runtime_ack_cas
BEFORE INSERT OR UPDATE ON plugin_runtime_publication_acks
FOR EACH ROW EXECUTE FUNCTION enforce_plugin_runtime_ack_cas();

-- Advancing a boot's active revision and completing its ack are one atomic
-- transaction. This deferred check permits either SQL statement order.
-- +goose StatementBegin
CREATE FUNCTION validate_plugin_runtime_node_progress() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF NEW.last_applied_revision = OLD.last_applied_revision THEN
    RETURN NULL;
  END IF;
  IF NEW.lease_expires_at <= statement_timestamp() OR NOT EXISTS (
    SELECT 1 FROM plugin_runtime_publication_acks
    WHERE publication_revision = NEW.last_applied_revision
      AND node_id = NEW.node_id
      AND process_role = NEW.process_role
      AND boot_id = NEW.boot_id
      AND status = 'applied'
  ) THEN
    RAISE EXCEPTION 'plugin runtime node revision has no live applied acknowledgement';
  END IF;
  RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE CONSTRAINT TRIGGER plugin_runtime_node_progress
AFTER UPDATE ON plugin_runtime_nodes
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_plugin_runtime_node_progress();

-- Ack rows are mutable only through their CAS state machine; applied member
-- rows are immutable convergence evidence.
-- +goose StatementBegin
CREATE FUNCTION reject_plugin_runtime_evidence_removal() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  RAISE EXCEPTION 'plugin runtime acknowledgement evidence cannot be removed';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER plugin_runtime_ack_no_delete
BEFORE DELETE ON plugin_runtime_publication_acks
FOR EACH ROW EXECUTE FUNCTION reject_plugin_runtime_evidence_removal();

CREATE TRIGGER plugin_runtime_ack_no_truncate
BEFORE TRUNCATE ON plugin_runtime_publication_acks
FOR EACH STATEMENT EXECUTE FUNCTION reject_plugin_runtime_evidence_removal();

CREATE TRIGGER plugin_runtime_applied_member_immutable
BEFORE UPDATE OR DELETE ON plugin_runtime_applied_members
FOR EACH ROW EXECUTE FUNCTION reject_plugin_runtime_evidence_removal();

CREATE TRIGGER plugin_runtime_applied_member_no_truncate
BEFORE TRUNCATE ON plugin_runtime_applied_members
FOR EACH STATEMENT EXECUTE FUNCTION reject_plugin_runtime_evidence_removal();

-- Deferred validation lets an applier insert every exact runtime-instance row
-- and flip its ack to applied atomically. Partial or foreign sets fail commit.
-- +goose StatementBegin
CREATE FUNCTION validate_plugin_runtime_applied_full_set() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  target_publication BIGINT;
  target_node TEXT;
  target_role TEXT;
  target_boot TEXT;
  ack_status TEXT;
  ack_count INTEGER;
  ack_digest TEXT;
  desired_count INTEGER;
  desired_digest TEXT;
  node_applied_revision BIGINT;
  node_lease_expires_at TIMESTAMPTZ;
  actual_count BIGINT;
  actual_digest TEXT;
BEGIN
  target_publication := NEW.publication_revision;
  target_node := NEW.node_id;
  target_role := NEW.process_role;
  target_boot := NEW.boot_id;

  SELECT a.status, a.applied_member_count, a.applied_members_digest,
         p.member_count, p.members_digest, n.last_applied_revision,
         n.lease_expires_at
    INTO ack_status, ack_count, ack_digest, desired_count, desired_digest,
         node_applied_revision, node_lease_expires_at
  FROM plugin_runtime_publication_acks AS a
  JOIN plugin_runtime_publications AS p
    ON p.revision = a.publication_revision
  JOIN plugin_runtime_nodes AS n
    ON n.node_id = a.node_id
   AND n.process_role = a.process_role
   AND n.boot_id = a.boot_id
  WHERE a.publication_revision = target_publication
    AND a.node_id = target_node
    AND a.process_role = target_role
    AND a.boot_id = target_boot;

  SELECT count(*), encode(sha256(convert_to(coalesce(string_agg(
    octet_length(extension_id)::text || ':' || extension_id ||
    octet_length(extension_version_id::text)::text || ':' || extension_version_id::text ||
    octet_length(extension_version)::text || ':' || extension_version ||
    octet_length(package_digest)::text || ':' || package_digest,
    '' ORDER BY extension_id COLLATE "C"
  ), ''), 'UTF8')), 'hex')
  INTO actual_count, actual_digest
  FROM plugin_runtime_applied_members
  WHERE publication_revision = target_publication
    AND node_id = target_node
    AND process_role = target_role
    AND boot_id = target_boot;

  IF node_lease_expires_at IS NULL
     OR node_lease_expires_at <= statement_timestamp() THEN
    RAISE EXCEPTION 'plugin runtime acknowledgement commit requires a live node lease';
  END IF;

  IF ack_status = 'applied' THEN
    IF ack_count IS DISTINCT FROM desired_count
       OR ack_digest IS DISTINCT FROM desired_digest
       OR actual_digest IS DISTINCT FROM desired_digest
       OR node_applied_revision IS DISTINCT FROM target_publication
       OR actual_count <> desired_count THEN
      RAISE EXCEPTION
        'plugin runtime applied set is incomplete for revision % and boot %/%/%',
        target_publication, target_node, target_role, target_boot;
    END IF;
  ELSIF actual_count <> 0 THEN
    RAISE EXCEPTION
      'plugin runtime evidence exists before applied for revision % and boot %/%/%',
      target_publication, target_node, target_role, target_boot;
  END IF;
  RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE CONSTRAINT TRIGGER plugin_runtime_ack_applied_full_set
AFTER INSERT OR UPDATE ON plugin_runtime_publication_acks
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_plugin_runtime_applied_full_set();

CREATE CONSTRAINT TRIGGER plugin_runtime_applied_member_full_set
AFTER INSERT ON plugin_runtime_applied_members
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION validate_plugin_runtime_applied_full_set();

-- pg_notify delivery is transaction-scoped: consumers observe this hint only
-- after the complete desired full set commits.
-- +goose StatementBegin
CREATE FUNCTION notify_plugin_runtime_publication() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  PERFORM pg_notify('sforum_plugin_runtime_publication', NEW.revision::text);
  RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER plugin_runtime_publication_notify
AFTER INSERT ON plugin_runtime_publications
FOR EACH ROW EXECUTE FUNCTION notify_plugin_runtime_publication();

-- +goose Down
-- Desired and applied runtime rows are durable convergence evidence. Refuse to
-- erase them after any revision or acknowledgement has been issued.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM plugin_runtime_publications)
     OR EXISTS (SELECT 1 FROM plugin_runtime_publication_acks)
     OR EXISTS (SELECT 1 FROM plugin_runtime_applied_members) THEN
    RAISE EXCEPTION 'cannot remove plugin runtime publication history';
  END IF;
END $$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS plugin_runtime_publication_notify ON plugin_runtime_publications;
DROP TRIGGER IF EXISTS plugin_runtime_ack_applied_full_set ON plugin_runtime_publication_acks;
DROP TRIGGER IF EXISTS plugin_runtime_applied_member_full_set ON plugin_runtime_applied_members;
DROP TRIGGER IF EXISTS plugin_runtime_node_progress ON plugin_runtime_nodes;
DROP TRIGGER IF EXISTS plugin_runtime_applied_member_lease ON plugin_runtime_applied_members;
DROP TRIGGER IF EXISTS plugin_runtime_applied_member_no_truncate ON plugin_runtime_applied_members;
DROP TRIGGER IF EXISTS plugin_runtime_applied_member_immutable ON plugin_runtime_applied_members;
DROP TRIGGER IF EXISTS plugin_runtime_ack_no_truncate ON plugin_runtime_publication_acks;
DROP TRIGGER IF EXISTS plugin_runtime_ack_no_delete ON plugin_runtime_publication_acks;
DROP TRIGGER IF EXISTS plugin_runtime_ack_cas ON plugin_runtime_publication_acks;
DROP TRIGGER IF EXISTS plugin_runtime_node_monotonic ON plugin_runtime_nodes;
DROP TRIGGER IF EXISTS plugin_runtime_publication_member_full_set ON plugin_runtime_publication_members;
DROP TRIGGER IF EXISTS plugin_runtime_publication_full_set ON plugin_runtime_publications;
DROP TRIGGER IF EXISTS plugin_runtime_extension_type_immutable ON extensions;
DROP TRIGGER IF EXISTS plugin_runtime_publication_member_type ON plugin_runtime_publication_members;
DROP TRIGGER IF EXISTS plugin_runtime_publication_member_no_truncate ON plugin_runtime_publication_members;
DROP TRIGGER IF EXISTS plugin_runtime_publication_member_immutable ON plugin_runtime_publication_members;
DROP TRIGGER IF EXISTS plugin_runtime_publication_no_truncate ON plugin_runtime_publications;
DROP TRIGGER IF EXISTS plugin_runtime_publication_immutable ON plugin_runtime_publications;
DROP FUNCTION IF EXISTS notify_plugin_runtime_publication();
DROP FUNCTION IF EXISTS validate_plugin_runtime_applied_full_set();
DROP FUNCTION IF EXISTS reject_plugin_runtime_evidence_removal();
DROP FUNCTION IF EXISTS validate_plugin_runtime_node_progress();
DROP FUNCTION IF EXISTS enforce_plugin_runtime_applied_member_lease();
DROP FUNCTION IF EXISTS enforce_plugin_runtime_ack_cas();
DROP FUNCTION IF EXISTS enforce_plugin_runtime_node_monotonicity();
DROP FUNCTION IF EXISTS validate_plugin_runtime_desired_full_set();
DROP FUNCTION IF EXISTS reject_published_plugin_type_change();
DROP FUNCTION IF EXISTS enforce_plugin_runtime_member_type();
DROP FUNCTION IF EXISTS reject_plugin_runtime_desired_mutation();
DROP INDEX IF EXISTS plugin_runtime_applied_members_instance_idx;
DROP TABLE IF EXISTS plugin_runtime_applied_members;
DROP INDEX IF EXISTS plugin_runtime_publication_acks_status_idx;
DROP TABLE IF EXISTS plugin_runtime_publication_acks;
DROP INDEX IF EXISTS plugin_runtime_nodes_live_idx;
DROP TABLE IF EXISTS plugin_runtime_nodes;
DROP INDEX IF EXISTS plugin_runtime_publication_members_version_idx;
DROP TABLE IF EXISTS plugin_runtime_publication_members;
DROP INDEX IF EXISTS extension_versions_plugin_runtime_identity_idx;
DROP INDEX IF EXISTS plugin_runtime_publications_created_idx;
DROP TABLE IF EXISTS plugin_runtime_publications;
