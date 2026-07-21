package contentregistry

import (
	"strings"
	"testing"
)

func TestBuildCatalogEmptyOnNilRegistry(t *testing.T) {
	t.Parallel()
	var registry *Registry
	catalog := registry.BuildCatalog()
	if catalog.SchemaVersion != CatalogSchemaVersion {
		t.Fatalf("schema = %q", catalog.SchemaVersion)
	}
	if len(catalog.Content) != 0 {
		t.Fatalf("expected empty content, got %#v", catalog.Content)
	}
}

func TestBuildCatalogProjectsPublishedContent(t *testing.T) {
	t.Parallel()
	registry := New()
	core := publication("core.content", true, 'a')
	core.Content = []Declaration{
		content("core.content.block.card", KindBlock, "content.block", "core.content.block.card.schema@1"),
		content("core.content.filter.links", KindRenderFilter, "content.filter", "core.content.filter.links.schema@1"),
	}
	plugin := publication("plugin.content", false, 'c')
	plugin.Content = []Declaration{
		content("plugin.content.shortcode.vote", KindShortcode, "content.shortcode", "plugin.content.shortcode.vote.schema@1"),
	}
	if _, err := registry.ReplaceAll([]Publication{core, plugin}, false); err != nil {
		t.Fatal(err)
	}

	catalog := registry.BuildCatalog()
	if catalog.SchemaVersion != CatalogSchemaVersion || catalog.Digest == "" || catalog.Revision == 0 {
		t.Fatalf("catalog meta = %#v", catalog)
	}
	if len(catalog.Content) != 3 {
		t.Fatalf("content = %#v", catalog.Content)
	}
	// Sorted by kind then id: block, render_filter, shortcode.
	if catalog.Content[0].Kind != KindBlock || catalog.Content[0].ID != "core.content.block.card" {
		t.Fatalf("first = %#v", catalog.Content[0])
	}
	if catalog.Content[1].Kind != KindRenderFilter {
		t.Fatalf("second kind = %q", catalog.Content[1].Kind)
	}
	if catalog.Content[2].ExtensionID != "plugin.content" ||
		!strings.HasPrefix(catalog.Content[2].PackageDigest, "cc") {
		t.Fatalf("plugin entry = %#v", catalog.Content[2])
	}
}
