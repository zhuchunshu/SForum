package mediaregistry

import "testing"

func TestBuildCatalogEmptyOnNilRegistry(t *testing.T) {
	t.Parallel()
	var registry *Registry
	catalog := registry.BuildCatalog()
	if catalog.SchemaVersion != CatalogSchemaVersion {
		t.Fatalf("schema = %q", catalog.SchemaVersion)
	}
	if len(catalog.Policies) != 0 || len(catalog.Processors) != 0 || len(catalog.Variants) != 0 {
		t.Fatalf("expected empty catalog, got %#v", catalog)
	}
}

func TestBuildCatalogProjectsPublishedMedia(t *testing.T) {
	t.Parallel()
	registry := New()
	if _, err := registry.Publish(pluginPublicationForTest()); err != nil {
		t.Fatal(err)
	}
	catalog := registry.BuildCatalog()
	if catalog.SchemaVersion != CatalogSchemaVersion || catalog.Revision != 1 || catalog.Digest == "" {
		t.Fatalf("catalog header = %#v", catalog)
	}
	if len(catalog.Policies) != 1 || catalog.Policies[0].ID != "demo.media.policy" {
		t.Fatalf("policies = %#v", catalog.Policies)
	}
	if catalog.Policies[0].ExtensionID != "demo.media" {
		t.Fatalf("policy owner = %#v", catalog.Policies[0])
	}
	if len(catalog.Processors) < 3 {
		t.Fatalf("processors = %#v", catalog.Processors)
	}
	if len(catalog.Variants) != 1 || catalog.Variants[0].ID != "demo.media.thumbnail" {
		t.Fatalf("variants = %#v", catalog.Variants)
	}
	if catalog.Variants[0].BindingStatus == "" {
		t.Fatalf("expected variant binding status, got %#v", catalog.Variants[0])
	}
}
