package extensions

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

func TestReadZipFileLimitedCapsInflation(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	writeZipFile(t, writer, zipFile{name: "big.txt", body: strings.Repeat("A", 64)})
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	reader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	if len(reader.File) != 1 {
		t.Fatalf("expected 1 file, got %d", len(reader.File))
	}
	if _, err := readZipFileLimited(reader.File[0], 16); !errors.Is(err, ErrInvalidArchive) {
		t.Fatalf("expected ErrInvalidArchive when inflate exceeds cap, got %v", err)
	}
	body, err := readZipFileLimited(reader.File[0], 128)
	if err != nil {
		t.Fatalf("expected success under higher cap: %v", err)
	}
	if len(body) != 64 {
		t.Fatalf("expected 64 bytes, got %d", len(body))
	}
}

func TestServiceInstallArchiveRequiresExtensionManagePermission(t *testing.T) {
	service := NewService(&fakeExtensionStore{}, t.TempDir())
	actor := identity.Actor{ID: 7, Status: identity.UserStatusActive, Permissions: map[string]bool{}}

	_, err := service.InstallArchive(context.Background(), actor, ArchiveInput{
		FileName: "sample.zip",
		Data:     extensionArchive(t, validManifest("demo.plugin", TypePlugin)),
	})
	if !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestServiceInstallArchiveValidatesManifestAndSafeZipPaths(t *testing.T) {
	service := NewService(&fakeExtensionStore{}, t.TempDir())
	actor := extensionManager()

	_, err := service.InstallArchive(context.Background(), actor, ArchiveInput{
		FileName: "missing.zip",
		Data:     extensionArchive(t, "", zipFile{name: "README.md", body: "hello"}),
	})
	if !errors.Is(err, ErrInvalidArchive) {
		t.Fatalf("expected missing manifest to be invalid archive, got %v", err)
	}

	_, err = service.InstallArchive(context.Background(), actor, ArchiveInput{
		FileName: "unsafe.zip",
		Data: extensionArchive(t, validManifest("demo.plugin", TypePlugin),
			zipFile{name: "../escape.txt", body: "oops"},
		),
	})
	if !errors.Is(err, ErrInvalidArchive) {
		t.Fatalf("expected unsafe path to be invalid archive, got %v", err)
	}

	_, err = service.InstallArchive(context.Background(), actor, ArchiveInput{
		FileName: "bad-manifest.zip",
		Data:     extensionArchive(t, `{"id":"Bad Space","name":"Broken","version":"1.0.0","type":"plugin","sforumVersion":"^1.0.0"}`),
	})
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid manifest, got %v", err)
	}
}

func TestServiceInstallArchiveRejectsZipSymlink(t *testing.T) {
	service := NewService(&fakeExtensionStore{}, t.TempDir())

	_, err := service.InstallArchive(context.Background(), extensionManager(), ArchiveInput{
		FileName: "symlink.zip",
		Data: extensionArchive(t, validManifest("demo.plugin", TypePlugin),
			zipFile{name: "backend/plugin", body: "../../outside", mode: os.ModeSymlink | 0o777},
		),
	})

	if !errors.Is(err, ErrInvalidArchive) {
		t.Fatalf("expected symlink entry to be an invalid archive, got %v", err)
	}
}

func TestServiceInstallArchiveRejectsAmbiguousZipEntries(t *testing.T) {
	tests := []struct {
		name  string
		files []zipFile
	}{
		{
			name: "duplicate normalized path",
			files: []zipFile{
				{name: "frontend//Cell.vue", body: "first"},
				{name: "frontend/Cell.vue", body: "second"},
			},
		},
		{
			name: "file and directory collision",
			files: []zipFile{
				{name: "assets", body: "file"},
				{name: "assets/icon.svg", body: "icon"},
			},
		},
		{
			name: "duplicate manifest",
			files: []zipFile{
				{name: ManifestFileName, body: validManifest("other.plugin", TypePlugin)},
			},
		},
		{
			name: "special file mode",
			files: []zipFile{
				{name: "frontend/channel", body: "special", mode: os.ModeNamedPipe | 0o644},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewService(&fakeExtensionStore{}, t.TempDir())
			_, err := service.InstallArchive(context.Background(), extensionManager(), ArchiveInput{
				FileName: "ambiguous.zip",
				Data:     extensionArchive(t, validManifest("demo.plugin", TypePlugin), test.files...),
			})
			if !errors.Is(err, ErrInvalidArchive) {
				t.Fatalf("expected invalid archive, got %v", err)
			}
		})
	}
}

func TestServiceInstallArchiveStoresManifestPackageAndEvent(t *testing.T) {
	store := &fakeExtensionStore{}
	extensionRoot := t.TempDir()
	service := NewService(store, extensionRoot)

	installed, err := service.InstallArchive(context.Background(), extensionManager(), ArchiveInput{
		FileName: "sample.zip",
		Data: extensionArchive(t, validManifest("demo.plugin", TypePlugin),
			zipFile{name: "backend/plugin", body: "#!/bin/sh\n"},
			zipFile{name: "README.md", body: "hello"},
		),
	})
	if err != nil {
		t.Fatalf("InstallArchive returned error: %v", err)
	}

	if installed.ID != "demo.plugin" || installed.Version != "1.0.0" || installed.Status != StatusInstalled {
		t.Fatalf("unexpected installed extension: %#v", installed)
	}
	if store.saved.Manifest.ID != "demo.plugin" || store.saved.Manifest.Type != TypePlugin {
		t.Fatalf("manifest was not saved: %#v", store.saved.Manifest)
	}
	if store.saved.PackagePath == "" || store.saved.PackageDigest == "" {
		t.Fatalf("expected snapshot path and digest to be stored: %#v", store.saved)
	}
	if installed.PackagePath != store.saved.PackagePath || installed.PackageDigest != store.saved.PackageDigest {
		t.Fatalf("installed extension did not propagate snapshot identity: installed=%#v saved=%#v", installed, store.saved)
	}
	packageInfo, err := os.Stat(installed.PackagePath)
	if err != nil || !packageInfo.IsDir() {
		t.Fatalf("expected package path to be a snapshot directory, info=%#v err=%v", packageInfo, err)
	}
	if _, err := os.Stat(filepath.Join(installed.PackagePath, ManifestFileName)); err != nil {
		t.Fatalf("expected canonical manifest in snapshot root: %v", err)
	}
	backendPath, ok := installedFilePath(installed, "backend/plugin")
	if !ok || backendPath != filepath.Join(installed.PackagePath, "backend", "plugin") {
		t.Fatalf("digest-backed extension path did not resolve from snapshot root: path=%q ok=%t", backendPath, ok)
	}
	if body, err := os.ReadFile(backendPath); err != nil || string(body) != "#!/bin/sh\n" {
		t.Fatalf("unexpected snapshotted backend: body=%q err=%v", body, err)
	}
	if _, err := os.Stat(filepath.Join(installed.PackagePath, "package.zip")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("snapshot must not retain the uploaded ZIP wrapper, got %v", err)
	}
	if len(store.events) != 1 || store.events[0].Action != EventInstalled {
		t.Fatalf("expected install event, got %#v", store.events)
	}
}

func TestServiceInstallArchiveKeepsDifferentContentForSameIDAndVersion(t *testing.T) {
	store := &fakeExtensionStore{}
	service := NewService(store, t.TempDir())
	manifest := validManifest("changed.plugin", TypePlugin)

	first, err := service.InstallArchive(context.Background(), extensionManager(), ArchiveInput{
		FileName: "first.zip",
		Data: extensionArchive(t, manifest,
			zipFile{name: "README.md", body: "first"},
		),
	})
	if err != nil {
		t.Fatalf("install first package: %v", err)
	}
	second, err := service.InstallArchive(context.Background(), extensionManager(), ArchiveInput{
		FileName: "second.zip",
		Data: extensionArchive(t, manifest,
			zipFile{name: "README.md", body: "second"},
		),
	})
	if err != nil {
		t.Fatalf("install changed package: %v", err)
	}

	if first.PackageDigest == "" || second.PackageDigest == "" {
		t.Fatalf("expected both package digests: first=%#v second=%#v", first, second)
	}
	if first.PackageDigest == second.PackageDigest || first.PackagePath == second.PackagePath {
		t.Fatalf("different package content reused one snapshot: first=%#v second=%#v", first, second)
	}
	firstBody, err := os.ReadFile(filepath.Join(first.PackagePath, "README.md"))
	if err != nil {
		t.Fatalf("read first immutable package: %v", err)
	}
	secondBody, err := os.ReadFile(filepath.Join(second.PackagePath, "README.md"))
	if err != nil {
		t.Fatalf("read second immutable package: %v", err)
	}
	if string(firstBody) != "first" || string(secondBody) != "second" {
		t.Fatalf("package snapshots were overwritten: first=%q second=%q", firstBody, secondBody)
	}
}

func TestServiceInstallArchiveRejectsReservedDefaultThemeID(t *testing.T) {
	service := NewService(&fakeExtensionStore{}, t.TempDir())

	_, err := service.InstallArchive(context.Background(), extensionManager(), ArchiveInput{
		FileName: "default-theme.zip",
		Data:     extensionArchive(t, validManifest(DefaultThemeID, TypeTheme)),
	})
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected reserved default theme id to be rejected, got %v", err)
	}
}

