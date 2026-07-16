package queryregistry

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestExecutionConcurrentCacheAndFilterOwnershipIsolation(t *testing.T) {
	cache := newMemoryQueryResultCache()
	shared := QueryRow{"id": "1", "title": "base"}
	var providerCalls atomic.Int32
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		providerCalls.Add(1)
		return ProviderExecutionResult{Rows: []QueryRow{shared}}, nil
	})
	artifact := publication("plugin.race-filter", false, 'd').Artifact
	filter := ResultFilterRegistration{
		ID: "plugin.race-filter.redact", ContractVersion: "plugin.race-filter.redact@1",
		QueryID: "core.execute.items", QueryContractVersion: "core.execute.items@1",
		QueryPlanVersion: "core.execute.items.plan@1", Priority: 10, Artifact: artifact,
		Dependency:     ResultFilterDependency{ExtensionID: "core.execute", VersionConstraint: "^1.0.0"},
		IdentityFields: []string{"id"},
		FailurePolicy:  ResultFilterFailClosed,
		Filter: ResultFilterFunc(func(_ context.Context, request ResultFilterRequest) (ResultFilterResult, error) {
			request.Rows[0]["title"] = "filtered"
			return ResultFilterResult{Rows: request.Rows}, nil
		}),
	}
	runtime, _ := executionTestRuntime(t, PaginationOffset, PermissionPolicyPublic, provider, []ResultFilterRegistration{filter}, func(config *ExecutionConfig) {
		config.Cache = cache
		config.Registry.WithPluginAdmission(func(candidate Artifact) bool { return candidate == artifact })
	})

	start := make(chan struct{})
	errorsCh := make(chan error, 1)
	var group sync.WaitGroup
	for worker := 0; worker < 32; worker++ {
		group.Add(1)
		go func(worker int) {
			defer group.Done()
			<-start
			for iteration := 0; iteration < 50; iteration++ {
				result, err := runtime.Execute(context.Background(), PlanRequest{
					QueryID: "core.execute.items", Pagination: PaginationRequest{Limit: 10},
					Permission: PermissionInput{ActorFingerprint: fmt.Sprintf("actor-%d", worker%4), PolicyFingerprint: "public-v1"},
					Locale:     "en-US", Scope: "forum.main",
				})
				if err != nil || len(result.Rows) != 1 || result.Rows[0]["title"] != "filtered" {
					select {
					case errorsCh <- fmt.Errorf("worker %d iteration %d: result=%#v err=%v", worker, iteration, result, err):
					default:
					}
					return
				}
				result.Rows[0]["title"] = "caller-owned"
			}
		}(worker)
	}
	close(start)
	group.Wait()
	select {
	case err := <-errorsCh:
		t.Fatal(err)
	default:
	}
	if shared["title"] != "base" || providerCalls.Load() == 0 {
		t.Fatalf("provider ownership leaked: shared=%#v calls=%d", shared, providerCalls.Load())
	}
}

func TestExecutionConcurrentCursorCodecIsDeterministicAndRaceFree(t *testing.T) {
	codec, err := NewHMACCursorCodec([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	claims := CursorClaims{
		SchemaVersion: cursorSchemaVersion, QueryID: "core.execute.items", ContractVersion: "core.execute.items@1",
		PlanVersion: "core.execute.items.plan@1", ResultSchema: "core.execute.items.result@1",
		ShapeDigest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", RegistryRevision: 1,
		RegistryDigest:  "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		ArtifactDigest:  "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		IsolationDigest: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		ExecutionDigest: "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Offset:          20, Limit: 10,
	}
	want, err := codec.EncodeQueryCursor(claims)
	if err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	errorsCh := make(chan error, 1)
	for worker := 0; worker < 32; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for iteration := 0; iteration < 100; iteration++ {
				encoded, encodeErr := codec.EncodeQueryCursor(claims)
				decoded, decodeErr := codec.DecodeQueryCursor(encoded)
				if encodeErr != nil || decodeErr != nil || encoded != want || decoded != claims {
					select {
					case errorsCh <- fmt.Errorf("cursor encoded=%q decoded=%#v encode=%v decode=%v", encoded, decoded, encodeErr, decodeErr):
					default:
					}
					return
				}
			}
		}()
	}
	group.Wait()
	select {
	case err := <-errorsCh:
		t.Fatal(err)
	default:
	}
}

func TestExecutionConcurrentResultFilterBudgetsAreRequestLocal(t *testing.T) {
	artifact := publication("plugin.concurrent-filter-budget", false, 'b').Artifact
	filters := executionBudgetFilters(artifact, 4, ResultFilterFailClosed, func(_ context.Context, request ResultFilterRequest, _ int) (ResultFilterResult, error) {
		return ResultFilterResult{Rows: request.Rows}, nil
	})
	provider := ExecutableProviderFunc(func(context.Context, ProviderExecutionRequest) (ProviderExecutionResult, error) {
		return ProviderExecutionResult{Rows: []QueryRow{{"id": "1", "title": strings.Repeat("x", 64)}}}, nil
	})
	runtime, _ := executionTestRuntime(t, PaginationNone, PermissionPolicyPublic, provider, filters, func(config *ExecutionConfig) {
		config.MaxResultBytes = 1024
		config.Registry.WithPluginAdmission(func(candidate Artifact) bool { return candidate == artifact })
	})

	start := make(chan struct{})
	errorsCh := make(chan error, 1)
	var group sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			for iteration := 0; iteration < 25; iteration++ {
				result, err := runtime.Execute(context.Background(), PlanRequest{QueryID: "core.execute.items"})
				if err != nil || len(result.Rows) != 1 || result.Rows[0]["title"] != strings.Repeat("x", 64) {
					select {
					case errorsCh <- fmt.Errorf("iteration %d: result=%#v err=%v", iteration, result, err):
					default:
					}
					return
				}
			}
		}()
	}
	close(start)
	group.Wait()
	select {
	case err := <-errorsCh:
		t.Fatal(err)
	default:
	}
}
