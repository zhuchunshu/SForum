package routes

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

func TestDispatcherExecutesBufferedChainInPlanOrder(t *testing.T) {
	steps := []RouteExecutionStep{
		dispatchPluginStep(RoutePhaseGlobal, "demo.route.global", extensionmanifest.RouteActionGlobalMiddleware),
		dispatchPluginStep(RoutePhaseBefore, "demo.route.before", extensionmanifest.RouteActionBefore),
		dispatchPluginStep(RoutePhaseFilter, "demo.route.filter", extensionmanifest.RouteActionFilter),
		dispatchPluginStep(RoutePhaseWrap, "demo.route.wrap", extensionmanifest.RouteActionWrap),
		dispatchCoreStep("core.route.topic.show"),
		dispatchPluginStep(RoutePhaseAfter, "demo.route.after", extensionmanifest.RouteActionAfter),
	}
	order := make([]string, 0, len(steps))
	invoker := &dispatchStepInvoker{invoke: func(_ context.Context, input RouteInvocation) (RouteInvocationResult, error) {
		order = append(order, string(input.Step.Phase)+":"+input.Step.RouteID)
		if input.Step.Phase == RoutePhaseAfter {
			if input.Response == nil || string(input.Response.Body) != `{"source":"core"}` {
				t.Fatalf("after response = %#v", input.Response)
			}
			value := cloneDispatchResponse(*input.Response)
			value.Headers.Set("X-After", "yes")
			return RouteInvocationResult{Response: &value}, nil
		}
		return RouteInvocationResult{}, nil
	}}
	core := &dispatchCoreInvoker{invoke: func(_ context.Context, step RouteExecutionStep, request DispatchRequest) (DispatchResponse, error) {
		order = append(order, string(step.Phase)+":"+step.RouteID)
		if request.Params["id"] != "42" || request.Headers.Get("X-Test") != "present" {
			t.Fatalf("core request = %#v", request)
		}
		return DispatchResponse{Status: http.StatusOK, Headers: http.Header{"Content-Type": []string{"application/json"}}, Body: []byte(`{"source":"core"}`)}, nil
	}}
	guard := &dispatchGuard{}
	schemas := &dispatchSchemas{}
	dispatcher := NewDispatcher(DispatcherConfig{
		Plans: dispatchPlanResolver{plan: dispatchPlan("GET", "/topics/42", map[string]string{"id": "42"}, steps, 4)},
		Steps: invoker, Guard: guard, Schemas: schemas,
	})

	result, err := dispatcher.Dispatch(context.Background(), DispatchRequest{
		Method: "GET", Path: "/topics/42", Headers: http.Header{"X-Test": []string{"present"}},
	}, core)
	if err != nil {
		t.Fatal(err)
	}
	wantOrder := []string{
		"global:demo.route.global", "before:demo.route.before", "filter:demo.route.filter",
		"wrap:demo.route.wrap", "handler:core.route.topic.show", "after:demo.route.after",
	}
	if !result.Handled || result.Response.Status != http.StatusOK || result.Response.Headers.Get("X-After") != "yes" || !reflect.DeepEqual(order, wantOrder) {
		t.Fatalf("result=%#v order=%#v", result, order)
	}
	if guard.calls != 5 || schemas.requestCalls != 5 || schemas.responseCalls != 1 {
		t.Fatalf("guard=%d request schemas=%d response schemas=%d", guard.calls, schemas.requestCalls, schemas.responseCalls)
	}
}

