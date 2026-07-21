package search

import (
	"context"
	"errors"
)

// ProviderSlot 是全文搜索引擎的宿主槽位。
// Host 拥有文档 schema、ACL、入队与公开搜索 API；插件实现引擎传输。
// 默认受保护内置站内搜索（sforum.search-site / PostgresSiteEngine）。
const ProviderSlot = "search.provider"

// DefaultSiteSearchExtensionID 是受保护内置站内搜索扩展 id。
// 不可卸载；RestoreDefault 与无显式 pin 时解析到此 id。
const DefaultSiteSearchExtensionID = "sforum.search-site"

// ErrEngineUnavailable 表示未配置/未启用搜索引擎提供方（外部引擎故障等）。
// 站内默认引擎正常时不应出现。
var ErrEngineUnavailable = errors.New("search: engine unavailable")

// Engine 是引擎无关的全文搜索传输接口。
// 站内引擎由 Host 进程内实现；外部引擎（Meilisearch 等）由插件 ProviderCall 适配。
type Engine interface {
	// Probe 探测引擎是否可达（可选；用于 readiness / 管理端测试）。
	Probe(ctx context.Context) error
	// EnsureIndex 幂等创建索引并应用 schema 约定的 settings。
	EnsureIndex(ctx context.Context) error
	// Index 写入或更新一篇主题搜索文档。
	Index(ctx context.Context, doc TopicSearchDoc) error
	// Delete 按主题 id 删除索引文档。
	Delete(ctx context.Context, topicID int64) error
	// Search 执行关键词检索。
	Search(ctx context.Context, input SearchInput) (SearchResult, error)
}

// UnavailableEngine 是显式不可用引擎：所有操作返回 ErrEngineUnavailable。
// 仅用于测试或刻意关闭搜索的装配；生产默认使用站内引擎。
type UnavailableEngine struct{}

func (UnavailableEngine) Probe(context.Context) error {
	return ErrEngineUnavailable
}
func (UnavailableEngine) EnsureIndex(context.Context) error {
	return ErrEngineUnavailable
}
func (UnavailableEngine) Index(context.Context, TopicSearchDoc) error {
	return ErrEngineUnavailable
}
func (UnavailableEngine) Delete(context.Context, int64) error {
	return ErrEngineUnavailable
}
func (UnavailableEngine) Search(context.Context, SearchInput) (SearchResult, error) {
	return SearchResult{}, ErrEngineUnavailable
}

// IndexUID 是宿主约定的主题索引标识；外部引擎插件应使用同一 UID 或映射到等价索引。
const IndexUID = "sforum_topics"

// PublicSearchStatuses 是公开搜索允许的主题状态（与 forum 公开列表一致）。
var PublicSearchStatuses = []string{"active", "locked"}

// IsPublicSearchStatus 判断状态是否可出现在公开搜索结果中。
func IsPublicSearchStatus(status string) bool {
	for _, s := range PublicSearchStatuses {
		if status == s {
			return true
		}
	}
	return false
}
