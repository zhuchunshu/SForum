-- +goose Up
-- V3 P4：queued plugin job payload 迁移必须插入 exact replacement，再取消旧 River row。
-- Host ledger 绑定 old row、迁移声明、source/target contract 与双方 trust grant，确保崩溃重试
-- 只能复用同一 replacement。River 会清理终态 row，因此这里刻意不建立 River/extension 外键。
CREATE TABLE extension_plugin_job_migrations (
  old_job_id BIGINT PRIMARY KEY CHECK (old_job_id > 0),
  extension_id TEXT NOT NULL CHECK (extension_id <> ''),
  migration_id TEXT NOT NULL CHECK (migration_id <> ''),
  source_contract JSONB NOT NULL
    CHECK (jsonb_typeof(source_contract) = 'object' AND source_contract <> '{}'::jsonb),
  source_trust_grant_id TEXT NOT NULL CHECK (source_trust_grant_id <> ''),
  target_contract JSONB NOT NULL
    CHECK (jsonb_typeof(target_contract) = 'object' AND target_contract <> '{}'::jsonb),
  target_trust_grant_id TEXT NOT NULL CHECK (target_trust_grant_id <> ''),
  new_job_id BIGINT UNIQUE CHECK (new_job_id > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT transaction_timestamp(),
  completed_at TIMESTAMPTZ,
  CHECK ((new_job_id IS NULL AND completed_at IS NULL)
      OR (new_job_id IS NOT NULL AND completed_at IS NOT NULL)),
  CHECK (completed_at IS NULL OR completed_at >= created_at)
);

CREATE INDEX extension_plugin_job_migrations_extension_created_idx
  ON extension_plugin_job_migrations (extension_id, created_at DESC, old_job_id DESC);

-- +goose Down
-- 只移除 Host replacement ledger；不修改或删除任何 River job、extension 或插件数据。
DROP TABLE IF EXISTS extension_plugin_job_migrations;
