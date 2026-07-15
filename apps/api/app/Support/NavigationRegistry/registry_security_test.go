package navigationregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestRegistryEmptySnapshotPublicationsUseJSONArray(t *testing.T) {
	snapshot := New().Snapshot()
	if snapshot.Publications == nil || len(snapshot.Publications) != 0 {
		t.Fatalf("empty publications must be a non-nil slice: %#v", snapshot.Publications)
	}
	body, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(body, &document); err != nil {
		t.Fatal(err)
	}
	if value := string(document["publications"]); value != "[]" {
		t.Fatalf("empty publications JSON=%s, snapshot=%s", value, body)
	}
}

func TestRegistryPublishRequiresExactArtifactCASForReplacement(t *testing.T) {
	registry := New()
	initial := publication("demo.navigation", false, 'a')
	initial.Navigation = []NavigationDeclaration{
		navigation("demo.navigation.item.initial", NavigationKindItem, ActionAdd, "", 0),
	}
	if _, err := registry.Publish(initial); err != nil {
		t.Fatal(err)
	}
	replacement := publication("demo.navigation", false, 'b')
	replacement.Artifact.ExtensionVersion = "2.0.0"
	replacement.Artifact.PackageDigest = strings.Repeat("b", 64)
	replacement.Navigation = []NavigationDeclaration{
		navigation("demo.navigation.item.replacement", NavigationKindItem, ActionAdd, "", 0),
	}
	before := registry.Snapshot()

	if revision, err := registry.Publish(replacement); !errors.Is(err, ErrArtifactConflict) || revision != before.Revision {
		t.Fatalf("unconditional replacement: revision=%d err=%v", revision, err)
	}
	staleExpected := initial.Artifact
	staleExpected.ImpactDigest = strings.Repeat("c", 64)
	if revision, err := registry.PublishIfArtifact(staleExpected, replacement); !errors.Is(err, ErrArtifactConflict) || revision != before.Revision {
		t.Fatalf("stale CAS: revision=%d err=%v", revision, err)
	}
	if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatalf("failed CAS changed snapshot: before=%#v after=%#v", before, after)
	}
	if revision, err := registry.PublishIfArtifact(initial.Artifact, replacement); err != nil || revision != before.Revision+1 {
		t.Fatalf("exact CAS: revision=%d err=%v", revision, err)
	}

	// 同 exact artifact 声明漂移必须 fail-closed，不能 silent overwrite。
	drifted := replacement
	drifted.Navigation = append([]NavigationDeclaration(nil), replacement.Navigation...)
	drifted.Navigation[0].Label = "drifted"
	if revision, err := registry.Publish(drifted); !errors.Is(err, ErrArtifactConflict) || revision != before.Revision+1 {
		t.Fatalf("same-artifact declaration drift: revision=%d err=%v", revision, err)
	}
	if after := registry.Snapshot(); after.Publications[0].Navigation[0].Label == "drifted" {
		t.Fatalf("failed drift CAS changed the active publication: %#v", after)
	}
	if revision, err := registry.Publish(replacement); err != nil || revision != before.Revision+1 {
		t.Fatalf("same-artifact equivalent replay: revision=%d err=%v", revision, err)
	}
}

