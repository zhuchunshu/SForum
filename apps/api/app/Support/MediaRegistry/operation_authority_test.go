package mediaregistry

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestDuplicateOperationSequentiallyReplaysTerminalResult(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	scan, _ := stepByStage(plan, StageScan)
	receipts := newTestReceiptAuthority()
	operation := operationForStepForTest(t, receipts, plan, scan.ID, 1)
	admission := newTestAdmission(scan.Processor.Artifact)
	var calls atomic.Int32
	executor := NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		calls.Add(1)
		return ProviderOutput{Decision: DecisionAllow}, nil
	}), receipts, nil)

	first, err := executor.ExecuteOperation(t.Context(), operation, allowAll())
	if err != nil || first.Replayed || first.Receipt.Evidence.ID == "" {
		t.Fatalf("first execution: result=%#v err=%v", first, err)
	}
	second, err := executor.ExecuteOperation(t.Context(), operation, allowAll())
	if err != nil || !second.Replayed || calls.Load() != 1 || second.Receipt != first.Receipt ||
		providerOutputDigest(second.Output) != providerOutputDigest(first.Output) {
		t.Fatalf("sequential duplicate: first=%#v second=%#v calls=%d err=%v", first, second, calls.Load(), err)
	}
}

func TestDuplicateFallbackAndSkipReplayWithoutProvider(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		stage    string
		attempt  int
		fallback bool
		skipped  bool
	}{
		{stage: StageTransform, attempt: 3, fallback: true},
		{stage: StageMetadata, attempt: 1, skipped: true},
	}
	for _, test := range tests {
		t.Run(test.stage, func(t *testing.T) {
			step, _ := stepByStage(plan, test.stage)
			receipts := newTestReceiptAuthority()
			operation := operationForStepForTest(t, receipts, plan, step.ID, test.attempt)
			executor := NewExecutor(registry, newTestAdmission(), invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
				t.Fatal("unavailable provider was invoked")
				return ProviderOutput{}, nil
			}), receipts, nil)
			first, err := executor.ExecuteOperation(t.Context(), operation, allowAll())
			if err != nil || first.FallbackOriginal != test.fallback || first.Skipped != test.skipped || first.Replayed {
				t.Fatalf("first terminal: result=%#v err=%v", first, err)
			}
			second, err := executor.ExecuteOperation(t.Context(), operation, allowAll())
			if err != nil || !second.Replayed || second.FallbackOriginal != test.fallback || second.Skipped != test.skipped || second.Receipt != first.Receipt {
				t.Fatalf("terminal replay: first=%#v second=%#v err=%v", first, second, err)
			}
			if test.stage == StageTransform {
				earlier, err := OperationForStep(t.Context(), receipts, plan, step.ID, 1, operation.Prerequisites)
				if err != nil {
					t.Fatal(err)
				}
				replayed, err := executor.ExecuteOperation(t.Context(), earlier, allowAll())
				if err != nil || !replayed.Replayed || !replayed.FallbackOriginal || replayed.Receipt != first.Receipt {
					t.Fatalf("cross-attempt redelivery: result=%#v err=%v", replayed, err)
				}
			}
		})
	}
}

