-- +goose Up
-- V3 P4: lifecycle cleanup is a retained, exact-artifact record. Uninstall
-- cleanup stages evidence only; it never deletes an extension, version, or
-- package. Terminal success only permits purge; finalized is written after an
-- idempotent physical purger returns durable exact-operation proof.
CREATE TABLE extension_lifecycle_cleanup_records (
  id BIGSERIAL PRIMARY KEY,
  cleanup_id TEXT NOT NULL UNIQUE
    CHECK (octet_length(cleanup_id) BETWEEN 1 AND 200),
  operation_id BIGINT NOT NULL
    REFERENCES extension_lifecycle_operations(id) ON DELETE RESTRICT,
  operation TEXT NOT NULL
    CHECK (operation IN ('disable', 'upgrade', 'rollback', 'uninstall')),
  step_id TEXT NOT NULL CHECK (octet_length(step_id) BETWEEN 1 AND 512),
  position INTEGER NOT NULL CHECK (position >= 0),
  first_attempt INTEGER NOT NULL CHECK (first_attempt > 0),
  last_attempt INTEGER NOT NULL CHECK (last_attempt >= first_attempt),
  cleanup_mode TEXT NOT NULL
    CHECK (cleanup_mode IN (
      'disable', 'retired_source', 'uninstall_preserve',
      'uninstall_export_then_remove', 'uninstall_complete_removal'
    )),
  record_kind TEXT NOT NULL
    CHECK (record_kind IN ('retention', 'uninstall_tombstone')),
  status TEXT NOT NULL
    CHECK (status IN ('retained', 'pending', 'finalized')),

  -- The retained source is the exact artifact needed by disable re-enable or
  -- upgrade/rollback recovery. These are snapshots, not foreign keys to rows
  -- that a later physical purge may remove.
  retained_extension_id TEXT NOT NULL CHECK (retained_extension_id <> ''),
  retained_extension_version TEXT NOT NULL CHECK (retained_extension_version <> ''),
  retained_package_digest TEXT NOT NULL
    CHECK (retained_package_digest ~ '^[0-9a-f]{64}$'),
  retained_version_id BIGINT NOT NULL CHECK (retained_version_id > 0),
  retained_runtime_instance_id TEXT NOT NULL
    CHECK (octet_length(retained_runtime_instance_id) BETWEEN 1 AND 512),
  retained_package_path TEXT NOT NULL CHECK (retained_package_path <> ''),
  identity_snapshot JSONB NOT NULL
    CHECK (jsonb_typeof(identity_snapshot) = 'object' AND identity_snapshot <> '{}'::jsonb),
  package_snapshot JSONB NOT NULL
    CHECK (jsonb_typeof(package_snapshot) = 'object' AND package_snapshot <> '{}'::jsonb),
  runtime_recovery_snapshot JSONB NOT NULL
    CHECK (jsonb_typeof(runtime_recovery_snapshot) = 'object'
      AND runtime_recovery_snapshot <> '{}'::jsonb),
  runtime_recovery_attempts JSONB NOT NULL DEFAULT '[]'::jsonb
    CHECK (jsonb_typeof(runtime_recovery_attempts) = 'array'),

  -- Target context fences the cleanup to the operation-selected artifact even
  -- when the retained source is an older upgrade/rollback version.
  target_extension_id TEXT NOT NULL CHECK (target_extension_id <> ''),
  target_extension_version TEXT NOT NULL CHECK (target_extension_version <> ''),
  target_package_digest TEXT NOT NULL
    CHECK (target_package_digest ~ '^[0-9a-f]{64}$'),
  target_version_id BIGINT NOT NULL CHECK (target_version_id > 0),
  -- Deactivation may have no target process. The source runtime above is the
  -- required recovery identity; target runtime remains optional context.
  target_runtime_instance_id TEXT NOT NULL DEFAULT ''
    CHECK (octet_length(target_runtime_instance_id) <= 512),
  target_package_path TEXT NOT NULL CHECK (target_package_path <> ''),

  -- Evidence retention is permanent audit/recovery state. Physical presence is
  -- tracked separately so a finalized purge cannot also claim resources exist.
  identity_recovery_evidence_retained BOOLEAN NOT NULL DEFAULT TRUE,
  package_recovery_evidence_retained BOOLEAN NOT NULL DEFAULT TRUE,
  runtime_recovery_evidence_retained BOOLEAN NOT NULL DEFAULT TRUE,
  physical_identity_present BOOLEAN NOT NULL DEFAULT TRUE,
  physical_package_present BOOLEAN NOT NULL DEFAULT TRUE,
  physical_runtime_recovery_present BOOLEAN NOT NULL DEFAULT TRUE,
  retention_marker TEXT CHECK (retention_marker IS NULL OR octet_length(retention_marker) BETWEEN 1 AND 200),
  export_artifact_id TEXT
    CHECK (export_artifact_id IS NULL OR export_artifact_id ~ '^[A-Za-z0-9._-]{1,200}$'),
  export_digest TEXT,
  export_evidence_action TEXT,
  export_evidence JSONB,
  export_evidence_digest TEXT,

  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  finalized_at TIMESTAMPTZ,
  finalized_operation_revision BIGINT,
  finalized_operation_completed_at TIMESTAMPTZ,
  purge_receipt_id TEXT
    CHECK (purge_receipt_id IS NULL OR purge_receipt_id ~ '^[A-Za-z0-9._-]{1,200}$'),
  purge_proof JSONB,
  purge_proof_digest TEXT,

  UNIQUE (operation_id, step_id, cleanup_mode),
  CHECK (identity_recovery_evidence_retained
    AND package_recovery_evidence_retained
    AND runtime_recovery_evidence_retained),
  CHECK (retained_extension_id = target_extension_id),
  CHECK ((cleanup_mode = 'disable' AND operation = 'disable')
      OR (cleanup_mode = 'retired_source' AND operation IN ('upgrade', 'rollback'))
      OR (cleanup_mode LIKE 'uninstall_%' AND operation = 'uninstall')),
  CHECK ((record_kind = 'retention' AND cleanup_mode IN ('disable', 'retired_source')
          AND status = 'retained')
      OR (record_kind = 'uninstall_tombstone' AND cleanup_mode LIKE 'uninstall_%'
          AND status IN ('pending', 'finalized'))),
  CHECK ((cleanup_mode = 'uninstall_preserve'
          AND retention_marker IS NOT NULL AND retention_marker <> '')
      OR (cleanup_mode <> 'uninstall_preserve' AND retention_marker IS NULL)),
  CHECK ((cleanup_mode = 'uninstall_export_then_remove'
          AND export_artifact_id IS NOT NULL AND export_artifact_id <> ''
          AND export_digest ~ '^[0-9a-f]{64}$'
          AND export_evidence_action IN ('uninstall', 'uninstall.after')
          AND export_evidence IS NOT NULL AND jsonb_typeof(export_evidence) = 'object'
          AND export_evidence_digest ~ '^[0-9a-f]{64}$')
      OR (cleanup_mode <> 'uninstall_export_then_remove'
          AND export_artifact_id IS NULL AND export_digest IS NULL
          AND export_evidence_action IS NULL AND export_evidence IS NULL
          AND export_evidence_digest IS NULL)),
  CHECK ((status = 'finalized'
          AND finalized_at IS NOT NULL
          AND finalized_operation_revision IS NOT NULL
          AND finalized_operation_completed_at IS NOT NULL
          AND purge_receipt_id IS NOT NULL AND purge_receipt_id <> ''
          AND purge_proof IS NOT NULL
          AND jsonb_typeof(purge_proof) = 'object' AND purge_proof <> '{}'::jsonb
          AND purge_proof_digest ~ '^[0-9a-f]{64}$')
      OR (status <> 'finalized'
          AND finalized_at IS NULL
          AND finalized_operation_revision IS NULL
          AND finalized_operation_completed_at IS NULL
          AND purge_receipt_id IS NULL AND purge_proof IS NULL
          AND purge_proof_digest IS NULL)),
  CHECK ((status IN ('retained', 'pending')
          AND physical_identity_present
          AND physical_package_present
          AND physical_runtime_recovery_present)
      OR (status = 'finalized'
          AND NOT physical_identity_present
          AND NOT physical_package_present
          AND NOT physical_runtime_recovery_present)),
  CHECK (updated_at >= created_at),
  CHECK (finalized_at IS NULL OR finalized_at >= created_at)
);

