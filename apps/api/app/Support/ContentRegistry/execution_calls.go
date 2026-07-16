package contentregistry

import (
	"context"
	"errors"
	"time"
)

type providerCallResult[T any] struct {
	value T
	err   error
}

type admissionAcquireResult struct {
	lease    AdmissionLease
	leaseCtx context.Context
	err      error
}

type ownedAdmissionLease struct {
	lease    AdmissionLease
	leaseCtx context.Context
	artifact Artifact
}

func invokeProvider[T any](
	e *Executor,
	ctx context.Context,
	plan executionPlan,
	step plannedBinding,
	permission ExecutionRequest,
	operation string,
	call func(context.Context) (T, error),
	finalize func(context.Context, T) (T, error),
) (T, error) {
	var zero T
	started := time.Now()
	claim := e.permissionClaim(plan, step, permission, operation)
	if err := e.authorize(ctx, permission.Permission, claim); err != nil {
		e.recordTrace(plan, step, operation, traceOutcome(err), time.Since(started))
		return zero, err
	}
	if err := e.runtimeQuarantineError(step.contribution.Artifact); err != nil {
		e.recordTrace(plan, step, operation, traceOutcome(err), time.Since(started))
		return zero, err
	}
	callCtx, cancel := context.WithTimeout(ctx, e.limits.CallTimeout)
	defer cancel()
	select {
	case e.runtimeSlots <- struct{}{}:
	case <-callCtx.Done():
		err := executionContextError(callCtx)
		e.recordTrace(plan, step, operation, traceOutcome(err), time.Since(started))
		return zero, err
	}
	if err := e.runtimeQuarantineError(step.contribution.Artifact); err != nil {
		<-e.runtimeSlots
		e.recordTrace(plan, step, operation, traceOutcome(err), time.Since(started))
		return zero, err
	}
	result := make(chan providerCallResult[T], 1)
	go func() {
		outcome := providerCallResult[T]{}
		var lease AdmissionLease
		var leaseCtx context.Context
		defer func() {
			if recover() != nil {
				e.quarantine.mark(step.contribution.Artifact)
				outcome = providerCallResult[T]{err: errors.Join(ErrRuntimeUnavailable, ErrRuntimeQuarantined)}
			}
			if lease != nil && releaseAdmissionLease(lease) {
				e.quarantine.mark(step.contribution.Artifact)
				outcome = providerCallResult[T]{err: errors.Join(ErrRuntimeUnavailable, ErrRuntimeQuarantined)}
			}
			<-e.runtimeSlots
			result <- outcome
		}()

		var acquireErr error
		lease, leaseCtx, acquireErr = acquireContentExecution(
			e.admission, callCtx, contentAdmissionRequest(plan, step, operation),
		)
		if acquireErr != nil {
			if callCtx.Err() != nil {
				outcome.err = executionContextError(callCtx)
			} else {
				outcome.err = ErrRuntimeUnavailable
			}
			return
		}
		if quarantineErr := e.runtimeQuarantineError(step.contribution.Artifact); quarantineErr != nil {
			outcome.err = quarantineErr
			return
		}
		if callCtx.Err() != nil {
			outcome.err = executionContextError(callCtx)
			return
		}

		// A backend lease context may be narrower than the Host deadline, but it
		// may never widen it. Provider callbacks therefore observe cancellation
		// from either authority.
		providerCtx, cancelProvider := context.WithCancel(callCtx)
		stopLeaseCancellation, contextPanicked := afterContentContextSafely(leaseCtx, cancelProvider)
		if contextPanicked {
			cancelProvider()
			e.quarantine.mark(step.contribution.Artifact)
			outcome.err = errors.Join(ErrRuntimeUnavailable, ErrRuntimeQuarantined)
			return
		}
		defer cancelProvider()
		defer stopLeaseCancellation()
		if providerCtx.Err() != nil {
			leaseErr, leasePanicked := contentContextErrSafely(leaseCtx)
			if leasePanicked || leaseErr != nil {
				if leasePanicked {
					e.quarantine.mark(step.contribution.Artifact)
					outcome.err = errors.Join(ErrRuntimeUnavailable, ErrRuntimeQuarantined)
				} else {
					outcome.err = ErrRuntimeUnavailable
				}
			} else {
				outcome.err = executionContextError(callCtx)
			}
			return
		}
		providerPanicked := false
		func() {
			defer func() {
				if recover() != nil {
					providerPanicked = true
				}
			}()
			outcome.value, outcome.err = call(providerCtx)
		}()
		leaseErr, leaseContextPanicked := contentContextErrSafely(leaseCtx)
		switch {
		case providerPanicked:
			outcome = providerCallResult[T]{err: ErrProviderPanic}
		case leaseContextPanicked:
			e.quarantine.mark(step.contribution.Artifact)
			outcome = providerCallResult[T]{err: errors.Join(ErrRuntimeUnavailable, ErrRuntimeQuarantined)}
		case leaseErr != nil:
			outcome = providerCallResult[T]{err: ErrRuntimeUnavailable}
		case callCtx.Err() != nil:
			outcome = providerCallResult[T]{err: executionContextError(callCtx)}
		case outcome.err != nil:
			outcome = providerCallResult[T]{err: ErrProviderFailed}
		}
		leaseErr, leaseContextPanicked = contentContextErrSafely(leaseCtx)
		if leaseContextPanicked {
			e.quarantine.mark(step.contribution.Artifact)
			outcome = providerCallResult[T]{err: errors.Join(ErrRuntimeUnavailable, ErrRuntimeQuarantined)}
		} else if leaseErr != nil {
			outcome = providerCallResult[T]{err: ErrRuntimeUnavailable}
		} else if callCtx.Err() != nil {
			outcome = providerCallResult[T]{err: executionContextError(callCtx)}
		}
	}()
	select {
	case outcome := <-result:
		if callCtx.Err() != nil || deadlineReached(callCtx) {
			err := executionContextError(callCtx)
			e.recordTrace(plan, step, operation, traceOutcome(err), time.Since(started))
			return zero, err
		}
		if outcome.err != nil {
			e.recordTrace(plan, step, operation, traceOutcome(outcome.err), time.Since(started))
			return zero, outcome.err
		}
		if finalize != nil {
			func() {
				defer func() {
					if recover() != nil {
						outcome = providerCallResult[T]{err: ErrExecutionInvalid}
					}
				}()
				outcome.value, outcome.err = finalize(callCtx, outcome.value)
			}()
			if callCtx.Err() != nil || deadlineReached(callCtx) {
				err := executionContextError(callCtx)
				e.recordTrace(plan, step, operation, traceOutcome(err), time.Since(started))
				return zero, err
			}
			if outcome.err != nil {
				e.recordTrace(plan, step, operation, traceOutcome(outcome.err), time.Since(started))
				return zero, outcome.err
			}
		}
		e.recordTrace(plan, step, operation, TraceSucceeded, time.Since(started))
		return outcome.value, nil
	case <-callCtx.Done():
		err := executionContextError(callCtx)
		select {
		case <-result:
			// The provider exited and released its lease; only its late value is discarded.
		default:
			// Until the ownership goroutine reports completion, the Host cannot
			// prove that admission/provider/release work stopped. Quarantine the
			// exact artifact conservatively.
			e.quarantine.mark(step.contribution.Artifact)
			err = errors.Join(err, ErrRuntimeQuarantined)
		}
		e.recordTrace(plan, step, operation, traceOutcome(err), time.Since(started))
		return zero, err
	}
}

