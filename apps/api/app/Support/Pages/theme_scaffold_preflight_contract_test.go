package pages

import (
	"os"
	"path/filepath"
	"testing"
)

// 页面预检的**冲突控制**：这一层不检查页面覆盖率（覆盖率契约由
// cmd/sforum/theme_scaffold_activation_test.go 的
// TestActiveThemeDoesNotRequireFullPageCoverage 固定），但它确实按路径签名
// 拒绝「add 路径已被别的扩展拥有」。本测试固定的是后者，避免下一次有人把
// 「激活被拒」再次误判成覆盖率问题。
func TestThemePreflightRejectsSamePathAddOwnedByAnotherExtension(t *testing.T) {
	registry := NewRegistry(NewMemoryStore())
	bridge := NewExtensionBridge(registry)

	// 先让另一个扩展拥有 /probe-conflict。
	if err := registry.RegisterContributions("demo.other-theme", []PageContribution{{
		ID: "demo.other-theme.page", Action: ActionAdd, Path: "/probe-conflict",
		Template: "templates/page.html", Access: AccessPublic,
		Contract: "sforum.page.demo_other_conflict@1",
	}}); err != nil {
		t.Fatalf("register conflicting contribution: %v", err)
	}

	// 脚手架形状的主题（单页 add）声明同一路径时必须被拒。
	conflicting := writeScaffoldShapedThemeWithPath(t, "/probe-conflict")
	_, err := bridge.PreflightThemePackage(conflicting, "sforum.default-theme")
	if err == nil {
		t.Fatal("a same-path add contribution owned by another extension must be rejected")
	}

	// 对照：换个不被占用的路径，同一形状的包必须通过。
	free := writeScaffoldShapedThemeWithPath(t, "/probe-free")
	if _, err := bridge.PreflightThemePackage(free, "sforum.default-theme"); err != nil {
		t.Fatalf("an unowned add path must preflight: %v", err)
	}
}

// writeScaffoldShapedThemeWithPath 生成 `make:theme` 脚手架形状的包：一个页面、
// 一个模板、一个皮肤 CSS、无外壳声明、无 components[]。
func writeScaffoldShapedThemeWithPath(t *testing.T, path string) ThemeExtension {
	t.Helper()
	root := t.TempDir()
	writePagesFixtureFile(t, root, "theme.json", `{
  "pages": [
    {
      "id": "acme.scaffold-theme.page",
      "action": "add",
      "path": "`+path+`",
      "template": "templates/page.html",
      "access": "public",
      "contract": "sforum.page.acme_scaffold_theme@1"
    }
  ],
  "skin": { "css": ["assets/theme.css"] }
}`)
	writePagesFixtureFile(t, root, "templates/page.html", `<main class="sf-scaffold-theme"><h1>Scaffold theme page</h1></main>`)
	writePagesFixtureFile(t, root, "assets/theme.css", ":root { --sf-scaffold-theme: 1; }")
	return ThemeExtension{
		ID: "acme.scaffold-theme", Version: "0.1.0",
		PackagePath: root, PackageDigest: "scaffold-digest",
	}
}

func writePagesFixtureFile(t *testing.T, root, relative, body string) {
	t.Helper()
	full := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
