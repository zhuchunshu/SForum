-- +goose Up
-- 用户活跃会话/登录设备目录。既支撑「活跃设备列表 / 登录历史」展示，
-- 也支撑「下线单个设备 / 下线其他设备」的远程失效。
--
-- 安全约束：本表只存 opaque 的 sid（server 生成的会话标识，非认证凭证）
-- 与 session_hash（cookie session id 的 HMAC，仅审计关联用），绝不存原始
-- cookie session id 或任何可用于劫持会话的凭证。即使本表泄漏也无法登录。
CREATE TABLE user_sessions (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  -- server 生成的稳定 opaque 会话标识，与 cookie session id 解耦：
  -- cookie session id 每 24h Regenerate 轮换，但 sid 写入 session payload 后随负载保留，
  -- 因此设备列表不会因续期而分裂成「新设备」。
  sid TEXT NOT NULL,
  -- cookie session id 的 HMAC，仅用于关联 audit_events.metadata.sessionHash，不可逆推原 id。
  session_hash TEXT NOT NULL DEFAULT '',
  -- 由 User-Agent 解析出的展示名，如 "Chrome on macOS"。
  device_name TEXT NOT NULL DEFAULT '',
  -- 解析出的浏览器名与操作系统名，列表展示用，避免每次查询都重新解析 UA。
  browser TEXT NOT NULL DEFAULT '',
  os TEXT NOT NULL DEFAULT '',
  -- 截断后的原始 User-Agent，供后续重新解析或排查；不直接暴露给前端。
  user_agent_raw TEXT NOT NULL DEFAULT '',
  -- 脱敏后的 IP（保留前缀，如 "1.2.3.*"），仅展示用。
  ip_prefix TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  -- NULL 表示活跃；非空表示已下线（logout / revoke_device / revoke_others / max_exceeded / password_reset）。
  revoked_at TIMESTAMPTZ,
  revoke_reason TEXT NOT NULL DEFAULT ''
);

-- sid 是前端指定「下线哪一条」的唯一句柄，需唯一且按用户+sid 定位。
CREATE UNIQUE INDEX user_sessions_sid_idx ON user_sessions (sid);
-- 活跃设备列表按用户查询、按最后活跃时间倒序。
CREATE INDEX user_sessions_user_active_idx ON user_sessions (user_id, last_seen_at DESC) WHERE revoked_at IS NULL;
-- 登录历史按用户查询、按登录时间倒序。
CREATE INDEX user_sessions_user_history_idx ON user_sessions (user_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS user_sessions_user_history_idx;
DROP INDEX IF EXISTS user_sessions_user_active_idx;
DROP INDEX IF EXISTS user_sessions_sid_idx;
DROP TABLE IF EXISTS user_sessions;
