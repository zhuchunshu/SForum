-- +goose Up
-- Notification Platform V2 keeps the V1 inbox/mail contract intact while
-- adding versioned descriptors, policy, recipient revisions, and external
-- channel persistence. mail_deliveries remains Mail's durable authority.

ALTER TABLE notifications
  ADD COLUMN IF NOT EXISTS category TEXT NOT NULL DEFAULT 'system',
  ADD COLUMN IF NOT EXISTS type_version INTEGER NOT NULL DEFAULT 1,
  ADD COLUMN IF NOT EXISTS payload_version INTEGER NOT NULL DEFAULT 1,
  ADD COLUMN IF NOT EXISTS target_meta JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE notifications
SET category = CASE type
  WHEN 'reply' THEN 'conversation'
  WHEN 'mention' THEN 'mention'
  WHEN 'moderation_approved' THEN 'moderation'
  WHEN 'moderation_rejected' THEN 'moderation'
  WHEN 'admin_test' THEN 'system'
  ELSE 'plugin_unknown'
END
WHERE category = 'system';

CREATE INDEX IF NOT EXISTS notifications_recipient_category_id_idx
  ON notifications (recipient_user_id, category, id DESC);
CREATE INDEX IF NOT EXISTS notifications_recipient_type_id_idx
  ON notifications (recipient_user_id, type, id DESC);

CREATE TABLE notification_type_descriptors (
  type TEXT PRIMARY KEY CHECK (type ~ '^[a-z][a-z0-9._-]{1,160}$'),
  contract_version INTEGER NOT NULL CHECK (contract_version > 0),
  payload_version INTEGER NOT NULL CHECK (payload_version > 0),
  owner_extension_id TEXT NOT NULL DEFAULT '',
  owner_artifact_digest TEXT NOT NULL DEFAULT '',
  category TEXT NOT NULL,
  payload_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
  payload_schema_digest TEXT NOT NULL DEFAULT '',
  presentation JSONB NOT NULL DEFAULT '{}'::jsonb,
  target_contract JSONB NOT NULL DEFAULT '{}'::jsonb,
  active BOOLEAN NOT NULL DEFAULT FALSE,
  required BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK ((required = FALSE) OR owner_extension_id = '')
);

CREATE TABLE notification_type_policies (
  type TEXT NOT NULL REFERENCES notification_type_descriptors(type) ON DELETE RESTRICT,
  channel TEXT NOT NULL CHECK (channel IN ('in_app', 'email', 'web_push')),
  enabled BOOLEAN NOT NULL DEFAULT FALSE,
  recommended_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  user_configurable BOOLEAN NOT NULL DEFAULT TRUE,
  required BOOLEAN NOT NULL DEFAULT FALSE,
  revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (type, channel),
  CHECK ((required = FALSE) OR (channel = 'in_app' AND enabled = TRUE AND user_configurable = FALSE))
);

CREATE TABLE notification_preferences (
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type TEXT NOT NULL REFERENCES notification_type_descriptors(type) ON DELETE RESTRICT,
  channel TEXT NOT NULL CHECK (channel IN ('in_app', 'email', 'web_push')),
  state TEXT NOT NULL CHECK (state IN ('inherit', 'enabled', 'disabled')),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, type, channel)
);

CREATE TABLE notification_recipient_revisions (
  user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  revision BIGINT NOT NULL DEFAULT 0 CHECK (revision >= 0),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- External channels use this generic delivery ledger. Email deliberately keeps
-- its existing mail_deliveries history, worker and Mail administration APIs.
CREATE TABLE notification_channel_deliveries (
  id BIGSERIAL PRIMARY KEY,
  notification_id BIGINT NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
  recipient_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type TEXT NOT NULL,
  channel TEXT NOT NULL CHECK (channel <> 'email' AND channel IN ('web_push')),
  provider_extension_id TEXT NOT NULL DEFAULT '',
  provider_artifact_digest TEXT NOT NULL DEFAULT '',
  payload_version INTEGER NOT NULL CHECK (payload_version > 0),
  idempotency_key TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'sending', 'sent', 'failed', 'skipped')),
  attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
  reason TEXT NOT NULL DEFAULT '',
  error_summary TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ
);
CREATE INDEX notification_channel_deliveries_recipient_created_idx
  ON notification_channel_deliveries (recipient_user_id, created_at DESC, id DESC);
CREATE INDEX notification_channel_deliveries_status_created_idx
  ON notification_channel_deliveries (status, created_at DESC);

