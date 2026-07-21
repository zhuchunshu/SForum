-- +goose Up
-- 公开列表默认排序（is_pinned, last_activity_at, id）的全局索引。
-- 分类过滤仍优先 topics_category_activity_idx；本索引服务无 category 的首页热路径，
-- 避免 1e6 行 parallel seq scan + top-N。
CREATE INDEX IF NOT EXISTS topics_public_activity_idx
  ON topics (is_pinned DESC, last_activity_at DESC, id DESC)
  WHERE status IN ('active', 'locked');

-- +goose Down
DROP INDEX IF EXISTS topics_public_activity_idx;
