-- +goose Up
-- Page Registry：绑定表持久化 contract_version，供 Resolve/审批精确校验。
ALTER TABLE page_provider_bindings
    ADD COLUMN IF NOT EXISTS contract_version TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE page_provider_bindings
    DROP COLUMN IF EXISTS contract_version;
