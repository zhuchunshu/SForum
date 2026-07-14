-- +goose Up
-- P8: the database stores desired theme runtime revisions and per-node
-- convergence evidence. PostgreSQL is authoritative; NOTIFY only wakes
-- watchers, which always recover from these durable rows after a disconnect.
CREATE TABLE theme_runtime_publications (
  revision BIGSERIAL PRIMARY KEY,
  desired_state TEXT NOT NULL
    CHECK (desired_state IN ('active', 'none')),
  theme_id TEXT,
  theme_version TEXT,
  package_digest TEXT,
  source_theme_id TEXT,
  source_theme_version TEXT,
  source_package_digest TEXT,
  core_replacements_approved BOOLEAN NOT NULL DEFAULT FALSE,
  actor_user_id BIGINT
    CHECK (actor_user_id IS NULL OR actor_user_id > 0),
  reason TEXT NOT NULL
    CHECK (reason IN ('activation', 'compensation', 'startup_repair')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  CHECK (
    (desired_state = 'active'
      AND theme_id IS NOT NULL AND theme_id <> ''
      AND theme_version IS NOT NULL AND theme_version <> ''
      AND package_digest ~ '^[0-9a-f]{64}$')
    OR
    (desired_state = 'none'
      AND theme_id IS NULL
      AND theme_version IS NULL
      AND package_digest IS NULL
      AND core_replacements_approved = FALSE)
  ),
  CHECK (
    (source_theme_id IS NULL
      AND source_theme_version IS NULL
      AND source_package_digest IS NULL)
    OR
    (source_theme_id IS NOT NULL AND source_theme_id <> ''
      AND source_theme_version IS NOT NULL AND source_theme_version <> ''
      AND source_package_digest ~ '^[0-9a-f]{64}$')
  ),
  CHECK (core_replacements_approved = FALSE OR actor_user_id IS NOT NULL)
);

CREATE INDEX theme_runtime_publications_created_idx
  ON theme_runtime_publications (created_at DESC, revision DESC);

-- Desired revisions are append-only. Compensation publishes a new revision;
-- mutating an old tuple would make an existing node acknowledgement ambiguous.
-- +goose StatementBegin
CREATE FUNCTION reject_theme_runtime_publication_mutation() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  RAISE EXCEPTION 'theme runtime publications are append-only';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER theme_runtime_publication_immutable
BEFORE UPDATE OR DELETE ON theme_runtime_publications
FOR EACH ROW EXECUTE FUNCTION reject_theme_runtime_publication_mutation();

CREATE TRIGGER theme_runtime_publication_no_truncate
BEFORE TRUNCATE ON theme_runtime_publications
FOR EACH STATEMENT EXECUTE FUNCTION reject_theme_runtime_publication_mutation();

-- A node row represents one boot, so a restarted process cannot inherit an
-- acknowledgement written by its predecessor.
CREATE TABLE theme_runtime_nodes (
  node_id TEXT NOT NULL
    CHECK (octet_length(node_id) BETWEEN 1 AND 128),
  boot_id TEXT NOT NULL
    CHECK (octet_length(boot_id) BETWEEN 1 AND 128),
  last_applied_revision BIGINT NOT NULL DEFAULT 0
    CHECK (last_applied_revision >= 0),
  first_seen_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  lease_expires_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (node_id, boot_id),
  CHECK (last_seen_at >= first_seen_at),
  CHECK (lease_expires_at > last_seen_at)
);

CREATE INDEX theme_runtime_nodes_live_idx
  ON theme_runtime_nodes (lease_expires_at DESC, node_id, boot_id);

CREATE TABLE theme_runtime_publication_acks (
  publication_revision BIGINT NOT NULL
    REFERENCES theme_runtime_publications(revision) ON DELETE RESTRICT,
  node_id TEXT NOT NULL,
  boot_id TEXT NOT NULL,
  status TEXT NOT NULL
    CHECK (status IN ('applying', 'applied', 'failed')),
  applied_state TEXT
    CHECK (applied_state IS NULL OR applied_state IN ('active', 'none')),
  applied_theme_id TEXT,
  applied_theme_version TEXT,
  applied_package_digest TEXT,
  error_reason TEXT NOT NULL DEFAULT ''
    CHECK (octet_length(error_reason) <= 2048),
  attempt_count INTEGER NOT NULL DEFAULT 1
    CHECK (attempt_count > 0),
  revision BIGINT NOT NULL DEFAULT 1
    CHECK (revision > 0),
  started_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  applied_at TIMESTAMPTZ,
  PRIMARY KEY (publication_revision, node_id, boot_id),
  FOREIGN KEY (node_id, boot_id)
    REFERENCES theme_runtime_nodes(node_id, boot_id) ON DELETE RESTRICT,
  CHECK (updated_at >= started_at),
  CHECK (
    (status = 'applied'
      AND applied_state IS NOT NULL
      AND applied_at IS NOT NULL
      AND error_reason = '')
    OR
    (status = 'failed'
      AND applied_state IS NULL
      AND applied_at IS NULL
      AND error_reason <> '')
    OR
    (status = 'applying'
      AND applied_state IS NULL
      AND applied_at IS NULL
      AND error_reason = '')
  ),
  CHECK (
    (applied_state = 'active'
      AND applied_theme_id IS NOT NULL AND applied_theme_id <> ''
      AND applied_theme_version IS NOT NULL AND applied_theme_version <> ''
      AND applied_package_digest ~ '^[0-9a-f]{64}$')
    OR
    (applied_state = 'none'
      AND applied_theme_id IS NULL
      AND applied_theme_version IS NULL
      AND applied_package_digest IS NULL)
    OR
    (applied_state IS NULL
      AND applied_theme_id IS NULL
      AND applied_theme_version IS NULL
      AND applied_package_digest IS NULL)
  )
);

CREATE INDEX theme_runtime_publication_acks_status_idx
  ON theme_runtime_publication_acks (status, publication_revision DESC, node_id, boot_id);

-- NOTIFY is emitted at transaction commit. Its payload is only a revision
-- hint; consumers must load and validate the durable publication row.
-- +goose StatementBegin
CREATE FUNCTION notify_theme_runtime_publication() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  PERFORM pg_notify('sforum_theme_runtime_publication', NEW.revision::text);
  RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER theme_runtime_publication_notify
AFTER INSERT ON theme_runtime_publications
FOR EACH ROW EXECUTE FUNCTION notify_theme_runtime_publication();

-- +goose Down
-- Publication and acknowledgement history is convergence evidence. Do not
-- silently erase it once a runtime revision has been issued.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM theme_runtime_publications)
     OR EXISTS (SELECT 1 FROM theme_runtime_publication_acks) THEN
    RAISE EXCEPTION 'cannot remove theme runtime publication history';
  END IF;
END $$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS theme_runtime_publication_notify ON theme_runtime_publications;
DROP TRIGGER IF EXISTS theme_runtime_publication_immutable ON theme_runtime_publications;
DROP TRIGGER IF EXISTS theme_runtime_publication_no_truncate ON theme_runtime_publications;
DROP FUNCTION IF EXISTS notify_theme_runtime_publication();
DROP FUNCTION IF EXISTS reject_theme_runtime_publication_mutation();
DROP INDEX IF EXISTS theme_runtime_publication_acks_status_idx;
DROP TABLE IF EXISTS theme_runtime_publication_acks;
DROP INDEX IF EXISTS theme_runtime_nodes_live_idx;
DROP TABLE IF EXISTS theme_runtime_nodes;
DROP INDEX IF EXISTS theme_runtime_publications_created_idx;
DROP TABLE IF EXISTS theme_runtime_publications;