func TestRegistryTracksExactOwnershipForEmptyPublication(t *testing.T) {
	registry := New()
	empty := publication("empty.navigation", false, 'a')
	empty.Dependencies = []Dependency{{Capability: "forum.navigation", Version: "1.0.0", Kind: DependencyProvides}}
	if revision, err := registry.Publish(empty); err != nil || revision != 1 {
		t.Fatalf("publish empty: revision=%d err=%v", revision, err)
	}
	snapshot := registry.Snapshot()
	if len(snapshot.Publications) != 1 || snapshot.Publications[0].Artifact != empty.Artifact ||
		len(snapshot.Publications[0].Navigation) != 0 || len(snapshot.Publications[0].Regions) != 0 ||
		len(snapshot.Publications[0].Dependencies) != 1 || snapshot.Digest == emptyState().digest ||
		len(snapshot.Navigation) != 0 {
		t.Fatalf("empty publication is not inspectable: %#v", snapshot)
	}

	// 空 publication 仍可授权 required capability 消费者。
	consumer := publication("consumer.navigation", false, 'b')
	consumer.Dependencies = []Dependency{{Capability: "forum.navigation", Version: "^1.0.0", Kind: DependencyRequired}}
	consumer.Navigation = []NavigationDeclaration{
		navigation("consumer.navigation.item.base", NavigationKindItem, ActionAdd, "", 0),
	}
	if _, err := registry.Publish(consumer); err != nil {
		t.Fatalf("consumer of empty capability owner: %v", err)
	}
	if snapshot := registry.Snapshot(); len(snapshot.Publications) != 2 || len(snapshot.Navigation) != 1 {
		t.Fatalf("empty capability owner graph=%#v", snapshot)
	}

	stale := empty.Artifact
	stale.PackageDigest = strings.Repeat("b", 64)
	if revision, removed, err := registry.Remove(stale); !errors.Is(err, ErrArtifactConflict) || removed || revision != 2 {
		t.Fatalf("stale empty remove: revision=%d removed=%t err=%v", revision, removed, err)
	}
	// 仍有 required 消费者时，空 capability owner 也不能被摘掉。
	if revision, removed, err := registry.Remove(empty.Artifact); !errors.Is(err, ErrDependency) || removed || revision != 2 {
		t.Fatalf("required empty owner removal: revision=%d removed=%t err=%v", revision, removed, err)
	}
	if _, removed, err := registry.Remove(consumer.Artifact); err != nil || !removed {
		t.Fatalf("remove consumer: removed=%t err=%v", removed, err)
	}
	if revision, removed, err := registry.Remove(empty.Artifact); err != nil || !removed || revision != 4 {
		t.Fatalf("exact empty remove: revision=%d removed=%t err=%v", revision, removed, err)
	}
}

func TestRegistryReplaceAllPreservesGraphOnInvalidOrConflictingInput(t *testing.T) {
	core := publication("core.navigation", true, 'a')
	core.Navigation = []NavigationDeclaration{
		navigation("core.navigation.item.home", NavigationKindItem, ActionAdd, "", 0),
	}
	plugin := publication("plugin.navigation", false, 'b')
	plugin.Navigation = []NavigationDeclaration{
		navigation("plugin.navigation.item.home", NavigationKindItem, ActionReplace, core.Navigation[0].ID, 0),
	}
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{core, plugin}); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()

	// 缺少 required 依赖的图不能部分落地。
	orphan := publication("orphan.navigation", false, 'c')
	orphan.Dependencies = []Dependency{{ExtensionID: "missing.navigation", Version: "^1.0.0", Kind: DependencyRequired}}
	if revision, err := registry.ReplaceAll([]Publication{core, orphan}); !errors.Is(err, ErrDependency) || revision != before.Revision {
		t.Fatalf("invalid replace-all: revision=%d err=%v", revision, err)
	}
	if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatalf("invalid replace-all changed snapshot: before=%#v after=%#v", before, after)
	}

	// 冲突图同样 all-or-nothing。
	duplicate := core
	duplicate.Navigation = append([]NavigationDeclaration(nil), core.Navigation...)
	if revision, err := registry.ReplaceAll([]Publication{core, duplicate}); !errors.Is(err, ErrConflict) || revision != before.Revision {
		t.Fatalf("conflicting replace-all: revision=%d err=%v", revision, err)
	}
	if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatalf("conflicting replace-all changed snapshot: before=%#v after=%#v", before, after)
	}
}

