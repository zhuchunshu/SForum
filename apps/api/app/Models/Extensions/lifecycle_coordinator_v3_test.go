package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestLifecycleCoordinatorPersistsAndRevalidatesEphemeralPlannedGate(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	plannedStepID := "lifecycle.upgrade.00.host.planned"
	host := &lifecycleCoordinatorTestHost{results: map[string][]LifecycleCoordinatorGateResult{
		plannedStepID: {
			{
				Checkpoint:         "prepared-1",
				SourceBinding:      LifecycleRuntimeBinding{RuntimeInstanceID: "source-instance"},
				TargetBinding:      LifecycleRuntimeBinding{RuntimeInstanceID: "target-instance-1"},
				RevalidationPolicy: LifecycleGateRevalidationRequired,
				ResultDocument:     json.RawMessage(`{"phase":"prepared"}`),
			},
			{
				Checkpoint:         "prepared-2",
				SourceBinding:      LifecycleRuntimeBinding{RuntimeInstanceID: "source-instance"},
				TargetBinding:      LifecycleRuntimeBinding{RuntimeInstanceID: "target-instance-2"},
				RevalidationPolicy: LifecycleGateRevalidationRequired,
				ResultDocument:     json.RawMessage(`{"phase":"recreated"}`),
			},
		},
	}}
	runtime := &lifecycleCoordinatorTestRuntime{behaviors: map[LifecycleMachineAction][]lifecycleCoordinatorTestBehavior{
		LifecycleMachineUpgradePlan: {{after: repository.failNextCompleteStep}},
	}}
	coordinator := NewLifecycleCoordinator(repository, runtime, host)
	input := lifecycleCoordinatorTestInput(LifecycleMachineUpgrade, false)
	input.Acquire.RequestedByUserID = 42
	input.Acquire.AuditEventID = 9001
	input.Acquire.AuthoritySnapshot = json.RawMessage(`{"schemaVersion":"test"}`)

	if _, err := coordinator.Run(context.Background(), input); !errors.Is(err, errLifecycleCoordinatorTestCrash) {
		t.Fatalf("first run = %v", err)
	}
	repository.expireOpenLease("")
	result, err := coordinator.Run(context.Background(), input)
	if err != nil || result.Operation.TerminalResult != LifecycleTerminalSucceeded {
		t.Fatalf("resumed run = %#v, %v", result, err)
	}

	gates := host.requestsSnapshot()
	if len(gates) < 2 || gates[0].StepID != plannedStepID || gates[0].Revalidation ||
		gates[1].StepID != plannedStepID || !gates[1].Revalidation ||
		gates[1].Checkpoint != "prepared-1" || len(gates[1].PreviousResult) == 0 {
		t.Fatalf("planned gate replay = %#v", gates)
	}
	if gates[0].OperationID != 1 || gates[0].ActorUserID != 42 || gates[0].AuditEventID != 9001 ||
		gates[0].SourceBinding.ExtensionVersion != "0.9.0" || gates[0].TargetBinding.ExtensionVersion != "1.0.0" ||
		string(gates[0].AuthoritySnapshot) != `{"schemaVersion":"test"}` {
		t.Fatalf("planned gate authority/bindings = %#v", gates[0])
	}

	actions := runtime.requestsSnapshot()
	if len(actions) < 2 || actions[1].Action != LifecycleMachineUpgradePlan ||
		actions[1].TargetBinding.RuntimeInstanceID != "target-instance-2" ||
		actions[1].OperationID != 1 || actions[1].ActorUserID != 42 || actions[1].AuditEventID != 9001 {
		t.Fatalf("resumed action binding = %#v", actions)
	}
	before := slices.IndexFunc(actions, func(request LifecycleCoordinatorActionRequest) bool {
		return request.Action == LifecycleMachineUpgradeBefore
	})
	if before < 0 || actions[before].RuntimeRole != LifecycleRuntimeSource ||
		actions[before].Extension.Version != "0.9.0" || actions[before].SourceBinding.RuntimeInstanceID != "source-instance" {
		t.Fatalf("upgrade.before source request = %#v", actions)
	}

	attempt, err := repository.LatestStepAttempt(context.Background(), 1, plannedStepID)
	if err != nil || attempt.Attempt != 2 || attempt.Checkpoint != "prepared-2" {
		t.Fatalf("typed Host attempt = %#v, %v", attempt, err)
	}
	gateResult, typed, err := decodeLifecycleHostGateResult(attempt.ResultDocument)
	if err != nil || !typed || gateResult.TargetBinding.RuntimeInstanceID != "target-instance-2" ||
		string(gateResult.ResultDocument) != `{"phase":"recreated"}` {
		t.Fatalf("typed Host result = %#v typed=%t err=%v", gateResult, typed, err)
	}
}

func TestLifecycleCoordinatorRetriesInitialPlannedHostGate(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	host := &lifecycleCoordinatorTestHost{failState: LifecycleMachinePlanned}
	runtime := &lifecycleCoordinatorTestRuntime{}
	coordinator := NewLifecycleCoordinator(repository, runtime, host)
	input := lifecycleCoordinatorTestInput(LifecycleMachineInstall, false)
	failed, err := coordinator.Run(context.Background(), input)
	if err == nil || failed.Operation.TerminalResult != LifecycleTerminalFailed || len(runtime.requestsSnapshot()) != 0 {
		t.Fatalf("initial planned failure = %#v, %v", failed, err)
	}
	lifecycleCoordinatorRetry(&input)
	recovered, err := coordinator.Run(context.Background(), input)
	if err != nil || recovered.Operation.TerminalResult != LifecycleTerminalSucceeded ||
		countLifecycleCoordinatorString(host.gateIDs(), "lifecycle.install.00.host.planned") != 2 {
		t.Fatalf("initial planned recovery = %#v, %v gates=%#v", recovered, err, host.gateIDs())
	}
}

