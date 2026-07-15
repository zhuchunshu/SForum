package assetregistry

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestRegistryRequiresConsistentOwnerKind(t *testing.T) {
	plugin := fixturePublication("owner.plugin", digestA, nil).Artifact
	theme := plugin
	theme.ExtensionID = "owner.theme"
	theme.OwnerKind = " THEME "
	core := plugin
	core.ExtensionID = "core.assets"
	core.OwnerKind = OwnerKindCore
	core.Core = true

	for name, artifact := range map[string]Artifact{
		"plugin": plugin,
		"theme":  theme,
		"core":   core,
	} {
		t.Run(name, func(t *testing.T) {
			normalized, err := normalizeArtifact(artifact)
			if err != nil {
				t.Fatalf("valid %s artifact: %v", name, err)
			}
			if normalized.OwnerKind != name {
				t.Fatalf("owner kind=%q want=%q", normalized.OwnerKind, name)
			}
		})
	}

	invalid := map[string]Artifact{
		"missing kind":           withOwnerKind(plugin, "", false, "owner.plugin"),
		"unknown kind":           withOwnerKind(plugin, "vendor", false, "owner.plugin"),
		"core kind noncore id":   withOwnerKind(plugin, OwnerKindCore, true, "owner.plugin"),
		"core kind without flag": withOwnerKind(core, OwnerKindCore, false, "core.assets"),
		"core flag plugin kind":  withOwnerKind(plugin, OwnerKindPlugin, true, "owner.plugin"),
		"core namespace plugin":  withOwnerKind(plugin, OwnerKindPlugin, false, "core.assets"),
		"core namespace theme":   withOwnerKind(theme, OwnerKindTheme, false, "core.theme"),
	}
	for name, artifact := range invalid {
		t.Run(name, func(t *testing.T) {
			if _, err := normalizeArtifact(artifact); !errors.Is(err, ErrInvalid) {
				t.Fatalf("expected invalid owner identity, got %v", err)
			}
		})
	}
}

func TestRegistryReplaceAllIfRevisionFencesStaleInvalidAndDrift(t *testing.T) {
	registry := New()
	initial := fixturePublication("batch.assets", digestA, []Declaration{
		assetDeclaration("batch.assets.initial", nil),
	})
	if _, err := registry.Publish(initial); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()
	replacement := fixturePublication("batch.assets", digestB, []Declaration{
		assetDeclaration("batch.assets.replacement", nil),
	})
	replacement.Artifact.ExtensionVersion = "2.0.0"
	replacement.Artifact.PackageDigest = digestB

	if revision, err := registry.ReplaceAllIfRevision(before.Revision-1, []Publication{replacement}); !errors.Is(err, ErrRevisionConflict) || revision != before.Revision {
		t.Fatalf("stale replacement: revision=%d err=%v", revision, err)
	}
	invalid := replacement
	invalid.Assets = append([]Declaration(nil), replacement.Assets...)
	invalid.Assets[0].Path = "../replacement.mjs"
	if revision, err := registry.ReplaceAllIfRevision(before.Revision, []Publication{invalid}); !errors.Is(err, ErrInvalid) || revision != before.Revision {
		t.Fatalf("invalid replacement: revision=%d err=%v", revision, err)
	}
	if revision, err := registry.Publish(invalid); !errors.Is(err, ErrInvalid) || revision != before.Revision {
		t.Fatalf("invalid incremental publication: revision=%d err=%v", revision, err)
	}
	drifted := initial
	drifted.Assets = append([]Declaration(nil), initial.Assets...)
	drifted.Assets[0].Path = "drifted.mjs"
	if revision, err := registry.ReplaceAllIfRevision(before.Revision, []Publication{drifted}); !errors.Is(err, ErrArtifactConflict) || revision != before.Revision {
		t.Fatalf("exact artifact drift: revision=%d err=%v", revision, err)
	}
	if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatalf("rejected replacement changed snapshot: before=%#v after=%#v", before, after)
	}
	if revision, err := registry.ReplaceAllIfRevision(before.Revision, []Publication{initial}); err != nil || revision != before.Revision {
		t.Fatalf("exact replay: revision=%d err=%v", revision, err)
	}
	revision, err := registry.ReplaceAllIfRevision(before.Revision, []Publication{replacement})
	if err != nil || revision != before.Revision+1 {
		t.Fatalf("fenced replacement: revision=%d err=%v", revision, err)
	}
	if revision, err = registry.ReplaceAllIfRevision(revision, nil); err != nil || revision != before.Revision+2 {
		t.Fatalf("fenced removal: revision=%d err=%v", revision, err)
	}
	empty := registry.Snapshot()
	if empty.Publications == nil || empty.Assets == nil || len(empty.Publications) != 0 || len(empty.Assets) != 0 {
		t.Fatalf("empty snapshot lost array contract: %#v", empty)
	}
}

