package cacheregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestRegistrySafeModeFiltersCorruptThirdPartyBeforeValidation(t *testing.T) {
	core := testPublication("core.cache", true, 'a')
	core.Caches = []Declaration{testDeclaration("core.cache.posts", PolicyPublic)}
	broken := testPublication("broken.cache", false, 'b')
	broken.Caches = []Declaration{{ID: "not-owned", ContractVersion: "BAD", Namespace: "victim.cache", Policy: "shared"}}

	registry := New()
	if _, err := registry.ReplaceAll([]Publication{core, broken}, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ordinary mode accepted corrupt plugin = %v", err)
	}
	revision, err := registry.ReplaceAll([]Publication{broken, core}, true)
	if err != nil || revision != 1 {
		t.Fatalf("safe mode recovery: revision=%d err=%v", revision, err)
	}
	snapshot := registry.Snapshot()
	if !snapshot.SafeMode || len(snapshot.Publications) != 1 || snapshot.Publications[0].Artifact.ExtensionID != "core.cache" {
		t.Fatalf("safe mode snapshot = %#v", snapshot)
	}
	if revision, err := registry.Publish(broken); !errors.Is(err, ErrSafeMode) || revision != snapshot.Revision {
		t.Fatalf("safe mode third-party publish: revision=%d err=%v", revision, err)
	}
}

func TestRegistryCoreAuthorityCannotBeForgedByFlagPrefixOrJSON(t *testing.T) {
	forged := testPublication("core.forged", false, 'a')
	forged.Artifact.Core = true
	forged.Artifact.VersionID = 0
	forged.Artifact.RuntimeInstanceID = ""
	forged.Caches = []Declaration{testDeclaration("core.forged.items", PolicyPublic)}
	if _, err := New().Publish(forged); !errors.Is(err, ErrInvalid) {
		t.Fatalf("flag/prefix forged Core = %v", err)
	}

	trusted := testPublication("core.cache", true, 'b')
	trusted.Caches = []Declaration{testDeclaration("core.cache.items", PolicyPublic)}
	body, err := json.Marshal(trusted)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Publication
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	if _, err := New().Publish(decoded); !errors.Is(err, ErrInvalid) {
		t.Fatalf("JSON-decoded Core retained authority = %v", err)
	}
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{decoded, forged}, true); err != nil {
		t.Fatalf("forged Core blocked Safe Mode = %v", err)
	}
	if snapshot := registry.Snapshot(); !snapshot.SafeMode || len(snapshot.Publications) != 0 {
		t.Fatalf("Safe Mode retained unsealed Core = %#v", snapshot)
	}
	if revision, err := registry.Publish(decoded); !errors.Is(err, ErrSafeMode) || revision != 1 {
		t.Fatalf("Safe Mode JSON fake publish: revision=%d err=%v", revision, err)
	}
}

func TestRegistryValidatesFrozenCacheDeclarationBounds(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Publication)
	}{
		{name: "bad extension version", edit: func(p *Publication) { p.Artifact.ExtensionVersion = "v1" }},
		{name: "bad package digest", edit: func(p *Publication) { p.Artifact.PackageDigest = "bad" }},
		{name: "missing version id", edit: func(p *Publication) { p.Artifact.VersionID = 0 }},
		{name: "empty runtime", edit: func(p *Publication) { p.Artifact.RuntimeInstanceID = "" }},
		{name: "runtime control", edit: func(p *Publication) { p.Artifact.RuntimeInstanceID = "runtime\nforged" }},
		{name: "cache id not owned", edit: func(p *Publication) { p.Caches[0].ID = "other.cache.items" }},
		{name: "bad contract", edit: func(p *Publication) { p.Caches[0].ContractVersion = "cache@0" }},
		{name: "long contract", edit: func(p *Publication) {
			p.Caches[0].ContractVersion = strings.Repeat("a", maxContractVersionLength) + "@1"
		}},
		{name: "namespace takeover", edit: func(p *Publication) { p.Caches[0].Namespace = "victim.cache.items" }},
		{name: "bad policy", edit: func(p *Publication) { p.Caches[0].Policy = "shared" }},
		{name: "bad tag", edit: func(p *Publication) { p.Caches[0].Tags = []string{"not/valid"} }},
		{name: "tag namespace escape", edit: func(p *Publication) { p.Caches[0].Tags = []string{"other.cache.tag"} }},
		{name: "duplicate tag", edit: func(p *Publication) { p.Caches[0].Tags = []string{"demo.cache.tag", "demo.cache.tag"} }},
		{name: "bad invalidator", edit: func(p *Publication) { p.Caches[0].Invalidators = []string{"x"} }},
		{name: "invalidator namespace escape", edit: func(p *Publication) { p.Caches[0].Invalidators = []string{"other.cache.invalidate"} }},
		{name: "bad provider", edit: func(p *Publication) { p.Caches[0].Provider = "provider/value" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			publication := testPublication("demo.cache", false, 'a')
			publication.Caches = []Declaration{testDeclaration("demo.cache.items", PolicyPrivate)}
			test.edit(&publication)
			if _, err := New().Publish(publication); !errors.Is(err, ErrInvalid) {
				t.Fatalf("expected invalid, got %v", err)
			}
		})
	}

	publication := testPublication("limit.cache", false, 'b')
	publication.Caches = make([]Declaration, maxCachesPerPublication+1)
	for index := range publication.Caches {
		publication.Caches[index] = testDeclaration(fmt.Sprintf("limit.cache.item.%03d", index), PolicyPrivate)
	}
	if _, err := New().Publish(publication); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cache count overflow = %v", err)
	}

	tags := make([]string, maxTagsPerCache+1)
	for index := range tags {
		tags[index] = fmt.Sprintf("limit.cache.tag.%03d", index)
	}
	publication = testPublication("limit.cache", false, 'c')
	declaration := testDeclaration("limit.cache.items", PolicyPrivate)
	declaration.Tags = tags
	publication.Caches = []Declaration{declaration}
	if _, err := New().Publish(publication); !errors.Is(err, ErrInvalid) {
		t.Fatalf("tag count overflow = %v", err)
	}
}

