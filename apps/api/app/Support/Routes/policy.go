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
	// RequestSizeBytes is the Host-enforced max request body (0 = platform default).
	RequestSizeBytes int64
	// CORSPolicy is a Host-named CORS profile; empty means platform default.
	CORSPolicy string
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

// RouteReplayResponseContract records the contract that actually governed the
// persisted response. A later response stage may be declared in the immutable
// plan yet never become applicable because its pre-invocation validation failed.
type RouteReplayResponseContract struct {
	StepIndex       int
	InvocationStage InvocationStage
	RouteID         string
	ContractVersion string
	ResponseSchema  string
}

type RouteIdempotencyReplay struct {
	Response              DispatchResponse
	Authorization         *RouteReplayAuthorization
	ResponseContractKnown bool
	ResponseContract      *RouteReplayResponseContract
}

type RouteIdempotencyCompletion struct {
	Response              DispatchResponse
	Authorization         *RouteReplayAuthorization
	ResponseContractKnown bool
	ResponseContract      *RouteReplayResponseContract
}

func cloneRouteReplayResponseContract(value *RouteReplayResponseContract) *RouteReplayResponseContract {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
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
