package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	routeReplayAuthorizationSchema  = "sforum.route-replay-authorization@1"
	routeReplayMutationMaximumBytes = routeMutationPatchMaximumBytes
)

func BuildRouteReplayBinding(plan RouteExecutionPlan, request DispatchRequest) (RouteReplayBinding, error) {
	if !plan.Valid() {
		return RouteReplayBinding{}, ErrInvalidExecutionPlan
	}
	planDigest, err := routeReplayPlanDigest(plan)
	if err != nil {
		return RouteReplayBinding{}, err
	}
	baseDigest, err := routeReplayRequestDigest(request)
	if err != nil {
		return RouteReplayBinding{}, err
	}
	return RouteReplayBinding{PlanDigest: planDigest, BaseDigest: baseDigest}, nil
}

func newRouteReplayAuthorization(
	binding RouteReplayBinding,
	mutations []RouteReplayRequestMutation,
) (*RouteReplayAuthorization, error) {
	if len(mutations) == 0 {
		return nil, nil
	}
	result := &RouteReplayAuthorization{
		Schema: routeReplayAuthorizationSchema, PlanDigest: binding.PlanDigest, BaseDigest: binding.BaseDigest,
		RequestMutations: cloneRouteReplayRequestMutations(mutations),
	}
	if err := validateRouteReplayAuthorization(result); err != nil {
		return nil, err
	}
	return result, nil
}

func appendRouteReplayRequestMutation(
	binding RouteReplayBinding,
	mutations []RouteReplayRequestMutation,
	mutation RouteReplayRequestMutation,
) ([]RouteReplayRequestMutation, error) {
	next := append(mutations, mutation)
	// 请求阶段在 handler 之前完成；逐步校验聚合预算，避免插件产生副作用后
	// 才发现持久化 transcript 无法封装。
	if _, err := newRouteReplayAuthorization(binding, next); err != nil {
		return nil, err
	}
	return next, nil
}

func cloneRouteReplayAuthorization(value *RouteReplayAuthorization) *RouteReplayAuthorization {
	if value == nil {
		return nil
	}
	return &RouteReplayAuthorization{
		Schema: value.Schema, PlanDigest: value.PlanDigest, BaseDigest: value.BaseDigest,
		RequestMutations: cloneRouteReplayRequestMutations(value.RequestMutations),
	}
}

func cloneRouteReplayRequestMutations(values []RouteReplayRequestMutation) []RouteReplayRequestMutation {
	result := make([]RouteReplayRequestMutation, len(values))
	for index, value := range values {
		result[index] = RouteReplayRequestMutation{
			StepIndex: value.StepIndex, BeforeDigest: value.BeforeDigest, AfterDigest: value.AfterDigest,
			Operations: cloneRoutePatchOperations(value.Operations),
		}
	}
	return result
}

func cloneRoutePatchOperations(values []RoutePatchOperation) []RoutePatchOperation {
	result := make([]RoutePatchOperation, len(values))
	for index, value := range values {
		result[index] = RoutePatchOperation{
			Kind: value.Kind, Path: value.Path, Value: append([]byte(nil), value.Value...),
		}
	}
	return result
}

func validateRouteReplayAuthorization(value *RouteReplayAuthorization) error {
	if value == nil || value.Schema != routeReplayAuthorizationSchema ||
		!validRouteReplayDigest(value.PlanDigest) || !validRouteReplayDigest(value.BaseDigest) ||
		len(value.RequestMutations) == 0 {
		return fmt.Errorf("%w: replay mutation transcript is invalid", ErrDispatchIdempotencyUnavailable)
	}
	encoded, err := json.Marshal(value)
	if err != nil || len(encoded) > routeReplayMutationMaximumBytes {
		return fmt.Errorf("%w: replay mutation transcript exceeds its budget", ErrDispatchIdempotencyUnavailable)
	}
	previousIndex := -1
	for _, mutation := range value.RequestMutations {
		if mutation.StepIndex <= previousIndex || !validRouteReplayDigest(mutation.BeforeDigest) ||
			!validRouteReplayDigest(mutation.AfterDigest) {
			return fmt.Errorf("%w: replay mutation order is invalid", ErrDispatchIdempotencyUnavailable)
		}
		previousIndex = mutation.StepIndex
		for _, operation := range mutation.Operations {
			switch operation.Kind {
			case RoutePatchAdd, RoutePatchReplace:
				if len(operation.Value) == 0 || !json.Valid(operation.Value) {
					return fmt.Errorf("%w: replay mutation value is invalid", ErrDispatchIdempotencyUnavailable)
				}
			case RoutePatchRemove:
				if len(operation.Value) != 0 {
					return fmt.Errorf("%w: replay remove has a value", ErrDispatchIdempotencyUnavailable)
				}
			default:
				return fmt.Errorf("%w: replay mutation kind is invalid", ErrDispatchIdempotencyUnavailable)
			}
		}
	}
	return nil
}

