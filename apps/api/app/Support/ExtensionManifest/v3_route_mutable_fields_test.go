package extensionmanifest

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestManifestV3RouteMutableFieldsAllowedActions(t *testing.T) {
	tests := []struct {
		action   string
		request  []string
		response []string
	}{
		{RouteActionGlobalMiddleware, []string{"/query", "/headers/x~1trace"}, nil},
		{RouteActionBefore, []string{"/body/title"}, nil},
		{RouteActionFilter, []string{"/body", "/params/~0debug"}, []string{"/status", "/headers/cache~1control"}},
		{RouteActionWrap, []string{"/"}, []string{"/body/items/0"}},
		{RouteActionAfter, nil, []string{"/body/summary"}},
	}
	for _, test := range tests {
		t.Run(test.action, func(t *testing.T) {
			manifest := mutableRouteManifest(test.action)
			manifest.Routes[0].MutableRequestFields = test.request
			manifest.Routes[0].MutableResponseFields = test.response
			if err := Validate(manifest); err != nil {
				t.Fatalf("allowed mutable fields rejected: %v", err)
			}
			body, err := json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidateV3JSONSchema(body); err != nil {
				t.Fatalf("allowed mutable fields rejected by JSON Schema: %v", err)
			}
		})
	}
}

func TestManifestV3RouteMutableFieldsNormalizeBeforeDuplicateCheck(t *testing.T) {
	manifest := mutableRouteManifest(RouteActionFilter)
	manifest.Routes[0].MutableRequestFields = []string{" /body/title ", "\t/headers/x~1trace\n"}
	manifest.Routes[0].MutableResponseFields = []string{" /status "}
	normalized := Normalize(manifest)
	if !reflect.DeepEqual(normalized.Routes[0].MutableRequestFields, []string{"/body/title", "/headers/x~1trace"}) ||
		!reflect.DeepEqual(normalized.Routes[0].MutableResponseFields, []string{"/status"}) {
		t.Fatalf("normalized mutable fields = %#v / %#v", normalized.Routes[0].MutableRequestFields, normalized.Routes[0].MutableResponseFields)
	}
	if err := Validate(manifest); err != nil {
		t.Fatalf("trimmed mutable fields should validate: %v", err)
	}

	duplicate := mutableRouteManifest(RouteActionFilter)
	duplicate.Routes[0].MutableRequestFields = []string{"/body/title", " /body/title "}
	if err := Validate(duplicate); err == nil {
		t.Fatal("mutable fields that duplicate after trimming must be rejected")
	}
}

func TestManifestV3RouteMutableFieldsRejectRootAndInvalidPointers(t *testing.T) {
	// RFC 6901 用空字符串表示整个文档；字段级最小权限 allowlist 明确禁止该 root pointer。
	for _, pointer := range []string{"", "body", "#/body", "/body/~", "/body/~2", "/body/~~0"} {
		t.Run(pointer, func(t *testing.T) {
			manifest := mutableRouteManifest(RouteActionFilter)
			manifest.Routes[0].MutableRequestFields = []string{pointer}
			if err := Validate(manifest); err == nil {
				t.Fatalf("invalid or forbidden route pointer %q accepted", pointer)
			}
			body, err := json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidateV3JSONSchema(body); err == nil {
				t.Fatalf("invalid or forbidden route pointer %q accepted by JSON Schema", pointer)
			}
		})
	}
}

func TestManifestV3RouteMutableFieldBudgets(t *testing.T) {
	exact := mutableRouteManifest(RouteActionFilter)
	exact.Routes[0].MutableRequestFields = routeMutableFieldPointers("request", RouteMutableFieldsMaximumCount)
	exact.Routes[0].MutableResponseFields = routeMutableFieldPointers("response", RouteMutableFieldsMaximumCount)
	exact.Routes[0].MutableRequestFields[0] = "/" + strings.Repeat("a", RouteMutableFieldMaximumBytes-1)
	exact.Routes[0].MutableResponseFields[0] = strings.Repeat("/token", RouteMutableFieldMaximumTokens)
	assertMutableRouteAccepted(t, exact)

	tests := []struct {
		name   string
		fields []string
	}{
		{"pointer count", routeMutableFieldPointers("over", RouteMutableFieldsMaximumCount+1)},
		{"pointer bytes", []string{"/" + strings.Repeat("a", RouteMutableFieldMaximumBytes)}},
		{"reference tokens", []string{strings.Repeat("/token", RouteMutableFieldMaximumTokens+1)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := mutableRouteManifest(RouteActionFilter)
			manifest.Routes[0].MutableRequestFields = test.fields
			assertMutableRouteRejected(t, manifest)
		})
	}
}