func TestRegistryReplaceAllIfRevisionFencesStaleWriter(t *testing.T) {
	initial := publication("batch.navigation", false, 'a')
	initial.Navigation = []NavigationDeclaration{
		navigation("batch.navigation.item.initial", NavigationKindItem, ActionAdd, "", 0),
	}
	replacement := publication("batch.navigation", false, 'b')
	replacement.Artifact.ExtensionVersion = "2.0.0"
	replacement.Navigation = []NavigationDeclaration{
		navigation("batch.navigation.item.replacement", NavigationKindItem, ActionAdd, "", 0),
	}
	registry := New()
	if _, err := registry.Publish(initial); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()

	if revision, err := registry.ReplaceAllIfRevision(before.Revision-1, []Publication{replacement}); !errors.Is(err, ErrRevisionConflict) || revision != before.Revision {
		t.Fatalf("stale full replacement: revision=%d err=%v", revision, err)
	}
	if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatalf("stale full replacement changed snapshot: before=%#v after=%#v", before, after)
	}
}

func TestRegistryReplaceAllRejectsExactArtifactDriftAndAllowsFencedTransitions(t *testing.T) {
	initial := publication("batch.drift", false, 'a')
	initial.Navigation = []NavigationDeclaration{
		navigation("batch.drift.item.initial", NavigationKindItem, ActionAdd, "", 0),
	}
	registry := New()
	if _, err := registry.Publish(initial); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()

	if revision, err := registry.ReplaceAllIfRevision(before.Revision, []Publication{initial}); err != nil || revision != before.Revision {
		t.Fatalf("exact graph replay: revision=%d err=%v", revision, err)
	}
	drifted := initial
	drifted.Navigation = append([]NavigationDeclaration(nil), initial.Navigation...)
	drifted.Navigation[0].Label = "drifted"
	if revision, err := registry.ReplaceAllIfRevision(before.Revision, []Publication{drifted}); !errors.Is(err, ErrArtifactConflict) || revision != before.Revision {
		t.Fatalf("exact-artifact drift: revision=%d err=%v", revision, err)
	}
	if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatalf("exact-artifact drift changed snapshot: before=%#v after=%#v", before, after)
	}

	replacement := publication("batch.drift", false, 'b')
	replacement.Artifact.ExtensionVersion = "2.0.0"
	replacement.Navigation = []NavigationDeclaration{
		navigation("batch.drift.item.replacement", NavigationKindItem, ActionAdd, "", 0),
	}
	revision, err := registry.ReplaceAllIfRevision(before.Revision, []Publication{replacement})
	if err != nil || revision != before.Revision+1 {
		t.Fatalf("revision-fenced artifact replacement: revision=%d err=%v", revision, err)
	}
	if revision, err = registry.ReplaceAllIfRevision(revision, nil); err != nil || revision != before.Revision+2 {
		t.Fatalf("revision-fenced artifact removal: revision=%d err=%v", revision, err)
	}
	if snapshot := registry.Snapshot(); len(snapshot.Publications) != 0 || snapshot.Publications == nil {
		t.Fatalf("full removal snapshot=%#v", snapshot)
	}
}

func TestRegistryReplaceAllDigestIsInputOrderIndependent(t *testing.T) {
	core := publication("core.navigation", true, 'a')
	core.Navigation = []NavigationDeclaration{
		navigation("core.navigation.item.public", NavigationKindItem, ActionAdd, "", 20),
	}
	theme := publication("theme.signal", false, 'b')
	theme.Regions = []RegionDeclaration{region("theme.signal.region.widget", RegionKindWidget, ActionAdd, "")}
	empty := publication("empty.capability", false, 'c')
	empty.Dependencies = []Dependency{{Capability: "theme.chrome", Version: "1.0.0", Kind: DependencyProvides}}

	first, second := New(), New()
	if _, err := first.ReplaceAll([]Publication{core, theme, empty}); err != nil {
		t.Fatal(err)
	}
	if _, err := second.ReplaceAll([]Publication{empty, theme, core}); err != nil {
		t.Fatal(err)
	}
	left, right := first.Snapshot(), second.Snapshot()
	if left.Digest != right.Digest || !reflect.DeepEqual(left.Publications, right.Publications) {
		t.Fatalf("input order changed digest/publications: left=%#v right=%#v", left, right)
	}
	if len(left.Publications) != 3 {
		t.Fatalf("expected three retained publications including empty owner: %#v", left.Publications)
	}
}

