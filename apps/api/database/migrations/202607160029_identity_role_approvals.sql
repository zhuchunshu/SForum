-- +goose Up
-- P7 Identity role suggestions remain recommendations until an active Host role
-- manager explicitly approves one. Permission catalog material is independent of
-- role suggestions: approval only consumes an existing catalog entry and adds
-- one additive role mapping plus immutable grant evidence.

-- Fence the entire upgrade before validation/backfill/trigger replacement so a
-- rolling pre-029 node cannot approve between those steps and leave an
-- inconsistent terminal row.
LOCK TABLE extension_permission_role_suggestions,
           extension_identity_registry_declarations,
           extension_identity_registry_owners,
           permissions,
           role_permissions,
           roles,
           users,
           user_roles,
           user_permission_overrides,
           audit_events
  IN ACCESS EXCLUSIVE MODE;

-- Existing terminal rows must already point at the audit evidence written by
-- the pre-029 repository. Missing or mismatched evidence cannot be repaired
-- safely because doing so would invent historical actor authority.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM extension_permission_role_suggestions AS suggestion
    LEFT JOIN audit_events AS event
      ON event.id = suggestion.decision_audit_event_id
    WHERE suggestion.approval_state <> 'pending'
      AND (
        event.id IS NULL
        OR event.actor_user_id IS DISTINCT FROM suggestion.decided_by_user_id
        OR event.action IS DISTINCT FROM CASE suggestion.approval_state
          WHEN 'approved' THEN 'identity.role_suggestion.approve'
          ELSE 'identity.role_suggestion.reject'
        END
        OR NOT event.metadata @> jsonb_build_object(
          'suggestionId', suggestion.id,
          'permissionKey', suggestion.permission_key,
          'ownerExtensionId', suggestion.owner_extension_id,
          'extensionVersionId', suggestion.extension_version_id,
          'extensionVersion', suggestion.extension_version,
          'packageDigest', suggestion.package_digest,
          'permissionContractVersion', suggestion.permission_contract_version,
          'declarationDigest', suggestion.declaration_digest,
          'roleKey', suggestion.role_key,
          'expectedRevision', suggestion.revision - 1,
          'approvalState', suggestion.approval_state
        )
      )
  ) THEN
    RAISE EXCEPTION 'cannot bind legacy identity role decisions without exact audit evidence';
  END IF;
END;
$$;
-- +goose StatementEnd

-- Every permanent permission owner must already have declaration history.
-- Backfilling a Host permission without a declaration-bound catalog row would
-- create authority that no exact artifact can inspect, approve, or tombstone.
-- +goose StatementBegin
DO $$
DECLARE
  orphan_permission_key TEXT;
BEGIN
  SELECT owner.stable_id INTO orphan_permission_key
  FROM extension_identity_registry_owners AS owner
  WHERE owner.identity_kind = 'permission'
    AND NOT EXISTS (
      SELECT 1
      FROM extension_identity_registry_declarations AS declaration
      WHERE declaration.identity_kind = owner.identity_kind
        AND declaration.stable_id = owner.stable_id
        AND declaration.owner_extension_id = owner.owner_extension_id
    )
  ORDER BY owner.stable_id
  LIMIT 1;

  IF orphan_permission_key IS NOT NULL THEN
    RAISE EXCEPTION
      'cannot backfill extension permission owner without declaration history: %',
      orphan_permission_key;
  END IF;
END;
$$;
-- +goose StatementEnd

