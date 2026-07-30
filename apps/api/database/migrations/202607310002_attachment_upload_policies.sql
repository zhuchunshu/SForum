-- +goose Up
CREATE TABLE attachment_role_upload_policies (
  role_id BIGINT PRIMARY KEY REFERENCES roles(id) ON DELETE CASCADE,
  max_file_size_bytes BIGINT NOT NULL CHECK (max_file_size_bytes > 0),
  updated_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE attachment_user_upload_policies (
  user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  max_file_size_bytes BIGINT NOT NULL CHECK (max_file_size_bytes > 0),
  updated_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO permissions (key, module, description)
VALUES (
  'attachment.upload_policy.manage',
  'attachment',
  'Manage per-role and per-user attachment upload size policies.'
)
ON CONFLICT (key) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, 'attachment.upload_policy.manage'
FROM roles
WHERE roles.key IN ('super_admin', 'operator')
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_permissions
WHERE permission_key = 'attachment.upload_policy.manage';

DELETE FROM permissions
WHERE key = 'attachment.upload_policy.manage';

DROP TABLE IF EXISTS attachment_user_upload_policies;
DROP TABLE IF EXISTS attachment_role_upload_policies;
