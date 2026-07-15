package routes

import (
	"context"
	"errors"
)

var ErrRoutePolicyNotFound = errors.New("routes: exact route policy is not published")

// RouteExecutionPolicy contains only Host-enforced facts needed on the request
// path. Documentation-specific source metadata remains in ExtensionOpenAPI.
type RouteExecutionPolicy struct {
	RateLimit           string
	Idempotency         string
	IdempotencyRequired bool
}

type RoutePolicyResolver interface {
	ResolveRouteExecutionPolicy(RouteExecutionStep) (RouteExecutionPolicy, error)
}

type RouteIdempotencyLease interface {
	Complete(context.Context, DispatchResponse) error
	Abort(context.Context) error
}

// RouteIdempotencyController owns the durable replay lease while Dispatcher
// remains transport-neutral and keeps one resolved execution plan.
type RouteIdempotencyController interface {
	Begin(
		context.Context,
		RouteExecutionPlan,
		RouteExecutionStep,
		RouteExecutionPolicy,
		DispatchRequest,
	) (RouteIdempotencyLease, *DispatchResponse, error)
}
