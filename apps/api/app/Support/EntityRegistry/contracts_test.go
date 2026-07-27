package entityregistry

import (
	"errors"
	"strings"
	"testing"
)

func publishDemo(t *testing.T) *Registry {
	t.Helper()
	registry := New()
	if _, err := registry.Publish(demoEntityPublication(strings.Repeat("ab", 32))); err != nil {
		t.Fatalf("publish: %v", err)
	}
	return registry
}

func TestEntityPermissionAllowAndDeny(t *testing.T) {
	t.Parallel()
	registry := publishDemo(t)
	entityID := "demo.catalog.entity.product"

	allowed, err := registry.EvaluatePermission(
		ActionRead, entityID, "",
		NewActorPermissions("demo.catalog.product.read"),
	)
	if err != nil || !allowed.Allowed || allowed.PermissionKey != "demo.catalog.product.read" {
		t.Fatalf("allow read = %#v err=%v", allowed, err)
	}

	_, err = registry.EvaluatePermission(
		ActionUpdate, entityID, "",
		NewActorPermissions("demo.catalog.product.read"),
	)
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("deny update err = %v", err)
	}

	// Empty actor always denies.
	_, err = registry.EvaluatePermission(ActionRead, entityID, "", nil)
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("empty actor err = %v", err)
	}
}

func TestEntityFieldPermissionAllowAndDeny(t *testing.T) {
	t.Parallel()
	registry := publishDemo(t)
	entityID := "demo.catalog.entity.product"
	fieldID := "demo.catalog.field.price"

	allowed, err := registry.EvaluatePermission(
		ActionReadField, entityID, fieldID,
		NewActorPermissions("demo.catalog.field.price.read"),
	)
	if err != nil || !allowed.Allowed || allowed.TargetID != fieldID {
		t.Fatalf("allow field read = %#v err=%v", allowed, err)
	}

	_, err = registry.EvaluatePermission(
		ActionWriteField, entityID, fieldID,
		NewActorPermissions("demo.catalog.field.price.read"),
	)
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("deny field write err = %v", err)
	}

	// Field must belong to the entity target.
	_, err = registry.EvaluatePermission(
		ActionReadField, entityID, "demo.catalog.field.missing",
		NewActorPermissions("demo.catalog.field.price.read"),
	)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing field err = %v", err)
	}
}

func TestTaxonomyPermissionAllowAndDeny(t *testing.T) {
	t.Parallel()
	registry := publishDemo(t)
	taxonomyID := "demo.catalog.taxonomy.category"

	allowed, err := registry.EvaluatePermission(
		ActionAssignTerms, taxonomyID, "",
		NewActorPermissions("demo.catalog.category.assign"),
	)
	if err != nil || !allowed.Allowed {
		t.Fatalf("allow assign = %#v err=%v", allowed, err)
	}
	_, err = registry.EvaluatePermission(
		ActionManageTerms, taxonomyID, "",
		NewActorPermissions("demo.catalog.category.assign"),
	)
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("deny manage err = %v", err)
	}
}

func TestImportExportPlanAndPolicy(t *testing.T) {
	t.Parallel()
	registry := publishDemo(t)
	entityID := "demo.catalog.entity.product"

	plan, err := registry.ImportExportPlanForEntity(entityID)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if !plan.CanImport || !plan.CanExport || plan.Policy != ImportExportAllow {
		t.Fatalf("plan = %#v", plan)
	}
	if len(plan.FieldIDs) != 2 || len(plan.TaxonomyIDs) != 1 {
		t.Fatalf("plan fields/taxonomies = %#v", plan)
	}

	allowed, err := registry.EvaluatePermission(
		ActionExport, entityID, "",
		NewActorPermissions("demo.catalog.product.export"),
	)
	if err != nil || !allowed.Allowed {
		t.Fatalf("export allow = %#v err=%v", allowed, err)
	}

	// Dry-run must never execute and must surface allow/deny without error.
	dryAllow, err := registry.DryRunImportExport(
		entityID, ActionExport, NewActorPermissions("demo.catalog.product.export"),
	)
	if err != nil || !dryAllow.DryRun || dryAllow.Executes || !dryAllow.Decision.Allowed {
		t.Fatalf("dry allow = %#v err=%v", dryAllow, err)
	}
	if dryAllow.SchemaVersion != ImportExportDryRunSchemaVersion || !dryAllow.Plan.CanExport {
		t.Fatalf("dry allow meta = %#v", dryAllow)
	}
	dryDeny, err := registry.DryRunImportExport(entityID, ActionExport, NewActorPermissions())
	if err != nil || dryDeny.Decision.Allowed || dryDeny.Decision.Reason != "permission_denied" {
		t.Fatalf("dry deny = %#v err=%v", dryDeny, err)
	}
	if _, err := registry.DryRunImportExport(entityID, ActionDelete, NewActorPermissions()); !errors.Is(err, ErrInvalid) {
		t.Fatalf("delete action dry-run err = %v", err)
	}

	// Export-only entity rejects import even with import permission key present.
	exportOnly := New()
	if _, err := exportOnly.Publish(Publication{
		Artifact: Artifact{
			ExtensionID: "demo.export", ExtensionVersion: "1.0.0",
			PackageDigest: strings.Repeat("cd", 32), VersionID: 3,
		},
		Entities: []Declaration{{
			ID: "demo.export.entity.item", ContractVersion: "demo.export.entity.item@1",
			Kind: KindEntity, Label: "Item", StorageKey: "demo.export.item",
			PermissionCreate:   "demo.export.item.create",
			PermissionRead:     "demo.export.item.read",
			PermissionUpdate:   "demo.export.item.update",
			PermissionDelete:   "demo.export.item.delete",
			PermissionExport:   "demo.export.item.export",
			ImportExportPolicy: ImportExportExportOnly,
			DeletionPolicy:     DeletionSoft,
		}},
	}); err != nil {
		t.Fatalf("export-only publish: %v", err)
	}
	_, err = exportOnly.EvaluatePermission(
		ActionImport, "demo.export.entity.item", "",
		NewActorPermissions("demo.export.item.export", "demo.export.item.import"),
	)
	if !errors.Is(err, ErrPolicyDenied) {
		t.Fatalf("export-only import err = %v", err)
	}
}

