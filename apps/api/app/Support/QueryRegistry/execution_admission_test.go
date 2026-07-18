package queryregistry

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestExecutionAcquiresPluginQueryLeaseForFreshAndCachedRelease(t *testing.T) {
	plugin := publication("plugin.cached-query", false, 'a')
	declaration := query("plugin.cached-query.items", "plugin.cached-query.item", PaginationOffset, PermissionPolicyPublic)
	declaration.Relations = nil
	plugin.Queries = []QueryDeclaration{declaration}
	registry := newPlanningRegistry().WithPluginAdmission(func(artifact Artifact) bool {
		return artifact == plugin.Artifact
	})
	if _, err := registry.Publish(plugin); err != nil {
		t.Fatal(err)
	}
	var active, acquired, released, providerCalls atomic.Int32
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		providerCalls.Add(1)
		if active.Load() != 1 {
			return ProviderExecutionResult{}, errors.New("query lease is not held")
		}
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "cached"}}}, nil
	})
	providers, err := NewStaticProviderResolver([]ExecutableProviderBinding{{
		QueryID: declaration.ID, ContractVersion: declaration.ContractVersion,
		PlanVersion: declaration.PlanVersion, ResultSchema: declaration.ResultSchema,
		Artifact: plugin.Artifact, Provider: provider,
	}})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := NewExecutionRuntime(ExecutionConfig{
		Registry: registry, Providers: providers, Schemas: allowExecutionSchema(), Cache: newMemoryQueryResultCache(),
		Admission: contextualTestAdmission(func(_ context.Context, artifact Artifact) (func(), error) {
			if artifact != plugin.Artifact {
				return nil, ErrArtifactUnavailable
			}
			acquired.Add(1)
			active.Add(1)
			return func() {
				active.Add(-1)
				released.Add(1)
			}, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	request := PlanRequest{QueryID: declaration.ID, Pagination: PaginationRequest{Limit: 10}}
	first, err := runtime.Execute(t.Context(), request)
	if err != nil || first.CacheHit {
		t.Fatalf("fresh result=%#v err=%v", first, err)
	}
	second, err := runtime.Execute(t.Context(), request)
	if err != nil || !second.CacheHit || providerCalls.Load() != 1 || acquired.Load() != 2 ||
		released.Load() != 2 || active.Load() != 0 {
		t.Fatalf("cached result=%#v err=%v provider=%d acquired=%d released=%d active=%d",
			second, err, providerCalls.Load(), acquired.Load(), released.Load(), active.Load())
	}
}

func TestExecutionRejectsReleaseOnlyAdmissionForPluginArtifact(t *testing.T) {
	plugin := publication("plugin.legacy-admission", false, 'a')
	declaration := query("plugin.legacy-admission.items", "plugin.legacy-admission.item", PaginationNone, PermissionPolicyPublic)
	declaration.Relations = nil
	plugin.Queries = []QueryDeclaration{declaration}
	registry := newPlanningRegistry().WithPluginAdmission(func(artifact Artifact) bool { return artifact == plugin.Artifact })
	if _, err := registry.Publish(plugin); err != nil {
		t.Fatal(err)
	}
	var admissionCalls, providerCalls atomic.Int32
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		providerCalls.Add(1)
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1"}}}, nil
	})
	providers, err := NewStaticProviderResolver([]ExecutableProviderBinding{executionProviderBinding(declaration, plugin.Artifact, provider)})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := NewExecutionRuntime(ExecutionConfig{
		Registry: registry, Providers: providers, Schemas: allowExecutionSchema(),
		Admission: ExecutionAdmissionFunc(func(context.Context, Artifact) (func(), error) {
			admissionCalls.Add(1)
			return func() {}, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Execute(t.Context(), PlanRequest{QueryID: declaration.ID}); !errors.Is(err, ErrArtifactUnavailable) || admissionCalls.Load() != 0 || providerCalls.Load() != 0 {
		t.Fatalf("legacy admission error=%v admission=%d provider=%d", err, admissionCalls.Load(), providerCalls.Load())
	}
}

func TestExecutionPreservesCallerCancellationDuringPluginAdmission(t *testing.T) {
	plugin := publication("plugin.cancel-admission", false, 'a')
	declaration := query("plugin.cancel-admission.items", "plugin.cancel-admission.item", PaginationNone, PermissionPolicyPublic)
	declaration.Relations = nil
	plugin.Queries = []QueryDeclaration{declaration}
	registry := newPlanningRegistry().WithPluginAdmission(func(artifact Artifact) bool { return artifact == plugin.Artifact })
	if _, err := registry.Publish(plugin); err != nil {
		t.Fatal(err)
	}
	var providerCalls atomic.Int32
	providers, err := NewStaticProviderResolver([]ExecutableProviderBinding{executionProviderBinding(
		declaration,
		plugin.Artifact,
		ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
			providerCalls.Add(1)
			return ProviderExecutionResult{Rows: []QueryRow{{"id": "1"}}}, nil
		}),
	)})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancelCause(t.Context())
	callerCause := errors.New("caller stopped admission")
	runtime, err := NewExecutionRuntime(ExecutionConfig{
		Registry: registry, Providers: providers, Schemas: allowExecutionSchema(),
		Admission: ContextualExecutionAdmissionFunc(func(context.Context, Artifact) (ExecutionAdmissionLease, error) {
			cancel(callerCause)
			return ExecutionAdmissionLease{}, callerCause
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Execute(ctx, PlanRequest{QueryID: declaration.ID}); !errors.Is(err, context.Canceled) ||
		!errors.Is(err, callerCause) ||
		errors.Is(err, ErrArtifactUnavailable) || providerCalls.Load() != 0 {
		t.Fatalf("caller cancellation error=%v provider=%d", err, providerCalls.Load())
	}
}

func TestExecutionPreservesIndependentAdmissionFailureBeforeLaterCallerCancel(t *testing.T) {
	plugin := publication("plugin.force-admission", false, 'a')
	declaration := query("plugin.force-admission.items", "plugin.force-admission.item", PaginationNone, PermissionPolicyPublic)
	declaration.Relations = nil
	plugin.Queries = []QueryDeclaration{declaration}
	registry := newPlanningRegistry().WithPluginAdmission(func(artifact Artifact) bool { return artifact == plugin.Artifact })
	if _, err := registry.Publish(plugin); err != nil {
		t.Fatal(err)
	}
	providers, err := NewStaticProviderResolver([]ExecutableProviderBinding{executionProviderBinding(
		declaration,
		plugin.Artifact,
		ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
			t.Fatal("provider executed after independent admission failure")
			return ProviderExecutionResult{}, nil
		}),
	)})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelCaller := context.WithCancelCause(t.Context())
	forceCause := errors.New("runtime already forced")
	callerCause := errors.New("later caller cancellation")
	runtime, err := NewExecutionRuntime(ExecutionConfig{
		Registry: registry, Providers: providers, Schemas: allowExecutionSchema(),
		Admission: ContextualExecutionAdmissionFunc(func(context.Context, Artifact) (ExecutionAdmissionLease, error) {
			independent := forceCause
			cancelCaller(callerCause)
			return ExecutionAdmissionLease{}, independent
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Execute(ctx, PlanRequest{QueryID: declaration.ID}); !errors.Is(err, ErrArtifactUnavailable) || !errors.Is(err, forceCause) || errors.Is(err, callerCause) {
		t.Fatalf("independent admission error=%v", err)
	}
}

func TestExecutionPropagatesOwnerLeaseCancellationToProvider(t *testing.T) {
	plugin := publication("plugin.cancelled-query", false, 'a')
	declaration := query("plugin.cancelled-query.items", "plugin.cancelled-query.item", PaginationNone, PermissionPolicyPublic)
	declaration.Relations = nil
	plugin.Queries = []QueryDeclaration{declaration}
	registry := newPlanningRegistry().WithPluginAdmission(func(artifact Artifact) bool {
		return artifact == plugin.Artifact
	})
	if _, err := registry.Publish(plugin); err != nil {
		t.Fatal(err)
	}
	providerStarted := make(chan struct{})
	provider := ExecutableProviderFunc(func(ctx context.Context, _ ProviderExecutionRequest) (ProviderExecutionResult, error) {
		close(providerStarted)
		<-ctx.Done()
		return ProviderExecutionResult{}, ctx.Err()
	})
	providers, err := NewStaticProviderResolver([]ExecutableProviderBinding{executionProviderBinding(declaration, plugin.Artifact, provider)})
	if err != nil {
		t.Fatal(err)
	}
	cancelReady := make(chan context.CancelCauseFunc, 1)
	var released atomic.Int32
	admission := ContextualExecutionAdmissionFunc(func(ctx context.Context, artifact Artifact) (ExecutionAdmissionLease, error) {
		if artifact != plugin.Artifact {
			return ExecutionAdmissionLease{}, ErrArtifactUnavailable
		}
		leaseCtx, cancel := context.WithCancelCause(ctx)
		cancelReady <- cancel
		return ExecutionAdmissionLease{Context: leaseCtx, Release: func() {
			cancel(nil)
			released.Add(1)
		}}, nil
	})
	runtime, err := NewExecutionRuntime(ExecutionConfig{
		Registry: registry, Providers: providers, Schemas: allowExecutionSchema(), Admission: admission,
	})
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		_, executeErr := runtime.Execute(t.Context(), PlanRequest{QueryID: declaration.ID})
		result <- executeErr
	}()
	cancel := <-cancelReady
	select {
	case <-providerStarted:
	case <-time.After(time.Second):
		t.Fatal("provider did not receive the contextual execution lease")
	}
	forceCause := errors.New("forced owner drain")
	cancel(forceCause)
	select {
	case err := <-result:
		if !errors.Is(err, ErrArtifactUnavailable) || !errors.Is(err, forceCause) || released.Load() != 1 {
			t.Fatalf("owner cancellation error=%v released=%d", err, released.Load())
		}
	case <-time.After(time.Second):
		t.Fatal("owner lease cancellation did not stop provider execution")
	}
}

func TestExecutionPreservesInflightCustomCallerCancellation(t *testing.T) {
	plugin := publication("plugin.custom-cancel", false, 'a')
	declaration := query("plugin.custom-cancel.items", "plugin.custom-cancel.item", PaginationNone, PermissionPolicyPublic)
	declaration.Relations = nil
	plugin.Queries = []QueryDeclaration{declaration}
	registry := newPlanningRegistry().WithPluginAdmission(func(artifact Artifact) bool { return artifact == plugin.Artifact })
	if _, err := registry.Publish(plugin); err != nil {
		t.Fatal(err)
	}
	providerStarted := make(chan struct{})
	provider := ExecutableProviderFunc(func(ctx context.Context, _ ProviderExecutionRequest) (ProviderExecutionResult, error) {
		close(providerStarted)
		<-ctx.Done()
		return ProviderExecutionResult{}, ctx.Err()
	})
	providers, err := NewStaticProviderResolver([]ExecutableProviderBinding{executionProviderBinding(declaration, plugin.Artifact, provider)})
	if err != nil {
		t.Fatal(err)
	}
	var released atomic.Int32
	runtime, err := NewExecutionRuntime(ExecutionConfig{
		Registry: registry, Providers: providers, Schemas: allowExecutionSchema(),
		Admission: ContextualExecutionAdmissionFunc(func(ctx context.Context, artifact Artifact) (ExecutionAdmissionLease, error) {
			if artifact != plugin.Artifact {
				return ExecutionAdmissionLease{}, ErrArtifactUnavailable
			}
			leaseCtx, cancel := context.WithCancelCause(ctx)
			return ExecutionAdmissionLease{Context: leaseCtx, Release: func() {
				cancel(nil)
				released.Add(1)
			}}, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelCaller := context.WithCancelCause(t.Context())
	result := make(chan error, 1)
	go func() {
		_, executeErr := runtime.Execute(ctx, PlanRequest{QueryID: declaration.ID})
		result <- executeErr
	}()
	select {
	case <-providerStarted:
	case <-time.After(time.Second):
		t.Fatal("provider did not start")
	}
	callerCause := errors.New("caller navigation closed query")
	cancelCaller(callerCause)
	select {
	case executeErr := <-result:
		if !errors.Is(executeErr, context.Canceled) || !errors.Is(executeErr, callerCause) ||
			errors.Is(executeErr, ErrArtifactUnavailable) || released.Load() != 1 ||
			executionTraceOutcome(executeErr) != TraceOutcomeCancelled {
			t.Fatalf("custom in-flight cancellation error=%v released=%d", executeErr, released.Load())
		}
	case <-time.After(time.Second):
		t.Fatal("custom caller cancellation did not stop provider")
	}
}

func TestExecutionPropagatesFailOpenFilterLeaseCancellationToCallback(t *testing.T) {
	filterArtifact := publication("plugin.cancelled-filter", false, 'f').Artifact
	filterStarted := make(chan struct{})
	filter := executionTestFilter(filterArtifact, "plugin.cancelled-filter.decorate", 10, ResultFilterFailOpen, nil)
	filter.Filter = ResultFilterFunc(func(ctx context.Context, _ ResultFilterRequest) (ResultFilterResult, error) {
		close(filterStarted)
		<-ctx.Done()
		return ResultFilterResult{}, ctx.Err()
	})
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "base"}}}, nil
	})
	cancelReady := make(chan context.CancelCauseFunc, 1)
	var released atomic.Int32
	runtime, _ := executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, provider, []ResultFilterRegistration{filter}, func(config *ExecutionConfig) {
		config.Registry.WithPluginAdmission(func(artifact Artifact) bool { return artifact == filterArtifact })
		config.Admission = ContextualExecutionAdmissionFunc(func(ctx context.Context, artifact Artifact) (ExecutionAdmissionLease, error) {
			if artifact != filterArtifact {
				return ExecutionAdmissionLease{}, ErrArtifactUnavailable
			}
			leaseCtx, cancel := context.WithCancelCause(ctx)
			cancelReady <- cancel
			return ExecutionAdmissionLease{Context: leaseCtx, Release: func() {
				cancel(nil)
				released.Add(1)
			}}, nil
		})
	})
	result := make(chan error, 1)
	go func() {
		_, executeErr := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"})
		result <- executeErr
	}()
	cancel := <-cancelReady
	select {
	case <-filterStarted:
	case <-time.After(time.Second):
		t.Fatal("filter did not receive the contextual execution lease")
	}
	forceCause := errors.New("forced filter drain")
	cancel(forceCause)
	select {
	case err := <-result:
		if !errors.Is(err, ErrArtifactUnavailable) || !errors.Is(err, forceCause) || released.Load() != 1 {
			t.Fatalf("filter cancellation error=%v released=%d", err, released.Load())
		}
	case <-time.After(time.Second):
		t.Fatal("filter lease cancellation did not stop callback execution")
	}
}

func TestExecutionSynchronouslyFencesFastOwnerLeaseCancellation(t *testing.T) {
	plugin := publication("plugin.fast-cancel", false, 'a')
	declaration := query("plugin.fast-cancel.items", "plugin.fast-cancel.item", PaginationNone, PermissionPolicyPublic)
	declaration.Relations = nil
	plugin.Queries = []QueryDeclaration{declaration}
	registry := newPlanningRegistry().WithPluginAdmission(func(artifact Artifact) bool { return artifact == plugin.Artifact })
	if _, err := registry.Publish(plugin); err != nil {
		t.Fatal(err)
	}
	forceCause := errors.New("fast owner cancellation")
	cancelReady := make(chan context.CancelCauseFunc, 1)
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		cancel := <-cancelReady
		cancel(forceCause)
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1"}}}, nil
	})
	providers, err := NewStaticProviderResolver([]ExecutableProviderBinding{executionProviderBinding(declaration, plugin.Artifact, provider)})
	if err != nil {
		t.Fatal(err)
	}
	var released atomic.Int32
	runtime, err := NewExecutionRuntime(ExecutionConfig{
		Registry: registry, Providers: providers, Schemas: allowExecutionSchema(),
		Admission: ContextualExecutionAdmissionFunc(func(ctx context.Context, artifact Artifact) (ExecutionAdmissionLease, error) {
			if artifact != plugin.Artifact {
				return ExecutionAdmissionLease{}, ErrArtifactUnavailable
			}
			leaseCtx, cancel := context.WithCancelCause(ctx)
			cancelReady <- cancel
			return ExecutionAdmissionLease{Context: leaseCtx, Release: func() { released.Add(1) }}, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Execute(t.Context(), PlanRequest{QueryID: declaration.ID}); !errors.Is(err, ErrArtifactUnavailable) || !errors.Is(err, forceCause) || released.Load() != 1 {
		t.Fatalf("fast owner cancellation error=%v released=%d", err, released.Load())
	}
}

func TestExecutionSynchronouslyFencesFastFailOpenFilterLeaseCancellation(t *testing.T) {
	filterArtifact := publication("plugin.fast-filter", false, 'f').Artifact
	forceCause := errors.New("fast filter cancellation")
	cancelReady := make(chan context.CancelCauseFunc, 1)
	filter := executionTestFilter(filterArtifact, "plugin.fast-filter.decorate", 10, ResultFilterFailOpen, nil)
	filter.Filter = ResultFilterFunc(func(_ context.Context, request ResultFilterRequest) (ResultFilterResult, error) {
		cancel := <-cancelReady
		cancel(forceCause)
		return ResultFilterResult{Rows: request.Rows}, nil
	})
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "base"}}}, nil
	})
	var released atomic.Int32
	runtime, _ := executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, provider, []ResultFilterRegistration{filter}, func(config *ExecutionConfig) {
		config.Registry.WithPluginAdmission(func(artifact Artifact) bool { return artifact == filterArtifact })
		config.Admission = ContextualExecutionAdmissionFunc(func(ctx context.Context, artifact Artifact) (ExecutionAdmissionLease, error) {
			if artifact != filterArtifact {
				return ExecutionAdmissionLease{}, ErrArtifactUnavailable
			}
			leaseCtx, cancel := context.WithCancelCause(ctx)
			cancelReady <- cancel
			return ExecutionAdmissionLease{Context: leaseCtx, Release: func() { released.Add(1) }}, nil
		})
	})
	if _, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"}); !errors.Is(err, ErrArtifactUnavailable) || !errors.Is(err, forceCause) || released.Load() != 1 {
		t.Fatalf("fast fail-open filter cancellation error=%v released=%d", err, released.Load())
	}
}

func TestExecutionPrefersExactForceDrainFromHostCallbacks(t *testing.T) {
	tests := []struct {
		name              string
		callback          string
		wantProviderCalls int32
	}{
		{name: "resolver", callback: "resolver"},
		{name: "cache_load", callback: "cache_load"},
		{name: "cache_store", callback: "cache_store", wantProviderCalls: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plugin := publication("plugin.callback-cancel-"+test.name, false, 'a')
			declaration := query("plugin.callback-cancel-"+test.name+".items", "plugin.callback-cancel.item", PaginationOffset, PermissionPolicyPublic)
			declaration.Relations = nil
			plugin.Queries = []QueryDeclaration{declaration}
			registry := newPlanningRegistry().WithPluginAdmission(func(artifact Artifact) bool { return artifact == plugin.Artifact })
			if _, err := registry.Publish(plugin); err != nil {
				t.Fatal(err)
			}
			forceCause := errors.New(test.name + " ForceDrain")
			cancelReady := make(chan context.CancelCauseFunc, 1)
			var providerCalls, released atomic.Int32
			provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
				providerCalls.Add(1)
				return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "result"}}}, nil
			})
			binding := executionProviderBinding(declaration, plugin.Artifact, provider)
			var providers ExecutableProviderResolver
			if test.callback == "resolver" {
				providers = executableProviderResolverFunc(func(context.Context, QueryPlan) (ExecutableProviderBinding, error) {
					cancel := <-cancelReady
					cancel(forceCause)
					return ExecutableProviderBinding{}, context.Canceled
				})
			} else {
				var err error
				providers, err = NewStaticProviderResolver([]ExecutableProviderBinding{binding})
				if err != nil {
					t.Fatal(err)
				}
			}
			var cache QueryResultCache
			switch test.callback {
			case "cache_load":
				cache = executionCallbackCache{
					load: func(context.Context, string) (CachedQueryResult, bool, error) {
						cancel := <-cancelReady
						cancel(forceCause)
						return CachedQueryResult{}, false, context.Canceled
					},
				}
			case "cache_store":
				cache = executionCallbackCache{
					load: func(context.Context, string) (CachedQueryResult, bool, error) {
						return CachedQueryResult{}, false, nil
					},
					store: func(context.Context, string, CachedQueryResult, []string) error {
						cancel := <-cancelReady
						cancel(forceCause)
						return context.Canceled
					},
				}
			}
			runtime, err := NewExecutionRuntime(ExecutionConfig{
				Registry: registry, Providers: providers, Schemas: allowExecutionSchema(), Cache: cache,
				Admission: ContextualExecutionAdmissionFunc(func(ctx context.Context, artifact Artifact) (ExecutionAdmissionLease, error) {
					if artifact != plugin.Artifact {
						return ExecutionAdmissionLease{}, ErrArtifactUnavailable
					}
					leaseCtx, cancel := context.WithCancelCause(ctx)
					cancelReady <- cancel
					return ExecutionAdmissionLease{Context: leaseCtx, Release: func() { released.Add(1) }}, nil
				}),
			})
			if err != nil {
				t.Fatal(err)
			}
			_, executeErr := runtime.Execute(t.Context(), PlanRequest{
				QueryID: declaration.ID, Pagination: PaginationRequest{Limit: 10},
			})
			if !errors.Is(executeErr, ErrArtifactUnavailable) || !errors.Is(executeErr, forceCause) ||
				errors.Is(executeErr, ErrProviderUnavailable) || providerCalls.Load() != test.wantProviderCalls || released.Load() != 1 {
				t.Fatalf("callback error=%v provider=%d released=%d", executeErr, providerCalls.Load(), released.Load())
			}
		})
	}
}

