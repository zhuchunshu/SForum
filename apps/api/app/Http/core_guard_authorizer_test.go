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
	if len(bindings) != len(registrations) || len(bindings) != 13 {
		t.Fatalf("bindings = %#v", bindings)
	}
	for _, binding := range bindings {
		if binding.ContractVersion != routes.CoreGuardEvaluatorContractV1 || !strings.HasPrefix(binding.EvaluatorID, "core.guard.") {
			t.Fatalf("binding = %#v", binding)
		}
	}
}

func TestProductionPagesAdminGuardClosesCatalogModule(t *testing.T) {
	type expectedRoute struct {
		method     string
		superAdmin bool
	}
	expected := map[string]expectedRoute{
		"core.route.pages.admin_list":       {method: "GET"},
		"core.route.pages.admin_get":        {method: "GET"},
		"core.route.pages.admin_approve":    {method: "POST", superAdmin: true},
		"core.route.pages.admin_restore":    {method: "POST", superAdmin: true},
		"core.route.pages.activate_preview": {method: "GET"},
		"core.route.pages.admin_added":      {method: "GET"},
	}
	var catalog []routes.CoreRoute
	for _, route := range routes.CoreRouteCatalog() {
		if route.Guard.EvaluatorID == "core.guard.pages.admin" {
			catalog = append(catalog, route)
		}
	}
	if len(catalog) != len(expected) {
		t.Fatalf("pages admin contextual catalog = %#v", catalog)
	}

	authorizer := NewProductionRouteGuardAuthorizer()
	for _, route := range catalog {
		want, exists := expected[route.ID]
		if !exists || route.Method != want.method || route.Guard.Kind != routes.CoreGuardContextual ||
			len(route.Guard.Permissions) != 0 {
			t.Fatalf("unexpected pages admin guard route = %#v", route)
		}
		delete(expected, route.ID)

		plan, step := productionCatalogInheritedGuardPlan(t, route)
		permission := identity.PermissionExtensionView
		if want.superAdmin {
			permission = "*"
		}
		allowed := productionGuardRequest(permission)
		allowed.Method, allowed.Path = plan.Method(), plan.Path()
		if err := authorizer.Authorize(context.Background(), plan, step, allowed); err != nil {
			t.Fatalf("%s allowed error = %v", route.ID, err)
		}

		denied := productionGuardRequest()
		denied.Method, denied.Path = plan.Method(), plan.Path()
		if err := authorizer.Authorize(context.Background(), plan, step, denied); !errors.Is(err, ErrRoutePermissionDenied) {
			t.Fatalf("%s permission denied error = %v", route.ID, err)
		}
		if want.superAdmin {
			themeManager := productionGuardRequest(identity.PermissionExtensionThemeManage)
			themeManager.Method, themeManager.Path = plan.Method(), plan.Path()
			if err := authorizer.Authorize(context.Background(), plan, step, themeManager); !errors.Is(err, ErrRoutePermissionDenied) {
				t.Fatalf("%s theme manager downgrade error = %v", route.ID, err)
			}
		}

		anonymous := routes.DispatchRequest{Method: plan.Method(), Path: plan.Path()}
		if err := authorizer.Authorize(context.Background(), plan, step, anonymous); !errors.Is(err, ErrRouteLoginRequired) {
			t.Fatalf("%s anonymous error = %v", route.ID, err)
		}
		forgedActor := allowed
		forgedActor.ActorID = 0
		if err := authorizer.Authorize(context.Background(), plan, step, forgedActor); !errors.Is(err, ErrRouteLoginRequired) {
			t.Fatalf("%s forged actor error = %v", route.ID, err)
		}

		forgedRequest := allowed
		forgedRequest.Path += "/forged"
		if err := authorizer.Authorize(context.Background(), plan, step, forgedRequest); !errors.Is(err, ErrRouteGuardUnavailable) {
			t.Fatalf("%s forged path error = %v", route.ID, err)
		}
		forgedRequest = allowed
		if forgedRequest.Method == "GET" {
			forgedRequest.Method = "POST"
		} else {
			forgedRequest.Method = "GET"
		}
		if err := authorizer.Authorize(context.Background(), plan, step, forgedRequest); !errors.Is(err, ErrRouteGuardUnavailable) {
			t.Fatalf("%s forged method error = %v", route.ID, err)
		}

		forgedStep := step
		forgedStep.RouteID += ".forged"
		if err := authorizer.Authorize(context.Background(), plan, forgedStep, allowed); !errors.Is(err, ErrRouteGuardUnavailable) {
			t.Fatalf("%s forged step error = %v", route.ID, err)
		}
	}
	if len(expected) != 0 {
		t.Fatalf("missing pages admin guard routes = %#v", expected)
	}
}