func TestLifecycleCoordinatorReconcilesCrashedHostRevalidationFailure(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	plannedStepID := "lifecycle.upgrade.00.host.planned"
	revalidationFailure := errors.New("prepared runtime disappeared")
	host := &lifecycleCoordinatorTestHost{
		results: map[string][]LifecycleCoordinatorGateResult{
			plannedStepID: {{
				SourceBinding:      LifecycleRuntimeBinding{RuntimeInstanceID: "source-instance"},
				TargetBinding:      LifecycleRuntimeBinding{RuntimeInstanceID: "target-instance"},
				RevalidationPolicy: LifecycleGateRevalidationRequired,
			}},
		},
		gateErrors: map[string][]error{plannedStepID: {nil, revalidationFailure}},
	}
	runtime := &lifecycleCoordinatorTestRuntime{behaviors: map[LifecycleMachineAction][]lifecycleCoordinatorTestBehavior{
		LifecycleMachineUpgradePlan: {{after: repository.failNextCompleteStep}},
	}}
	coordinator := NewLifecycleCoordinator(repository, runtime, host)
	input := lifecycleCoordinatorTestInput(LifecycleMachineUpgrade, false)
	if _, err := coordinator.Run(context.Background(), input); !errors.Is(err, errLifecycleCoordinatorTestCrash) {
		t.Fatalf("prepare run = %v", err)
	}
	repository.expireOpenLease("")
	repository.failNextTransition()
	if _, err := coordinator.Run(context.Background(), input); !errors.Is(err, errLifecycleCoordinatorTestCrash) {
		t.Fatalf("failure persistence crash = %v", err)
	}
	reconciled, err := coordinator.Run(context.Background(), input)
	var runErr *LifecycleCoordinatorRunError
	if !errors.As(err, &runErr) || reconciled.Operation.TerminalResult != LifecycleTerminalFailed ||
		reconciled.Operation.Error.Message != revalidationFailure.Error() ||
		countLifecycleCoordinatorString(host.gateIDs(), plannedStepID) != 2 {
		t.Fatalf("reconciled revalidation = %#v, %v gates=%#v", reconciled, err, host.gateIDs())
	}
}

func TestLifecycleCoordinatorRequestsCarryExactOperationContext(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	runtime := &lifecycleCoordinatorTestRuntime{}
	host := &lifecycleCoordinatorTestHost{}
	input := lifecycleCoordinatorTestInput(LifecycleMachineUninstall, true)
	input.Acquire.RequestedByUserID = 71
	input.Acquire.AuditEventID = 72
	input.Acquire.RemovalMode = LifecycleRemovalComplete
	input.Acquire.AuthoritySnapshot = json.RawMessage(`{"authority":"builtin"}`)
	result, err := NewLifecycleCoordinator(repository, runtime, host).Run(context.Background(), input)
	if err != nil || result.Operation.TerminalResult != LifecycleTerminalSucceeded {
		t.Fatalf("uninstall = %#v, %v", result, err)
	}
	for _, request := range runtime.requestsSnapshot() {
		if request.OperationID != result.Operation.ID || request.RemovalMode != LifecycleRemovalComplete ||
			request.ActorUserID != 71 || request.AuditEventID != 72 || !request.Forced ||
			request.RuntimeRole != LifecycleRuntimeSource || request.SourceBinding.PackageDigest != input.Extension.PackageDigest ||
			string(request.AuthoritySnapshot) != `{"authority":"builtin"}` {
			t.Fatalf("action request lost operation context: %#v", request)
		}
	}
	for _, request := range host.requestsSnapshot() {
		if request.OperationID != result.Operation.ID || request.RemovalMode != LifecycleRemovalComplete ||
			request.ActorUserID != 71 || request.AuditEventID != 72 || !request.Forced ||
			request.SourceBinding.PackageDigest != input.Extension.PackageDigest ||
			string(request.AuthoritySnapshot) != `{"authority":"builtin"}` {
			t.Fatalf("Host request lost operation context: %#v", request)
		}
	}
}

func TestLifecycleCoordinatorV1SnapshotMigrationPreservesStableHistory(t *testing.T) {
	legacyUpgradeAfter := LifecycleStateMachine{
		Operation: LifecycleMachineUpgrade, State: LifecycleMachineEnabled,
		Action: LifecycleMachineUpgradeAfter, Position: 8,
		Progress: LifecycleProgressCursor{CompletedUnits: 7, TotalUnits: 8},
	}
	decoded := decodeLifecycleCoordinatorV1TestMachine(t, legacyUpgradeAfter)
	if decoded.Position != 9 || decoded.Action != LifecycleMachineUpgradeAfter || decoded.StepComplete ||
		decoded.Progress.TotalUnits != 10 || lifecycleCoordinatorStepID(decoded.Operation, decoded.Position, decoded.State, decoded.Action) != "lifecycle.upgrade.08.upgrade.after" {
		t.Fatalf("open upgrade.after migration = %#v", decoded)
	}

	legacyInitial := LifecycleStateMachine{
		Operation: LifecycleMachineEnable, State: LifecycleMachinePlanned, Position: 0, StepComplete: true,
		Progress: LifecycleProgressCursor{TotalUnits: 5},
	}
	if decoded := decodeLifecycleCoordinatorV1TestMachine(t, legacyInitial); decoded.StepComplete || decoded.Position != 0 {
		t.Fatalf("legacy synthetic initial gate was not reopened: %#v", decoded)
	}

	terminals := []LifecycleStateMachine{
		{Operation: LifecycleMachineDisable, State: LifecycleMachineDraining, Action: LifecycleMachineDisableAction, Position: 2, StepComplete: true, TerminalResult: LifecycleMachineSucceeded, Progress: LifecycleProgressCursor{CompletedUnits: 2, TotalUnits: 2}},
		{Operation: LifecycleMachineUpgrade, State: LifecycleMachineEnabled, Action: LifecycleMachineUpgradeAfter, Position: 8, StepComplete: true, TerminalResult: LifecycleMachineSucceeded, Progress: LifecycleProgressCursor{CompletedUnits: 8, TotalUnits: 8}},
		{Operation: LifecycleMachineUninstall, State: LifecycleMachineUninstalling, Action: LifecycleMachineUninstallAfter, Position: 5, StepComplete: true, TerminalResult: LifecycleMachineSucceeded, Progress: LifecycleProgressCursor{CompletedUnits: 5, TotalUnits: 5}},
	}
	for _, legacy := range terminals {
		decoded := decodeLifecycleCoordinatorV1TestMachine(t, legacy)
		path, _ := RecommendedLifecyclePath(legacy.Operation)
		if decoded.TerminalResult != LifecycleMachineSucceeded || decoded.Position != len(path)-1 ||
			!decoded.StepComplete || decoded.Action != "" || decoded.Progress.CompletedUnits != decoded.Progress.TotalUnits {
			t.Fatalf("terminal %s migration = %#v", legacy.Operation, decoded)
		}
	}
}