func TestConcurrentDuplicateOperationHasOneInvokerAndSameAttemptBusyRetry(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	scan, _ := stepByStage(plan, StageScan)
	receipts := newTestReceiptAuthority()
	operation := operationForStepForTest(t, receipts, plan, scan.ID, 1)
	admission := newTestAdmission(scan.Processor.Artifact)
	started := make(chan struct{})
	unblock := make(chan struct{})
	var calls atomic.Int32
	executor := NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		if calls.Add(1) == 1 {
			close(started)
		}
		<-unblock
		return ProviderOutput{Decision: DecisionAllow}, nil
	}), receipts, nil)
	type execution struct {
		result ExecutionResult
		err    error
	}
	firstDone := make(chan execution, 1)
	go func() {
		result, executeErr := executor.ExecuteOperation(context.Background(), operation, allowAll())
		firstDone <- execution{result: result, err: executeErr}
	}()
	<-started

	busy, err := executor.ExecuteOperation(t.Context(), operation, allowAll())
	if !errors.Is(err, ErrOperationBusy) || !busy.Retry.Retry || busy.Retry.NextAttempt != operation.Attempt || calls.Load() != 1 {
		t.Fatalf("concurrent duplicate: result=%#v calls=%d err=%v", busy, calls.Load(), err)
	}
	close(unblock)
	first := <-firstDone
	if first.err != nil || first.result.Receipt.Evidence.ID == "" {
		t.Fatalf("claimed execution: result=%#v err=%v", first.result, first.err)
	}
	replayed, err := executor.ExecuteOperation(t.Context(), operation, allowAll())
	if err != nil || !replayed.Replayed || replayed.Receipt != first.result.Receipt || calls.Load() != 1 {
		t.Fatalf("post-completion duplicate: result=%#v calls=%d err=%v", replayed, calls.Load(), err)
	}
}

func TestRevokedOrTamperedTerminalNeverReopensProvider(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	scan, _ := stepByStage(plan, StageScan)
	for _, test := range []struct {
		name   string
		mutate func(*testReceiptAuthority, OperationCompletion)
	}{
		{name: "revoked evidence", mutate: func(authority *testReceiptAuthority, completion OperationCompletion) {
			authority.forget(completion.Receipt.Evidence)
		}},
		{name: "tampered output", mutate: func(authority *testReceiptAuthority, _ OperationCompletion) {
			authority.mu.Lock()
			authority.operations[operationKey(plan, scan)].completion.Output.Decision = DecisionReject
			authority.mu.Unlock()
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			receipts := newTestReceiptAuthority()
			operation := operationForStepForTest(t, receipts, plan, scan.ID, 1)
			admission := newTestAdmission(scan.Processor.Artifact)
			var calls atomic.Int32
			executor := NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
				calls.Add(1)
				return ProviderOutput{Decision: DecisionAllow}, nil
			}), receipts, nil)
			first, err := executor.ExecuteOperation(t.Context(), operation, allowAll())
			if err != nil {
				t.Fatal(err)
			}
			test.mutate(receipts, *receipts.operationCompletion(operation.Key))
			if result, err := executor.ExecuteOperation(t.Context(), operation, allowAll()); !errors.Is(err, ErrReceiptInvalid) || calls.Load() != 1 || !result.Replayed {
				t.Fatalf("invalid terminal reopened provider: result=%#v calls=%d err=%v first=%#v", result, calls.Load(), err, first)
			}
		})
	}
}

func TestRuntimeReleaseFailureCannotPublishTerminalReceipt(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	scan, _ := stepByStage(plan, StageScan)
	receipts := newTestReceiptAuthority()
	operation := operationForStepForTest(t, receipts, plan, scan.ID, 1)
	admission := newTestAdmission(scan.Processor.Artifact)
	admission.panicRelease = true
	result, err := NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		return ProviderOutput{Decision: DecisionAllow}, nil
	}), receipts, nil).ExecuteOperation(t.Context(), operation, allowAll())
	if !errors.Is(err, ErrRuntimeQuarantined) || result.Receipt.Evidence.ID != "" || receipts.operationCompletion(operation.Key) != nil {
		t.Fatalf("release failure published terminal: result=%#v err=%v", result, err)
	}
}