func TestProductionPagesAdminGuardRequiresExactParsedResource(t *testing.T) {
	var target routes.CoreRoute
	for _, route := range routes.CoreRouteCatalog() {
		if route.ID == "core.route.pages.admin_get" {
			target = route
			break
		}
	}
	if target.ID == "" {
		t.Fatal("pages admin get route is missing")
	}
	plan, step := productionParameterizedInheritedGuardPlan(t, target,
		"/guard/production/pages/:pageId", "/guard/production/pages/forum.home")
	request := productionGuardRequest(identity.PermissionExtensionView)
	request.Method, request.Path, request.Params = plan.Method(), plan.Path(), plan.Params()
	if err := NewProductionRouteGuardAuthorizer().Authorize(context.Background(), plan, step, request); err != nil {
		t.Fatalf("exact resource error = %v", err)
	}

	request.Params["pageId"] = "admin.secret"
	if err := NewProductionRouteGuardAuthorizer().Authorize(context.Background(), plan, step, request); !errors.Is(err, ErrRouteGuardUnavailable) {
		t.Fatalf("forged resource error = %v", err)
	}
}

func TestProductionPagesAdminGuardRejectsForeignRouteID(t *testing.T) {
	descriptor := routes.CoreGuardDescriptor{
		Kind: routes.CoreGuardContextual, EvaluatorID: "core.guard.pages.admin",
	}
	plan, step := productionInheritedGuardPlan(t, "core.route.pages.foreign", descriptor)
	request := productionGuardRequest(identity.PermissionExtensionView)
	request.Method, request.Path = plan.Method(), plan.Path()
	if err := NewProductionRouteGuardAuthorizer().Authorize(context.Background(), plan, step, request); !errors.Is(err, ErrRouteGuardUnavailable) {
		t.Fatalf("foreign pages admin route error = %v", err)
	}
}

func TestProductionNotificationRecipientGuardClosesCatalogModule(t *testing.T) {
	expected := map[string]string{
		"core.route.notifications.list":          "GET",
		"core.route.notifications.mark_read":     "PATCH",
		"core.route.notifications.mark_all_read": "POST",
		"core.route.notifications.unread_count":  "GET",
	}
	var catalog []routes.CoreRoute
	for _, route := range routes.CoreRouteCatalog() {
		if route.Guard.EvaluatorID == "core.guard.notifications.recipient" {
			catalog = append(catalog, route)
		}
	}
	if len(catalog) != len(expected) {
		t.Fatalf("notification contextual catalog = %#v", catalog)
	}

	authorizer := NewProductionRouteGuardAuthorizer()
	for _, route := range catalog {
		method, exists := expected[route.ID]
		if !exists || route.Method != method || route.Guard.Kind != routes.CoreGuardContextual {
			t.Fatalf("unexpected notification guard route = %#v", route)
		}
		delete(expected, route.ID)

		plan, step := productionCatalogInheritedGuardPlan(t, route)
		// 通知收件箱是登录主体能力，不要求额外 scope；PAT 权限仍保持收窄，
		// 资源所有权由核心通知查询的 recipient_user_id 条件继续强制。
		patRequest := productionGuardRequest(identity.PermissionPostCreate)
		patRequest.Method, patRequest.Path = plan.Method(), plan.Path()
		if err := authorizer.Authorize(context.Background(), plan, step, patRequest); err != nil {
			t.Fatalf("%s scoped PAT error = %v", route.ID, err)
		}

		anonymous := routes.DispatchRequest{Method: plan.Method(), Path: plan.Path()}
		if err := authorizer.Authorize(context.Background(), plan, step, anonymous); !errors.Is(err, ErrRouteLoginRequired) {
			t.Fatalf("%s anonymous error = %v", route.ID, err)
		}

		forgedRequest := patRequest
		forgedRequest.Path += "/forged"
		if err := authorizer.Authorize(context.Background(), plan, step, forgedRequest); !errors.Is(err, ErrRouteGuardUnavailable) {
			t.Fatalf("%s forged request error = %v", route.ID, err)
		}

		forgedStep := step
		forgedStep.RouteID += ".forged"
		if err := authorizer.Authorize(context.Background(), plan, forgedStep, patRequest); !errors.Is(err, ErrRouteGuardUnavailable) {
			t.Fatalf("%s forged step error = %v", route.ID, err)
		}
	}
	if len(expected) != 0 {
		t.Fatalf("missing notification guard routes = %#v", expected)
	}
}

