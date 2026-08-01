-- +goose Up
-- A browser subscription is an explicit device opt-in. Make untouched Core
-- Web Push rows useful by default while preserving every operator-edited row.
WITH changed AS (
  UPDATE notification_type_policies
  SET enabled = TRUE,
      recommended_enabled = TRUE,
      revision = revision + 1,
      updated_at = now()
  WHERE type IN ('reply', 'mention', 'moderation_approved', 'moderation_rejected')
    AND channel = 'web_push'
    AND revision = 1
    AND enabled = FALSE
    AND recommended_enabled = FALSE
  RETURNING 1
)
UPDATE notification_policy_revisions
SET revision = revision + 1,
    updated_at = now()
WHERE singleton = TRUE
  AND EXISTS (SELECT 1 FROM changed);

-- +goose Down
-- This recommendation correction is intentionally irreversible. A later
-- operator choice cannot be distinguished safely from the migrated default.
