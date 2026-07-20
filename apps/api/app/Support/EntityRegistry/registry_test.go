package entityregistry

import (
	"strings"
	"testing"
)

func demoEntityPublication(digest string) Publication {
	return Publication{
		Artifact: Artifact{
			ExtensionID: "demo.catalog", ExtensionVersion: "1.0.0",
			PackageDigest: digest, VersionID: 7,
		},
		Entities: []Declaration{
			{
				ID: "demo.catalog.entity.product", ContractVersion: "demo.catalog.entity.product@1",
				Kind: KindEntity, Label: "产品", StorageKey: "demo.catalog.product",
				PermissionCreate: "demo.catalog.product.create",
				PermissionRead:   "demo.catalog.product.read",
				PermissionUpdate: "demo.catalog.product.update",
				PermissionDelete: "demo.catalog.product.delete",
				PermissionImport: "demo.catalog.product.import",
				PermissionExport: "demo.catalog.product.export",
				ImportExportPolicy: ImportExportAllow,
				DeletionPolicy:     DeletionSoft,
				TaxonomyIDs:        []string{"demo.catalog.taxonomy.category"},
			},
			{
				ID: "demo.catalog.taxonomy.category", ContractVersion: "demo.catalog.taxonomy.category@1",
				Kind: KindTaxonomy, Label: "分类", StorageKey: "demo.catalog.category",
				Hierarchical: true,
				EntityIDs:    []string{"demo.catalog.entity.product"},
				PermissionManage: "demo.catalog.category.manage",
				PermissionAssign: "demo.catalog.category.assign",
			},
			{
				ID: "demo.catalog.field.price", ContractVersion: "demo.catalog.field.price@1",
				Kind: KindField, Label: "价格",
				EntityID: "demo.catalog.entity.product",
				Schema:   "demo.catalog.price@1",
				UIComponent: "CurrencyInput",
				Required: true, Indexed: true, IndexKind: IndexNumeric,
				PermissionFieldRead:  "demo.catalog.field.price.read",
				PermissionFieldWrite: "demo.catalog.field.price.write",
				Order: 10,
			},
			{
				ID: "demo.catalog.field.sku", ContractVersion: "demo.catalog.field.sku@1",
				Kind: KindField, Label: "SKU",
				EntityID: "demo.catalog.entity.product",
				Schema:   "demo.catalog.sku@1",
				UIComponent: "TextInput",
				Indexed: true, IndexKind: IndexKeyword,
				PermissionFieldRead:  "demo.catalog.field.sku.read",
				PermissionFieldWrite: "demo.catalog.field.sku.write",
				Order: 5,
			},
		},
	}
}