func TestExecutionTraceClassifiesForceDrainBeforeCancellationShape(t *testing.T) {
	err := errors.Join(ErrArtifactUnavailable, context.Canceled)
	if outcome := executionTraceOutcome(err); outcome != TraceOutcomeRuntimeStale {
		t.Fatalf("ForceDrain outcome=%q", outcome)
	}
}

func TestExecutionFencesCancellationInsideFinalPermissionRecheck(t *testing.T) {
	for _, cacheHit := range []bool{false, true} {
		name := "fresh"
		pagination := PaginationNone
		finalCheck := int32(4)
		if cacheHit {
			name = "cache_hit"
			pagination = PaginationOffset
			finalCheck = 3
		}
		t.Run(name, func(t *testing.T) {
			plugin := publication("plugin.final-permission-"+name, false, 'a')
			declaration := query("plugin.final-permission-"+name+".items", "plugin.final-permission.item", pagination, PermissionPolicyPublic)
			declaration.Relations = nil
			plugin.Queries = []QueryDeclaration{declaration}
			var admitted atomic.Bool
			admitted.Store(true)
			registry := newPlanningRegistry().WithPluginAdmission(func(artifact Artifact) bool {
				return admitted.Load() && artifact == plugin.Artifact
			})
			if _, err := registry.Publish(plugin); err != nil {
				t.Fatal(err)
			}
			provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
				return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "result"}}}, nil
			})
			providers, err := NewStaticProviderResolver([]ExecutableProviderBinding{executionProviderBinding(declaration, plugin.Artifact, provider)})
			if err != nil {
				t.Fatal(err)
			}
			cancelReady := make(chan context.CancelCauseFunc, 2)
			var released atomic.Int32
			runtime, err := NewExecutionRuntime(ExecutionConfig{
				Registry: registry, Providers: providers, Schemas: allowExecutionSchema(), Cache: newMemoryQueryResultCache(),
				Admission: ContextualExecutionAdmissionFunc(func(ctx context.Context, artifact Artifact) (ExecutionAdmissionLease, error) {
					if artifact != plugin.Artifact {
						return ExecutionAdmissionLease{}, ErrArtifactUnavailable
					}
					leaseCtx, cancel := context.WithCancelCause(ctx)
					cancelReady <- cancel
					return ExecutionAdmissionLease{Context: leaseCtx, Release: func() { released.Add(1) }}, nil
				}),
			})
			if err != nil {
				t.Fatal(err)
			}
			request := PlanRequest{QueryID: declaration.ID}
			if cacheHit {
				request.Pagination.Limit = 10
				if result, err := runtime.Execute(t.Context(), request); err != nil || result.CacheHit {
					t.Fatalf("prime cache result=%#v err=%v", result, err)
				}
				<-cancelReady
			}
			forceCause := errors.New("final permission ForceDrain")
			var checks atomic.Int32
			request.Permission = PermissionInput{
				ActorFingerprint: "actor", PolicyFingerprint: "public-v1",
				Recheck: PermissionRecheckFunc(func(context.Context, PermissionClaim) error {
					if checks.Add(1) == finalCheck {
						cancel := <-cancelReady
						cancel(forceCause)
						admitted.Store(false)
						return context.Canceled
					}
					return nil
				}),
			}
			if _, err := runtime.Execute(t.Context(), request); !errors.Is(err, ErrArtifactUnavailable) ||
				!errors.Is(err, forceCause) || errors.Is(err, ErrDenied) {
				t.Fatalf("final permission cancellation error=%v checks=%d", err, checks.Load())
			}
			wantReleased := int32(1)
			if cacheHit {
				wantReleased = 2
			}
			if released.Load() != wantReleased {
				t.Fatalf("released=%d want=%d", released.Load(), wantReleased)
			}
		})
	}
}

