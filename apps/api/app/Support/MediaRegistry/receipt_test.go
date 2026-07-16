package mediaregistry

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestOperationReceiptsEnforceExactOrderedPipeline(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	transform, found := stepByStage(plan, StageTransform)
	if !found {
		t.Fatal("transform step missing")
	}
	authority := newTestReceiptAuthority()
	prerequisites := sourcePrerequisitesForTest(t, authority, plan)
	if _, err := OperationForStep(t.Context(), authority, plan, transform.ID, 1, prerequisites); !errors.Is(err, ErrPredecessorRequired) {
		t.Fatalf("out-of-order transform construction = %v", err)
	}
	scan, _ := stepByStage(plan, StageScan)
	if _, err := OperationForStep(t.Context(), authority, plan, scan.ID, 1, OperationPrerequisites{}); !errors.Is(err, ErrReceiptInvalid) {
		t.Fatalf("background scan without source anchor = %v", err)
	}
	if _, err := BackgroundOperations(t.Context(), authority, plan, OperationPrerequisites{}); !errors.Is(err, ErrReceiptInvalid) {
		t.Fatalf("background scheduler without source anchor = %v", err)
	}
	if _, err := OperationForStep(t.Context(), nil, plan, scan.ID, 1, prerequisites); !errors.Is(err, ErrReceiptAuthority) {
		t.Fatalf("background scan without Host authority = %v", err)
	}
	otherRequest := uploadRequestForTest()
	otherRequest.Source.ID = "attachment-other"
	otherRequest.Source.Digest = strings.Repeat("e", 64)
	otherPlan, err := registry.Plan(t.Context(), otherRequest, allowAll())
	if err != nil {
		t.Fatal(err)
	}
	otherSource := sourcePrerequisitesForTest(t, authority, otherPlan)
	if _, err := OperationForStep(t.Context(), authority, plan, scan.ID, 1, otherSource); !errors.Is(err, ErrReceiptInvalid) {
		t.Fatalf("source evidence crossed exact plans = %v", err)
	}
	forged := BackgroundOperation{
		SchemaVersion: SchemaVersion, Key: operationKey(plan, transform), StepID: transform.ID,
		Attempt: 1, Plan: clonePlan(plan), Prerequisites: prerequisites,
	}
	called := false
	if _, err := NewExecutor(registry, newTestAdmission(transform.Processor.Artifact), invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		called = true
		return ProviderOutput{}, nil
	}), authority, nil).ExecuteOperation(t.Context(), forged, allowAll()); !errors.Is(err, ErrPredecessorRequired) || called {
		t.Fatalf("out-of-order transform executed: called=%t err=%v", called, err)
	}

	artifacts := make([]Artifact, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		artifacts = append(artifacts, step.Processor.Artifact)
	}
	admission := newTestAdmission(artifacts...)
	executor := NewExecutor(registry, admission, invokerFunc(func(_ context.Context, invocation Invocation) (ProviderOutput, error) {
		switch invocation.Step.Processor.Stage {
		case StageScan:
			return ProviderOutput{Decision: DecisionAllow}, nil
		case StageMetadata:
			return ProviderOutput{Metadata: map[string]string{"camera.model": "safe"}}, nil
		case StageTransform:
			return ProviderOutput{Variants: []VariantOutput{{
				Name: "thumbnail", Handle: "variant/ordered-thumbnail", Digest: strings.Repeat("b", 64),
				SourceDigest: plan.Source.Digest, MIME: "image/webp", SizeBytes: 512,
			}}}, nil
		default:
			return ProviderOutput{}, nil
		}
	}), authority, nil)

	for _, step := range plan.Steps {
		operation, operationErr := OperationForStep(t.Context(), authority, plan, step.ID, 1, prerequisites)
		if operationErr != nil {
			t.Fatal(operationErr)
		}
		result, executeErr := executor.ExecuteOperation(t.Context(), operation, allowAll())
		if executeErr != nil {
			t.Fatalf("execute %s: %v", step.ID, executeErr)
		}
		if result.Receipt.StepID != step.ID || result.Receipt.Digest == "" || result.Receipt.PlanDigest != plan.Digest {
			t.Fatalf("receipt for %s = %#v", step.ID, result.Receipt)
		}
		prerequisites.Steps = append(prerequisites.Steps, result.Receipt)
		if step.ID == transform.ID {
			break
		}
	}
	last := prerequisites.Steps[len(prerequisites.Steps)-1]
	if last.CumulativeUsage.MetadataBytes != len("camera.model")+len("safe") || last.CumulativeUsage.Variants != 1 {
		t.Fatalf("cumulative usage = %#v", last.CumulativeUsage)
	}

	encoded, err := json.Marshal(prerequisites)
	if err != nil {
		t.Fatal(err)
	}
	var durable OperationPrerequisites
	if err := json.Unmarshal(encoded, &durable); err != nil {
		t.Fatal(err)
	}
	// Remove the transform receipt itself: retry construction needs only its predecessors.
	durable.Steps = durable.Steps[:len(durable.Steps)-1]
	if _, err := OperationForStep(t.Context(), authority, plan, transform.ID, 2, durable); err != nil {
		t.Fatalf("durable receipt round trip: %v", err)
	}

	tampered := cloneOperationPrerequisites(durable)
	tampered.Steps[0].Artifact.RuntimeInstanceID = "other-runtime"
	// A recomputed public checksum still cannot forge Host-ledger evidence.
	tampered.Steps[0].Digest = receiptIntegrityDigest(operationReceiptClaim(tampered.Steps[0]))
	if _, err := OperationForStep(t.Context(), authority, plan, transform.ID, 1, tampered); !errors.Is(err, ErrReceiptInvalid) {
		t.Fatalf("tampered exact artifact receipt = %v", err)
	}
	tampered = cloneOperationPrerequisites(durable)
	tampered.Steps[0].CumulativeUsage.MetadataBytes++
	tampered.Steps[0].Digest = receiptIntegrityDigest(operationReceiptClaim(tampered.Steps[0]))
	if _, err := OperationForStep(t.Context(), authority, plan, transform.ID, 1, tampered); !errors.Is(err, ErrReceiptInvalid) {
		t.Fatalf("tampered cumulative receipt = %v", err)
	}
	tampered = cloneOperationPrerequisites(durable)
	tampered.Steps[0].Evidence.Seal += "-forged"
	if _, err := OperationForStep(t.Context(), authority, plan, transform.ID, 1, tampered); !errors.Is(err, ErrReceiptInvalid) {
		t.Fatalf("forged opaque evidence seal = %v", err)
	}
	authority.forget(durable.Source.Evidence)
	if _, err := OperationForStep(t.Context(), authority, plan, transform.ID, 1, durable); !errors.Is(err, ErrReceiptInvalid) {
		t.Fatalf("missing durable source evidence = %v", err)
	}
}