func (e *Executor) checkNonCallStep(
	ctx context.Context,
	plan executionPlan,
	step plannedBinding,
	request ExecutionRequest,
	operation string,
) error {
	_, err := invokeProvider(e, ctx, plan, step, request, operation,
		func(context.Context) (struct{}, error) { return struct{}{}, nil }, nil)
	return err
}

// invokeHostCallback bounds Host policy/schema work separately from runtime
// callbacks, so a stalled Host policy cannot consume runtime admission capacity.
func invokeHostCallback(e *Executor, ctx context.Context, callback func(context.Context) error) error {
	if e == nil || ctx == nil || callback == nil {
		return ErrExecutionInvalid
	}
	select {
	case e.hostSlots <- struct{}{}:
	case <-ctx.Done():
		return executionContextError(ctx)
	}
	result := make(chan error, 1)
	go func() {
		defer func() { <-e.hostSlots }()
		var callbackErr error
		func() {
			defer func() {
				if recover() != nil {
					callbackErr = ErrExecutionInvalid
				}
			}()
			callbackErr = callback(ctx)
		}()
		result <- callbackErr
	}()
	select {
	case callbackErr := <-result:
		if ctx.Err() != nil || deadlineReached(ctx) {
			return executionContextError(ctx)
		}
		return callbackErr
	case <-ctx.Done():
		return executionContextError(ctx)
	}
}