func TestExecutionPreservesIndependentForceCauseAcrossMultipleArtifacts(t *testing.T) {
	owner := publication("plugin.a-query-owner", false, 'a')
	declaration := query("plugin.a-query-owner.items", "plugin.a-query-owner.item", PaginationNone, PermissionPolicyPublic)
	declaration.Relations = nil
	owner.Queries = []QueryDeclaration{declaration}
	filterArtifact := publication("plugin.z-query-filter", false, 'f').Artifact
	registry := newPlanningRegistry().WithPluginAdmission(func(artifact Artifact) bool {
		return artifact == owner.Artifact || artifact == filterArtifact
	})
	if _, err := registry.Publish(owner); err != nil {
		t.Fatal(err)
	}
	requestCtx, cancelRequest := context.WithCancelCause(t.Context())
	forceCause := errors.New("filter ForceDrain won before caller cancel")
	cancels := map[Artifact]context.CancelCauseFunc{}
	releases := map[Artifact]*atomic.Int32{
		owner.Artifact: {}, filterArtifact: {},
	}
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		cancels[filterArtifact](forceCause)
		cancelRequest(errors.New("later caller cancellation"))
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "result"}}}, nil
	})
	providers, err := NewStaticProviderResolver([]ExecutableProviderBinding{executionProviderBinding(declaration, owner.Artifact, provider)})
	if err != nil {
		t.Fatal(err)
	}
	filter := executionTestFilter(filterArtifact, "plugin.z-query-filter.decorate", 10, ResultFilterFailOpen, func(rows []QueryRow) []QueryRow { return rows })
	filter.QueryID = declaration.ID
	filter.QueryContractVersion = declaration.ContractVersion
	filter.QueryPlanVersion = declaration.PlanVersion
	filter.Dependency = ResultFilterDependency{ExtensionID: owner.Artifact.ExtensionID, VersionConstraint: "^1.0.0"}
	runtime, err := NewExecutionRuntime(ExecutionConfig{
		Registry: registry, Providers: providers, Schemas: allowExecutionSchema(), ResultFilters: []ResultFilterRegistration{filter},
		Admission: ContextualExecutionAdmissionFunc(func(ctx context.Context, artifact Artifact) (ExecutionAdmissionLease, error) {
			leaseCtx, cancel := context.WithCancelCause(ctx)
			cancels[artifact] = cancel
			return ExecutionAdmissionLease{Context: leaseCtx, Release: func() { releases[artifact].Add(1) }}, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Execute(requestCtx, PlanRequest{QueryID: declaration.ID}); !errors.Is(err, ErrArtifactUnavailable) || !errors.Is(err, forceCause) {
		t.Fatalf("multi-artifact cancellation error=%v", err)
	}
	for artifact, released := range releases {
		if released.Load() != 1 {
			t.Fatalf("artifact %s released=%d", artifact.ExtensionID, released.Load())
		}
	}
}

func TestExecutionPreservesForceCauseAcrossSchemaAndCacheValidation(t *testing.T) {
	for _, cacheHit := range []bool{false, true} {
		name := "fresh_schema"
		pagination := PaginationNone
		if cacheHit {
			name = "cached_schema"
			pagination = PaginationOffset
		}
		t.Run(name, func(t *testing.T) {
			plugin := publication("plugin.schema-cancel-"+name, false, 'a')
			declaration := query("plugin.schema-cancel-"+name+".items", "plugin.schema-cancel.item", pagination, PermissionPolicyPublic)
			declaration.Relations = nil
			plugin.Queries = []QueryDeclaration{declaration}
			registry := newPlanningRegistry().WithPluginAdmission(func(artifact Artifact) bool { return artifact == plugin.Artifact })
			if _, err := registry.Publish(plugin); err != nil {
				t.Fatal(err)
			}
			provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
				return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "result"}}}, nil
			})
			providers, err := NewStaticProviderResolver([]ExecutableProviderBinding{executionProviderBinding(declaration, plugin.Artifact, provider)})
			if err != nil {
				t.Fatal(err)
			}
			cancelReady := make(chan context.CancelCauseFunc, 2)
			forceCause := errors.New("schema validation ForceDrain")
			var armed atomic.Bool
			runtime, err := NewExecutionRuntime(ExecutionConfig{
				Registry: registry, Providers: providers, Cache: newMemoryQueryResultCache(),
				Schemas: ResultSchemaValidatorFunc(func(context.Context, ResultSchemaClaim, QueryRow) error {
					if armed.CompareAndSwap(true, false) {
						cancel := <-cancelReady
						cancel(forceCause)
						return context.Canceled
					}
					return nil
				}),
				Admission: ContextualExecutionAdmissionFunc(func(ctx context.Context, artifact Artifact) (ExecutionAdmissionLease, error) {
					if artifact != plugin.Artifact {
						return ExecutionAdmissionLease{}, ErrArtifactUnavailable
					}
					leaseCtx, cancel := context.WithCancelCause(ctx)
					cancelReady <- cancel
					return ExecutionAdmissionLease{Context: leaseCtx, Release: func() {}}, nil
				}),
			})
			if err != nil {
				t.Fatal(err)
			}
			request := PlanRequest{QueryID: declaration.ID}
			if cacheHit {
				request.Pagination.Limit = 10
				if result, err := runtime.Execute(t.Context(), request); err != nil || result.CacheHit {
					t.Fatalf("prime schema cache result=%#v err=%v", result, err)
				}
				<-cancelReady
			}
			armed.Store(true)
			if _, err := runtime.Execute(t.Context(), request); !errors.Is(err, ErrArtifactUnavailable) ||
				!errors.Is(err, forceCause) ||
				errors.Is(err, ErrCachePoisoned) || errors.Is(err, ErrResultInvalid) {
				t.Fatalf("schema cancellation error=%v", err)
			}
		})
	}
}

