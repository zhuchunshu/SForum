-- +goose Up
CREATE TABLE notifications (
  id BIGSERIAL PRIMARY KEY,
  recipient_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type TEXT NOT NULL,
  actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  target_type TEXT NOT NULL,
  target_id BIGINT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  dedupe_key TEXT NOT NULL UNIQUE,
  read_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX notifications_recipient_created_idx ON notifications (recipient_user_id, created_at DESC, id DESC);
CREATE INDEX notifications_recipient_unread_idx ON notifications (recipient_user_id, created_at DESC) WHERE read_at IS NULL;

CREATE TABLE mail_deliveries (
  id BIGSERIAL PRIMARY KEY,
  recipient TEXT NOT NULL,
  template_key TEXT NOT NULL,
  template_data JSONB NOT NULL DEFAULT '{}'::jsonb,
  idempotency_key TEXT NOT NULL UNIQUE,
  correlation_id TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'sending', 'sent', 'failed', 'skipped')),
  extension_id TEXT NOT NULL DEFAULT '',
  attempt_count INTEGER NOT NULL DEFAULT 0,
  reason TEXT NOT NULL DEFAULT '',
  error_summary TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ
);

CREATE INDEX mail_deliveries_created_idx ON mail_deliveries (created_at DESC, id DESC);
CREATE INDEX mail_deliveries_status_idx ON mail_deliveries (status, created_at DESC);

CREATE TABLE mail_provider_selection (
  slot TEXT PRIMARY KEY,
  extension_id TEXT NOT NULL REFERENCES extensions(id) ON DELETE CASCADE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE runtime_migrations (
  key TEXT PRIMARY KEY,
  completed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS runtime_migrations;
DROP TABLE IF EXISTS mail_provider_selection;
DROP TABLE IF EXISTS mail_deliveries;
DROP TABLE IF EXISTS notifications;
