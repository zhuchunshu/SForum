package mediaregistry

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestExecutionHoldsExactLeaseAndValidatesTransformOutput(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	step, found := stepByStage(plan, StageTransform)
	if !found {
		t.Fatal("transform step missing")
	}
	receipts := newTestReceiptAuthority()
	operation := operationForStepForTest(t, receipts, plan, step.ID, 1)
	admission := newTestAdmission(step.Processor.Artifact)
	invoker := invokerFunc(func(_ context.Context, invocation Invocation) (ProviderOutput, error) {
		active, _ := admission.counts()
		if active != 1 {
			t.Errorf("provider ran without held lease: %d", active)
		}
		if invocation.Source != providerSource(plan.Source) || !invocation.Source.Immutable || invocation.Step.Processor.Artifact != step.Processor.Artifact {
			t.Errorf("invocation lost exact immutable input: %#v", invocation)
		}
		return ProviderOutput{Variants: []VariantOutput{{Name: "thumbnail", Handle: "variant/thumbnail-42", Digest: strings.Repeat("b", 64), SourceDigest: plan.Source.Digest, MIME: "image/webp", SizeBytes: 512}}}, nil
	})
	executor := NewExecutor(registry, admission, invoker, receipts, nil)
	result, err := executor.ExecuteOperation(t.Context(), operation, allowAll())
	if err != nil {
		t.Fatal(err)
	}
	if result.FallbackOriginal || len(result.Output.Variants) != 1 || result.Output.Variants[0].SourceDigest != plan.Source.Digest {
		t.Fatalf("unexpected transform result: %#v", result)
	}
	active, released := admission.counts()
	if active != 0 || released != 1 {
		t.Fatalf("lease counts active=%d released=%d", active, released)
	}

	badInvoker := invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		return ProviderOutput{Variants: []VariantOutput{{Name: "thumbnail", Handle: plan.Source.ID, Digest: strings.Repeat("c", 64), SourceDigest: strings.Repeat("d", 64), MIME: "image/webp", SizeBytes: 1}}}, nil
	})
	badReceipts := newTestReceiptAuthority()
	badOperation := operationForStepForTest(t, badReceipts, plan, step.ID, 1)
	result, err = NewExecutor(registry, admission, badInvoker, badReceipts, nil).ExecuteOperation(t.Context(), badOperation, allowAll())
	if !errors.Is(err, ErrOutputRejected) || result.FallbackOriginal {
		t.Fatalf("original mutation was not fail closed: result=%#v err=%v", result, err)
	}
}

func TestExactLeaseMismatchAndPermissionRevocationFailClosed(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	step, _ := stepByStage(plan, StageScan)
	receipts := newTestReceiptAuthority()
	operation := operationForStepForTest(t, receipts, plan, step.ID, 1)
	admission := newTestAdmission(step.Processor.Artifact)
	admission.leaseAs = pluginArtifactForTest("other.media", '8')
	called := false
	executor := NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		called = true
		return ProviderOutput{Decision: DecisionAllow}, nil
	}), receipts, nil)
	result, err := executor.ExecuteOperation(t.Context(), operation, allowAll())
	if !errors.Is(err, ErrRuntimeUnavailable) || called || !result.Retry.Retry {
		t.Fatalf("wrong exact lease accepted: result=%#v err=%v", result, err)
	}
	active, released := admission.counts()
	if active != 0 || released != 1 {
		t.Fatalf("mismatched lease leak active=%d released=%d", active, released)
	}

	admission = newTestAdmission(step.Processor.Artifact)
	var permitted atomic.Bool
	permitted.Store(true)
	authorizer := authorizerFunc(func(context.Context, AuthorizationRequest) bool { return permitted.Load() })
	traces := NewTraceRing(8)
	executor = NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		permitted.Store(false)
		return ProviderOutput{Decision: DecisionAllow}, nil
	}), receipts, traces)
	result, err = executor.ExecuteOperation(t.Context(), operation, authorizer)
	if !errors.Is(err, ErrPermissionDenied) || result.FallbackOriginal || result.Retry.Retry {
		t.Fatalf("revoked permission did not fail closed: result=%#v err=%v", result, err)
	}
	records := traces.MediaTraces(10)
	if len(records) != 1 || records[0].Outcome != TraceDenied || records[0].Reason != "permission_denied" {
		t.Fatalf("post-invoke permission denial lost denied trace: %#v", records)
	}
}

