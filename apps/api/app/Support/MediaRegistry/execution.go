package mediaregistry

import (
	"context"
	"errors"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type Executor struct {
	registry   *Registry
	admission  RuntimeAdmission
	invoker    ProviderInvoker
	authority  ReceiptAuthority
	traces     TraceSink
	limits     ExecutionLimits
	callSlots  chan struct{}
	quarantine runtimeQuarantine
}

func NewExecutor(registry *Registry, admission RuntimeAdmission, invoker ProviderInvoker, authority ReceiptAuthority, traces TraceSink) *Executor {
	limits, _ := normalizeExecutionLimits(ExecutionLimits{})
	return &Executor{registry: registry, admission: admission, invoker: invoker, authority: authority, traces: traces, limits: limits, callSlots: make(chan struct{}, limits.MaxConcurrentCalls)}
}

func NewExecutorWithLimits(registry *Registry, admission RuntimeAdmission, invoker ProviderInvoker, authority ReceiptAuthority, traces TraceSink, limits ExecutionLimits) (*Executor, error) {
	if registry == nil || invoker == nil || authority == nil {
		return nil, ErrInvalid
	}
	normalized, err := normalizeExecutionLimits(limits)
	if err != nil {
		return nil, err
	}
	return &Executor{registry: registry, admission: admission, invoker: invoker, authority: authority, traces: traces, limits: normalized, callSlots: make(chan struct{}, normalized.MaxConcurrentCalls)}, nil
}

func (e *Executor) ExecuteOperation(ctx context.Context, operation BackgroundOperation, authorizer Authorizer) (ExecutionResult, error) {
	operation = cloneOperation(operation)
	started := time.Now()
	result := ExecutionResult{OperationKey: operation.Key, StepID: operation.StepID}
	step, stepFound := findPlanStep(operation.Plan, operation.StepID)
	artifact := Artifact{}
	stage := ""
	if stepFound {
		artifact = step.Processor.Artifact
		stage = step.Processor.Stage
	}
	outcome, reason := TraceFailedClosed, "invalid_operation"
	defer func() {
		e.appendTrace(TraceEvent{Revision: operation.Plan.Revision, OperationKey: operation.Key, PlanKind: operation.Plan.Kind, Stage: stage, StepID: operation.StepID, Outcome: outcome, Reason: reason, Duration: time.Since(started), Artifact: artifact})
	}()
	// 执行器配置错误与 plan 伪造/漂移分离：前者 ErrInvalid，后者 ErrPlanStale。
	if e == nil || e.registry == nil || e.invoker == nil || e.callSlots == nil || ctx == nil || authorizer == nil {
		return result, ErrInvalid
	}
	if e.authority == nil {
		return result, ErrReceiptAuthority
	}
	operationCtx, cancelOperation := context.WithTimeout(ctx, e.limits.OperationTimeout)
	defer cancelOperation()
	if operation.SchemaVersion != SchemaVersion || operation.Attempt < 1 || !stepFound || operation.Key != operationKey(operation.Plan, step) || operation.Attempt > effectiveMaxAttempts(step.Processor) {
		return result, ErrPlanStale
	}
	usage, err := validateOperationPrerequisites(operationCtx, e.authority, operation.Plan, step, operation.Prerequisites)
	if err != nil {
		reason = safeFailureReason(err)
		return result, err
	}
	remainingBudget, err := remainingOperationBudget(operation.Plan.Policy.Budget, usage)
	if err != nil {
		reason = safeFailureReason(err)
		return result, err
	}
	receiptLease, err := acquireReceiptLease(operationCtx, e.authority, operationReceiptBindings(step, operation.Prerequisites))
	if err != nil {
		reason = safeFailureReason(err)
		return result, err
	}
	receiptLeaseOwned := true
	defer func() {
		if receiptLeaseOwned {
			releaseReceiptLease(receiptLease)
		}
	}()
	receiptCtx, cancelReceipt := context.WithCancelCause(operationCtx)
	defer cancelReceipt(nil)
	receiptLeaseCtx, ok := safeReceiptLeaseContext(receiptLease)
	if !ok || receiptLeaseCtx.Err() != nil {
		if operationCtx.Err() != nil {
			return result, executionContextError(operationCtx)
		}
		return result, ErrReceiptInvalid
	}
	stopReceiptCancellation := context.AfterFunc(receiptLeaseCtx, func() { cancelReceipt(ErrReceiptInvalid) })
	defer stopReceiptCancellation()
	operationCtx = receiptCtx
	if err := operationCtx.Err(); err != nil {
		return result, executionContextError(operationCtx)
	}
	if err := normalizeExecutionError(operationCtx, e.registry.ValidatePlan(operationCtx, operation.Plan, authorizer)); err != nil {
		if errors.Is(err, ErrPermissionDenied) {
			outcome, reason = TraceDenied, "permission_denied"
		}
		return result, err
	}
	if !receiptLeaseCurrent(receiptLease) {
		return result, ErrReceiptInvalid
	}
	canonicalStep, found := e.registry.canonicalPlanStep(step)
	if !found {
		return result, ErrPlanStale
	}
	step = canonicalStep
	artifact = step.Processor.Artifact
	stage = step.Processor.Stage
	acquisition, err := acquireOperationClaim(operationCtx, e.authority, operationClaim(operation, step))
	if err != nil {
		reason = safeFailureReason(err)
		if errors.Is(err, ErrOperationBusy) {
			result.Retry = retrySameAttempt(step.Processor, operation.Attempt)
			outcome = TraceRetry
		}
		return result, err
	}
	if acquisition.Replay != nil {
		if err := normalizeExecutionError(operationCtx, e.registry.ValidatePlan(operationCtx, operation.Plan, authorizer)); err != nil {
			if errors.Is(err, ErrPermissionDenied) {
				outcome, reason = TraceDenied, "permission_denied"
			}
			return result, err
		}
		if !receiptLeaseCurrent(receiptLease) {
			return result, ErrReceiptInvalid
		}
		replayed, err := replayOperationCompletion(operationCtx, e.authority, operation, step, usage, remainingBudget, *acquisition.Replay)
		if err != nil {
			reason = safeFailureReason(err)
			return replayed, err
		}
		outcome = replayed.Receipt.Outcome
		reason = ""
		return replayed, nil
	}
	operationLease := acquisition.Lease
	operationLeaseOwned := true
	defer func() {
		if operationLeaseOwned {
			releaseOperationLease(operationLease)
		}
	}()
	claimCtx, cancelClaim := context.WithCancelCause(operationCtx)
	defer cancelClaim(nil)
	operationLeaseCtx, ok := safeOperationLeaseContext(operationLease)
	if !ok || operationLeaseCtx.Err() != nil {
		if operationCtx.Err() != nil {
			return result, executionContextError(operationCtx)
		}
		return result, ErrReceiptInvalid
	}
	stopClaimCancellation := context.AfterFunc(operationLeaseCtx, func() { cancelClaim(ErrReceiptInvalid) })
	defer stopClaimCancellation()
	operationCtx = claimCtx
	if !receiptLeaseCurrent(receiptLease) {
		return result, ErrReceiptInvalid
	}
	if err := operationCtx.Err(); err != nil {
		return result, executionContextError(operationCtx)
	}
	output, lease, lateCallback, invokeErr := e.invokeBounded(operationCtx, step.Processor.Artifact, Invocation{
		OperationKey: operation.Key, Attempt: operation.Attempt, PlanKind: operation.Plan.Kind,
		Purpose: operation.Plan.Purpose, Source: providerSource(operation.Plan.Source),
		Budget: remainingBudget, Step: clonePlanStep(step),
	})
	if lateCallback != nil {
		callerDone := make(chan struct{})
		defer close(callerDone)
		operationLeaseOwned = false
		receiptLeaseOwned = false
		go releaseExecutionLeasesAfter(lateCallback, callerDone, operationLease, receiptLease)
	}
	if invokeErr != nil {
		if lateCallback != nil || errors.Is(invokeErr, ErrRuntimeLeaseRelease) {
			outcome, reason = TraceFailedClosed, safeFailureReason(invokeErr)
			result.Retry = ClassifyRetry(step.Processor, invokeErr, operation.Attempt)
			return result, invokeErr
		}
		return e.handleOperationFailure(operationCtx, authorizer, result, operation, step, operationLease, receiptLease, invokeErr, &outcome, &reason)
	}
	leaseOwned := lease != nil
	defer func() {
		if leaseOwned {
			e.releaseLease(step.Processor.Artifact, lease)
		}
	}()
	finishFailure := func(failure error) (ExecutionResult, error) {
		if leaseOwned {
			if e.releaseLease(step.Processor.Artifact, lease) {
				leaseOwned = false
				failure = errors.Join(failure, ErrRuntimeLeaseRelease, ErrRuntimeUnavailable, ErrRuntimeQuarantined)
				outcome, reason = TraceFailedClosed, safeFailureReason(failure)
				return result, failure
			}
			leaseOwned = false
		}
		return e.handleOperationFailure(operationCtx, authorizer, result, operation, step, operationLease, receiptLease, failure, &outcome, &reason)
	}
	if !e.leaseCurrent(step.Processor.Artifact, lease) {
		return finishFailure(ErrRuntimeUnavailable)
	}
	if err := normalizeExecutionError(operationCtx, e.registry.ValidatePlan(operationCtx, operation.Plan, authorizer)); err != nil {
		if errors.Is(err, ErrPermissionDenied) {
			outcome, reason = TraceDenied, "permission_denied"
		}
		return finishFailure(err)
	}
	normalized, err := validateProviderOutput(output, operation.Plan, step, remainingBudget)
	if err != nil {
		outcome, reason = TraceFailedClosed, safeFailureReason(err)
		result.Retry = ClassifyRetry(step.Processor, err, operation.Attempt)
		return result, err
	}
	// Output validation may do bounded Host work. Repeat every mutable authority
	// fence immediately before releasing the result and exact runtime lease.
	if !e.leaseCurrent(step.Processor.Artifact, lease) {
		return finishFailure(ErrRuntimeUnavailable)
	}
	if err := normalizeExecutionError(operationCtx, e.registry.ValidatePlan(operationCtx, operation.Plan, authorizer)); err != nil {
		if errors.Is(err, ErrPermissionDenied) {
			outcome, reason = TraceDenied, "permission_denied"
		}
		return finishFailure(err)
	}
	if !e.leaseCurrent(step.Processor.Artifact, lease) {
		return finishFailure(ErrRuntimeUnavailable)
	}
	if err := operationCtx.Err(); err != nil {
		return finishFailure(executionContextError(operationCtx))
	}
	if e.releaseLease(step.Processor.Artifact, lease) {
		leaseOwned = false
		releaseErr := errors.Join(ErrRuntimeLeaseRelease, ErrRuntimeUnavailable, ErrRuntimeQuarantined)
		outcome, reason = TraceFailedClosed, safeFailureReason(releaseErr)
		return result, releaseErr
	}
	leaseOwned = false
	if err := operationCtx.Err(); err != nil {
		contextErr := executionContextError(operationCtx)
		outcome, reason = TraceFailedClosed, safeFailureReason(contextErr)
		return result, contextErr
	}
	if !receiptLeaseCurrent(receiptLease) || !operationLeaseCurrent(operationLease) {
		outcome, reason = TraceFailedClosed, "receipt_invalid"
		return result, ErrReceiptInvalid
	}
	if err := normalizeExecutionError(operationCtx, e.registry.ValidatePlan(operationCtx, operation.Plan, authorizer)); err != nil {
		if errors.Is(err, ErrPermissionDenied) {
			outcome, reason = TraceDenied, "permission_denied"
		} else {
			outcome, reason = TraceFailedClosed, safeFailureReason(err)
		}
		return result, err
	}
	completion, err := commitOperationCompletion(operationCtx, e.authority, operationLease, receiptLease, operation, step, normalized, TraceSucceeded, false, false)
	if err != nil {
		outcome, reason = TraceFailedClosed, safeFailureReason(err)
		return result, err
	}
	result.Output = normalized
	result.Receipt = completion.Receipt
	result.Retry = RetryDecision{Class: RetryNone}
	outcome, reason = TraceSucceeded, ""
	return result, nil
}

func (e *Executor) invoke(ctx context.Context, invocation Invocation) (output ProviderOutput, err error) {
	defer func() {
		if recover() != nil {
			output = ProviderOutput{}
			err = &ProviderError{Class: RetryCrash, Code: "provider.panic", Cause: errors.New("media provider panicked")}
		}
	}()
	return e.invoker.Invoke(ctx, invocation)
}

type providerCallResult struct {
	output ProviderOutput
	err    error
}

type providerCallOwnership struct {
	once            sync.Once
	decision        chan bool
	done            chan struct{}
	mu              sync.Mutex
	releasePanicked bool
}

func (ownership *providerCallOwnership) decide(retain bool) {
	if ownership != nil {
		ownership.once.Do(func() { ownership.decision <- retain })
	}
}

func (ownership *providerCallOwnership) decideAndWait(retain bool) bool {
	if ownership == nil {
		return false
	}
	ownership.decide(retain)
	<-ownership.done
	ownership.mu.Lock()
	defer ownership.mu.Unlock()
	return ownership.releasePanicked
}

func (e *Executor) invokeBounded(ctx context.Context, artifact Artifact, invocation Invocation) (ProviderOutput, RuntimeLease, <-chan struct{}, error) {
	if ctx == nil || ctx.Err() != nil {
		return ProviderOutput{}, nil, nil, executionContextError(ctx)
	}
	if err := e.quarantineError(artifact); err != nil {
		return ProviderOutput{}, nil, nil, err
	}
	callCtx, cancelCall := context.WithTimeout(ctx, e.limits.CallTimeout)
	defer cancelCall()
	select {
	case e.callSlots <- struct{}{}:
	case <-callCtx.Done():
		return ProviderOutput{}, nil, nil, executionContextError(callCtx)
	}
	if callCtx.Err() != nil {
		<-e.callSlots
		return ProviderOutput{}, nil, nil, executionContextError(callCtx)
	}
	if err := e.quarantineError(artifact); err != nil {
		<-e.callSlots
		return ProviderOutput{}, nil, nil, err
	}
	lease, err := e.acquire(callCtx, artifact)
	if err != nil {
		<-e.callSlots
		return ProviderOutput{}, nil, nil, err
	}
	if callCtx.Err() != nil {
		e.releaseLease(artifact, lease)
		<-e.callSlots
		return ProviderOutput{}, nil, nil, executionContextError(callCtx)
	}

	invokeCtx, cancelInvoke := context.WithCancelCause(callCtx)
	defer cancelInvoke(nil)
	stopLeaseCancellation := func() bool { return true }
	if lease != nil {
		leaseCtx, _, validLease := safeRuntimeLeaseState(lease)
		if !validLease {
			e.quarantine.mark(artifact)
			e.releaseLease(artifact, lease)
			<-e.callSlots
			if callCtx.Err() != nil {
				return ProviderOutput{}, nil, nil, errors.Join(executionContextError(callCtx), ErrRuntimeQuarantined)
			}
			return ProviderOutput{}, nil, nil, errors.Join(ErrRuntimeUnavailable, ErrRuntimeQuarantined)
		}
		stopLeaseCancellation = context.AfterFunc(leaseCtx, func() { cancelInvoke(ErrRuntimeUnavailable) })
	}
	defer stopLeaseCancellation()
	result := make(chan providerCallResult, 1)
	providerReturned := make(chan struct{})
	ownership := &providerCallOwnership{decision: make(chan bool, 1), done: make(chan struct{})}
	go func() {
		current := providerCallResult{}
		current.output, current.err = e.invoke(invokeCtx, invocation)
		close(providerReturned)
		result <- current
		if !<-ownership.decision {
			panicked := e.releaseLease(artifact, lease)
			ownership.mu.Lock()
			ownership.releasePanicked = panicked
			ownership.mu.Unlock()
		}
		<-e.callSlots
		close(ownership.done)
	}()

	finishContextFailure := func() (ProviderOutput, RuntimeLease, <-chan struct{}, error) {
		err := providerContextError(ctx, callCtx, lease)
		select {
		case <-providerReturned:
			if ownership.decideAndWait(false) {
				err = errors.Join(err, ErrRuntimeLeaseRelease, ErrRuntimeUnavailable, ErrRuntimeQuarantined)
			}
			return ProviderOutput{}, nil, nil, err
		default:
			e.quarantine.mark(artifact)
			err = errors.Join(err, ErrRuntimeQuarantined)
		}
		ownership.decide(false)
		return ProviderOutput{}, nil, ownership.done, err
	}

	select {
	case current := <-result:
		if invokeCtx.Err() != nil || deadlineReached(callCtx) {
			err := providerContextError(ctx, callCtx, lease)
			if ownership.decideAndWait(false) {
				err = errors.Join(err, ErrRuntimeLeaseRelease, ErrRuntimeUnavailable, ErrRuntimeQuarantined)
			}
			return ProviderOutput{}, nil, nil, err
		}
		if current.err != nil {
			if ownership.decideAndWait(false) {
				current.err = errors.Join(current.err, ErrRuntimeLeaseRelease, ErrRuntimeUnavailable, ErrRuntimeQuarantined)
			}
			return ProviderOutput{}, nil, nil, current.err
		}
		ownership.decideAndWait(true)
		return current.output, lease, nil, nil
	case <-invokeCtx.Done():
		cancelInvoke(context.Cause(invokeCtx))
		return finishContextFailure()
	}
}

func releaseExecutionLeasesAfter(callbackDone, callerDone <-chan struct{}, operationLease OperationLease, receiptLease ReceiptLease) {
	if callbackDone == nil || callerDone == nil || operationLease == nil || receiptLease == nil {
		return
	}
	<-callbackDone
	<-callerDone
	releaseOperationLease(operationLease)
	releaseReceiptLease(receiptLease)
}

func releaseReceiptLease(lease ReceiptLease) (panicked bool) {
	if lease == nil {
		return false
	}
	defer func() { panicked = recover() != nil }()
	lease.Release()
	return false
}

func safeReceiptLeaseContext(lease ReceiptLease) (ctx context.Context, ok bool) {
	if lease == nil {
		return nil, false
	}
	defer func() {
		if recover() != nil {
			ctx, ok = nil, false
		}
	}()
	ctx = lease.Context()
	return ctx, ctx != nil
}

func receiptLeaseCurrent(lease ReceiptLease) bool {
	ctx, ok := safeReceiptLeaseContext(lease)
	return ok && ctx.Err() == nil
}

func (e *Executor) acquire(ctx context.Context, artifact Artifact) (lease RuntimeLease, resultErr error) {
	if validCoreArtifactSeal(artifact) {
		return nil, nil
	}
	if e.admission == nil {
		return nil, ErrRuntimeUnavailable
	}
	available, panicked := safeAdmissionAvailable(e.admission, artifact)
	if panicked {
		e.quarantine.mark(artifact)
		if ctx.Err() != nil {
			return nil, errors.Join(executionContextError(ctx), ErrRuntimeQuarantined)
		}
		return nil, errors.Join(ErrRuntimeUnavailable, ErrRuntimeQuarantined)
	}
	if !available {
		return nil, ErrRuntimeUnavailable
	}
	lease, resultErr, panicked = safeAdmissionAcquire(e.admission, ctx, artifact)
	if panicked {
		e.quarantine.mark(artifact)
		if ctx.Err() != nil {
			return nil, errors.Join(executionContextError(ctx), ErrRuntimeQuarantined)
		}
		return nil, errors.Join(ErrRuntimeUnavailable, ErrRuntimeQuarantined)
	}
	leaseCtx, leasedArtifact, validLease := safeRuntimeLeaseState(lease)
	if resultErr != nil || !validLease || leasedArtifact != artifact || leaseCtx.Err() != nil {
		if lease != nil {
			if e.releaseLease(artifact, lease) {
				return nil, errors.Join(ErrRuntimeLeaseRelease, ErrRuntimeUnavailable, ErrRuntimeQuarantined)
			}
		}
		if !validLease {
			e.quarantine.mark(artifact)
			if ctx.Err() != nil {
				return nil, errors.Join(executionContextError(ctx), ErrRuntimeQuarantined)
			}
			return nil, errors.Join(ErrRuntimeUnavailable, ErrRuntimeQuarantined)
		}
		if ctx.Err() != nil {
			return nil, executionContextError(ctx)
		}
		return nil, ErrRuntimeUnavailable
	}
	return lease, nil
}

func safeAdmissionAvailable(admission RuntimeAdmission, artifact Artifact) (available, panicked bool) {
	defer func() {
		if recover() != nil {
			available, panicked = false, true
		}
	}()
	return admission.Available(artifact), false
}

func safeAdmissionAcquire(admission RuntimeAdmission, ctx context.Context, artifact Artifact) (lease RuntimeLease, err error, panicked bool) {
	defer func() {
		if recover() != nil {
			lease, err, panicked = nil, ErrRuntimeUnavailable, true
		}
	}()
	lease, err = admission.Acquire(ctx, artifact)
	return lease, err, false
}

func safeRuntimeLeaseState(lease RuntimeLease) (ctx context.Context, artifact Artifact, ok bool) {
	if lease == nil {
		return nil, Artifact{}, false
	}
	defer func() {
		if recover() != nil {
			ctx, artifact, ok = nil, Artifact{}, false
		}
	}()
	ctx = lease.Context()
	artifact = lease.Artifact()
	return ctx, artifact, ctx != nil
}

func (e *Executor) leaseCurrent(artifact Artifact, lease RuntimeLease) (current bool) {
	if e == nil || e.quarantine.contains(artifact) {
		return false
	}
	if validCoreArtifactSeal(artifact) {
		return lease == nil
	}
	leaseCtx, leasedArtifact, ok := safeRuntimeLeaseState(lease)
	if !ok {
		e.quarantine.mark(artifact)
		return false
	}
	available, panicked := safeAdmissionAvailable(e.admission, artifact)
	if panicked {
		e.quarantine.mark(artifact)
		return false
	}
	return leaseCtx.Err() == nil && leasedArtifact == artifact && available
}

func providerSource(source SourceAsset) ProviderSource {
	return ProviderSource{ID: source.ID, Digest: source.Digest, Kind: source.Kind, MIME: source.MIME, SizeBytes: source.SizeBytes, Immutable: source.Immutable}
}

func providerContextError(parent, call context.Context, lease RuntimeLease) error {
	if parent != nil && parent.Err() != nil {
		if errors.Is(context.Cause(parent), ErrReceiptInvalid) {
			return ErrReceiptInvalid
		}
		if errors.Is(parent.Err(), context.DeadlineExceeded) {
			return errors.Join(ErrExecutionTimeout, context.DeadlineExceeded)
		}
		return parent.Err()
	}
	if lease != nil {
		leaseCtx, _, ok := safeRuntimeLeaseState(lease)
		if !ok || leaseCtx.Err() != nil {
			return ErrRuntimeUnavailable
		}
	}
	if call != nil && errors.Is(call.Err(), context.DeadlineExceeded) {
		return errors.Join(ErrExecutionTimeout, context.DeadlineExceeded)
	}
	if call != nil && call.Err() != nil {
		return call.Err()
	}
	return ErrRuntimeUnavailable
}

func executionContextError(ctx context.Context) error {
	if ctx != nil && errors.Is(context.Cause(ctx), ErrReceiptInvalid) {
		return ErrReceiptInvalid
	}
	if ctx != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return errors.Join(ErrExecutionTimeout, context.DeadlineExceeded)
	}
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return ErrExecutionTimeout
}

