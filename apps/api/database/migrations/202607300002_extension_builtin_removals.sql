-- +goose Up
CREATE TABLE extension_builtin_removals (
  extension_id TEXT PRIMARY KEY REFERENCES extensions(id) ON DELETE RESTRICT,
  extension_type TEXT NOT NULL CHECK (extension_type IN ('plugin', 'theme')),
  extension_version_id BIGINT NOT NULL CHECK (extension_version_id > 0),
  extension_version TEXT NOT NULL,
  package_digest TEXT NOT NULL CHECK (package_digest ~ '^[0-9a-f]{64}$'),
  removed_at TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp()
);

CREATE INDEX extension_builtin_removals_removed_idx
  ON extension_builtin_removals (removed_at DESC, extension_id);

-- +goose Down
DROP TABLE IF EXISTS extension_builtin_removals;