func TestRegistryRejectsDuplicateIDsNamespacesAndTakeover(t *testing.T) {
	duplicateID := testPublication("dup.cache", false, 'a')
	first := testDeclaration("dup.cache.items", PolicyPrivate)
	second := first
	second.Namespace = "dup.cache.other"
	duplicateID.Caches = []Declaration{first, second}
	if _, err := New().Publish(duplicateID); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate id = %v", err)
	}

	duplicateNamespace := testPublication("dup.cache", false, 'b')
	second = testDeclaration("dup.cache.other", PolicyPublic)
	second.Namespace = first.Namespace
	duplicateNamespace.Caches = []Declaration{second, first}
	if _, err := New().Publish(duplicateNamespace); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate namespace = %v", err)
	}

	owner := testPublication("owner.cache", false, 'c')
	owner.Caches = []Declaration{testDeclaration("owner.cache.items", PolicyPrivate)}
	attacker := testPublication("attacker.cache", false, 'd')
	takeover := testDeclaration("attacker.cache.items", PolicyPrivate)
	takeover.Namespace = owner.Caches[0].Namespace
	attacker.Caches = []Declaration{takeover}
	registry := New()
	before := registry.Snapshot()
	if _, err := registry.ReplaceAll([]Publication{attacker, owner}, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("namespace takeover = %v", err)
	}
	if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatal("namespace takeover mutated graph")
	}

	parent := testPublication("tree.cache", false, 'e')
	child := testPublication("tree.cache.child", false, 'f')
	parentCache := testDeclaration("tree.cache.child.items", PolicyPrivate)
	parentCache.Namespace = "tree.cache.parent.namespace"
	childCache := testDeclaration("tree.cache.child.items", PolicyPublic)
	childCache.Namespace = "tree.cache.child.namespace"
	parent.Caches, child.Caches = []Declaration{parentCache}, []Declaration{childCache}
	assertDeterministicConflict(t, []Publication{parent, child}, []Publication{child, parent}, "duplicate cache id")

	parentCache.ID = "tree.cache.parent.items"
	parentCache.Namespace = "tree.cache.child.shared"
	childCache.ID = "tree.cache.child.items"
	childCache.Namespace = parentCache.Namespace
	parent.Caches, child.Caches = []Declaration{parentCache}, []Declaration{childCache}
	assertDeterministicConflict(t, []Publication{parent, child}, []Publication{child, parent}, "duplicate cache namespace")
}

func assertDeterministicConflict(t *testing.T, first, second []Publication, contains string) {
	t.Helper()
	_, firstErr := New().ReplaceAll(first, false)
	_, secondErr := New().ReplaceAll(second, false)
	if !errors.Is(firstErr, ErrConflict) || !errors.Is(secondErr, ErrConflict) || firstErr.Error() != secondErr.Error() ||
		!strings.Contains(firstErr.Error(), contains) {
		t.Fatalf("non-deterministic conflict: first=%v second=%v", firstErr, secondErr)
	}
}

func TestRegistryPublicationAndFingerprintBoundaries(t *testing.T) {
	publications := make([]Publication, maxPublications+1)
	for index := range publications {
		id := fmt.Sprintf("bound.cache.%03d", index)
		publications[index] = testPublication(id, false, byte('a'+index%6))
	}
	if _, err := New().ReplaceAll(publications, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("publication overflow = %v", err)
	}
	actor := testPublication("actor.cache", false, 'e')
	actor.Caches = []Declaration{testDeclaration("actor.cache.items", PolicyActor)}
	registry := New().WithPluginAdmission(func(artifact Artifact) bool { return artifact == actor.Artifact })
	if _, err := registry.Publish(actor); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Plan(PlanRequest{
		CacheID: actor.Caches[0].ID, ActorFingerprint: strings.Repeat("a", maxFingerprintLength+1),
	}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("fingerprint overflow = %v", err)
	}
}
