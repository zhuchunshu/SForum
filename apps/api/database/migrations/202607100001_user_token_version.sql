-- +goose Up
-- M8: 用户令牌版本号，用于密码重置/封禁后使旧会话失效。
-- session 存储创建时的版本号，校验时与用户当前版本号比对，不一致即视为会话失效。
ALTER TABLE users ADD COLUMN IF NOT EXISTS current_token_version BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS current_token_version;
