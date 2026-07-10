-- +goose Up
ALTER TABLE web_releases
  ADD COLUMN IF NOT EXISTS composition_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb;

-- +goose Down
ALTER TABLE web_releases DROP COLUMN IF EXISTS composition_snapshot;