func acquireReceiptLease(ctx context.Context, authority ReceiptAuthority, bindings []ReceiptBinding) (ReceiptLease, error) {
	if ctx == nil || authority == nil || len(bindings) == 0 {
		return nil, ErrReceiptAuthority
	}
	lease, err := authority.AcquireMediaReceipts(ctx, append([]ReceiptBinding(nil), bindings...))
	if err != nil || !receiptLeaseCurrent(lease) {
		if lease != nil {
			releaseReceiptLease(lease)
		}
		if ctx.Err() != nil {
			return nil, executionContextError(ctx)
		}
		return nil, ErrReceiptInvalid
	}
	return lease, nil
}

func normalizeExecutionError(ctx context.Context, err error) error {
	if err != nil && ctx != nil && ctx.Err() != nil {
		return executionContextError(ctx)
	}
	return err
}

func deadlineReached(ctx context.Context) bool {
	deadline, found := ctx.Deadline()
	return found && !time.Now().Before(deadline)
}

func releaseRuntimeLease(lease RuntimeLease) (panicked bool) {
	if lease == nil {
		return false
	}
	defer func() { panicked = recover() != nil }()
	lease.Release()
	return false
}

func (e *Executor) releaseLease(artifact Artifact, lease RuntimeLease) bool {
	panicked := releaseRuntimeLease(lease)
	if panicked && e != nil {
		e.quarantine.mark(artifact)
	}
	return panicked
}

