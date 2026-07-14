package extensions

import (
	"errors"
	"fmt"
	"strings"
)

type LifecycleMachineOperation string

const (
	LifecycleMachineInstall   LifecycleMachineOperation = "install"
	LifecycleMachineEnable    LifecycleMachineOperation = "enable"
	LifecycleMachineDisable   LifecycleMachineOperation = "disable"
	LifecycleMachineUpgrade   LifecycleMachineOperation = "upgrade"
	LifecycleMachineRollback  LifecycleMachineOperation = "rollback"
	LifecycleMachineUninstall LifecycleMachineOperation = "uninstall"
)

type LifecycleMachineState string

const (
	LifecycleMachinePlanned      LifecycleMachineState = "planned"
	LifecycleMachineMigrating    LifecycleMachineState = "migrating"
	LifecycleMachineStarting     LifecycleMachineState = "starting"
	LifecycleMachineHealthy      LifecycleMachineState = "healthy"
	LifecycleMachineRegistering  LifecycleMachineState = "registering"
	LifecycleMachineEnabled      LifecycleMachineState = "enabled"
	LifecycleMachineDraining     LifecycleMachineState = "draining"
	LifecycleMachineUninstalling LifecycleMachineState = "uninstalling"
	LifecycleMachineFailed       LifecycleMachineState = "failed"
	LifecycleMachineRecovery     LifecycleMachineState = "recovery"
)

type LifecycleMachineAction string

const (
	LifecycleMachineInstallPlan    LifecycleMachineAction = "install.plan"
	LifecycleMachineInstallAction  LifecycleMachineAction = "install"
	LifecycleMachineEnableAction   LifecycleMachineAction = "enable"
	LifecycleMachineDisableAction  LifecycleMachineAction = "disable"
	LifecycleMachineUpgradePlan    LifecycleMachineAction = "upgrade.plan"
	LifecycleMachineUpgradeBefore  LifecycleMachineAction = "upgrade.before"
	LifecycleMachineUpgradeAfter   LifecycleMachineAction = "upgrade.after"
	LifecycleMachineRollbackAction LifecycleMachineAction = "rollback"
	LifecycleMachineUninstallPlan  LifecycleMachineAction = "uninstall.plan"
	LifecycleMachineUninstallStep  LifecycleMachineAction = "uninstall"
	LifecycleMachineUninstallAfter LifecycleMachineAction = "uninstall.after"
)

type LifecycleMachineTerminal string

const (
	LifecycleMachineSucceeded LifecycleMachineTerminal = "succeeded"
	LifecycleMachineFailedRun LifecycleMachineTerminal = "failed"
	LifecycleMachineCancelled LifecycleMachineTerminal = "cancelled"
	LifecycleMachineSkipped   LifecycleMachineTerminal = "skipped"
)

var (
	ErrLifecycleStateMachineInvalid     = errors.New("extensions: invalid lifecycle state machine")
	ErrLifecycleStateTransitionDenied   = errors.New("extensions: lifecycle state transition denied")
	ErrLifecycleStateProgressRegression = errors.New("extensions: lifecycle state progress regressed")
)

// LifecycleProgressCursor is operation-wide. Checkpoints are opaque, so their
// explicit sequence is the only safe ordering signal.
type LifecycleProgressCursor struct {
	CompletedUnits     uint64
	TotalUnits         uint64
	Checkpoint         string
	CheckpointSequence uint64
}

type LifecycleRecommendedStep struct {
	State         LifecycleMachineState
	Action        LifecycleMachineAction
	Skippable     bool
	ForceRequired bool
}

// LifecycleRuntimeBinding pins lifecycle work to one immutable artifact and,
// once started, one exact runtime instance. RuntimeInstanceID may be empty only
// before the Host has prepared or discovered the process.
type LifecycleRuntimeBinding struct {
	ExtensionID       string `json:"extensionId,omitempty"`
	ExtensionVersion  string `json:"extensionVersion,omitempty"`
	PackageDigest     string `json:"packageDigest,omitempty"`
	RuntimeInstanceID string `json:"runtimeInstanceId,omitempty"`
	VersionID         int64  `json:"versionId,omitempty"`
}

type LifecycleGateRevalidation struct {
	StepID   string `json:"stepId,omitempty"`
	Position int    `json:"position,omitempty"`
}

