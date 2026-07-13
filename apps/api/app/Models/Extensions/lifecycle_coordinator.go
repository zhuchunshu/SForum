package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	lifecycleCoordinatorSnapshotSchema  = "sforum.lifecycle.coordinator@1"
	lifecycleCoordinatorTerminalTimeout = 5 * time.Second
)

var (
	ErrLifecycleCoordinatorInvalid       = errors.New("extensions: invalid lifecycle coordinator input")
	ErrLifecycleCoordinatorUnavailable   = errors.New("extensions: lifecycle coordinator dependency unavailable")
	ErrLifecycleCoordinatorRetryRequired = errors.New("extensions: lifecycle operation requires explicit retry")
	ErrLifecycleCoordinatorActionFailed  = errors.New("extensions: lifecycle action failed")
)

type LifecycleCoordinatorRepository interface {
	AcquireOperation(context.Context, AcquireLifecycleOperationInput) (AcquireLifecycleOperationResult, error)
	TransitionOperation(context.Context, TransitionLifecycleOperationInput) (LifecycleOperation, error)
	CompleteOperation(context.Context, CompleteLifecycleOperationInput) (LifecycleOperation, error)
	ResumeOperation(context.Context, ResumeLifecycleOperationInput) (LifecycleOperation, error)
	BeginStepAttempt(context.Context, BeginLifecycleStepAttemptInput) (BeginLifecycleStepAttemptResult, error)
	UpdateStepProgress(context.Context, UpdateLifecycleStepProgressInput) (LifecycleStepAttempt, error)
	CompleteStepAttempt(context.Context, CompleteLifecycleStepAttemptInput) (LifecycleStepAttempt, error)
	LatestStepAttempt(context.Context, int64, string) (LifecycleStepAttempt, error)
}

type LifecycleCoordinatorRuntime interface {
	RunLifecycleAction(context.Context, LifecycleCoordinatorActionRequest, func(LifecycleCoordinatorActionProgress) error) (LifecycleCoordinatorActionResult, error)
}

type LifecycleCoordinatorHost interface {
	RunLifecycleHostGate(context.Context, LifecycleCoordinatorGateRequest) error
}

type LifecycleCoordinatorFailureCarrier interface {
	LifecycleCoordinatorFailure() LifecycleExecutionError
}

type LifecycleCoordinatorActionRequest struct {
	Extension     Extension
	Operation     LifecycleMachineOperation
	Action        LifecycleMachineAction
	StepID        string
	PlanVersion   string
	Attempt       int
	Checkpoint    string
	InputDocument json.RawMessage
	Forced        bool
}

type LifecycleCoordinatorActionProgress struct {
	Status         string
	Checkpoint     string
	CompletedUnits int64
	TotalUnits     int64
	Message        string
}

type LifecycleCoordinatorActionResult struct {
	Status         string
	Checkpoint     string
	CompletedUnits int64
	TotalUnits     int64
	Message        string
	ResultDocument json.RawMessage
	Error          LifecycleExecutionError
}

type LifecycleCoordinatorGateRequest struct {
	Extension Extension
	Operation LifecycleMachineOperation
	State     LifecycleMachineState
	StepID    string
	Forced    bool
}

type LifecycleCoordinatorRunInput struct {
	Extension      Extension
	Acquire        AcquireLifecycleOperationInput
	ActionInputs   map[LifecycleMachineAction]json.RawMessage
	Retry          bool
	SkipFailedStep bool
	SkipReason     string
}

type LifecycleCoordinatorRunResult struct {
	Operation LifecycleOperation
	Replayed  bool
}

type LifecycleCoordinator struct {
	repository LifecycleCoordinatorRepository
	runtime    LifecycleCoordinatorRuntime
	host       LifecycleCoordinatorHost
}

func NewLifecycleCoordinator(repository LifecycleCoordinatorRepository, runtime LifecycleCoordinatorRuntime, host LifecycleCoordinatorHost) *LifecycleCoordinator {
	return &LifecycleCoordinator{repository: repository, runtime: runtime, host: host}
}

