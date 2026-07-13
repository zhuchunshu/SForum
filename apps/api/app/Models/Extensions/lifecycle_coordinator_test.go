package extensions

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

var errLifecycleCoordinatorTestCrash = errors.New("simulated coordinator crash")

func TestLifecycleCoordinatorRunsEveryRecommendedOperationAndAction(t *testing.T) {
	tests := []struct {
		operation LifecycleMachineOperation
		actions   []LifecycleMachineAction
	}{
		{LifecycleMachineInstall, []LifecycleMachineAction{LifecycleMachineInstallPlan, LifecycleMachineInstallAction, LifecycleMachineEnableAction}},
		{LifecycleMachineEnable, []LifecycleMachineAction{LifecycleMachineEnableAction}},
		{LifecycleMachineDisable, []LifecycleMachineAction{LifecycleMachineDisableAction}},
		{LifecycleMachineUpgrade, []LifecycleMachineAction{LifecycleMachineUpgradePlan, LifecycleMachineUpgradeBefore, LifecycleMachineUpgradeAfter}},
		{LifecycleMachineRollback, []LifecycleMachineAction{LifecycleMachineRollbackAction}},
		{LifecycleMachineUninstall, []LifecycleMachineAction{LifecycleMachineUninstallPlan, LifecycleMachineUninstallStep, LifecycleMachineUninstallAfter}},
	}
	covered := map[LifecycleMachineAction]bool{}
	for _, test := range tests {
		t.Run(string(test.operation), func(t *testing.T) {
			repository := newLifecycleCoordinatorTestRepository()
			events := &lifecycleCoordinatorTestEvents{}
			runtime := &lifecycleCoordinatorTestRuntime{events: events}
			host := &lifecycleCoordinatorTestHost{events: events}
			coordinator := NewLifecycleCoordinator(repository, runtime, host)
			input := lifecycleCoordinatorTestInput(test.operation, false)
			result, err := coordinator.Run(context.Background(), input)
			if err != nil {
				t.Fatal(err)
			}
			if result.Operation.TerminalResult != LifecycleTerminalSucceeded {
				t.Fatalf("operation = %#v", result.Operation)
			}
			if got := runtime.actionNames(); !slices.Equal(got, test.actions) {
				t.Fatalf("actions = %#v want %#v", got, test.actions)
			}
			for _, action := range test.actions {
				covered[action] = true
			}

			path, _ := RecommendedLifecyclePath(test.operation)
			wantEvents := make([]string, 0, len(path)-1)
			for _, step := range path[1:] {
				if step.Action == "" {
					wantEvents = append(wantEvents, "host:"+string(step.State))
				} else {
					wantEvents = append(wantEvents, "action:"+string(step.Action))
				}
			}
			if got := events.snapshot(); !slices.Equal(got, wantEvents) {
				t.Fatalf("ordered gates = %#v want %#v", got, wantEvents)
			}
		})
	}
	for _, action := range allLifecycleMachineActions() {
		if !covered[action] {
			t.Fatalf("action %q was not invoked", action)
		}
	}
}

func TestLifecycleCoordinatorDistinguishesFirstTrustedEnableFromOrdinaryEnable(t *testing.T) {
	installRuntime := &lifecycleCoordinatorTestRuntime{}
	installResult, err := NewLifecycleCoordinator(
		newLifecycleCoordinatorTestRepository(), installRuntime, &lifecycleCoordinatorTestHost{},
	).Run(context.Background(), lifecycleCoordinatorTestInput(LifecycleMachineInstall, false))
	if err != nil || installResult.Operation.TerminalResult != LifecycleTerminalSucceeded {
		t.Fatalf("first enable = %#v, %v", installResult, err)
	}
	if got := installRuntime.actionNames(); !slices.Equal(got, []LifecycleMachineAction{
		LifecycleMachineInstallPlan, LifecycleMachineInstallAction, LifecycleMachineEnableAction,
	}) {
		t.Fatalf("first enable actions = %#v", got)
	}

	enableRuntime := &lifecycleCoordinatorTestRuntime{}
	_, err = NewLifecycleCoordinator(
		newLifecycleCoordinatorTestRepository(), enableRuntime, &lifecycleCoordinatorTestHost{},
	).Run(context.Background(), lifecycleCoordinatorTestInput(LifecycleMachineEnable, false))
	if err != nil {
		t.Fatal(err)
	}
	if got := enableRuntime.actionNames(); !slices.Equal(got, []LifecycleMachineAction{LifecycleMachineEnableAction}) {
		t.Fatalf("ordinary enable actions = %#v", got)
	}
}