func TestDispatcherLeavesCoreOnlyPlanEntirelyUnhandled(t *testing.T) {
	core := &dispatchCoreInvoker{invoke: func(context.Context, RouteExecutionStep, DispatchRequest) (DispatchResponse, error) {
		t.Fatal("core-only plan must not enter buffered CoreInvoker")
		return DispatchResponse{}, nil
	}}
	dispatcher := NewDispatcher(DispatcherConfig{
		Plans: dispatchPlanResolver{plan: dispatchPlan(
			"GET", "/download", nil, []RouteExecutionStep{dispatchCoreStep("core.route.download")}, 0,
		)},
	})
	result, err := dispatcher.Dispatch(context.Background(), DispatchRequest{Method: "GET", Path: "/download"}, core)
	if err != nil || result.Handled || core.calls != 0 {
		t.Fatalf("result=%#v core calls=%d err=%v", result, core.calls, err)
	}
}

func TestDispatcherSafeFallbackRequiresPristineGETOrHEAD(t *testing.T) {
	for _, method := range []string{"GET", "HEAD"} {
		t.Run(method, func(t *testing.T) {
			step := dispatchPluginStep(RoutePhaseHandler, "demo.route.replace", extensionmanifest.RouteActionReplace)
			step.Fallback = "readonly_core"
			core := &dispatchCoreInvoker{invoke: func(context.Context, RouteExecutionStep, DispatchRequest) (DispatchResponse, error) {
				return DispatchResponse{Status: http.StatusOK, Body: []byte("core")}, nil
			}}
			dispatcher := dispatchTestDispatcher(dispatchPlan(method, "/resource", nil, []RouteExecutionStep{step}, 0), &dispatchStepInvoker{
				invoke: func(context.Context, RouteInvocation) (RouteInvocationResult, error) {
					return RouteInvocationResult{}, errors.New("plugin unavailable")
				},
			})
			result, err := dispatcher.Dispatch(context.Background(), DispatchRequest{Method: method, Path: "/resource"}, core)
			if err != nil || !result.Handled || string(result.Response.Body) != "core" || core.calls != 1 {
				t.Fatalf("result=%#v core=%d err=%v", result, core.calls, err)
			}
		})
	}
}

func TestDispatcherBufferedErrorResponseRemainsSafeToDiscard(t *testing.T) {
	step := dispatchPluginStep(RoutePhaseHandler, "demo.route.replace", extensionmanifest.RouteActionReplace)
	step.Fallback = "readonly_core"
	core := &dispatchCoreInvoker{invoke: func(context.Context, RouteExecutionStep, DispatchRequest) (DispatchResponse, error) {
		return DispatchResponse{Status: http.StatusOK, Body: []byte("core")}, nil
	}}
	dispatcher := dispatchTestDispatcher(dispatchPlan("GET", "/resource", nil, []RouteExecutionStep{step}, 0), &dispatchStepInvoker{
		invoke: func(context.Context, RouteInvocation) (RouteInvocationResult, error) {
			buffered := DispatchResponse{Status: http.StatusBadGateway, Body: []byte("discarded")}
			return RouteInvocationResult{Response: &buffered}, errors.New("plugin unavailable")
		},
	})
	result, err := dispatcher.Dispatch(context.Background(), DispatchRequest{Method: "GET", Path: "/resource"}, core)
	if err != nil || string(result.Response.Body) != "core" || core.calls != 1 {
		t.Fatalf("result=%#v core=%d err=%v", result, core.calls, err)
	}
}

func TestDispatcherNeverRunsSecondWriterAfterSideEffectOrOnUnsafeMethod(t *testing.T) {
	tests := []struct {
		name            string
		method          string
		sideEffect      bool
		responseStarted bool
	}{
		{name: "get after side effect", method: "GET", sideEffect: true},
		{name: "get after response started", method: "GET", responseStarted: true},
		{name: "unsafe pristine", method: "POST", sideEffect: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			step := dispatchPluginStep(RoutePhaseHandler, "demo.route.replace", extensionmanifest.RouteActionReplace)
			step.Fallback = "readonly_core"
			core := &dispatchCoreInvoker{}
			dispatcher := dispatchTestDispatcher(dispatchPlan(test.method, "/resource", nil, []RouteExecutionStep{step}, 0), &dispatchStepInvoker{
				invoke: func(_ context.Context, input RouteInvocation) (RouteInvocationResult, error) {
					if test.sideEffect {
						input.Commit.SideEffectStarted()
					}
					if test.responseStarted {
						input.Commit.ResponseStarted()
					}
					return RouteInvocationResult{
						SideEffectStarted: test.sideEffect, ResponseStarted: test.responseStarted,
					}, errors.New("writer failed")
				},
			})
			result, err := dispatcher.Dispatch(context.Background(), DispatchRequest{Method: test.method, Path: "/resource"}, core)
			if !errors.Is(err, ErrDispatchTransport) || result.Handled || core.calls != 0 {
				t.Fatalf("result=%#v core=%d err=%v", result, core.calls, err)
			}
		})
	}
}