func TestFinalReleaseFencesPermissionAndExactRuntimeAfterOutputValidation(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	step, _ := stepByStage(plan, StageScan)
	output := ProviderOutput{Decision: DecisionAllow}

	t.Run("permission", func(t *testing.T) {
		receipts := newTestReceiptAuthority()
		operation := operationForStepForTest(t, receipts, plan, step.ID, 1)
		admission := newTestAdmission(step.Processor.Artifact)
		var checks atomic.Int32
		authorizer := authorizerFunc(func(context.Context, AuthorizationRequest) bool {
			return checks.Add(1) < 3
		})
		result, err := NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
			return output, nil
		}), receipts, nil).ExecuteOperation(t.Context(), operation, authorizer)
		if !errors.Is(err, ErrPermissionDenied) || result.Output.Decision != "" || checks.Load() != 3 {
			t.Fatalf("final permission fence: result=%#v checks=%d err=%v", result, checks.Load(), err)
		}
		if active, released := admission.counts(); active != 0 || released != 1 {
			t.Fatalf("permission fence lease active=%d released=%d", active, released)
		}
	})

	t.Run("runtime", func(t *testing.T) {
		receipts := newTestReceiptAuthority()
		operation := operationForStepForTest(t, receipts, plan, step.ID, 1)
		admission := newTestAdmission(step.Processor.Artifact)
		var checks atomic.Int32
		authorizer := authorizerFunc(func(context.Context, AuthorizationRequest) bool {
			if checks.Add(1) == 3 {
				admission.mu.Lock()
				admission.available[step.Processor.Artifact] = false
				admission.mu.Unlock()
			}
			return true
		})
		result, err := NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
			return output, nil
		}), receipts, nil).ExecuteOperation(t.Context(), operation, authorizer)
		if !errors.Is(err, ErrRuntimeUnavailable) || result.Output.Decision != "" || checks.Load() != 3 {
			t.Fatalf("final runtime fence: result=%#v checks=%d err=%v", result, checks.Load(), err)
		}
	})
}

func TestInvalidExecutorConfigurationReturnsErrInvalid(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	step, _ := stepByStage(plan, StageScan)
	receipts := newTestReceiptAuthority()
	operation := operationForStepForTest(t, receipts, plan, step.ID, 1)
	invoker := invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		t.Fatal("invalid executor must not invoke")
		return ProviderOutput{}, nil
	})
	cases := []struct {
		name     string
		executor *Executor
		ctx      context.Context
		auth     Authorizer
	}{
		{"nil executor", nil, t.Context(), allowAll()},
		{"nil registry", NewExecutor(nil, nil, invoker, receipts, nil), t.Context(), allowAll()},
		{"nil invoker", NewExecutor(registry, nil, nil, receipts, nil), t.Context(), allowAll()},
		{"nil context", NewExecutor(registry, nil, invoker, receipts, nil), nil, allowAll()},
		{"nil authorizer", NewExecutor(registry, nil, invoker, receipts, nil), t.Context(), nil},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.executor.ExecuteOperation(test.ctx, operation, test.auth)
			if !errors.Is(err, ErrInvalid) {
				t.Fatalf("got %v, want ErrInvalid", err)
			}
		})
	}
	if _, err := NewExecutor(registry, nil, invoker, nil, nil).ExecuteOperation(t.Context(), operation, allowAll()); !errors.Is(err, ErrReceiptAuthority) {
		t.Fatalf("nil receipt authority = %v", err)
	}
}

func TestExecutionLimitsRejectUnboundedHostConfiguration(t *testing.T) {
	registry := registryWithMediaForTest()
	receipts := newTestReceiptAuthority()
	invoker := invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) { return ProviderOutput{}, nil })
	invalid := []ExecutionLimits{
		{OperationTimeout: hardMaxOperationTimeout + time.Millisecond},
		{CallTimeout: hardMaxCallTimeout + time.Millisecond},
		{OperationTimeout: 5 * time.Millisecond, CallTimeout: 6 * time.Millisecond},
		{MaxConcurrentCalls: hardMaxConcurrentCalls + 1},
	}
	for _, limits := range invalid {
		if executor, err := NewExecutorWithLimits(registry, nil, invoker, receipts, nil, limits); !errors.Is(err, ErrInvalid) || executor != nil {
			t.Fatalf("unbounded limits accepted: limits=%#v executor=%#v err=%v", limits, executor, err)
		}
	}
}