func (c *LifecycleCoordinator) Run(ctx context.Context, input LifecycleCoordinatorRunInput) (LifecycleCoordinatorRunResult, error) {
	if err := c.validateInput(ctx, input); err != nil {
		return LifecycleCoordinatorRunResult{}, err
	}
	acquired, err := c.repository.AcquireOperation(ctx, input.Acquire)
	if err != nil {
		return LifecycleCoordinatorRunResult{}, err
	}
	operation := acquired.Operation
	machine, operation, err := c.loadOrInitializeMachine(ctx, operation, acquired.Created)
	if err != nil {
		return LifecycleCoordinatorRunResult{Operation: operation}, err
	}
	operation, machine, err = c.reconcilePendingStepTerminal(ctx, operation, machine)
	if err != nil {
		return LifecycleCoordinatorRunResult{Operation: operation}, err
	}
	operation, machine, err = c.reconcilePendingTerminal(ctx, operation, machine)
	if err != nil {
		return LifecycleCoordinatorRunResult{Operation: operation}, err
	}
	if operation.CompletedAt != nil {
		switch operation.TerminalResult {
		case LifecycleTerminalSucceeded, LifecycleTerminalSkipped:
			return LifecycleCoordinatorRunResult{Operation: operation, Replayed: !acquired.Created}, nil
		case LifecycleTerminalFailed, LifecycleTerminalCancelled:
			if !input.Retry {
				return LifecycleCoordinatorRunResult{Operation: operation, Replayed: true}, ErrLifecycleCoordinatorRetryRequired
			}
			operation, machine, err = c.resume(ctx, operation, machine)
			if err != nil {
				return LifecycleCoordinatorRunResult{Operation: operation}, err
			}
		default:
			return LifecycleCoordinatorRunResult{Operation: operation}, fmt.Errorf("%w: completed operation has no valid terminal result", ErrLifecycleCoordinatorInvalid)
		}
	} else if input.Retry && operation.AttemptCount <= 1 {
		return LifecycleCoordinatorRunResult{Operation: operation}, fmt.Errorf("%w: retry requires a completed failed or cancelled operation", ErrLifecycleCoordinatorInvalid)
	}

	operation, machine, err = c.drive(ctx, operation, machine, input)
	if err != nil {
		return LifecycleCoordinatorRunResult{Operation: operation}, err
	}
	return LifecycleCoordinatorRunResult{Operation: operation}, nil
}

func (c *LifecycleCoordinator) reconcilePendingStepTerminal(
	ctx context.Context,
	operation LifecycleOperation,
	machine LifecycleStateMachine,
) (LifecycleOperation, LifecycleStateMachine, error) {
	if operation.CompletedAt != nil || machine.TerminalResult != "" || machine.Action == "" || machine.StepComplete {
		return operation, machine, nil
	}
	stepID := lifecycleCoordinatorStepID(machine.Operation, machine.Position, machine.State, machine.Action)
	attempt, err := c.repository.LatestStepAttempt(ctx, operation.ID, stepID)
	if errors.Is(err, ErrLifecycleStepNotFound) {
		return operation, machine, nil
	}
	if err != nil {
		return operation, machine, err
	}
	terminal := LifecycleMachineTerminal("")
	switch attempt.Status {
	case LifecycleStepFailed:
		terminal = LifecycleMachineFailedRun
	case LifecycleStepCancelled:
		terminal = LifecycleMachineCancelled
	default:
		return operation, machine, nil
	}
	failure := attempt.Error
	if failure.Code == "" || failure.Reason == "" {
		failure = LifecycleExecutionError{
			Code: "lifecycle.coordinator_interrupted", Reason: "lifecycle.coordinator_interrupted",
			Message: "The Host resumed after the action terminal state was persisted but before the operation was finalized.", Retryable: true,
		}
	}
	return c.completeFailure(ctx, operation, machine, terminal, failure, attempt.ResultDocument)
}

func (c *LifecycleCoordinator) validateInput(ctx context.Context, input LifecycleCoordinatorRunInput) error {
	if c == nil || c.repository == nil {
		return fmt.Errorf("%w: repository is required", ErrLifecycleCoordinatorUnavailable)
	}
	if ctx == nil {
		return fmt.Errorf("%w: context is required", ErrLifecycleCoordinatorInvalid)
	}
	operation := LifecycleMachineOperation(input.Acquire.Operation)
	if _, err := RecommendedLifecyclePath(operation); err != nil {
		return fmt.Errorf("%w: %v", ErrLifecycleCoordinatorInvalid, err)
	}
	if input.Extension.ID != input.Acquire.ExtensionID || input.Extension.Version != input.Acquire.ExtensionVersion ||
		input.Extension.PackageDigest != input.Acquire.PackageDigest {
		return fmt.Errorf("%w: extension snapshot does not match the acquired artifact", ErrLifecycleCoordinatorInvalid)
	}
	if input.Extension.Type != TypePlugin || input.Acquire.PlanVersion == "" || input.Acquire.IdempotencyKey == "" || input.Acquire.RequestFingerprint == "" {
		return fmt.Errorf("%w: exact plugin, plan, idempotency key, and fingerprint are required", ErrLifecycleCoordinatorInvalid)
	}
	if input.SkipFailedStep && (!input.Retry || strings.TrimSpace(input.SkipReason) == "" || input.SkipReason != strings.TrimSpace(input.SkipReason)) {
		return fmt.Errorf("%w: skip-step requires retry and a stable reason", ErrLifecycleCoordinatorInvalid)
	}
	if !input.SkipFailedStep && strings.TrimSpace(input.SkipReason) != "" {
		return fmt.Errorf("%w: skip reason requires skip-step", ErrLifecycleCoordinatorInvalid)
	}
	return nil
}