func TestServiceInstallArchiveAllowsThemeSettingsAndAdminPages(t *testing.T) {
	service := NewService(&fakeExtensionStore{}, t.TempDir())
	actor := extensionManager()

	installed, err := service.InstallArchive(context.Background(), actor, ArchiveInput{
		FileName: "starter-theme.zip",
		Data: extensionArchive(t, validThemeManifest("starter.theme"),
			zipFile{name: "theme.json", body: `{"schemaVersion":1,"styles":{"tokens":{}}}`},
		),
	})
	if err != nil {
		t.Fatalf("expected minimal theme manifest to install, got %v", err)
	}
	if installed.Type != TypeTheme {
		t.Fatalf("unexpected installed theme: %#v", installed)
	}

	themeWithAdmin, err := service.InstallArchive(context.Background(), actor, ArchiveInput{
		FileName: "starter-theme-admin.zip",
		Data: extensionArchive(t, `{
			"id":"starter.admin.theme",
			"name":"Starter Admin Theme",
			"description":"Theme with admin settings.",
			"url":"https://example.com/starter-admin-theme",
			"author":{"name":"SForum Team","url":"https://example.com","email":"dev@example.com"},
			"version":"1.0.0",
			"type":"theme",
			"sforumVersion":"^1.0.0",
			"settings":[{"key":"theme.banner","label":"Banner","type":"text","default":"Welcome"}],
			"adminPages":[{"path":"/settings","label":"Settings","view":"settings","icon":"i-lucide-settings","order":10}],
			"frontend":{"layer":"frontend/layer"}
		}`,
			zipFile{name: "frontend/layer/nuxt.config.ts", body: "export default defineNuxtConfig({})\n"},
		),
	})
	if err != nil {
		t.Fatalf("expected theme settings/admin page manifest to install, got %v", err)
	}
	if len(themeWithAdmin.Manifest.Settings) != 1 || len(themeWithAdmin.Manifest.AdminPages) != 1 {
		t.Fatalf("expected theme settings and admin pages to be preserved, got %#v", themeWithAdmin.Manifest)
	}

	cases := []struct {
		name     string
		manifest string
	}{
		{
			name: "missing frontend layer",
			manifest: `{
				"id":"bad.theme","name":"Bad Theme","version":"1.0.0","type":"theme","sforumVersion":"^1.0.0"
			}`,
		},
		{
			name: "theme declares permissions",
			manifest: `{
				"id":"bad.theme","name":"Bad Theme","version":"1.0.0","type":"theme","sforumVersion":"^1.0.0",
				"permissions":["topic.create"],"frontend":{"layer":"frontend/layer"}
			}`,
		},
		{
			name: "theme declares migrations",
			manifest: `{
				"id":"bad.theme","name":"Bad Theme","version":"1.0.0","type":"theme","sforumVersion":"^1.0.0",
				"migrations":[{"path":"migrations/001.sql"}],"frontend":{"layer":"frontend/layer"}
			}`,
		},
		{
			name: "theme declares backend",
			manifest: `{
				"id":"bad.theme","name":"Bad Theme","version":"1.0.0","type":"theme","sforumVersion":"^1.0.0",
				"backend":{"entry":"backend/plugin","rpc":"hashicorp-go-plugin"},"frontend":{"layer":"frontend/layer"}
			}`,
		},
		{
			name: "theme declares routes",
			manifest: `{
				"id":"bad.theme","name":"Bad Theme","version":"1.0.0","type":"theme","sforumVersion":"^1.0.0",
				"routes":[{"path":"/hello","methods":["GET"]}],"frontend":{"layer":"frontend/layer"}
			}`,
		},
		{
			name: "theme declares hooks",
			manifest: `{
				"id":"bad.theme","name":"Bad Theme","version":"1.0.0","type":"theme","sforumVersion":"^1.0.0",
				"hooks":[{"name":"topic.created"}],"frontend":{"layer":"frontend/layer"}
			}`,
		},
		{
			name: "theme declares events",
			manifest: `{
				"id":"bad.theme","name":"Bad Theme","version":"1.0.0","type":"theme","sforumVersion":"^1.0.0",
				"events":[{"name":"topic.created","kind":"observe"}],"frontend":{"layer":"frontend/layer"}
			}`,
		},
		{
			name: "theme declares jobs",
			manifest: `{
				"id":"bad.theme","name":"Bad Theme","version":"1.0.0","type":"theme","sforumVersion":"^1.0.0",
				"jobs":[{"name":"demo.sync"}],"frontend":{"layer":"frontend/layer"}
			}`,
		},
		{
			name: "theme declares providers",
			manifest: `{
				"id":"bad.theme","name":"Bad Theme","version":"1.0.0","type":"theme","sforumVersion":"^1.0.0",
				"providers":[{"slot":"search.provider","label":"Search"}],"frontend":{"layer":"frontend/layer"}
			}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.InstallArchive(context.Background(), actor, ArchiveInput{
				FileName: "bad-theme.zip",
				Data:     extensionArchive(t, tc.manifest),
			})
			if !errors.Is(err, ErrInvalidManifest) {
				t.Fatalf("expected invalid theme manifest, got %v", err)
			}
		})
	}
}

func TestServiceListsContributionPointsAndEffectiveContributions(t *testing.T) {
	store := &fakeExtensionStore{items: map[string]Extension{}}
	service := NewService(store, t.TempDir())
	store.items["beta.plugin"] = contributionTestPlugin("beta.plugin", StatusEnabled, []ManifestContribution{
		topicActionContribution(t, "beta.later", 200, "/topic-actions/later"),
	})
	store.items["alpha.plugin"] = contributionTestPlugin("alpha.plugin", StatusEnabled, []ManifestContribution{
		topicActionContribution(t, "alpha.first", 100, "/topic-actions/first"),
		topicActionContribution(t, "alpha.same", 200, "/topic-actions/same"),
	})
	store.items["disabled.plugin"] = contributionTestPlugin("disabled.plugin", StatusDisabled, []ManifestContribution{
		topicActionContribution(t, "disabled.hidden", 1, "/topic-actions/hidden"),
	})
	store.items["theme.demo"] = Extension{
		ID:      "theme.demo",
		Name:    "Theme Demo",
		Version: "1.0.0",
		Type:    TypeTheme,
		Status:  StatusEnabled,
		Manifest: Manifest{
			ID:            "theme.demo",
			Name:          "Theme Demo",
			Description:   "Theme demo.",
			URL:           "https://example.com/theme",
			Author:        ManifestAuthor{Name: "SForum Team"},
			Version:       "1.0.0",
			Type:          TypeTheme,
			SForumVersion: "^1.0.0",
		},
	}

	points, err := service.ContributionPoints(context.Background(), extensionManager())
	if err != nil {
		t.Fatalf("ContributionPoints returned error: %v", err)
	}
	// 与 ExtensionManifest.ContributionPointDefinitions 生产目录对齐（含 F4.3 + E2）。
	if len(points) != 10 || points[0].ID != "forum.topic.actions" {
		t.Fatalf("unexpected contribution points: %#v", points)
	}
	pointIDs := make(map[string]bool, len(points))
	for _, point := range points {
		pointIDs[point.ID] = true
	}
	for _, id := range []string{
		"forum.topic.sidebar",
		"forum.topic.badges",
		"forum.comment.actions",
		"forum.nav.items",
		"forum.topic.list.badges",
		"forum.composer.toolbar",
		"forum.profile.tabs",
		"admin.dashboard.widgets",
		"system.health.checks",
	} {
		if !pointIDs[id] {
			t.Fatalf("missing F4.3/E2 contribution point %q in %#v", id, points)
		}
	}

	contributions, err := service.Contributions(context.Background(), extensionManager())
	if err != nil {
		t.Fatalf("Contributions returned error: %v", err)
	}
	if got := contributionIDs(contributions); !slices.Equal(got, []string{"alpha.plugin:alpha.first", "alpha.plugin:alpha.same", "beta.plugin:beta.later"}) {
		t.Fatalf("unexpected contribution order: %#v", got)
	}
	if contributions[0].ExtensionName != "Alpha Plugin" || contributions[0].Point != "forum.topic.actions" {
		t.Fatalf("unexpected contribution metadata: %#v", contributions[0])
	}
}

func TestServiceEffectiveContributionsResolveWithoutAdminActor(t *testing.T) {
	store := &fakeExtensionStore{items: map[string]Extension{}}
	service := NewService(store, t.TempDir())
	store.items["beta.plugin"] = contributionTestPlugin("beta.plugin", StatusEnabled, []ManifestContribution{
		topicActionContribution(t, "beta.later", 200, "/topic-actions/later"),
	})
	store.items["alpha.plugin"] = contributionTestPlugin("alpha.plugin", StatusEnabled, []ManifestContribution{
		topicActionContribution(t, "alpha.first", 100, "/topic-actions/first"),
	})
	store.items["disabled.plugin"] = contributionTestPlugin("disabled.plugin", StatusDisabled, []ManifestContribution{
		topicActionContribution(t, "disabled.hidden", 1, "/topic-actions/hidden"),
	})

	contributions, err := service.EffectiveContributions(context.Background())
	if err != nil {
		t.Fatalf("EffectiveContributions returned error: %v", err)
	}
	if got := contributionIDs(contributions); !slices.Equal(got, []string{"alpha.plugin:alpha.first", "beta.plugin:beta.later"}) {
		t.Fatalf("unexpected runtime contribution order: %#v", got)
	}
}

func TestServiceContributionInspectionRequiresExtensionManage(t *testing.T) {
	service := NewService(&fakeExtensionStore{}, t.TempDir())
	actor := identity.Actor{ID: 7, Status: identity.UserStatusActive, Permissions: map[string]bool{}}

	if _, err := service.ContributionPoints(context.Background(), actor); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected permission denied for contribution points, got %v", err)
	}
	if _, err := service.Contributions(context.Background(), actor); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected permission denied for contributions, got %v", err)
	}
}

func TestServiceSyncBuiltinsPrunesRemovedBuiltinExtensions(t *testing.T) {
	builtinRoot := t.TempDir()
	themeRoot := filepath.Join(builtinRoot, "themes", "sforum-default")
	if err := os.MkdirAll(themeRoot, 0o755); err != nil {
		t.Fatalf("create builtin theme root: %v", err)
	}
	defaultTheme := protectedBuiltinExtension(DefaultThemeID, TypeTheme)
	defaultTheme.PackagePath = themeRoot
	if err := writeManifest(themeRoot, defaultTheme.Manifest); err != nil {
		t.Fatalf("write builtin theme manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeRoot, "theme.json"), []byte(`{"schemaVersion":1,"styles":{"tokens":{}}}`), 0o644); err != nil {
		t.Fatalf("write builtin theme contract: %v", err)
	}

	stalePlugin := protectedBuiltinExtension("sforum.default", TypePlugin)
	stalePlugin.Name = "SForum Default Plugin"
	stalePlugin.Manifest.Name = stalePlugin.Name
	stalePlugin.Manifest.Backend = ManifestBackend{RPC: "hashicorp-go-plugin"}
	stalePlugin.PackagePath = filepath.Join(builtinRoot, "plugins", "sforum-default")
	store := &fakeExtensionStore{items: map[string]Extension{
		stalePlugin.ID: stalePlugin,
	}}
	service := NewServiceWithBuiltins(store, t.TempDir(), builtinRoot)

	if _, err := service.SyncBuiltins(context.Background()); err != nil {
		t.Fatalf("SyncBuiltins returned error: %v", err)
	}
	if _, ok := store.items[stalePlugin.ID]; ok {
		t.Fatalf("expected stale builtin plugin to be pruned")
	}
	if item, ok := store.items[DefaultThemeID]; !ok || item.Source != SourceBuiltin {
		t.Fatalf("expected current builtin theme to remain, got %#v", item)
	}
}

func TestServiceSyncBuiltinsStoresImmutableSnapshotIdentity(t *testing.T) {
	builtinRoot := t.TempDir()
	extensionRoot := t.TempDir()
	themeRoot := filepath.Join(builtinRoot, "themes", "sforum-default")
	defaultTheme := protectedBuiltinExtension(DefaultThemeID, TypeTheme)
	if err := os.MkdirAll(themeRoot, 0o755); err != nil {
		t.Fatalf("create builtin theme root: %v", err)
	}
	if err := writeManifest(themeRoot, defaultTheme.Manifest); err != nil {
		t.Fatalf("write builtin theme manifest: %v", err)
	}
	const sourceBody = `{"schemaVersion":1,"styles":{"tokens":{}}}`
	sourceTheme := filepath.Join(themeRoot, "theme.json")
	if err := os.WriteFile(sourceTheme, []byte(sourceBody), 0o644); err != nil {
		t.Fatalf("write builtin theme contract: %v", err)
	}
	store := &fakeExtensionStore{items: map[string]Extension{}}
	service := NewServiceWithBuiltins(store, extensionRoot, builtinRoot)

	if _, err := service.SyncBuiltins(context.Background()); err != nil {
		t.Fatalf("SyncBuiltins returned error: %v", err)
	}
	saved := store.items[DefaultThemeID]
	if saved.PackagePath == "" || saved.PackageDigest == "" {
		t.Fatalf("expected builtin snapshot identity, got %#v", saved)
	}
	if saved.PackagePath == themeRoot || !strings.HasPrefix(saved.PackagePath, filepath.Clean(extensionRoot)+string(os.PathSeparator)) {
		t.Fatalf("builtin package was not copied below extension root: source=%q saved=%q", themeRoot, saved.PackagePath)
	}
	snapshotTheme := filepath.Join(saved.PackagePath, "theme.json")
	if err := os.WriteFile(sourceTheme, []byte("changed after sync"), 0o644); err != nil {
		t.Fatalf("change builtin source after sync: %v", err)
	}
	body, err := os.ReadFile(snapshotTheme)
	if err != nil {
		t.Fatalf("read builtin snapshot layer: %v", err)
	}
	if string(body) != sourceBody {
		t.Fatalf("builtin snapshot changed with source tree: %q", body)
	}
}

func TestServiceInstallArchiveValidatesRuntimeManifestDeclarations(t *testing.T) {
	service := NewService(&fakeExtensionStore{}, t.TempDir())
	actor := extensionManager()

	valid := validManifest("runtime.plugin", TypePlugin)
	installed, err := service.InstallArchive(context.Background(), actor, ArchiveInput{
		FileName: "runtime.zip",
		Data: extensionArchive(t, valid,
			zipFile{name: "backend/plugin", body: "#!/bin/sh\n"},
		),
	})
	if err != nil {
		t.Fatalf("expected runtime manifest to install, got %v", err)
	}
	if events := DeclaredManifestEvents(installed.Manifest); len(events) != 1 || events[0].Name != "topic.created" {
		t.Fatalf("expected hooks compatibility event declaration, got %#v", events)
	}

	eventManifest := `{
		"id":"event.plugin",
		"name":"Event Plugin",
		"description":"Event plugin.",
		"url":"https://example.com/event-plugin",
		"author":{"name":"SForum Team","url":"https://example.com","email":"dev@example.com"},
		"version":"1.0.0",
		"type":"plugin",
		"sforumVersion":"^1.0.0",
		"events":[{"name":"topic.before_create","kind":"filter","timeoutMs":1000}]
	}`
	installed, err = service.InstallArchive(context.Background(), actor, ArchiveInput{
		FileName: "event.zip",
		Data:     extensionArchive(t, eventManifest),
	})
	if err != nil {
		t.Fatalf("expected event manifest to install, got %v", err)
	}
	if events := DeclaredManifestEvents(installed.Manifest); len(events) != 1 || events[0].Kind != "filter" {
		t.Fatalf("expected filter event declaration, got %#v", events)
	}

	cases := []struct {
		name     string
		manifest string
	}{
		{
			name: "unsafe route path",
			manifest: `{
				"id":"bad.route","name":"Bad Route","version":"1.0.0","type":"plugin","sforumVersion":"^1.0.0",
				"routes":[{"path":"../escape","methods":["GET"]}]
			}`,
		},
		{
			name: "public write route",
			manifest: `{
				"id":"bad.public","name":"Bad Public","version":"1.0.0","type":"plugin","sforumVersion":"^1.0.0",
				"routes":[{"path":"/write","methods":["POST"],"access":"public"}]
			}`,
		},
		{
			name: "unknown hook",
			manifest: `{
				"id":"bad.hook","name":"Bad Hook","version":"1.0.0","type":"plugin","sforumVersion":"^1.0.0",
				"hooks":[{"name":"topic.destroyed"}]
			}`,
		},
		{
			name: "unknown provider",
			manifest: `{
				"id":"bad.provider","name":"Bad Provider","version":"1.0.0","type":"plugin","sforumVersion":"^1.0.0",
				"providers":[{"slot":"unknown.provider","label":"Unknown"}]
			}`,
		},
		{
			name: "unknown event",
			manifest: `{
				"id":"bad.event","name":"Bad Event","version":"1.0.0","type":"plugin","sforumVersion":"^1.0.0",
				"events":[{"name":"topic.destroyed","kind":"observe"}]
			}`,
		},
		{
			name: "wrong event kind",
			manifest: `{
				"id":"bad.event.kind","name":"Bad Event Kind","version":"1.0.0","type":"plugin","sforumVersion":"^1.0.0",
				"events":[{"name":"topic.before_create","kind":"observe"}]
			}`,
		},
		{
			name: "permission route without manifest permission",
			manifest: `{
				"id":"bad.permission","name":"Bad Permission","version":"1.0.0","type":"plugin","sforumVersion":"^1.0.0",
				"routes":[{"path":"/admin","methods":["POST"],"access":"permission","permission":"extension.bad.manage"}]
			}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.InstallArchive(context.Background(), actor, ArchiveInput{
				FileName: "bad.zip",
				Data:     extensionArchive(t, tc.manifest),
			})
			if !errors.Is(err, ErrInvalidManifest) {
				t.Fatalf("expected invalid manifest, got %v", err)
			}
		})
	}
}