func TestNonCooperativeTimeoutRetainsLeaseAndCapacityAndQuarantinesExactRuntime(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	step, _ := stepByStage(plan, StageTransform)
	receipts := newTestReceiptAuthority()
	operation := operationForStepForTest(t, receipts, plan, step.ID, 1)
	admission := newTestAdmission(step.Processor.Artifact)
	started := make(chan struct{})
	unblock := make(chan struct{})
	var closeUnblock sync.Once
	t.Cleanup(func() { closeUnblock.Do(func() { close(unblock) }) })
	var calls atomic.Int32
	executor, err := NewExecutorWithLimits(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		calls.Add(1)
		close(started)
		<-unblock
		return ProviderOutput{}, nil
	}), receipts, nil, ExecutionLimits{OperationTimeout: 100 * time.Millisecond, CallTimeout: 10 * time.Millisecond, MaxConcurrentCalls: 1})
	if err != nil {
		t.Fatal(err)
	}
	result, err := executor.ExecuteOperation(t.Context(), operation, allowAll())
	<-started
	if !errors.Is(err, ErrRuntimeQuarantined) || result.FallbackOriginal || result.Retry.Retry || result.Receipt.Evidence.ID != "" || receipts.operationCompletion(operation.Key) != nil {
		t.Fatalf("non-cooperative timeout: result=%#v err=%v", result, err)
	}
	if active, released := admission.counts(); active != 1 || released != 0 || len(executor.callSlots) != 1 || receipts.activeReceiptLeases() != 1 {
		t.Fatalf("timed-out callback released ownership early: active=%d released=%d slots=%d receiptLeases=%d", active, released, len(executor.callSlots), receipts.activeReceiptLeases())
	}
	if !executor.IsQuarantined(step.Processor.Artifact) {
		t.Fatal("timed-out exact runtime was not quarantined")
	}
	if result, err := executor.ExecuteOperation(t.Context(), operation, allowAll()); !errors.Is(err, ErrOperationBusy) || !result.Retry.Retry || result.Retry.NextAttempt != operation.Attempt || calls.Load() != 1 {
		t.Fatalf("live callback operation claim escaped: result=%#v calls=%d err=%v", result, calls.Load(), err)
	}
	closeUnblock.Do(func() { close(unblock) })
	deadline := time.Now().Add(time.Second)
	for {
		active, released := admission.counts()
		if active == 0 && released == 1 && len(executor.callSlots) == 0 && receipts.activeReceiptLeases() == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("ownership did not release after callback exit: active=%d released=%d slots=%d", active, released, len(executor.callSlots))
		}
		time.Sleep(time.Millisecond)
	}
	if !executor.IsQuarantined(step.Processor.Artifact) {
		t.Fatal("late callback completion silently reopened exact runtime")
	}
	if result, err := executor.ExecuteOperation(t.Context(), operation, allowAll()); err != nil || !result.FallbackOriginal || result.Receipt.Outcome != TraceFallback || calls.Load() != 1 {
		t.Fatalf("settled quarantine did not produce one terminal fallback: result=%#v calls=%d err=%v", result, calls.Load(), err)
	}
}

func TestRuntimeDrainCancelsCallButRetainsNonCooperativeLease(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	step, _ := stepByStage(plan, StageTransform)
	receipts := newTestReceiptAuthority()
	operation := operationForStepForTest(t, receipts, plan, step.ID, 1)
	leaseCtx, cancelLease := context.WithCancel(context.Background())
	admission := newTestAdmission(step.Processor.Artifact)
	admission.leaseCtx = leaseCtx
	started := make(chan struct{})
	unblock := make(chan struct{})
	var closeUnblock sync.Once
	t.Cleanup(func() { closeUnblock.Do(func() { close(unblock) }) })
	executor, err := NewExecutorWithLimits(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		close(started)
		<-unblock
		return ProviderOutput{}, nil
	}), receipts, nil, ExecutionLimits{OperationTimeout: time.Second, CallTimeout: time.Second, MaxConcurrentCalls: 1})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		result, executeErr := executor.ExecuteOperation(context.Background(), operation, allowAll())
		if errors.Is(executeErr, ErrRuntimeQuarantined) && !result.FallbackOriginal && result.Receipt.Evidence.ID == "" {
			executeErr = nil
		} else {
			executeErr = errors.New("non-cooperative drained processor did not fail closed")
		}
		done <- executeErr
	}()
	<-started
	cancelLease()
	if err := <-done; err != nil {
		t.Fatalf("non-cooperative drained call: %v", err)
	}
	if active, released := admission.counts(); active != 1 || released != 0 || len(executor.callSlots) != 1 || receipts.activeReceiptLeases() != 1 {
		t.Fatalf("drain released live callback: active=%d released=%d slots=%d receiptLeases=%d", active, released, len(executor.callSlots), receipts.activeReceiptLeases())
	}
	closeUnblock.Do(func() { close(unblock) })
	deadline := time.Now().Add(time.Second)
	for {
		active, released := admission.counts()
		if active == 0 && released == 1 && len(executor.callSlots) == 0 && receipts.activeReceiptLeases() == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("drained lease did not release after exit: active=%d released=%d slots=%d", active, released, len(executor.callSlots))
		}
		time.Sleep(time.Millisecond)
	}
}

