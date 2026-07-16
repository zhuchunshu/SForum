package queryregistry

import (
	"context"
	"errors"
	"testing"
)

func TestPlanRequiresReviewedHostCostPolicy(t *testing.T) {
	registry := New()
	core := publication("core.query", true, 'a')
	core.Queries = []QueryDeclaration{query("core.query.items", "core.item", PaginationNone, "public")}
	if _, err := registry.Publish(core); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: "core.query.items", Permission: PermissionInput{},
	}); !errors.Is(err, ErrContractInsufficient) {
		t.Fatalf("planning without reviewed cost policy = %v", err)
	}
}

func TestPlanValidatesShapePaginationCostAndProviders(t *testing.T) {
	registry := newPlanningRegistry()
	core := publication("core.query", true, 'a')
	core.Queries = []QueryDeclaration{query("core.query.items", "core.item", PaginationCursor, "public")}
	if _, err := registry.Publish(core); err != nil {
		t.Fatal(err)
	}

	plan, err := registry.Plan(context.Background(), PlanRequest{
		QueryID:    "core.query.items",
		Fields:     []string{"title", "id"},
		Filters:    []FilterValue{{Field: "status", Value: "open"}},
		Sorts:      []SortValue{{Field: "created_at", Descending: true}},
		Pagination: PaginationRequest{Limit: 10},
		Locale:     "zh-CN",
		Scope:      "public.list",
		Permission: PermissionInput{ActorFingerprint: "anon", PolicyFingerprint: "public"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.ShapeDigest == "" || plan.CacheKey == "" || plan.Cost.Units <= 0 {
		t.Fatalf("plan incomplete=%#v", plan)
	}
	if len(plan.Fields) != 2 || plan.Fields[0] != "title" || plan.Pagination.Limit != 10 {
		t.Fatalf("selection=%#v", plan)
	}
	if plan.Recheck.PermissionPolicy != "public" || plan.Recheck.QueryID != "core.query.items" {
		t.Fatalf("recheck claim=%#v", plan.Recheck)
	}
	if plan.Recheck.Locale != "zh-CN" || plan.Recheck.Scope != "public.list" {
		t.Fatalf("permission claim lost locale/scope=%#v", plan.Recheck)
	}
	if len(plan.Providers) < 2 || plan.Providers[0].Kind != ProviderKindQuery {
		t.Fatalf("providers=%#v", plan.Providers)
	}

	// Unsupported field fails closed.
	if _, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: "core.query.items", Fields: []string{"secret"},
		Permission: PermissionInput{},
	}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unsupported field=%v", err)
	}
	// Cost ceiling.
	if _, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: "core.query.items", Pagination: PaginationRequest{Limit: 50}, MaxCost: 20,
		Permission: PermissionInput{},
	}); !errors.Is(err, ErrCostExceeded) {
		t.Fatalf("cost=%v", err)
	}
	if _, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: "core.query.items", MaxCost: -1, Permission: PermissionInput{},
	}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("negative Host cost maximum=%v", err)
	}
}

func TestPlanRejectsSnapshotSwapInsideCostPolicy(t *testing.T) {
	concurrent := publication("core.concurrent", true, 'b')
	concurrent.Queries = []QueryDeclaration{query("core.concurrent.items", "core.concurrent.item", PaginationNone, "public")}
	var registry *Registry
	registry = New(WithCostPolicy(CostPolicyFunc(func(QueryCostInput) (QueryCost, error) {
		if _, err := registry.Publish(concurrent); err != nil {
			return QueryCost{}, err
		}
		return QueryCost{Units: 1, Maximum: 100}, nil
	})))
	core := publication("core.query", true, 'a')
	core.Queries = []QueryDeclaration{query("core.query.items", "core.item", PaginationNone, "public")}
	if _, err := registry.Publish(core); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: "core.query.items", Permission: PermissionInput{},
	}); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("plan survived snapshot swap inside cost policy: %v", err)
	}
}