-- Stable catalog ownership is independent of role suggestions. It binds a Host
-- permissions row to the permanent owner plus the exact durable declaration
-- that published the permission. A required suggestion or decision audit would
-- incorrectly couple catalog material to Host review.
CREATE TABLE extension_permission_catalog (
  identity_kind TEXT NOT NULL DEFAULT 'permission'
    CHECK (identity_kind = 'permission'),
  permission_key TEXT PRIMARY KEY
    REFERENCES permissions(key) ON DELETE RESTRICT,
  owner_extension_id TEXT NOT NULL,
  declaration_revision BIGINT NOT NULL CHECK (declaration_revision > 0),
  extension_version_id BIGINT NOT NULL CHECK (extension_version_id > 0),
  extension_version TEXT NOT NULL CHECK (extension_version <> ''),
  package_digest TEXT NOT NULL
    CHECK (package_digest ~ '^[0-9a-f]{64}$'),
  contract_version TEXT NOT NULL
    CHECK (contract_version ~ '^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$'),
  declaration_digest TEXT NOT NULL
    CHECK (declaration_digest ~ '^[0-9a-f]{64}$'),
  -- Optional compatibility column for the already-committed Audit cleanup path.
  -- Catalog registration never requires it; leave NULL for declaration-bound rows.
  registered_audit_event_id BIGINT
    REFERENCES audit_events(id) ON DELETE RESTRICT,
  registered_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  FOREIGN KEY (identity_kind, permission_key, owner_extension_id)
    REFERENCES extension_identity_registry_owners(
      identity_kind, stable_id, owner_extension_id
    ) ON DELETE RESTRICT,
  FOREIGN KEY (identity_kind, permission_key, declaration_revision)
    REFERENCES extension_identity_registry_declarations(
      identity_kind, stable_id, revision
    ) ON DELETE RESTRICT
);

CREATE INDEX extension_permission_catalog_owner_idx
  ON extension_permission_catalog (owner_extension_id, permission_key);

CREATE INDEX extension_permission_catalog_declaration_idx
  ON extension_permission_catalog (
    identity_kind, permission_key, declaration_revision
  );

CREATE INDEX extension_permission_catalog_audit_idx
  ON extension_permission_catalog (registered_audit_event_id)
  WHERE registered_audit_event_id IS NOT NULL;

-- +goose StatementBegin
CREATE FUNCTION validate_extension_permission_catalog() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  declaration extension_identity_registry_declarations%ROWTYPE;
BEGIN
  SELECT * INTO declaration
  FROM extension_identity_registry_declarations
  WHERE identity_kind = NEW.identity_kind
    AND stable_id = NEW.permission_key
    AND revision = NEW.declaration_revision
  FOR KEY SHARE;

  IF declaration.stable_id IS NULL
    OR declaration.owner_extension_id IS DISTINCT FROM NEW.owner_extension_id
    OR declaration.extension_version_id IS DISTINCT FROM NEW.extension_version_id
    OR declaration.extension_version IS DISTINCT FROM NEW.extension_version
    OR declaration.package_digest IS DISTINCT FROM NEW.package_digest
    OR declaration.contract_version IS DISTINCT FROM NEW.contract_version
    OR declaration.declaration_digest IS DISTINCT FROM NEW.declaration_digest THEN
    RAISE EXCEPTION 'extension permission catalog declaration is invalid';
  END IF;

  RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER extension_permission_catalog_valid
BEFORE INSERT ON extension_permission_catalog
FOR EACH ROW EXECUTE FUNCTION validate_extension_permission_catalog();

-- Catalog ownership is authority history. Plugin disable/uninstall preserves it
-- alongside role grants, so it cannot be reassigned, deleted, or truncated.
CREATE TRIGGER extension_permission_catalog_immutable
BEFORE UPDATE OR DELETE ON extension_permission_catalog
FOR EACH ROW EXECUTE FUNCTION reject_extension_identity_registry_history_mutation();

CREATE TRIGGER extension_permission_catalog_no_truncate
BEFORE TRUNCATE ON extension_permission_catalog
FOR EACH STATEMENT EXECUTE FUNCTION reject_extension_identity_registry_history_mutation();

