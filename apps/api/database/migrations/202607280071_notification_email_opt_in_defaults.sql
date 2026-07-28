-- +goose Up
-- Core notification email is opt-in. Only untouched V2 policy rows without a
-- saved legacy choice are migrated, preserving every explicit operator choice.
UPDATE notification_type_policies AS policy
SET enabled = FALSE,
    recommended_enabled = FALSE,
    revision = revision + 1,
    updated_at = now()
FROM (VALUES
  ('reply', 'notification.reply.email'),
  ('mention', 'notification.mention.email'),
  ('moderation_approved', 'notification.moderation.email'),
  ('moderation_rejected', 'notification.moderation.email')
) AS defaults(type, option_name)
WHERE policy.type = defaults.type
  AND policy.channel = 'email'
  AND policy.revision = 2
  AND NOT EXISTS (
    SELECT 1 FROM web_options saved
    WHERE saved.name = defaults.option_name AND saved.value <> ''
  );

UPDATE notification_policy_revisions
SET revision = revision + 1, updated_at = now()
WHERE singleton = TRUE;

INSERT INTO web_options (name, value)
VALUES
  ('notification.reply.email', 'disabled'),
  ('notification.mention.email', 'disabled'),
  ('notification.moderation.email', 'disabled')
ON CONFLICT (name) DO UPDATE
SET value = 'disabled'
WHERE web_options.value = '';

-- +goose Down
-- This default transition is intentionally irreversible. Re-enabling email on
-- rollback would overwrite operator policy selected after this migration.