func TestPlanPermissionRecheckIsHostOwned(t *testing.T) {
	registry := newPlanningRegistry()
	core := publication("core.query", true, 'a')
	core.Queries = []QueryDeclaration{query("core.query.private", "core.item", PaginationNone, "core.query.read")}
	if _, err := registry.Publish(core); err != nil {
		t.Fatal(err)
	}

	// Fingerprints alone never authorize a permission-key policy.
	if _, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: "core.query.private",
		Permission: PermissionInput{
			Authenticated: true, ActorFingerprint: "user-1", PolicyFingerprint: "core.query.read",
		},
	}); !errors.Is(err, ErrDenied) {
		t.Fatalf("fingerprint-only authorize=%v", err)
	}
	if _, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: "core.query.private",
		Permission: PermissionInput{
			Authenticated: true, ActorFingerprint: "user-1", PolicyFingerprint: "core.query.read",
			Recheck: denyAll(),
		},
	}); !errors.Is(err, ErrDenied) {
		t.Fatalf("denied recheck=%v", err)
	}
	plan, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: "core.query.private",
		Permission: PermissionInput{
			Authenticated: true, ActorFingerprint: "user-1", PolicyFingerprint: "role:admin",
			Recheck: allowAll(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Recheck.PermissionPolicy != "core.query.read" {
		t.Fatalf("policy rewritten=%#v", plan.Recheck)
	}

	// Second recheck before release.
	if err := registry.RecheckBeforeRelease(context.Background(), plan, PermissionInput{
		Authenticated: true, ActorFingerprint: "user-1", PolicyFingerprint: "role:admin", Recheck: allowAll(),
	}); err != nil {
		t.Fatalf("second recheck=%v", err)
	}
	if err := registry.RecheckBeforeRelease(context.Background(), plan, PermissionInput{
		Authenticated: true, ActorFingerprint: "user-1", PolicyFingerprint: "role:admin", Recheck: denyAll(),
	}); !errors.Is(err, ErrDenied) {
		t.Fatalf("second deny=%v", err)
	}
	if err := registry.RecheckBeforeRelease(context.Background(), plan, PermissionInput{
		Authenticated: true, ActorFingerprint: "user-2", PolicyFingerprint: "role:admin", Recheck: allowAll(),
	}); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("different actor released cached plan=%v", err)
	}
	if err := registry.RecheckBeforeRelease(context.Background(), plan, PermissionInput{
		Authenticated: true, ActorFingerprint: "user-1", PolicyFingerprint: "role:member", Recheck: allowAll(),
	}); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("different policy projection released cached plan=%v", err)
	}

	// Login policy requires authenticated Host fact.
	login := publication("core.login", true, 'b')
	login.Queries = []QueryDeclaration{query("core.login.items", "core.login.item", PaginationNone, PermissionPolicyLogin)}
	if _, err := registry.Publish(login); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Plan(context.Background(), PlanRequest{
		QueryID:    "core.login.items",
		Permission: PermissionInput{Authenticated: false, ActorFingerprint: "user-1", PolicyFingerprint: "session:v1", Recheck: allowAll()},
	}); !errors.Is(err, ErrDenied) {
		t.Fatalf("login unauthenticated=%v", err)
	}
	if _, err := registry.Plan(context.Background(), PlanRequest{
		QueryID:    "core.login.items",
		Permission: PermissionInput{Authenticated: true, ActorFingerprint: "user-1", PolicyFingerprint: "session:v1", Recheck: allowAll()},
	}); err != nil {
		t.Fatalf("login authenticated=%v", err)
	}
	for _, permission := range []PermissionInput{
		{Authenticated: true, PolicyFingerprint: "role:admin", Recheck: allowAll()},
		{Authenticated: true, ActorFingerprint: "user-1", Recheck: allowAll()},
	} {
		if _, err := registry.Plan(context.Background(), PlanRequest{
			QueryID: "core.query.private", Permission: permission,
		}); !errors.Is(err, ErrDenied) {
			t.Fatalf("non-public plan accepted empty identity projection=%v", err)
		}
	}
	for _, permission := range []PermissionInput{
		{Authenticated: true, PolicyFingerprint: "session:v1", Recheck: allowAll()},
		{Authenticated: true, ActorFingerprint: "user-1", Recheck: allowAll()},
	} {
		if _, err := registry.Plan(context.Background(), PlanRequest{
			QueryID: "core.login.items", Permission: permission,
		}); !errors.Is(err, ErrDenied) {
			t.Fatalf("login plan accepted empty identity projection=%v", err)
		}
	}
}

