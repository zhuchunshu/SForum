-- +goose Up
-- V3 P4：multi-node step claim 使用数据库时间和单调 revision 做 CAS。
-- lease duration 上限属于 Host policy；DB 只保证租约元组完整且终态 step 不再持有租约。
-- Repository claim 必须匹配 open status + expected revision + 空/已过期 lease；heartbeat/release
-- 必须匹配 exact owner token + revision，且每次成功写都递增 revision，不能用时间相等判断所有权。
ALTER TABLE extension_lifecycle_steps
  ADD COLUMN lease_owner_token TEXT NOT NULL DEFAULT '',
  ADD COLUMN lease_expires_at TIMESTAMPTZ,
  ADD COLUMN lease_revision BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN lease_heartbeat_at TIMESTAMPTZ;

ALTER TABLE extension_lifecycle_steps
  ADD CONSTRAINT extension_lifecycle_steps_lease_revision_check
    CHECK (lease_revision >= 0),
  ADD CONSTRAINT extension_lifecycle_steps_lease_tuple_check
    CHECK (
      (lease_owner_token = '' AND lease_expires_at IS NULL AND lease_heartbeat_at IS NULL)
      OR
      (octet_length(lease_owner_token) BETWEEN 1 AND 512
       AND lease_expires_at IS NOT NULL
       AND lease_heartbeat_at IS NOT NULL
       AND lease_revision > 0)
    ),
  ADD CONSTRAINT extension_lifecycle_steps_lease_open_status_check
    CHECK (lease_owner_token = '' OR status IN ('planned', 'running', 'waiting')),
  ADD CONSTRAINT extension_lifecycle_steps_lease_window_check
    CHECK (lease_expires_at IS NULL OR lease_expires_at > lease_heartbeat_at);

-- NULL expiry（从未 claim/release 后）优先，其后是最早过期的 open attempt。
-- predicate 不包含 now()/transaction_timestamp()，避免依赖 volatile wall clock 的索引语义。
CREATE INDEX extension_lifecycle_steps_claimable_idx
  ON extension_lifecycle_steps (lease_expires_at NULLS FIRST, created_at, id)
  WHERE status IN ('planned', 'running', 'waiting');

CREATE INDEX extension_lifecycle_steps_lease_owner_idx
  ON extension_lifecycle_steps (lease_owner_token, lease_revision)
  WHERE lease_owner_token <> '';

-- +goose Down
-- 只移除 lease 索引和列；step/operation/audit/extension 历史均保持不变。
DROP INDEX IF EXISTS extension_lifecycle_steps_lease_owner_idx;
DROP INDEX IF EXISTS extension_lifecycle_steps_claimable_idx;

ALTER TABLE extension_lifecycle_steps
  DROP COLUMN IF EXISTS lease_heartbeat_at,
  DROP COLUMN IF EXISTS lease_revision,
  DROP COLUMN IF EXISTS lease_expires_at,
  DROP COLUMN IF EXISTS lease_owner_token;
