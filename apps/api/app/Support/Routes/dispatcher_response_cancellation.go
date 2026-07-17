package routes

import (
	"context"
	"errors"
	"time"
)

type routeObservedCallerCancellationError struct {
	cause error
}

func newRouteObservedCallerCancellation(cause error) error {
	return &routeObservedCallerCancellationError{cause: cause}
}

func (e *routeObservedCallerCancellationError) Error() string {
	if e == nil || e.cause == nil {
		return context.Canceled.Error()
	}
	return e.cause.Error()
}

func (e *routeObservedCallerCancellationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func routeObservedCallerCancellation(err error) (error, bool) {
	var observed *routeObservedCallerCancellationError
	if !errors.As(err, &observed) || observed == nil || observed.cause == nil {
		return nil, false
	}
	return observed.cause, true
}

func preserveRouteResponseOnObservedCallerCancellation(
	err error,
	stage InvocationStage,
	response *DispatchResponse,
) bool {
	_, observed := routeObservedCallerCancellation(err)
	return stage == InvocationStageResponse && response != nil && observed
}

func preserveRouteResponseOnCallerCancellation(
	ctx context.Context,
	err error,
	stage InvocationStage,
	response *DispatchResponse,
) bool {
	return stage == InvocationStageResponse && response != nil && routeCallerCancellation(ctx, err)
}

func routeCallerCancellation(ctx context.Context, err error) bool {
	if ctx == nil || err == nil {
		return false
	}
	ctxErr := ctx.Err()
	return ctxErr != nil && errors.Is(err, ctxErr)
}

func routeResponseFinalizationContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return context.WithTimeout(context.WithoutCancel(ctx), timeout)
}
