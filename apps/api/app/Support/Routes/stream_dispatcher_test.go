package routes

import (
	"context"
	"errors"
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
	_, err := NewDispatcher(DispatcherConfig{Plans: streamRegistryResolver{registry}}).PrepareStream(
		context.Background(), DispatchRequest{Method: "GET", Path: "/denied"},
	)
	if !errors.Is(err, ErrDispatchDenied) {
		t.Fatalf("guard error=%v", err)
	}
}

type allowStreamGuard struct{}

func (allowStreamGuard) Authorize(context.Context, RouteExecutionPlan, RouteExecutionStep, DispatchRequest) error {
	return nil
}

type streamRegistryResolver struct{ registry *Registry }

func (r streamRegistryResolver) BuildExecutionPlan(_ context.Context, method, path string) (RouteExecutionPlan, error) {
	return r.registry.BuildExecutionPlan(method, path)
}
