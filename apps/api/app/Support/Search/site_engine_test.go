package search

import (
	"context"
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

type leakyEngine struct {
	items []TopicSearchDoc
}

func (leakyEngine) Probe(context.Context) error                         { return nil }
func (leakyEngine) EnsureIndex(context.Context) error                   { return nil }
func (leakyEngine) Index(context.Context, TopicSearchDoc) error         { return nil }
func (leakyEngine) Delete(context.Context, int64) error                 { return nil }
func (e leakyEngine) Search(context.Context, SearchInput) (SearchResult, error) {
	return SearchResult{Items: e.items, Total: int64(len(e.items)), Page: 1, PerPage: 20}, nil
}

type fixedTopicReader struct {
	doc TopicSearchDoc
}

func (r fixedTopicReader) GetTopicForSearch(context.Context, int64) (TopicSearchDoc, error) {
	return r.doc, nil
}
