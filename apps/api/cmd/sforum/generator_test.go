package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

func TestGeneratePluginScaffoldNonInteractive(t *testing.T) {
	target := filepath.Join(t.TempDir(), "plugin")
	created, err := GenerateExtensionScaffold(makeOptions{
		Kind:          "plugin",
		ID:            "acme.demo",
		Name:          "Acme Demo",
		Description:   "Demo plugin.",
		URL:           "https://example.com/acme-demo",
		AuthorName:    "Acme",
		AuthorURL:     "https://example.com",
		AuthorEmail:   "dev@example.com",
		Out:           target,
		NoInteraction: true,
		Backend:       true,
	})
	if err != nil {
		t.Fatalf("GenerateExtensionScaffold returned error: %v", err)
	}
	if created != target {
		t.Fatalf("expected target %q, got %q", target, created)
	}
	manifest := readGeneratedManifest(t, target)
	if manifest.Type != extensionmanifest.TypePlugin || manifest.Backend.Entry != "backend/plugin" {
		t.Fatalf("unexpected plugin manifest: %#v", manifest)
	}
	if extensionmanifest.EffectiveManifestVersion(manifest) != extensionmanifest.ManifestVersionV3 || manifest.Backend.Digest == "" || len(manifest.PackageFiles) != 1 {
		t.Fatalf("plugin scaffold must be exact-digest Manifest V3: %#v", manifest)
	}
	if len(manifest.PermissionDefinitions) != 1 || manifest.PermissionDefinitions[0].AssignmentPolicy != "host" {
		t.Fatalf("plugin permissions must remain Host-assigned: %#v", manifest.PermissionDefinitions)
	}
	if manifest.Admin.Entry != "/settings" || len(manifest.Admin.Pages) != 1 || manifest.Admin.Pages[0].Menu {
		t.Fatalf("expected v2 plugin admin settings page without sidebar menu: %#v", manifest.Admin)
	}
	if len(manifest.AdminPages) != 0 {
		t.Fatalf("new scaffolds should not use legacy adminPages: %#v", manifest.AdminPages)
	}
	if len(manifest.Contributions) != 0 {
		t.Fatalf("new plugin scaffolds should not enable demo contributions by default: %#v", manifest.Contributions)
	}
	if manifest.SettingsDocument.SchemaVersion != 1 || manifest.SettingsDocument.UI.Mode != extensionmanifest.SettingsUIModeSchema || manifest.SettingsDocument.UI.Layout != extensionmanifest.SettingsLayoutTabs {
		t.Fatalf("new scaffolds must default to versioned Schema UI: %#v", manifest.SettingsDocument)
	}
	readme, err := os.ReadFile(filepath.Join(target, "README.md"))
	if err != nil {
		t.Fatalf("read generated README: %v", err)
	}
	if !strings.Contains(string(readme), "forum.topic.actions") || !strings.Contains(string(readme), "\"contributions\"") {
		t.Fatalf("expected README to document contribution example, got:\n%s", readme)
	}
	if err := extensionmanifest.Validate(manifest); err != nil {
		t.Fatalf("generated plugin manifest should validate: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "backend", "plugin")); err != nil {
		t.Fatalf("expected backend stub: %v", err)
	}
}

func TestGenerateComplexPluginScaffold(t *testing.T) {
	target := filepath.Join(t.TempDir(), "complex-plugin")
	created, err := GenerateExtensionScaffold(makeOptions{
		Kind:          "plugin",
		ID:            "acme.complex",
		Name:          "Acme Complex",
		Description:   "Complex multi-file plugin.",
		URL:           "https://example.com/acme-complex",
		AuthorName:    "Acme",
		Out:           target,
		NoInteraction: true,
		Complex:       true,
	})
	if err != nil {
		t.Fatalf("GenerateExtensionScaffold complex: %v", err)
	}
	if created != target {
		t.Fatalf("target mismatch: %s", created)
	}
	// 必须通过 LoadPackage（解析 includes + settings 分片）。
	manifest, err := extensionmanifest.LoadPackage(target)
	if err != nil {
		t.Fatalf("LoadPackage generated complex package: %v", err)
	}
	if manifest.ID != "acme.complex" {
		t.Fatalf("id = %s", manifest.ID)
	}
	if extensionmanifest.EffectiveManifestVersion(manifest) != extensionmanifest.ManifestVersionV3 {
		t.Fatalf("complex scaffold contract = %s", extensionmanifest.ManifestContract(manifest))
	}
	if len(manifest.Settings) != 2 {
		t.Fatalf("expected 2 settings from shards, got %d", len(manifest.Settings))
	}
	if _, ok := manifest.Langs["zh-CN"]; !ok {
		t.Fatalf("expected zh-CN lang file, got %#v", manifest.Langs)
	}
	if manifest.Admin.Entry != "/settings" {
		t.Fatalf("admin: %#v", manifest.Admin)
	}
	if !manifest.SettingsDocument.Explicit || manifest.SettingsDocument.UI.Layout != extensionmanifest.SettingsLayoutTabs {
		t.Fatalf("complex scaffold must use one versioned Settings Document: %#v", manifest.SettingsDocument)
	}
	rootBody, err := os.ReadFile(filepath.Join(target, extensionmanifest.ManifestFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rootBody), `"includes"`) {
		t.Fatalf("root should declare includes: %s", rootBody)
	}
	// includes 值会包含 "settings" 路径；禁止的是根级 settings 数组字段。
	if strings.Contains(string(rootBody), `"settings": [`) {
		t.Fatalf("root should not inline settings array when complex: %s", rootBody)
	}
}

func TestGeneratePrebuiltProviderSettingsScaffold(t *testing.T) {
	target := filepath.Join(t.TempDir(), "prebuilt-plugin")
	_, err := GenerateExtensionScaffold(makeOptions{
		Kind: "plugin", ID: "acme.prebuilt", Name: "Acme Prebuilt", Description: "Prebuilt settings.",
		URL: "https://example.com/prebuilt", AuthorName: "Acme", Out: target, NoInteraction: true,
		Backend: true, PrebuiltSettings: true, ProviderSlot: "mail.provider",
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest := readGeneratedManifest(t, target)
	component := manifest.SettingsDocument.UI.Component
	if component == nil || manifest.SettingsDocument.UI.Mode != extensionmanifest.SettingsUIModeComponent || component.Entry != "frontend/admin/dist/settings.mjs" {
		t.Fatalf("missing prebuilt component contract: %#v", manifest.SettingsDocument)
	}
	if len(manifest.SettingsDocument.Actions) != 1 || manifest.SettingsDocument.Actions[0].Kind != extensionmanifest.SettingsActionProviderProbe || len(manifest.Providers) != 1 {
		t.Fatalf("missing provider probe scaffold: actions=%#v providers=%#v", manifest.SettingsDocument.Actions, manifest.Providers)
	}
	if manifest.Providers[0].ID == "" || manifest.Providers[0].ContractVersion == "" || manifest.Providers[0].Handler == "" || len(manifest.PackageFiles) != 3 {
		t.Fatalf("prebuilt V3 provider declarations are incomplete: provider=%#v files=%#v", manifest.Providers[0], manifest.PackageFiles)
	}
	for _, relative := range []string{"frontend/admin/dist/settings.mjs", "frontend/admin/dist/settings.css"} {
		if _, err := os.Stat(filepath.Join(target, relative)); err != nil {
			t.Fatalf("missing prebuilt file %s: %v", relative, err)
		}
	}
	if _, err := extensionmanifest.LoadPackage(target); err != nil {
		t.Fatalf("generated prebuilt package must validate: %v", err)
	}
}

func TestGenerateThemeScaffoldNonInteractive(t *testing.T) {
	target := filepath.Join(t.TempDir(), "theme")
	_, err := GenerateExtensionScaffold(makeOptions{
		Kind:          "theme",
		ID:            "acme.theme",
		Name:          "Acme Theme",
		Description:   "Demo theme.",
		URL:           "https://example.com/acme-theme",
		AuthorName:    "Acme",
		AuthorURL:     "https://example.com",
		Out:           target,
		NoInteraction: true,
	})
	if err != nil {
		t.Fatalf("GenerateExtensionScaffold returned error: %v", err)
	}
	manifest := readGeneratedManifest(t, target)
	if manifest.Type != extensionmanifest.TypeTheme {
		t.Fatalf("unexpected theme manifest: %#v", manifest)
	}
	if extensionmanifest.EffectiveManifestVersion(manifest) != extensionmanifest.ManifestVersionV3 || len(manifest.Templates) != 1 || len(manifest.Components) != 1 || len(manifest.Assets) != 2 || len(manifest.PackageFiles) != 3 {
		t.Fatalf("theme scaffold must declare V3 presentation contracts: %#v", manifest)
	}
	if manifest.Admin.Entry != "/settings" || len(manifest.Admin.Pages) != 1 || len(manifest.Settings) == 0 {
		t.Fatalf("expected theme v2 admin page and settings declarations: %#v", manifest)
	}
	if len(manifest.AdminPages) != 0 {
		t.Fatalf("new theme scaffolds should not use legacy adminPages: %#v", manifest.AdminPages)
	}
	if err := extensionmanifest.Validate(manifest); err != nil {
		t.Fatalf("generated theme manifest should validate: %v", err)
	}
	for _, rel := range []string{"theme.json", "assets/theme.css", "templates/home.html"} {
		if _, err := os.Stat(filepath.Join(target, rel)); err != nil {
			t.Fatalf("expected runtime theme file %s: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(target, "layer", "nuxt.config.ts")); err == nil {
		t.Fatal("runtime theme scaffold must not create Nuxt layer")
	}
}

func readGeneratedManifest(t *testing.T, dir string) extensionmanifest.Manifest {
	t.Helper()
	// 统一走 LoadPackage，兼容单文件与 complex includes。
	manifest, err := extensionmanifest.LoadPackage(dir)
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}
	return manifest
}

func TestExtensionValidateCommand(t *testing.T) {
	// 使用真实 SMTP 多文件包做 CLI 冒烟。
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// apps/api/cmd/sforum -> repo root
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../.."))
	smtpRoot := filepath.Join(repoRoot, "extensions/builtin/plugins/sforum-smtp")

	cmd := newRootCommand()
	cmd.SetArgs([]string{"extension", "validate", smtpRoot})
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("extension validate: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "sforum.smtp") {
		t.Fatalf("expected id in output:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "zh-CN") {
		t.Fatalf("expected langs in output:\n%s", out.String())
	}
	// 内置 SMTP 已迁到 Manifest V3；validate 输出应报告 sforum.manifest@3。
	if !strings.Contains(out.String(), "contract: sforum.manifest@3") {
		t.Fatalf("expected manifest contract in output:\n%s", out.String())
	}
}

func TestExtensionDigestCommandRefreshesGeneratedArtifact(t *testing.T) {
	target := filepath.Join(t.TempDir(), "digest-plugin")
	_, err := GenerateExtensionScaffold(makeOptions{
		Kind: "plugin", ID: "acme.digest", Name: "Acme Digest", Description: "Digest refresh.",
		URL: "https://example.com/digest", AuthorName: "Acme", Out: target, NoInteraction: true, Backend: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	before := readGeneratedManifest(t, target).Backend.Digest
	if err := os.WriteFile(filepath.Join(target, "backend", "plugin"), []byte("replacement binary\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := extensionmanifest.LoadPackage(target); err == nil {
		t.Fatal("changed bytes must make the old manifest stale")
	}
	cmd := newRootCommand()
	cmd.SetArgs([]string{"extension", "digest", "--write", target})
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("digest refresh: %v\n%s", err, out.String())
	}
	after := readGeneratedManifest(t, target).Backend.Digest
	if after == "" || after == before || !strings.Contains(out.String(), "updated ") {
		t.Fatalf("digest not refreshed: before=%s after=%s output=%s", before, after, out.String())
	}
}
