package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 主题声明外壳时，激活前预检必须确认该模板可加载——外壳承担插件公开页的 chrome，
// 缺失时若拖到渲染期才发现，插件页会静默失去导航。
func TestThemeShellDeclarationIsPreflighted(t *testing.T) {
	write := func(dir, shellPath string, createFile bool) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(dir, "templates"), 0o755); err != nil {
			t.Fatal(err)
		}
		body := `{"pages":[],"shell":"` + shellPath + `"}`
		if shellPath == "" {
			body = `{"pages":[]}`
		}
		if err := os.WriteFile(filepath.Join(dir, "theme.json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if createFile {
			if err := os.WriteFile(filepath.Join(dir, shellPath), []byte("<div><sf-navbar></sf-navbar><sf-plugin-page-body></sf-plugin-page-body><sf-footer></sf-footer></div>"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	preflight := func(dir string) error {
		bridge := &ExtensionBridge{}
		_, err := bridge.PreflightThemePackage(ThemeExtension{ID: "demo.theme", Version: "1.0.0", PackagePath: dir}, "")
		return err
	}

	okDir := t.TempDir()
	write(okDir, "templates/shell.html", true)
	if err := preflight(okDir); err != nil {
		t.Fatalf("declared shell should preflight: %v", err)
	}

	missingDir := t.TempDir()
	write(missingDir, "templates/missing.html", false)
	err := preflight(missingDir)
	if err == nil {
		t.Fatal("declared shell pointing at a missing template must be rejected")
	}
	if !strings.Contains(err.Error(), "shell template") {
		t.Fatalf("error should name the shell template, got: %v", err)
	}

	// 未声明外壳的主题照常通过：外壳是可选能力。
	noShellDir := t.TempDir()
	write(noShellDir, "", false)
	if err := preflight(noShellDir); err != nil {
		t.Fatalf("theme without a shell declaration should still preflight: %v", err)
	}
}
