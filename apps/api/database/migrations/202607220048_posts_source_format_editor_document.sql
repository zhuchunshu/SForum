-- +goose Up
-- Host 正文管线将 Tiptap native JSON 以 source_format=editor-document 落库，
-- 但 posts 表 CHECK 仍停留在 markdown/html/json，导致发帖 500。
-- 扩展 allowlist，与 app/Models/Forum SourceFormat* 常量对齐。
ALTER TABLE posts DROP CONSTRAINT IF EXISTS posts_source_format_check;
ALTER TABLE posts
  ADD CONSTRAINT posts_source_format_check
  CHECK (source_format = ANY (ARRAY[
    'markdown'::text,
    'html'::text,
    'json'::text,
    'editor-document'::text
  ]));

-- +goose Down
-- 回滚前若已存在 editor-document 行会失败；Down 仅用于空库/测试回滚。
ALTER TABLE posts DROP CONSTRAINT IF EXISTS posts_source_format_check;
ALTER TABLE posts
  ADD CONSTRAINT posts_source_format_check
  CHECK (source_format = ANY (ARRAY[
    'markdown'::text,
    'html'::text,
    'json'::text
  ]));