func TestOperationReceiptsEnforceCumulativeMetadataBudget(t *testing.T) {
	core, plugin := corePublicationForTest(), pluginPublicationForTest()
	plugin.Policies[0].Budget.MaxMetadataBytes = 15
	plugin.Processors = append(plugin.Processors, ProcessorDeclaration{
		ID: "demo.media.metadata_second", ContractVersion: "demo.media.metadata_second@1",
		Stage: StageMetadata, Purpose: "general", MIMEs: []string{"image/*"}, Handler: "metadata_second",
		Mode: ProcessorCompose, Execution: ExecutionSync, FailureMode: FailureSkip, RequiredPermission: "attachment.upload",
	})
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{core, plugin}, false); err != nil {
		t.Fatal(err)
	}
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	metadataSteps := []PlanStep{}
	for _, step := range plan.Steps {
		if step.Processor.Stage == StageMetadata {
			metadataSteps = append(metadataSteps, step)
		}
	}
	if len(metadataSteps) != 2 {
		t.Fatalf("metadata steps = %d", len(metadataSteps))
	}
	authority := newTestReceiptAuthority()
	admission := newTestAdmission(plugin.Artifact)
	call := 0
	executor := NewExecutor(registry, admission, invokerFunc(func(_ context.Context, invocation Invocation) (ProviderOutput, error) {
		if invocation.Step.Processor.Stage == StageScan {
			return ProviderOutput{Decision: DecisionAllow}, nil
		}
		call++
		if call == 2 && invocation.Budget.MaxMetadataBytes != 7 {
			t.Errorf("second metadata remaining budget = %d, want 7", invocation.Budget.MaxMetadataBytes)
		}
		return ProviderOutput{Metadata: map[string]string{"key": "12345"}}, nil
	}), authority, nil)
	prerequisites := sourcePrerequisitesForTest(t, authority, plan)
	for _, step := range plan.Steps {
		operation, operationErr := OperationForStep(t.Context(), authority, plan, step.ID, 1, prerequisites)
		if operationErr != nil {
			t.Fatal(operationErr)
		}
		result, executeErr := executor.ExecuteOperation(t.Context(), operation, allowAll())
		if step.ID == metadataSteps[1].ID {
			if !errors.Is(executeErr, ErrBudgetExceeded) || result.Receipt.Digest != "" || result.Skipped {
				t.Fatalf("cumulative metadata budget: result=%#v err=%v", result, executeErr)
			}
			break
		}
		if executeErr != nil {
			t.Fatalf("execute %s: %v", step.ID, executeErr)
		}
		prerequisites.Steps = append(prerequisites.Steps, result.Receipt)
	}
	if call != 2 {
		t.Fatalf("metadata calls = %d", call)
	}
}

