package queryregistry

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
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
		Admission: ExecutionAdmissionFunc(func(_ context.Context, artifact Artifact) (func(), error) {
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
		config.Admission = ExecutionAdmissionFunc(func(_ context.Context, artifact Artifact) (func(), error) {
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
			config.Admission = ExecutionAdmissionFunc(func(context.Context, Artifact) (func(), error) {
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
		config.Admission = ExecutionAdmissionFunc(func(context.Context, Artifact) (func(), error) {
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
