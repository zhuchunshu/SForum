-- +goose Up
-- Empty placement icons inherit the definition icon. Keep explicit icon
-- suppression separate so old documents retain their existing behavior.
ALTER TABLE site_navigation_placements
ADD COLUMN icon_hidden BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE site_navigation_placements
DROP COLUMN IF EXISTS icon_hidden;
