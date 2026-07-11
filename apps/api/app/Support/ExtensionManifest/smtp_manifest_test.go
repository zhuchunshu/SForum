package extensionmanifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBuiltinSMTPManifestValidatesWithLocalizedSettingsAndSettingsPageSlot(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// apps/api/app/Support/ExtensionManifest -> repo root
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../../"))
	manifestPath := filepath.Join(root, "extensions/builtin/plugins/sforum-smtp/sforum.extension.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read smtp manifest: %v", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal smtp manifest: %v", err)
	}
	if err := Validate(manifest); err != nil {
		t.Fatalf("smtp manifest invalid: %v", err)
	}
	normalized := Normalize(manifest)
	if len(normalized.Settings) < 5 {
		t.Fatalf("expected smtp settings, got %d", len(normalized.Settings))
	}
	host := normalized.Settings[0]
	if host.Key != "host" {
		t.Fatalf("expected host first, got %s", host.Key)
	}
	if ResolveSettingPresentation(host, "zh-CN").Label == "" {
		t.Fatal("zh label empty")
	}
	if ResolveSettingPresentation(host, "en-US").Label == "" {
		t.Fatal("en label empty")
	}
	if normalized.Frontend.Admin == nil || normalized.Frontend.Admin.Components["smtp-settings-page"] == "" {
		t.Fatal("expected frontend.admin smtp-settings-page component")
	}
	found := false
	for _, contribution := range normalized.Contributions {
		if contribution.Point == "admin.extension.settings.page" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected admin.extension.settings.page contribution")
	}
}
