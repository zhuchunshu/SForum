-- +goose Up
-- 内置角色模板：moderator / operator / tech_admin。
-- 系统角色、不可删除；权限可在后台调整。升级时只补齐缺失权限，不覆盖已有别名与额外权限。

INSERT INTO roles (key, alias, description, is_system, is_default, is_deletable, is_enabled)
VALUES
  (
    'moderator',
    '版主',
    '内容运营：审核、锁帖/置顶、编辑删除任意主题与回复，并可封禁用户。',
    TRUE, FALSE, FALSE, TRUE
  ),
  (
    'operator',
    '站点运营',
    '站点日常运营：用户与内容结构、站点/邮件/外观等设置；不含技术发布与权限例外。',
    TRUE, FALSE, FALSE, TRUE
  ),
  (
    'tech_admin',
    '技术管理',
    '技术运维：扩展与发布、搜索索引、后台任务、数据库浏览与附件存储设置。',
    TRUE, FALSE, FALSE, TRUE
  )
ON CONFLICT (key) DO UPDATE SET
  is_system = TRUE,
  is_deletable = FALSE,
  is_enabled = TRUE,
  updated_at = now();

-- moderator：内容运营包
INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, permissions.key
FROM roles
CROSS JOIN (VALUES
  ('admin.access'),
  ('moderation.review'),
  ('topic.lock'),
  ('topic.pin'),
  ('topic.edit_any'),
  ('topic.delete_any'),
  ('post.edit_any'),
  ('post.delete_any'),
  ('user.ban')
) AS permissions(key)
WHERE roles.key = 'moderator'
ON CONFLICT DO NOTHING;

-- operator：站点运营包（不含 user.permission_override / extension.release / database / jobs）
INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, permissions.key
FROM roles
CROSS JOIN (VALUES
  ('admin.access'),
  ('user.view'),
  ('user.manage'),
  ('settings.site.manage'),
  ('settings.mail.manage'),
  ('settings.avatar.manage'),
  ('settings.appearance.manage'),
  ('forum.settings.manage'),
  ('seo.manage'),
  ('category.manage'),
  ('tag.manage'),
  ('attachment.manage'),
  ('attachment.settings.manage')
) AS permissions(key)
WHERE roles.key = 'operator'
ON CONFLICT DO NOTHING;

-- tech_admin：技术运维包
INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, permissions.key
FROM roles
CROSS JOIN (VALUES
  ('admin.access'),
  ('extension.view'),
  ('extension.plugin.manage'),
  ('extension.theme.manage'),
  ('extension.release.manage'),
  ('jobs.view'),
  ('jobs.manage'),
  ('search.manage'),
  ('database.manage'),
  ('attachment.settings.manage')
) AS permissions(key)
WHERE roles.key = 'tech_admin'
ON CONFLICT DO NOTHING;

-- +goose Down
-- 仅移除本迁移写入的模板角色权限与角色本身。
-- 若站点已把用户挂到这些角色上，Down 前需先解绑；user_roles 对 roles 为 RESTRICT。

DELETE FROM role_permissions
WHERE role_id IN (
  SELECT id FROM roles WHERE key IN ('moderator', 'operator', 'tech_admin')
);

DELETE FROM roles
WHERE key IN ('moderator', 'operator', 'tech_admin')
  AND is_system = TRUE
  AND is_deletable = FALSE;
