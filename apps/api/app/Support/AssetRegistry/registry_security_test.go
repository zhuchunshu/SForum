package assetregistry

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
)

func TestRegistryRemoveBindsExactArtifact(t *testing.T) {
	registry := New()
	initial := fixturePublication("demo.assets", digestA, []Declaration{{
		Handle: "demo.assets.old", ContractVersion: "demo.assets.old@1", Type: "style", Path: "old.css", Digest: digestB,
	}})
	if _, err := registry.Publish(initial); err != nil {
		t.Fatal(err)
	}
	active := fixturePublication("demo.assets", digestC, []Declaration{{
		Handle: "demo.assets.new", ContractVersion: "demo.assets.new@1", Type: "style", Path: "new.css", Digest: digestC,
	}})
	active.Artifact.ExtensionVersion = "2.0.0"
	active.Artifact.PackageDigest = digestB
	if _, err := registry.PublishIfArtifact(initial.Artifact, active); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()

	staleArtifacts := []Artifact{
		{ExtensionID: "other.assets", ExtensionVersion: active.Artifact.ExtensionVersion, PackageDigest: active.Artifact.PackageDigest, ImpactDigest: active.Artifact.ImpactDigest, OwnerKind: OwnerKindPlugin},
		{ExtensionID: active.Artifact.ExtensionID, ExtensionVersion: "1.0.0", PackageDigest: active.Artifact.PackageDigest, ImpactDigest: active.Artifact.ImpactDigest, OwnerKind: OwnerKindPlugin},
		{ExtensionID: active.Artifact.ExtensionID, ExtensionVersion: active.Artifact.ExtensionVersion, PackageDigest: digestC, ImpactDigest: active.Artifact.ImpactDigest, OwnerKind: OwnerKindPlugin},
		{ExtensionID: active.Artifact.ExtensionID, ExtensionVersion: active.Artifact.ExtensionVersion, PackageDigest: active.Artifact.PackageDigest, ImpactDigest: digestB, OwnerKind: OwnerKindPlugin},
	}
	for index, stale := range staleArtifacts {
		revision, removed, err := registry.Remove(stale)
		if index == 0 {
			if err != nil || removed || revision != before.Revision {
				t.Fatalf("unknown artifact: revision=%d removed=%t err=%v", revision, removed, err)
			}
		} else if !errors.Is(err, ErrArtifactConflict) || removed || revision != before.Revision {
			t.Fatalf("stale artifact %d: revision=%d removed=%t err=%v", index, revision, removed, err)
		}
		if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
			t.Fatalf("stale artifact %d changed snapshot: before=%#v after=%#v", index, before, after)
		}
	}

	if revision, removed, err := registry.Remove(active.Artifact); err != nil || !removed || revision != before.Revision+1 {
		t.Fatalf("exact remove: revision=%d removed=%t err=%v", revision, removed, err)
	}
	if _, ok := registry.Resolve("demo.assets.new"); ok {
		t.Fatal("exact remove retained the active artifact")
	}
}

func TestRegistryTracksExactOwnershipForEmptyPublication(t *testing.T) {
	registry := New()
	publication := fixturePublication("empty.assets", digestA, nil)
	if revision, err := registry.Publish(publication); err != nil || revision != 1 {
		t.Fatalf("publish empty: revision=%d err=%v", revision, err)
	}
	snapshot := registry.Snapshot()
	if len(snapshot.Publications) != 1 || snapshot.Publications[0].Artifact != publication.Artifact ||
		len(snapshot.Publications[0].Assets) != 0 || snapshot.Digest == emptyState().digest {
		t.Fatalf("empty publication is not inspectable: %#v", snapshot)
	}
	stale := publication.Artifact
	stale.PackageDigest = digestB
	if revision, removed, err := registry.Remove(stale); !errors.Is(err, ErrArtifactConflict) || removed || revision != 1 {
		t.Fatalf("stale empty remove: revision=%d removed=%t err=%v", revision, removed, err)
	}
	if revision, removed, err := registry.Remove(publication.Artifact); err != nil || !removed || revision != 2 {
		t.Fatalf("exact empty remove: revision=%d removed=%t err=%v", revision, removed, err)
	}
}

