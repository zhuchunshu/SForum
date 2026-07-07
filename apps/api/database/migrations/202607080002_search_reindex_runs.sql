-- +goose Up
-- 搜索索引重建运行记录。每次后台触发重建插入一行，用于进度追踪与历史审计。
-- 进度的"已处理数"不在此表累加，而是 GET 端点实时查询 river_job 表的剩余 job 数得出，
-- 避免在 IndexTopicWorker 里耦合 reindex 回调逻辑。
CREATE TABLE search_reindex_runs (
  id BIGSERIAL PRIMARY KEY,
  total BIGINT NOT NULL DEFAULT 0 CHECK (total >= 0),
  status TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'completed', 'failed')),
  started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  finished_at TIMESTAMPTZ,
  started_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  error TEXT NOT NULL DEFAULT ''
);

CREATE INDEX search_reindex_runs_status_started_idx
  ON search_reindex_runs (status, started_at DESC);

-- +goose Down
DROP INDEX IF EXISTS search_reindex_runs_status_started_idx;
DROP TABLE IF EXISTS search_reindex_runs;
