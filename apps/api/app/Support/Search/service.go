package search

import (
	"context"
	"fmt"
	"strings"

	"github.com/meilisearch/meilisearch-go"
)

// Service 提供面向公众的主题搜索查询。
// 搜索路径完全不碰 PostgreSQL，直接查询 Meilisearch。
type Service struct {
	client    meilisearch.ServiceManager
	pageSizes TopicPageSizeResolver
}

func NewService(client meilisearch.ServiceManager, pageSizes TopicPageSizeResolver) *Service {
	return &Service{client: client, pageSizes: pageSizes}
}

// maxSearchPage 限制搜索的深翻页，与 forum.normalizePage 的上限保持一致语义。
const maxSearchPage = 200

func normalizeSearchPage(page, perPage, defaultPerPage int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if page > maxSearchPage {
		page = maxSearchPage
	}
	if perPage <= 0 {
		perPage = defaultPerPage
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}

// Search 执行关键词检索，支持 categorySlug/tagSlug/status 过滤。
// 仅返回 active/locked 状态的公开主题（与 forum 公开列表一致）。
func (s *Service) Search(ctx context.Context, input SearchInput) (SearchResult, error) {
	defaultPerPage := 20
	if input.PerPage <= 0 && s.pageSizes != nil {
		var err error
		defaultPerPage, err = s.pageSizes.TopicPageSize(ctx)
		if err != nil {
			return SearchResult{}, err
		}
	}
	input.Page, input.PerPage = normalizeSearchPage(input.Page, input.PerPage, defaultPerPage)
	query := strings.TrimSpace(input.Query)

	// 构造 Meilisearch filter 表达式：仅公开状态 + 可选分类/标签过滤。
	filters := []string{
		`status IN ["active", "locked"]`,
	}
	if c := strings.TrimSpace(input.CategorySlug); c != "" {
		filters = append(filters, fmt.Sprintf(`categorySlug = "%s"`, escapeFilterValue(c)))
	}
	if t := strings.TrimSpace(input.TagSlug); t != "" {
		filters = append(filters, fmt.Sprintf(`tagSlugs = "%s"`, escapeFilterValue(t)))
	}

	req := &meilisearch.SearchRequest{
		Offset: int64((input.Page - 1) * input.PerPage),
		Limit:  int64(input.PerPage),
		Filter: strings.Join(filters, " AND "),
		// 置顶优先、最新活跃优先，与 PG 列表排序一致。
		Sort: []string{"isPinned:desc", "lastActivityAt:desc"},
	}

	resp, err := s.client.Index(IndexUID).Search(query, req)
	if err != nil {
		return SearchResult{}, fmt.Errorf("meilisearch search: %w", err)
	}

	// 使用 meilisearch-go 提供的 DecodeInto 批量解码为 TopicSearchDoc 切片。
	items := make([]TopicSearchDoc, 0, len(resp.Hits))
	if err := resp.Hits.DecodeInto(&items); err != nil {
		return SearchResult{}, fmt.Errorf("decode search hits: %w", err)
	}

	return SearchResult{
		Items:   items,
		Total:   resp.EstimatedTotalHits,
		Page:    input.Page,
		PerPage: input.PerPage,
	}, nil
}

// escapeFilterValue 转义 Meilisearch filter 字符串值中的双引号。
func escapeFilterValue(v string) string {
	return strings.ReplaceAll(v, `"`, `\"`)
}
