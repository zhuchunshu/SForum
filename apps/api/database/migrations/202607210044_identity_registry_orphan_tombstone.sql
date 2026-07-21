-- +goose Up
-- Startup safety: allow completing identity declaration/root tombstones after a
-- plugin row was force-removed (manual DELETE / incomplete cleanup). Active
-- inserts still require a live exact artifact. Tombstones only skip the live
-- artifact lock when the owner extension row is already gone, and previous tip
-- matching remains mandatory so a random tombstone cannot retire a foreign tip.

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION validate_extension_identity_registry_declaration() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  stored_owner TEXT;
  previous extension_identity_registry_declarations%ROWTYPE;
  artifact_found BOOLEAN := FALSE;
BEGIN
  -- Lock artifact identity before its owner row when the package still exists.
  PERFORM 1
  FROM extension_versions
  JOIN extensions ON extensions.id = extension_versions.extension_id
  WHERE extension_versions.id = NEW.extension_version_id
    AND extension_versions.extension_id = NEW.owner_extension_id
    AND extension_versions.version = NEW.extension_version
    AND extension_versions.package_digest = NEW.package_digest
    AND extensions.type = 'plugin'
  FOR KEY SHARE OF extension_versions, extensions;

  IF FOUND THEN
    artifact_found := TRUE;
  END IF;

  IF NOT artifact_found THEN
    -- Active publications always need a live exact artifact.
    -- Tombstones may finish retirement after force-delete only when the owner
    -- extension row is absent; if the extension still exists, require the version.
    IF NEW.registry_state IS DISTINCT FROM 'tombstone'
      OR EXISTS (SELECT 1 FROM extensions WHERE id = NEW.owner_extension_id) THEN
      RAISE EXCEPTION 'identity registry declaration exact artifact is invalid';
    END IF;
  END IF;

  SELECT owner_extension_id INTO stored_owner
  FROM extension_identity_registry_owners
  WHERE identity_kind = NEW.identity_kind
    AND stable_id = NEW.stable_id
  FOR UPDATE;

  IF stored_owner IS NULL OR stored_owner IS DISTINCT FROM NEW.owner_extension_id THEN
    RAISE EXCEPTION 'identity registry declaration owner mismatch';
  END IF;

  SELECT * INTO previous
  FROM extension_identity_registry_declarations
  WHERE identity_kind = NEW.identity_kind
    AND stable_id = NEW.stable_id
  ORDER BY revision DESC
  LIMIT 1;

  IF previous.revision IS NULL THEN
    IF NEW.revision <> 1 OR NEW.registry_state <> 'active' THEN
      RAISE EXCEPTION 'identity registry declaration must begin active at revision 1';
    END IF;
    RETURN NEW;
  END IF;

  IF NEW.revision <> previous.revision + 1 THEN
    RAISE EXCEPTION 'identity registry declaration revision conflict';
  END IF;

  IF NEW.registry_state = 'tombstone' AND (
    previous.registry_state <> 'active'
    OR NEW.extension_version_id IS DISTINCT FROM previous.extension_version_id
    OR NEW.extension_version IS DISTINCT FROM previous.extension_version
    OR NEW.package_digest IS DISTINCT FROM previous.package_digest
    OR NEW.contract_version IS DISTINCT FROM previous.contract_version
    OR NEW.declaration_digest IS DISTINCT FROM previous.declaration_digest
  ) THEN
    RAISE EXCEPTION 'identity registry tombstone does not match the active artifact';
  END IF;

  IF NEW.registry_state = 'active'
    AND previous.registry_state = 'active'
    AND NEW.extension_version_id = previous.extension_version_id
    AND NEW.extension_version = previous.extension_version
    AND NEW.package_digest = previous.package_digest THEN
    RAISE EXCEPTION 'identity registry exact artifact cannot drift or replay as a new revision';
  END IF;

  IF NEW.registry_state = 'active'
    AND previous.registry_state = 'tombstone'
    AND NEW.extension_version_id = previous.extension_version_id
    AND NEW.extension_version = previous.extension_version
    AND NEW.package_digest = previous.package_digest
    AND (
      NEW.contract_version IS DISTINCT FROM previous.contract_version
      OR NEW.declaration_digest IS DISTINCT FROM previous.declaration_digest
    ) THEN
    RAISE EXCEPTION 'identity registry exact artifact cannot drift on reactivation';
  END IF;

  RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION validate_extension_identity_registry_publication() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  previous extension_identity_registry_publications%ROWTYPE;
  artifact_found BOOLEAN := FALSE;