func TestOperationTimeoutCoversPostProviderHostFences(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	step, _ := stepByStage(plan, StageScan)
	receipts := newTestReceiptAuthority()
	operation := operationForStepForTest(t, receipts, plan, step.ID, 1)
	admission := newTestAdmission(step.Processor.Artifact)
	executor, err := NewExecutorWithLimits(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		return ProviderOutput{Decision: DecisionAllow}, nil
	}), receipts, nil, ExecutionLimits{OperationTimeout: 20 * time.Millisecond, CallTimeout: 10 * time.Millisecond, MaxConcurrentCalls: 1})
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
	if !errors.Is(err, ErrExecutionTimeout) || !result.Retry.Retry || result.Receipt.Evidence.ID != "" || receipts.operationCompletion(operation.Key) != nil {
		t.Fatalf("operation timeout did not cover Host release work: result=%#v checks=%d err=%v", result, checks.Load(), err)
	}
	if active, released := admission.counts(); active != 0 || released != 1 {
		t.Fatalf("operation timeout lease active=%d released=%d", active, released)
	}
}

func TestProviderFailureAndURLNeverEnterTraceInspection(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	traces := NewTraceRing(8)
	receipts := newTestReceiptAuthority()
	transform, _ := stepByStage(plan, StageTransform)
	transformOperation := operationForStepForTest(t, receipts, plan, transform.ID, 3)
	admission := newTestAdmission(transform.Processor.Artifact)
	secretClass := "private_provider_token"
	result, err := NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		return ProviderOutput{}, &ProviderError{Class: secretClass, Code: "private.error", Cause: errors.New("private provider details")}
	}), receipts, traces).ExecuteOperation(t.Context(), transformOperation, allowAll())
	if err != nil || !result.FallbackOriginal {
		t.Fatalf("invalid provider class fallback: result=%#v err=%v", result, err)
	}

	cdn, _ := stepByStage(plan, StageCDN)
	cdnOperation := operationForStepForTest(t, receipts, plan, cdn.ID, 1)
	cdnURL := "https://cdn.example.test/private/object?token=do-not-trace"
	admission = newTestAdmission(cdn.Processor.Artifact)
	result, err = NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		return ProviderOutput{CDNURL: cdnURL}, nil
	}), receipts, traces).ExecuteOperation(t.Context(), cdnOperation, allowAll())
	if err != nil || result.Output.CDNURL != cdnURL {
		t.Fatalf("CDN execution: result=%#v err=%v", result, err)
	}
	encoded, err := json.Marshal(Inspect(registry, traces, 10))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{secretClass, "private provider details", cdnURL, "do-not-trace"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("trace inspection leaked %q: %s", secret, encoded)
		}
	}
}

type blockingTraceSink struct {
	started chan struct{}
	unblock chan struct{}
	once    sync.Once
}

func (sink *blockingTraceSink) AppendMediaTrace(TraceEvent) {
	sink.once.Do(func() { close(sink.started) })
	<-sink.unblock
}

type panickingTraceSink struct{}

func (panickingTraceSink) AppendMediaTrace(TraceEvent) { panic("test trace sink panic") }

