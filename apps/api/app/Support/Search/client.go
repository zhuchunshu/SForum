// Package search 提供基于 Meilisearch 的论坛主题全文搜索支持。
//
// 设计要点：
//   - search 包不直接依赖 forum 包（forum 是底层域），通过 TopicReader /
//     TopicSearchIndexer 接口解耦，避免循环依赖。
//   - Indexer 同时实现 forum.TopicSearchIndexer（EnqueueIndex/EnqueueDelete，
//     供 forum.Service 在写流程调度）和 searchjobs.TopicIndexer（IndexTopic，
//     供 River worker 实际执行）。
//   - 搜索为派生数据：Meilisearch 文档可从 PostgreSQL 完全重建，索引失败
//     不阻断主流程（见 service.go 的 slog 处理）。
package search

import (
	"net/http"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

// IndexUID 是主题搜索索引的唯一标识。
const IndexUID = "sforum_topics"

// NewClient 创建 Meilisearch 客户端。host 为空时使用默认本地地址。
func NewClient(host, apiKey string) meilisearch.ServiceManager {
	return meilisearch.New(host, meilisearch.WithAPIKey(apiKey))
}

// NewClientWithTimeout 创建带请求总超时的 Meilisearch 客户端。
// meilisearch-go 没有 WithTimeout 选项，必须通过 WithCustomClient 注入
// 自定义 http.Client 来设置 Timeout。timeout <= 0 时不设超时（兼容旧行为）。
func NewClientWithTimeout(host, apiKey string, timeout time.Duration) meilisearch.ServiceManager {
	if timeout <= 0 {
		return NewClient(host, apiKey)
	}
	return meilisearch.New(host,
		meilisearch.WithAPIKey(apiKey),
		meilisearch.WithCustomClient(&http.Client{Timeout: timeout}),
	)
}
