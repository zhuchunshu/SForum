package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

func (c *LifecycleCoordinator) drive(
	ctx context.Context,
	operation LifecycleOperation,
	machine LifecycleStateMachine,
	input LifecycleCoordinatorRunInput,
) (LifecycleOperation, LifecycleStateMachine, error) {
	path, _ := RecommendedLifecyclePath(machine.Operation)
	for {
		if machine.State == LifecycleMachineRecovery {
			var err error
			operation, machine, err = c.recoverGate(ctx, operation, machine, input)
			if err != nil {
				return operation, machine, err
			}
			continue
		}
		if !machine.StepComplete {
			var result json.RawMessage
			var err error
			operation, machine, result, err = c.executeCurrentGate(ctx, operation, machine, input)
			if err != nil {
				return operation, machine, err
			}
			_ = result
			continue
		}
		if machine.Position == len(path)-1 {
			completed, terminalMachine, err := c.completeSuccess(ctx, operation, machine)
			return completed, terminalMachine, err
		}
		next := path[machine.Position+1]
		transition := LifecycleStateTransition{
			State: next.State, Action: next.Action, Progress: machine.Progress,
		}
		started, err := ApplyLifecycleTransition(machine, transition)
		if err != nil {
			return operation, machine, err
		}
		stepID := lifecycleCoordinatorStepID(started.Operation, started.Position, started.State, started.Action)
		operation, err = c.persistMachine(ctx, operation, started, stepID)
		if err != nil {
			return operation, machine, err
		}
		machine = started
	}
}

func (c *LifecycleCoordinator) recoverGate(
	ctx context.Context,
	operation LifecycleOperation,
	machine LifecycleStateMachine,
	input LifecycleCoordinatorRunInput,
) (LifecycleOperation, LifecycleStateMachine, error) {
	path, _ := RecommendedLifecyclePath(machine.Operation)
	step := path[machine.RecoveryPosition]
	stepID := lifecycleCoordinatorStepID(machine.Operation, machine.RecoveryPosition, step.State, step.Action)
	recoveryCheckpoint := ""
	if step.Action != "" {
		latest, err := c.repository.LatestStepAttempt(ctx, operation.ID, stepID)
		if err == nil {
			recoveryCheckpoint = latest.Checkpoint
			if latest.Status == LifecycleStepSkipped {
				return c.persistRecoverySkip(ctx, operation, machine, step, latest.SkipReason)
			}
		} else if !errors.Is(err, ErrLifecycleStepNotFound) {
			return operation, machine, err
		}
	}
	if input.SkipFailedStep {
		if step.Action == "" {
			return operation, machine, fmt.Errorf("%w: Host safety gates cannot be skipped", ErrLifecycleCoordinatorInvalid)
		}
		skipTransition := LifecycleStateTransition{
			State: step.State, Action: step.Action, SkipStep: true,
			SkipReason: input.SkipReason, Progress: machine.Progress,
		}
		if err := ValidateLifecycleTransition(machine, skipTransition); err != nil {
			return operation, machine, err
		}
		begin, err := c.repository.BeginStepAttempt(ctx, BeginLifecycleStepAttemptInput{
			OperationID: operation.ID, StepID: stepID, LifecycleAction: string(step.Action),
			PlanVersion: operation.PlanVersion, InputDocument: cloneLifecycleJSON(input.ActionInputs[step.Action]),
			Checkpoint: recoveryCheckpoint, ActorUserID: operation.RequestedByUserID, AuditEventID: operation.AuditEventID,
		})
		if err != nil {
			return operation, machine, err
		}
		if _, err := c.repository.CompleteStepAttempt(ctx, CompleteLifecycleStepAttemptInput{
			AttemptID: begin.Attempt.ID, Status: LifecycleStepSkipped, Checkpoint: begin.Attempt.Checkpoint,
			CompletedUnits: begin.Attempt.CompletedUnits, TotalUnits: begin.Attempt.TotalUnits,
			SkipReason: input.SkipReason, Forced: machine.Forced,
			ActorUserID: operation.RequestedByUserID, AuditEventID: operation.AuditEventID,
		}); err != nil {
			return operation, machine, err
		}
		return c.persistRecoverySkip(ctx, operation, machine, step, input.SkipReason)
	}
	transition := LifecycleStateTransition{State: step.State, Action: step.Action, Progress: machine.Progress}
	reentered, err := ApplyLifecycleTransition(machine, transition)
	if err != nil {
		return operation, machine, err
	}
	if step.Action != "" {
		if _, err := c.repository.BeginStepAttempt(ctx, BeginLifecycleStepAttemptInput{
			OperationID: operation.ID, StepID: stepID, LifecycleAction: string(step.Action),
			PlanVersion: operation.PlanVersion, InputDocument: cloneLifecycleJSON(input.ActionInputs[step.Action]),
			Checkpoint: recoveryCheckpoint, ActorUserID: operation.RequestedByUserID, AuditEventID: operation.AuditEventID,
		}); err != nil {
			return operation, machine, err
		}
	}
	operation, err = c.persistMachine(ctx, operation, reentered, stepID)
	return operation, reentered, err
}

