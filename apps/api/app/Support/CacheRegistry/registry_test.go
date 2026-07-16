package cacheregistry

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestRegistryPublishesDeterministicImmutableSnapshots(t *testing.T) {
	core := testPublication("core.cache", true, 'a')
	core.Caches = []Declaration{testDeclaration("core.cache.posts", PolicyPublic)}
	plugin := testPublication("demo.cache", false, 'b')
	actorCache := testDeclaration("demo.cache.feed", PolicyActor)
	actorCache.Tags = []string{"demo.cache.tag.z", "demo.cache.tag.a"}
	actorCache.Invalidators = []string{"demo.cache.invalidate.z", "demo.cache.invalidate.a"}
	actorCache.Provider = "demo.cache.provider"
	plugin.Caches = []Declaration{actorCache}

	first := New().WithPluginAdmission(func(artifact Artifact) bool { return artifact == plugin.Artifact })
	revision, err := first.ReplaceAll([]Publication{plugin, core}, false)
	if err != nil || revision != 1 {
		t.Fatalf("replace all: revision=%d err=%v", revision, err)
	}
	second := New()
	if _, err := second.ReplaceAll([]Publication{core, plugin}, false); err != nil {
		t.Fatal(err)
	}
	left, right := first.Snapshot(), second.Snapshot()
	if left.Digest == "" || left.Digest != right.Digest {
		t.Fatalf("input order changed digest: %s vs %s", left.Digest, right.Digest)
	}
	if len(left.Publications) != 2 || len(left.Caches) != 2 || left.Caches[0].Namespace > left.Caches[1].Namespace {
		t.Fatalf("snapshot order/content = %#v", left)
	}

	resolved, err := first.Resolve(actorCache.ID)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Artifact != plugin.Artifact || resolved.Policy != PolicyActor || resolved.Provider != actorCache.Provider ||
		!reflect.DeepEqual(resolved.Tags, []string{"demo.cache.tag.a", "demo.cache.tag.z"}) ||
		!reflect.DeepEqual(resolved.Invalidators, []string{"demo.cache.invalidate.a", "demo.cache.invalidate.z"}) {
		t.Fatalf("resolved frozen declaration = %#v", resolved)
	}
	plan, err := first.Plan(PlanRequest{CacheID: actorCache.ID, ActorFingerprint: "actor:42"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Cache.Artifact != plugin.Artifact || plan.Cache.Namespace != actorCache.Namespace ||
		plan.Isolation.Artifact != plugin.Artifact || plan.Isolation.Policy != PolicyActor ||
		plan.Isolation.ActorFingerprint != "actor:42" || plan.Isolation.SegmentDigest == "" {
		t.Fatalf("plan omitted owner/isolation metadata = %#v", plan)
	}

	left.Publications[0].Caches[0].Tags = []string{"mutated"}
	left.Caches[0].Invalidators = []string{"mutated"}
	again := first.Snapshot()
	if reflect.DeepEqual(left, again) || again.Caches[0].Invalidators[0] == "mutated" {
		t.Fatal("snapshot mutation leaked into registry state")
	}
}

func TestRegistryPublishSemanticReplayCASRemoveAndRevisionFence(t *testing.T) {
	registry := New()
	initial := testPublication("demo.cache", false, 'a')
	first := testDeclaration("demo.cache.first", PolicyPrivate)
	first.Tags = []string{"demo.cache.z", "demo.cache.a"}
	second := testDeclaration("demo.cache.second", PolicyPublic)
	initial.Caches = []Declaration{first, second}
	if revision, err := registry.Publish(initial); err != nil || revision != 1 {
		t.Fatalf("publish: revision=%d err=%v", revision, err)
	}

	replay := initial
	replay.Caches = []Declaration{second, first}
	replay.Caches[1].Tags = []string{"demo.cache.a", "demo.cache.z"}
	if revision, err := registry.Publish(replay); err != nil || revision != 1 {
		t.Fatalf("semantic replay: revision=%d err=%v", revision, err)
	}
	drift := initial
	drift.Caches[0].Provider = "demo.cache.other-provider"
	if _, err := registry.Publish(drift); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("same artifact declaration drift = %v", err)
	}

	replacement := testPublication("demo.cache", false, 'b')
	replacement.Artifact.ExtensionVersion = "2.0.0"
	replacement.Artifact.VersionID = 2
	replacement.Artifact.RuntimeInstanceID = "runtime-demo-cache-v2"
	replacement.Caches = []Declaration{testDeclaration("demo.cache.first", PolicyPrivate)}
	if _, err := registry.Publish(replacement); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("artifact replacement without CAS = %v", err)
	}
	if revision, err := registry.PublishIfArtifact(initial.Artifact, replacement); err != nil || revision != 2 {
		t.Fatalf("artifact CAS: revision=%d err=%v", revision, err)
	}
	if revision, err := registry.PublishIfArtifact(initial.Artifact, initial); !errors.Is(err, ErrArtifactConflict) || revision != 2 {
		t.Fatalf("stale artifact CAS: revision=%d err=%v", revision, err)
	}
	if revision, removed, err := registry.Remove(initial.Artifact); !errors.Is(err, ErrArtifactConflict) || removed || revision != 2 {
		t.Fatalf("stale remove: revision=%d removed=%t err=%v", revision, removed, err)
	}

	observed := registry.Revision()
	extra := testPublication("extra.cache", false, 'c')
	extra.Caches = []Declaration{testDeclaration("extra.cache.items", PolicyPublic)}
	if _, err := registry.Publish(extra); err != nil {
		t.Fatal(err)
	}
	if revision, err := registry.ReplaceAllIfRevision(observed, []Publication{replacement}, false); !errors.Is(err, ErrRevisionConflict) || revision != observed+1 {
		t.Fatalf("stale revision CAS: revision=%d err=%v", revision, err)
	}
	if _, found := registry.SnapshotPublication(extra.Artifact.ExtensionID); !found {
		t.Fatal("stale ReplaceAll swallowed concurrent publication")
	}
	if revision, removed, err := registry.Remove(replacement.Artifact); err != nil || !removed || revision != 4 {
		t.Fatalf("exact remove: revision=%d removed=%t err=%v", revision, removed, err)
	}
}