// LifecycleStateMachine is a pure snapshot suitable for persistence in the
// operation progress document. Position points into RecommendedLifecyclePath.
type LifecycleStateMachine struct {
	Operation        LifecycleMachineOperation
	State            LifecycleMachineState
	Action           LifecycleMachineAction
	Position         int
	StepComplete     bool
	TerminalResult   LifecycleMachineTerminal
	Forced           bool
	RecoveryPosition int
	Progress         LifecycleProgressCursor
	SourceBinding    LifecycleRuntimeBinding   `json:"sourceBinding,omitempty"`
	TargetBinding    LifecycleRuntimeBinding   `json:"targetBinding,omitempty"`
	Revalidation     LifecycleGateRevalidation `json:"revalidation,omitempty"`
	// HostSideEffectsStarted distinguishes historical side-effect-free skipped
	// snapshots from V2 planned gates that may already have created a process.
	HostSideEffectsStarted bool `json:"hostSideEffectsStarted,omitempty"`
}

type LifecycleStateTransition struct {
	State          LifecycleMachineState
	Action         LifecycleMachineAction
	CompleteStep   bool
	TerminalResult LifecycleMachineTerminal
	Retry          bool
	SkipStep       bool
	SkipReason     string
	EscalateForced bool
	Progress       LifecycleProgressCursor
}

var lifecycleRecommendedPaths = map[LifecycleMachineOperation][]LifecycleRecommendedStep{
	LifecycleMachineInstall: {
		{State: LifecycleMachinePlanned},
		{State: LifecycleMachinePlanned, Action: LifecycleMachineInstallPlan},
		{State: LifecycleMachineMigrating},
		{State: LifecycleMachineMigrating, Action: LifecycleMachineInstallAction},
		{State: LifecycleMachineStarting},
		{State: LifecycleMachineStarting, Action: LifecycleMachineEnableAction},
		{State: LifecycleMachineHealthy},
		{State: LifecycleMachineRegistering},
		{State: LifecycleMachineEnabled},
	},
	LifecycleMachineEnable: {
		{State: LifecycleMachinePlanned},
		{State: LifecycleMachineStarting},
		{State: LifecycleMachineStarting, Action: LifecycleMachineEnableAction},
		{State: LifecycleMachineHealthy},
		{State: LifecycleMachineRegistering},
		{State: LifecycleMachineEnabled},
	},
	LifecycleMachineDisable: {
		{State: LifecycleMachinePlanned},
		{State: LifecycleMachineDraining},
		{State: LifecycleMachineDraining, Action: LifecycleMachineDisableAction},
		{State: LifecycleMachineDraining},
	},
	LifecycleMachineUpgrade: {
		{State: LifecycleMachinePlanned},
		{State: LifecycleMachinePlanned, Action: LifecycleMachineUpgradePlan},
		{State: LifecycleMachineDraining},
		{State: LifecycleMachineDraining, Action: LifecycleMachineUpgradeBefore},
		{State: LifecycleMachineMigrating},
		{State: LifecycleMachineStarting},
		{State: LifecycleMachineHealthy},
		{State: LifecycleMachineRegistering},
		// Activation is a Host-owned atomic publication boundary. upgrade.after
		// runs only after the new exact instance is active.
		{State: LifecycleMachineEnabled},
		{State: LifecycleMachineEnabled, Action: LifecycleMachineUpgradeAfter, Skippable: true},
		{State: LifecycleMachineEnabled},
	},
	LifecycleMachineRollback: {
		{State: LifecycleMachinePlanned},
		{State: LifecycleMachineDraining},
		{State: LifecycleMachineStarting},
		{State: LifecycleMachineStarting, Action: LifecycleMachineRollbackAction},
		{State: LifecycleMachineHealthy},
		{State: LifecycleMachineRegistering},
		{State: LifecycleMachineEnabled},
	},
	LifecycleMachineUninstall: {
		{State: LifecycleMachinePlanned},
		{State: LifecycleMachinePlanned, Action: LifecycleMachineUninstallPlan},
		{State: LifecycleMachineDraining},
		{State: LifecycleMachineUninstalling},
		{State: LifecycleMachineUninstalling, Action: LifecycleMachineUninstallStep, Skippable: true, ForceRequired: true},
		{State: LifecycleMachineUninstalling, Action: LifecycleMachineUninstallAfter, Skippable: true, ForceRequired: true},
		{State: LifecycleMachineUninstalling},
	},
}