-- Immutable evidence that an approved suggestion actually applied one additive
-- role mapping. Legacy 028 approved rows remain review-only (no grant row) until
-- an explicit Host apply with expected revision 2.
CREATE TABLE extension_permission_role_grants (
  suggestion_id BIGINT PRIMARY KEY
    REFERENCES extension_permission_role_suggestions(id) ON DELETE RESTRICT,
  permission_key TEXT NOT NULL,
  owner_extension_id TEXT NOT NULL,
  role_key TEXT NOT NULL
    CHECK (role_key ~ '^[a-z0-9][a-z0-9._-]{1,120}$')
    CHECK (role_key <> 'super_admin'),
  role_id BIGINT NOT NULL
    REFERENCES roles(id) ON DELETE RESTRICT,
  -- Historical numeric actor id; no FK so privacy erasure cannot drop evidence.
  applied_by_user_id BIGINT NOT NULL CHECK (applied_by_user_id > 0),
  applied_audit_event_id BIGINT NOT NULL
    REFERENCES audit_events(id) ON DELETE RESTRICT,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  FOREIGN KEY (permission_key)
    REFERENCES extension_permission_catalog(permission_key) ON DELETE RESTRICT
);

CREATE INDEX extension_permission_role_grants_mapping_idx
  ON extension_permission_role_grants (role_id, permission_key);

CREATE INDEX extension_permission_role_grants_audit_idx
  ON extension_permission_role_grants (applied_audit_event_id);

CREATE INDEX extension_permission_role_grants_permission_idx
  ON extension_permission_role_grants (permission_key, role_key);

-- +goose StatementBegin
CREATE FUNCTION validate_extension_permission_role_grant() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  suggestion extension_permission_role_suggestions%ROWTYPE;
  catalog extension_permission_catalog%ROWTYPE;
  event audit_events%ROWTYPE;
  stored_role_key TEXT;
  mapping_exists BOOLEAN;
BEGIN
  SELECT * INTO suggestion
  FROM extension_permission_role_suggestions
  WHERE id = NEW.suggestion_id
  FOR KEY SHARE;

  -- Pending is allowed so the same transaction can write grant evidence before
  -- the CAS decision flips the suggestion to approved. Rejected rows never grant.
  IF suggestion.id IS NULL
    OR suggestion.approval_state NOT IN ('pending', 'approved')
    OR suggestion.permission_key IS DISTINCT FROM NEW.permission_key
    OR suggestion.owner_extension_id IS DISTINCT FROM NEW.owner_extension_id
    OR suggestion.role_key IS DISTINCT FROM NEW.role_key THEN
    RAISE EXCEPTION 'extension permission role grant suggestion is invalid';
  END IF;

  SELECT * INTO catalog
  FROM extension_permission_catalog
  WHERE permission_key = NEW.permission_key
  FOR KEY SHARE;

  IF catalog.permission_key IS NULL
    OR catalog.owner_extension_id IS DISTINCT FROM NEW.owner_extension_id THEN
    RAISE EXCEPTION 'extension permission role grant catalog is unavailable';
  END IF;

  SELECT key INTO stored_role_key
  FROM roles
  WHERE id = NEW.role_id
    AND key = NEW.role_key
    AND key <> 'super_admin'
    AND is_enabled = TRUE
  FOR KEY SHARE;

  IF stored_role_key IS NULL THEN
    RAISE EXCEPTION 'extension permission role grant target is unavailable';
  END IF;

  SELECT EXISTS (
    SELECT 1
    FROM role_permissions
    WHERE role_id = NEW.role_id
      AND permission_key = NEW.permission_key
  ) INTO mapping_exists;

  IF NOT mapping_exists THEN
    RAISE EXCEPTION 'extension permission role grant additive mapping is missing';
  END IF;

  SELECT * INTO event
  FROM audit_events
  WHERE id = NEW.applied_audit_event_id
  FOR KEY SHARE;

  -- rolePermissionAdded records provenance, not success: false means the Host
  -- mapping already existed before this transaction. The grant itself must
  -- still be positively recorded by roleGrantApplied=true.
  IF event.id IS NULL
    OR event.actor_user_id IS DISTINCT FROM NEW.applied_by_user_id
    OR event.action IS DISTINCT FROM 'identity.role_suggestion.approve'
    OR NOT event.metadata @> jsonb_build_object(
      'suggestionId', suggestion.id,
      'permissionKey', suggestion.permission_key,
      'ownerExtensionId', suggestion.owner_extension_id,
      'extensionVersionId', suggestion.extension_version_id,
      'extensionVersion', suggestion.extension_version,
      'packageDigest', suggestion.package_digest,
      'permissionContractVersion', suggestion.permission_contract_version,
      'declarationDigest', suggestion.declaration_digest,
      'roleKey', suggestion.role_key,
      'approvalState', 'approved'
    )
    OR jsonb_typeof(event.metadata -> 'rolePermissionAdded') IS DISTINCT FROM 'boolean'
    OR event.metadata -> 'roleGrantApplied' IS DISTINCT FROM 'true'::jsonb THEN
    RAISE EXCEPTION 'extension permission role grant audit evidence is invalid';
  END IF;

  RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER extension_permission_role_grant_valid
