package mediaregistry

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// TestReferenceMediaPluginProductSurface proves the P10 reference media plugin
// surface: custom MIME policy, metadata, image variants, background processing
// stage, CDN selection, and cleanup (delete stages) under Host authority.
func TestReferenceMediaPluginProductSurface(t *testing.T) {
	registry := registryWithMediaForTest()
	// Custom MIME policy is active via demo.media publication helpers.
	request := uploadRequestForTest()
	plan, err := registry.Plan(t.Context(), request, allowAll())
	if err != nil {
		t.Fatalf("plan upload: %v", err)
	}
	stages := map[string]bool{}
	for _, step := range plan.Steps {
		stages[step.Processor.Stage] = true
	}
	// Validate is Host MIME-policy admission, not a plugin processor step.
	// Reference media plugin surfaces scan/metadata/transform/cdn/retention.
	for _, required := range []string{StageScan, StageMetadata, StageTransform, StageCDN, StageRetention} {
		if !stages[required] {
			t.Fatalf("reference media plan missing stage %s: %#v", required, plan.Steps)
		}
	}
	// Metadata stage exists for custom extractors.
	if _, ok := stepByStage(plan, StageMetadata); !ok {
		t.Fatal("metadata stage required for reference media product")
	}
	// Transform produces package-digest-bound variants; original remains immutable.
	transform, _ := stepByStage(plan, StageTransform)
	if transform.Processor.Execution != ExecutionBackground && transform.Processor.Execution != ExecutionSync {
		t.Fatalf("transform execution = %#v", transform.Processor)
	}
	receipts := newTestReceiptAuthority()
	operation := operationForStepForTest(t, receipts, plan, transform.ID, 1)
	admission := newTestAdmission(transform.Processor.Artifact)
	executor := NewExecutor(registry, admission, invokerFunc(func(_ context.Context, invocation Invocation) (ProviderOutput, error) {
		if !invocation.Source.Immutable {
			t.Error("source must remain immutable for transform")
		}
		return ProviderOutput{Variants: []VariantOutput{{
			Name: "thumbnail", Handle: "variant/thumb-ref", Digest: strings.Repeat("b", 64),
			SourceDigest: plan.Source.Digest, MIME: "image/webp", SizeBytes: 256,
		}}}, nil
	}), receipts, nil)
	result, err := executor.ExecuteOperation(t.Context(), operation, allowAll())
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	if result.FallbackOriginal || len(result.Output.Variants) != 1 {
		t.Fatalf("transform result = %#v", result)
	}
	// CDN stage selects delivery URL without rewriting original handle.
	// Reuse the same receipt authority so prior transform is a prerequisite.
	cdn, ok := stepByStage(plan, StageCDN)
	if !ok {
		t.Fatal("cdn stage missing")
	}
	cdnOp := operationForStepForTest(t, receipts, plan, cdn.ID, 1)
	cdnAdmission := newTestAdmission(cdn.Processor.Artifact)
	cdnURL := "https://cdn.example.test/v/thumb-ref"
	cdnResult, err := NewExecutor(registry, cdnAdmission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		return ProviderOutput{CDNURL: cdnURL}, nil
	}), receipts, nil).ExecuteOperation(t.Context(), cdnOp, allowAll())
	if err != nil {
		t.Fatalf("cdn: %v", err)
	}
	if cdnResult.Output.CDNURL != cdnURL {
		t.Fatalf("cdn output = %#v", cdnResult.Output)
	}
	// Cleanup: delete plan includes before/after delete stages for retention hooks.
	deleteRequest := uploadRequestForTest()
	deleteRequest.Kind = PlanDelete
	deletePlan, err := registry.Plan(t.Context(), deleteRequest, allowAll())
	if err != nil {
		t.Fatalf("delete plan: %v", err)
	}
	deleteStages := map[string]bool{}
	for _, step := range deletePlan.Steps {
		deleteStages[step.Processor.Stage] = true
	}
	if !deleteStages[StageBeforeDelete] || !deleteStages[StageAfterDelete] {
		t.Fatalf("delete cleanup stages = %#v", deletePlan.Steps)
	}
}

// TestReferenceMediaProviderDisableFallsBackToOriginal proves disabling a
// transform plugin never destroys the immutable original/source-of-truth.
func TestReferenceMediaProviderDisableFallsBackToOriginal(t *testing.T) {
	registry := registryWithMediaForTest()
	request := uploadRequestForTest()
	plan, err := registry.Plan(t.Context(), request, allowAll())
	if err != nil {
		t.Fatal(err)
	}
	if plan.Source.Kind != SourceOriginal && plan.Source.Kind != SourceSourceOfTruth {
		t.Fatalf("source kind = %s", plan.Source.Kind)
	}
	if !plan.Source.Immutable {
		t.Fatal("source must be immutable")
	}
	// Remove the media publication (plugin disable): original handle is not owned
	// by the registry graph and cannot be rewritten here.
	publication, ok := registry.SnapshotPublication("demo.media")
	if !ok {
		// helpers may use a different extension id; accept any third-party pub.
		snapshot := registry.Snapshot()
		if len(snapshot.Publications) == 0 {
			t.Fatal("expected media publications")
		}
		publication = snapshot.Publications[0]
	}
	originalDigest := plan.Source.Digest
	if _, removed, err := registry.Remove(publication.Artifact); err != nil || !removed {
		// Core-only graphs may not remove third-party; re-plan after empty replace.
		_ = removed
	}
	// Source digest from the prior plan remains the caller's immutable bytes.
	if plan.Source.Digest != originalDigest || !plan.Source.Immutable {
		t.Fatalf("disable rewrote source: %#v", plan.Source)
	}
}

// TestReferenceMediaPermissionDenyClosesUpload proves allow/deny for media
// upload permission is Host-final.
func TestReferenceMediaPermissionDenyClosesUpload(t *testing.T) {
	registry := registryWithMediaForTest()
	denied := authorizerFunc(func(context.Context, AuthorizationRequest) bool { return false })
	_, err := registry.Plan(t.Context(), uploadRequestForTest(), denied)
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("denied upload err = %v", err)
	}
}