func TestRegistrySnapshotAndResolvePlansAreDeepCopied(t *testing.T) {
	core := publication("core.navigation", true, 'a')
	core.Navigation = []NavigationDeclaration{
		navigation("core.navigation.item.home", NavigationKindItem, ActionAdd, "", 0),
	}
	alpha := publication("alpha.navigation", false, 'b')
	alpha.Dependencies = []Dependency{{Capability: "nav.slot", Version: "1.0.0", Kind: DependencyProvides}}
	alpha.Navigation = []NavigationDeclaration{
		navigation("alpha.navigation.item.home", NavigationKindItem, ActionReplace, core.Navigation[0].ID, 0),
	}
	beta := publication("beta.navigation", false, 'c')
	beta.Navigation = []NavigationDeclaration{
		navigation("beta.navigation.item.home", NavigationKindItem, ActionReplace, core.Navigation[0].ID, 0),
	}
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{core, alpha, beta}); err != nil {
		t.Fatal(err)
	}

	snapshot := registry.Snapshot()
	snapshot.Publications[1].Dependencies[0].Capability = "forged.capability"
	snapshot.Publications[1].Navigation[0].Label = "forged publication"
	snapshot.Navigation[0].Label = "forged navigation"
	snapshot.NavigationConflicts[0].Candidates[0].Label = "forged conflict"
	resolved := mustResolveNavigation(t, registry, NavigationResolveRequest{})
	resolved.Targets[0].ReplaceCandidates[0].Label = "forged plan"
	resolved.Targets[0].Provider.Label = "forged provider"

	againSnapshot := registry.Snapshot()
	againResolved := mustResolveNavigation(t, registry, NavigationResolveRequest{})
	if againSnapshot.Publications[1].Dependencies[0].Capability == "forged.capability" ||
		againSnapshot.Publications[1].Navigation[0].Label == "forged publication" ||
		againSnapshot.Navigation[0].Label == "forged navigation" ||
		againSnapshot.NavigationConflicts[0].Candidates[0].Label == "forged conflict" ||
		againResolved.Targets[0].ReplaceCandidates[0].Label == "forged plan" ||
		againResolved.Targets[0].Provider.Label == "forged provider" {
		t.Fatalf("caller mutation escaped: snapshot=%#v resolution=%#v", againSnapshot, againResolved)
	}
}

func TestRegistryConcurrentArtifactCASPublishesOneReplacement(t *testing.T) {
	registry := New()
	initial := publication("cas.navigation", false, 'a')
	initial.Navigation = []NavigationDeclaration{
		navigation("cas.navigation.item.initial", NavigationKindItem, ActionAdd, "", 0),
	}
	if _, err := registry.Publish(initial); err != nil {
		t.Fatal(err)
	}
	first := publication("cas.navigation", false, 'b')
	first.Artifact.ExtensionVersion, first.Artifact.PackageDigest = "2.0.0", strings.Repeat("b", 64)
	first.Navigation = []NavigationDeclaration{
		navigation("cas.navigation.item.first", NavigationKindItem, ActionAdd, "", 0),
	}
	second := publication("cas.navigation", false, 'c')
	second.Artifact.ExtensionVersion, second.Artifact.PackageDigest = "2.0.0", strings.Repeat("c", 64)
	second.Navigation = []NavigationDeclaration{
		navigation("cas.navigation.item.second", NavigationKindItem, ActionAdd, "", 0),
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	for _, item := range []Publication{first, second} {
		item := item
		go func() {
			<-start
			_, err := registry.PublishIfArtifact(initial.Artifact, item)
			results <- err
		}()
	}
	close(start)
	succeeded, conflicted := 0, 0
	for range 2 {
		switch err := <-results; {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrArtifactConflict):
			conflicted++
		default:
			t.Fatalf("unexpected CAS result: %v", err)
		}
	}
	snapshot := registry.Snapshot()
	if succeeded != 1 || conflicted != 1 || snapshot.Revision != 2 || len(snapshot.Publications) != 1 ||
		len(snapshot.Navigation) != 1 ||
		(snapshot.Navigation[0].ID != "cas.navigation.item.first" && snapshot.Navigation[0].ID != "cas.navigation.item.second") {
		t.Fatalf("CAS did not publish exactly one whole replacement: success=%d conflict=%d snapshot=%#v", succeeded, conflicted, snapshot)
	}
}