func NewLifecycleStateMachine(operation LifecycleMachineOperation, forced bool) (LifecycleStateMachine, error) {
	path, ok := lifecycleRecommendedPaths[operation]
	if !ok {
		return LifecycleStateMachine{}, fmt.Errorf("%w: unknown operation %q", ErrLifecycleStateMachineInvalid, operation)
	}
	if forced && operation != LifecycleMachineUninstall {
		return LifecycleStateMachine{}, fmt.Errorf("%w: forced execution is reserved for uninstall", ErrLifecycleStateMachineInvalid)
	}
	return LifecycleStateMachine{
		Operation: operation, State: path[0].State, Action: path[0].Action,
		Position: 0, StepComplete: false, Forced: forced,
	}, nil
}

func RecommendedLifecyclePath(operation LifecycleMachineOperation) ([]LifecycleRecommendedStep, error) {
	path, ok := lifecycleRecommendedPaths[operation]
	if !ok {
		return nil, fmt.Errorf("%w: unknown operation %q", ErrLifecycleStateMachineInvalid, operation)
	}
	return append([]LifecycleRecommendedStep(nil), path...), nil
}

func ValidateLifecycleTransition(current LifecycleStateMachine, transition LifecycleStateTransition) error {
	path, err := validateLifecycleMachine(current)
	if err != nil {
		return err
	}
	if err := validateLifecycleMachineProgress(current.Progress, transition.Progress); err != nil {
		return err
	}
	if transition.EscalateForced && current.Operation != LifecycleMachineUninstall {
		return lifecycleTransitionDenied("forced execution is reserved for uninstall")
	}
	if current.TerminalResult != "" {
		return validateLifecycleRetry(current, transition)
	}
	if transition.Retry {
		return lifecycleTransitionDenied("only a failed or cancelled terminal operation may retry")
	}
	if current.State == LifecycleMachineRecovery {
		return validateLifecycleRecovery(current, transition, path)
	}
	if transition.EscalateForced {
		return lifecycleTransitionDenied("forced uninstall may only be selected during recovery")
	}
	if transition.TerminalResult != "" {
		return validateLifecycleTerminal(current, transition, path)
	}
	if transition.SkipStep || strings.TrimSpace(transition.SkipReason) != "" {
		return lifecycleTransitionDenied("a step may only be skipped during recovery")
	}

	currentStep := path[current.Position]
	if transition.State == current.State && transition.Action == current.Action {
		if current.StepComplete {
			return lifecycleTransitionDenied("the current gate is already complete")
		}
		return nil
	}
	if !current.StepComplete || current.Position+1 >= len(path) {
		return lifecycleTransitionDenied("the current gate must complete before advancing")
	}
	next := path[current.Position+1]
	if transition.State != next.State || transition.Action != next.Action || transition.CompleteStep {
		return lifecycleTransitionDenied(fmt.Sprintf("expected next gate %s/%s after %s/%s", next.State, next.Action, currentStep.State, currentStep.Action))
	}
	return nil
}

func ApplyLifecycleTransition(current LifecycleStateMachine, transition LifecycleStateTransition) (LifecycleStateMachine, error) {
	if err := ValidateLifecycleTransition(current, transition); err != nil {
		return LifecycleStateMachine{}, err
	}
	path := lifecycleRecommendedPaths[current.Operation]
	next := current
	next.Progress = transition.Progress
	if transition.EscalateForced {
		next.Forced = true
	}
	if current.TerminalResult != "" {
		next.State = LifecycleMachineRecovery
		next.Action = ""
		next.TerminalResult = ""
		next.StepComplete = false
		return next, nil
	}
	if current.State == LifecycleMachineRecovery {
		step := path[current.RecoveryPosition]
		next.State = step.State
		next.Action = step.Action
		next.Position = current.RecoveryPosition
		next.StepComplete = transition.SkipStep
		return next, nil
	}
	if transition.TerminalResult != "" {
		next.TerminalResult = transition.TerminalResult
		if transition.TerminalResult == LifecycleMachineFailedRun || transition.TerminalResult == LifecycleMachineCancelled {
			next.State = LifecycleMachineFailed
			next.RecoveryPosition = current.Position
			next.StepComplete = false
		}
		return next, nil
	}
	if transition.State == current.State && transition.Action == current.Action {
		next.StepComplete = transition.CompleteStep
		return next, nil
	}
	next.Position++
	next.State = transition.State
	next.Action = transition.Action
	next.StepComplete = false
	return next, nil
}

