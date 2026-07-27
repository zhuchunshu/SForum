package extensionmanifest

import (
	"encoding/json"
	"testing"
)

func TestV3EntityTaxonomyFieldDeclarations(t *testing.T) {
	t.Parallel()
	manifest := completeV3Manifest()
	manifest.Entities = []ManifestEntity{
		{
			ID: "demo.v3.entity.product", ContractVersion: "demo.v3.entity.product@1",
			Kind: "entity", Label: "Product", StorageKey: "demo.v3.product",
			PermissionCreate:   "demo.v3.product.create",
			PermissionRead:     "demo.v3.product.read",
			PermissionUpdate:   "demo.v3.product.update",
			PermissionDelete:   "demo.v3.product.delete",
			PermissionImport:   "demo.v3.product.import",
			PermissionExport:   "demo.v3.product.export",
			ImportExportPolicy: "allow",
			DeletionPolicy:     "soft",
			TaxonomyIDs:        []string{"demo.v3.taxonomy.category"},
		},
		{
			ID: "demo.v3.taxonomy.category", ContractVersion: "demo.v3.taxonomy.category@1",
			Kind: "taxonomy", Label: "Category", StorageKey: "demo.v3.category",
			Hierarchical:     true,
			EntityIDs:        []string{"demo.v3.entity.product"},
			PermissionManage: "demo.v3.category.manage",
			PermissionAssign: "demo.v3.category.assign",
		},
		{
			ID: "demo.v3.field.price", ContractVersion: "demo.v3.field.price@1",
			Kind: "field", Label: "Price",
			EntityID:    "demo.v3.entity.product",
			Schema:      "demo.v3.price@1",
			UIComponent: "CurrencyInput",
			Required:    true, Indexed: true, IndexKind: "numeric",
			PermissionFieldRead:  "demo.v3.field.price.read",
			PermissionFieldWrite: "demo.v3.field.price.write",
			Order:                10,
		},
	}
	if err := Validate(manifest); err != nil {
		t.Fatalf("Validate entities: %v", err)
	}
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := ValidateV3JSONSchema(body); err != nil {
		t.Fatalf("ValidateV3JSONSchema entities: %v", err)
	}
}

func TestV3EntityRejectsFieldWithoutEntityAndMissingPermissions(t *testing.T) {
	t.Parallel()
	manifest := completeV3Manifest()
	manifest.Entities = []ManifestEntity{{
		ID: "demo.v3.field.orphan", ContractVersion: "demo.v3.field.orphan@1",
		Kind: "field", EntityID: "demo.v3.entity.missing",
		Schema: "demo.v3.orphan@1", UIComponent: "TextInput",
		PermissionFieldRead:  "demo.v3.field.orphan.read",
		PermissionFieldWrite: "demo.v3.field.orphan.write",
	}}
	if err := Validate(manifest); err == nil {
		t.Fatal("expected field without entity to fail")
	}

	manifest = completeV3Manifest()
	manifest.Entities = []ManifestEntity{{
		ID: "demo.v3.entity.product", ContractVersion: "demo.v3.entity.product@1",
		Kind: "entity", Label: "Product", StorageKey: "demo.v3.product",
		// Missing CRUD permissions.
		ImportExportPolicy: "deny",
		DeletionPolicy:     "soft",
	}}
	if err := Validate(manifest); err == nil {
		t.Fatal("expected entity without permissions to fail")
	}
}

func TestV3EntityCrossPackageFieldRequiresRequiredDependency(t *testing.T) {
	t.Parallel()
	manifest := completeV3Manifest()
	manifest.Entities = []ManifestEntity{{
		ID: "demo.v3.field.sale", ContractVersion: "demo.v3.field.sale@1",
		Kind: "field", Label: "Sale",
		EntityID:             "owner.catalog.entity.product",
		Schema:               "demo.v3.sale@1",
		UIComponent:          "CurrencyInput",
		PermissionFieldRead:  "demo.v3.field.sale.read",
		PermissionFieldWrite: "demo.v3.field.sale.write",
	}}
	if err := Validate(manifest); err == nil {
		t.Fatal("expected foreign field without dependency to fail")
	}
	manifest.Dependencies = append(manifest.Dependencies, ManifestDependency{
		ID: "owner.catalog", Version: "^1.0.0", Kind: "optional",
	})
	if err := Validate(manifest); err == nil {
		t.Fatal("expected optional dependency to fail for cross-package field")
	}
	manifest.Dependencies[len(manifest.Dependencies)-1].Kind = "required"
	if err := Validate(manifest); err != nil {
		t.Fatalf("required dependency should allow cross-package field: %v", err)
	}
}
