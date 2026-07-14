package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	stdhttp "net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/gofiber/fiber/v3"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

var (
	ErrRouteLoginRequired     = errors.New("route dispatcher login required")
	ErrRouteGuestRequired     = errors.New("route dispatcher guest required")
	ErrRoutePermissionDenied  = errors.New("route dispatcher permission denied")
	ErrRouteGuardUnavailable  = errors.New("route dispatcher guard unavailable")
	ErrRouteSchemaUnavailable = errors.New("route dispatcher schema unavailable")
	ErrRouteRuntimeArtifact   = errors.New("route dispatcher runtime artifact mismatch")
	ErrRouteRuntimeTarget     = errors.New("route dispatcher runtime target is invalid")
	ErrRouteResponseTooLarge  = errors.New("route dispatcher response exceeds buffer limit")
)

const defaultRouteResponseLimit int64 = 8 << 20

type RouteActorLoader func(fiber.Ctx) (identity.Actor, error)

// RouteSchemaCatalog owns compiled JSON Schema contracts. A missing catalog
// deliberately rejects declared schemas instead of treating their ids as prose.
type RouteSchemaCatalog interface {
	ValidateRouteSchema(context.Context, routes.PluginArtifact, string, string, string, string, string, string, string, string, int, []byte) error
}

type ExactRouteRuntime interface {
	InspectRuntimeInstance(extensionsruntime.RuntimeInstanceIdentity) (extensionsruntime.RuntimeInstanceSnapshot, error)
	AcquireRuntimeCall(context.Context, extensionsruntime.RuntimeInstanceIdentity, extensionsruntime.RuntimeCallClass) (*extensionsruntime.RuntimeAdmissionLease, error)
}

type HostRouteGuardAuthorizer struct{}

func (HostRouteGuardAuthorizer) Authorize(
	_ context.Context,
	_ routes.RouteExecutionPlan,
	step routes.RouteExecutionStep,
	request routes.DispatchRequest,
) error {
	switch step.Guard {
	case extensionmanifest.GuardCorePublic:
		return nil
	case extensionmanifest.GuardCoreLogin:
		if request.Authenticated {
			return nil
		}
		return ErrRouteLoginRequired
	case extensionmanifest.GuardCoreGuest:
		if !request.Authenticated {
			return nil
		}
		return ErrRouteGuestRequired
	case extensionmanifest.GuardCorePermission:
		if !request.Authenticated {
			return ErrRouteLoginRequired
		}
		if request.Permissions["*"] || request.Permissions[step.Permission] {
			return nil
		}
		return ErrRoutePermissionDenied
	case extensionmanifest.GuardCoreInherit:
		// The current core catalog has stable ids but no executable guard handle.
		// Running a plugin before that guard would expose data, so remain closed.
		return ErrRouteGuardUnavailable
	case extensionmanifest.GuardCoreRaw:
		return ErrRouteGuardUnavailable
	default:
		// Custom/raw guard execution needs the separately trusted guard runtime.
		return ErrRouteGuardUnavailable
	}
}

type CatalogRouteSchemaValidator struct {
	Catalog RouteSchemaCatalog
}

func (v CatalogRouteSchemaValidator) ValidateRequest(ctx context.Context, step routes.RouteExecutionStep, request routes.DispatchRequest) error {
	return v.validate(ctx, step, "request", step.RequestSchema, request.Method, request.Headers, 0, request.Body)
}

func (v CatalogRouteSchemaValidator) ValidateResponse(ctx context.Context, step routes.RouteExecutionStep, request routes.DispatchRequest, response routes.DispatchResponse) error {
	return v.validate(ctx, step, "response", step.ResponseSchema, request.Method, response.Headers, response.Status, response.Body)
}

func (v CatalogRouteSchemaValidator) validate(
	ctx context.Context,
	step routes.RouteExecutionStep,
	direction string,
	reference string,
	actualMethod string,
	headers stdhttp.Header,
	responseStatus int,
	body []byte,
) error {
	if strings.TrimSpace(reference) == "" {
		return nil
	}
	if v.Catalog == nil || step.Provider.Kind != routes.ProviderPlugin {
		return ErrRouteSchemaUnavailable
	}
	mediaType, err := normalizedRouteSchemaMediaType(headers)
	if err != nil {
		return err
	}
	method := step.Method
	if step.Action == extensionmanifest.RouteActionGlobalMiddleware {
		method = "*"
	}
	return v.Catalog.ValidateRouteSchema(
		ctx, step.Provider.Artifact, direction, step.RouteID, method, actualMethod,
		step.ContractVersion, step.Action, reference, mediaType, responseStatus, append([]byte(nil), body...),
	)
}

