package routes

import "errors"

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