func routeReplayAuthorizationMatchesPlan(
	value *RouteReplayAuthorization,
	binding RouteReplayBinding,
	sequence []routeInvocationExecution,
) bool {
	if validateRouteReplayAuthorization(value) != nil ||
		value.PlanDigest != binding.PlanDigest || value.BaseDigest != binding.BaseDigest {
		return false
	}
	expected := make([]int, 0)
	for _, execution := range sequence {
		if execution.stage == InvocationStageRequest {
			expected = append(expected, execution.index)
		}
	}
	if len(expected) != len(value.RequestMutations) {
		return false
	}
	for index, stepIndex := range expected {
		if value.RequestMutations[index].StepIndex != stepIndex {
			return false
		}
	}
	return true
}

func routeReplayPlanDigest(plan RouteExecutionPlan) (string, error) {
	chain := plan.Chain()
	for index := range chain {
		// Runtime instance ids are process admission identities, not artifact
		// contracts. Restarts of the same digest must not invalidate 24h replay.
		chain[index].Provider.Artifact.RuntimeInstanceID = ""
	}
	policy, policyBound := plan.ExecutionPolicy()
	var document []byte
	var err error
	if policyBound {
		document, err = json.Marshal(struct {
			Schema string               `json:"schema"`
			Chain  []RouteExecutionStep `json:"chain"`
			Policy RouteExecutionPolicy `json:"policy"`
		}{Schema: "sforum.required-route-plan@2", Chain: chain, Policy: policy})
	} else {
		// Legacy/unbound callers keep the exact v1 digest during the rolling
		// transition. Only an immutable policy-bound plan may use v2.
		document, err = json.Marshal(struct {
			Schema string               `json:"schema"`
			Chain  []RouteExecutionStep `json:"chain"`
		}{Schema: "sforum.required-route-plan@1", Chain: chain})
	}
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(document)
	return hex.EncodeToString(digest[:]), nil
}

func routeReplayRequestDigest(request DispatchRequest) (string, error) {
	query, err := canonicalRouteReplayQuery(request.Query)
	if err != nil {
		return "", err
	}
	headers, err := routeReplayDigestHeaders(request.Headers)
	if err != nil {
		return "", err
	}
	document, err := json.Marshal(struct {
		Schema  string            `json:"schema"`
		Method  string            `json:"method"`
		Path    string            `json:"path"`
		Query   string            `json:"query"`
		Headers http.Header       `json:"headers"`
		Body    []byte            `json:"body"`
		Params  map[string]string `json:"params"`
	}{
		Schema: "sforum.required-route-request@1", Method: request.Method, Path: request.Path,
		Query: query, Headers: headers, Body: request.Body, Params: request.Params,
	})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(document)
	return hex.EncodeToString(digest[:]), nil
}

func canonicalRouteReplayQuery(raw string) (string, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return "", err
	}
	// url.Values.Encode sorts keys but preserves each key's value order.
	return values.Encode(), nil
}

func routeReplayDigestHeaders(source http.Header) (http.Header, error) {
	connectionHeaders, err := routeMutationConnectionHeaderTokens(source)
	if err != nil {
		return nil, err
	}
	result := make(http.Header)
	for name, values := range source {
		canonical := strings.ToLower(strings.TrimSpace(name))
		if _, blocked := connectionHeaders[canonical]; blocked || routePolicyLiveCredentialHeader(canonical) ||
			canonical == "" || strings.HasPrefix(canonical, "x-sforum-") {
			continue
		}
		switch canonical {
		case "host", "content-length", "connection", "keep-alive", "proxy-authenticate", "proxy-connection",
			"te", "trailer", "transfer-encoding", "upgrade":
			continue
		}
		for _, value := range values {
			result.Add(canonical, value)
		}
	}
	return result, nil
}

func routeChainHasCredentialMutableRequestFields(chain []RouteExecutionStep) bool {
	for _, step := range chain {
		if step.Provider.Kind != ProviderPlugin || !requestStageAction(step.Action) {
			continue
		}
		for _, pointer := range step.MutableRequestFields {
			tokens, err := routeMutationPointerTokens(pointer)
			if err == nil && len(tokens) >= 2 && tokens[0] == "headers" && routePolicyLiveCredentialHeader(tokens[1]) {
				return true
			}
		}
	}
	return false
}

func validRouteReplayDigest(value string) bool {
	if len(value) != sha256.Size*2 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
