package http

import (
	"context"
	"errors"
	stdhttp "net/http"
	"strings"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

func TestRuntimePluginRouteGuardEvaluatorInvokesExactSanitizedGuard(t *testing.T) {
	plan, step := productionDirectGuardPlan(t, "guard.production.custom", "")
	policy := &testExtensionGuardPolicy{lookup: exactPluginGuardLookup(step), ok: true}
	runtime := newTestPluginGuardRuntime(t, step)
	evaluator := NewRuntimePluginRouteGuardEvaluator(runtime, policy)
	request := routes.DispatchRequest{
		Method: plan.Method(), Path: plan.Path(), Query: "preview=1", Params: plan.Params(),
		Headers: stdhttp.Header{
			"X-Request-ID": []string{"request-41"}, "Cookie": []string{"session=secret"},
			"Authorization": []string{"Bearer secret"}, "X-SForum-Forged": []string{"forged"},
		},
		ActorID: 42, Authenticated: true, Permissions: map[string]bool{"topic.read": true},
	}
	if err := evaluator.EvaluatePluginGuard(context.Background(), routes.PluginGuardEvaluation{
		PlanRevision: plan.Revision(), RequestMethod: plan.Method(), RequestPath: plan.Path(), Step: step, Request: request,
		Authority: resolvedPluginGuardTestAuthority(step),
	}); err != nil {
		t.Fatal(err)
	}
	if runtime.inspectCalls != 1 || runtime.acquireCalls != 1 || runtime.invokeCalls != 1 ||
		runtime.class != extensionsruntime.RuntimeCallGuard || runtime.request.GuardID != step.PluginGuard.ID ||
		runtime.request.GuardContractVersion != step.PluginGuard.ContractVersion ||
		runtime.request.RouteID != step.RouteID || runtime.request.RouteContractVersion != step.ContractVersion ||
		runtime.request.QueryParameters["preview"] != "1" || runtime.request.Actor.UserID != 42 ||
		runtime.request.Headers.Get("X-Request-ID") != "request-41" || runtime.request.Headers.Get("Cookie") != "" ||
		runtime.request.Headers.Get("Authorization") != "" || runtime.request.Headers.Get("X-SForum-Forged") != "" {
		t.Fatalf("runtime guard = %#v", runtime)
	}
}

func TestRuntimePluginRouteGuardEvaluatorMapsDeniedAndRuntimeFailure(t *testing.T) {
	plan, step := productionDirectGuardPlan(t, "guard.production.custom", "")
	request := routes.DispatchRequest{Method: plan.Method(), Path: plan.Path(), Params: plan.Params()}
	policy := &testExtensionGuardPolicy{lookup: exactPluginGuardLookup(step), ok: true}
	for _, test := range []struct {
		name string
		err  error
		want error
	}{
		{name: "denied", err: extensionsruntime.ErrProtocolV2GuardDenied, want: routes.ErrCoreGuardPermissionDenied},
		{name: "crash", err: errors.New("runtime crashed"), want: routes.ErrCoreGuardEvaluatorUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			runtime := newTestPluginGuardRuntime(t, step)
			runtime.invokeErr = test.err
			err := NewRuntimePluginRouteGuardEvaluator(runtime, policy).EvaluatePluginGuard(context.Background(), routes.PluginGuardEvaluation{
				PlanRevision: plan.Revision(), RequestMethod: plan.Method(), RequestPath: plan.Path(), Step: step, Request: request,
				Authority: resolvedPluginGuardTestAuthority(step),
			})
			if !errors.Is(err, test.want) || runtime.invokeCalls != 1 {
				t.Fatalf("error = %v, calls = %d", err, runtime.invokeCalls)
			}
		})
	}
}

func TestRuntimePluginRouteGuardEvaluatorFailsClosedOnTrustDrift(t *testing.T) {
	plan, step := productionDirectGuardPlan(t, "guard.production.custom", "")
	request := routes.DispatchRequest{Method: plan.Method(), Path: plan.Path(), Params: plan.Params()}
	base := exactPluginGuardLookup(step)
	tests := []struct {
		name   string
		mutate func(*extensions.GuardPolicyLookup)
	}{
		{name: "missing snapshot", mutate: func(value *extensions.GuardPolicyLookup) { value.Revision = 0 }},
		{name: "not found", mutate: func(value *extensions.GuardPolicyLookup) { value.Found = false }},
		{name: "safe mode", mutate: func(value *extensions.GuardPolicyLookup) { value.SafeMode = true }},
		{name: "disabled", mutate: func(value *extensions.GuardPolicyLookup) { value.Entry.Status = extensions.StatusDisabled }},
		{name: "version", mutate: func(value *extensions.GuardPolicyLookup) { value.Entry.Version = "2.0.0" }},
		{name: "digest", mutate: func(value *extensions.GuardPolicyLookup) { value.Entry.PackageDigest = strings.Repeat("d", 64) }},
		{name: "revoked", mutate: func(value *extensions.GuardPolicyLookup) { value.Entry.CurrentArtifactTrusted = false }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lookup := base
			test.mutate(&lookup)
			policy := &testExtensionGuardPolicy{lookup: lookup, ok: true}
			if lookup.Revision == 0 {
				policy.ok = false
			}
			runtime := newTestPluginGuardRuntime(t, step)
			err := NewRuntimePluginRouteGuardEvaluator(runtime, policy).EvaluatePluginGuard(context.Background(), routes.PluginGuardEvaluation{
				PlanRevision: plan.Revision(), RequestMethod: plan.Method(), RequestPath: plan.Path(), Step: step, Request: request,
				Authority: resolvedPluginGuardTestAuthority(step),
			})
			if !errors.Is(err, routes.ErrCoreGuardEvaluatorUnavailable) || runtime.invokeCalls != 0 {
				t.Fatalf("error = %v, calls = %d", err, runtime.invokeCalls)
			}
		})
	}
}

