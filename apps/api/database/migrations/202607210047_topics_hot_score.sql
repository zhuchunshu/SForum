-- +goose Up
-- M2：可索引的热度列，替代 ListTopics sort=hot 的表达式排序。
-- 公式与产品一致：hot_score = comment_count * 5 + view_count（无时间衰减）。
ALTER TABLE topics
  ADD COLUMN IF NOT EXISTS hot_score BIGINT NOT NULL DEFAULT 0;

-- 存量回填（1e6 行级可接受；新写入由 comment/view 维护路径保持）。
UPDATE topics
SET hot_score = comment_count * 5 + view_count
WHERE hot_score <> comment_count * 5 + view_count;

-- 首页 / 无分类过滤的 hot 排序。
CREATE INDEX IF NOT EXISTS topics_public_hot_idx
  ON topics (is_pinned DESC, hot_score DESC, id DESC)
  WHERE status IN ('active', 'locked');

-- 分类过滤下的 hot 排序（与 activity 索引并列）。
CREATE INDEX IF NOT EXISTS topics_category_hot_idx
  ON topics (category_id, is_pinned DESC, hot_score DESC, id DESC)
  WHERE status IN ('active', 'locked');

-- +goose Down
DROP INDEX IF EXISTS topics_category_hot_idx;
DROP INDEX IF EXISTS topics_public_hot_idx;
ALTER TABLE topics DROP COLUMN IF EXISTS hot_score;
