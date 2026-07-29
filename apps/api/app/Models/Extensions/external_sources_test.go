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
