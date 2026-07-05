-- +goose Up
ALTER TABLE extensions
  ADD COLUMN source TEXT NOT NULL DEFAULT 'uploaded' CHECK (source IN ('builtin', 'uploaded')),
  ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN is_deletable BOOLEAN NOT NULL DEFAULT true;

CREATE INDEX extensions_source_idx ON extensions (source);

-- +goose Down
DROP INDEX IF EXISTS extensions_source_idx;

ALTER TABLE extensions
  DROP COLUMN IF EXISTS is_deletable,
  DROP COLUMN IF EXISTS is_system,
  DROP COLUMN IF EXISTS source;