func (c *LifecycleCoordinator) persistRecoverySkip(
	ctx context.Context,
	operation LifecycleOperation,
	machine LifecycleStateMachine,
	step LifecycleRecommendedStep,
	reason string,
) (LifecycleOperation, LifecycleStateMachine, error) {
	progress := machine.Progress
	progress.CompletedUnits = uint64(machine.RecoveryPosition)
	transition := LifecycleStateTransition{
		State: step.State, Action: step.Action, SkipStep: true, SkipReason: reason, Progress: progress,
	}
	skipped, err := ApplyLifecycleTransition(machine, transition)
	if err != nil {
		return operation, machine, err
	}
	operation, err = c.persistMachine(ctx, operation, skipped, lifecycleCoordinatorStepID(skipped.Operation, skipped.Position, skipped.State, skipped.Action))
	return operation, skipped, err
}

func (c *LifecycleCoordinator) executeCurrentGate(
	ctx context.Context,
	operation LifecycleOperation,
	machine LifecycleStateMachine,
	input LifecycleCoordinatorRunInput,
) (LifecycleOperation, LifecycleStateMachine, json.RawMessage, error) {
	stepID := lifecycleCoordinatorStepID(machine.Operation, machine.Position, machine.State, machine.Action)
	if machine.Action == "" {
		if c.host == nil {
			return c.failHostGate(ctx, operation, machine, fmt.Errorf("%w: Host gate runner is required", ErrLifecycleCoordinatorUnavailable))
		}
		err := c.host.RunLifecycleHostGate(ctx, LifecycleCoordinatorGateRequest{
			Extension: input.Extension, Operation: machine.Operation, State: machine.State, StepID: stepID, Forced: machine.Forced,
		})
		if err != nil {
			return c.failHostGate(ctx, operation, machine, err)
		}
		if err := ctx.Err(); err != nil {
			return c.failHostGate(ctx, operation, machine, err)
		}
		operation, machine, err = c.completeMachineGate(ctx, operation, machine, "")
		return operation, machine, nil, err
	}
	return c.executeAction(ctx, operation, machine, input, stepID)
}

