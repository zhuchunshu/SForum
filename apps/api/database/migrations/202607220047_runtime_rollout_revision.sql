-- +goose Up
-- Optimistic concurrency for concurrent Ack/Promote/Rollback on rollout plans.
ALTER TABLE runtime_rollout_plans
    ADD COLUMN IF NOT EXISTS revision BIGINT NOT NULL DEFAULT 1
        CHECK (revision >= 1);

-- +goose Down
ALTER TABLE runtime_rollout_plans DROP COLUMN IF EXISTS revision;
