-- +goose Up
-- 举报与审核队列。
CREATE TABLE moderation_reports (
  id BIGSERIAL PRIMARY KEY,
  reporter_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  target_type TEXT NOT NULL CHECK (target_type IN ('topic', 'comment')),
  target_id BIGINT NOT NULL,
  reason_code TEXT NOT NULL CHECK (reason_code IN ('spam', 'abuse', 'illegal', 'off_topic', 'other')),
  body TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'reviewing', 'resolved', 'rejected')),
  reviewer_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  review_note TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  resolved_at TIMESTAMPTZ
);

-- 同一举报者对同一目标只能有一条 open 状态的举报。
CREATE UNIQUE INDEX moderation_reports_open_unique_idx
  ON moderation_reports (reporter_user_id, target_type, target_id)
  WHERE status = 'open' AND reporter_user_id IS NOT NULL;

CREATE INDEX moderation_reports_status_idx ON moderation_reports (status, created_at DESC);
CREATE INDEX moderation_reports_target_idx ON moderation_reports (target_type, target_id);

-- +goose Down
DROP INDEX IF EXISTS moderation_reports_target_idx;
DROP INDEX IF EXISTS moderation_reports_status_idx;
DROP INDEX IF EXISTS moderation_reports_open_unique_idx;
DROP TABLE IF EXISTS moderation_reports;
