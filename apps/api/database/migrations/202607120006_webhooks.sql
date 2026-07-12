-- +goose Up
-- 出站 Webhook 端点与投递记录（F3.3）。状态词表对齐 Support/Outbox。
CREATE TABLE webhook_endpoints (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  target_url TEXT NOT NULL,
  secret TEXT NOT NULL DEFAULT '',
  -- events: JSON 字符串数组，如 ["topic.created","comment.created"]；空数组表示订阅全部 observe 事件。
  events JSONB NOT NULL DEFAULT '[]'::jsonb,
  enabled BOOLEAN NOT NULL DEFAULT true,
  -- 新手默认：新建即可用；一键禁用走 enabled=false，不删历史投递。
  description TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX webhook_endpoints_enabled_idx ON webhook_endpoints (enabled, id);

CREATE TABLE webhook_deliveries (
  id BIGSERIAL PRIMARY KEY,
  endpoint_id BIGINT NOT NULL REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
  event_name TEXT NOT NULL,
  event_id TEXT NOT NULL DEFAULT '',
  correlation_id TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  status TEXT NOT NULL DEFAULT 'queued'
    CHECK (status IN ('queued', 'sending', 'sent', 'failed', 'skipped', 'dead')),
  attempt_count INTEGER NOT NULL DEFAULT 0,
  http_status INTEGER NOT NULL DEFAULT 0,
  response_snippet TEXT NOT NULL DEFAULT '',
  reason TEXT NOT NULL DEFAULT '',
  error_summary TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ
);

CREATE INDEX webhook_deliveries_endpoint_created_idx
  ON webhook_deliveries (endpoint_id, created_at DESC, id DESC);
CREATE INDEX webhook_deliveries_status_idx
  ON webhook_deliveries (status, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS webhook_deliveries_status_idx;
DROP INDEX IF EXISTS webhook_deliveries_endpoint_created_idx;
DROP TABLE IF EXISTS webhook_deliveries;
DROP INDEX IF EXISTS webhook_endpoints_enabled_idx;
DROP TABLE IF EXISTS webhook_endpoints;
