-- +goose NO TRANSACTION
-- +goose Up
-- 评论区 flat 视图改为时间流排序（created_at ASC, id ASC）：新回复恒在最底楼，
-- 既有楼层号不因他人回复而漂移。索引让分页 ORDER BY 与 keyset seek 都走 Index Cond，
-- 不携带正文，可用 CONCURRENTLY 在线创建。
CREATE INDEX CONCURRENTLY IF NOT EXISTS comments_topic_created_idx
  ON comments (topic_id, created_at, id);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS comments_topic_created_idx;
