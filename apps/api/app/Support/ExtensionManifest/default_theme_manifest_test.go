package extensionmanifest

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBuiltinDefaultThemeManifestValidatesWithSettingsPage(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// apps/api/app/Support/ExtensionManifest -> repo root
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../../"))
	packageRoot := filepath.Join(root, "extensions/builtin/themes/sforum-default")
	normalized, err := LoadPackage(packageRoot)
	if err != nil {
		t.Fatalf("load default theme package: %v", err)
	}
	if normalized.Type != TypeTheme {
		t.Fatalf("expected theme type, got %s", normalized.Type)
	}
	// Runtime themes no longer require frontend.layer (L0/L1 via theme.json).
	if normalized.Frontend.Layer != "" {
		t.Fatalf("expected empty layer for runtime default theme, got %q", normalized.Frontend.Layer)
	}
	// theme.json must exist at package root for L0/L1.
	if _, err := os.Stat(filepath.Join(packageRoot, "theme.json")); err != nil {
		t.Fatalf("expected theme.json for runtime package: %v", err)
	}
	if len(normalized.Settings) < 15 {
		t.Fatalf("expected expanded theme settings, got %d", len(normalized.Settings))
	}
	if normalized.Frontend.Admin == nil || normalized.Frontend.Admin.Components["theme-settings-page"] == "" {
		t.Fatal("expected frontend.admin theme-settings-page component")
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
	// 主题仍不得声明后端/提供商等插件能力。
	if normalized.Backend.Entry != "" || len(normalized.Providers) != 0 || len(normalized.Hooks) != 0 {
		t.Fatal("theme must not declare plugin-only surfaces")
	}
}
