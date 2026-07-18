package queryregistry

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type memoryQueryResultCache struct {
	mu      sync.Mutex
	entries map[string]CachedQueryResult
	tags    map[string][]string
	loads   int
	stores  int
}

func newMemoryQueryResultCache() *memoryQueryResultCache {
	return &memoryQueryResultCache{entries: map[string]CachedQueryResult{}, tags: map[string][]string{}}
}

func (c *memoryQueryResultCache) LoadQueryResult(_ context.Context, key string) (CachedQueryResult, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loads++
	value, ok := c.entries[key]
	return value, ok, nil
}

func (c *memoryQueryResultCache) StoreQueryResult(_ context.Context, key string, value CachedQueryResult, tags []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stores++
	c.entries[key] = value
	c.tags[key] = slices.Clone(tags)
	return nil
}

func TestExecutionRechecksPermissionBeforeProviderAndRelease(t *testing.T) {
	var calls atomic.Int32
	var providerCalls atomic.Int32
	permission := PermissionInput{
		Authenticated: true, ActorFingerprint: "user-1", PolicyFingerprint: "role:reader",
		Recheck: PermissionRecheckFunc(func(context.Context, PermissionClaim) error {
			if calls.Add(1) == 2 {
				return ErrDenied
			}
			return nil
		}),
	}
	runtime, _ := executionTestRuntime(t, PaginationOffset, "core.execute.read", ExecutableProviderFunc(
		func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
			providerCalls.Add(1)
			return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "private"}}}, nil
		},
	), nil, nil)
	_, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items", Permission: permission})
	if !errors.Is(err, ErrDenied) || providerCalls.Load() != 0 || calls.Load() != 2 {
		t.Fatalf("permission fence: err=%v provider=%d checks=%d", err, providerCalls.Load(), calls.Load())
	}

	calls.Store(0)
	permission.Recheck = PermissionRecheckFunc(func(context.Context, PermissionClaim) error {
		calls.Add(1)
		return nil
	})
	result, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items", Permission: permission})
	if err != nil || len(result.Rows) != 1 || providerCalls.Load() != 1 || calls.Load() < 3 {
		t.Fatalf("allowed execution: result=%#v err=%v provider=%d checks=%d", result, err, providerCalls.Load(), calls.Load())
	}

	calls.Store(0)
	permission.Recheck = PermissionRecheckFunc(func(context.Context, PermissionClaim) error {
		if calls.Add(1) == 3 {
			return ErrDenied
		}
		return nil
	})
	result, err = runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items", Permission: permission})
	if !errors.Is(err, ErrDenied) || len(result.Rows) != 0 || providerCalls.Load() != 2 || calls.Load() != 3 {
		t.Fatalf("release permission fence: result=%#v err=%v provider=%d checks=%d",
			result, err, providerCalls.Load(), calls.Load())
	}
}

func TestExecutionCostAndProviderFailuresFailBeforeRelease(t *testing.T) {
	var calls atomic.Int32
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		calls.Add(1)
		return ProviderExecutionResult{}, errors.New("provider offline")
	})
	runtime, _ := executionTestRuntime(t, PaginationOffset, PermissionPolicyPublic, provider, nil, nil)
	if _, err := runtime.Execute(t.Context(), PlanRequest{
		QueryID: "core.execute.items", MaxCost: 1,
	}); !errors.Is(err, ErrCostExceeded) || calls.Load() != 0 {
		t.Fatalf("cost gate: err=%v calls=%d", err, calls.Load())
	}
	if _, err := runtime.Execute(t.Context(), PlanRequest{
		QueryID: "core.execute.items",
	}); !errors.Is(err, ErrProviderFailed) || calls.Load() != 1 {
		t.Fatalf("provider failure: err=%v calls=%d", err, calls.Load())
	}

	blocking := ExecutableProviderFunc(func(ctx context.Context, _ ProviderExecutionRequest) (ProviderExecutionResult, error) {
		<-ctx.Done()
		return ProviderExecutionResult{}, ctx.Err()
	})
	runtime, _ = executionTestRuntime(t, PaginationOffset, PermissionPolicyPublic, blocking, nil, func(config *ExecutionConfig) {
		config.Timeout = 10 * time.Millisecond
	})
	if _, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("provider timeout=%v", err)
	}

	oversized := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": strings.Repeat("x", 2048)}}}, nil
	})
	runtime, _ = executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, oversized, nil, func(config *ExecutionConfig) {
		config.MaxResultBytes = 1024
	})
	if _, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"}); !errors.Is(err, ErrResultTooLarge) {
		t.Fatalf("oversized provider result=%v", err)
	}
	overRows := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "one"}, {"id": "2", "title": "two"}}}, nil
	})
	runtime, _ = executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, overRows, nil, nil)
	if _, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"}); !errors.Is(err, ErrResultTooLarge) {
		t.Fatalf("provider row overflow=%v", err)
	}
}

