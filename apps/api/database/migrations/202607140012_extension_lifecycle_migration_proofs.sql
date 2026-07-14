-- +goose Up
-- V3 P4 records migration-boundary decisions separately from the legacy
-- extension_migration_ledger. That v1 table records checksums only and is never
-- accepted as evidence that SQL ran or that an old artifact can resume safely.
CREATE TABLE extension_lifecycle_migration_proofs (
  id BIGSERIAL PRIMARY KEY,
  operation_id BIGINT NOT NULL
    REFERENCES extension_lifecycle_operations(id) ON DELETE RESTRICT,
  operation TEXT NOT NULL
    CHECK (operation IN ('install', 'upgrade', 'rollback')),
  migration_mode TEXT NOT NULL
    CHECK (migration_mode IN ('install', 'upgrade', 'rollback')),
  step_id TEXT NOT NULL CHECK (octet_length(step_id) BETWEEN 1 AND 512),
  position INTEGER NOT NULL CHECK (position >= 0),

  source_extension_id TEXT,
  source_extension_version TEXT,
  source_package_digest TEXT,
  source_version_id BIGINT,
  source_migrations_digest TEXT,

  target_extension_id TEXT NOT NULL CHECK (target_extension_id <> ''),
  target_extension_version TEXT NOT NULL CHECK (target_extension_version <> ''),
  target_package_digest TEXT NOT NULL
    CHECK (target_package_digest ~ '^[0-9a-f]{64}$'),
  target_version_id BIGINT NOT NULL CHECK (target_version_id > 0),
  target_migrations_digest TEXT NOT NULL
    CHECK (target_migrations_digest ~ '^[0-9a-f]{64}$'),
  plan_digest TEXT NOT NULL CHECK (plan_digest ~ '^[0-9a-f]{64}$'),

  first_attempt INTEGER NOT NULL DEFAULT 0 CHECK (first_attempt >= 0),
  last_attempt INTEGER NOT NULL DEFAULT 0 CHECK (last_attempt >= first_attempt),
  last_observed_step_id TEXT NOT NULL DEFAULT ''
    CHECK (octet_length(last_observed_step_id) <= 512),
  last_observed_attempt INTEGER NOT NULL DEFAULT 0
    CHECK (last_observed_attempt >= 0),
  status TEXT NOT NULL DEFAULT 'not_started'
    CHECK (status IN ('not_started', 'blocked', 'executing', 'target_ready')),
  target_ready BOOLEAN NOT NULL DEFAULT FALSE,
  source_resume_safe BOOLEAN NOT NULL DEFAULT TRUE,

  proof_kind TEXT CHECK (proof_kind IN ('host_noop', 'p5_engine', 'reused')),
  proof_id TEXT
    CHECK (proof_id IS NULL OR proof_id ~ '^[A-Za-z0-9._:-]{1,200}$'),
  proof_digest TEXT,
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  execution_started_at TIMESTAMPTZ,
  proven_at TIMESTAMPTZ,

  UNIQUE (operation_id, migration_mode),
  CHECK (operation = migration_mode),
  CHECK (
    (source_extension_id IS NULL
      AND source_extension_version IS NULL
      AND source_package_digest IS NULL
      AND source_version_id IS NULL
      AND source_migrations_digest IS NULL)
    OR
    (source_extension_id IS NOT NULL AND source_extension_id <> ''
      AND source_extension_version IS NOT NULL AND source_extension_version <> ''
      AND source_package_digest ~ '^[0-9a-f]{64}$'
      AND source_version_id > 0
      AND source_migrations_digest ~ '^[0-9a-f]{64}$')
  ),
  CHECK (source_extension_id IS NULL OR source_extension_id = target_extension_id),
  CHECK ((operation = 'install' AND source_extension_id IS NULL)
      OR (operation IN ('upgrade', 'rollback') AND source_extension_id IS NOT NULL)),
  CHECK ((first_attempt = 0 AND last_attempt = 0)
      OR (first_attempt > 0 AND last_attempt >= first_attempt)),
  CHECK ((last_observed_step_id = '' AND last_observed_attempt = 0)
      OR (last_observed_step_id <> '' AND last_observed_attempt > 0)),
  CHECK ((status = 'target_ready' AND target_ready)
      OR (status <> 'target_ready' AND NOT target_ready)),
  CHECK ((status = 'executing' AND execution_started_at IS NOT NULL)
      OR status <> 'executing'),
  CHECK (status <> 'executing' OR source_resume_safe = FALSE),
  CHECK (status = 'not_started' OR first_attempt > 0),
  CHECK (status <> 'not_started'
    OR (first_attempt = 0 AND last_attempt = 0
      AND proof_kind IS NULL AND execution_started_at IS NULL)),
  CHECK (
    (proof_kind IS NULL AND proof_id IS NULL AND proof_digest IS NULL AND proven_at IS NULL)
    OR
    (proof_kind IS NOT NULL AND proof_id IS NOT NULL
      AND proof_digest ~ '^[0-9a-f]{64}$' AND proven_at IS NOT NULL)
  ),
  CHECK (target_ready = FALSE OR proof_kind IS NOT NULL),
  CHECK (proof_kind NOT IN ('host_noop', 'reused') OR target_ready = TRUE),
  CHECK (updated_at >= created_at),
  CHECK (execution_started_at IS NULL OR execution_started_at >= created_at),
  CHECK (proven_at IS NULL OR proven_at >= created_at)
);

CREATE INDEX extension_lifecycle_migration_proofs_operation_idx
  ON extension_lifecycle_migration_proofs (operation_id, position, id);
CREATE INDEX extension_lifecycle_migration_proofs_target_ready_idx
  ON extension_lifecycle_migration_proofs (
    target_extension_id, target_version_id, target_package_digest,
    target_migrations_digest, id DESC
  ) WHERE target_ready = TRUE;
CREATE INDEX extension_lifecycle_migration_proofs_unsafe_source_idx
  ON extension_lifecycle_migration_proofs (updated_at, operation_id, id)
  WHERE source_resume_safe = FALSE;

-- +goose Down
-- A proof may be the only durable reason an old runtime can or cannot reopen.
-- Refuse rollback instead of silently falling back to the v1 checksum ledger.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM extension_lifecycle_migration_proofs) THEN
    RAISE EXCEPTION 'cannot remove lifecycle migration proofs';
  END IF;
END $$;
-- +goose StatementEnd

DROP INDEX IF EXISTS extension_lifecycle_migration_proofs_unsafe_source_idx;
DROP INDEX IF EXISTS extension_lifecycle_migration_proofs_target_ready_idx;
DROP INDEX IF EXISTS extension_lifecycle_migration_proofs_operation_idx;
DROP TABLE IF EXISTS extension_lifecycle_migration_proofs;
