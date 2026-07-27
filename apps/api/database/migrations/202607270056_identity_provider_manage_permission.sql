-- +goose Up
-- Host-owned external auth provider management permission.
-- 见 plans/2026-07-27-github-social-login-builtin-plugin.md M3 与
-- decisions/2026-07-27-github-social-login-builtin-v1.md。
-- 注：executable trust 仍为 super_admin-only，即便 provider 激活被委托。

INSERT INTO permissions (key, module, description)
VALUES ('identity.provider.manage', 'identity',
        'Activate, order, probe, and reset Host-owned external auth providers (login/registration/link). Executable trust remains super_admin-only.')
ON CONFLICT (key) DO NOTHING;

-- super_admin 通过 policy bypass 拥有全部权限，这里仍显式 seed 以保证 catalog 一致。
INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, 'identity.provider.manage'
FROM roles
WHERE roles.key IN ('super_admin', 'operator')
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_permissions
WHERE permission_key = 'identity.provider.manage';

DELETE FROM permissions
WHERE key = 'identity.provider.manage';
