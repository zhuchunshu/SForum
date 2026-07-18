package queryregistry

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
)

func TestExecutionCacheIsolationHitAndPoisonFence(t *testing.T) {
	cache := newMemoryQueryResultCache()
	var providerCalls atomic.Int32
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		providerCalls.Add(1)
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "cached"}}}, nil
	})
	runtime, registry := executionTestRuntime(t, PaginationOffset, "core.execute.read", provider, nil, func(config *ExecutionConfig) {
		config.Cache = cache
	})
	permission := PermissionInput{
		Authenticated: true, ActorFingerprint: "user-a", PolicyFingerprint: "role:reader", Recheck: allowAll(),
	}
	request := PlanRequest{
		QueryID: "core.execute.items", Fields: []string{"id", "title"},
		Pagination: PaginationRequest{Limit: 10}, Permission: permission, Locale: "en-US", Scope: "forum.main",
	}
	first, err := runtime.Execute(t.Context(), request)
	if err != nil || first.CacheHit || providerCalls.Load() != 1 || len(first.CacheTags) != 2 {
		t.Fatalf("first cache result=%#v err=%v calls=%d", first, err, providerCalls.Load())
	}
	for _, tag := range first.CacheTags {
		if !strings.HasPrefix(tag, "query:") || strings.Contains(tag, "core.execute.items") || strings.Contains(tag, "user-a") {
			t.Fatalf("cache tag is not isolated/opaque: %q", tag)
		}
	}
	firstSharedTag, firstIsolatedTag := splitExecutionCacheTags(t, first.CacheTags)
	second, err := runtime.Execute(t.Context(), request)
	if err != nil || !second.CacheHit || providerCalls.Load() != 1 || !slices.Equal(first.CacheTags, second.CacheTags) {
		t.Fatalf("cache hit=%#v err=%v calls=%d", second, err, providerCalls.Load())
	}
	second.Rows[0]["title"] = "caller mutation"
	third, err := runtime.Execute(t.Context(), request)
	if err != nil || third.Rows[0]["title"] != "cached" {
		t.Fatalf("caller mutation poisoned cache: result=%#v err=%v", third, err)
	}

	otherActor := request
	otherActor.Permission.ActorFingerprint = "user-b"
	actorResult, err := runtime.Execute(t.Context(), otherActor)
	if err != nil {
		t.Fatalf("actor isolation execution: %v", err)
	}
	actorSharedTag, actorIsolatedTag := splitExecutionCacheTags(t, actorResult.CacheTags)
	if actorResult.CacheKey == first.CacheKey || actorSharedTag != firstSharedTag ||
		actorIsolatedTag == firstIsolatedTag || providerCalls.Load() != 2 {
		t.Fatalf("actor isolation: result=%#v err=%v calls=%d", actorResult, err, providerCalls.Load())
	}
	otherLocale := request
	otherLocale.Locale = "zh-CN"
	localeResult, err := runtime.Execute(t.Context(), otherLocale)
	if err != nil {
		t.Fatalf("locale isolation execution: %v", err)
	}
	localeSharedTag, localeIsolatedTag := splitExecutionCacheTags(t, localeResult.CacheTags)
	if localeResult.CacheKey == first.CacheKey || localeSharedTag != firstSharedTag ||
		localeIsolatedTag == firstIsolatedTag || providerCalls.Load() != 3 {
		t.Fatalf("locale isolation: result=%#v err=%v calls=%d", localeResult, err, providerCalls.Load())
	}
	otherPage := request
	otherPage.Pagination.Offset = 10
	pageResult, err := runtime.Execute(t.Context(), otherPage)
	if err != nil {
		t.Fatalf("page isolation execution: %v", err)
	}
	pageSharedTag, pageIsolatedTag := splitExecutionCacheTags(t, pageResult.CacheTags)
	if pageResult.CacheKey == first.CacheKey || pageSharedTag != firstSharedTag ||
		pageIsolatedTag == firstIsolatedTag || providerCalls.Load() != 4 {
		t.Fatalf("page isolation: result=%#v err=%v calls=%d", pageResult, err, providerCalls.Load())
	}
	storedTagSets := make(map[string][]string, 4)
	cache.mu.Lock()
	for _, key := range []string{first.CacheKey, actorResult.CacheKey, localeResult.CacheKey, pageResult.CacheKey} {
		storedTagSets[key] = slices.Clone(cache.tags[key])
	}
	cache.mu.Unlock()
	for key, tags := range storedTagSets {
		if !slices.Contains(tags, firstSharedTag) {
			t.Fatalf("shared invalidation tag %q missing from cache key %q", firstSharedTag, key)
		}
	}

	cache.mu.Lock()
	poisoned := cache.entries[first.CacheKey]
	poisoned.ContractVersion = "forged.contract@1"
	cache.entries[first.CacheKey] = poisoned
	cache.mu.Unlock()
	if _, err := runtime.Execute(t.Context(), request); !errors.Is(err, ErrCachePoisoned) || providerCalls.Load() != 4 {
		t.Fatalf("poisoned metadata fell through: err=%v calls=%d", err, providerCalls.Load())
	}

	cache.mu.Lock()
	poisoned = cache.entries[first.CacheKey]
	poisoned.ContractVersion = "core.execute.items@1"
	poisoned.Rows[0]["secret"] = "cache leak"
	cache.entries[first.CacheKey] = poisoned
	cache.mu.Unlock()
	if _, err := runtime.Execute(t.Context(), request); !errors.Is(err, ErrCachePoisoned) || providerCalls.Load() != 4 {
		t.Fatalf("poisoned row fell through: err=%v calls=%d", err, providerCalls.Load())
	}

	// A registry revision change creates a disjoint key/tag namespace even when
	// the selected query declaration itself is unchanged.
	cache.mu.Lock()
	delete(cache.entries, first.CacheKey)
	cache.mu.Unlock()
	concurrent := publication("core.cache-extra", true, 'e')
	concurrent.Queries = []QueryDeclaration{query("core.cache-extra.items", "core.cache-extra.item", PaginationNone, PermissionPolicyPublic)}
	if _, err := registry.Publish(concurrent); err != nil {
		t.Fatal(err)
	}
	afterSnapshot, err := runtime.Execute(t.Context(), request)
	if err != nil || afterSnapshot.CacheKey == first.CacheKey || slices.Equal(afterSnapshot.CacheTags, first.CacheTags) {
		t.Fatalf("snapshot isolation: result=%#v err=%v", afterSnapshot, err)
	}

	active, ok := registry.SnapshotPublication("core.execute")
	if !ok {
		t.Fatal("core query publication disappeared")
	}
	replacementArtifact, err := NewCoreArtifact("core.execute", "1.0.1", strings.Repeat("b", 64))
	if err != nil {
		t.Fatal(err)
	}
	replacement := active
	replacement.Artifact = replacementArtifact
	if _, err := registry.PublishIfArtifact(active.Artifact, replacement); err != nil {
		t.Fatal(err)
	}
	replacementProvider, err := NewStaticProviderResolver([]ExecutableProviderBinding{{
		QueryID: replacement.Queries[0].ID, ContractVersion: replacement.Queries[0].ContractVersion,
		PlanVersion: replacement.Queries[0].PlanVersion, ResultSchema: replacement.Queries[0].ResultSchema,
		Artifact: replacementArtifact, FailurePolicy: ProviderFailureFailClosed, Provider: provider,
	}})
	if err != nil {
		t.Fatal(err)
	}
	replacementRuntime, err := NewExecutionRuntime(ExecutionConfig{
		Registry: registry, Providers: replacementProvider, Schemas: allowExecutionSchema(), Cache: cache,
	})
	if err != nil {
		t.Fatal(err)
	}
	afterArtifact, err := replacementRuntime.Execute(t.Context(), request)
	if err != nil || afterArtifact.CacheKey == afterSnapshot.CacheKey || slices.Equal(afterArtifact.CacheTags, afterSnapshot.CacheTags) {
		t.Fatalf("artifact isolation: result=%#v err=%v", afterArtifact, err)
	}
}

