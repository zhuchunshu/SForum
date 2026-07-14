package extensions

import (
	"errors"
	"slices"
	"testing"
)

func TestLifecycleRecommendedPathsCoverAllOperationsStatesAndActions(t *testing.T) {
	operations := allLifecycleMachineOperations()
	states := map[LifecycleMachineState]bool{
		LifecycleMachineFailed: true, LifecycleMachineRecovery: true,
	}
	actions := map[LifecycleMachineAction]int{}
	for _, operation := range operations {
		path, err := RecommendedLifecyclePath(operation)
		if err != nil {
			t.Fatal(err)
		}
		if len(path) < 2 || path[0] != (LifecycleRecommendedStep{State: LifecycleMachinePlanned}) {
			t.Fatalf("%s path = %#v", operation, path)
		}
		for _, step := range path {
			states[step.State] = true
			if step.Action != "" {
				actions[step.Action]++
			}
		}
		path[0].State = LifecycleMachineFailed
		fresh, _ := RecommendedLifecyclePath(operation)
		if fresh[0].State != LifecycleMachinePlanned {
			t.Fatal("recommended path leaked mutable storage")
		}
	}
	if len(states) != 10 {
		t.Fatalf("states = %#v", states)
	}
	for _, action := range allLifecycleMachineActions() {
		if actions[action] == 0 {
			t.Fatalf("action %q is absent from every recommended path", action)
		}
	}
}

func TestLifecycleStateMachineCompletesEveryRecommendedPath(t *testing.T) {
	for _, operation := range allLifecycleMachineOperations() {
		t.Run(string(operation), func(t *testing.T) {
			machine, err := NewLifecycleStateMachine(operation, false)
			if err != nil {
				t.Fatal(err)
			}
			path, _ := RecommendedLifecyclePath(operation)
			machine = applyLifecycleTestTransition(t, machine, LifecycleStateTransition{
				State: machine.State, Action: machine.Action, CompleteStep: true,
				Progress: LifecycleProgressCursor{TotalUnits: uint64(len(path) - 1)},
			})
			for index := 1; index < len(path); index++ {
				step := path[index]
				beginProgress := machine.Progress
				machine = applyLifecycleTestTransition(t, machine, LifecycleStateTransition{
					State: step.State, Action: step.Action, Progress: beginProgress,
				})
				progress := LifecycleProgressCursor{
					CompletedUnits: uint64(index), TotalUnits: uint64(len(path) - 1),
					Checkpoint: string(operation) + "-checkpoint-" + string(rune('a'+index)), CheckpointSequence: uint64(index),
				}
				machine = applyLifecycleTestTransition(t, machine, LifecycleStateTransition{
					State: step.State, Action: step.Action, CompleteStep: true, Progress: progress,
				})
			}
			machine = applyLifecycleTestTransition(t, machine, LifecycleStateTransition{
				State: machine.State, Action: machine.Action,
				TerminalResult: LifecycleMachineSucceeded, Progress: machine.Progress,
			})
			if machine.TerminalResult != LifecycleMachineSucceeded {
				t.Fatalf("terminal machine = %#v", machine)
			}
		})
	}
}

func TestLifecycleRecommendedPathsCompleteHostSafetyGatesBeforePluginActions(t *testing.T) {
	checks := []struct {
		operation LifecycleMachineOperation
		action    LifecycleMachineAction
		state     LifecycleMachineState
	}{
		{LifecycleMachineInstall, LifecycleMachineInstallAction, LifecycleMachineMigrating},
		{LifecycleMachineInstall, LifecycleMachineEnableAction, LifecycleMachineStarting},
		{LifecycleMachineEnable, LifecycleMachineEnableAction, LifecycleMachineStarting},
		{LifecycleMachineDisable, LifecycleMachineDisableAction, LifecycleMachineDraining},
		{LifecycleMachineUpgrade, LifecycleMachineUpgradeBefore, LifecycleMachineDraining},
		{LifecycleMachineRollback, LifecycleMachineRollbackAction, LifecycleMachineStarting},
		{LifecycleMachineUninstall, LifecycleMachineUninstallStep, LifecycleMachineUninstalling},
	}
	for _, check := range checks {
		path, _ := RecommendedLifecyclePath(check.operation)
		index := lifecycleTestActionIndex(path, check.action)
		if index <= 0 || path[index].State != check.state || path[index-1].State != check.state || path[index-1].Action != "" {
			t.Fatalf("%s action %s is not behind its %s Host gate: %#v", check.operation, check.action, check.state, path)
		}
	}
}

