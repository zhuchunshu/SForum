-- +goose Up
CREATE INDEX attachments_active_orphan_cleanup_idx
  ON attachments (created_at)
  WHERE status = 'active' AND reference_count = 0;

-- +goose Down
DROP INDEX IF EXISTS attachments_active_orphan_cleanup_idx;
