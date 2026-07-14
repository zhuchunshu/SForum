package pages

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
	firstSkin, ok := registry.ActiveSkin()
	if !ok || firstSkin.ExtensionID != source.Artifact().ExtensionID || firstSkin.NodeRevision != 1 {
		t.Fatalf("first skin=%#v ok=%v", firstSkin, ok)
	}
	if _, staged, err := registry.Stage(target); err != nil || !staged {
		t.Fatalf("stage target: staged=%v err=%v", staged, err)
	}
	if _, ok := registry.Resolve(source.Artifact(), "forum.home", "source.home"); !ok {
		t.Fatal("source disappeared while target was staged")
	}
	resolvedSource, _ := registry.Resolve(source.Artifact(), "forum.home", "source.home")
	output, err := resolvedSource.Render(context.Background(), CorePageViewModelRequest{
		PageID: "forum.home", Locale: "zh-CN", Path: "/",
	}, "source.home")
	if err != nil || output.NodeRevision != 2 {
		t.Fatalf("staged cache revision output=%#v err=%v", output, err)
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

func TestThemeRuntimeFourLevelFallbackIsExactAndZeroIO(t *testing.T) {
	const pluginTemplateID = "plugin.demo.template.plugin.demo.home"
	failing := `<main>{{asset "missing"}}</main><sf-home-page></sf-home-page>`
	defaultTheme, defaultRoot := buildFallbackRuntime(t, "sforum.default-theme", "1", RuntimeTemplateTheme,
		`<main>default output</main><sf-home-page></sf-home-page>`, "", "")
	activeTheme, activeRoot := buildFallbackRuntime(t, "active.theme", "2", RuntimeTemplateTheme,
		failing, pluginTemplateID, failing)
	plugin, pluginRoot := buildFallbackRuntime(t, "plugin.demo", "3", RuntimeTemplatePlugin, failing, "", "")

	registry := NewThemeRuntimeRegistry()
	if _, _, err := registry.Stage(defaultTheme); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.SetDefaultExact(defaultTheme.Artifact()); err != nil {
		t.Fatal(err)
	}
	if _, _, err := registry.Stage(activeTheme); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.ActivateExact(activeTheme.Artifact()); err != nil {
		t.Fatal(err)
	}
	if _, _, err := registry.Stage(plugin); err != nil {
		t.Fatal(err)
	}

	// A published fallback plan owns no path or filesystem handle.
	for _, root := range []string{defaultRoot, activeRoot, pluginRoot} {
		if err := os.RemoveAll(root); err != nil {
			t.Fatal(err)
		}
	}
	renderer, ok := registry.Resolve(plugin.Artifact(), "forum.home", "plugin.demo.home")
	if !ok {
		t.Fatal("plugin render plan did not resolve")
	}
	output, err := renderer.Render(context.Background(), CorePageViewModelRequest{
		PageID: "forum.home", Locale: "zh-CN", Path: "/", SEO: themecompiler.PageSEOView{Title: "Home"},
	}, "plugin.demo.home")
	if err != nil {
		t.Fatal(err)
	}
	if output.Source != ThemeRenderSourceDefaultTheme || !output.Fallback ||
		!strings.Contains(renderer.LegacyHTML(output), "default output") {
		t.Fatalf("unexpected fallback output: %#v", output)
	}
	wantSources := []string{
		ThemeRenderSourceActiveOverride, ThemeRenderSourcePlugin,
		ThemeRenderSourceActiveTheme, ThemeRenderSourceDefaultTheme,
	}
	if len(output.Attempts) != len(wantSources) {
		t.Fatalf("attempts=%#v", output.Attempts)
	}
	for index, source := range wantSources {
		if output.Attempts[index].Source != source {
			t.Fatalf("attempt %d source=%q want=%q", index, output.Attempts[index].Source, source)
		}
		if index < len(wantSources)-1 && output.Attempts[index].FailureCode != "render_failed" {
			t.Fatalf("attempt %d failure=%q", index, output.Attempts[index].FailureCode)
		}
	}
}

func TestThemeRuntimeFallbackUsesOverrideFirstAndEmergencyLast(t *testing.T) {
	const pluginTemplateID = "plugin.demo.template.plugin.demo.home"
	failing := `<main>{{asset "missing"}}</main><sf-home-page></sf-home-page>`
	t.Run("override", func(t *testing.T) {
		active, _ := buildFallbackRuntime(t, "active.theme", "4", RuntimeTemplateTheme,
			`<main>active</main><sf-home-page></sf-home-page>`, pluginTemplateID,
			`<main>override</main><sf-home-page></sf-home-page>`)
		plugin, _ := buildFallbackRuntime(t, "plugin.demo", "5", RuntimeTemplatePlugin,
			`<main>plugin</main><sf-home-page></sf-home-page>`, "", "")
		registry := NewThemeRuntimeRegistry()
		_, _, _ = registry.Stage(active)
		_, _ = registry.ActivateExact(active.Artifact())
		_, _, _ = registry.Stage(plugin)
		renderer, _ := registry.Resolve(plugin.Artifact(), "forum.home", "plugin.demo.home")
		output, err := renderer.Render(context.Background(), CorePageViewModelRequest{
			PageID: "forum.home", Locale: "en-US", Path: "/", SEO: themecompiler.PageSEOView{Title: "Home"},
		}, "plugin.demo.home")
		if err != nil || output.Source != ThemeRenderSourceActiveOverride || output.Fallback ||
			!strings.Contains(renderer.LegacyHTML(output), "override") {
			t.Fatalf("output=%#v err=%v", output, err)
		}
	})

	t.Run("emergency", func(t *testing.T) {
		active, _ := buildFallbackRuntime(t, "active.theme", "6", RuntimeTemplateTheme, failing, "", "")
		plugin, _ := buildFallbackRuntime(t, "plugin.demo", "7", RuntimeTemplatePlugin, failing, "", "")
		registry := NewThemeRuntimeRegistry()
		_, _, _ = registry.Stage(active)
		_, _ = registry.ActivateExact(active.Artifact())
		_, _, _ = registry.Stage(plugin)
		renderer, _ := registry.Resolve(plugin.Artifact(), "forum.home", "plugin.demo.home")
		output, err := renderer.Render(context.Background(), CorePageViewModelRequest{
			PageID: "forum.home", Locale: "en-US", Path: "/",
			SEO: themecompiler.PageSEOView{Title: `<unsafe>`},
		}, "plugin.demo.home")
		if err != nil || output.Source != ThemeRenderSourceEmergency || !output.Fallback ||
			!strings.Contains(renderer.LegacyHTML(output), "&lt;unsafe&gt;") {
			t.Fatalf("output=%#v html=%q err=%v", output, renderer.LegacyHTML(output), err)
		}
	})
}

func TestThemeRuntimeV3TemplateDeclarationIsDigestAndContractBound(t *testing.T) {
	root := t.TempDir()
	body := `<main>plugin</main><sf-home-page></sf-home-page>`
	writeThemeRuntimeTestFile(t, root, "templates/home.html", body)
	digest := sha256.Sum256([]byte(body))
	artifact := RuntimeArtifact{
		ExtensionID: "plugin.v3", ExtensionVersion: "1.0.0", PackageDigest: strings.Repeat("9", 64),
	}
	contribution := PageContribution{
		ID: "plugin.v3.home", Action: ActionReplace, Target: "forum.home", Template: "templates/home.html",
		Contract: "sforum.page.home@1", ExtensionID: artifact.ExtensionID,
		Version: artifact.ExtensionVersion, PackageDigest: artifact.PackageDigest,
	}
	declaration := RuntimeTemplateDeclaration{
		ID: "plugin.v3.template.home", ContractVersion: "plugin.v3.template.home@1", Action: "add",
		Path: "templates/home.html", Digest: hex.EncodeToString(digest[:]), ViewModelSchema: contribution.Contract,
	}
	input := ThemeRuntimeBuildInput{
		Artifact: artifact, PackageRoot: root, Contributions: []PageContribution{contribution},
		Templates: []RuntimeTemplateDeclaration{declaration}, PackageKind: RuntimeTemplatePlugin,
		RequireDeclaredTemplates: true,
	}
	if _, err := BuildThemeRuntimeSnapshot(input); err != nil {
		t.Fatal(err)
	}
	input.Templates[0].Digest = strings.Repeat("a", 64)
	if _, err := BuildThemeRuntimeSnapshot(input); !errors.Is(err, ErrThemeRuntimeConflict) {
		t.Fatalf("digest mismatch error=%v", err)
	}
	input.Templates[0] = declaration
	input.Templates[0].ViewModelSchema = "sforum.page.topic_show@1"
	if _, err := BuildThemeRuntimeSnapshot(input); !errors.Is(err, ErrThemeRuntimeConflict) {
		t.Fatalf("contract mismatch error=%v", err)
	}
}

func buildFallbackRuntime(
	t *testing.T,
	extensionID, digestByte string,
	kind RuntimeTemplatePackageKind,
	mainBody, overrideTarget, overrideBody string,
) (*ThemeRuntimeSnapshot, string) {
	t.Helper()
	root := t.TempDir()
	writeThemeRuntimeTestFile(t, root, "templates/home.html", mainBody)
	declarations := []RuntimeTemplateDeclaration(nil)
	if overrideTarget != "" {
		name := "templates/plugins/plugin.demo/home.html"
		writeThemeRuntimeTestFile(t, root, name, overrideBody)
		digest := sha256.Sum256([]byte(overrideBody))
		declarations = append(declarations, RuntimeTemplateDeclaration{
			ID: extensionID + ".template.override", ContractVersion: extensionID + ".template.override@1",
			Action: "replace", TargetID: overrideTarget, Path: name, Digest: hex.EncodeToString(digest[:]),
			ViewModelSchema: "sforum.page.home@1",
		})
	}
	artifact := RuntimeArtifact{
		ExtensionID: extensionID, ExtensionVersion: "1.0.0", PackageDigest: strings.Repeat(digestByte, 64),
	}
	contribution := PageContribution{
		ID: extensionID + ".home", Action: ActionReplace, Target: "forum.home", Template: "templates/home.html",
		Contract: "sforum.page.home@1", ExtensionID: extensionID, Version: "1.0.0", PackageDigest: artifact.PackageDigest,
	}
	snapshot, err := BuildThemeRuntimeSnapshot(ThemeRuntimeBuildInput{
		Artifact: artifact, PackageRoot: root, Contributions: []PageContribution{contribution},
		Templates: declarations, PackageKind: kind, SiteName: "SForum",
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot, root
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
