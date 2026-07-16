package contentregistry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestExecutorConcurrentCallsRemainIsolated(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "raceexec.content.block.card", ContractVersion: "raceexec.content.block.card@1", Kind: KindBlock, Handler: "card", Schema: "raceexec.content.schema@1"},
	)
	binding := ExecutionBinding{
		TargetID: target.ID, TargetContractVersion: target.ContractVersion,
		DeclarationID: target.ID, ContractVersion: target.ContractVersion,
		Artifact: target.Artifact, Action: ActionAdd, Fallback: FallbackClosed,
		CacheTags: []string{"raceexec:card"},
		Providers: ProviderSet{Renderer: RendererProviderFunc(func(_ context.Context, request RendererProviderRequest) (RenderSegments, error) {
			return executionRender(request.Target, "<p>"+request.ResourceID+"</p>"), nil
		})},
	}
	trace := NewContentTraceRing(4096)
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, acceptingExecutionSchema,
		ExecutionLimits{MaxConcurrentCalls: 32}, WithContentTraceSink(trace))

	start := make(chan struct{})
	errorsCh := make(chan error, 1)
	var group sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		worker := worker
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			for index := 0; index < 50; index++ {
				request := executionRequest(target, fmt.Sprintf("actor-%d", worker))
				request.ResourceID = fmt.Sprintf("topic:%d:%d", worker, index)
				result, err := executor.Execute(t.Context(), request)
				if err != nil {
					select {
					case errorsCh <- err:
					default:
					}
					return
				}
				if result.Render.PlainText != request.ResourceID || !strings.Contains(result.CacheKey, "content:") ||
					!reflectStringSliceEqual(result.CacheTags, []string{"content:" + target.ID, "raceexec:card"}) {
					select {
					case errorsCh <- fmt.Errorf("cross-call result: %#v", result):
					default:
					}
					return
				}
				result.Render.Segments[0].HTML = "mutated"
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
	// Release fences are not provider calls and deliberately do not add trace
	// noise, so there is one renderer trace per execution.
	if len(trace.ContentTraces(0)) != 400 {
		t.Fatalf("trace count = %d", len(trace.ContentTraces(0)))
	}
}

func TestExecutorRegistrySwapDuringCallCannotReleaseOldResult(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "swapexec.content.block.card", ContractVersion: "swapexec.content.block.card@1", Kind: KindBlock, Handler: "card", Schema: "swapexec.content.schema@1"},
	)
	entered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	binding := ExecutionBinding{
		TargetID: target.ID, TargetContractVersion: target.ContractVersion,
		DeclarationID: target.ID, ContractVersion: target.ContractVersion,
		Artifact: target.Artifact, Action: ActionAdd, Fallback: FallbackClosed,
		Providers: ProviderSet{Renderer: RendererProviderFunc(func(_ context.Context, request RendererProviderRequest) (RenderSegments, error) {
			once.Do(func() { close(entered) })
			<-release
			return executionRender(request.Target, "<p>old</p>"), nil
		})},
	}
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, acceptingExecutionSchema,
		ExecutionLimits{CallTimeout: time.Second})
	type outcome struct {
		result ExecutionResult
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		result, err := executor.Execute(t.Context(), executionRequest(target, "actor"))
		done <- outcome{result: result, err: err}
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("provider was not entered")
	}
	if _, removed, err := registry.Remove(target.Artifact); err != nil || !removed {
		t.Fatalf("remove during call removed=%t error=%v", removed, err)
	}
	close(release)
	select {
	case result := <-done:
		if !errors.Is(result.err, ErrContractStale) || result.result.SchemaVersion != "" {
			t.Fatalf("stale result = %#v error=%v", result.result, result.err)
		}
	case <-time.After(time.Second):
		t.Fatal("stale execution did not finish")
	}
}

func TestExecutorPermissionRevocationAtReleaseWinsRace(t *testing.T) {
	registry, target := executionRegistry(t, false,
		Declaration{ID: "permissionrace.content.block.card", ContractVersion: "permissionrace.content.block.card@1", Kind: KindBlock, Handler: "card", Schema: "permissionrace.content.schema@1"},
	)
	binding := ExecutionBinding{
		TargetID: target.ID, TargetContractVersion: target.ContractVersion,
		DeclarationID: target.ID, ContractVersion: target.ContractVersion,
		Artifact: target.Artifact, Action: ActionAdd, Fallback: FallbackClosed,
		Providers: ProviderSet{Renderer: staticExecutionRenderer("private")},
	}
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, acceptingExecutionSchema, ExecutionLimits{})
	request := executionRequest(target, "actor")
	request.Permission.Recheck = PermissionRecheckFunc(func(_ context.Context, claim PermissionClaim) error {
		if claim.Operation == OperationRelease {
			return ErrExecutionDenied
		}
		return nil
	})
	if _, err := executor.Execute(t.Context(), request); !errors.Is(err, ErrExecutionDenied) {
		t.Fatalf("release permission revocation = %v", err)
	}
}

func reflectStringSliceEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