func TestPlanCacheKeyIsolatesActorProvidersAndLocale(t *testing.T) {
	registry := newPlanningRegistry()
	core := publication("core.query", true, 'a')
	core.Queries = []QueryDeclaration{query("core.query.items", "core.item", PaginationOffset, "core.query.read")}
	if _, err := registry.Publish(core); err != nil {
		t.Fatal(err)
	}
	base := PlanRequest{
		QueryID:    "core.query.items",
		Fields:     []string{"id"},
		Pagination: PaginationRequest{Limit: 10, Offset: 0},
		Permission: PermissionInput{
			Authenticated: true, ActorFingerprint: "user-a", PolicyFingerprint: "pol-a", Recheck: allowAll(),
		},
		Locale: "en-US",
		Scope:  "forum.main",
	}
	left, err := registry.Plan(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	rightReq := base
	rightReq.Permission.ActorFingerprint = "user-b"
	right, err := registry.Plan(context.Background(), rightReq)
	if err != nil {
		t.Fatal(err)
	}
	if left.CacheKey == right.CacheKey {
		t.Fatal("actor fingerprint did not isolate cache key")
	}
	policyReq := base
	policyReq.Permission.PolicyFingerprint = "pol-b"
	policyPlan, err := registry.Plan(context.Background(), policyReq)
	if err != nil {
		t.Fatal(err)
	}
	if left.CacheKey == policyPlan.CacheKey {
		t.Fatal("policy fingerprint did not isolate cache key")
	}
	localeReq := base
	localeReq.Locale = "zh-CN"
	localePlan, err := registry.Plan(context.Background(), localeReq)
	if err != nil {
		t.Fatal(err)
	}
	if left.CacheKey == localePlan.CacheKey {
		t.Fatal("locale did not isolate cache key")
	}
	pageReq := base
	pageReq.Pagination.Offset = 20
	pagePlan, err := registry.Plan(context.Background(), pageReq)
	if err != nil {
		t.Fatal(err)
	}
	if left.CacheKey == pagePlan.CacheKey {
		t.Fatal("pagination offset did not isolate cache key")
	}
	// Denied private and public must not share keys even with same shape request.
	public := publication("core.public", true, 'c')
	public.Queries = []QueryDeclaration{query("core.public.items", "core.public.item", PaginationOffset, "public")}
	public.Queries[0].Fields = []string{"id"}
	if _, err := registry.Publish(public); err != nil {
		t.Fatal(err)
	}
	publicPlan, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: "core.public.items", Fields: []string{"id"}, Pagination: PaginationRequest{Limit: 10},
		Permission: PermissionInput{ActorFingerprint: "user-a", PolicyFingerprint: "pol-a"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if publicPlan.CacheKey == left.CacheKey {
		t.Fatal("private and public plans shared cache key")
	}
}

func TestPlanRejectsResultFiltersAndSchemaMismatch(t *testing.T) {
	registry := newPlanningRegistry()
	core := publication("core.query", true, 'a')
	core.Queries = []QueryDeclaration{query("core.query.items", "core.item", PaginationOffset, "public")}
	if _, err := registry.Publish(core); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: "core.query.items", ResultFilters: []string{"redact"},
		Permission: PermissionInput{},
	}); !errors.Is(err, ErrContractInsufficient) {
		t.Fatalf("result filters=%v", err)
	}
	if _, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: "core.query.items", ResultSchema: "other.result@1",
		Permission: PermissionInput{},
	}); !errors.Is(err, ErrConflict) {
		t.Fatalf("schema mismatch=%v", err)
	}
}

func TestPlanCursorContinuationFailsClosedUntilHostCodecExists(t *testing.T) {
	registry := newPlanningRegistry()
	core := publication("core.query", true, 'a')
	core.Queries = []QueryDeclaration{query("core.query.items", "core.item", PaginationCursor, "public")}
	if _, err := registry.Publish(core); err != nil {
		t.Fatal(err)
	}
	first, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: "core.query.items", Fields: []string{"id"},
		Pagination: PaginationRequest{Limit: 5},
		Permission: PermissionInput{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Pagination.Mode != PaginationCursor || first.Pagination.Offset != 0 || first.Pagination.Cursor != "" {
		t.Fatalf("first cursor page = %#v", first.Pagination)
	}
	_, err = registry.Plan(context.Background(), PlanRequest{
		QueryID: "core.query.items", Fields: []string{"id"},
		Pagination: PaginationRequest{Limit: 5, Cursor: "executor-owned-token"},
		Permission: PermissionInput{},
	})
	if !errors.Is(err, ErrContractInsufficient) {
		t.Fatalf("unreviewed cursor continuation contract = %v", err)
	}
	if _, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: "core.query.items", Fields: []string{"id"},
		Pagination: PaginationRequest{Limit: 5, Offset: 5},
		Permission: PermissionInput{},
	}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cursor pagination accepted caller offset = %v", err)
	}
}

