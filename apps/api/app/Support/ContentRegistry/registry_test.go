package contentregistry

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestRegistryReplaceAllIsOrderIndependentAndListsByKind(t *testing.T) {
	core := publication("core.content", true, 'a')
	core.Content = []Declaration{
		content("core.content.block.card", KindBlock, "content.block", "core.content.block.card.schema@1"),
		content("core.content.sanitizer.default", KindSanitizer, "content.sanitize", "core.content.sanitizer.default.schema@1"),
	}
	empty := publication("plugin.empty", false, 'b')
	plugin := publication("plugin.content", false, 'c')
	plugin.Content = []Declaration{
		content("plugin.content.shortcode.vote", KindShortcode, "content.shortcode", "plugin.content.shortcode.vote.schema@1"),
		{
			ID: "plugin.content.embed.video", ContractVersion: "plugin.content.embed.video@1",
			Kind: KindEmbed, Schema: "schemas/embed.json", Renderer: "plugin.content.template.video",
			Migration: "plugin.content.migration.embed-v1",
		},
	}

	first := New()
	revision, err := first.ReplaceAll([]Publication{plugin, empty, core}, false)
	if err != nil || revision != 1 {
		t.Fatalf("replace all: revision=%d err=%v", revision, err)
	}
	second := New()
	if _, err := second.ReplaceAll([]Publication{core, plugin, empty}, false); err != nil {
		t.Fatal(err)
	}
	left, right := first.Snapshot(), second.Snapshot()
	if left.Digest != right.Digest || left.Digest == "" {
		t.Fatalf("digest order dependent: %s vs %s", left.Digest, right.Digest)
	}
	if left.SchemaVersion != SchemaVersion || len(left.Publications) != 3 || len(left.Content) != 4 {
		t.Fatalf("snapshot=%#v", left)
	}
	// Deterministic listing: kind then id.
	if left.Content[0].Kind != KindBlock || left.Content[1].Kind != KindEmbed ||
		left.Content[2].Kind != KindSanitizer || left.Content[3].Kind != KindShortcode {
		t.Fatalf("content order=%#v", left.Content)
	}

	emptyPub, ok := first.SnapshotPublication("plugin.empty")
	if !ok || len(emptyPub.Content) != 0 || emptyPub.Artifact.ExtensionID != "plugin.empty" {
		t.Fatalf("empty publication not inspectable=%#v ok=%t", emptyPub, ok)
	}
	blocks := first.List(KindBlock)
	if len(blocks) != 1 || blocks[0].ID != "core.content.block.card" {
		t.Fatalf("list block=%#v", blocks)
	}
	all := first.List("")
	if len(all) != 4 {
		t.Fatalf("list all=%#v", all)
	}
	if first.List("unknown-kind") != nil {
		t.Fatal("invalid kind list should return nil")
	}

	resolved, err := first.Resolve("plugin.content.embed.video")
	if err != nil || resolved.Renderer != "plugin.content.template.video" ||
		resolved.Migration != "plugin.content.migration.embed-v1" ||
		resolved.Schema != "schemas/embed.json" || resolved.Artifact.RuntimeInstanceID == "" {
		t.Fatalf("resolve embed=%#v err=%v", resolved, err)
	}
}

func TestRegistrySafeModeFiltersNonCoreBeforeValidation(t *testing.T) {
	core := publication("core.content", true, 'a')
	core.Content = []Declaration{content("core.content.block.card", KindBlock, "content.block", "core.content.block.card.schema@1")}
	broken := publication("broken.content", false, 'b')
	broken.Content = []Declaration{{
		ID: "not-namespaced", ContractVersion: "x@1", Kind: KindBlock,
		Handler: "h", Schema: "s@1",
	}}

	registry := New()
	if _, err := registry.ReplaceAll([]Publication{core, broken}, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected invalid broken publication, got %v", err)
	}
	revision, err := registry.ReplaceAll([]Publication{core, broken}, true)
	if err != nil || revision != 1 {
		t.Fatalf("safe mode: revision=%d err=%v", revision, err)
	}
	snapshot := registry.Snapshot()
	if !snapshot.SafeMode || len(snapshot.Publications) != 1 || len(snapshot.Content) != 1 {
		t.Fatalf("safe mode snapshot=%#v", snapshot)
	}
	if snapshot.Publications[0].Artifact.ExtensionID != "core.content" {
		t.Fatalf("safe mode retained plugin=%#v", snapshot.Publications)
	}
	plugin := publication("plugin.content", false, 'c')
	plugin.Content = []Declaration{content("plugin.content.block.x", KindBlock, "h", "plugin.content.block.x.schema@1")}
	if revision, err := registry.Publish(plugin); !errors.Is(err, ErrSafeMode) || revision != snapshot.Revision {
		t.Fatalf("safe mode plugin publish: revision=%d err=%v", revision, err)
	}
	coreExtra := publication("core.extra", true, 'd')
	coreExtra.Content = []Declaration{content("core.extra.mark.bold", KindMark, "content.mark", "core.extra.mark.bold.schema@1")}
	if _, err := registry.Publish(coreExtra); err != nil {
		t.Fatalf("safe mode rejected Host core publication: %v", err)
	}
}