func (c *LifecycleCoordinator) loadOrInitializeMachine(ctx context.Context, operation LifecycleOperation, created bool) (LifecycleStateMachine, LifecycleOperation, error) {
	if !created && len(operation.Progress) > 0 && string(operation.Progress) != "{}" {
		machine, err := decodeLifecycleCoordinatorMachine(operation.Progress)
		if err != nil {
			return LifecycleStateMachine{}, operation, err
		}
		// ResumeOperation and the coordinator snapshot are separate durable CAS
		// writes. A crash between them is repaired from the retained terminal
		// machine without incrementing attempt_count a second time.
		if operation.State == LifecycleStateRecovery && machine.State == LifecycleMachineFailed &&
			(machine.TerminalResult == LifecycleMachineFailedRun || machine.TerminalResult == LifecycleMachineCancelled) {
			recovered, applyErr := ApplyLifecycleTransition(machine, LifecycleStateTransition{
				State: LifecycleMachineRecovery, Retry: true, Progress: machine.Progress,
			})
			if applyErr != nil {
				return LifecycleStateMachine{}, operation, applyErr
			}
			operation, err = c.persistMachine(ctx, operation, recovered, operation.CurrentStepID)
			return recovered, operation, err
		}
		if string(machine.Operation) != operation.Operation || string(machine.State) != operation.State || machine.Forced != operation.Forced {
			return LifecycleStateMachine{}, operation, fmt.Errorf("%w: durable operation and machine snapshot diverged", ErrLifecycleCoordinatorInvalid)
		}
		return machine, operation, nil
	}
	if !created && operation.State != LifecycleStatePlanned {
		return LifecycleStateMachine{}, operation, fmt.Errorf("%w: non-planned operation has no coordinator snapshot", ErrLifecycleCoordinatorInvalid)
	}
	machine, err := NewLifecycleStateMachine(LifecycleMachineOperation(operation.Operation), operation.Forced)
	if err != nil {
		return LifecycleStateMachine{}, operation, err
	}
	path, _ := RecommendedLifecyclePath(machine.Operation)
	machine.Progress.TotalUnits = uint64(len(path) - 1)
	operation, err = c.persistMachine(ctx, operation, machine, "")
	return machine, operation, err
}

func (c *LifecycleCoordinator) reconcilePendingTerminal(ctx context.Context, operation LifecycleOperation, machine LifecycleStateMachine) (LifecycleOperation, LifecycleStateMachine, error) {
	if operation.CompletedAt != nil || machine.TerminalResult == "" {
		return operation, machine, nil
	}
	terminal := machineTerminalResult(machine.TerminalResult)
	failure := LifecycleExecutionError{}
	result := json.RawMessage(nil)
	if machine.Action != "" {
		stepID := lifecycleCoordinatorStepID(machine.Operation, machine.Position, machine.State, machine.Action)
		if attempt, err := c.repository.LatestStepAttempt(ctx, operation.ID, stepID); err == nil {
			failure = attempt.Error
			result = cloneLifecycleJSON(attempt.ResultDocument)
		} else if !errors.Is(err, ErrLifecycleStepNotFound) {
			return operation, machine, err
		}
	}
	if terminal == LifecycleTerminalFailed || terminal == LifecycleTerminalCancelled {
		if failure.Code == "" || failure.Reason == "" {
			failure = LifecycleExecutionError{
				Code: "lifecycle.coordinator_interrupted", Reason: "lifecycle.coordinator_interrupted",
				Message: "The Host resumed after the terminal state was persisted but before completion was finalized.", Retryable: true,
			}
		}
	}
	completed, err := c.repository.CompleteOperation(ctx, CompleteLifecycleOperationInput{
		OperationID: operation.ID, ExpectedRevision: operation.Revision, ExpectedState: operation.State,
		State: string(machine.State), TerminalResult: terminal, ResultDocument: result, Error: failure,
		AuditEventID: operation.AuditEventID,
	})
	return completed, machine, err
}

func (c *LifecycleCoordinator) resume(ctx context.Context, operation LifecycleOperation, machine LifecycleStateMachine) (LifecycleOperation, LifecycleStateMachine, error) {
	transition := LifecycleStateTransition{
		State: LifecycleMachineRecovery, Retry: true, Progress: machine.Progress,
	}
	recovered, err := ApplyLifecycleTransition(machine, transition)
	if err != nil {
		return operation, machine, err
	}
	resumed, err := c.repository.ResumeOperation(ctx, ResumeLifecycleOperationInput{
		OperationID: operation.ID, ExpectedRevision: operation.Revision, ExpectedState: operation.State,
	})
	if err != nil {
		return operation, machine, err
	}
	resumed, err = c.persistMachine(ctx, resumed, recovered, operation.CurrentStepID)
	return resumed, recovered, err
}
