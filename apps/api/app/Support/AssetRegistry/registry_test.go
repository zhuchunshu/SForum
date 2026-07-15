package assetregistry

import (
	"errors"
	"reflect"
	"testing"
)

const (
	digestA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	digestB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	digestC = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
)

func TestRegistryPublishesScopedDependencyFirstPlan(t *testing.T) {
	registry := New()
	publication := fixturePublication("demo.assets", digestA, []Declaration{
		{Handle: "demo.assets.style", ContractVersion: "demo.assets.style@1", Type: "style", Path: "public/style.css", Digest: digestB, Scope: []string{"forum.component.topic"}},
		{Handle: "demo.assets.entry", ContractVersion: "demo.assets.entry@1", Type: "script", Path: "public/entry.mjs", Digest: digestC, Module: true, Loading: "lazy", Dependencies: []string{"demo.assets.shared"}, Scope: []string{"forum.component.topic"}},
		{Handle: "demo.assets.shared", ContractVersion: "demo.assets.shared@1", Type: "script", Path: "public/shared.mjs", Digest: digestB, Module: true},
	})
	revision, err := registry.Publish(publication)
	if err != nil || revision != 1 {
		t.Fatalf("publish: revision=%d err=%v", revision, err)
	}
	plan, err := registry.Plan(PlanRequest{Scopes: []string{"forum.component.topic"}})
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(plan))
	for _, asset := range plan {
		got = append(got, asset.Handle)
		if asset.Integrity == "" || asset.Artifact.ImpactDigest != digestA {
			t.Fatalf("asset identity incomplete: %#v", asset)
		}
	}
	want := []string{"demo.assets.shared", "demo.assets.entry", "demo.assets.style"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("plan=%v want=%v", got, want)
	}
}

func TestRegistryRejectsConflictWithoutPublishingPartialState(t *testing.T) {
	registry := New()
	if _, err := registry.Publish(fixturePublication("one.assets", digestA, []Declaration{
		{Handle: "shared.asset.handle", ContractVersion: "shared.asset.handle@1", Type: "style", Path: "style.css", Digest: digestB},
	})); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()
	if _, err := registry.Publish(fixturePublication("two.assets", digestB, []Declaration{
		{Handle: "shared.asset.handle", ContractVersion: "shared.asset.handle@1", Type: "style", Path: "other.css", Digest: digestC},
	})); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	after := registry.Snapshot()
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("failed publication changed snapshot: before=%#v after=%#v", before, after)
	}
}

func TestRegistryRejectsMissingAndCyclicDependencies(t *testing.T) {
	registry := New()
	missing := fixturePublication("demo.assets", digestA, []Declaration{{
		Handle: "demo.assets.entry", ContractVersion: "demo.assets.entry@1", Type: "script",
		Path: "entry.mjs", Digest: digestB, Dependencies: []string{"missing.assets.file"},
	}})
	if _, err := registry.Publish(missing); !errors.Is(err, ErrDependency) {
		t.Fatalf("missing dependency: %v", err)
	}
	cyclic := fixturePublication("demo.assets", digestA, []Declaration{
		{Handle: "demo.assets.one", ContractVersion: "demo.assets.one@1", Type: "script", Path: "one.mjs", Digest: digestB, Dependencies: []string{"demo.assets.two"}},
		{Handle: "demo.assets.two", ContractVersion: "demo.assets.two@1", Type: "script", Path: "two.mjs", Digest: digestC, Dependencies: []string{"demo.assets.one"}},
	})
	if _, err := registry.Publish(cyclic); !errors.Is(err, ErrDependency) {
		t.Fatalf("cyclic dependency: %v", err)
	}
}

func TestRegistryReplaceAndRemoveCleanOwnedHandles(t *testing.T) {
	registry := New()
	if _, err := registry.Publish(fixturePublication("demo.assets", digestA, []Declaration{
		{Handle: "demo.assets.old", ContractVersion: "demo.assets.old@1", Type: "style", Path: "old.css", Digest: digestB},
	})); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Publish(fixturePublication("demo.assets", digestC, []Declaration{
		{Handle: "demo.assets.new", ContractVersion: "demo.assets.new@1", Type: "style", Path: "new.css", Digest: digestC},
	})); err != nil {
		t.Fatal(err)
	}
	if _, ok := registry.Resolve("demo.assets.old"); ok {
		t.Fatal("replacement retained stale handle")
	}
	if _, ok := registry.Resolve("demo.assets.new"); !ok {
		t.Fatal("replacement did not publish new handle")
	}
	if revision, err := registry.Remove("demo.assets"); err != nil || revision != 3 {
		t.Fatalf("remove revision=%d err=%v", revision, err)
	}
	if len(registry.Snapshot().Assets) != 0 {
		t.Fatal("remove retained extension assets")
	}
}

func TestRegistryRemoveRejectsDanglingDependency(t *testing.T) {
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
	if revision, err := registry.Remove("owner.assets"); !errors.Is(err, ErrDependency) || revision != before.Revision {
		t.Fatalf("remove dangling owner: revision=%d err=%v", revision, err)
	}
	if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatalf("failed remove changed snapshot: before=%#v after=%#v", before, after)
	}
}