func TestExternalTraceSinkCannotDelayOrContradictTerminal(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	scan, _ := stepByStage(plan, StageScan)
	for _, test := range []struct {
		name string
		sink TraceSink
	}{
		{
			name: "blocked",
			sink: &blockingTraceSink{started: make(chan struct{}), unblock: make(chan struct{})},
		},
		{name: "panic", sink: panickingTraceSink{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			receipts := newTestReceiptAuthority()
			operation := operationForStepForTest(t, receipts, plan, scan.ID, 1)
			executor := NewExecutor(registry, newTestAdmission(scan.Processor.Artifact), invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
				return ProviderOutput{Decision: DecisionAllow}, nil
			}), receipts, test.sink)
			done := make(chan struct{})
			var result ExecutionResult
			var executeErr error
			go func() {
				result, executeErr = executor.ExecuteOperation(context.Background(), operation, allowAll())
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("external trace sink delayed committed terminal")
			}
			if executeErr != nil || result.Receipt.Evidence.ID == "" || receipts.operationCompletion(operation.Key) == nil {
				t.Fatalf("terminal contradicted by trace sink: result=%#v err=%v", result, executeErr)
			}
			if sink, ok := test.sink.(*blockingTraceSink); ok {
				select {
				case <-sink.started:
				case <-time.After(time.Second):
					t.Fatal("trace sink did not start")
				}
				close(sink.unblock)
			}
		})
	}
}

func TestRuntimeRevocationDuringCallAndProviderPanicAreRetryable(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	scan, _ := stepByStage(plan, StageScan)
	receipts := newTestReceiptAuthority()
	scanOperation := operationForStepForTest(t, receipts, plan, scan.ID, 1)
	admission := newTestAdmission(scan.Processor.Artifact)
	executor := NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		admission.mu.Lock()
		admission.available[scan.Processor.Artifact] = false
		admission.mu.Unlock()
		return ProviderOutput{Decision: DecisionAllow}, nil
	}), receipts, nil)
	result, err := executor.ExecuteOperation(t.Context(), scanOperation, allowAll())
	if !errors.Is(err, ErrRuntimeUnavailable) || !result.Retry.Retry || result.FallbackOriginal {
		t.Fatalf("mid-call revoke: result=%#v err=%v", result, err)
	}
	active, released := admission.counts()
	if active != 0 || released != 1 {
		t.Fatalf("revoked lease leak active=%d released=%d", active, released)
	}

	transform, _ := stepByStage(plan, StageTransform)
	transformOperation := operationForStepForTest(t, receipts, plan, transform.ID, 1)
	admission = newTestAdmission(transform.Processor.Artifact)
	executor = NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) { panic("provider secret panic") }), receipts, nil)
	result, err = executor.ExecuteOperation(t.Context(), transformOperation, allowAll())
	var providerError *ProviderError
	if !errors.As(err, &providerError) || providerError.Class != RetryCrash || !result.Retry.Retry {
		t.Fatalf("panic classification: result=%#v err=%v", result, err)
	}
	active, released = admission.counts()
	if active != 0 || released != 1 {
		t.Fatalf("panic lease leak active=%d released=%d", active, released)
	}
}

func TestSuccessfulScannerMetadataCDNAndRetentionOutputs(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	retainUntil := time.Now().UTC().Add(24 * time.Hour)
	tests := []struct {
		stage  string
		output ProviderOutput
		check  func(ExecutionResult) bool
	}{
		{StageScan, ProviderOutput{Decision: DecisionAllow}, func(result ExecutionResult) bool { return result.Output.Decision == DecisionAllow }},
		{StageMetadata, ProviderOutput{Metadata: map[string]string{"camera.model": "SForum Test"}}, func(result ExecutionResult) bool { return result.Output.Metadata["camera.model"] == "SForum Test" }},
		{StageCDN, ProviderOutput{CDNURL: "https://cdn.example.test/media/42?variant=original"}, func(result ExecutionResult) bool { return result.Output.CDNURL != "" }},
		{StageRetention, ProviderOutput{RetainUntil: retainUntil}, func(result ExecutionResult) bool { return result.Output.RetainUntil.Equal(retainUntil) }},
	}
	for _, test := range tests {
		t.Run(test.stage, func(t *testing.T) {
			step, found := stepByStage(plan, test.stage)
			if !found {
				t.Fatalf("stage %s missing", test.stage)
			}
			receipts := newTestReceiptAuthority()
			operation := operationForStepForTest(t, receipts, plan, step.ID, 1)
			admission := newTestAdmission(step.Processor.Artifact)
			result, err := NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) { return test.output, nil }), receipts, nil).ExecuteOperation(t.Context(), operation, allowAll())
			if err != nil || !test.check(result) {
				t.Fatalf("result=%#v err=%v", result, err)
			}
		})
	}
}

