-- +goose Up
-- Mobile navigation now owns the category list that previously occupied page
-- content. Preserve any existing placement and make the new default operator-
-- configurable after migration.
INSERT INTO site_navigation_placements (
    source_key, location, position, enabled, visibility, max_items
)
VALUES (
    'core.dynamic.categories', 'public.mobile.primary', 40, TRUE, 'public', 0
)
ON CONFLICT (source_key, location) DO NOTHING;

UPDATE site_navigation_state
SET revision = revision + 1,
    updated_at = now()
WHERE id = 1;

INSERT INTO web_options (name, value)
VALUES ('site.public_surface_revision', '2')
ON CONFLICT (name) DO UPDATE SET value = (
    CASE
        WHEN trim(web_options.value) ~ '^[0-9]+$'
            THEN greatest(web_options.value::numeric, 1) + 1
        ELSE 2
    END
)::text;

-- +goose Down
-- Deliberately no-op. The placement becomes operator-managed state after the
-- migration and may have been edited independently.
