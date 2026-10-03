package contentregistry

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestExecutorExecuteBatchIsBoundedOrderedAndConcurrent(t *testing.T) {
	registry, target := executionRegistry(t, false, Declaration{
		ID: "batch.content.block.card", ContractVersion: "batch.content.block.card@1",
		Kind: KindBlock, Handler: "card", Schema: "batch.content.schema@1",
	})
	var active atomic.Int64
	var peak atomic.Int64
	renderer := RendererProviderFunc(func(_ context.Context, request RendererProviderRequest) (RenderSegments, error) {
		current := active.Add(1)
		defer active.Add(-1)
		for observed := peak.Load(); current > observed && !peak.CompareAndSwap(observed, current); observed = peak.Load() {
		}
		time.Sleep(5 * time.Millisecond)
		return executionRender(target, fmt.Sprintf("<p>%s</p>", request.ResourceID)), nil
	})
	binding := executionBinding(target, target.ID, ActionAdd, 0, ProviderSet{Renderer: renderer})
	binding.ContractVersion, binding.Artifact, binding.Fallback = target.ContractVersion, target.Artifact, FallbackClosed
	executor := newExecutionTestExecutor(t, registry, []ExecutionBinding{binding}, &executionTestAdmission{}, acceptingExecutionSchema, ExecutionLimits{
		MaxBatchSize: 4, MaxConcurrentCalls: 2, CallTimeout: time.Second,
	})
	requests := make([]ExecutionRequest, 4)
	for index := range requests {
		requests[index] = executionRequest(target, "actor")
		requests[index].ResourceID = fmt.Sprintf("topic:%d", index+1)
	}
	results, err := executor.ExecuteBatch(t.Context(), requests)
	if err != nil {
		t.Fatal(err)
	}
	if peak.Load() != 2 {
		t.Fatalf("batch peak concurrency=%d", peak.Load())
	}
	for index, result := range results {
		want := fmt.Sprintf("<p>topic:%d</p>", index+1)
		if result.Render.Segments[0].HTML != want {
			t.Fatalf("result[%d]=%#v want=%q", index, result.Render, want)
		}
	}
	if _, err := executor.ExecuteBatch(t.Context(), append(requests, requests[0])); !errors.Is(err, ErrExecutionInvalid) {
		t.Fatalf("oversized batch=%v", err)
	}
}
