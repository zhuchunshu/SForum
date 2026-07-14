package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	pluginv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/plugin/v2"
	protocolv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
)

var (
	ErrProtocolV2GuardInvalid = errors.New("protocol v2 guard invocation is invalid")
	ErrProtocolV2GuardDenied  = errors.New("protocol v2 guard denied the request")
)

// ProtocolV2GuardRequest binds a custom guard to one exact frozen route. The
// request headers have already passed the Host credential filter; raw browser
// credentials are a separate authority and are never inferred here.
type ProtocolV2GuardRequest struct {
	GuardID              string
	GuardContractVersion string
	RouteID              string
	RouteContractVersion string
	Method               string
	Path                 string
	Headers              http.Header
	PathParameters       map[string]string
	QueryParameters      map[string]string
	RequestSchema        string
	Body                 map[string]any
	BodyPresent          bool
	Actor                *ProtocolV2RouteActor
	CorrelationID        string
	Timeout              time.Duration
}

type protocolV2GuardContextInvoker interface {
	InvokeGuardContext(context.Context, ProtocolV2GuardRequest) error
}

func (c *protocolV2Client) InvokeGuardContext(parent context.Context, input ProtocolV2GuardRequest) error {
	if c == nil || c.client == nil || c.identity == nil || parent == nil {
		return ErrProtocolV2GuardInvalid
	}
	guard, route, err := c.validateFrozenGuard(input)
	if err != nil {
		return err
	}
	timeout := input.Timeout
	if timeout <= 0 {
		timeout = DefaultProtocolV2RequestTimeout
	}
	ctx, cancel := protocolV2Deadline(parent, timeout)
	defer cancel()

	requestContext := c.requestContext(ctx, input.CorrelationID)
	if input.Actor != nil {
		requestContext.Actor = &protocolv2.Actor{
			UserId: input.Actor.UserID, PermissionKeys: append([]string(nil), input.Actor.PermissionKeys...),
		}
	}
	var body *protocolv2.TypedDocument
	if input.BodyPresent {
		schemaID, schemaVersion, err := protocolV2SchemaRef(input.RequestSchema)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrProtocolV2GuardInvalid, err)
		}
		body, err = protocolV2Document(schemaID, schemaVersion, input.Body)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrProtocolV2GuardInvalid, err)
		}
	}
	headers, err := protocolV2GuardHeaders(input.Headers)
	if err != nil {
		return err
	}
	headers.Set("X-SForum-Guard-Route-ID", route.ID)
	headers.Set("X-SForum-Guard-Route-Contract", route.ContractVersion)
	headers.Set("X-SForum-Guard-Kind", guard.Kind)
	response, err := c.client.InvokeRoute(ctx, &pluginv2.RouteRequest{
		Context: requestContext, RouteId: guard.ID, ContractVersion: guard.ContractVersion,
		Method: input.Method, Path: input.Path, Headers: protocolV2RouteHeaders(headers),
		PathParameters:  cloneProtocolV2RouteParameters(input.PathParameters),
		QueryParameters: cloneProtocolV2RouteParameters(input.QueryParameters), Body: body,
	})
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	if err := validateProtocolV2RouteResponseContext(response.GetContext(), requestContext); err != nil {
		return fmt.Errorf("%w: %v", ErrProtocolV2GuardInvalid, err)
	}
	if err := protocolV2Error(response.GetError()); err != nil {
		return err
	}
	if response.GetStreamFollows() || response.GetBody() != nil || len(response.GetHeaders()) != 0 {
		return fmt.Errorf("%w: guard responses cannot mutate the request or response", ErrProtocolV2GuardInvalid)
	}
	switch int(response.GetStatusCode()) {
	case http.StatusNoContent:
		return nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrProtocolV2GuardDenied
	default:
		return fmt.Errorf("%w: invalid guard status %d", ErrProtocolV2GuardInvalid, response.GetStatusCode())
	}
}

func (c *protocolV2Client) validateFrozenGuard(
	input ProtocolV2GuardRequest,
) (extensions.ManifestGuard, extensions.ManifestRoute, error) {
	if strings.TrimSpace(input.GuardID) == "" || strings.TrimSpace(input.GuardContractVersion) == "" ||
		strings.TrimSpace(input.RouteID) == "" || strings.TrimSpace(input.RouteContractVersion) == "" ||
		strings.TrimSpace(input.Method) == "" || !strings.HasPrefix(input.Path, "/") ||
		input.BodyPresent && strings.TrimSpace(input.RequestSchema) == "" ||
		input.Actor != nil && input.Actor.UserID <= 0 {
		return extensions.ManifestGuard{}, extensions.ManifestRoute{}, ErrProtocolV2GuardInvalid
	}
	var guard *extensions.ManifestGuard
	for index := range c.guards {
		candidate := &c.guards[index]
		if candidate.ID == input.GuardID && candidate.ContractVersion == input.GuardContractVersion {
			if guard != nil {
				return extensions.ManifestGuard{}, extensions.ManifestRoute{}, ErrProtocolV2GuardInvalid
			}
			guard = candidate
		}
	}
	if guard == nil || (guard.Kind != "custom" && guard.Kind != "raw_request") {
		return extensions.ManifestGuard{}, extensions.ManifestRoute{}, ErrProtocolV2GuardInvalid
	}
	var route *extensions.ManifestRoute
	for index := range c.routes {
		candidate := &c.routes[index]
		if candidate.ID != input.RouteID || candidate.ContractVersion != input.RouteContractVersion ||
			candidate.Guard != guard.ID || candidate.RequestSchema != input.RequestSchema ||
			!protocolV2RouteMethodMatches(*candidate, input.Method) {
			continue
		}
		if route != nil {
			return extensions.ManifestGuard{}, extensions.ManifestRoute{}, ErrProtocolV2GuardInvalid
		}
		route = candidate
	}
	if route == nil {
		return extensions.ManifestGuard{}, extensions.ManifestRoute{}, ErrProtocolV2GuardInvalid
	}
	return *guard, *route, nil
}

func protocolV2RouteMethodMatches(route extensions.ManifestRoute, method string) bool {
	if route.Action == "global_middleware" && len(route.Methods) == 0 {
		return true
	}
	for _, declared := range route.Methods {
		if declared == "*" || strings.EqualFold(declared, method) ||
			route.Mode == "http" && strings.EqualFold(method, http.MethodHead) && strings.EqualFold(declared, http.MethodGet) {
			return true
		}
	}
	return false
}

func protocolV2GuardHeaders(source http.Header) (http.Header, error) {
	result := make(http.Header)
	for name, values := range source {
		canonical := strings.ToLower(strings.TrimSpace(name))
		if strings.HasPrefix(canonical, "x-sforum-") {
			return nil, fmt.Errorf("%w: reserved guard header", ErrProtocolV2GuardInvalid)
		}
		switch canonical {
		case "", "cookie", "authorization", "proxy-authorization":
			return nil, fmt.Errorf("%w: request credentials require separate raw authority", ErrProtocolV2GuardInvalid)
		}
		for _, value := range values {
			result.Add(name, value)
		}
	}
	return result, nil
}

func cloneProtocolV2Guards(values []extensions.ManifestGuard) []extensions.ManifestGuard {
	result := append([]extensions.ManifestGuard(nil), values...)
	for index := range result {
		result[index].Permissions = append([]string(nil), values[index].Permissions...)
	}
	return result
}

var _ protocolV2GuardContextInvoker = (*protocolV2Client)(nil)