func TestExecutionCacheFencePreventsStaleProviderRevival(t *testing.T) {
	cache := newMemoryQueryResultCache()
	providerStarted := make(chan struct{})
	providerRelease := make(chan struct{})
	var providerCalls atomic.Int32
	provider := ExecutableProviderFunc(func(ctx context.Context, _ ProviderExecutionRequest) (ProviderExecutionResult, error) {
		call := providerCalls.Add(1)
		if call == 1 {
			close(providerStarted)
			select {
			case <-providerRelease:
			case <-ctx.Done():
				return ProviderExecutionResult{}, context.Cause(ctx)
			}
		}
		title := "fresh"
		if call == 1 {
			title = "pre-invalidation"
		}
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": title}}}, nil
	})
	runtime, _ := executionTestRuntime(t, PaginationOffset, PermissionPolicyPublic, provider, nil, func(config *ExecutionConfig) {
		config.Cache = cache
	})
	request := PlanRequest{QueryID: "core.execute.items", Pagination: PaginationRequest{Limit: 10}}
	type executionOutcome struct {
		result QueryResult
		err    error
	}
	firstDone := make(chan executionOutcome, 1)
	go func() {
		result, err := runtime.Execute(t.Context(), request)
		firstDone <- executionOutcome{result: result, err: err}
	}()
	select {
	case <-providerStarted:
	case <-t.Context().Done():
		t.Fatal("provider did not start before test context ended")
	}
	cache.invalidate()
	close(providerRelease)
	first := <-firstDone
	if first.err != nil || first.result.CacheHit || first.result.Rows[0]["title"] != "pre-invalidation" {
		t.Fatalf("in-flight result=%#v err=%v", first.result, first.err)
	}
	second, err := runtime.Execute(t.Context(), request)
	if err != nil || second.CacheHit || second.Rows[0]["title"] != "fresh" || providerCalls.Load() != 2 {
		t.Fatalf("stale provider result revived: result=%#v err=%v calls=%d", second, err, providerCalls.Load())
	}
}

