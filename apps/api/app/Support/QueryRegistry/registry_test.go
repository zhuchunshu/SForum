package queryregistry

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestRegistryReplaceAllIsOrderIndependentAndInspectsEmptyPublications(t *testing.T) {
	core := publication("core.query", true, 'a')
	core.Queries = []QueryDeclaration{query("core.query.items", "core.query.item", PaginationCursor, "public")}
	empty := publication("plugin.empty", false, 'b')
	plugin := publication("plugin.query", false, 'c')
	plugin.Queries = []QueryDeclaration{query("plugin.query.items", "plugin.query.item", PaginationOffset, "plugin.query.read")}

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
	if len(left.Publications) != 3 || len(left.Queries) != 2 {
		t.Fatalf("snapshot=%#v", left)
	}
	emptyPub, ok := first.SnapshotPublication("plugin.empty")
	if !ok || len(emptyPub.Queries) != 0 || emptyPub.Artifact.ExtensionID != "plugin.empty" {
		t.Fatalf("empty publication not inspectable=%#v ok=%t", emptyPub, ok)
	}
}

func TestRegistrySafeModeFiltersNonCoreBeforeValidation(t *testing.T) {
	core := publication("core.query", true, 'a')
	core.Queries = []QueryDeclaration{query("core.query.items", "core.query.item", PaginationNone, "public")}
	broken := publication("broken.query", false, 'b')
	broken.Queries = []QueryDeclaration{{
		ID: "not-namespaced", ContractVersion: "x@1", Entity: "e", PlanVersion: "p@1",
		Fields: []string{"id"}, Pagination: "cursor", ResultSchema: "r@1", PermissionPolicy: "public",
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
	if !snapshot.SafeMode || len(snapshot.Publications) != 1 || len(snapshot.Queries) != 1 {
		t.Fatalf("safe mode snapshot=%#v", snapshot)
	}
	if snapshot.Publications[0].Artifact.ExtensionID != "core.query" {
		t.Fatalf("safe mode retained plugin=%#v", snapshot.Publications)
	}
	plugin := publication("plugin.query", false, 'c')
	plugin.Queries = []QueryDeclaration{query("plugin.query.items", "plugin.query.item", PaginationNone, "public")}
	if revision, err := registry.Publish(plugin); !errors.Is(err, ErrSafeMode) || revision != snapshot.Revision {
		t.Fatalf("safe mode plugin publish: revision=%d err=%v", revision, err)
	}
	coreExtra := publication("core.extra", true, 'd')
	coreExtra.Queries = []QueryDeclaration{query("core.extra.items", "core.extra.item", PaginationNone, "public")}
	if _, err := registry.Publish(coreExtra); err != nil {
		t.Fatalf("safe mode rejected Host core publication: %v", err)
	}
}

func TestRegistryPublishCASAndExactRemove(t *testing.T) {
	registry := New()
	initial := publication("plugin.query", false, 'a')
	initial.Queries = []QueryDeclaration{query("plugin.query.items", "plugin.query.item", PaginationOffset, "public")}
	if revision, err := registry.Publish(initial); err != nil || revision != 1 {
		t.Fatalf("publish: revision=%d err=%v", revision, err)
	}
	// Exact artifact replay is idempotent.
	if revision, err := registry.Publish(initial); err != nil || revision != 1 {
		t.Fatalf("idempotent publish: revision=%d err=%v", revision, err)
	}
	drift := initial
	drift.Queries[0].CacheTags = []string{"plugin.query.other"}
	if _, err := registry.Publish(drift); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("same artifact drift=%v", err)
	}

	replacement := publication("plugin.query", false, 'b')
	replacement.Artifact.ExtensionVersion = "2.0.0"
	replacement.Queries = []QueryDeclaration{query("plugin.query.items", "plugin.query.item", PaginationOffset, "public")}
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
}

func TestRegistryReplaceAllRevisionFencePreservesConcurrentPublication(t *testing.T) {
	registry := New()
	base := publication("base.query", false, 'a')
	base.Queries = []QueryDeclaration{query("base.query.items", "base.item", PaginationNone, "public")}
	if _, err := registry.Publish(base); err != nil {
		t.Fatal(err)
	}
	observed := registry.Snapshot()
	concurrent := publication("concurrent.query", false, 'b')
	concurrent.Queries = []QueryDeclaration{query("concurrent.query.items", "concurrent.item", PaginationNone, "public")}
	if _, err := registry.Publish(concurrent); err != nil {
		t.Fatal(err)
	}
	replacement := publication("replacement.query", false, 'c')
	replacement.Queries = []QueryDeclaration{query("replacement.query.items", "replacement.item", PaginationNone, "public")}
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

func TestRegistryReplaceAllRejectsSameArtifactDeclarationDrift(t *testing.T) {
	registry := New()
	active := publication("drift.query", false, 'a')
	active.Queries = []QueryDeclaration{query("drift.query.items", "drift.item", PaginationNone, "public")}
	if _, err := registry.Publish(active); err != nil {
		t.Fatal(err)
	}
	drift := active
	drift.Queries[0].Fields = []string{"changed"}
	before := registry.Snapshot()
	if revision, err := registry.ReplaceAllIfRevision(before.Revision, []Publication{drift}, false); !errors.Is(err, ErrArtifactConflict) || revision != before.Revision {
		t.Fatalf("same artifact declaration drift: revision=%d err=%v", revision, err)
	}
	if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatal("same artifact drift mutated the active graph")
	}
}

func TestRegistryAllowsDistinctQueriesForTheSameEntityPlanContract(t *testing.T) {
	core := publication("core.query", true, 'a')
	list := query("core.query.list", "shared.entity", PaginationCursor, "public")
	detail := query("core.query.detail", "shared.entity", PaginationNone, "public")
	detail.PlanVersion = list.PlanVersion
	detail.Fields = []string{"id", "body"}
	core.Queries = []QueryDeclaration{list, detail}

	registry := New()
	if _, err := registry.Publish(core); err != nil {
		t.Fatalf("same entity plan contract rejected: %v", err)
	}
	for _, queryID := range []string{list.ID, detail.ID} {
		resolved, err := registry.Resolve(queryID)
		if err != nil || resolved.ID != queryID {
			t.Fatalf("resolve %s = %#v, %v", queryID, resolved, err)
		}
	}
}

func TestRegistryRejectsDuplicateQueryID(t *testing.T) {
	dup := publication("dup.query", false, 'c')
	dup.Queries = []QueryDeclaration{
		query("dup.query.items", "dup.entity", PaginationNone, "public"),
		query("dup.query.items", "dup.entity.other", PaginationNone, "public"),
	}
	if _, err := New().ReplaceAll([]Publication{dup}, false); !errors.Is(err, ErrConflict) && !errors.Is(err, ErrInvalid) {
		t.Fatalf("duplicate query id=%v", err)
	}
}

func TestRegistrySnapshotDeepCopyIsolation(t *testing.T) {
	core := publication("core.query", true, 'a')
	core.Queries = []QueryDeclaration{query("core.query.items", "core.item", PaginationOffset, "public")}
	core.Queries[0].CacheTags = []string{"core.items"}
	registry := New()
	if _, err := registry.Publish(core); err != nil {
		t.Fatal(err)
	}
	snapshot := registry.Snapshot()
	snapshot.Queries[0].Fields[0] = "mutated"
	snapshot.Publications[0].Queries[0].CacheTags[0] = "mutated"
	again := registry.Snapshot()
	if again.Queries[0].Fields[0] != "id" || again.Publications[0].Queries[0].CacheTags[0] != "core.items" {
		t.Fatalf("mutation leaked into registry: %#v", again)
	}
}

func TestRegistryAllOrNothingValidation(t *testing.T) {
	registry := New()
	good := publication("core.query", true, 'a')
	good.Queries = []QueryDeclaration{query("core.query.items", "core.item", PaginationNone, "public")}
	if _, err := registry.Publish(good); err != nil {
		t.Fatal(err)
	}
	before := registry.Snapshot()
	bad := publication("bad.query", false, 'b')
	bad.Queries = []QueryDeclaration{{
		ID: "bad.query.items", ContractVersion: "bad.query.items@1", Entity: "bad.item",
		PlanVersion: "not-a-contract", Fields: []string{"id"}, Pagination: "cursor",
		ResultSchema: "bad.query.items.result@1", PermissionPolicy: "public",
	}}
	if _, err := registry.Publish(bad); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected invalid, got %v", err)
	}
	if after := registry.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatalf("failed publish mutated snapshot")
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

func query(id, entity, pagination, policy string) QueryDeclaration {
	return QueryDeclaration{
		ID:               id,
		ContractVersion:  id + "@1",
		Entity:           entity,
		PlanVersion:      id + ".plan@1",
		Fields:           []string{"id", "title"},
		Relations:        []string{"owner"},
		Filters:          []string{"status"},
		Sort:             []string{"created_at"},
		Pagination:       pagination,
		ResultSchema:     id + ".result@1",
		PermissionPolicy: policy,
		CacheTags:        []string{id},
	}
}

func allowAll() PermissionRecheck {
	return PermissionRecheckFunc(func(context.Context, PermissionClaim) error { return nil })
}

func denyAll() PermissionRecheck {
	return PermissionRecheckFunc(func(context.Context, PermissionClaim) error { return ErrDenied })
}

func newPlanningRegistry() *Registry {
	return New(WithCostPolicy(CostPolicyFunc(func(input QueryCostInput) (QueryCost, error) {
		maximum := 2_000
		if input.RequestedMaximum > 0 {
			maximum = input.RequestedMaximum
		}
		units := 10 + len(input.Fields) + len(input.Relations)*8 +
			len(input.Filters)*3 + len(input.Sorts)*2 + input.Pagination.Limit
		return QueryCost{Units: units, Maximum: maximum}, nil
	})))
}
