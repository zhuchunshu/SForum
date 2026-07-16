-- +goose Up
-- Repair databases that applied a pre-commit draft of 202607160029 under the
-- same Goose version. Fresh installs that already have the current declaration-
-- bound catalog and grants table take the idempotent path (semantic no-op).
-- Never invent actor, artifact, contract, or digest authority: catalog rows
-- backfill only from exact registered-suggestion + unique declaration evidence.
-- Rollback is forward repair only; see Down.
-- This file intentionally exceeds 1000 lines: schema conversion and the exact
-- current-029 authority functions/triggers must replace the draft atomically.
-- Splitting them would expose rolling nodes to a mixed authorization contract.

-- Rolling nodes may still publish declarations or decide role suggestions.
-- Acquire a deterministic repair fence with NOWAIT inside a subtransaction:
-- a partial lock set is released before retry, so migration
-- fencing cannot deadlock with a transaction that already owns a later table.
-- +goose StatementBegin
DO $$
DECLARE
  lock_targets TEXT[] := ARRAY[
    'extensions',
    'extension_versions',
    'extension_identity_registry_publications',
    'extension_identity_registry_owners',
    'extension_identity_registry_declarations',
    'permissions',
    'extension_permission_catalog',
    'extension_permission_role_suggestions',
    'users',
    'user_roles',
    'user_permission_overrides',
    'roles',
    'role_permissions',
    'audit_events',
    'extension_permission_role_grants'
  ];
  rel TEXT;
  fq TEXT;
  retry_deadline TIMESTAMPTZ := clock_timestamp() + INTERVAL '30 seconds';
BEGIN
  LOOP
    BEGIN
      FOREACH rel IN ARRAY lock_targets LOOP
        fq := format('%I.%I', current_schema(), rel);
        IF to_regclass(fq) IS NOT NULL THEN
          EXECUTE format('LOCK TABLE %s IN ACCESS EXCLUSIVE MODE NOWAIT', fq);
        END IF;
      END LOOP;
      EXIT;
    EXCEPTION
      WHEN lock_not_available THEN
        IF clock_timestamp() >= retry_deadline THEN
          RAISE EXCEPTION
            'timed out acquiring identity role approval repair fence'
            USING ERRCODE = '55P03';
        END IF;
    END;
    PERFORM pg_sleep(0.05);
  END LOOP;
END;
$$;
-- +goose StatementEnd

-- Existing terminal suggestion rows must already point at exact audit evidence.
-- Missing or mismatched evidence cannot be repaired without inventing history.
-- +goose StatementBegin
DO $$
BEGIN
  IF to_regclass(format('%I.extension_permission_role_suggestions', current_schema())) IS NULL THEN
    RAISE EXCEPTION
      'extension_permission_role_suggestions missing; apply 202607160028/029 before 202607170034';
  END IF;

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

-- Repair drifted suggestion-bound catalog rows into declaration-bound authority.
-- +goose StatementBegin
DO $$
DECLARE
  catalog_reg regclass;
  has_declaration_revision BOOLEAN;
  has_registered_suggestion BOOLEAN;
  unproven_key TEXT;