func TestServiceNavigationUsesOnlyExplicitMenuPagesFromEnabledPluginsAndActiveTheme(t *testing.T) {
	enabledPlugin := installedExtension("enabled.plugin", TypePlugin, ManifestBackend{})
	enabledPlugin.Status = StatusEnabled
	enabledPlugin.Manifest.Admin = ManifestAdmin{
		Entry: "/settings",
		Pages: []ManifestAdminPage{
			{Path: "/settings", Label: "Settings", View: "settings", Icon: "i-lucide-settings", Order: 20},
			{Path: "/dashboard", Label: "Dashboard", View: "about", Icon: "i-lucide-layout-dashboard", Order: 10, Menu: true},
		},
	}
	disabledPlugin := installedExtension("disabled.plugin", TypePlugin, ManifestBackend{})
	disabledPlugin.Status = StatusDisabled
	disabledPlugin.Manifest.Admin = ManifestAdmin{
		Pages: []ManifestAdminPage{{Path: "/hidden", Label: "Hidden", View: "about", Menu: true}},
	}
	activeTheme := installedExtension("active.theme", TypeTheme, ManifestBackend{})
	activeTheme.Status = StatusEnabled
	activeTheme.Manifest.Admin = ManifestAdmin{
		Pages: []ManifestAdminPage{{Path: "/theme", Label: "Theme", View: "about", Order: 30, Menu: true}},
	}
	store := &fakeExtensionStore{items: map[string]Extension{
		enabledPlugin.ID:  enabledPlugin,
		disabledPlugin.ID: disabledPlugin,
		activeTheme.ID:    activeTheme,
	}}
	service := NewService(store, t.TempDir())

	items, err := service.Navigation(context.Background(), extensionManager())
	if err != nil {
		t.Fatalf("Navigation returned error: %v", err)
	}
	if navigationContains(items, "enabled.plugin", "/about") || navigationContains(items, "enabled.plugin", "/settings") {
		t.Fatalf("generated and non-menu pages should not inject sidebar navigation: %#v", items)
	}
	if !navigationContains(items, "enabled.plugin", "/dashboard") {
		t.Fatalf("expected enabled plugin menu page, got %#v", items)
	}
	if !navigationContains(items, "active.theme", "/theme") {
		t.Fatalf("expected active theme explicit menu page, got %#v", items)
	}
	if navigationContains(items, "disabled.plugin", "/hidden") {
		t.Fatalf("disabled plugin should not inject sidebar navigation: %#v", items)
	}
}

