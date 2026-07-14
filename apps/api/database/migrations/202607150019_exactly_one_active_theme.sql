-- +goose Up
-- Historical releases could leave multiple enabled themes. Keep the same
-- deterministic winner used by ActiveTheme before enforcing the invariant.
WITH ranked AS (
    SELECT id,
           ROW_NUMBER() OVER (
               ORDER BY (source = 'uploaded') DESC, updated_at DESC, id ASC
           ) AS position
    FROM extensions
    WHERE type = 'theme' AND status = 'enabled'
)
UPDATE extensions
SET status = 'disabled', updated_at = NOW()
WHERE id IN (SELECT id FROM ranked WHERE position > 1);

CREATE UNIQUE INDEX extensions_one_active_theme_idx
    ON extensions ((type))
    WHERE type = 'theme' AND status = 'enabled';

-- +goose Down
DROP INDEX IF EXISTS extensions_one_active_theme_idx;