func TestRegistryReplaceAllConvergesAcrossPublicationOrderAndRestart(t *testing.T) {
	owner := fixturePublication("owner.assets", digestA, []Declaration{{
		Handle: "owner.assets.shared", ContractVersion: "owner.assets.shared@1", Type: "script", Path: "shared.mjs", Digest: digestB,
	}})
	consumer := fixturePublication("consumer.assets", digestB, []Declaration{{
		Handle: "consumer.assets.entry", ContractVersion: "consumer.assets.entry@1", Type: "script", Path: "entry.mjs", Digest: digestC,
		Dependencies: []string{"owner.assets.shared"},
	}})
	first, second := New(), New()
	if revision, err := first.ReplaceAll([]Publication{owner, consumer}); err != nil || revision != 1 {
		t.Fatalf("first replace: revision=%d err=%v", revision, err)
	}
	if revision, err := second.ReplaceAll([]Publication{consumer, owner}); err != nil || revision != 1 {
		t.Fatalf("restart replace: revision=%d err=%v", revision, err)
	}
	if left, right := first.Snapshot(), second.Snapshot(); !reflect.DeepEqual(left, right) {
		t.Fatalf("publication order changed snapshot: left=%#v right=%#v", left, right)
	}
	if revision, err := second.ReplaceAll([]Publication{owner, consumer}); err != nil || revision != 1 {
		t.Fatalf("idempotent restart churned revision: revision=%d err=%v", revision, err)
	}
}

func TestRegistryIdempotentPublicationDoesNotChurnRevision(t *testing.T) {
	registry := New()
	publication := fixturePublication("demo.assets", digestA, []Declaration{{
		Handle: "demo.assets.entry", ContractVersion: "demo.assets.entry@1", Type: "script", Path: "entry.mjs", Digest: digestB,
	}})
	first, err := registry.Publish(publication)
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.Publish(publication)
	if err != nil || second != first {
		t.Fatalf("idempotent publication churned revision: first=%d second=%d err=%v", first, second, err)
	}
}

func TestRegistryValidatesPackagePathIntegrityAndCSP(t *testing.T) {
	tests := []Declaration{
		{Handle: "demo.assets.badpath", ContractVersion: "demo.assets.badpath@1", Type: "script", Path: "../entry.mjs", Digest: digestB},
		{Handle: "demo.assets.badmime", ContractVersion: "demo.assets.badmime@1", Type: "script", Path: "entry.css", Digest: digestB},
		{Handle: "demo.assets.badintegrity", ContractVersion: "demo.assets.badintegrity@1", Type: "script", Path: "entry.mjs", Digest: digestB, Integrity: "sha256-wrong"},
		{Handle: "demo.assets.badcsp", ContractVersion: "demo.assets.badcsp@1", Type: "script", Path: "entry.mjs", Digest: digestB, CSP: []string{"script-src 'self'; default-src *"}},
		{Handle: "demo.assets.badcontract", ContractVersion: "anything", Type: "script", Path: "entry.mjs", Digest: digestB},
		{Handle: "demo.assets.duplicate", ContractVersion: "demo.assets.duplicate@1", Type: "script", Path: "entry.mjs", Digest: digestB, Dependencies: []string{"core.asset.vue", " CORE.ASSET.VUE "}},
	}
	for _, declaration := range tests {
		registry := New()
		if _, err := registry.Publish(fixturePublication("demo.assets", digestA, []Declaration{declaration})); !errors.Is(err, ErrInvalid) {
			t.Fatalf("%s: expected invalid, got %v", declaration.Handle, err)
		}
	}
}

func TestRegistryReturnsDetachedSnapshots(t *testing.T) {
	registry := New()
	if _, err := registry.Publish(fixturePublication("demo.assets", digestA, []Declaration{{
		Handle: "demo.assets.entry", ContractVersion: "demo.assets.entry@1", Type: "script", Path: "entry.mjs", Digest: digestB,
		Dependencies: []string{"core.asset.vue"}, Scope: []string{"forum.component.topic"}, CSP: []string{"connect-src 'self'"},
	}})); err != nil {
		t.Fatal(err)
	}
	snapshot := registry.Snapshot()
	snapshot.Assets[0].Dependencies[0] = "mutated"
	snapshot.Assets[0].Scope[0] = "mutated"
	snapshot.Assets[0].CSP[0] = "mutated"
	asset, ok := registry.Resolve("demo.assets.entry")
	if !ok || asset.Dependencies[0] != "core.asset.vue" || asset.Scope[0] != "forum.component.topic" || asset.CSP[0] != "connect-src 'self'" {
		t.Fatalf("snapshot exposed mutable state: %#v", asset)
	}
}

func fixturePublication(extensionID, impactDigest string, assets []Declaration) Publication {
	return Publication{Artifact: Artifact{
		ExtensionID: extensionID, ExtensionVersion: "1.0.0", PackageDigest: digestA, ImpactDigest: impactDigest,
	}, Assets: assets}
}