func TestServicePublicActiveThemeSettingsOmitsSecrets(t *testing.T) {
	theme := protectedBuiltinExtension(DefaultThemeID, TypeTheme)
	theme.Status = StatusEnabled
	theme.Manifest.Settings = []ManifestSetting{
		{Key: "home.notice.zh-CN", Label: LocalizedText{Default: "Notice"}, Type: "text", Default: "默认提示"},
		{Key: "home.right_rail.enabled", Label: LocalizedText{Default: "Rail"}, Type: "boolean", Default: "true"},
		{Key: "secret.token", Label: LocalizedText{Default: "Secret"}, Type: "secret", Default: ""},
	}
	store := &fakeExtensionStore{
		items: map[string]Extension{
			DefaultThemeID: theme,
		},
		activeThemeID: DefaultThemeID,
		settings: map[string]map[string]string{
			DefaultThemeID: {
				"home.notice.zh-CN": "自定义提示",
				"secret.token":      "should-not-leak",
			},
		},
	}
	service := NewService(store, t.TempDir())

	got, err := service.PublicActiveThemeSettings(context.Background())
	if err != nil {
		t.Fatalf("PublicActiveThemeSettings: %v", err)
	}
	if got.ThemeID != DefaultThemeID {
		t.Fatalf("themeId=%q", got.ThemeID)
	}
	if got.Settings["home.notice.zh-CN"] != "自定义提示" {
		t.Fatalf("expected stored notice, got %#v", got.Settings)
	}
	if got.Settings["home.right_rail.enabled"] != "true" {
		t.Fatalf("expected default boolean, got %#v", got.Settings)
	}
	if _, ok := got.Settings["secret.token"]; ok {
		t.Fatalf("secret must not appear in public settings: %#v", got.Settings)
	}
}

func TestServiceThemeSettingsRequireThemeManagePermission(t *testing.T) {
	theme := protectedBuiltinExtension(DefaultThemeID, TypeTheme)
	theme.Status = StatusEnabled
	theme.Manifest.Settings = []ManifestSetting{
		{Key: "home.notice.zh-CN", Label: LocalizedText{Default: "Notice"}, Type: "text", Default: "默认"},
	}
	store := &fakeExtensionStore{
		items: map[string]Extension{DefaultThemeID: theme},
	}
	service := NewService(store, t.TempDir())

	themeActor := identity.Actor{
		ID:     2,
		Status: identity.UserStatusActive,
		Permissions: map[string]bool{
			identity.PermissionExtensionThemeManage: true,
		},
	}
	if _, err := service.Settings(context.Background(), themeActor, DefaultThemeID, "zh-CN"); err != nil {
		t.Fatalf("theme manager should read settings: %v", err)
	}

	pluginOnly := identity.Actor{
		ID:     3,
		Status: identity.UserStatusActive,
		Permissions: map[string]bool{
			identity.PermissionExtensionPluginManage: true,
		},
	}
	if _, err := service.Settings(context.Background(), pluginOnly, DefaultThemeID, "zh-CN"); !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("plugin manager must not manage theme settings, got %v", err)
	}
}

func TestServiceSettingsResolveUpdateAndResetDefaults(t *testing.T) {
	item := installedExtension("settings.plugin", TypePlugin, ManifestBackend{})
	// 设置读写仅对已启用扩展开放。
	item.Status = StatusEnabled
	item.Manifest.Settings = []ManifestSetting{
		{Key: "demo.enabled", Label: LocalizedText{Default: "Enabled"}, Type: "boolean", Default: "true"},
		{Key: "demo.title", Label: LocalizedText{Default: "Title"}, Type: "select", Default: "Hello", RecommendedValue: "Hello", Placeholder: LocalizedText{Default: "Choose"}, Group: LocalizedText{Default: "general"}, Options: []ManifestSettingOption{{Value: "Hello", Label: LocalizedText{Default: "Hello"}}, {Value: "World", Label: LocalizedText{Default: "World"}}}},
	}
	store := &fakeExtensionStore{items: map[string]Extension{item.ID: item}}
	service := NewService(store, t.TempDir())

	settings, err := service.Settings(context.Background(), extensionManager(), item.ID, "zh-CN")
	if err != nil {
		t.Fatalf("Settings returned error: %v", err)
	}
	if settingValue(settings, "demo.title") != "Hello" {
		t.Fatalf("expected default setting value, got %#v", settings)
	}
	if settings.Items[1].RecommendedValue != "Hello" || settings.Items[1].Placeholder != "Choose" || settings.Items[1].Group != "general" || len(settings.Items[1].Options) != 2 {
		t.Fatalf("expected presentation metadata, got %#v", settings.Items[1])
	}

	updated, err := service.UpdateSettings(context.Background(), extensionManager(), item.ID, UpdateSettingsInput{Values: map[string]string{"demo.title": "Updated"}}, "zh-CN")
	if err != nil {
		t.Fatalf("UpdateSettings returned error: %v", err)
	}
	if settingValue(updated, "demo.title") != "Updated" {
		t.Fatalf("expected updated setting value, got %#v", updated)
	}

	_, err = service.UpdateSettings(context.Background(), extensionManager(), item.ID, UpdateSettingsInput{Values: map[string]string{"unknown": "bad"}}, "zh-CN")
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected invalid setting key, got %v", err)
	}

	reset, err := service.ResetSettings(context.Background(), extensionManager(), item.ID, "zh-CN")
	if err != nil {
		t.Fatalf("ResetSettings returned error: %v", err)
	}
	if settingValue(reset, "demo.title") != "Hello" {
		t.Fatalf("expected default after reset, got %#v", reset)
	}
}

func TestUpdateSettingsPreservesOmittedValues(t *testing.T) {
	item := installedExtension("partial.plugin", TypePlugin, ManifestBackend{})
	item.Status = StatusEnabled
	item.Manifest.Settings = []ManifestSetting{
		{Key: "a", Label: LocalizedText{Default: "A"}, Type: "text", Default: "da"},
		{Key: "b", Label: LocalizedText{Default: "B"}, Type: "text", Default: "db"},
		{Key: "token", Label: LocalizedText{Default: "T"}, Type: "secret"},
	}
	store := &fakeExtensionStore{
		items: map[string]Extension{item.ID: item},
		settings: map[string]map[string]string{
			item.ID: {"a": "keep-a", "b": "keep-b", "token": "secret-value"},
		},
	}
	service := NewService(store, t.TempDir())

	// 只更新 b；a 与 token 必须保留。
	updated, err := service.UpdateSettings(context.Background(), extensionManager(), item.ID, UpdateSettingsInput{
		Values: map[string]string{"b": "new-b"},
	}, "zh-CN")
	if err != nil {
		t.Fatal(err)
	}
	if settingValue(updated, "b") != "new-b" {
		t.Fatalf("b=%q", settingValue(updated, "b"))
	}
	if store.settings[item.ID]["a"] != "keep-a" || store.settings[item.ID]["b"] != "new-b" {
		t.Fatalf("store=%#v", store.settings[item.ID])
	}
	if store.settings[item.ID]["token"] != "secret-value" {
		t.Fatalf("secret lost: %#v", store.settings[item.ID])
	}
	// 掩码
	if settingValue(updated, "token") != "" {
		t.Fatal("secret must stay masked in response")
	}
}

func TestUpdateSettingsInvalidKeyChangesNothing(t *testing.T) {
	item := installedExtension("bad.plugin", TypePlugin, ManifestBackend{})
	item.Status = StatusEnabled
	item.Manifest.Settings = []ManifestSetting{
		{Key: "a", Label: LocalizedText{Default: "A"}, Type: "text", Default: "da"},
	}
	store := &fakeExtensionStore{
		items:    map[string]Extension{item.ID: item},
		settings: map[string]map[string]string{item.ID: {"a": "keep"}},
	}
	service := NewService(store, t.TempDir())
	_, err := service.UpdateSettings(context.Background(), extensionManager(), item.ID, UpdateSettingsInput{
		Values: map[string]string{"a": "x", "unknown": "y"},
	}, "zh-CN")
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("got %v", err)
	}
	if store.settings[item.ID]["a"] != "keep" {
		t.Fatalf("store mutated: %#v", store.settings[item.ID])
	}
}

func TestServiceSettingsAllowConfigurationWhenExtensionDisabled(t *testing.T) {
	item := installedExtension("settings.plugin", TypePlugin, ManifestBackend{})
	item.Status = StatusDisabled
	item.Manifest.Settings = []ManifestSetting{
		{Key: "demo.title", Label: LocalizedText{Default: "Title"}, Type: "text", Default: "Hello"},
	}
	store := &fakeExtensionStore{items: map[string]Extension{item.ID: item}}
	service := NewService(store, t.TempDir())
	actor := extensionManager()

	if _, err := service.Settings(context.Background(), actor, item.ID, "zh-CN"); err != nil {
		t.Fatalf("expected disabled settings read, got %v", err)
	}
	if _, err := service.UpdateSettings(context.Background(), actor, item.ID, UpdateSettingsInput{Values: map[string]string{"demo.title": "x"}}, "zh-CN"); err != nil {
		t.Fatalf("expected disabled settings update, got %v", err)
	}
	if _, err := service.ResetSettings(context.Background(), actor, item.ID, "zh-CN"); err != nil {
		t.Fatalf("expected disabled settings reset, got %v", err)
	}
}

