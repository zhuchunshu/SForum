-- +goose Up
-- P5 owns physical PostgreSQL identities and migration history independently
-- from core Goose history and the V1 checksum-only extension ledger.
CREATE TABLE extension_database_resources (
  extension_id TEXT PRIMARY KEY CHECK (extension_id <> ''),
  schema_name TEXT NOT NULL UNIQUE
    CHECK (schema_name ~ '^[a-z_][a-z0-9_]{0,62}$'),
  owner_role_name TEXT NOT NULL UNIQUE
    CHECK (owner_role_name ~ '^[a-z_][a-z0-9_]{0,62}$'),
  runtime_role_name TEXT NOT NULL UNIQUE
    CHECK (runtime_role_name ~ '^[a-z_][a-z0-9_]{0,62}$'),
  resource_revision BIGINT NOT NULL DEFAULT 1 CHECK (resource_revision > 0),
  status TEXT NOT NULL DEFAULT 'provisioned'
    CHECK (status IN ('provisioned', 'revoked', 'failed')),
  failure_code TEXT NOT NULL DEFAULT ''
    CHECK (failure_code = '' OR failure_code ~ '^[a-z0-9][a-z0-9._-]{0,127}$'),
  schema_retained BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  revoked_at TIMESTAMPTZ,
  CHECK (schema_name <> owner_role_name),
  CHECK (schema_name <> runtime_role_name),
  CHECK (owner_role_name <> runtime_role_name),
  CHECK (updated_at >= created_at),
  CHECK ((status = 'revoked' AND revoked_at IS NOT NULL)
      OR (status <> 'revoked' AND revoked_at IS NULL))
);

