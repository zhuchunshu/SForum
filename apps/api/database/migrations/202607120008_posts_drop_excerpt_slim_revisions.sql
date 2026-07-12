-- +goose Up
-- posts.excerpt 可从 plain_text 按运营配置截断派生，不必落库。
-- post_revisions 只保留源文与渲染元数据，展示历史时再渲染。
ALTER TABLE posts DROP COLUMN IF EXISTS excerpt;

ALTER TABLE post_revisions DROP COLUMN IF EXISTS html_content;
ALTER TABLE post_revisions DROP COLUMN IF EXISTS plain_text;
ALTER TABLE post_revisions DROP COLUMN IF EXISTS excerpt;

-- +goose Down
ALTER TABLE posts
  ADD COLUMN IF NOT EXISTS excerpt TEXT NOT NULL DEFAULT '';

-- 回滚时从当前 posts 回填摘要（近似）；html/plain 无法从 revision raw 自动恢复。
UPDATE posts
SET excerpt = CASE
  WHEN char_length(plain_text) <= 180 THEN plain_text
  ELSE left(plain_text, 180) || '...'
END
WHERE excerpt = '';

ALTER TABLE post_revisions
  ADD COLUMN IF NOT EXISTS html_content TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS plain_text TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS excerpt TEXT NOT NULL DEFAULT '';
