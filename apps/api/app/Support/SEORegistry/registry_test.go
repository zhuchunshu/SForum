package seoregistry

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestRegistryPublishesDeterministicImmutableSnapshotsAndConflicts(t *testing.T) {
	first := testPublication("plugin.alpha", 'a')
	first.Contributions = []Declaration{
		testDeclaration(first, "canonical", GlobalScope, KindCanonical, ActionReplace, FailurePolicyFallback, 10),
		testDeclaration(first, "title", "core.page.topic", KindTitle, ActionAdd, FailurePolicyFailClosed, 5),
	}
	second := testPublication("plugin.beta", 'b')
	second.Contributions = []Declaration{
		testDeclaration(second, "canonical", "core.page.topic", KindCanonical, ActionReplace, FailurePolicyFailClosed, 10),
		testDeclaration(second, "title", "core.page.topic", KindTitle, ActionAdd, FailurePolicyFallback, 1),
	}

	left := New()
	leftRevision, err := left.ReplaceAll([]Publication{second, first}, false)
	if err != nil {
		t.Fatal(err)
	}
	right := New()
	if _, err := right.ReplaceAll([]Publication{first, second}, false); err != nil {
		t.Fatal(err)
	}
	leftSnapshot, rightSnapshot := left.Snapshot(), right.Snapshot()
	if leftRevision != 1 || leftSnapshot.Digest != rightSnapshot.Digest ||
		!reflect.DeepEqual(leftSnapshot.Publications, rightSnapshot.Publications) {
		t.Fatalf("nondeterministic snapshots: left=%#v right=%#v", leftSnapshot, rightSnapshot)
	}
	inspection, err := left.Inspect("core.page.topic")
	if err != nil {
		t.Fatal(err)
	}
	if len(inspection.Conflicts) != 2 {
		t.Fatalf("effective conflicts=%#v", inspection.Conflicts)
	}
	for _, conflict := range inspection.Conflicts {
		if conflict.Kind == KindCanonical && conflict.Winner.Artifact != second.Artifact {
			t.Fatalf("exact scope did not win equal-priority canonical conflict=%#v", conflict)
		}
		if conflict.Kind == KindTitle && conflict.Winner.Artifact != first.Artifact {
			t.Fatalf("higher priority did not win title conflict=%#v", conflict)
		}
	}

	first.Contributions[0].Handler = "forged.handler"
	leftSnapshot.Publications[0].Contributions[0].Handler = "forged.snapshot"
	leftSnapshot.Conflicts[0].Candidates[0].Handler = "forged.conflict"
	fresh := left.Snapshot()
	if strings.Contains(fresh.Publications[0].Contributions[0].Handler, "forged") ||
		strings.Contains(fresh.Conflicts[0].Candidates[0].Handler, "forged") {
		t.Fatalf("caller mutation escaped immutable snapshot=%#v", fresh)
	}
}

func TestRegistryExactArtifactCASAndRemoval(t *testing.T) {
	initial := testPublication("plugin.cas", 'a')
	initial.Contributions = []Declaration{
		testDeclaration(initial, "title", "core.page.topic", KindTitle, ActionFilter, FailurePolicyFailClosed, 0),
	}
	registry := New()
	if revision, err := registry.Publish(initial); err != nil || revision != 1 {
		t.Fatalf("publish revision=%d err=%v", revision, err)
	}
	if revision, err := registry.Publish(initial); err != nil || revision != 1 {
		t.Fatalf("idempotent revision=%d err=%v", revision, err)
	}
	mutated := initial
	mutated.Contributions = append([]Declaration(nil), initial.Contributions...)
	mutated.Contributions[0].Priority = 1
	if _, err := registry.Publish(mutated); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("same artifact declaration mutation=%v", err)
	}

	replacement := testPublication("plugin.cas", 'b')
	replacement.Artifact.ExtensionVersion = "2.0.0"
	replacement.Artifact.VersionID = 2
	replacement.Artifact.RuntimeInstanceID = "runtime-plugin-cas-v2"
	replacement.Contributions = []Declaration{
		testDeclaration(replacement, "title", "core.page.topic", KindTitle, ActionFilter, FailurePolicyFailClosed, 0),
	}
	if _, err := registry.Publish(replacement); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("unfenced replacement=%v", err)
	}
	if revision, err := registry.PublishIfArtifact(initial.Artifact, replacement); err != nil || revision != 2 {
		t.Fatalf("exact replacement revision=%d err=%v", revision, err)
	}
	if _, _, err := registry.Remove(initial.Artifact); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("stale remove=%v", err)
	}
	if revision, removed, err := registry.Remove(replacement.Artifact); err != nil || !removed || revision != 3 {
		t.Fatalf("exact remove revision=%d removed=%t err=%v", revision, removed, err)
	}
}