func TestPlanEnforcesTotalVariantBudget(t *testing.T) {
	core, plugin := corePublicationForTest(), pluginPublicationForTest()
	plugin.Policies[0].Budget.MaxVariants = 1
	plugin.Variants = append(plugin.Variants, VariantDeclaration{
		ID: "demo.media.preview", ContractVersion: "demo.media.preview@1", Purpose: "general", Name: "preview",
		ProcessorID: "demo.media.transform", ProcessorContractVersion: "demo.media.transform@1",
		ProcessorOwnerExtensionID: "demo.media", ProcessorPackageDigest: plugin.Artifact.PackageDigest, OutputMIME: "image/webp",
	})
	registry := New()
	if _, err := registry.ReplaceAll([]Publication{core, plugin}, false); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll()); !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("total variant budget = %v", err)
	}
}

func TestLifecyclePlansDoNotReapplyCurrentUploadAdmission(t *testing.T) {
	core := corePublicationForTest()
	policy := &core.Policies[0]
	policy.AllowedMIMEs = []string{"text/plain"}
	policy.DeniedMIMEs = nil
	policy.AllowedExtensions = []string{"txt"}
	policy.RequireExpandedSize = true
	policy.Budget.MaxFileBytes = 1
	policy.Budget.MaxFilenameBytes = 1
	registry := New()
	if _, err := registry.Publish(core); err != nil {
		t.Fatal(err)
	}
	upload := uploadRequestForTest()
	if _, err := registry.Plan(t.Context(), upload, allowAll()); !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("new upload bypassed current admission = %v", err)
	}
	for _, kind := range []string{PlanDelivery, PlanRetention, PlanDelete} {
		t.Run(kind, func(t *testing.T) {
			request := upload
			request.Kind = kind
			request.Permission = "attachment.manage"
			request.Upload = UploadFacts{}
			authorizer := authorizerFunc(func(_ context.Context, input AuthorizationRequest) bool {
				return input.Permission != "attachment.upload"
			})
			plan, err := registry.Plan(t.Context(), request, authorizer)
			if err != nil {
				t.Fatalf("historical lifecycle plan: %v", err)
			}
			if plan.Upload.DeclaredMIME != request.Source.MIME || len(plan.Upload.DetectedMIMEs) != 1 || plan.Upload.DetectedMIMEs[0] != request.Source.MIME {
				t.Fatalf("lifecycle facts = %#v", plan.Upload)
			}
		})
	}

	plugin := pluginPublicationForTest()
	plugin.Policies = nil
	withoutUploadPolicy := New()
	if _, err := withoutUploadPolicy.Publish(plugin); err != nil {
		t.Fatal(err)
	}
	request := uploadRequestForTest()
	request.Kind = PlanDelete
	request.Permission = "attachment.manage"
	request.Source.Filename = ".legacy/archive.png"
	request.Source.SizeBytes = MaximumBudget().MaxFileBytes + 1
	request.Upload = UploadFacts{Archive: true, DecompressedBytes: -1}
	plan, err := withoutUploadPolicy.Plan(t.Context(), request, allowAll())
	if err != nil {
		t.Fatalf("lifecycle plan without current upload policy: %v", err)
	}
	if plan.Policy.ID != "core.media.lifecycle" || plan.Source.Filename != request.Source.Filename {
		t.Fatalf("Host lifecycle policy/source = %#v / %#v", plan.Policy, plan.Source)
	}
}