func TestLifecycleCoordinatorResumesOpenAttemptWithStableIdentity(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	repository.failCompleteStepOnce = true
	runtime := &lifecycleCoordinatorTestRuntime{}
	coordinator := NewLifecycleCoordinator(repository, runtime, &lifecycleCoordinatorTestHost{})
	input := lifecycleCoordinatorTestInput(LifecycleMachineEnable, false)
	if _, err := coordinator.Run(context.Background(), input); !errors.Is(err, errLifecycleCoordinatorTestCrash) {
		t.Fatalf("first run = %v", err)
	}
	result, err := coordinator.Run(context.Background(), input)
	if err != nil || result.Operation.TerminalResult != LifecycleTerminalSucceeded {
		t.Fatalf("resumed run = %#v, %v", result, err)
	}
	requests := runtime.requestsSnapshot()
	if len(requests) != 2 || requests[0].StepID != requests[1].StepID || requests[0].Attempt != requests[1].Attempt {
		t.Fatalf("requests = %#v", requests)
	}
	if requests[0].StepID != "lifecycle.enable.02.enable" {
		t.Fatalf("unstable step id %q", requests[0].StepID)
	}
}

func TestLifecycleCoordinatorDoesNotRepeatPersistedSuccessfulAction(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	runtime := &lifecycleCoordinatorTestRuntime{}
	runtime.behaviors = map[LifecycleMachineAction][]lifecycleCoordinatorTestBehavior{
		LifecycleMachineEnableAction: {{after: repository.failNextTransition}},
	}
	coordinator := NewLifecycleCoordinator(repository, runtime, &lifecycleCoordinatorTestHost{})
	input := lifecycleCoordinatorTestInput(LifecycleMachineEnable, false)
	if _, err := coordinator.Run(context.Background(), input); !errors.Is(err, errLifecycleCoordinatorTestCrash) {
		t.Fatalf("first run = %v", err)
	}
	result, err := coordinator.Run(context.Background(), input)
	if err != nil || result.Operation.TerminalResult != LifecycleTerminalSucceeded {
		t.Fatalf("resumed run = %#v, %v", result, err)
	}
	if got := runtime.actionNames(); !slices.Equal(got, []LifecycleMachineAction{LifecycleMachineEnableAction}) {
		t.Fatalf("successful action replayed: %#v", got)
	}
}

