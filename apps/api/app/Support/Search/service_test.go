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
	// 引擎返回 2 条，其中 99 在权威表不存在。total 是索引侧稳定统计，不能按当前页下调。
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
	if res.Total != 2 {
		t.Fatalf("expected stable engine total 2, got %d", res.Total)
	}
}

func TestSearchLiveHydrationDoesNotRefillFromAdjacentEnginePages(t *testing.T) {
	engine := &pageAwareEngine{pages: map[int]SearchResult{
		1: {Items: []TopicSearchDoc{
			{ID: 99, Status: "active"}, {ID: 1, Status: "active"}, {ID: 1, Status: "active"}, {ID: 0, Status: "active"},
		}, Total: 5, Page: 1, PerPage: 4},
		2: {Items: []TopicSearchDoc{{ID: 2, Status: "active"}, {ID: 3, Status: "active"}}, Total: 5, Page: 2, PerPage: 4},
	}}
	live := fakeLiveSource{docs: map[int64]TopicSearchDoc{
		1: {ID: 1, Status: "active", Title: "page one"},
		2: {ID: 2, Status: "active", Title: "page two"},
		3: {ID: 3, Status: "active", Title: "page two"},
	}}
	service := NewService(engine, nil).WithLiveSource(live)
	first, err := service.Search(context.Background(), SearchInput{Query: "x", Page: 1, PerPage: 4})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Search(context.Background(), SearchInput{Query: "x", Page: 2, PerPage: 4})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 1 || first.Items[0].ID != 1 || len(second.Items) != 2 || second.Items[0].ID != 2 || second.Items[1].ID != 3 {
		t.Fatalf("page results = first:%#v second:%#v", first.Items, second.Items)
	}
	if first.Total != 5 || second.Total != 5 {
		t.Fatalf("page-dependent totals = %d/%d", first.Total, second.Total)
	}
	if len(engine.calls) != 2 || engine.calls[0].Page != 1 || engine.calls[1].Page != 2 {
		t.Fatalf("engine calls = %#v", engine.calls)
	}
	seen := map[int64]bool{}
	for _, doc := range append(first.Items, second.Items...) {
		if seen[doc.ID] {
			t.Fatalf("adjacent pages duplicate topic %d", doc.ID)
		}
		seen[doc.ID] = true
	}
}

func TestSearchLiveHydrationReturnsNoPartialResultsOnLiveFailure(t *testing.T) {
	engine := &pageAwareEngine{pages: map[int]SearchResult{1: {Items: []TopicSearchDoc{{ID: 1, Status: "active"}}, Total: 1, Page: 1, PerPage: 20}}}
	expected := errors.New("live source unavailable")
	service := NewService(engine, nil).WithLiveSource(fakeLiveSource{err: expected})
	result, err := service.Search(context.Background(), SearchInput{Query: "x"})
	if !errors.Is(err, expected) || len(result.Items) != 0 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

type stubEngine struct{}

func (stubEngine) Probe(context.Context) error                 { return nil }
func (stubEngine) EnsureIndex(context.Context) error           { return nil }
func (stubEngine) Index(context.Context, TopicSearchDoc) error { return nil }
func (stubEngine) Delete(context.Context, int64) error         { return nil }
func (stubEngine) Search(context.Context, SearchInput) (SearchResult, error) {
	return SearchResult{}, nil
}

type hitsEngine struct {
	result SearchResult
}

type pageAwareEngine struct {
	pages map[int]SearchResult
	calls []SearchInput
}

func (*pageAwareEngine) Probe(context.Context) error                 { return nil }
func (*pageAwareEngine) EnsureIndex(context.Context) error           { return nil }
func (*pageAwareEngine) Index(context.Context, TopicSearchDoc) error { return nil }
func (*pageAwareEngine) Delete(context.Context, int64) error         { return nil }
func (e *pageAwareEngine) Search(_ context.Context, input SearchInput) (SearchResult, error) {
	e.calls = append(e.calls, input)
	return e.pages[input.Page], nil
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
