-- +goose Up
-- Host 自动升级受保护内置插件时使用的系统角色。
-- 该角色不授予 admin.access；它只让 builtin-auto 系统账号能够读取扩展目录
-- 并执行插件升级/重启。运营可以移除这里的权限来关闭自动更新通道。

INSERT INTO roles (key, alias, description, is_system, is_default, is_deletable, is_enabled)
VALUES (
  'builtin_auto',
  '内置插件自动更新',
  'Host 在启动时自动升级项目维护的内置插件；该角色不授予后台访问，不应分配给人类用户。',
  TRUE, FALSE, FALSE, TRUE
)
ON CONFLICT (key) DO UPDATE SET
  alias = EXCLUDED.alias,
  description = EXCLUDED.description,
  is_system = TRUE,
  is_deletable = FALSE,
  is_enabled = TRUE,
  updated_at = now();

INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, permissions.key
FROM roles
JOIN permissions ON permissions.key IN ('extension.view', 'extension.plugin.manage')
WHERE roles.key = 'builtin_auto'
ON CONFLICT DO NOTHING;

-- +goose Down
-- 先解绑系统账号，否则 user_roles 的 RESTRICT 会阻止删除角色。
DELETE FROM user_roles
WHERE role_id IN (SELECT id FROM roles WHERE key = 'builtin_auto');

DELETE FROM role_permissions
WHERE role_id IN (SELECT id FROM roles WHERE key = 'builtin_auto');

DELETE FROM roles
WHERE key = 'builtin_auto'
  AND is_system = TRUE
  AND is_deletable = FALSE;