func executionProviderBinding(
	declaration QueryDeclaration,
	artifact Artifact,
	provider ExecutableProvider,
) ExecutableProviderBinding {
	return ExecutableProviderBinding{
		QueryID: declaration.ID, ContractVersion: declaration.ContractVersion,
		PlanVersion: declaration.PlanVersion, ResultSchema: declaration.ResultSchema,
		Artifact: artifact, Provider: provider,
	}
}

func contextualTestAdmission(
	acquire func(context.Context, Artifact) (func(), error),
) ContextualExecutionAdmissionFunc {
	return func(ctx context.Context, artifact Artifact) (ExecutionAdmissionLease, error) {
		release, err := acquire(ctx, artifact)
		if err != nil {
			return ExecutionAdmissionLease{}, err
		}
		if release == nil {
			return ExecutionAdmissionLease{}, ErrArtifactUnavailable
		}
		leaseCtx, cancel := context.WithCancelCause(ctx)
		return ExecutionAdmissionLease{
			Context: leaseCtx,
			Release: func() {
				cancel(nil)
				release()
			},
		}, nil
	}
}

type executionCallbackCache struct {
	load  func(context.Context, string) (CachedQueryResult, bool, error)
	store func(context.Context, string, CachedQueryResult, []string) error
}

func (c executionCallbackCache) LoadQueryResult(ctx context.Context, key string) (CachedQueryResult, bool, error) {
	if c.load == nil {
		return CachedQueryResult{}, false, nil
	}
	return c.load(ctx, key)
}