func TestRegistryReservesCoreAssetNamespaceForCoreArtifacts(t *testing.T) {
	declaration := Declaration{
		Handle: "core.asset.vue", ContractVersion: "sforum.asset.vue@1", Type: "script", Path: "vue.mjs", Digest: digestB,
	}
	if _, err := New().Publish(fixturePublication("demo.assets", digestA, []Declaration{declaration})); !errors.Is(err, ErrInvalid) {
		t.Fatalf("third party occupied core namespace: %v", err)
	}
	core := fixturePublication("core.assets", digestA, []Declaration{declaration})
	core.Artifact.Core = true
	core.Artifact.OwnerKind = OwnerKindCore
	registry := New()
	if _, err := registry.Publish(core); err != nil {
		t.Fatalf("core publication: %v", err)
	}
	plan, err := registry.Plan(PlanRequest{Handles: []string{declaration.Handle}})
	if err != nil || len(plan) != 1 || plan[0].Artifact != core.Artifact {
		t.Fatalf("core plan=%#v err=%v", plan, err)
	}
	forgedCore := fixturePublication("core.forged", digestA, []Declaration{declaration})
	if _, err := New().Publish(forgedCore); !errors.Is(err, ErrInvalid) {
		t.Fatalf("id-only core forgery: %v", err)
	}
	reservedRoot := fixturePublication("core", digestA, []Declaration{{
		Handle: "core.asset", ContractVersion: "core.asset@1", Type: "script", Path: "asset.mjs", Digest: digestB,
	}})
	if _, err := New().Publish(reservedRoot); !errors.Is(err, ErrInvalid) {
		t.Fatalf("core root namespace forgery: %v", err)
	}
	reservedOther := fixturePublication("core", digestA, []Declaration{{
		Handle: "core.other", ContractVersion: "core.other@1", Type: "script", Path: "other.mjs", Digest: digestB,
	}})
	if _, err := New().Publish(reservedOther); !errors.Is(err, ErrInvalid) {
		t.Fatalf("non-asset core namespace forgery: %v", err)
	}
	coreRoot := fixturePublication("core.assets", digestA, []Declaration{{
		Handle: "core.asset", ContractVersion: "sforum.asset@1", Type: "script", Path: "asset.mjs", Digest: digestB,
	}})
	coreRoot.Artifact.Core = true
	coreRoot.Artifact.OwnerKind = OwnerKindCore
	if _, err := New().Publish(coreRoot); !errors.Is(err, ErrInvalid) {
		t.Fatalf("core artifact published the reserved family root: %v", err)
	}
}

func TestRegistryBindsHandlesToOwnerAndContractVersion(t *testing.T) {
	tests := []struct {
		name        string
		declaration Declaration
	}{
		{
			name: "foreign owner",
			declaration: Declaration{
				Handle: "other.assets.entry", ContractVersion: "other.assets.entry@1",
				Type: "script", Path: "entry.mjs", Digest: digestB,
			},
		},
		{
			name: "unbound contract",
			declaration: Declaration{
				Handle: "demo.assets.entry", ContractVersion: "demo.assets.other@1",
				Type: "script", Path: "entry.mjs", Digest: digestB,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := New().Publish(fixturePublication("demo.assets", digestA, []Declaration{test.declaration})); !errors.Is(err, ErrInvalid) {
				t.Fatalf("expected invalid identity, got %v", err)
			}
		})
	}

	core := fixturePublication("core.assets", digestA, []Declaration{{
		Handle: "core.asset.vue", ContractVersion: "core.asset.vue@1",
		Type: "script", Path: "vue.mjs", Digest: digestB,
	}})
	core.Artifact.Core = true
	core.Artifact.OwnerKind = OwnerKindCore
	if _, err := New().Publish(core); !errors.Is(err, ErrInvalid) {
		t.Fatalf("core handle accepted a non-sforum contract: %v", err)
	}
}

func TestRegistryPublishRequiresExactArtifactCASForReplacement(t *testing.T) {
	registry := New()
	initial := fixturePublication("demo.assets", digestA, []Declaration{{
		Handle: "demo.assets.initial", ContractVersion: "demo.assets.initial@1",
		Type: "script", Path: "initial.mjs", Digest: digestA,
	}})
	if _, err := registry.Publish(initial); err != nil {
		t.Fatal(err)
	}
	replacement := fixturePublication("demo.assets", digestB, []Declaration{{
		Handle: "demo.assets.replacement", ContractVersion: "demo.assets.replacement@1",
		Type: "script", Path: "replacement.mjs", Digest: digestB,
	}})
	replacement.Artifact.ExtensionVersion = "2.0.0"
	replacement.Artifact.PackageDigest = digestB
	before := registry.Snapshot()
	if revision, err := registry.Publish(replacement); !errors.Is(err, ErrArtifactConflict) || revision != before.Revision {
		t.Fatalf("unconditional replacement: revision=%d err=%v", revision, err)
	}
	staleExpected := initial.Artifact
	staleExpected.ImpactDigest = digestC
	if revision, err := registry.PublishIfArtifact(staleExpected, replacement); !errors.Is(err, ErrArtifactConflict) || revision != before.Revision {
		t.Fatalf("stale CAS: revision=%d err=%v", revision, err)
	}
	if revision, err := registry.PublishIfArtifact(initial.Artifact, replacement); err != nil || revision != before.Revision+1 {
		t.Fatalf("exact CAS: revision=%d err=%v", revision, err)
	}

	drifted := replacement
	drifted.Assets = append([]Declaration(nil), replacement.Assets...)
	drifted.Assets[0].Path = "drifted.mjs"
	if revision, err := registry.Publish(drifted); !errors.Is(err, ErrArtifactConflict) || revision != before.Revision+1 {
		t.Fatalf("same-artifact declaration drift: revision=%d err=%v", revision, err)
	}
	if after := registry.Snapshot(); after.Publications[0].Assets[0].Path != "replacement.mjs" {
		t.Fatalf("failed CAS changed the active publication: %#v", after)
	}
}