func TestLifecycleCoordinatorPersistsFailureAndRetriesFromCheckpointedRecovery(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	runtime := &lifecycleCoordinatorTestRuntime{behaviors: map[LifecycleMachineAction][]lifecycleCoordinatorTestBehavior{
		LifecycleMachineEnableAction: {
			{
				progress: []LifecycleCoordinatorActionProgress{{
					Status: LifecycleStepRunning, Checkpoint: "half", CompletedUnits: 1, TotalUnits: 2,
				}},
				result: LifecycleCoordinatorActionResult{
					Status: LifecycleStepFailed, Checkpoint: "half", CompletedUnits: 1, TotalUnits: 2,
					Error: LifecycleExecutionError{Code: "plugin.failed", Reason: "plugin.failed", Message: "try again", Retryable: true},
				},
			},
			{result: LifecycleCoordinatorActionResult{Status: LifecycleStepSucceeded, Checkpoint: "done", CompletedUnits: 2, TotalUnits: 2}},
		},
	}}
	coordinator := NewLifecycleCoordinator(repository, runtime, &lifecycleCoordinatorTestHost{})
	input := lifecycleCoordinatorTestInput(LifecycleMachineEnable, false)
	failed, err := coordinator.Run(context.Background(), input)
	var runErr *LifecycleCoordinatorRunError
	if !errors.As(err, &runErr) || failed.Operation.State != LifecycleStateFailed ||
		failed.Operation.TerminalResult != LifecycleTerminalFailed || !failed.Operation.Error.Retryable {
		t.Fatalf("failed = %#v, %v", failed, err)
	}
	if _, err := coordinator.Run(context.Background(), input); !errors.Is(err, ErrLifecycleCoordinatorRetryRequired) {
		t.Fatalf("implicit retry = %v", err)
	}
	input.Retry = true
	recovered, err := coordinator.Run(context.Background(), input)
	if err != nil || recovered.Operation.TerminalResult != LifecycleTerminalSucceeded || recovered.Operation.AttemptCount != 2 {
		t.Fatalf("recovered = %#v, %v", recovered, err)
	}
	requests := runtime.requestsSnapshot()
	if len(requests) != 2 || requests[1].Checkpoint != "half" || requests[1].Attempt != 2 || requests[0].StepID != requests[1].StepID {
		t.Fatalf("retry requests = %#v", requests)
	}
	if !slices.Contains(repository.statesSnapshot(), LifecycleStateRecovery) {
		t.Fatalf("states = %#v", repository.statesSnapshot())
	}
}

func TestLifecycleCoordinatorReturnsProgressPersistenceErrorWithoutBusinessFailure(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	runtime := &lifecycleCoordinatorTestRuntime{behaviors: map[LifecycleMachineAction][]lifecycleCoordinatorTestBehavior{
		LifecycleMachineEnableAction: {
			{
				beforeProgress: repository.failNextTransition,
				progress: []LifecycleCoordinatorActionProgress{{
					Status: LifecycleStepRunning, Checkpoint: "half", CompletedUnits: 1, TotalUnits: 2,
				}},
			},
			{result: LifecycleCoordinatorActionResult{
				Status: LifecycleStepSucceeded, Checkpoint: "done", CompletedUnits: 2, TotalUnits: 2,
			}},
		},
	}}
	coordinator := NewLifecycleCoordinator(repository, runtime, &lifecycleCoordinatorTestHost{})
	input := lifecycleCoordinatorTestInput(LifecycleMachineEnable, false)
	interrupted, err := coordinator.Run(context.Background(), input)
	var runErr *LifecycleCoordinatorRunError
	if !errors.Is(err, errLifecycleCoordinatorTestCrash) || errors.As(err, &runErr) ||
		interrupted.Operation.TerminalResult != "" || interrupted.Operation.CompletedAt != nil {
		t.Fatalf("persistence error became business failure: %#v, %v", interrupted, err)
	}
	latest, latestErr := repository.LatestStepAttempt(context.Background(), 1, "lifecycle.enable.02.enable")
	if latestErr != nil || latest.Status != LifecycleStepRunning || latest.Checkpoint != "half" {
		t.Fatalf("open progress = %#v, %v", latest, latestErr)
	}
	resumed, err := coordinator.Run(context.Background(), input)
	if err != nil || resumed.Operation.TerminalResult != LifecycleTerminalSucceeded {
		t.Fatalf("resumed = %#v, %v", resumed, err)
	}
	requests := runtime.requestsSnapshot()
	if len(requests) != 2 || requests[1].Checkpoint != "half" || requests[1].Attempt != 1 {
		t.Fatalf("resumed requests = %#v", requests)
	}
}

