-- +goose Up
CREATE TABLE extension_theme_releases (
  id BIGSERIAL PRIMARY KEY,
  extension_id TEXT NOT NULL REFERENCES extensions(id) ON DELETE CASCADE,
  extension_version TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('queued', 'building', 'built', 'activating', 'active', 'failed', 'rolled_back')),
  layer_path TEXT NOT NULL,
  artifact_path TEXT NOT NULL DEFAULT '',
  server_entry TEXT NOT NULL DEFAULT '',
  message TEXT NOT NULL DEFAULT '',
  build_log TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  activated_at TIMESTAMPTZ
);

CREATE INDEX extension_theme_releases_extension_created_idx
  ON extension_theme_releases (extension_id, created_at DESC, id DESC);

CREATE UNIQUE INDEX extension_theme_releases_single_active_idx
  ON extension_theme_releases ((status))
  WHERE status = 'active';

-- +goose Down
DROP INDEX IF EXISTS extension_theme_releases_single_active_idx;
DROP INDEX IF EXISTS extension_theme_releases_extension_created_idx;
DROP TABLE IF EXISTS extension_theme_releases;
