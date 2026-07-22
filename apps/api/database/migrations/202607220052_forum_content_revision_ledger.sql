-- +goose NO TRANSACTION
-- +goose Up
ALTER TABLE posts
  ADD COLUMN IF NOT EXISTS current_revision BIGINT NOT NULL DEFAULT 0;

-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = current_schema()
      AND table_name = 'post_revisions'
      AND column_name = 'edited_by_user_id'
  ) AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = current_schema()
      AND table_name = 'post_revisions'
      AND column_name = 'superseded_by_user_id'
  ) THEN
    ALTER TABLE post_revisions RENAME COLUMN edited_by_user_id TO superseded_by_user_id;
  END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE post_revisions
  ADD COLUMN IF NOT EXISTS revision_no BIGINT,
  ADD COLUMN IF NOT EXISTS actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS operation TEXT,
  ADD COLUMN IF NOT EXISTS origin TEXT,
  ADD COLUMN IF NOT EXISTS changed_fields TEXT[] NOT NULL DEFAULT '{}',
  ADD COLUMN IF NOT EXISTS attachment_ids BIGINT[] NOT NULL DEFAULT '{}',
  ADD COLUMN IF NOT EXISTS committed_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS restored_from_revision_id BIGINT,
  ADD COLUMN IF NOT EXISTS snapshot_complete BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS redacted_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS redacted_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS redaction_reason TEXT NOT NULL DEFAULT '';

ALTER TABLE post_revisions
  DROP CONSTRAINT IF EXISTS post_revisions_restored_from_revision_fk,
  DROP CONSTRAINT IF EXISTS post_revisions_redaction_consistent,
  DROP CONSTRAINT IF EXISTS post_revisions_restore_pointer_consistent,
  DROP CONSTRAINT IF EXISTS post_revisions_changed_fields_allowed,
  DROP CONSTRAINT IF EXISTS post_revisions_origin_allowed,
  DROP CONSTRAINT IF EXISTS post_revisions_operation_allowed,
  DROP CONSTRAINT IF EXISTS post_revisions_revision_no_positive;

ALTER TABLE post_revisions
  ADD CONSTRAINT post_revisions_revision_no_positive
    CHECK (revision_no IS NULL OR revision_no > 0) NOT VALID,
  ADD CONSTRAINT post_revisions_operation_allowed
    CHECK (operation IS NULL OR operation IN ('create', 'edit', 'restore', 'migration')) NOT VALID,
  ADD CONSTRAINT post_revisions_origin_allowed
    CHECK (origin IS NULL OR origin IN ('self', 'staff', 'migration')) NOT VALID,
  ADD CONSTRAINT post_revisions_changed_fields_allowed
    CHECK (changed_fields <@ ARRAY['title', 'content', 'category', 'tags', 'attachments']::text[]) NOT VALID,
  ADD CONSTRAINT post_revisions_restore_pointer_consistent
    CHECK (
      (operation = 'restore' AND restored_from_revision_id IS NOT NULL)
      OR ((operation IS NULL OR operation <> 'restore') AND restored_from_revision_id IS NULL)
    ) NOT VALID,
  ADD CONSTRAINT post_revisions_redaction_consistent
    CHECK (
      (redacted_at IS NULL AND redacted_by_user_id IS NULL AND redaction_reason = '')
      OR (redacted_at IS NOT NULL AND redacted_by_user_id IS NOT NULL AND length(btrim(redaction_reason)) > 0)
    ) NOT VALID;

ALTER TABLE post_revisions
  ADD CONSTRAINT post_revisions_restored_from_revision_fk
    FOREIGN KEY (restored_from_revision_id) REFERENCES post_revisions(id) ON DELETE SET NULL NOT VALID;

CREATE TABLE IF NOT EXISTS topic_revision_snapshots (
  post_revision_id BIGINT PRIMARY KEY REFERENCES post_revisions(id) ON DELETE CASCADE,
  topic_id BIGINT NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
  title TEXT,
  category_slug TEXT,
  tag_slugs TEXT[] NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS post_revisions_post_revision_no_unique_idx
  ON post_revisions (post_id, revision_no)
  WHERE revision_no IS NOT NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS post_revisions_post_revision_desc_idx
  ON post_revisions (post_id, revision_no DESC)
  WHERE revision_no IS NOT NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS topic_revision_snapshots_topic_revision_idx
  ON topic_revision_snapshots (topic_id, post_revision_id);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS topic_revision_snapshots_topic_revision_idx;
DROP INDEX CONCURRENTLY IF EXISTS post_revisions_post_revision_desc_idx;
DROP INDEX CONCURRENTLY IF EXISTS post_revisions_post_revision_no_unique_idx;

DROP TABLE IF EXISTS topic_revision_snapshots;

ALTER TABLE post_revisions
  DROP CONSTRAINT IF EXISTS post_revisions_restored_from_revision_fk,
  DROP CONSTRAINT IF EXISTS post_revisions_redaction_consistent,
  DROP CONSTRAINT IF EXISTS post_revisions_restore_pointer_consistent,
  DROP CONSTRAINT IF EXISTS post_revisions_changed_fields_allowed,
  DROP CONSTRAINT IF EXISTS post_revisions_origin_allowed,
  DROP CONSTRAINT IF EXISTS post_revisions_operation_allowed,
  DROP CONSTRAINT IF EXISTS post_revisions_revision_no_positive;

ALTER TABLE post_revisions
  DROP COLUMN IF EXISTS redaction_reason,
  DROP COLUMN IF EXISTS redacted_by_user_id,
  DROP COLUMN IF EXISTS redacted_at,
  DROP COLUMN IF EXISTS snapshot_complete,
  DROP COLUMN IF EXISTS restored_from_revision_id,
  DROP COLUMN IF EXISTS committed_at,
  DROP COLUMN IF EXISTS attachment_ids,
  DROP COLUMN IF EXISTS changed_fields,
  DROP COLUMN IF EXISTS origin,
  DROP COLUMN IF EXISTS operation,
  DROP COLUMN IF EXISTS actor_user_id,
  DROP COLUMN IF EXISTS revision_no;

-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = current_schema()
      AND table_name = 'post_revisions'
      AND column_name = 'superseded_by_user_id'
  ) AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = current_schema()
      AND table_name = 'post_revisions'
      AND column_name = 'edited_by_user_id'
  ) THEN
    ALTER TABLE post_revisions RENAME COLUMN superseded_by_user_id TO edited_by_user_id;
  END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE posts
  DROP COLUMN IF EXISTS current_revision;