func (c *LifecycleCoordinator) executeAction(
	ctx context.Context,
	operation LifecycleOperation,
	machine LifecycleStateMachine,
	input LifecycleCoordinatorRunInput,
	stepID string,
) (LifecycleOperation, LifecycleStateMachine, json.RawMessage, error) {
	latest, latestErr := c.repository.LatestStepAttempt(ctx, operation.ID, stepID)
	if latestErr == nil && (latest.Status == LifecycleStepSucceeded || latest.Status == LifecycleStepSkipped) {
		operation, machine, err := c.completeMachineGate(ctx, operation, machine, lifecycleCoordinatorCheckpoint(stepID, latest.Checkpoint))
		return operation, machine, cloneLifecycleJSON(latest.ResultDocument), err
	}
	if latestErr != nil && !errors.Is(latestErr, ErrLifecycleStepNotFound) {
		return operation, machine, nil, latestErr
	}
	resumeCheckpoint := ""
	if latestErr == nil {
		resumeCheckpoint = latest.Checkpoint
	}
	begin, err := c.repository.BeginStepAttempt(ctx, BeginLifecycleStepAttemptInput{
		OperationID: operation.ID, StepID: stepID, LifecycleAction: string(machine.Action),
		PlanVersion: operation.PlanVersion, InputDocument: cloneLifecycleJSON(input.ActionInputs[machine.Action]),
		Checkpoint: resumeCheckpoint, ActorUserID: operation.RequestedByUserID, AuditEventID: operation.AuditEventID,
	})
	if err != nil {
		return operation, machine, nil, err
	}
	attempt := begin.Attempt
	if attempt.Checkpoint != "" {
		resumeCheckpoint = attempt.Checkpoint
	}
	if c.runtime == nil {
		return c.failAction(ctx, operation, machine, attempt, LifecycleCoordinatorActionResult{}, fmt.Errorf("%w: lifecycle runtime is required", ErrLifecycleCoordinatorUnavailable))
	}
	var progressPersistenceErr error
	result, runErr := c.runtime.RunLifecycleAction(ctx, LifecycleCoordinatorActionRequest{
		Extension: input.Extension, Operation: machine.Operation, Action: machine.Action,
		StepID: stepID, PlanVersion: operation.PlanVersion, Attempt: attempt.Attempt,
		Checkpoint: resumeCheckpoint, InputDocument: cloneLifecycleJSON(input.ActionInputs[machine.Action]), Forced: machine.Forced,
	}, func(progress LifecycleCoordinatorActionProgress) error {
		nextOperation, nextMachine, nextAttempt, updateErr := c.persistActionProgress(ctx, operation, machine, attempt, stepID, progress)
		if updateErr != nil {
			progressPersistenceErr = updateErr
			return updateErr
		}
		operation, machine, attempt = nextOperation, nextMachine, nextAttempt
		return nil
	})
	if progressPersistenceErr != nil {
		return operation, machine, nil, progressPersistenceErr
	}
	if runErr == nil && ctx.Err() != nil {
		runErr = ctx.Err()
	}
	if runErr != nil || result.Status == LifecycleStepFailed || result.Status == LifecycleStepCancelled {
		return c.failAction(ctx, operation, machine, attempt, result, runErr)
	}
	if result.Status != LifecycleStepSucceeded {
		return c.failAction(ctx, operation, machine, attempt, result, fmt.Errorf("%w: action returned non-terminal status %q", ErrLifecycleCoordinatorActionFailed, result.Status))
	}
	completed, err := c.repository.CompleteStepAttempt(ctx, CompleteLifecycleStepAttemptInput{
		AttemptID: attempt.ID, Status: LifecycleStepSucceeded, Checkpoint: result.Checkpoint,
		CompletedUnits: result.CompletedUnits, TotalUnits: result.TotalUnits, Message: result.Message,
		ResultDocument: cloneLifecycleJSON(result.ResultDocument),
		ActorUserID:    operation.RequestedByUserID, AuditEventID: operation.AuditEventID,
	})
	if err != nil {
		return operation, machine, nil, err
	}
	operation, machine, err = c.completeMachineGate(ctx, operation, machine, lifecycleCoordinatorCheckpoint(stepID, completed.Checkpoint))
	return operation, machine, cloneLifecycleJSON(completed.ResultDocument), err
}

func (c *LifecycleCoordinator) persistActionProgress(
	ctx context.Context,
	operation LifecycleOperation,
	machine LifecycleStateMachine,
	attempt LifecycleStepAttempt,
	stepID string,
	progress LifecycleCoordinatorActionProgress,
) (LifecycleOperation, LifecycleStateMachine, LifecycleStepAttempt, error) {
	if progress.Status != LifecycleStepPlanned && progress.Status != LifecycleStepRunning && progress.Status != LifecycleStepWaiting {
		return operation, machine, attempt, fmt.Errorf("%w: invalid non-terminal action progress %q", ErrLifecycleCoordinatorInvalid, progress.Status)
	}
	updated, err := c.repository.UpdateStepProgress(ctx, UpdateLifecycleStepProgressInput{
		AttemptID: attempt.ID, Status: progress.Status, Checkpoint: progress.Checkpoint,
		CompletedUnits: progress.CompletedUnits, TotalUnits: progress.TotalUnits, Message: progress.Message,
	})
	if err != nil {
		return operation, machine, attempt, err
	}
	cursor := lifecycleCoordinatorProgress(machine.Progress, stepID, progress.Checkpoint)
	progressTransition := LifecycleStateTransition{State: machine.State, Action: machine.Action, Progress: cursor}
	next, err := ApplyLifecycleTransition(machine, progressTransition)
	if err != nil {
		return operation, machine, updated, err
	}
	operation, err = c.persistMachine(ctx, operation, next, stepID)
	return operation, next, updated, err
}

func (c *LifecycleCoordinator) completeMachineGate(
	ctx context.Context,
	operation LifecycleOperation,
	machine LifecycleStateMachine,
	checkpoint string,
) (LifecycleOperation, LifecycleStateMachine, error) {
	cursor := machine.Progress
	cursor.CompletedUnits = uint64(machine.Position)
	if checkpoint != "" {
		cursor = lifecycleCoordinatorProgress(cursor, "", checkpoint)
	}
	transition := LifecycleStateTransition{
		State: machine.State, Action: machine.Action, CompleteStep: true, Progress: cursor,
	}
	completed, err := ApplyLifecycleTransition(machine, transition)
	if err != nil {
		return operation, machine, err
	}
	operation, err = c.persistMachine(ctx, operation, completed, lifecycleCoordinatorStepID(completed.Operation, completed.Position, completed.State, completed.Action))
	return operation, completed, err
}