func TestLifecycleStateMachineRejectsEveryNonRecommendedInitialGate(t *testing.T) {
	states := allLifecycleMachineStates()
	actions := append(allLifecycleMachineActions(), "")
	for _, operation := range allLifecycleMachineOperations() {
		machine, _ := NewLifecycleStateMachine(operation, false)
		path, _ := RecommendedLifecyclePath(operation)
		for _, state := range states {
			for _, action := range actions {
				transition := LifecycleStateTransition{State: state, Action: action, Progress: machine.Progress}
				err := ValidateLifecycleTransition(machine, transition)
				allowed := state == path[0].State && action == path[0].Action
				if allowed && err != nil {
					t.Fatalf("%s rejected its initial Host gate %s/%s: %v", operation, state, action, err)
				}
				if !allowed && !errors.Is(err, ErrLifecycleStateTransitionDenied) {
					t.Fatalf("%s advanced before initial Host gate completion to %s/%s: %v", operation, state, action, err)
				}
			}
		}
		machine = applyLifecycleTestTransition(t, machine, LifecycleStateTransition{
			State: machine.State, Action: machine.Action, CompleteStep: true, Progress: machine.Progress,
		})
		if err := ValidateLifecycleTransition(machine, LifecycleStateTransition{
			State: path[1].State, Action: path[1].Action, Progress: machine.Progress,
		}); err != nil {
			t.Fatalf("%s rejected first post-Host gate %s/%s: %v", operation, path[1].State, path[1].Action, err)
		}
	}
}

func TestLifecycleStateMachineExhaustivelyRejectsGateJumps(t *testing.T) {
	states := allLifecycleMachineStates()
	actions := append(allLifecycleMachineActions(), "")
	for _, operation := range allLifecycleMachineOperations() {
		machine, _ := NewLifecycleStateMachine(operation, false)
		path, _ := RecommendedLifecyclePath(operation)
		for position := 0; position < len(path); position++ {
			if !machine.StepComplete {
				machine = applyLifecycleTestTransition(t, machine, LifecycleStateTransition{
					State: machine.State, Action: machine.Action, CompleteStep: true, Progress: machine.Progress,
				})
			}
			if machine.Position != position || !machine.StepComplete {
				t.Fatalf("%s position %d snapshot = %#v", operation, position, machine)
			}
			for _, state := range states {
				for _, action := range actions {
					transition := LifecycleStateTransition{State: state, Action: action, Progress: machine.Progress}
					err := ValidateLifecycleTransition(machine, transition)
					allowed := position+1 < len(path) && state == path[position+1].State && action == path[position+1].Action
					if allowed && err != nil {
						t.Fatalf("%s position %d rejected next %s/%s: %v", operation, position, state, action, err)
					}
					if !allowed && !errors.Is(err, ErrLifecycleStateTransitionDenied) {
						t.Fatalf("%s position %d accepted jump %s/%s: %v", operation, position, state, action, err)
					}
				}
			}
			if position+1 >= len(path) {
				continue
			}
			nextStep := path[position+1]
			machine = applyLifecycleTestTransition(t, machine, LifecycleStateTransition{
				State: nextStep.State, Action: nextStep.Action, Progress: machine.Progress,
			})
			for _, state := range states {
				for _, action := range actions {
					err := ValidateLifecycleTransition(machine, LifecycleStateTransition{
						State: state, Action: action, Progress: machine.Progress,
					})
					allowed := state == machine.State && action == machine.Action
					if allowed && err != nil {
						t.Fatalf("%s running position %d rejected self %s/%s: %v", operation, position+1, state, action, err)
					}
					if !allowed && !errors.Is(err, ErrLifecycleStateTransitionDenied) {
						t.Fatalf("%s running position %d accepted jump %s/%s: %v", operation, position+1, state, action, err)
					}
				}
			}
			machine = applyLifecycleTestTransition(t, machine, LifecycleStateTransition{
				State: machine.State, Action: machine.Action, CompleteStep: true, Progress: machine.Progress,
			})
		}
	}
}

