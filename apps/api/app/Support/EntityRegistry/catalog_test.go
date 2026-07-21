package entityregistry

import (
	"strings"
	"testing"
)

func TestBuildCatalogEmptyOnNilRegistry(t *testing.T) {
	t.Parallel()
	var registry *Registry
	catalog := registry.BuildCatalog()
	if catalog.SchemaVersion != CatalogSchemaVersion || len(catalog.Entities) != 0 {
		t.Fatalf("nil catalog = %#v", catalog)
	}
}

func TestBuildCatalogProjectsEntityPlansAndFields(t *testing.T) {
	t.Parallel()
	registry := New()
	if _, err := registry.Publish(demoEntityPublication(strings.Repeat("ab", 32))); err != nil {
		t.Fatal(err)
	}
	catalog := registry.BuildCatalog()
	if catalog.SchemaVersion != CatalogSchemaVersion || catalog.Revision != 1 || len(catalog.Entities) != 1 {
		t.Fatalf("catalog = %#v", catalog)
	}
	entity := catalog.Entities[0]
	if entity.ID != "demo.catalog.entity.product" || entity.StorageKey != "demo.catalog.product" {
		t.Fatalf("entity = %#v", entity)
	}
	if !entity.ImportExport.CanImport || !entity.ImportExport.CanExport {
		t.Fatalf("importExport = %#v", entity.ImportExport)
	}
	if len(entity.Index.Fields) == 0 {
		t.Fatalf("expected indexed fields, got %#v", entity.Index)
	}
	if len(entity.Fields) == 0 || len(entity.Taxonomies) == 0 {
		t.Fatalf("fields/taxonomies = %#v %#v", entity.Fields, entity.Taxonomies)
	}
}
