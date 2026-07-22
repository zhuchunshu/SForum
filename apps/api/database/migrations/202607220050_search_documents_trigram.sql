-- +goose Up
-- pg_trgm 由 migrator 的管理员连接预装到 Host-owned schema，避免污染 Core public schema。
CREATE SCHEMA IF NOT EXISTS sforum_host_extensions;
CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA sforum_host_extensions;

ALTER TABLE search_documents
  ADD COLUMN IF NOT EXISTS fuzzy_text TEXT GENERATED ALWAYS AS (
    title || E'\n' || excerpt
  ) STORED;

CREATE INDEX IF NOT EXISTS search_documents_fuzzy_text_trgm_idx
  ON search_documents USING GIN (fuzzy_text sforum_host_extensions.gin_trgm_ops);

-- +goose Down
DROP INDEX IF EXISTS search_documents_fuzzy_text_trgm_idx;
ALTER TABLE search_documents DROP COLUMN IF EXISTS fuzzy_text;
-- pg_trgm 可能由其他模块或插件共享，回滚本迁移时不删除扩展。