BEGIN
  catalog_reg := to_regclass(format('%I.extension_permission_catalog', current_schema()));
  IF catalog_reg IS NULL THEN
    RAISE EXCEPTION
      'extension_permission_catalog missing; apply 202607160029 before 202607170034';
  END IF;

  SELECT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = current_schema()
      AND table_name = 'extension_permission_catalog'
      AND column_name = 'declaration_revision'
  ) INTO has_declaration_revision;

  SELECT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = current_schema()
      AND table_name = 'extension_permission_catalog'
      AND column_name = 'registered_suggestion_id'
  ) INTO has_registered_suggestion;

  -- Current declaration-bound catalog with no legacy repair work: leave rows
  -- untouched. Column/constraint ensure steps below still run idempotently.
  IF has_declaration_revision AND NOT has_registered_suggestion THEN
    RETURN;
  END IF;

  -- Allow exact backfill of authority history while repair runs.
  DROP TRIGGER IF EXISTS extension_permission_catalog_immutable
    ON extension_permission_catalog;
  DROP TRIGGER IF EXISTS extension_permission_catalog_valid
    ON extension_permission_catalog;
  DROP TRIGGER IF EXISTS extension_permission_catalog_no_truncate
    ON extension_permission_catalog;

  IF NOT has_declaration_revision THEN
    ALTER TABLE extension_permission_catalog
      ADD COLUMN declaration_revision BIGINT,
      ADD COLUMN extension_version_id BIGINT,
      ADD COLUMN extension_version TEXT,
      ADD COLUMN package_digest TEXT,
      ADD COLUMN contract_version TEXT,
      ADD COLUMN declaration_digest TEXT;
  END IF;

  -- Legacy suggestion-bound columns become optional compatibility evidence so
  -- current declaration-bound inserts need no suggestion.
  IF has_registered_suggestion THEN
    ALTER TABLE extension_permission_catalog
      ALTER COLUMN registered_suggestion_id DROP NOT NULL;
  END IF;

  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = current_schema()
      AND table_name = 'extension_permission_catalog'
      AND column_name = 'registered_by_user_id'
  ) THEN
    ALTER TABLE extension_permission_catalog
      ALTER COLUMN registered_by_user_id DROP NOT NULL;
  END IF;

  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = current_schema()
      AND table_name = 'extension_permission_catalog'
      AND column_name = 'registered_audit_event_id'
  ) THEN
    ALTER TABLE extension_permission_catalog
      ALTER COLUMN registered_audit_event_id DROP NOT NULL;
  END IF;

  -- Only rows that still lack declaration binding need proof. Bind the catalog
  -- to the declaration tip that was current when its exact suggestion was
  -- created. Later tombstone/reactivation history must not make valid evidence
  -- ambiguous or retroactively change authority.
  IF has_registered_suggestion THEN
    SELECT catalog.permission_key INTO unproven_key
    FROM extension_permission_catalog AS catalog
    LEFT JOIN extension_permission_role_suggestions AS suggestion
      ON suggestion.id = catalog.registered_suggestion_id
     AND suggestion.permission_key = catalog.permission_key
     AND suggestion.owner_extension_id = catalog.owner_extension_id
    LEFT JOIN LATERAL (
      SELECT candidate.*
      FROM extension_identity_registry_declarations AS candidate
      WHERE candidate.identity_kind = catalog.identity_kind
        AND candidate.stable_id = catalog.permission_key
        AND candidate.created_at <= suggestion.created_at
      ORDER BY candidate.revision DESC
      LIMIT 1
    ) AS declaration ON TRUE
    WHERE catalog.declaration_revision IS NULL
      AND (
        suggestion.id IS NULL
        OR declaration.revision IS NULL
        OR declaration.registry_state <> 'active'
        OR declaration.owner_extension_id IS DISTINCT FROM catalog.owner_extension_id
        OR declaration.extension_version_id IS DISTINCT FROM suggestion.extension_version_id
        OR declaration.extension_version IS DISTINCT FROM suggestion.extension_version
        OR declaration.package_digest IS DISTINCT FROM suggestion.package_digest
        OR declaration.contract_version IS DISTINCT FROM suggestion.permission_contract_version
        OR declaration.declaration_digest IS DISTINCT FROM suggestion.declaration_digest
      )
    ORDER BY catalog.permission_key
    LIMIT 1;

    IF unproven_key IS NOT NULL THEN
      RAISE EXCEPTION
        'cannot repair extension permission catalog without exact suggestion/declaration evidence: %',
        unproven_key;
    END IF;

    WITH bindings AS (
      SELECT catalog.permission_key, declaration.revision,
             declaration.extension_version_id, declaration.extension_version,
             declaration.package_digest, declaration.contract_version,
             declaration.declaration_digest
      FROM extension_permission_catalog AS catalog
      JOIN extension_permission_role_suggestions AS suggestion
        ON suggestion.id = catalog.registered_suggestion_id
       AND suggestion.permission_key = catalog.permission_key
       AND suggestion.owner_extension_id = catalog.owner_extension_id
      JOIN LATERAL (
        SELECT candidate.*
        FROM extension_identity_registry_declarations AS candidate
        WHERE candidate.identity_kind = catalog.identity_kind
          AND candidate.stable_id = catalog.permission_key
          AND candidate.created_at <= suggestion.created_at
        ORDER BY candidate.revision DESC
        LIMIT 1
      ) AS declaration
        ON declaration.registry_state = 'active'
       AND declaration.owner_extension_id = catalog.owner_extension_id
       AND declaration.extension_version_id = suggestion.extension_version_id
       AND declaration.extension_version = suggestion.extension_version
       AND declaration.package_digest = suggestion.package_digest
       AND declaration.contract_version = suggestion.permission_contract_version
       AND declaration.declaration_digest = suggestion.declaration_digest
      WHERE catalog.declaration_revision IS NULL
    )
    UPDATE extension_permission_catalog AS catalog
    SET declaration_revision = bindings.revision,
        extension_version_id = bindings.extension_version_id,
        extension_version = bindings.extension_version,
        package_digest = bindings.package_digest,
        contract_version = bindings.contract_version,
        declaration_digest = bindings.declaration_digest
    FROM bindings
    WHERE catalog.permission_key = bindings.permission_key
      AND catalog.declaration_revision IS NULL;
  END IF;

  -- Fail closed if any catalog row still lacks exact declaration binding.
  SELECT permission_key INTO unproven_key
  FROM extension_permission_catalog
  WHERE declaration_revision IS NULL
     OR extension_version_id IS NULL
     OR extension_version IS NULL
     OR package_digest IS NULL
     OR contract_version IS NULL
     OR declaration_digest IS NULL
  ORDER BY permission_key
  LIMIT 1;

  IF unproven_key IS NOT NULL THEN
    RAISE EXCEPTION
      'cannot enforce declaration-bound extension permission catalog without proof: %',
      unproven_key;
  END IF;

  ALTER TABLE extension_permission_catalog
    ALTER COLUMN declaration_revision SET NOT NULL,
    ALTER COLUMN extension_version_id SET NOT NULL,
    ALTER COLUMN extension_version SET NOT NULL,
    ALTER COLUMN package_digest SET NOT NULL,
    ALTER COLUMN contract_version SET NOT NULL,
    ALTER COLUMN declaration_digest SET NOT NULL;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conrelid = catalog_reg
      AND conname = 'extension_permission_catalog_declaration_revision_check'
  ) THEN
    ALTER TABLE extension_permission_catalog
      ADD CONSTRAINT extension_permission_catalog_declaration_revision_check
      CHECK (declaration_revision > 0);
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conrelid = catalog_reg
      AND conname = 'extension_permission_catalog_extension_version_id_check'
  ) THEN
    ALTER TABLE extension_permission_catalog
      ADD CONSTRAINT extension_permission_catalog_extension_version_id_check
      CHECK (extension_version_id > 0);
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conrelid = catalog_reg
      AND conname = 'extension_permission_catalog_extension_version_check'
  ) THEN
    ALTER TABLE extension_permission_catalog
      ADD CONSTRAINT extension_permission_catalog_extension_version_check
      CHECK (extension_version <> '');
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conrelid = catalog_reg
      AND conname = 'extension_permission_catalog_package_digest_check'
  ) THEN
    ALTER TABLE extension_permission_catalog
      ADD CONSTRAINT extension_permission_catalog_package_digest_check
      CHECK (package_digest ~ '^[0-9a-f]{64}$');
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conrelid = catalog_reg
      AND conname = 'extension_permission_catalog_contract_version_check'
  ) THEN
    ALTER TABLE extension_permission_catalog
      ADD CONSTRAINT extension_permission_catalog_contract_version_check
      CHECK (contract_version ~ '^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$');
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conrelid = catalog_reg
      AND conname = 'extension_permission_catalog_declaration_digest_check'
  ) THEN
    ALTER TABLE extension_permission_catalog
      ADD CONSTRAINT extension_permission_catalog_declaration_digest_check
      CHECK (declaration_digest ~ '^[0-9a-f]{64}$');
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conrelid = catalog_reg
      AND conname = 'extension_permission_catalog_declaration_fk'
  ) THEN
    ALTER TABLE extension_permission_catalog
      ADD CONSTRAINT extension_permission_catalog_declaration_fk
      FOREIGN KEY (identity_kind, permission_key, declaration_revision)
      REFERENCES extension_identity_registry_declarations(
        identity_kind, stable_id, revision
      ) ON DELETE RESTRICT;
  END IF;

  CREATE INDEX IF NOT EXISTS extension_permission_catalog_owner_idx
    ON extension_permission_catalog (owner_extension_id, permission_key);

  CREATE INDEX IF NOT EXISTS extension_permission_catalog_declaration_idx
    ON extension_permission_catalog (
      identity_kind, permission_key, declaration_revision
    );

  CREATE INDEX IF NOT EXISTS extension_permission_catalog_audit_idx
    ON extension_permission_catalog (registered_audit_event_id)
    WHERE registered_audit_event_id IS NOT NULL;