func TestRecheckBeforeReleaseDetectsSnapshotSwap(t *testing.T) {
	registry := newPlanningRegistry()
	core := publication("core.query", true, 'a')
	core.Queries = []QueryDeclaration{query("core.query.items", "core.item", PaginationNone, "core.query.read")}
	if _, err := registry.Publish(core); err != nil {
		t.Fatal(err)
	}
	plan, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: "core.query.items",
		Permission: PermissionInput{
			Authenticated: true, ActorFingerprint: "user-1", PolicyFingerprint: "role:reader", Recheck: allowAll(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	replacement := publication("core.query", true, 'b')
	replacement.Artifact, err = NewCoreArtifact(
		replacement.Artifact.ExtensionID, "1.0.1", replacement.Artifact.PackageDigest,
	)
	if err != nil {
		t.Fatal(err)
	}
	replacement.Queries = []QueryDeclaration{query("core.query.items", "core.item", PaginationNone, "core.query.read")}
	if _, err := registry.PublishIfArtifact(core.Artifact, replacement); err != nil {
		t.Fatal(err)
	}
	if err := registry.RecheckBeforeRelease(context.Background(), plan, PermissionInput{
		Authenticated: true, ActorFingerprint: "user-1", PolicyFingerprint: "role:reader", Recheck: allowAll(),
	}); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("stale plan release=%v", err)
	}
}

func TestPlanRejectsSnapshotSwapInsidePermissionRecheck(t *testing.T) {
	registry := newPlanningRegistry()
	core := publication("core.query", true, 'a')
	core.Queries = []QueryDeclaration{query("core.query.items", "core.item", PaginationNone, "core.query.read")}
	if _, err := registry.Publish(core); err != nil {
		t.Fatal(err)
	}
	concurrent := publication("core.concurrent", true, 'b')
	concurrent.Queries = []QueryDeclaration{query("core.concurrent.items", "core.concurrent.item", PaginationNone, "public")}

	_, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: "core.query.items",
		Permission: PermissionInput{
			Authenticated: true, ActorFingerprint: "user-1", PolicyFingerprint: "role:reader",
			Recheck: PermissionRecheckFunc(func(context.Context, PermissionClaim) error {
				_, publishErr := registry.Publish(concurrent)
				return publishErr
			}),
		},
	})
	if !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("plan survived snapshot swap inside permission recheck: %v", err)
	}
}

func TestRecheckBeforeReleaseRejectsSnapshotSwapInsidePermissionCallback(t *testing.T) {
	registry := newPlanningRegistry()
	core := publication("core.query", true, 'a')
	core.Queries = []QueryDeclaration{query("core.query.items", "core.item", PaginationNone, "core.query.read")}
	if _, err := registry.Publish(core); err != nil {
		t.Fatal(err)
	}
	plan, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: "core.query.items",
		Permission: PermissionInput{
			Authenticated: true, ActorFingerprint: "user-1", PolicyFingerprint: "role:reader", Recheck: allowAll(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	concurrent := publication("core.concurrent", true, 'b')
	concurrent.Queries = []QueryDeclaration{query("core.concurrent.items", "core.concurrent.item", PaginationNone, "public")}

	err = registry.RecheckBeforeRelease(context.Background(), plan, PermissionInput{
		Authenticated: true, ActorFingerprint: "user-1", PolicyFingerprint: "role:reader",
		Recheck: PermissionRecheckFunc(func(context.Context, PermissionClaim) error {
			_, publishErr := registry.Publish(concurrent)
			return publishErr
		}),
	})
	if !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("release survived snapshot swap inside permission callback: %v", err)
	}
}

func TestRecheckBeforeReleaseRejectsSnapshotSwapInsideCostCallback(t *testing.T) {
	concurrent := publication("plugin.concurrent", false, 'b')
	concurrent.Queries = []QueryDeclaration{query("plugin.concurrent.items", "plugin.concurrent.item", PaginationNone, "public")}
	var registry *Registry
	swap := false
	registry = New(WithCostPolicy(CostPolicyFunc(func(QueryCostInput) (QueryCost, error) {
		if swap {
			if _, err := registry.Publish(concurrent); err != nil {
				return QueryCost{}, err
			}
		}
		return QueryCost{Units: 1, Maximum: 100}, nil
	})))
	core := publication("core.query", true, 'a')
	core.Queries = []QueryDeclaration{query("core.query.items", "core.item", PaginationNone, "public")}
	if _, err := registry.Publish(core); err != nil {
		t.Fatal(err)
	}
	plan, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: "core.query.items", Permission: PermissionInput{},
	})
	if err != nil {
		t.Fatal(err)
	}
	swap = true
	if err := registry.RecheckBeforeRelease(context.Background(), plan, PermissionInput{}); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("release survived snapshot swap inside cost callback: %v", err)
	}
}

