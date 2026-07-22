package search

import (
	"context"
	"errors"
	"strings"
)

// Service 提供面向公众的主题搜索查询。
// 关键词检索委托选中的 Engine；可选 LiveTopicSource 用权威 topics 表剔除幽灵命中。
type Service struct {
	engine    Engine
	pageSizes TopicPageSizeResolver
	live      LiveTopicSource
}

func NewService(engine Engine, pageSizes TopicPageSizeResolver) *Service {
	if engine == nil {
		engine = UnavailableEngine{}
	}
	return &Service{engine: engine, pageSizes: pageSizes}
}

// WithLiveSource 注入权威主题校验源。生产路径应始终注入，避免引擎脏索引返回 404 帖。
func (s *Service) WithLiveSource(live LiveTopicSource) *Service {
	if s == nil {
		return s
	}
	s.live = live
	return s
}

// maxSearchPage 限制搜索的深翻页，与 forum.normalizePage 的上限保持一致语义。
const maxSearchPage = 200

// maxLiveRefillPages 幽灵命中过多时，最多再向引擎多取几页以凑满 perPage。
const maxLiveRefillPages = 5

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
// 若注入 LiveTopicSource，会剔除引擎脏索引中的幽灵文档，并尽量回填满当前页。
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

	// 无 live 源：单次引擎查询 + 状态过滤（旧行为）。
	if s.live == nil {
		return s.searchOnce(ctx, input)
	}
	return s.searchWithLiveHydration(ctx, input)
}

func (s *Service) searchOnce(ctx context.Context, input SearchInput) (SearchResult, error) {
	result, err := s.engine.Search(ctx, input)
	if err != nil {
		if errors.Is(err, ErrEngineUnavailable) {
			return SearchResult{}, ErrEngineUnavailable
		}
		return SearchResult{}, err
	}
	result.Items = filterPublicDocs(result.Items)
	return result, nil
}

func (s *Service) searchWithLiveHydration(ctx context.Context, input SearchInput) (SearchResult, error) {
	// 从请求页开始向后扫引擎页，hydrate 后凑满 perPage；total 按本趟丢弃数下调（近似）。
	var (
		collected []TopicSearchDoc
		total     int64
		dropped   int64
		seenIDs   = map[int64]struct{}{}
	)
	enginePage := input.Page
	for attempt := 0; attempt <= maxLiveRefillPages; attempt++ {
		if enginePage > maxSearchPage {
			break
		}
		pageInput := input
		pageInput.Page = enginePage
		raw, err := s.engine.Search(ctx, pageInput)
		if err != nil {
			if errors.Is(err, ErrEngineUnavailable) {
				return SearchResult{}, ErrEngineUnavailable
			}
			return SearchResult{}, err
		}
		if attempt == 0 {
			total = raw.Total
		}
		candidates := filterPublicDocs(raw.Items)
		if len(candidates) == 0 {
			break
		}
		// 跨页去重，避免引擎分页重叠或 refill 重复。
		deduped := make([]TopicSearchDoc, 0, len(candidates))
		for _, doc := range candidates {
			if doc.ID <= 0 {
				continue
			}
			if _, ok := seenIDs[doc.ID]; ok {
				continue
			}
			seenIDs[doc.ID] = struct{}{}
			deduped = append(deduped, doc)
		}
		hydrated, pageDropped, liveErr := hydrateLiveSearchHits(ctx, s.live, deduped)
		if liveErr != nil {
			return SearchResult{}, liveErr
		}
		dropped += pageDropped
		for _, doc := range hydrated {
			collected = append(collected, doc)
			if len(collected) >= input.PerPage {
				break
			}
		}
		if len(collected) >= input.PerPage {
			break
		}
		// 引擎本页已不足一页，没有更多可 refill。
		if len(raw.Items) < input.PerPage {
			break
		}
		enginePage++
	}
	if collected == nil {
		collected = []TopicSearchDoc{}
	}
	if len(collected) > input.PerPage {
		collected = collected[:input.PerPage]
	}
	if dropped > 0 && total >= dropped {
		total -= dropped
	}
	return SearchResult{
		Items:   collected,
		Total:   total,
		Page:    input.Page,
		PerPage: input.PerPage,
	}, nil
}

// hydrateLiveSearchHits 保持引擎排序，用 live 快照替换字段；缺失 id 丢弃。
func hydrateLiveSearchHits(ctx context.Context, live LiveTopicSource, items []TopicSearchDoc) ([]TopicSearchDoc, int64, error) {
	if live == nil || len(items) == 0 {
		return items, 0, nil
	}
	ids := make([]int64, 0, len(items))
	seen := make(map[int64]struct{}, len(items))
	for _, doc := range items {
		if doc.ID <= 0 {
			continue
		}
		if _, ok := seen[doc.ID]; ok {
			continue
		}
		seen[doc.ID] = struct{}{}
		ids = append(ids, doc.ID)
	}
	if len(ids) == 0 {
		return []TopicSearchDoc{}, int64(len(items)), nil
	}
	liveByID, err := live.ListPublicByIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	out := make([]TopicSearchDoc, 0, len(items))
	var dropped int64
	for _, doc := range items {
		liveDoc, ok := liveByID[doc.ID]
		if !ok {
			dropped++
			continue
		}
		liveDoc.PlainText = ""
		out = append(out, liveDoc)
	}
	return out, dropped, nil
}