func TestScannerRejectsAndMalformedOptionalOutputsNeverFallback(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	scan, _ := stepByStage(plan, StageScan)
	receipts := newTestReceiptAuthority()
	scanOperation := operationForStepForTest(t, receipts, plan, scan.ID, 1)
	admission := newTestAdmission(scan.Processor.Artifact)
	result, err := NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		return ProviderOutput{Decision: DecisionReject, ReasonCode: "malware.detected"}, nil
	}), receipts, nil).ExecuteOperation(t.Context(), scanOperation, allowAll())
	if !errors.Is(err, ErrMediaRejected) || result.Retry.Retry || result.FallbackOriginal {
		t.Fatalf("scanner rejection: result=%#v err=%v", result, err)
	}

	cdn, _ := stepByStage(plan, StageCDN)
	cdnOperation := operationForStepForTest(t, receipts, plan, cdn.ID, 1)
	result, err = NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		return ProviderOutput{CDNURL: "http://127.0.0.1/private"}, nil
	}), receipts, nil).ExecuteOperation(t.Context(), cdnOperation, allowAll())
	if !errors.Is(err, ErrOutputRejected) || result.FallbackOriginal {
		t.Fatalf("unsafe CDN output fell back silently: result=%#v err=%v", result, err)
	}
	result, err = NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		return ProviderOutput{CDNURL: "https://cdn.example.test/" + strings.Repeat("x", maxURLBytes)}, nil
	}), receipts, nil).ExecuteOperation(t.Context(), cdnOperation, allowAll())
	if !errors.Is(err, ErrOutputRejected) || result.FallbackOriginal {
		t.Fatalf("oversized CDN output fell back silently: result=%#v err=%v", result, err)
	}

	metadata, _ := stepByStage(plan, StageMetadata)
	metadataOperation := operationForStepForTest(t, receipts, plan, metadata.ID, 1)
	tooLarge := strings.Repeat("x", maxStringBytes+1)
	result, err = NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		return ProviderOutput{Metadata: map[string]string{"camera.model": tooLarge}}, nil
	}), receipts, nil).ExecuteOperation(t.Context(), metadataOperation, allowAll())
	if !errors.Is(err, ErrOutputRejected) || result.Skipped {
		t.Fatalf("malformed metadata skipped silently: result=%#v err=%v", result, err)
	}
}

func TestBackgroundKeysRetriesFallbackAndRedactedTrace(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	receipts := newTestReceiptAuthority()
	initial := sourcePrerequisitesForTest(t, receipts, plan)
	first, err := BackgroundOperations(t.Context(), receipts, plan, initial)
	if err != nil {
		t.Fatal(err)
	}
	second, err := BackgroundOperations(t.Context(), receipts, plan, initial)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || len(second) != 1 || first[0].Key != second[0].Key || first[0].StepID != "scan:demo.media.scan" {
		t.Fatalf("non-deterministic operation keys: %#v %#v", first, second)
	}
	transform, _ := stepByStage(plan, StageTransform)
	transformPrerequisites := prerequisitesBeforeStepForTest(t, receipts, plan, transform.ID)
	runnableTransform, err := BackgroundOperations(t.Context(), receipts, plan, transformPrerequisites)
	if err != nil || len(runnableTransform) != 1 || runnableTransform[0].StepID != transform.ID || runnableTransform[0].Key == first[0].Key {
		t.Fatalf("ordered background transform: operations=%#v err=%v", runnableTransform, err)
	}
	attemptOne := operationForStepForTest(t, receipts, plan, transform.ID, 1)
	attemptThree := operationForStepForTest(t, receipts, plan, transform.ID, 3)
	if attemptOne.Key != attemptThree.Key {
		t.Fatal("retry changed idempotency key")
	}

	admission := newTestAdmission(transform.Processor.Artifact)
	traces := NewTraceRing(8)
	crash := NewProviderError(RetryCrash, "transform.crash", errors.New("secret-token-should-not-be-traced"))
	executor := NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) { return ProviderOutput{}, crash }), receipts, traces)
	result, err := executor.ExecuteOperation(t.Context(), attemptOne, allowAll())
	if err == nil || !result.Retry.Retry || result.Retry.Class != RetryCrash || result.Retry.NextAttempt != 2 || result.Retry.Delay != 2*time.Second {
		t.Fatalf("first retry decision: result=%#v err=%v", result, err)
	}
	result, err = executor.ExecuteOperation(t.Context(), attemptThree, allowAll())
	if err != nil || !result.FallbackOriginal || result.Retry.Retry {
		t.Fatalf("retry exhaustion did not fall back: result=%#v err=%v", result, err)
	}

	inspection := Inspect(registry, traces, 10)
	encoded, err := json.Marshal(inspection)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, secret := range []string{"secret-token-should-not-be-traced", plan.Actor.ID, plan.Actor.PermissionFingerprint, plan.Source.Filename} {
		if strings.Contains(text, secret) {
			t.Fatalf("inspection leaked %q: %s", secret, text)
		}
	}
	if len(inspection.Traces) != 2 || inspection.Traces[0].Outcome != TraceRetry || inspection.Traces[1].Outcome != TraceFallback {
		t.Fatalf("unexpected traces: %#v", inspection.Traces)
	}
}

