package routes

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

func TestStreamDispatcherPreparesExactTerminalAndPublishesCommitTrace(t *testing.T) {
	registry := NewRegistry()
	artifact := routeArtifact("stream.dispatch", "1.0.0", 'a')
	stream := pluginRoute("stream.dispatch.events", "/events", 0, "GET")
	stream.Mode = extensionmanifest.RouteModeSSE
	if _, err := registry.Publish(Publication{Plugins: []PluginRouteSet{{Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{stream}}}}); err != nil {
		t.Fatal(err)
	}
	traces := NewRouteTraceRing(8)
	dispatcher := NewDispatcher(DispatcherConfig{
		Plans: streamRegistryResolver{registry}, Guard: allowStreamGuard{}, Trace: traces,
	})
	prepared, err := dispatcher.PrepareStream(context.Background(), DispatchRequest{
		Method: "get", Path: "/events", Headers: map[string][]string{"X-Test": {"one"}},
	})
	if err != nil || !prepared.Handled || prepared.Dispatch == nil {
		t.Fatalf("prepared=%#v err=%v", prepared, err)
	}
	if step := prepared.Dispatch.Step(); step.RouteID != stream.ID || step.Provider.Artifact != artifact {
		t.Fatalf("step=%#v", step)
	}
	request := prepared.Dispatch.Request()
	request.Headers.Set("X-Test", "mutated")
	if prepared.Dispatch.Request().Headers.Get("X-Test") != "one" {
		t.Fatal("stream request was not detached")
	}
	prepared.Dispatch.RequestStarted()
	prepared.Dispatch.ResponseStarted()
	if err := prepared.Dispatch.Complete(); err != nil {
		t.Fatal(err)
	}
	if err := prepared.Dispatch.Complete(); !errors.Is(err, ErrDispatchAlreadyCommitted) {
		t.Fatalf("second complete error=%v", err)
	}
	records := traces.RouteTraces(8)
	if len(records) != 2 || records[0].Outcome != RouteTraceSucceeded || records[1].Outcome != RouteTraceCommitted ||
		records[1].CommitState != RouteCommitFinal {
		t.Fatalf("traces=%#v", records)
	}
}

func TestRouteStreamDispatchStepIsDetached(t *testing.T) {
	dispatch := &RouteStreamDispatch{step: RouteExecutionStep{
		MutableRequestFields:  []string{"/query"},
		MutableResponseFields: []string{"/status"},
	}}
	step := dispatch.Step()
	step.MutableRequestFields[0] = "/caller-request"
	step.MutableResponseFields[0] = "/caller-response"

	again := dispatch.Step()
	if again.MutableRequestFields[0] != "/query" || again.MutableResponseFields[0] != "/status" {
		t.Fatalf("stream step leaked mutable field slices: %#v", again)
	}
}

