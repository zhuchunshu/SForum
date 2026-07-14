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
	lifecycleCoordinatorSnapshotSchemaV1 = "sforum.lifecycle.coordinator@1"
	lifecycleCoordinatorSnapshotSchema   = "sforum.lifecycle.coordinator@2"
	lifecycleCoordinatorTerminalTimeout  = 5 * time.Second
	lifecycleCoordinatorGateResultSchema = "sforum.lifecycle.host-gate-result@1"
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
	ClaimStepLease(context.Context, ClaimLifecycleStepLeaseInput) (LifecycleStepAttempt, error)
	HeartbeatStepLease(context.Context, HeartbeatLifecycleStepLeaseInput) (LifecycleStepAttempt, error)
	ReleaseStepLease(context.Context, ReleaseLifecycleStepLeaseInput) (LifecycleStepAttempt, error)
}

type LifecycleCoordinatorRuntime interface {
	// RunLifecycleAction 必须同步、串行调用进度回调，并且不得在返回后继续调用。
	RunLifecycleAction(context.Context, LifecycleCoordinatorActionRequest, func(LifecycleCoordinatorActionProgress) error) (LifecycleCoordinatorActionResult, error)
}

type LifecycleCoordinatorHost interface {
	RunLifecycleHostGate(context.Context, LifecycleCoordinatorGateRequest) (LifecycleCoordinatorGateResult, error)
}

type LifecycleCoordinatorFailureCarrier interface {
	LifecycleCoordinatorFailure() LifecycleExecutionError
}

type LifecycleCoordinatorActionRequest struct {
	// Extension remains the artifact that must execute this action. Source and
	// Target make both sides explicit for upgrade/rollback cleanup.
	Extension         Extension
	SourceExtension   *Extension
	TargetExtension   Extension
	RuntimeRole       LifecycleCoordinatorRuntimeRole
	SourceBinding     LifecycleRuntimeBinding
	TargetBinding     LifecycleRuntimeBinding
	OperationID       int64
	Operation         LifecycleMachineOperation
	Action            LifecycleMachineAction
	StepID            string
	PlanVersion       string
	Attempt           int
	Checkpoint        string
	InputDocument     json.RawMessage
	AuthorityType     string
	TrustGrantID      int64
	AuthoritySnapshot json.RawMessage
	RemovalMode       string
	Forced            bool
	ActorUserID       int64
	AuditEventID      int64
}

type LifecycleCoordinatorRuntimeRole string

const (
	LifecycleRuntimeSource LifecycleCoordinatorRuntimeRole = "source"
	LifecycleRuntimeTarget LifecycleCoordinatorRuntimeRole = "target"
)

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
	Extension       Extension
	SourceExtension *Extension
	TargetExtension Extension
	OperationID     int64
	Operation       LifecycleMachineOperation
	State           LifecycleMachineState
	Position        int
	StepID          string
	Attempt         int
	Checkpoint      string
	PreviousResult  json.RawMessage
	// ActionResults is rebuilt from the durable step ledger for allowlisted
	// plugin actions before Position. Hosts may mutate this local copy only.
	ActionResults     map[LifecycleMachineAction]json.RawMessage
	SourceBinding     LifecycleRuntimeBinding
	TargetBinding     LifecycleRuntimeBinding
	AuthorityType     string
	TrustGrantID      int64
	AuthoritySnapshot json.RawMessage
	RemovalMode       string
	Forced            bool
	ActorUserID       int64
	AuditEventID      int64
	Revalidation      bool
}

type LifecycleGateRevalidationPolicy string

const (
	LifecycleGateRevalidationRequired LifecycleGateRevalidationPolicy = "required"
	LifecycleGateDurable              LifecycleGateRevalidationPolicy = "durable"
)

// LifecycleCoordinatorGateResult is persisted as the terminal step result.
// A Host that returns RevalidationRequired is promising process-local state,
// so a later coordinator invocation must revalidate/recreate it before the
// next executable action.
type LifecycleCoordinatorGateResult struct {
	Checkpoint         string                          `json:"checkpoint,omitempty"`
	SourceBinding      LifecycleRuntimeBinding         `json:"sourceBinding,omitempty"`
	TargetBinding      LifecycleRuntimeBinding         `json:"targetBinding,omitempty"`
	RevalidationPolicy LifecycleGateRevalidationPolicy `json:"revalidationPolicy,omitempty"`
	ResultDocument     json.RawMessage                 `json:"resultDocument,omitempty"`
}

type LifecycleCoordinatorRunInput struct {
	Extension       Extension
	SourceExtension *Extension
	Acquire         AcquireLifecycleOperationInput
	ActionInputs    map[LifecycleMachineAction]json.RawMessage
	Retry           bool
	SkipFailedStep  bool
	SkipReason      string
}

type LifecycleCoordinatorRunResult struct {
	Operation LifecycleOperation
	Replayed  bool
}

type LifecycleCoordinator struct {
	repository             LifecycleCoordinatorRepository
	runtime                LifecycleCoordinatorRuntime
	host                   LifecycleCoordinatorHost
	leaseDuration          time.Duration
	leaseHeartbeatInterval time.Duration
}

