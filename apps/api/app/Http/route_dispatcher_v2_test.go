package http

import (
	"context"
	"errors"
	stdhttp "net/http"
	"reflect"
	"testing"

	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

func TestBufferedRouteStepInvokerSelectsV2WithOneAdmission(t *testing.T) {
	runtime := newRouteDispatcherV2Runtime(t)
	invoker := NewBufferedRouteStepInvoker(runtime)
	commit := routes.NewRouteCommitObserver()
	result, err := invoker.Invoke(context.Background(), routeDispatcherV2Invocation(commit))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.calls != 1 || runtime.activeCalls != 1 {
		t.Fatalf("calls=%d activeCalls=%d", runtime.calls, runtime.activeCalls)
	}
	if runtime.request.RouteID != "demo.route" || runtime.request.ContractVersion != "demo.route@1" ||
		runtime.request.PathParameters["id"] != "41" || runtime.request.QueryParameters["page"] != "2" ||
		runtime.request.Body["title"] != "hello" || runtime.request.Actor.UserID != 42 ||
		!reflect.DeepEqual(runtime.request.Actor.PermissionKeys, []string{"*", "topics.write"}) {
		t.Fatalf("request = %#v", runtime.request)
	}
	if result.Response == nil || result.Response.Status != stdhttp.StatusCreated || string(result.Response.Body) != `{"ok":true}` ||
		result.Response.Headers.Get("X-Result") != "ok" || result.Response.Headers.Get("X-SForum-Actor-ID") != "" ||
		result.Response.Headers.Get("Set-Cookie") != "" || !result.SideEffectStarted || !result.ResponseStarted {
		t.Fatalf("result = %#v", result)
	}
	if snapshot := runtime.gate.Snapshot(); snapshot.ActiveTotal != 0 {
		t.Fatalf("admission leaked: %#v", snapshot)
	}
}

func TestBufferedRouteStepInvokerV2FencesFallbackBeforeRemoteError(t *testing.T) {
	runtime := newRouteDispatcherV2Runtime(t)
	runtime.err = errors.New("plugin crashed after receiving request")
	commit := routes.NewRouteCommitObserver()
	result, err := NewBufferedRouteStepInvoker(runtime).Invoke(context.Background(), routeDispatcherV2Invocation(commit))
	if err == nil || !result.SideEffectStarted || result.ResponseStarted || commit.State() != routes.RouteCommitSideEffectStarted {
		t.Fatalf("result=%#v commit=%s err=%v", result, commit.State(), err)
	}
}

func TestBufferedRouteStepInvokerV2RejectsAmbiguousOrUntypedInput(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*routes.RouteInvocation)
	}{
		{"repeated query", func(input *routes.RouteInvocation) { input.Request.Query = "page=1&page=2" }},
		{"malformed JSON", func(input *routes.RouteInvocation) { input.Request.Body = []byte(`{"title":`) }},
		{"JSON array", func(input *routes.RouteInvocation) { input.Request.Body = []byte(`[1,2]`) }},
		{"JSON scalar", func(input *routes.RouteInvocation) { input.Request.Body = []byte(`true`) }},
		{"body without schema", func(input *routes.RouteInvocation) { input.Step.RequestSchema = "" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runtime := newRouteDispatcherV2Runtime(t)
			input := routeDispatcherV2Invocation(routes.NewRouteCommitObserver())
			test.mutate(&input)
			if _, err := NewBufferedRouteStepInvoker(runtime).Invoke(context.Background(), input); !errors.Is(err, ErrRouteRuntimeTarget) {
				t.Fatalf("error = %v", err)
			}
			if runtime.calls != 0 {
				t.Fatalf("remote called %d times", runtime.calls)
			}
		})
	}
}

func TestBufferedRouteStepInvokerV2EnforcesResponseLimit(t *testing.T) {
	runtime := newRouteDispatcherV2Runtime(t)
	runtime.response.Body = map[string]any{"value": "too large"}
	invoker := NewBufferedRouteStepInvoker(runtime)
	invoker.ResponseLimit = 4
	if _, err := invoker.Invoke(context.Background(), routeDispatcherV2Invocation(routes.NewRouteCommitObserver())); !errors.Is(err, ErrRouteResponseTooLarge) {
		t.Fatalf("error = %v", err)
	}
}