func validateLifecycleMachine(current LifecycleStateMachine) ([]LifecycleRecommendedStep, error) {
	path, ok := lifecycleRecommendedPaths[current.Operation]
	if !ok || current.Position < 0 || current.Position >= len(path) {
		return nil, fmt.Errorf("%w: invalid operation or path position", ErrLifecycleStateMachineInvalid)
	}
	if current.Forced && current.Operation != LifecycleMachineUninstall {
		return nil, fmt.Errorf("%w: forced execution is reserved for uninstall", ErrLifecycleStateMachineInvalid)
	}
	if err := validateLifecycleMachineProgress(LifecycleProgressCursor{}, current.Progress); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLifecycleStateMachineInvalid, err)
	}
	step := path[current.Position]
	if err := validateLifecycleRevalidationMarker(current, path); err != nil {
		return nil, err
	}
	switch current.State {
	case LifecycleMachineFailed:
		if current.TerminalResult != LifecycleMachineFailedRun && current.TerminalResult != LifecycleMachineCancelled {
			return nil, fmt.Errorf("%w: failed state requires a failed or cancelled result", ErrLifecycleStateMachineInvalid)
		}
		if current.RecoveryPosition != current.Position || current.Action != step.Action || current.StepComplete {
			return nil, fmt.Errorf("%w: failed state lost its exact recovery gate", ErrLifecycleStateMachineInvalid)
		}
	case LifecycleMachineRecovery:
		if current.TerminalResult != "" || current.RecoveryPosition != current.Position || current.Action != "" || current.StepComplete {
			return nil, fmt.Errorf("%w: recovery must retain one incomplete gate", ErrLifecycleStateMachineInvalid)
		}
	default:
		if current.State != step.State || current.Action != step.Action {
			return nil, fmt.Errorf("%w: snapshot does not match its recommended gate", ErrLifecycleStateMachineInvalid)
		}
		if current.TerminalResult == LifecycleMachineFailedRun || current.TerminalResult == LifecycleMachineCancelled {
			return nil, fmt.Errorf("%w: failed or cancelled result requires failed state", ErrLifecycleStateMachineInvalid)
		}
		if current.TerminalResult != "" && current.TerminalResult != LifecycleMachineSucceeded && current.TerminalResult != LifecycleMachineSkipped {
			return nil, fmt.Errorf("%w: invalid terminal result %q", ErrLifecycleStateMachineInvalid, current.TerminalResult)
		}
		if current.TerminalResult == LifecycleMachineSucceeded &&
			(current.Position != len(path)-1 || !current.StepComplete ||
				(current.Progress.TotalUnits > 0 && current.Progress.CompletedUnits != current.Progress.TotalUnits)) {
			return nil, fmt.Errorf("%w: succeeded snapshot did not complete its final gate", ErrLifecycleStateMachineInvalid)
		}
		if current.TerminalResult == LifecycleMachineSkipped &&
			(current.Position != 0 || !current.StepComplete || current.HostSideEffectsStarted) {
			return nil, fmt.Errorf("%w: skipped snapshot has already crossed a side-effect gate", ErrLifecycleStateMachineInvalid)
		}
	}
	return path, nil
}

func validateLifecycleRetry(current LifecycleStateMachine, transition LifecycleStateTransition) error {
	if current.TerminalResult != LifecycleMachineFailedRun && current.TerminalResult != LifecycleMachineCancelled {
		return lifecycleTransitionDenied("successful or skipped operations cannot be revived")
	}
	if !transition.Retry || transition.State != LifecycleMachineRecovery || transition.Action != "" ||
		transition.TerminalResult != "" || transition.CompleteStep || transition.SkipStep || strings.TrimSpace(transition.SkipReason) != "" {
		return lifecycleTransitionDenied("failed or cancelled operations may only retry into recovery")
	}
	return nil
}

func validateLifecycleRecovery(current LifecycleStateMachine, transition LifecycleStateTransition, path []LifecycleRecommendedStep) error {
	if transition.Retry || transition.TerminalResult != "" || transition.CompleteStep {
		return lifecycleTransitionDenied("recovery must re-enter or explicitly skip its exact failed gate")
	}
	step := path[current.RecoveryPosition]
	if transition.State != step.State || transition.Action != step.Action {
		return lifecycleTransitionDenied("recovery cannot jump to a different gate")
	}
	if !transition.SkipStep {
		if strings.TrimSpace(transition.SkipReason) != "" {
			return lifecycleTransitionDenied("skip reason requires skip-step")
		}
		return nil
	}
	if strings.TrimSpace(transition.SkipReason) == "" || transition.SkipReason != strings.TrimSpace(transition.SkipReason) {
		return lifecycleTransitionDenied("skip-step requires a stable non-empty reason")
	}
	if step.Action == "" || !step.Skippable {
		return lifecycleTransitionDenied("this safety gate cannot be skipped")
	}
	forced := current.Forced || transition.EscalateForced
	if step.ForceRequired && !forced {
		return lifecycleTransitionDenied("this cleanup step requires forced uninstall")
	}
	return nil
}