func normalizedRouteSchemaMediaType(headers stdhttp.Header) (string, error) {
	values := headers.Values("Content-Type")
	if len(values) != 1 || strings.TrimSpace(values[0]) == "" {
		return "", fmt.Errorf("%w: exactly one Content-Type is required", ErrRouteSchemaUnavailable)
	}
	mediaType, _, err := mime.ParseMediaType(values[0])
	mediaType = strings.ToLower(mediaType)
	jsonMediaType := mediaType == "application/json" ||
		strings.HasPrefix(mediaType, "application/") && strings.HasSuffix(mediaType, "+json") && len(mediaType) > len("application/+json")
	if err != nil || strings.Contains(mediaType, "*") || !jsonMediaType {
		return "", fmt.Errorf("%w: invalid Content-Type", ErrRouteSchemaUnavailable)
	}
	return mediaType, nil
}

type BufferedRouteStepInvoker struct {
	Runtime       ExactRouteRuntime
	Client        *stdhttp.Client
	ResponseLimit int64
}

// routeTransportEvidence only records facts observed by the Host transport.
// Once request headers leave the Host, the plugin may have acted; fallback is
// no longer safe even when the connection dies before returning a response.
type routeTransportEvidence struct {
	requestStarted  atomic.Bool
	responseStarted atomic.Bool
	commit          *routes.RouteCommitObserver
}

func (e *routeTransportEvidence) markRequestStarted() {
	if e.requestStarted.CompareAndSwap(false, true) && e.commit != nil {
		e.commit.SideEffectStarted()
	}
}

func (e *routeTransportEvidence) markResponseStarted() {
	if e.responseStarted.CompareAndSwap(false, true) && e.commit != nil {
		e.commit.ResponseStarted()
	}
}

func (e *routeTransportEvidence) result() routes.RouteInvocationResult {
	return routes.RouteInvocationResult{
		SideEffectStarted: e.requestStarted.Load(), ResponseStarted: e.responseStarted.Load(),
	}
}

func NewBufferedRouteStepInvoker(runtime ExactRouteRuntime) *BufferedRouteStepInvoker {
	transport := stdhttp.DefaultTransport.(*stdhttp.Transport).Clone()
	transport.Proxy = nil
	return &BufferedRouteStepInvoker{
		Runtime: runtime, Client: &stdhttp.Client{Transport: transport}, ResponseLimit: defaultRouteResponseLimit,
	}
}

func (*BufferedRouteStepInvoker) SupportsMode(mode string) bool {
	return mode == extensionmanifest.RouteModeHTTP
}

