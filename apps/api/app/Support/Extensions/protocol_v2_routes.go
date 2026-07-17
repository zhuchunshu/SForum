package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	pluginv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/plugin/v2"
	protocolv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
	"golang.org/x/net/http/httpguts"
	"google.golang.org/protobuf/proto"
)

var (
	ErrProtocolV2RouteInvalid     = errors.New("protocol v2 route invocation is invalid")
	ErrProtocolV2RouteUnsupported = errors.New("protocol v2 unary route is unsupported")
	ErrProtocolV2RouteStream      = errors.New("protocol v2 route requires the streaming transport")
)

// ProtocolV2InvocationActor is Host-authored request authority shared by route
// and admin invocations. Plugin output can never add actor or permission data.
type ProtocolV2InvocationActor struct {
	UserID         int64
	PermissionKeys []string
}

// ProtocolV2RouteActor preserves the route transport API name.
type ProtocolV2RouteActor = ProtocolV2InvocationActor

func NewProtocolV2InvocationActor(userID int64, authenticated bool, permissions map[string]bool) *ProtocolV2InvocationActor {
	if !authenticated || userID <= 0 {
		return nil
	}
	keys := make([]string, 0, len(permissions))
	for key, allowed := range permissions {
		key = strings.TrimSpace(key)
		if allowed && key != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return &ProtocolV2InvocationActor{UserID: userID, PermissionKeys: keys}
}

// NewProtocolV2RouteActor converts an authenticated Host dispatch snapshot to
// the wire actor. Anonymous and inactive actors are represented by nil.
func NewProtocolV2RouteActor(userID int64, authenticated bool, permissions map[string]bool) *ProtocolV2RouteActor {
	return NewProtocolV2InvocationActor(userID, authenticated, permissions)
}

// ProtocolV2RouteRequest is transport-neutral and binds one unary call to the
// immutable Route Registry step selected by the Host.
type ProtocolV2RouteRequest struct {
	RouteID         string
	ContractVersion string
	Method          string
	Path            string
	Headers         http.Header
	PathParameters  map[string]string
	QueryParameters map[string]string
	RequestSchema   string
	ResponseSchema  string
	Body            map[string]any
	BodyPresent     bool
	Authority       ProtocolV2RequestAuthority
	Actor           *ProtocolV2RouteActor
	IdempotencyKey  string
	CorrelationID   string
	Timeout         time.Duration
}

type ProtocolV2RouteResponse struct {
	StatusCode    int
	Headers       http.Header
	Body          map[string]any
	BodyPresent   bool
	StreamFollows bool
}

type protocolV2RouteContextInvoker interface {
	InvokeRouteContext(context.Context, ProtocolV2RouteRequest) (ProtocolV2RouteResponse, error)
}

func (c *protocolV2Client) InvokeRouteContext(parent context.Context, input ProtocolV2RouteRequest) (ProtocolV2RouteResponse, error) {
	if c == nil || c.client == nil || c.identity == nil || parent == nil {
		return ProtocolV2RouteResponse{}, ErrProtocolV2RouteInvalid
	}
	if err := validateProtocolV2RouteRequest(input); err != nil {
		return ProtocolV2RouteResponse{}, err
	}
	if err := c.validateFrozenRoute(input); err != nil {
		return ProtocolV2RouteResponse{}, err
	}
	requestHeaders, err := protocolV2AuthorizedRequestHeaders(input.Headers, input.Authority)
	if err != nil {
		return ProtocolV2RouteResponse{}, wrapProtocolV2AuthorityError(ErrProtocolV2RouteInvalid, err)
	}
	authorityMode, guardKind, err := protocolV2WireRequestAuthority(input.Authority)
	if err != nil {
		return ProtocolV2RouteResponse{}, wrapProtocolV2AuthorityError(ErrProtocolV2RouteInvalid, err)
	}
	timeout := input.Timeout
	if timeout <= 0 {
		timeout = DefaultProtocolV2RequestTimeout
	}
	ctx, cancel := protocolV2Deadline(parent, timeout)
	defer cancel()

	requestContext, err := c.invocationRequestContext(ctx, input.CorrelationID, input.Actor, input.IdempotencyKey)
	if err != nil {
		return ProtocolV2RouteResponse{}, err
	}
	var body *protocolv2.TypedDocument
	if input.BodyPresent {
		schemaID, schemaVersion, err := protocolV2SchemaRef(input.RequestSchema)
		if err != nil {
			return ProtocolV2RouteResponse{}, fmt.Errorf("%w: %v", ErrProtocolV2RouteInvalid, err)
		}
		body, err = protocolV2Document(schemaID, schemaVersion, input.Body)
		if err != nil {
			return ProtocolV2RouteResponse{}, fmt.Errorf("%w: %v", ErrProtocolV2RouteInvalid, err)
		}
	}
	response, err := c.client.InvokeRoute(ctx, &pluginv2.RouteRequest{
		Context: requestContext, RouteId: input.RouteID, ContractVersion: input.ContractVersion,
		Method: input.Method, Path: input.Path, Headers: protocolV2RouteHeaders(requestHeaders),
		RequestAuthorityMode: authorityMode, GuardKind: guardKind,
		PathParameters:  cloneProtocolV2RouteParameters(input.PathParameters),
		QueryParameters: cloneProtocolV2RouteParameters(input.QueryParameters), Body: body,
	})
	if err != nil {
		if ctx.Err() != nil {
			return ProtocolV2RouteResponse{}, ctx.Err()
		}
		return ProtocolV2RouteResponse{}, err
	}
	if err := validateProtocolV2RouteResponseContext(response.GetContext(), requestContext); err != nil {
		return ProtocolV2RouteResponse{}, err
	}
	if err := protocolV2Error(response.GetError()); err != nil {
		return ProtocolV2RouteResponse{}, err
	}
	status := int(response.GetStatusCode())
	if status < 100 || status > 599 {
		return ProtocolV2RouteResponse{}, fmt.Errorf("%w: invalid response status %d", ErrProtocolV2RouteInvalid, status)
	}
	if response.GetStreamFollows() && response.GetBody() != nil {
		return ProtocolV2RouteResponse{}, fmt.Errorf("%w: streaming response cannot include a buffered body", ErrProtocolV2RouteInvalid)
	}
	if response.GetStreamFollows() {
		// The declared response schema applies to the stream representation; the
		// unary preflight authenticates status and headers only.
	} else if input.ResponseSchema == "" {
		if response.GetBody() != nil {
			return ProtocolV2RouteResponse{}, fmt.Errorf("%w: undeclared response body", ErrProtocolV2RouteInvalid)
		}
	} else if err := validateProtocolV2DocumentRef(response.GetBody(), input.ResponseSchema, "route response"); err != nil {
		return ProtocolV2RouteResponse{}, fmt.Errorf("%w: %v", ErrProtocolV2RouteInvalid, err)
	}
	headers, err := protocolV2RouteHTTPHeaders(response.GetHeaders())
	if err != nil {
		return ProtocolV2RouteResponse{}, err
	}
	result := ProtocolV2RouteResponse{
		StatusCode: status, Headers: headers, BodyPresent: response.GetBody() != nil,
		StreamFollows: response.GetStreamFollows(),
	}
	if result.BodyPresent {
		result.Body = protocolV2Values(response.GetBody())
	}
	return result, nil
}

func validateProtocolV2RouteRequest(input ProtocolV2RouteRequest) error {
	if strings.TrimSpace(input.RouteID) == "" || strings.TrimSpace(input.ContractVersion) == "" ||
		strings.TrimSpace(input.Method) == "" || !strings.HasPrefix(input.Path, "/") {
		return ErrProtocolV2RouteInvalid
	}
	if input.BodyPresent && strings.TrimSpace(input.RequestSchema) == "" {
		return fmt.Errorf("%w: typed request body requires a schema", ErrProtocolV2RouteInvalid)
	}
	if input.Actor != nil && input.Actor.UserID <= 0 {
		return fmt.Errorf("%w: authenticated actor id is invalid", ErrProtocolV2RouteInvalid)
	}
	if input.IdempotencyKey != "" && !validProtocolV2InvocationIdempotencyKey(input.IdempotencyKey) {
		return fmt.Errorf("%w: idempotency key is invalid", ErrProtocolV2RouteInvalid)
	}
	if err := validateProtocolV2RequestAuthority(input.Authority); err != nil {
		return wrapProtocolV2AuthorityError(ErrProtocolV2RouteInvalid, err)
	}
	return nil
}

func (c *protocolV2Client) validateFrozenRoute(input ProtocolV2RouteRequest) error {
	for _, route := range c.routes {
		if route.ID != input.RouteID || route.ContractVersion != input.ContractVersion ||
			route.RequestSchema != input.RequestSchema || route.ResponseSchema != input.ResponseSchema {
			continue
		}
		if route.Action == extensionmanifest.RouteActionGlobalMiddleware && len(route.Methods) == 0 {
			if err := c.validateFrozenRouteAuthority(route, input.Authority); err != nil {
				return wrapProtocolV2AuthorityError(ErrProtocolV2RouteInvalid, err)
			}
			return nil
		}
		for _, method := range route.Methods {
			if method == "*" || strings.EqualFold(method, input.Method) ||
				route.Mode == extensionmanifest.RouteModeHTTP && strings.EqualFold(input.Method, http.MethodHead) && strings.EqualFold(method, http.MethodGet) {
				if err := c.validateFrozenRouteAuthority(route, input.Authority); err != nil {
					return wrapProtocolV2AuthorityError(ErrProtocolV2RouteInvalid, err)
				}
				return nil
			}
		}
	}
	return fmt.Errorf("%w: route %q contract %q is not frozen for method %q", ErrProtocolV2RouteInvalid, input.RouteID, input.ContractVersion, input.Method)
}

func cloneProtocolV2Routes(values []extensions.ManifestRoute) []extensions.ManifestRoute {
	result := make([]extensions.ManifestRoute, len(values))
	copy(result, values)
	for index := range result {
		result[index].Methods = append([]string(nil), values[index].Methods...)
		result[index].MutableRequestFields = append([]string(nil), values[index].MutableRequestFields...)
		result[index].MutableResponseFields = append([]string(nil), values[index].MutableResponseFields...)
	}
	return result
}

func validateProtocolV2RouteResponseContext(response *protocolv2.ResponseContext, request *protocolv2.RequestContext) error {
	if response == nil || request == nil || response.GetRequestId() != request.GetRequestId() ||
		!proto.Equal(response.GetTrace(), request.GetTrace()) || !proto.Equal(response.GetExtension(), request.GetExtension()) {
		return fmt.Errorf("%w: response context does not match the exact runtime request", ErrProtocolV2RouteInvalid)
	}
	if response.GetServerTime() == nil || !response.GetServerTime().IsValid() {
		return fmt.Errorf("%w: response server time is invalid", ErrProtocolV2RouteInvalid)
	}
	return nil
}

func protocolV2RouteHeaders(headers http.Header) []*protocolv2.Header {
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]*protocolv2.Header, 0, len(names))
	for _, name := range names {
		result = append(result, &protocolv2.Header{Name: name, Values: append([]string(nil), headers.Values(name)...)})
	}
	return result
}

func protocolV2RouteHTTPHeaders(headers []*protocolv2.Header) (http.Header, error) {
	result := make(http.Header)
	for _, header := range headers {
		if header == nil || !httpguts.ValidHeaderFieldName(header.GetName()) {
			return nil, fmt.Errorf("%w: invalid response header name", ErrProtocolV2RouteInvalid)
		}
		for _, value := range header.GetValues() {
			if !httpguts.ValidHeaderFieldValue(value) {
				return nil, fmt.Errorf("%w: invalid response header value", ErrProtocolV2RouteInvalid)
			}
			result.Add(header.GetName(), value)
		}
	}
	return result, nil
}

func cloneProtocolV2RouteParameters(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

var _ protocolV2RouteContextInvoker = (*protocolV2Client)(nil)
