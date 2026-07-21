package contentregistry

import (
	"context"
	"strings"
	"testing"
)

func TestForumPostFilterEmptyRegistryIsIdentity(t *testing.T) {
	t.Parallel()
	filter := NewForumPostFilter(New())
	html, plain, err := filter.AfterHostRender(context.Background(), "<p>host</p>", "host", "topic", "new")
	if err != nil {
		t.Fatal(err)
	}
	if html != "<p>host</p>" || plain != "host" {
		t.Fatalf("identity failed: %q %q", html, plain)
	}
	if filter.HasFilterContributions() {
		t.Fatal("empty registry must report no filters")
	}
}

func TestForumPostFilterWithFilterDeclarationsStillIdentityWithoutInvoker(t *testing.T) {
	t.Parallel()
	registry := New()
	item := publication("demo.content", false, 'a')
	item.Content = []Declaration{{
		ID: "demo.content.filter.final", ContractVersion: "demo.content.filter.final@1",
		Kind: KindRenderFilter, Handler: "filter", Schema: "demo.content.filter.schema@1",
	}}
	if _, err := registry.Publish(item); err != nil {
		t.Fatal(err)
	}
	filter := NewForumPostFilter(registry)
	if !filter.HasFilterContributions() {
		t.Fatal("expected filter contributions")
	}
	// 无 Protocol 调度时不得改写 Host HTML。
	html, plain, err := filter.AfterHostRender(context.Background(),
		`<p class="language-go">safe</p>`, "safe", "topic", "1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "language-go") || plain != "safe" {
		t.Fatalf("must keep host HTML without invoker: %q %q", html, plain)
	}
}

func TestForumPostFilterNilSafe(t *testing.T) {
	t.Parallel()
	var filter *ForumPostFilter
	html, plain, err := filter.AfterHostRender(context.Background(), "<p>x</p>", "x", "topic", "new")
	if err != nil || html != "<p>x</p>" || plain != "x" {
		t.Fatalf("nil filter: %v %q %q", err, html, plain)
	}
}
