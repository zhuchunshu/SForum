package extensionmanifest

import (
	"errors"
	"strings"
	"testing"
)

func TestAdminComponentPageValidatesAndNormalizes(t *testing.T) {
	manifest := validBaseManifest()
	manifest.Admin = ManifestAdmin{
		Entry: "/dashboard",
		Pages: []ManifestAdminPage{{
			Path: "dashboard", Label: "Dashboard", View: "COMPONENT", Menu: true,
			Component: &AdminComponent{
				ID: "Demo.Dashboard", APIVersion: 1,
				Entry: "frontend/admin/dist/dashboard.mjs", CSS: "frontend/admin/dist/dashboard.css",
			},
		}},
	}
	manifest.PackageFiles = []ManifestPackageFile{
		{ID: "demo.plugin.file.dashboard", Kind: "frontend", Path: "frontend/admin/dist/dashboard.mjs", Digest: strings.Repeat("a", 64)},
		{ID: "demo.plugin.file.dashboard-style", Kind: "asset", Path: "frontend/admin/dist/dashboard.css", Digest: strings.Repeat("b", 64)},
	}

	if err := Validate(manifest); err != nil {
		t.Fatalf("component page should validate: %v", err)
	}
	normalized := Normalize(manifest)
	page := normalized.Admin.Pages[0]
	if page.Path != "/dashboard" || page.View != "component" || page.Component == nil || page.Component.ID != "demo.dashboard" {
		t.Fatalf("unexpected normalized component page: %#v", page)
	}
	bindings := DeclaredAdminComponents(normalized)
	if len(bindings) != 1 || bindings[0].Surface != "page:/dashboard" || bindings[0].Component.ID != "demo.dashboard" {
		t.Fatalf("unexpected component bindings: %#v", bindings)
	}
}

func TestAdminComponentPageRejectsMissingArtifactAndDuplicateID(t *testing.T) {
	manifest := validBaseManifest()
	component := &AdminComponent{ID: "dashboard", APIVersion: 1, Entry: "frontend/admin/dist/dashboard.mjs"}
	manifest.Admin.Pages = []ManifestAdminPage{{Path: "/dashboard", Label: "Dashboard", View: "component", Component: component}}
	if err := Validate(manifest); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("missing package file should fail, got %v", err)
	}

	manifest.PackageFiles = []ManifestPackageFile{{
		ID: "demo.plugin.file.dashboard", Kind: "frontend", Path: component.Entry, Digest: strings.Repeat("a", 64),
	}}
	manifest.Admin.Pages = append(manifest.Admin.Pages, ManifestAdminPage{
		Path: "/reports", Label: "Reports", View: "component",
		Component: &AdminComponent{ID: "dashboard", APIVersion: 1, Entry: "frontend/admin/dist/dashboard.mjs"},
	})
	if err := Validate(manifest); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("duplicate component id should fail, got %v", err)
	}
}
