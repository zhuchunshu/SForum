package webreleaseruntime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

func TestGenerateRegistryWritesDeterministicMetadataLocalesAndLiteralLoaders(t *testing.T) {
	root := t.TempDir()
	pluginRoot := filepath.Join(root, "extensions", "demo.plugin")
	adminRoot := filepath.Join(pluginRoot, "frontend", "admin")
	if err := os.MkdirAll(filepath.Join(adminRoot, "components"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(adminRoot, "locales"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(adminRoot, "components", "Cell.vue"), []byte("<template />"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(adminRoot, "locales", "zh-CN.json"), []byte(`{"action":{"run":"运行"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(adminRoot, "locales", "en-US.json"), []byte(`{"action":{"run":"Run"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]any{"component": "cell", "width": 120})
	input := RegistryInput{
		Root:      filepath.Join(root, "registry"),
		ReleaseID: 42,
		Extensions: []RegistryExtension{{
			SourceRoot: pluginRoot,
			Snapshot: extensions.WebReleaseExtension{
				ExtensionID: "demo.plugin", FrontendRoot: "frontend/admin",
				ComponentMap:      map[string]string{"cell": "components/Cell.vue"},
				LocaleMap:         map[string]string{"zh-CN": "locales/zh-CN.json", "en-US": "locales/en-US.json"},
				TrustedComponents: []extensions.ManifestContribution{{Point: "admin.test.fixture", ID: "latency", Order: 10, Label: map[string]string{"en-US": "Latency"}, Payload: payload}},
			},
		}},
	}

	result, err := GenerateRegistry(input)
	if err != nil {
		t.Fatal(err)
	}
	metadata, _ := os.ReadFile(result.MetadataPath)
	registry, _ := os.ReadFile(result.RegistryPath)
	if !strings.Contains(string(metadata), `export const releaseId = "42"`) || !strings.Contains(string(metadata), `"demo.plugin"`) || !strings.Contains(string(metadata), `"width":120`) {
		t.Fatalf("unexpected metadata:\n%s", metadata)
	}
	if !strings.Contains(string(registry), `"demo.plugin:latency": () => import("../extensions/demo.plugin/frontend/admin/components/Cell.vue")`) {
		t.Fatalf("registry must contain a static literal lazy import:\n%s", registry)
	}

	second := filepath.Join(root, "registry-second")
	input.Root = second
	if _, err := GenerateRegistry(input); err != nil {
		t.Fatal(err)
	}
	secondMetadata, _ := os.ReadFile(filepath.Join(second, "metadata.ts"))
	if string(metadata) != string(secondMetadata) {
		t.Fatal("metadata generation is not deterministic")
	}
}

func TestGenerateRegistryRejectsModuleRootEscape(t *testing.T) {
	_, err := GenerateRegistry(RegistryInput{
		Root: t.TempDir(), ReleaseID: 1,
		Extensions: []RegistryExtension{{SourceRoot: t.TempDir(), Snapshot: extensions.WebReleaseExtension{
			ExtensionID: "demo.plugin", FrontendRoot: "frontend/admin",
			ComponentMap: map[string]string{"cell": "../escape.vue"},
		}}},
	})
	if err == nil {
		t.Fatal("expected module root escape rejection")
	}
}
