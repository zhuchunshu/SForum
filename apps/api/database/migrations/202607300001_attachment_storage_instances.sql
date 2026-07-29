-- +goose Up
CREATE TABLE attachment_storage_instances (
  id UUID PRIMARY KEY,
  extension_id TEXT NOT NULL REFERENCES extensions(id) ON DELETE RESTRICT,
  name TEXT NOT NULL,
  settings JSONB NOT NULL DEFAULT '{}'::jsonb,
  config_revision BIGINT NOT NULL DEFAULT 1 CHECK (config_revision > 0),
  status TEXT NOT NULL DEFAULT 'unverified' CHECK (status IN ('unverified', 'ready', 'error')),
  last_probe_status TEXT NOT NULL DEFAULT '',
  last_probe_message TEXT NOT NULL DEFAULT '',
  last_probe_at TIMESTAMPTZ,
  created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (extension_id, name)
);

CREATE INDEX attachment_storage_instances_extension_idx
  ON attachment_storage_instances (extension_id, created_at);

-- +goose Down
DROP TABLE IF EXISTS attachment_storage_instances;
