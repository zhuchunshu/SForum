-- +goose Up
-- 扩展版本 GC（Version GC）：显式、可审计、fail-closed 的历史版本回收。
-- 1) extension_version_gc_removals：每次成功回收的持久锚点（对齐
--    extension_missing_artifact_removals 的审计锚点模式），audit_events 按
--    保留期清理后这里仍保留逐条结果。
-- 2) extension_gc_confirmations：一次性执行确认 token（只存 sha256 摘要，
--    明文只在 dry-run 输出返回给操作者），token 与计算出的回收计划摘要绑定，
--    计划漂移即失效。

CREATE TABLE extension_version_gc_removals (
  id BIGSERIAL PRIMARY KEY,
  extension_id TEXT NOT NULL REFERENCES extensions(id) ON DELETE RESTRICT,
  extension_version TEXT NOT NULL CHECK (extension_version <> ''),
  package_digest TEXT NOT NULL CHECK (package_digest ~ '^[0-9a-f]{64}$'),
  package_path TEXT NOT NULL,
  bytes_freed BIGINT NOT NULL DEFAULT 0 CHECK (bytes_freed >= 0),
  requested_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX extension_version_gc_removals_created_idx
  ON extension_version_gc_removals (created_at DESC);
CREATE INDEX extension_version_gc_removals_extension_idx
  ON extension_version_gc_removals (extension_id, created_at DESC);

CREATE TABLE extension_gc_confirmations (
  id BIGSERIAL PRIMARY KEY,
  token_hash TEXT NOT NULL UNIQUE CHECK (token_hash ~ '^[0-9a-f]{64}$'),
  plan_digest TEXT NOT NULL CHECK (plan_digest ~ '^[0-9a-f]{64}$'),
  candidate_count INTEGER NOT NULL CHECK (candidate_count >= 0),
  options JSONB NOT NULL DEFAULT '{}'::jsonb
    CHECK (jsonb_typeof(options) = 'object'),
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (expires_at > created_at),
  CHECK (consumed_at IS NULL OR consumed_at >= created_at)
);

CREATE INDEX extension_gc_confirmations_created_idx
  ON extension_gc_confirmations (created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS extension_gc_confirmations;
DROP INDEX IF EXISTS extension_version_gc_removals_extension_idx;
DROP INDEX IF EXISTS extension_version_gc_removals_created_idx;
DROP TABLE IF EXISTS extension_version_gc_removals;