func TestUpdateSettingsRestoresSnapshotWhenPluginRestartFails(t *testing.T) {
	item := installedExtension("settings-restart.plugin", TypePlugin, ManifestBackend{Entry: "backend/plugin"})
	item.Status = StatusEnabled
	item.Manifest.Settings = []ManifestSetting{{Key: "name", Label: LocalizedText{Default: "Name"}, Type: "text"}}
	store := &fakeExtensionStore{
		items:    map[string]Extension{item.ID: item},
		settings: map[string]map[string]string{item.ID: {"name": "before"}},
	}
	runtime := &fakeRuntimeManager{startErr: errors.New("start failed")}
	service := NewServiceWithRuntime(store, t.TempDir(), runtime)

	_, err := service.UpdateSettings(context.Background(), extensionManager(), item.ID, UpdateSettingsInput{
		Values: map[string]string{"name": "after"},
	}, "zh-CN")
	if err == nil {
		t.Fatal("restart failure must be returned")
	}
	if got := store.settings[item.ID]["name"]; got != "before" {
		t.Fatalf("previous settings were not restored: %q", got)
	}
}

func TestResetSettingsRestoresSnapshotWhenPluginRestartFails(t *testing.T) {
	item := installedExtension("settings-reset.plugin", TypePlugin, ManifestBackend{Entry: "backend/plugin"})
	item.Status = StatusEnabled
	item.Manifest.Settings = []ManifestSetting{{Key: "name", Label: LocalizedText{Default: "Name"}, Type: "text"}}
	store := &fakeExtensionStore{
		items:    map[string]Extension{item.ID: item},
		settings: map[string]map[string]string{item.ID: {"name": "keep"}},
	}
	service := NewServiceWithRuntime(store, t.TempDir(), &fakeRuntimeManager{startErr: errors.New("start failed")})

	_, err := service.ResetSettings(context.Background(), extensionManager(), item.ID, "zh-CN")
	if err == nil {
		t.Fatal("restart failure must be returned")
	}
	if got := store.settings[item.ID]["name"]; got != "keep" {
		t.Fatalf("reset lost the previous settings: %q", got)
	}
}

func TestSettingsRollbackFailureReturnsDiagnosticError(t *testing.T) {
	item := installedExtension("settings-rollback.plugin", TypePlugin, ManifestBackend{Entry: "backend/plugin"})
	item.Status = StatusEnabled
	item.Manifest.Settings = []ManifestSetting{{Key: "name", Label: LocalizedText{Default: "Name"}, Type: "text"}}
	store := &fakeExtensionStore{
		items:        map[string]Extension{item.ID: item},
		settings:     map[string]map[string]string{item.ID: {"name": "before"}},
		replaceErrAt: 2,
		replaceErr:   errors.New("database unavailable"),
	}
	service := NewServiceWithRuntime(store, t.TempDir(), &fakeRuntimeManager{startErr: errors.New("start failed")})

	_, err := service.UpdateSettings(context.Background(), extensionManager(), item.ID, UpdateSettingsInput{
		Values: map[string]string{"name": "after"},
	}, "zh-CN")
	if !errors.Is(err, ErrSettingsRollbackFailed) {
		t.Fatalf("expected diagnostic rollback error, got %v", err)
	}
}

func TestServiceEnableRunsPluginPreflightBeforeStatusChange(t *testing.T) {
	expected := errors.New("rpc handshake failed")
	store := &fakeExtensionStore{items: map[string]Extension{
		"demo.plugin": withInstalledPackage(t, installedExtension("demo.plugin", TypePlugin, ManifestBackend{Entry: "backend/plugin"})),
	}}
	service := NewServiceWithHooks(store, t.TempDir(), fakeRuntime{err: expected})

	_, err := service.Enable(context.Background(), extensionManager(), "demo.plugin", EnableInput{ConfirmCapabilities: true})
	if !errors.Is(err, ErrPreflightFailed) {
		t.Fatalf("expected preflight failure, got %v", err)
	}
	if store.enabledID != "" {
		t.Fatalf("extension status changed despite failed preflight: %q", store.enabledID)
	}
	if last := store.events[len(store.events)-1]; last.Action != EventEnableFailed || last.Message == "" {
		t.Fatalf("expected enable failure event, got %#v", store.events)
	}

	service = NewServiceWithHooks(store, t.TempDir(), fakeRuntime{})
	enabled, err := service.Enable(context.Background(), extensionManager(), "demo.plugin", EnableInput{ConfirmCapabilities: true})
	if err != nil {
		t.Fatalf("Enable returned error: %v", err)
	}
	if enabled.Status != StatusEnabled || store.enabledID != "demo.plugin" {
		t.Fatalf("expected enabled plugin, got enabled=%#v store=%q", enabled, store.enabledID)
	}
}

func TestServiceEnableRejectsMissingInstalledPackage(t *testing.T) {
	missing := uploadedExtension("ghost.plugin", TypePlugin)
	missing.Manifest.Backend = ManifestBackend{}
	missing.PackagePath = filepath.Join(t.TempDir(), "ghost.plugin", "1.0.0", "package.zip")
	store := &fakeExtensionStore{items: map[string]Extension{
		missing.ID: missing,
	}}
	service := NewServiceWithRuntime(store, t.TempDir(), &fakeRuntimeManager{})

	_, err := service.Enable(context.Background(), extensionManager(), missing.ID, EnableInput{ConfirmCapabilities: true})
	if !errors.Is(err, ErrPreflightFailed) {
		t.Fatalf("expected missing package preflight failure, got %v", err)
	}
	if store.enabledID != "" {
		t.Fatalf("missing package should not enable extension, got %q", store.enabledID)
	}
	if last := store.events[len(store.events)-1]; last.Action != EventEnableFailed || last.Message == "" {
		t.Fatalf("expected enable failure event, got %#v", store.events)
	}
}
func TestServiceEnableRejectsTamperedDigestBackedPackage(t *testing.T) {
	tests := []struct {
		name   string
		tamper func(t *testing.T, snapshotRoot string)
	}{
		{
			name: "changed file bytes",
			tamper: func(t *testing.T, snapshotRoot string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(snapshotRoot, "README.md"), []byte("tampered"), 0o644); err != nil {
					t.Fatalf("tamper snapshot file: %v", err)
				}
			},
		},
		{
			name: "snapshot root replaced by symlink",
			tamper: func(t *testing.T, snapshotRoot string) {
				t.Helper()
				realRoot := snapshotRoot + ".real"
				if err := os.Rename(snapshotRoot, realRoot); err != nil {
					t.Fatalf("move snapshot root: %v", err)
				}
				if err := os.Symlink(realRoot, snapshotRoot); err != nil {
					t.Fatalf("replace snapshot root with symlink: %v", err)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &fakeExtensionStore{}
			runtime := &fakeRuntimeManager{}
			service := NewServiceWithRuntime(store, t.TempDir(), runtime)
			installed, err := service.InstallArchive(context.Background(), extensionManager(), ArchiveInput{
				FileName: "digest.zip",
				Data: extensionArchive(t, validManifest("digest.plugin", TypePlugin),
					zipFile{name: "README.md", body: "approved"},
				),
			})
			if err != nil {
				t.Fatalf("install digest-backed package: %v", err)
			}
			test.tamper(t, installed.PackagePath)

			_, err = service.Enable(context.Background(), extensionManager(), installed.ID, EnableInput{ConfirmCapabilities: true})
			if !errors.Is(err, ErrPreflightFailed) {
				t.Fatalf("expected tampered package preflight failure, got %v", err)
			}
			if store.enabledID != "" || len(runtime.started) != 0 {
				t.Fatalf("tampered package reached runtime: enabled=%q started=%#v", store.enabledID, runtime.started)
			}
		})
	}
}

func TestServiceEnableStartsRuntimeAndRollsBackOnStartFailure(t *testing.T) {
	expected := errors.New("bind failed")
	store := &fakeExtensionStore{items: map[string]Extension{
		"demo.plugin": withInstalledPackage(t, installedExtension("demo.plugin", TypePlugin, ManifestBackend{Entry: "backend/plugin"})),
	}}
	runtime := &fakeRuntimeManager{startErr: expected}
	service := NewServiceWithRuntime(store, t.TempDir(), runtime)

	_, err := service.Enable(context.Background(), extensionManager(), "demo.plugin", EnableInput{ConfirmCapabilities: true})
	if !errors.Is(err, ErrRuntimeFailed) {
		t.Fatalf("expected runtime failure, got %v", err)
	}
	if store.enabledID != "demo.plugin" || store.disabledID != "demo.plugin" {
		t.Fatalf("expected enable then rollback disable, enabled=%q disabled=%q", store.enabledID, store.disabledID)
	}
	if len(runtime.started) != 1 || runtime.started[0] != "demo.plugin" {
		t.Fatalf("expected runtime start attempt, got %#v", runtime.started)
	}
	if last := store.events[len(store.events)-1]; last.Action != EventEnableFailed || last.Message == "" {
		t.Fatalf("expected enable failure event, got %#v", store.events)
	}
}

