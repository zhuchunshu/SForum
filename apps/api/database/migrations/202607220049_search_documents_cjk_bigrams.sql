-- +goose Up
-- 默认 PostgreSQL 引擎为中文标题、摘要和正文增加零依赖 n-gram 索引。
-- 升级后旧文档需通过现有 search reindex 流程重建该派生列。
ALTER TABLE search_documents
  ADD COLUMN IF NOT EXISTS cjk_tsv TSVECTOR NOT NULL DEFAULT ''::tsvector;

CREATE INDEX IF NOT EXISTS search_documents_cjk_tsv_idx
  ON search_documents USING GIN (cjk_tsv);

-- +goose Down
DROP INDEX IF EXISTS search_documents_cjk_tsv_idx;
ALTER TABLE search_documents DROP COLUMN IF EXISTS cjk_tsv;