func TestLifecycleRetryHasOneRecoveryPathAndCannotJumpBoundary(t *testing.T) {
	machine, _ := NewLifecycleStateMachine(LifecycleMachineUpgrade, false)
	path, _ := RecommendedLifecyclePath(machine.Operation)
	starting := lifecycleTestStepIndex(path, LifecycleMachineStarting, "")
	for index := 0; index < starting; index++ {
		machine = beginAndCompleteLifecycleTestGate(t, machine, path[index])
	}
	// Enter starting but fail before completing its host-owned start gate.
	machine = applyLifecycleTestTransition(t, machine, LifecycleStateTransition{
		State: path[starting].State, Action: path[starting].Action, Progress: machine.Progress,
	})
	machine = applyLifecycleTestTransition(t, machine, LifecycleStateTransition{
		State: LifecycleMachineFailed, Action: machine.Action,
		TerminalResult: LifecycleMachineFailedRun, Progress: machine.Progress,
	})
	for _, state := range allLifecycleMachineStates() {
		transition := LifecycleStateTransition{State: state, Progress: machine.Progress}
		if state == LifecycleMachineRecovery {
			transition.Retry = true
		}
		err := ValidateLifecycleTransition(machine, transition)
		if state == LifecycleMachineRecovery && err != nil {
			t.Fatalf("retry to recovery failed: %v", err)
		}
		if state != LifecycleMachineRecovery && !errors.Is(err, ErrLifecycleStateTransitionDenied) {
			t.Fatalf("terminal operation revived into %s: %v", state, err)
		}
	}
	machine = applyLifecycleTestTransition(t, machine, LifecycleStateTransition{
		State: LifecycleMachineRecovery, Retry: true, Progress: machine.Progress,
	})
	for index, step := range path {
		transition := LifecycleStateTransition{State: step.State, Action: step.Action, Progress: machine.Progress}
		err := ValidateLifecycleTransition(machine, transition)
		if index == starting && err != nil {
			t.Fatalf("exact recovery gate failed: %v", err)
		}
		if index != starting && !errors.Is(err, ErrLifecycleStateTransitionDenied) {
			t.Fatalf("recovery jumped to gate %d: %v", index, err)
		}
	}
}

func TestLifecycleTerminalSuccessAndSkipCannotBeRevived(t *testing.T) {
	skipped, _ := NewLifecycleStateMachine(LifecycleMachineDisable, false)
	skipped = applyLifecycleTestTransition(t, skipped, LifecycleStateTransition{
		State: skipped.State, Action: skipped.Action, CompleteStep: true, Progress: skipped.Progress,
	})
	skipped = applyLifecycleTestTransition(t, skipped, LifecycleStateTransition{
		State: LifecycleMachinePlanned, TerminalResult: LifecycleMachineSkipped, Progress: skipped.Progress,
	})
	if err := ValidateLifecycleTransition(skipped, LifecycleStateTransition{
		State: LifecycleMachineRecovery, Retry: true, Progress: skipped.Progress,
	}); !errors.Is(err, ErrLifecycleStateTransitionDenied) {
		t.Fatalf("skipped operation revived: %v", err)
	}

	succeeded, _ := NewLifecycleStateMachine(LifecycleMachineDisable, false)
	path, _ := RecommendedLifecyclePath(succeeded.Operation)
	for index := 0; index < len(path); index++ {
		succeeded = beginAndCompleteLifecycleTestGate(t, succeeded, path[index])
	}
	succeeded = applyLifecycleTestTransition(t, succeeded, LifecycleStateTransition{
		State: succeeded.State, Action: succeeded.Action,
		TerminalResult: LifecycleMachineSucceeded, Progress: succeeded.Progress,
	})
	if err := ValidateLifecycleTransition(succeeded, LifecycleStateTransition{
		State: LifecycleMachineRecovery, Retry: true, Progress: succeeded.Progress,
	}); !errors.Is(err, ErrLifecycleStateTransitionDenied) {
		t.Fatalf("successful operation revived: %v", err)
	}
}