func TestPostReleaseAuthorityLossCannotPublishVerifiableTerminal(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(context.CancelFunc, *testReceiptAuthority, BackgroundOperation)
		target error
	}{
		{
			name: "caller context",
			mutate: func(cancel context.CancelFunc, _ *testReceiptAuthority, _ BackgroundOperation) {
				cancel()
			},
			target: context.Canceled,
		},
		{
			name: "operation claim",
			mutate: func(_ context.CancelFunc, receipts *testReceiptAuthority, operation BackgroundOperation) {
				receipts.cancelOperation(operation.Key)
			},
			target: ErrReceiptInvalid,
		},
		{
			name: "predecessor evidence",
			mutate: func(_ context.CancelFunc, receipts *testReceiptAuthority, operation BackgroundOperation) {
				receipts.forget(operation.Prerequisites.Source.Evidence)
			},
			target: ErrReceiptInvalid,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry := registryWithMediaForTest()
			plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
			if err != nil {
				t.Fatal(err)
			}
			scan, _ := stepByStage(plan, StageScan)
			receipts := newTestReceiptAuthority()
			operation := operationForStepForTest(t, receipts, plan, scan.ID, 1)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			admission := newTestAdmission(scan.Processor.Artifact)
			admission.onRelease = func() { test.mutate(cancel, receipts, operation) }

			result, executeErr := NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
				return ProviderOutput{Decision: DecisionAllow}, nil
			}), receipts, nil).ExecuteOperation(ctx, operation, allowAll())
			if !errors.Is(executeErr, test.target) || result.Receipt.Evidence.ID != "" || receipts.operationCompletion(operation.Key) != nil {
				t.Fatalf("post-release authority loss published terminal: result=%#v err=%v", result, executeErr)
			}
			claim := operationReceiptClaim(OperationReceipt{
				SchemaVersion: OperationReceiptSchemaVersion,
				PlanDigest:    plan.Digest, RegistryDigest: plan.RegistryDigest, PlanKind: plan.Kind,
				SourceDigest: plan.Source.Digest, OperationKey: operation.Key, StepID: scan.ID,
				Stage: scan.Processor.Stage, Artifact: receiptArtifact(scan.Processor.Artifact), Attempt: 1,
				Outcome: TraceSucceeded, OutputDigest: providerOutputDigest(ProviderOutput{Decision: DecisionAllow}),
			})
			claim.Digest = receiptIntegrityDigest(claim)
			if verifyErr := receipts.VerifyMediaReceipt(context.Background(), claim, ReceiptEvidence{ID: "missing", Seal: "missing"}); verifyErr == nil {
				t.Fatal("post-release failure left verifiable terminal evidence")
			}
		})
	}
}

func TestDirectRecordCannotWriteOperationCompletion(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	scan, _ := stepByStage(plan, StageScan)
	receipts := newTestReceiptAuthority()
	operation := operationForStepForTest(t, receipts, plan, scan.ID, 1)
	receipt, err := operationReceiptTemplate(t.Context(), receipts, operation, scan, ProviderOutput{Decision: DecisionAllow}, TraceSucceeded)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := recordReceiptEvidence(t.Context(), receipts, operationReceiptClaim(receipt)); !errors.Is(err, ErrReceiptInvalid) {
		t.Fatalf("direct operation record = %v", err)
	}
	if calls := receipts.directRecordCalls(); calls != 1 {
		// operationForStepForTest recorded only the source admission. The rejected
		// operation claim must not cross the generic authority method boundary.
		t.Fatalf("direct writer calls = %d, want source-only call", calls)
	}
	if completion := receipts.operationCompletion(operation.Key); completion != nil {
		t.Fatalf("direct operation record created completion: %#v", completion)
	}
}