func TestExecutionOffsetAndAuthenticatedCursorPagination(t *testing.T) {
	var mu sync.Mutex
	var pages []PaginationPlan
	provider := ExecutableProviderFunc(func(_ context.Context, request ProviderExecutionRequest) (ProviderExecutionResult, error) {
		mu.Lock()
		pages = append(pages, request.Plan.Pagination)
		mu.Unlock()
		start := request.Plan.Pagination.Offset
		rows := make([]QueryRow, 0, request.FetchLimit)
		for index := 0; index < request.FetchLimit; index++ {
			rows = append(rows, QueryRow{"id": start + index + 1, "title": "row"})
		}
		return ProviderExecutionResult{Rows: rows}, nil
	})

	offsetRuntime, _ := executionTestRuntime(t, PaginationOffset, PermissionPolicyPublic, provider, nil, nil)
	offset, err := offsetRuntime.Execute(t.Context(), PlanRequest{
		QueryID: "core.execute.items", Pagination: PaginationRequest{Offset: 20, Limit: 2},
	})
	if err != nil || len(offset.Rows) != 2 || !offset.Page.HasMore || offset.Page.NextOffset != 22 {
		t.Fatalf("offset result=%#v err=%v", offset, err)
	}

	cursorRuntime, cursorRegistry := executionTestRuntime(t, PaginationCursor, PermissionPolicyPublic, provider, nil, func(config *ExecutionConfig) {
		codec, codecErr := NewHMACCursorCodec([]byte(strings.Repeat("cursor-secret-", 3)))
		if codecErr != nil {
			t.Fatal(codecErr)
		}
		config.Registry.cursorCodec = codec
	})
	first, err := cursorRuntime.Execute(t.Context(), PlanRequest{
		QueryID: "core.execute.items", Pagination: PaginationRequest{Limit: 2},
		Permission: PermissionInput{ActorFingerprint: "actor-a", PolicyFingerprint: "public-v1"},
		Locale:     "zh-CN", Scope: "forum.list",
	})
	if err != nil || first.Page.NextCursor == "" || len(first.Rows) != 2 {
		t.Fatalf("first cursor result=%#v err=%v", first, err)
	}
	secondRequest := PlanRequest{
		QueryID: "core.execute.items", Pagination: PaginationRequest{Cursor: first.Page.NextCursor},
		Permission: PermissionInput{ActorFingerprint: "actor-a", PolicyFingerprint: "public-v1"},
		Locale:     "zh-CN", Scope: "forum.list",
	}
	second, err := cursorRuntime.Execute(t.Context(), secondRequest)
	if err != nil || second.Page.Offset != 2 || len(second.Rows) != 2 {
		t.Fatalf("second cursor result=%#v err=%v", second, err)
	}
	queryContribution, err := cursorRegistry.Resolve("core.execute.items")
	if err != nil {
		t.Fatal(err)
	}
	changedProviders, err := NewStaticProviderResolver([]ExecutableProviderBinding{{
		QueryID: queryContribution.ID, ContractVersion: queryContribution.ContractVersion,
		PlanVersion: queryContribution.PlanVersion, ResultSchema: queryContribution.ResultSchema,
		Artifact: queryContribution.Artifact, ProviderDigest: strings.Repeat("f", 64), Provider: provider,
	}})
	if err != nil {
		t.Fatal(err)
	}
	changedRuntime, err := NewExecutionRuntime(ExecutionConfig{
		Registry: cursorRegistry, Providers: changedProviders, Schemas: allowExecutionSchema(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := changedRuntime.Execute(t.Context(), secondRequest); !errors.Is(err, ErrCursorInvalid) {
		t.Fatalf("cursor survived provider mapping change=%v", err)
	}
	tampered := secondRequest
	replacement := "A"
	if strings.HasSuffix(first.Page.NextCursor, replacement) {
		replacement = "B"
	}
	tampered.Pagination.Cursor = first.Page.NextCursor[:len(first.Page.NextCursor)-1] + replacement
	if _, err := cursorRuntime.Execute(t.Context(), tampered); !errors.Is(err, ErrCursorInvalid) {
		t.Fatalf("tampered cursor=%v", err)
	}
	wrongActor := secondRequest
	wrongActor.Permission.ActorFingerprint = "actor-b"
	if _, err := cursorRuntime.Execute(t.Context(), wrongActor); !errors.Is(err, ErrCursorInvalid) {
		t.Fatalf("cross-actor cursor=%v", err)
	}
	wrongLocale := secondRequest
	wrongLocale.Locale = "en-US"
	if _, err := cursorRuntime.Execute(t.Context(), wrongLocale); !errors.Is(err, ErrCursorInvalid) {
		t.Fatalf("cross-locale cursor=%v", err)
	}
	wrongShape := secondRequest
	wrongShape.Fields = []string{"id"}
	if _, err := cursorRuntime.Execute(t.Context(), wrongShape); !errors.Is(err, ErrCursorInvalid) {
		t.Fatalf("cross-shape cursor=%v", err)
	}
	extra := publication("core.cursor-extra", true, 'e')
	extra.Queries = []QueryDeclaration{query("core.cursor-extra.items", "core.cursor-extra.item", PaginationNone, PermissionPolicyPublic)}
	if _, err := cursorRegistry.Publish(extra); err != nil {
		t.Fatal(err)
	}
	if _, err := cursorRuntime.Execute(t.Context(), secondRequest); !errors.Is(err, ErrCursorInvalid) {
		t.Fatalf("cross-snapshot cursor=%v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(pages) != 3 || pages[len(pages)-1].Offset != 2 {
		t.Fatalf("provider pages=%#v", pages)
	}
}

func TestExecutionHoldsExactPluginAdmissionLeaseAcrossProviderCall(t *testing.T) {
	plugin := publication("plugin.execute", false, 'a')
	declaration := query("plugin.execute.items", "plugin.execute.item", PaginationNone, PermissionPolicyPublic)
	declaration.Relations = nil
	plugin.Queries = []QueryDeclaration{declaration}
	registry := newPlanningRegistry().WithPluginAdmission(func(artifact Artifact) bool {
		return artifact == plugin.Artifact
	})
	if _, err := registry.Publish(plugin); err != nil {
		t.Fatal(err)
	}
	var providerCalls atomic.Int32
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		providerCalls.Add(1)
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "leased"}}}, nil
	})
	providers, err := NewStaticProviderResolver([]ExecutableProviderBinding{{
		QueryID: declaration.ID, ContractVersion: declaration.ContractVersion,
		PlanVersion: declaration.PlanVersion, ResultSchema: declaration.ResultSchema,
		Artifact: plugin.Artifact, FailurePolicy: ProviderFailureFailClosed, Provider: provider,
	}})
	if err != nil {
		t.Fatal(err)
	}
	withoutLease, err := NewExecutionRuntime(ExecutionConfig{
		Registry: registry, Providers: providers, Schemas: allowExecutionSchema(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := withoutLease.Execute(t.Context(), PlanRequest{QueryID: declaration.ID}); !errors.Is(err, ErrArtifactUnavailable) || providerCalls.Load() != 0 {
		t.Fatalf("plugin executed without lease: err=%v calls=%d", err, providerCalls.Load())
	}
	var acquired, released atomic.Int32
	withLease, err := NewExecutionRuntime(ExecutionConfig{
		Registry: registry, Providers: providers, Schemas: allowExecutionSchema(),
		Admission: ExecutionAdmissionFunc(func(_ context.Context, artifact Artifact) (func(), error) {
			if artifact != plugin.Artifact {
				return nil, ErrArtifactUnavailable
			}
			acquired.Add(1)
			return func() { released.Add(1) }, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := withLease.Execute(t.Context(), PlanRequest{QueryID: declaration.ID})
	if err != nil || result.Rows[0]["title"] != "leased" || acquired.Load() != 1 ||
		released.Load() != 1 || providerCalls.Load() != 1 {
		t.Fatalf("leased plugin result=%#v err=%v acquired=%d released=%d calls=%d",
			result, err, acquired.Load(), released.Load(), providerCalls.Load())
	}
}

func TestExecutionValidatesProviderAndFilterResultsDeterministically(t *testing.T) {
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "base"}}}, nil
	})
	order := make([]string, 0, 2)
	var mu sync.Mutex
	filterArtifact := publication("plugin.filter", false, 'f').Artifact
	filters := []ResultFilterRegistration{
		executionTestFilter(filterArtifact, "plugin.filter.low", 10, ResultFilterFailClosed, func(rows []QueryRow) []QueryRow {
			mu.Lock()
			order = append(order, "low")
			mu.Unlock()
			rows[0]["title"] = rows[0]["title"].(string) + "-low"
			return rows
		}),
		executionTestFilter(filterArtifact, "plugin.filter.high", 20, ResultFilterFailClosed, func(rows []QueryRow) []QueryRow {
			mu.Lock()
			order = append(order, "high")
			mu.Unlock()
			rows[0]["title"] = rows[0]["title"].(string) + "-high"
			return rows
		}),
	}
	runtime, registry := executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, provider, filters, func(config *ExecutionConfig) {
		config.Registry.WithPluginAdmission(func(artifact Artifact) bool { return artifact == filterArtifact })
	})
	result, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"})
	if err != nil || result.Rows[0]["title"] != "base-high-low" || !slices.Equal(order, []string{"high", "low"}) {
		t.Fatalf("filter composition: result=%#v order=%#v err=%v", result, order, err)
	}

	invalidProvider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "base", "secret": "leak"}}}, nil
	})
	invalidRuntime, _ := executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, invalidProvider, nil, nil)
	if _, err := invalidRuntime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"}); !errors.Is(err, ErrResultInvalid) {
		t.Fatalf("undeclared provider field=%v", err)
	}

	cardinalityFilter := executionTestFilter(filterArtifact, "plugin.filter.drop", 30, ResultFilterFailClosed, func([]QueryRow) []QueryRow {
		return []QueryRow{}
	})
	invalidRuntime, _ = executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, provider, []ResultFilterRegistration{cardinalityFilter}, func(config *ExecutionConfig) {
		config.Registry.WithPluginAdmission(func(artifact Artifact) bool { return artifact == filterArtifact })
	})
	if _, err := invalidRuntime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"}); !errors.Is(err, ErrResultInvalid) {
		t.Fatalf("cardinality-changing filter=%v", err)
	}

	// Snapshot swaps inside a provider are caught before any result release.
	swapProvider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		concurrent := publication("core.concurrent", true, 'c')
		concurrent.Queries = []QueryDeclaration{query("core.concurrent.items", "core.concurrent.item", PaginationNone, PermissionPolicyPublic)}
		_, publishErr := registry.Publish(concurrent)
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "stale"}}}, publishErr
	})
	binding, err := NewStaticProviderResolver([]ExecutableProviderBinding{executionTestBinding(registry, swapProvider)})
	if err != nil {
		t.Fatal(err)
	}
	swapRuntime, err := NewExecutionRuntime(ExecutionConfig{Registry: registry, Providers: binding, Schemas: allowExecutionSchema()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := swapRuntime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"}); !errors.Is(err, ErrArtifactConflict) {
		t.Fatalf("snapshot swap released=%v", err)
	}
}