func TestLifecyclePlannedHostSideEffectsCannotBecomeSkipped(t *testing.T) {
	input := lifecycleCoordinatorTestInput(LifecycleMachineInstall, false)
	machine, _ := NewLifecycleStateMachine(LifecycleMachineInstall, false)
	if err := hydrateLifecycleCoordinatorBindings(&machine, input.Extension, input.SourceExtension); err != nil {
		t.Fatal(err)
	}
	var err error
	machine, err = applyLifecycleHostGateResult(machine, "lifecycle.install.00.host.planned", 0, LifecycleCoordinatorGateResult{
		TargetBinding:      LifecycleRuntimeBinding{RuntimeInstanceID: "target-instance"},
		RevalidationPolicy: LifecycleGateRevalidationRequired,
	})
	if err != nil {
		t.Fatal(err)
	}
	machine = applyLifecycleTestTransition(t, machine, LifecycleStateTransition{
		State: machine.State, Action: machine.Action, CompleteStep: true, Progress: machine.Progress,
	})
	if err := ValidateLifecycleTransition(machine, LifecycleStateTransition{
		State: LifecycleMachinePlanned, TerminalResult: LifecycleMachineSkipped, Progress: machine.Progress,
	}); !errors.Is(err, ErrLifecycleStateTransitionDenied) {
		t.Fatalf("planned Host side effects became skipped: %v", err)
	}

	// Old snapshots did not carry the side-effect bit. Their already-closed
	// skipped state remains readable for compatibility.
	historical, _ := NewLifecycleStateMachine(LifecycleMachineInstall, false)
	historical.StepComplete = true
	historical.TerminalResult = LifecycleMachineSkipped
	if _, err := validateLifecycleMachine(historical); err != nil {
		t.Fatalf("historical skipped snapshot = %v", err)
	}
}

func TestLifecycleSkipAndForcedUninstallPolicies(t *testing.T) {
	upgradePath, _ := RecommendedLifecyclePath(LifecycleMachineUpgrade)
	upgrade := failedLifecycleMachineAtGate(t, LifecycleMachineUpgrade, lifecycleTestActionIndex(upgradePath, LifecycleMachineUpgradeAfter), false)
	upgrade = applyLifecycleTestTransition(t, upgrade, LifecycleStateTransition{
		State: LifecycleMachineRecovery, Retry: true, Progress: upgrade.Progress,
	})
	upgrade = applyLifecycleTestTransition(t, upgrade, LifecycleStateTransition{
		State: LifecycleMachineEnabled, Action: LifecycleMachineUpgradeAfter,
		SkipStep: true, SkipReason: "post-upgrade cleanup accepted", Progress: upgrade.Progress,
	})
	if !upgrade.StepComplete {
		t.Fatal("upgrade.after skip did not complete its exact gate")
	}

	uninstallPath, _ := RecommendedLifecyclePath(LifecycleMachineUninstall)
	uninstall := failedLifecycleMachineAtGate(t, LifecycleMachineUninstall, lifecycleTestActionIndex(uninstallPath, LifecycleMachineUninstallAfter), false)
	uninstall = applyLifecycleTestTransition(t, uninstall, LifecycleStateTransition{
		State: LifecycleMachineRecovery, Retry: true, Progress: uninstall.Progress,
	})
	withoutForce := LifecycleStateTransition{
		State: LifecycleMachineUninstalling, Action: LifecycleMachineUninstallAfter,
		SkipStep: true, SkipReason: "external cleanup remains", Progress: uninstall.Progress,
	}
	if err := ValidateLifecycleTransition(uninstall, withoutForce); !errors.Is(err, ErrLifecycleStateTransitionDenied) {
		t.Fatalf("unforced cleanup skip = %v", err)
	}
	withoutForce.EscalateForced = true
	uninstall = applyLifecycleTestTransition(t, uninstall, withoutForce)
	if !uninstall.Forced || !uninstall.StepComplete {
		t.Fatalf("forced uninstall = %#v", uninstall)
	}

	if _, err := NewLifecycleStateMachine(LifecycleMachineEnable, true); !errors.Is(err, ErrLifecycleStateMachineInvalid) {
		t.Fatalf("forced enable = %v", err)
	}
	installPlan := failedLifecycleMachineAtGate(t, LifecycleMachineInstall, 1, false)
	installPlan = applyLifecycleTestTransition(t, installPlan, LifecycleStateTransition{
		State: LifecycleMachineRecovery, Retry: true, Progress: installPlan.Progress,
	})
	if err := ValidateLifecycleTransition(installPlan, LifecycleStateTransition{
		State: LifecycleMachinePlanned, Action: LifecycleMachineInstallPlan,
		SkipStep: true, SkipReason: "skip disclosure", Progress: installPlan.Progress,
	}); !errors.Is(err, ErrLifecycleStateTransitionDenied) {
		t.Fatalf("safety plan skip = %v", err)
	}
}