func TestServiceDisableStopsRuntimeAndListDecoratesRuntimeStatus(t *testing.T) {
	store := &fakeExtensionStore{items: map[string]Extension{
		"demo.plugin": installedExtension("demo.plugin", TypePlugin, ManifestBackend{Entry: "backend/plugin"}),
	}}
	store.items["demo.plugin"] = extensionWithStatus(store.items["demo.plugin"], StatusEnabled)
	runtime := &fakeRuntimeManager{statuses: map[string]RuntimeStatus{
		"demo.plugin": {State: RuntimeRunning, RouteCount: 1, HookCount: 1, ProviderCount: 1},
	}}
	service := NewServiceWithRuntime(store, t.TempDir(), runtime)

	items, err := service.List(context.Background(), extensionManager())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if items[0].Runtime == nil || items[0].Runtime.State != RuntimeRunning {
		t.Fatalf("expected decorated runtime status, got %#v", items[0].Runtime)
	}

	_, err = service.Disable(context.Background(), extensionManager(), "demo.plugin")
	if err != nil {
		t.Fatalf("Disable returned error: %v", err)
	}
	if len(runtime.stopped) != 1 || runtime.stopped[0] != "demo.plugin" {
		t.Fatalf("expected runtime stop, got %#v", runtime.stopped)
	}
}

func TestServiceEmitsPluginLifecycleHooks(t *testing.T) {
	store := &fakeExtensionStore{items: map[string]Extension{
		"demo.plugin": withInstalledPackage(t, installedExtension("demo.plugin", TypePlugin, ManifestBackend{Entry: "backend/plugin"})),
	}}
	runtime := &fakeRuntimeManager{}
	service := NewServiceWithRuntime(store, t.TempDir(), runtime)

	if _, err := service.Enable(context.Background(), extensionManager(), "demo.plugin", EnableInput{ConfirmCapabilities: true}); err != nil {
		t.Fatalf("Enable returned error: %v", err)
	}
	if _, err := service.Disable(context.Background(), extensionManager(), "demo.plugin"); err != nil {
		t.Fatalf("Disable returned error: %v", err)
	}
	expected := []string{"extension.enabled", "extension.disabled"}
	if !slices.Equal(runtime.hooks, expected) {
		t.Fatalf("expected lifecycle hooks %#v, got %#v", expected, runtime.hooks)
	}
}

func TestServiceEnableRejectsThemesBecauseThemesUseActivation(t *testing.T) {
	store := &fakeExtensionStore{items: map[string]Extension{
		"starter.theme": installedExtension("starter.theme", TypeTheme, ManifestBackend{}),
	}}
	service := NewServiceWithHooks(store, t.TempDir(), nil)

	_, err := service.Enable(context.Background(), extensionManager(), "starter.theme", EnableInput{ConfirmCapabilities: true})
	if !errors.Is(err, ErrThemeActivationRequired) {
		t.Fatalf("expected theme activation requirement, got %v", err)
	}
	if store.enabledID != "" {
		t.Fatalf("theme should not be enabled through plugin lifecycle, got %q", store.enabledID)
	}
}

func TestServiceVerifyThemeMissingPackageReturnsBuildFailed(t *testing.T) {
	missing := uploadedExtension("ghost.theme", TypeTheme)
	missing.PackagePath = filepath.Join(t.TempDir(), "ghost.theme", "1.0.0", "package.zip")
	store := &fakeExtensionStore{items: map[string]Extension{
		missing.ID: missing,
	}}
	service := NewServiceWithHooks(store, t.TempDir(), nil)

	_, err := service.VerifyExtension(context.Background(), extensionManager(), missing.ID)
	if !errors.Is(err, ErrBuildFailed) {
		t.Fatalf("expected missing theme package build failure, got %v", err)
	}
	if last := store.events[len(store.events)-1]; last.Action != EventEnableFailed || last.Message == "" {
		t.Fatalf("expected verify failure event, got %#v", store.events)
	}
}

func TestServiceVerifyThemeMissingManifestReturnsBuildFailed(t *testing.T) {
	missing := uploadedExtension("manifestless.theme", TypeTheme)
	root := filepath.Join(t.TempDir(), missing.ID, missing.Version)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("create package root: %v", err)
	}
	missing.PackagePath = filepath.Join(root, "package.zip")
	if err := os.WriteFile(missing.PackagePath, []byte("zip"), 0o600); err != nil {
		t.Fatalf("write package archive: %v", err)
	}
	store := &fakeExtensionStore{items: map[string]Extension{
		missing.ID: missing,
	}}
	service := NewServiceWithHooks(store, t.TempDir(), nil)

	_, err := service.VerifyExtension(context.Background(), extensionManager(), missing.ID)
	if !errors.Is(err, ErrBuildFailed) {
		t.Fatalf("expected missing theme manifest build failure, got %v", err)
	}
	if last := store.events[len(store.events)-1]; last.Action != EventEnableFailed || last.Message == "" {
		t.Fatalf("expected verify failure event, got %#v", store.events)
	}
}

func TestServiceVerifyThemeMissingManifestLayerReturnsBuildFailed(t *testing.T) {
	// 无 layer 且无 theme.json/assets 时仍应失败。
	theme := withInstalledPackage(t, installedExtension("layerless.theme", TypeTheme, ManifestBackend{}))
	// 上传包：PackagePath 是 package.zip；manifest 在同级目录。
	root := filepath.Dir(theme.PackagePath)
	if err := writeManifest(root, theme.Manifest); err != nil {
		t.Fatalf("rewrite uploaded manifest: %v", err)
	}
	_ = os.Remove(filepath.Join(root, "theme.json"))
	_ = os.RemoveAll(filepath.Join(root, "assets"))
	_ = os.RemoveAll(filepath.Join(root, "files"))
	store := &fakeExtensionStore{items: map[string]Extension{
		theme.ID: theme,
	}}
	service := NewServiceWithHooks(store, t.TempDir(), nil)

	_, err := service.VerifyExtension(context.Background(), extensionManager(), theme.ID)
	if !errors.Is(err, ErrBuildFailed) {
		t.Fatalf("expected missing theme package build failure, got %v", err)
	}
	if last := store.events[len(store.events)-1]; last.Action != EventEnableFailed || last.Message == "" {
		t.Fatalf("expected verify failure event, got %#v", store.events)
	}
}

func TestServiceActivateThemeActivatesUploadedThemeImmediately(t *testing.T) {
	// Runtime Page Registry：上传主题同步激活，不触发 Nuxt 构建。
	store := newFakeExtensionStore(map[string]Extension{
		"starter.theme": withInstalledPackage(t, installedExtension("starter.theme", TypeTheme, ManifestBackend{})),
	})
	service := NewServiceWithOptions(store, t.TempDir(), "", LocalRuntimeManager{})

	active, err := service.ActivateTheme(context.Background(), extensionManager(), "starter.theme")
	if err != nil {
		t.Fatalf("ActivateTheme returned error: %v", err)
	}
	if active.Status != StatusEnabled || store.activeThemeID != "starter.theme" {
		t.Fatalf("expected uploaded theme active immediately, got active=%#v activeID=%q", active, store.activeThemeID)
	}
}

func TestServiceActivateThemeRestoresBuiltinDefaultThemeImmediately(t *testing.T) {
	store := &fakeExtensionStore{items: map[string]Extension{
		DefaultThemeID: withInstalledPackage(t, protectedBuiltinExtension(DefaultThemeID, TypeTheme)),
	}}
	service := NewServiceWithHooks(store, t.TempDir(), nil)

	active, err := service.ActivateTheme(context.Background(), extensionManager(), DefaultThemeID)
	if err != nil {
		t.Fatalf("ActivateTheme returned error: %v", err)
	}
	if active.Status != StatusEnabled || store.activeThemeID != DefaultThemeID {
		t.Fatalf("expected default theme active, got active=%#v activeID=%q", active, store.activeThemeID)
	}
	if last := store.events[len(store.events)-1]; last.Action != EventThemeActivated {
		t.Fatalf("expected theme activated event, got %#v", store.events)
	}
}

func TestServiceEnsureDefaultThemeActiveRepairsUnsafeThemeState(t *testing.T) {
	store := &fakeExtensionStore{items: map[string]Extension{
		DefaultThemeID: protectedBuiltinExtension(DefaultThemeID, TypeTheme),
		"starter.theme": {
			ID:      "starter.theme",
			Name:    "Starter Theme",
			Version: "1.0.0",
			Type:    TypeTheme,
			Status:  StatusEnabled,
			Source:  SourceUploaded,
			Manifest: Manifest{
				ID:            "starter.theme",
				Name:          "Starter Theme",
				Version:       "1.0.0",
				Type:          TypeTheme,
				SForumVersion: "^1.0.0",
			},
			InstalledAt: time.Now(),
			UpdatedAt:   time.Now(),
		},
	}}
	store.activeThemeID = "starter.theme"
	service := NewServiceWithHooks(store, t.TempDir(), nil)

	active, err := service.EnsureDefaultThemeActive(context.Background())
	if err != nil {
		t.Fatalf("EnsureDefaultThemeActive returned error: %v", err)
	}
	if active.ID != DefaultThemeID || store.activeThemeID != DefaultThemeID {
		t.Fatalf("expected unsafe active theme repaired to default, got active=%#v activeID=%q", active, store.activeThemeID)
	}

	active, err = service.EnsureDefaultThemeActive(context.Background())
	if err != nil {
		t.Fatalf("EnsureDefaultThemeActive second call returned error: %v", err)
	}
	if active.ID != DefaultThemeID || store.activeThemeID != DefaultThemeID {
		t.Fatalf("expected idempotent default active theme, got active=%#v activeID=%q", active, store.activeThemeID)
	}
}

