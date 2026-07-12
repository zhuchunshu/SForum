-- +goose Up
-- Page Registry：核心页的生效提供者绑定（replace 需 super_admin 审批后写入）。
CREATE TABLE IF NOT EXISTS page_provider_bindings (
    page_id          TEXT PRIMARY KEY,
    extension_id     TEXT NOT NULL,
    contribution_id  TEXT NOT NULL,
    version          TEXT NOT NULL DEFAULT '',
    package_digest   TEXT NOT NULL DEFAULT '',
    approved_by      BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    template_path    TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS page_provider_bindings_extension_id_idx
    ON page_provider_bindings (extension_id);

-- +goose Down
DROP TABLE IF EXISTS page_provider_bindings;
