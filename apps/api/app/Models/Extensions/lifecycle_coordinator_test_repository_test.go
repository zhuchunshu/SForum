package extensions

import (
	"context"
	"sync"
	"time"
)

type lifecycleCoordinatorTestRepository struct {
	mu                              sync.Mutex
	operation                       LifecycleOperation
	acquired                        bool
	steps                           map[string][]LifecycleStepAttempt
	recoveryDecisions               []LifecycleRecoveryDecision
	nextStepID                      int64
	states                          []string
	failCompleteStepOnce            bool
	failTransitionOnce              bool
	failCompleteOperationOnce       bool
	failLatestStepOnce              bool
	failLatestAfterRecoveryReentry  bool
	cancelDuringTerminal            context.CancelFunc
	failAfterLeaseClaimAction       string
	failHeartbeatAction             string
	failHeartbeatOnce               error
	leaseHeartbeatCount             int
	lastProgressLeaseRevision       int64
	lastActionCompleteLeaseRevision int64
	leaseHeartbeatNotify            chan struct{}
	recordStepTerminalAction        string
	recordStepTerminalStatus        string
	terminalContexts                []lifecycleCoordinatorTestContextRecord
}

type lifecycleCoordinatorTestContextRecord struct {
	method      string
	err         error
	hasDeadline bool
	remaining   time.Duration
}

func newLifecycleCoordinatorTestRepository() *lifecycleCoordinatorTestRepository {
	return &lifecycleCoordinatorTestRepository{steps: map[string][]LifecycleStepAttempt{}, nextStepID: 1}
}