func TestProductionNotificationRecipientGuardRejectsForeignRouteID(t *testing.T) {
	descriptor := routes.CoreGuardDescriptor{
		Kind: routes.CoreGuardContextual, EvaluatorID: "core.guard.notifications.recipient",
	}
	plan, step := productionInheritedGuardPlan(t, "core.route.notifications.foreign", descriptor)
	request := productionGuardRequest()
	request.Method, request.Path = plan.Method(), plan.Path()
	if err := NewProductionRouteGuardAuthorizer().Authorize(context.Background(), plan, step, request); !errors.Is(err, ErrRouteGuardUnavailable) {
		t.Fatalf("foreign notification route error = %v", err)
	}
}

func productionCatalogInheritedGuardPlan(
	t *testing.T,
	target routes.CoreRoute,
) (routes.RouteExecutionPlan, routes.RouteExecutionStep) {
	t.Helper()
	registry := routes.NewRegistry()
	alias := extensionmanifest.ManifestRoute{
		ID: "guard.production.catalog_alias", ContractVersion: "guard.production.catalog_alias@1",
		Action: extensionmanifest.RouteActionAlias, TargetID: target.ID,
		Path: "/guard/production/catalog", Methods: []string{target.Method}, Guard: extensionmanifest.GuardCoreInherit,
		Fallback: "closed", Mode: extensionmanifest.RouteModeHTTP,
	}
	if _, err := registry.Publish(routes.Publication{
		Core: []routes.CoreRoute{target},
		Plugins: []routes.PluginRouteSet{{
			Artifact: routes.PluginArtifact{
				ExtensionID: "guard.production", ExtensionVersion: "1.0.0",
				PackageDigest: strings.Repeat("c", 64), RuntimeInstanceID: "runtime-c",
			},
			Routes: []extensionmanifest.ManifestRoute{alias},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	plan, err := registry.BuildExecutionPlan(target.Method, alias.Path)
	if err != nil {
		t.Fatal(err)
	}
	return plan, plan.Terminal()
}

func productionParameterizedInheritedGuardPlan(
	t *testing.T,
	target routes.CoreRoute,
	aliasPath string,
	requestPath string,
) (routes.RouteExecutionPlan, routes.RouteExecutionStep) {
	t.Helper()
	registry := routes.NewRegistry()
	alias := extensionmanifest.ManifestRoute{
		ID: "guard.production.parameterized_alias", ContractVersion: "guard.production.parameterized_alias@1",
		Action: extensionmanifest.RouteActionAlias, TargetID: target.ID,
		Path: aliasPath, Methods: []string{target.Method}, Guard: extensionmanifest.GuardCoreInherit,
		Fallback: "closed", Mode: extensionmanifest.RouteModeHTTP,
	}
	if _, err := registry.Publish(routes.Publication{
		Core: []routes.CoreRoute{target},
		Plugins: []routes.PluginRouteSet{{
			Artifact: routes.PluginArtifact{
				ExtensionID: "guard.production", ExtensionVersion: "1.0.0",
				PackageDigest: strings.Repeat("d", 64), RuntimeInstanceID: "runtime-d",
			},
			Routes: []extensionmanifest.ManifestRoute{alias},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	plan, err := registry.BuildExecutionPlan(target.Method, requestPath)
	if err != nil {
		t.Fatal(err)
	}
	return plan, plan.Terminal()
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
