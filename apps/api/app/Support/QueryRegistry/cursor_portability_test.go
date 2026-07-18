package queryregistry

import (
	"context"
	"strings"
	"testing"
)

func TestCursorContinuesAcrossConvergedRegistriesWithDifferentLocalRevisions(t *testing.T) {
	key := []byte(strings.Repeat("portable-query-cursor-key-", 2))
	codecA, err := NewHMACCursorCodec(key)
	if err != nil {
		t.Fatal(err)
	}
	codecB, err := NewHMACCursorCodec(key)
	if err != nil {
		t.Fatal(err)
	}
	cost := WithCostPolicy(CostPolicyFunc(func(QueryCostInput) (QueryCost, error) {
		return QueryCost{Units: 1, Maximum: 100}, nil
	}))
	registryA := New(cost, WithCursorCodec(codecA))
	registryB := New(cost, WithCursorCodec(codecB))

	active := publication("core.portable", true, 'a')
	active.Queries = []QueryDeclaration{query("core.portable.items", "core.portable.item", PaginationCursor, PermissionPolicyPublic)}
	if _, err := registryA.Publish(active); err != nil {
		t.Fatal(err)
	}
	transient := publication("core.transient", true, 'b')
	transient.Queries = []QueryDeclaration{query("core.transient.items", "core.transient.item", PaginationNone, PermissionPolicyPublic)}
	if _, err := registryB.Publish(transient); err != nil {
		t.Fatal(err)
	}
	if _, _, err := registryB.Remove(transient.Artifact); err != nil {
		t.Fatal(err)
	}
	if _, err := registryB.Publish(active); err != nil {
		t.Fatal(err)
	}
	if registryA.Snapshot().Revision == registryB.Snapshot().Revision || registryA.Snapshot().Digest != registryB.Snapshot().Digest {
		t.Fatal("test requires equal graphs reached through different local revisions")
	}

	first, err := registryA.Plan(context.Background(), PlanRequest{
		QueryID: active.Queries[0].ID, Pagination: PaginationRequest{Limit: 2}, Permission: PermissionInput{},
	})
	if err != nil {
		t.Fatal(err)
	}
	cursor, err := registryA.EncodeNextCursor(first, strings.Repeat("c", 64), strings.Repeat("d", 64))
	if err != nil {
		t.Fatal(err)
	}
	continued, err := registryB.Plan(context.Background(), PlanRequest{
		QueryID: active.Queries[0].ID, Pagination: PaginationRequest{Cursor: cursor}, Permission: PermissionInput{},
	})
	if err != nil {
		t.Fatalf("portable cursor rejected by converged registry: %v", err)
	}
	if continued.Pagination.Offset != 2 || continued.Pagination.Limit != 2 {
		t.Fatalf("continued pagination = %#v", continued.Pagination)
	}
}