func TestPanickingLeaseIdentityBoundariesFailClosedWithoutLeaks(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*testReceiptAuthority, *testAdmission)
		target error
	}{
		{
			name: "receipt context",
			setup: func(receipts *testReceiptAuthority, _ *testAdmission) {
				receipts.panicReceiptLeaseContext = true
			},
			target: ErrReceiptInvalid,
		},
		{
			name: "operation context",
			setup: func(receipts *testReceiptAuthority, _ *testAdmission) {
				receipts.panicOperationLeaseContext = true
			},
			target: ErrReceiptInvalid,
		},
		{
			name: "runtime context",
			setup: func(_ *testReceiptAuthority, admission *testAdmission) {
				admission.panicLeaseContext = true
			},
			target: ErrRuntimeQuarantined,
		},
		{
			name: "runtime artifact",
			setup: func(_ *testReceiptAuthority, admission *testAdmission) {
				admission.panicLeaseArtifact = true
			},
			target: ErrRuntimeQuarantined,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry := registryWithMediaForTest()
			plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
			if err != nil {
				t.Fatal(err)
			}
			scan, _ := stepByStage(plan, StageScan)
			receipts := newTestReceiptAuthority()
			operation := operationForStepForTest(t, receipts, plan, scan.ID, 1)
			admission := newTestAdmission(scan.Processor.Artifact)
			test.setup(receipts, admission)
			var invoked atomic.Int32
			result, executeErr := NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
				invoked.Add(1)
				return ProviderOutput{Decision: DecisionAllow}, nil
			}), receipts, nil).ExecuteOperation(t.Context(), operation, allowAll())
			if !errors.Is(executeErr, test.target) || result.Receipt.Evidence.ID != "" || receipts.operationCompletion(operation.Key) != nil {
				t.Fatalf("panicking lease boundary: result=%#v err=%v", result, executeErr)
			}
			active, released := admission.counts()
			if active != 0 || (test.name == "runtime context" || test.name == "runtime artifact") && released != 1 {
				t.Fatalf("lease leaked: active=%d released=%d invoked=%d", active, released, invoked.Load())
			}
			if receipts.activeReceiptLeases() != 0 {
				t.Fatal("receipt lease leaked after context panic")
			}
		})
	}
}

type cancelPanickingAdmission struct{ cancel context.CancelFunc }

func (admission cancelPanickingAdmission) Available(Artifact) bool {
	admission.cancel()
	panic("test admission panic after cancellation")
}

func (cancelPanickingAdmission) Acquire(context.Context, Artifact) (RuntimeLease, error) {
	panic("unreachable test acquire")
}

func TestAdmissionPanicDoesNotHideCallerCancellation(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	scan, _ := stepByStage(plan, StageScan)
	receipts := newTestReceiptAuthority()
	operation := operationForStepForTest(t, receipts, plan, scan.ID, 1)
	ctx, cancel := context.WithCancel(t.Context())
	result, executeErr := NewExecutor(registry, cancelPanickingAdmission{cancel: cancel}, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		t.Fatal("panicking admission invoked provider")
		return ProviderOutput{}, nil
	}), receipts, nil).ExecuteOperation(ctx, operation, allowAll())
	if !errors.Is(executeErr, context.Canceled) || !errors.Is(executeErr, ErrRuntimeQuarantined) ||
		result.Receipt.Evidence.ID != "" || receipts.operationCompletion(operation.Key) != nil {
		t.Fatalf("admission panic hid cancellation: result=%#v err=%v", result, executeErr)
	}
}

func TestProviderFailureWithRuntimeReleasePanicCannotPublishFallback(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	transform, _ := stepByStage(plan, StageTransform)
	receipts := newTestReceiptAuthority()
	operation := operationForStepForTest(t, receipts, plan, transform.ID, 3)
	admission := newTestAdmission(transform.Processor.Artifact)
	admission.panicRelease = true
	providerErr := NewProviderError(RetryCrash, "transform.release", errors.New("provider failed"))
	result, err := NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		return ProviderOutput{}, providerErr
	}), receipts, nil).ExecuteOperation(t.Context(), operation, allowAll())
	if !errors.Is(err, ErrRuntimeLeaseRelease) || result.FallbackOriginal || result.Skipped ||
		result.Receipt.Evidence.ID != "" || receipts.operationCompletion(operation.Key) != nil {
		t.Fatalf("release failure published fallback: result=%#v err=%v", result, err)
	}
}

func TestPostReleasePlanDriftCannotPublishTerminalReceipt(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	scan, _ := stepByStage(plan, StageScan)
	receipts := newTestReceiptAuthority()
	operation := operationForStepForTest(t, receipts, plan, scan.ID, 1)
	admission := newTestAdmission(scan.Processor.Artifact)
	var removeErr error
	admission.onRelease = func() {
		_, _, removeErr = registry.Remove(pluginPublicationForTest().Artifact)
	}
	result, err := NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		return ProviderOutput{Decision: DecisionAllow}, nil
	}), receipts, nil).ExecuteOperation(t.Context(), operation, allowAll())
	if removeErr != nil || !errors.Is(err, ErrPlanStale) || result.Receipt.Evidence.ID != "" || receipts.operationCompletion(operation.Key) != nil {
		t.Fatalf("plan drift published terminal: result=%#v removeErr=%v err=%v", result, removeErr, err)
	}
}

