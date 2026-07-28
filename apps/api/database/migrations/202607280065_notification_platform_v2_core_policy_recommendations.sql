-- +goose Up
-- The V1 global flags were both active policy and the recommendation inherited
-- by every recipient. Preserve that meaning during the V2 compatibility move.
UPDATE notification_type_policies
SET recommended_enabled = enabled, revision = revision + 1, updated_at = now()
WHERE type IN ('reply', 'mention', 'moderation_approved', 'moderation_rejected')
  AND channel IN ('in_app', 'email');

-- +goose Down
-- This data correction is intentionally irreversible: the prior value did not
-- represent a user-visible recommendation and cannot be reconstructed safely.