func TestLifecycleCoordinatorV1OpenSnapshotsPrepareExactRuntimeBeforeContinuing(t *testing.T) {
	tests := []struct {
		name      string
		operation LifecycleMachineOperation
		machine   LifecycleStateMachine
		assert    func(*testing.T, *lifecycleCoordinatorTestHost, *lifecycleCoordinatorTestRuntime)
	}{
		{
			name: "synthetic position zero", operation: LifecycleMachineEnable,
			machine: LifecycleStateMachine{
				Operation: LifecycleMachineEnable, State: LifecycleMachinePlanned,
				Position: 0, StepComplete: true, Progress: LifecycleProgressCursor{TotalUnits: 5},
			},
			assert: func(t *testing.T, host *lifecycleCoordinatorTestHost, runtime *lifecycleCoordinatorTestRuntime) {
				gates := host.requestsSnapshot()
				if len(gates) == 0 || gates[0].StepID != "lifecycle.enable.00.host.planned" || gates[0].Revalidation {
					t.Fatalf("position zero planned gate = %#v", gates)
				}
				requests := runtime.requestsSnapshot()
				if len(requests) == 0 || requests[0].TargetBinding.RuntimeInstanceID == "" {
					t.Fatalf("position zero action binding = %#v", requests)
				}
			},
		},
		{
			name: "completed install plan", operation: LifecycleMachineInstall,
			machine: LifecycleStateMachine{
				Operation: LifecycleMachineInstall, State: LifecycleMachinePlanned,
				Action: LifecycleMachineInstallPlan, Position: 1, StepComplete: true,
				Progress: LifecycleProgressCursor{CompletedUnits: 1, TotalUnits: 8},
			},
			assert: func(t *testing.T, host *lifecycleCoordinatorTestHost, runtime *lifecycleCoordinatorTestRuntime) {
				gates := host.requestsSnapshot()
				if len(gates) == 0 || gates[0].StepID != "lifecycle.install.00.host.planned" || !gates[0].Revalidation || len(gates[0].PreviousResult) != 0 {
					t.Fatalf("install plan recovery gate = %#v", gates)
				}
				requests := runtime.requestsSnapshot()
				if len(requests) == 0 || requests[0].Action != LifecycleMachineInstallAction ||
					requests[0].TargetBinding.RuntimeInstanceID == "" {
					t.Fatalf("install continuation = %#v", requests)
				}
			},
		},
		{
			name: "before upgrade source cleanup", operation: LifecycleMachineUpgrade,
			machine: LifecycleStateMachine{
				Operation: LifecycleMachineUpgrade, State: LifecycleMachineDraining,
				Action: LifecycleMachineUpgradeBefore, Position: 3,
				Progress: LifecycleProgressCursor{CompletedUnits: 2, TotalUnits: 8},
			},
			assert: func(t *testing.T, host *lifecycleCoordinatorTestHost, runtime *lifecycleCoordinatorTestRuntime) {
				gates := host.requestsSnapshot()
				if len(gates) == 0 || gates[0].StepID != "lifecycle.upgrade.00.host.planned" || !gates[0].Revalidation {
					t.Fatalf("upgrade recovery gate = %#v", gates)
				}
				requests := runtime.requestsSnapshot()
				if len(requests) == 0 || requests[0].StepID != "lifecycle.upgrade.03.upgrade.before" ||
					requests[0].RuntimeRole != LifecycleRuntimeSource || requests[0].SourceBinding.RuntimeInstanceID == "" ||
					requests[0].TargetBinding.RuntimeInstanceID == "" {
					t.Fatalf("upgrade source continuation = %#v", requests)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := newLifecycleCoordinatorTestRepository()
			input := lifecycleCoordinatorTestInput(test.operation, false)
			seedLifecycleCoordinatorV1OpenOperation(t, repository, input, test.machine)
			host := &lifecycleCoordinatorTestHost{}
			runtime := &lifecycleCoordinatorTestRuntime{}
			result, err := NewLifecycleCoordinator(repository, runtime, host).Run(context.Background(), input)
			if err != nil || result.Operation.TerminalResult != LifecycleTerminalSucceeded {
				t.Fatalf("recovered operation = %#v, %v", result, err)
			}
			test.assert(t, host, runtime)
		})
	}
}

func TestLifecycleCoordinatorV1SucceededSnapshotRemainsClosed(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	input := lifecycleCoordinatorTestInput(LifecycleMachineUpgrade, false)
	machine := LifecycleStateMachine{
		Operation: LifecycleMachineUpgrade, State: LifecycleMachineEnabled,
		Action: LifecycleMachineUpgradeAfter, Position: 8, StepComplete: true,
		TerminalResult: LifecycleMachineSucceeded,
		Progress:       LifecycleProgressCursor{CompletedUnits: 8, TotalUnits: 8},
	}
	seedLifecycleCoordinatorV1OpenOperation(t, repository, input, machine)
	now := time.Now()
	repository.mu.Lock()
	repository.operation.TerminalResult = LifecycleTerminalSucceeded
	repository.operation.CompletedAt = &now
	repository.mu.Unlock()
	host := &lifecycleCoordinatorTestHost{}
	runtime := &lifecycleCoordinatorTestRuntime{}
	result, err := NewLifecycleCoordinator(repository, runtime, host).Run(context.Background(), input)
	if err != nil || result.Operation.TerminalResult != LifecycleTerminalSucceeded || !result.Replayed ||
		len(host.requestsSnapshot()) != 0 || len(runtime.requestsSnapshot()) != 0 {
		t.Fatalf("closed v1 terminal replay = %#v, %v host=%#v runtime=%#v", result, err, host.requestsSnapshot(), runtime.requestsSnapshot())
	}
}

func TestLifecycleCoordinatorDurableGateCannotClearRuntimeRevalidation(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	plannedStepID := "lifecycle.install.00.host.planned"
	migratingStepID := "lifecycle.install.02.host.migrating"
	host := &lifecycleCoordinatorTestHost{
		results: map[string][]LifecycleCoordinatorGateResult{
			plannedStepID: {
				{TargetBinding: LifecycleRuntimeBinding{RuntimeInstanceID: "target-one"}, RevalidationPolicy: LifecycleGateRevalidationRequired},
				{TargetBinding: LifecycleRuntimeBinding{RuntimeInstanceID: "target-two"}, RevalidationPolicy: LifecycleGateRevalidationRequired},
			},
			migratingStepID: {{RevalidationPolicy: LifecycleGateDurable}},
		},
		afterStep: map[string][]func(){migratingStepID: {repository.failNextTransition}},
	}
	runtime := &lifecycleCoordinatorTestRuntime{}
	coordinator := NewLifecycleCoordinator(repository, runtime, host)
	input := lifecycleCoordinatorTestInput(LifecycleMachineInstall, false)
	if _, err := coordinator.Run(context.Background(), input); !errors.Is(err, errLifecycleCoordinatorTestCrash) {
		t.Fatalf("durable gate crash = %v", err)
	}
	result, err := coordinator.Run(context.Background(), input)
	if err != nil || result.Operation.TerminalResult != LifecycleTerminalSucceeded {
		t.Fatalf("durable gate recovery = %#v, %v", result, err)
	}
	gates := host.requestsSnapshot()
	planned := make([]LifecycleCoordinatorGateRequest, 0, 2)
	for _, request := range gates {
		if request.StepID == plannedStepID {
			planned = append(planned, request)
		}
	}
	if len(planned) != 2 || planned[0].Revalidation || !planned[1].Revalidation ||
		planned[1].TargetBinding.RuntimeInstanceID != "target-one" {
		t.Fatalf("planned revalidation after durable gate = %#v", planned)
	}
	install := slices.IndexFunc(runtime.requestsSnapshot(), func(request LifecycleCoordinatorActionRequest) bool {
		return request.Action == LifecycleMachineInstallAction
	})
	requests := runtime.requestsSnapshot()
	if install < 0 || requests[install].TargetBinding.RuntimeInstanceID != "target-two" {
		t.Fatalf("recreated target binding = %#v", requests)
	}
}

func TestLifecycleCoordinatorReplaysFinalHostGateBeforeRuntimeRevalidation(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	plannedStepID := "lifecycle.install.00.host.planned"
	finalStepID := "lifecycle.install.08.host.enabled"
	host := &lifecycleCoordinatorTestHost{afterStep: map[string][]func(){
		finalStepID: {repository.failNextTransition},
	}}
	coordinator := NewLifecycleCoordinator(repository, &lifecycleCoordinatorTestRuntime{}, host)
	input := lifecycleCoordinatorTestInput(LifecycleMachineInstall, false)

	if _, err := coordinator.Run(context.Background(), input); !errors.Is(err, errLifecycleCoordinatorTestCrash) {
		t.Fatalf("final Host persistence crash = %v", err)
	}
	result, err := coordinator.Run(context.Background(), input)
	if err != nil || result.Operation.TerminalResult != LifecycleTerminalSucceeded {
		t.Fatalf("final Host replay = %#v, %v", result, err)
	}
	if countLifecycleCoordinatorString(host.gateIDs(), plannedStepID) != 1 ||
		countLifecycleCoordinatorString(host.gateIDs(), finalStepID) != 1 {
		t.Fatalf("terminal replay rebuilt runtime: %#v", host.gateIDs())
	}
}

func TestLifecycleCoordinatorLegacyPlannedHostReplayRemainsSideEffecting(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	coordinator := NewLifecycleCoordinator(repository, nil, nil)
	input := lifecycleCoordinatorTestInput(LifecycleMachineInstall, false)
	acquired, err := repository.AcquireOperation(context.Background(), input.Acquire)
	if err != nil {
		t.Fatal(err)
	}
	machine, _ := NewLifecycleStateMachine(LifecycleMachineInstall, false)
	if err := hydrateLifecycleCoordinatorBindings(&machine, input.Extension, input.SourceExtension); err != nil {
		t.Fatal(err)
	}
	path, _ := RecommendedLifecyclePath(machine.Operation)
	machine.Progress.TotalUnits = uint64(len(path) - 1)
	stepID := "lifecycle.install.00.host.planned"
	operation, err := coordinator.persistMachine(context.Background(), acquired.Operation, machine, stepID)
	if err != nil {
		t.Fatal(err)
	}
	begin, err := repository.BeginStepAttempt(context.Background(), BeginLifecycleStepAttemptInput{
		OperationID: operation.ID, StepID: stepID, LifecycleAction: lifecycleCoordinatorHostGateAction,
		PlanVersion: operation.PlanVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.CompleteStepAttempt(context.Background(), CompleteLifecycleStepAttemptInput{
		AttemptID: begin.Attempt.ID, Status: LifecycleStepSucceeded,
		ResultDocument: json.RawMessage(`{"legacy":"untyped"}`),
	}); err != nil {
		t.Fatal(err)
	}

	_, replayed, err := coordinator.reconcilePendingStepTerminal(context.Background(), operation, machine)
	if err != nil || !replayed.StepComplete || !replayed.HostSideEffectsStarted {
		t.Fatalf("legacy planned Host replay = %#v, %v", replayed, err)
	}
}

func TestLifecycleCoordinatorClaimsCurrentStepAcrossRevalidationAndReleasesOnFailure(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	plannedStepID := "lifecycle.upgrade.00.host.planned"
	currentStepID := "lifecycle.upgrade.01.upgrade.plan"
	revalidationFailure := errors.New("prepared runtime disappeared")
	currentLeaseObserved := false
	host := &lifecycleCoordinatorTestHost{
		gateErrors: map[string][]error{plannedStepID: {nil, revalidationFailure}},
		afterStep: map[string][]func(){plannedStepID: {
			nil,
			func() {
				attempt, err := repository.LatestStepAttempt(context.Background(), 1, currentStepID)
				currentLeaseObserved = err == nil && attempt.LeaseOwnerToken != ""
			},
		}},
	}
	runtime := &lifecycleCoordinatorTestRuntime{behaviors: map[LifecycleMachineAction][]lifecycleCoordinatorTestBehavior{
		LifecycleMachineUpgradePlan: {{after: repository.failNextCompleteStep}},
	}}
	coordinator := NewLifecycleCoordinator(repository, runtime, host)
	input := lifecycleCoordinatorTestInput(LifecycleMachineUpgrade, false)

	if _, err := coordinator.Run(context.Background(), input); !errors.Is(err, errLifecycleCoordinatorTestCrash) {
		t.Fatalf("prepare current step = %v", err)
	}
	repository.expireOpenLease("")
	result, err := coordinator.Run(context.Background(), input)
	var runErr *LifecycleCoordinatorRunError
	if !errors.As(err, &runErr) || result.Operation.TerminalResult != LifecycleTerminalFailed || !currentLeaseObserved {
		t.Fatalf("revalidation failure = %#v, %v currentLease=%t", result, err, currentLeaseObserved)
	}
	attempt, latestErr := repository.LatestStepAttempt(context.Background(), 1, currentStepID)
	if latestErr != nil || attempt.LeaseOwnerToken != "" || attempt.LeaseExpiresAt != nil || attempt.LeaseHeartbeatAt != nil {
		t.Fatalf("current lease was not precisely released: %#v, %v", attempt, latestErr)
	}
}

func TestLifecycleRequiredRevalidationProvesEveryRuntimeRole(t *testing.T) {
	input := lifecycleCoordinatorTestInput(LifecycleMachineUpgrade, false)
	machine, _ := NewLifecycleStateMachine(LifecycleMachineUpgrade, false)
	if err := hydrateLifecycleCoordinatorBindings(&machine, input.Extension, input.SourceExtension); err != nil {
		t.Fatal(err)
	}
	machine.SourceBinding.RuntimeInstanceID = "source-old"
	machine.TargetBinding.RuntimeInstanceID = "target-old"
	original := machine

	partial := []LifecycleCoordinatorGateResult{
		{
			TargetBinding:      LifecycleRuntimeBinding{RuntimeInstanceID: "target-new"},
			RevalidationPolicy: LifecycleGateRevalidationRequired,
		},
		{
			SourceBinding:      LifecycleRuntimeBinding{RuntimeInstanceID: "source-new"},
			RevalidationPolicy: LifecycleGateRevalidationRequired,
		},
	}
	for _, result := range partial {
		if _, err := applyLifecycleHostGateResult(machine, "lifecycle.upgrade.00.host.planned", 0, result); !errors.Is(err, ErrLifecycleCoordinatorInvalid) {
			t.Fatalf("partial upgrade revalidation %#v = %v", result, err)
		}
		if machine != original {
			t.Fatalf("partial revalidation mutated caller snapshot: before=%#v after=%#v", original, machine)
		}
	}
}

func TestLifecycleCoordinatorRejectsNonCanonicalRevalidationMarker(t *testing.T) {
	machine, _ := NewLifecycleStateMachine(LifecycleMachineInstall, false)
	machine.Revalidation = LifecycleGateRevalidation{
		StepID: "lifecycle.install.99.host.planned", Position: 0,
	}
	if _, err := validateLifecycleMachine(machine); !errors.Is(err, ErrLifecycleStateMachineInvalid) {
		t.Fatalf("non-canonical marker = %v", err)
	}
}

func TestLifecycleCoordinatorRejectsInvalidSourceBeforeRepositoryAcquire(t *testing.T) {
	tests := []struct {
		name      string
		operation LifecycleMachineOperation
		mutate    func(*LifecycleCoordinatorRunInput)
	}{
		{
			name: "install with source", operation: LifecycleMachineInstall,
			mutate: func(input *LifecycleCoordinatorRunInput) {
				source := input.Extension
				input.SourceExtension = &source
			},
		},
		{
			name: "disable with historical source", operation: LifecycleMachineDisable,
			mutate: func(input *LifecycleCoordinatorRunInput) {
				input.SourceExtension.Version = "0.9.0"
				input.SourceExtension.PackageDigest = strings.Repeat("b", 64)
				input.SourceExtension.ActiveVersionID++
			},
		},
		{
			name: "uninstall with historical source", operation: LifecycleMachineUninstall,
			mutate: func(input *LifecycleCoordinatorRunInput) {
				input.SourceExtension.Version = "0.9.0"
				input.SourceExtension.PackageDigest = strings.Repeat("b", 64)
				input.SourceExtension.ActiveVersionID++
			},
		},
		{
			name: "upgrade with target as source", operation: LifecycleMachineUpgrade,
			mutate: func(input *LifecycleCoordinatorRunInput) {
				source := input.Extension
				input.SourceExtension = &source
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := newLifecycleCoordinatorTestRepository()
			input := lifecycleCoordinatorTestInput(test.operation, false)
			test.mutate(&input)
			_, err := NewLifecycleCoordinator(repository, &lifecycleCoordinatorTestRuntime{}, &lifecycleCoordinatorTestHost{}).
				Run(context.Background(), input)
			repository.mu.Lock()
			acquired := repository.acquired
			repository.mu.Unlock()
			if !errors.Is(err, ErrLifecycleCoordinatorInvalid) || acquired {
				t.Fatalf("preflight = %v acquired=%t", err, acquired)
			}
		})
	}
}

func TestLifecycleSourceArtifactRequiresEveryCurrentIdentityField(t *testing.T) {
	target := lifecycleCoordinatorTestInput(LifecycleMachineDisable, false).Extension
	target.Source = SourceUploaded
	target.ActiveVersionID = 12
	target.Manifest.Lifecycle = &ManifestLifecycle{ContractVersion: "demo.plugin.lifecycle@1"}

	mutations := map[string]func(*Extension){
		"version":    func(source *Extension) { source.Version = "0.9.0" },
		"digest":     func(source *Extension) { source.PackageDigest = strings.Repeat("b", 64) },
		"version id": func(source *Extension) { source.ActiveVersionID-- },
		"source":     func(source *Extension) { source.Source = SourceBuiltin },
		"type":       func(source *Extension) { source.Type = TypeTheme },
		"manifest contract": func(source *Extension) {
			source.Manifest.Lifecycle = &ManifestLifecycle{ContractVersion: "demo.plugin.lifecycle@2"}
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			source := target
			mutate(&source)
			if err := validateLifecycleSourceArtifact(LifecycleMachineDisable, target, &source); !errors.Is(err, ErrLifecycleCoordinatorInvalid) {
				t.Fatalf("mismatched source = %#v error=%v", source, err)
			}
		})
	}
}

func TestLifecycleSourceArtifactAllowsCrossContractUpgrade(t *testing.T) {
	input := lifecycleCoordinatorTestInput(LifecycleMachineUpgrade, false)
	if input.SourceExtension == nil ||
		input.SourceExtension.Manifest.Lifecycle.ContractVersion == input.Extension.Manifest.Lifecycle.ContractVersion {
		t.Fatal("test fixture did not retain distinct frozen lifecycle contracts")
	}
	if err := validateLifecycleSourceArtifact(LifecycleMachineUpgrade, input.Extension, input.SourceExtension); err != nil {
		t.Fatalf("cross-contract upgrade source = %v", err)
	}
}

func TestLifecycleSourceArtifactRejectsSameStableVersionWithDifferentDatabaseID(t *testing.T) {
	input := lifecycleCoordinatorTestInput(LifecycleMachineUpgrade, false)
	source := input.Extension
	source.ActiveVersionID = input.Extension.ActiveVersionID + 1
	source.Manifest.Lifecycle = &ManifestLifecycle{ContractVersion: "demo.plugin.lifecycle@0"}
	if err := validateLifecycleSourceArtifact(LifecycleMachineUpgrade, input.Extension, &source); !errors.Is(err, ErrLifecycleCoordinatorInvalid) {
		t.Fatalf("same stable artifact with different database id = %v", err)
	}
}

func TestLifecycleCoordinatorRejectsRuntimeBindingWithoutRevalidationPolicy(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	host := &lifecycleCoordinatorTestHost{results: map[string][]LifecycleCoordinatorGateResult{
		"lifecycle.install.00.host.planned": {{
			TargetBinding: LifecycleRuntimeBinding{RuntimeInstanceID: "unbound-target"},
		}},
	}}
	runtime := &lifecycleCoordinatorTestRuntime{}
	result, err := NewLifecycleCoordinator(repository, runtime, host).
		Run(context.Background(), lifecycleCoordinatorTestInput(LifecycleMachineInstall, false))
	var runErr *LifecycleCoordinatorRunError
	if !errors.As(err, &runErr) || result.Operation.TerminalResult != LifecycleTerminalFailed ||
		!strings.Contains(result.Operation.Error.Message, "requires explicit revalidation") || len(runtime.requestsSnapshot()) != 0 {
		t.Fatalf("unbound runtime result = %#v, %v", result, err)
	}
	machine, decodeErr := decodeLifecycleCoordinatorMachine(result.Operation.Progress)
	if decodeErr != nil || machine.TargetBinding.RuntimeInstanceID != "" {
		t.Fatalf("rejected binding leaked into snapshot: %#v, %v", machine, decodeErr)
	}
}

func TestLifecycleCoordinatorHostGatesReceivePriorDurableActionResults(t *testing.T) {
	tests := []struct {
		name      string
		operation LifecycleMachineOperation
		gateID    string
		behaviors map[LifecycleMachineAction][]lifecycleCoordinatorTestBehavior
		expected  map[LifecycleMachineAction]string
	}{
		{
			name: "install migrating receives plan", operation: LifecycleMachineInstall,
			gateID: "lifecycle.install.02.host.migrating",
			behaviors: map[LifecycleMachineAction][]lifecycleCoordinatorTestBehavior{
				LifecycleMachineInstallPlan: {{result: LifecycleCoordinatorActionResult{
					Status: LifecycleStepSucceeded, ResultDocument: json.RawMessage(`{"plan":"install"}`),
				}}},
			},
			expected: map[LifecycleMachineAction]string{LifecycleMachineInstallPlan: `{"plan":"install"}`},
		},
		{
			name: "upgrade migrating receives plan and before", operation: LifecycleMachineUpgrade,
			gateID: "lifecycle.upgrade.04.host.migrating",
			behaviors: map[LifecycleMachineAction][]lifecycleCoordinatorTestBehavior{
				LifecycleMachineUpgradePlan: {{result: LifecycleCoordinatorActionResult{
					Status: LifecycleStepSucceeded, ResultDocument: json.RawMessage(`{"plan":"upgrade"}`),
				}}},
				LifecycleMachineUpgradeBefore: {{result: LifecycleCoordinatorActionResult{
					Status: LifecycleStepSucceeded, ResultDocument: json.RawMessage(`{"before":"drained"}`),
				}}},
			},
			expected: map[LifecycleMachineAction]string{
				LifecycleMachineUpgradePlan:   `{"plan":"upgrade"}`,
				LifecycleMachineUpgradeBefore: `{"before":"drained"}`,
			},
		},
		{
			name: "uninstall draining receives plan", operation: LifecycleMachineUninstall,
			gateID: "lifecycle.uninstall.02.host.draining",
			behaviors: map[LifecycleMachineAction][]lifecycleCoordinatorTestBehavior{
				LifecycleMachineUninstallPlan: {{result: LifecycleCoordinatorActionResult{
					Status: LifecycleStepSucceeded, ResultDocument: json.RawMessage(`{"plan":"uninstall"}`),
				}}},
			},
			expected: map[LifecycleMachineAction]string{LifecycleMachineUninstallPlan: `{"plan":"uninstall"}`},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := newLifecycleCoordinatorTestRepository()
			host := &lifecycleCoordinatorTestHost{}
			runtime := &lifecycleCoordinatorTestRuntime{behaviors: test.behaviors}
			result, err := NewLifecycleCoordinator(repository, runtime, host).
				Run(context.Background(), lifecycleCoordinatorTestInput(test.operation, false))
			if err != nil || result.Operation.TerminalResult != LifecycleTerminalSucceeded {
				t.Fatalf("operation = %#v, %v", result, err)
			}
			requests := host.requestsSnapshot()
			index := slices.IndexFunc(requests, func(request LifecycleCoordinatorGateRequest) bool {
				return request.StepID == test.gateID
			})
			if index < 0 {
				t.Fatalf("Host gate %q missing from %#v", test.gateID, requests)
			}
			request := requests[index]
			if len(request.PreviousResult) != 0 || len(request.ActionResults) != len(test.expected) {
				t.Fatalf("Host result channels = previous:%s actions:%#v", request.PreviousResult, request.ActionResults)
			}
			for action, expected := range test.expected {
				if string(request.ActionResults[action]) != expected {
					t.Fatalf("action %q result = %s", action, request.ActionResults[action])
				}
			}
		})
	}
}

func TestLifecycleCoordinatorRebuildsActionResultsAcrossCrashReplayAndRevalidation(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	host := &lifecycleCoordinatorTestHost{}
	runtime := &lifecycleCoordinatorTestRuntime{behaviors: map[LifecycleMachineAction][]lifecycleCoordinatorTestBehavior{
		LifecycleMachineInstallPlan: {{
			result: LifecycleCoordinatorActionResult{
				Status: LifecycleStepSucceeded, ResultDocument: json.RawMessage(`{"plan":"durable"}`),
			},
			after: repository.failNextTransition,
		}},
	}}
	coordinator := NewLifecycleCoordinator(repository, runtime, host)
	input := lifecycleCoordinatorTestInput(LifecycleMachineInstall, false)
	if _, err := coordinator.Run(context.Background(), input); !errors.Is(err, errLifecycleCoordinatorTestCrash) {
		t.Fatalf("action terminal crash = %v", err)
	}
	result, err := coordinator.Run(context.Background(), input)
	if err != nil || result.Operation.TerminalResult != LifecycleTerminalSucceeded ||
		countLifecycleCoordinatorAction(runtime.actionNames(), LifecycleMachineInstallPlan) != 1 {
		t.Fatalf("replayed operation = %#v, %v actions=%#v", result, err, runtime.actionNames())
	}

	requests := host.requestsSnapshot()
	planned := make([]LifecycleCoordinatorGateRequest, 0, 2)
	for _, request := range requests {
		if request.StepID == "lifecycle.install.00.host.planned" {
			planned = append(planned, request)
		}
	}
	if len(planned) != 2 || planned[0].Revalidation || !planned[1].Revalidation ||
		len(planned[0].ActionResults) != 0 || len(planned[1].ActionResults) != 0 || len(planned[1].PreviousResult) == 0 {
		t.Fatalf("planned Host result separation = %#v", planned)
	}
	migrating := slices.IndexFunc(requests, func(request LifecycleCoordinatorGateRequest) bool {
		return request.StepID == "lifecycle.install.02.host.migrating"
	})
	if migrating < 0 || string(requests[migrating].ActionResults[LifecycleMachineInstallPlan]) != `{"plan":"durable"}` {
		t.Fatalf("durable action result after replay = %#v", requests)
	}
}

func TestLifecycleCoordinatorHostActionResultsFailClosed(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{name: "missing"},
		{name: "planned", status: LifecycleStepPlanned},
		{name: "failed", status: LifecycleStepFailed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := newLifecycleCoordinatorTestRepository()
			input := lifecycleCoordinatorTestInput(LifecycleMachineInstall, false)
			acquired, err := repository.AcquireOperation(context.Background(), input.Acquire)
			if err != nil {
				t.Fatal(err)
			}
			if test.status != "" {
				begin, err := repository.BeginStepAttempt(context.Background(), BeginLifecycleStepAttemptInput{
					OperationID: acquired.Operation.ID, StepID: "lifecycle.install.01.install.plan",
					LifecycleAction: string(LifecycleMachineInstallPlan), PlanVersion: input.Acquire.PlanVersion,
				})
				if err != nil {
					t.Fatal(err)
				}
				if test.status == LifecycleStepFailed {
					if _, err := repository.CompleteStepAttempt(context.Background(), CompleteLifecycleStepAttemptInput{
						AttemptID: begin.Attempt.ID, Status: LifecycleStepFailed,
					}); err != nil {
						t.Fatal(err)
					}
				}
			}
			_, err = NewLifecycleCoordinator(repository, nil, nil).lifecycleHostActionResults(
				context.Background(), acquired.Operation.ID, LifecycleMachineInstall, 2,
			)
			if !errors.Is(err, ErrLifecycleCoordinatorInvalid) {
				t.Fatalf("action result state %q = %v", test.status, err)
			}
		})
	}
}