func (c executionCallbackCache) StoreQueryResult(ctx context.Context, key string, value CachedQueryResult, tags []string) error {
	if c.store == nil {
		return nil
	}
	return c.store(ctx, key, value, tags)
}

func TestExecutionHoldsResultFilterLeaseAcrossFinalPermissionAndBypassesUnsafeCache(t *testing.T) {
	filterArtifact := publication("plugin.release-filter", false, 'f').Artifact
	var filterCalls atomic.Int32
	filter := executionTestFilter(filterArtifact, "plugin.release-filter.decorate", 10, ResultFilterFailClosed, func(rows []QueryRow) []QueryRow {
		rows[0]["title"] = "filtered-" + strconv.Itoa(int(filterCalls.Add(1)))
		return rows
	})
	var active, acquired, released, checks atomic.Int32
	permission := PermissionInput{
		ActorFingerprint: "actor-1", PolicyFingerprint: "public-v1",
		Recheck: PermissionRecheckFunc(func(context.Context, PermissionClaim) error {
			call := checks.Add(1)
			// Plan authorization happens before execution leases exist. Every later
			// Host recheck must observe the filter artifact retained.
			if call > 1 && active.Load() != 1 {
				return errors.New("filter lease released before Host permission fence")
			}
			return nil
		}),
	}
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "base"}}}, nil
	})
	runtime, _ := executionTestRuntime(t, PaginationOffset, PermissionPolicyPublic, provider, []ResultFilterRegistration{filter}, func(config *ExecutionConfig) {
		config.Cache = newMemoryQueryResultCache()
		config.Registry.WithPluginAdmission(func(artifact Artifact) bool { return artifact == filterArtifact })
		config.Admission = contextualTestAdmission(func(_ context.Context, artifact Artifact) (func(), error) {
			if artifact != filterArtifact {
				return nil, ErrArtifactUnavailable
			}
			acquired.Add(1)
			active.Add(1)
			return func() {
				active.Add(-1)
				released.Add(1)
			}, nil
		})
	})
	request := PlanRequest{
		QueryID: "core.execute.items", Pagination: PaginationRequest{Limit: 10}, Permission: permission,
	}
	first, err := runtime.Execute(t.Context(), request)
	if err != nil || first.CacheHit || first.Rows[0]["title"] != "filtered-1" {
		t.Fatalf("fresh filtered result=%#v err=%v", first, err)
	}
	checks.Store(0)
	second, err := runtime.Execute(t.Context(), request)
	if err != nil || second.CacheHit || second.Rows[0]["title"] != "filtered-2" || filterCalls.Load() != 2 ||
		acquired.Load() != 2 || released.Load() != 2 || active.Load() != 0 {
		t.Fatalf("uncached filtered result=%#v err=%v filters=%d acquired=%d released=%d active=%d",
			second, err, filterCalls.Load(), acquired.Load(), released.Load(), active.Load())
	}
}

