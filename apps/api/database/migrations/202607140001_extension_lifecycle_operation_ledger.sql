-- +goose Up
-- V3 P4：Host 拥有的 lifecycle v2 逻辑操作账本。扩展卸载后仍保留工件、授权和审计证据，
-- 因此 extension_id/version 采用不可变快照，不对 extensions 建立级联外键。
CREATE TABLE extension_lifecycle_operations (
  id BIGSERIAL PRIMARY KEY,
  extension_id TEXT NOT NULL CHECK (extension_id <> ''),
  extension_version TEXT NOT NULL CHECK (extension_version <> ''),
  package_digest TEXT NOT NULL CHECK (package_digest ~ '^[0-9a-f]{64}$'),
  artifact_digests JSONB NOT NULL DEFAULT '{}'::jsonb
    CHECK (jsonb_typeof(artifact_digests) = 'object'),
  operation TEXT NOT NULL
    CHECK (operation IN ('install', 'enable', 'disable', 'upgrade', 'rollback', 'uninstall')),
  state TEXT NOT NULL DEFAULT 'planned'
    CHECK (state IN (
      'planned', 'migrating', 'starting', 'healthy', 'registering',
      'enabled', 'draining', 'uninstalling', 'failed', 'recovery'
    )),
  plan_version TEXT NOT NULL CHECK (plan_version <> ''),
  idempotency_key TEXT NOT NULL
    CHECK (octet_length(idempotency_key) BETWEEN 1 AND 512),
  request_fingerprint TEXT NOT NULL
    CHECK (request_fingerprint ~ '^[0-9a-f]{64}$'),
  authority_type TEXT NOT NULL
    CHECK (authority_type IN ('builtin', 'trust_grant')),
  trust_grant_id BIGINT REFERENCES extension_trust_grants(id) ON DELETE RESTRICT,
  authority_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb
    CHECK (jsonb_typeof(authority_snapshot) = 'object'),
  requested_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  -- audit_events 按保留期清理；只保留不可复用的关联 id，不能用 FK 阻断全站清理。
  audit_event_id BIGINT,
  removal_mode TEXT
    CHECK (removal_mode IN ('preserve', 'export_then_remove', 'complete_removal')),
  forced BOOLEAN NOT NULL DEFAULT FALSE,
  attempt_count INTEGER NOT NULL DEFAULT 1 CHECK (attempt_count > 0),
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  current_step_id TEXT NOT NULL DEFAULT '',
  checkpoint JSONB NOT NULL DEFAULT '{}'::jsonb
    CHECK (jsonb_typeof(checkpoint) = 'object'),
  progress JSONB NOT NULL DEFAULT '{}'::jsonb
    CHECK (jsonb_typeof(progress) = 'object'),
  terminal_result TEXT
    CHECK (terminal_result IN ('succeeded', 'failed', 'cancelled', 'skipped')),
  result_document JSONB
    CHECK (result_document IS NULL OR jsonb_typeof(result_document) = 'object'),
  error_code TEXT NOT NULL DEFAULT '',
  error_reason TEXT NOT NULL DEFAULT '',
  error_message TEXT NOT NULL DEFAULT '',
  error_retryable BOOLEAN NOT NULL DEFAULT FALSE,
  error_retry_after TIMESTAMPTZ,
  error_metadata JSONB NOT NULL DEFAULT '{}'::jsonb
    CHECK (jsonb_typeof(error_metadata) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  UNIQUE (extension_id, idempotency_key),
  CHECK ((authority_type = 'builtin' AND trust_grant_id IS NULL)
      OR (authority_type = 'trust_grant' AND trust_grant_id IS NOT NULL
          AND authority_snapshot <> '{}'::jsonb)),
  CHECK ((operation = 'uninstall' AND removal_mode IS NOT NULL)
      OR (operation <> 'uninstall' AND removal_mode IS NULL)),
  CHECK (forced = FALSE OR operation = 'uninstall'),
  CHECK (state <> 'failed' OR (error_code <> '' AND error_reason <> '')),
  CHECK ((completed_at IS NULL AND terminal_result IS NULL)
      OR (completed_at IS NOT NULL AND terminal_result IS NOT NULL)),
  CHECK (terminal_result IS DISTINCT FROM 'failed' OR state = 'failed'),
  CHECK (terminal_result IS NULL OR terminal_result NOT IN ('failed', 'cancelled')
      OR (error_code <> '' AND error_reason <> '')),
  CHECK (updated_at >= created_at),
  CHECK (started_at IS NULL OR started_at >= created_at),
  CHECK (completed_at IS NULL OR completed_at >= COALESCE(started_at, created_at))
);

-- 同一扩展只能有一个未完成逻辑操作；崩溃恢复必须复用原 idempotency_key。
CREATE UNIQUE INDEX extension_lifecycle_operations_one_open_idx
  ON extension_lifecycle_operations (extension_id)
  WHERE completed_at IS NULL;
CREATE INDEX extension_lifecycle_operations_extension_created_idx
  ON extension_lifecycle_operations (extension_id, created_at DESC, id DESC);
CREATE INDEX extension_lifecycle_operations_state_updated_idx
  ON extension_lifecycle_operations (state, updated_at, id);

-- 每个稳定 step_id 可产生多个 attempt；终态 attempt 保留，恢复时插入下一 attempt。
CREATE TABLE extension_lifecycle_steps (
  id BIGSERIAL PRIMARY KEY,
  operation_id BIGINT NOT NULL
    REFERENCES extension_lifecycle_operations(id) ON DELETE CASCADE,
  step_id TEXT NOT NULL CHECK (octet_length(step_id) BETWEEN 1 AND 512),
  lifecycle_action TEXT NOT NULL
    CHECK (lifecycle_action IN (
      'install.plan', 'install', 'enable', 'disable',
      'upgrade.plan', 'upgrade.before', 'upgrade.after', 'rollback',
      'uninstall.plan', 'uninstall', 'uninstall.after'
    )),
  plan_version TEXT NOT NULL CHECK (plan_version <> ''),
  attempt INTEGER NOT NULL DEFAULT 1 CHECK (attempt > 0),
  status TEXT NOT NULL DEFAULT 'planned'
    CHECK (status IN ('planned', 'running', 'waiting', 'succeeded', 'failed', 'cancelled', 'skipped')),
  checkpoint TEXT NOT NULL DEFAULT '',
  completed_units BIGINT NOT NULL DEFAULT 0 CHECK (completed_units >= 0),
  total_units BIGINT NOT NULL DEFAULT 0 CHECK (total_units >= 0),
  progress_message TEXT NOT NULL DEFAULT '',
  input_document JSONB
    CHECK (input_document IS NULL OR jsonb_typeof(input_document) = 'object'),
  result_document JSONB
    CHECK (result_document IS NULL OR jsonb_typeof(result_document) = 'object'),
  error_code TEXT NOT NULL DEFAULT '',
  error_reason TEXT NOT NULL DEFAULT '',
  error_message TEXT NOT NULL DEFAULT '',
  error_retryable BOOLEAN NOT NULL DEFAULT FALSE,
  error_retry_after TIMESTAMPTZ,
  error_metadata JSONB NOT NULL DEFAULT '{}'::jsonb
    CHECK (jsonb_typeof(error_metadata) = 'object'),
  skip_reason TEXT NOT NULL DEFAULT '',
  forced BOOLEAN NOT NULL DEFAULT FALSE,
  actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  -- 与 operation 相同，保留关联 id 但不阻断 audit retention。
  audit_event_id BIGINT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  UNIQUE (operation_id, step_id, attempt),
  CHECK (total_units = 0 OR completed_units <= total_units),
  CHECK ((status IN ('planned', 'running', 'waiting') AND completed_at IS NULL)
      OR (status IN ('succeeded', 'failed', 'cancelled', 'skipped') AND completed_at IS NOT NULL)),
  CHECK (status NOT IN ('running', 'waiting') OR started_at IS NOT NULL),
  CHECK (status NOT IN ('failed', 'cancelled') OR (error_code <> '' AND error_reason <> '')),
  CHECK (status <> 'skipped' OR skip_reason <> ''),
  CHECK (updated_at >= created_at),
  CHECK (started_at IS NULL OR started_at >= created_at),
  CHECK (completed_at IS NULL OR completed_at >= COALESCE(started_at, created_at))
);

CREATE UNIQUE INDEX extension_lifecycle_steps_one_open_attempt_idx
  ON extension_lifecycle_steps (operation_id, step_id)
  WHERE status IN ('planned', 'running', 'waiting');
CREATE INDEX extension_lifecycle_steps_operation_created_idx
  ON extension_lifecycle_steps (operation_id, created_at, id);
CREATE INDEX extension_lifecycle_steps_retryable_idx
  ON extension_lifecycle_steps (error_retry_after, operation_id, id)
  WHERE status = 'failed' AND error_retryable = TRUE;

-- +goose Down
-- 只移除 Host lifecycle 账本；绝不删除 extensions、extension_migration_ledger
-- 或任何插件拥有/管理的 schema、表、文件和外部资源。
DROP TABLE IF EXISTS extension_lifecycle_steps;
DROP TABLE IF EXISTS extension_lifecycle_operations;
