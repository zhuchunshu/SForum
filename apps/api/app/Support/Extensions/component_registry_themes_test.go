package extensionsruntime

import (
	"errors"
	"strings"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

func TestComponentRegistryThemeTransitionPublishesExactTargetAndRemovesSource(t *testing.T) {
	source := componentThemeL2Extension(t, "component.theme-source", 10)
	target := componentThemeL2Extension(t, "component.theme-target", 20)
	registry := NewComponentRegistry()
	if err := registry.ReplaceRuntime(source, componentPackageRuntimeInstanceID(source)); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()
	if err := registry.ValidateThemeTransition(&target, &source); err != nil {
		t.Fatalf("preflight exact theme transition: %v", err)
	}
	if after := registry.Snapshot(); after.Revision != before.Revision ||
		len(after.Contributions) != len(before.Contributions) {
		t.Fatalf("preflight mutated component snapshot: before=%#v after=%#v", before, after)
	}
	if err := registry.PublishThemeTransition(&target, &source, 41); err != nil {
		t.Fatalf("publish exact theme transition: %v", err)
	}
	if _, ok := registry.RuntimeSnapshot(source.ID); ok {
		t.Fatal("source theme component publication survived the switch")
	}
	runtime, ok := registry.RuntimeSnapshot(target.ID)
	if !ok || runtime.Extension.Version != target.Version ||
		runtime.Extension.PackageDigest != target.PackageDigest ||
		runtime.InstanceID != componentPackageRuntimeInstanceID(target) ||
		!strings.HasPrefix(runtime.InstanceID, "host-component-package:") {
		t.Fatalf("target runtime=%#v ok=%t", runtime, ok)
	}
	if !registry.AdmitPublicComponent(target, target.Manifest.Components[0]) {
		t.Fatal("new exact theme component was not admitted immediately")
	}
	if registry.AdmitPublicComponent(source, source.Manifest.Components[0]) {
		t.Fatal("removed source theme component remained admitted")
	}
}

func TestComponentRegistryThemeTransitionFailureIsAtomic(t *testing.T) {
	sourceID := "component.theme-owner"
	source := componentTestExtension(t, sourceID, extensions.TypeTheme, componentTestContribution(
		sourceID, "owned-slot", extensionmanifest.ComponentActionAdd, 0, "", "",
	))
	consumerID := "component.theme-consumer"
	consumer := componentTestExtension(t, consumerID, extensions.TypePlugin, componentTestContribution(
		consumerID, "before-owned-slot", extensionmanifest.ComponentActionBefore, 0,
		source.Manifest.Components[0].ID, source.Manifest.Components[0].ContractVersion,
	))
	consumer.Manifest.Dependencies = []extensions.ManifestDependency{{
		ID: source.ID, Version: "^1.0.0", Kind: "required",
	}}
	target := componentTestExtension(t, "component.theme-empty", extensions.TypeTheme)
	registry := NewComponentRegistry()
	if err := registry.ReplaceAll([]ComponentRuntimeSnapshot{
		{Extension: source, InstanceID: componentPackageRuntimeInstanceID(source)},
		{Extension: consumer, InstanceID: componentPackageRuntimeInstanceID(consumer)},
	}); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()
	if err := registry.ValidateThemeTransition(&target, &source); !errors.Is(err, ErrComponentRegistryConflict) {
		t.Fatalf("dependency-breaking preflight error=%v", err)
	}
	if err := registry.PublishThemeTransition(&target, &source, 42); !errors.Is(err, ErrComponentRegistryConflict) {
		t.Fatalf("dependency-breaking publish error=%v", err)
	}
	after := registry.Snapshot()
	if after.Revision != before.Revision || len(after.Targets) != len(before.Targets) ||
		len(after.Contributions) != len(before.Contributions) || registry.themePublicationRevision != 0 {
		t.Fatalf("failed transition changed snapshot: before=%#v after=%#v themeRevision=%d", before, after, registry.themePublicationRevision)
	}
	if snapshot, ok := registry.RuntimeSnapshot(source.ID); !ok ||
		snapshot.InstanceID != componentPackageRuntimeInstanceID(source) {
		t.Fatalf("failed transition lost source runtime=%#v ok=%t", snapshot, ok)
	}
}

func TestComponentRegistryThemeRollbackAllowsRetryAndFencesStalePublication(t *testing.T) {
	source := componentThemeL2Extension(t, "component.theme-retry-source", 10)
	target := componentThemeL2Extension(t, "component.theme-retry-target", 20)
	newest := componentThemeL2Extension(t, "component.theme-newest", 30)
	registry := NewComponentRegistry()
	if err := registry.ReplaceRuntime(source, componentPackageRuntimeInstanceID(source)); err != nil {
		t.Fatal(err)
	}
	if err := registry.PublishThemeTransition(&target, &source, 50); err != nil {
		t.Fatal(err)
	}
	if err := registry.RollbackThemeTransition(&source, &target, 50); err != nil {
		t.Fatalf("rollback failed Page publication: %v", err)
	}
	if snapshot, ok := registry.RuntimeSnapshot(source.ID); !ok ||
		snapshot.InstanceID != componentPackageRuntimeInstanceID(source) {
		t.Fatalf("rolled-back source runtime=%#v ok=%t", snapshot, ok)
	}
	if err := registry.PublishThemeTransition(&target, &source, 50); err != nil {
		t.Fatalf("retry same durable publication: %v", err)
	}
	if err := registry.PublishThemeTransition(&newest, &target, 51); err != nil {
		t.Fatalf("publish newer exact artifact: %v", err)
	}

	if err := registry.RollbackThemeTransition(&source, &target, 50); !errors.Is(err, ErrComponentRegistryRevisionConflict) {
		t.Fatalf("stale rollback error=%v", err)
	}
	if err := registry.PublishThemeTransition(&source, &target, 50); !errors.Is(err, ErrComponentRegistryRevisionConflict) {
		t.Fatalf("stale publication error=%v", err)
	}
	if snapshot, ok := registry.RuntimeSnapshot(newest.ID); !ok ||
		snapshot.InstanceID != componentPackageRuntimeInstanceID(newest) {
		t.Fatalf("stale operation revoked newest runtime=%#v ok=%t", snapshot, ok)
	}
	if !registry.AdmitPublicComponent(newest, newest.Manifest.Components[0]) {
		t.Fatal("newest exact artifact lost admission after stale operations")
	}
}

func TestComponentRegistryThemeRollbackRestoresActualPrepublicationSnapshot(t *testing.T) {
	source := componentThemeL2Extension(t, "component.theme-durable-source", 10)
	target := componentThemeL2Extension(t, "component.theme-already-restored", 20)
	registry := NewComponentRegistry()
	// Startup restore may already reflect the durable target before the watcher
	// idempotently reapplies its latest publication.
	if err := registry.ReplaceRuntime(target, componentPackageRuntimeInstanceID(target)); err != nil {
		t.Fatal(err)
	}
	if err := registry.PublishThemeTransition(&target, &source, 61); err != nil {
		t.Fatal(err)
	}
	if err := registry.RollbackThemeTransition(&source, &target, 61); err != nil {
		t.Fatal(err)
	}
	if _, ok := registry.RuntimeSnapshot(source.ID); ok {
		t.Fatal("rollback invented the durable source instead of restoring the local snapshot")
	}
	if snapshot, ok := registry.RuntimeSnapshot(target.ID); !ok ||
		snapshot.InstanceID != componentPackageRuntimeInstanceID(target) {
		t.Fatalf("prepublication target snapshot=%#v ok=%t", snapshot, ok)
	}
}

func componentThemeL2Extension(t *testing.T, id string, priority int) extensions.Extension {
	t.Helper()
	declaration := componentTestContribution(
		id, "browser", extensionmanifest.ComponentActionReplace, priority,
		componentTestCoreTarget, componentTestCoreContract,
	)
	extension := componentTestExtension(t, id, extensions.TypeTheme, declaration)
	entryID := id + ".file.browser"
	extension.Manifest.Components[0].L2Component = entryID
	extension.Manifest.PackageFiles = append(extension.Manifest.PackageFiles, extensions.ManifestPackageFile{
		ID: entryID, Kind: "frontend", Path: "frontend/public/browser.mjs", Digest: strings.Repeat("9", 64),
	})
	return extension
}
