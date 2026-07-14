-- +goose Up
-- P5 data disposition is independent from physical schemas and roles so its
-- exact cleanup receipt survives preserve, removal, reinstall, and role drop.
CREATE TABLE extension_database_dispositions (
  id BIGSERIAL PRIMARY KEY,
  cleanup_id TEXT NOT NULL UNIQUE
    CHECK (cleanup_id ~ '^[A-Za-z0-9._:-]{1,200}$'),
  operation_id BIGINT NOT NULL UNIQUE CHECK (operation_id > 0),
  cleanup_mode TEXT NOT NULL
    CHECK (cleanup_mode IN (
      'uninstall_preserve',
      'uninstall_export_then_remove',
      'uninstall_complete_removal'
    )),
  extension_id TEXT NOT NULL CHECK (extension_id <> ''),
  extension_version_id BIGINT NOT NULL CHECK (extension_version_id > 0),
  extension_version TEXT NOT NULL CHECK (extension_version <> ''),
  package_digest TEXT NOT NULL CHECK (package_digest ~ '^[0-9a-f]{64}$'),
  schema_name TEXT NOT NULL CHECK (schema_name ~ '^[a-z_][a-z0-9_]{0,62}$'),
  owner_role_name TEXT NOT NULL CHECK (owner_role_name ~ '^[a-z_][a-z0-9_]{0,62}$'),
  runtime_role_name TEXT NOT NULL CHECK (runtime_role_name ~ '^[a-z_][a-z0-9_]{0,62}$'),
  export_artifact_id TEXT,
  export_digest TEXT,
  export_evidence_digest TEXT,
  status TEXT NOT NULL DEFAULT 'prepared'
    CHECK (status IN ('prepared', 'applied')),
  data_disposition TEXT
    CHECK (data_disposition IN ('preserved', 'exported_then_removed', 'removed')),
  credential_revoked BOOLEAN NOT NULL DEFAULT FALSE,
  schema_retained BOOLEAN,
  roles_removed BOOLEAN NOT NULL DEFAULT FALSE,
  receipt_id TEXT UNIQUE
    CHECK (receipt_id IS NULL OR receipt_id ~ '^[A-Za-z0-9._:-]{1,200}$'),
  proof JSONB CHECK (proof IS NULL OR jsonb_typeof(proof) = 'object'),
  proof_digest TEXT
    CHECK (proof_digest IS NULL OR proof_digest ~ '^[0-9a-f]{64}$'),
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
  applied_at TIMESTAMPTZ,
  CHECK (schema_name <> owner_role_name),
  CHECK (schema_name <> runtime_role_name),
  CHECK (owner_role_name <> runtime_role_name),
  CHECK (
    (cleanup_mode = 'uninstall_export_then_remove'
      AND export_artifact_id IS NOT NULL AND export_artifact_id <> ''
      AND export_digest ~ '^[0-9a-f]{64}$'
      AND export_evidence_digest ~ '^[0-9a-f]{64}$')
    OR
    (cleanup_mode <> 'uninstall_export_then_remove'
      AND export_artifact_id IS NULL
      AND export_digest IS NULL
      AND export_evidence_digest IS NULL)
  ),
  CHECK (
    (status = 'prepared'
      AND data_disposition IS NULL
      AND NOT credential_revoked
      AND schema_retained IS NULL
      AND NOT roles_removed
      AND receipt_id IS NULL
      AND proof IS NULL
      AND proof_digest IS NULL
      AND applied_at IS NULL)
    OR
    (status = 'applied'
      AND data_disposition IS NOT NULL
      AND credential_revoked
      AND schema_retained IS NOT NULL
      AND receipt_id IS NOT NULL
      AND proof IS NOT NULL
      AND proof_digest IS NOT NULL
      AND applied_at IS NOT NULL)
  ),
  CHECK (
    (cleanup_mode = 'uninstall_preserve'
      AND (status = 'prepared' OR (data_disposition = 'preserved' AND schema_retained)))
    OR
    (cleanup_mode = 'uninstall_export_then_remove'
      AND (status = 'prepared' OR (data_disposition = 'exported_then_removed' AND NOT schema_retained)))
    OR
    (cleanup_mode = 'uninstall_complete_removal'
      AND (status = 'prepared' OR (data_disposition = 'removed' AND NOT schema_retained)))
  ),
  CHECK (updated_at >= created_at),
  CHECK (applied_at IS NULL OR applied_at >= created_at)
);

CREATE INDEX extension_database_dispositions_extension_idx
  ON extension_database_dispositions (extension_id, created_at DESC, id DESC);
CREATE INDEX extension_database_dispositions_prepared_idx
  ON extension_database_dispositions (updated_at, id) WHERE status = 'prepared';

-- +goose Down
-- A disposition receipt may be the only durable evidence that retained data
-- was preserved or that destructive schema removal was explicitly authorized.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM extension_database_dispositions) THEN
    RAISE EXCEPTION 'cannot remove extension database disposition evidence';
  END IF;
END $$;
-- +goose StatementEnd

DROP INDEX IF EXISTS extension_database_dispositions_prepared_idx;
DROP INDEX IF EXISTS extension_database_dispositions_extension_idx;
DROP TABLE IF EXISTS extension_database_dispositions;
