-- +goose Up
-- T1D: bind recent-auth to (user_id, session_fingerprint).
--
-- password_hash 保持 migration 055 / 0001 建立的 NOT NULL 不变量；
-- external-only 账号以「无 user_credentials 行」表示，本迁移不得改动 password_hash。
-- 见 T8C / plans/2026-07-27-github-social-login-builtin-plugin.md。

-- Rebuild recent-auth as session-bound. Intermediate M1 rows were user-scoped
-- only and must not satisfy step-up for a different session of the same user.
DROP INDEX IF EXISTS user_recent_auth_expires_idx;
DROP TABLE IF EXISTS user_recent_auth;

CREATE TABLE user_recent_auth (
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  session_fingerprint TEXT NOT NULL
    CHECK (session_fingerprint ~ '^[a-f0-9]{64}$'),
  auth_method TEXT NOT NULL CHECK (auth_method IN ('password','external')),
  auth_provider_id TEXT,
  authenticated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, session_fingerprint)
);
CREATE INDEX user_recent_auth_expires_idx ON user_recent_auth (expires_at);

-- +goose Down
DROP INDEX IF EXISTS user_recent_auth_expires_idx;
DROP TABLE IF EXISTS user_recent_auth;

-- Down 回退到 055 的 session-bound 形态之前的 user-scoped 表时，
-- 不得 DROP password_hash NOT NULL（055/0001 不变量必须保留）。
CREATE TABLE user_recent_auth (
  user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  auth_method TEXT NOT NULL CHECK (auth_method IN ('password','external')),
  auth_provider_id TEXT,
  authenticated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX user_recent_auth_expires_idx ON user_recent_auth (expires_at);
