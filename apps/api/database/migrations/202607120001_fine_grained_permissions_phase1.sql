-- +goose Up
-- 第 1 期细粒度权限：入库新 key、兼容父权限持有者、修正 member 主题 own 权限。

INSERT INTO permissions (key, module, description) VALUES
  ('topic.edit_own', 'forum', 'Edit own topics.'),
  ('topic.delete_own', 'forum', 'Delete own topics.'),
  ('settings.site.manage', 'admin', 'Manage core site identity, locale, verification, and security settings.'),
  ('settings.mail.manage', 'admin', 'Manage mail providers, notification policy, and delivery tests.'),
  ('settings.avatar.manage', 'admin', 'Manage avatar upload and default avatar settings.'),
  ('settings.appearance.manage', 'admin', 'Manage appearance theme and public chrome personalization.'),
  ('forum.settings.manage', 'forum', 'Manage forum runtime limits, reading, and behavior settings.'),
  ('user.view', 'identity', 'View user accounts without changing assignments.'),
  ('user.permission_override', 'identity', 'Edit per-user permission allow and deny overrides.'),
  ('extension.view', 'extension', 'View installed extensions, events, and contributions.'),
  ('extension.plugin.manage', 'extension', 'Enable, disable, and configure plugins.'),
  ('extension.theme.manage', 'extension', 'Activate and manage themes.'),
  ('extension.release.manage', 'extension', 'Build and activate trusted admin web releases.')
ON CONFLICT (key) DO UPDATE SET module = EXCLUDED.module, description = EXCLUDED.description;

-- 刷新既有父权限描述，标明其为兼容父权限。
UPDATE permissions SET description = 'Legacy parent: manage all non-SEO site settings groups.'
WHERE key = 'settings.manage';
UPDATE permissions SET description = 'Manage user accounts and role assignments.'
WHERE key = 'user.manage';
UPDATE permissions SET description = 'Legacy parent: manage all extension capabilities.'
WHERE key = 'extension.manage';
UPDATE permissions SET description = 'Edit own replies.'
WHERE key = 'post.edit_own';
UPDATE permissions SET description = 'Delete own replies.'
WHERE key = 'post.delete_own';

-- super_admin 获得全部新权限（幂等）。
INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, permissions.key
FROM roles
CROSS JOIN permissions
WHERE roles.key = 'super_admin'
  AND permissions.key IN (
    'topic.edit_own', 'topic.delete_own',
    'settings.site.manage', 'settings.mail.manage', 'settings.avatar.manage', 'settings.appearance.manage',
    'forum.settings.manage',
    'user.view', 'user.permission_override',
    'extension.view', 'extension.plugin.manage', 'extension.theme.manage', 'extension.release.manage'
  )
ON CONFLICT DO NOTHING;

-- member：作者主题编辑/删除与回复权限对齐。
INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, permissions.key
FROM roles
CROSS JOIN (VALUES ('topic.edit_own'), ('topic.delete_own')) AS permissions(key)
WHERE roles.key = 'member'
ON CONFLICT DO NOTHING;

-- 已持有 post.edit_own / post.delete_own 的角色补上主题 own（兼容旧语义）。
INSERT INTO role_permissions (role_id, permission_key)
SELECT DISTINCT rp.role_id, 'topic.edit_own'
FROM role_permissions rp
WHERE rp.permission_key = 'post.edit_own'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key)
SELECT DISTINCT rp.role_id, 'topic.delete_own'
FROM role_permissions rp
WHERE rp.permission_key = 'post.delete_own'
ON CONFLICT DO NOTHING;

-- 持有父权限的角色自动获得对应子权限（升级不收权）。
INSERT INTO role_permissions (role_id, permission_key)
SELECT DISTINCT rp.role_id, child.key
FROM role_permissions rp
CROSS JOIN (VALUES
  ('settings.site.manage'),
  ('settings.mail.manage'),
  ('settings.avatar.manage'),
  ('settings.appearance.manage'),
  ('forum.settings.manage')
) AS child(key)
WHERE rp.permission_key = 'settings.manage'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key)
SELECT DISTINCT rp.role_id, child.key
FROM role_permissions rp
CROSS JOIN (VALUES
  ('user.view'),
  ('user.permission_override')
) AS child(key)
WHERE rp.permission_key = 'user.manage'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key)
SELECT DISTINCT rp.role_id, child.key
FROM role_permissions rp
CROSS JOIN (VALUES
  ('extension.view'),
  ('extension.plugin.manage'),
  ('extension.theme.manage'),
  ('extension.release.manage')
) AS child(key)
WHERE rp.permission_key = 'extension.manage'
ON CONFLICT DO NOTHING;