BEFORE INSERT ON extension_permission_role_grants
FOR EACH ROW EXECUTE FUNCTION validate_extension_permission_role_grant();

CREATE TRIGGER extension_permission_role_grant_immutable
BEFORE UPDATE OR DELETE ON extension_permission_role_grants
FOR EACH ROW EXECUTE FUNCTION reject_extension_identity_registry_history_mutation();

CREATE TRIGGER extension_permission_role_grant_no_truncate
BEFORE TRUNCATE ON extension_permission_role_grants
FOR EACH STATEMENT EXECUTE FUNCTION reject_extension_identity_registry_history_mutation();

-- Backfill generic Host permissions and catalog rows for every extension-owned
-- permission declaration. Fail closed when an untracked Host permission already
-- occupies the key: inventing ownership would claim Core/legacy authority.
-- +goose StatementBegin
DO $$
DECLARE
  collision_key TEXT;
BEGIN
  SELECT permission.key INTO collision_key
  FROM extension_identity_registry_owners AS owner
  JOIN permissions AS permission
    ON permission.key = owner.stable_id
  WHERE owner.identity_kind = 'permission'
  LIMIT 1;

  IF collision_key IS NOT NULL THEN
    RAISE EXCEPTION
      'cannot claim untracked Host permission for extension catalog: %',
      collision_key;
  END IF;

  INSERT INTO permissions (key, module, description)
  SELECT owner.stable_id, 'extension', ''
  FROM extension_identity_registry_owners AS owner
  WHERE owner.identity_kind = 'permission'
  ON CONFLICT (key) DO NOTHING;

  INSERT INTO extension_permission_catalog (
    permission_key,
    owner_extension_id,
    declaration_revision,
    extension_version_id,
    extension_version,
    package_digest,
    contract_version,
    declaration_digest
  )
  SELECT DISTINCT ON (declaration.stable_id)
    declaration.stable_id,
    declaration.owner_extension_id,
    declaration.revision,
    declaration.extension_version_id,
    declaration.extension_version,
    declaration.package_digest,
    declaration.contract_version,
    declaration.declaration_digest
  FROM extension_identity_registry_declarations AS declaration
  WHERE declaration.identity_kind = 'permission'
  ORDER BY declaration.stable_id ASC, declaration.revision DESC;
END;
$$;
-- +goose StatementEnd

ALTER TABLE extension_permission_role_suggestions
  ADD CONSTRAINT extension_permission_role_suggestions_decision_audit_fk
  FOREIGN KEY (decision_audit_event_id)
  REFERENCES audit_events(id) ON DELETE RESTRICT;

CREATE INDEX extension_permission_role_suggestions_decision_audit_idx
  ON extension_permission_role_suggestions (decision_audit_event_id)
  WHERE decision_audit_event_id IS NOT NULL;

