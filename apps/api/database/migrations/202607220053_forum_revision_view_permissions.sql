-- +goose Up
-- 内容版本历史 V1 M2：历史查看权限只授予 super_admin 与内置 moderator 模板。

INSERT INTO permissions (key, module, description) VALUES
  ('topic.revision.view_any', 'forum', 'Inspect any topic revision history.'),
  ('post.revision.view_any', 'forum', 'Inspect any comment revision history.')
ON CONFLICT (key) DO UPDATE SET
  module = EXCLUDED.module,
  description = EXCLUDED.description;

INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, permissions.key
FROM roles
CROSS JOIN permissions
WHERE roles.key = 'super_admin'
  AND permissions.key IN ('topic.revision.view_any', 'post.revision.view_any')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, permission_keys.key
FROM roles
CROSS JOIN (VALUES
  ('topic.revision.view_any'),
  ('post.revision.view_any')
) AS permission_keys(key)
WHERE roles.key = 'moderator'
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_permissions
WHERE permission_key IN ('topic.revision.view_any', 'post.revision.view_any');

DELETE FROM permissions
WHERE key IN ('topic.revision.view_any', 'post.revision.view_any');
