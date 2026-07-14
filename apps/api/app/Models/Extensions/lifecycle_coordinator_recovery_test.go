package extensions

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLifecycleCoordinatorUsesBoundedDetachedContextForCallerTermination(t *testing.T) {
	tests := []struct {
		name     string
		makeCtx  func() (context.Context, context.CancelFunc)
		behavior func(context.CancelFunc) lifecycleCoordinatorTestBehavior
		code     string
	}{
		{
			name: "cancelled",
			makeCtx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			behavior: func(cancel context.CancelFunc) lifecycleCoordinatorTestBehavior {
				return lifecycleCoordinatorTestBehavior{cancel: cancel, err: context.Canceled}
			},
			code: "lifecycle.cancelled",
		},
		{
			name: "deadline",
			makeCtx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 5*time.Millisecond)
			},
			behavior: func(context.CancelFunc) lifecycleCoordinatorTestBehavior {
				return lifecycleCoordinatorTestBehavior{waitForContext: true}
			},
			code: "lifecycle.deadline_exceeded",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := newLifecycleCoordinatorTestRepository()
			ctx, cancel := test.makeCtx()
			defer cancel()
			runtime := &lifecycleCoordinatorTestRuntime{behaviors: map[LifecycleMachineAction][]lifecycleCoordinatorTestBehavior{
				LifecycleMachineEnableAction: {test.behavior(cancel)},
			}}
			result, err := NewLifecycleCoordinator(repository, runtime, &lifecycleCoordinatorTestHost{}).
				Run(ctx, lifecycleCoordinatorTestInput(LifecycleMachineEnable, false))
			if err == nil || result.Operation.TerminalResult != LifecycleTerminalCancelled || result.Operation.Error.Code != test.code {
				t.Fatalf("termination = %#v, %v", result, err)
			}
			records := repository.terminalContextsSnapshot()
			if len(records) < 3 {
				t.Fatalf("terminal contexts = %#v", records)
			}
			for _, record := range records {
				if record.err != nil || !record.hasDeadline || record.remaining < 4*time.Second || record.remaining > lifecycleCoordinatorTerminalTimeout+time.Second {
					t.Fatalf("terminal context %s = %#v", record.method, record)
				}
			}
		})
	}
}

func TestLifecycleCoordinatorDetachesActiveCallerDuringTerminalPersistence(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repository.cancelDuringTerminal = cancel
	runtime := &lifecycleCoordinatorTestRuntime{behaviors: map[LifecycleMachineAction][]lifecycleCoordinatorTestBehavior{
		LifecycleMachineEnableAction: {{result: LifecycleCoordinatorActionResult{
			Status: LifecycleStepFailed,
			Error:  LifecycleExecutionError{Code: "plugin.failed", Reason: "plugin.failed", Message: "retry"},
		}}},
	}}
	result, err := NewLifecycleCoordinator(repository, runtime, &lifecycleCoordinatorTestHost{}).
		Run(ctx, lifecycleCoordinatorTestInput(LifecycleMachineEnable, false))
	if err == nil || result.Operation.TerminalResult != LifecycleTerminalFailed {
		t.Fatalf("business failure = %#v, %v", result, err)
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("caller context = %v", ctx.Err())
	}
	records := repository.terminalContextsSnapshot()
	if len(records) < 3 {
		t.Fatalf("terminal contexts = %#v", records)
	}
	for _, record := range records {
		if record.err != nil || !record.hasDeadline || record.remaining < 4*time.Second ||
			record.remaining > lifecycleCoordinatorTerminalTimeout+time.Second {
			t.Fatalf("detached business failure context %s = %#v", record.method, record)
		}
	}
}

