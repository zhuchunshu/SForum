-- +goose Up
CREATE TABLE extension_missing_artifact_removals (
    extension_id text PRIMARY KEY REFERENCES extensions(id) ON DELETE RESTRICT,
    extension_type text NOT NULL CHECK (extension_type IN ('plugin', 'theme')),
    extension_version text NOT NULL,
    package_digest text NOT NULL,
    package_path text NOT NULL,
    data_mode text NOT NULL CHECK (data_mode IN ('preserve', 'discard_settings')),
    requested_by_user_id bigint REFERENCES users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX extension_missing_artifact_removals_created_idx
    ON extension_missing_artifact_removals (created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS extension_missing_artifact_removals;