func (r *lifecycleCoordinatorTestRepository) AcquireOperation(_ context.Context, input AcquireLifecycleOperationInput) (AcquireLifecycleOperationResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.acquired {
		if r.operation.IdempotencyKey != input.IdempotencyKey || r.operation.RequestFingerprint != input.RequestFingerprint {
			return AcquireLifecycleOperationResult{}, ErrLifecycleFingerprintConflict
		}
		return AcquireLifecycleOperationResult{Operation: cloneLifecycleCoordinatorTestOperation(r.operation)}, nil
	}
	if input.ExistingOnly {
		return AcquireLifecycleOperationResult{}, ErrLifecycleOperationNotFound
	}
	r.acquired = true
	r.operation = LifecycleOperation{
		ID: 1, ExtensionID: input.ExtensionID, ExtensionVersion: input.ExtensionVersion,
		PackageDigest: input.PackageDigest, Operation: input.Operation, State: LifecycleStatePlanned,
		PlanVersion: input.PlanVersion, IdempotencyKey: input.IdempotencyKey,
		RequestFingerprint: input.RequestFingerprint, AuthorityType: input.AuthorityType,
		TrustGrantID: input.TrustGrantID, AuthoritySnapshot: cloneLifecycleJSON(input.AuthoritySnapshot),
		RemovalMode: input.RemovalMode, Forced: input.Forced,
		RequestedByUserID: input.RequestedByUserID, AuditEventID: input.AuditEventID,
		AttemptCount: 1, Revision: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	r.states = append(r.states, r.operation.State)
	return AcquireLifecycleOperationResult{Operation: cloneLifecycleCoordinatorTestOperation(r.operation), Created: true}, nil
}

func (r *lifecycleCoordinatorTestRepository) TransitionOperation(ctx context.Context, input TransitionLifecycleOperationInput) (LifecycleOperation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failTransitionOnce {
		r.failTransitionOnce = false
		return LifecycleOperation{}, errLifecycleCoordinatorTestCrash
	}
	if r.operation.Revision != input.ExpectedRevision || r.operation.State != input.ExpectedState || r.operation.CompletedAt != nil {
		return LifecycleOperation{}, ErrLifecycleRevisionConflict
	}
	if r.failLatestAfterRecoveryReentry && r.operation.State == LifecycleStateRecovery && input.State != LifecycleStateRecovery {
		r.failLatestAfterRecoveryReentry = false
		r.failLatestStepOnce = true
	}
	if input.State == LifecycleStateFailed {
		r.recordTerminalContext(ctx, "transition_operation")
	}
	r.operation.State = input.State
	if input.CurrentStepID != "" {
		r.operation.CurrentStepID = input.CurrentStepID
	}
	if len(input.Checkpoint) > 0 {
		r.operation.Checkpoint = cloneLifecycleJSON(input.Checkpoint)
	}
	if len(input.Progress) > 0 {
		r.operation.Progress = cloneLifecycleJSON(input.Progress)
	}
	r.operation.Revision++
	r.operation.UpdatedAt = time.Now()
	r.states = append(r.states, r.operation.State)
	return cloneLifecycleCoordinatorTestOperation(r.operation), nil
}

func (r *lifecycleCoordinatorTestRepository) CompleteOperation(ctx context.Context, input CompleteLifecycleOperationInput) (LifecycleOperation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failCompleteOperationOnce {
		r.failCompleteOperationOnce = false
		return LifecycleOperation{}, errLifecycleCoordinatorTestCrash
	}
	if r.operation.Revision != input.ExpectedRevision || r.operation.State != input.ExpectedState || r.operation.CompletedAt != nil {
		return LifecycleOperation{}, ErrLifecycleRevisionConflict
	}
	if input.TerminalResult == LifecycleTerminalFailed || input.TerminalResult == LifecycleTerminalCancelled {
		r.recordTerminalContext(ctx, "complete_operation")
	}
	now := time.Now()
	r.operation.State = input.State
	r.operation.TerminalResult = input.TerminalResult
	r.operation.ResultDocument = cloneLifecycleJSON(input.ResultDocument)
	r.operation.Error = input.Error
	if r.operation.AuditEventID == 0 && input.AuditEventID != 0 {
		r.operation.AuditEventID = input.AuditEventID
	}
	r.operation.CompletedAt = &now
	r.operation.Revision++
	r.states = append(r.states, r.operation.State)
	return cloneLifecycleCoordinatorTestOperation(r.operation), nil
}

func (r *lifecycleCoordinatorTestRepository) ResumeOperation(_ context.Context, input ResumeLifecycleOperationInput) (LifecycleOperation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := validateLifecycleRecoveryInput(input); err != nil {
		return LifecycleOperation{}, err
	}
	if r.operation.Revision != input.ExpectedRevision || r.operation.State != input.ExpectedState || r.operation.CompletedAt == nil {
		return LifecycleOperation{}, ErrLifecycleRevisionConflict
	}
	if r.operation.TerminalResult != LifecycleTerminalFailed && r.operation.TerminalResult != LifecycleTerminalCancelled {
		return LifecycleOperation{}, ErrLifecycleNotRecoverable
	}
	if input.EscalateForced && (r.operation.Operation != LifecycleOperationUninstall || r.operation.Forced) {
		return LifecycleOperation{}, ErrLifecycleInvalidInput
	}
	nextAttempt := r.operation.AttemptCount + 1
	r.recoveryDecisions = append(r.recoveryDecisions, LifecycleRecoveryDecision{
		ID: int64(len(r.recoveryDecisions) + 1), OperationID: r.operation.ID,
		OperationAttempt: nextAttempt, Decision: input.Decision,
		EscalateForced: input.EscalateForced, Reason: input.Reason,
		ActorUserID: input.ActorUserID, AuditEventID: input.AuditEventID, CreatedAt: time.Now(),
	})
	r.operation.State = LifecycleStateRecovery
	r.operation.TerminalResult = ""
	r.operation.CompletedAt = nil
	r.operation.AttemptCount++
	r.operation.RecoveryActorUserID = input.ActorUserID
	r.operation.RecoveryAuditEventID = input.AuditEventID
	r.operation.Forced = r.operation.Forced || input.EscalateForced
	r.operation.Revision++
	r.states = append(r.states, r.operation.State)
	return cloneLifecycleCoordinatorTestOperation(r.operation), nil
}

func (r *lifecycleCoordinatorTestRepository) RecoveryDecision(
	_ context.Context,
	operationID int64,
	operationAttempt int,
) (LifecycleRecoveryDecision, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, decision := range r.recoveryDecisions {
		if decision.OperationID == operationID && decision.OperationAttempt == operationAttempt {
			return decision, nil
		}
	}
	return LifecycleRecoveryDecision{}, ErrLifecycleRecoveryNotFound
}

func (r *lifecycleCoordinatorTestRepository) recoveryDecisionsSnapshot() []LifecycleRecoveryDecision {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]LifecycleRecoveryDecision(nil), r.recoveryDecisions...)
}

