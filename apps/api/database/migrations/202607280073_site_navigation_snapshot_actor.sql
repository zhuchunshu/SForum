-- +goose Up
ALTER TABLE site_navigation_snapshots
    ADD COLUMN actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX site_navigation_snapshots_actor_idx
    ON site_navigation_snapshots (actor_user_id, created_at DESC)
    WHERE actor_user_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS site_navigation_snapshots_actor_idx;
ALTER TABLE site_navigation_snapshots DROP COLUMN IF EXISTS actor_user_id;
