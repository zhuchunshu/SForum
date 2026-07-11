-- +goose Up
-- 撤销「user.manage 兼容展开」写入的 user.permission_override。
-- Phase1 曾把该子权限扩散到所有持有 user.manage 的角色；产品意图是个人权限例外
-- 必须显式单独授予，builtin operator / moderator / tech_admin 均不应持有。
-- super_admin 在代码层全能，不依赖 catalog 行，但保留其 role_permissions 行无害且便于审计展示。

DELETE FROM role_permissions
WHERE permission_key = 'user.permission_override'
  AND role_id IN (
    SELECT id FROM roles WHERE key <> 'super_admin'
  );

-- 用户直接 allow 覆盖中的同权限：仅当该用户同时持有 user.manage 角色权限且无独立
-- 运营意图时难以区分。安全策略：不批量删除 user_permission_overrides 中的
-- user.permission_override allow——那是管理员显式写给个人的例外，应保留。
-- 若某角色曾仅因迁移获得 role 级 grant，上面 DELETE 已覆盖。

-- +goose Down
-- 无法安全恢复「仅因 user.manage 迁移写入」的行；Down 为 no-op。
SELECT 1;
