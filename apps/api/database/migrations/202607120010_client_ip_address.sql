-- +goose Up
-- 登录会话与内容创作落库真实客户端 IP（全文，供管理/风控）。
-- 用户端设备列表仍用 ip_prefix 脱敏展示；公开 API 不返回全文 IP。

ALTER TABLE user_sessions
  ADD COLUMN IF NOT EXISTS ip_address TEXT NOT NULL DEFAULT '';

ALTER TABLE topics
  ADD COLUMN IF NOT EXISTS ip_address TEXT NOT NULL DEFAULT '';

ALTER TABLE comments
  ADD COLUMN IF NOT EXISTS ip_address TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE comments DROP COLUMN IF EXISTS ip_address;
ALTER TABLE topics DROP COLUMN IF EXISTS ip_address;
ALTER TABLE user_sessions DROP COLUMN IF EXISTS ip_address;