func (r *lifecycleCoordinatorTestRepository) BeginStepAttempt(_ context.Context, input BeginLifecycleStepAttemptInput) (BeginLifecycleStepAttemptResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := r.steps[input.StepID]
	for index := len(items) - 1; index >= 0; index-- {
		if !lifecycleStepTerminal(items[index].Status) {
			return BeginLifecycleStepAttemptResult{Attempt: items[index]}, nil
		}
	}
	attempt := LifecycleStepAttempt{
		ID: r.nextStepID, OperationID: input.OperationID, StepID: input.StepID,
		LifecycleAction: input.LifecycleAction, PlanVersion: input.PlanVersion,
		Attempt: len(items) + 1, Status: LifecycleStepPlanned, Checkpoint: input.Checkpoint,
		InputDocument: cloneLifecycleJSON(input.InputDocument), ActorUserID: input.ActorUserID,
		AuditEventID: input.AuditEventID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	r.nextStepID++
	r.steps[input.StepID] = append(items, attempt)
	return BeginLifecycleStepAttemptResult{Attempt: attempt, Created: true}, nil
}

func (r *lifecycleCoordinatorTestRepository) UpdateStepProgress(_ context.Context, input UpdateLifecycleStepProgressInput) (LifecycleStepAttempt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	attempt, err := r.stepByID(input.AttemptID)
	if err != nil {
		return LifecycleStepAttempt{}, err
	}
	if err := r.authorizeLease(*attempt, input.LeaseOwnerToken, input.LeaseRevision); err != nil {
		return LifecycleStepAttempt{}, err
	}
	r.lastProgressLeaseRevision = input.LeaseRevision
	if lifecycleStepTerminal(attempt.Status) || input.CompletedUnits < attempt.CompletedUnits || input.TotalUnits < attempt.TotalUnits ||
		(input.TotalUnits > 0 && input.CompletedUnits > input.TotalUnits) {
		return LifecycleStepAttempt{}, ErrLifecycleProgressRegression
	}
	attempt.Status = input.Status
	if input.Checkpoint != "" {
		attempt.Checkpoint = input.Checkpoint
	}
	attempt.CompletedUnits = input.CompletedUnits
	attempt.TotalUnits = input.TotalUnits
	attempt.ProgressMessage = input.Message
	attempt.UpdatedAt = time.Now()
	r.replaceStep(*attempt)
	return *attempt, nil
}

func (r *lifecycleCoordinatorTestRepository) CompleteStepAttempt(ctx context.Context, input CompleteLifecycleStepAttemptInput) (LifecycleStepAttempt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failCompleteStepOnce {
		r.failCompleteStepOnce = false
		return LifecycleStepAttempt{}, errLifecycleCoordinatorTestCrash
	}
	attempt, err := r.stepByID(input.AttemptID)
	if err != nil {
		return LifecycleStepAttempt{}, err
	}
	if err := r.authorizeLease(*attempt, input.LeaseOwnerToken, input.LeaseRevision); err != nil {
		return LifecycleStepAttempt{}, err
	}
	if attempt.LifecycleAction != lifecycleCoordinatorHostGateAction {
		r.lastActionCompleteLeaseRevision = input.LeaseRevision
	}
	if lifecycleStepTerminal(attempt.Status) || input.CompletedUnits < attempt.CompletedUnits || input.TotalUnits < attempt.TotalUnits ||
		(input.TotalUnits > 0 && input.CompletedUnits > input.TotalUnits) {
		return LifecycleStepAttempt{}, ErrLifecycleProgressRegression
	}
	if input.Status == LifecycleStepFailed || input.Status == LifecycleStepCancelled ||
		(attempt.LifecycleAction == r.recordStepTerminalAction && input.Status == r.recordStepTerminalStatus) {
		r.recordTerminalContext(ctx, "complete_step")
	}
	now := time.Now()
	attempt.Status = input.Status
	if input.Checkpoint != "" {
		attempt.Checkpoint = input.Checkpoint
	}
	attempt.CompletedUnits = input.CompletedUnits
	attempt.TotalUnits = input.TotalUnits
	attempt.ProgressMessage = input.Message
	attempt.ResultDocument = cloneLifecycleJSON(input.ResultDocument)
	attempt.Error = input.Error
	attempt.SkipReason = input.SkipReason
	attempt.Forced = input.Forced
	if input.ActorUserID != 0 {
		attempt.ActorUserID = input.ActorUserID
	}
	if input.AuditEventID != 0 {
		attempt.AuditEventID = input.AuditEventID
	}
	attempt.CompletedAt = &now
	attempt.UpdatedAt = now
	if attempt.LeaseOwnerToken != "" {
		attempt.LeaseRevision++
	}
	attempt.LeaseOwnerToken = ""
	attempt.LeaseExpiresAt = nil
	attempt.LeaseHeartbeatAt = nil
	r.replaceStep(*attempt)
	return *attempt, nil
}

func (r *lifecycleCoordinatorTestRepository) ClaimStepLease(_ context.Context, input ClaimLifecycleStepLeaseInput) (LifecycleStepAttempt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	attempt, err := r.stepByID(input.AttemptID)
	if err != nil {
		return LifecycleStepAttempt{}, err
	}
	now := time.Now()
	if lifecycleStepTerminal(attempt.Status) {
		return LifecycleStepAttempt{}, ErrLifecycleStepClosed
	}
	if attempt.LeaseRevision != input.ExpectedRevision ||
		(attempt.LeaseOwnerToken != "" && attempt.LeaseExpiresAt != nil && attempt.LeaseExpiresAt.After(now)) {
		return LifecycleStepAttempt{}, ErrLifecycleStepLeaseConflict
	}
	expires := now.Add(time.Duration(input.DurationMS) * time.Millisecond)
	attempt.LeaseOwnerToken = input.OwnerToken
	attempt.LeaseHeartbeatAt = &now
	attempt.LeaseExpiresAt = &expires
	attempt.LeaseRevision++
	r.replaceStep(*attempt)
	if r.failAfterLeaseClaimAction == attempt.LifecycleAction {
		r.failAfterLeaseClaimAction = ""
		return LifecycleStepAttempt{}, errLifecycleCoordinatorTestCrash
	}
	return *attempt, nil
}

func (r *lifecycleCoordinatorTestRepository) HeartbeatStepLease(_ context.Context, input HeartbeatLifecycleStepLeaseInput) (LifecycleStepAttempt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	attempt, err := r.stepByID(input.AttemptID)
	if err != nil {
		return LifecycleStepAttempt{}, err
	}
	if err := r.authorizeLease(*attempt, input.OwnerToken, input.Revision); err != nil {
		return LifecycleStepAttempt{}, err
	}
	if r.failHeartbeatOnce != nil && (r.failHeartbeatAction == "" || r.failHeartbeatAction == attempt.LifecycleAction) {
		err := r.failHeartbeatOnce
		r.failHeartbeatOnce = nil
		return LifecycleStepAttempt{}, err
	}
	now := time.Now()
	expires := now.Add(time.Duration(input.DurationMS) * time.Millisecond)
	attempt.LeaseHeartbeatAt = &now
	attempt.LeaseExpiresAt = &expires
	attempt.LeaseRevision++
	r.leaseHeartbeatCount++
	r.replaceStep(*attempt)
	if r.leaseHeartbeatNotify != nil && attempt.LifecycleAction != lifecycleCoordinatorHostGateAction {
		select {
		case r.leaseHeartbeatNotify <- struct{}{}:
		default:
		}
	}
	return *attempt, nil
}

func (r *lifecycleCoordinatorTestRepository) ReleaseStepLease(_ context.Context, input ReleaseLifecycleStepLeaseInput) (LifecycleStepAttempt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.operation.CompletedAt != nil {
		return LifecycleStepAttempt{}, ErrLifecycleOperationClosed
	}
	attempt, err := r.stepByID(input.AttemptID)
	if err != nil {
		return LifecycleStepAttempt{}, err
	}
	if err := r.authorizeLease(*attempt, input.OwnerToken, input.Revision); err != nil {
		return LifecycleStepAttempt{}, err
	}
	attempt.LeaseOwnerToken = ""
	attempt.LeaseExpiresAt = nil
	attempt.LeaseHeartbeatAt = nil
	attempt.LeaseRevision++
	r.replaceStep(*attempt)
	return *attempt, nil
}

func (r *lifecycleCoordinatorTestRepository) LatestStepAttempt(_ context.Context, _ int64, stepID string) (LifecycleStepAttempt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failLatestStepOnce {
		r.failLatestStepOnce = false
		return LifecycleStepAttempt{}, errLifecycleCoordinatorTestCrash
	}
	items := r.steps[stepID]
	if len(items) == 0 {
		return LifecycleStepAttempt{}, ErrLifecycleStepNotFound
	}
	return items[len(items)-1], nil
}

func (r *lifecycleCoordinatorTestRepository) statesSnapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.states...)
}