func (i *BufferedRouteStepInvoker) Invoke(ctx context.Context, input routes.RouteInvocation) (routes.RouteInvocationResult, error) {
	if i == nil || i.Runtime == nil || i.Client == nil || ctx == nil || input.Step.Provider.Kind != routes.ProviderPlugin {
		return routes.RouteInvocationResult{}, routes.ErrDispatchTransport
	}
	artifact := input.Step.Provider.Artifact
	identity := extensionsruntime.RuntimeInstanceIdentity{
		ExtensionID: artifact.ExtensionID, InstanceID: artifact.RuntimeInstanceID,
	}
	snapshot, err := i.Runtime.InspectRuntimeInstance(identity)
	if err != nil {
		return routes.RouteInvocationResult{}, err
	}
	if !snapshot.Active || snapshot.Identity != identity || snapshot.ExtensionVersion != artifact.ExtensionVersion ||
		snapshot.ArtifactDigest != artifact.PackageDigest {
		return routes.RouteInvocationResult{}, ErrRouteRuntimeArtifact
	}
	target, err := exactLoopbackRouteURL(snapshot.Target.BaseURL, input.Request.Path, input.Request.Query)
	if err != nil {
		return routes.RouteInvocationResult{}, err
	}
	lease, err := i.Runtime.AcquireRuntimeCall(ctx, identity, extensionsruntime.RuntimeCallRoute)
	if err != nil {
		return routes.RouteInvocationResult{}, err
	}
	defer lease.Release()

	request, err := stdhttp.NewRequestWithContext(lease.Context, input.Request.Method, target, bytes.NewReader(input.Request.Body))
	if err != nil {
		return routes.RouteInvocationResult{}, err
	}
	copyRouteRequestHeaders(request.Header, input.Request.Headers)
	request.Header.Set("X-SForum-Extension-ID", artifact.ExtensionID)
	request.Header.Set("X-SForum-Route-ID", input.Step.RouteID)
	request.Header.Set("X-SForum-Route-Contract", input.Step.ContractVersion)
	request.Header.Set("X-SForum-Route-Action", input.Step.Action)
	request.Header.Set("X-SForum-Route-Phase", string(input.Step.Phase))
	request.Header.Set("X-SForum-Route-Handler", input.Step.Handler)
	request.Header.Set("X-SForum-Route-Plan-Revision", strconv.FormatUint(input.PlanRevision, 10))
	if input.Request.ActorID > 0 {
		request.Header.Set("X-SForum-Actor-ID", strconv.FormatInt(input.Request.ActorID, 10))
	}
	if input.Response != nil {
		request.Header.Set("X-SForum-Route-Response-Status", strconv.Itoa(input.Response.Status))
	}
	evidence := &routeTransportEvidence{commit: input.Commit}
	request = request.WithContext(httptrace.WithClientTrace(request.Context(), &httptrace.ClientTrace{
		WroteHeaders:         evidence.markRequestStarted,
		WroteRequest:         func(httptrace.WroteRequestInfo) { evidence.markRequestStarted() },
		GotFirstResponseByte: evidence.markResponseStarted,
	}))

	response, err := i.Client.Do(request)
	if err != nil {
		return evidence.result(), err
	}
	defer response.Body.Close()
	limit := i.ResponseLimit
	if limit <= 0 {
		limit = defaultRouteResponseLimit
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return evidence.result(), err
	}
	if int64(len(body)) > limit {
		return evidence.result(), ErrRouteResponseTooLarge
	}
	value := routes.DispatchResponse{Status: response.StatusCode, Headers: filteredRouteResponseHeaders(response.Header), Body: body}
	result := evidence.result()
	result.Response = &value
	return result, nil
}

func routeDispatcherMiddleware(dispatcher *routes.Dispatcher, actors RouteActorLoader) fiber.Handler {
	return func(c fiber.Ctx) error {
		if dispatcher == nil {
			return c.Next()
		}
		actor := identity.Actor{}
		if actors != nil {
			var err error
			actor, err = actors(c)
			if err != nil {
				return err
			}
		}
		request := routeDispatchRequest(c, actor)
		core := &fiberCoreRouteInvoker{ctx: c}
		result, err := dispatcher.Dispatch(c.Context(), request, core)
		if err != nil {
			return mapRouteDispatchError(err)
		}
		if !result.Handled {
			return c.Next()
		}
		writeRouteDispatchResponse(c, result.Response)
		return nil
	}
}

type fiberCoreRouteInvoker struct {
	ctx    fiber.Ctx
	called bool
}

func (i *fiberCoreRouteInvoker) InvokeCore(_ context.Context, _ routes.RouteExecutionStep, request routes.DispatchRequest) (routes.DispatchResponse, error) {
	if i == nil || i.ctx == nil || i.called {
		return routes.DispatchResponse{}, routes.ErrDispatchAlreadyCommitted
	}
	i.called = true
	applyRouteDispatchRequest(i.ctx, request)
	i.ctx.Response().Reset()
	if err := i.ctx.Next(); err != nil {
		i.ctx.Response().Reset()
		return routes.DispatchResponse{}, err
	}
	response := routes.DispatchResponse{
		Status: i.ctx.Response().StatusCode(), Headers: fasthttpResponseHeaders(i.ctx),
		Body: append([]byte(nil), i.ctx.Response().Body()...),
	}
	i.ctx.Response().Reset()
	return response, nil
}

