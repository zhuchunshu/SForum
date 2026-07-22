-- +goose NO TRANSACTION
-- +goose Up
-- 管理内容工作台按更新时间翻页；索引不携带正文或修订 payload。
CREATE INDEX CONCURRENTLY IF NOT EXISTS topics_admin_content_updated_idx
  ON topics (updated_at DESC, id DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS topics_admin_content_status_updated_idx
  ON topics (status, updated_at DESC, id DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS comments_admin_content_updated_idx
  ON comments (updated_at DESC, id DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS comments_admin_content_status_updated_idx
  ON comments (status, updated_at DESC, id DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS comments_admin_content_topic_updated_idx
  ON comments (topic_id, updated_at DESC, id DESC);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS comments_admin_content_topic_updated_idx;
DROP INDEX CONCURRENTLY IF EXISTS comments_admin_content_status_updated_idx;
DROP INDEX CONCURRENTLY IF EXISTS comments_admin_content_updated_idx;
DROP INDEX CONCURRENTLY IF EXISTS topics_admin_content_status_updated_idx;
DROP INDEX CONCURRENTLY IF EXISTS topics_admin_content_updated_idx;