func TestLifecycleCoordinatorHostCannotMutateDurableActionResults(t *testing.T) {
	repository := newLifecycleCoordinatorTestRepository()
	runtime := &lifecycleCoordinatorTestRuntime{behaviors: map[LifecycleMachineAction][]lifecycleCoordinatorTestBehavior{
		LifecycleMachineInstallPlan: {{result: LifecycleCoordinatorActionResult{
			Status: LifecycleStepSucceeded, ResultDocument: json.RawMessage(`{"plan":"immutable"}`),
		}}},
	}}
	result, err := NewLifecycleCoordinator(repository, runtime, lifecycleCoordinatorMutatingActionResultsHost{}).
		Run(context.Background(), lifecycleCoordinatorTestInput(LifecycleMachineInstall, false))
	if err != nil || result.Operation.TerminalResult != LifecycleTerminalSucceeded {
		t.Fatalf("operation = %#v, %v", result, err)
	}
	attempt, err := repository.LatestStepAttempt(context.Background(), result.Operation.ID, "lifecycle.install.01.install.plan")
	if err != nil || string(attempt.ResultDocument) != `{"plan":"immutable"}` {
		t.Fatalf("Host mutated ledger result = %#v, %v", attempt, err)
	}
}

type lifecycleCoordinatorMutatingActionResultsHost struct{}

