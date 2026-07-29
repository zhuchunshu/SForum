-- +goose Up
-- A stale recommendation cannot grant authority, but an authorized Host actor
-- must still be able to reject it and remove it from the pending review queue.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION validate_extension_permission_role_suggestion_decision() RETURNS trigger
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

  IF NEW.approval_state = 'approved' THEN
    -- Approval consumes live extension authority. Keep declaration publication's
    -- exact artifact -> permanent owner lock order to avoid lifecycle deadlocks.
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
  END IF;

  -- Rejection does not consume extension authority, but both decisions remain
  -- actor-bound and require live role.manage authority at commit time.
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

-- +goose Down
-- Rejected stale recommendations become valid immutable audit history. Rolling
-- back the authority rule would strand future records and reintroduce the bug.
-- +goose StatementBegin
DO $$
BEGIN
  RAISE EXCEPTION
    'cannot reverse stale role suggestion rejection repair 202607290077; roll forward with a new repair migration';
END;
$$;
-- +goose StatementEnd
