package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	extensionscontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Extensions"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
)

func TestPublicFrontendRuntimeMatchesExactAcquiredInstance(t *testing.T) {
	target := extensionsruntime.RuntimeInstanceSnapshot{
		Identity:         extensionsruntime.RuntimeInstanceIdentity{ExtensionID: "demo.plugin", InstanceID: "runtime-1"},
		ExtensionVersion: "1.0.0",
		ArtifactDigest:   "package-v1",
	}
	if !publicFrontendRuntimeMatches(nil, target) {
		t.Fatal("ordinary extension route unexpectedly required a public frontend identity")
	}
	exact := &extensionscontroller.PublicFrontendBridgeIdentity{
		ExtensionID: "demo.plugin", ExtensionVersion: "1.0.0", PackageDigest: "package-v1",
	}
	if !publicFrontendRuntimeMatches(exact, target) {
		t.Fatal("exact public frontend bridge did not match its acquired runtime")
	}
	stale := *exact
	stale.PackageDigest = "package-v2"
	if publicFrontendRuntimeMatches(&stale, target) {
		t.Fatal("stale public frontend package reached a different runtime")
	}
}

func TestExtensionRouteGatewayStripsPluginLinkOnProductionAdapter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Add("Link", `<https://evil.example/>; rel="canonical"`)
		writer.Header().Add("Link", `</asset.js>; rel="preload"`)
		writer.Header().Set("X-Plugin-Metadata", "kept")
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	identity := extensionsruntime.RuntimeInstanceIdentity{ExtensionID: "demo.plugin", InstanceID: "runtime-1"}
	gate, err := extensionsruntime.NewRuntimeAdmissionGate(identity)
	if err != nil {
		t.Fatal(err)
	}
	runtime := &providerRouteRuntime{
		gate: gate,
		snapshot: extensionsruntime.RuntimeInstanceSnapshot{
			Identity: identity, Active: true,
			Target: extensionsruntime.RouteTarget{BaseURL: server.URL, InstanceID: identity.InstanceID},
		},
	}
	gateway := extensionRouteGateway{runtime: runtime, gateway: extensionsruntime.NewRouteGateway()}
	app := fiber.New()
	app.Get("/proxy", func(c fiber.Ctx) error {
		return gateway.Proxy(c, extensionscontroller.ProxyInput{Matched: extensions.MatchedRoute{
			Extension: extensions.Extension{ID: identity.ExtensionID}, Path: "/links",
		}})
	})

	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/proxy", nil))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusAccepted || len(response.Header.Values("Link")) != 0 ||
		response.Header.Get("X-Plugin-Metadata") != "kept" || runtime.calls != 1 ||
		runtime.extensionID != identity.ExtensionID || runtime.class != extensionsruntime.RuntimeCallRoute {
		t.Fatalf("status=%d headers=%v runtime=%#v", response.StatusCode, response.Header, runtime)
	}
}

type providerRouteRuntime struct {
	extensions.RuntimeManager
	snapshot    extensionsruntime.RuntimeInstanceSnapshot
	gate        *extensionsruntime.RuntimeAdmissionGate
	calls       int
	extensionID string
	class       extensionsruntime.RuntimeCallClass
}

func (r *providerRouteRuntime) RouteTarget(extensionID string) (extensionsruntime.RouteTarget, bool) {
	return r.snapshot.Target, extensionID == r.snapshot.Identity.ExtensionID
}

func (r *providerRouteRuntime) AcquireActiveRuntimeCall(
	ctx context.Context,
	extensionID string,
	class extensionsruntime.RuntimeCallClass,
) (extensionsruntime.RuntimeInstanceSnapshot, *extensionsruntime.RuntimeAdmissionLease, error) {
	r.calls++
	r.extensionID = extensionID
	r.class = class
	lease, err := r.gate.Acquire(ctx, class)
	return r.snapshot, lease, err
}