CREATE TABLE notification_channel_selections (
  channel TEXT PRIMARY KEY CHECK (channel IN ('web_push')),
  extension_id TEXT NOT NULL REFERENCES extensions(id) ON DELETE RESTRICT,
  artifact_digest TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE web_push_subscriptions (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  endpoint TEXT NOT NULL,
  endpoint_hash TEXT NOT NULL UNIQUE CHECK (endpoint_hash ~ '^[a-f0-9]{64}$'),
  p256dh_key BYTEA NOT NULL,
  auth_key BYTEA NOT NULL,
  content_encoding TEXT NOT NULL DEFAULT 'aes128gcm',
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked', 'failed')),
  last_failure_reason TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  revoked_at TIMESTAMPTZ
);
CREATE INDEX web_push_subscriptions_user_status_idx
  ON web_push_subscriptions (user_id, status, id DESC);

INSERT INTO notification_type_descriptors (type, contract_version, payload_version, category, active, required, presentation, target_contract)
VALUES
  ('reply', 1, 1, 'conversation', TRUE, FALSE, '{"icon":"i-lucide-message-circle"}'::jsonb, '{"kind":"forum"}'::jsonb),
  ('mention', 1, 1, 'mention', TRUE, FALSE, '{"icon":"i-lucide-at-sign"}'::jsonb, '{"kind":"forum"}'::jsonb),
  ('moderation_approved', 1, 1, 'moderation', TRUE, FALSE, '{"icon":"i-lucide-circle-check"}'::jsonb, '{"kind":"forum"}'::jsonb),
  ('moderation_rejected', 1, 1, 'moderation', TRUE, FALSE, '{"icon":"i-lucide-circle-x"}'::jsonb, '{"kind":"forum"}'::jsonb),
  ('admin_test', 1, 1, 'system', TRUE, FALSE, '{"icon":"i-lucide-bell"}'::jsonb, '{"kind":"system"}'::jsonb)
ON CONFLICT (type) DO NOTHING;

INSERT INTO notification_type_policies (type, channel, enabled, recommended_enabled, user_configurable, required)
SELECT descriptor.type, channel.channel,
  CASE
    WHEN descriptor.type = 'reply' AND channel.channel = 'in_app' THEN COALESCE((SELECT value = 'enabled' FROM web_options WHERE name = 'notification.reply.in_app'), TRUE)
    WHEN descriptor.type = 'reply' AND channel.channel = 'email' THEN COALESCE((SELECT value = 'enabled' FROM web_options WHERE name = 'notification.reply.email'), TRUE)
    WHEN descriptor.type = 'mention' AND channel.channel = 'in_app' THEN COALESCE((SELECT value = 'enabled' FROM web_options WHERE name = 'notification.mention.in_app'), TRUE)
    WHEN descriptor.type = 'mention' AND channel.channel = 'email' THEN COALESCE((SELECT value = 'enabled' FROM web_options WHERE name = 'notification.mention.email'), TRUE)
    WHEN descriptor.type LIKE 'moderation_%' AND channel.channel = 'in_app' THEN COALESCE((SELECT value = 'enabled' FROM web_options WHERE name = 'notification.moderation.in_app'), TRUE)
    WHEN descriptor.type LIKE 'moderation_%' AND channel.channel = 'email' THEN COALESCE((SELECT value = 'enabled' FROM web_options WHERE name = 'notification.moderation.email'), TRUE)
    WHEN descriptor.type = 'admin_test' AND channel.channel = 'in_app' THEN TRUE
    ELSE FALSE
  END,
  FALSE, TRUE, FALSE
FROM notification_type_descriptors descriptor
CROSS JOIN (VALUES ('in_app'), ('email'), ('web_push')) AS channel(channel)
WHERE descriptor.owner_extension_id = ''
ON CONFLICT (type, channel) DO NOTHING;

INSERT INTO permissions (key, module, description)
VALUES ('settings.notifications.manage', 'admin', 'Manage notification type policy, channels, and delivery health.')
ON CONFLICT (key) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, 'settings.notifications.manage'
FROM roles WHERE roles.key IN ('super_admin', 'operator')
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_permissions WHERE permission_key = 'settings.notifications.manage';
DELETE FROM permissions WHERE key = 'settings.notifications.manage';
DROP TABLE IF EXISTS web_push_subscriptions;
DROP TABLE IF EXISTS notification_channel_selections;
DROP TABLE IF EXISTS notification_channel_deliveries;
DROP TABLE IF EXISTS notification_recipient_revisions;
DROP TABLE IF EXISTS notification_preferences;
DROP TABLE IF EXISTS notification_type_policies;
DROP TABLE IF EXISTS notification_type_descriptors;
DROP INDEX IF EXISTS notifications_recipient_type_id_idx;
DROP INDEX IF EXISTS notifications_recipient_category_id_idx;
ALTER TABLE notifications
  DROP COLUMN IF EXISTS target_meta,
  DROP COLUMN IF EXISTS payload_version,
  DROP COLUMN IF EXISTS type_version,
  DROP COLUMN IF EXISTS category;
