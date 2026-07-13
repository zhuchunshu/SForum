package extensions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAdminFrontendDigestTracksPrebuiltEntryAndCSS(t *testing.T) {
	root := t.TempDir()
	writeDigestFixture(t, root, "frontend/admin/dist/settings.mjs", "export const apiVersion = 1")
	writeDigestFixture(t, root, "frontend/admin/dist/settings.css", ".settings { color: red }")
	manifest := Manifest{SettingsDocument: SettingsDocument{SchemaVersion: 1, UI: SettingsUI{
		Mode: "component", Component: &SettingsComponent{ID: "settings", APIVersion: 1, Entry: "frontend/admin/dist/settings.mjs", CSS: "frontend/admin/dist/settings.css"},
	}}}
	first, err := ComputeAdminFrontendDigest(manifest, root)
	if err != nil {
		t.Fatal(err)
	}
	writeDigestFixture(t, root, "frontend/admin/dist/settings.mjs", "export const apiVersion = 1\nexport const changed = true")
	second, err := ComputeAdminFrontendDigest(manifest, root)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("prebuilt entry bytes must change admin digest")
	}
}

func TestAdminFrontendDigestRejectsSymlinkedPrebuiltAsset(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "settings.mjs")
	if err := os.WriteFile(outside, []byte("export {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "frontend/admin/dist/settings.mjs")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, target); err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{SettingsDocument: SettingsDocument{UI: SettingsUI{Component: &SettingsComponent{Entry: "frontend/admin/dist/settings.mjs"}}}}
	if _, err := ComputeAdminFrontendDigest(manifest, root); err == nil {
		t.Fatal("symlinked prebuilt asset must be rejected")
	}
}

func writeDigestFixture(t *testing.T, root, relative, body string) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
