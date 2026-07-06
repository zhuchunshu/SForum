-- +goose Up
CREATE TABLE extension_event_deliveries (
  id BIGSERIAL PRIMARY KEY,
  extension_id TEXT NOT NULL REFERENCES extensions(id) ON DELETE CASCADE,
  event_name TEXT NOT NULL,
  event_kind TEXT NOT NULL CHECK (event_kind IN ('observe', 'validate', 'filter')),
  status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'skipped')),
  reason TEXT NOT NULL DEFAULT '',
  message TEXT NOT NULL DEFAULT '',
  correlation_id TEXT NOT NULL DEFAULT '',
  attempt_count INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ
);

CREATE INDEX extension_event_deliveries_extension_created_idx
  ON extension_event_deliveries (extension_id, created_at DESC, id DESC);
CREATE INDEX extension_event_deliveries_event_status_idx
  ON extension_event_deliveries (event_name, status, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS extension_event_deliveries_event_status_idx;
DROP INDEX IF EXISTS extension_event_deliveries_extension_created_idx;
DROP TABLE IF EXISTS extension_event_deliveries;