func (e *Executor) handleOperationFailure(
	ctx context.Context,
	authorizer Authorizer,
	result ExecutionResult,
	operation BackgroundOperation,
	step PlanStep,
	operationLease OperationLease,
	receiptLease ReceiptLease,
	err error,
	outcome, reason *string,
) (ExecutionResult, error) {
	result, finalErr := e.handleFailure(result, step, operation.Attempt, err, outcome, reason)
	if finalErr != nil || !result.FallbackOriginal && !result.Skipped {
		return result, finalErr
	}
	// Optional-provider fallback/skip still advances the ordered pipeline, so it
	// receives a receipt only after a fresh Host permission and exact-plan fence.
	if validateErr := normalizeExecutionError(ctx, e.registry.ValidatePlan(ctx, operation.Plan, authorizer)); validateErr != nil {
		result.FallbackOriginal = false
		result.Skipped = false
		if errors.Is(validateErr, ErrPermissionDenied) {
			*outcome, *reason = TraceDenied, "permission_denied"
		} else {
			*outcome, *reason = TraceFailedClosed, safeFailureReason(validateErr)
		}
		return result, validateErr
	}
	receiptOutcome := TraceFallback
	if result.Skipped {
		receiptOutcome = TraceSkipped
	}
	if ctx.Err() != nil || !operationLeaseCurrent(operationLease) {
		result.FallbackOriginal = false
		result.Skipped = false
		contextErr := ErrReceiptInvalid
		if ctx.Err() != nil {
			contextErr = executionContextError(ctx)
		}
		*outcome, *reason = TraceFailedClosed, safeFailureReason(contextErr)
		return result, contextErr
	}
	completion, receiptErr := commitOperationCompletion(ctx, e.authority, operationLease, receiptLease, operation, step, ProviderOutput{}, receiptOutcome, result.FallbackOriginal, result.Skipped)
	if receiptErr != nil {
		result.FallbackOriginal = false
		result.Skipped = false
		*outcome, *reason = TraceFailedClosed, safeFailureReason(receiptErr)
		return result, receiptErr
	}
	result.Receipt = completion.Receipt
	return result, nil
}

