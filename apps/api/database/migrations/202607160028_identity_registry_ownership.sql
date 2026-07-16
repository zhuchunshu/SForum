-- +goose Up
-- P7 Identity/Permission ownership is durable even after a declaration is
-- disabled or uninstalled. These tables record catalog authority only. They
-- never write role_permissions or user_permission_overrides.
CREATE TABLE extension_identity_registry_owners (
  identity_kind TEXT NOT NULL
    CHECK (identity_kind IN ('permission', 'user_field', 'provider')),
  stable_id TEXT NOT NULL
    CHECK (stable_id ~ '^[a-z0-9][a-z0-9._-]{1,120}$'),
  owner_extension_id TEXT NOT NULL
    CHECK (owner_extension_id ~ '^[a-z0-9][a-z0-9._-]{1,120}$'),
  claimed_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  PRIMARY KEY (identity_kind, stable_id),
  UNIQUE (identity_kind, stable_id, owner_extension_id),
  CHECK (
    left(stable_id, length(owner_extension_id) + 1) = owner_extension_id || '.'
  ),
  CHECK (owner_extension_id !~ '^core([.]|$)')
);

CREATE INDEX extension_identity_registry_owners_extension_idx
  ON extension_identity_registry_owners (
    owner_extension_id, identity_kind, stable_id
  );

-- +goose StatementBegin
CREATE FUNCTION validate_extension_identity_registry_owner() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  stored_type TEXT;
BEGIN
  SELECT type INTO stored_type
  FROM extensions
  WHERE id = NEW.owner_extension_id
  FOR NO KEY UPDATE;

  IF stored_type IS DISTINCT FROM 'plugin' THEN
    RAISE EXCEPTION 'identity registry owner must be an installed plugin';
  END IF;
  RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER extension_identity_registry_owner_valid
BEFORE INSERT ON extension_identity_registry_owners
FOR EACH ROW EXECUTE FUNCTION validate_extension_identity_registry_owner();

-- One owner row is the permanent tombstone. It cannot be reassigned, removed,
-- or truncated; a package with a nested extension id cannot reclaim it later.
-- +goose StatementBegin
CREATE FUNCTION reject_extension_identity_registry_history_mutation() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  RAISE EXCEPTION 'extension identity registry ownership history is append-only';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER extension_identity_registry_owner_immutable
BEFORE UPDATE OR DELETE ON extension_identity_registry_owners
FOR EACH ROW EXECUTE FUNCTION reject_extension_identity_registry_history_mutation();

CREATE TRIGGER extension_identity_registry_owner_no_truncate
BEFORE TRUNCATE ON extension_identity_registry_owners
FOR EACH STATEMENT EXECUTE FUNCTION reject_extension_identity_registry_history_mutation();

-- Preserve the plugin classification for every claimed stable identity. The
-- extension row may still be deleted by uninstall; the owner tombstone remains.
-- +goose StatementBegin
CREATE FUNCTION reject_extension_identity_registry_owner_type_change() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF NEW.type IS DISTINCT FROM OLD.type AND EXISTS (
    SELECT 1
    FROM extension_identity_registry_owners
    WHERE owner_extension_id = OLD.id
  ) THEN
    RAISE EXCEPTION 'identity registry owner extension type is immutable';
  END IF;
  RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER extension_identity_registry_owner_type_immutable
BEFORE UPDATE OF type ON extensions
FOR EACH ROW EXECUTE FUNCTION reject_extension_identity_registry_owner_type_change();