func TestRegistryConcurrentArtifactCASPublishesOneReplacement(t *testing.T) {
	registry := New()
	initial := fixturePublication("cas.assets", digestA, []Declaration{{
		Handle: "cas.assets.initial", ContractVersion: "cas.assets.initial@1",
		Type: "script", Path: "initial.mjs", Digest: digestA,
	}})
	if _, err := registry.Publish(initial); err != nil {
		t.Fatal(err)
	}
	first := fixturePublication("cas.assets", digestB, []Declaration{{
		Handle: "cas.assets.first", ContractVersion: "cas.assets.first@1",
		Type: "script", Path: "first.mjs", Digest: digestB,
	}})
	first.Artifact.ExtensionVersion, first.Artifact.PackageDigest = "2.0.0", digestB
	second := fixturePublication("cas.assets", digestC, []Declaration{{
		Handle: "cas.assets.second", ContractVersion: "cas.assets.second@1",
		Type: "script", Path: "second.mjs", Digest: digestC,
	}})
	second.Artifact.ExtensionVersion, second.Artifact.PackageDigest = "2.0.0", digestC

	start := make(chan struct{})
	results := make(chan error, 2)
	for _, publication := range []Publication{first, second} {
		publication := publication
		go func() {
			<-start
			_, err := registry.PublishIfArtifact(initial.Artifact, publication)
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
	if succeeded != 1 || conflicted != 1 || snapshot.Revision != 2 || len(snapshot.Assets) != 1 ||
		(snapshot.Assets[0].Handle != "cas.assets.first" && snapshot.Assets[0].Handle != "cas.assets.second") {
		t.Fatalf("CAS did not publish exactly one whole replacement: success=%d conflict=%d snapshot=%#v", succeeded, conflicted, snapshot)
	}
}

func TestRegistryClonesPublishReplaceAllAndEveryReadResult(t *testing.T) {
	publication := fixturePublication("demo.assets", digestA, []Declaration{{
		Handle: "demo.assets.entry", ContractVersion: "demo.assets.entry@1", Type: "script", Path: "entry.mjs", Digest: digestB,
		Dependencies: []string{"core.asset.vue"}, Scope: []string{"forum.component.topic"}, CSP: []string{"connect-src 'self'"},
	}})
	registry := New()
	if _, err := registry.Publish(publication); err != nil {
		t.Fatal(err)
	}
	mutatePublication(&publication)
	assertDetachedRegistryAsset(t, registry)

	asset, ok := registry.Resolve("demo.assets.entry")
	if !ok {
		t.Fatal("resolve failed")
	}
	mutateAsset(&asset)
	plan, err := registry.Plan(PlanRequest{Handles: []string{"demo.assets.entry"}})
	if err != nil || len(plan) != 1 {
		t.Fatalf("plan=%#v err=%v", plan, err)
	}
	mutateAsset(&plan[0])
	snapshot := registry.Snapshot()
	mutateAsset(&snapshot.Assets[0])
	mutatePublication(&snapshot.Publications[0])
	assertDetachedRegistryAsset(t, registry)

	replacement := fixturePublication("demo.assets", digestA, []Declaration{{
		Handle: "demo.assets.entry", ContractVersion: "demo.assets.entry@1", Type: "script", Path: "entry.mjs", Digest: digestB,
		Dependencies: []string{"core.asset.vue"}, Scope: []string{"forum.component.topic"}, CSP: []string{"connect-src 'self'"},
	}})
	restarted := New()
	input := []Publication{replacement}
	if _, err := restarted.ReplaceAll(input); err != nil {
		t.Fatal(err)
	}
	mutatePublication(&input[0])
	input[0] = Publication{}
	assertDetachedRegistryAsset(t, restarted)
}

func TestRegistryProviderDisableAndReplacementFailClosed(t *testing.T) {
	registry := New()
	owner := fixturePublication("owner.assets", digestA, []Declaration{{
		Handle: "owner.assets.shared", ContractVersion: "owner.assets.shared@1", Type: "script", Path: "shared.mjs", Digest: digestB,
	}})
	consumer := fixturePublication("consumer.assets", digestB, []Declaration{{
		Handle: "consumer.assets.entry", ContractVersion: "consumer.assets.entry@1", Type: "script", Path: "entry.mjs", Digest: digestC,
		Dependencies: []string{"owner.assets.shared"},
	}})
	if _, err := registry.ReplaceAll([]Publication{owner, consumer}); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()

	if _, err := registry.ReplaceAll([]Publication{consumer}); !errors.Is(err, ErrDependency) {
		t.Fatalf("disable provider with live consumer: %v", err)
	}
	withoutHandle := fixturePublication("owner.assets", digestC, []Declaration{{
		Handle: "owner.assets.other", ContractVersion: "owner.assets.other@1", Type: "script", Path: "other.mjs", Digest: digestC,
	}})
	if _, err := registry.PublishIfArtifact(owner.Artifact, withoutHandle); !errors.Is(err, ErrDependency) {
		t.Fatalf("provider replacement dropped required handle: %v", err)
	}
	if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatalf("failed provider changes published partial state: before=%#v after=%#v", before, after)
	}

	if _, removed, err := registry.Remove(consumer.Artifact); err != nil || !removed {
		t.Fatalf("remove consumer: removed=%t err=%v", removed, err)
	}
	if _, removed, err := registry.Remove(owner.Artifact); err != nil || !removed {
		t.Fatalf("remove unreferenced provider: removed=%t err=%v", removed, err)
	}
}

func TestRegistryConcurrentReadersObserveWholePublications(t *testing.T) {
	registry := New()
	first := fixturePublication("demo.assets", digestA, []Declaration{
		{Handle: "demo.assets.first.a", ContractVersion: "demo.assets.first.a@1", Type: "script", Path: "first-a.mjs", Digest: digestA},
		{Handle: "demo.assets.first.b", ContractVersion: "demo.assets.first.b@1", Type: "script", Path: "first-b.mjs", Digest: digestB},
	})
	second := fixturePublication("demo.assets", digestB, []Declaration{
		{Handle: "demo.assets.second.a", ContractVersion: "demo.assets.second.a@1", Type: "script", Path: "second-a.mjs", Digest: digestB},
		{Handle: "demo.assets.second.b", ContractVersion: "demo.assets.second.b@1", Type: "script", Path: "second-b.mjs", Digest: digestC},
	})
	second.Artifact.PackageDigest = digestB
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
				if !wholeAssetPair(snapshot.Assets) {
					problems <- fmt.Errorf("partial snapshot: %#v", snapshot.Assets)
					return
				}
				plan, err := registry.Plan(PlanRequest{IncludeGlobal: true})
				if err != nil || !wholeAssetPair(plan) {
					problems <- fmt.Errorf("partial plan: %#v: %w", plan, err)
					return
				}
			}
		}()
	}
	close(start)
	current := first
	for iteration := 0; iteration < 300; iteration++ {
		publication := first
		if iteration%2 == 0 {
			publication = second
		}
		if _, err := registry.PublishIfArtifact(current.Artifact, publication); err != nil {
			t.Fatal(err)
		}
		current = publication
	}
	readers.Wait()
	close(problems)
	for problem := range problems {
		t.Fatal(problem)
	}
}