func (e *Executor) handleFailure(result ExecutionResult, step PlanStep, attempt int, err error, outcome, reason *string) (ExecutionResult, error) {
	result.Retry = ClassifyRetry(step.Processor, err, attempt)
	if result.Retry.Retry {
		*outcome, *reason = TraceRetry, safeFailureReason(err)
		return result, err
	}
	// 权限拒绝保持 denied 轨迹；伪造/过期 plan 与畸形输出仍 fail-closed，且永不 fallback。
	if errors.Is(err, ErrPermissionDenied) {
		*outcome, *reason = TraceDenied, "permission_denied"
		return result, err
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, ErrPlanStale) || errors.Is(err, ErrOutputRejected) ||
		errors.Is(err, ErrMediaRejected) || errors.Is(err, ErrDeletionFence) || errors.Is(err, ErrBudgetExceeded) ||
		errors.Is(err, ErrPredecessorRequired) || errors.Is(err, ErrReceiptInvalid) || errors.Is(err, ErrReceiptAuthority) ||
		errors.Is(err, ErrRuntimeLeaseRelease) {
		*outcome, *reason = TraceFailedClosed, safeFailureReason(err)
		return result, err
	}
	switch step.Processor.FailureMode {
	case FailureFallbackOriginal:
		result.FallbackOriginal = true
		*outcome, *reason = TraceFallback, safeFailureReason(err)
		return result, nil
	case FailureSkip:
		result.Skipped = true
		*outcome, *reason = TraceSkipped, safeFailureReason(err)
		return result, nil
	default:
		*outcome, *reason = TraceFailedClosed, safeFailureReason(err)
		return result, err
	}
}