func TestServiceDisableRequiresPermissionAndRecordsEvent(t *testing.T) {
	store := &fakeExtensionStore{items: map[string]Extension{
		"demo.plugin": installedExtension("demo.plugin", TypePlugin, ManifestBackend{}),
	}}
	service := NewService(store, t.TempDir())

	_, err := service.Disable(context.Background(), identity.Actor{ID: 9, Status: identity.UserStatusActive}, "demo.plugin")
	if !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}

	disabled, err := service.Disable(context.Background(), extensionManager(), "demo.plugin")
	if err != nil {
		t.Fatalf("Disable returned error: %v", err)
	}
	if disabled.Status != StatusDisabled || store.disabledID != "demo.plugin" {
		t.Fatalf("expected disabled extension, got disabled=%#v store=%q", disabled, store.disabledID)
	}
	if last := store.events[len(store.events)-1]; last.Action != EventDisabled {
		t.Fatalf("expected disabled event, got %#v", store.events)
	}
}

// extensionManager 成功路径默认用 super_admin：含后端入口的上传包仅其可装/启。
func extensionManager() identity.Actor {
	return identity.Actor{
		ID:       42,
		Status:   identity.UserStatusActive,
		RoleKeys: []string{identity.RoleSuperAdmin},
	}
}

// techAdminPluginManager 模拟 tech_admin：有 plugin.manage，但非 super_admin。
func techAdminPluginManager() identity.Actor {
	return identity.Actor{
		ID:     43,
		Status: identity.UserStatusActive,
		Permissions: map[string]bool{
			identity.PermissionExtensionPluginManage: true,
		},
	}
}

func validManifest(id string, extensionType string) string {
	return `{
		"id": "` + id + `",
		"name": "Demo Extension",
		"description": "Demo extension for SForum tests.",
		"url": "https://example.com/demo-extension",
		"author": {"name": "SForum Team", "url": "https://example.com", "email": "dev@example.com"},
		"version": "1.0.0",
		"type": "` + extensionType + `",
		"sforumVersion": "^1.0.0",
		"permissions": ["topic.create"],
		"settings": [{"key": "demo.enabled", "label": "Enabled", "type": "boolean"}],
		"migrations": [{"path": "migrations/001_init.sql"}],
		"backend": {"entry": "backend/plugin", "rpc": "hashicorp-go-plugin"},
		"adminPages": [{"path": "/demo", "label": "Demo", "permission": "extension.demo.manage"}],
		"routes": [{"path": "/hello", "methods": ["GET"]}],
		"hooks": [{"name": "topic.created"}],
		"jobs": [{"name": "demo.sync"}]
	}`
}

func validThemeManifest(id string) string {
	return `{
		"id": "` + id + `",
		"name": "Demo Theme",
		"description": "Demo theme for SForum tests.",
		"url": "https://example.com/demo-theme",
		"author": {"name": "SForum Team", "url": "https://example.com", "email": "dev@example.com"},
		"version": "1.0.0",
		"type": "theme",
		"sforumVersion": "^1.0.0"
	}`
}

type zipFile struct {
	name string
	body string
	mode os.FileMode
}

func extensionArchive(t *testing.T, manifest string, files ...zipFile) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	if manifest != "" {
		writeZipFile(t, writer, zipFile{name: ManifestFileName, body: manifest})
	}
	for _, file := range files {
		writeZipFile(t, writer, file)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buffer.Bytes()
}

func writeZipFile(t *testing.T, writer *zip.Writer, file zipFile) {
	t.Helper()
	header := &zip.FileHeader{Name: file.name, Method: zip.Deflate}
	if file.mode != 0 {
		header.SetMode(file.mode)
	}
	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatalf("create zip file %s: %v", file.name, err)
	}
	if _, err := io.WriteString(entry, file.body); err != nil {
		t.Fatalf("write zip file %s: %v", file.name, err)
	}
}

func withInstalledPackage(t *testing.T, item Extension) Extension {
	t.Helper()
	root := filepath.Join(t.TempDir(), item.ID, item.Version)
	if item.Source == SourceBuiltin {
		item.PackagePath = root
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatalf("create builtin package root: %v", err)
		}
		if err := writeManifest(root, item.Manifest); err != nil {
			t.Fatalf("write builtin manifest: %v", err)
		}
		writeTestThemeContract(t, root, item)
		return item
	}

	item.Source = SourceUploaded
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("create uploaded package root: %v", err)
	}
	item.PackagePath = filepath.Join(root, "package.zip")
	if err := os.WriteFile(item.PackagePath, []byte("zip"), 0o600); err != nil {
		t.Fatalf("write uploaded package archive: %v", err)
	}
	if err := writeManifest(root, item.Manifest); err != nil {
		t.Fatalf("write uploaded manifest: %v", err)
	}
	writeTestThemeContract(t, root, item)
	return item
}

func writeTestThemeContract(t *testing.T, root string, item Extension) {
	t.Helper()
	if item.Type != TypeTheme {
		return
	}
	if err := os.WriteFile(filepath.Join(root, "theme.json"), []byte(`{"schemaVersion":1,"styles":{"tokens":{}}}`), 0o644); err != nil {
		t.Fatalf("write theme contract: %v", err)
	}
}