func TestExecutionFailOpenFilterCannotOverrideHostPermissionDenial(t *testing.T) {
	filterArtifact := publication("plugin.fail-open", false, 'f').Artifact
	filter := executionTestFilter(filterArtifact, "plugin.fail-open.decorate", 10, ResultFilterFailOpen, func(rows []QueryRow) []QueryRow {
		return rows
	})
	var checks atomic.Int32
	permission := PermissionInput{
		Authenticated: true, ActorFingerprint: "actor-1", PolicyFingerprint: "role:reader",
		Recheck: PermissionRecheckFunc(func(context.Context, PermissionClaim) error {
			if checks.Add(1) == 3 {
				return ErrDenied
			}
			return nil
		}),
	}
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "private"}}}, nil
	})
	runtime, _ := executionTestRuntime(t, PaginationNone, "core.execute.read", provider, []ResultFilterRegistration{filter}, func(config *ExecutionConfig) {
		config.Registry.WithPluginAdmission(func(artifact Artifact) bool { return artifact == filterArtifact })
	})
	if _, err := runtime.Execute(t.Context(), PlanRequest{
		QueryID: "core.execute.items", Permission: permission,
	}); !errors.Is(err, ErrDenied) || checks.Load() != 3 {
		t.Fatalf("fail-open permission result err=%v checks=%d", err, checks.Load())
	}
}

