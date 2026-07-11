package extensionmanifest

import (
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
	packageRoot := filepath.Join(root, "extensions/builtin/plugins/sforum-smtp")
	// LoadPackage 支持单文件与 includes 多文件；SMTP 迁移后仍走此路径。
	normalized, err := LoadPackage(packageRoot)
	if err != nil {
		t.Fatalf("load smtp package: %v", err)
	}
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