func TestExecutionFilterFailureDependencyAndAdmissionPolicies(t *testing.T) {
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "base"}}}, nil
	})
	artifact := publication("plugin.filter", false, 'd').Artifact
	failing := executionTestFilter(artifact, "plugin.filter.failure", 10, ResultFilterFailOpen, func([]QueryRow) []QueryRow {
		return nil
	})
	failing.Filter = ResultFilterFunc(func(context.Context, ResultFilterRequest) (ResultFilterResult, error) {
		return ResultFilterResult{}, errors.New("filter failed")
	})
	runtime, _ := executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, provider, []ResultFilterRegistration{failing}, func(config *ExecutionConfig) {
		config.Registry.WithPluginAdmission(func(candidate Artifact) bool { return candidate == artifact })
	})
	result, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"})
	if err != nil || result.Rows[0]["title"] != "base" {
		t.Fatalf("fail-open filter: result=%#v err=%v", result, err)
	}
	failing.FailurePolicy = ResultFilterFailClosed
	runtime, _ = executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, provider, []ResultFilterRegistration{failing}, func(config *ExecutionConfig) {
		config.Registry.WithPluginAdmission(func(candidate Artifact) bool { return candidate == artifact })
	})
	if _, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"}); !errors.Is(err, ErrProviderFailed) {
		t.Fatalf("fail-closed filter=%v", err)
	}

	missingDependency := executionTestFilter(artifact, "plugin.filter.dependency", 10, ResultFilterFailClosed, func(rows []QueryRow) []QueryRow { return rows })
	missingDependency.Dependency = ResultFilterDependency{}
	if _, _, err := executionTestRuntimeError(t, PaginationNone, PermissionPolicyPublic, provider, []ResultFilterRegistration{missingDependency}, func(config *ExecutionConfig) {
		config.Registry.WithPluginAdmission(func(candidate Artifact) bool { return candidate == artifact })
	}); !errors.Is(err, ErrDependencyDenied) {
		t.Fatalf("undeclared cross-plugin dependency=%v", err)
	}
	wrongVersion := executionTestFilter(artifact, "plugin.filter.version", 10, ResultFilterFailClosed, func(rows []QueryRow) []QueryRow { return rows })
	wrongVersion.Dependency.VersionConstraint = "^2.0.0"
	if _, _, err := executionTestRuntimeError(t, PaginationNone, PermissionPolicyPublic, provider, []ResultFilterRegistration{wrongVersion}, func(config *ExecutionConfig) {
		config.Registry.WithPluginAdmission(func(candidate Artifact) bool { return candidate == artifact })
	}); !errors.Is(err, ErrDependencyDenied) {
		t.Fatalf("incompatible dependency=%v", err)
	}

	missingIdentity := executionTestFilter(artifact, "plugin.filter.identity", 10, ResultFilterFailClosed, func(rows []QueryRow) []QueryRow { return rows })
	missingIdentity.IdentityFields = nil
	if _, _, err := executionTestRuntimeError(t, PaginationOffset, PermissionPolicyPublic, provider, []ResultFilterRegistration{missingIdentity}, func(config *ExecutionConfig) {
		config.Registry.WithPluginAdmission(func(candidate Artifact) bool { return candidate == artifact })
	}); !errors.Is(err, ErrContractInsufficient) {
		t.Fatalf("paginated filter without identity=%v", err)
	}

	timed := executionTestFilter(artifact, "plugin.filter.timeout", 10, ResultFilterFailOpen, func(rows []QueryRow) []QueryRow { return rows })
	timed.Timeout = 10 * time.Millisecond
	timed.Filter = ResultFilterFunc(func(ctx context.Context, _ ResultFilterRequest) (ResultFilterResult, error) {
		<-ctx.Done()
		return ResultFilterResult{}, ctx.Err()
	})
	runtime, _ = executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, provider, []ResultFilterRegistration{timed}, func(config *ExecutionConfig) {
		config.Registry.WithPluginAdmission(func(candidate Artifact) bool { return candidate == artifact })
	})
	if result, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"}); err != nil || result.Rows[0]["title"] != "base" {
		t.Fatalf("fail-open filter timeout: result=%#v err=%v", result, err)
	}
	timed.FailurePolicy = ResultFilterFailClosed
	runtime, _ = executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, provider, []ResultFilterRegistration{timed}, func(config *ExecutionConfig) {
		config.Registry.WithPluginAdmission(func(candidate Artifact) bool { return candidate == artifact })
	})
	if _, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("fail-closed filter timeout=%v", err)
	}

	runtime, _ = executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, provider, []ResultFilterRegistration{
		executionTestFilter(artifact, "plugin.filter.disabled", 10, ResultFilterFailClosed, func(rows []QueryRow) []QueryRow { return rows }),
	}, func(config *ExecutionConfig) {
		config.Registry.WithPluginAdmission(func(Artifact) bool { return false })
	})
	if _, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"}); !errors.Is(err, ErrDependencyDenied) {
		t.Fatalf("disabled filter dependency=%v", err)
	}
}