func TestRegistrySnapshotPublicationIsDeepCopiedAndEmptyJSONUsesArrays(t *testing.T) {
	body, err := json.Marshal(New().Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	var empty struct {
		Publications json.RawMessage `json:"publications"`
		Assets       json.RawMessage `json:"assets"`
	}
	if err := json.Unmarshal(body, &empty); err != nil {
		t.Fatal(err)
	}
	if string(empty.Publications) != "[]" || string(empty.Assets) != "[]" {
		t.Fatalf("empty snapshot JSON=%s", body)
	}
	emptyOwnerRegistry := New()
	if _, err := emptyOwnerRegistry.Publish(fixturePublication("empty.copy", digestA, nil)); err != nil {
		t.Fatal(err)
	}
	body, err = json.Marshal(emptyOwnerRegistry.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	var withEmptyOwner struct {
		Publications []struct {
			Assets json.RawMessage `json:"assets"`
		} `json:"publications"`
	}
	if err := json.Unmarshal(body, &withEmptyOwner); err != nil {
		t.Fatal(err)
	}
	if len(withEmptyOwner.Publications) != 1 || string(withEmptyOwner.Publications[0].Assets) != "[]" {
		t.Fatalf("empty publication JSON=%s", body)
	}

	registry := New()
	publication := fixturePublication("copy.assets", digestA, []Declaration{{
		Handle: "copy.assets.entry", ContractVersion: "copy.assets.entry@1", Type: "script",
		Path: "entry.mjs", Digest: digestB, Dependencies: []string{"core.asset.vue"},
		Scope: []string{"forum.component.topic"}, CSP: []string{"connect-src 'self'"},
	}})
	if _, err := registry.Publish(publication); err != nil {
		t.Fatal(err)
	}
	detached, ok := registry.SnapshotPublication(" COPY.ASSETS ")
	if !ok {
		t.Fatal("snapshot publication not found")
	}
	detached.Artifact.OwnerKind = OwnerKindTheme
	detached.Assets[0].Dependencies[0] = "forged.asset"
	detached.Assets[0].Scope[0] = "forged.scope"
	detached.Assets[0].CSP[0] = "connect-src https://forged.invalid"
	again, ok := registry.SnapshotPublication("copy.assets")
	if !ok || again.Artifact.OwnerKind != OwnerKindPlugin ||
		again.Assets[0].Dependencies[0] != "core.asset.vue" ||
		again.Assets[0].Scope[0] != "forum.component.topic" ||
		again.Assets[0].CSP[0] != "connect-src 'self'" {
		t.Fatalf("snapshot publication exposed mutable state: %#v", again)
	}
}

func TestRegistryQuarantineExactRemovesTransitiveDependents(t *testing.T) {
	owner := fixturePublication("cascade.root", digestA, []Declaration{
		assetDeclaration("cascade.root.shared", nil),
	})
	direct := fixturePublication("cascade.zdirect", digestB, []Declaration{
		assetDeclaration("cascade.zdirect.entry", []string{"cascade.root.shared"}),
	})
	transitive := fixturePublication("cascade.atransitive", digestC, []Declaration{
		assetDeclaration("cascade.atransitive.entry", []string{"cascade.zdirect.entry"}),
	})
	unrelated := fixturePublication("cascade.unrelated", digestA, []Declaration{
		assetDeclaration("cascade.unrelated.entry", nil),
	})
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{transitive, unrelated, direct, owner}); err != nil {
		t.Fatal(err)
	}
	revision, quarantined, err := registry.QuarantineExact(owner.Artifact)
	if err != nil || revision != 2 {
		t.Fatalf("quarantine: revision=%d artifacts=%#v err=%v", revision, quarantined, err)
	}
	wantIDs := []string{"cascade.atransitive", "cascade.root", "cascade.zdirect"}
	if got := artifactExtensionIDs(quarantined); !reflect.DeepEqual(got, wantIDs) {
		t.Fatalf("quarantined IDs=%v want=%v", got, wantIDs)
	}
	snapshot := registry.Snapshot()
	if len(snapshot.Publications) != 1 || snapshot.Publications[0].Artifact != unrelated.Artifact ||
		len(snapshot.Assets) != 1 || snapshot.Assets[0].Handle != "cascade.unrelated.entry" {
		t.Fatalf("quarantine did not preserve only unrelated publication: %#v", snapshot)
	}
	if _, ok := registry.Resolve("cascade.root.shared"); ok {
		t.Fatal("quarantine retained owner handle")
	}
	if _, ok := registry.SnapshotPublication("cascade.zdirect"); ok {
		t.Fatal("quarantine retained direct dependent publication")
	}
	absent := fixturePublication("absent.assets", digestA, nil).Artifact
	if nextRevision, items, err := registry.QuarantineExact(absent); err != nil || nextRevision != revision || items == nil || len(items) != 0 {
		t.Fatalf("absent quarantine: revision=%d artifacts=%#v err=%v", nextRevision, items, err)
	}
}

func TestRegistryQuarantineExactClosesPublicationDependencyCycle(t *testing.T) {
	alpha := fixturePublication("cycle.alpha", digestA, []Declaration{
		assetDeclaration("cycle.alpha.shared", nil),
		assetDeclaration("cycle.alpha.bridge", []string{"cycle.beta.entry"}),
	})
	beta := fixturePublication("cycle.beta", digestB, []Declaration{
		assetDeclaration("cycle.beta.entry", []string{"cycle.alpha.shared"}),
	})
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{beta, alpha}); err != nil {
		t.Fatalf("valid asset DAG with publication cycle: %v", err)
	}
	_, quarantined, err := registry.QuarantineExact(alpha.Artifact)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := artifactExtensionIDs(quarantined), []string{"cycle.alpha", "cycle.beta"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("quarantined IDs=%v want=%v", got, want)
	}
	if snapshot := registry.Snapshot(); len(snapshot.Publications) != 0 || len(snapshot.Assets) != 0 {
		t.Fatalf("publication cycle survived quarantine: %#v", snapshot)
	}
}

