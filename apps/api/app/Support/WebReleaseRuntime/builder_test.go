package webreleaseruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
)

func TestBuilderPrepareCopiesVerifiedInputsAndExcludesHostSecrets(t *testing.T) {
	root := t.TempDir()
	webRoot := filepath.Join(root, "web")
	if err := os.MkdirAll(filepath.Join(webRoot, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(webRoot, "package.json"), `{}`)
	writeTestFile(t, filepath.Join(webRoot, ".env"), `DATABASE_URL=secret`)
	writeTestFile(t, filepath.Join(webRoot, "app", "app.vue"), `<template />`)

	extensionRoot := filepath.Join(root, "extensions")
	themeRoot, themeDigest := publishTestSnapshot(t, extensionRoot, "demo.theme", "1.0.0", map[string]string{
		"layer/nuxt.config.ts": "export default {}",
	})
	pluginRoot, pluginDigest := publishTestSnapshot(t, extensionRoot, "demo.plugin", "1.0.0", map[string]string{
		"frontend/admin/components/Cell.vue": `<template />`,
		"frontend/admin/locales/zh-CN.json":  `{"title":"标题"}`,
		"frontend/admin/locales/en-US.json":  `{"title":"Title"}`,
	})
	defaultThemeLayer := filepath.Join(root, "default-theme", "layer")
	writeTestFile(t, filepath.Join(defaultThemeLayer, "nuxt.config.ts"), `export default {}`)
	payload, _ := json.Marshal(map[string]any{"component": "cell", "width": 120})
	plugin := extensions.WebReleaseExtension{
		ExtensionID: "demo.plugin", ExtensionVersion: "1.0.0", PackageDigest: pluginDigest,
		FrontendRoot: "frontend/admin", ComponentMap: map[string]string{"cell": "components/Cell.vue"},
		LocaleMap:         map[string]string{"zh-CN": "locales/zh-CN.json", "en-US": "locales/en-US.json"},
		TrustedComponents: []extensions.ManifestContribution{{Point: "admin.test.fixture", ID: "cell", Payload: payload}},
	}
	composition := extensions.WebComposition{
		Theme:      extensions.WebThemeSnapshot{ExtensionID: "demo.theme", Version: "1.0.0", PackageDigest: themeDigest, LayerPath: filepath.Join(themeRoot, "layer")},
		SDKVersion: AdminSDKVersion, Contract: BuildContractVersion, BunVersion: BunVersion,
	}
	composition.Extensions = []extensions.WebExtensionSnapshot{{ExtensionID: "demo.plugin"}}
	composition.Extensions[0] = extensions.WebExtensionSnapshot{
		ExtensionID: "demo.plugin", Version: "1.0.0", PackageDigest: pluginDigest,
		FrontendRoot: "frontend/admin", Dependencies: extensions.DependencySummary{},
	}
	compositionBody, _ := json.Marshal(composition)
	compositionHash := sha256Hex(compositionBody)
	detail := extensions.WebReleaseDetail{WebRelease: extensions.WebRelease{
		ID: 9, CompositionSnapshot: compositionBody, CompositionHash: compositionHash,
		ActiveThemeID: "demo.theme", ThemeVersion: "1.0.0", ThemePackageDigest: themeDigest,
		ThemeLayerPath: filepath.Join(themeRoot, "layer"), ReloadMode: extensions.WebReleaseReloadPrompt,
	}, Extensions: []extensions.WebReleaseExtension{plugin}}

	builder := NewBuilder(Config{ReleaseRoot: filepath.Join(root, "releases"), WebRoot: webRoot, ExtensionRoot: extensionRoot, DefaultThemeLayer: defaultThemeLayer})
	prepared, err := builder.Prepare(context.Background(), detail)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(prepared.Workspace, ".env")); !os.IsNotExist(err) {
		t.Fatalf("host .env must not enter build workspace: %v", err)
	}
	if _, err := os.Stat(filepath.Join(prepared.RegistryRoot, "registry.client.ts")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(prepared.DevInput, "guard-policy.json")); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(prepared.DefaultThemeLayer); err != nil || !info.IsDir() {
		t.Fatalf("isolated default theme fallback is missing: %v", err)
	}
	if digest, err := extensionpackage.DigestTree(prepared.PluginRoots["demo.plugin"]); err != nil || digest != pluginDigest {
		t.Fatalf("copied plugin identity changed: digest=%q err=%v", digest, err)
	}
	if pluginRoot == prepared.PluginRoots["demo.plugin"] {
		t.Fatal("builder must use an isolated copied plugin snapshot")
	}
}