func TestLifecycleCoordinatorUsesDetachedContextForSuccessfulAndSkippedStepTerminal(t *testing.T) {
	tests := []struct {
		name      string
		operation LifecycleMachineOperation
		forced    bool
		action    string
		status    string
		prepare   func(*lifecycleCoordinatorTestRepository, *LifecycleCoordinator, *LifecycleCoordinatorRunInput)
	}{
		{
			name: "Host success", operation: LifecycleMachineEnable,
			action: lifecycleCoordinatorHostGateAction, status: LifecycleStepSucceeded,
		},
		{
			name: "action success", operation: LifecycleMachineEnable,
			action: string(LifecycleMachineEnableAction), status: LifecycleStepSucceeded,
		},
		{
			name: "forced skip", operation: LifecycleMachineUninstall, forced: true,
			action: string(LifecycleMachineUninstallAfter), status: LifecycleStepSkipped,
			prepare: func(repository *lifecycleCoordinatorTestRepository, coordinator *LifecycleCoordinator, input *LifecycleCoordinatorRunInput) {
				coordinator.runtime = &lifecycleCoordinatorTestRuntime{behaviors: map[LifecycleMachineAction][]lifecycleCoordinatorTestBehavior{
					LifecycleMachineUninstallAfter: {{result: LifecycleCoordinatorActionResult{
						Status: LifecycleStepFailed,
						Error:  LifecycleExecutionError{Code: "cleanup.failed", Reason: "cleanup.failed", Message: "retry"},
					}}},
				}}
				if result, err := coordinator.Run(context.Background(), *input); err == nil || result.Operation.TerminalResult != LifecycleTerminalFailed {
					t.Fatalf("prepare forced skip = %#v, %v", result, err)
				}
				lifecycleCoordinatorRetry(input)
				input.SkipFailedStep = true
				input.SkipReason = "operator accepted residual cleanup risk"
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := newLifecycleCoordinatorTestRepository()
			coordinator := NewLifecycleCoordinator(repository, &lifecycleCoordinatorTestRuntime{}, &lifecycleCoordinatorTestHost{})
			input := lifecycleCoordinatorTestInput(test.operation, test.forced)
			if test.prepare != nil {
				test.prepare(repository, coordinator, &input)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			previousRecords := len(repository.terminalContextsSnapshot())
			repository.recordStepTerminalAction = test.action
			repository.recordStepTerminalStatus = test.status
			repository.cancelDuringTerminal = cancel
			_, _ = coordinator.Run(ctx, input)
			if !errors.Is(ctx.Err(), context.Canceled) {
				t.Fatalf("caller context = %v", ctx.Err())
			}
			records := repository.terminalContextsSnapshot()
			if len(records) <= previousRecords {
				t.Fatal("step terminal context was not recorded")
			}
			record := records[previousRecords]
			if record.method != "complete_step" || record.err != nil || !record.hasDeadline ||
				record.remaining < 4*time.Second || record.remaining > lifecycleCoordinatorTerminalTimeout+time.Second {
				t.Fatalf("step terminal context = %#v", record)
			}
		})
	}
}

func TestLifecycleCoordinatorRecoversExactHostFailureAfterCompletionCrash(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	repository.failCompleteOperationOnce = true
	host := &lifecycleCoordinatorTestHost{failState: LifecycleMachineStarting}
	coordinator := NewLifecycleCoordinator(repository, &lifecycleCoordinatorTestRuntime{}, host)
	input := lifecycleCoordinatorTestInput(LifecycleMachineEnable, false)
	if _, err := coordinator.Run(context.Background(), input); !errors.Is(err, errLifecycleCoordinatorTestCrash) {
		t.Fatalf("Host completion crash = %v", err)
	}
	recovered, err := coordinator.Run(context.Background(), input)
	if !errors.Is(err, ErrLifecycleCoordinatorRetryRequired) || recovered.Operation.TerminalResult != LifecycleTerminalFailed {
		t.Fatalf("Host failure recovery = %#v, %v", recovered, err)
	}
	if recovered.Operation.Error.Code != "lifecycle.execution_failed" ||
		recovered.Operation.Error.Reason != "lifecycle.execution_failed" ||
		recovered.Operation.Error.Message != "host gate starting failed" || recovered.Operation.Error.Retryable {
		t.Fatalf("Host failure changed during recovery: %#v", recovered.Operation.Error)
	}
	if countLifecycleCoordinatorString(host.gateIDs(), "lifecycle.enable.01.host.starting") != 1 {
		t.Fatalf("Host failure was re-executed: %#v", host.gateIDs())
	}
}

func TestLifecycleCoordinatorPreservesProgressOnTransportTermination(t *testing.T) {
	tests := []struct {
		name       string
		makeCtx    func() (context.Context, context.CancelFunc)
		behavior   func(context.CancelFunc) lifecycleCoordinatorTestBehavior
		stepStatus string
		terminal   string
	}{
		{
			name: "transport_error",
			makeCtx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			behavior: func(context.CancelFunc) lifecycleCoordinatorTestBehavior {
				return lifecycleCoordinatorTestBehavior{
					progress: []LifecycleCoordinatorActionProgress{{
						Status: LifecycleStepRunning, Checkpoint: "half", CompletedUnits: 1, TotalUnits: 2, Message: "halfway",
					}},
					err: errors.New("transport unavailable"),
				}
			},
			stepStatus: LifecycleStepFailed,
			terminal:   LifecycleTerminalFailed,
		},
		{
			name: "caller_cancelled",
			makeCtx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			behavior: func(cancel context.CancelFunc) lifecycleCoordinatorTestBehavior {
				return lifecycleCoordinatorTestBehavior{
					progress: []LifecycleCoordinatorActionProgress{{
						Status: LifecycleStepRunning, Checkpoint: "half", CompletedUnits: 1, TotalUnits: 2, Message: "halfway",
					}},
					cancel: cancel,
					err:    context.Canceled,
				}
			},
			stepStatus: LifecycleStepCancelled,
			terminal:   LifecycleTerminalCancelled,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := newLifecycleCoordinatorTestRepository()
			ctx, cancel := test.makeCtx()
			defer cancel()
			runtime := &lifecycleCoordinatorTestRuntime{behaviors: map[LifecycleMachineAction][]lifecycleCoordinatorTestBehavior{
				LifecycleMachineEnableAction: {test.behavior(cancel)},
			}}
			result, err := NewLifecycleCoordinator(repository, runtime, &lifecycleCoordinatorTestHost{}).
				Run(ctx, lifecycleCoordinatorTestInput(LifecycleMachineEnable, false))
			if err == nil || result.Operation.TerminalResult != test.terminal {
				t.Fatalf("terminal result = %#v, %v", result, err)
			}
			attempt, attemptErr := repository.LatestStepAttempt(context.Background(), result.Operation.ID, "lifecycle.enable.02.enable")
			if attemptErr != nil || attempt.Status != test.stepStatus || attempt.Checkpoint != "half" ||
				attempt.CompletedUnits != 1 || attempt.TotalUnits != 2 || attempt.ProgressMessage != "halfway" {
				t.Fatalf("terminal attempt = %#v, %v", attempt, attemptErr)
			}
		})
	}
}