-- Effective role.manage is checked again in PostgreSQL. This closes stale actor
-- snapshots and direct repository calls while preserving super_admin's existing
-- non-deniable policy.
-- +goose StatementBegin
CREATE FUNCTION extension_identity_actor_can_manage_roles(candidate_user_id BIGINT)
RETURNS BOOLEAN
LANGUAGE sql
STABLE
AS $$
  SELECT EXISTS (
    SELECT 1
    FROM users AS actor
    WHERE actor.id = candidate_user_id
      AND actor.status = 'active'
      AND (
        EXISTS (
          SELECT 1
          FROM user_roles
          JOIN roles ON roles.id = user_roles.role_id
          WHERE user_roles.user_id = actor.id
            AND roles.key = 'super_admin'
            AND roles.is_enabled = TRUE
        )
        OR (
          NOT EXISTS (
            SELECT 1
            FROM user_permission_overrides
            WHERE user_id = actor.id
              AND permission_key = 'role.manage'
              AND effect = 'deny'
          )
          AND (
            EXISTS (
              SELECT 1
              FROM user_permission_overrides
              WHERE user_id = actor.id
                AND permission_key = 'role.manage'
                AND effect = 'allow'
            )
            OR EXISTS (
              SELECT 1
              FROM user_roles
              JOIN roles ON roles.id = user_roles.role_id
              JOIN role_permissions ON role_permissions.role_id = roles.id
              WHERE user_roles.user_id = actor.id
                AND roles.is_enabled = TRUE
                AND role_permissions.permission_key = 'role.manage'
            )
          )
        )
      )
  );
$$;
-- +goose StatementEnd

-- Decision and grant audit rows authorize durable authority. Actor/target may
-- follow users ON DELETE SET NULL for privacy, while action/metadata/created_at
-- and the historical numeric ids on authority tables remain immutable.
-- +goose StatementBegin
CREATE FUNCTION reject_extension_identity_decision_audit_mutation() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM extension_permission_role_suggestions
    WHERE decision_audit_event_id = OLD.id
  ) OR EXISTS (
    SELECT 1
    FROM extension_permission_role_grants
    WHERE applied_audit_event_id = OLD.id
  ) OR EXISTS (
    SELECT 1
    FROM extension_permission_catalog
    WHERE registered_audit_event_id = OLD.id
  ) THEN
    IF TG_OP = 'DELETE'
      OR NEW.id IS DISTINCT FROM OLD.id
      OR NEW.action IS DISTINCT FROM OLD.action
      OR NEW.metadata IS DISTINCT FROM OLD.metadata
      OR NEW.created_at IS DISTINCT FROM OLD.created_at
      OR (
        NEW.actor_user_id IS DISTINCT FROM OLD.actor_user_id
        AND NOT (OLD.actor_user_id IS NOT NULL AND NEW.actor_user_id IS NULL)
      )
      OR (
        NEW.target_user_id IS DISTINCT FROM OLD.target_user_id
        AND NOT (OLD.target_user_id IS NOT NULL AND NEW.target_user_id IS NULL)
      ) THEN
      RAISE EXCEPTION 'identity role decision audit evidence is immutable';
    END IF;
  END IF;
  RETURN COALESCE(NEW, OLD);
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER extension_identity_decision_audit_immutable
BEFORE UPDATE OR DELETE ON audit_events
FOR EACH ROW EXECUTE FUNCTION reject_extension_identity_decision_audit_mutation();

-- Cleanup scans expired audit_events by created_at. The composite index keeps
-- retention walks from sorting the full heap as authority evidence grows.
CREATE INDEX IF NOT EXISTS audit_events_created_at_id_idx
  ON audit_events (created_at, id);

-- Replace only the trigger binding. The 028 validation function remains intact
-- so an empty Down migration can restore the old compatibility behavior.
DROP TRIGGER extension_permission_role_suggestion_update_valid
  ON extension_permission_role_suggestions;

