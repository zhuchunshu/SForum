package pages

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	themecompiler "github.com/zhuchunshu/sforum/apps/api/app/Support/ThemeCompiler"
)

func TestCorePageViewModelFactoryBuildsEveryCatalogContract(t *testing.T) {
	digest := strings.Repeat("a", 64)
	for _, page := range Catalog() {
		value, err := BuildCorePageViewModel(CorePageViewModelRequest{
			PageID: page.ID, Locale: "zh-CN", Path: "/test",
			SEO: themecompiler.PageSEOView{Title: page.ID},
		})
		if err != nil {
			t.Fatalf("build %s: %v", page.ID, err)
		}
		if _, err := themecompiler.CorePageViewModelRegistry().Bind(page.ID, page.ContractVersion, digest, value); err != nil {
			t.Fatalf("bind %s: %v", page.ID, err)
		}
	}
}

func TestThemeRuntimeSnapshotCompilesOnceAndRendersExactProvider(t *testing.T) {
	root := defaultThemeFixtureRoot(t)
	artifact := RuntimeArtifact{
		ExtensionID: "sforum-default", ExtensionVersion: "1.0.0", PackageDigest: strings.Repeat("b", 64),
	}
	contribution := PageContribution{
		ID: "sforum.default-theme.home", Action: ActionReplace, Target: "forum.home",
		Template: "templates/home.html", Contract: "sforum.page.home@1",
		ExtensionID: artifact.ExtensionID, Version: artifact.ExtensionVersion, PackageDigest: artifact.PackageDigest,
	}
	snapshot, err := BuildThemeRuntimeSnapshot(ThemeRuntimeBuildInput{
		Artifact: artifact, PackageRoot: root, Contributions: []PageContribution{contribution},
		SiteName: "SForum", Locales: []string{"en-US", "zh-CN", "zh-CN"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.RuntimeKey() == "" || !snapshot.Covers("forum.home", contribution.ID) {
		t.Fatalf("snapshot key=%q coverage=%v", snapshot.RuntimeKey(), snapshot.Covers("forum.home", contribution.ID))
	}
	output, err := snapshot.Render(context.Background(), CorePageViewModelRequest{
		PageID: "forum.home", Locale: "zh-CN", Path: "/",
		SEO: themecompiler.PageSEOView{Title: "Home"},
	}, contribution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(output.HTMLSegments) == 0 || len(output.Islands) != 1 ||
		output.Islands[0].ComponentID != "forum.component.home_page" || !strings.Contains(snapshot.LegacyHTML(output), "<sf-home-page>") {
		t.Fatalf("output=%#v legacy=%q", output, snapshot.LegacyHTML(output))
	}

	registry := NewThemeRuntimeRegistry()
	if revision := registry.Publish(snapshot); revision != 1 {
		t.Fatalf("revision=%d", revision)
	}
	if _, ok := registry.Resolve(artifact, "forum.home", contribution.ID); !ok {
		t.Fatal("exact snapshot did not resolve")
	}
	stale := artifact
	stale.PackageDigest = strings.Repeat("c", 64)
	if _, ok := registry.Resolve(stale, "forum.home", contribution.ID); ok {
		t.Fatal("stale artifact resolved")
	}
	if _, err := registry.RemoveExact(stale); err != ErrThemeRuntimeConflict {
		t.Fatalf("stale removal error=%v", err)
	}
	if _, err := registry.RemoveExact(artifact); err != nil {
		t.Fatal(err)
	}
}

func TestThemeRuntimeSnapshotIgnoresUncoveredLegacyAddTemplates(t *testing.T) {
	root := t.TempDir()
	writeThemeRuntimeTestFile(t, root, "templates/core.html", `<main>core</main><sf-home-page></sf-home-page>`)
	writeThemeRuntimeTestFile(t, root, "templates/plugin.html", `<main>legacy plugin page</main>`)
	writeThemeRuntimeTestFile(t, root, "theme.json", `{"pages":[]}`)
	artifact := RuntimeArtifact{ExtensionID: "mixed.theme", ExtensionVersion: "1.0.0", PackageDigest: strings.Repeat("d", 64)}
	snapshot, err := BuildThemeRuntimeSnapshot(ThemeRuntimeBuildInput{
		Artifact: artifact, PackageRoot: root, SiteName: "SForum",
		Contributions: []PageContribution{
			{ID: "mixed.home", Action: ActionReplace, Target: "forum.home", Template: "templates/core.html", Contract: "sforum.page.home@1", ExtensionID: artifact.ExtensionID, Version: artifact.ExtensionVersion, PackageDigest: artifact.PackageDigest},
			{ID: "mixed.add", Action: ActionAdd, Path: "/mixed", Template: "templates/plugin.html", Contract: "mixed.page@1", ExtensionID: artifact.ExtensionID, Version: artifact.ExtensionVersion, PackageDigest: artifact.PackageDigest},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Covers("mixed.add", "mixed.add") {
		t.Fatal("untyped plugin business page entered core Theme runtime")
	}
}

func TestThemeRuntimeRegistryStagesSwitchWithoutDroppingSource(t *testing.T) {
	source := buildThemeRuntimeFixture(t, "source.theme", strings.Repeat("e", 64), "source.home")
	target := buildThemeRuntimeFixture(t, "target.theme", strings.Repeat("f", 64), "target.home")
	registry := NewThemeRuntimeRegistry()
	registry.Publish(source)
	if _, staged, err := registry.Stage(target); err != nil || !staged {
		t.Fatalf("stage target: staged=%v err=%v", staged, err)
	}
	if _, ok := registry.Resolve(source.Artifact(), "forum.home", "source.home"); !ok {
		t.Fatal("source disappeared while target was staged")
	}
	if _, ok := registry.Resolve(target.Artifact(), "forum.home", "target.home"); !ok {
		t.Fatal("staged target is not exact-resolvable")
	}
	if active, _, ok := registry.Active(); !ok || active.Artifact() != source.Artifact() {
		t.Fatalf("staging changed active snapshot: %#v", active)
	}
	if _, err := registry.ActivateExact(target.Artifact()); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.RemoveExact(source.Artifact()); err != nil {
		t.Fatal(err)
	}
	if active, _, ok := registry.Active(); !ok || active.Artifact() != target.Artifact() {
		t.Fatalf("target not active: %#v", active)
	}
}

func buildThemeRuntimeFixture(t *testing.T, extensionID, digest, contributionID string) *ThemeRuntimeSnapshot {
	t.Helper()
	root := t.TempDir()
	writeThemeRuntimeTestFile(t, root, "templates/home.html", `<main>home</main><sf-home-page></sf-home-page>`)
	writeThemeRuntimeTestFile(t, root, "theme.json", `{"pages":[]}`)
	artifact := RuntimeArtifact{ExtensionID: extensionID, ExtensionVersion: "1.0.0", PackageDigest: digest}
	snapshot, err := BuildThemeRuntimeSnapshot(ThemeRuntimeBuildInput{
		Artifact: artifact, PackageRoot: root, SiteName: "SForum",
		Contributions: []PageContribution{{
			ID: contributionID, Action: ActionReplace, Target: "forum.home", Template: "templates/home.html",
			Contract: "sforum.page.home@1", ExtensionID: extensionID, Version: "1.0.0", PackageDigest: digest,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func defaultThemeFixtureRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../../extensions/builtin/themes/sforum-default"))
}

func writeThemeRuntimeTestFile(t *testing.T, root, name, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
