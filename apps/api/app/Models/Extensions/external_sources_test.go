package extensions

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncExternalSourcesInstallsInertSnapshotAndIsIdempotent(t *testing.T) {
	sourceRoot := t.TempDir()
	packageRoot := writeExternalTestPackage(t, sourceRoot, "demo.external", TypePlugin, "first")
	destination := t.TempDir()
	store := &fakeExtensionStore{}
	service := NewServiceWithOptions(store, destination, "", nil, WithExternalExtensionRoots([]string{sourceRoot}))

	result, err := service.SyncExternalSources(context.Background())
	if err != nil {
		t.Fatalf("sync external sources: %v", err)
	}
	if len(result.Diagnostics) != 0 || len(result.Items) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	installed := result.Items[0]
	if installed.Source != SourceUploaded || installed.Status != StatusInstalled || !installed.IsDeletable {
		t.Fatalf("external package must remain inert uploaded software: %+v", installed)
	}
	if installed.PackagePath == packageRoot || !strings.HasPrefix(filepath.Clean(installed.PackagePath), filepath.Clean(destination)+string(os.PathSeparator)) {
		t.Fatalf("package was not snapshotted into EXTENSION_ROOT: source=%q installed=%q", packageRoot, installed.PackagePath)
	}
	if body, err := os.ReadFile(filepath.Join(installed.PackagePath, "README.md")); err != nil || string(body) != "first" {
		t.Fatalf("snapshotted body=%q err=%v", body, err)
	}
	if len(store.events) != 1 || store.events[0].Action != EventInstalled {
		t.Fatalf("expected one install event: %+v", store.events)
	}

	result, err = service.SyncExternalSources(context.Background())
	if err != nil || len(result.Items) != 1 || len(result.Diagnostics) != 0 {
		t.Fatalf("repeat sync result=%+v err=%v", result, err)
	}
	if len(store.events) != 1 {
		t.Fatalf("unchanged source must not append events: %+v", store.events)
	}
}

