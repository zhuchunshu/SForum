-- +goose Up
-- Preference CAS is separate from inbox realtime revisions. A user may save
-- settings without making the current notification stream appear changed.
CREATE TABLE notification_preference_revisions (
  user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  revision BIGINT NOT NULL DEFAULT 0 CHECK (revision >= 0),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS notification_preference_revisions;