func TestUnavailableOptionalProviderFallsBackOnlyAfterRetryPolicy(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	transform, _ := stepByStage(plan, StageTransform)
	receipts := newTestReceiptAuthority()
	admission := newTestAdmission()
	executor := NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		t.Fatal("unavailable provider invoked")
		return ProviderOutput{}, nil
	}), receipts, nil)
	first := operationForStepForTest(t, receipts, plan, transform.ID, 1)
	result, err := executor.ExecuteOperation(t.Context(), first, allowAll())
	if !errors.Is(err, ErrRuntimeUnavailable) || !result.Retry.Retry || result.FallbackOriginal {
		t.Fatalf("initial unavailable decision: result=%#v err=%v", result, err)
	}
	last := operationForStepForTest(t, receipts, plan, transform.ID, 3)
	result, err = executor.ExecuteOperation(t.Context(), last, allowAll())
	if err != nil || !result.FallbackOriginal || result.Retry.Retry {
		t.Fatalf("exhausted unavailable decision: result=%#v err=%v", result, err)
	}
}

func TestTerminalReceiptIsNotRecordedWhenPermissionChangesDuringRuntimeRelease(t *testing.T) {
	registry := registryWithMediaForTest()
	plan, err := registry.Plan(t.Context(), uploadRequestForTest(), allowAll())
	if err != nil {
		t.Fatal(err)
	}
	scan, _ := stepByStage(plan, StageScan)
	receipts := newTestReceiptAuthority()
	operation := operationForStepForTest(t, receipts, plan, scan.ID, 1)
	var permitted atomic.Bool
	permitted.Store(true)
	admission := newTestAdmission(scan.Processor.Artifact)
	admission.onRelease = func() { permitted.Store(false) }
	authorizer := authorizerFunc(func(context.Context, AuthorizationRequest) bool { return permitted.Load() })
	var invoked atomic.Int32
	executor := NewExecutor(registry, admission, invokerFunc(func(context.Context, Invocation) (ProviderOutput, error) {
		invoked.Add(1)
		return ProviderOutput{Decision: DecisionAllow}, nil
	}), receipts, nil)
	result, err := executor.ExecuteOperation(t.Context(), operation, authorizer)
	if !errors.Is(err, ErrPermissionDenied) || result.Receipt.Evidence.ID != "" || receipts.operationCompletion(operation.Key) != nil || invoked.Load() != 1 {
		t.Fatalf("terminal escaped pre-commit authority fence: result=%#v invoked=%d err=%v", result, invoked.Load(), err)
	}
}