func (r *lifecycleCoordinatorTestRepository) terminalContextsSnapshot() []lifecycleCoordinatorTestContextRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]lifecycleCoordinatorTestContextRecord(nil), r.terminalContexts...)
}

func (r *lifecycleCoordinatorTestRepository) stepsSnapshot() []LifecycleStepAttempt {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]LifecycleStepAttempt, 0)
	for _, items := range r.steps {
		result = append(result, items...)
	}
	return result
}

// Caller holds r.mu.
func (r *lifecycleCoordinatorTestRepository) recordTerminalContext(ctx context.Context, method string) {
	if r.cancelDuringTerminal != nil {
		r.cancelDuringTerminal()
		r.cancelDuringTerminal = nil
	}
	deadline, hasDeadline := ctx.Deadline()
	record := lifecycleCoordinatorTestContextRecord{method: method, err: ctx.Err(), hasDeadline: hasDeadline}
	if hasDeadline {
		record.remaining = time.Until(deadline)
	}
	r.terminalContexts = append(r.terminalContexts, record)
}

func (r *lifecycleCoordinatorTestRepository) failNextTransition() {
	r.mu.Lock()
	r.failTransitionOnce = true
	r.mu.Unlock()
}

func (r *lifecycleCoordinatorTestRepository) failNextCompleteStep() {
	r.mu.Lock()
	r.failCompleteStepOnce = true
	r.mu.Unlock()
}