func (lifecycleCoordinatorMutatingActionResultsHost) RunLifecycleHostGate(
	_ context.Context,
	request LifecycleCoordinatorGateRequest,
) (LifecycleCoordinatorGateResult, error) {
	if request.StepID == "lifecycle.install.02.host.migrating" {
		if raw := request.ActionResults[LifecycleMachineInstallPlan]; len(raw) > 0 {
			raw[0] = '['
		}
		request.ActionResults[LifecycleMachineInstallPlan] = json.RawMessage(`{"plan":"forged"}`)
		request.ActionResults[LifecycleMachineDisableAction] = json.RawMessage(`{"not":"allowlisted"}`)
	}
	return lifecycleCoordinatorTestPlannedGateResult(request), nil
}

func TestLifecycleRecommendedPathsFinalizeDestructiveOperationsAndActivateUpgrade(t *testing.T) {
	tests := []struct {
		operation LifecycleMachineOperation
		lastState LifecycleMachineState
	}{
		{LifecycleMachineDisable, LifecycleMachineDraining},
		{LifecycleMachineUninstall, LifecycleMachineUninstalling},
	}
	for _, test := range tests {
		path, _ := RecommendedLifecyclePath(test.operation)
		last := path[len(path)-1]
		if last.State != test.lastState || last.Action != "" || path[len(path)-2].Action == "" {
			t.Fatalf("%s finalization path = %#v", test.operation, path)
		}
	}

	upgrade, _ := RecommendedLifecyclePath(LifecycleMachineUpgrade)
	after := lifecycleTestActionIndex(upgrade, LifecycleMachineUpgradeAfter)
	if after < 1 || upgrade[after-1] != (LifecycleRecommendedStep{State: LifecycleMachineEnabled}) ||
		after+1 >= len(upgrade) || upgrade[after+1] != (LifecycleRecommendedStep{State: LifecycleMachineEnabled}) ||
		!strings.HasSuffix(lifecycleCoordinatorStepID(LifecycleMachineUpgrade, after, upgrade[after].State, upgrade[after].Action), ".08.upgrade.after") {
		t.Fatalf("upgrade activation/finalization path = %#v", upgrade)
	}
}