CREATE TABLE extension_database_grants (
  id BIGSERIAL PRIMARY KEY,
  extension_id TEXT NOT NULL
    REFERENCES extension_database_resources(extension_id) ON DELETE RESTRICT,
  extension_version_id BIGINT NOT NULL CHECK (extension_version_id > 0),
  extension_version TEXT NOT NULL CHECK (extension_version <> ''),
  package_digest TEXT NOT NULL CHECK (package_digest ~ '^[0-9a-f]{64}$'),
  database_contract_version TEXT NOT NULL
    CHECK (database_contract_version ~ '^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$'),
  authority TEXT NOT NULL
    CHECK (authority IN ('own_schema', 'core_views', 'host_commands', 'raw_core', 'kernel')),
  requested_schema TEXT NOT NULL DEFAULT '',
  requested_role TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'active'
    CHECK (status IN ('active', 'revoked', 'failed')),
  active_credential_revision BIGINT NOT NULL DEFAULT 0
    CHECK (active_credential_revision >= 0),
  grant_revision BIGINT NOT NULL DEFAULT 1 CHECK (grant_revision > 0),
  granted_by_user_id BIGINT NOT NULL CHECK (granted_by_user_id > 0),
  grant_audit_event_id BIGINT NOT NULL CHECK (grant_audit_event_id > 0),
  revoked_by_user_id BIGINT,
  revoke_audit_event_id BIGINT,
  failure_code TEXT NOT NULL DEFAULT ''
    CHECK (failure_code = '' OR failure_code ~ '^[a-z0-9][a-z0-9._-]{0,127}$'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  revoked_at TIMESTAMPTZ,
  UNIQUE (extension_id, extension_version_id, extension_version, package_digest),
  CHECK (updated_at >= created_at),
  CHECK ((status = 'active' AND revoked_at IS NULL)
      OR (status = 'revoked' AND revoked_at IS NOT NULL)
      OR status = 'failed'),
  CHECK ((revoked_by_user_id IS NULL AND revoke_audit_event_id IS NULL)
      OR (revoked_by_user_id > 0 AND revoke_audit_event_id > 0))
);

CREATE UNIQUE INDEX extension_database_grants_active_extension_idx
  ON extension_database_grants (extension_id) WHERE status = 'active';
CREATE INDEX extension_database_grants_artifact_idx
  ON extension_database_grants (
    extension_id, extension_version_id, package_digest, id DESC
  );

CREATE TABLE extension_database_credentials (
  id BIGSERIAL PRIMARY KEY,
  grant_id BIGINT NOT NULL
    REFERENCES extension_database_grants(id) ON DELETE RESTRICT,
  extension_id TEXT NOT NULL
    REFERENCES extension_database_resources(extension_id) ON DELETE RESTRICT,
  credential_revision BIGINT NOT NULL CHECK (credential_revision > 0),
  role_name TEXT NOT NULL CHECK (role_name ~ '^[a-z_][a-z0-9_]{0,62}$'),
  credential_fingerprint TEXT NOT NULL
    CHECK (credential_fingerprint ~ '^[0-9a-f]{64}$'),
  issued_by_user_id BIGINT NOT NULL CHECK (issued_by_user_id > 0),
  issue_audit_event_id BIGINT NOT NULL CHECK (issue_audit_event_id > 0),
  status TEXT NOT NULL DEFAULT 'active'
    CHECK (status IN ('active', 'revoked')),
  issued_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  revoked_by_user_id BIGINT,
  revoke_audit_event_id BIGINT,
  revoked_at TIMESTAMPTZ,
  UNIQUE (extension_id, credential_revision),
  CHECK ((status = 'active' AND revoked_at IS NULL)
      OR (status = 'revoked' AND revoked_at IS NOT NULL)),
  CHECK ((revoked_by_user_id IS NULL AND revoke_audit_event_id IS NULL)
      OR (revoked_by_user_id > 0 AND revoke_audit_event_id > 0))
);

CREATE UNIQUE INDEX extension_database_credentials_active_extension_idx
  ON extension_database_credentials (extension_id) WHERE status = 'active';
CREATE INDEX extension_database_credentials_grant_idx
  ON extension_database_credentials (grant_id, credential_revision DESC);

CREATE TABLE extension_database_migration_plans (
  id BIGSERIAL PRIMARY KEY,
  operation_id BIGINT NOT NULL
    REFERENCES extension_lifecycle_operations(id) ON DELETE RESTRICT,
  operation TEXT NOT NULL CHECK (operation IN ('install', 'upgrade', 'rollback')),
  migration_mode TEXT NOT NULL CHECK (migration_mode IN ('install', 'upgrade', 'rollback')),
  step_id TEXT NOT NULL CHECK (octet_length(step_id) BETWEEN 1 AND 512),
  attempt INTEGER NOT NULL CHECK (attempt > 0),
  plan_digest TEXT NOT NULL UNIQUE CHECK (plan_digest ~ '^[0-9a-f]{64}$'),
  extension_id TEXT NOT NULL
    REFERENCES extension_database_resources(extension_id) ON DELETE RESTRICT,

  source_extension_version_id BIGINT,
  source_extension_version TEXT,
  source_package_digest TEXT,
  source_migrations_digest TEXT,
  target_extension_version_id BIGINT NOT NULL CHECK (target_extension_version_id > 0),
  target_extension_version TEXT NOT NULL CHECK (target_extension_version <> ''),
  target_package_digest TEXT NOT NULL CHECK (target_package_digest ~ '^[0-9a-f]{64}$'),
  target_migrations_digest TEXT NOT NULL CHECK (target_migrations_digest ~ '^[0-9a-f]{64}$'),

  schema_name TEXT NOT NULL CHECK (schema_name ~ '^[a-z_][a-z0-9_]{0,62}$'),
  owner_role_name TEXT NOT NULL CHECK (owner_role_name ~ '^[a-z_][a-z0-9_]{0,62}$'),
  dry_run_digest TEXT NOT NULL CHECK (dry_run_digest ~ '^[0-9a-f]{64}$'),
  status TEXT NOT NULL DEFAULT 'planned'
    CHECK (status IN ('planned', 'running', 'succeeded', 'failed', 'indeterminate')),
  current_step INTEGER NOT NULL DEFAULT 0 CHECK (current_step >= 0),
  total_steps INTEGER NOT NULL CHECK (total_steps >= 0),
  failure_code TEXT NOT NULL DEFAULT ''
    CHECK (failure_code = '' OR failure_code ~ '^[a-z0-9][a-z0-9._-]{0,127}$'),
  warning_code TEXT NOT NULL DEFAULT ''
    CHECK (warning_code = '' OR warning_code ~ '^[a-z0-9][a-z0-9._-]{0,127}$'),
  has_non_transactional BOOLEAN NOT NULL DEFAULT FALSE,
  target_ready BOOLEAN NOT NULL DEFAULT FALSE,
  source_resume_safe BOOLEAN NOT NULL DEFAULT TRUE,
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  execution_started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  UNIQUE (operation_id, migration_mode),
  CHECK (operation = migration_mode),
  CHECK (
    (source_extension_version_id IS NULL
      AND source_extension_version IS NULL
      AND source_package_digest IS NULL
      AND source_migrations_digest IS NULL)
    OR
    (source_extension_version_id > 0
      AND source_extension_version IS NOT NULL AND source_extension_version <> ''
      AND source_package_digest ~ '^[0-9a-f]{64}$'
      AND source_migrations_digest ~ '^[0-9a-f]{64}$')
  ),
  CHECK ((operation = 'install' AND source_extension_version_id IS NULL)
      OR (operation IN ('upgrade', 'rollback') AND source_extension_version_id IS NOT NULL)),
  CHECK (current_step <= total_steps),
  CHECK ((status = 'succeeded' AND target_ready AND completed_at IS NOT NULL)
      OR (status <> 'succeeded' AND NOT target_ready)),
  CHECK ((status = 'running' AND execution_started_at IS NOT NULL)
      OR status <> 'running'),
  CHECK (updated_at >= created_at),
  CHECK (execution_started_at IS NULL OR execution_started_at >= created_at),
  CHECK (completed_at IS NULL OR completed_at >= created_at)
);

CREATE INDEX extension_database_migration_plans_extension_idx
  ON extension_database_migration_plans (extension_id, created_at DESC, id DESC);
CREATE INDEX extension_database_migration_plans_recovery_idx
  ON extension_database_migration_plans (status, updated_at, id)
  WHERE status IN ('running', 'failed', 'indeterminate');

CREATE TABLE extension_database_migration_steps (
  id BIGSERIAL PRIMARY KEY,
  plan_id BIGINT NOT NULL
    REFERENCES extension_database_migration_plans(id) ON DELETE RESTRICT,
  position INTEGER NOT NULL CHECK (position > 0),
  direction TEXT NOT NULL CHECK (direction IN ('up', 'down')),
  migration_id TEXT NOT NULL CHECK (migration_id <> ''),
  contract_version TEXT NOT NULL
    CHECK (contract_version ~ '^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$'),
  package_path TEXT NOT NULL CHECK (package_path <> ''),
  checksum TEXT NOT NULL CHECK (checksum ~ '^[0-9a-f]{64}$'),
  transaction_policy TEXT NOT NULL
    CHECK (transaction_policy IN ('required', 'forbidden', 'auto')),
  execution_mode TEXT NOT NULL
    CHECK (execution_mode IN ('transactional', 'non_transactional')),
  status TEXT NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'skipped', 'running', 'applied', 'failed', 'indeterminate')),
  failure_code TEXT NOT NULL DEFAULT ''
    CHECK (failure_code = '' OR failure_code ~ '^[a-z0-9][a-z0-9._-]{0,127}$'),
  warning_code TEXT NOT NULL DEFAULT ''
    CHECK (warning_code = '' OR warning_code ~ '^[a-z0-9][a-z0-9._-]{0,127}$'),
  result_digest TEXT
    CHECK (result_digest IS NULL OR result_digest ~ '^[0-9a-f]{64}$'),
  execution_started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  UNIQUE (plan_id, position),
  UNIQUE (plan_id, migration_id, direction),
  CHECK ((status = 'running' AND execution_started_at IS NOT NULL)
      OR status <> 'running'),
  CHECK ((status IN ('applied', 'skipped') AND completed_at IS NOT NULL)
      OR status NOT IN ('applied', 'skipped')),
  CHECK (completed_at IS NULL OR execution_started_at IS NULL
      OR completed_at >= execution_started_at)
);