func TestRegistryQuarantineExactFencesStaleArtifactAndIsDeterministic(t *testing.T) {
	initial := fixturePublication("stale.assets", digestA, []Declaration{
		assetDeclaration("stale.assets.initial", nil),
	})
	active := fixturePublication("stale.assets", digestB, []Declaration{
		assetDeclaration("stale.assets.active", nil),
	})
	active.Artifact.ExtensionVersion = "2.0.0"
	active.Artifact.PackageDigest = digestB
	registry := New()
	if _, err := registry.Publish(initial); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.PublishIfArtifact(initial.Artifact, active); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()
	if revision, quarantined, err := registry.QuarantineExact(initial.Artifact); !errors.Is(err, ErrArtifactConflict) || revision != before.Revision || quarantined == nil || len(quarantined) != 0 {
		t.Fatalf("stale quarantine: revision=%d artifacts=%#v err=%v", revision, quarantined, err)
	}
	wrongKind := active.Artifact
	wrongKind.OwnerKind = OwnerKindTheme
	if revision, quarantined, err := registry.QuarantineExact(wrongKind); !errors.Is(err, ErrArtifactConflict) || revision != before.Revision || quarantined == nil || len(quarantined) != 0 {
		t.Fatalf("owner-kind stale quarantine: revision=%d artifacts=%#v err=%v", revision, quarantined, err)
	}
	if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatalf("stale quarantine changed snapshot: before=%#v after=%#v", before, after)
	}

	owner := fixturePublication("deterministic.owner", digestA, []Declaration{
		assetDeclaration("deterministic.owner.shared", nil),
	})
	consumer := fixturePublication("deterministic.consumer", digestB, []Declaration{
		assetDeclaration("deterministic.consumer.entry", []string{"deterministic.owner.shared"}),
	})
	first, second := New(), New()
	if _, err := first.ReplaceAll([]Publication{owner, consumer}); err != nil {
		t.Fatal(err)
	}
	if _, err := second.ReplaceAll([]Publication{consumer, owner}); err != nil {
		t.Fatal(err)
	}
	_, left, leftErr := first.QuarantineExact(owner.Artifact)
	_, right, rightErr := second.QuarantineExact(owner.Artifact)
	if leftErr != nil || rightErr != nil || !reflect.DeepEqual(left, right) {
		t.Fatalf("input order changed quarantine: left=%#v err=%v right=%#v err=%v", left, leftErr, right, rightErr)
	}
}

func withOwnerKind(artifact Artifact, kind string, core bool, extensionID string) Artifact {
	artifact.OwnerKind = kind
	artifact.Core = core
	artifact.ExtensionID = extensionID
	return artifact
}

func assetDeclaration(handle string, dependencies []string) Declaration {
	return Declaration{
		Handle: handle, ContractVersion: handle + "@1", Type: "script",
		Path: handle + ".mjs", Digest: digestB, Dependencies: dependencies,
	}
}

func artifactExtensionIDs(artifacts []Artifact) []string {
	result := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		result = append(result, artifact.ExtensionID)
	}
	return result
}