func wholeAssetPair(assets []Asset) bool {
	if len(assets) != 2 {
		return false
	}
	return assets[0].Handle == "demo.assets.first.a" && assets[1].Handle == "demo.assets.first.b" ||
		assets[0].Handle == "demo.assets.second.a" && assets[1].Handle == "demo.assets.second.b"
}

func mutatePublication(publication *Publication) {
	publication.Artifact.ExtensionID = "mutated.assets"
	publication.Assets[0].Handle = "mutated.assets.entry"
	publication.Assets[0].Dependencies[0] = "mutated.asset.dependency"
	publication.Assets[0].Scope[0] = "mutated.asset.scope"
	publication.Assets[0].CSP[0] = "connect-src https://mutated.invalid"
}

func mutateAsset(asset *Asset) {
	asset.Handle = "mutated.assets.entry"
	asset.Dependencies[0] = "mutated.asset.dependency"
	asset.Scope[0] = "mutated.asset.scope"
	asset.CSP[0] = "connect-src https://mutated.invalid"
}

func assertDetachedRegistryAsset(t *testing.T, registry *Registry) {
	t.Helper()
	asset, ok := registry.Resolve("demo.assets.entry")
	if !ok || asset.Handle != "demo.assets.entry" || asset.Dependencies[0] != "core.asset.vue" ||
		asset.Scope[0] != "forum.component.topic" || asset.CSP[0] != "connect-src 'self'" {
		t.Fatalf("registry state was aliased: %#v", asset)
	}
}