func TestBufferedRouteStepInvokerV2RejectsStreamPreflight(t *testing.T) {
	runtime := newRouteDispatcherV2Runtime(t)
	runtime.response.Body = nil
	runtime.response.BodyPresent = false
	runtime.response.StreamFollows = true
	result, err := NewBufferedRouteStepInvoker(runtime).Invoke(
		context.Background(), routeDispatcherV2Invocation(routes.NewRouteCommitObserver()),
	)
	if !errors.Is(err, ErrRouteRuntimeTarget) || !result.SideEffectStarted || result.ResponseStarted {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

type routeDispatcherV2Runtime struct {
	snapshot    extensionsruntime.RuntimeInstanceSnapshot
	gate        *extensionsruntime.RuntimeAdmissionGate
	request     extensionsruntime.ProtocolV2RouteRequest
	response    extensionsruntime.ProtocolV2RouteResponse
	err         error
	calls       int
	activeCalls int
}

func newRouteDispatcherV2Runtime(t *testing.T) *routeDispatcherV2Runtime {
	t.Helper()
	identity := extensionsruntime.RuntimeInstanceIdentity{ExtensionID: "demo.plugin", InstanceID: "runtime-1"}
	gate, err := extensionsruntime.NewRuntimeAdmissionGate(identity)
	if err != nil {
		t.Fatal(err)
	}
	return &routeDispatcherV2Runtime{
		gate: gate,
		snapshot: extensionsruntime.RuntimeInstanceSnapshot{
			Identity: identity, ExtensionVersion: "1.0.0", ArtifactDigest: "digest-v1",
			Target: extensionsruntime.RouteTarget{InstanceID: identity.InstanceID}, Active: true,
		},
		response: extensionsruntime.ProtocolV2RouteResponse{
			StatusCode: stdhttp.StatusCreated,
			Headers: stdhttp.Header{
				"X-Result": {"ok"}, "X-SForum-Actor-ID": {"forged"}, "Set-Cookie": {"forged=1"},
			},
			Body: map[string]any{"ok": true}, BodyPresent: true,
		},
	}
}

func (r *routeDispatcherV2Runtime) InspectRuntimeInstance(identity extensionsruntime.RuntimeInstanceIdentity) (extensionsruntime.RuntimeInstanceSnapshot, error) {
	if identity != r.snapshot.Identity {
		return extensionsruntime.RuntimeInstanceSnapshot{}, extensionsruntime.ErrRuntimeInstanceNotFound
	}
	return r.snapshot, nil
}

func (r *routeDispatcherV2Runtime) AcquireRuntimeCall(ctx context.Context, identity extensionsruntime.RuntimeInstanceIdentity, class extensionsruntime.RuntimeCallClass) (*extensionsruntime.RuntimeAdmissionLease, error) {
	if identity != r.snapshot.Identity || class != extensionsruntime.RuntimeCallRoute {
		return nil, extensionsruntime.ErrRuntimeInstanceNotActive
	}
	return r.gate.Acquire(ctx, class)
}

func (r *routeDispatcherV2Runtime) InvokeRouteInstance(
	_ context.Context,
	identity extensionsruntime.RuntimeInstanceIdentity,
	request extensionsruntime.ProtocolV2RouteRequest,
) (extensionsruntime.ProtocolV2RouteResponse, error) {
	if identity != r.snapshot.Identity {
		return extensionsruntime.ProtocolV2RouteResponse{}, extensionsruntime.ErrRuntimeInstanceNotFound
	}
	r.calls++
	r.activeCalls = r.gate.Snapshot().ActiveTotal
	r.request = request
	return r.response, r.err
}

func routeDispatcherV2Invocation(commit *routes.RouteCommitObserver) routes.RouteInvocation {
	return routes.RouteInvocation{
		PlanRevision: 7, StepIndex: 2, Commit: commit,
		Step: routes.RouteExecutionStep{
			RouteID: "demo.route", ContractVersion: "demo.route@1",
			RequestSchema: "demo.request@1", ResponseSchema: "demo.response@1",
			Provider: routes.Provider{Kind: routes.ProviderPlugin, Artifact: routes.PluginArtifact{
				ExtensionID: "demo.plugin", ExtensionVersion: "1.0.0", PackageDigest: "digest-v1", RuntimeInstanceID: "runtime-1",
			}},
		},
		Request: routes.DispatchRequest{
			Method: stdhttp.MethodPost, Path: "/demo/41", Query: "page=2",
			Headers: stdhttp.Header{"Content-Type": {"application/json"}, "X-Test": {"value"}, "X-SForum-Forged": {"bad"}},
			Body:    []byte(`{"title":"hello"}`), Params: map[string]string{"id": "41"},
			ActorID: 42, Authenticated: true, Permissions: map[string]bool{"topics.write": true, "*": true, "ignored": false},
		},
	}
}

var _ ExactRouteRuntime = (*routeDispatcherV2Runtime)(nil)
var _ exactRouteV2Runtime = (*routeDispatcherV2Runtime)(nil)