func TestMetadataNormalizationRejectsCaseCollisionsDeterministically(t *testing.T) {
	for iteration := 0; iteration < 100; iteration++ {
		if _, err := normalizeMetadata(map[string]string{"Camera.Model": "one", "camera.model": "two"}, DefaultBudget().MaxMetadataBytes); !errors.Is(err, ErrOutputRejected) {
			t.Fatalf("iteration %d collision = %v", iteration, err)
		}
	}
	processor := ProcessorContribution{ProcessorDeclaration: ProcessorDeclaration{
		Retry: RetryPolicy{MaxAttempts: 3, BaseDelaySeconds: 1, MaxDelaySeconds: 2},
	}}
	decision := ClassifyRetry(processor, errors.Join(ErrRuntimeUnavailable, ErrRuntimeQuarantined), 1)
	if decision.Retry || decision.Class != RetryPermanent {
		t.Fatalf("quarantined exact runtime retry = %#v", decision)
	}
}

func TestExpiredFinalAuthorityFenceNeverIssuesFallbackReceipt(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	transform, _ := stepByStage(plan, StageTransform)
	receipts := newTestReceiptAuthority()
	operation := operationForStepForTest(t, receipts, plan, transform.ID, 3)
	executor, err := NewExecutorWithLimits(
		registry,
		newTestAdmission(transform.Processor.Artifact),
		invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
			return ProviderOutput{Variants: []VariantOutput{{
				Name: "thumbnail", Handle: "variant/deadline", Digest: strings.Repeat("c", 64),
				SourceDigest: plan.Source.Digest, MIME: "image/webp", SizeBytes: 128,
			}}}, nil
		}),
		receipts,
		nil,
		ExecutionLimits{OperationTimeout: 20 * time.Millisecond, CallTimeout: 10 * time.Millisecond, MaxConcurrentCalls: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	var checks atomic.Int32
	authorizer := authorizerFunc(func(ctx context.Context, _ AuthorizationRequest) bool {
		if checks.Add(1) == 2 {
			<-ctx.Done()
		}
		return true
	})
	result, err := executor.ExecuteOperation(t.Context(), operation, authorizer)
	if !errors.Is(err, ErrExecutionTimeout) || result.FallbackOriginal || result.Skipped || result.Receipt.Digest != "" || result.Retry.Retry {
		t.Fatalf("expired final authority fence: checks=%d result=%#v err=%v", checks.Load(), result, err)
	}
}