func TestDeletionPlanAndRetainPolicy(t *testing.T) {
	t.Parallel()
	registry := publishDemo(t)
	plan, err := registry.DeletionPlanForEntity("demo.catalog.entity.product")
	if err != nil || !plan.SoftDelete || plan.HardDelete || plan.Retain {
		t.Fatalf("soft plan = %#v err=%v", plan, err)
	}

	retain := New()
	if _, err := retain.Publish(Publication{
		Artifact: Artifact{
			ExtensionID: "demo.retain", ExtensionVersion: "1.0.0",
			PackageDigest: strings.Repeat("ef", 32), VersionID: 4,
		},
		Entities: []Declaration{{
			ID: "demo.retain.entity.log", ContractVersion: "demo.retain.entity.log@1",
			Kind: KindEntity, Label: "Log", StorageKey: "demo.retain.log",
			PermissionCreate:   "demo.retain.log.create",
			PermissionRead:     "demo.retain.log.read",
			PermissionUpdate:   "demo.retain.log.update",
			PermissionDelete:   "demo.retain.log.delete",
			ImportExportPolicy: ImportExportDeny,
			DeletionPolicy:     DeletionRetain,
		}},
	}); err != nil {
		t.Fatalf("retain publish: %v", err)
	}
	deletion, err := retain.DeletionPlanForEntity("demo.retain.entity.log")
	if err != nil || !deletion.Retain || deletion.SoftDelete || deletion.HardDelete {
		t.Fatalf("retain plan = %#v err=%v", deletion, err)
	}
	_, err = retain.EvaluatePermission(
		ActionDelete, "demo.retain.entity.log", "",
		NewActorPermissions("demo.retain.log.delete"),
	)
	if !errors.Is(err, ErrPolicyDenied) {
		t.Fatalf("retain delete err = %v", err)
	}
}

func TestIndexPlanForEntity(t *testing.T) {
	t.Parallel()
	registry := publishDemo(t)
	plan, err := registry.IndexPlanForEntity("demo.catalog.entity.product")
	if err != nil {
		t.Fatalf("index plan: %v", err)
	}
	if plan.StorageKey != "demo.catalog.product" || len(plan.Fields) != 2 {
		t.Fatalf("plan = %#v", plan)
	}
	// Fields sorted by id: price then sku? price < sku lexicographically.
	if plan.Fields[0].FieldID != "demo.catalog.field.price" ||
		plan.Fields[0].IndexKind != IndexNumeric ||
		plan.Fields[1].FieldID != "demo.catalog.field.sku" ||
		plan.Fields[1].IndexKind != IndexKeyword {
		t.Fatalf("fields = %#v", plan.Fields)
	}
}

func TestAllowPermissionConvenience(t *testing.T) {
	t.Parallel()
	registry := publishDemo(t)
	if err := registry.AllowPermission(
		ActionCreate, "demo.catalog.entity.product", "",
		NewActorPermissions("demo.catalog.product.create"),
	); err != nil {
		t.Fatalf("allow: %v", err)
	}
	if err := registry.AllowPermission(
		ActionCreate, "demo.catalog.entity.product", "",
		NewActorPermissions(),
	); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("deny: %v", err)
	}
}