func TestRecheckBeforeReleaseRejectsMutatedPlanMaterial(t *testing.T) {
	registry := newPlanningRegistry()
	core := publication("core.query", true, 'a')
	core.Queries = []QueryDeclaration{query("core.query.items", "core.item", PaginationOffset, "core.query.read")}
	if _, err := registry.Publish(core); err != nil {
		t.Fatal(err)
	}
	newPlan := func() QueryPlan {
		plan, err := registry.Plan(context.Background(), PlanRequest{
			QueryID: "core.query.items", Fields: []string{"id"}, Pagination: PaginationRequest{Limit: 10},
			Permission: PermissionInput{
				Authenticated: true, ActorFingerprint: "user-1", PolicyFingerprint: "role:reader", Recheck: allowAll(),
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		return plan
	}
	tests := []struct {
		name   string
		mutate func(*QueryPlan)
	}{
		{name: "field", mutate: func(plan *QueryPlan) { plan.Fields[0] = "title" }},
		{name: "cache tag", mutate: func(plan *QueryPlan) { plan.CacheTags[0] = "other.tag" }},
		{name: "provider", mutate: func(plan *QueryPlan) { plan.Providers[0].Artifact.PackageDigest = "forged" }},
		{name: "shape", mutate: func(plan *QueryPlan) { plan.ShapeDigest = "forged" }},
		{name: "claim", mutate: func(plan *QueryPlan) { plan.Recheck.ShapeDigest = "forged" }},
		{name: "cost", mutate: func(plan *QueryPlan) { plan.Cost.Units++ }},
		{name: "cache key", mutate: func(plan *QueryPlan) { plan.CacheKey = "forged" }},
		{name: "locale", mutate: func(plan *QueryPlan) { plan.Locale = "zh-CN\nforged" }},
		{name: "scope", mutate: func(plan *QueryPlan) { plan.Scope = "INVALID!" }},
		{name: "valid locale drift", mutate: func(plan *QueryPlan) { plan.Locale = "en-US" }},
		{name: "valid scope drift", mutate: func(plan *QueryPlan) { plan.Scope = "forum.other" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := newPlan()
			test.mutate(&plan)
			if err := registry.RecheckBeforeRelease(context.Background(), plan, PermissionInput{
				Authenticated: true, ActorFingerprint: "user-1", PolicyFingerprint: "role:reader", Recheck: allowAll(),
			}); !errors.Is(err, ErrArtifactConflict) {
				t.Fatalf("mutated plan release = %v", err)
			}
		})
	}
}

func TestPlanMutationIsolation(t *testing.T) {
	registry := newPlanningRegistry()
	core := publication("core.query", true, 'a')
	core.Queries = []QueryDeclaration{query("core.query.items", "core.item", PaginationOffset, "public")}
	if _, err := registry.Publish(core); err != nil {
		t.Fatal(err)
	}
	plan, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: "core.query.items", Fields: []string{"id", "title"},
		Permission: PermissionInput{},
	})
	if err != nil {
		t.Fatal(err)
	}
	plan.Fields[0] = "mutated"
	plan.Query.Fields[0] = "mutated"
	again, err := registry.Plan(context.Background(), PlanRequest{
		QueryID: "core.query.items", Fields: []string{"id", "title"},
		Permission: PermissionInput{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if again.Fields[0] != "id" || again.Query.Fields[0] != "id" {
		t.Fatalf("plan mutation leaked=%#v", again)
	}
}