func TestExecutionFailOpenFilterCannotOverrideHostResultValidation(t *testing.T) {
	filterArtifact := publication("plugin.fail-open-schema", false, 'f').Artifact
	filter := executionTestFilter(
		filterArtifact,
		"plugin.fail-open-schema.decorate",
		10,
		ResultFilterFailOpen,
		func(rows []QueryRow) []QueryRow {
			rows[0]["secret"] = "undeclared"
			return rows
		},
	)
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "base"}}}, nil
	})
	runtime, _ := executionTestRuntime(
		t,
		PaginationNone,
		PermissionPolicyPublic,
		provider,
		[]ResultFilterRegistration{filter},
		func(config *ExecutionConfig) {
			config.Registry.WithPluginAdmission(func(artifact Artifact) bool { return artifact == filterArtifact })
		},
	)

	if _, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"}); !errors.Is(err, ErrResultInvalid) {
		t.Fatalf("fail-open Host result validation error=%v", err)
	}
}

func TestExecutionFailOpenFilterCannotOverrideHostAdmissionFailure(t *testing.T) {
	filterArtifact := publication("plugin.fail-open-admission", false, 'f').Artifact
	filter := executionTestFilter(
		filterArtifact,
		"plugin.fail-open-admission.decorate",
		10,
		ResultFilterFailOpen,
		func(rows []QueryRow) []QueryRow { return rows },
	)
	var providerCalls atomic.Int32
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		providerCalls.Add(1)
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "base"}}}, nil
	})
	runtime, _ := executionTestRuntime(
		t,
		PaginationNone,
		PermissionPolicyPublic,
		provider,
		[]ResultFilterRegistration{filter},
		func(config *ExecutionConfig) {
			config.Registry.WithPluginAdmission(func(artifact Artifact) bool { return artifact == filterArtifact })
			config.Admission = contextualTestAdmission(func(context.Context, Artifact) (func(), error) {
				return nil, ErrArtifactUnavailable
			})
		},
	)

	if _, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"}); !errors.Is(err, ErrDependencyDenied) || providerCalls.Load() != 0 {
		t.Fatalf("fail-open Host admission error=%v provider calls=%d", err, providerCalls.Load())
	}
}

