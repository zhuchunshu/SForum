package extensions

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zhuchunshu/sforum/apps/api/app/Support/Pages"
)

func TestPageRegistryAdapterThemeRuntimeSwitchAndRollback(t *testing.T) {
	registry := pages.NewRegistry(pages.NewMemoryStore())
	runtimeRegistry := pages.NewThemeRuntimeRegistry()
	adapter := NewPageRegistryAdapter(registry).WithThemeRuntime(runtimeRegistry, "SForum", []string{"zh-CN"})
	source := themeRuntimeExtensionFixture(t, "source.theme", strings.Repeat("1", 64), "/shared", false)
	target := themeRuntimeExtensionFixture(t, "target.theme", strings.Repeat("2", 64), "/shared", false)

	if err := adapter.RegisterThemePackage(context.Background(), source); err != nil {
		t.Fatal(err)
	}
	assertActiveThemeRuntime(t, runtimeRegistry, source.ID)
	if err := adapter.RegisterThemePackageReplacing(context.Background(), target, source.ID); err != nil {
		t.Fatal(err)
	}
	assertActiveThemeRuntime(t, runtimeRegistry, target.ID)
	if err := adapter.RegisterThemePackageReplacing(context.Background(), source, target.ID); err != nil {
		t.Fatal(err)
	}
	assertActiveThemeRuntime(t, runtimeRegistry, source.ID)
}

func TestPageRegistryAdapterFailedSwitchKeepsSourceRuntime(t *testing.T) {
	registry := pages.NewRegistry(pages.NewMemoryStore())
	if err := registry.RegisterContributions("plugin.owner", []pages.PageContribution{{
		ID: "plugin.page", Action: pages.ActionAdd, Path: "/occupied",
		ExtensionID: "plugin.owner", Version: "1.0.0", PackageDigest: "plugin-digest",
	}}); err != nil {
		t.Fatal(err)
	}
	runtimeRegistry := pages.NewThemeRuntimeRegistry()
	adapter := NewPageRegistryAdapter(registry).WithThemeRuntime(runtimeRegistry, "SForum", []string{"zh-CN"})
	source := themeRuntimeExtensionFixture(t, "source.theme", strings.Repeat("3", 64), "/source", false)
	conflicting := themeRuntimeExtensionFixture(t, "target.theme", strings.Repeat("4", 64), "/occupied", true)
	if err := adapter.RegisterThemePackage(context.Background(), source); err != nil {
		t.Fatal(err)
	}
	if err := adapter.RegisterThemePackageReplacing(context.Background(), conflicting, source.ID); err == nil {
		t.Fatal("expected add-route conflict")
	}
	assertActiveThemeRuntime(t, runtimeRegistry, source.ID)
	if _, ok := runtimeRegistry.Resolve(pages.RuntimeArtifact{
		ExtensionID: conflicting.ID, ExtensionVersion: conflicting.Version, PackageDigest: conflicting.PackageDigest,
	}, "forum.home", conflicting.ID+".home"); ok {
		t.Fatal("failed target snapshot remained staged")
	}
}

func themeRuntimeExtensionFixture(t *testing.T, id, digest, addPath string, includeAdd bool) Extension {
	t.Helper()
	root := t.TempDir()
	pagesJSON := `[{"id":"` + id + `.home","action":"replace","target":"forum.home","template":"templates/home.html","contract":"sforum.page.home@1"}`
	if includeAdd {
		pagesJSON += `,{"id":"` + id + `.add","action":"add","path":"` + addPath + `","template":"templates/add.html","contract":"` + id + `.page@1"}`
	}
	pagesJSON += `]`
	writeThemeRuntimeExtensionFile(t, root, "theme.json", `{"pages":`+pagesJSON+`}`)
	writeThemeRuntimeExtensionFile(t, root, "templates/home.html", `<main>home</main><sf-home-page></sf-home-page>`)
	writeThemeRuntimeExtensionFile(t, root, "templates/add.html", `<main>add</main>`)
	return Extension{ID: id, Type: TypeTheme, Status: StatusEnabled, Version: "1.0.0", PackageDigest: digest, PackagePath: root}
}

func writeThemeRuntimeExtensionFile(t *testing.T, root, name, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertActiveThemeRuntime(t *testing.T, registry *pages.ThemeRuntimeRegistry, extensionID string) {
	t.Helper()
	snapshot, _, ok := registry.Active()
	if !ok || snapshot.Artifact().ExtensionID != extensionID {
		t.Fatalf("active runtime=%#v", snapshot)
	}
}
