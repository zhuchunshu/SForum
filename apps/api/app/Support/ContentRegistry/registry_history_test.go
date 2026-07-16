package contentregistry

import (
	"errors"
	"reflect"
	"testing"
)

func TestRegistryAppendOnlyOwnershipStopsNestedNamespaceTakeover(t *testing.T) {
	registry := New()
	owner := publication("vendor.foo", false, 'a')
	owner.Content = []Declaration{
		content("vendor.foo.bar.block", KindBlock, "content.block", "vendor.foo.bar.block.schema@1"),
	}
	if _, err := registry.Publish(owner); err != nil {
		t.Fatal(err)
	}
	if _, removed, err := registry.Remove(owner.Artifact); err != nil || !removed {
		t.Fatalf("Remove() removed=%t error=%v", removed, err)
	}

	thief := publication("vendor.foo.bar", false, 'b')
	thief.Content = []Declaration{
		content("vendor.foo.bar.block", KindBlock, "content.stolen", "vendor.foo.bar.block.schema@1"),
	}
	before := registry.Snapshot()
	if _, err := registry.Publish(thief); !errors.Is(err, ErrConflict) {
		t.Fatalf("nested namespace takeover error = %v", err)
	}
	if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatal("rejected takeover mutated the snapshot")
	}
}

func TestRegistryDefinitionChangeRequiresContractVersionChange(t *testing.T) {
	registry := New()
	initial := publication("semantic.content", false, 'a')
	initial.Content = []Declaration{
		content("semantic.content.block.card", KindBlock, "content.card.v1", "semantic.content.block.card.schema@1"),
	}
	if _, err := registry.Publish(initial); err != nil {
		t.Fatal(err)
	}

	replacement := publication("semantic.content", false, 'b')
	replacement.Artifact.ExtensionVersion = "2.0.0"
	replacement.Artifact.VersionID = 2
	replacement.Artifact.RuntimeInstanceID = "runtime-semantic.content-v2"
	replacement.Content = []Declaration{
		content("semantic.content.block.card", KindBlock, "content.card.v2", "semantic.content.block.card.schema@1"),
	}
	if _, err := registry.PublishIfArtifact(initial.Artifact, replacement); !errors.Is(err, ErrConflict) {
		t.Fatalf("same-contract semantic change error = %v", err)
	}

	replacement.Content[0].ContractVersion = "semantic.content.block.card@2"
	if revision, err := registry.PublishIfArtifact(initial.Artifact, replacement); err != nil || revision != 2 {
		t.Fatalf("versioned replacement revision=%d error=%v", revision, err)
	}
	snapshot := registry.Snapshot()
	if len(snapshot.Tombstones) != 2 || len(snapshot.History) != 2 {
		t.Fatalf("append-only history = %#v / %#v", snapshot.Tombstones, snapshot.History)
	}
}

func TestRegistrySafeModePreservesExactPackageHistory(t *testing.T) {
	registry := New()
	plugin := publication("safe.content", false, 'a')
	plugin.Content = []Declaration{
		content("safe.content.block.card", KindBlock, "content.card", "safe.content.block.card.schema@1"),
	}
	if _, err := registry.Publish(plugin); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.ReplaceAllIfRevision(registry.Revision(), []Publication{plugin}, true); err != nil {
		t.Fatal(err)
	}
	safe := registry.Snapshot()
	if !safe.SafeMode || len(safe.Publications) != 0 || len(safe.Tombstones) != 1 || len(safe.History) != 1 {
		t.Fatalf("Safe Mode snapshot = %#v", safe)
	}

	drift := plugin
	drift.Content = append(drift.Content,
		content("safe.content.block.added", KindBlock, "content.added", "safe.content.block.added.schema@1"))
	if revision, err := registry.ReplaceAllIfRevision(safe.Revision, []Publication{drift}, false); !errors.Is(err, ErrArtifactConflict) || revision != safe.Revision {
		t.Fatalf("Safe Mode exact-package drift revision=%d error=%v", revision, err)
	}
	if after := registry.Snapshot(); !reflect.DeepEqual(safe, after) {
		t.Fatal("rejected Safe Mode exit mutated the snapshot")
	}
}

func TestRegistrySafeModeExitAcceptsTheExactRecordedPackage(t *testing.T) {
	registry := New()
	plugin := publication("safe.exit", false, 'a')
	plugin.Content = []Declaration{
		content("safe.exit.block.card", KindBlock, "content.card", "safe.exit.block.card.schema@1"),
	}
	if _, err := registry.Publish(plugin); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.ReplaceAllIfRevision(registry.Revision(), []Publication{plugin}, true); err != nil {
		t.Fatal(err)
	}
	safe := registry.Snapshot()
	if revision, err := registry.ReplaceAllIfRevision(safe.Revision, []Publication{plugin}, false); err != nil || revision != safe.Revision+1 {
		t.Fatalf("Safe Mode exit revision=%d error=%v", revision, err)
	}
	active := registry.Snapshot()
	if active.SafeMode || len(active.Publications) != 1 || len(active.Tombstones) != 1 || len(active.History) != 1 {
		t.Fatalf("active snapshot = %#v", active)
	}
}