func TestLifecycleCoordinatorRetriesFailedHostSafetyGateBeforeAction(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	host := &lifecycleCoordinatorTestHost{failState: LifecycleMachineStarting}
	runtime := &lifecycleCoordinatorTestRuntime{}
	coordinator := NewLifecycleCoordinator(repository, runtime, host)
	input := lifecycleCoordinatorTestInput(LifecycleMachineEnable, false)
	failed, err := coordinator.Run(context.Background(), input)
	if err == nil || failed.Operation.TerminalResult != LifecycleTerminalFailed || len(runtime.actionNames()) != 0 {
		t.Fatalf("failed gate = %#v, %v", failed, err)
	}
	input.Retry = true
	recovered, err := coordinator.Run(context.Background(), input)
	if err != nil || recovered.Operation.TerminalResult != LifecycleTerminalSucceeded {
		t.Fatalf("recovered gate = %#v, %v", recovered, err)
	}
	if got := host.gateIDs(); len(got) < 2 || got[0] != got[1] {
		t.Fatalf("host gate ids = %#v", got)
	}
	if got := runtime.actionNames(); !slices.Equal(got, []LifecycleMachineAction{LifecycleMachineEnableAction}) {
		t.Fatalf("runtime actions = %#v", got)
	}
}

func TestLifecycleCoordinatorForcedUninstallCanSkipFailedCleanup(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	runtime := &lifecycleCoordinatorTestRuntime{behaviors: map[LifecycleMachineAction][]lifecycleCoordinatorTestBehavior{
		LifecycleMachineUninstallAfter: {{result: LifecycleCoordinatorActionResult{
			Status: LifecycleStepFailed,
			Error:  LifecycleExecutionError{Code: "cloud.failed", Reason: "cloud.failed", Message: "subscription remains"},
		}}},
	}}
	coordinator := NewLifecycleCoordinator(repository, runtime, &lifecycleCoordinatorTestHost{})
	input := lifecycleCoordinatorTestInput(LifecycleMachineUninstall, true)
	input.Acquire.RequestedByUserID = 42
	input.Acquire.AuditEventID = 9001
	failed, err := coordinator.Run(context.Background(), input)
	if err == nil || failed.Operation.TerminalResult != LifecycleTerminalFailed {
		t.Fatalf("uninstall failure = %#v, %v", failed, err)
	}
	input.Retry = true
	input.SkipFailedStep = true
	input.SkipReason = "external subscription remains; operator accepted residual risk"
	result, err := coordinator.Run(context.Background(), input)
	if err != nil || result.Operation.TerminalResult != LifecycleTerminalSucceeded || !result.Operation.Forced {
		t.Fatalf("forced result = %#v, %v", result, err)
	}
	actions := runtime.actionNames()
	if countLifecycleCoordinatorAction(actions, LifecycleMachineUninstallAfter) != 1 {
		t.Fatalf("cleanup was reinvoked: %#v", actions)
	}
	latest, err := repository.LatestStepAttempt(context.Background(), result.Operation.ID, "lifecycle.uninstall.05.uninstall.after")
	if err != nil || latest.Status != LifecycleStepSkipped || latest.SkipReason != input.SkipReason || !latest.Forced ||
		latest.ActorUserID != 42 || latest.AuditEventID != 9001 {
		t.Fatalf("skipped attempt = %#v, %v", latest, err)
	}
}

func TestLifecycleCoordinatorRejectsUnforcedCleanupSkipBeforeWritingStep(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	runtime := &lifecycleCoordinatorTestRuntime{behaviors: map[LifecycleMachineAction][]lifecycleCoordinatorTestBehavior{
		LifecycleMachineUninstallAfter: {{result: LifecycleCoordinatorActionResult{
			Status: LifecycleStepFailed,
			Error:  LifecycleExecutionError{Code: "cloud.failed", Reason: "cloud.failed", Message: "subscription remains"},
		}}},
	}}
	coordinator := NewLifecycleCoordinator(repository, runtime, &lifecycleCoordinatorTestHost{})
	input := lifecycleCoordinatorTestInput(LifecycleMachineUninstall, false)
	_, _ = coordinator.Run(context.Background(), input)
	input.Retry = true
	input.SkipFailedStep = true
	input.SkipReason = "unsafe skip"
	if _, err := coordinator.Run(context.Background(), input); !errors.Is(err, ErrLifecycleStateTransitionDenied) {
		t.Fatalf("unforced skip = %v", err)
	}
	latest, err := repository.LatestStepAttempt(context.Background(), 1, "lifecycle.uninstall.05.uninstall.after")
	if err != nil || latest.Attempt != 1 || latest.Status != LifecycleStepFailed {
		t.Fatalf("invalid skip wrote a step: %#v, %v", latest, err)
	}
}