func TestLifecycleProgressAndCheckpointAreMonotonic(t *testing.T) {
	machine, _ := NewLifecycleStateMachine(LifecycleMachineEnable, false)
	machine.Progress = LifecycleProgressCursor{
		CompletedUnits: 2, TotalUnits: 5, Checkpoint: "checkpoint-2", CheckpointSequence: 2,
	}
	machine = applyLifecycleTestTransition(t, machine, LifecycleStateTransition{
		State: machine.State, Action: machine.Action, CompleteStep: true, Progress: machine.Progress,
	})
	invalid := []LifecycleProgressCursor{
		{CompletedUnits: 1, TotalUnits: 5, Checkpoint: "checkpoint-2", CheckpointSequence: 2},
		{CompletedUnits: 2, TotalUnits: 4, Checkpoint: "checkpoint-2", CheckpointSequence: 2},
		{CompletedUnits: 6, TotalUnits: 5, Checkpoint: "checkpoint-2", CheckpointSequence: 2},
		{CompletedUnits: 2, TotalUnits: 5},
		{CompletedUnits: 2, TotalUnits: 5, Checkpoint: "checkpoint-1", CheckpointSequence: 1},
		{CompletedUnits: 2, TotalUnits: 5, Checkpoint: "changed", CheckpointSequence: 2},
		{CompletedUnits: 2, TotalUnits: 5, Checkpoint: "checkpoint-2", CheckpointSequence: 3},
		{CompletedUnits: 2, TotalUnits: 5, CheckpointSequence: 3},
	}
	for _, progress := range invalid {
		err := ValidateLifecycleTransition(machine, LifecycleStateTransition{
			State: LifecycleMachineStarting, Action: LifecycleMachineEnableAction, Progress: progress,
		})
		if !errors.Is(err, ErrLifecycleStateProgressRegression) {
			t.Fatalf("progress %#v = %v", progress, err)
		}
	}
	valid := LifecycleProgressCursor{
		CompletedUnits: 3, TotalUnits: 5, Checkpoint: "checkpoint-3", CheckpointSequence: 3,
	}
	path, _ := RecommendedLifecyclePath(machine.Operation)
	if err := ValidateLifecycleTransition(machine, LifecycleStateTransition{
		State: path[1].State, Action: path[1].Action, Progress: valid,
	}); err != nil {
		t.Fatalf("valid progress = %v", err)
	}
}

func TestApplyLifecycleTransitionDoesNotMutateInput(t *testing.T) {
	current, _ := NewLifecycleStateMachine(LifecycleMachineEnable, false)
	current = applyLifecycleTestTransition(t, current, LifecycleStateTransition{
		State: current.State, Action: current.Action, CompleteStep: true, Progress: current.Progress,
	})
	want := current
	path, _ := RecommendedLifecyclePath(current.Operation)
	_, err := ApplyLifecycleTransition(current, LifecycleStateTransition{
		State: path[1].State, Action: path[1].Action, Progress: current.Progress,
	})
	if err != nil {
		t.Fatal(err)
	}
	if current != want {
		t.Fatalf("input mutated: got %#v want %#v", current, want)
	}
}

