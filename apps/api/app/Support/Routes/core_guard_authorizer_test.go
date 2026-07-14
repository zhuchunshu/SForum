package routes

import (
	"context"
	"errors"
	"testing"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

func TestCoreGuardEvaluatorRegistryIsVersionedExactAndInspectable(t *testing.T) {
	evaluator := CoreGuardEvaluatorFunc(func(context.Context, CoreGuardEvaluation) error { return nil })
	registry, err := NewCoreGuardEvaluatorRegistry([]CoreGuardEvaluatorRegistration{
		{EvaluatorID: "core.guard.test.second", ContractVersion: CoreGuardEvaluatorContractV1, Evaluator: evaluator},
		{EvaluatorID: "core.guard.test.first", ContractVersion: CoreGuardEvaluatorContractV1, Evaluator: evaluator},
	})
	if err != nil {
		t.Fatal(err)
	}
	bindings := registry.Bindings()
	if len(bindings) != 2 || bindings[0].EvaluatorID != "core.guard.test.first" ||
		bindings[1].EvaluatorID != "core.guard.test.second" ||
		bindings[0].ContractVersion != CoreGuardEvaluatorContractV1 {
		t.Fatalf("bindings = %#v", bindings)
	}
	bindings[0].EvaluatorID = "mutated"
	if registry.Bindings()[0].EvaluatorID != "core.guard.test.first" {
		t.Fatal("bindings escaped immutable registry")
	}

	invalid := [][]CoreGuardEvaluatorRegistration{
		{{EvaluatorID: "plugin.guard.test", ContractVersion: CoreGuardEvaluatorContractV1, Evaluator: evaluator}},
		{{EvaluatorID: "core.guard.test", ContractVersion: "sforum.route.guard_evaluator@2", Evaluator: evaluator}},
		{{EvaluatorID: "core.guard.test", ContractVersion: CoreGuardEvaluatorContractV1}},
		{
			{EvaluatorID: "core.guard.test", ContractVersion: CoreGuardEvaluatorContractV1, Evaluator: evaluator},
			{EvaluatorID: "core.guard.test", ContractVersion: CoreGuardEvaluatorContractV1, Evaluator: evaluator},
		},
	}
	for index, registrations := range invalid {
		if _, err := NewCoreGuardEvaluatorRegistry(registrations); !errors.Is(err, ErrCoreGuardRegistryInvalid) {
			t.Fatalf("invalid registration %d error = %v", index, err)
		}
	}
}

func TestCoreGuardAuthorizerExecutesTypedInheritedKinds(t *testing.T) {
	tests := []struct {
		name       string
		descriptor CoreGuardDescriptor
		request    DispatchRequest
		want       error
	}{
		{name: "public", descriptor: CoreGuardDescriptor{Kind: CoreGuardPublic}},
		{name: "login", descriptor: CoreGuardDescriptor{Kind: CoreGuardLogin}, request: authenticatedGuardRequest()},
		{name: "login denied", descriptor: CoreGuardDescriptor{Kind: CoreGuardLogin}, want: ErrCoreGuardLoginRequired},
		{name: "guest", descriptor: CoreGuardDescriptor{Kind: CoreGuardGuest}},
		{name: "guest denied", descriptor: CoreGuardDescriptor{Kind: CoreGuardGuest}, request: authenticatedGuardRequest(), want: ErrCoreGuardGuestRequired},
		{name: "super admin", descriptor: CoreGuardDescriptor{Kind: CoreGuardSuperAdmin}, request: guardRequestWithPermissions("*")},
		{name: "super admin denied", descriptor: CoreGuardDescriptor{Kind: CoreGuardSuperAdmin}, request: authenticatedGuardRequest(), want: ErrCoreGuardPermissionDenied},
		{name: "permission any", descriptor: CoreGuardDescriptor{Kind: CoreGuardPermissionAny, Permissions: []string{"topic.edit_own", "topic.edit_any"}}, request: guardRequestWithPermissions("topic.edit_any")},
		{name: "permission denied", descriptor: CoreGuardDescriptor{Kind: CoreGuardPermissionAny, Permissions: []string{"topic.edit_own", "topic.edit_any"}}, request: authenticatedGuardRequest(), want: ErrCoreGuardPermissionDenied},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan, step := inheritedCoreGuardPlan(t, test.descriptor)
			request := test.request
			request.Method, request.Path = plan.Method(), plan.Path()
			err := (CoreGuardAuthorizer{}).Authorize(context.Background(), plan, step, request)
			if !errors.Is(err, test.want) || test.want == nil && err != nil {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestCoreGuardAuthorizerRequiresExplicitContextualEvaluator(t *testing.T) {
	descriptor := CoreGuardDescriptor{
		Kind: CoreGuardContextual, Permissions: []string{"topic.edit_any"}, EvaluatorID: "core.guard.test.contextual",
	}
	plan, step := inheritedCoreGuardPlan(t, descriptor)
	request := guardRequestWithPermissions("topic.edit_any")
	request.Method, request.Path = plan.Method(), plan.Path()
	if err := (CoreGuardAuthorizer{}).Authorize(context.Background(), plan, step, request); !errors.Is(err, ErrCoreGuardEvaluatorUnavailable) {
		t.Fatalf("missing evaluator error = %v", err)
	}

	called := 0
	registry := MustNewCoreGuardEvaluatorRegistry([]CoreGuardEvaluatorRegistration{{
		EvaluatorID: descriptor.EvaluatorID, ContractVersion: CoreGuardEvaluatorContractV1,
		Evaluator: CoreGuardEvaluatorFunc(func(_ context.Context, evaluation CoreGuardEvaluation) error {
			called++
			if evaluation.Descriptor.RouteID != step.TargetID || evaluation.PlanRevision != plan.Revision() ||
				evaluation.RequestMethod != "PATCH" || evaluation.RequestPath != "/guard/alias" ||
				evaluation.Request.ActorID != request.ActorID {
				t.Fatalf("evaluation = %#v", evaluation)
			}
			evaluation.Descriptor.Permissions[0] = "mutated"
			evaluation.Step.CoreGuard.Permissions[0] = "mutated"
			evaluation.Request.Permissions["mutated"] = true
			return nil
		}),
	}})
	authorizer := CoreGuardAuthorizer{Evaluators: registry}
	if err := authorizer.Authorize(context.Background(), plan, step, request); err != nil {
		t.Fatal(err)
	}
	if called != 1 || step.CoreGuard.Permissions[0] != "topic.edit_any" ||
		plan.Terminal().CoreGuard.Permissions[0] != "topic.edit_any" || request.Permissions["mutated"] {
		t.Fatal("contextual evaluator mutated caller-owned authority data")
	}
}

func TestCoreGuardAuthorizerRejectsForgedInheritedAuthority(t *testing.T) {
	plan, step := inheritedCoreGuardPlan(t, CoreGuardDescriptor{Kind: CoreGuardPublic})
	request := DispatchRequest{Method: plan.Method(), Path: plan.Path()}
	for _, mutate := range []func(*RouteExecutionStep, *DispatchRequest){
		func(step *RouteExecutionStep, _ *DispatchRequest) { step.TargetID = "core.route.forged" },
		func(step *RouteExecutionStep, _ *DispatchRequest) {
			step.CoreGuard.ContractVersion = "sforum.route.forged@1"
		},
		func(step *RouteExecutionStep, _ *DispatchRequest) { step.CoreGuard.Kind = CoreGuardSuperAdmin },
		func(_ *RouteExecutionStep, request *DispatchRequest) { request.Method = "GET" },
		func(_ *RouteExecutionStep, request *DispatchRequest) { request.Path = "/forged" },
	} {
		candidateStep := step
		candidateStep.CoreGuard = cloneCoreGuardDescriptor(step.CoreGuard)
		candidateRequest := cloneDispatchRequest(request)
		mutate(&candidateStep, &candidateRequest)
		if err := (CoreGuardAuthorizer{}).Authorize(context.Background(), plan, candidateStep, candidateRequest); !errors.Is(err, ErrCoreGuardEvaluatorUnavailable) {
			t.Fatalf("forged authority error = %v", err)
		}
	}
}

func TestCoreGuardAuthorizerPreservesDirectPluginGuardsAndClosesCustomAuthority(t *testing.T) {
	authorizer := CoreGuardAuthorizer{}
	authenticated := guardRequestWithPermissions("topic.create")
	tests := []struct {
		guard      string
		permission string
		request    DispatchRequest
		want       error
	}{
		{guard: extensionmanifest.GuardCorePublic},
		{guard: extensionmanifest.GuardCoreLogin, request: authenticated},
		{guard: extensionmanifest.GuardCoreLogin, want: ErrCoreGuardLoginRequired},
		{guard: extensionmanifest.GuardCoreGuest},
		{guard: extensionmanifest.GuardCoreGuest, request: authenticated, want: ErrCoreGuardGuestRequired},
		{guard: extensionmanifest.GuardCoreGuest, request: DispatchRequest{ActorID: 42}, want: ErrCoreGuardGuestRequired},
		{guard: extensionmanifest.GuardCorePermission, permission: "topic.create", request: authenticated},
		{guard: extensionmanifest.GuardCorePermission, permission: "topic.create", request: authenticatedGuardRequest(), want: ErrCoreGuardPermissionDenied},
		{guard: extensionmanifest.GuardCoreRaw, request: authenticated, want: ErrCoreGuardEvaluatorUnavailable},
		{guard: "guard.direct.custom", request: authenticated, want: ErrCoreGuardEvaluatorUnavailable},
	}
	for index, test := range tests {
		plan, step := directCoreGuardPlan(t, test.guard, test.permission)
		request := test.request
		request.Method, request.Path = plan.Method(), plan.Path()
		err := authorizer.Authorize(context.Background(), plan, step, request)
		if !errors.Is(err, test.want) || test.want == nil && err != nil {
			t.Fatalf("case %d error = %v, want %v", index, err, test.want)
		}
	}
}

func TestCoreGuardAuthorizerRejectsDirectGuardOutsideExactPlan(t *testing.T) {
	plan, step := directCoreGuardPlan(t, extensionmanifest.GuardCorePublic, "")
	request := DispatchRequest{Method: plan.Method(), Path: plan.Path()}
	for _, mutate := range []func(*RouteExecutionStep, *DispatchRequest){
		func(step *RouteExecutionStep, _ *DispatchRequest) { step.RouteID = "guard.plugin.forged" },
		func(step *RouteExecutionStep, _ *DispatchRequest) { step.Provider = Provider{} },
		func(_ *RouteExecutionStep, request *DispatchRequest) { request.Method = "POST" },
		func(_ *RouteExecutionStep, request *DispatchRequest) { request.Path = "/forged" },
	} {
		candidateStep := step
		candidateRequest := cloneDispatchRequest(request)
		mutate(&candidateStep, &candidateRequest)
		if err := (CoreGuardAuthorizer{}).Authorize(context.Background(), plan, candidateStep, candidateRequest); !errors.Is(err, ErrCoreGuardEvaluatorUnavailable) {
			t.Fatalf("forged direct guard error = %v", err)
		}
	}
}

func inheritedCoreGuardPlan(t *testing.T, descriptor CoreGuardDescriptor) (RouteExecutionPlan, RouteExecutionStep) {
	t.Helper()
	registry := NewRegistry()
	target := coreRoute("core.route.guard.target", "PATCH", "/guard/target")
	descriptor.RouteID = target.ID
	descriptor.ContractVersion = target.ContractVersion
	descriptor.Method = target.Method
	target.Guard = descriptor
	alias := pluginRoute("guard.plugin.alias", "/guard/alias", 0, "PATCH")
	alias.Action, alias.TargetID, alias.Handler, alias.ResponseSchema = extensionmanifest.RouteActionAlias, target.ID, "", ""
	alias.Guard = extensionmanifest.GuardCoreInherit
	if _, err := registry.Publish(Publication{
		Core:    []CoreRoute{target},
		Plugins: []PluginRouteSet{{Artifact: routeArtifact("guard.plugin", "1.0.0", 'a'), Routes: []extensionmanifest.ManifestRoute{alias}}},
	}); err != nil {
		t.Fatal(err)
	}
	plan, err := registry.BuildExecutionPlan("PATCH", "/guard/alias")
	if err != nil {
		t.Fatal(err)
	}
	return plan, plan.Terminal()
}

func directCoreGuardPlan(t *testing.T, guard, permission string) (RouteExecutionPlan, RouteExecutionStep) {
	t.Helper()
	registry := NewRegistry()
	route := pluginRoute("guard.direct.route", "/guard/direct", 0, "GET")
	route.Guard, route.Permission = guard, permission
	var guards []extensionmanifest.ManifestGuard
	if guard == "guard.direct.custom" {
		guards = []extensionmanifest.ManifestGuard{pluginGuard(guard, "custom")}
	}
	if _, err := registry.Publish(Publication{Plugins: []PluginRouteSet{{
		Artifact: routeArtifact("guard.direct", "1.0.0", 'b'), Routes: []extensionmanifest.ManifestRoute{route}, Guards: guards,
	}}}); err != nil {
		t.Fatal(err)
	}
	plan, err := registry.BuildExecutionPlan("GET", route.Path)
	if err != nil {
		t.Fatal(err)
	}
	return plan, plan.Terminal()
}

func authenticatedGuardRequest() DispatchRequest {
	return DispatchRequest{ActorID: 42, Authenticated: true, Permissions: map[string]bool{}}
}

func guardRequestWithPermissions(permissions ...string) DispatchRequest {
	request := authenticatedGuardRequest()
	for _, permission := range permissions {
		request.Permissions[permission] = true
	}
	return request
}