func TestFallbackFinalPermissionDenialCannotPublishTerminalReceipt(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	transform, _ := stepByStage(plan, StageTransform)
	receipts := newTestReceiptAuthority()
	operation := operationForStepForTest(t, receipts, plan, transform.ID, 3)
	var checks atomic.Int32
	authorizer := authorizerFunc(func(context.Context, AuthorizationRequest) bool { return checks.Add(1) == 1 })
	result, err := NewExecutor(registry, newTestAdmission(), invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		t.Fatal("unavailable provider was invoked")
		return ProviderOutput{}, nil
	}), receipts, nil).ExecuteOperation(t.Context(), operation, authorizer)
	if !errors.Is(err, ErrPermissionDenied) || result.FallbackOriginal || result.Skipped ||
		result.Receipt.Evidence.ID != "" || receipts.operationCompletion(operation.Key) != nil {
		t.Fatalf("denied fallback published terminal: result=%#v checks=%d err=%v", result, checks.Load(), err)
	}
}

func TestPrerequisiteRevocationAtCommitCannotPublishTerminalReceipt(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	scan, _ := stepByStage(plan, StageScan)
	receipts := newTestReceiptAuthority()
	operation := operationForStepForTest(t, receipts, plan, scan.ID, 1)
	receipts.beforeCommit = func() {
		receipts.beforeCommit = nil
		receipts.forget(operation.Prerequisites.Source.Evidence)
	}
	var calls atomic.Int32
	result, err := NewExecutor(registry, newTestAdmission(scan.Processor.Artifact), invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		calls.Add(1)
		return ProviderOutput{Decision: DecisionAllow}, nil
	}), receipts, nil).ExecuteOperation(t.Context(), operation, allowAll())
	if !errors.Is(err, ErrReceiptInvalid) || result.Receipt.Evidence.ID != "" ||
		receipts.operationCompletion(operation.Key) != nil || calls.Load() != 1 {
		t.Fatalf("revoked prerequisite published terminal: result=%#v calls=%d err=%v", result, calls.Load(), err)
	}
}

func TestRetentionTerminalReplayDoesNotReapplyTimeRelativeAdmission(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	retention, _ := stepByStage(plan, StageRetention)
	receipts := newTestReceiptAuthority()
	operation := operationForStepForTest(t, receipts, plan, retention.ID, 1)
	receiptLease, err := acquireReceiptLease(t.Context(), receipts, operationReceiptBindings(retention, operation.Prerequisites))
	if err != nil {
		t.Fatal(err)
	}
	acquisition, err := acquireOperationClaim(t.Context(), receipts, operationClaim(operation, retention))
	if err != nil {
		t.Fatal(err)
	}
	past := time.Now().UTC().Add(-2 * time.Minute)
	completion, err := commitOperationCompletion(t.Context(), receipts, acquisition.Lease, receiptLease, operation, retention,
		ProviderOutput{RetainUntil: past}, TraceSucceeded, false, false)
	if err != nil {
		t.Fatal(err)
	}
	releaseOperationLease(acquisition.Lease)
	releaseReceiptLease(receiptLease)

	result, err := NewExecutor(registry, newTestAdmission(retention.Processor.Artifact), invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		t.Fatal("aged terminal replay invoked provider")
		return ProviderOutput{}, nil
	}), receipts, nil).ExecuteOperation(t.Context(), operation, allowAll())
	if err != nil || !result.Replayed || result.Receipt != completion.Receipt || !result.Output.RetainUntil.Equal(past) {
		t.Fatalf("aged retention replay: result=%#v err=%v", result, err)
	}
}