// acquireOwnedAdmission keeps one runtime slot until release. If acquisition
// ignores cancellation, the owner goroutine retains the slot and releases any
// late lease instead of orphaning runtime authority.
func (e *Executor) acquireOwnedAdmission(
	ctx context.Context,
	request AdmissionRequest,
	artifact Artifact,
) (*ownedAdmissionLease, error) {
	select {
	case e.runtimeSlots <- struct{}{}:
	case <-ctx.Done():
		return nil, executionContextError(ctx)
	}
	result := make(chan admissionAcquireResult)
	go func() {
		lease, leaseCtx, err := acquireContentExecution(e.admission, ctx, request)
		if err != nil {
			if lease != nil && releaseAdmissionLease(lease) {
				e.quarantine.mark(artifact)
				err = errors.Join(ErrRuntimeUnavailable, ErrRuntimeQuarantined)
			} else {
				err = ErrRuntimeUnavailable
			}
			<-e.runtimeSlots
			select {
			case result <- admissionAcquireResult{err: err}:
			case <-ctx.Done():
			}
			return
		}
		select {
		case result <- admissionAcquireResult{lease: lease, leaseCtx: leaseCtx}:
			// The receiver now owns both the lease and its runtime slot.
		case <-ctx.Done():
			if releaseAdmissionLease(lease) {
				e.quarantine.mark(artifact)
			}
			<-e.runtimeSlots
		}
	}()
	select {
	case acquired := <-result:
		if acquired.err != nil {
			return nil, acquired.err
		}
		return &ownedAdmissionLease{lease: acquired.lease, leaseCtx: acquired.leaseCtx, artifact: artifact}, nil
	case <-ctx.Done():
		e.quarantine.mark(artifact)
		return nil, errors.Join(executionContextError(ctx), ErrRuntimeUnavailable, ErrRuntimeQuarantined)
	}
}

func (e *Executor) releaseOwnedAdmission(ctx context.Context, owned *ownedAdmissionLease) error {
	if owned == nil || owned.lease == nil {
		return nil
	}
	lease := owned.lease
	owned.lease = nil
	result := make(chan bool, 1)
	go func() {
		panicked := releaseAdmissionLease(lease)
		<-e.runtimeSlots
		result <- panicked
	}()
	select {
	case panicked := <-result:
		if panicked {
			e.quarantine.mark(owned.artifact)
			return errors.Join(ErrRuntimeUnavailable, ErrRuntimeQuarantined)
		}
		return nil
	case <-ctx.Done():
		e.quarantine.mark(owned.artifact)
		return errors.Join(executionContextError(ctx), ErrRuntimeUnavailable, ErrRuntimeQuarantined)
	}
}

func acquireContentExecution(
	admission RuntimeAdmission,
	ctx context.Context,
	request AdmissionRequest,
) (lease AdmissionLease, leaseCtx context.Context, err error) {
	defer func() {
		if recover() != nil {
			leaseCtx = nil
			err = ErrRuntimeUnavailable
		}
	}()
	if admission == nil || ctx == nil {
		return nil, nil, ErrRuntimeUnavailable
	}
	if parentErr, panicked := contentContextErrSafely(ctx); panicked || parentErr != nil {
		return nil, nil, ErrRuntimeUnavailable
	}
	lease, err = admission.AcquireContentExecution(ctx, request)
	if err != nil || lease == nil {
		return lease, nil, ErrRuntimeUnavailable
	}
	if parentErr, panicked := contentContextErrSafely(ctx); panicked || parentErr != nil {
		return lease, nil, ErrRuntimeUnavailable
	}
	leaseCtx = lease.CallContext()
	if leaseCtx == nil {
		return lease, leaseCtx, ErrRuntimeUnavailable
	}
	parentErr, parentPanicked := contentContextErrSafely(ctx)
	leaseErr, leasePanicked := contentContextErrSafely(leaseCtx)
	if parentPanicked || leasePanicked || parentErr != nil || leaseErr != nil {
		return lease, leaseCtx, ErrRuntimeUnavailable
	}
	return lease, leaseCtx, nil
}

func contentContextErrSafely(ctx context.Context) (err error, panicked bool) {
	if ctx == nil {
		return ErrRuntimeUnavailable, false
	}
	defer func() {
		if recover() != nil {
			err = ErrRuntimeUnavailable
			panicked = true
		}
	}()
	return ctx.Err(), false
}

func afterContentContextSafely(ctx context.Context, callback func()) (stop func() bool, panicked bool) {
	defer func() {
		if recover() != nil {
			stop = nil
			panicked = true
		}
	}()
	return context.AfterFunc(ctx, callback), false
}
