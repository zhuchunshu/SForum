package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

func (c *LifecycleCoordinator) completeSuccess(ctx context.Context, operation LifecycleOperation, machine LifecycleStateMachine) (LifecycleOperation, LifecycleStateMachine, error) {
	transition := LifecycleStateTransition{
		State: machine.State, Action: machine.Action, TerminalResult: LifecycleMachineSucceeded, Progress: machine.Progress,
	}
	succeeded, err := ApplyLifecycleTransition(machine, transition)
	if err != nil {
		return operation, machine, err
	}
	operation, err = c.persistMachine(ctx, operation, succeeded, operation.CurrentStepID)
	if err != nil {
		return operation, machine, err
	}
	completed, err := c.repository.CompleteOperation(ctx, CompleteLifecycleOperationInput{
		OperationID: operation.ID, ExpectedRevision: operation.Revision, ExpectedState: operation.State,
		State: string(succeeded.State), TerminalResult: LifecycleTerminalSucceeded, AuditEventID: operation.AuditEventID,
	})
	return completed, succeeded, err
}

func (c *LifecycleCoordinator) persistMachine(ctx context.Context, operation LifecycleOperation, machine LifecycleStateMachine, stepID string) (LifecycleOperation, error) {
	progress, checkpoint, err := encodeLifecycleCoordinatorMachine(machine, stepID)
	if err != nil {
		return operation, err
	}
	return c.repository.TransitionOperation(ctx, TransitionLifecycleOperationInput{
		OperationID: operation.ID, ExpectedRevision: operation.Revision, ExpectedState: operation.State,
		State: string(machine.State), CurrentStepID: stepID, Checkpoint: checkpoint, Progress: progress,
	})
}

type lifecycleCoordinatorSnapshot struct {
	Schema  string                `json:"schema"`
	Machine LifecycleStateMachine `json:"machine"`
}

type lifecycleCoordinatorCheckpointSnapshot struct {
	StepID     string `json:"stepId,omitempty"`
	Checkpoint string `json:"checkpoint,omitempty"`
	Sequence   uint64 `json:"sequence,omitempty"`
}

func encodeLifecycleCoordinatorMachine(machine LifecycleStateMachine, stepID string) (json.RawMessage, json.RawMessage, error) {
	progress, err := json.Marshal(lifecycleCoordinatorSnapshot{Schema: lifecycleCoordinatorSnapshotSchema, Machine: machine})
	if err != nil {
		return nil, nil, err
	}
	checkpoint, err := json.Marshal(lifecycleCoordinatorCheckpointSnapshot{
		StepID: stepID, Checkpoint: machine.Progress.Checkpoint, Sequence: machine.Progress.CheckpointSequence,
	})
	return progress, checkpoint, err
}

func decodeLifecycleCoordinatorMachine(value json.RawMessage) (LifecycleStateMachine, error) {
	var snapshot lifecycleCoordinatorSnapshot
	if err := json.Unmarshal(value, &snapshot); err != nil {
		return LifecycleStateMachine{}, fmt.Errorf("%w: decode coordinator snapshot: %v", ErrLifecycleCoordinatorInvalid, err)
	}
	switch snapshot.Schema {
	case lifecycleCoordinatorSnapshotSchemaV1:
		snapshot.Machine = migrateLifecycleCoordinatorV1(snapshot.Machine)
	case lifecycleCoordinatorSnapshotSchema:
		// Current snapshot.
	default:
		return LifecycleStateMachine{}, fmt.Errorf("%w: unsupported coordinator snapshot %q", ErrLifecycleCoordinatorInvalid, snapshot.Schema)
	}
	if _, err := validateLifecycleMachine(snapshot.Machine); err != nil {
		return LifecycleStateMachine{}, err
	}
	return snapshot.Machine, nil
}