func executionTestRuntime(
	t *testing.T,
	pagination, permission string,
	provider ExecutableProvider,
	filters []ResultFilterRegistration,
	configure func(*ExecutionConfig),
) (*ExecutionRuntime, *Registry) {
	t.Helper()
	runtime, registry, err := executionTestRuntimeError(t, pagination, permission, provider, filters, configure)
	if err != nil {
		t.Fatal(err)
	}
	return runtime, registry
}

func executionTestRuntimeError(
	t *testing.T,
	pagination, permission string,
	provider ExecutableProvider,
	filters []ResultFilterRegistration,
	configure func(*ExecutionConfig),
) (*ExecutionRuntime, *Registry, error) {
	t.Helper()
	registry := newPlanningRegistry()
	core := publication("core.execute", true, 'a')
	declaration := query("core.execute.items", "core.execute.item", pagination, permission)
	declaration.Relations = nil
	core.Queries = []QueryDeclaration{declaration}
	if _, err := registry.Publish(core); err != nil {
		t.Fatal(err)
	}
	binding := ExecutableProviderBinding{
		QueryID: declaration.ID, ContractVersion: declaration.ContractVersion,
		PlanVersion: declaration.PlanVersion, ResultSchema: declaration.ResultSchema,
		Artifact: core.Artifact, FailurePolicy: ProviderFailureFailClosed, Provider: provider,
	}
	providers, err := NewStaticProviderResolver([]ExecutableProviderBinding{binding})
	if err != nil {
		return nil, registry, err
	}
	config := ExecutionConfig{
		Registry: registry, Providers: providers, Schemas: allowExecutionSchema(), ResultFilters: filters,
		Admission: ExecutionAdmissionFunc(func(context.Context, Artifact) (func(), error) {
			return func() {}, nil
		}),
	}
	if configure != nil {
		configure(&config)
	}
	runtime, err := NewExecutionRuntime(config)
	if err != nil {
		return nil, registry, err
	}
	// Dependency errors are plan-specific and surface before provider execution.
	if _, filterErr := runtime.matchingFilters(coreQueryContribution(core)); filterErr != nil {
		return runtime, registry, filterErr
	}
	return runtime, registry, nil
}

