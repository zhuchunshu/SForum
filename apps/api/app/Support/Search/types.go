package search

import (
	"context"
	"time"
)

// TopicReader 抽象 forum 包对主题详情的读取，供 Indexer 构造搜索文档。
// 由 forum.Service 或其 store 实现并注入，避免 search 反向依赖 forum。
type TopicReader interface {
	// GetTopicForSearch 返回用于建索引的主题快照。注意：与 forum.GetTopicForAction
	// 类似，不做公开可见性过滤——已删除/隐藏主题的索引删除由 EnqueueDelete 负责。
	GetTopicForSearch(ctx context.Context, topicID int64) (TopicSearchDoc, error)
}

// TopicIDSource 抽象 forum 包扫描全部可索引主题 ID 的能力，供 Reindex 批量入队。
// 由 forum.Store 实现并注入，避免 search 反向依赖 forum。
type TopicIDSource interface {
	ListAllTopicIDs(ctx context.Context) ([]int64, error)
}

// TopicPageSizeResolver supplies the operator-configured default without
// coupling Search to the Forum package.
type TopicPageSizeResolver interface {
	TopicPageSize(ctx context.Context) (int, error)
}

// TopicSearchDoc 是写入搜索引擎的主题文档结构。字段与 forum.TopicSummary 对齐，
// 但独立声明以解耦。tagSlugs 用数组供引擎侧过滤。
type TopicSearchDoc struct {
	ID                int64     `json:"id"`
	Title             string    `json:"title"`
	Excerpt           string    `json:"excerpt"`
	PlainText         string    `json:"plainText"` // 用于全文检索的纯文本正文
	CategoryID        int64     `json:"categoryId"`
	CategorySlug      string    `json:"categorySlug"`
	CategoryName      string    `json:"categoryName"`
	AuthorUserID      int64     `json:"authorUserId"`
	AuthorUsername    string    `json:"authorUsername"`
	AuthorDisplayName string    `json:"authorDisplayName"`
	Slug              string    `json:"slug"`
	Status            string    `json:"status"`
	IsPinned          bool      `json:"isPinned"`
	CommentCount      int64     `json:"commentCount"`
	ViewCount         int64     `json:"viewCount"`
	TagSlugs          []string  `json:"tagSlugs"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
	LastActivityAt    time.Time `json:"lastActivityAt"`
}

// SearchInput 是搜索请求的归一化输入。
type SearchInput struct {
	Query        string
	CategorySlug string
	TagSlug      string
	Page         int
	PerPage      int
}

// SearchResult 是搜索响应，结构与 forum.TopicList 对齐便于前端复用。
type SearchResult struct {
	Items   []TopicSearchDoc `json:"items"`
	Total   int64            `json:"total"`
	Page    int              `json:"page"`
	PerPage int              `json:"perPage"`
}
