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

type RouteReplayRequestMutation struct {
	StepIndex    int
	BeforeDigest string
	AfterDigest  string
	Operations   []RoutePatchOperation
}

// RouteReplayAuthorization contains only Host-validated patch evidence. The
// replaying request always supplies live actor, permission, client, and
// credential authority; modifier plugins are never invoked a second time.
type RouteReplayAuthorization struct {
	Schema           string
	PlanDigest       string
	BaseDigest       string
	RequestMutations []RouteReplayRequestMutation
}

type RouteReplayBinding struct {
	PlanDigest string
	BaseDigest string
}

type RouteIdempotencyReplay struct {
	Response      DispatchResponse
	Authorization *RouteReplayAuthorization
}

type RouteIdempotencyCompletion struct {
	Response      DispatchResponse
	Authorization *RouteReplayAuthorization
}

type RouteIdempotencyLease interface {
	Complete(context.Context, RouteIdempotencyCompletion) error
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
	) (RouteIdempotencyLease, *RouteIdempotencyReplay, error)
}

type RouteMutationReplayCapability interface {
	MutationReplayAvailable() bool
}
