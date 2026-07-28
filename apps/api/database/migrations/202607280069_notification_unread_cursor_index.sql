-- +goose Up
-- Inbox pagination uses the monotonic id cursor, including unread-only scans.
DROP INDEX IF EXISTS notifications_recipient_unread_idx;
CREATE INDEX notifications_recipient_unread_idx
  ON notifications (recipient_user_id, id DESC)
  WHERE read_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS notifications_recipient_unread_idx;
CREATE INDEX notifications_recipient_unread_idx
  ON notifications (recipient_user_id, created_at DESC)
  WHERE read_at IS NULL;