func TestExecutionCacheFailurePolicyAndTrace(t *testing.T) {
	backendErr := errors.New("cache backend unavailable")
	tests := []struct {
		name          string
		loadErr       error
		storeErr      error
		missingFence  bool
		wantErr       error
		wantStatus    string
		wantProviders int32
		wantStores    int32
	}{
		{name: "load poison", loadErr: ErrCachePoisoned, wantErr: ErrCachePoisoned, wantStatus: "poisoned"},
		{name: "load io error", loadErr: backendErr, wantStatus: "load_error", wantProviders: 1},
		{name: "missing miss fence", missingFence: true, wantStatus: "load_error", wantProviders: 1},
		{name: "store conflict", storeErr: ErrCacheFenceConflict, wantStatus: "store_conflict", wantProviders: 1, wantStores: 1},
		{name: "store poison", storeErr: ErrCachePoisoned, wantStatus: "store_poisoned", wantProviders: 1, wantStores: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var providerCalls, storeCalls atomic.Int32
			cache := executionCallbackCache{
				load: func(context.Context, string, []string) (CachedQueryResult, QueryResultCacheFence, bool, error) {
					if test.loadErr != nil {
						return CachedQueryResult{}, nil, false, test.loadErr
					}
					if test.missingFence {
						return CachedQueryResult{}, nil, false, nil
					}
					return CachedQueryResult{}, executionCallbackCacheFence(test.name), false, nil
				},
				store: func(context.Context, string, CachedQueryResult, []string, QueryResultCacheFence) error {
					storeCalls.Add(1)
					return test.storeErr
				},
			}
			ring := NewExecutionTraceRing(1)
			provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
				providerCalls.Add(1)
				return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "visible"}}}, nil
			})
			runtime, _ := executionTestRuntime(t, PaginationOffset, PermissionPolicyPublic, provider, nil, func(config *ExecutionConfig) {
				config.Cache = cache
				config.Trace = ring
			})
			result, err := runtime.Execute(t.Context(), PlanRequest{
				QueryID: "core.execute.items", Pagination: PaginationRequest{Limit: 10},
			})
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) || len(result.Rows) != 0 {
					t.Fatalf("result=%#v err=%v", result, err)
				}
			} else if err != nil || len(result.Rows) != 1 {
				t.Fatalf("result=%#v err=%v", result, err)
			}
			traces := ring.ExecutionTraces(1)
			if providerCalls.Load() != test.wantProviders || storeCalls.Load() != test.wantStores ||
				len(traces) != 1 || traces[0].CacheStatus != test.wantStatus {
				t.Fatalf("provider=%d store=%d traces=%#v", providerCalls.Load(), storeCalls.Load(), traces)
			}
		})
	}
}