func validateProviderOutput(input ProviderOutput, plan Plan, step PlanStep, remaining Budget) (ProviderOutput, error) {
	return validateProviderOutputMode(input, plan, step, remaining, false)
}

func validateReplayedProviderOutput(input ProviderOutput, plan Plan, step PlanStep, remaining Budget) (ProviderOutput, error) {
	return validateProviderOutputMode(input, plan, step, remaining, true)
}

func validateProviderOutputMode(input ProviderOutput, plan Plan, step PlanStep, remaining Budget, replay bool) (ProviderOutput, error) {
	input.ReasonCode = strings.TrimSpace(input.ReasonCode)
	if len(input.ReasonCode) > maxReasonCodeBytes || !validPlainString(input.ReasonCode, maxReasonCodeBytes) {
		return ProviderOutput{}, ErrOutputRejected
	}
	input.ReasonCode = strings.ToLower(input.ReasonCode)
	if input.ReasonCode != "" && (!idPattern.MatchString(input.ReasonCode) || len(input.ReasonCode) > maxReasonCodeBytes) {
		return ProviderOutput{}, ErrOutputRejected
	}
	switch step.Processor.Stage {
	case StageValidate, StageScan:
		if input.Decision == DecisionReject {
			return ProviderOutput{}, ErrMediaRejected
		}
		if input.Decision != DecisionAllow || len(input.Metadata) > 0 || len(input.Variants) > 0 || input.CDNURL != "" || !input.RetainUntil.IsZero() {
			return ProviderOutput{}, ErrOutputRejected
		}
	case StageMetadata:
		metadata, err := normalizeMetadata(input.Metadata, remaining.MaxMetadataBytes)
		if err != nil {
			return ProviderOutput{}, err
		}
		input.Metadata = metadata
		if input.Decision != "" || len(input.Variants) > 0 || input.CDNURL != "" || !input.RetainUntil.IsZero() {
			return ProviderOutput{}, ErrOutputRejected
		}
	case StageTransform:
		variants, err := normalizeVariantOutputs(input.Variants, plan, step, remaining)
		if err != nil {
			return ProviderOutput{}, err
		}
		input.Variants = variants
		if input.Decision != "" || len(input.Metadata) > 0 || input.CDNURL != "" || !input.RetainUntil.IsZero() {
			return ProviderOutput{}, ErrOutputRejected
		}
	case StageCDN:
		value, err := normalizeCDNURL(input.CDNURL)
		if err != nil {
			return ProviderOutput{}, err
		}
		input.CDNURL = value
		if input.Decision != "" || len(input.Metadata) > 0 || len(input.Variants) > 0 || !input.RetainUntil.IsZero() {
			return ProviderOutput{}, ErrOutputRejected
		}
	case StageRetention:
		outsideCommitWindow := !replay && (input.RetainUntil.Before(time.Now().UTC().Add(-time.Minute)) || input.RetainUntil.After(time.Now().UTC().AddDate(100, 0, 0)))
		if input.RetainUntil.IsZero() || outsideCommitWindow || input.Decision != "" || len(input.Metadata) > 0 || len(input.Variants) > 0 || input.CDNURL != "" {
			return ProviderOutput{}, ErrOutputRejected
		}
		input.RetainUntil = input.RetainUntil.UTC()
	case StageBeforeDelete, StageAfterDelete:
		if input.Decision != "" || len(input.Metadata) > 0 || len(input.Variants) > 0 || input.CDNURL != "" || !input.RetainUntil.IsZero() {
			return ProviderOutput{}, ErrOutputRejected
		}
	default:
		return ProviderOutput{}, ErrOutputRejected
	}
	return cloneProviderOutput(input), nil
}