-- Active/tombstone transitions are append-only events. revision is a per-id
-- CAS sequence. A tombstone must match the exact currently active artifact, so
-- a stale disable cannot retire a replacement version.
CREATE TABLE extension_identity_registry_declarations (
  identity_kind TEXT NOT NULL,
  stable_id TEXT NOT NULL,
  owner_extension_id TEXT NOT NULL,
  revision BIGINT NOT NULL CHECK (revision > 0),
  registry_state TEXT NOT NULL
    CHECK (registry_state IN ('active', 'tombstone')),
  extension_version_id BIGINT NOT NULL CHECK (extension_version_id > 0),
  extension_version TEXT NOT NULL CHECK (extension_version <> ''),
  package_digest TEXT NOT NULL
    CHECK (package_digest ~ '^[0-9a-f]{64}$'),
  contract_version TEXT NOT NULL
    CHECK (contract_version ~ '^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$'),
  declaration_digest TEXT NOT NULL
    CHECK (declaration_digest ~ '^[0-9a-f]{64}$'),
  actor_user_id BIGINT CHECK (actor_user_id IS NULL OR actor_user_id > 0),
  audit_event_id BIGINT CHECK (audit_event_id IS NULL OR audit_event_id > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  PRIMARY KEY (identity_kind, stable_id, revision),
  FOREIGN KEY (identity_kind, stable_id, owner_extension_id)
    REFERENCES extension_identity_registry_owners(
      identity_kind, stable_id, owner_extension_id
    ) ON DELETE RESTRICT
);

CREATE INDEX extension_identity_registry_declarations_current_idx
  ON extension_identity_registry_declarations (
    identity_kind, stable_id, revision DESC
  );

CREATE INDEX extension_identity_registry_declarations_artifact_idx
  ON extension_identity_registry_declarations (
    owner_extension_id, extension_version_id, package_digest, revision DESC
  );

-- +goose StatementBegin
CREATE FUNCTION validate_extension_identity_registry_declaration() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  stored_owner TEXT;
  previous extension_identity_registry_declarations%ROWTYPE;
BEGIN
  -- Lock artifact identity before its owner row. Delete guards take the same
  -- artifact/extension -> owner order, avoiding a declaration/uninstall deadlock.
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

CREATE TRIGGER extension_identity_registry_declaration_valid
BEFORE INSERT ON extension_identity_registry_declarations
FOR EACH ROW EXECUTE FUNCTION validate_extension_identity_registry_declaration();

CREATE TRIGGER extension_identity_registry_declaration_immutable
BEFORE UPDATE OR DELETE ON extension_identity_registry_declarations
FOR EACH ROW EXECUTE FUNCTION reject_extension_identity_registry_history_mutation();

CREATE TRIGGER extension_identity_registry_declaration_no_truncate
BEFORE TRUNCATE ON extension_identity_registry_declarations
FOR EACH STATEMENT EXECUTE FUNCTION reject_extension_identity_registry_history_mutation();

-- An active declaration must be retired while its exact artifact still exists.
-- Normal and forced lifecycle cleanup therefore publish tombstones before the
-- package/version purge; retained owner and declaration history survives it.
-- +goose StatementBegin
CREATE FUNCTION reject_extension_identity_registry_active_extension_delete() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  PERFORM identity_kind
  FROM extension_identity_registry_owners
  WHERE owner_extension_id = OLD.id
  ORDER BY identity_kind, stable_id
  FOR UPDATE;

  IF EXISTS (
    SELECT 1
    FROM extension_identity_registry_owners AS owners
    CROSS JOIN LATERAL (
      SELECT registry_state
      FROM extension_identity_registry_declarations AS declarations
      WHERE declarations.identity_kind = owners.identity_kind
        AND declarations.stable_id = owners.stable_id
      ORDER BY revision DESC
      LIMIT 1
    ) AS current_declaration
    WHERE owners.owner_extension_id = OLD.id
      AND current_declaration.registry_state = 'active'
  ) THEN
    RAISE EXCEPTION 'extension identity registry declarations must be tombstoned before uninstall';
  END IF;
  RETURN OLD;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER extension_identity_registry_active_extension_delete_guard
BEFORE DELETE ON extensions
FOR EACH ROW EXECUTE FUNCTION reject_extension_identity_registry_active_extension_delete();

-- +goose StatementBegin
CREATE FUNCTION reject_extension_identity_registry_active_version_delete() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  PERFORM identity_kind
  FROM extension_identity_registry_owners
  WHERE owner_extension_id = OLD.extension_id
  ORDER BY identity_kind, stable_id
  FOR UPDATE;

  IF EXISTS (
    SELECT 1
    FROM extension_identity_registry_owners AS owners
    CROSS JOIN LATERAL (
      SELECT registry_state, extension_version_id
      FROM extension_identity_registry_declarations AS declarations
      WHERE declarations.identity_kind = owners.identity_kind
        AND declarations.stable_id = owners.stable_id
      ORDER BY revision DESC
      LIMIT 1
    ) AS current_declaration
    WHERE owners.owner_extension_id = OLD.extension_id
      AND current_declaration.registry_state = 'active'
      AND current_declaration.extension_version_id = OLD.id
  ) THEN
    RAISE EXCEPTION 'active identity registry artifact cannot be removed before tombstone';
  END IF;
  RETURN OLD;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER extension_identity_registry_active_version_delete_guard
BEFORE DELETE ON extension_versions
FOR EACH ROW EXECUTE FUNCTION reject_extension_identity_registry_active_version_delete();

-- Recommendations are descriptive and start pending. An approved row records
-- Host review only; it never inserts or updates role_permissions.
CREATE TABLE extension_permission_role_suggestions (
  id BIGSERIAL PRIMARY KEY,
  identity_kind TEXT NOT NULL DEFAULT 'permission'
    CHECK (identity_kind = 'permission'),
  permission_key TEXT NOT NULL,
  owner_extension_id TEXT NOT NULL,
  extension_version_id BIGINT NOT NULL CHECK (extension_version_id > 0),
  extension_version TEXT NOT NULL CHECK (extension_version <> ''),
  package_digest TEXT NOT NULL
    CHECK (package_digest ~ '^[0-9a-f]{64}$'),
  permission_contract_version TEXT NOT NULL
    CHECK (permission_contract_version ~ '^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$'),
  declaration_digest TEXT NOT NULL
    CHECK (declaration_digest ~ '^[0-9a-f]{64}$'),
  role_key TEXT NOT NULL
    CHECK (role_key ~ '^[a-z0-9][a-z0-9._-]{1,120}$'),
  approval_state TEXT NOT NULL DEFAULT 'pending'
    CHECK (approval_state IN ('pending', 'approved', 'rejected')),
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  decided_by_user_id BIGINT
    CHECK (decided_by_user_id IS NULL OR decided_by_user_id > 0),
  decision_audit_event_id BIGINT
    CHECK (decision_audit_event_id IS NULL OR decision_audit_event_id > 0),
  decided_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  UNIQUE (
    permission_key, owner_extension_id, extension_version_id,
    package_digest, permission_contract_version, role_key
  ),
  FOREIGN KEY (identity_kind, permission_key, owner_extension_id)
    REFERENCES extension_identity_registry_owners(
      identity_kind, stable_id, owner_extension_id
    ) ON DELETE RESTRICT,
  CHECK (role_key <> 'super_admin'),
  CHECK (
    (approval_state = 'pending'
      AND revision = 1
      AND decided_by_user_id IS NULL
      AND decision_audit_event_id IS NULL
      AND decided_at IS NULL)
    OR
    (approval_state IN ('approved', 'rejected')
      AND revision = 2
      AND decided_by_user_id IS NOT NULL
      AND decision_audit_event_id IS NOT NULL
      AND decided_at IS NOT NULL)
  )
);

CREATE INDEX extension_permission_role_suggestions_state_idx
  ON extension_permission_role_suggestions (
    approval_state, role_key, created_at, id
  );

-- +goose StatementBegin
CREATE FUNCTION validate_extension_permission_role_suggestion_insert() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  active extension_identity_registry_declarations%ROWTYPE;
BEGIN
  IF NEW.approval_state <> 'pending' OR NEW.revision <> 1
    OR NEW.decided_by_user_id IS NOT NULL
    OR NEW.decision_audit_event_id IS NOT NULL
    OR NEW.decided_at IS NOT NULL THEN
    RAISE EXCEPTION 'permission role suggestion must begin pending';
  END IF;

  PERFORM 1
  FROM extension_identity_registry_owners
  WHERE identity_kind = 'permission'
    AND stable_id = NEW.permission_key
    AND owner_extension_id = NEW.owner_extension_id
  FOR UPDATE;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'permission role suggestion owner is unavailable';
  END IF;

  SELECT * INTO active
  FROM extension_identity_registry_declarations
  WHERE identity_kind = 'permission'
    AND stable_id = NEW.permission_key
  ORDER BY revision DESC
  LIMIT 1;

  IF active.revision IS NULL OR active.registry_state <> 'active'
    OR active.owner_extension_id IS DISTINCT FROM NEW.owner_extension_id
    OR active.extension_version_id IS DISTINCT FROM NEW.extension_version_id
    OR active.extension_version IS DISTINCT FROM NEW.extension_version
    OR active.package_digest IS DISTINCT FROM NEW.package_digest
    OR active.contract_version IS DISTINCT FROM NEW.permission_contract_version
    OR active.declaration_digest IS DISTINCT FROM NEW.declaration_digest THEN
    RAISE EXCEPTION 'permission role suggestion does not match the active exact declaration';
  END IF;

  RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER extension_permission_role_suggestion_insert_valid
BEFORE INSERT ON extension_permission_role_suggestions
FOR EACH ROW EXECUTE FUNCTION validate_extension_permission_role_suggestion_insert();

-- +goose StatementBegin
CREATE FUNCTION validate_extension_permission_role_suggestion_update() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  active extension_identity_registry_declarations%ROWTYPE;
BEGIN
  IF NEW.id IS DISTINCT FROM OLD.id
    OR NEW.identity_kind IS DISTINCT FROM OLD.identity_kind
    OR NEW.permission_key IS DISTINCT FROM OLD.permission_key
    OR NEW.owner_extension_id IS DISTINCT FROM OLD.owner_extension_id
    OR NEW.extension_version_id IS DISTINCT FROM OLD.extension_version_id
    OR NEW.extension_version IS DISTINCT FROM OLD.extension_version
    OR NEW.package_digest IS DISTINCT FROM OLD.package_digest
    OR NEW.permission_contract_version IS DISTINCT FROM OLD.permission_contract_version
    OR NEW.declaration_digest IS DISTINCT FROM OLD.declaration_digest
    OR NEW.role_key IS DISTINCT FROM OLD.role_key
    OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
    RAISE EXCEPTION 'permission role suggestion identity is immutable';
  END IF;

  IF OLD.approval_state <> 'pending'
    OR NEW.approval_state NOT IN ('approved', 'rejected')
    OR NEW.revision <> OLD.revision + 1
    OR NEW.decided_by_user_id IS NULL
    OR NEW.decision_audit_event_id IS NULL THEN
    RAISE EXCEPTION 'permission role suggestion decision requires Host CAS evidence';
  END IF;

  PERFORM 1
  FROM extension_identity_registry_owners
  WHERE identity_kind = 'permission'
    AND stable_id = OLD.permission_key
    AND owner_extension_id = OLD.owner_extension_id
  FOR UPDATE;

  SELECT * INTO active
  FROM extension_identity_registry_declarations
  WHERE identity_kind = 'permission'
    AND stable_id = OLD.permission_key
  ORDER BY revision DESC
  LIMIT 1;

  IF active.revision IS NULL OR active.registry_state <> 'active'
    OR active.owner_extension_id IS DISTINCT FROM OLD.owner_extension_id
    OR active.extension_version_id IS DISTINCT FROM OLD.extension_version_id
    OR active.extension_version IS DISTINCT FROM OLD.extension_version
    OR active.package_digest IS DISTINCT FROM OLD.package_digest
    OR active.contract_version IS DISTINCT FROM OLD.permission_contract_version
    OR active.declaration_digest IS DISTINCT FROM OLD.declaration_digest THEN
    RAISE EXCEPTION 'permission role suggestion decision is stale';
  END IF;

  PERFORM 1
  FROM users
  WHERE id = NEW.decided_by_user_id
    AND status = 'active'
  FOR KEY SHARE;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'permission role suggestion decision actor is not active';
  END IF;

  IF NEW.approval_state = 'approved' THEN
    PERFORM 1
    FROM roles
    WHERE key = NEW.role_key
      AND key <> 'super_admin'
    FOR KEY SHARE;

    IF NOT FOUND THEN
      RAISE EXCEPTION 'permission role suggestion approval target is unavailable';
    END IF;
  END IF;

  NEW.decided_at := statement_timestamp();
  NEW.updated_at := statement_timestamp();
  RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER extension_permission_role_suggestion_update_valid
BEFORE UPDATE ON extension_permission_role_suggestions
FOR EACH ROW EXECUTE FUNCTION validate_extension_permission_role_suggestion_update();

-- +goose StatementBegin
CREATE FUNCTION reject_extension_permission_role_suggestion_removal() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  RAISE EXCEPTION 'permission role suggestion history cannot be removed';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER extension_permission_role_suggestion_no_delete
BEFORE DELETE ON extension_permission_role_suggestions
FOR EACH ROW EXECUTE FUNCTION reject_extension_permission_role_suggestion_removal();

CREATE TRIGGER extension_permission_role_suggestion_no_truncate
BEFORE TRUNCATE ON extension_permission_role_suggestions
FOR EACH STATEMENT EXECUTE FUNCTION reject_extension_permission_role_suggestion_removal();

-- +goose Down
-- Lock before probing emptiness. Otherwise a concurrent writer can commit after
-- the probe but before DROP obtains its own lock, silently erasing new history.
LOCK TABLE extension_permission_role_suggestions,
           extension_identity_registry_declarations,
           extension_identity_registry_owners
  IN ACCESS EXCLUSIVE MODE;

-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM extension_identity_registry_owners)
    OR EXISTS (SELECT 1 FROM extension_identity_registry_declarations)
    OR EXISTS (SELECT 1 FROM extension_permission_role_suggestions) THEN
    RAISE EXCEPTION 'cannot remove extension identity registry ownership history';
  END IF;