func coreQueryContribution(publication Publication) QueryContribution {
	return QueryContribution{QueryDeclaration: publication.Queries[0], Artifact: publication.Artifact}
}

func executionTestBinding(registry *Registry, provider ExecutableProvider) ExecutableProviderBinding {
	query, _ := registry.Resolve("core.execute.items")
	return ExecutableProviderBinding{
		QueryID: query.ID, ContractVersion: query.ContractVersion, PlanVersion: query.PlanVersion,
		ResultSchema: query.ResultSchema, Artifact: query.Artifact,
		FailurePolicy: ProviderFailureFailClosed, Provider: provider,
	}
}

func allowExecutionSchema() ResultSchemaValidator {
	return ResultSchemaValidatorFunc(func(context.Context, ResultSchemaClaim, QueryRow) error { return nil })
}

func executionTestFilter(
	artifact Artifact,
	id string,
	priority int,
	failurePolicy string,
	filter func([]QueryRow) []QueryRow,
) ResultFilterRegistration {
	return ResultFilterRegistration{
		ID: id, ContractVersion: id + "@1", QueryID: "core.execute.items",
		QueryContractVersion: "core.execute.items@1", QueryPlanVersion: "core.execute.items.plan@1",
		Priority: priority, Artifact: artifact,
		Dependency:     ResultFilterDependency{ExtensionID: "core.execute", VersionConstraint: "^1.0.0"},
		IdentityFields: []string{"id"},
		FailurePolicy:  failurePolicy, Timeout: time.Second,
		Filter: ResultFilterFunc(func(_ context.Context, request ResultFilterRequest) (ResultFilterResult, error) {
			return ResultFilterResult{Rows: filter(request.Rows)}, nil
		}),
	}
}