func TestExecutionCachePassesExactFenceAndClonesTags(t *testing.T) {
	token := executionCallbackCacheFence("exact-token")
	var loadedTags, storedTags []string
	cache := executionCallbackCache{
		load: func(_ context.Context, _ string, tags []string) (CachedQueryResult, QueryResultCacheFence, bool, error) {
			loadedTags = slices.Clone(tags)
			tags[0] = "load-caller-mutation"
			return CachedQueryResult{}, &token, false, nil
		},
		store: func(_ context.Context, _ string, value CachedQueryResult, tags []string, fence QueryResultCacheFence) error {
			if fence != &token {
				t.Fatalf("store fence=%#v, want exact load token=%p", fence, &token)
			}
			if !slices.Equal(tags, value.CacheTags) || slices.Contains(tags, "load-caller-mutation") {
				t.Fatalf("store tags=%#v cached tags=%#v", tags, value.CacheTags)
			}
			storedTags = slices.Clone(tags)
			tags[0] = "store-caller-mutation"
			return nil
		},
	}
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "visible"}}}, nil
	})
	runtime, _ := executionTestRuntime(t, PaginationOffset, PermissionPolicyPublic, provider, nil, func(config *ExecutionConfig) {
		config.Cache = cache
	})
	result, err := runtime.Execute(t.Context(), PlanRequest{
		QueryID: "core.execute.items", Pagination: PaginationRequest{Limit: 10},
	})
	if err != nil || !slices.Equal(loadedTags, storedTags) || !slices.Equal(result.CacheTags, storedTags) ||
		slices.Contains(result.CacheTags, "store-caller-mutation") {
		t.Fatalf("result=%#v err=%v loaded=%#v stored=%#v", result, err, loadedTags, storedTags)
	}
}

func splitExecutionCacheTags(t *testing.T, tags []string) (shared, isolated string) {
	t.Helper()
	if len(tags) != 2 {
		t.Fatalf("cache tag pair=%#v", tags)
	}
	for _, tag := range tags {
		switch {
		case strings.HasPrefix(tag, "query:shared:"):
			shared = tag
		case strings.HasPrefix(tag, "query:"):
			isolated = tag
		}
	}
	if shared == "" || isolated == "" || shared == isolated {
		t.Fatalf("cache tag pair is incomplete: %#v", tags)
	}
	return shared, isolated
}

func TestExecutionSchemaValidatorRunsAtProviderFilterCacheAndReleaseFences(t *testing.T) {
	cache := newMemoryQueryResultCache()
	var validations atomic.Int32
	validator := ResultSchemaValidatorFunc(func(_ context.Context, claim ResultSchemaClaim, row QueryRow) error {
		validations.Add(1)
		if claim.QueryID != "core.execute.items" || claim.ResultSchema != "core.execute.items.result@1" ||
			claim.Artifact.ExtensionID != "core.execute" || claim.ShapeDigest == "" || claim.RowIndex < 0 {
			return errors.New("invalid schema claim")
		}
		if row["title"] == "invalid" {
			return errors.New("invalid title")
		}
		return nil
	})
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "valid"}}}, nil
	})
	runtime, _ := executionTestRuntime(t, PaginationOffset, PermissionPolicyPublic, provider, nil, func(config *ExecutionConfig) {
		config.Cache = cache
		config.Schemas = validator
	})
	request := PlanRequest{QueryID: "core.execute.items", Pagination: PaginationRequest{Limit: 10}}
	if _, err := runtime.Execute(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	firstCount := validations.Load()
	if firstCount < 2 {
		t.Fatalf("fresh result validation count=%d", firstCount)
	}
	if _, err := runtime.Execute(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	if validations.Load() <= firstCount {
		t.Fatal("cache hit bypassed result schema validation")
	}

	invalidProvider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "invalid"}}}, nil
	})
	invalidRuntime, _ := executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, invalidProvider, nil, func(config *ExecutionConfig) {
		config.Schemas = validator
	})
	if _, err := invalidRuntime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"}); !errors.Is(err, ErrResultInvalid) {
		t.Fatalf("schema-invalid provider result=%v", err)
	}
}