func TestRegistrySafeModeRetainsSealedCoreAndRequiresExplicitThirdPartyRepublish(t *testing.T) {
	publication := testPublication("plugin.safe", 'a')
	publication.Contributions = []Declaration{
		testDeclaration(publication, "title", "core.page.topic", KindTitle, ActionAdd, FailurePolicyFailClosed, 0),
	}
	registry := New()
	if _, err := registry.Publish(publication); err != nil {
		t.Fatal(err)
	}
	coreArtifact, err := NewCoreArtifact("core.seo", "1.0.0", strings.Repeat("c", 64), strings.Repeat("d", 64))
	if err != nil {
		t.Fatal(err)
	}
	core := Publication{Artifact: coreArtifact}
	core.Contributions = []Declaration{
		testDeclaration(core, "title", "core.page.topic", KindTitle, ActionAdd, FailurePolicyFailClosed, 0),
	}
	// Invalid third-party input is filtered before parsing and cannot block recovery.
	if _, err := registry.ReplaceAll([]Publication{core, {}}, true); err != nil {
		t.Fatal(err)
	}
	snapshot := registry.Snapshot()
	if !snapshot.SafeMode || len(snapshot.Publications) != 1 || snapshot.Publications[0].Artifact != coreArtifact ||
		len(snapshot.Contributions) != 1 || snapshot.Contributions[0].Artifact != coreArtifact {
		t.Fatalf("safe mode snapshot=%#v", snapshot)
	}
	if _, err := registry.Publish(publication); !errors.Is(err, ErrSafeMode) {
		t.Fatalf("safe mode publish=%v", err)
	}
	if _, err := registry.Publish(core); err != nil {
		t.Fatalf("safe mode core replay=%v", err)
	}
	if _, err := registry.ReplaceAll(nil, false); err != nil {
		t.Fatal(err)
	}
	if len(registry.Snapshot().Contributions) != 0 {
		t.Fatal("disabled Safe Mode resurrected hidden publication")
	}
}

func TestRegistryRejectsDeclarationDriftAfterRemoveAndRuntimeRestart(t *testing.T) {
	publication := testPublication("plugin.replay-history", 'a')
	publication.Contributions = []Declaration{
		testDeclaration(publication, "title", "core.page.topic", KindTitle, ActionAdd, FailurePolicyFailClosed, 0),
	}
	registry := New()
	if _, err := registry.Publish(publication); err != nil {
		t.Fatal(err)
	}
	if _, removed, err := registry.Remove(publication.Artifact); err != nil || !removed {
		t.Fatalf("remove exact artifact: removed=%t err=%v", removed, err)
	}
	drifted := clonePublication(publication)
	drifted.Artifact.RuntimeInstanceID = "runtime-restarted"
	drifted.Contributions[0].Priority++
	if _, err := registry.Publish(drifted); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("remove/restart declaration drift=%v", err)
	}
	drifted.Contributions[0] = publication.Contributions[0]
	if _, err := registry.Publish(drifted); err != nil {
		t.Fatalf("same immutable declarations after restart=%v", err)
	}
}

func TestGlobalScopeInspectionAndExecutionPlanRetainGlobalContributions(t *testing.T) {
	publication := testPublication("plugin.global", 'a')
	publication.Contributions = []Declaration{
		testDeclaration(publication, "title", GlobalScope, KindTitle, ActionFilter, FailurePolicyFailClosed, 0),
	}
	registry := New()
	if _, err := registry.Publish(publication); err != nil {
		t.Fatal(err)
	}
	inspection, err := registry.Inspect(GlobalScope)
	if err != nil || len(inspection.Contributions) != 1 || inspection.Contributions[0].Scope != GlobalScope {
		t.Fatalf("global inspection=%#v err=%v", inspection, err)
	}
}

func TestRegistryRejectsForgedCoreAndUnboundedDeclarations(t *testing.T) {
	forged := testPublication("core.forged", 'a')
	forged.Artifact.Core = true
	forged.Artifact.VersionID = 0
	forged.Artifact.RuntimeInstanceID = ""
	if _, err := New().Publish(forged); !errors.Is(err, ErrInvalid) {
		t.Fatalf("forged core artifact=%v", err)
	}
	digest := strings.Repeat("a", 64)
	core, err := NewCoreArtifact("core.seo", "1.0.0", digest, digest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := New().Publish(Publication{Artifact: core}); err != nil {
		t.Fatalf("sealed core artifact=%v", err)
	}

	publication := testPublication("plugin.invalid", 'b')
	declaration := testDeclaration(publication, "title", "core.page.topic", KindTitle, ActionAdd, "", 0)
	publication.Contributions = []Declaration{declaration}
	if _, err := New().Publish(publication); !errors.Is(err, ErrInvalid) {
		t.Fatalf("implicit failure policy=%v", err)
	}
	declaration.FailurePolicy = FailurePolicyFailClosed
	declaration.Priority = maxPriority + 1
	publication.Contributions = []Declaration{declaration}
	if _, err := New().Publish(publication); !errors.Is(err, ErrInvalid) {
		t.Fatalf("priority overflow=%v", err)
	}
}
