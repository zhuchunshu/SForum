package http

import (
	"context"
	"errors"
	"strings"
	"testing"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

func TestProductionRouteGuardAuthorizerMapsTypedGuardFailures(t *testing.T) {
	authorizer := NewProductionRouteGuardAuthorizer()
	tests := []struct {
		guard      string
		permission string
		request    routes.DispatchRequest
		want       error
	}{
		{guard: extensionmanifest.GuardCoreLogin, want: ErrRouteLoginRequired},
		{guard: extensionmanifest.GuardCoreGuest, request: productionGuardRequest(), want: ErrRouteGuestRequired},
		{guard: extensionmanifest.GuardCorePermission, permission: "topic.create", request: productionGuardRequest(), want: ErrRoutePermissionDenied},
		{guard: extensionmanifest.GuardCoreRaw, request: productionGuardRequest(), want: ErrRouteGuardUnavailable},
	}
	for index, test := range tests {
		plan, step := productionDirectGuardPlan(t, test.guard, test.permission)
		request := test.request
		request.Method, request.Path = plan.Method(), plan.Path()
		err := authorizer.Authorize(context.Background(), plan, step, request)
		if !errors.Is(err, test.want) {
			t.Fatalf("case %d error = %v, want %v", index, err, test.want)
		}
	}
}

func TestProductionRouteGuardAuthorizerRejectsForgedDirectPublicStep(t *testing.T) {
	plan, step := productionDirectGuardPlan(t, extensionmanifest.GuardCorePublic, "")
	request := routes.DispatchRequest{Method: plan.Method(), Path: plan.Path()}
	step.RouteID = "guard.production.forged"
	if err := NewProductionRouteGuardAuthorizer().Authorize(context.Background(), plan, step, request); !errors.Is(err, ErrRouteGuardUnavailable) {
		t.Fatalf("forged public guard error = %v", err)
	}
}

func TestProductionRouteGuardAuthorizerRunsExplicitContextualAdapters(t *testing.T) {
	authorizer := NewProductionRouteGuardAuthorizer()
	tests := []struct {
		name        string
		descriptor  routes.CoreGuardDescriptor
		permissions []string
		want        error
	}{
		{
			name: "profile self", descriptor: routes.CoreGuardDescriptor{
				Kind: routes.CoreGuardContextual, EvaluatorID: "core.guard.profile.self",
			},
		},
		{
			name: "profile upload denied", descriptor: routes.CoreGuardDescriptor{
				Kind: routes.CoreGuardContextual, EvaluatorID: "core.guard.profile.self",
			}, want: ErrRoutePermissionDenied,
		},
		{
			name: "profile upload", descriptor: routes.CoreGuardDescriptor{
				Kind: routes.CoreGuardContextual, EvaluatorID: "core.guard.profile.self",
			}, permissions: []string{identity.PermissionAttachmentUpload},
		},
		{
			name: "topic own edit remains closed", descriptor: routes.CoreGuardDescriptor{
				Kind: routes.CoreGuardContextual, Permissions: []string{identity.PermissionTopicEditOwn, identity.PermissionTopicEditAny},
				EvaluatorID: "core.guard.forum.topic_edit",
			}, permissions: []string{identity.PermissionTopicEditOwn}, want: ErrRoutePermissionDenied,
		},
		{
			name: "topic global edit", descriptor: routes.CoreGuardDescriptor{
				Kind: routes.CoreGuardContextual, Permissions: []string{identity.PermissionTopicEditOwn, identity.PermissionTopicEditAny},
				EvaluatorID: "core.guard.forum.topic_edit",
			}, permissions: []string{identity.PermissionTopicEditAny},
		},
		{
			name: "unknown evaluator", descriptor: routes.CoreGuardDescriptor{
				Kind: routes.CoreGuardContextual, EvaluatorID: "core.guard.unregistered",
			}, want: ErrRouteGuardUnavailable,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			routeID := "core.route.profile.my_profile"
			if strings.Contains(test.name, "upload") {
				routeID = "core.route.profile.upload_avatar"
			}
			if strings.Contains(test.name, "topic") {
				routeID = "core.route.forum.update_topic"
			}
			if strings.Contains(test.name, "unknown") {
				routeID = "core.route.test.unknown"
			}
			plan, step := productionInheritedGuardPlan(t, routeID, test.descriptor)
			request := productionGuardRequest(test.permissions...)
			request.Method, request.Path = plan.Method(), plan.Path()
			err := authorizer.Authorize(context.Background(), plan, step, request)
			if !errors.Is(err, test.want) || test.want == nil && err != nil {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestProductionCoreGuardEvaluatorRegistryIsExplicitAndVersioned(t *testing.T) {
	registrations := productionCoreGuardEvaluatorRegistrations()
	registry, err := routes.NewCoreGuardEvaluatorRegistry(registrations)
	if err != nil {
		t.Fatal(err)
	}
	bindings := registry.Bindings()
	if len(bindings) != len(registrations) || len(bindings) != 11 {
		t.Fatalf("bindings = %#v", bindings)
	}
	for _, binding := range bindings {
		if binding.ContractVersion != routes.CoreGuardEvaluatorContractV1 || !strings.HasPrefix(binding.EvaluatorID, "core.guard.") {
			t.Fatalf("binding = %#v", binding)
		}
	}
}

func productionInheritedGuardPlan(
	t *testing.T,
	targetID string,
	descriptor routes.CoreGuardDescriptor,
) (routes.RouteExecutionPlan, routes.RouteExecutionStep) {
	t.Helper()
	const method = "PATCH"
	contract := "sforum.route." + strings.TrimPrefix(targetID, "core.route.") + "@1"
	descriptor.RouteID, descriptor.ContractVersion, descriptor.Method = targetID, contract, method
	registry := routes.NewRegistry()
	alias := extensionmanifest.ManifestRoute{
		ID: "guard.production.alias", ContractVersion: "guard.production.alias@1",
		Action: extensionmanifest.RouteActionAlias, TargetID: targetID,
		Path: "/guard/production", Methods: []string{method}, Guard: extensionmanifest.GuardCoreInherit,
		Fallback: "closed", Mode: extensionmanifest.RouteModeHTTP,
	}
	if _, err := registry.Publish(routes.Publication{
		Core: []routes.CoreRoute{{ID: targetID, ContractVersion: contract, Method: method, Path: "/guard/target", Guard: descriptor}},
		Plugins: []routes.PluginRouteSet{{
			Artifact: routes.PluginArtifact{
				ExtensionID: "guard.production", ExtensionVersion: "1.0.0",
				PackageDigest: strings.Repeat("a", 64), RuntimeInstanceID: "runtime-a",
			},
			Routes: []extensionmanifest.ManifestRoute{alias},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	plan, err := registry.BuildExecutionPlan(method, alias.Path)
	if err != nil {
		t.Fatal(err)
	}
	return plan, plan.Terminal()
}

func productionDirectGuardPlan(t *testing.T, guard, permission string) (routes.RouteExecutionPlan, routes.RouteExecutionStep) {
	t.Helper()
	registry := routes.NewRegistry()
	route := extensionmanifest.ManifestRoute{
		ID: "guard.production.direct", ContractVersion: "guard.production.direct@1",
		Action: extensionmanifest.RouteActionAdd, Path: "/guard/direct", Methods: []string{"GET"},
		Guard: guard, Permission: permission, Fallback: "closed", Mode: extensionmanifest.RouteModeHTTP,
		Handler: "route.handle", ResponseSchema: "guard.production.direct.response@1",
	}
	if _, err := registry.Publish(routes.Publication{Plugins: []routes.PluginRouteSet{{
		Artifact: routes.PluginArtifact{
			ExtensionID: "guard.production", ExtensionVersion: "1.0.0",
			PackageDigest: strings.Repeat("b", 64), RuntimeInstanceID: "runtime-b",
		},
		Routes: []extensionmanifest.ManifestRoute{route},
	}}}); err != nil {
		t.Fatal(err)
	}
	plan, err := registry.BuildExecutionPlan("GET", route.Path)
	if err != nil {
		t.Fatal(err)
	}
	return plan, plan.Terminal()
}

func productionGuardRequest(permissions ...string) routes.DispatchRequest {
	request := routes.DispatchRequest{ActorID: 42, Authenticated: true, Permissions: map[string]bool{}}
	for _, permission := range permissions {
		request.Permissions[permission] = true
	}
	return request
}
