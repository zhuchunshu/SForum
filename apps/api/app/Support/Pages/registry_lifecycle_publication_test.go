package pages

import (
	"errors"
	"testing"
)

func TestLifecyclePagePublicationUsesExactRevisionAndRuntimeAdmission(t *testing.T) {
	registry := NewRegistry(nil)
	visible := false
	registry.WithRuntimeAdmission(func(artifact RuntimeArtifact) bool {
		return visible && artifact.RuntimeInstanceID == "runtime-2"
	})
	artifact := RuntimeArtifact{
		ExtensionID: "demo.pages", ExtensionVersion: "2.0.0",
		PackageDigest: "digest-2", RuntimeInstanceID: "runtime-2",
	}
	if _, err := registry.PublishExtensionIfRevision(artifact, []PageContribution{{
		ID: "demo.pages.docs", Action: ActionAdd, Path: "/lifecycle-docs",
	}}, 0); err != nil {
		t.Fatal(err)
	}
	if _, ok := registry.ResolveAddedPath("/lifecycle-docs"); ok {
		t.Fatal("drained target page became visible before runtime admission")
	}
	visible = true
	if page, ok := registry.ResolveAddedPath("/lifecycle-docs"); !ok || page.RuntimeInstanceID != "runtime-2" {
		t.Fatalf("admitted target page = %#v, %t", page, ok)
	}
	if _, err := registry.PublishExtensionIfRevision(artifact, nil, 0); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale revision = %v", err)
	}
	stale := artifact
	stale.RuntimeInstanceID = "runtime-1"
	if _, err := registry.RemoveExtensionIfRevision("demo.pages", stale, registry.Revision()); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("stale runtime removal = %v", err)
	}
}

func TestLifecyclePagePublicationRetainsExactEmptySet(t *testing.T) {
	registry := NewRegistry(nil)
	artifact := RuntimeArtifact{
		ExtensionID: "demo.empty", ExtensionVersion: "1.0.0",
		PackageDigest: "digest-empty", RuntimeInstanceID: "runtime-empty",
	}
	if _, err := registry.PublishExtensionIfRevision(artifact, nil, 0); err != nil {
		t.Fatal(err)
	}
	snapshot, ok := registry.ExtensionSnapshot(artifact.ExtensionID)
	if !ok || snapshot.Artifact != artifact || len(snapshot.Contributions) != 0 {
		t.Fatalf("empty exact page set = %#v, %t", snapshot, ok)
	}
}

func TestLegacyEmptyPagePublicationClearsWithoutBlankArtifact(t *testing.T) {
	registry := NewRegistry(nil)
	if err := registry.RegisterContributions("demo.empty", []PageContribution{{
		ID: "demo.empty.docs", Action: ActionAdd, Path: "/legacy-docs",
		Version: "1.0.0", PackageDigest: "digest-legacy",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterContributions("demo.empty", nil); err != nil {
		t.Fatal(err)
	}
	if snapshot, exists := registry.ExtensionSnapshot("demo.empty"); exists {
		t.Fatalf("legacy empty publication retained blank artifact: %#v", snapshot)
	}
	if _, exists := registry.ResolveAddedPath("/legacy-docs"); exists {
		t.Fatal("legacy empty publication retained old page contribution")
	}
}