func TestRegistryConcurrentFullReplaceCASAndArtifactCASPublishOneWinner(t *testing.T) {
	registry := New()
	initial := publication("batch.concurrent", false, 'a')
	initial.Navigation = []NavigationDeclaration{
		navigation("batch.concurrent.item.initial", NavigationKindItem, ActionAdd, "", 0),
	}
	if _, err := registry.Publish(initial); err != nil {
		t.Fatal(err)
	}
	expectedRevision := registry.Revision()
	batch := publication("batch.concurrent", false, 'b')
	batch.Artifact.ExtensionVersion = "2.0.0"
	batch.Navigation = []NavigationDeclaration{
		navigation("batch.concurrent.item.batch", NavigationKindItem, ActionAdd, "", 0),
	}
	incremental := publication("batch.concurrent", false, 'c')
	incremental.Artifact.ExtensionVersion = "2.0.0"
	incremental.Navigation = []NavigationDeclaration{
		navigation("batch.concurrent.item.incremental", NavigationKindItem, ActionAdd, "", 0),
	}

	type outcome struct {
		kind     string
		revision uint64
		err      error
	}
	start := make(chan struct{})
	results := make(chan outcome, 2)
	go func() {
		<-start
		revision, err := registry.ReplaceAllIfRevision(expectedRevision, []Publication{batch})
		results <- outcome{kind: "batch", revision: revision, err: err}
	}()
	go func() {
		<-start
		revision, err := registry.PublishIfArtifact(initial.Artifact, incremental)
		results <- outcome{kind: "incremental", revision: revision, err: err}
	}()
	close(start)

	succeeded := ""
	for range 2 {
		result := <-results
		if result.err == nil {
			if succeeded != "" || result.revision != expectedRevision+1 {
				t.Fatalf("unexpected successful outcome: previous=%s result=%#v", succeeded, result)
			}
			succeeded = result.kind
			continue
		}
		if result.revision != expectedRevision+1 ||
			(result.kind == "batch" && !errors.Is(result.err, ErrRevisionConflict)) ||
			(result.kind == "incremental" && !errors.Is(result.err, ErrArtifactConflict)) {
			t.Fatalf("unexpected losing outcome: %#v", result)
		}
	}
	if succeeded == "" {
		t.Fatal("both concurrent publication paths failed")
	}
	snapshot := registry.Snapshot()
	if snapshot.Revision != expectedRevision+1 || len(snapshot.Publications) != 1 || len(snapshot.Navigation) != 1 {
		t.Fatalf("concurrent final snapshot=%#v", snapshot)
	}
	winnerID := snapshot.Navigation[0].ID
	if (succeeded == "batch" && winnerID != batch.Navigation[0].ID) ||
		(succeeded == "incremental" && winnerID != incremental.Navigation[0].ID) {
		t.Fatalf("success=%s winner=%s snapshot=%#v", succeeded, winnerID, snapshot)
	}
}

