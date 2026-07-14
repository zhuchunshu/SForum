-- +goose Up
-- Compensation must restore the desired source theme's prior Core replacement
-- approval rather than reusing the failed target activation actor.
ALTER TABLE theme_runtime_publications
  ADD COLUMN source_core_replacements_approved BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN source_actor_user_id BIGINT
    CHECK (source_actor_user_id IS NULL OR source_actor_user_id > 0),
  ADD CONSTRAINT theme_runtime_publications_source_approval_check
    CHECK (
      (source_core_replacements_approved = FALSE AND source_actor_user_id IS NULL)
      OR
      (source_core_replacements_approved = TRUE
        AND source_actor_user_id IS NOT NULL
        AND source_theme_id IS NOT NULL)
    );

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM theme_runtime_publications) THEN
    RAISE EXCEPTION 'cannot remove theme runtime source approval history';
  END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE theme_runtime_publications
  DROP CONSTRAINT IF EXISTS theme_runtime_publications_source_approval_check,
  DROP COLUMN IF EXISTS source_actor_user_id,
  DROP COLUMN IF EXISTS source_core_replacements_approved;