CREATE INDEX extension_database_migration_steps_plan_status_idx
  ON extension_database_migration_steps (plan_id, status, position);

CREATE TABLE extension_database_migration_state (
  extension_id TEXT NOT NULL
    REFERENCES extension_database_resources(extension_id) ON DELETE RESTRICT,
  migration_id TEXT NOT NULL,
  contract_version TEXT NOT NULL
    CHECK (contract_version ~ '^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$'),
  package_path TEXT NOT NULL CHECK (package_path <> ''),
  checksum TEXT NOT NULL CHECK (checksum ~ '^[0-9a-f]{64}$'),
  transaction_policy TEXT NOT NULL
    CHECK (transaction_policy IN ('required', 'forbidden', 'auto')),
  applied_extension_version_id BIGINT NOT NULL CHECK (applied_extension_version_id > 0),
  applied_package_digest TEXT NOT NULL CHECK (applied_package_digest ~ '^[0-9a-f]{64}$'),
  applied_plan_id BIGINT NOT NULL
    REFERENCES extension_database_migration_plans(id) ON DELETE RESTRICT,
  applied_step_id BIGINT NOT NULL
    REFERENCES extension_database_migration_steps(id) ON DELETE RESTRICT,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  PRIMARY KEY (extension_id, migration_id)
);