func (c *LifecycleCoordinator) failAction(
	ctx context.Context,
	operation LifecycleOperation,
	machine LifecycleStateMachine,
	attempt LifecycleStepAttempt,
	result LifecycleCoordinatorActionResult,
	runErr error,
) (LifecycleOperation, LifecycleStateMachine, json.RawMessage, error) {
	failure := result.Error
	if runErr != nil {
		failure = lifecycleCoordinatorFailure(runErr)
	}
	status := result.Status
	terminal := LifecycleMachineFailedRun
	if errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded) || status == LifecycleStepCancelled {
		status = LifecycleStepCancelled
		terminal = LifecycleMachineCancelled
	} else {
		status = LifecycleStepFailed
	}
	if failure.Code == "" || failure.Reason == "" {
		failure = LifecycleExecutionError{
			Code: "lifecycle.action_failed", Reason: "lifecycle.action_failed",
			Message: "The lifecycle action failed without a typed error.",
		}
	}
	terminalCtx, cancel := lifecycleCoordinatorTerminalContext(ctx)
	defer cancel()
	checkpoint := result.Checkpoint
	if checkpoint == "" {
		checkpoint = attempt.Checkpoint
	}
	completedUnits := max(result.CompletedUnits, attempt.CompletedUnits)
	totalUnits := max(result.TotalUnits, attempt.TotalUnits)
	message := result.Message
	if message == "" {
		message = attempt.ProgressMessage
	}
	completed, err := c.repository.CompleteStepAttempt(terminalCtx, CompleteLifecycleStepAttemptInput{
		AttemptID: attempt.ID, Status: status, Checkpoint: checkpoint,
		CompletedUnits: completedUnits, TotalUnits: totalUnits, Message: message,
		ResultDocument: cloneLifecycleJSON(result.ResultDocument), Error: failure,
		ActorUserID: operation.RequestedByUserID, AuditEventID: operation.AuditEventID,
	})
	if err != nil {
		return operation, machine, nil, err
	}
	operation, machine, err = c.completeFailure(terminalCtx, operation, machine, terminal, failure, completed.ResultDocument)
	if err != nil {
		return operation, machine, nil, err
	}
	return operation, machine, nil, &LifecycleCoordinatorRunError{Failure: failure, Cause: runErr}
}

func (c *LifecycleCoordinator) failHostGate(
	ctx context.Context,
	operation LifecycleOperation,
	machine LifecycleStateMachine,
	cause error,
) (LifecycleOperation, LifecycleStateMachine, json.RawMessage, error) {
	failure := lifecycleCoordinatorFailure(cause)
	terminal := LifecycleMachineFailedRun
	if errors.Is(cause, context.Canceled) || errors.Is(cause, context.DeadlineExceeded) {
		terminal = LifecycleMachineCancelled
	}
	terminalCtx, cancel := lifecycleCoordinatorTerminalContext(ctx)
	defer cancel()
	operation, machine, err := c.completeFailure(terminalCtx, operation, machine, terminal, failure, nil)
	if err != nil {
		return operation, machine, nil, err
	}
	return operation, machine, nil, &LifecycleCoordinatorRunError{Failure: failure, Cause: cause}
}

func (c *LifecycleCoordinator) completeFailure(
	ctx context.Context,
	operation LifecycleOperation,
	machine LifecycleStateMachine,
	terminal LifecycleMachineTerminal,
	failure LifecycleExecutionError,
	result json.RawMessage,
) (LifecycleOperation, LifecycleStateMachine, error) {
	transition := LifecycleStateTransition{
		State: LifecycleMachineFailed, Action: machine.Action, TerminalResult: terminal, Progress: machine.Progress,
	}
	failed, err := ApplyLifecycleTransition(machine, transition)
	if err != nil {
		return operation, machine, err
	}
	operation, err = c.persistMachine(ctx, operation, failed, operation.CurrentStepID)
	if err != nil {
		return operation, machine, err
	}
	completed, err := c.repository.CompleteOperation(ctx, CompleteLifecycleOperationInput{
		OperationID: operation.ID, ExpectedRevision: operation.Revision, ExpectedState: operation.State,
		State: string(failed.State), TerminalResult: machineTerminalResult(terminal),
		ResultDocument: cloneLifecycleJSON(result), Error: failure, AuditEventID: operation.AuditEventID,
	})
	return completed, failed, err
}