func TestStreamDispatcherBypassesCoreAndBufferedPlansAndFailsClosedOnComposition(t *testing.T) {
	registry := NewRegistry()
	core := coreRoute("core.route.stream.bypass", "GET", "/core")
	artifact := routeArtifact("stream.composed", "1.0.0", 'c')
	buffered := pluginRoute("stream.composed.buffered", "/buffered", 0, "GET")
	composed := pluginRoute("stream.composed.handler", "/composed", 0, "GET")
	composed.Mode = extensionmanifest.RouteModeSSE
	global := extensionmanifest.ManifestRoute{
		ID: "stream.composed.global", ContractVersion: "stream.composed.global@1",
		Action: extensionmanifest.RouteActionGlobalMiddleware, Guard: extensionmanifest.GuardCorePublic,
		Fallback: "closed", Mode: extensionmanifest.RouteModeHTTP, Handler: "route.global",
		RequestSchema: "stream.composed.global.request@1", ResponseSchema: "stream.composed.global.response@1",
	}
	if _, err := registry.Publish(Publication{
		Core: []CoreRoute{core}, Plugins: []PluginRouteSet{{
			Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{buffered, composed, global},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDispatcher(DispatcherConfig{Plans: streamRegistryResolver{registry}, Guard: allowStreamGuard{}})
	for _, path := range []string{"/core", "/buffered", "/missing"} {
		prepared, err := dispatcher.PrepareStream(context.Background(), DispatchRequest{Method: "GET", Path: path})
		if err != nil || prepared.Handled {
			t.Fatalf("path=%s prepared=%#v err=%v", path, prepared, err)
		}
	}
	if _, err := dispatcher.PrepareStream(context.Background(), DispatchRequest{Method: "GET", Path: "/composed"}); !errors.Is(err, ErrDispatchTransport) {
		t.Fatalf("composed stream error=%v", err)
	}
}

func TestStreamDispatcherFailsClosedWithoutGuard(t *testing.T) {
	registry := NewRegistry()
	artifact := routeArtifact("stream.denied", "1.0.0", 'd')
	stream := pluginRoute("stream.denied.events", "/denied", 0, "GET")
	stream.Mode = extensionmanifest.RouteModeStream
	if _, err := registry.Publish(Publication{Plugins: []PluginRouteSet{{Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{stream}}}}); err != nil {
		t.Fatal(err)
	}
	prepared, err := NewDispatcher(DispatcherConfig{Plans: streamRegistryResolver{registry}}).PrepareStream(
		context.Background(), DispatchRequest{Method: "GET", Path: "/denied"},
	)
	if err != nil || prepared.Dispatch == nil {
		t.Fatalf("prepared=%#v error=%v", prepared, err)
	}
	if _, err := prepared.Dispatch.Open(context.Background()); !errors.Is(err, ErrDispatchDenied) {
		t.Fatalf("open guard error=%v", err)
	}
}

func TestStreamDispatcherRejectsPublishedRequiredIdempotencyPolicy(t *testing.T) {
	registry := NewRegistry()
	artifact := routeArtifact("stream.idempotency", "1.0.0", '1')
	stream := pluginRoute("stream.idempotency.events", "/idempotent-stream", 0, "POST")
	stream.Mode = extensionmanifest.RouteModeMultipart
	if _, err := registry.Publish(Publication{Plugins: []PluginRouteSet{{Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{stream}}}}); err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDispatcher(DispatcherConfig{
		Plans: streamRegistryResolver{registry}, Guard: allowStreamGuard{},
		Policies: dispatchPolicyResolver{policy: RouteExecutionPolicy{
			Idempotency: "required.24h@1", IdempotencyRequired: true,
		}},
	})
	prepared, err := dispatcher.PrepareStream(
		context.Background(), DispatchRequest{Method: "POST", Path: "/idempotent-stream"},
	)
	if prepared.Handled || !errors.Is(err, ErrDispatchIdempotencyUnavailable) {
		t.Fatalf("prepared=%#v error=%v", prepared, err)
	}
}

func TestStreamDispatcherPreservesAuthorizedRawRequestStamp(t *testing.T) {
	registry := NewRegistry()
	artifact := routeArtifact("stream.raw", "1.0.0", 'a')
	stream := pluginRoute("stream.raw.events", "/raw-events", 0, "GET")
	stream.Mode = extensionmanifest.RouteModeSSE
	stream.Guard = extensionmanifest.GuardCoreRaw
	if _, err := registry.Publish(Publication{Plugins: []PluginRouteSet{{Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{stream}}}}); err != nil {
		t.Fatal(err)
	}
	invoker := &authorityStreamInvoker{}
	dispatcher := NewDispatcher(DispatcherConfig{
		Plans: streamRegistryResolver{registry}, Guard: allowStreamGuard{}, Steps: invoker,
	})
	prepared, err := dispatcher.PrepareStream(
		context.Background(), DispatchRequest{Method: "GET", Path: "/raw-events"},
	)
	if err != nil || prepared.Dispatch == nil {
		t.Fatalf("prepared=%#v error=%v", prepared, err)
	}
	start, err := prepared.Dispatch.Open(context.Background())
	if err != nil || start.Session == nil || !invoker.raw {
		t.Fatalf("start=%#v raw=%v error=%v", start, invoker.raw, err)
	}
	if _, err := prepared.Dispatch.Open(context.Background()); !errors.Is(err, ErrDispatchAlreadyCommitted) || invoker.calls != 1 {
		t.Fatalf("second open error=%v calls=%d", err, invoker.calls)
	}
	start.Session.Cancel()
}

type authorityStreamInvoker struct {
	raw   bool
	calls int
}

func (*authorityStreamInvoker) SupportsMode(string) bool { return false }

func (*authorityStreamInvoker) Invoke(context.Context, RouteInvocation) (RouteInvocationResult, error) {
	return RouteInvocationResult{}, ErrDispatchTransport
}

func (i *authorityStreamInvoker) OpenStream(_ context.Context, input RouteInvocation) (RouteStreamStart, error) {
	i.calls++
	i.raw = input.RawRequestAuthorized()
	return RouteStreamStart{
		Response: DispatchResponse{Status: http.StatusOK}, Session: authorityStreamSession{},
	}, nil
}

type authorityStreamSession struct{}

func (authorityStreamSession) Send([]byte, bool) error            { return nil }
func (authorityStreamSession) CloseRequest() error                { return nil }
func (authorityStreamSession) Recv() (RouteStreamChunk, error)    { return RouteStreamChunk{}, io.EOF }
func (authorityStreamSession) Response() (DispatchResponse, bool) { return DispatchResponse{}, false }
func (authorityStreamSession) Cancel()                            {}

type allowStreamGuard struct{}

func (allowStreamGuard) Authorize(context.Context, RouteExecutionPlan, RouteExecutionStep, DispatchRequest) error {
	return nil
}

func (allowStreamGuard) AuthorizeRoute(
	_ context.Context,
	plan RouteExecutionPlan,
	stepIndex int,
	step RouteExecutionStep,
	request DispatchRequest,
) (RouteGuardAuthorization, error) {
	authorization, ok := authorizedRouteGuardAuthorization(plan, stepIndex, step, request)
	if !ok {
		return RouteGuardAuthorization{}, ErrCoreGuardEvaluatorUnavailable
	}
	return authorization, nil
}

type streamRegistryResolver struct{ registry *Registry }

func (r streamRegistryResolver) BuildExecutionPlan(_ context.Context, method, path string) (RouteExecutionPlan, error) {
	return r.registry.BuildExecutionPlan(method, path)
}
