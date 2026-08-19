package forum

import (
	"context"
	"errors"
	"testing"
)

type topicEventLinkResolverStub struct {
	url string
	err error
}

type topicEventSnapshotStoreStub struct {
	Store
	snapshot TopicSummary
	err      error
}

func (s topicEventSnapshotStoreStub) GetTopicEventSnapshot(context.Context, int64) (TopicSummary, error) {
	return s.snapshot, s.err
}

func (s topicEventLinkResolverStub) TopicEventURL(context.Context, TopicSummary) (string, error) {
	return s.url, s.err
}

func TestTopicEventPayloadIncludesPublicSafeResolvedSnapshot(t *testing.T) {
	service := &Service{topicEventLinks: topicEventLinkResolverStub{url: "https://forum.example/t/42/hello"}}
	payload := buildTopicEventPayload(context.Background(), service.topicEventLinks, TopicSummary{
		ID:           42,
		CategoryID:   3,
		CategorySlug: "general",
		CategoryName: "General discussion",
		AuthorUserID: 7,
		Author:       &UserSummary{ID: 7, Username: "alice", DisplayName: "Alice"},
		Title:        "Hello",
		Slug:         "hello",
		Status:       TopicStatusActive,
		Tags: []TopicTagSummary{
			{ID: 9, Slug: "code", Name: "Code", Status: TagStatusActive},
		},
	})

	if payload["topicId"] != int64(42) || payload["authorUserId"] != int64(7) || payload["categorySlug"] != "general" {
		t.Fatalf("legacy identifiers missing from payload: %#v", payload)
	}
	if payload["url"] != "https://forum.example/t/42/hello" || payload["path"] != "/t/42/hello" {
		t.Fatalf("unexpected topic links: url=%#v path=%#v", payload["url"], payload["path"])
	}
	author, ok := payload["author"].(map[string]any)
	if !ok || author["username"] != "alice" || author["displayName"] != "Alice" || len(author) != 3 {
		t.Fatalf("unexpected safe author snapshot: %#v", payload["author"])
	}
	category, ok := payload["category"].(map[string]any)
	if !ok || category["id"] != int64(3) || category["slug"] != "general" || category["name"] != "General discussion" {
		t.Fatalf("unexpected category snapshot: %#v", payload["category"])
	}
	tags, ok := payload["tags"].([]map[string]any)
	if !ok || len(tags) != 1 || tags[0]["slug"] != "code" || tags[0]["name"] != "Code" {
		t.Fatalf("unexpected tag snapshots: %#v", payload["tags"])
	}
	tagSlugs, ok := payload["tagSlugs"].([]string)
	if !ok || len(tagSlugs) != 1 || tagSlugs[0] != "code" {
		t.Fatalf("unexpected legacy tag slugs: %#v", payload["tagSlugs"])
	}
}

func TestTopicEventPayloadFallsBackToStableRelativePath(t *testing.T) {
	service := &Service{topicEventLinks: topicEventLinkResolverStub{err: errors.New("settings unavailable")}}
	payload := buildTopicEventPayload(context.Background(), service.topicEventLinks, TopicSummary{ID: 42, Slug: "hello"})
	if payload["url"] != "/t/42/hello" || payload["path"] != "/t/42/hello" {
		t.Fatalf("unexpected fallback links: %#v", payload)
	}
}

func TestResolveTopicEventSnapshotIsFailOpen(t *testing.T) {
	fallback := TopicSummary{ID: 42, Title: "Committed topic"}
	failed := topicEventSnapshotStoreStub{err: errors.New("taxonomy read failed")}
	if got := resolveTopicEventSnapshot(context.Background(), failed, fallback); got.Title != fallback.Title {
		t.Fatalf("snapshot failure changed committed fallback: %#v", got)
	}

	resolved := topicEventSnapshotStoreStub{snapshot: TopicSummary{ID: 42, Title: "Committed topic", CategoryName: "General"}}
	if got := resolveTopicEventSnapshot(context.Background(), resolved, fallback); got.CategoryName != "General" {
		t.Fatalf("resolved snapshot was not used: %#v", got)
	}
}
