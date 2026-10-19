-- +goose Up
-- +sforum OnlineSafe
-- A3：AI run 关联 id。一次 run 的全部模型调用共用 (resource_type, resource_id)，
-- 使「这次回复花了几次调用、每一步为什么失败」可以被聚合成一条时间线。
--
-- 两列都是归属标识而不是内容：resource_id 是 16 字节随机 hex，无法从中还原任何
-- 用户内容，因此 ai_executions「不含提示词与响应正文」的宣告仍然成立。
--
-- 历史行取空串默认值（不设 NULL）：回放查询按 (type, id) 精确匹配，空串行自然
-- 不匹配任何 run，界面据此不显示展开入口。
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '1min';

ALTER TABLE ai_executions
  ADD COLUMN resource_type TEXT NOT NULL DEFAULT '',
  ADD COLUMN resource_id TEXT NOT NULL DEFAULT '';

-- 回放查询是「按归属取全部记录，按时间正序」，索引顺序与之对齐（DESC 建索引、
-- ASC 扫描在 PostgreSQL 上同样可用，且列表页的 DESC 查询也吃这条索引）。
CREATE INDEX ai_executions_resource_idx
  ON ai_executions (resource_type, resource_id, created_at DESC, id DESC);

-- +goose Down
SET LOCAL lock_timeout = '5s';

-- 先删索引再删列：DROP COLUMN 会连带删掉依赖它的索引，但显式顺序让 Down 在
-- 只回滚了一半的中断场景下仍然可重放。
DROP INDEX IF EXISTS ai_executions_resource_idx;
ALTER TABLE ai_executions
  DROP COLUMN IF EXISTS resource_id,
  DROP COLUMN IF EXISTS resource_type;
