package extensionscontroller

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	navigationregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/NavigationRegistry"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

func TestCompositionAndNavigationInspectorsAllowExtensionView(t *testing.T) {
	app := newCompositionInspectorTestApp(t, true)
	viewer := loginCompositionInspectorUser(t, app, 1)
	for _, path := range []string{
		"/api/v1/admin/extensions/component-inspector",
		"/api/v1/admin/extensions/navigation-inspector",
	} {
		response := performExtensionRequest(t, app, http.MethodGet, path, viewer)
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, response.StatusCode, responseBody(t, response))
		}
		var envelope testEnvelope[map[string]any]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if envelope.Data == nil {
			t.Fatalf("%s empty data", path)
		}
	}
}

func TestCompositionAndNavigationInspectorsDenyWithoutPermission(t *testing.T) {
	app := newCompositionInspectorTestApp(t, true)
	denied := loginCompositionInspectorUser(t, app, 3)
	for _, path := range []string{
		"/api/v1/admin/extensions/component-inspector",
		"/api/v1/admin/extensions/navigation-inspector",
	} {
		response := performExtensionRequest(t, app, http.MethodGet, path, denied)
		if response.StatusCode != http.StatusForbidden && response.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s denied status=%d", path, response.StatusCode)
		}
		_ = response.Body.Close()
	}
}

func TestCompositionInspectorFailsClosedWhenUnavailable(t *testing.T) {
	app := newCompositionInspectorTestApp(t, false)
	viewer := loginCompositionInspectorUser(t, app, 1)
	for _, path := range []string{
		"/api/v1/admin/extensions/component-inspector",
		"/api/v1/admin/extensions/navigation-inspector",
	} {
		response := performExtensionRequest(t, app, http.MethodGet, path, viewer)
		if response.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("%s unavailable status=%d body=%s", path, response.StatusCode, responseBody(t, response))
		}
		_ = response.Body.Close()
	}
}

func TestCompositionInspectorRejectsInvalidLimit(t *testing.T) {
	app := newCompositionInspectorTestApp(t, true)
	viewer := loginCompositionInspectorUser(t, app, 1)
	response := performExtensionRequest(t, app, http.MethodGet, "/api/v1/admin/extensions/component-inspector?limit=0", viewer)
	if response.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("invalid limit status=%d", response.StatusCode)
	}
	_ = response.Body.Close()
}

func newCompositionInspectorTestApp(t *testing.T, configured bool) *fiber.App {
	t.Helper()
	manager := authsession.NewManager(session.NewStore(), authsession.Config{HashSecret: "test-secret"})
	actors := controllerActors{actors: map[int64]identity.Actor{
		1: {ID: 1, Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionExtensionView: true}},
		3: {ID: 3, Status: identity.UserStatusActive},
	}}
	controller := NewController(nil, actors, manager)
	if configured {
		registry := extensionsruntime.NewComponentRegistry()
		composition, err := extensionsruntime.NewProductionComponentComposition(
			extensionsruntime.ProductionComponentCompositionConfig{Registry: registry},
		)
		if err != nil {
			t.Fatal(err)
		}
		nav := navigationregistry.New()
		if _, err := nav.Publish(navigationregistry.CorePublication()); err != nil {
			t.Fatal(err)
		}
		controller.WithComponentCompositionInspector(registry, composition).
			WithNavigationInspector(navigationregistry.NewInspector(nav, navigationregistry.NewTraceRing(32)))
	}
	login := extensionRouteProviderFunc(func(router fiber.Router) {
		router.Post("/composition-inspector-login/:id", func(c fiber.Ctx) error {
			id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
			_, err := manager.Start(c, id)
			return err
		})
	})
	return apphttp.NewApp(config.Config{AppName: "SForum", AppEnv: "test", CSRFEnabled: false}, slog.Default(), apphttp.Dependencies{
		RouteProviders: []apphttp.RouteProvider{controller, login},
	})
}

func loginCompositionInspectorUser(t *testing.T, app *fiber.App, userID int64) *http.Cookie {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, "/api/v1/composition-inspector-login/"+strconv.FormatInt(userID, 10), nil)
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
