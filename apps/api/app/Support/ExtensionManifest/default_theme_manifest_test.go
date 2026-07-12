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
	// 普通主题使用宿主 schema-driven 设置页，不再声明 frontend.admin / ThemeSettingsPage。
	if normalized.Frontend.Admin != nil {
		t.Fatal("runtime theme must not declare frontend.admin (host schema settings only)")
	}
	for _, contribution := range normalized.Contributions {
		if contribution.Point == "admin.extension.settings.page" {
			t.Fatal("runtime theme must not declare admin.extension.settings.page Vue contribution")
		}
	}
	// 主题仍不得声明后端/提供商等插件能力。
	if normalized.Backend.Entry != "" || len(normalized.Providers) != 0 || len(normalized.Hooks) != 0 {
		t.Fatal("theme must not declare plugin-only surfaces")
	}
}
