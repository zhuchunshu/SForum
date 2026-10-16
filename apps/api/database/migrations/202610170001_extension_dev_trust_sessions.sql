-- +goose Up
-- +sforum OnlineSafe
-- 开发期信任会话：super_admin 在开发环境对 extensions/dev 包一次性确认后，
-- CLI watch 循环内每个新 digest 自动写入 exact live grant（逐条审计）。
-- 生产环境没有 dev 包根，且所有写入路径都要求 APP_ENV=development，fail-closed。
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '1min';

CREATE TABLE extension_dev_trust_sessions (
  id TEXT PRIMARY KEY
    CHECK (id ~ '^[0-9a-f]{32}$'),
  extension_id TEXT NOT NULL REFERENCES extensions(id) ON DELETE CASCADE,
  package_slug TEXT NOT NULL CHECK (package_slug <> ''),
  root_path TEXT NOT NULL CHECK (root_path <> ''),
  actor_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL
    CHECK (expires_at > granted_at),
  last_package_digest TEXT NOT NULL DEFAULT ''
    CHECK (last_package_digest = '' OR last_package_digest ~ '^[0-9a-f]{64}$'),
  revoked_at TIMESTAMPTZ,
  revoked_reason TEXT NOT NULL DEFAULT '',
  CHECK (revoked_at IS NULL OR revoked_at >= granted_at)
);

-- 一个包同时最多一个未撤销会话（索引谓词只能用 IMMUTABLE 表达式，
-- expires_at 的时间判定由 GrantDevArtifact/Active 查询与创建时的过期回收完成）。
CREATE UNIQUE INDEX extension_dev_trust_sessions_active_slug_idx
  ON extension_dev_trust_sessions (package_slug)
  WHERE revoked_at IS NULL;
CREATE INDEX extension_dev_trust_sessions_extension_idx
  ON extension_dev_trust_sessions (extension_id, granted_at DESC);
CREATE INDEX extension_dev_trust_sessions_actor_idx
  ON extension_dev_trust_sessions (actor_user_id, granted_at DESC);

-- +goose Down
DROP TABLE IF EXISTS extension_dev_trust_sessions;