END;
$$;
-- +goose StatementEnd

-- Grant evidence table is required by current decision/grant triggers. Never
-- synthesize grant rows for legacy approved suggestions (review-only). An
-- unknown pre-existing shape is rejected before CREATE IF NOT EXISTS can hide
-- it; missing authority columns cannot be backfilled without invented history.
-- +goose StatementBegin
DO $$
DECLARE
  grants_reg regclass;
  column_mismatch BOOLEAN;
  constraint_mismatch BOOLEAN;
  index_mismatch BOOLEAN;
BEGIN
  grants_reg := to_regclass(format('%I.extension_permission_role_grants', current_schema()));
  IF grants_reg IS NULL THEN
    RETURN;
  END IF;

  WITH expected(name, data_type, nullable, default_value) AS (
    VALUES
      ('suggestion_id', 'bigint', 'NO', NULL::TEXT),
      ('permission_key', 'text', 'NO', NULL::TEXT),
      ('owner_extension_id', 'text', 'NO', NULL::TEXT),
      ('role_key', 'text', 'NO', NULL::TEXT),
      ('role_id', 'bigint', 'NO', NULL::TEXT),
      ('applied_by_user_id', 'bigint', 'NO', NULL::TEXT),
      ('applied_audit_event_id', 'bigint', 'NO', NULL::TEXT),
      ('applied_at', 'timestamp with time zone', 'NO', 'statement_timestamp()')
  )
  SELECT (
    SELECT count(*) <> 8
    FROM information_schema.columns
    WHERE table_schema = current_schema()
      AND table_name = 'extension_permission_role_grants'
  ) OR count(*) FILTER (
    WHERE column_info.column_name IS NULL
      OR column_info.data_type IS DISTINCT FROM expected.data_type
      OR column_info.is_nullable IS DISTINCT FROM expected.nullable
      OR column_info.column_default IS DISTINCT FROM expected.default_value
  ) > 0
  INTO column_mismatch
  FROM expected
  LEFT JOIN information_schema.columns AS column_info
    ON column_info.table_schema = current_schema()
   AND column_info.table_name = 'extension_permission_role_grants'
   AND column_info.column_name = expected.name;

  SELECT count(*) <> 8
    OR count(*) FILTER (WHERE definition = 'PRIMARY KEY (suggestion_id)') <> 1
    OR count(*) FILTER (WHERE definition =
      'FOREIGN KEY (suggestion_id) REFERENCES extension_permission_role_suggestions(id) ON DELETE RESTRICT') <> 1
    OR count(*) FILTER (WHERE definition =
      'FOREIGN KEY (permission_key) REFERENCES extension_permission_catalog(permission_key) ON DELETE RESTRICT') <> 1
    OR count(*) FILTER (WHERE definition =
      'FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE RESTRICT') <> 1
    OR count(*) FILTER (WHERE definition =
      'FOREIGN KEY (applied_audit_event_id) REFERENCES audit_events(id) ON DELETE RESTRICT') <> 1
    OR count(*) FILTER (WHERE definition = 'CHECK ((applied_by_user_id > 0))') <> 1
    OR count(*) FILTER (WHERE definition =
      'CHECK ((role_key ~ ''^[a-z0-9][a-z0-9._-]{1,120}$''::text))') <> 1
    OR count(*) FILTER (WHERE definition =
      'CHECK ((role_key <> ''super_admin''::text))') <> 1
  INTO constraint_mismatch
  FROM (
    SELECT pg_get_constraintdef(oid) AS definition
    FROM pg_constraint
    WHERE conrelid = grants_reg
  ) AS constraints;

  WITH actual AS (
    SELECT index_class.relname AS name,
           index_info.indisunique AS unique_index,
           index_info.indisprimary AS primary_index,
           index_info.indisvalid AS valid_index,
           index_info.indpred IS NULL AS no_predicate,
           index_info.indexprs IS NULL AS no_expressions,
           index_info.indnatts = index_info.indnkeyatts AS no_include,
           ARRAY(
             SELECT attribute.attname
             FROM unnest(index_info.indkey) WITH ORDINALITY AS key(attnum, position)
             JOIN pg_attribute AS attribute
               ON attribute.attrelid = index_info.indrelid
              AND attribute.attnum = key.attnum
             WHERE key.position <= index_info.indnkeyatts
             ORDER BY key.position
           ) AS key_columns
    FROM pg_index AS index_info
    JOIN pg_class AS index_class ON index_class.oid = index_info.indexrelid
    WHERE index_info.indrelid = grants_reg
  ), expected(name, unique_index, primary_index, key_columns) AS (
    VALUES
      ('extension_permission_role_grants_pkey', TRUE, TRUE, ARRAY['suggestion_id']::NAME[]),
      ('extension_permission_role_grants_mapping_idx', FALSE, FALSE, ARRAY['role_id', 'permission_key']::NAME[]),
      ('extension_permission_role_grants_audit_idx', FALSE, FALSE, ARRAY['applied_audit_event_id']::NAME[]),
      ('extension_permission_role_grants_permission_idx', FALSE, FALSE, ARRAY['permission_key', 'role_key']::NAME[])
  )
  SELECT (SELECT count(*) <> 4 FROM actual)
    OR count(*) FILTER (
      WHERE actual.name IS NULL
        OR actual.unique_index IS DISTINCT FROM expected.unique_index
        OR actual.primary_index IS DISTINCT FROM expected.primary_index
        OR NOT actual.valid_index
        OR NOT actual.no_predicate
        OR NOT actual.no_expressions
        OR NOT actual.no_include
        OR actual.key_columns IS DISTINCT FROM expected.key_columns
    ) > 0
  INTO index_mismatch
  FROM expected
  LEFT JOIN actual ON actual.name = expected.name;

  IF column_mismatch OR constraint_mismatch OR index_mismatch THEN
    RAISE EXCEPTION
      'existing extension_permission_role_grants schema is incompatible with migration 202607170034';
  END IF;