BEGIN
  PERFORM 1
  FROM extension_versions
  JOIN extensions ON extensions.id = extension_versions.extension_id
  WHERE extension_versions.id = NEW.extension_version_id
    AND extension_versions.extension_id = NEW.owner_extension_id
    AND extension_versions.version = NEW.extension_version
    AND extension_versions.package_digest = NEW.package_digest
    AND extensions.type = 'plugin'
  FOR KEY SHARE OF extension_versions, extensions;

  IF FOUND THEN
    artifact_found := TRUE;
  END IF;

  IF NOT artifact_found THEN
    IF NEW.registry_state IS DISTINCT FROM 'tombstone'
      OR EXISTS (SELECT 1 FROM extensions WHERE id = NEW.owner_extension_id) THEN
      RAISE EXCEPTION 'identity registry root publication exact artifact is invalid';
    END IF;
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

-- +goose Down
-- Restore strict live-artifact requirement for every insert (including tombstones).

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION validate_extension_identity_registry_declaration() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  stored_owner TEXT;
  previous extension_identity_registry_declarations%ROWTYPE;
BEGIN
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
    RAISE EXCEPTION 'identity registry declaration exact artifact is invalid';
  END IF;

  SELECT owner_extension_id INTO stored_owner
  FROM extension_identity_registry_owners
  WHERE identity_kind = NEW.identity_kind
    AND stable_id = NEW.stable_id
  FOR UPDATE;

  IF stored_owner IS NULL OR stored_owner IS DISTINCT FROM NEW.owner_extension_id THEN
    RAISE EXCEPTION 'identity registry declaration owner mismatch';
  END IF;

  SELECT * INTO previous
  FROM extension_identity_registry_declarations
  WHERE identity_kind = NEW.identity_kind
    AND stable_id = NEW.stable_id
  ORDER BY revision DESC
  LIMIT 1;

  IF previous.revision IS NULL THEN
    IF NEW.revision <> 1 OR NEW.registry_state <> 'active' THEN
      RAISE EXCEPTION 'identity registry declaration must begin active at revision 1';
    END IF;
    RETURN NEW;
  END IF;

  IF NEW.revision <> previous.revision + 1 THEN
    RAISE EXCEPTION 'identity registry declaration revision conflict';
  END IF;

  IF NEW.registry_state = 'tombstone' AND (
    previous.registry_state <> 'active'
    OR NEW.extension_version_id IS DISTINCT FROM previous.extension_version_id
    OR NEW.extension_version IS DISTINCT FROM previous.extension_version
    OR NEW.package_digest IS DISTINCT FROM previous.package_digest
    OR NEW.contract_version IS DISTINCT FROM previous.contract_version
    OR NEW.declaration_digest IS DISTINCT FROM previous.declaration_digest
  ) THEN
    RAISE EXCEPTION 'identity registry tombstone does not match the active artifact';
  END IF;

  IF NEW.registry_state = 'active'
    AND previous.registry_state = 'active'
    AND NEW.extension_version_id = previous.extension_version_id
    AND NEW.extension_version = previous.extension_version
    AND NEW.package_digest = previous.package_digest THEN
    RAISE EXCEPTION 'identity registry exact artifact cannot drift or replay as a new revision';
  END IF;

  IF NEW.registry_state = 'active'
    AND previous.registry_state = 'tombstone'
    AND NEW.extension_version_id = previous.extension_version_id
    AND NEW.extension_version = previous.extension_version
    AND NEW.package_digest = previous.package_digest
    AND (
      NEW.contract_version IS DISTINCT FROM previous.contract_version
      OR NEW.declaration_digest IS DISTINCT FROM previous.declaration_digest
    ) THEN
    RAISE EXCEPTION 'identity registry exact artifact cannot drift on reactivation';
  END IF;

  RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION validate_extension_identity_registry_publication() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  previous extension_identity_registry_publications%ROWTYPE;
BEGIN
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