END;
$$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS extension_permission_role_suggestion_no_truncate
  ON extension_permission_role_suggestions;
DROP TRIGGER IF EXISTS extension_permission_role_suggestion_no_delete
  ON extension_permission_role_suggestions;
DROP TRIGGER IF EXISTS extension_permission_role_suggestion_update_valid
  ON extension_permission_role_suggestions;
DROP TRIGGER IF EXISTS extension_permission_role_suggestion_insert_valid
  ON extension_permission_role_suggestions;
DROP TABLE IF EXISTS extension_permission_role_suggestions;
DROP FUNCTION IF EXISTS reject_extension_permission_role_suggestion_removal();
DROP FUNCTION IF EXISTS validate_extension_permission_role_suggestion_update();
DROP FUNCTION IF EXISTS validate_extension_permission_role_suggestion_insert();

DROP TRIGGER IF EXISTS extension_identity_registry_declaration_no_truncate
  ON extension_identity_registry_declarations;
DROP TRIGGER IF EXISTS extension_identity_registry_declaration_immutable
  ON extension_identity_registry_declarations;
DROP TRIGGER IF EXISTS extension_identity_registry_declaration_valid
  ON extension_identity_registry_declarations;
DROP TRIGGER IF EXISTS extension_identity_registry_active_version_delete_guard
  ON extension_versions;
DROP FUNCTION IF EXISTS reject_extension_identity_registry_active_version_delete();
DROP TRIGGER IF EXISTS extension_identity_registry_active_extension_delete_guard
  ON extensions;
DROP FUNCTION IF EXISTS reject_extension_identity_registry_active_extension_delete();
DROP TABLE IF EXISTS extension_identity_registry_declarations;
DROP FUNCTION IF EXISTS validate_extension_identity_registry_declaration();

DROP TRIGGER IF EXISTS extension_identity_registry_owner_type_immutable
  ON extensions;
DROP FUNCTION IF EXISTS reject_extension_identity_registry_owner_type_change();
DROP TRIGGER IF EXISTS extension_identity_registry_owner_no_truncate
  ON extension_identity_registry_owners;
DROP TRIGGER IF EXISTS extension_identity_registry_owner_immutable
  ON extension_identity_registry_owners;
DROP TRIGGER IF EXISTS extension_identity_registry_owner_valid
  ON extension_identity_registry_owners;
DROP TABLE IF EXISTS extension_identity_registry_owners;
DROP FUNCTION IF EXISTS reject_extension_identity_registry_history_mutation();
DROP FUNCTION IF EXISTS validate_extension_identity_registry_owner();
