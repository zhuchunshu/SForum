package extensionmanifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestManifestV3AuthoritativeFixtures(t *testing.T) {
	paths, err := filepath.Glob("testdata/v3/*.json")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"admin-plugin.json", "dependency-base.json", "dependency-consumer.json",
		"dependency-provider.json", "full-route-plugin.json", "identity-plugin.json",
		"l2-plugin.json", "media-plugin.json", "minimal-theme.json",
		"navigation-plugin.json", "query-plugin.json", "raw-db-plugin.json",
		"safe-plugin.json", "trusted-application-plugin.json",
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
	for _, mode := range []string{RouteModeHTTP, RouteModeSSE, RouteModeWebSocket, RouteModeStream, RouteModeMultipart} {
		if !modes[mode] {
			t.Fatalf("full route fixture missing mode %s", mode)
		}
	}
}