// migrateLifecycleCoordinatorV1 preserves completed historical operations and
// moves only open snapshots onto the additive Host-finalization path.
func migrateLifecycleCoordinatorV1(machine LifecycleStateMachine) LifecycleStateMachine {
	path := lifecycleRecommendedPaths[machine.Operation]
	if len(path) == 0 {
		return machine
	}
	machine.Progress.TotalUnits = uint64(len(path) - 1)

	legacyTerminal := machine.TerminalResult == LifecycleMachineSucceeded
	switch machine.Operation {
	case LifecycleMachineDisable:
		if legacyTerminal {
			machine.Position = len(path) - 1
			machine.RecoveryPosition = machine.Position
			machine.State = path[machine.Position].State
			machine.Action = path[machine.Position].Action
			machine.StepComplete = true
			machine.Progress.CompletedUnits = machine.Progress.TotalUnits
		}
	case LifecycleMachineUpgrade:
		if legacyTerminal {
			machine.Position = len(path) - 1
			machine.RecoveryPosition = machine.Position
			machine.State = path[machine.Position].State
			machine.Action = path[machine.Position].Action
			machine.StepComplete = true
			machine.Progress.CompletedUnits = machine.Progress.TotalUnits
		} else if machine.Position == 8 {
			// V1 position 8 was upgrade.after. It retains its historical step id
			// but lives after the new atomic activation gate.
			machine.Position = 9
			machine.RecoveryPosition = 9
			if machine.StepComplete && machine.State != LifecycleMachineFailed && machine.State != LifecycleMachineRecovery {
				machine.Progress.CompletedUnits = 9
			}
		}
	case LifecycleMachineUninstall:
		if legacyTerminal {
			machine.Position = len(path) - 1
			machine.RecoveryPosition = machine.Position
			machine.State = path[machine.Position].State
			machine.Action = path[machine.Position].Action
			machine.StepComplete = true
			machine.Progress.CompletedUnits = machine.Progress.TotalUnits
		}
	}

	historicallyClosed := machine.TerminalResult == LifecycleMachineSucceeded || machine.TerminalResult == LifecycleMachineSkipped
	if historicallyClosed {
		return machine
	}
	if machine.Position == 0 && machine.State == LifecycleMachinePlanned && machine.Action == "" {
		// V1 constructed this bit as complete without invoking Host code. Open
		// operations must execute the newly authoritative planned gate.
		machine.StepComplete = false
		return machine
	}
	if machine.Position > 0 && !lifecycleRuntimeBindingsReady(machine) {
		machine.Revalidation = LifecycleGateRevalidation{
			StepID:   lifecycleCoordinatorStepID(machine.Operation, 0, LifecycleMachinePlanned, ""),
			Position: 0,
		}
	}
	return machine
}

func lifecycleCoordinatorProgress(current LifecycleProgressCursor, stepID, checkpoint string) LifecycleProgressCursor {
	if checkpoint == "" {
		return current
	}
	marker := checkpoint
	if stepID != "" {
		marker = stepID + ":" + checkpoint
	}
	if marker != current.Checkpoint {
		current.Checkpoint = marker
		current.CheckpointSequence++
	}
	return current
}

func lifecycleCoordinatorCheckpoint(stepID, checkpoint string) string {
	if checkpoint == "" {
		return ""
	}
	return stepID + ":" + checkpoint
}

func lifecycleCoordinatorStepID(operation LifecycleMachineOperation, position int, state LifecycleMachineState, action LifecycleMachineAction) string {
	// upgrade.after shipped at position 8. Stable ids are durable ledger keys,
	// so the additive activation gate must not rename historical attempts.
	if operation == LifecycleMachineUpgrade && action == LifecycleMachineUpgradeAfter {
		position = 8
	}
	name := string(action)
	if name == "" {
		name = "host." + string(state)
	}
	return fmt.Sprintf("lifecycle.%s.%02d.%s", operation, position, name)
}

func lifecycleCoordinatorFailure(err error) LifecycleExecutionError {
	if err == nil {
		return LifecycleExecutionError{}
	}
	var carrier LifecycleCoordinatorFailureCarrier
	if errors.As(err, &carrier) {
		failure := carrier.LifecycleCoordinatorFailure()
		if failure.Code != "" && failure.Reason != "" {
			return failure
		}
	}
	if errors.Is(err, context.Canceled) {
		return LifecycleExecutionError{
			Code: "lifecycle.cancelled", Reason: "lifecycle.cancelled", Message: "The lifecycle operation was cancelled.", Retryable: true,
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return LifecycleExecutionError{
			Code: "lifecycle.deadline_exceeded", Reason: "lifecycle.deadline_exceeded", Message: "The lifecycle operation deadline expired.", Retryable: true,
		}
	}
	return LifecycleExecutionError{Code: "lifecycle.execution_failed", Reason: "lifecycle.execution_failed", Message: err.Error()}
}

func lifecycleCoordinatorTerminalContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), lifecycleCoordinatorTerminalTimeout)
}

func machineTerminalResult(value LifecycleMachineTerminal) string {
	switch value {
	case LifecycleMachineSucceeded:
		return LifecycleTerminalSucceeded
	case LifecycleMachineFailedRun:
		return LifecycleTerminalFailed
	case LifecycleMachineCancelled:
		return LifecycleTerminalCancelled
	case LifecycleMachineSkipped:
		return LifecycleTerminalSkipped
	default:
		return ""
	}
}

type LifecycleCoordinatorRunError struct {
	Failure LifecycleExecutionError
	Cause   error
}

func (e *LifecycleCoordinatorRunError) Error() string {
	if e == nil {
		return ""
	}
	if e.Failure.Reason == "" {
		return ErrLifecycleCoordinatorActionFailed.Error()
	}
	return e.Failure.Reason + ": " + e.Failure.Message
}

func (e *LifecycleCoordinatorRunError) Unwrap() error {
	if e == nil || e.Cause == nil {
		return ErrLifecycleCoordinatorActionFailed
	}
	return errors.Join(ErrLifecycleCoordinatorActionFailed, e.Cause)
}

var _ error = (*LifecycleCoordinatorRunError)(nil)