func TestLifecycleCoordinatorCancelsAfterSuccessfulHostGate(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	ctx, cancel := context.WithCancel(context.Background())
	host := &lifecycleCoordinatorTestHost{cancel: cancel}
	runtime := &lifecycleCoordinatorTestRuntime{}
	result, err := NewLifecycleCoordinator(repository, runtime, host).
		Run(ctx, lifecycleCoordinatorTestInput(LifecycleMachineEnable, false))
	if err == nil || result.Operation.TerminalResult != LifecycleTerminalCancelled ||
		result.Operation.Error.Code != "lifecycle.cancelled" {
		t.Fatalf("host cancellation = %#v, %v", result, err)
	}
	if len(runtime.actionNames()) != 0 {
		t.Fatal("cancelled Host gate reached plugin runtime")
	}
}

func TestLifecycleCoordinatorReconcilesTerminalStepBeforeExplicitRetry(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	runtime := &lifecycleCoordinatorTestRuntime{behaviors: map[LifecycleMachineAction][]lifecycleCoordinatorTestBehavior{
		LifecycleMachineEnableAction: {
			{
				result: LifecycleCoordinatorActionResult{
					Status: LifecycleStepFailed,
					Error:  LifecycleExecutionError{Code: "plugin.failed", Reason: "plugin.failed", Message: "retry"},
				},
				after: repository.failNextTransition,
			},
			{result: LifecycleCoordinatorActionResult{Status: LifecycleStepSucceeded}},
		},
	}}
	coordinator := NewLifecycleCoordinator(repository, runtime, &lifecycleCoordinatorTestHost{})
	input := lifecycleCoordinatorTestInput(LifecycleMachineEnable, false)
	if _, err := coordinator.Run(context.Background(), input); !errors.Is(err, errLifecycleCoordinatorTestCrash) {
		t.Fatalf("failure snapshot crash = %v", err)
	}
	reconciled, err := coordinator.Run(context.Background(), input)
	if !errors.Is(err, ErrLifecycleCoordinatorRetryRequired) || reconciled.Operation.TerminalResult != LifecycleTerminalFailed {
		t.Fatalf("reconciled terminal = %#v, %v", reconciled, err)
	}
	if len(runtime.actionNames()) != 1 {
		t.Fatalf("runtime repeated before retry: %#v", runtime.actionNames())
	}
	lifecycleCoordinatorRetry(&input)
	recovered, err := coordinator.Run(context.Background(), input)
	if err != nil || recovered.Operation.TerminalResult != LifecycleTerminalSucceeded || len(runtime.actionNames()) != 2 {
		t.Fatalf("explicit retry = %#v, %v actions=%#v", recovered, err, runtime.actionNames())
	}
}

