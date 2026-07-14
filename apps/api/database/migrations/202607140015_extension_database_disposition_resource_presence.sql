-- +goose Up
-- A lifecycle cleanup can target an extension that never provisioned a plugin
-- database. Persist that fact instead of claiming a nonexistent schema was
-- retained. Historical rows predate this distinction and represent real
-- Registry resources, so they backfill to true.
ALTER TABLE extension_database_dispositions
  ADD COLUMN resource_existed BOOLEAN NOT NULL DEFAULT TRUE;

-- Migration 014 intentionally used an anonymous CHECK. Locate only its exact
-- cleanup-mode outcome constraint so applied databases do not depend on the
-- server-generated constraint suffix.
-- +goose StatementBegin
DO $$
DECLARE
  existing_constraint TEXT;
  matching_constraints INTEGER;
BEGIN
  SELECT COUNT(*), MIN(conname)
  INTO matching_constraints, existing_constraint
  FROM pg_constraint
  WHERE conrelid = 'extension_database_dispositions'::regclass
    AND contype = 'c'
    AND pg_get_constraintdef(oid) LIKE '%uninstall_preserve%'
    AND pg_get_constraintdef(oid) LIKE '%exported_then_removed%'
    AND pg_get_constraintdef(oid) LIKE '%schema_retained%';

  IF matching_constraints <> 1 THEN
    RAISE EXCEPTION 'expected exactly one database disposition mode outcome constraint, found %', matching_constraints;
  END IF;
  EXECUTE format(
    'ALTER TABLE extension_database_dispositions DROP CONSTRAINT %I',
    existing_constraint
  );
END $$;
-- +goose StatementEnd

ALTER TABLE extension_database_dispositions
  ADD CONSTRAINT extension_database_dispositions_mode_outcome_check
  CHECK (
    (resource_existed AND (
      (cleanup_mode = 'uninstall_preserve'
        AND (status = 'prepared' OR (data_disposition = 'preserved' AND schema_retained)))
      OR
      (cleanup_mode = 'uninstall_export_then_remove'
        AND (status = 'prepared' OR (data_disposition = 'exported_then_removed' AND NOT schema_retained)))
      OR
      (cleanup_mode = 'uninstall_complete_removal'
        AND (status = 'prepared' OR (data_disposition = 'removed' AND NOT schema_retained)))
    ))
    OR
    (NOT resource_existed AND (
      status = 'prepared'
      OR (
        NOT schema_retained
        AND NOT roles_removed
        AND (
          (cleanup_mode = 'uninstall_preserve' AND data_disposition = 'preserved')
          OR (cleanup_mode = 'uninstall_export_then_remove' AND data_disposition = 'exported_then_removed')
          OR (cleanup_mode = 'uninstall_complete_removal' AND data_disposition = 'removed')
        )
      )
    ))
  );

-- +goose Down
-- Resource presence is part of the durable physical cleanup proof. Refuse to
-- erase it after the first disposition is prepared or applied.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM extension_database_dispositions) THEN
    RAISE EXCEPTION 'cannot remove extension database resource presence evidence';
  END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE extension_database_dispositions
  DROP CONSTRAINT extension_database_dispositions_mode_outcome_check;

ALTER TABLE extension_database_dispositions
  ADD CONSTRAINT extension_database_dispositions_mode_outcome_check
  CHECK (
    (cleanup_mode = 'uninstall_preserve'
      AND (status = 'prepared' OR (data_disposition = 'preserved' AND schema_retained)))
    OR
    (cleanup_mode = 'uninstall_export_then_remove'
      AND (status = 'prepared' OR (data_disposition = 'exported_then_removed' AND NOT schema_retained)))
    OR
    (cleanup_mode = 'uninstall_complete_removal'
      AND (status = 'prepared' OR (data_disposition = 'removed' AND NOT schema_retained)))
  );

ALTER TABLE extension_database_dispositions
  DROP COLUMN resource_existed;