-- 用户直接 allow 覆盖也展开子权限，避免仅靠 override 的管理员升级后丢能力。
INSERT INTO user_permission_overrides (user_id, permission_key, effect)
SELECT DISTINCT o.user_id, child.key, 'allow'
FROM user_permission_overrides o
CROSS JOIN (VALUES
  ('settings.site.manage'),
  ('settings.mail.manage'),
  ('settings.avatar.manage'),
  ('settings.appearance.manage'),
  ('forum.settings.manage')
) AS child(key)
WHERE o.permission_key = 'settings.manage' AND o.effect = 'allow'
ON CONFLICT DO NOTHING;

INSERT INTO user_permission_overrides (user_id, permission_key, effect)
SELECT DISTINCT o.user_id, child.key, 'allow'
FROM user_permission_overrides o
CROSS JOIN (VALUES
  ('user.view'),
  ('user.permission_override')
) AS child(key)
WHERE o.permission_key = 'user.manage' AND o.effect = 'allow'
ON CONFLICT DO NOTHING;

INSERT INTO user_permission_overrides (user_id, permission_key, effect)
SELECT DISTINCT o.user_id, child.key, 'allow'
FROM user_permission_overrides o
CROSS JOIN (VALUES
  ('extension.view'),
  ('extension.plugin.manage'),
  ('extension.theme.manage'),
  ('extension.release.manage')
) AS child(key)
WHERE o.permission_key = 'extension.manage' AND o.effect = 'allow'
ON CONFLICT DO NOTHING;

INSERT INTO user_permission_overrides (user_id, permission_key, effect)
SELECT DISTINCT o.user_id, 'topic.edit_own', 'allow'
FROM user_permission_overrides o
WHERE o.permission_key = 'post.edit_own' AND o.effect = 'allow'
ON CONFLICT DO NOTHING;

INSERT INTO user_permission_overrides (user_id, permission_key, effect)
SELECT DISTINCT o.user_id, 'topic.delete_own', 'allow'
FROM user_permission_overrides o
WHERE o.permission_key = 'post.delete_own' AND o.effect = 'allow'
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM user_permission_overrides
WHERE permission_key IN (
  'topic.edit_own', 'topic.delete_own',
  'settings.site.manage', 'settings.mail.manage', 'settings.avatar.manage', 'settings.appearance.manage',
  'forum.settings.manage',
  'user.view', 'user.permission_override',
  'extension.view', 'extension.plugin.manage', 'extension.theme.manage', 'extension.release.manage'
);

DELETE FROM role_permissions
WHERE permission_key IN (
  'topic.edit_own', 'topic.delete_own',
  'settings.site.manage', 'settings.mail.manage', 'settings.avatar.manage', 'settings.appearance.manage',
  'forum.settings.manage',
  'user.view', 'user.permission_override',
  'extension.view', 'extension.plugin.manage', 'extension.theme.manage', 'extension.release.manage'
);

DELETE FROM permissions
WHERE key IN (
  'topic.edit_own', 'topic.delete_own',
  'settings.site.manage', 'settings.mail.manage', 'settings.avatar.manage', 'settings.appearance.manage',
  'forum.settings.manage',
  'user.view', 'user.permission_override',
  'extension.view', 'extension.plugin.manage', 'extension.theme.manage', 'extension.release.manage'
);

UPDATE permissions SET description = 'Manage system settings.' WHERE key = 'settings.manage';
UPDATE permissions SET description = 'Manage user accounts and assignments.' WHERE key = 'user.manage';
UPDATE permissions SET description = 'Install and manage extensions and themes.' WHERE key = 'extension.manage';
UPDATE permissions SET description = 'Edit own posts.' WHERE key = 'post.edit_own';
UPDATE permissions SET description = 'Delete own posts.' WHERE key = 'post.delete_own';
