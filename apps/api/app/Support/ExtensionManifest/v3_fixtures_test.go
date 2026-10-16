package extensionmanifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// full-route-plugin fixture 的 replace 路由用它声明 raw 请求权威：决定 R1 之后，
// raw 权限必须来自包内 kind=raw_request 守卫，Host 拥有的 core.guard.raw_request 被拒绝。
const fullRouteRawGuardID = "fixture.routes.guard.raw"

func TestManifestV3AuthoritativeFixtures(t *testing.T) {
	paths, err := filepath.Glob("testdata/v3/*.json")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"admin-plugin.json", "ai-tool-plugin.json", "dependency-base.json",
		"dependency-consumer.json", "dependency-provider.json", "full-route-plugin.json",
		"identity-plugin.json", "l2-plugin.json", "media-plugin.json",
		"minimal-theme.json", "navigation-plugin.json", "query-plugin.json",
		"raw-db-plugin.json", "safe-plugin.json", "trusted-application-plugin.json",
	}
	got := make([]string, 0, len(paths))
	manifests := map[string]Manifest{}
	for _, path := range paths {
		got = append(got, filepath.Base(path))
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidateV3JSONSchema(body); err != nil {
			t.Fatalf("%s JSON Schema: %v", path, err)
		}
		var manifest Manifest
		if err := json.Unmarshal(body, &manifest); err != nil {
			t.Fatalf("%s decode: %v", path, err)
		}
		if err := Validate(manifest); err != nil {
			t.Fatalf("%s semantic validation: %v", path, err)
		}
		manifests[filepath.Base(path)] = Normalize(manifest)
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("fixture set = %#v, want %#v", got, want)
	}
	if database := manifests["trusted-application-plugin.json"].Database; database == nil ||
		database.Authority != "" || !reflect.DeepEqual(database.Grants, []string{DatabaseGrantOwnSchema}) {
		t.Fatalf("trusted application fixture does not use exact database grants: %#v", database)
	}
	if database := manifests["raw-db-plugin.json"].Database; database == nil ||
		database.Authority != "" || !HasDatabaseGrant(database, DatabaseGrantRawCore) {
		t.Fatalf("legacy raw database fixture did not normalize cumulatively: %#v", database)
	}

	graph, err := ResolvePackageGraph([]Manifest{
		manifests["dependency-consumer.json"],
		manifests["dependency-provider.json"],
		manifests["dependency-base.json"],
	})
	if err != nil {
		t.Fatalf("dependency fixture graph: %v", err)
	}
	wantOrder := []string{"fixture.dependency-base", "fixture.dependency-provider", "fixture.dependency-consumer"}
	if !reflect.DeepEqual(graph.Order, wantOrder) {
		t.Fatalf("dependency fixture order = %#v, want %#v", graph.Order, wantOrder)
	}

	routes := manifests["full-route-plugin.json"].Routes
	actions := map[string]bool{}
	modes := map[string]bool{}
	for _, route := range routes {
		actions[route.Action] = true
		modes[route.Mode] = true
	}
	for _, action := range []string{RouteActionAdd, RouteActionAlias, RouteActionRedirect, RouteActionRewrite, RouteActionBefore, RouteActionAfter, RouteActionFilter, RouteActionWrap, RouteActionReplace, RouteActionGlobalMiddleware} {
		if !actions[action] {
			t.Fatalf("full route fixture missing action %s", action)
		}
	}
	for _, action := range []string{RouteActionAlias, RouteActionRewrite, RouteActionBefore, RouteActionAfter, RouteActionFilter, RouteActionWrap} {
		for _, route := range routes {
			if route.Action == action && route.Guard != GuardCoreInherit {
				t.Fatalf("target action %s normalized guard = %q", action, route.Guard)
			}
		}
	}
	for _, route := range routes {
		if route.Action == RouteActionReplace && route.Guard != fullRouteRawGuardID {
			t.Fatalf("explicit raw replacement guard = %q", route.Guard)
		}
	}
	// 反向覆盖：同一个 fixture 把 replace 守卫换成 Host 拥有的 core.guard.raw_request
	// 后必须被拒绝，且错误要指向 kind=raw_request 的包内守卫作为替代。
	coreRaw := manifests["full-route-plugin.json"]
	for index := range coreRaw.Routes {
		if coreRaw.Routes[index].Action == RouteActionReplace {
			coreRaw.Routes[index].Guard = GuardCoreRaw
		}
	}
	assertCoreRawRouteGuardRefused(t, coreRaw)
	for _, mode := range []string{RouteModeHTTP, RouteModeSSE, RouteModeWebSocket, RouteModeStream, RouteModeMultipart} {
		if !modes[mode] {
			t.Fatalf("full route fixture missing mode %s", mode)
		}
	}
}