func TestRegistryPublishCASAndExactRemove(t *testing.T) {
	registry := New()
	initial := publication("plugin.content", false, 'a')
	initial.Content = []Declaration{content("plugin.content.block.card", KindBlock, "content.block", "plugin.content.block.card.schema@1")}
	if revision, err := registry.Publish(initial); err != nil || revision != 1 {
		t.Fatalf("publish: revision=%d err=%v", revision, err)
	}
	// Exact artifact replay is idempotent even when declaration order differs.
	replay := publication("plugin.content", false, 'a')
	replay.Content = []Declaration{
		content("plugin.content.node.heading", KindNode, "content.node", "plugin.content.node.heading.schema@1"),
		content("plugin.content.block.card", KindBlock, "content.block", "plugin.content.block.card.schema@1"),
	}
	// Different declaration set with same artifact must fail closed.
	if _, err := registry.Publish(replay); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("same artifact drift=%v", err)
	}
	if revision, err := registry.Publish(initial); err != nil || revision != 1 {
		t.Fatalf("idempotent publish: revision=%d err=%v", revision, err)
	}
	drift := initial
	drift.Content[0].Renderer = "plugin.content.template.other"
	if _, err := registry.Publish(drift); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("same artifact renderer drift=%v", err)
	}

	replacement := publication("plugin.content", false, 'b')
	replacement.Artifact.ExtensionVersion = "2.0.0"
	replacement.Artifact.VersionID = 2
	replacement.Artifact.RuntimeInstanceID = "runtime-plugin.content-v2"
	replacement.Content = []Declaration{content("plugin.content.block.card", KindBlock, "content.block", "plugin.content.block.card.schema@1")}
	if _, err := registry.Publish(replacement); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("artifact change without CAS=%v", err)
	}
	if revision, err := registry.PublishIfArtifact(initial.Artifact, replacement); err != nil || revision != 2 {
		t.Fatalf("cas publish: revision=%d err=%v", revision, err)
	}
	stale := initial.Artifact
	stale.PackageDigest = strings.Repeat("f", 64)
	if revision, removed, err := registry.Remove(stale); !errors.Is(err, ErrArtifactConflict) || removed || revision != 2 {
		t.Fatalf("stale remove: revision=%d removed=%t err=%v", revision, removed, err)
	}
	if revision, removed, err := registry.Remove(replacement.Artifact); err != nil || !removed || revision != 3 {
		t.Fatalf("exact remove: revision=%d removed=%t err=%v", revision, removed, err)
	}
	if _, err := registry.Resolve("plugin.content.block.card"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("resolve after remove=%v", err)
	}
}

