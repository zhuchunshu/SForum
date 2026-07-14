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
	if len(bindings) != len(registrations) || len(bindings) != 17 {
		t.Fatalf("bindings = %#v", bindings)
	}
	for _, binding := range bindings {
		if binding.ContractVersion != routes.CoreGuardEvaluatorContractV1 || !strings.HasPrefix(binding.EvaluatorID, "core.guard.") {
			t.Fatalf("binding = %#v", binding)
		}
	}
}

func TestProductionExtensionsMutationGuardPartitionsCatalogByProvablePolicy(t *testing.T) {
	type expectedRoute struct {
		method      string
		permissions []string
		body        string
		supported   bool
	}
	plugin := []string{identity.PermissionExtensionPluginManage, identity.PermissionExtensionManage}
	theme := []string{identity.PermissionExtensionThemeManage, identity.PermissionExtensionManage}
	superAdmin := []string{"*"}
	expected := map[string]expectedRoute{
		"core.route.extensions.activate":                         {method: "POST", supported: true, permissions: theme},
		"core.route.extensions.disable":                          {method: "POST", supported: true, permissions: plugin},
		"core.route.extensions.recover_lifecycle_operation":      {method: "POST", supported: true, permissions: plugin, body: `{"decision":"retry"}`},
		"core.route.extensions.rollback":                         {method: "POST", supported: true, permissions: plugin},
		"core.route.extensions.revoke_executable_trust":          {method: "DELETE", supported: true, permissions: superAdmin},
		"core.route.extensions.issue_executable_trust_challenge": {method: "POST", supported: true, permissions: superAdmin},
		"core.route.extensions.select_route_provider":            {method: "POST", supported: true, permissions: superAdmin},
		"core.route.extensions.reset_route_provider":             {method: "POST", supported: true, permissions: superAdmin},

		"core.route.extensions.install":                 {method: "POST"},
		"core.route.extensions.uninstall":               {method: "DELETE"},
		"core.route.extensions.enable":                  {method: "POST"},
		"core.route.extensions.apply_migrations":        {method: "POST"},
		"core.route.extensions.update_settings":         {method: "PUT"},
		"core.route.extensions.execute_settings_action": {method: "POST"},
		"core.route.extensions.reset_settings":          {method: "POST"},
		"core.route.extensions.upgrade":                 {method: "POST"},
		"core.route.extensions.verify":                  {method: "POST"},
	}
	var catalog []routes.CoreRoute
	for _, route := range routes.CoreRouteCatalog() {
		if route.Guard.EvaluatorID == "core.guard.extensions.mutation" {
			catalog = append(catalog, route)
		}
	}
	if len(catalog) != len(expected) {
		t.Fatalf("extensions mutation contextual catalog = %#v", catalog)
	}

	authorizer := NewProductionRouteGuardAuthorizer()
	for _, route := range catalog {
		want, exists := expected[route.ID]
		if !exists || route.Method != want.method || route.Guard.Kind != routes.CoreGuardContextual ||
			len(route.Guard.Permissions) != 0 {
			t.Fatalf("unexpected extensions mutation guard route = %#v", route)
		}
		delete(expected, route.ID)

		plan, step := productionCatalogInheritedGuardPlan(t, route)
		if !want.supported {
			request := productionGuardRequest("*")
			request.Method, request.Path = plan.Method(), plan.Path()
			if err := authorizer.Authorize(context.Background(), plan, step, request); !errors.Is(err, ErrRouteGuardUnavailable) {
				t.Fatalf("%s artifact-dependent policy error = %v", route.ID, err)
			}
			continue
		}

		for _, permission := range want.permissions {
			allowed := productionGuardRequest(permission)
			allowed.Method, allowed.Path, allowed.Body = plan.Method(), plan.Path(), []byte(want.body)
			if err := authorizer.Authorize(context.Background(), plan, step, allowed); err != nil {
				t.Fatalf("%s permission %s error = %v", route.ID, permission, err)
			}
		}

		denied := productionGuardRequest(identity.PermissionPostCreate)
		denied.Method, denied.Path, denied.Body = plan.Method(), plan.Path(), []byte(want.body)
		if err := authorizer.Authorize(context.Background(), plan, step, denied); !errors.Is(err, ErrRoutePermissionDenied) {
			t.Fatalf("%s permission denied error = %v", route.ID, err)
		}

		anonymous := routes.DispatchRequest{Method: plan.Method(), Path: plan.Path(), Body: []byte(want.body)}
		if err := authorizer.Authorize(context.Background(), plan, step, anonymous); !errors.Is(err, ErrRouteLoginRequired) {
			t.Fatalf("%s anonymous error = %v", route.ID, err)
		}

		allowed := productionGuardRequest(want.permissions[0])
		allowed.Method, allowed.Path, allowed.Body = plan.Method(), plan.Path(), []byte(want.body)
		forgedStep := step
		forgedStep.RouteID += ".forged"
		if err := authorizer.Authorize(context.Background(), plan, forgedStep, allowed); !errors.Is(err, ErrRouteGuardUnavailable) {
			t.Fatalf("%s forged step error = %v", route.ID, err)
		}
	}
	if len(expected) != 0 {
		t.Fatalf("missing extensions mutation guard routes = %#v", expected)
	}
}

