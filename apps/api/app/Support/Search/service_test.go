package search

import (
	"context"
	"errors"
	"testing"
)

func TestNormalizeSearchPageUsesConfiguredDefault(t *testing.T) {
	page, perPage := normalizeSearchPage(0, 0, 30)
	if page != 1 || perPage != 30 {
		t.Fatalf("normalized page = %d/%d, want 1/30", page, perPage)
	}
	_, perPage = normalizeSearchPage(1, 150, 30)
	if perPage != 100 {
		t.Fatalf("clamped perPage = %d, want 100", perPage)
	}
}

func TestSearchPropagatesPageSizeResolverError(t *testing.T) {
	expected := errors.New("settings unavailable")
	// 分页解析失败应在调用引擎前返回错误。
	service := NewService(stubEngine{}, fakeTopicPageSizeResolver{err: expected})
	_, err := service.Search(context.Background(), SearchInput{Query: "go"})
	if !errors.Is(err, expected) {
		t.Fatalf("Search error = %v, want %v", err, expected)
	}
}

func TestSearchUnavailableWithoutEngine(t *testing.T) {
	service := NewService(nil, nil)
	_, err := service.Search(context.Background(), SearchInput{Query: "go"})
	if !errors.Is(err, ErrEngineUnavailable) {
		t.Fatalf("Search error = %v, want %v", err, ErrEngineUnavailable)
	}
}

func TestSearchDropsGhostHitsViaLiveSource(t *testing.T) {
	// 引擎返回 2 条，其中 99 在权威表不存在 → 应从结果剔除并下调 total。
	engine := &hitsEngine{result: SearchResult{
		Items: []TopicSearchDoc{
			{ID: 1, Title: "real", Status: "active", Slug: "real"},
			{ID: 99, Title: "ghost", Status: "active", Slug: "ghost"},
		},
		Total:   2,
		Page:    1,
		PerPage: 20,
	}}
	live := fakeLiveSource{docs: map[int64]TopicSearchDoc{
		1: {ID: 1, Title: "real-live", Status: "active", Slug: "real", Excerpt: "from-pg"},
	}}
	service := NewService(engine, nil).WithLiveSource(live)
	res, err := service.Search(context.Background(), SearchInput{Query: "x"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Items) != 1 || res.Items[0].ID != 1 {
		t.Fatalf("expected only live topic 1, got %+v", res.Items)
	}
	if res.Items[0].Title != "real-live" || res.Items[0].Excerpt != "from-pg" {
		t.Fatalf("expected live fields, got %+v", res.Items[0])
	}
	if res.Total != 1 {
		t.Fatalf("expected total adjusted to 1, got %d", res.Total)
	}
}

type stubEngine struct{}

func (stubEngine) Probe(context.Context) error                         { return nil }
func (stubEngine) EnsureIndex(context.Context) error                   { return nil }
func (stubEngine) Index(context.Context, TopicSearchDoc) error         { return nil }
func (stubEngine) Delete(context.Context, int64) error                 { return nil }
func (stubEngine) Search(context.Context, SearchInput) (SearchResult, error) {
	return SearchResult{}, nil
}

type hitsEngine struct {
	result SearchResult
}

func (hitsEngine) Probe(context.Context) error                 { return nil }
func (hitsEngine) EnsureIndex(context.Context) error           { return nil }
func (hitsEngine) Index(context.Context, TopicSearchDoc) error { return nil }
func (hitsEngine) Delete(context.Context, int64) error         { return nil }
func (e hitsEngine) Search(context.Context, SearchInput) (SearchResult, error) {
	return e.result, nil
}

type fakeLiveSource struct {
	docs map[int64]TopicSearchDoc
	err  error
}

func (f fakeLiveSource) ListPublicByIDs(_ context.Context, ids []int64) (map[int64]TopicSearchDoc, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make(map[int64]TopicSearchDoc, len(ids))
	for _, id := range ids {
		if doc, ok := f.docs[id]; ok {
			out[id] = doc
		}
	}
	return out, nil
}

type fakeTopicPageSizeResolver struct {
	size int
	err  error
}

func (r fakeTopicPageSizeResolver) TopicPageSize(context.Context) (int, error) {
	return r.size, r.err
}