func normalizeMetadata(input map[string]string, limit int) (map[string]string, error) {
	if len(input) > maxMetadataEntries {
		return nil, ErrOutputRejected
	}
	result := make(map[string]string, len(input))
	total := 0
	for rawKey, rawValue := range input {
		key := strings.TrimSpace(rawKey)
		value := strings.TrimSpace(rawValue)
		if len(key) > maxStringBytes || !validPlainString(key, maxStringBytes) || !validPlainString(value, maxStringBytes) {
			return nil, ErrOutputRejected
		}
		key = strings.ToLower(key)
		if !idPattern.MatchString(key) {
			return nil, ErrOutputRejected
		}
		if _, duplicate := result[key]; duplicate {
			// map iteration order must not decide which case-variant value wins.
			return nil, ErrOutputRejected
		}
		total += len(key) + len(value)
		if total > limit {
			return nil, ErrBudgetExceeded
		}
		result[key] = value
	}
	return result, nil
}

func normalizeVariantOutputs(input []VariantOutput, plan Plan, step PlanStep, remaining Budget) ([]VariantOutput, error) {
	if len(input) > remaining.MaxVariants {
		return nil, ErrBudgetExceeded
	}
	if len(input) > len(step.Variants) {
		return nil, ErrOutputRejected
	}
	declarations := map[string]VariantContribution{}
	for _, variant := range step.Variants {
		declarations[variant.Name] = variant
	}
	seen := map[string]struct{}{}
	result := make([]VariantOutput, 0, len(input))
	for _, value := range input {
		value.Name = strings.TrimSpace(value.Name)
		value.Handle = strings.TrimSpace(value.Handle)
		value.Digest = strings.TrimSpace(value.Digest)
		value.SourceDigest = strings.TrimSpace(value.SourceDigest)
		if len(value.Name) > maxStringBytes || len(value.Handle) > maxStringBytes || len(value.Digest) != 64 || len(value.SourceDigest) != 64 {
			return nil, ErrOutputRejected
		}
		value.Name = strings.ToLower(value.Name)
		value.Digest = strings.ToLower(value.Digest)
		value.SourceDigest = strings.ToLower(value.SourceDigest)
		mimeType, err := normalizeExactMIME(value.MIME)
		if err != nil {
			return nil, ErrOutputRejected
		}
		value.MIME = mimeType
		declaration, found := declarations[value.Name]
		_, duplicate := seen[value.Name]
		if !found || duplicate || !validVariantHandle(value.Handle) || value.Handle == plan.Source.ID || !digestPattern.MatchString(value.Digest) || value.SourceDigest != plan.Source.Digest || value.MIME != declaration.OutputMIME || value.SizeBytes <= 0 || value.SizeBytes > plan.Policy.Budget.MaxFileBytes {
			return nil, ErrOutputRejected
		}
		seen[value.Name] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func remainingOperationBudget(budget Budget, usage OperationBudgetUsage) (Budget, error) {
	if !usageWithinBudget(usage, budget) {
		return Budget{}, ErrBudgetExceeded
	}
	budget.MaxMetadataBytes -= usage.MetadataBytes
	budget.MaxVariants -= usage.Variants
	return budget, nil
}

func validVariantHandle(value string) bool {
	return handlePattern.MatchString(value) && !strings.Contains(value, "..") && !strings.ContainsAny(value, "\\?#")
}
func normalizeCDNURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > maxURLBytes || !validPlainString(value, maxURLBytes) {
		return "", ErrOutputRejected
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" {
		return "", ErrOutputRejected
	}
	return parsed.String(), nil
}
func cloneProviderOutput(value ProviderOutput) ProviderOutput {
	if value.Metadata != nil {
		copied := make(map[string]string, len(value.Metadata))
		for k, v := range value.Metadata {
			copied[k] = v
		}
		value.Metadata = copied
	}
	value.Variants = append([]VariantOutput(nil), value.Variants...)
	return value
}

func safeFailureReason(err error) string {
	var provider *ProviderError
	switch {
	case errors.As(err, &provider) && validRetryClass(provider.Class):
		return provider.Class
	case errors.Is(err, ErrRuntimeQuarantined):
		return "runtime_quarantined"
	case errors.Is(err, ErrRuntimeUnavailable):
		return "runtime_unavailable"
	case errors.Is(err, ErrOperationBusy):
		return "operation_busy"
	case errors.Is(err, ErrExecutionTimeout), errors.Is(err, context.DeadlineExceeded):
		return "execution_timeout"
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case errors.Is(err, ErrDeletionFence):
		return "deletion_fence_required"
	case errors.Is(err, ErrPredecessorRequired):
		return "predecessor_required"
	case errors.Is(err, ErrReceiptInvalid):
		return "receipt_invalid"
	case errors.Is(err, ErrReceiptAuthority):
		return "receipt_authority_unavailable"
	case errors.Is(err, ErrBudgetExceeded):
		return "budget_exceeded"
	case errors.Is(err, ErrPlanStale):
		return "plan_stale"
	case errors.Is(err, ErrPermissionDenied):
		return "permission_denied"
	case errors.Is(err, ErrOutputRejected):
		return "output_rejected"
	case errors.Is(err, ErrMediaRejected):
		return "media_rejected"
	default:
		return "provider_failed"
	}
}
func (e *Executor) appendTrace(event TraceEvent) {
	if e == nil || e.traces == nil {
		return
	}
	if _, trusted := e.traces.(hostSynchronousTraceSink); trusted {
		appendMediaTraceSafely(e.traces, event)
		return
	}
	enqueueExternalMediaTrace(e.traces, event)
}

func appendMediaTraceSafely(sink TraceSink, event TraceEvent) {
	defer func() { _ = recover() }()
	sink.AppendMediaTrace(event)
}
