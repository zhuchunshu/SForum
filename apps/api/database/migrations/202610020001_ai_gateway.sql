-- +goose Up
CREATE TABLE ai_settings (
  singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),
  document JSONB NOT NULL,
  revision BIGINT NOT NULL DEFAULT 1,
  updated_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 三维用量计数：provider 归因由 ai_executions 承担，这里只负责闸门判定所需的
-- 作用域窗口。window_start 统一按 UTC 对齐，避免站点时区变化让窗口漂移。
CREATE TABLE ai_usage_counters (
  scope TEXT NOT NULL CHECK (scope IN ('site', 'extension', 'user')),
  scope_key TEXT NOT NULL DEFAULT '',
  window_kind TEXT NOT NULL CHECK (window_kind IN ('minute', 'day', 'month')),
  window_start TIMESTAMPTZ NOT NULL,
  calls BIGINT NOT NULL DEFAULT 0,
  input_tokens BIGINT NOT NULL DEFAULT 0,
  output_tokens BIGINT NOT NULL DEFAULT 0,
  spend_micros BIGINT NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (scope, scope_key, window_kind, window_start)
);

CREATE INDEX ai_usage_counters_window_idx
  ON ai_usage_counters (window_kind, window_start);

-- 执行 trace：只保存可解释决策所需的元数据，不保存提示词与响应正文。
CREATE TABLE ai_executions (
  id BIGSERIAL PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  purpose TEXT NOT NULL,
  cost_class TEXT NOT NULL DEFAULT '',
  caller_extension_id TEXT NOT NULL DEFAULT '',
  profile_id TEXT NOT NULL DEFAULT '',
  protocol TEXT NOT NULL DEFAULT '',
  model TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL CHECK (status IN ('succeeded', 'failed', 'denied')),
  gate_reason TEXT NOT NULL DEFAULT '',
  gate_scope TEXT NOT NULL DEFAULT '',
  prompt_version TEXT NOT NULL DEFAULT '',
  config_revision BIGINT NOT NULL DEFAULT 0,
  latency_ms INTEGER NOT NULL DEFAULT 0,
  input_tokens INTEGER NOT NULL DEFAULT 0,
  output_tokens INTEGER NOT NULL DEFAULT 0,
  cached_tokens INTEGER NOT NULL DEFAULT 0,
  spend_micros BIGINT NOT NULL DEFAULT 0,
  cache_hit BOOLEAN NOT NULL DEFAULT FALSE,
  error_summary TEXT NOT NULL DEFAULT ''
);

CREATE INDEX ai_executions_created_idx ON ai_executions (created_at DESC);
CREATE INDEX ai_executions_purpose_idx ON ai_executions (purpose, created_at DESC);
CREATE INDEX ai_executions_caller_idx ON ai_executions (caller_extension_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS ai_executions;
DROP TABLE IF EXISTS ai_usage_counters;
DROP TABLE IF EXISTS ai_settings;