func TestRegistryReplaceAllRevisionFencePreservesConcurrentPublication(t *testing.T) {
	registry := New()
	base := publication("base.content", false, 'a')
	base.Content = []Declaration{content("base.content.block.a", KindBlock, "h", "base.content.block.a.schema@1")}
	if _, err := registry.Publish(base); err != nil {
		t.Fatal(err)
	}
	observed := registry.Snapshot()
	concurrent := publication("concurrent.content", false, 'b')
	concurrent.Content = []Declaration{content("concurrent.content.block.b", KindBlock, "h", "concurrent.content.block.b.schema@1")}
	if _, err := registry.Publish(concurrent); err != nil {
		t.Fatal(err)
	}
	replacement := publication("replacement.content", false, 'c')
	replacement.Content = []Declaration{content("replacement.content.block.c", KindBlock, "h", "replacement.content.block.c.schema@1")}
	if revision, err := registry.ReplaceAllIfRevision(
		observed.Revision, []Publication{replacement}, false,
	); !errors.Is(err, ErrRevisionConflict) || revision != observed.Revision+1 {
		t.Fatalf("stale full replacement: revision=%d err=%v", revision, err)
	}
	if _, found := registry.SnapshotPublication(concurrent.Artifact.ExtensionID); !found {
		t.Fatal("stale ReplaceAll swallowed a concurrent publication")
	}
	if _, found := registry.SnapshotPublication(replacement.Artifact.ExtensionID); found {
		t.Fatal("stale ReplaceAll published its replacement graph")
	}
}

func TestRegistryReplaceAllConvenienceCannotReplayStaleGraph(t *testing.T) {
	registry := New()
	initial := publication("stale.content", false, 'a')
	initial.Content = []Declaration{
		content("stale.content.block.card", KindBlock, "content.card", "stale.content.block.card.schema@1"),
	}
	if revision, err := registry.ReplaceAll([]Publication{initial}, false); err != nil || revision != 1 {
		t.Fatalf("initial ReplaceAll() revision=%d error=%v", revision, err)
	}
	replacement := publication("stale.content", false, 'b')
	replacement.Artifact.ExtensionVersion = "2.0.0"
	replacement.Artifact.VersionID = 2
	replacement.Artifact.RuntimeInstanceID = "runtime-stale-v2"
	replacement.Content = []Declaration{initial.Content[0]}
	if _, err := registry.PublishIfArtifact(initial.Artifact, replacement); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()
	if revision, err := registry.ReplaceAll([]Publication{initial}, false); !errors.Is(err, ErrRevisionConflict) || revision != before.Revision {
		t.Fatalf("stale convenience replay revision=%d error=%v", revision, err)
	}
	active, found := registry.SnapshotPublication("stale.content")
	if !found || active.Artifact != replacement.Artifact {
		t.Fatalf("stale replay replaced active artifact = %#v found=%t", active, found)
	}
}

func TestRegistryFullGraphCannotOmitCorePublication(t *testing.T) {
	registry := New()
	core := publication("core.content", true, 'a')
	core.Content = []Declaration{
		content("core.content.block.card", KindBlock, "content.card", "core.content.block.card.schema@1"),
	}
	if _, err := registry.ReplaceAll([]Publication{core}, false); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()
	if revision, err := registry.ReplaceAllIfRevision(before.Revision, nil, false); !errors.Is(err, ErrArtifactConflict) || revision != before.Revision {
		t.Fatalf("Core omission revision=%d error=%v", revision, err)
	}
	if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatal("rejected Core omission mutated snapshot")
	}
	if revision, removed, err := registry.Remove(core.Artifact); err != nil || !removed || revision != before.Revision+1 {
		t.Fatalf("exact Core remove revision=%d removed=%t error=%v", revision, removed, err)
	}
}

func TestRegistryReplaceAllRejectsSameArtifactDeclarationDrift(t *testing.T) {
	registry := New()
	active := publication("drift.content", false, 'a')
	active.Content = []Declaration{content("drift.content.block.a", KindBlock, "h", "drift.content.block.a.schema@1")}
	if _, err := registry.Publish(active); err != nil {
		t.Fatal(err)
	}
	drift := active
	drift.Content[0].Handler = "changed.handler"
	before := registry.Snapshot()
	if revision, err := registry.ReplaceAllIfRevision(before.Revision, []Publication{drift}, false); !errors.Is(err, ErrArtifactConflict) || revision != before.Revision {
		t.Fatalf("same artifact declaration drift: revision=%d err=%v", revision, err)
	}
	if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatal("same artifact drift mutated the active graph")
	}
}

