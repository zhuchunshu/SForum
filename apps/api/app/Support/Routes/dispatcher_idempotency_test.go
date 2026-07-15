package routes

import (
	"context"
	"errors"
	"net/http"
	"testing"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

func TestDispatcherCompletesRequiredIdempotencyAroundFirstExecution(t *testing.T) {
	step := dispatchPluginStep(RoutePhaseHandler, "demo.route.create", extensionmanifest.RouteActionAdd)
	lease := &dispatchIdempotencyLease{}
	replay := &dispatchIdempotencyController{lease: lease}
	invoker := &dispatchStepInvoker{invoke: func(context.Context, RouteInvocation) (RouteInvocationResult, error) {
		response := DispatchResponse{Status: http.StatusCreated, Body: []byte(`{"id":42}`)}
		return RouteInvocationResult{Response: &response}, nil
	}}
	dispatcher := NewDispatcher(DispatcherConfig{
		Plans: dispatchPlanResolver{plan: dispatchPlan("POST", "/custom", nil, []RouteExecutionStep{step}, 0)},
		Steps: invoker, Guard: &dispatchGuard{}, Schemas: &dispatchSchemas{},
		Policies:    dispatchPolicyResolver{policy: RouteExecutionPolicy{Idempotency: "required.24h@1", IdempotencyRequired: true}},
		Idempotency: replay,
	})
	result, err := dispatcher.Dispatch(context.Background(), DispatchRequest{Method: "POST", Path: "/custom"}, nil)
	if err != nil || !result.Handled || result.Response.Status != http.StatusCreated ||
		replay.calls != 1 || lease.completeCalls != 1 || lease.abortCalls != 0 {
		t.Fatalf("result=%#v replay=%#v lease=%#v err=%v", result, replay, lease, err)
	}
}

func TestDispatcherReauthorizesReplayWithoutCallingPlugin(t *testing.T) {
	step := dispatchPluginStep(RoutePhaseHandler, "demo.route.create", extensionmanifest.RouteActionAdd)
	guard := &dispatchGuard{}
	replay := &dispatchIdempotencyController{replay: &DispatchResponse{
		Status: http.StatusCreated, Body: []byte(`{"id":42}`),
	}}
	invoker := &dispatchStepInvoker{invoke: func(context.Context, RouteInvocation) (RouteInvocationResult, error) {
		t.Fatal("replay invoked plugin")
		return RouteInvocationResult{}, nil
	}}
	dispatcher := NewDispatcher(DispatcherConfig{
		Plans: dispatchPlanResolver{plan: dispatchPlan("POST", "/custom", nil, []RouteExecutionStep{step}, 0)},
		Steps: invoker, Guard: guard, Schemas: &dispatchSchemas{},
		Policies:    dispatchPolicyResolver{policy: RouteExecutionPolicy{IdempotencyRequired: true}},
		Idempotency: replay,
	})
	result, err := dispatcher.Dispatch(context.Background(), DispatchRequest{Method: "POST", Path: "/custom"}, nil)
	if err != nil || !result.Handled || guard.calls != 1 || replay.calls != 1 {
		t.Fatalf("result=%#v guard=%d replay=%d err=%v", result, guard.calls, replay.calls, err)
	}

	guard.err = errors.New("permission revoked")
	if _, err := dispatcher.Dispatch(context.Background(), DispatchRequest{Method: "POST", Path: "/custom"}, nil); !errors.Is(err, ErrDispatchDenied) {
		t.Fatalf("revoked replay error = %v", err)
	}
}

func TestDispatcherAbortsFailedExecutionButPreservesUnknownCompletion(t *testing.T) {
	step := dispatchPluginStep(RoutePhaseHandler, "demo.route.create", extensionmanifest.RouteActionAdd)
	policy := dispatchPolicyResolver{policy: RouteExecutionPolicy{IdempotencyRequired: true}}
	t.Run("handler failure releases lease", func(t *testing.T) {
		lease := &dispatchIdempotencyLease{}
		dispatcher := NewDispatcher(DispatcherConfig{
			Plans: dispatchPlanResolver{plan: dispatchPlan("POST", "/custom", nil, []RouteExecutionStep{step}, 0)},
			Steps: &dispatchStepInvoker{invoke: func(context.Context, RouteInvocation) (RouteInvocationResult, error) {
				return RouteInvocationResult{}, errors.New("plugin failed")
			}},
			Guard: &dispatchGuard{}, Schemas: &dispatchSchemas{}, Policies: policy,
			Idempotency: &dispatchIdempotencyController{lease: lease},
		})
		if _, err := dispatcher.Dispatch(context.Background(), DispatchRequest{Method: "POST", Path: "/custom"}, nil); !errors.Is(err, ErrDispatchTransport) {
			t.Fatalf("error = %v", err)
		}
		if lease.abortCalls != 1 || lease.completeCalls != 0 {
			t.Fatalf("lease = %#v", lease)
		}
	})

	t.Run("completion failure keeps pending", func(t *testing.T) {
		lease := &dispatchIdempotencyLease{completeErr: errors.New("redis unavailable")}
		response := DispatchResponse{Status: http.StatusOK}
		dispatcher := NewDispatcher(DispatcherConfig{
			Plans: dispatchPlanResolver{plan: dispatchPlan("POST", "/custom", nil, []RouteExecutionStep{step}, 0)},
			Steps: &dispatchStepInvoker{invoke: func(context.Context, RouteInvocation) (RouteInvocationResult, error) {
				return RouteInvocationResult{Response: &response}, nil
			}},
			Guard: &dispatchGuard{}, Schemas: &dispatchSchemas{}, Policies: policy,
			Idempotency: &dispatchIdempotencyController{lease: lease},
		})
		if _, err := dispatcher.Dispatch(context.Background(), DispatchRequest{Method: "POST", Path: "/custom"}, nil); !errors.Is(err, ErrDispatchIdempotencyUnavailable) {
			t.Fatalf("error = %v", err)
		}
		if lease.completeCalls != 1 || lease.abortCalls != 0 {
			t.Fatalf("lease = %#v", lease)
		}
	})
}

type dispatchPolicyResolver struct {
	policy RouteExecutionPolicy
	err    error
}

func (r dispatchPolicyResolver) ResolveRouteExecutionPolicy(RouteExecutionStep) (RouteExecutionPolicy, error) {
	return r.policy, r.err
}

type dispatchIdempotencyController struct {
	lease  RouteIdempotencyLease
	replay *DispatchResponse
	err    error
	calls  int
}

func (c *dispatchIdempotencyController) Begin(
	context.Context,
	RouteExecutionPlan,
	RouteExecutionStep,
	RouteExecutionPolicy,
	DispatchRequest,
) (RouteIdempotencyLease, *DispatchResponse, error) {
	c.calls++
	return c.lease, c.replay, c.err
}

type dispatchIdempotencyLease struct {
	completeCalls int
	abortCalls    int
	completeErr   error
	abortErr      error
}

func (l *dispatchIdempotencyLease) Complete(context.Context, DispatchResponse) error {
	l.completeCalls++
	return l.completeErr
}

func (l *dispatchIdempotencyLease) Abort(context.Context) error {
	l.abortCalls++
	return l.abortErr
}