-- +goose StatementBegin
CREATE FUNCTION validate_extension_permission_role_suggestion_decision() RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  active extension_identity_registry_declarations%ROWTYPE;
  actor_status TEXT;
  event audit_events%ROWTYPE;
  expected_action TEXT;
  target_role_id BIGINT;
  grant_exists BOOLEAN;
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

  -- Match the lock order used by declaration publication: exact artifact first,
  -- then the permanent owner. This prevents approval/disable deadlocks.
  PERFORM 1
  FROM extension_versions AS version
  JOIN extensions AS extension ON extension.id = version.extension_id
  WHERE version.id = OLD.extension_version_id
    AND version.extension_id = OLD.owner_extension_id
    AND version.version = OLD.extension_version
    AND version.package_digest = OLD.package_digest
    AND extension.type = 'plugin'
    AND extension.status = 'enabled'
    AND extension.active_version_id = version.id
  FOR NO KEY UPDATE OF version, extension;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'permission role suggestion decision exact artifact is inactive';
  END IF;

  PERFORM 1
  FROM extension_identity_registry_owners
  WHERE identity_kind = 'permission'
    AND stable_id = OLD.permission_key
    AND owner_extension_id = OLD.owner_extension_id
  FOR UPDATE;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'permission role suggestion owner is unavailable';
  END IF;

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

  -- KEY SHARE only: SERIALIZABLE predicate reads recheck authority. FOR UPDATE
  -- on the actor before role_permissions deadlocks with role replacement.
  SELECT status INTO actor_status
  FROM users
  WHERE id = NEW.decided_by_user_id
  FOR KEY SHARE;

  IF actor_status IS DISTINCT FROM 'active'
    OR NOT extension_identity_actor_can_manage_roles(NEW.decided_by_user_id) THEN
    RAISE EXCEPTION 'permission role suggestion decision actor lacks role.manage';
  END IF;

  expected_action := CASE NEW.approval_state
    WHEN 'approved' THEN 'identity.role_suggestion.approve'
    ELSE 'identity.role_suggestion.reject'
  END;

  SELECT * INTO event
  FROM audit_events
  WHERE id = NEW.decision_audit_event_id
  FOR KEY SHARE;

  IF event.id IS NULL
    OR event.actor_user_id IS DISTINCT FROM NEW.decided_by_user_id
    OR event.action IS DISTINCT FROM expected_action
    OR NOT event.metadata @> jsonb_build_object(
      'suggestionId', OLD.id,
      'permissionKey', OLD.permission_key,
      'ownerExtensionId', OLD.owner_extension_id,
      'extensionVersionId', OLD.extension_version_id,
      'extensionVersion', OLD.extension_version,
      'packageDigest', OLD.package_digest,
      'permissionContractVersion', OLD.permission_contract_version,
      'declarationDigest', OLD.declaration_digest,
      'roleKey', OLD.role_key,
      'expectedRevision', OLD.revision,
      'approvalState', NEW.approval_state
    )
    OR jsonb_typeof(event.metadata -> 'rolePermissionAdded') IS DISTINCT FROM 'boolean'
    OR jsonb_typeof(event.metadata -> 'roleGrantApplied') IS DISTINCT FROM 'boolean' THEN
    RAISE EXCEPTION 'permission role suggestion decision audit evidence is invalid';
  END IF;

  IF NEW.approval_state = 'approved' THEN
    SELECT id INTO target_role_id
    FROM roles
    WHERE key = NEW.role_key
      AND key <> 'super_admin'
      AND is_enabled = TRUE
    FOR NO KEY UPDATE;

    IF target_role_id IS NULL THEN
      RAISE EXCEPTION 'permission role suggestion approval target is unavailable';
    END IF;

    PERFORM 1
    FROM permissions AS permission
    JOIN extension_permission_catalog AS catalog
      ON catalog.permission_key = permission.key
    WHERE permission.key = NEW.permission_key
      AND catalog.owner_extension_id = NEW.owner_extension_id
    FOR KEY SHARE OF permission, catalog;

    IF NOT FOUND THEN
      RAISE EXCEPTION 'permission role suggestion Host catalog is unavailable';
    END IF;

    PERFORM 1
    FROM role_permissions
    WHERE role_id = target_role_id
      AND permission_key = NEW.permission_key
    FOR KEY SHARE;

    IF NOT FOUND THEN
      RAISE EXCEPTION 'permission role suggestion additive grant is missing';
    END IF;

    SELECT EXISTS (
      SELECT 1
      FROM extension_permission_role_grants
      WHERE suggestion_id = NEW.id
        AND permission_key = NEW.permission_key
        AND role_key = NEW.role_key
        AND applied_by_user_id = NEW.decided_by_user_id
        AND applied_audit_event_id = NEW.decision_audit_event_id
    ) INTO grant_exists;

    IF NOT grant_exists THEN
      RAISE EXCEPTION 'permission role suggestion grant evidence is missing';
    END IF;

    -- A false rolePermissionAdded is valid when the checked Host mapping
    -- predates this transaction; roleGrantApplied remains positive evidence.
    IF event.metadata -> 'roleGrantApplied' IS DISTINCT FROM 'true'::jsonb THEN
      RAISE EXCEPTION 'permission role suggestion approval audit omits grant evidence';
    END IF;
  ELSE
    IF event.metadata -> 'rolePermissionAdded' IS DISTINCT FROM 'false'::jsonb
      OR event.metadata -> 'roleGrantApplied' IS DISTINCT FROM 'false'::jsonb THEN
      RAISE EXCEPTION 'permission role suggestion rejection audit claims a grant';
    END IF;

    IF EXISTS (
      SELECT 1
      FROM extension_permission_role_grants
      WHERE suggestion_id = NEW.id
    ) THEN
      RAISE EXCEPTION 'permission role suggestion rejection cannot carry grant evidence';
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
FOR EACH ROW EXECUTE FUNCTION validate_extension_permission_role_suggestion_decision();