func TestProductionExtensionsMutationGuardProtectsForcedRecovery(t *testing.T) {
	var target routes.CoreRoute
	for _, route := range routes.CoreRouteCatalog() {
		if route.ID == "core.route.extensions.recover_lifecycle_operation" {
			target = route
			break
		}
	}
	if target.ID == "" {
		t.Fatal("lifecycle recovery route is missing")
	}
	plan, step := productionCatalogInheritedGuardPlan(t, target)
	authorizer := NewProductionRouteGuardAuthorizer()
	tests := []struct {
		name        string
		body        string
		permissions []string
		want        error
	}{
		{name: "ordinary recovery", body: `{"decision":"retry"}`, permissions: []string{identity.PermissionExtensionPluginManage}},
		{name: "forced denied", body: `{"decision":"retry","escalateForced":true}`, permissions: []string{identity.PermissionExtensionPluginManage}, want: ErrRoutePermissionDenied},
		{name: "forced super admin", body: `{"decision":"retry","escalateForced":true}`, permissions: []string{"*"}},
		{name: "unknown field closes", body: `{"decision":"retry","future":true}`, permissions: []string{identity.PermissionExtensionPluginManage}, want: ErrRouteGuardUnavailable},
		{name: "malformed closes", body: `{"decision":`, permissions: []string{identity.PermissionExtensionPluginManage}, want: ErrRouteGuardUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := productionGuardRequest(test.permissions...)
			request.Method, request.Path, request.Body = plan.Method(), plan.Path(), []byte(test.body)
			err := authorizer.Authorize(context.Background(), plan, step, request)
			if !errors.Is(err, test.want) || test.want == nil && err != nil {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestProductionExtensionsMutationGuardRejectsForeignRouteID(t *testing.T) {
	descriptor := routes.CoreGuardDescriptor{
		Kind: routes.CoreGuardContextual, EvaluatorID: "core.guard.extensions.mutation",
	}
	plan, step := productionInheritedGuardPlan(t, "core.route.extensions.mutation.foreign", descriptor)
	request := productionGuardRequest("*")
	request.Method, request.Path = plan.Method(), plan.Path()
	if err := NewProductionRouteGuardAuthorizer().Authorize(context.Background(), plan, step, request); !errors.Is(err, ErrRouteGuardUnavailable) {
		t.Fatalf("foreign extensions mutation route error = %v", err)
	}
}

func TestProductionExtensionsReadGuardPartitionsCatalogByProvablePolicy(t *testing.T) {
	type expectedRoute struct {
		method      string
		permissions []string
		supported   bool
	}
	viewer := []string{identity.PermissionExtensionView, identity.PermissionExtensionManage}
	migrations := []string{
		identity.PermissionExtensionView,
		identity.PermissionExtensionPluginManage,
		identity.PermissionExtensionManage,
	}
	trust := []string{
		identity.PermissionExtensionView,
		identity.PermissionExtensionPluginManage,
		identity.PermissionExtensionThemeManage,
		identity.PermissionExtensionManage,
	}
	expected := map[string]expectedRoute{
		"core.route.extensions.list":                     {method: "GET", supported: true, permissions: viewer},
		"core.route.extensions.events":                   {method: "GET", supported: true, permissions: viewer},
		"core.route.extensions.lifecycle_operations":     {method: "GET", supported: true, permissions: viewer},
		"core.route.extensions.lifecycle_operation":      {method: "GET", supported: true, permissions: viewer},
		"core.route.extensions.list_migrations":          {method: "GET", supported: true, permissions: migrations},
		"core.route.extensions.executable_trust_status":  {method: "GET", supported: true, permissions: trust},
		"core.route.extensions.contribution_points":      {method: "GET", supported: true, permissions: viewer},
		"core.route.extensions.contributions":            {method: "GET", supported: true, permissions: viewer},
		"core.route.extensions.event_definitions":        {method: "GET", supported: true, permissions: viewer},
		"core.route.extensions.event_deliveries":         {method: "GET", supported: true, permissions: viewer},
		"core.route.extensions.navigation":               {method: "GET", supported: true, permissions: viewer},
		"core.route.extensions.inspect_route":            {method: "GET", supported: true, permissions: viewer},
		"core.route.extensions.route_provider_conflicts": {method: "GET", supported: true, permissions: viewer},
		"core.route.extensions.route_provider_events":    {method: "GET", supported: true, permissions: viewer},
		"core.route.extensions.route_provider_selection": {method: "GET", supported: true, permissions: viewer},

		"core.route.extensions.frontend_status": {method: "GET"},
		"core.route.extensions.frontend_asset":  {method: "GET"},
		"core.route.extensions.settings":        {method: "GET"},
	}
	var catalog []routes.CoreRoute
	for _, route := range routes.CoreRouteCatalog() {
		if route.Guard.EvaluatorID == "core.guard.extensions.read" {
			catalog = append(catalog, route)
		}
	}
	if len(catalog) != len(expected) {
		t.Fatalf("extensions read contextual catalog = %#v", catalog)
	}

	authorizer := NewProductionRouteGuardAuthorizer()
	for _, route := range catalog {
		want, exists := expected[route.ID]
		if !exists || route.Method != want.method || route.Guard.Kind != routes.CoreGuardContextual ||
			len(route.Guard.Permissions) != 0 {
			t.Fatalf("unexpected extensions read guard route = %#v", route)
		}
		delete(expected, route.ID)

		plan, step := productionCatalogInheritedGuardPlan(t, route)
		if !want.supported {
			request := productionGuardRequest("*")
			request.Method, request.Path = plan.Method(), plan.Path()
			if err := authorizer.Authorize(context.Background(), plan, step, request); !errors.Is(err, ErrRouteGuardUnavailable) {
				t.Fatalf("%s target-dependent policy error = %v", route.ID, err)
			}
			continue
		}

		for _, permission := range want.permissions {
			allowed := productionGuardRequest(permission)
			allowed.Method, allowed.Path = plan.Method(), plan.Path()
			if err := authorizer.Authorize(context.Background(), plan, step, allowed); err != nil {
				t.Fatalf("%s permission %s error = %v", route.ID, permission, err)
			}
		}

		denied := productionGuardRequest(identity.PermissionPostCreate)
		denied.Method, denied.Path = plan.Method(), plan.Path()
		if err := authorizer.Authorize(context.Background(), plan, step, denied); !errors.Is(err, ErrRoutePermissionDenied) {
			t.Fatalf("%s permission denied error = %v", route.ID, err)
		}

		anonymous := routes.DispatchRequest{Method: plan.Method(), Path: plan.Path()}
		if err := authorizer.Authorize(context.Background(), plan, step, anonymous); !errors.Is(err, ErrRouteLoginRequired) {
			t.Fatalf("%s anonymous error = %v", route.ID, err)
		}

		allowed := productionGuardRequest(want.permissions[0])
		allowed.Method, allowed.Path = plan.Method(), plan.Path()
		forgedStep := step
		forgedStep.RouteID += ".forged"
		if err := authorizer.Authorize(context.Background(), plan, forgedStep, allowed); !errors.Is(err, ErrRouteGuardUnavailable) {
			t.Fatalf("%s forged step error = %v", route.ID, err)
		}
		forgedRequest := allowed
		forgedRequest.Path += "/forged"
		if err := authorizer.Authorize(context.Background(), plan, step, forgedRequest); !errors.Is(err, ErrRouteGuardUnavailable) {
			t.Fatalf("%s forged request error = %v", route.ID, err)
		}
	}
	if len(expected) != 0 {
		t.Fatalf("missing extensions read guard routes = %#v", expected)
	}
}

func TestProductionExtensionsReadGuardRejectsForeignRouteID(t *testing.T) {
	descriptor := routes.CoreGuardDescriptor{
		Kind: routes.CoreGuardContextual, EvaluatorID: "core.guard.extensions.read",
	}
	plan, step := productionInheritedGuardPlan(t, "core.route.extensions.read.foreign", descriptor)
	request := productionGuardRequest("*")
	request.Method, request.Path = plan.Method(), plan.Path()
	if err := NewProductionRouteGuardAuthorizer().Authorize(context.Background(), plan, step, request); !errors.Is(err, ErrRouteGuardUnavailable) {
		t.Fatalf("foreign extensions read route error = %v", err)
	}
}

func TestProductionIdentityAdminGuardPartitionsCatalogByProvablePolicy(t *testing.T) {
	type expectedRoute struct {
		method      string
		permissions []string
		supported   bool
	}
	expected := map[string]expectedRoute{
		"core.route.identity.list_permissions": {method: "GET", supported: true, permissions: []string{
			identity.PermissionRoleManage, identity.PermissionUserManage,
			identity.PermissionUserView, identity.PermissionUserPermissionOverride,
		}},
		"core.route.identity.permission_matrix": {method: "GET", supported: true, permissions: []string{
			identity.PermissionRoleManage, identity.PermissionUserManage,
			identity.PermissionUserView, identity.PermissionUserPermissionOverride,
		}},
		"core.route.identity.list_roles":  {method: "GET", supported: true, permissions: []string{identity.PermissionRoleManage}},
		"core.route.identity.create_role": {method: "POST", supported: true, permissions: []string{identity.PermissionRoleManage}},
		"core.route.identity.update_role": {method: "PATCH", supported: true, permissions: []string{identity.PermissionRoleManage}},
		"core.route.identity.list_users":  {method: "GET", supported: true, permissions: []string{identity.PermissionUserView, identity.PermissionUserManage}},
		"core.route.identity.get_user":    {method: "GET", supported: true, permissions: []string{identity.PermissionUserView, identity.PermissionUserManage}},

		"core.route.identity.delete_role":                       {method: "DELETE"},
		"core.route.identity.replace_role_permissions":          {method: "PUT"},
		"core.route.identity.update_user":                       {method: "PATCH"},
		"core.route.identity.admin_clear_user_client_ips":       {method: "POST"},
		"core.route.identity.replace_user_permission_overrides": {method: "PUT"},
		"core.route.identity.replace_user_roles":                {method: "PUT"},
		"core.route.identity.admin_revoke_user_sessions":        {method: "POST"},
	}
	var catalog []routes.CoreRoute
	for _, route := range routes.CoreRouteCatalog() {
		if route.Guard.EvaluatorID == "core.guard.identity.admin" {
			catalog = append(catalog, route)
		}
	}
	if len(catalog) != len(expected) {
		t.Fatalf("identity admin contextual catalog = %#v", catalog)
	}

	authorizer := NewProductionRouteGuardAuthorizer()
	for _, route := range catalog {
		want, exists := expected[route.ID]
		if !exists || route.Method != want.method || route.Guard.Kind != routes.CoreGuardContextual ||
			len(route.Guard.Permissions) != 0 {
			t.Fatalf("unexpected identity admin guard route = %#v", route)
		}
		delete(expected, route.ID)

		plan, step := productionCatalogInheritedGuardPlan(t, route)
		if !want.supported {
			request := productionGuardRequest("*")
			request.Method, request.Path = plan.Method(), plan.Path()
			if err := authorizer.Authorize(context.Background(), plan, step, request); !errors.Is(err, ErrRouteGuardUnavailable) {
				t.Fatalf("%s unprovable policy error = %v", route.ID, err)
			}
			continue
		}

		for _, permission := range want.permissions {
			allowed := productionGuardRequest(permission)
			allowed.Method, allowed.Path = plan.Method(), plan.Path()
			if err := authorizer.Authorize(context.Background(), plan, step, allowed); err != nil {
				t.Fatalf("%s permission %s error = %v", route.ID, permission, err)
			}
		}

		denied := productionGuardRequest(identity.PermissionPostCreate)
		denied.Method, denied.Path = plan.Method(), plan.Path()
		if err := authorizer.Authorize(context.Background(), plan, step, denied); !errors.Is(err, ErrRoutePermissionDenied) {
			t.Fatalf("%s permission denied error = %v", route.ID, err)
		}

		anonymous := routes.DispatchRequest{Method: plan.Method(), Path: plan.Path()}
		if err := authorizer.Authorize(context.Background(), plan, step, anonymous); !errors.Is(err, ErrRouteLoginRequired) {
			t.Fatalf("%s anonymous error = %v", route.ID, err)
		}

		allowed := productionGuardRequest(want.permissions[0])
		allowed.Method, allowed.Path = plan.Method(), plan.Path()
		forgedStep := step
		forgedStep.RouteID += ".forged"
		if err := authorizer.Authorize(context.Background(), plan, forgedStep, allowed); !errors.Is(err, ErrRouteGuardUnavailable) {
			t.Fatalf("%s forged step error = %v", route.ID, err)
		}
		forgedRequest := allowed
		forgedRequest.Path += "/forged"
		if err := authorizer.Authorize(context.Background(), plan, step, forgedRequest); !errors.Is(err, ErrRouteGuardUnavailable) {
			t.Fatalf("%s forged request error = %v", route.ID, err)
		}
	}
	if len(expected) != 0 {
		t.Fatalf("missing identity admin guard routes = %#v", expected)
	}
}

func TestProductionIdentityAdminGuardRejectsForeignRouteID(t *testing.T) {
	descriptor := routes.CoreGuardDescriptor{
		Kind: routes.CoreGuardContextual, EvaluatorID: "core.guard.identity.admin",
	}
	plan, step := productionInheritedGuardPlan(t, "core.route.identity.admin.foreign", descriptor)
	request := productionGuardRequest("*")
	request.Method, request.Path = plan.Method(), plan.Path()
	if err := NewProductionRouteGuardAuthorizer().Authorize(context.Background(), plan, step, request); !errors.Is(err, ErrRouteGuardUnavailable) {
		t.Fatalf("foreign identity admin route error = %v", err)
	}
}

func TestProductionForumSettingsGuardClosesCatalogModule(t *testing.T) {
	expected := map[string]string{
		"core.route.forum.admin_settings":        "GET",
		"core.route.forum.admin_update_settings": "PUT",
		"core.route.forum.admin_reset_settings":  "POST",
	}
	var catalog []routes.CoreRoute
	for _, route := range routes.CoreRouteCatalog() {
		if route.Guard.EvaluatorID == "core.guard.forum.settings" {
			catalog = append(catalog, route)
		}
	}
	if len(catalog) != len(expected) {
		t.Fatalf("forum settings contextual catalog = %#v", catalog)
	}

	authorizer := NewProductionRouteGuardAuthorizer()
	for _, route := range catalog {
		method, exists := expected[route.ID]
		if !exists || route.Method != method || route.Guard.Kind != routes.CoreGuardContextual ||
			!sameStringSet(route.Guard.Permissions, []string{
				identity.PermissionCategoryManage, identity.PermissionTagManage, identity.PermissionForumSettingsManage,
			}) {
			t.Fatalf("unexpected forum settings guard route = %#v", route)
		}
		delete(expected, route.ID)

		plan, step := productionCatalogInheritedGuardPlan(t, route)
		allowed := productionGuardRequest(identity.PermissionCategoryManage)
		allowed.Method, allowed.Path = plan.Method(), plan.Path()
		if route.ID == "core.route.forum.admin_update_settings" {
			allowed.Body = []byte(`{"defaultCategorySlug":"general"}`)
		}
		if err := authorizer.Authorize(context.Background(), plan, step, allowed); err != nil {
			t.Fatalf("%s allowed error = %v", route.ID, err)
		}

		denied := productionGuardRequest()
		denied.Method, denied.Path, denied.Body = plan.Method(), plan.Path(), allowed.Body
		if err := authorizer.Authorize(context.Background(), plan, step, denied); !errors.Is(err, ErrRoutePermissionDenied) {
			t.Fatalf("%s permission denied error = %v", route.ID, err)
		}

		forgedStep := step
		forgedStep.RouteID += ".forged"
		if err := authorizer.Authorize(context.Background(), plan, forgedStep, allowed); !errors.Is(err, ErrRouteGuardUnavailable) {
			t.Fatalf("%s forged step error = %v", route.ID, err)
		}
	}
	if len(expected) != 0 {
		t.Fatalf("missing forum settings guard routes = %#v", expected)
	}
}

func TestProductionForumSettingsGuardEnforcesFieldOwners(t *testing.T) {
	var target routes.CoreRoute
	for _, route := range routes.CoreRouteCatalog() {
		if route.ID == "core.route.forum.admin_update_settings" {
			target = route
			break
		}
	}
	if target.ID == "" {
		t.Fatal("forum settings update route is missing")
	}
	plan, step := productionCatalogInheritedGuardPlan(t, target)
	authorizer := NewProductionRouteGuardAuthorizer()
	tests := []struct {
		name        string
		body        string
		permissions []string
		want        error
	}{
		{name: "category", body: `{"defaultCategorySlug":"general"}`, permissions: []string{identity.PermissionCategoryManage}},
		{name: "category denied", body: `{"defaultCategorySlug":"general"}`, permissions: []string{identity.PermissionTagManage}, want: ErrRoutePermissionDenied},
		{name: "tag", body: `{"tagCreationMode":"review","tagMaxPerTopic":5}`, permissions: []string{identity.PermissionTagManage}},
		{name: "tag denied", body: `{"tagPublicPages":true}`, permissions: []string{identity.PermissionCategoryManage}, want: ErrRoutePermissionDenied},
		{name: "runtime", body: `{"topicsPerPage":30}`, permissions: []string{identity.PermissionForumSettingsManage}},
		{name: "community runtime", body: `{"guestRead":"login_required","mentionsEnabled":false}`, permissions: []string{identity.PermissionForumSettingsManage}},
		{name: "runtime denied", body: `{"listDefaultSort":"hot"}`, permissions: []string{identity.PermissionTagManage}, want: ErrRoutePermissionDenied},
		{name: "mixed", body: `{"defaultCategorySlug":"general","tagMinPerTopic":1}`, permissions: []string{identity.PermissionCategoryManage, identity.PermissionTagManage}},
		{name: "mixed body escalation", body: `{"defaultCategorySlug":"general","tagMinPerTopic":1}`, permissions: []string{identity.PermissionCategoryManage}, want: ErrRoutePermissionDenied},
		{name: "empty requires module authority", body: `{}`, permissions: []string{identity.PermissionTagManage}},
		{name: "unknown field closes", body: `{"futureSetting":true}`, permissions: []string{identity.PermissionForumSettingsManage}, want: ErrRouteGuardUnavailable},
		{name: "malformed closes", body: `{"topicsPerPage":`, permissions: []string{identity.PermissionForumSettingsManage}, want: ErrRouteGuardUnavailable},
		{name: "trailing value closes", body: `{} {}`, permissions: []string{identity.PermissionForumSettingsManage}, want: ErrRouteGuardUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := productionGuardRequest(test.permissions...)
			request.Method, request.Path, request.Body = plan.Method(), plan.Path(), []byte(test.body)
			err := authorizer.Authorize(context.Background(), plan, step, request)
			if !errors.Is(err, test.want) || test.want == nil && err != nil {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestProductionForumSettingsGuardRejectsForeignRouteID(t *testing.T) {
	descriptor := routes.CoreGuardDescriptor{
		Kind: routes.CoreGuardContextual,
		Permissions: []string{
			identity.PermissionCategoryManage, identity.PermissionTagManage, identity.PermissionForumSettingsManage,
		},
		EvaluatorID: "core.guard.forum.settings",
	}
	plan, step := productionInheritedGuardPlan(t, "core.route.forum.settings.foreign", descriptor)
	request := productionGuardRequest(identity.PermissionForumSettingsManage)
	request.Method, request.Path = plan.Method(), plan.Path()
	if err := NewProductionRouteGuardAuthorizer().Authorize(context.Background(), plan, step, request); !errors.Is(err, ErrRouteGuardUnavailable) {
		t.Fatalf("foreign forum settings route error = %v", err)
	}
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	set := make(map[string]struct{}, len(left))
	for _, value := range left {
		set[value] = struct{}{}
	}
	for _, value := range right {
		if _, exists := set[value]; !exists {
			return false
		}
	}
	return true
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
