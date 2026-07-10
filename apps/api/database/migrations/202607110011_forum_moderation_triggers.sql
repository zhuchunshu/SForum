-- +goose Up
ALTER TABLE topics
  ADD COLUMN moderation_triggers JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE comments
  ADD COLUMN moderation_triggers JSONB NOT NULL DEFAULT '[]'::jsonb;

-- +goose Down
ALTER TABLE comments DROP COLUMN IF EXISTS moderation_triggers;
ALTER TABLE topics DROP COLUMN IF EXISTS moderation_triggers;
