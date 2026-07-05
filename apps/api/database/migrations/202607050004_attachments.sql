-- +goose Up
CREATE TABLE attachments (
  id BIGSERIAL PRIMARY KEY,
  public_id TEXT NOT NULL UNIQUE,
  owner_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  provider TEXT NOT NULL,
  object_key TEXT NOT NULL,
  original_name TEXT NOT NULL,
  content_type TEXT NOT NULL,
  extension TEXT NOT NULL DEFAULT '',
  size_bytes BIGINT NOT NULL CHECK (size_bytes >= 0),
  sha256 TEXT NOT NULL,
  image_width INTEGER,
  image_height INTEGER,
  visibility TEXT NOT NULL DEFAULT 'public' CHECK (visibility IN ('public', 'private')),
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'deleted')),
  reference_count INTEGER NOT NULL DEFAULT 0 CHECK (reference_count >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  UNIQUE (provider, object_key)
);

CREATE INDEX attachments_owner_created_idx ON attachments (owner_user_id, created_at DESC);
CREATE INDEX attachments_provider_status_idx ON attachments (provider, status);
CREATE INDEX attachments_content_type_idx ON attachments (content_type);
CREATE INDEX attachments_deleted_cleanup_idx
  ON attachments (deleted_at)
  WHERE status = 'deleted' AND reference_count = 0;

CREATE TABLE attachment_references (
  id BIGSERIAL PRIMARY KEY,
  attachment_id BIGINT NOT NULL REFERENCES attachments(id) ON DELETE CASCADE,
  resource_type TEXT NOT NULL,
  resource_id BIGINT NOT NULL,
  context TEXT NOT NULL DEFAULT '',
  created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (attachment_id, resource_type, resource_id, context)
);

CREATE INDEX attachment_references_resource_idx
  ON attachment_references (resource_type, resource_id);

INSERT INTO permissions (key, module, description)
VALUES
  ('attachment.upload', 'attachment', 'Upload attachments.'),
  ('attachment.manage', 'attachment', 'Manage uploaded attachments.'),
  ('attachment.settings.manage', 'attachment', 'Manage attachment storage and upload settings.')
ON CONFLICT (key) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, permissions.key
FROM roles
CROSS JOIN permissions
WHERE roles.key = 'super_admin'
  AND permissions.key IN ('attachment.upload', 'attachment.manage', 'attachment.settings.manage')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, 'attachment.upload'
FROM roles
WHERE roles.key = 'member'
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_permissions
WHERE permission_key IN ('attachment.upload', 'attachment.manage', 'attachment.settings.manage');

DELETE FROM permissions
WHERE key IN ('attachment.upload', 'attachment.manage', 'attachment.settings.manage');

DROP TABLE IF EXISTS attachment_references;
DROP INDEX IF EXISTS attachments_deleted_cleanup_idx;
DROP INDEX IF EXISTS attachments_content_type_idx;
DROP INDEX IF EXISTS attachments_provider_status_idx;
DROP INDEX IF EXISTS attachments_owner_created_idx;
DROP TABLE IF EXISTS attachments;
