package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

const lifecycleCoordinatorHostGateAction = "host.gate"

func (c *LifecycleCoordinator) drive(
	ctx context.Context,
	operation LifecycleOperation,
	machine LifecycleStateMachine,
	input LifecycleCoordinatorRunInput,
	revalidateBindings bool,
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
			var err error
			operation, machine, err = c.reconcilePendingStepTerminal(ctx, operation, machine)
			if err != nil {
				return operation, machine, err
			}
			if operation.CompletedAt != nil {
				return operation, machine, &LifecycleCoordinatorRunError{Failure: operation.Error}
			}
			if machine.StepComplete {
				// Persisted succeeded/skipped steps are authoritative. Advance first;
				// a later executable step will still consume the revalidation barrier.
				continue
			}
			claim, err := c.claimCurrentLifecycleGate(ctx, operation, machine, input)
			if errors.Is(err, ErrLifecycleStepClosed) {
				// The step became terminal after the replay check. Loop back and
				// consume that durable result instead of creating another attempt.
				continue
			}
			if err != nil {
				return operation, machine, err
			}
			if revalidateBindings {
				operation, machine, err = c.revalidateLifecycleHostState(ctx, operation, machine, input, claim)
				if err != nil {
					if !claim.released {
						releaseErr := c.releaseLifecycleGateClaim(ctx, claim)
						if releaseErr != nil {
							err = errors.Join(err, releaseErr)
						}
					}
					return operation, machine, err
				}
				revalidateBindings = false
			}
			var result json.RawMessage
			operation, machine, result, err = c.executeCurrentGate(ctx, operation, machine, input, claim)
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
	decision, err := c.repository.RecoveryDecision(ctx, operation.ID, operation.AttemptCount)
	if err != nil {
		return operation, machine, err
	}
	if decision.OperationID != operation.ID || decision.OperationAttempt != operation.AttemptCount ||
		decision.ActorUserID != operation.RecoveryActorUserID || decision.AuditEventID != operation.RecoveryAuditEventID ||
		(decision.EscalateForced && !operation.Forced) {
		return operation, machine, fmt.Errorf("%w: recovery decision does not match the durable operation", ErrLifecycleCoordinatorInvalid)
	}
	recoveryCheckpoint := ""
	latest, err := c.repository.LatestStepAttempt(ctx, operation.ID, stepID)
	if err == nil {
		recoveryCheckpoint = latest.Checkpoint
		if latest.Status == LifecycleStepSkipped {
			if decision.Decision != LifecycleRecoverySkipStep || latest.SkipReason != decision.Reason {
				return operation, machine, fmt.Errorf("%w: skipped step does not match the durable recovery decision", ErrLifecycleCoordinatorInvalid)
			}
			return c.persistRecoverySkip(ctx, operation, machine, step, latest.SkipReason)
		}
	} else if !errors.Is(err, ErrLifecycleStepNotFound) {
		return operation, machine, err
	}
	if decision.Decision == LifecycleRecoverySkipStep {
		if step.Action == "" {
			return operation, machine, fmt.Errorf("%w: Host safety gates cannot be skipped", ErrLifecycleCoordinatorInvalid)
		}
		skipTransition := LifecycleStateTransition{
			State: step.State, Action: step.Action, SkipStep: true,
			SkipReason: decision.Reason, Progress: machine.Progress,
		}
		if err := ValidateLifecycleTransition(machine, skipTransition); err != nil {
			return operation, machine, err
		}
		begin, err := c.repository.BeginStepAttempt(ctx, BeginLifecycleStepAttemptInput{
			OperationID: operation.ID, StepID: stepID, LifecycleAction: string(step.Action),
			PlanVersion: operation.PlanVersion, InputDocument: cloneLifecycleJSON(input.ActionInputs[step.Action]),
			Checkpoint: recoveryCheckpoint, ActorUserID: lifecycleOperationActorUserID(operation),
			AuditEventID: lifecycleOperationAuditEventID(operation),
		})
		if err != nil {
			return operation, machine, err
		}
		leaseCtx, cancelLease := context.WithCancel(ctx)
		lease, err := c.claimStepLease(leaseCtx, begin.Attempt, cancelLease)
		if err != nil {
			cancelLease()
			return operation, machine, err
		}
		lease.stopHeartbeat()
		cancelLease()
		terminalCtx, cancelTerminal := lifecycleCoordinatorTerminalContext(ctx)
		_, err = lease.complete(terminalCtx, CompleteLifecycleStepAttemptInput{
			AttemptID: begin.Attempt.ID, Status: LifecycleStepSkipped, Checkpoint: begin.Attempt.Checkpoint,
			CompletedUnits: begin.Attempt.CompletedUnits, TotalUnits: begin.Attempt.TotalUnits,
			SkipReason: decision.Reason, Forced: machine.Forced,
			ActorUserID:  lifecycleOperationActorUserID(operation),
			AuditEventID: lifecycleOperationAuditEventID(operation),
		})
		cancelTerminal()
		if err != nil {
			return operation, machine, err
		}
		return c.persistRecoverySkip(ctx, operation, machine, step, decision.Reason)
	}
	if decision.Decision != LifecycleRecoveryRetry {
		return operation, machine, fmt.Errorf("%w: unsupported durable recovery decision", ErrLifecycleCoordinatorInvalid)
	}
	transition := LifecycleStateTransition{State: step.State, Action: step.Action, Progress: machine.Progress}
	reentered, err := ApplyLifecycleTransition(machine, transition)
	if err != nil {
		return operation, machine, err
	}
	if _, err := c.repository.BeginStepAttempt(ctx, BeginLifecycleStepAttemptInput{
		OperationID: operation.ID, StepID: stepID, LifecycleAction: lifecycleCoordinatorActionName(step.Action),
		PlanVersion: operation.PlanVersion, InputDocument: cloneLifecycleJSON(input.ActionInputs[step.Action]),
		Checkpoint: recoveryCheckpoint, ActorUserID: lifecycleOperationActorUserID(operation),
		AuditEventID: lifecycleOperationAuditEventID(operation),
	}); err != nil {
		return operation, machine, err
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

type lifecycleCoordinatorGateClaim struct {
	attempt   LifecycleStepAttempt
	lease     *lifecycleCoordinatorLeaseSession
	runCtx    context.Context
	cancelRun context.CancelFunc
	released  bool
}

func (c *LifecycleCoordinator) claimCurrentLifecycleGate(
	ctx context.Context,
	operation LifecycleOperation,
	machine LifecycleStateMachine,
	input LifecycleCoordinatorRunInput,
) (*lifecycleCoordinatorGateClaim, error) {
	stepID := lifecycleCoordinatorStepID(machine.Operation, machine.Position, machine.State, machine.Action)
	checkpoint := ""
	if latest, err := c.repository.LatestStepAttempt(ctx, operation.ID, stepID); err == nil {
		if lifecycleStepTerminal(latest.Status) {
			return nil, ErrLifecycleStepClosed
		}
		checkpoint = latest.Checkpoint
	} else if !errors.Is(err, ErrLifecycleStepNotFound) {
		return nil, err
	}
	begin, err := c.repository.BeginStepAttempt(ctx, BeginLifecycleStepAttemptInput{
		OperationID: operation.ID, StepID: stepID, LifecycleAction: lifecycleCoordinatorActionName(machine.Action),
		PlanVersion: operation.PlanVersion, InputDocument: cloneLifecycleJSON(input.ActionInputs[machine.Action]),
		Checkpoint: checkpoint, ActorUserID: lifecycleOperationActorUserID(operation),
		AuditEventID: lifecycleOperationAuditEventID(operation),
	})
	if err != nil {
		return nil, err
	}
	runCtx, cancelRun := context.WithCancel(ctx)
	lease, err := c.claimStepLease(runCtx, begin.Attempt, cancelRun)
	if err != nil {
		cancelRun()
		return nil, err
	}
	return &lifecycleCoordinatorGateClaim{
		attempt: begin.Attempt, lease: lease, runCtx: runCtx, cancelRun: cancelRun,
	}, nil
}

func (c *LifecycleCoordinator) releaseLifecycleGateClaim(
	ctx context.Context,
	claim *lifecycleCoordinatorGateClaim,
) error {
	if claim == nil || claim.released {
		return nil
	}
	claim.lease.stopHeartbeat()
	claim.cancelRun()
	terminalCtx, cancel := lifecycleCoordinatorTerminalContext(ctx)
	defer cancel()
	claim.lease.mu.Lock()
	defer claim.lease.mu.Unlock()
	_, err := c.repository.ReleaseStepLease(terminalCtx, ReleaseLifecycleStepLeaseInput{
		AttemptID: claim.lease.attemptID, OwnerToken: claim.lease.ownerToken,
		Revision: claim.lease.revision,
	})
	if err == nil {
		claim.released = true
	}
	return err
}

func (c *LifecycleCoordinator) executeCurrentGate(
	ctx context.Context,
	operation LifecycleOperation,
	machine LifecycleStateMachine,
	input LifecycleCoordinatorRunInput,
	claim *lifecycleCoordinatorGateClaim,
) (LifecycleOperation, LifecycleStateMachine, json.RawMessage, error) {
	stepID := lifecycleCoordinatorStepID(machine.Operation, machine.Position, machine.State, machine.Action)
	if machine.Action == "" {
		return c.executeHostGate(ctx, operation, machine, input, stepID, claim)
	}
	return c.executeAction(ctx, operation, machine, input, stepID, claim)
}

func (c *LifecycleCoordinator) executeHostGate(
	ctx context.Context,
	operation LifecycleOperation,
	machine LifecycleStateMachine,
	input LifecycleCoordinatorRunInput,
	stepID string,
	claim *lifecycleCoordinatorGateClaim,
) (LifecycleOperation, LifecycleStateMachine, json.RawMessage, error) {
	return c.runLifecycleHostGate(
		ctx, operation, machine, input, machine.Position, machine.State, stepID,
		LifecycleStepAttempt{}, false, true, claim, nil,
	)
}

func (c *LifecycleCoordinator) runLifecycleHostGate(
	ctx context.Context,
	operation LifecycleOperation,
	machine LifecycleStateMachine,
	input LifecycleCoordinatorRunInput,
	gatePosition int,
	gateState LifecycleMachineState,
	stepID string,
	previous LifecycleStepAttempt,
	revalidation bool,
	completeCurrent bool,
	claimed *lifecycleCoordinatorGateClaim,
	failureBarrier *lifecycleCoordinatorGateClaim,
) (LifecycleOperation, LifecycleStateMachine, json.RawMessage, error) {
	if claimed == nil {
		begin, err := c.repository.BeginStepAttempt(ctx, BeginLifecycleStepAttemptInput{
			OperationID: operation.ID, StepID: stepID, LifecycleAction: lifecycleCoordinatorHostGateAction,
			PlanVersion: operation.PlanVersion, Checkpoint: previous.Checkpoint,
			ActorUserID:  lifecycleOperationActorUserID(operation),
			AuditEventID: lifecycleOperationAuditEventID(operation),
		})
		if err != nil {
			return operation, machine, nil, err
		}
		runCtx, cancelRun := context.WithCancel(ctx)
		lease, err := c.claimStepLease(runCtx, begin.Attempt, cancelRun)
		if err != nil {
			cancelRun()
			return operation, machine, nil, err
		}
		claimed = &lifecycleCoordinatorGateClaim{
			attempt: begin.Attempt, lease: lease, runCtx: runCtx, cancelRun: cancelRun,
		}
	}
	if claimed.attempt.StepID != stepID || claimed.attempt.LifecycleAction != lifecycleCoordinatorHostGateAction {
		return operation, machine, nil, fmt.Errorf("%w: claimed Host gate does not match the current step", ErrLifecycleCoordinatorInvalid)
	}
	attempt, lease, runCtx := claimed.attempt, claimed.lease, claimed.runCtx
	defer claimed.cancelRun()
	actionResults, err := c.lifecycleHostActionResults(ctx, operation.ID, machine.Operation, gatePosition)
	if err != nil {
		lease.stopHeartbeat()
		return c.failLifecycleHostGate(ctx, operation, machine, attempt, lease, failureBarrier, err)
	}
	var result LifecycleCoordinatorGateResult
	var runErr error
	if c.host == nil {
		runErr = fmt.Errorf("%w: Host gate runner is required", ErrLifecycleCoordinatorUnavailable)
	} else {
		result, runErr = c.host.RunLifecycleHostGate(runCtx, LifecycleCoordinatorGateRequest{
			Extension: input.Extension, SourceExtension: lifecycleSourceExtension(input), TargetExtension: input.Extension,
			OperationID: operation.ID, Operation: machine.Operation, State: gateState, Position: gatePosition,
			StepID: stepID, Attempt: attempt.Attempt, Checkpoint: attempt.Checkpoint,
			PreviousResult: cloneLifecycleJSON(previous.ResultDocument),
			ActionResults:  actionResults,
			SourceBinding:  machine.SourceBinding, TargetBinding: machine.TargetBinding,
			AuthorityType: operation.AuthorityType, TrustGrantID: operation.TrustGrantID,
			AuthoritySnapshot:    cloneLifecycleJSON(operation.AuthoritySnapshot),
			AuthorityActorUserID: lifecycleOperationAuthorityActorUserID(operation), RemovalMode: operation.RemovalMode,
			Forced: machine.Forced, ActorUserID: lifecycleOperationActorUserID(operation),
			AuditEventID: lifecycleOperationAuditEventID(operation),
			Revalidation: revalidation,
		})
		if runErr == nil && runCtx.Err() != nil {
			runErr = runCtx.Err()
		}
	}
	lease.stopHeartbeat()
	if leaseErr := lease.failure(); leaseErr != nil {
		return operation, machine, nil, leaseErr
	}
	if runErr != nil {
		return c.failLifecycleHostGate(ctx, operation, machine, attempt, lease, failureBarrier, runErr)
	}
	updatedMachine, err := applyLifecycleHostGateResult(machine, stepID, gatePosition, result)
	if err != nil {
		return c.failLifecycleHostGate(ctx, operation, machine, attempt, lease, failureBarrier, err)
	}
	machine = updatedMachine
	if gatePosition == 0 && gateState == LifecycleMachinePlanned && !lifecycleRuntimeBindingsReady(machine) {
		return c.failLifecycleHostGate(ctx, operation, machine, attempt, lease, failureBarrier, fmt.Errorf(
			"%w: planned Host gate did not bind every required runtime instance", ErrLifecycleCoordinatorInvalid,
		))
	}
	if revalidation && result.RevalidationPolicy != LifecycleGateRevalidationRequired {
		return c.failLifecycleHostGate(ctx, operation, machine, attempt, lease, failureBarrier, fmt.Errorf(
			"%w: runtime revalidation must remain explicitly process-local", ErrLifecycleCoordinatorInvalid,
		))
	}
	resultDocument, err := encodeLifecycleHostGateResult(result)
	if err != nil {
		return c.failLifecycleHostGate(ctx, operation, machine, attempt, lease, failureBarrier, err)
	}
	terminalCtx, cancelTerminal := lifecycleCoordinatorTerminalContext(ctx)
	completed, err := lease.complete(terminalCtx, CompleteLifecycleStepAttemptInput{
		AttemptID: attempt.ID, Status: LifecycleStepSucceeded, Checkpoint: result.Checkpoint,
		ResultDocument: resultDocument,
		ActorUserID:    lifecycleOperationActorUserID(operation),
		AuditEventID:   lifecycleOperationAuditEventID(operation),
	})
	cancelTerminal()
	if err != nil {
		return operation, machine, nil, err
	}
	if completeCurrent {
		operation, machine, err = c.completeMachineGate(ctx, operation, machine, lifecycleCoordinatorCheckpoint(stepID, completed.Checkpoint))
		return operation, machine, cloneLifecycleJSON(completed.ResultDocument), err
	}
	cursor := lifecycleCoordinatorProgress(machine.Progress, stepID, completed.Checkpoint)
	updated, err := ApplyLifecycleTransition(machine, LifecycleStateTransition{
		State: machine.State, Action: machine.Action, Progress: cursor,
	})
	if err != nil {
		return operation, machine, nil, err
	}
	operation, err = c.persistMachine(ctx, operation, updated, operation.CurrentStepID)
	return operation, updated, cloneLifecycleJSON(completed.ResultDocument), err
}

func (c *LifecycleCoordinator) failLifecycleHostGate(
	ctx context.Context,
	operation LifecycleOperation,
	machine LifecycleStateMachine,
	attempt LifecycleStepAttempt,
	lease *lifecycleCoordinatorLeaseSession,
	failureBarrier *lifecycleCoordinatorGateClaim,
	cause error,
) (LifecycleOperation, LifecycleStateMachine, json.RawMessage, error) {
	if failureBarrier != nil && !failureBarrier.released {
		if err := c.releaseLifecycleGateClaim(ctx, failureBarrier); err != nil {
			return operation, machine, nil, errors.Join(cause, fmt.Errorf("release current lifecycle gate lease: %w", err))
		}
	}
	return c.failHostGate(ctx, operation, machine, attempt, lease, cause)
}

func (c *LifecycleCoordinator) revalidateLifecycleHostState(
	ctx context.Context,
	operation LifecycleOperation,
	machine LifecycleStateMachine,
	input LifecycleCoordinatorRunInput,
	failureBarrier *lifecycleCoordinatorGateClaim,
) (LifecycleOperation, LifecycleStateMachine, error) {
	marker := machine.Revalidation
	if marker.StepID == "" {
		return operation, machine, nil
	}
	path, _ := RecommendedLifecyclePath(machine.Operation)
	if err := validateLifecycleRevalidationMarker(machine, path); err != nil {
		return operation, machine, fmt.Errorf("%w: invalid Host revalidation marker", ErrLifecycleCoordinatorInvalid)
	}
	if marker.StepID == lifecycleCoordinatorStepID(machine.Operation, machine.Position, machine.State, machine.Action) {
		return operation, machine, nil
	}
	latest, err := c.repository.LatestStepAttempt(ctx, operation.ID, marker.StepID)
	if errors.Is(err, ErrLifecycleStepNotFound) {
		operation, machine, _, err = c.runLifecycleHostGate(
			ctx, operation, machine, input, marker.Position, path[marker.Position].State,
			marker.StepID, LifecycleStepAttempt{}, true, false, nil, failureBarrier,
		)
		return operation, machine, err
	}
	if err != nil {
		return operation, machine, err
	}
	if latest.Status == LifecycleStepFailed || latest.Status == LifecycleStepCancelled {
		terminal := LifecycleMachineFailedRun
		if latest.Status == LifecycleStepCancelled {
			terminal = LifecycleMachineCancelled
		}
		failure := latest.Error
		if failure.Code == "" || failure.Reason == "" {
			failure = LifecycleExecutionError{
				Code: "lifecycle.coordinator_interrupted", Reason: "lifecycle.coordinator_interrupted",
				Message: "The Host resumed after revalidation failed but before the operation was finalized.", Retryable: true,
			}
		}
		if failureBarrier != nil && !failureBarrier.released {
			if releaseErr := c.releaseLifecycleGateClaim(ctx, failureBarrier); releaseErr != nil {
				return operation, machine, fmt.Errorf("release current lifecycle gate lease: %w", releaseErr)
			}
		}
		operation, machine, err = c.completeFailure(ctx, operation, machine, terminal, failure, latest.ResultDocument)
		if err != nil {
			return operation, machine, err
		}
		return operation, machine, &LifecycleCoordinatorRunError{Failure: failure}
	}
	previous, typed, err := decodeLifecycleHostGateResult(latest.ResultDocument)
	if err != nil {
		return operation, machine, err
	}
	if latest.Status != LifecycleStepSucceeded || !typed {
		return operation, machine, fmt.Errorf("%w: ephemeral Host state has no durable revalidation result", ErrLifecycleCoordinatorInvalid)
	}
	if previous.RevalidationPolicy != LifecycleGateRevalidationRequired {
		return operation, machine, fmt.Errorf("%w: ephemeral Host state lost its revalidation policy", ErrLifecycleCoordinatorInvalid)
	}
	operation, machine, _, err = c.runLifecycleHostGate(
		ctx, operation, machine, input, marker.Position, path[marker.Position].State,
		marker.StepID, latest, true, false, nil, failureBarrier,
	)
	return operation, machine, err
}

func (c *LifecycleCoordinator) executeAction(
	ctx context.Context,
	operation LifecycleOperation,
	machine LifecycleStateMachine,
	input LifecycleCoordinatorRunInput,
	stepID string,
	claim *lifecycleCoordinatorGateClaim,
) (LifecycleOperation, LifecycleStateMachine, json.RawMessage, error) {
	if claim == nil || claim.attempt.StepID != stepID || claim.attempt.LifecycleAction != string(machine.Action) {
		return operation, machine, nil, fmt.Errorf("%w: claimed action does not match the current step", ErrLifecycleCoordinatorInvalid)
	}
	attempt, lease, runCtx := claim.attempt, claim.lease, claim.runCtx
	defer claim.cancelRun()
	resumeCheckpoint := attempt.Checkpoint
	if c.runtime == nil {
		lease.stopHeartbeat()
		return c.failAction(ctx, operation, machine, attempt, lease, LifecycleCoordinatorActionResult{}, fmt.Errorf("%w: lifecycle runtime is required", ErrLifecycleCoordinatorUnavailable))
	}
	var progressPersistenceErr error
	role := lifecycleActionRuntimeRole(machine.Action)
	result, runErr := c.runtime.RunLifecycleAction(runCtx, LifecycleCoordinatorActionRequest{
		Extension: lifecycleActionExtension(input, role), SourceExtension: lifecycleSourceExtension(input), TargetExtension: input.Extension,
		RuntimeRole: role, SourceBinding: machine.SourceBinding, TargetBinding: machine.TargetBinding,
		OperationID: operation.ID, Operation: machine.Operation, Action: machine.Action,
		StepID: stepID, PlanVersion: operation.PlanVersion, Attempt: attempt.Attempt,
		Checkpoint: resumeCheckpoint, InputDocument: cloneLifecycleJSON(input.ActionInputs[machine.Action]),
		AuthorityType: operation.AuthorityType, TrustGrantID: operation.TrustGrantID,
		AuthoritySnapshot:    cloneLifecycleJSON(operation.AuthoritySnapshot),
		AuthorityActorUserID: lifecycleOperationAuthorityActorUserID(operation), RemovalMode: operation.RemovalMode,
		Forced: machine.Forced, ActorUserID: lifecycleOperationActorUserID(operation),
		AuditEventID: lifecycleOperationAuditEventID(operation),
	}, func(progress LifecycleCoordinatorActionProgress) error {
		nextOperation, nextMachine, nextAttempt, updateErr := c.persistActionProgress(runCtx, operation, machine, attempt, lease, stepID, progress)
		if updateErr != nil {
			progressPersistenceErr = updateErr
			return updateErr
		}
		operation, machine, attempt = nextOperation, nextMachine, nextAttempt
		return nil
	})
	lease.stopHeartbeat()
	if leaseErr := lease.failure(); leaseErr != nil {
		return operation, machine, nil, leaseErr
	}
	if progressPersistenceErr != nil {
		return operation, machine, nil, progressPersistenceErr
	}
	if runErr == nil && ctx.Err() != nil {
		runErr = ctx.Err()
	}
	if runErr != nil || result.Status == LifecycleStepFailed || result.Status == LifecycleStepCancelled {
		return c.failAction(ctx, operation, machine, attempt, lease, result, runErr)
	}
	if result.Status != LifecycleStepSucceeded {
		return c.failAction(ctx, operation, machine, attempt, lease, result, fmt.Errorf("%w: action returned non-terminal status %q", ErrLifecycleCoordinatorActionFailed, result.Status))
	}
	terminalCtx, cancelTerminal := lifecycleCoordinatorTerminalContext(ctx)
	completed, err := lease.complete(terminalCtx, CompleteLifecycleStepAttemptInput{
		AttemptID: attempt.ID, Status: LifecycleStepSucceeded, Checkpoint: result.Checkpoint,
		CompletedUnits: result.CompletedUnits, TotalUnits: result.TotalUnits, Message: result.Message,
		ResultDocument: cloneLifecycleJSON(result.ResultDocument),
		ActorUserID:    lifecycleOperationActorUserID(operation),
		AuditEventID:   lifecycleOperationAuditEventID(operation),
	})
	cancelTerminal()
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
	lease *lifecycleCoordinatorLeaseSession,
	stepID string,
	progress LifecycleCoordinatorActionProgress,
) (LifecycleOperation, LifecycleStateMachine, LifecycleStepAttempt, error) {
	if progress.Status != LifecycleStepPlanned && progress.Status != LifecycleStepRunning && progress.Status != LifecycleStepWaiting {
		return operation, machine, attempt, fmt.Errorf("%w: invalid non-terminal action progress %q", ErrLifecycleCoordinatorInvalid, progress.Status)
	}
	updated, err := lease.updateProgress(ctx, UpdateLifecycleStepProgressInput{
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
	lease *lifecycleCoordinatorLeaseSession,
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
	completed, err := lease.complete(terminalCtx, CompleteLifecycleStepAttemptInput{
		AttemptID: attempt.ID, Status: status, Checkpoint: checkpoint,
		CompletedUnits: completedUnits, TotalUnits: totalUnits, Message: message,
		ResultDocument: cloneLifecycleJSON(result.ResultDocument), Error: failure,
		ActorUserID:  lifecycleOperationActorUserID(operation),
		AuditEventID: lifecycleOperationAuditEventID(operation),
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
	attempt LifecycleStepAttempt,
	lease *lifecycleCoordinatorLeaseSession,
	cause error,
) (LifecycleOperation, LifecycleStateMachine, json.RawMessage, error) {
	failure := lifecycleCoordinatorFailure(cause)
	terminal := LifecycleMachineFailedRun
	if errors.Is(cause, context.Canceled) || errors.Is(cause, context.DeadlineExceeded) {
		terminal = LifecycleMachineCancelled
	}
	terminalCtx, cancel := lifecycleCoordinatorTerminalContext(ctx)
	defer cancel()
	status := LifecycleStepFailed
	if terminal == LifecycleMachineCancelled {
		status = LifecycleStepCancelled
	}
	if _, err := lease.complete(terminalCtx, CompleteLifecycleStepAttemptInput{
		AttemptID: attempt.ID, Status: status, Error: failure,
		ActorUserID:  lifecycleOperationActorUserID(operation),
		AuditEventID: lifecycleOperationAuditEventID(operation),
	}); err != nil {
		return operation, machine, nil, err
	}
	operation, machine, err := c.completeFailure(terminalCtx, operation, machine, terminal, failure, nil)
	if err != nil {
		return operation, machine, nil, err
	}
	return operation, machine, nil, &LifecycleCoordinatorRunError{Failure: failure, Cause: cause}
}

func lifecycleCoordinatorActionName(action LifecycleMachineAction) string {
	if action == "" {
		return lifecycleCoordinatorHostGateAction
	}
	return string(action)
}

func lifecycleOperationActorUserID(operation LifecycleOperation) int64 {
	if operation.RecoveryActorUserID > 0 && operation.RecoveryAuditEventID > 0 {
		return operation.RecoveryActorUserID
	}
	return operation.RequestedByUserID
}

func lifecycleOperationAuthorityActorUserID(operation LifecycleOperation) int64 {
	var authority LifecycleAuthoritySnapshot
	if json.Unmarshal(operation.AuthoritySnapshot, &authority) == nil && authority.ActorUserID > 0 {
		return authority.ActorUserID
	}
	// Compatibility with coordinator rows written before request and authority
	// actors were split. Exact runtime validation still rejects malformed rows.
	return operation.RequestedByUserID
}

func lifecycleOperationAuditEventID(operation LifecycleOperation) int64 {
	if operation.RecoveryActorUserID > 0 && operation.RecoveryAuditEventID > 0 {
		return operation.RecoveryAuditEventID
	}
	return operation.AuditEventID
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
	operation, err = c.persistFailedMachine(ctx, operation, failed, operation.CurrentStepID, failure)
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
