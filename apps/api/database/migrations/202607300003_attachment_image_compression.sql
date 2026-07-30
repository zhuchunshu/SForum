-- +goose Up
CREATE TABLE attachment_variants (
  id BIGSERIAL PRIMARY KEY,
  attachment_id BIGINT NOT NULL REFERENCES attachments(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  provider TEXT NOT NULL,
  object_key TEXT NOT NULL,
  content_type TEXT NOT NULL,
  size_bytes BIGINT NOT NULL CHECK (size_bytes > 0),
  sha256 TEXT NOT NULL,
  image_width INTEGER NOT NULL CHECK (image_width > 0),
  image_height INTEGER NOT NULL CHECK (image_height > 0),
  source_sha256 TEXT NOT NULL,
  policy_digest TEXT NOT NULL,
  compression_strength INTEGER NOT NULL CHECK (compression_strength BETWEEN 0 AND 100),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (attachment_id, name),
  UNIQUE (provider, object_key)
);

CREATE INDEX attachment_variants_attachment_idx
  ON attachment_variants (attachment_id, name);

CREATE TABLE attachment_compression_tasks (
  id BIGSERIAL PRIMARY KEY,
  attachment_id BIGINT NOT NULL REFERENCES attachments(id) ON DELETE CASCADE,
  variant_name TEXT NOT NULL,
  source_sha256 TEXT NOT NULL,
  policy_digest TEXT NOT NULL,
  compression_strength INTEGER NOT NULL CHECK (compression_strength BETWEEN 0 AND 100),
  status TEXT NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'running', 'succeeded', 'skipped', 'failed')),
  attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
  error_code TEXT NOT NULL DEFAULT '',
  available_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (attachment_id, variant_name, policy_digest)
);

CREATE INDEX attachment_compression_tasks_pending_idx
  ON attachment_compression_tasks (available_at, id)
  WHERE status IN ('pending', 'failed');

-- +goose Down
DROP TABLE IF EXISTS attachment_compression_tasks;
DROP TABLE IF EXISTS attachment_variants;
