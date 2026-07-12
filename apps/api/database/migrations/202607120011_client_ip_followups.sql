-- +goose Up
-- 真实 IP 后续能力：查看权限、编辑 IP、版主模板授权。

-- 全文 IP 仅对持有 moderation.view_ip 的 actor 出现在审核/管理 API。
INSERT INTO permissions (key, module, description) VALUES
  ('moderation.view_ip', 'moderation', 'View full client IP addresses on content and sessions for moderation.')
ON CONFLICT (key) DO UPDATE SET module = EXCLUDED.module, description = EXCLUDED.description;

INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, 'moderation.view_ip'
FROM roles
WHERE roles.key = 'super_admin'
ON CONFLICT DO NOTHING;

-- 版主默认需要看 IP 做风控；已有 moderator 角色幂等补权。
INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, 'moderation.view_ip'
FROM roles
WHERE roles.key = 'moderator'
ON CONFLICT DO NOTHING;

-- 持有 moderation.review 的自定义角色升级不丢「看 IP」能力（运营可再收回）。
INSERT INTO role_permissions (role_id, permission_key)
SELECT DISTINCT rp.role_id, 'moderation.view_ip'
FROM role_permissions rp
WHERE rp.permission_key = 'moderation.review'
ON CONFLICT DO NOTHING;

-- 编辑时记录 last_edit_ip（创建 IP 保持不变）。
ALTER TABLE topics
  ADD COLUMN IF NOT EXISTS last_edit_ip TEXT NOT NULL DEFAULT '';

ALTER TABLE comments
  ADD COLUMN IF NOT EXISTS last_edit_ip TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE comments DROP COLUMN IF EXISTS last_edit_ip;
ALTER TABLE topics DROP COLUMN IF EXISTS last_edit_ip;

DELETE FROM role_permissions WHERE permission_key = 'moderation.view_ip';
DELETE FROM permissions WHERE key = 'moderation.view_ip';
