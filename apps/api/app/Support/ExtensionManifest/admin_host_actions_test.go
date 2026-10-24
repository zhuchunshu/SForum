package extensionmanifest

import (
	"errors"
	"strings"
	"testing"
)

// hostActionManifest builds a component page that declares host actions, with
// the package file the component contract requires.
func hostActionManifest(actions []string) Manifest {
	manifest := validBaseManifest()
	manifest.Admin = ManifestAdmin{
		Entry: "/catalog",
		Pages: []ManifestAdminPage{{
			Path:        "/catalog",
			Label:       "Catalog",
			View:        "component",
			Menu:        true,
			HostActions: actions,
			Component: &AdminComponent{
				ID: "catalog", APIVersion: 1,
				Entry: "frontend/admin/dist/catalog.mjs",
			},
		}},
	}
	manifest.PackageFiles = []ManifestPackageFile{{
		ID: "demo.plugin.file.catalog", Kind: "frontend",
		Path: "frontend/admin/dist/catalog.mjs", Digest: strings.Repeat("a", 64),
	}}
	return manifest
}

func TestAdminHostActionsNormalizeAndValidate(t *testing.T) {
	manifest := hostActionManifest([]string{" Extension.Install ", "extension.install", ""})
	if err := Validate(manifest); err != nil {
		t.Fatalf("component page host action should validate: %v", err)
	}
	page := EffectiveAdminPages(manifest)[0]
	if len(page.HostActions) != 1 || page.HostActions[0] != AdminHostActionExtensionInstall {
		t.Fatalf("host actions should be trimmed, lowercased and de-duplicated: %#v", page.HostActions)
	}
	if actions := AdminHostActions(); len(actions) != 1 || actions[0] != AdminHostActionExtensionInstall {
		t.Fatalf("the first host action vocabulary is extension.install only: %#v", actions)
	}
}

func TestAdminHostActionsRejectNonComponentPage(t *testing.T) {
	manifest := hostActionManifest([]string{AdminHostActionExtensionInstall})
	manifest.Admin.Pages[0].View = "about"
	manifest.Admin.Pages[0].Component = nil
	manifest.PackageFiles = nil
	if err := Validate(manifest); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("host actions on a non-component page must be rejected, got %v", err)
	}
}

func TestAdminHostActionsRejectUnknownAction(t *testing.T) {
	manifest := hostActionManifest([]string{"extensions.install"})
	if err := Validate(manifest); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("unknown host action must be rejected, got %v", err)
	}
}

func TestAdminHostActionsKeepOmittedForOrdinaryComponentPage(t *testing.T) {
	manifest := hostActionManifest(nil)
	if err := Validate(manifest); err != nil {
		t.Fatalf("a component page without host actions stays valid: %v", err)
	}
	if page := EffectiveAdminPages(manifest)[0]; page.HostActions != nil {
		t.Fatalf("omitted host actions must stay nil, got %#v", page.HostActions)
	}
}