func routeDispatchRequest(c fiber.Ctx, actor identity.Actor) routes.DispatchRequest {
	permissions := make(map[string]bool, len(actor.Permissions)+1)
	for key, allowed := range actor.Permissions {
		permissions[key] = allowed
	}
	if actor.IsSuperAdmin() {
		permissions["*"] = true
	}
	return routes.DispatchRequest{
		Method: c.Method(), Path: c.Path(), Query: string(c.Request().URI().QueryString()),
		Headers: fasthttpRequestHeaders(c), Body: append([]byte(nil), c.Body()...),
		ActorID: actor.ID, Authenticated: actor.ID > 0 && actor.IsActive(), Permissions: permissions,
	}
}

func applyRouteDispatchRequest(c fiber.Ctx, request routes.DispatchRequest) {
	c.Request().Header.Reset()
	for name, values := range request.Headers {
		for _, value := range values {
			c.Request().Header.Add(name, value)
		}
	}
	c.Request().SetBodyRaw(append([]byte(nil), request.Body...))
}

func writeRouteDispatchResponse(c fiber.Ctx, response routes.DispatchResponse) {
	c.Response().Reset()
	c.Status(response.Status)
	for name, values := range response.Headers {
		for _, value := range values {
			c.Response().Header.Add(name, value)
		}
	}
	c.Response().SetBodyRaw(append([]byte(nil), response.Body...))
}

func fasthttpRequestHeaders(c fiber.Ctx) stdhttp.Header {
	headers := make(stdhttp.Header)
	for key, value := range c.Request().Header.All() {
		headers.Add(string(key), string(value))
	}
	return headers
}

func fasthttpResponseHeaders(c fiber.Ctx) stdhttp.Header {
	headers := make(stdhttp.Header)
	for key, value := range c.Response().Header.All() {
		headers.Add(string(key), string(value))
	}
	return headers
}

func exactLoopbackRouteURL(base, path, query string) (string, error) {
	target, err := url.Parse(strings.TrimSpace(base))
	if err != nil || target.Scheme != "http" && target.Scheme != "https" || target.User != nil || target.Host == "" {
		return "", ErrRouteRuntimeTarget
	}
	host := target.Hostname()
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) || target.Path != "" && target.Path != "/" || target.RawQuery != "" || target.Fragment != "" {
		return "", ErrRouteRuntimeTarget
	}
	relative, err := url.ParseRequestURI(path)
	if err != nil || !strings.HasPrefix(relative.Path, "/") {
		return "", ErrRouteRuntimeTarget
	}
	target.Path = relative.Path
	target.RawPath = relative.RawPath
	target.RawQuery = query
	return target.String(), nil
}

func copyRouteRequestHeaders(target, source stdhttp.Header) {
	for name, values := range source {
		if !routeRequestHeaderAllowed(name) {
			continue
		}
		for _, value := range values {
			target.Add(name, value)
		}
	}
}

func routeRequestHeaderAllowed(name string) bool {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(name)), "x-sforum-") {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "host", "content-length", "cookie", "authorization", "proxy-authorization",
		"connection", "keep-alive", "proxy-authenticate", "te", "trailer", "transfer-encoding", "upgrade":
		return false
	default:
		return true
	}
}

func filteredRouteResponseHeaders(source stdhttp.Header) stdhttp.Header {
	result := make(stdhttp.Header)
	for name, values := range source {
		canonical := strings.ToLower(strings.TrimSpace(name))
		if strings.HasPrefix(canonical, "x-sforum-") {
			continue
		}
		switch canonical {
		case "", "content-length", "connection", "keep-alive", "proxy-authenticate", "proxy-authorization",
			"set-cookie", "te", "trailer", "transfer-encoding", "upgrade":
			continue
		}
		for _, value := range values {
			result.Add(name, value)
		}
	}
	return result
}

func mapRouteDispatchError(err error) error {
	switch {
	case errors.Is(err, ErrRouteLoginRequired):
		return fiber.NewError(fiber.StatusUnauthorized, "auth.required")
	case errors.Is(err, routes.ErrDispatchDenied):
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	case errors.Is(err, routes.ErrAmbiguousRoute), errors.Is(err, routes.ErrProviderSelectionStale):
		return fiber.NewError(fiber.StatusServiceUnavailable, "extensions.route_provider_unavailable")
	case errors.Is(err, routes.ErrDispatchSchema):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "extensions.route_schema_invalid")
	case errors.Is(err, routes.ErrDispatchTransport):
		return fiber.NewError(fiber.StatusBadGateway, "extensions.route_unavailable")
	default:
		return fmt.Errorf("route dispatch: %w", err)
	}
}