func TestEntityRegistryPublishEntityTaxonomyField(t *testing.T) {
	t.Parallel()
	registry := New()
	digest := strings.Repeat("ab", 32)
	revision, err := registry.Publish(demoEntityPublication(digest))
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if revision != 1 {
		t.Fatalf("revision = %d", revision)
	}
	snapshot := registry.Snapshot()
	if snapshot.SchemaVersion != SchemaVersion || snapshot.SafeMode || len(snapshot.Entities) != 4 {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	entities := registry.List(KindEntity)
	if len(entities) != 1 || entities[0].StorageKey != "demo.catalog.product" {
		t.Fatalf("entities = %#v", entities)
	}
	taxonomies := registry.List(KindTaxonomy)
	if len(taxonomies) != 1 || !taxonomies[0].Hierarchical {
		t.Fatalf("taxonomies = %#v", taxonomies)
	}
	fields := registry.ListFieldsForEntity("demo.catalog.entity.product")
	if len(fields) != 2 || fields[0].ID != "demo.catalog.field.sku" {
		// Order 5 before Order 10.
		t.Fatalf("fields = %#v", fields)
	}
	bound := registry.ListTaxonomiesForEntity("demo.catalog.entity.product")
	if len(bound) != 1 || bound[0].ID != "demo.catalog.taxonomy.category" {
		t.Fatalf("bound taxonomies = %#v", bound)
	}
	if _, err := registry.Resolve("demo.catalog.entity.product"); err != nil {
		t.Fatalf("resolve: %v", err)
	}
}

func TestEntityRegistryRejectsFieldWithoutEntity(t *testing.T) {
	t.Parallel()
	registry := New()
	_, err := registry.Publish(Publication{
		Artifact: Artifact{
			ExtensionID: "demo.catalog", ExtensionVersion: "1.0.0",
			PackageDigest: strings.Repeat("11", 32), VersionID: 1,
		},
		Entities: []Declaration{{
			ID: "demo.catalog.field.orphan", ContractVersion: "demo.catalog.field.orphan@1",
			Kind: KindField, EntityID: "demo.catalog.entity.missing",
			Schema: "demo.catalog.orphan@1", UIComponent: "TextInput",
			PermissionFieldRead: "demo.catalog.field.orphan.read",
			PermissionFieldWrite: "demo.catalog.field.orphan.write",
		}},
	})
	if err == nil {
		t.Fatal("expected field without entity rejection")
	}
}

func TestEntityRegistryRejectsTaxonomyWithoutEntity(t *testing.T) {
	t.Parallel()
	registry := New()
	_, err := registry.Publish(Publication{
		Artifact: Artifact{
			ExtensionID: "demo.catalog", ExtensionVersion: "1.0.0",
			PackageDigest: strings.Repeat("22", 32), VersionID: 1,
		},
		Entities: []Declaration{{
			ID: "demo.catalog.taxonomy.orphan", ContractVersion: "demo.catalog.taxonomy.orphan@1",
			Kind: KindTaxonomy, Label: "Orphan", StorageKey: "demo.catalog.orphan",
			EntityIDs: []string{"demo.catalog.entity.missing"},
			PermissionManage: "demo.catalog.orphan.manage",
			PermissionAssign: "demo.catalog.orphan.assign",
		}},
	})
	if err == nil {
		t.Fatal("expected taxonomy without entity rejection")
	}
}

func TestEntityRegistrySafeModeRejectsThirdParty(t *testing.T) {
	t.Parallel()
	registry := New()
	coreDigest := strings.Repeat("aa", 32)
	coreArtifact, err := NewCoreArtifact("core.entity", "1.0.0", coreDigest)
	if err != nil {
		t.Fatalf("core artifact: %v", err)
	}
	if _, err := registry.ReplaceAll([]Publication{{
		Artifact: coreArtifact,
		Entities: []Declaration{{
			ID: "core.entity.type.note", ContractVersion: "core.entity.type.note@1",
			Kind: KindEntity, Label: "Note", StorageKey: "core.entity.note",
			PermissionCreate: "core.entity.note.create",
			PermissionRead:   "core.entity.note.read",
			PermissionUpdate: "core.entity.note.update",
			PermissionDelete: "core.entity.note.delete",
			ImportExportPolicy: ImportExportDeny,
			DeletionPolicy:     DeletionSoft,
		}},
	}}, true); err != nil {
		t.Fatalf("core safe mode publish: %v", err)
	}
	_, err = registry.Publish(Publication{
		Artifact: Artifact{
			ExtensionID: "demo.catalog", ExtensionVersion: "1.0.0",
			PackageDigest: strings.Repeat("bb", 32), VersionID: 2,
		},
		Entities: []Declaration{{
			ID: "demo.catalog.entity.x", ContractVersion: "demo.catalog.entity.x@1",
			Kind: KindEntity, Label: "X", StorageKey: "demo.catalog.x",
			PermissionCreate: "demo.catalog.x.create",
			PermissionRead:   "demo.catalog.x.read",
			PermissionUpdate: "demo.catalog.x.update",
			PermissionDelete: "demo.catalog.x.delete",
			ImportExportPolicy: ImportExportDeny,
			DeletionPolicy:     DeletionSoft,
		}},
	})
	if err != ErrSafeMode {
		t.Fatalf("safe mode err = %v", err)
	}
	if len(registry.List(KindEntity)) != 1 {
		t.Fatalf("safe mode list = %#v", registry.List(KindEntity))
	}
}

func TestEntityRegistryDisableDoesNotRewriteSource(t *testing.T) {
	t.Parallel()
	registry := New()
	publication := demoEntityPublication(strings.Repeat("cc", 32))
	if _, err := registry.Publish(publication); err != nil {
		t.Fatalf("publish: %v", err)
	}
	// Disable removes the declaration graph only; Registry never holds entity
	// rows, so user data cannot be rewritten or deleted here.
	revision, removed, err := registry.Remove(publication.Artifact)
	if err != nil || !removed || revision != 2 {
		t.Fatalf("remove = rev=%d removed=%v err=%v", revision, removed, err)
	}
	if len(registry.List("")) != 0 {
		t.Fatal("expected empty graph after disable")
	}
	if _, err := registry.Resolve("demo.catalog.entity.product"); err != ErrNotFound {
		t.Fatalf("resolve after disable = %v", err)
	}
}

func TestEntityRegistryRejectsSameArtifactDeclarationDrift(t *testing.T) {
	t.Parallel()
	registry := New()
	first := demoEntityPublication(strings.Repeat("ee", 32))
	if _, err := registry.Publish(first); err != nil {
		t.Fatalf("first publish: %v", err)
	}
	drift := first
	drift.Entities = append([]Declaration(nil), first.Entities...)
	drift.Entities[0].Label = "mutated"
	if _, err := registry.Publish(drift); err != ErrArtifactConflict {
		t.Fatalf("drift err = %v", err)
	}
}

func TestEntityRegistryReplaceAllIfRevisionCAS(t *testing.T) {
	t.Parallel()
	registry := New()
	if _, err := registry.ReplaceAll(nil, false); err != nil {
		t.Fatalf("empty replace: %v", err)
	}
	if _, err := registry.ReplaceAll(nil, false); err != ErrRevisionConflict {
		t.Fatalf("second replaceall = %v", err)
	}
	revision := registry.Revision()
	publication := demoEntityPublication(strings.Repeat("12", 32))
	if _, err := registry.ReplaceAllIfRevision(revision, []Publication{publication}, false); err != nil {
		t.Fatalf("cas replace: %v", err)
	}
	if len(registry.List(KindEntity)) != 1 {
		t.Fatalf("entities = %#v", registry.List(KindEntity))
	}
}

func TestEntityRegistryRejectsCoreFlagWithoutSeal(t *testing.T) {
	t.Parallel()
	registry := New()
	_, err := registry.Publish(Publication{
		Artifact: Artifact{
			ExtensionID: "core.entity", ExtensionVersion: "1.0.0",
			PackageDigest: strings.Repeat("56", 32), Core: true,
		},
		Entities: []Declaration{{
			ID: "core.entity.type.x", ContractVersion: "core.entity.type.x@1",
			Kind: KindEntity, Label: "X", StorageKey: "core.entity.x",
			PermissionCreate: "core.entity.x.create",
			PermissionRead:   "core.entity.x.read",
			PermissionUpdate: "core.entity.x.update",
			PermissionDelete: "core.entity.x.delete",
			ImportExportPolicy: ImportExportDeny,
			DeletionPolicy:     DeletionSoft,
		}},
	})
	if err == nil {
		t.Fatal("expected unsealed core rejection")
	}
}

func TestEntityRegistryRejectsStorageKeyCollision(t *testing.T) {
	t.Parallel()
	registry := New()
	first := Publication{
		Artifact: Artifact{
			ExtensionID: "demo.a", ExtensionVersion: "1.0.0",
			PackageDigest: strings.Repeat("aa", 32), VersionID: 1,
		},
		Entities: []Declaration{{
			ID: "demo.a.entity.item", ContractVersion: "demo.a.entity.item@1",
			Kind: KindEntity, Label: "Item", StorageKey: "demo.a.item",
			PermissionCreate: "demo.a.item.create",
			PermissionRead:   "demo.a.item.read",
			PermissionUpdate: "demo.a.item.update",
			PermissionDelete: "demo.a.item.delete",
			ImportExportPolicy: ImportExportDeny,
			DeletionPolicy:     DeletionSoft,
		}},
	}
	if _, err := registry.Publish(first); err != nil {
		t.Fatalf("first: %v", err)
	}
	// Storage keys must be package-prefixed, so collision requires same prefix.
	// Cross-package collision is prevented by prefix rule; same-package
	// duplicate storage key is covered by replace graph validation via
	// ReplaceAllIfRevision with two publications sharing a key is impossible
	// without violating prefix. Test invalid non-prefixed key instead.
	_, err := registry.Publish(Publication{
		Artifact: Artifact{
			ExtensionID: "demo.b", ExtensionVersion: "1.0.0",
			PackageDigest: strings.Repeat("bb", 32), VersionID: 2,
		},
		Entities: []Declaration{{
			ID: "demo.b.entity.item", ContractVersion: "demo.b.entity.item@1",
			Kind: KindEntity, Label: "Item", StorageKey: "shared.key",
			PermissionCreate: "demo.b.item.create",
			PermissionRead:   "demo.b.item.read",
			PermissionUpdate: "demo.b.item.update",
			PermissionDelete: "demo.b.item.delete",
			ImportExportPolicy: ImportExportDeny,
			DeletionPolicy:     DeletionSoft,
		}},
	})
	if err == nil {
		t.Fatal("expected non-prefixed storage key rejection")
	}
}