func TestRegistryPartialUpgradeKeepsRemovedIdentityOwned(t *testing.T) {
	registry := New()
	initial := publication("partial.content", false, 'a')
	initial.Content = []Declaration{
		content("partial.content.block.kept", KindBlock, "content.kept", "partial.content.block.kept.schema@1"),
		content("partial.content.child.block", KindBlock, "content.retired", "partial.content.child.block.schema@1"),
	}
	if _, err := registry.Publish(initial); err != nil {
		t.Fatal(err)
	}
	replacement := publication("partial.content", false, 'b')
	replacement.Artifact.ExtensionVersion = "2.0.0"
	replacement.Artifact.VersionID = 2
	replacement.Artifact.RuntimeInstanceID = "runtime-partial-v2"
	replacement.Content = []Declaration{initial.Content[0]}
	if _, err := registry.PublishIfArtifact(initial.Artifact, replacement); err != nil {
		t.Fatal(err)
	}

	thief := publication("partial.content.child", false, 'c')
	thief.Content = []Declaration{
		content("partial.content.child.block", KindBlock, "content.stolen", "partial.content.child.block.schema@1"),
	}
	if _, err := registry.Publish(thief); !errors.Is(err, ErrConflict) {
		t.Fatalf("retired identity takeover error = %v", err)
	}
}

func TestRegistryRestoresDurableOwnershipAndPackageHistory(t *testing.T) {
	source := New()
	owner := publication("restore.content", false, 'a')
	owner.Content = []Declaration{
		content("restore.content.block.card", KindBlock, "content.card", "restore.content.block.card.schema@1"),
	}
	if _, err := source.Publish(owner); err != nil {
		t.Fatal(err)
	}
	if _, _, err := source.Remove(owner.Artifact); err != nil {
		t.Fatal(err)
	}
	durable := source.Snapshot()

	restored := New()
	if revision, err := restored.RestoreIfRevision(
		0, nil, durable.Tombstones, durable.History, false,
	); err != nil || revision != 1 {
		t.Fatalf("RestoreIfRevision() revision=%d error=%v", revision, err)
	}
	thief := publication("restore.content.block", false, 'b')
	thief.Content = []Declaration{
		content("restore.content.block.card", KindBlock, "content.stolen", "restore.content.block.card.schema@1"),
	}
	if _, err := restored.Publish(thief); !errors.Is(err, ErrConflict) {
		t.Fatalf("restored ownership takeover error = %v", err)
	}

	drift := owner
	drift.Content = append(drift.Content,
		content("restore.content.block.added", KindBlock, "content.added", "restore.content.block.added.schema@1"))
	if _, err := restored.Publish(drift); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("restored exact-package drift error = %v", err)
	}
}

func TestRegistryRestoreCannotEraseProcessLocalHistory(t *testing.T) {
	registry := New()
	plugin := publication("local.history", false, 'a')
	plugin.Content = []Declaration{
		content("local.history.block.card", KindBlock, "content.card", "local.history.block.card.schema@1"),
	}
	if _, err := registry.Publish(plugin); err != nil {
		t.Fatal(err)
	}
	if _, _, err := registry.Remove(plugin.Artifact); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()
	if revision, err := registry.RestoreIfRevision(before.Revision, nil, nil, nil, false); err != nil || revision != before.Revision {
		t.Fatalf("empty RestoreIfRevision() revision=%d error=%v", revision, err)
	}
	if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatal("empty restore erased process-local history")
	}
}

func TestRegistryRejectsOversizedDurableHistory(t *testing.T) {
	registry := New()
	if _, err := registry.RestoreIfRevision(
		0, nil, make([]Tombstone, maxTombstonesTotal+1), nil, false,
	); !errors.Is(err, ErrInvalid) {
		t.Fatalf("tombstone overflow error = %v", err)
	}
	if _, err := registry.RestoreIfRevision(
		0, nil, nil, make([]PublicationRecord, maxPublicationHistoryTotal+1), false,
	); !errors.Is(err, ErrInvalid) {
		t.Fatalf("publication history overflow error = %v", err)
	}
	if registry.Revision() != 0 {
		t.Fatalf("oversized restore changed revision = %d", registry.Revision())
	}
}

func TestRegistrySnapshotHistoryIsCopyIsolated(t *testing.T) {
	registry := New()
	plugin := publication("copy.content", false, 'a')
	plugin.Content = []Declaration{
		content("copy.content.block.card", KindBlock, "content.card", "copy.content.block.card.schema@1"),
	}
	if _, err := registry.Publish(plugin); err != nil {
		t.Fatal(err)
	}
	snapshot := registry.Snapshot()
	snapshot.Tombstones[0].OwnerExtensionID = "mutated.owner"
	snapshot.History[0].ContentDigest = "mutated"
	again := registry.Snapshot()
	if again.Tombstones[0].OwnerExtensionID != "copy.content" ||
		again.History[0].ContentDigest == "mutated" {
		t.Fatalf("history mutation leaked into registry = %#v", again)
	}
}