func installedExtension(id string, extensionType string, backend ManifestBackend) Extension {
	return Extension{
		ID:      id,
		Name:    "Demo Extension",
		Version: "1.0.0",
		Type:    extensionType,
		Status:  StatusInstalled,
		Manifest: Manifest{
			ID:            id,
			Name:          "Demo Extension",
			Description:   "Demo extension for SForum tests.",
			URL:           "https://example.com/demo-extension",
			Author:        ManifestAuthor{Name: "SForum Team", URL: "https://example.com", Email: "dev@example.com"},
			Version:       "1.0.0",
			Type:          extensionType,
			SForumVersion: "^1.0.0",
			Backend:       backend,
		},
		PackagePath: "/tmp/demo.zip",
		InstalledAt: time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func protectedBuiltinExtension(id string, extensionType string) Extension {
	item := installedExtension(id, extensionType, ManifestBackend{})
	item.Status = StatusEnabled
	item.Source = SourceBuiltin
	item.IsSystem = true
	item.IsDeletable = false
	return item
}

func uploadedExtension(id string, extensionType string) Extension {
	item := installedExtension(id, extensionType, ManifestBackend{})
	item.Source = SourceUploaded
	item.IsSystem = false
	item.IsDeletable = true
	return item
}

func contributionTestPlugin(id string, status string, contributions []ManifestContribution) Extension {
	name := "Demo Plugin"
	if id == "alpha.plugin" {
		name = "Alpha Plugin"
	}
	if id == "beta.plugin" {
		name = "Beta Plugin"
	}
	return Extension{
		ID:      id,
		Name:    name,
		Version: "1.0.0",
		Type:    TypePlugin,
		Status:  status,
		Manifest: Manifest{
			ID:            id,
			Name:          name,
			Description:   "Contribution test plugin.",
			URL:           "https://example.com/" + id,
			Author:        ManifestAuthor{Name: "SForum Team"},
			Version:       "1.0.0",
			Type:          TypePlugin,
			SForumVersion: "^1.0.0",
			Contributions: contributions,
		},
		InstalledAt: time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func topicActionContribution(t *testing.T, id string, order int, routePath string) ManifestContribution {
	t.Helper()
	payload, err := json.Marshal(TopicActionContributionPayload{
		Type:   "extensionRoute",
		Method: "POST",
		Path:   routePath,
	})
	if err != nil {
		t.Fatalf("marshal topic action payload: %v", err)
	}
	return ManifestContribution{
		Point:   "forum.topic.actions",
		ID:      id,
		Order:   order,
		Label:   map[string]string{"zh-CN": "动作", "en-US": "Action"},
		Icon:    "i-lucide-bookmark",
		Payload: payload,
	}
}

func contributionIDs(items []EffectiveContribution) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ExtensionID+":"+item.ID)
	}
	return ids
}

func extensionWithStatus(item Extension, status string) Extension {
	item.Status = status
	return item
}

func navigationContains(items []ExtensionAdminNavigationItem, extensionID string, pagePath string) bool {
	for _, item := range items {
		if item.ExtensionID == extensionID && item.Path == pagePath {
			return true
		}
	}
	return false
}

func settingValue(settings ExtensionSettings, key string) string {
	for _, item := range settings.Items {
		if item.Key == key {
			return item.Value
		}
	}
	return ""
}

type fakeRuntime struct {
	err error
}

func (r fakeRuntime) Check(context.Context, Extension) error {
	return r.err
}

type fakeRuntimeManager struct {
	err      error
	startErr error
	started  []string
	stopped  []string
	hooks    []string
	statuses map[string]RuntimeStatus
}

func (r *fakeRuntimeManager) Check(context.Context, Extension) error {
	return r.err
}

func (r *fakeRuntimeManager) Start(_ context.Context, extension Extension) error {
	r.started = append(r.started, extension.ID)
	return r.startErr
}

func (r *fakeRuntimeManager) Stop(_ context.Context, extension Extension) error {
	r.stopped = append(r.stopped, extension.ID)
	return nil
}

func (r *fakeRuntimeManager) Status(_ context.Context, extension Extension) RuntimeStatus {
	if r.statuses != nil {
		if status, ok := r.statuses[extension.ID]; ok {
			return status
		}
	}
	return RuntimeStatus{State: RuntimeStopped}
}

func (r *fakeRuntimeManager) EmitHook(_ context.Context, name string, _ map[string]any) {
	r.hooks = append(r.hooks, name)
}

type fakeExtensionStore struct {
	items         map[string]Extension
	saved         Extension
	enabledID     string
	disabledID    string
	activeThemeID string
	settings      map[string]map[string]string
	replaceCalls  int
	replaceErrAt  int
	replaceErr    error
	beforeCAS     func()
	events        []ExtensionEvent
	deliveries    []ExtensionEventDelivery
	// migrations 按 extension_id 存账本（F2.4）。
	migrations map[string][]MigrationRecord
}

func newFakeExtensionStore(items map[string]Extension) *fakeExtensionStore {
	return &fakeExtensionStore{items: items}
}

func (s *fakeExtensionStore) List(context.Context) ([]Extension, error) {
	items := make([]Extension, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	return items, nil
}

func (s *fakeExtensionStore) Get(_ context.Context, id string) (Extension, error) {
	if item, ok := s.items[id]; ok {
		return item, nil
	}
	return Extension{}, ErrExtensionNotFound
}

func (s *fakeExtensionStore) SaveInstalled(_ context.Context, input SaveInstalledInput) (Extension, error) {
	item := Extension{
		ID:                  input.Manifest.ID,
		Name:                input.Manifest.Name,
		Version:             input.Manifest.Version,
		Type:                input.Manifest.Type,
		Status:              StatusInstalled,
		Source:              SourceUploaded,
		IsDeletable:         true,
		Manifest:            input.Manifest,
		PackageDigest:       input.PackageDigest,
		AdminFrontendDigest: input.AdminFrontendDigest,
		PackagePath:         input.PackagePath,
		InstalledAt:         time.Now(),
		UpdatedAt:           time.Now(),
	}
	s.saved = item
	if s.items == nil {
		s.items = map[string]Extension{}
	}
	s.items[item.ID] = item
	return item, nil
}

func (s *fakeExtensionStore) SaveBuiltin(_ context.Context, input SaveBuiltinInput) (Extension, error) {
	item := Extension{
		ID:                  input.Manifest.ID,
		Name:                input.Manifest.Name,
		Version:             input.Manifest.Version,
		Type:                input.Manifest.Type,
		Status:              StatusEnabled,
		Source:              SourceBuiltin,
		IsSystem:            true,
		IsDeletable:         false,
		Manifest:            input.Manifest,
		PackageDigest:       input.PackageDigest,
		AdminFrontendDigest: input.AdminFrontendDigest,
		PackagePath:         input.PackagePath,
		InstalledAt:         time.Now(),
		UpdatedAt:           time.Now(),
	}
	if s.items == nil {
		s.items = map[string]Extension{}
	}
	s.items[item.ID] = item
	return item, nil
}

func (s *fakeExtensionStore) PruneMissingBuiltins(_ context.Context, activeIDs []string) error {
	active := map[string]bool{}
	for _, id := range activeIDs {
		active[id] = true
	}
	for id, item := range s.items {
		if item.Source == SourceBuiltin && !active[id] {
			delete(s.items, id)
		}
	}
	return nil
}

func (s *fakeExtensionStore) Enable(_ context.Context, id string, extensionType string) (Extension, error) {
	item, ok := s.items[id]
	if !ok {
		return Extension{}, ErrExtensionNotFound
	}
	item.Status = StatusEnabled
	item.UpdatedAt = time.Now()
	s.items[id] = item
	s.enabledID = id
	if extensionType == TypeTheme {
		s.activeThemeID = id
	}
	return item, nil
}

func (s *fakeExtensionStore) ActivateTheme(_ context.Context, id string) (Extension, error) {
	item, ok := s.items[id]
	if !ok {
		return Extension{}, ErrExtensionNotFound
	}
	for currentID, current := range s.items {
		if current.Type == TypeTheme && current.ID != id && current.Status == StatusEnabled {
			current.Status = StatusDisabled
			current.UpdatedAt = time.Now()
			s.items[currentID] = current
		}
	}
	item.Status = StatusEnabled
	item.UpdatedAt = time.Now()
	s.items[id] = item
	s.activeThemeID = id
	return item, nil
}

func (s *fakeExtensionStore) ActiveTheme(context.Context) (Extension, error) {
	if s.activeThemeID != "" {
		if item, ok := s.items[s.activeThemeID]; ok {
			return item, nil
		}
	}
	for _, item := range s.items {
		if item.Type == TypeTheme && item.Status == StatusEnabled {
			return item, nil
		}
	}
	return Extension{}, ErrExtensionNotFound
}

func (s *fakeExtensionStore) Disable(_ context.Context, id string) (Extension, error) {
	item, ok := s.items[id]
	if !ok {
		return Extension{}, ErrExtensionNotFound
	}
	item.Status = StatusDisabled
	item.UpdatedAt = time.Now()
	s.items[id] = item
	s.disabledID = id
	if s.activeThemeID == id {
		s.activeThemeID = ""
	}
	return item, nil
}

func (s *fakeExtensionStore) CreateEvent(_ context.Context, input EventInput) (ExtensionEvent, error) {
	event := ExtensionEvent{
		ID:          int64(len(s.events) + 1),
		ExtensionID: input.ExtensionID,
		ActorUserID: input.ActorUserID,
		Action:      input.Action,
		Message:     input.Message,
		CreatedAt:   time.Now(),
	}
	s.events = append(s.events, event)
	return event, nil
}

func (s *fakeExtensionStore) ListEvents(context.Context, string, int) ([]ExtensionEvent, error) {
	return s.events, nil
}

func (s *fakeExtensionStore) ListSettings(_ context.Context, extensionID string) (map[string]string, error) {
	if s.settings == nil {
		return map[string]string{}, nil
	}
	values := map[string]string{}
	for key, value := range s.settings[extensionID] {
		values[key] = value
	}
	return values, nil
}

func (s *fakeExtensionStore) ReplaceSettings(_ context.Context, extensionID string, values map[string]string) error {
	s.replaceCalls++
	if s.replaceErrAt == s.replaceCalls {
		if s.replaceErr != nil {
			return s.replaceErr
		}
		return errors.New("replace settings failed")
	}
	if s.settings == nil {
		s.settings = map[string]map[string]string{}
	}
	next := map[string]string{}
	for key, value := range values {
		next[key] = value
	}
	s.settings[extensionID] = next
	return nil
}

func (s *fakeExtensionStore) CompareAndSwapSetting(_ context.Context, extensionID, name, oldValue, newValue string) (bool, error) {
	if s.beforeCAS != nil {
		before := s.beforeCAS
		s.beforeCAS = nil
		before()
	}
	if s.settings == nil || s.settings[extensionID][name] != oldValue {
		return false, nil
	}
	s.settings[extensionID][name] = newValue
	return true, nil
}

func (s *fakeExtensionStore) ResetSettings(_ context.Context, extensionID string) error {
	if s.settings != nil {
		delete(s.settings, extensionID)
	}
	return nil
}

func (s *fakeExtensionStore) Delete(_ context.Context, id string) error {
	if _, ok := s.items[id]; !ok {
		return ErrExtensionNotFound
	}
	delete(s.items, id)
	if s.settings != nil {
		delete(s.settings, id)
	}
	if s.migrations != nil {
		delete(s.migrations, id)
	}
	return nil
}

func (s *fakeExtensionStore) ListMigrationLedger(_ context.Context, extensionID string) ([]MigrationRecord, error) {
	if s.migrations == nil {
		return []MigrationRecord{}, nil
	}
	items := s.migrations[extensionID]
	if items == nil {
		return []MigrationRecord{}, nil
	}
	out := make([]MigrationRecord, len(items))
	copy(out, items)
	return out, nil
}

func (s *fakeExtensionStore) RecordMigration(_ context.Context, extensionID string, record MigrationRecord) error {
	if s.migrations == nil {
		s.migrations = map[string][]MigrationRecord{}
	}
	if record.AppliedAt.IsZero() {
		record.AppliedAt = time.Now()
	}
	list := s.migrations[extensionID]
	for i := range list {
		if list[i].Path == record.Path {
			list[i] = record
			s.migrations[extensionID] = list
			return nil
		}
	}
	s.migrations[extensionID] = append(list, record)
	return nil
}

func (s *fakeExtensionStore) CreateEventDelivery(_ context.Context, input EventDeliveryInput) (ExtensionEventDelivery, error) {
	delivery := ExtensionEventDelivery{
		ID:            int64(len(s.deliveries) + 1),
		ExtensionID:   input.ExtensionID,
		EventName:     input.EventName,
		EventKind:     input.EventKind,
		Status:        input.Status,
		Reason:        input.Reason,
		Message:       input.Message,
		CorrelationID: input.CorrelationID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	s.deliveries = append(s.deliveries, delivery)
	return delivery, nil
}

func (s *fakeExtensionStore) UpdateEventDelivery(_ context.Context, input EventDeliveryUpdateInput) error {
	for index := range s.deliveries {
		if s.deliveries[index].ID == input.ID {
			s.deliveries[index].Status = input.Status
			s.deliveries[index].Reason = input.Reason
			s.deliveries[index].Message = input.Message
			s.deliveries[index].AttemptCount = input.AttemptCount
			s.deliveries[index].UpdatedAt = time.Now()
			if input.Completed {
				completedAt := time.Now()
				s.deliveries[index].CompletedAt = &completedAt
			}
			return nil
		}
	}
	return ErrExtensionNotFound
}

func (s *fakeExtensionStore) ListEventDeliveries(_ context.Context, input EventDeliveryListInput) ([]ExtensionEventDelivery, error) {
	items := []ExtensionEventDelivery{}
	for _, delivery := range s.deliveries {
		if input.ExtensionID != "" && delivery.ExtensionID != input.ExtensionID {
			continue
		}
		if input.EventName != "" && delivery.EventName != input.EventName {
			continue
		}
		if input.Status != "" && delivery.Status != input.Status {
			continue
		}
		items = append(items, delivery)
	}
	return items, nil
}