func TestExecutionSafeModeBypassesStaleThirdPartyResultFilters(t *testing.T) {
	filterArtifact := publication("plugin.safe-filter", false, 'e').Artifact
	var filterCalls, admissionCalls atomic.Int32
	filter := executionTestFilter(filterArtifact, "plugin.safe-filter.decorate", 10, ResultFilterFailClosed, func(rows []QueryRow) []QueryRow {
		filterCalls.Add(1)
		rows[0]["title"] = "unsafe"
		return rows
	})
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": "core"}}}, nil
	})
	runtime, registry := executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, provider, []ResultFilterRegistration{filter}, func(config *ExecutionConfig) {
		config.Registry.WithPluginAdmission(func(Artifact) bool { return true })
		config.Admission = contextualTestAdmission(func(context.Context, Artifact) (func(), error) {
			admissionCalls.Add(1)
			return func() {}, nil
		})
	})
	snapshot := registry.Snapshot()
	if _, err := registry.ReplaceAll(snapshot.Publications, true); err != nil {
		t.Fatal(err)
	}
	result, err := runtime.Execute(t.Context(), PlanRequest{QueryID: "core.execute.items"})
	if err != nil || len(result.Rows) != 1 || result.Rows[0]["title"] != "core" ||
		filterCalls.Load() != 0 || admissionCalls.Load() != 0 {
		t.Fatalf("Safe Mode result=%#v err=%v filters=%d admissions=%d",
			result, err, filterCalls.Load(), admissionCalls.Load())
	}
}

func TestExecutionRejectsDuplicatePaginationIdentityBeforeResultFilter(t *testing.T) {
	filterArtifact := publication("plugin.identity-filter", false, 'd').Artifact
	filter := executionTestFilter(filterArtifact, "plugin.identity-filter.decorate", 10, ResultFilterFailClosed, func(rows []QueryRow) []QueryRow {
		return rows
	})
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{Rows: []QueryRow{
			{"id": "1", "title": "one"}, {"id": "1", "title": "duplicate"}, {"id": "2", "title": "two"},
		}}, nil
	})
	runtime, _ := executionTestRuntime(t, PaginationCursor, PermissionPolicyPublic, provider, []ResultFilterRegistration{filter}, func(config *ExecutionConfig) {
		codec, err := NewHMACCursorCodec([]byte(strings.Repeat("identity-cursor-key", 2)))
		if err != nil {
			t.Fatal(err)
		}
		config.Registry.cursorCodec = codec
		config.Registry.WithPluginAdmission(func(artifact Artifact) bool { return artifact == filterArtifact })
	})
	if _, err := runtime.Execute(t.Context(), PlanRequest{
		QueryID: "core.execute.items", Pagination: PaginationRequest{Limit: 2},
	}); !errors.Is(err, ErrResultInvalid) {
		t.Fatalf("duplicate pagination identity=%v", err)
	}
}
