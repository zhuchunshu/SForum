package http

import (
	"fmt"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

func TestRouteDispatcherInternallyExecutesAliasAndRewriteTargets(t *testing.T) {
	registry := routes.NewRegistry()
	artifact := routes.PluginArtifact{
		ExtensionID: "dispatch.mapping", ExtensionVersion: "1.0.0",
		PackageDigest: strings.Repeat("a", 64), RuntimeInstanceID: "runtime-a",
	}
	aliasTarget := routeMappingCoreRoute("core.route.mapping.alias", stdhttp.MethodGet, "/api/v1/core/:topicID")
	rewriteTarget := routeMappingCoreRoute("core.route.mapping.rewrite", stdhttp.MethodPost, "/api/v1/core/:topicID/rewrite")
	alias := routeMappingPluginRoute("dispatch.mapping.alias", extensionmanifest.RouteActionAlias, aliasTarget.ID, "/legacy/:id", stdhttp.MethodGet)
	rewrite := routeMappingPluginRoute("dispatch.mapping.rewrite", extensionmanifest.RouteActionRewrite, rewriteTarget.ID, "/internal/:id", stdhttp.MethodPost)
	if _, err := registry.Publish(routes.Publication{
		Core:    []routes.CoreRoute{aliasTarget, rewriteTarget},
		Plugins: []routes.PluginRouteSet{{Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{alias, rewrite}}},
	}); err != nil {
		t.Fatal(err)
	}
	trace := routes.NewRouteTraceRing(8)
	dispatcher := routes.NewDispatcher(routes.DispatcherConfig{
		Plans: routeRegistryPlanResolver{registry: registry}, Guard: NewProductionRouteGuardAuthorizer(), Trace: trace,
	})
	app := fiber.New()
	middlewareCalls := 0
	app.Use(func(c fiber.Ctx) error {
		middlewareCalls++
		return c.Next()
	})
	app.Use(routeDispatcherMiddleware(dispatcher, nil))
	app.Get(aliasTarget.Path, func(c fiber.Ctx) error {
		return c.SendString("alias:" + c.Params("topicID") + ":" + c.Query("page"))
	})
	app.Post(rewriteTarget.Path, func(c fiber.Ctx) error {
		return c.SendString("rewrite:" + c.Params("topicID") + ":" + string(c.Body()))
	})

	response, err := app.Test(httptest.NewRequest(stdhttp.MethodGet, "/legacy/41?page=2", nil))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != stdhttp.StatusOK || string(body) != "alias:41:2" {
		t.Fatalf("alias status = %d, body = %q", response.StatusCode, body)
	}

	request := httptest.NewRequest(stdhttp.MethodPost, "/internal/42", strings.NewReader("payload"))
	response, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != stdhttp.StatusOK || string(body) != "rewrite:42:payload" {
		t.Fatalf("rewrite status = %d, body = %q", response.StatusCode, body)
	}
	if middlewareCalls != 2 {
		t.Fatalf("host middleware replayed %d times", middlewareCalls)
	}

	records := trace.RouteTraces(0)
	if len(records) != 4 {
		t.Fatalf("route traces = %#v", records)
	}
	for index, record := range records {
		want := routes.RouteTraceSucceeded
		if index%2 == 1 {
			want = routes.RouteTraceCommitted
		}
		if record.Outcome != want || record.Provider.Artifact.ExtensionID != artifact.ExtensionID {
			t.Fatalf("trace %d = %#v", index, record)
		}
	}
}

func TestRouteDispatcherFailsClosedForUnmappableAliasTarget(t *testing.T) {
	registry := routes.NewRegistry()
	artifact := routes.PluginArtifact{
		ExtensionID: "dispatch.closed", ExtensionVersion: "1.0.0",
		PackageDigest: strings.Repeat("b", 64), RuntimeInstanceID: "runtime-b",
	}
	target := routeMappingCoreRoute("core.route.mapping.closed", stdhttp.MethodGet, "/core/:id")
	alias := routeMappingPluginRoute("dispatch.closed.alias", extensionmanifest.RouteActionAlias, target.ID, "/legacy", stdhttp.MethodGet)
	if _, err := registry.Publish(routes.Publication{
		Core:    []routes.CoreRoute{target},
		Plugins: []routes.PluginRouteSet{{Artifact: artifact, Routes: []extensionmanifest.ManifestRoute{alias}}},
	}); err != nil {
		t.Fatal(err)
	}
	dispatcher := routes.NewDispatcher(routes.DispatcherConfig{
		Plans: routeRegistryPlanResolver{registry: registry}, Guard: NewProductionRouteGuardAuthorizer(),
	})
	app := fiber.New()
	app.Use(routeDispatcherMiddleware(dispatcher, nil))
	coreCalls := 0
	app.Get(target.Path, func(c fiber.Ctx) error {
		coreCalls++
		return c.SendStatus(stdhttp.StatusNoContent)
	})
	response, err := app.Test(httptest.NewRequest(stdhttp.MethodGet, "/legacy", nil))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != stdhttp.StatusInternalServerError || coreCalls != 0 {
		t.Fatalf("status = %d, core calls = %d", response.StatusCode, coreCalls)
	}
}

func routeMappingCoreRoute(id, method, path string) routes.CoreRoute {
	contract := "sforum." + strings.TrimPrefix(id, "core.") + "@1"
	return routes.CoreRoute{
		ID: id, ContractVersion: contract, Method: method, Path: path,
		Guard: routes.CoreGuardDescriptor{
			RouteID: id, ContractVersion: contract, Method: method, Kind: routes.CoreGuardPublic,
		},
	}
}

func routeMappingPluginRoute(id, action, targetID, path, method string) extensionmanifest.ManifestRoute {
	return extensionmanifest.ManifestRoute{
		ID: id, ContractVersion: fmt.Sprintf("%s@1", id), Action: action, TargetID: targetID,
		Path: path, Methods: []string{method}, Guard: extensionmanifest.GuardCoreInherit,
		Fallback: "closed", Mode: extensionmanifest.RouteModeHTTP,
	}
}
