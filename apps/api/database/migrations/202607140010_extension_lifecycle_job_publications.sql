-- +goose Up
-- V3 P4: River reconciliation is irreversible once committed, so lifecycle
-- publication first persists an exact desired job/schedule snapshot. Queue
-- mutations run only after the shared publication marker commits.
CREATE TABLE extension_lifecycle_job_publications (
  id BIGSERIAL PRIMARY KEY,
  operation_id BIGINT NOT NULL,
  operation TEXT NOT NULL
    CHECK (operation IN ('install', 'enable', 'disable', 'upgrade', 'rollback', 'uninstall')),
  step_id TEXT NOT NULL CHECK (octet_length(step_id) BETWEEN 1 AND 512),
  position INTEGER NOT NULL CHECK (position >= 0),
  publication_mode TEXT NOT NULL
    CHECK (publication_mode IN ('activate', 'deactivate')),

  source_extension_id TEXT,
  source_extension_version TEXT,
  source_package_digest TEXT,
  source_version_id BIGINT,
  source_runtime_instance_id TEXT,

  target_extension_id TEXT NOT NULL CHECK (target_extension_id <> ''),
  target_extension_version TEXT NOT NULL CHECK (target_extension_version <> ''),
  target_package_digest TEXT NOT NULL
    CHECK (target_package_digest ~ '^[0-9a-f]{64}$'),
  target_version_id BIGINT NOT NULL CHECK (target_version_id > 0),
  target_runtime_instance_id TEXT NOT NULL DEFAULT ''
    CHECK (target_runtime_instance_id = btrim(target_runtime_instance_id)
      AND octet_length(target_runtime_instance_id) <= 512),

  first_attempt INTEGER NOT NULL CHECK (first_attempt > 0),
  last_attempt INTEGER NOT NULL CHECK (last_attempt >= first_attempt),
  runtime_attempts JSONB NOT NULL DEFAULT '[]'::jsonb
    CHECK (jsonb_typeof(runtime_attempts) = 'array'),
  source_snapshot JSONB NOT NULL CHECK (jsonb_typeof(source_snapshot) = 'object'),
  target_snapshot JSONB NOT NULL CHECK (jsonb_typeof(target_snapshot) = 'object'),
  reconciliation_plan JSONB NOT NULL CHECK (jsonb_typeof(reconciliation_plan) = 'object'),
  publication_state TEXT NOT NULL DEFAULT 'source'
    CHECK (publication_state IN ('source', 'target')),

  reconciliation_state TEXT NOT NULL DEFAULT 'pending'
    CHECK (reconciliation_state IN ('pending', 'running', 'failed', 'succeeded')),
  reconciliation_attempt INTEGER NOT NULL DEFAULT 0 CHECK (reconciliation_attempt >= 0),
  reconciliation_result JSONB
    CHECK (reconciliation_result IS NULL OR jsonb_typeof(reconciliation_result) = 'object'),
  reconciliation_error TEXT NOT NULL DEFAULT ''
    CHECK (octet_length(reconciliation_error) <= 128
      AND (reconciliation_error = '' OR reconciliation_error ~ '^[a-z0-9]+([._-][a-z0-9]+)*$')),
  reconciled_by_user_id BIGINT,
  reconciliation_audit_event_id BIGINT,
  reconciled_at TIMESTAMPTZ,

  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  prepared_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),

  UNIQUE (operation_id, step_id, publication_mode),
  FOREIGN KEY (operation_id, step_id, publication_mode)
    REFERENCES extension_lifecycle_publications(operation_id, step_id, publication_mode)
    ON DELETE CASCADE,
  CHECK (
    (source_extension_id IS NULL
      AND source_extension_version IS NULL
      AND source_package_digest IS NULL
      AND source_version_id IS NULL
      AND source_runtime_instance_id IS NULL)
    OR
    (source_extension_id IS NOT NULL AND source_extension_id <> ''
      AND source_extension_version IS NOT NULL AND source_extension_version <> ''
      AND source_package_digest ~ '^[0-9a-f]{64}$'
      AND source_version_id > 0
      AND source_runtime_instance_id IS NOT NULL
      AND source_runtime_instance_id = btrim(source_runtime_instance_id)
      AND octet_length(source_runtime_instance_id) <= 512)
  ),
  CHECK (source_extension_id IS NULL OR source_extension_id = target_extension_id),
  CHECK ((publication_mode = 'activate' AND target_runtime_instance_id <> '')
      OR (publication_mode = 'deactivate'
        AND source_runtime_instance_id IS NOT NULL
        AND source_runtime_instance_id <> '')),
  CHECK ((reconciliation_state = 'succeeded'
      AND reconciliation_result IS NOT NULL
      AND reconciliation_error = ''
      AND reconciled_by_user_id > 0
      AND reconciliation_audit_event_id > 0
      AND reconciled_at IS NOT NULL)
    OR reconciliation_state <> 'succeeded'),
  CHECK ((reconciliation_state = 'failed' AND reconciliation_error <> '')
      OR (reconciliation_state <> 'failed' AND reconciliation_error = '')),
  CHECK ((reconciliation_state = 'succeeded') = (reconciliation_result IS NOT NULL)),
  CHECK (reconciliation_state <> 'succeeded' OR publication_state = 'target'),
  CHECK ((reconciliation_state = 'pending'
      AND reconciled_by_user_id IS NULL
      AND reconciliation_audit_event_id IS NULL)
    OR (reconciliation_state <> 'pending'
      AND reconciled_by_user_id > 0
      AND reconciliation_audit_event_id > 0)),
  CHECK (updated_at >= prepared_at)
);

CREATE INDEX extension_lifecycle_job_publications_operation_idx
  ON extension_lifecycle_job_publications (operation_id, position, id);
CREATE INDEX extension_lifecycle_job_publications_reconcile_idx
  ON extension_lifecycle_job_publications (reconciliation_state, updated_at, operation_id)
  WHERE publication_state = 'target' AND reconciliation_state <> 'succeeded';

-- +goose Down
-- A retained row may be the only evidence of a committed queue migration or
-- cancellation. Never erase it during an ordinary rollback.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM extension_lifecycle_job_publications) THEN
    RAISE EXCEPTION 'cannot remove lifecycle job publication history';
  END IF;
END $$;
-- +goose StatementEnd

DROP INDEX IF EXISTS extension_lifecycle_job_publications_reconcile_idx;
DROP INDEX IF EXISTS extension_lifecycle_job_publications_operation_idx;
DROP TABLE IF EXISTS extension_lifecycle_job_publications;
