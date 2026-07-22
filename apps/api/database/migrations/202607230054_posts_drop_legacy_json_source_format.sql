-- +goose Up
-- `json` 从未进入正文 Accept 管线；结构化正文使用明确的 editor-document 契约。
-- 若存在绕过应用写入的旧数据，拒绝猜测转换，要求操作员先确认其实际格式。
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM posts WHERE source_format = 'json') THEN
    RAISE EXCEPTION 'posts contains unsupported source_format=json rows; migrate them explicitly before applying this migration';
  END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE posts DROP CONSTRAINT IF EXISTS posts_source_format_check;
ALTER TABLE posts
  ADD CONSTRAINT posts_source_format_check
  CHECK (source_format = ANY (ARRAY[
    'markdown'::text,
    'html'::text,
    'editor-document'::text
  ]));

-- +goose Down
ALTER TABLE posts DROP CONSTRAINT IF EXISTS posts_source_format_check;
ALTER TABLE posts
  ADD CONSTRAINT posts_source_format_check
  CHECK (source_format = ANY (ARRAY[
    'markdown'::text,
    'html'::text,
    'json'::text,
    'editor-document'::text
  ]));