func TestLifecycleCoordinatorPersistsRetryAttemptBeforeRecoveryReentry(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	runtime := &lifecycleCoordinatorTestRuntime{behaviors: map[LifecycleMachineAction][]lifecycleCoordinatorTestBehavior{
		LifecycleMachineEnableAction: {
			{result: LifecycleCoordinatorActionResult{
				Status: LifecycleStepFailed, Checkpoint: "half", CompletedUnits: 1, TotalUnits: 2,
				Error: LifecycleExecutionError{Code: "plugin.failed", Reason: "plugin.failed", Message: "retry"},
			}},
			{result: LifecycleCoordinatorActionResult{
				Status: LifecycleStepSucceeded, Checkpoint: "done", CompletedUnits: 2, TotalUnits: 2,
			}},
		},
	}}
	coordinator := NewLifecycleCoordinator(repository, runtime, &lifecycleCoordinatorTestHost{})
	input := lifecycleCoordinatorTestInput(LifecycleMachineEnable, false)
	if failed, err := coordinator.Run(context.Background(), input); err == nil || failed.Operation.TerminalResult != LifecycleTerminalFailed {
		t.Fatalf("initial failure = %#v, %v", failed, err)
	}
	repository.failLatestAfterRecoveryReentry = true
	lifecycleCoordinatorRetry(&input)
	if _, err := coordinator.Run(context.Background(), input); !errors.Is(err, errLifecycleCoordinatorTestCrash) {
		t.Fatalf("reentry crash = %v", err)
	}
	open, err := repository.LatestStepAttempt(context.Background(), 1, "lifecycle.enable.02.enable")
	if err != nil || open.Attempt != 2 || open.Status != LifecycleStepPlanned || open.Checkpoint != "half" ||
		open.CompletedUnits != 0 || open.TotalUnits != 0 {
		t.Fatalf("durable retry attempt = %#v, %v", open, err)
	}
	lifecycleCoordinatorRecoveryReplay(&input)
	recovered, err := coordinator.Run(context.Background(), input)
	if err != nil || recovered.Operation.TerminalResult != LifecycleTerminalSucceeded ||
		recovered.Operation.AttemptCount != 2 || len(runtime.actionNames()) != 2 {
		t.Fatalf("reentry recovery = %#v, %v actions=%#v", recovered, err, runtime.actionNames())
	}
	requests := runtime.requestsSnapshot()
	if requests[1].Attempt != 2 || requests[1].Checkpoint != "half" {
		t.Fatalf("retry request = %#v", requests[1])
	}
}

func TestLifecycleCoordinatorRepairsCrashBetweenResumeAndRecoverySnapshot(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	runtime := &lifecycleCoordinatorTestRuntime{behaviors: map[LifecycleMachineAction][]lifecycleCoordinatorTestBehavior{
		LifecycleMachineEnableAction: {
			{result: LifecycleCoordinatorActionResult{
				Status: LifecycleStepFailed,
				Error:  LifecycleExecutionError{Code: "plugin.failed", Reason: "plugin.failed", Message: "retry"},
			}},
			{result: LifecycleCoordinatorActionResult{Status: LifecycleStepSucceeded}},
		},
	}}
	coordinator := NewLifecycleCoordinator(repository, runtime, &lifecycleCoordinatorTestHost{})
	input := lifecycleCoordinatorTestInput(LifecycleMachineEnable, false)
	_, _ = coordinator.Run(context.Background(), input)
	repository.failNextTransition()
	lifecycleCoordinatorRetry(&input)
	if _, err := coordinator.Run(context.Background(), input); !errors.Is(err, errLifecycleCoordinatorTestCrash) {
		t.Fatalf("resume crash = %v", err)
	}
	result, err := coordinator.Run(context.Background(), input)
	if err != nil || result.Operation.TerminalResult != LifecycleTerminalSucceeded || result.Operation.AttemptCount != 2 {
		t.Fatalf("repaired = %#v, %v", result, err)
	}
	if len(runtime.actionNames()) != 2 {
		t.Fatalf("runtime requests = %#v", runtime.requestsSnapshot())
	}
}
