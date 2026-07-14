package extensionscontroller

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

func TestRouteInspectorHTTPAllowsViewAndLegacyManageWithoutRawRequestData(t *testing.T) {
	app, _ := newRouteInspectorTestApp(t, true)
	viewer := loginRouteInspectorUser(t, app, 1)
	legacyManager := loginRouteInspectorUser(t, app, 2)

	values := url.Values{}
	values.Set("method", "POST")
	values.Set("path", "/controller/topics?token=secret-value")
	endpoint := "/api/v1/admin/extensions/route-inspector?" + values.Encode()
	for name, cookie := range map[string]*http.Cookie{"viewer": viewer, "legacy_manager": legacyManager} {
		t.Run(name, func(t *testing.T) {
			response := performExtensionRequest(t, app, http.MethodGet, endpoint, cookie)
			if response.StatusCode != http.StatusOK {
				t.Fatalf("status=%d body=%s", response.StatusCode, responseBody(t, response))
			}
			defer response.Body.Close()
			var envelope testEnvelope[routes.RouteInspectorSnapshot]
			if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.Data.Method != "POST" || envelope.Data.Resolution != routes.InspectionAmbiguous ||
				envelope.Data.Revision == 0 || len(envelope.Data.Chain) != 0 || len(envelope.Data.Conflicts) != 1 {
				t.Fatalf("inspection snapshot = %#v", envelope.Data)
			}
			encoded, err := json.Marshal(envelope.Data)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(encoded), "secret-value") || strings.Contains(string(encoded), "token=") {
				t.Fatalf("inspector leaked raw request data: %s", encoded)
			}
		})
	}
}

func TestRouteInspectorHTTPRequiresAuthenticationAndExtensionView(t *testing.T) {
	app, _ := newRouteInspectorTestApp(t, true)
	endpoint := "/api/v1/admin/extensions/route-inspector?method=POST&path=%2Fcontroller%2Ftopics"
	unauthenticated := performExtensionRequest(t, app, http.MethodGet, endpoint, nil)
	if unauthenticated.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status=%d", unauthenticated.StatusCode)
	}
	unauthenticated.Body.Close()

	denied := loginRouteInspectorUser(t, app, 3)
	response := performExtensionRequest(t, app, http.MethodGet, endpoint, denied)
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("no-view status=%d body=%s", response.StatusCode, responseBody(t, response))
	}
	response.Body.Close()
}

func TestRouteInspectorHTTPValidatesExplicitMethodAndPathAndFailsClosedWhenUnavailable(t *testing.T) {
	app, _ := newRouteInspectorTestApp(t, true)
	viewer := loginRouteInspectorUser(t, app, 1)
	for _, endpoint := range []string{
		"/api/v1/admin/extensions/route-inspector?path=%2Fcontroller%2Ftopics",
		"/api/v1/admin/extensions/route-inspector?method=POST",
		"/api/v1/admin/extensions/route-inspector?method=%2A&path=%2Fcontroller%2Ftopics",
		"/api/v1/admin/extensions/route-inspector?method=POST&path=relative",
	} {
		response := performExtensionRequest(t, app, http.MethodGet, endpoint, viewer)
		if response.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("%s status=%d body=%s", endpoint, response.StatusCode, responseBody(t, response))
		}
		var envelope testEnvelope[testErrorData]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if envelope.Data.Reason != routeInspectorInvalidReason {
			t.Fatalf("%s reason=%q", endpoint, envelope.Data.Reason)
		}
	}

	unavailableApp, _ := newRouteInspectorTestApp(t, false)
	unavailableViewer := loginRouteInspectorUser(t, unavailableApp, 1)
	response := performExtensionRequest(
		t, unavailableApp, http.MethodGet,
		"/api/v1/admin/extensions/route-inspector?method=POST&path=%2Fcontroller%2Ftopics",
		unavailableViewer,
	)
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("unavailable status=%d body=%s", response.StatusCode, responseBody(t, response))
	}
	response.Body.Close()
}

func newRouteInspectorTestApp(t *testing.T, configured bool) (*fiber.App, *authsession.Manager) {
	t.Helper()
	manager := authsession.NewManager(session.NewStore(), authsession.Config{HashSecret: "test-secret"})
	actors := controllerActors{actors: map[int64]identity.Actor{
		1: {ID: 1, Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionExtensionView: true}},
		2: {ID: 2, Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionExtensionManage: true}},
		3: {ID: 3, Status: identity.UserStatusActive},
	}}
	controller := NewController(nil, actors, manager)
	if configured {
		registry, _, _, _ := routeProviderControllerRegistry(t)
		providers := routes.NewProviderSelectionAPI(registry, &routeProviderControllerStore{})
		controller.WithRouteProviderSelection(providers, &routeProviderControllerAuditor{nextID: 70})
	}
	login := extensionRouteProviderFunc(func(router fiber.Router) {
		router.Post("/route-inspector-login/:id", func(c fiber.Ctx) error {
			id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
			_, err := manager.Start(c, id)
			return err
		})
	})
	app := apphttp.NewApp(config.Config{AppName: "SForum", AppEnv: "test", CSRFEnabled: false}, slog.Default(), apphttp.Dependencies{
		RouteProviders: []apphttp.RouteProvider{controller, login},
	})
	return app, manager
}

func loginRouteInspectorUser(t *testing.T, app *fiber.App, userID int64) *http.Cookie {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, "/api/v1/route-inspector-login/"+strconv.FormatInt(userID, 10), nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || len(response.Cookies()) == 0 {
		t.Fatalf("login %d status=%d cookies=%d", userID, response.StatusCode, len(response.Cookies()))
	}
	return response.Cookies()[0]
}