-- +goose Down
LOCK TABLE extension_permission_role_grants,
           extension_permission_catalog,
           extension_permission_role_suggestions,
           audit_events
  IN ACCESS EXCLUSIVE MODE;

-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM extension_permission_role_grants)
    OR EXISTS (SELECT 1 FROM extension_permission_catalog)
    OR EXISTS (
      SELECT 1
      FROM extension_permission_role_suggestions
      WHERE approval_state <> 'pending'
    ) THEN
    RAISE EXCEPTION 'cannot remove identity role approval authority history';
  END IF;
END;
$$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS extension_permission_role_suggestion_update_valid
  ON extension_permission_role_suggestions;
CREATE TRIGGER extension_permission_role_suggestion_update_valid
BEFORE UPDATE ON extension_permission_role_suggestions
FOR EACH ROW EXECUTE FUNCTION validate_extension_permission_role_suggestion_update();
DROP FUNCTION IF EXISTS validate_extension_permission_role_suggestion_decision();

DROP TRIGGER IF EXISTS extension_identity_decision_audit_immutable
  ON audit_events;
DROP FUNCTION IF EXISTS reject_extension_identity_decision_audit_mutation();

ALTER TABLE extension_permission_role_suggestions
  DROP CONSTRAINT IF EXISTS extension_permission_role_suggestions_decision_audit_fk;
DROP INDEX IF EXISTS extension_permission_role_suggestions_decision_audit_idx;

DROP TRIGGER IF EXISTS extension_permission_role_grant_no_truncate
  ON extension_permission_role_grants;
DROP TRIGGER IF EXISTS extension_permission_role_grant_immutable
  ON extension_permission_role_grants;
DROP TRIGGER IF EXISTS extension_permission_role_grant_valid
  ON extension_permission_role_grants;
DROP INDEX IF EXISTS extension_permission_role_grants_permission_idx;
DROP INDEX IF EXISTS extension_permission_role_grants_audit_idx;
DROP INDEX IF EXISTS extension_permission_role_grants_mapping_idx;
DROP TABLE IF EXISTS extension_permission_role_grants;
DROP FUNCTION IF EXISTS validate_extension_permission_role_grant();

DROP TRIGGER IF EXISTS extension_permission_catalog_no_truncate
  ON extension_permission_catalog;
DROP TRIGGER IF EXISTS extension_permission_catalog_immutable
  ON extension_permission_catalog;
DROP TRIGGER IF EXISTS extension_permission_catalog_valid
  ON extension_permission_catalog;
DROP INDEX IF EXISTS extension_permission_catalog_audit_idx;
DROP INDEX IF EXISTS extension_permission_catalog_declaration_idx;
DROP INDEX IF EXISTS extension_permission_catalog_owner_idx;
DROP TABLE IF EXISTS extension_permission_catalog;
DROP FUNCTION IF EXISTS validate_extension_permission_catalog();
DROP FUNCTION IF EXISTS extension_identity_actor_can_manage_roles(BIGINT);

-- The cleanup index is shared operational infrastructure and is intentionally
-- left in place on Down; removing it would not restore authority history.