func TestExecutionTraceAndInspectorExposeBoundedExactEvidence(t *testing.T) {
	ring := NewExecutionTraceRing(8)
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "visible"}}}, nil
	})
	filterArtifact := publication("plugin.inspect-filter", false, 'f').Artifact
	filter := ResultFilterRegistration{
		ID: "plugin.inspect-filter.redact", ContractVersion: "plugin.inspect-filter.redact@1",
		QueryID: "core.execute.items", QueryContractVersion: "core.execute.items@1",
		QueryPlanVersion: "core.execute.items.plan@1", Priority: 20, Artifact: filterArtifact,
		Dependency:     ResultFilterDependency{ExtensionID: "core.execute", VersionConstraint: "^1.0.0"},
		IdentityFields: []string{"id"},
		FailurePolicy:  ResultFilterFailClosed,
		Filter: ResultFilterFunc(func(_ context.Context, request ResultFilterRequest) (ResultFilterResult, error) {
			return ResultFilterResult{Rows: request.Rows}, nil
		}),
	}
	runtime, _ := executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, provider, []ResultFilterRegistration{filter}, func(config *ExecutionConfig) {
		config.Trace = ring
		config.Registry.WithPluginAdmission(func(artifact Artifact) bool { return artifact == filterArtifact })
	})
	if _, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"}); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items", MaxCost: 1}); !errors.Is(err, ErrCostExceeded) {
		t.Fatal(err)
	}
	traces := ring.ExecutionTraces(10)
	if len(traces) != 2 || traces[0].Outcome != TraceOutcomeCostExceeded || traces[1].Outcome != TraceOutcomeAllowed ||
		traces[1].ShapeDigest == "" || len(traces[1].ProviderDigest) != 64 ||
		traces[1].FilterPlan == "" || traces[1].FilterCount != 1 ||
		len(traces[1].ResultFilters) != 1 || traces[1].ResultFilters[0].ID != filter.ID ||
		traces[1].ResultFilters[0].Outcome != ResultFilterTraceApplied {
		t.Fatalf("traces=%#v", traces)
	}
	inspection, err := runtime.Inspect("core.execute.items", ring)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Query.Artifact.ExtensionID != "core.execute" || !inspection.ProviderBound ||
		!inspection.ProviderResolved || len(inspection.ProviderDigest) != 64 || !inspection.SchemaBound ||
		inspection.FilterPlan == "" || len(inspection.ResultFilters) != 1 ||
		inspection.ResultFilters[0].ID != filter.ID || !inspection.ResultFilters[0].Admitted || len(inspection.Traces) != 2 {
		t.Fatalf("inspection=%#v", inspection)
	}
	traces[1].ResultFilters[0].Outcome = "caller-mutation"
	if retained := ring.ExecutionTraces(10); retained[1].ResultFilters[0].Outcome != ResultFilterTraceApplied {
		t.Fatalf("trace ring leaked nested slice ownership=%#v", retained)
	}
	for _, trace := range traces {
		if strings.Contains(trace.FilterPlan, "visible") || strings.Contains(trace.ShapeDigest, "user") {
			t.Fatalf("trace leaked payload material=%#v", trace)
		}
	}
}

func TestExecutionTraceHashesUnknownQueryIDsAndStripsControls(t *testing.T) {
	ring := NewExecutionTraceRing(4)
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		t.Fatal("invalid query reached provider")
		return ProviderExecutionResult{}, nil
	})
	runtime, _ := executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, provider, nil, func(config *ExecutionConfig) {
		config.Trace = ring
	})
	rawQueryID := "Bearer secret\r\nforged-log-line"
	if _, err := runtime.Execute(t.Context(), PlanRequest{QueryID: rawQueryID}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid query result=%v", err)
	}
	traces := ring.ExecutionTraces(1)
	if len(traces) != 1 || traces[0].QueryID != unplannedExecutionTraceQueryID(rawQueryID) ||
		!strings.HasPrefix(traces[0].QueryID, unplannedExecutionTracePrefix) ||
		strings.Contains(traces[0].QueryID, "secret") || strings.ContainsAny(traces[0].QueryID, "\r\n") {
		t.Fatalf("unplanned query trace=%#v", traces)
	}

	ring.AppendExecutionTrace(ExecutionTrace{QueryID: "known\r\nforged", Stage: "plan", Outcome: TraceOutcomeInvalid})
	bounded := ring.ExecutionTraces(1)
	if len(bounded) != 1 || strings.ContainsAny(bounded[0].QueryID, "\r\n") ||
		bounded[0].QueryID != unplannedExecutionTraceQueryID("known\r\nforged") {
		t.Fatalf("control characters survived trace bound=%#v", bounded)
	}
}