// Caller holds r.mu.
func (r *lifecycleCoordinatorTestRepository) authorizeLease(attempt LifecycleStepAttempt, owner string, revision int64) error {
	if attempt.LeaseOwnerToken == "" {
		if owner == "" && revision == 0 {
			return nil
		}
		return ErrLifecycleStepLeaseConflict
	}
	if attempt.LeaseOwnerToken != owner || attempt.LeaseRevision != revision {
		return ErrLifecycleStepLeaseConflict
	}
	if attempt.LeaseExpiresAt == nil || !attempt.LeaseExpiresAt.After(time.Now()) {
		return ErrLifecycleStepLeaseExpired
	}
	return nil
}

func (r *lifecycleCoordinatorTestRepository) expireOpenLease(checkpoint string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, items := range r.steps {
		for _, item := range items {
			if item.LeaseOwnerToken != "" && !lifecycleStepTerminal(item.Status) {
				item.Checkpoint = checkpoint
				expires := time.Now().Add(-time.Second)
				item.LeaseExpiresAt = &expires
				r.replaceStep(item)
				return
			}
		}
	}
}

func (r *lifecycleCoordinatorTestRepository) heartbeatCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.leaseHeartbeatCount
}

func (r *lifecycleCoordinatorTestRepository) actionLeaseRevisions() (int64, int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastProgressLeaseRevision, r.lastActionCompleteLeaseRevision
}

func (r *lifecycleCoordinatorTestRepository) stepByID(id int64) (*LifecycleStepAttempt, error) {
	for _, items := range r.steps {
		for index := range items {
			if items[index].ID == id {
				copy := items[index]
				return &copy, nil
			}
		}
	}
	return nil, ErrLifecycleStepNotFound
}

func (r *lifecycleCoordinatorTestRepository) replaceStep(value LifecycleStepAttempt) {
	items := r.steps[value.StepID]
	for index := range items {
		if items[index].ID == value.ID {
			items[index] = value
			r.steps[value.StepID] = items
			return
		}
	}
}
