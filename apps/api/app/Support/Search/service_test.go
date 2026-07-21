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

type stubEngine struct{}

func (stubEngine) Probe(context.Context) error                         { return nil }
func (stubEngine) EnsureIndex(context.Context) error                   { return nil }
func (stubEngine) Index(context.Context, TopicSearchDoc) error         { return nil }
func (stubEngine) Delete(context.Context, int64) error                 { return nil }
func (stubEngine) Search(context.Context, SearchInput) (SearchResult, error) {
	return SearchResult{}, nil
}

type fakeTopicPageSizeResolver struct {
	size int
	err  error
}

func (r fakeTopicPageSizeResolver) TopicPageSize(context.Context) (int, error) {
	return r.size, r.err
}
