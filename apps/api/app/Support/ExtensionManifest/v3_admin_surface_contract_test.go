package extensionmanifest

import "testing"

func TestManifestV3AdminSurfaceFreezesPlacementAndDistinctSchemas(t *testing.T) {
	manifest := completeV3Manifest()
	surface := Normalize(manifest).AdminSurfaces[0]
	if surface.PlacementID != "core.component.page.admin" ||
		surface.PlacementContractVersion != "sforum.component.page.admin@1" ||
		surface.PropsSchema == surface.ResultSchema || surface.Operation != AdminSurfaceOperationQuery {
		t.Fatalf("normalized admin surface = %#v", surface)
	}
	if err := Validate(manifest); err != nil {
		t.Fatalf("typed admin surface should validate: %v", err)
	}
}

func TestManifestV3AdminSurfaceLegacySchemaRemainsReadable(t *testing.T) {
	manifest := completeV3Manifest()
	manifest.AdminSurfaces[0] = ManifestAdminSurface{
		ID: "demo.v3.admin.legacy", ContractVersion: "demo.v3.admin.legacy@1",
		Kind: "notice", Action: "add", Label: "Legacy", Handler: "admin.legacy",
		Schema: "demo.v3.admin.legacy.schema@1", Permission: "demo.v3.manage",
	}
	normalized := Normalize(manifest).AdminSurfaces[0]
	if normalized.PropsSchema != normalized.Schema || normalized.ResultSchema != normalized.Schema ||
		normalized.Operation != AdminSurfaceOperationQuery {
		t.Fatalf("legacy admin surface = %#v", normalized)
	}
	if err := Validate(manifest); err != nil {
		t.Fatalf("legacy admin surface should remain valid: %v", err)
	}
}

func TestManifestV3AdminSurfaceRejectsUnfrozenTypedContracts(t *testing.T) {
	tests := []struct {
		name   string
		change func(*ManifestAdminSurface)
	}{
		{name: "missing placement", change: func(surface *ManifestAdminSurface) { surface.PlacementID = ""; surface.PlacementContractVersion = "" }},
		{name: "public placement", change: func(surface *ManifestAdminSurface) {
			surface.PlacementID = "core.component.page.forum.home"
			surface.PlacementContractVersion = "sforum.component.page.forum.home@1"
		}},
		{name: "placement version drift", change: func(surface *ManifestAdminSurface) {
			surface.PlacementContractVersion = "sforum.component.page.admin@2"
		}},
		{name: "missing props", change: func(surface *ManifestAdminSurface) { surface.PropsSchema = "" }},
		{name: "missing result", change: func(surface *ManifestAdminSurface) { surface.ResultSchema = "" }},
		{name: "unknown operation", change: func(surface *ManifestAdminSurface) { surface.Operation = "mutate" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := completeV3Manifest()
			test.change(&manifest.AdminSurfaces[0])
			if err := Validate(manifest); err == nil {
				t.Fatal("unfrozen typed admin surface must be rejected")
			}
		})
	}
}
