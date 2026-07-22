-- +goose Up
-- 作者、分类、标签和 slug 属于稳定的主题检索元数据；单独建向量以便独立调权和维护。
ALTER TABLE search_documents
  ADD COLUMN IF NOT EXISTS metadata_tsv TSVECTOR NOT NULL DEFAULT ''::tsvector;

-- 先回填已有文档，升级后无需等待一次重建即可按完整元数据精确搜索。
UPDATE search_documents
SET metadata_tsv =
  setweight(to_tsvector('simple', concat_ws(' ', author_username, author_display_name)), 'B') ||
  setweight(to_tsvector('simple', concat_ws(' ', category_slug, category_name)), 'C') ||
  setweight(to_tsvector('simple', concat_ws(' ', slug, array_to_string(tag_slugs, ' '))), 'D');

CREATE INDEX IF NOT EXISTS search_documents_metadata_tsv_idx
  ON search_documents USING GIN (metadata_tsv);

-- +goose Down
DROP INDEX IF EXISTS search_documents_metadata_tsv_idx;
ALTER TABLE search_documents DROP COLUMN IF EXISTS metadata_tsv;