func TestDispatcherFailsClosedForGuardAndSchema(t *testing.T) {
	step := dispatchPluginStep(RoutePhaseHandler, "demo.route.add", extensionmanifest.RouteActionAdd)
	plan := dispatchPlan("POST", "/custom", nil, []RouteExecutionStep{step}, 0)
	t.Run("guard", func(t *testing.T) {
		dispatcher := NewDispatcher(DispatcherConfig{
			Plans: dispatchPlanResolver{plan: plan}, Steps: &dispatchStepInvoker{},
			Guard: &dispatchGuard{err: errors.New("denied")}, Schemas: &dispatchSchemas{},
		})
		_, err := dispatcher.Dispatch(context.Background(), DispatchRequest{Method: "POST", Path: "/custom"}, nil)
		if !errors.Is(err, ErrDispatchDenied) {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("missing schema validator", func(t *testing.T) {
		dispatcher := NewDispatcher(DispatcherConfig{
			Plans: dispatchPlanResolver{plan: plan}, Steps: &dispatchStepInvoker{}, Guard: &dispatchGuard{},
		})
		_, err := dispatcher.Dispatch(context.Background(), DispatchRequest{Method: "POST", Path: "/custom"}, nil)
		if !errors.Is(err, ErrDispatchSchema) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestDispatcherPropagatesTimeoutAndCallerCancellation(t *testing.T) {
	for _, test := range []struct {
		name    string
		parent  func() (context.Context, context.CancelFunc)
		wantErr error
	}{
		{
			name: "step timeout",
			parent: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			wantErr: context.DeadlineExceeded,
		},
		{
			name: "caller cancellation",
			parent: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
			wantErr: context.Canceled,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			step := dispatchPluginStep(RoutePhaseHandler, "demo.route.slow", extensionmanifest.RouteActionAdd)
			step.TimeoutMS = 5
			dispatcher := dispatchTestDispatcher(dispatchPlan("GET", "/slow", nil, []RouteExecutionStep{step}, 0), &dispatchStepInvoker{
				invoke: func(ctx context.Context, _ RouteInvocation) (RouteInvocationResult, error) {
					<-ctx.Done()
					return RouteInvocationResult{}, ctx.Err()
				},
			})
			ctx, cancel := test.parent()
			defer cancel()
			_, err := dispatcher.Dispatch(ctx, DispatchRequest{Method: "GET", Path: "/slow"}, nil)
			if !errors.Is(err, ErrDispatchTransport) || !errors.Is(err, test.wantErr) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestRouteCommitObserverIsSingleFinalizerAndNeverReturnsToPristine(t *testing.T) {
	observer := NewRouteCommitObserver()
	var wait sync.WaitGroup
	wait.Add(2)
	go func() { defer wait.Done(); observer.ResponseStarted() }()
	go func() { defer wait.Done(); observer.SideEffectStarted() }()
	wait.Wait()
	if observer.State() != RouteCommitResponseStarted {
		t.Fatalf("state=%q", observer.State())
	}
	if !observer.Finalize() || observer.Finalize() || observer.State() != RouteCommitFinal {
		t.Fatalf("final state=%q", observer.State())
	}
}

type dispatchPlanResolver struct {
	plan RouteExecutionPlan
	err  error
}

func (r dispatchPlanResolver) BuildExecutionPlan(context.Context, string, string) (RouteExecutionPlan, error) {
	return r.plan, r.err
}

type dispatchStepInvoker struct {
	invoke func(context.Context, RouteInvocation) (RouteInvocationResult, error)
}

func (*dispatchStepInvoker) SupportsMode(mode string) bool {
	return mode == extensionmanifest.RouteModeHTTP
}

func (i *dispatchStepInvoker) Invoke(ctx context.Context, input RouteInvocation) (RouteInvocationResult, error) {
	if i.invoke == nil {
		return RouteInvocationResult{}, nil
	}
	return i.invoke(ctx, input)
}

type dispatchGuard struct {
	calls int
	err   error
}

func (g *dispatchGuard) Authorize(context.Context, RouteExecutionPlan, RouteExecutionStep, DispatchRequest) error {
	g.calls++
	return g.err
}

type dispatchSchemas struct {
	requestCalls  int
	responseCalls int
	err           error
}

func (s *dispatchSchemas) ValidateRequest(context.Context, RouteExecutionStep, DispatchRequest) error {
	s.requestCalls++
	return s.err
}

func (s *dispatchSchemas) ValidateResponse(context.Context, RouteExecutionStep, DispatchResponse) error {
	s.responseCalls++
	return s.err
}

type dispatchCoreInvoker struct {
	calls  int
	invoke func(context.Context, RouteExecutionStep, DispatchRequest) (DispatchResponse, error)
}

func (i *dispatchCoreInvoker) InvokeCore(ctx context.Context, step RouteExecutionStep, request DispatchRequest) (DispatchResponse, error) {
	i.calls++
	if i.invoke == nil {
		return DispatchResponse{}, errors.New("unexpected core invocation")
	}
	return i.invoke(ctx, step, request)
}

func dispatchTestDispatcher(plan RouteExecutionPlan, invoker StepInvoker) *Dispatcher {
	return NewDispatcher(DispatcherConfig{
		Plans: dispatchPlanResolver{plan: plan}, Steps: invoker,
		Guard: &dispatchGuard{}, Schemas: &dispatchSchemas{}, DefaultTimeout: 50 * time.Millisecond,
	})
}

func dispatchPlan(method, path string, params map[string]string, steps []RouteExecutionStep, terminal int) RouteExecutionPlan {
	return RouteExecutionPlan{
		revision: 1, method: method, path: path, params: cloneRouteExecutionParams(params),
		unsafeMethod: method != "GET" && method != "HEAD", terminalIndex: terminal,
		chain: append([]RouteExecutionStep(nil), steps...),
	}
}

func dispatchPluginStep(phase RouteExecutionPhase, id, action string) RouteExecutionStep {
	return RouteExecutionStep{
		Phase: phase, Action: action, RouteID: id, ContractVersion: id + "@1",
		Provider: Provider{Kind: ProviderPlugin, Artifact: PluginArtifact{
			ExtensionID: "demo.route", ExtensionVersion: "1.0.0",
			PackageDigest:     "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			RuntimeInstanceID: "runtime-1",
		}},
		Guard: extensionmanifest.GuardCorePublic, Access: "public", Mode: extensionmanifest.RouteModeHTTP,
		Handler: "route.handle", RequestSchema: id + ".request@1", ResponseSchema: id + ".response@1",
		Fallback: "closed",
	}
}

func dispatchCoreStep(id string) RouteExecutionStep {
	return RouteExecutionStep{
		Phase: RoutePhaseHandler, Action: extensionmanifest.RouteActionAdd, RouteID: id,
		ContractVersion: id + "@1", Provider: Provider{Kind: ProviderCore}, Mode: extensionmanifest.RouteModeHTTP,
	}
}