func TestBuilderRunsTypecheckBeforeBuildWithSanitizedEnvironment(t *testing.T) {
	root := t.TempDir()
	runner := &recordingBuildRunner{}
	builder := NewBuilder(Config{
		ReleaseRoot: root, BunPath: "bun", Runner: runner,
		SourceEnvironment: []string{"PATH=/usr/bin", "DATABASE_URL=postgres://secret", "SESSION_HASH_SECRET=secret", "NUXT_PUBLIC_API_BASE_URL=/api/v1"},
	})
	prepared := PreparedRelease{
		Detail:     extensions.WebReleaseDetail{WebRelease: extensions.WebRelease{ID: 12}},
		ReleaseDir: filepath.Join(root, "releases", "12"), Workspace: filepath.Join(root, "workspace"),
		RegistryRoot: filepath.Join(root, "registry"), ThemeLayer: filepath.Join(root, "theme", "layer"),
	}
	if err := os.MkdirAll(prepared.Workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	result, err := builder.Build(context.Background(), prepared, "install")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(runner.commands, ",") != "typecheck,build" {
		t.Fatalf("unexpected build command order: %v", runner.commands)
	}
	if strings.Contains(runner.environment, "DATABASE_URL") || strings.Contains(runner.environment, "SESSION_HASH_SECRET") {
		t.Fatalf("build environment leaked secrets: %s", runner.environment)
	}
	if !strings.Contains(runner.environment, "NUXT_PUBLIC_API_BASE_URL=/api/v1") || result.ServerEntry == "" {
		t.Fatalf("missing approved public environment or result: env=%s result=%#v", runner.environment, result)
	}
}

func TestWriteJSONAtomicPublishesCompleteDocument(t *testing.T) {
	target := filepath.Join(t.TempDir(), "release.json")
	if err := writeJSONAtomic(target, map[string]any{"releaseId": 42}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"releaseId": 42`) {
		t.Fatalf("unexpected JSON: %s", body)
	}
	if _, err := os.Stat(target + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temporary file remains: %v", err)
	}
}

func TestBoundedLogRedactsCredentialsAndSecretValues(t *testing.T) {
	log := boundedLog("fetch https://user:pass@registry.example.com token=abc password=hunter2")
	if strings.Contains(log, "user:pass") || strings.Contains(log, "token=abc") || strings.Contains(log, "hunter2") {
		t.Fatalf("build log leaked credentials: %s", log)
	}
}

func TestLinkPluginHostPeersUsesHostOwnedSymlinks(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	frontend := filepath.Join(root, "plugin")
	for _, target := range []string{
		filepath.Join(workspace, "node_modules/vue"),
		filepath.Join(workspace, "node_modules/nuxt"),
		filepath.Join(workspace, "node_modules/@nuxt/ui"),
		filepath.Join(workspace, "node_modules/vue-router"),
		filepath.Join(workspace, "packages/admin-sdk"),
	} {
		if err := os.MkdirAll(target, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := linkPluginHostPeers(frontend, workspace); err != nil {
		t.Fatal(err)
	}
	for _, peer := range []string{"vue", "nuxt", "@nuxt/ui", "vue-router", "@sforum/admin-sdk"} {
		info, err := os.Lstat(filepath.Join(frontend, "node_modules", filepath.FromSlash(peer)))
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("host peer %s is not a symlink: %v", peer, err)
		}
	}
}

func TestArtifactDigestAllowsInternalSymlinkAndRejectsEscape(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "packages"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(root, "packages", "target.js"), "export const ready = true\n")
	link := filepath.Join(root, "linked.js")
	if err := os.Symlink("packages/target.js", link); err != nil {
		t.Fatal(err)
	}
	if _, err := ArtifactDigestTree(root); err != nil {
		t.Fatalf("internal artifact symlink rejected: %v", err)
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../outside.js", link); err != nil {
		t.Fatal(err)
	}
	if _, err := ArtifactDigestTree(root); err == nil {
		t.Fatal("expected escaping artifact symlink rejection")
	}
}

type recordingBuildRunner struct {
	commands    []string
	environment string
}

func (r *recordingBuildRunner) Run(_ context.Context, command Command) (string, error) {
	action := command.Args[len(command.Args)-1]
	r.commands = append(r.commands, action)
	r.environment = strings.Join(command.Env, "\n")
	if action == "build" {
		var output string
		for _, item := range command.Env {
			if strings.HasPrefix(item, "SFORUM_NITRO_OUTPUT_DIR=") {
				output = strings.TrimPrefix(item, "SFORUM_NITRO_OUTPUT_DIR=")
			}
		}
		if err := os.MkdirAll(filepath.Join(output, "server"), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(output, "server", "index.mjs"), []byte("export {}"), 0o644); err != nil {
			return "", err
		}
	}
	return action + " ok", nil
}

func publishTestSnapshot(t *testing.T, extensionRoot, id, version string, files map[string]string) (string, string) {
	t.Helper()
	staging := t.TempDir()
	for name, body := range files {
		writeTestFile(t, filepath.Join(staging, filepath.FromSlash(name)), body)
	}
	digest, err := extensionpackage.DigestTree(staging)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(extensionRoot, id, version, digest)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(staging, target); err != nil {
		t.Fatal(err)
	}
	return target, digest
}

func writeTestFile(t *testing.T, target, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func sha256Hex(body []byte) string {
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}