func TestSyncExternalSourcesStagesChangedEnabledPackage(t *testing.T) {
	sourceRoot := t.TempDir()
	packageRoot := writeExternalTestPackage(t, sourceRoot, "demo.external", TypePlugin, "first")
	store := &fakeExtensionStore{}
	service := NewServiceWithOptions(store, t.TempDir(), "", nil, WithExternalExtensionRoots([]string{sourceRoot}))
	if _, err := service.SyncExternalSources(context.Background()); err != nil {
		t.Fatalf("initial sync: %v", err)
	}
	active := store.items["demo.external"]
	active.Status = StatusEnabled
	store.items[active.ID] = active
	if err := os.WriteFile(filepath.Join(packageRoot, "README.md"), []byte("second"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := service.SyncExternalSources(context.Background())
	if err != nil || len(result.Diagnostics) != 0 || len(result.Items) != 1 {
		t.Fatalf("changed sync result=%+v err=%v", result, err)
	}
	after := store.items[active.ID]
	if after.Status != StatusEnabled || after.PackageDigest != active.PackageDigest || after.StagedVersion == nil {
		t.Fatalf("changed source replaced active artifact instead of staging: before=%+v after=%+v", active, after)
	}
	if after.StagedVersion.PackageDigest == active.PackageDigest {
		t.Fatal("staged digest must differ from active digest")
	}
	if len(store.events) != 2 || store.events[1].Action != EventUpgraded {
		t.Fatalf("expected staged upgrade event: %+v", store.events)
	}
}

// 制品目录被人工删除后必须能自愈：快照发布本身会在内容寻址路径重建并校验目录，
// 幂等判定只要认这条路径，就不再需要换 digest 或手工改库。
func TestSyncExternalSourcesRepublishesDeletedSnapshot(t *testing.T) {
	sourceRoot := t.TempDir()
	writeExternalTestPackage(t, sourceRoot, "demo.external", TypePlugin, "first")
	store := &fakeExtensionStore{}
	service := NewServiceWithOptions(store, t.TempDir(), "", nil, WithExternalExtensionRoots([]string{sourceRoot}))

	if _, err := service.SyncExternalSources(context.Background()); err != nil {
		t.Fatalf("initial sync: %v", err)
	}
	installed := store.items["demo.external"]
	if err := os.RemoveAll(installed.PackagePath); err != nil {
		t.Fatal(err)
	}
	if artifactState(installed.PackagePath) != ArtifactMissing {
		t.Fatal("fixture must start from a deleted artifact directory")
	}
	eventsBefore := len(store.events)

	result, err := service.SyncExternalSources(context.Background())
	if err != nil || len(result.Diagnostics) != 0 || len(result.Items) != 1 {
		t.Fatalf("self-heal sync result=%+v err=%v", result, err)
	}
	restored := store.items["demo.external"]
	if restored.PackagePath != installed.PackagePath {
		t.Fatalf("content-addressed path changed: before=%q after=%q", installed.PackagePath, restored.PackagePath)
	}
	if artifactState(restored.PackagePath) != ArtifactAvailable {
		t.Fatalf("artifact was not re-published at %q", restored.PackagePath)
	}
	if body, err := os.ReadFile(filepath.Join(restored.PackagePath, "README.md")); err != nil || string(body) != "first" {
		t.Fatalf("re-published body=%q err=%v", body, err)
	}
	// 同一路径重建是静默自愈：内容没变，不写新台账。
	if len(store.events) != eventsBefore {
		t.Fatalf("quiet self-heal must not append events: %+v", store.events[eventsBefore:])
	}

	if _, err := service.SyncExternalSources(context.Background()); err != nil {
		t.Fatalf("stable resync: %v", err)
	}
	if len(store.events) != eventsBefore {
		t.Fatalf("steady state must stay quiet: %+v", store.events[eventsBefore:])
	}
}

// 记录路径漂移（旧 EXTENSION_ROOT / 手工改库 / 目录搬迁）等价于「行在、盘缺」：
// 版本与摘要都没变，但记录指向别处。这次必须落盘到当前内容寻址根并修好行。
func TestSyncExternalSourcesRepairsRelocatedSnapshotPath(t *testing.T) {
	sourceRoot := t.TempDir()
	writeExternalTestPackage(t, sourceRoot, "demo.external", TypePlugin, "first")
	destination := t.TempDir()
	store := &fakeExtensionStore{}
	service := NewServiceWithOptions(store, destination, "", nil, WithExternalExtensionRoots([]string{sourceRoot}))

	if _, err := service.SyncExternalSources(context.Background()); err != nil {
		t.Fatalf("initial sync: %v", err)
	}
	canonical := store.items["demo.external"].PackagePath
	stale := filepath.Join(t.TempDir(), "old-root", "demo.external")

	drifted := store.items["demo.external"]
	drifted.PackagePath = stale
	store.items[drifted.ID] = drifted
	eventsBefore := len(store.events)

	if _, err := service.SyncExternalSources(context.Background()); err != nil {
		t.Fatalf("repair sync: %v", err)
	}
	repaired := store.items["demo.external"]
	if repaired.PackagePath != canonical {
		t.Fatalf("recorded path was not repaired: got %q want %q", repaired.PackagePath, canonical)
	}
	if len(store.events) != eventsBefore+1 || store.events[eventsBefore].Action != EventVerified {
		t.Fatalf("expected one re-publish event: %+v", store.events[eventsBefore:])
	}
	if message := store.events[eventsBefore].Message; !strings.Contains(message, "re-published") {
		t.Fatalf("re-publish event message=%q", message)
	}

	// staged 记录的路径漂移同样要修：dev auto 只按 staged 摘要升级，路径错了
	// 就永久 missing。
	if _, err := service.SyncExternalSources(context.Background()); err != nil {
		t.Fatalf("stable resync: %v", err)
	}
}

// staged 记录漂移是 dev auto 最常撞到的形态：升级只看 staged 摘要，路径一错
// 就永远 missing。这一例把源码推进到新摘要，再破坏 staged 路径，验证修回。
func TestSyncExternalSourcesRepairsStagedSnapshotPath(t *testing.T) {
	sourceRoot := t.TempDir()
	packageRoot := writeExternalTestPackage(t, sourceRoot, "demo.external", TypePlugin, "first")
	store := &fakeExtensionStore{}
	service := NewServiceWithOptions(store, t.TempDir(), "", nil, WithExternalExtensionRoots([]string{sourceRoot}))

	if _, err := service.SyncExternalSources(context.Background()); err != nil {
		t.Fatalf("initial sync: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, "README.md"), []byte("second"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SyncExternalSources(context.Background()); err != nil {
		t.Fatalf("staged sync: %v", err)
	}
	staged := store.items["demo.external"].StagedVersion
	if staged == nil {
		t.Fatal("changed source must stage an immutable candidate")
	}
	canonical := staged.PackagePath
	staged.PackagePath = filepath.Join(t.TempDir(), "old-root", "demo.external")
	eventsBefore := len(store.events)

	if _, err := service.SyncExternalSources(context.Background()); err != nil {
		t.Fatalf("staged repair sync: %v", err)
	}
	repaired := store.items["demo.external"].StagedVersion
	if repaired == nil || repaired.ID != staged.ID || repaired.PackagePath != canonical {
		t.Fatalf("staged recorded path was not repaired: %+v", repaired)
	}
	if len(store.events) != eventsBefore+1 || store.events[eventsBefore].Action != EventVerified {
		t.Fatalf("expected one re-publish event: %+v", store.events[eventsBefore:])
	}
}

func TestSyncExternalSourcesRejectsDuplicateIDsAcrossRoots(t *testing.T) {
	firstRoot := t.TempDir()
	secondRoot := t.TempDir()
	writeExternalTestPackage(t, firstRoot, "duplicate.external", TypePlugin, "first")
	writeExternalTestPackage(t, secondRoot, "duplicate.external", TypePlugin, "second")
	store := &fakeExtensionStore{}
	service := NewServiceWithOptions(store, t.TempDir(), "", nil,
		WithExternalExtensionRoots([]string{firstRoot, secondRoot}))

	result, err := service.SyncExternalSources(context.Background())
	if err != nil {
		t.Fatalf("sync duplicates: %v", err)
	}
	if len(result.Items) != 0 || len(result.Diagnostics) != 2 {
		t.Fatalf("duplicates must all be rejected: %+v", result)
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code != ExternalDiagnosticIDConflict {
			t.Fatalf("unexpected diagnostic: %+v", diagnostic)
		}
	}
	if len(store.items) != 0 {
		t.Fatalf("duplicate source mutated store: %+v", store.items)
	}
}

func TestSyncExternalSourcesCannotReplaceBuiltinAndReportsMissingRoot(t *testing.T) {
	sourceRoot := t.TempDir()
	writeExternalTestPackage(t, sourceRoot, "builtin.external", TypePlugin, "external")
	store := newFakeExtensionStore(map[string]Extension{
		"builtin.external": {
			ID: "builtin.external", Type: TypePlugin, Status: StatusEnabled,
			Source: SourceBuiltin, IsSystem: true,
		},
	})
	missing := filepath.Join(t.TempDir(), "missing")
	service := NewServiceWithOptions(store, t.TempDir(), "", nil,
		WithExternalExtensionRoots([]string{sourceRoot, missing}))

	result, err := service.SyncExternalSources(context.Background())
	if err != nil {
		t.Fatalf("sync conflicts: %v", err)
	}
	if len(result.Items) != 0 || len(result.Diagnostics) != 2 {
		t.Fatalf("unexpected diagnostics: %+v", result)
	}
	codes := map[string]bool{}
	for _, diagnostic := range result.Diagnostics {
		codes[diagnostic.Code] = true
	}
	if !codes[ExternalDiagnosticBuiltinConflict] || !codes[ExternalDiagnosticRootUnavailable] {
		t.Fatalf("missing expected diagnostics: %+v", result.Diagnostics)
	}
}

func writeExternalTestPackage(t *testing.T, root, id, extensionType, readme string) string {
	t.Helper()
	group := "plugins"
	if extensionType == TypeTheme {
		group = "themes"
	}
	packageRoot := filepath.Join(root, group, id)
	if err := os.MkdirAll(packageRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, ManifestFileName), []byte(validManifest(id, extensionType)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(packageRoot, "backend"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, "backend", "plugin"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(packageRoot, "migrations"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, "migrations", "001_init.sql"), []byte("SELECT 1;"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, "README.md"), []byte(readme), 0o644); err != nil {
		t.Fatal(err)
	}
	return packageRoot
}