func TestLifecycleCoordinatorFinalizesPersistedTerminalAfterCrash(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	repository.failCompleteOperationOnce = true
	runtime := &lifecycleCoordinatorTestRuntime{}
	coordinator := NewLifecycleCoordinator(repository, runtime, &lifecycleCoordinatorTestHost{})
	input := lifecycleCoordinatorTestInput(LifecycleMachineEnable, false)
	if _, err := coordinator.Run(context.Background(), input); !errors.Is(err, errLifecycleCoordinatorTestCrash) {
		t.Fatalf("first completion = %v", err)
	}
	result, err := coordinator.Run(context.Background(), input)
	if err != nil || result.Operation.TerminalResult != LifecycleTerminalSucceeded {
		t.Fatalf("finalized = %#v, %v", result, err)
	}
	if len(runtime.actionNames()) != 1 {
		t.Fatalf("terminal replay invoked runtime: %#v", runtime.actionNames())
	}
}

func TestLifecycleCoordinatorSuccessfulReplayIsSideEffectFree(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	runtime := &lifecycleCoordinatorTestRuntime{}
	host := &lifecycleCoordinatorTestHost{}
	coordinator := NewLifecycleCoordinator(repository, runtime, host)
	input := lifecycleCoordinatorTestInput(LifecycleMachineEnable, false)
	if _, err := coordinator.Run(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	actionCount := len(runtime.actionNames())
	gateCount := len(host.gateIDs())
	replayed, err := coordinator.Run(context.Background(), input)
	if err != nil || !replayed.Replayed || len(runtime.actionNames()) != actionCount || len(host.gateIDs()) != gateCount {
		t.Fatalf("replay = %#v, %v actions=%#v gates=%#v", replayed, err, runtime.actionNames(), host.gateIDs())
	}
}

func TestLifecycleCoordinatorPreservesActorAndAuditOnEveryActionAttempt(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	input := lifecycleCoordinatorTestInput(LifecycleMachineInstall, false)
	input.Acquire.RequestedByUserID = 42
	input.Acquire.AuditEventID = 9001
	result, err := NewLifecycleCoordinator(repository, &lifecycleCoordinatorTestRuntime{}, &lifecycleCoordinatorTestHost{}).
		Run(context.Background(), input)
	if err != nil || result.Operation.AuditEventID != 9001 || result.Operation.RequestedByUserID != 42 {
		t.Fatalf("operation actor/audit = %#v, %v", result.Operation, err)
	}
	attempts := repository.stepsSnapshot()
	if len(attempts) != 3 {
		t.Fatalf("attempts = %#v", attempts)
	}
	for _, attempt := range attempts {
		if attempt.ActorUserID != 42 || attempt.AuditEventID != 9001 {
			t.Fatalf("attempt lost actor/audit: %#v", attempt)
		}
	}
}

func TestLifecycleCoordinatorRejectsArtifactMismatchBeforeAcquire(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	input := lifecycleCoordinatorTestInput(LifecycleMachineEnable, false)
	input.Extension.PackageDigest = strings.Repeat("b", 64)
	_, err := NewLifecycleCoordinator(repository, &lifecycleCoordinatorTestRuntime{}, &lifecycleCoordinatorTestHost{}).
		Run(context.Background(), input)
	if !errors.Is(err, ErrLifecycleCoordinatorInvalid) || repository.acquired {
		t.Fatalf("artifact mismatch = %v acquired=%t", err, repository.acquired)
	}
}
