package webreleaseruntime

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
)

func TestTrustedAdminFixtureMatchesProductionContracts(t *testing.T) {
	fixture, manifest := loadTrustedAdminFixture(t)
	if err := extensionmanifest.Validate(manifest); err != nil {
		t.Fatalf("validate trusted admin fixture manifest: %v", err)
	}
	admin := manifest.Frontend.Admin
	summary, err := extensionpackage.InspectAdminFrontend(extensionpackage.FrontendInspectInput{
		PackageRoot: fixture,
		Root:        admin.Root,
		Components:  admin.Components,
		Locales:     admin.Locales,
		HostPeers:   HostPeers(),
	})
	if err != nil {
		t.Fatalf("inspect trusted admin fixture: %v", err)
	}
	if len(summary.Resolved) != 1 || summary.Resolved[0].Name != "sforum-fixture-dependency" {
		t.Fatalf("unexpected fixture dependency summary: %#v", summary)
	}

	root := t.TempDir()
	result, err := GenerateRegistry(RegistryInput{
		Root:       root,
		ReleaseID:  42,
		ReloadMode: extensions.WebReleaseReloadPrompt,
		Extensions: []RegistryExtension{{SourceRoot: fixture, Snapshot: extensions.WebReleaseExtension{
			ExtensionID: "sforum.trusted-admin-fixture", FrontendRoot: admin.Root,
			ComponentMap: admin.Components, LocaleMap: admin.Locales, TrustedComponents: manifest.Contributions,
		}}},
	})
	if err != nil {
		t.Fatalf("generate trusted admin fixture registry: %v", err)
	}
	for _, target := range []string{result.MetadataPath, result.RegistryPath} {
		if info, err := os.Stat(target); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("generated registry file missing: %s (%v)", target, err)
		}
	}
}

