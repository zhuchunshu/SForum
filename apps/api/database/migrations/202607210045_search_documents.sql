-- +goose Up
-- 站内搜索（search.provider 默认引擎）文档表。
-- PostgreSQL 为源真相的派生索引：可从 topics/posts 全量重建；Host 拥有 schema。
CREATE TABLE search_documents (
  topic_id BIGINT PRIMARY KEY,
  title TEXT NOT NULL DEFAULT '',
  excerpt TEXT NOT NULL DEFAULT '',
  plain_text TEXT NOT NULL DEFAULT '',
  category_id BIGINT NOT NULL DEFAULT 0,
  category_slug TEXT NOT NULL DEFAULT '',
  category_name TEXT NOT NULL DEFAULT '',
  author_user_id BIGINT NOT NULL DEFAULT 0,
  author_username TEXT NOT NULL DEFAULT '',
  author_display_name TEXT NOT NULL DEFAULT '',
  slug TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT '',
  is_pinned BOOLEAN NOT NULL DEFAULT false,
  comment_count BIGINT NOT NULL DEFAULT 0,
  view_count BIGINT NOT NULL DEFAULT 0,
  tag_slugs TEXT[] NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_activity_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  -- pg_catalog 配置：中英混排分词良好（'simple' 对中文支持较弱）。
  tsv tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('pg_catalog', coalesce(title, '')), 'A') ||
    setweight(to_tsvector('pg_catalog', coalesce(excerpt, '')), 'B') ||
    setweight(to_tsvector('pg_catalog', coalesce(plain_text, '')), 'C')
  ) STORED
);

CREATE INDEX search_documents_tsv_idx ON search_documents USING GIN (tsv);
CREATE INDEX search_documents_status_activity_idx
  ON search_documents (status, is_pinned DESC, last_activity_at DESC);
CREATE INDEX search_documents_category_slug_idx ON search_documents (category_slug);
CREATE INDEX search_documents_tag_slugs_idx ON search_documents USING GIN (tag_slugs);

-- +goose Down
DROP INDEX IF EXISTS search_documents_tag_slugs_idx;
DROP INDEX IF EXISTS search_documents_category_slug_idx;
DROP INDEX IF EXISTS search_documents_status_activity_idx;
DROP INDEX IF EXISTS search_documents_tsv_idx;
DROP TABLE IF EXISTS search_documents;
