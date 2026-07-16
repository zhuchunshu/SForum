-- +goose Up
-- Identity root publications bind the complete normalized publication graph,
-- including session policy and risk hooks that have no leaf ownership row.
-- Runtime instance ids are intentionally excluded from publication_json: the
-- Host proves the current process binding separately on every publication.
CREATE TABLE extension_identity_registry_publications (
  owner_extension_id TEXT NOT NULL
    CHECK (owner_extension_id ~ '^[a-z0-9][a-z0-9._-]{1,120}$'),
  revision BIGINT NOT NULL CHECK (revision > 0),
  registry_state TEXT NOT NULL
    CHECK (registry_state IN ('active', 'tombstone')),
  extension_version_id BIGINT NOT NULL CHECK (extension_version_id > 0),
  extension_version TEXT NOT NULL CHECK (extension_version <> ''),
  package_digest TEXT NOT NULL
    CHECK (package_digest ~ '^[0-9a-f]{64}$'),
  schema_version TEXT NOT NULL
    CHECK (schema_version = 'sforum.identity-registry@1'),
  publication_digest TEXT NOT NULL
    CHECK (publication_digest ~ '^[0-9a-f]{64}$'),
  publication_json JSONB NOT NULL
    CHECK (jsonb_typeof(publication_json) = 'object'),
  actor_user_id BIGINT NOT NULL CHECK (actor_user_id > 0),
  audit_event_id BIGINT NOT NULL CHECK (audit_event_id > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  PRIMARY KEY (owner_extension_id, revision),
  CHECK (owner_extension_id !~ '^core([.]|$)')
);

CREATE INDEX extension_identity_registry_publications_artifact_idx
  ON extension_identity_registry_publications (
    owner_extension_id, extension_version_id, package_digest, revision DESC
  );

-- +goose StatementBegin
CREATE FUNCTION validate_extension_identity_registry_publication() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  previous extension_identity_registry_publications%ROWTYPE;
BEGIN
  -- Keep the artifact -> publication lock order aligned with lifecycle writes
  -- and delete guards, so concurrent upgrade/uninstall cannot deadlock.
  PERFORM 1
  FROM extension_versions
  JOIN extensions ON extensions.id = extension_versions.extension_id
  WHERE extension_versions.id = NEW.extension_version_id
    AND extension_versions.extension_id = NEW.owner_extension_id
    AND extension_versions.version = NEW.extension_version
    AND extension_versions.package_digest = NEW.package_digest
    AND extensions.type = 'plugin'
  FOR KEY SHARE OF extension_versions, extensions;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'identity registry root publication exact artifact is invalid';
  END IF;

  IF NEW.publication_json #>> '{artifact,extensionId}'
      IS DISTINCT FROM NEW.owner_extension_id
    OR NEW.publication_json #>> '{artifact,versionId}'
      IS DISTINCT FROM NEW.extension_version_id::TEXT
    OR NEW.publication_json #>> '{artifact,extensionVersion}'
      IS DISTINCT FROM NEW.extension_version
    OR NEW.publication_json #>> '{artifact,packageDigest}'
      IS DISTINCT FROM NEW.package_digest
    OR COALESCE(NEW.publication_json #>> '{artifact,runtimeInstanceId}', '') <> ''
    OR COALESCE(NEW.publication_json #>> '{artifact,core}', 'false') <> 'false' THEN
    RAISE EXCEPTION 'identity registry root publication JSON artifact mismatch';
  END IF;

  SELECT * INTO previous
  FROM extension_identity_registry_publications
  WHERE owner_extension_id = NEW.owner_extension_id
  ORDER BY revision DESC
  LIMIT 1
  FOR UPDATE;

  IF previous.revision IS NULL THEN
    IF NEW.revision <> 1 OR NEW.registry_state <> 'active' THEN
      RAISE EXCEPTION 'identity registry root publication must begin active at revision 1';
    END IF;
    RETURN NEW;
  END IF;

  IF NEW.revision <> previous.revision + 1 THEN
    RAISE EXCEPTION 'identity registry root publication revision conflict';
  END IF;

  IF NEW.registry_state = 'tombstone' AND (
    previous.registry_state <> 'active'
    OR NEW.extension_version_id IS DISTINCT FROM previous.extension_version_id
    OR NEW.extension_version IS DISTINCT FROM previous.extension_version
    OR NEW.package_digest IS DISTINCT FROM previous.package_digest
    OR NEW.schema_version IS DISTINCT FROM previous.schema_version
    OR NEW.publication_digest IS DISTINCT FROM previous.publication_digest
    OR NEW.publication_json IS DISTINCT FROM previous.publication_json
  ) THEN
    RAISE EXCEPTION 'identity registry root tombstone does not match the active publication';
  END IF;

  IF NEW.registry_state = 'active'
    AND previous.registry_state = 'active'
    AND NEW.extension_version_id = previous.extension_version_id
    AND NEW.extension_version = previous.extension_version
    AND NEW.package_digest = previous.package_digest THEN
    RAISE EXCEPTION 'identity registry root exact artifact cannot drift or replay as a new revision';
  END IF;

  IF NEW.registry_state = 'active'
    AND previous.registry_state = 'tombstone'
    AND NEW.extension_version_id = previous.extension_version_id
    AND NEW.extension_version = previous.extension_version
    AND NEW.package_digest = previous.package_digest
    AND (
      NEW.schema_version IS DISTINCT FROM previous.schema_version
      OR NEW.publication_digest IS DISTINCT FROM previous.publication_digest
      OR NEW.publication_json IS DISTINCT FROM previous.publication_json
    ) THEN
    RAISE EXCEPTION 'identity registry root exact artifact cannot drift on reactivation';
  END IF;

  RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER extension_identity_registry_publication_valid
BEFORE INSERT ON extension_identity_registry_publications
FOR EACH ROW EXECUTE FUNCTION validate_extension_identity_registry_publication();

CREATE TRIGGER extension_identity_registry_publication_immutable
BEFORE UPDATE OR DELETE ON extension_identity_registry_publications
FOR EACH ROW EXECUTE FUNCTION reject_extension_identity_registry_history_mutation();

CREATE TRIGGER extension_identity_registry_publication_no_truncate
BEFORE TRUNCATE ON extension_identity_registry_publications
FOR EACH STATEMENT EXECUTE FUNCTION reject_extension_identity_registry_history_mutation();

-- +goose StatementBegin
CREATE FUNCTION reject_extension_identity_registry_publication_owner_type_change() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF NEW.type IS DISTINCT FROM OLD.type AND EXISTS (
    SELECT 1
    FROM extension_identity_registry_publications
    WHERE owner_extension_id = OLD.id
  ) THEN
    RAISE EXCEPTION 'identity registry root publication owner type is immutable';
  END IF;
  RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER extension_identity_registry_publication_owner_type_immutable
BEFORE UPDATE OF type ON extensions
FOR EACH ROW EXECUTE FUNCTION reject_extension_identity_registry_publication_owner_type_change();

-- +goose StatementBegin
CREATE FUNCTION reject_extension_identity_registry_publication_active_extension_delete() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  active_state TEXT;
BEGIN
  SELECT registry_state INTO active_state
  FROM extension_identity_registry_publications
  WHERE owner_extension_id = OLD.id
  ORDER BY revision DESC
  LIMIT 1
  FOR UPDATE;

  IF active_state = 'active' THEN
    RAISE EXCEPTION 'identity registry root publication must be tombstoned before uninstall';
  END IF;
  RETURN OLD;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER extension_identity_registry_publication_active_extension_delete_guard
BEFORE DELETE ON extensions
FOR EACH ROW EXECUTE FUNCTION reject_extension_identity_registry_publication_active_extension_delete();

-- +goose StatementBegin
CREATE FUNCTION reject_extension_identity_registry_publication_active_version_delete() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  active extension_identity_registry_publications%ROWTYPE;
BEGIN
  SELECT * INTO active
  FROM extension_identity_registry_publications
  WHERE owner_extension_id = OLD.extension_id
  ORDER BY revision DESC
  LIMIT 1
  FOR UPDATE;

  IF active.registry_state = 'active'
    AND active.extension_version_id = OLD.id THEN
    RAISE EXCEPTION 'active identity registry root artifact cannot be removed before tombstone';
  END IF;
  RETURN OLD;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER extension_identity_registry_publication_active_version_delete_guard
BEFORE DELETE ON extension_versions
FOR EACH ROW EXECUTE FUNCTION reject_extension_identity_registry_publication_active_version_delete();

-- +goose Down
LOCK TABLE extension_identity_registry_publications IN ACCESS EXCLUSIVE MODE;

-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM extension_identity_registry_publications) THEN
    RAISE EXCEPTION 'cannot remove extension identity registry root publication history';
  END IF;
END;
$$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS extension_identity_registry_publication_active_version_delete_guard
  ON extension_versions;
DROP FUNCTION IF EXISTS reject_extension_identity_registry_publication_active_version_delete();
DROP TRIGGER IF EXISTS extension_identity_registry_publication_active_extension_delete_guard
  ON extensions;
DROP FUNCTION IF EXISTS reject_extension_identity_registry_publication_active_extension_delete();
DROP TRIGGER IF EXISTS extension_identity_registry_publication_owner_type_immutable
  ON extensions;
DROP FUNCTION IF EXISTS reject_extension_identity_registry_publication_owner_type_change();
DROP TRIGGER IF EXISTS extension_identity_registry_publication_no_truncate
  ON extension_identity_registry_publications;
DROP TRIGGER IF EXISTS extension_identity_registry_publication_immutable
  ON extension_identity_registry_publications;
DROP TRIGGER IF EXISTS extension_identity_registry_publication_valid
  ON extension_identity_registry_publications;
DROP TABLE IF EXISTS extension_identity_registry_publications;
DROP FUNCTION IF EXISTS validate_extension_identity_registry_publication();