func TestExecutionRejectsMissingAndVersionDriftedProviderBindings(t *testing.T) {
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		t.Fatal("invalid binding executed")
		return ProviderExecutionResult{}, nil
	})
	runtime, registry := executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, provider, nil, nil)
	query, err := registry.Resolve("core.execute.items")
	if err != nil {
		t.Fatal(err)
	}
	missing, err := NewStaticProviderResolver(nil)
	if err != nil {
		t.Fatal(err)
	}
	missingRuntime, err := NewExecutionRuntime(ExecutionConfig{Registry: registry, Providers: missing, Schemas: allowExecutionSchema()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := missingRuntime.Execute(t.Context(), PlanRequest{QueryID: query.ID}); !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("missing provider=%v", err)
	}
	_ = runtime

	drifted := executionTestBinding(registry, provider)
	drifted.PlanVersion = "core.execute.items.plan@2"
	resolver := executableProviderResolverFunc(func(context.Context, QueryPlan) (ExecutableProviderBinding, error) {
		return drifted, nil
	})
	driftRuntime, err := NewExecutionRuntime(ExecutionConfig{Registry: registry, Providers: resolver, Schemas: allowExecutionSchema()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := driftRuntime.Execute(t.Context(), PlanRequest{QueryID: query.ID}); !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("version-drifted provider=%v", err)
	}
	invalidFailure := executionTestBinding(registry, provider)
	invalidFailure.FailurePolicy = ResultFilterFailOpen
	if _, err := NewStaticProviderResolver([]ExecutableProviderBinding{invalidFailure}); !errors.Is(err, ErrExecutionInvalid) {
		t.Fatalf("provider fail-open accepted=%v", err)
	}
}

func TestExecutionCacheIsFencedByResolvedProviderMapping(t *testing.T) {
	cache := newMemoryQueryResultCache()
	var calls atomic.Int32
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		calls.Add(1)
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "mapped"}}}, nil
	})
	_, registry := executionTestRuntime(t, PaginationOffset, PermissionPolicyPublic, provider, nil, nil)
	binding := executionTestBinding(registry, provider)
	var digest atomic.Value
	digest.Store(strings.Repeat("a", 64))
	resolver := executableProviderResolverFunc(func(context.Context, QueryPlan) (ExecutableProviderBinding, error) {
		candidate := binding
		candidate.ProviderDigest = digest.Load().(string)
		return candidate, nil
	})
	runtime, err := NewExecutionRuntime(ExecutionConfig{
		Registry: registry, Providers: resolver, Schemas: allowExecutionSchema(), Cache: cache,
	})
	if err != nil {
		t.Fatal(err)
	}
	request := PlanRequest{QueryID: "core.execute.items", Pagination: PaginationRequest{Limit: 10}}
	first, err := runtime.Execute(t.Context(), request)
	if err != nil || first.CacheHit || first.ProviderDigest != strings.Repeat("a", 64) {
		t.Fatalf("first provider mapping result=%#v err=%v", first, err)
	}
	digest.Store(strings.Repeat("b", 64))
	second, err := runtime.Execute(t.Context(), request)
	if err != nil || second.CacheHit || second.ProviderDigest != strings.Repeat("b", 64) ||
		second.CacheKey == first.CacheKey || calls.Load() != 2 {
		t.Fatalf("changed provider mapping result=%#v err=%v calls=%d", second, err, calls.Load())
	}
}

func TestExecutionFailOpenContractMismatchIsInspectable(t *testing.T) {
	ring := NewExecutionTraceRing(4)
	artifact := publication("plugin.stale-filter", false, 'd').Artifact
	filter := executionTestFilter(artifact, "plugin.stale-filter.decorate", 10, ResultFilterFailOpen, func(rows []QueryRow) []QueryRow {
		return rows
	})
	filter.QueryPlanVersion = "core.execute.items.plan@2"
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "base"}}}, nil
	})
	runtime, _ := executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, provider, []ResultFilterRegistration{filter}, func(config *ExecutionConfig) {
		config.Trace = ring
		config.Registry.WithPluginAdmission(func(candidate Artifact) bool { return candidate == artifact })
	})
	if _, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"}); err != nil {
		t.Fatal(err)
	}
	traces := ring.ExecutionTraces(1)
	if len(traces) != 1 || traces[0].FilterCount != 0 || len(traces[0].ResultFilters) != 1 ||
		traces[0].ResultFilters[0].Outcome != ResultFilterTraceContractMismatch {
		t.Fatalf("contract mismatch trace=%#v", traces)
	}
	inspection, err := runtime.Inspect("core.execute.items", ring)
	if err != nil {
		t.Fatal(err)
	}
	if len(inspection.ResultFilters) != 1 ||
		inspection.ResultFilters[0].Status != ResultFilterTraceContractMismatch {
		t.Fatalf("contract mismatch inspection=%#v", inspection)
	}
}
