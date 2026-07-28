-- +goose Up
-- Migration 068 gained this table after some development databases had already
-- recorded that version. Keep the forward migration idempotent so those
-- databases receive the same per-subscription replay ledger as clean installs.
CREATE TABLE IF NOT EXISTS web_push_delivery_attempts (
  delivery_id BIGINT NOT NULL REFERENCES notification_channel_deliveries(id) ON DELETE CASCADE,
  subscription_id BIGINT NOT NULL REFERENCES web_push_subscriptions(id) ON DELETE CASCADE,
  status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'sent', 'failed', 'skipped')),
  attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
  reason TEXT NOT NULL DEFAULT '',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (delivery_id, subscription_id)
);

-- +goose Down
-- Version 068 owns the table on clean installations.
SELECT 1;
