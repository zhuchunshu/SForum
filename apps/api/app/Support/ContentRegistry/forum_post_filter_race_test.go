package contentregistry

import (
	"context"
	"sync"
	"testing"
)

func TestForumPostFilterConcurrentRegistryPublicationAndRendering(t *testing.T) {
	core, err := ForumPostBodyCorePublication()
	if err != nil {
		t.Fatal(err)
	}
	plugin := publication("race.content", false, 'e')
	plugin.Content = []Declaration{{
		ID: "race.content.filter", ContractVersion: "race.content.filter@1",
		Kind: KindRenderFilter, Handler: "race.filter", Schema: "race.content.filter.schema@1",
	}}
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{core, plugin}, false); err != nil {
		t.Fatal(err)
	}
	resolver := ContentProviderResolverFunc(func(Contribution) (ProviderSet, error) {
		return ProviderSet{Filter: FilterProviderFunc(func(_ context.Context, request FilterProviderRequest) (RenderSegments, error) {
			return request.Render, nil
		})}, nil
	})
	filter, err := NewProductionForumPostFilter(ForumPostFilterConfig{
		Registry: registry, Admission: &executionTestAdmission{}, Providers: resolver,
	})
	if err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for iteration := 0; iteration < 50; iteration++ {
				html, plain, err := filter.AfterHostRender(
					context.Background(), "<p>host</p>", "host", "topic", "42", "public",
				)
				if err != nil || html != "<p>host</p>" || plain != "host" {
					t.Errorf("concurrent result html=%q plain=%q err=%v", html, plain, err)
					return
				}
			}
		}()
	}
	for iteration := 0; iteration < 50; iteration++ {
		if _, removed, err := registry.Remove(plugin.Artifact); err != nil || !removed {
			t.Fatalf("remove iteration=%d removed=%t err=%v", iteration, removed, err)
		}
		if _, err := registry.Publish(plugin); err != nil {
			t.Fatalf("publish iteration=%d err=%v", iteration, err)
		}
	}
	wait.Wait()
}
