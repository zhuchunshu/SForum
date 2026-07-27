-- +goose Up
-- Host 记录每个搜索 provider 已确认写入的主题版本，供周期对账补漏。
-- 不使用 topics 外键：主题删除后仍需保留账本行，直到 delete job 成功。
CREATE TABLE search_index_state (
  provider_id TEXT NOT NULL,
  topic_id BIGINT NOT NULL,
  source_updated_at TIMESTAMPTZ NOT NULL,
  indexed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (provider_id, topic_id)
);

-- 默认站内搜索已有文档可直接成为初始账本；其中的幽灵文档会在首轮对账删除。
INSERT INTO search_index_state (provider_id, topic_id, source_updated_at, indexed_at)
SELECT 'sforum.search-site', topic_id, updated_at, now()
FROM search_documents
ON CONFLICT (provider_id, topic_id) DO NOTHING;

CREATE INDEX search_index_state_provider_indexed_idx
  ON search_index_state (provider_id, indexed_at, topic_id);

-- 对账只扫描公开主题，并按最旧源版本优先补偿。
CREATE INDEX topics_search_reconcile_idx
  ON topics (updated_at, id)
  WHERE status IN ('active', 'locked');

-- +goose Down
DROP INDEX IF EXISTS topics_search_reconcile_idx;
DROP INDEX IF EXISTS search_index_state_provider_indexed_idx;
DROP TABLE IF EXISTS search_index_state;
