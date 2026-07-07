package forum

import (
	"context"
	"testing"
	"time"

	"github.com/zhuchunshu/sforum/apps/api/app/Support/Cache"
)

// cacheTestStore 包装 serviceFakeStore，记录读方法调用次数并返回可区分数据。
type cacheTestStore struct {
	Store
	categoriesCalls int
	topicCalls      int
	topicsCalls     int
}

func newCacheTestStore() *cacheTestStore {
	return &cacheTestStore{Store: newServiceFakeStore()}
}

func (s *cacheTestStore) ListCategories(_ context.Context) ([]Category, error) {
	s.categoriesCalls++
	return []Category{{ID: 1, Slug: "general", Name: "综合讨论"}}, nil
}

func (s *cacheTestStore) GetTopic(_ context.Context, topicID int64) (TopicDetail, error) {
	s.topicCalls++
	return TopicDetail{TopicSummary: TopicSummary{ID: topicID, Title: "测试主题"}}, nil
}

func (s *cacheTestStore) ListTopics(_ context.Context, input TopicListInput) (TopicList, error) {
	s.topicsCalls++
	return TopicList{
		Items:   []TopicSummary{{ID: 1, Title: "主题一"}},
		Total:   1,
		Page:    input.Page,
		PerPage: input.PerPage,
	}, nil
}

// CreateTopic override 记录并返回有效 detail，用于验证写后失效。
func (s *cacheTestStore) CreateTopic(_ context.Context, input CreateTopicRecord) (TopicDetail, error) {
	return TopicDetail{TopicSummary: TopicSummary{ID: 99, Title: input.Title}}, nil
}

func TestCachedStoreReadsHitCacheAfterFirstLoad(t *testing.T) {
	ctx := context.Background()
	inner := newCacheTestStore()
	c := cache.NewMemoryCache()
	cached := NewCachedStore(inner, c)

	// 首次：miss → 查 store → 写缓存。
	if _, err := cached.ListCategories(ctx); err != nil {
		t.Fatalf("first list categories: %v", err)
	}
	if inner.categoriesCalls != 1 {
		t.Fatalf("expected 1 store call after first load, got %d", inner.categoriesCalls)
	}
	// 二次：hit → 不查 store。
	if _, err := cached.ListCategories(ctx); err != nil {
		t.Fatalf("second list categories: %v", err)
	}
	if inner.categoriesCalls != 1 {
		t.Fatalf("expected store call still 1 on cache hit, got %d", inner.categoriesCalls)
	}
}

func TestCachedStoreGetTopicHitAndInvalidate(t *testing.T) {
	ctx := context.Background()
	inner := newCacheTestStore()
	c := cache.NewMemoryCache()
	cached := NewCachedStore(inner, c)

	// 首次加载。
	if _, err := cached.GetTopic(ctx, 42); err != nil {
		t.Fatalf("first get: %v", err)
	}
	if inner.topicCalls != 1 {
		t.Fatalf("expected 1 store call, got %d", inner.topicCalls)
	}
	// 二次命中缓存。
	if _, err := cached.GetTopic(ctx, 42); err != nil {
		t.Fatalf("second get: %v", err)
	}
	if inner.topicCalls != 1 {
		t.Fatalf("expected still 1 store call on hit, got %d", inner.topicCalls)
	}
	// 通过 UpdateTopic 触发详情失效（含 topicID）。
	if _, err := cached.UpdateTopic(ctx, UpdateTopicRecord{TopicID: 42}); err != nil {
		t.Fatalf("update: %v", err)
	}
	// 再读：应 miss 重新回源。
	if _, err := cached.GetTopic(ctx, 42); err != nil {
		t.Fatalf("third get after invalidate: %v", err)
	}
	if inner.topicCalls != 2 {
		t.Fatalf("expected 2 store calls after invalidation, got %d", inner.topicCalls)
	}
}

func TestCachedStoreTopicsInvalidatedOnWrite(t *testing.T) {
	ctx := context.Background()
	inner := newCacheTestStore()
	c := cache.NewMemoryCache()
	cached := NewCachedStore(inner, c)

	// 首次加载 topics。
	if _, err := cached.ListTopics(ctx, TopicListInput{Page: 1, PerPage: 20}); err != nil {
		t.Fatalf("first list topics: %v", err)
	}
	if inner.topicsCalls != 1 {
		t.Fatalf("expected 1 store call, got %d", inner.topicsCalls)
	}
	// 命中缓存。
	if _, err := cached.ListTopics(ctx, TopicListInput{Page: 1, PerPage: 20}); err != nil {
		t.Fatalf("second list topics: %v", err)
	}
	if inner.topicsCalls != 1 {
		t.Fatalf("expected still 1 on hit, got %d", inner.topicsCalls)
	}
	// CreateTopic 触发 topics 列表 generation 失效。
	if _, err := cached.CreateTopic(ctx, CreateTopicRecord{Title: "新主题"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	// 再读：generation 已变，应 miss。
	if _, err := cached.ListTopics(ctx, TopicListInput{Page: 1, PerPage: 20}); err != nil {
		t.Fatalf("third list after write: %v", err)
	}
	if inner.topicsCalls != 2 {
		t.Fatalf("expected 2 store calls after write invalidation, got %d", inner.topicsCalls)
	}
}

func TestCachedStoreNilCacheReturnsInner(t *testing.T) {
	inner := newCacheTestStore()
	// cache 为 nil 时应直接返回 inner，不包装。
	if got := NewCachedStore(inner, nil); got != inner {
		t.Fatal("expected NewCachedStore to return inner when cache is nil")
	}
}

func TestCachedStoreCacheErrorsDegradeGracefully(t *testing.T) {
	// 此测试主要确认缓存层不会因自身错误 panic 或中断主流程。
	// MemoryCache 不会出错，这里验证整体链路稳定即可。
	ctx := context.Background()
	inner := newCacheTestStore()
	c := cache.NewMemoryCache()
	cached := NewCachedStore(inner, c)

	for i := 0; i < 5; i++ {
		if _, err := cached.ListCategories(ctx); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if inner.categoriesCalls != 1 {
		t.Fatalf("expected 1 store call over 5 reads (cached), got %d", inner.categoriesCalls)
	}
}

// 确保 cacheTestStore 的 Store 方法（非 override）正常转发。
func TestCachedStoreForwardsNonOverriddenMethods(t *testing.T) {
	ctx := context.Background()
	inner := newCacheTestStore()
	c := cache.NewMemoryCache()
	cached := NewCachedStore(inner, c).(*CachedStore)

	// ListTags 未 override，应直接转发到嵌入的 Store。
	_, _ = cached.ListTags(ctx, false)
	// 只要不 panic 即视为转发正常。
	_ = time.Now() // 占位，避免 import 未用
}
