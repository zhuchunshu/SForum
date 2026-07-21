package search

import (
	"context"
	"errors"
	"strings"
)

// Service 提供面向公众的主题搜索查询。
// 搜索路径完全不碰 PostgreSQL，直接委托选中的 Engine。
type Service struct {
	engine    Engine
	pageSizes TopicPageSizeResolver
}

func NewService(engine Engine, pageSizes TopicPageSizeResolver) *Service {
	if engine == nil {
		engine = UnavailableEngine{}
	}
	return &Service{engine: engine, pageSizes: pageSizes}
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
// 仅返回 active/locked 状态的公开主题（与 forum 公开列表一致；过滤由引擎侧应用）。
func (s *Service) Search(ctx context.Context, input SearchInput) (SearchResult, error) {
	if s == nil || s.engine == nil {
		return SearchResult{}, ErrEngineUnavailable
	}
	defaultPerPage := 20
	if input.PerPage <= 0 && s.pageSizes != nil {
		var err error
		defaultPerPage, err = s.pageSizes.TopicPageSize(ctx)
		if err != nil {
			return SearchResult{}, err
		}
	}
	input.Page, input.PerPage = normalizeSearchPage(input.Page, input.PerPage, defaultPerPage)
	input.Query = strings.TrimSpace(input.Query)
	input.CategorySlug = strings.TrimSpace(input.CategorySlug)
	input.TagSlug = strings.TrimSpace(input.TagSlug)

	result, err := s.engine.Search(ctx, input)
	if err != nil {
		if errors.Is(err, ErrEngineUnavailable) {
			return SearchResult{}, ErrEngineUnavailable
		}
		return SearchResult{}, err
	}
	// Host 权威 ACL：过滤非公开状态，并剥离 plainText。
	if len(result.Items) > 0 {
		filtered := make([]TopicSearchDoc, 0, len(result.Items))
		for _, doc := range result.Items {
			if !IsPublicSearchStatus(doc.Status) {
				continue
			}
			doc.PlainText = ""
			filtered = append(filtered, doc)
		}
		result.Items = filtered
	}
	return result, nil
}
