package search

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestMemorySiteEngineIndexSearchDeleteAndACL(t *testing.T) {
	engine := NewMemorySiteEngine()
	ctx := context.Background()
	now := time.Now().UTC()

	// 公开主题：应命中
	if err := engine.Index(ctx, TopicSearchDoc{
		ID: 1, Title: "Go concurrency patterns", PlainText: "goroutine channel select",
		Status: "active", CategorySlug: "dev", TagSlugs: []string{"go"},
		CreatedAt: now, UpdatedAt: now, LastActivityAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	// locked 仍公开
	if err := engine.Index(ctx, TopicSearchDoc{
		ID: 2, Title: "Locked thread about go", PlainText: "still searchable",
		Status: "locked", CreatedAt: now, UpdatedAt: now, LastActivityAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	// draft / deleted / hidden 不得出现
	for _, doc := range []TopicSearchDoc{
		{ID: 3, Title: "draft go secret", Status: "draft", PlainText: "go"},
		{ID: 4, Title: "deleted go", Status: "deleted", PlainText: "go"},
		{ID: 5, Title: "pending go", Status: "pending", PlainText: "go"},
	} {
		doc.CreatedAt, doc.UpdatedAt, doc.LastActivityAt = now, now, now
		if err := engine.Index(ctx, doc); err != nil {
			t.Fatal(err)
		}
	}

	res, err := engine.Search(ctx, SearchInput{Query: "go", Page: 1, PerPage: 20})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 2 {
		t.Fatalf("total=%d want 2 (public only); items=%+v", res.Total, res.Items)
	}
	for _, item := range res.Items {
		if !IsPublicSearchStatus(item.Status) {
			t.Fatalf("non-public leaked: %+v", item)
		}
		if item.PlainText != "" {
			t.Fatalf("plainText must be stripped from public results: %+v", item)
		}
	}

	// category 过滤
	res, err = engine.Search(ctx, SearchInput{Query: "go", CategorySlug: "dev", Page: 1, PerPage: 10})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 || res.Items[0].ID != 1 {
		t.Fatalf("category filter = %+v", res)
	}

	// tag 过滤
	res, err = engine.Search(ctx, SearchInput{Query: "go", TagSlug: "go", Page: 1, PerPage: 10})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 || res.Items[0].ID != 1 {
		t.Fatalf("tag filter = %+v", res)
	}

	// 删除后不再命中
	if err := engine.Delete(ctx, 1); err != nil {
		t.Fatal(err)
	}
	res, err = engine.Search(ctx, SearchInput{Query: "concurrency", Page: 1, PerPage: 10})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 0 {
		t.Fatalf("after delete total=%d", res.Total)
	}
}

func TestServiceSearchStripsNonPublicEvenIfEngineLeaks(t *testing.T) {
	// 恶意/错误引擎返回 draft：Service 必须过滤。
	leaky := leakyEngine{items: []TopicSearchDoc{
		{ID: 1, Title: "public", Status: "active"},
		{ID: 2, Title: "secret", Status: "draft", PlainText: "should not appear"},
	}}
	svc := NewService(leaky, nil)
	res, err := svc.Search(context.Background(), SearchInput{Query: "x", PerPage: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 1 || res.Items[0].ID != 1 {
		t.Fatalf("ACL filter failed: %+v", res.Items)
	}
	if res.Items[0].PlainText != "" {
		t.Fatal("plainText not stripped")
	}
}

func TestIndexerWithMemoryEngine(t *testing.T) {
	engine := NewMemorySiteEngine()
	reader := fixedTopicReader{doc: TopicSearchDoc{
		ID: 42, Title: "indexed topic", PlainText: "body keyword unique-xyz",
		Status: "active", CreatedAt: time.Now(), UpdatedAt: time.Now(), LastActivityAt: time.Now(),
	}}
	indexer := NewIndexer(engine, reader, nil)
	// 无 dispatcher 时 Enqueue 是 no-op，但 IndexTopic 直接写引擎
	if err := indexer.IndexTopic(context.Background(), 42); err != nil {
		t.Fatal(err)
	}
	res, err := engine.Search(context.Background(), SearchInput{Query: "unique-xyz", Page: 1, PerPage: 5})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 {
		t.Fatalf("indexed search total=%d", res.Total)
	}
	if err := indexer.DeleteTopic(context.Background(), 42); err != nil {
		t.Fatal(err)
	}
	res, err = engine.Search(context.Background(), SearchInput{Query: "unique-xyz", Page: 1, PerPage: 5})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 0 {
		t.Fatalf("after delete total=%d", res.Total)
	}
}

type recordingIndexStateStore struct {
	indexedProvider string
	indexedTopicID  int64
	indexedSourceAt time.Time
	deletedProvider string
	deletedTopicID  int64
	err             error
}

func (s *recordingIndexStateStore) MarkIndexed(_ context.Context, providerID string, topicID int64, sourceUpdatedAt time.Time) error {
	s.indexedProvider = providerID
	s.indexedTopicID = topicID
	s.indexedSourceAt = sourceUpdatedAt
	return s.err
}

func (s *recordingIndexStateStore) MarkDeleted(_ context.Context, providerID string, topicID int64) error {
	s.deletedProvider = providerID
	s.deletedTopicID = topicID
	return s.err
}

func (*recordingIndexStateStore) ListStaleTopicIDs(context.Context, string, int) ([]int64, error) {
	return nil, nil
}

func (*recordingIndexStateStore) ListObsoleteTopicIDs(context.Context, string, int) ([]int64, error) {
	return nil, nil
}

func TestIndexerCommitsStateOnlyAfterEngineSuccess(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Microsecond)
	state := &recordingIndexStateStore{}
	engine := NewMemorySiteEngine()
	indexer := NewIndexer(engine, fixedTopicReader{doc: TopicSearchDoc{
		ID: 42, Title: "stateful", Status: "active",
		CreatedAt: now, UpdatedAt: now, LastActivityAt: now,
	}}, nil).WithIndexStateStore(state)

	if err := indexer.IndexTopic(context.Background(), 42); err != nil {
		t.Fatal(err)
	}
	if state.indexedProvider != DefaultSiteSearchExtensionID ||
		state.indexedTopicID != 42 ||
		!state.indexedSourceAt.Equal(now) {
		t.Fatalf("indexed state = %+v", state)
	}
	if err := indexer.DeleteTopic(context.Background(), 42); err != nil {
		t.Fatal(err)
	}
	if state.deletedProvider != DefaultSiteSearchExtensionID || state.deletedTopicID != 42 {
		t.Fatalf("deleted state = %+v", state)
	}
}

func TestIndexerStateFailureRemainsRetryable(t *testing.T) {
	expected := errors.New("state unavailable")
	now := time.Now().UTC()
	indexer := NewIndexer(NewMemorySiteEngine(), fixedTopicReader{doc: TopicSearchDoc{
		ID: 42, Title: "retry", Status: "active",
		CreatedAt: now, UpdatedAt: now, LastActivityAt: now,
	}}, nil).WithIndexStateStore(&recordingIndexStateStore{err: expected})

	if err := indexer.IndexTopic(context.Background(), 42); !errors.Is(err, expected) {
		t.Fatalf("IndexTopic error = %v, want state error", err)
	}
}

func TestIndexerReturnsUnavailableSoRiverCanRetry(t *testing.T) {
	indexer := NewIndexer(UnavailableEngine{}, fixedTopicReader{doc: TopicSearchDoc{ID: 42}}, nil)

	err := indexer.IndexTopic(context.Background(), 42)
	if !errors.Is(err, ErrEngineUnavailable) {
		t.Fatalf("IndexTopic error = %v, want ErrEngineUnavailable", err)
	}
	err = indexer.DeleteTopic(context.Background(), 42)
	if !errors.Is(err, ErrEngineUnavailable) {
		t.Fatalf("DeleteTopic error = %v, want ErrEngineUnavailable", err)
	}
}

func TestIsPublicSearchStatus(t *testing.T) {
	if !IsPublicSearchStatus("active") || !IsPublicSearchStatus("locked") {
		t.Fatal("active/locked should be public")
	}
	for _, s := range []string{"draft", "deleted", "pending", "hidden", ""} {
		if IsPublicSearchStatus(s) {
			t.Fatalf("%q must not be public", s)
		}
	}
}

func TestDefaultSiteSearchExtensionIDStable(t *testing.T) {
	if DefaultSiteSearchExtensionID != "sforum.search-site" {
		t.Fatalf("id drifted: %s", DefaultSiteSearchExtensionID)
	}
	if !strings.HasPrefix(DefaultSiteSearchExtensionID, "sforum.") {
		t.Fatal("expected sforum.* id")
	}
}

func TestCJKNgramTerms(t *testing.T) {
	if got := cjkNgrams("分"); got != "分" {
		t.Fatalf("single Han term = %q", got)
	}
	if got := cjkNgrams("分享代码"); got != "分 享 代 码 分享 享代 代码" {
		t.Fatalf("unigrams and overlapping bigrams = %q", got)
	}
	if got := cjkNgrams("分享 Go 代码"); got != "分 享 分享 代 码 代码" {
		t.Fatalf("separate Han runs = %q", got)
	}
	if got := cjkNgrams("哈哈"); got != "哈 哈哈" {
		t.Fatalf("terms must be deduplicated: %q", got)
	}
	if got := cjkNgrams("english only"); got != "" {
		t.Fatalf("non-CJK text must not add ngrams: %q", got)
	}
}

func TestSiteTSQueryIncludesIndexedFuzzyPath(t *testing.T) {
	predicate, rank, args := siteTSQuery("分享代玛", false)
	for _, want := range []string{"tsv @@", "metadata_tsv @@", "cjk_tsv @@", "fuzzy_text ILIKE", "OPERATOR(sforum_host_extensions.<%)"} {
		if !strings.Contains(predicate, want) {
			t.Fatalf("predicate %q missing %q", predicate, want)
		}
	}
	if !strings.Contains(rank, "sforum_host_extensions.word_similarity") ||
		!strings.Contains(rank, "sforum_host_extensions.strict_word_similarity") ||
		!strings.Contains(rank, "lower(title)") {
		t.Fatalf("rank does not boost fuzzy/title matches: %q", rank)
	}
	if len(args) != 4 || args[0] != "分享代玛" || args[1] != "分 享 代 玛 分享 享代 代玛" || args[2] != "%分享代玛%" || args[3] != "分享代玛" {
		t.Fatalf("query args = %#v", args)
	}
}

func TestMemorySiteEngineSearchesMetadata(t *testing.T) {
	engine := NewMemorySiteEngine()
	now := time.Now().UTC()
	if err := engine.Index(context.Background(), TopicSearchDoc{
		ID: 1, Title: "普通主题", Status: "active",
		AuthorUsername: "dalao", AuthorDisplayName: "小明",
		CategorySlug: "kai-fa", CategoryName: "开发交流",
		Slug: "ordinary-topic", TagSlugs: []string{"golang"},
		CreatedAt: now, UpdatedAt: now, LastActivityAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"小明", "dalao", "开发交流", "golang", "ordinary-topic"} {
		result, err := engine.Search(context.Background(), SearchInput{Query: query})
		if err != nil || result.Total != 1 {
			t.Fatalf("metadata query %q result=%+v err=%v", query, result, err)
		}
	}
}

func TestEscapeLikePattern(t *testing.T) {
	if got := escapeLikePattern(`50%_off\today`); got != `50\%\_off\\today` {
		t.Fatalf("escaped LIKE pattern = %q", got)
	}
}

type leakyEngine struct {
	items []TopicSearchDoc
}

func (leakyEngine) Probe(context.Context) error                 { return nil }
func (leakyEngine) EnsureIndex(context.Context) error           { return nil }
func (leakyEngine) Index(context.Context, TopicSearchDoc) error { return nil }
func (leakyEngine) Delete(context.Context, int64) error         { return nil }
func (e leakyEngine) Search(context.Context, SearchInput) (SearchResult, error) {
	return SearchResult{Items: e.items, Total: int64(len(e.items)), Page: 1, PerPage: 20}, nil
}

type fixedTopicReader struct {
	doc TopicSearchDoc
}

func (r fixedTopicReader) GetTopicForSearch(context.Context, int64) (TopicSearchDoc, error) {
	return r.doc, nil
}
