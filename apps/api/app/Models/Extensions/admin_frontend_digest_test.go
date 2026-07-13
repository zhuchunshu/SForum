package extensions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAdminFrontendDigestTracksOnlyAdminInputs(t *testing.T) {
	root := t.TempDir()
	writeDigestFixture(t, root, "frontend/admin/components/Settings.vue", "<template>one</template>")
	writeDigestFixture(t, root, "frontend/admin/locales/zh-CN.json", `{"title":"设置"}`)
	writeDigestFixture(t, root, "frontend/admin/locales/en-US.json", `{"title":"Settings"}`)
	writeDigestFixture(t, root, "frontend/admin/package.json", `{"name":"fixture"}`)
	writeDigestFixture(t, root, "frontend/admin/bun.lock", "lock-v1")
	payload, _ := json.Marshal(AdminComponentContributionPayload{Component: "settings"})
	manifest := Manifest{
		ID: "digest.plugin", Version: "1.0.0", Type: TypePlugin,
		Frontend:      ManifestFrontend{Admin: &ManifestAdminFrontend{Root: "frontend/admin", APIVersion: 1, Components: map[string]string{"settings": "components/Settings.vue"}, Locales: map[string]string{"zh-CN": "locales/zh-CN.json", "en-US": "locales/en-US.json"}}},
		Contributions: []ManifestContribution{{Point: "admin.extension.settings.page", ID: "settings", Payload: payload}},
	}
	first, err := ComputeAdminFrontendDigest(manifest, root)
	if err != nil {
		t.Fatal(err)
	}
	writeDigestFixture(t, root, "backend/plugin", "backend-v2")
	writeDigestFixture(t, root, "templates/home.html", "public-theme-v2")
	manifest.Settings = []ManifestSetting{{Key: "title", Label: LocalizedText{Default: "Title"}, Type: "text", Default: "changed"}}
	unchanged, err := ComputeAdminFrontendDigest(manifest, root)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged != first {
		t.Fatalf("backend/settings/public theme changes altered admin digest: %s != %s", unchanged, first)
	}
	writeDigestFixture(t, root, "frontend/admin/locales/en-US.json", `{"title":"Changed"}`)
	changed, err := ComputeAdminFrontendDigest(manifest, root)
	if err != nil {
		t.Fatal(err)
	}
	if changed == first {
		t.Fatal("locale bytes must change admin digest")
	}
	writeDigestFixture(t, root, "frontend/admin/bun.lock", "lock-v2")
	lockChanged, err := ComputeAdminFrontendDigest(manifest, root)
	if err != nil {
		t.Fatal(err)
	}
	if lockChanged == changed {
		t.Fatal("lockfile bytes must change admin digest")
	}
}

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