func TestTrustedAdminFixtureBuildIntegration(t *testing.T) {
	if os.Getenv("SFORUM_RUN_WEB_RELEASE_INTEGRATION") != "1" {
		t.Skip("set SFORUM_RUN_WEB_RELEASE_INTEGRATION=1 with the local fixture registry running")
	}
	repo := repositoryRoot(t)
	fixture, manifest := loadTrustedAdminFixture(t)
	admin := manifest.Frontend.Admin
	dependencySummary, err := extensionpackage.InspectAdminFrontend(extensionpackage.FrontendInspectInput{
		PackageRoot: fixture, Root: admin.Root, Components: admin.Components, Locales: admin.Locales, HostPeers: HostPeers(),
	})
	if err != nil {
		t.Fatal(err)
	}

	extensionRoot := filepath.Join(t.TempDir(), "extensions")
	pluginRoot, pluginDigest := publishIntegrationSnapshot(t, extensionRoot, manifest.ID, manifest.Version, fixture)
	themeSource := filepath.Join(repo, "extensions/builtin/themes/sforum-default")
	themeRoot, themeDigest := publishIntegrationSnapshot(t, extensionRoot, "sforum.default-theme", "1.0.0", themeSource)
	host, err := CompositionHost(filepath.Join(repo, "apps/web"))
	if err != nil {
		t.Fatal(err)
	}
	webExtension := extensions.WebReleaseExtension{
		ExtensionID: manifest.ID, ExtensionVersion: manifest.Version, PackageDigest: pluginDigest,
		FrontendRoot: admin.Root, ComponentMap: admin.Components, APIVersion: admin.APIVersion,
		TrustedComponents: manifest.Contributions, LocaleMap: admin.Locales, LockfileDigest: dependencySummary.LockDigest,
	}
	composition := extensions.WebComposition{
		Theme: extensions.WebThemeSnapshot{
			ExtensionID: "sforum.default-theme", Version: "1.0.0", LayerPath: filepath.Join(themeRoot, "layer"), PackageDigest: themeDigest,
		},
		Extensions: []extensions.WebExtensionSnapshot{{
			ExtensionID: manifest.ID, Version: manifest.Version, PackageDigest: pluginDigest,
			FrontendRoot: admin.Root, Dependencies: extensions.DependencySummary{
				Direct: dependencySummary.Direct, Resolved: dependencySummary.Resolved, LockDigest: dependencySummary.LockDigest,
			},
		}},
		WebSource: host.WebSource, WebLock: host.WebLock, SDKVersion: host.SDKVersion,
		BunVersion: host.BunVersion, Contract: host.Contract,
	}
	compositionBody, err := json.Marshal(composition)
	if err != nil {
		t.Fatal(err)
	}
	detail := extensions.WebReleaseDetail{WebRelease: extensions.WebRelease{
		ID: 4242, CompositionSnapshot: compositionBody, CompositionHash: sha256Hex(compositionBody),
		ActiveThemeID: composition.Theme.ExtensionID, ThemeVersion: composition.Theme.Version,
		ThemeLayerPath: composition.Theme.LayerPath, ThemePackageDigest: themeDigest, ReloadMode: extensions.WebReleaseReloadPrompt,
	}, Extensions: []extensions.WebReleaseExtension{webExtension}}

	builder := NewBuilder(Config{
		ReleaseRoot: filepath.Join(t.TempDir(), "releases"), WebRoot: filepath.Join(repo, "apps/web"), ExtensionRoot: extensionRoot,
		DefaultThemeLayer: filepath.Join(themeSource, "layer"),
		BunPath:           "bun", HostPeers: HostPeers(), SourceEnvironment: append(os.Environ(), "NPM_CONFIG_REGISTRY=http://127.0.0.1:4873"),
	})
	prepared, err := builder.Prepare(context.Background(), detail)
	if err != nil {
		t.Fatal(err)
	}
	snapshots, installLog, err := builder.Install(context.Background(), prepared)
	if err != nil {
		t.Fatalf("frozen fixture install: %v\n%s", err, installLog)
	}
	if len(snapshots) != 1 {
		t.Fatalf("unexpected dependency snapshots: %#v", snapshots)
	}
	for _, sentinel := range []string{
		filepath.Join(prepared.PluginFrontends[manifest.ID], "postinstall-ran"),
		filepath.Join(prepared.PluginFrontends[manifest.ID], "node_modules/sforum-fixture-dependency/postinstall-ran"),
	} {
		if _, err := os.Stat(sentinel); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("lifecycle sentinel exists at %s: %v", sentinel, err)
		}
	}
	for peer := range HostPeers() {
		info, err := os.Lstat(filepath.Join(prepared.PluginFrontends[manifest.ID], "node_modules", filepath.FromSlash(peer)))
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("host peer %s was not deduplicated through a symlink: %v", peer, err)
		}
	}
	result, err := builder.Build(context.Background(), prepared, installLog)
	if err != nil {
		t.Fatalf("build trusted fixture release: %v\n%s", err, result.BuildLog)
	}
	result, err = builder.Verify(context.Background(), prepared, result)
	if err != nil {
		t.Fatalf("verify trusted fixture release: %v\n%s", err, result.BuildLog)
	}
	if result.ArtifactDigest == "" || result.ManifestPath == "" {
		t.Fatalf("verified release lacks immutable metadata: %#v", result)
	}
	if pluginRoot == prepared.PluginRoots[manifest.ID] {
		t.Fatal("integration build did not isolate the plugin snapshot")
	}
}

func loadTrustedAdminFixture(t *testing.T) (string, extensionmanifest.Manifest) {
	t.Helper()
	fixture := filepath.Join(repositoryRoot(t), "tests/fixtures/extensions/trusted-admin-plugin")
	body, err := os.ReadFile(filepath.Join(fixture, extensionmanifest.ManifestFileName))
	if err != nil {
		t.Fatal(err)
	}
	var manifest extensionmanifest.Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatal(err)
	}
	return fixture, extensionmanifest.Normalize(manifest)
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	working, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs(filepath.Join(working, "../../../../.."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func publishIntegrationSnapshot(t *testing.T, extensionRoot, id, version, source string) (string, string) {
	t.Helper()
	digest, err := extensionpackage.DigestTree(source)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(extensionRoot, id, version, digest)
	if err := copyTree(source, target, nil); err != nil {
		t.Fatal(err)
	}
	return target, digest
}