func TestRegistryConcurrentReadersObserveWholePublications(t *testing.T) {
	registry := New()
	first := publication("demo.navigation", false, 'a')
	first.Navigation = []NavigationDeclaration{
		navigation("demo.navigation.item.a", NavigationKindItem, ActionAdd, "", 10),
		navigation("demo.navigation.item.b", NavigationKindItem, ActionAdd, "", 20),
	}
	second := publication("demo.navigation", false, 'b')
	second.Artifact.PackageDigest = strings.Repeat("b", 64)
	second.Navigation = []NavigationDeclaration{
		navigation("demo.navigation.item.c", NavigationKindItem, ActionAdd, "", 10),
		navigation("demo.navigation.item.d", NavigationKindItem, ActionAdd, "", 20),
	}
	if _, err := registry.Publish(first); err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	problems := make(chan error, 8)
	var readers sync.WaitGroup
	for reader := 0; reader < 8; reader++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			<-start
			for iteration := 0; iteration < 300; iteration++ {
				snapshot := registry.Snapshot()
				if !wholeNavigationPair(snapshot.Navigation) || len(snapshot.Publications) != 1 {
					problems <- fmt.Errorf("partial snapshot: %#v", snapshot)
					return
				}
				resolution, err := registry.ResolveNavigation(NavigationResolveRequest{})
				if err != nil || !wholeNavigationPlan(resolution) {
					problems <- fmt.Errorf("partial plan: %#v: %w", resolution, err)
					return
				}
			}
		}()
	}
	close(start)
	current := first
	for iteration := 0; iteration < 300; iteration++ {
		next := first
		if iteration%2 == 0 {
			next = second
		}
		if _, err := registry.PublishIfArtifact(current.Artifact, next); err != nil {
			t.Fatal(err)
		}
		current = next
	}
	readers.Wait()
	close(problems)
	for problem := range problems {
		t.Fatal(problem)
	}
}

func TestRegistryPublishInputMutationIsIsolated(t *testing.T) {
	input := publication("mutated.navigation", false, 'a')
	input.Dependencies = []Dependency{{Capability: "mutated.capability", Version: "1.0.0", Kind: DependencyProvides}}
	input.Navigation = []NavigationDeclaration{
		navigation("mutated.navigation.item.base", NavigationKindItem, ActionAdd, "", 0),
	}
	input.Regions = []RegionDeclaration{
		region("mutated.navigation.region.base", RegionKindWidget, ActionAdd, ""),
	}
	registry := New()
	if _, err := registry.Publish(input); err != nil {
		t.Fatal(err)
	}
	input.Artifact.PackageDigest = strings.Repeat("f", 64)
	input.Dependencies[0].Capability = "forged.capability"
	input.Navigation[0].Label = "forged navigation"
	input.Regions[0].Label = "forged region"

	snapshot := registry.Snapshot()
	if snapshot.Publications[0].Artifact.PackageDigest != strings.Repeat("a", 64) ||
		snapshot.Publications[0].Dependencies[0].Capability != "mutated.capability" ||
		snapshot.Publications[0].Navigation[0].Label == "forged navigation" ||
		snapshot.Publications[0].Regions[0].Label == "forged region" {
		t.Fatalf("publish input mutation escaped: %#v", snapshot.Publications[0])
	}

	clean := publication("mutated.navigation", false, 'a')
	clean.Navigation = []NavigationDeclaration{
		navigation("mutated.navigation.item.base", NavigationKindItem, ActionAdd, "", 0),
	}
	restarted := New()
	batch := []Publication{clean}
	if _, err := restarted.ReplaceAll(batch); err != nil {
		t.Fatal(err)
	}
	batch[0].Navigation[0].Label = "forged batch"
	batch[0] = Publication{}
	if after := restarted.Snapshot(); after.Navigation[0].Label == "forged batch" || len(after.Publications) != 1 {
		t.Fatalf("replace-all input mutation escaped: %#v", after)
	}
}

func wholeNavigationPair(values []NavigationContribution) bool {
	if len(values) != 2 {
		return false
	}
	return values[0].ID == "demo.navigation.item.a" && values[1].ID == "demo.navigation.item.b" ||
		values[0].ID == "demo.navigation.item.c" && values[1].ID == "demo.navigation.item.d"
}

func wholeNavigationPlan(resolution NavigationResolution) bool {
	if len(resolution.Targets) != 2 {
		return false
	}
	return wholeNavigationPair([]NavigationContribution{
		resolution.Targets[0].Target, resolution.Targets[1].Target,
	})
}