CREATE INDEX extension_lifecycle_cleanup_records_operation_idx
  ON extension_lifecycle_cleanup_records (operation_id, position, id);
CREATE UNIQUE INDEX extension_lifecycle_cleanup_records_one_tombstone_idx
  ON extension_lifecycle_cleanup_records (operation_id)
  WHERE record_kind = 'uninstall_tombstone';
CREATE INDEX extension_lifecycle_cleanup_records_pending_idx
  ON extension_lifecycle_cleanup_records (updated_at, operation_id, id)
  WHERE record_kind = 'uninstall_tombstone' AND status = 'pending';
CREATE INDEX extension_lifecycle_cleanup_records_retained_artifact_idx
  ON extension_lifecycle_cleanup_records (
    retained_extension_id, retained_version_id, retained_package_digest, id
  );

-- +goose Down
-- Cleanup evidence may be the only remaining recovery map after an uninstall.
-- Refuse rollback instead of silently deleting it.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM extension_lifecycle_cleanup_records) THEN
    RAISE EXCEPTION 'cannot remove lifecycle cleanup history';
  END IF;
END $$;
-- +goose StatementEnd

DROP INDEX IF EXISTS extension_lifecycle_cleanup_records_retained_artifact_idx;
DROP INDEX IF EXISTS extension_lifecycle_cleanup_records_pending_idx;
DROP INDEX IF EXISTS extension_lifecycle_cleanup_records_one_tombstone_idx;
DROP INDEX IF EXISTS extension_lifecycle_cleanup_records_operation_idx;
DROP TABLE IF EXISTS extension_lifecycle_cleanup_records;