func TestRegistryRejectsDuplicateContentID(t *testing.T) {
	dup := publication("dup.content", false, 'c')
	dup.Content = []Declaration{
		content("dup.content.block.a", KindBlock, "h1", "dup.content.block.a.schema@1"),
		content("dup.content.block.a", KindShortcode, "h2", "dup.content.block.a.schema@1"),
	}
	if _, err := New().ReplaceAll([]Publication{dup}, false); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate content id=%v", err)
	}

	left := publication("left.content", false, 'a')
	left.Content = []Declaration{content("left.content.block.a", KindBlock, "h", "left.content.block.a.schema@1")}
	// Same extension id cannot appear twice in one ReplaceAll graph.
	cross := publication("left.content", false, 'b')
	cross.Content = []Declaration{content("left.content.block.b", KindBlock, "h", "left.content.block.b.schema@1")}
	if _, err := New().ReplaceAll([]Publication{left, cross}, false); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate publication extension=%v", err)
	}
}

func TestRegistrySnapshotDeepCopyIsolation(t *testing.T) {
	core := publication("core.content", true, 'a')
	core.Content = []Declaration{content("core.content.block.card", KindBlock, "content.block", "core.content.block.card.schema@1")}
	core.Content[0].Renderer = "core.content.template.card"
	registry := New()
	if _, err := registry.Publish(core); err != nil {
		t.Fatal(err)
	}
	snapshot := registry.Snapshot()
	snapshot.Content[0].Handler = "mutated"
	snapshot.Publications[0].Content[0].Renderer = "mutated"
	list := registry.List(KindBlock)
	list[0].Handler = "mutated-list"
	again := registry.Snapshot()
	if again.Content[0].Handler != "content.block" || again.Publications[0].Content[0].Renderer != "core.content.template.card" {
		t.Fatalf("mutation leaked into registry: %#v", again)
	}
	if resolved, err := registry.Resolve("core.content.block.card"); err != nil || resolved.Handler != "content.block" {
		t.Fatalf("resolve leaked mutation=%#v err=%v", resolved, err)
	}
}

func TestRegistryAllOrNothingValidation(t *testing.T) {
	registry := New()
	good := publication("core.content", true, 'a')
	good.Content = []Declaration{content("core.content.block.card", KindBlock, "content.block", "core.content.block.card.schema@1")}
	if _, err := registry.Publish(good); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()
	bad := publication("bad.content", false, 'b')
	bad.Content = []Declaration{{
		ID: "bad.content.block", ContractVersion: "not-a-contract", Kind: KindBlock,
		Handler: "h", Schema: "bad.content.block.schema@1",
	}}
	if _, err := registry.Publish(bad); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected invalid, got %v", err)
	}
	if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatalf("failed publish mutated snapshot")
	}
}

func TestRegistryPreservesSchemaRendererMigrationReferences(t *testing.T) {
	plugin := publication("demo.content", false, 'a')
	plugin.Content = []Declaration{{
		ID: "demo.content.block.product", ContractVersion: "demo.content.block.product@1",
		Kind: KindBlock, Handler: "content.block.product",
		Schema:   "demo.content.block.product.schema@1",
		Renderer: "demo.content.template.product", Migration: "demo.content.migration.product-v1",
	}}
	registry := New()
	if _, err := registry.Publish(plugin); err != nil {
		t.Fatal(err)
	}
	got, err := registry.Resolve("demo.content.block.product")
	if err != nil {
		t.Fatal(err)
	}
	if got.Schema != "demo.content.block.product.schema@1" ||
		got.Renderer != "demo.content.template.product" ||
		got.Migration != "demo.content.migration.product-v1" ||
		got.Handler != "content.block.product" {
		t.Fatalf("lost frozen references=%#v", got)
	}
}

func publication(extensionID string, core bool, digest byte) Publication {
	artifact := Artifact{
		ExtensionID: extensionID, ExtensionVersion: "1.0.0",
		PackageDigest: strings.Repeat(string(digest), 64),
	}
	if core {
		var err error
		artifact, err = NewCoreArtifact(artifact.ExtensionID, artifact.ExtensionVersion, artifact.PackageDigest)
		if err != nil {
			panic(err)
		}
	} else {
		artifact.VersionID = 1
		artifact.RuntimeInstanceID = "runtime-" + extensionID
	}
	return Publication{Artifact: artifact}
}

func content(id, kind, handler, schema string) Declaration {
	return Declaration{
		ID:              id,
		ContractVersion: id + "@1",
		Kind:            kind,
		Handler:         handler,
		Schema:          schema,
	}
}
