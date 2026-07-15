package extensionopenapi

import (
	"strings"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

// HostRoutePolicies materializes only policy facts enforced or deliberately
// disabled by the Host. Plugin OpenAPI prose must match this exact snapshot.
func HostRoutePolicies(manifest extensionmanifest.Manifest) []RoutePolicy {
	if len(manifest.OpenAPI) == 0 {
		return nil
	}
	result := make([]RoutePolicy, 0, len(manifest.Routes))
	for _, route := range manifest.Routes {
		if !openAPIAddressableRouteAction(route.Action) {
			continue
		}
		for _, method := range route.Methods {
			result = append(result, RoutePolicy{
				RouteID: route.ID, Method: strings.ToUpper(method),
				RateLimit: PolicyDisabled, Idempotency: PolicyDisabled,
				Security: securityForDeclaredGuard(route.Guard),
			})
		}
	}
	return result
}

// CoreOperations converts the runtime catalog into aggregate collision
// reservations without publishing Host operations in the plugin document.
func CoreOperations(catalog []routes.CoreRoute) []CoreOperation {
	result := make([]CoreOperation, 0, len(catalog))
	for _, route := range catalog {
		methods := []string{route.Method}
		if route.Method == "*" {
			// OpenAPI has no wildcard method. Reserve every representable method so
			// a plugin fragment cannot overlap the legacy catch-all proxy route.
			methods = openAPIMethodOrder
		}
		for _, method := range methods {
			operationID := route.ID
			if route.Method == "*" {
				operationID += "." + strings.ToLower(method)
			}
			result = append(result, CoreOperation{
				RouteID: route.ID, Path: route.Path, Method: method, OperationID: operationID,
			})
		}
	}
	return result
}

func openAPIAddressableRouteAction(action string) bool {
	return action == extensionmanifest.RouteActionAdd || action == extensionmanifest.RouteActionAlias ||
		action == extensionmanifest.RouteActionRedirect || action == extensionmanifest.RouteActionRewrite ||
		action == extensionmanifest.RouteActionReplace
}

func securityForDeclaredGuard(guard string) string {
	switch guard {
	case extensionmanifest.GuardCorePublic, extensionmanifest.GuardCoreGuest:
		return SecurityPublic
	case extensionmanifest.GuardCoreLogin, extensionmanifest.GuardCorePermission:
		return SecurityAuthenticated
	case extensionmanifest.GuardCoreInherit:
		return SecurityHostInherited
	default:
		return SecurityPluginOwned
	}
}