func TestManifestV3RouteMutableFieldBudgetUsesUTF8Bytes(t *testing.T) {
	exact := mutableRouteManifest(RouteActionFilter)
	exact.Routes[0].MutableRequestFields = []string{"/" + strings.Repeat("界", 85)}
	if got := len(exact.Routes[0].MutableRequestFields[0]); got != RouteMutableFieldMaximumBytes {
		t.Fatalf("exact UTF-8 pointer bytes = %d, want %d", got, RouteMutableFieldMaximumBytes)
	}
	assertMutableRouteAccepted(t, exact)

	over := mutableRouteManifest(RouteActionFilter)
	over.Routes[0].MutableRequestFields = []string{"/" + strings.Repeat("界", 86)}
	if got := len([]rune(over.Routes[0].MutableRequestFields[0])); got >= RouteMutableFieldMaximumBytes {
		t.Fatalf("test pointer rune count = %d, want below JSON Schema maxLength", got)
	}
	assertMutableRouteRejected(t, over)
}

func TestLegacyManifestRejectsRouteMutableFields(t *testing.T) {
	for _, version := range []int{0, ManifestVersionV1, ManifestVersionV2} {
		for _, direction := range []string{"request", "response"} {
			t.Run(fmt.Sprintf("version_%d_%s", version, direction), func(t *testing.T) {
				manifest := versionedTestManifest(version)
				manifest.Routes = []ManifestRoute{{
					Path: "/legacy", Methods: []string{"GET"}, Access: RouteAccessPublic,
				}}
				if direction == "request" {
					manifest.Routes[0].MutableRequestFields = []string{"/query"}
				} else {
					manifest.Routes[0].MutableResponseFields = []string{"/status"}
				}
				if err := Validate(manifest); err == nil {
					t.Fatal("legacy manifest accepted a Manifest V3 route mutation declaration")
				}
			})
		}
	}
}

func TestManifestV3RouteMutableFieldsRejectWrongActions(t *testing.T) {
	requestDenied := []string{
		RouteActionAdd, RouteActionAlias, RouteActionRedirect, RouteActionRewrite,
		RouteActionAfter, RouteActionReplace,
	}
	for _, action := range requestDenied {
		t.Run(action+" request", func(t *testing.T) {
			manifest := mutableRouteManifest(action)
			manifest.Routes[0].MutableRequestFields = []string{"/body"}
			assertMutableRouteRejected(t, manifest)
		})
	}

	responseDenied := []string{
		RouteActionAdd, RouteActionAlias, RouteActionRedirect, RouteActionRewrite,
		RouteActionBefore, RouteActionReplace, RouteActionGlobalMiddleware,
	}
	for _, action := range responseDenied {
		t.Run(action+" response", func(t *testing.T) {
			manifest := mutableRouteManifest(action)
			manifest.Routes[0].MutableResponseFields = []string{"/status"}
			assertMutableRouteRejected(t, manifest)
		})
	}
}

func assertMutableRouteRejected(t *testing.T, manifest Manifest) {
	t.Helper()
	if err := Validate(manifest); err == nil {
		t.Fatal("invalid mutable route fields accepted by semantic validation")
	}
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateV3JSONSchema(body); err == nil {
		t.Fatal("invalid mutable route fields accepted by JSON Schema")
	}
}

func assertMutableRouteAccepted(t *testing.T, manifest Manifest) {
	t.Helper()
	if err := Validate(manifest); err != nil {
		t.Fatalf("valid mutable route rejected by semantic validation: %v", err)
	}
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateV3JSONSchema(body); err != nil {
		t.Fatalf("valid mutable route rejected by JSON Schema: %v", err)
	}
}

func routeMutableFieldPointers(prefix string, count int) []string {
	values := make([]string, count)
	for index := range values {
		values[index] = fmt.Sprintf("/%s/%d", prefix, index)
	}
	return values
}

func mutableRouteManifest(action string) Manifest {
	manifest := completeV3Manifest()
	route := &manifest.Routes[0]
	route.Action = action
	route.TargetID = ""
	route.Destination = ""
	route.MutableRequestFields = nil
	route.MutableResponseFields = nil
	switch action {
	case RouteActionAlias, RouteActionRewrite:
		route.TargetID = "core.route.demo.target"
		route.Guard = GuardCoreInherit
		route.Handler = ""
		route.RequestSchema = ""
		route.ResponseSchema = ""
	case RouteActionBefore, RouteActionAfter, RouteActionFilter, RouteActionWrap, RouteActionReplace:
		route.TargetID = "core.route.demo.target"
		route.Guard = GuardCoreInherit
	case RouteActionRedirect:
		route.Destination = "/api/v3/demo/new"
		route.Handler = ""
		route.RequestSchema = ""
		route.ResponseSchema = ""
	case RouteActionGlobalMiddleware:
		route.Path = ""
		route.Methods = nil
	}
	return manifest
}
