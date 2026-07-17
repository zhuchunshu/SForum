package routes

import "errors"

// resolvePlanRouteExecutionPolicy keeps current request execution on one
// immutable Registry revision. The external resolver is compatibility-only for
// legacy publications that predate plan-bound policies.
func resolvePlanRouteExecutionPolicy(
	plan RouteExecutionPlan,
	terminal RouteExecutionStep,
	legacy RoutePolicyResolver,
) (RouteExecutionPolicy, bool, error) {
	if policy, bound := plan.ExecutionPolicy(); bound {
		return policy, true, nil
	}
	if legacy == nil {
		return RouteExecutionPolicy{}, false, nil
	}
	policy, err := legacy.ResolveRouteExecutionPolicy(terminal)
	if errors.Is(err, ErrRoutePolicyNotFound) {
		return RouteExecutionPolicy{}, false, nil
	}
	if err != nil {
		return RouteExecutionPolicy{}, false, err
	}
	return policy, true, nil
}
