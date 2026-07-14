package http

import (
	"context"
	"encoding/json"
	"fmt"
	stdhttp "net/http"
	"net/url"
	"strings"

	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

type exactRouteV2Runtime interface {
	InvokeRouteInstance(context.Context, extensionsruntime.RuntimeInstanceIdentity, extensionsruntime.ProtocolV2RouteRequest) (extensionsruntime.ProtocolV2RouteResponse, error)
}

func (i *BufferedRouteStepInvoker) invokeProtocolV2(
	ctx context.Context,
	identity extensionsruntime.RuntimeInstanceIdentity,
	input routes.RouteInvocation,
) (routes.RouteInvocationResult, error) {
	runtime, ok := i.Runtime.(exactRouteV2Runtime)
	if !ok {
		return routes.RouteInvocationResult{}, ErrRouteRuntimeTarget
	}
	query, err := exactProtocolV2RouteQuery(input.Request.Query)
	if err != nil {
		return routes.RouteInvocationResult{}, err
	}
	body, present, err := exactProtocolV2RouteBody(input.Request.Body, input.Step.RequestSchema)
	if err != nil {
		return routes.RouteInvocationResult{}, err
	}
	headers := make(stdhttp.Header)
	copyRouteRequestHeaders(headers, input.Request.Headers)
	evidence := &routeTransportEvidence{commit: input.Commit}
	// A unary gRPC error cannot prove that the plugin did not receive the call.
	// Fence fallback before dispatch so a crash cannot create a second writer.
	evidence.markRequestStarted()
	response, err := runtime.InvokeRouteInstance(ctx, identity, extensionsruntime.ProtocolV2RouteRequest{
		RouteID: input.Step.RouteID, ContractVersion: input.Step.ContractVersion,
		Method: input.Request.Method, Path: input.Request.Path, Headers: headers,
		PathParameters: input.Request.Params, QueryParameters: query,
		RequestSchema: input.Step.RequestSchema, ResponseSchema: input.Step.ResponseSchema,
		Body: body, BodyPresent: present,
		Actor: extensionsruntime.NewProtocolV2RouteActor(
			input.Request.ActorID, input.Request.Authenticated, input.Request.Permissions,
		),
	})
	if err != nil {
		return evidence.result(), err
	}
	evidence.markResponseStarted()
	var responseBody []byte
	if response.BodyPresent {
		responseBody, err = json.Marshal(response.Body)
		if err != nil {
			return evidence.result(), fmt.Errorf("%w: encode typed route response: %v", ErrRouteRuntimeTarget, err)
		}
	}
	limit := i.ResponseLimit
	if limit <= 0 {
		limit = defaultRouteResponseLimit
	}
	if int64(len(responseBody)) > limit {
		return evidence.result(), ErrRouteResponseTooLarge
	}
	value := routes.DispatchResponse{
		Status: response.StatusCode, Headers: filteredRouteResponseHeaders(response.Headers), Body: responseBody,
	}
	result := evidence.result()
	result.Response = &value
	return result, nil
}

func exactProtocolV2RouteQuery(raw string) (map[string]string, error) {
	if raw == "" {
		return nil, nil
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid query parameters", ErrRouteRuntimeTarget)
	}
	result := make(map[string]string, len(values))
	for key, items := range values {
		// The v2 proto intentionally uses map<string,string>; silently dropping a
		// repeated value would change the signed route input.
		if len(items) != 1 {
			return nil, fmt.Errorf("%w: repeated query parameter %q", ErrRouteRuntimeTarget, key)
		}
		result[key] = items[0]
	}
	return result, nil
}

func exactProtocolV2RouteBody(raw []byte, schema string) (map[string]any, bool, error) {
	if len(raw) == 0 {
		return nil, false, nil
	}
	if strings.TrimSpace(schema) == "" {
		return nil, false, fmt.Errorf("%w: typed JSON body requires a request schema", ErrRouteRuntimeTarget)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil || body == nil {
		return nil, false, fmt.Errorf("%w: unary v2 body must be a JSON object", ErrRouteRuntimeTarget)
	}
	return body, true, nil
}