CREATE INDEX extension_database_migration_state_plan_idx
  ON extension_database_migration_state (applied_plan_id, applied_step_id);

CREATE TABLE extension_database_migration_proofs (
  id BIGSERIAL PRIMARY KEY,
  plan_id BIGINT NOT NULL UNIQUE
    REFERENCES extension_database_migration_plans(id) ON DELETE RESTRICT,
  plan_digest TEXT NOT NULL UNIQUE CHECK (plan_digest ~ '^[0-9a-f]{64}$'),
  proof_id TEXT NOT NULL UNIQUE
    CHECK (proof_id ~ '^[A-Za-z0-9._:-]{1,200}$'),
  proof_digest TEXT NOT NULL CHECK (proof_digest ~ '^[0-9a-f]{64}$'),
  target_ready BOOLEAN NOT NULL,
  source_resume_safe BOOLEAN NOT NULL,
  warning_codes JSONB NOT NULL DEFAULT '[]'::jsonb
    CHECK (jsonb_typeof(warning_codes) = 'array'),
  failure_code TEXT NOT NULL DEFAULT ''
    CHECK (failure_code = '' OR failure_code ~ '^[a-z0-9][a-z0-9._-]{0,127}$'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp()
);

CREATE INDEX extension_database_migration_proofs_outcome_idx
  ON extension_database_migration_proofs (target_ready, source_resume_safe, created_at DESC, id DESC);

-- +goose Down
-- Registry rows may be the only durable explanation for a retained schema or
-- an indeterminate non-transactional migration. Never erase them implicitly.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM extension_database_resources)
    OR EXISTS (SELECT 1 FROM extension_database_grants)
    OR EXISTS (SELECT 1 FROM extension_database_credentials)
    OR EXISTS (SELECT 1 FROM extension_database_migration_plans)
    OR EXISTS (SELECT 1 FROM extension_database_migration_steps)
    OR EXISTS (SELECT 1 FROM extension_database_migration_state)
    OR EXISTS (SELECT 1 FROM extension_database_migration_proofs) THEN
    RAISE EXCEPTION 'cannot remove extension database registry evidence';
  END IF;
END $$;
-- +goose StatementEnd

DROP INDEX IF EXISTS extension_database_migration_proofs_outcome_idx;
DROP TABLE IF EXISTS extension_database_migration_proofs;
DROP INDEX IF EXISTS extension_database_migration_state_plan_idx;
DROP TABLE IF EXISTS extension_database_migration_state;
DROP INDEX IF EXISTS extension_database_migration_steps_plan_status_idx;
DROP TABLE IF EXISTS extension_database_migration_steps;
DROP INDEX IF EXISTS extension_database_migration_plans_recovery_idx;
DROP INDEX IF EXISTS extension_database_migration_plans_extension_idx;
DROP TABLE IF EXISTS extension_database_migration_plans;
DROP INDEX IF EXISTS extension_database_credentials_grant_idx;
DROP INDEX IF EXISTS extension_database_credentials_active_extension_idx;
DROP TABLE IF EXISTS extension_database_credentials;
DROP INDEX IF EXISTS extension_database_grants_artifact_idx;
DROP INDEX IF EXISTS extension_database_grants_active_extension_idx;
DROP TABLE IF EXISTS extension_database_grants;
DROP TABLE IF EXISTS extension_database_resources;
