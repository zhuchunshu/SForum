-- +goose Up
-- F2.4：插件 schema 迁移账本。路径相对包根，与 manifest.migrations[].path 对齐。
CREATE TABLE extension_migration_ledger (
  extension_id TEXT NOT NULL REFERENCES extensions(id) ON DELETE CASCADE,
  path TEXT NOT NULL,
  checksum TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'applied'
    CHECK (status IN ('applied', 'recorded', 'failed')),
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  message TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (extension_id, path)
);

CREATE INDEX extension_migration_ledger_applied_idx
  ON extension_migration_ledger (extension_id, applied_at);

-- +goose Down
DROP INDEX IF EXISTS extension_migration_ledger_applied_idx;
DROP TABLE IF EXISTS extension_migration_ledger;