func validateLifecycleTerminal(current LifecycleStateMachine, transition LifecycleStateTransition, path []LifecycleRecommendedStep) error {
	if transition.Retry || transition.SkipStep || strings.TrimSpace(transition.SkipReason) != "" || transition.CompleteStep ||
		transition.State == LifecycleMachineRecovery {
		return lifecycleTransitionDenied("terminal transition contains incompatible controls")
	}
	switch transition.TerminalResult {
	case LifecycleMachineSucceeded:
		if current.Position != len(path)-1 || !current.StepComplete || transition.State != current.State || transition.Action != current.Action {
			return lifecycleTransitionDenied("success requires the final recommended gate to complete")
		}
		if transition.Progress.TotalUnits > 0 && transition.Progress.CompletedUnits != transition.Progress.TotalUnits {
			return lifecycleTransitionDenied("success requires all progress units to complete")
		}
	case LifecycleMachineSkipped:
		if current.Position != 0 || !current.StepComplete || current.HostSideEffectsStarted ||
			transition.State != LifecycleMachinePlanned || transition.Action != "" {
			return lifecycleTransitionDenied("only a side-effect-free planned operation may be skipped")
		}
	case LifecycleMachineFailedRun, LifecycleMachineCancelled:
		if current.StepComplete || transition.State != LifecycleMachineFailed || transition.Action != current.Action {
			return lifecycleTransitionDenied("failure or cancellation must close the current incomplete gate")
		}
	default:
		return lifecycleTransitionDenied(fmt.Sprintf("unknown terminal result %q", transition.TerminalResult))
	}
	return nil
}

func validateLifecycleRevalidationMarker(current LifecycleStateMachine, path []LifecycleRecommendedStep) error {
	marker := current.Revalidation
	if marker.StepID == "" {
		if marker.Position != 0 {
			return fmt.Errorf("%w: Host revalidation position has no step id", ErrLifecycleStateMachineInvalid)
		}
		return nil
	}
	if marker.Position < 0 || marker.Position >= len(path) || path[marker.Position].Action != "" {
		return fmt.Errorf("%w: Host revalidation marker does not identify a Host gate", ErrLifecycleStateMachineInvalid)
	}
	step := path[marker.Position]
	canonical := lifecycleCoordinatorStepID(current.Operation, marker.Position, step.State, step.Action)
	if marker.StepID != canonical {
		return fmt.Errorf("%w: Host revalidation step id %q is not canonical", ErrLifecycleStateMachineInvalid, marker.StepID)
	}
	return nil
}

func validateLifecycleMachineProgress(current, next LifecycleProgressCursor) error {
	if next.CompletedUnits < current.CompletedUnits || next.TotalUnits < current.TotalUnits ||
		(next.TotalUnits > 0 && next.CompletedUnits > next.TotalUnits) {
		return fmt.Errorf("%w: counters decreased or exceeded total", ErrLifecycleStateProgressRegression)
	}
	if (next.Checkpoint == "") != (next.CheckpointSequence == 0) {
		return fmt.Errorf("%w: checkpoint and sequence must be set together", ErrLifecycleStateProgressRegression)
	}
	if current.Checkpoint == "" {
		return nil
	}
	if next.Checkpoint == "" || next.CheckpointSequence < current.CheckpointSequence {
		return fmt.Errorf("%w: checkpoint was cleared or its sequence decreased", ErrLifecycleStateProgressRegression)
	}
	if next.CheckpointSequence == current.CheckpointSequence && next.Checkpoint != current.Checkpoint {
		return fmt.Errorf("%w: checkpoint changed without advancing its sequence", ErrLifecycleStateProgressRegression)
	}
	if next.CheckpointSequence > current.CheckpointSequence && next.Checkpoint == current.Checkpoint {
		return fmt.Errorf("%w: checkpoint sequence advanced without a new checkpoint", ErrLifecycleStateProgressRegression)
	}
	return nil
}

func lifecycleTransitionDenied(reason string) error {
	return fmt.Errorf("%w: %s", ErrLifecycleStateTransitionDenied, reason)
}