END;
$$;
-- +goose StatementEnd

CREATE TABLE IF NOT EXISTS extension_permission_role_grants (
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

CREATE INDEX IF NOT EXISTS extension_permission_role_grants_mapping_idx
  ON extension_permission_role_grants (role_id, permission_key);

CREATE INDEX IF NOT EXISTS extension_permission_role_grants_audit_idx
  ON extension_permission_role_grants (applied_audit_event_id);

CREATE INDEX IF NOT EXISTS extension_permission_role_grants_permission_idx
  ON extension_permission_role_grants (permission_key, role_key);

-- Current declaration-bound catalog validation (must match 029 source).
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION validate_extension_permission_catalog() RETURNS trigger
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

-- Current grant validation (must match 029 source).
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION validate_extension_permission_role_grant() RETURNS trigger
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

-- Effective role.manage policy used by decision triggers (must match 029).
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION extension_identity_actor_can_manage_roles(candidate_user_id BIGINT)
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

-- Decision/grant/catalog audit immutability with privacy SET NULL allowance.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION reject_extension_identity_decision_audit_mutation() RETURNS trigger
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

-- Current suggestion decision authority (must match 029 source).
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

-- Rebind triggers to the current function bodies without duplicating objects.
DROP TRIGGER IF EXISTS extension_permission_catalog_valid
  ON extension_permission_catalog;
CREATE TRIGGER extension_permission_catalog_valid
BEFORE INSERT ON extension_permission_catalog
FOR EACH ROW EXECUTE FUNCTION validate_extension_permission_catalog();

DROP TRIGGER IF EXISTS extension_permission_catalog_immutable
  ON extension_permission_catalog;
CREATE TRIGGER extension_permission_catalog_immutable
BEFORE UPDATE OR DELETE ON extension_permission_catalog
FOR EACH ROW EXECUTE FUNCTION reject_extension_identity_registry_history_mutation();

DROP TRIGGER IF EXISTS extension_permission_catalog_no_truncate
  ON extension_permission_catalog;
CREATE TRIGGER extension_permission_catalog_no_truncate
BEFORE TRUNCATE ON extension_permission_catalog
FOR EACH STATEMENT EXECUTE FUNCTION reject_extension_identity_registry_history_mutation();

DROP TRIGGER IF EXISTS extension_permission_role_grant_valid
  ON extension_permission_role_grants;
CREATE TRIGGER extension_permission_role_grant_valid
BEFORE INSERT ON extension_permission_role_grants
FOR EACH ROW EXECUTE FUNCTION validate_extension_permission_role_grant();

DROP TRIGGER IF EXISTS extension_permission_role_grant_immutable
  ON extension_permission_role_grants;
CREATE TRIGGER extension_permission_role_grant_immutable
BEFORE UPDATE OR DELETE ON extension_permission_role_grants
FOR EACH ROW EXECUTE FUNCTION reject_extension_identity_registry_history_mutation();

DROP TRIGGER IF EXISTS extension_permission_role_grant_no_truncate
  ON extension_permission_role_grants;
CREATE TRIGGER extension_permission_role_grant_no_truncate
BEFORE TRUNCATE ON extension_permission_role_grants
FOR EACH STATEMENT EXECUTE FUNCTION reject_extension_identity_registry_history_mutation();

DROP TRIGGER IF EXISTS extension_identity_decision_audit_immutable
  ON audit_events;
CREATE TRIGGER extension_identity_decision_audit_immutable
BEFORE UPDATE OR DELETE ON audit_events
FOR EACH ROW EXECUTE FUNCTION reject_extension_identity_decision_audit_mutation();

DROP TRIGGER IF EXISTS extension_permission_role_suggestion_update_valid
  ON extension_permission_role_suggestions;
CREATE TRIGGER extension_permission_role_suggestion_update_valid
BEFORE UPDATE ON extension_permission_role_suggestions
FOR EACH ROW EXECUTE FUNCTION validate_extension_permission_role_suggestion_decision();

-- Decision audit FK/index may already exist on draft-029 installs.
-- +goose StatementBegin
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conrelid = format('%I.extension_permission_role_suggestions', current_schema())::regclass
      AND conname = 'extension_permission_role_suggestions_decision_audit_fk'
  ) THEN
    ALTER TABLE extension_permission_role_suggestions
      ADD CONSTRAINT extension_permission_role_suggestions_decision_audit_fk
      FOREIGN KEY (decision_audit_event_id)
      REFERENCES audit_events(id) ON DELETE RESTRICT;
  END IF;
END;
$$;
-- +goose StatementEnd

CREATE INDEX IF NOT EXISTS extension_permission_role_suggestions_decision_audit_idx
  ON extension_permission_role_suggestions (decision_audit_event_id)
  WHERE decision_audit_event_id IS NOT NULL;

-- Shared cleanup index from 029; safe to leave on every shape.
CREATE INDEX IF NOT EXISTS audit_events_created_at_id_idx
  ON audit_events (created_at, id);

-- +goose Down
-- Additive repair cannot be inverted without inventing or erasing authority
-- history. Roll forward with a new repair migration if a further change is
-- required.
-- +goose StatementBegin
DO $$
BEGIN
  RAISE EXCEPTION
    'cannot reverse identity role approval schema repair 202607170034; roll forward with a new repair migration';
END;
$$;
-- +goose StatementEnd
