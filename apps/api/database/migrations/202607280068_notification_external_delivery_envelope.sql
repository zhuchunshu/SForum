-- +goose Up
-- External projections must remain independent from the in-app projection.
-- When in_app is disabled there is deliberately no visible notifications row,
-- so the generic channel ledger carries its own bounded structured envelope.
ALTER TABLE notification_channel_deliveries
  ALTER COLUMN notification_id DROP NOT NULL,
  ADD COLUMN payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  ADD COLUMN target_meta JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE TABLE web_push_delivery_attempts (
  delivery_id BIGINT NOT NULL REFERENCES notification_channel_deliveries(id) ON DELETE CASCADE,
  subscription_id BIGINT NOT NULL REFERENCES web_push_subscriptions(id) ON DELETE CASCADE,
  status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'sent', 'failed', 'skipped')),
  attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
  reason TEXT NOT NULL DEFAULT '',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (delivery_id, subscription_id)
);

-- +goose Down
DROP TABLE IF EXISTS web_push_delivery_attempts;
ALTER TABLE notification_channel_deliveries
  DROP COLUMN IF EXISTS target_meta,
  DROP COLUMN IF EXISTS payload,
  ALTER COLUMN notification_id SET NOT NULL;