func TestRuntimePluginRouteGuardEvaluatorConfirmsRawAuthorityWithoutCredentials(t *testing.T) {
	plan, step := productionDirectGuardPlan(t, "core.guard.raw_request", "")
	policy := &testExtensionGuardPolicy{lookup: exactPluginGuardLookup(step), ok: true}
	runtime := newTestPluginGuardRuntime(t, step)
	err := NewRuntimePluginRouteGuardEvaluator(runtime, policy).EvaluatePluginGuard(context.Background(), routes.PluginGuardEvaluation{
		PlanRevision: plan.Revision(), RequestMethod: plan.Method(), RequestPath: plan.Path(), Step: step,
		Authority: resolvedPluginGuardTestAuthority(step),
		Request:   routes.DispatchRequest{Method: plan.Method(), Path: plan.Path(), Params: plan.Params()},
	})
	if err != nil || runtime.inspectCalls != 0 || runtime.invokeCalls != 0 {
		t.Fatalf("raw authority error = %v, runtime = %#v", err, runtime)
	}
}

func resolvedPluginGuardTestAuthority(step routes.RouteExecutionStep) routes.ResolvedRequestAuthority {
	if step.Guard == "core.guard.raw_request" || step.PluginGuard.Kind == "raw_request" {
		return routes.ResolvedRequestAuthority{Mode: routes.RequestAuthorityRaw, GuardKind: routes.RequestGuardRawRequest}
	}
	return routes.ResolvedRequestAuthority{Mode: routes.RequestAuthorityFiltered, GuardKind: routes.RequestGuardCustom}
}

func exactPluginGuardLookup(step routes.RouteExecutionStep) extensions.GuardPolicyLookup {
	return extensions.GuardPolicyLookup{
		Revision: 1, Found: true,
		Entry: extensions.GuardPolicyEntry{
			ExtensionID: step.Provider.Artifact.ExtensionID, ExtensionType: extensions.TypePlugin,
			Status: extensions.StatusEnabled, Version: step.Provider.Artifact.ExtensionVersion,
			PackageDigest:        step.Provider.Artifact.PackageDigest,
			CurrentTrustRequired: true, CurrentArtifactTrusted: true,
		},
	}
}

type testPluginGuardRuntime struct {
	snapshot     extensionsruntime.RuntimeInstanceSnapshot
	gate         *extensionsruntime.RuntimeAdmissionGate
	request      extensionsruntime.ProtocolV2GuardRequest
	class        extensionsruntime.RuntimeCallClass
	invokeErr    error
	inspectCalls int
	acquireCalls int
	invokeCalls  int
}

func newTestPluginGuardRuntime(t *testing.T, step routes.RouteExecutionStep) *testPluginGuardRuntime {
	t.Helper()
	identity := extensionsruntime.RuntimeInstanceIdentity{
		ExtensionID: step.Provider.Artifact.ExtensionID, InstanceID: step.Provider.Artifact.RuntimeInstanceID,
	}
	gate, err := extensionsruntime.NewRuntimeAdmissionGate(identity)
	if err != nil {
		t.Fatal(err)
	}
	return &testPluginGuardRuntime{
		gate: gate,
		snapshot: extensionsruntime.RuntimeInstanceSnapshot{
			Identity: identity, ExtensionVersion: step.Provider.Artifact.ExtensionVersion,
			ArtifactDigest: step.Provider.Artifact.PackageDigest, Active: true,
		},
	}
}

func (r *testPluginGuardRuntime) InspectRuntimeInstance(identity extensionsruntime.RuntimeInstanceIdentity) (extensionsruntime.RuntimeInstanceSnapshot, error) {
	r.inspectCalls++
	if identity != r.snapshot.Identity {
		return extensionsruntime.RuntimeInstanceSnapshot{}, extensionsruntime.ErrRuntimeInstanceNotFound
	}
	return r.snapshot, nil
}

func (r *testPluginGuardRuntime) AcquireRuntimeCall(
	ctx context.Context,
	identity extensionsruntime.RuntimeInstanceIdentity,
	class extensionsruntime.RuntimeCallClass,
) (*extensionsruntime.RuntimeAdmissionLease, error) {
	r.acquireCalls++
	r.class = class
	if identity != r.snapshot.Identity {
		return nil, extensionsruntime.ErrRuntimeInstanceNotFound
	}
	return r.gate.Acquire(ctx, class)
}

func (r *testPluginGuardRuntime) InvokeGuardInstance(
	_ context.Context,
	identity extensionsruntime.RuntimeInstanceIdentity,
	request extensionsruntime.ProtocolV2GuardRequest,
) error {
	r.invokeCalls++
	if identity != r.snapshot.Identity {
		return extensionsruntime.ErrRuntimeInstanceNotFound
	}
	r.request = request
	return r.invokeErr
}

var _ ExactPluginGuardRuntime = (*testPluginGuardRuntime)(nil)