func NewLifecycleCoordinator(repository LifecycleCoordinatorRepository, runtime LifecycleCoordinatorRuntime, host LifecycleCoordinatorHost) *LifecycleCoordinator {
	return &LifecycleCoordinator{
		repository: repository, runtime: runtime, host: host,
		leaseDuration:          lifecycleCoordinatorLeaseDuration,
		leaseHeartbeatInterval: lifecycleCoordinatorLeaseHeartbeatInterval,
	}
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
	machine, operation, err := c.loadOrInitializeMachine(ctx, operation, acquired.Created, input)
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

	operation, machine, err = c.drive(ctx, operation, machine, input, !acquired.Created)
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
	if operation.CompletedAt != nil || machine.TerminalResult != "" || machine.StepComplete {
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
	switch attempt.Status {
	case LifecycleStepSucceeded, LifecycleStepSkipped:
		if attempt.Status == LifecycleStepSkipped && machine.Action == "" {
			return operation, machine, fmt.Errorf("%w: Host safety gates cannot be skipped", ErrLifecycleCoordinatorInvalid)
		}
		if attempt.Status == LifecycleStepSucceeded && machine.Action == "" {
			previous, typed, decodeErr := decodeLifecycleHostGateResult(attempt.ResultDocument)
			if decodeErr != nil {
				return operation, machine, decodeErr
			}
			if typed {
				machine, decodeErr = applyLifecycleHostGateResult(machine, stepID, machine.Position, previous)
				if decodeErr != nil {
					return operation, machine, decodeErr
				}
			}
			if machine.Position == 0 {
				// Legacy untyped Host results still prove that the planned gate ran.
				machine.HostSideEffectsStarted = true
			}
		}
		return c.completeMachineGate(
			ctx, operation, machine, lifecycleCoordinatorCheckpoint(stepID, attempt.Checkpoint),
		)
	case LifecycleStepFailed:
		return c.reconcilePendingStepFailure(ctx, operation, machine, LifecycleMachineFailedRun, attempt)
	case LifecycleStepCancelled:
		return c.reconcilePendingStepFailure(ctx, operation, machine, LifecycleMachineCancelled, attempt)
	default:
		return operation, machine, nil
	}

}

func (c *LifecycleCoordinator) reconcilePendingStepFailure(
	ctx context.Context,
	operation LifecycleOperation,
	machine LifecycleStateMachine,
	terminal LifecycleMachineTerminal,
	attempt LifecycleStepAttempt,
) (LifecycleOperation, LifecycleStateMachine, error) {
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
	if err := validateLifecycleSourceArtifact(operation, input.Extension, input.SourceExtension); err != nil {
		return err
	}
	if input.SkipFailedStep && (!input.Retry || strings.TrimSpace(input.SkipReason) == "" || input.SkipReason != strings.TrimSpace(input.SkipReason)) {
		return fmt.Errorf("%w: skip-step requires retry and a stable reason", ErrLifecycleCoordinatorInvalid)
	}
	if !input.SkipFailedStep && strings.TrimSpace(input.SkipReason) != "" {
		return fmt.Errorf("%w: skip reason requires skip-step", ErrLifecycleCoordinatorInvalid)
	}
	return nil
}

func (c *LifecycleCoordinator) loadOrInitializeMachine(
	ctx context.Context,
	operation LifecycleOperation,
	created bool,
	input LifecycleCoordinatorRunInput,
) (LifecycleStateMachine, LifecycleOperation, error) {
	if !created && len(operation.Progress) > 0 && string(operation.Progress) != "{}" {
		machine, err := decodeLifecycleCoordinatorMachine(operation.Progress)
		if err != nil {
			return LifecycleStateMachine{}, operation, err
		}
		if err := hydrateLifecycleCoordinatorBindings(&machine, input.Extension, input.SourceExtension); err != nil {
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
	if err := hydrateLifecycleCoordinatorBindings(&machine, input.Extension, input.SourceExtension); err != nil {
		return LifecycleStateMachine{}, operation, err
	}
	path, _ := RecommendedLifecyclePath(machine.Operation)
	machine.Progress.TotalUnits = uint64(len(path) - 1)
	stepID := lifecycleCoordinatorStepID(machine.Operation, machine.Position, machine.State, machine.Action)
	operation, err = c.persistMachine(ctx, operation, machine, stepID)
	return machine, operation, err
}

func (c *LifecycleCoordinator) reconcilePendingTerminal(ctx context.Context, operation LifecycleOperation, machine LifecycleStateMachine) (LifecycleOperation, LifecycleStateMachine, error) {
	if operation.CompletedAt != nil || machine.TerminalResult == "" {
		return operation, machine, nil
	}
	terminal := machineTerminalResult(machine.TerminalResult)
	failure := LifecycleExecutionError{}
	result := json.RawMessage(nil)
	stepID := operation.CurrentStepID
	if stepID == "" {
		stepID = lifecycleCoordinatorStepID(machine.Operation, machine.Position, machine.State, machine.Action)
	}
	if attempt, err := c.repository.LatestStepAttempt(ctx, operation.ID, stepID); err == nil {
		failure = attempt.Error
		result = cloneLifecycleJSON(attempt.ResultDocument)
	} else if !errors.Is(err, ErrLifecycleStepNotFound) {
		return operation, machine, err
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
