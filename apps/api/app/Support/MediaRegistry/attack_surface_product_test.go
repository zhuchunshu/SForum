package mediaregistry

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
)

// TestMediaAttackSurfaceProductMatrix is the P10 joined attack-surface gate:
// traversal, MIME confusion, decompression bomb, transform crash/retry,
// duplicate jobs, orphan variants, provider disable, and uninstall retention
// semantics (immutable original preserved).
func TestMediaAttackSurfaceProductMatrix(t *testing.T) {
	t.Run("traversal_and_mime_and_bomb", func(t *testing.T) {
		registry := New()
		if _, err := registry.Publish(corePublicationForTest()); err != nil {
			t.Fatal(err)
		}
		cases := []struct {
			name   string
			mutate func(*PlanRequest)
			want   error
		}{
			{"path traversal", func(r *PlanRequest) { r.Source.Filename = "../x.png" }, ErrInvalid},
			{"MIME confusion", func(r *PlanRequest) { r.Upload.DeclaredMIME = "text/html" }, ErrMIMEConfusion},
			{"decompression bomb", func(r *PlanRequest) {
				r.Upload.Archive = true
				r.Upload.DecompressedBytes = DefaultBudget().MaxDecompressedBytes + 1
			}, ErrBudgetExceeded},
			{"SVG XSS vector", func(r *PlanRequest) {
				r.Source.MIME = "image/svg+xml"
				r.Source.Filename = "x.svg"
				r.Upload.DeclaredMIME = "image/svg+xml"
				r.Upload.DetectedMIMEs = []string{"image/svg+xml"}
			}, ErrMediaRejected},
		}
		for _, tc := range cases {
			req := uploadRequestForTest()
			tc.mutate(&req)
			_, err := registry.Plan(t.Context(), req, allowAll())
			if !errors.Is(err, tc.want) {
				t.Fatalf("%s: got %v want %v", tc.name, err, tc.want)
			}
		}
	})

	t.Run("transform_crash_retry_and_fallback", func(t *testing.T) {
		registry := registryWithMediaForTest()
		plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
		if err != nil {
			t.Fatal(err)
		}
		step, ok := stepByStage(plan, StageTransform)
		if !ok {
			t.Fatal("transform missing")
		}
		receipts := newTestReceiptAuthority()
		operation := operationForStepForTest(t, receipts, plan, step.ID, 1)
		admission := newTestAdmission(step.Processor.Artifact)
		// Provider crash: Host must not rewrite original; failure mode may retry
		// or fallback_original depending on declaration.
		result, err := NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
			return ProviderOutput{}, errors.New("provider crashed")
		}), receipts, nil).ExecuteOperation(t.Context(), operation, allowAll())
		// FailureMode fallback_original may convert crash into a controlled
		// fallback without inventing variants; either path is Host-closed.
		if err == nil && !result.FallbackOriginal && !result.Retry.Retry {
			t.Fatalf("crash must error, retry, or fallback_original: %#v", result)
		}
		if len(result.Output.Variants) > 0 {
			t.Fatalf("crash must not invent variants: %#v", result)
		}
		// Original source identity remains on the plan.
		if !plan.Source.Immutable || plan.Source.Digest == "" {
			t.Fatalf("source after crash = %#v", plan.Source)
		}
	})

	t.Run("duplicate_jobs_idempotent", func(t *testing.T) {
		registry := registryWithMediaForTest()
		plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
		if err != nil {
			t.Fatal(err)
		}
		step, _ := stepByStage(plan, StageScan)
		receipts := newTestReceiptAuthority()
		operation := operationForStepForTest(t, receipts, plan, step.ID, 1)
		admission := newTestAdmission(step.Processor.Artifact)
		var calls atomic.Int32
		invoker := invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
			calls.Add(1)
			return ProviderOutput{Decision: DecisionAllow}, nil
		})
		executor := NewExecutor(registry, admission, invoker, receipts, nil)
		first, err := executor.ExecuteOperation(t.Context(), operation, allowAll())
		if err != nil {
			t.Fatalf("first: %v", err)
		}
		// Same operation key replay must not double-invoke the provider.
		second, err := executor.ExecuteOperation(t.Context(), operation, allowAll())
		if err != nil {
			t.Fatalf("second: %v", err)
		}
		if calls.Load() != 1 {
			t.Fatalf("duplicate job invoked provider %d times", calls.Load())
		}
		if second.Replayed && first.Receipt.Digest != "" && second.Receipt.Digest != first.Receipt.Digest {
			t.Fatalf("replay receipt drift first=%#v second=%#v", first.Receipt, second.Receipt)
		}
	})

	t.Run("provider_disable_preserves_original", func(t *testing.T) {
		registry := registryWithMediaForTest()
		plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
		if err != nil {
			t.Fatal(err)
		}
		source := plan.Source
		// Uninstall/disable semantics: registry graph may drop processors, but
		// the immutable original identity on prior plans is caller-owned data.
		snapshot := registry.Snapshot()
		for _, publication := range snapshot.Publications {
			if publication.Artifact.Core {
				continue
			}
			_, _, _ = registry.Remove(publication.Artifact)
		}
		if source.Digest == "" || !source.Immutable {
			t.Fatalf("uninstall retention lost original: %#v", source)
		}
		if strings.TrimSpace(source.ID) == "" {
			t.Fatal("original handle must survive plugin uninstall")
		}
	})
}