func beginAndCompleteLifecycleTestGate(t *testing.T, machine LifecycleStateMachine, step LifecycleRecommendedStep) LifecycleStateMachine {
	t.Helper()
	machine = applyLifecycleTestTransition(t, machine, LifecycleStateTransition{
		State: step.State, Action: step.Action, Progress: machine.Progress,
	})
	return applyLifecycleTestTransition(t, machine, LifecycleStateTransition{
		State: step.State, Action: step.Action, CompleteStep: true, Progress: machine.Progress,
	})
}

func failedLifecycleMachineAtFinalGate(t *testing.T, operation LifecycleMachineOperation, forced bool) LifecycleStateMachine {
	t.Helper()
	path, _ := RecommendedLifecyclePath(operation)
	return failedLifecycleMachineAtGate(t, operation, len(path)-1, forced)
}

func failedLifecycleMachineAtGate(t *testing.T, operation LifecycleMachineOperation, target int, forced bool) LifecycleStateMachine {
	t.Helper()
	machine, err := NewLifecycleStateMachine(operation, forced)
	if err != nil {
		t.Fatal(err)
	}
	path, _ := RecommendedLifecyclePath(operation)
	for index := 0; index < target; index++ {
		machine = beginAndCompleteLifecycleTestGate(t, machine, path[index])
	}
	machine = applyLifecycleTestTransition(t, machine, LifecycleStateTransition{
		State: path[target].State, Action: path[target].Action, Progress: machine.Progress,
	})
	return applyLifecycleTestTransition(t, machine, LifecycleStateTransition{
		State: LifecycleMachineFailed, Action: machine.Action,
		TerminalResult: LifecycleMachineFailedRun, Progress: machine.Progress,
	})
}

func applyLifecycleTestTransition(t *testing.T, current LifecycleStateMachine, transition LifecycleStateTransition) LifecycleStateMachine {
	t.Helper()
	next, err := ApplyLifecycleTransition(current, transition)
	if err != nil {
		t.Fatalf("apply %#v to %#v: %v", transition, current, err)
	}
	return next
}

func allLifecycleMachineOperations() []LifecycleMachineOperation {
	return []LifecycleMachineOperation{
		LifecycleMachineInstall, LifecycleMachineEnable, LifecycleMachineDisable,
		LifecycleMachineUpgrade, LifecycleMachineRollback, LifecycleMachineUninstall,
	}
}

func allLifecycleMachineStates() []LifecycleMachineState {
	return []LifecycleMachineState{
		LifecycleMachinePlanned, LifecycleMachineMigrating, LifecycleMachineStarting,
		LifecycleMachineHealthy, LifecycleMachineRegistering, LifecycleMachineEnabled,
		LifecycleMachineDraining, LifecycleMachineUninstalling, LifecycleMachineFailed,
		LifecycleMachineRecovery,
	}
}

func allLifecycleMachineActions() []LifecycleMachineAction {
	actions := []LifecycleMachineAction{
		LifecycleMachineInstallPlan, LifecycleMachineInstallAction, LifecycleMachineEnableAction,
		LifecycleMachineDisableAction, LifecycleMachineUpgradePlan, LifecycleMachineUpgradeBefore,
		LifecycleMachineUpgradeAfter, LifecycleMachineRollbackAction, LifecycleMachineUninstallPlan,
		LifecycleMachineUninstallStep, LifecycleMachineUninstallAfter,
	}
	if slices.Contains(actions, LifecycleMachineAction("")) {
		panic("empty lifecycle action")
	}
	return actions
}

func lifecycleTestActionIndex(path []LifecycleRecommendedStep, action LifecycleMachineAction) int {
	for index, step := range path {
		if step.Action == action {
			return index
		}
	}
	return -1
}

func lifecycleTestStepIndex(path []LifecycleRecommendedStep, state LifecycleMachineState, action LifecycleMachineAction) int {
	for index, step := range path {
		if step.State == state && step.Action == action {
			return index
		}
	}
	return -1
}
