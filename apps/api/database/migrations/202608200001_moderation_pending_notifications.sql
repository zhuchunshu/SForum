-- +goose Up
-- +sforum OnlineSafe
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '1min';

INSERT INTO notification_type_descriptors (
  type, contract_version, payload_version, category, active, required,
  presentation, target_contract
) VALUES (
  'moderation_pending', 1, 1, 'moderation', TRUE, FALSE,
  '{"icon":"i-lucide-shield-alert"}'::jsonb,
  '{"kind":"moderation_workbench"}'::jsonb
)
ON CONFLICT (type) DO NOTHING;

INSERT INTO notification_type_policies (
  type, channel, enabled, recommended_enabled, user_configurable, required
) VALUES
  ('moderation_pending', 'in_app', TRUE, TRUE, TRUE, FALSE),
  ('moderation_pending', 'email', FALSE, FALSE, TRUE, FALSE),
  ('moderation_pending', 'web_push', TRUE, TRUE, TRUE, FALSE)
ON CONFLICT (type, channel) DO NOTHING;

UPDATE notification_policy_revisions
SET revision = revision + 1, updated_at = now()
WHERE singleton = TRUE;

-- +goose Down
DELETE FROM notification_preferences WHERE type = 'moderation_pending';
DELETE FROM notification_type_policies WHERE type = 'moderation_pending';
DELETE FROM notification_type_descriptors WHERE type = 'moderation_pending';
UPDATE notification_policy_revisions
SET revision = revision + 1, updated_at = now()
WHERE singleton = TRUE;