func TestDeletePlanContainsHooksButCannotRewriteOriginal(t *testing.T) {
	registry := registryWithMediaForTest()
	request := uploadRequestForTest()
	request.Kind = PlanDelete
	request.Permission = "attachment.manage"
	plan, err := registry.Plan(t.Context(), request, allowAll())
	if err != nil {
		t.Fatal(err)
	}
	if plan.Source != plan.OriginalFallback || !plan.Source.Immutable {
		t.Fatalf("delete plan lost source invariant: %#v", plan)
	}
	before, beforeFound := stepByStage(plan, StageBeforeDelete)
	after, afterFound := stepByStage(plan, StageAfterDelete)
	if !beforeFound || !afterFound {
		t.Fatalf("deletion hooks absent: %#v", plan.Steps)
	}
	receipts := newTestReceiptAuthority()
	beforeOperation := operationForStepForTest(t, receipts, plan, before.ID, 1)
	admission := newTestAdmission(before.Processor.Artifact, after.Processor.Artifact)
	invoked := 0
	executor := NewExecutor(registry, admission, invokerFunc(func(_ context.Context, invocation Invocation) (ProviderOutput, error) {
		invoked++
		if invocation.Source != providerSource(plan.Source) {
			t.Fatal("hook source changed or leaked its original filename")
		}
		return ProviderOutput{}, nil
	}), receipts, nil)
	result, err := executor.ExecuteOperation(t.Context(), beforeOperation, allowAll())
	if err != nil || result.Output.Decision != "" || len(result.Output.Metadata) != 0 || len(result.Output.Variants) != 0 || result.Output.CDNURL != "" || !result.Output.RetainUntil.IsZero() {
		t.Fatalf("before-delete hook result=%#v err=%v", result, err)
	}
	afterPrerequisites := cloneOperationPrerequisites(beforeOperation.Prerequisites)
	afterPrerequisites.Steps = append(afterPrerequisites.Steps, result.Receipt)
	if _, err := OperationForStep(t.Context(), receipts, plan, after.ID, 1, afterPrerequisites); !errors.Is(err, ErrDeletionFence) {
		t.Fatalf("after-delete operation without Host fence: %v", err)
	}
	forged := BackgroundOperation{SchemaVersion: SchemaVersion, Key: operationKey(plan, after), StepID: after.ID, Attempt: 1, Plan: clonePlan(plan), Prerequisites: afterPrerequisites}
	if result, err := executor.ExecuteOperation(t.Context(), forged, allowAll()); !errors.Is(err, ErrDeletionFence) || invoked != 1 || result.Output.Decision != "" || len(result.Output.Metadata) != 0 || len(result.Output.Variants) != 0 || result.Output.CDNURL != "" || !result.Output.RetainUntil.IsZero() {
		t.Fatalf("unfenced after-delete executed: result=%#v invoked=%d err=%v", result, invoked, err)
	}
	operations, err := BackgroundOperations(t.Context(), receipts, plan, afterPrerequisites)
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range operations {
		if operation.StepID == after.ID {
			t.Fatal("after-delete escaped into background operations without a Host fence")
		}
	}
	deletion, err := RecordDeletionReceipt(t.Context(), receipts, plan, afterPrerequisites)
	if err != nil {
		t.Fatal(err)
	}
	forgedDeletion := deletion
	forgedDeletion.PredecessorDigest = strings.Repeat("f", 64)
	forgedDeletion.Digest = receiptIntegrityDigest(deletionReceiptClaim(forgedDeletion))
	forgedPrerequisites := cloneOperationPrerequisites(afterPrerequisites)
	forgedPrerequisites.Deletion = &forgedDeletion
	if _, err := OperationForStep(t.Context(), receipts, plan, after.ID, 1, forgedPrerequisites); !errors.Is(err, ErrDeletionFence) {
		t.Fatalf("recomputed forged deletion receipt = %v", err)
	}
	afterPrerequisites.Deletion = &deletion
	afterOperation, err := OperationForStep(t.Context(), receipts, plan, after.ID, 1, afterPrerequisites)
	if err != nil {
		t.Fatal(err)
	}
	receipts.forget(deletion.Evidence)
	if _, err := executor.ExecuteOperation(t.Context(), afterOperation, allowAll()); !errors.Is(err, ErrDeletionFence) || invoked != 1 {
		t.Fatalf("revoked deletion evidence: invoked=%d err=%v", invoked, err)
	}
	afterPrerequisites.Deletion = nil
	deletion, err = RecordDeletionReceipt(t.Context(), receipts, plan, afterPrerequisites)
	if err != nil {
		t.Fatal(err)
	}
	afterPrerequisites.Deletion = &deletion
	afterOperation, err = OperationForStep(t.Context(), receipts, plan, after.ID, 1, afterPrerequisites)
	if err != nil {
		t.Fatal(err)
	}
	var revokedDuringFence atomic.Bool
	revokingAuthorizer := authorizerFunc(func(context.Context, AuthorizationRequest) bool {
		if revokedDuringFence.CompareAndSwap(false, true) {
			receipts.forget(deletion.Evidence)
		}
		return true
	})
	if _, err := executor.ExecuteOperation(t.Context(), afterOperation, revokingAuthorizer); !errors.Is(err, ErrReceiptInvalid) || invoked != 1 {
		t.Fatalf("deletion evidence revoked under lease: invoked=%d err=%v", invoked, err)
	}
	afterPrerequisites.Deletion = nil
	deletion, err = RecordDeletionReceipt(t.Context(), receipts, plan, afterPrerequisites)
	if err != nil {
		t.Fatal(err)
	}
	afterPrerequisites.Deletion = &deletion
	afterOperation, err = OperationForStep(t.Context(), receipts, plan, after.ID, 1, afterPrerequisites)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := executor.ExecuteOperation(t.Context(), afterOperation, allowAll()); err != nil || invoked != 2 {
		t.Fatalf("durably fenced after-delete: invoked=%d err=%v", invoked, err)
	}
}
