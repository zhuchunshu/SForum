-- +goose Up
-- Zero preserves the historical behavior: render every available category.
ALTER TABLE site_navigation_placements
ADD COLUMN max_items INTEGER NOT NULL DEFAULT 0
CHECK (max_items BETWEEN 0 AND 100);

-- +goose Down
ALTER TABLE site_navigation_placements
DROP COLUMN IF EXISTS max_items;