func TestRegistryReplaceAllRejectsExactArtifactDeclarationDrift(t *testing.T) {
	registry := New()
	active := testPublication("drift.cache", false, 'a')
	active.Caches = []Declaration{testDeclaration("drift.cache.items", PolicyPublic)}
	if _, err := registry.Publish(active); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()
	drift := active
	drift.Caches[0].Tags = []string{"drift.cache.changed"}
	if revision, err := registry.ReplaceAllIfRevision(before.Revision, []Publication{drift}, false); !errors.Is(err, ErrArtifactConflict) || revision != before.Revision {
		t.Fatalf("exact artifact drift: revision=%d err=%v", revision, err)
	}
	if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatal("failed exact replay mutated registry")
	}
}

func TestRegistryRemoveFencesEveryExactArtifactField(t *testing.T) {
	active := testPublication("exact.cache", false, 'a')
	active.Caches = []Declaration{testDeclaration("exact.cache.items", PolicyPrivate)}
	registry := New()
	if _, err := registry.Publish(active); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		edit func(*Artifact)
		want error
	}{
		{name: "extension version", edit: func(value *Artifact) { value.ExtensionVersion = "1.0.1" }, want: ErrArtifactConflict},
		{name: "package digest", edit: func(value *Artifact) { value.PackageDigest = strings.Repeat("b", 64) }, want: ErrArtifactConflict},
		{name: "version id", edit: func(value *Artifact) { value.VersionID++ }, want: ErrArtifactConflict},
		{name: "runtime instance", edit: func(value *Artifact) { value.RuntimeInstanceID = "replacement-runtime" }, want: ErrArtifactConflict},
		{name: "core authority", edit: func(value *Artifact) { value.Core = true }, want: ErrInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stale := active.Artifact
			test.edit(&stale)
			revision, removed, err := registry.Remove(stale)
			if !errors.Is(err, test.want) || removed || revision != 1 {
				t.Fatalf("revision=%d removed=%t err=%v", revision, removed, err)
			}
		})
	}
}

func TestRegistryRemovalPreservesImmutablePackageDeclarationSeal(t *testing.T) {
	publication := testPublication("history.cache", false, 'a')
	publication.Caches = []Declaration{testDeclaration("history.cache.items", PolicyPublic)}
	registry := New()
	if _, err := registry.Publish(publication); err != nil {
		t.Fatal(err)
	}
	if _, removed, err := registry.Remove(publication.Artifact); err != nil || !removed {
		t.Fatalf("remove removed=%t err=%v", removed, err)
	}

	drifted := publication
	drifted.Artifact.RuntimeInstanceID = "history-cache-restarted"
	drifted.Caches = []Declaration{cloneDeclaration(publication.Caches[0])}
	drifted.Caches[0].Policy = PolicyPrivate
	if _, err := registry.Publish(drifted); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("removed package declaration drift = %v", err)
	}
	if snapshot := registry.Snapshot(); len(snapshot.Publications) != 0 {
		t.Fatalf("drift changed empty graph = %#v", snapshot)
	}

	restarted := publication
	restarted.Artifact.RuntimeInstanceID = "history-cache-restarted"
	if _, err := registry.Publish(restarted); err != nil {
		t.Fatalf("same package declaration after restart = %v", err)
	}
	if got, found := registry.SnapshotPublication(publication.Artifact.ExtensionID); !found ||
		got.Artifact.RuntimeInstanceID != restarted.Artifact.RuntimeInstanceID {
		t.Fatalf("restarted publication = %#v found=%t", got, found)
	}
}

func testPublication(extensionID string, core bool, digest byte) Publication {
	artifact := Artifact{
		ExtensionID: extensionID, ExtensionVersion: "1.0.0", PackageDigest: strings.Repeat(string(digest), 64),
	}
	if core {
		var err error
		artifact, err = NewCoreArtifact(artifact.ExtensionID, artifact.ExtensionVersion, artifact.PackageDigest)
		if err != nil {
			panic(err)
		}
	} else {
		artifact.VersionID = 1
		artifact.RuntimeInstanceID = "runtime-" + strings.ReplaceAll(extensionID, ".", "-")
	}
	return Publication{Artifact: artifact}
}

func testDeclaration(id, policy string) Declaration {
	return Declaration{
		ID: id, ContractVersion: id + "@1", Namespace: id + ".namespace", Policy: policy,
		Tags: []string{id + ".tag"}, Invalidators: []string{id + ".invalidate"},
	}
}