func decodeLifecycleCoordinatorV1TestMachine(t *testing.T, machine LifecycleStateMachine) LifecycleStateMachine {
	t.Helper()
	value, err := json.Marshal(lifecycleCoordinatorSnapshot{Schema: lifecycleCoordinatorSnapshotSchemaV1, Machine: machine})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeLifecycleCoordinatorMachine(value)
	if err != nil {
		t.Fatalf("decode v1 snapshot: %v", err)
	}
	return decoded
}

func seedLifecycleCoordinatorV1OpenOperation(
	t *testing.T,
	repository *lifecycleCoordinatorTestRepository,
	input LifecycleCoordinatorRunInput,
	machine LifecycleStateMachine,
) {
	t.Helper()
	acquired, err := repository.AcquireOperation(context.Background(), input.Acquire)
	if err != nil {
		t.Fatal(err)
	}
	if machine.TerminalResult == "" {
		path, _ := RecommendedLifecyclePath(machine.Operation)
		for index, step := range path {
			if index > machine.Position || (index == machine.Position && !machine.StepComplete) {
				break
			}
			if step.Action == "" {
				continue
			}
			stepID := lifecycleCoordinatorStepID(machine.Operation, index, step.State, step.Action)
			begin, beginErr := repository.BeginStepAttempt(context.Background(), BeginLifecycleStepAttemptInput{
				OperationID: acquired.Operation.ID, StepID: stepID, LifecycleAction: string(step.Action),
				PlanVersion: input.Acquire.PlanVersion,
			})
			if beginErr != nil {
				t.Fatal(beginErr)
			}
			if _, completeErr := repository.CompleteStepAttempt(context.Background(), CompleteLifecycleStepAttemptInput{
				AttemptID: begin.Attempt.ID, Status: LifecycleStepSucceeded,
			}); completeErr != nil {
				t.Fatal(completeErr)
			}
		}
	}
	value, err := json.Marshal(lifecycleCoordinatorSnapshot{Schema: lifecycleCoordinatorSnapshotSchemaV1, Machine: machine})
	if err != nil {
		t.Fatal(err)
	}
	repository.mu.Lock()
	repository.operation.State = string(machine.State)
	repository.operation.CurrentStepID = lifecycleCoordinatorStepID(machine.Operation, machine.Position, machine.State, machine.Action)
	repository.operation.Progress = value
	repository.mu.Unlock()
}
