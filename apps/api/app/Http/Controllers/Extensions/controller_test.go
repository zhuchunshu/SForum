package extensionscontroller

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

type testEnvelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type testErrorData struct {
	Reason string `json:"reason"`
}

func TestControllerRequiresLoginAndExtensionManagePermission(t *testing.T) {
	app, manager, _ := newExtensionTestApp()

	resp := performExtensionRequest(t, app, http.MethodGet, "/api/v1/admin/extensions", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without session, got %d", resp.StatusCode)
	}

	cookie := loginExtensionUser(t, app, manager, 2)
	resp = performExtensionRequest(t, app, http.MethodGet, "/api/v1/admin/extensions", cookie)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 without extension.manage, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var body testEnvelope[testErrorData]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if body.Data.Reason != "permission.denied" {
		t.Fatalf("expected permission.denied, got %q", body.Data.Reason)
	}
}

func TestControllerListsAndEnablesExtensionsForManager(t *testing.T) {
	app, manager, store := newExtensionTestApp()
	cookie := loginExtensionUser(t, app, manager, 1)

	resp := performExtensionRequest(t, app, http.MethodGet, "/api/v1/admin/extensions", cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 list, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var listBody testEnvelope[[]extensions.Extension]
	if err := json.NewDecoder(resp.Body).Decode(&listBody); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listBody.Data) != 1 || listBody.Data[0].ID != "demo.plugin" {
		t.Fatalf("unexpected extension list: %#v", listBody.Data)
	}

	resp = performExtensionRequest(t, app, http.MethodPost, "/api/v1/admin/extensions/demo.plugin/enable", cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 enable, got %d", resp.StatusCode)
	}
	if store.enabledID != "demo.plugin" {
		t.Fatalf("expected store enable call for demo.plugin, got %q", store.enabledID)
	}
}

func newExtensionTestApp() (*fiber.App, *authsession.Manager, *controllerFakeStore) {
	manager := authsession.NewManager(session.NewStore(), authsession.Config{HashSecret: "test-secret"})
	users := controllerActors{actors: map[int64]identity.Actor{
		1: {
			ID:          1,
			Status:      identity.UserStatusActive,
			Permissions: map[string]bool{identity.PermissionExtensionManage: true},
		},
		2: {ID: 2, Status: identity.UserStatusActive, Permissions: map[string]bool{}},
	}}
	store := &controllerFakeStore{items: map[string]extensions.Extension{
		"demo.plugin": {
			ID:      "demo.plugin",
			Name:    "Demo Plugin",
			Version: "1.0.0",
			Type:    extensions.TypePlugin,
			Status:  extensions.StatusInstalled,
			Manifest: extensions.Manifest{
				ID:            "demo.plugin",
				Name:          "Demo Plugin",
				Version:       "1.0.0",
				Type:          extensions.TypePlugin,
				SForumVersion: "^1.0.0",
			},
			InstalledAt: time.Now(),
			UpdatedAt:   time.Now(),
		},
	}}
	controller := NewController(extensions.NewService(store, "storage/extensions"), users, manager)
	loginProvider := extensionRouteProviderFunc(func(api fiber.Router) {
		api.Post("/test-login/:id", func(c fiber.Ctx) error {
			var id int64 = 1
			if c.Params("id") == "2" {
				id = 2
			}
			_, err := manager.Start(c, id)
			return err
		})
	})
	app := apphttp.NewApp(config.Config{AppName: "SForum", AppEnv: "test", AppLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}, slog.Default(), apphttp.Dependencies{
		RouteProviders: []apphttp.RouteProvider{controller, loginProvider},
	})
	return app, manager, store
}

func loginExtensionUser(t *testing.T, app *fiber.App, _ *authsession.Manager, userID int64) *http.Cookie {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test-login/1", nil)
	if userID == 2 {
		req = httptest.NewRequest(http.MethodPost, "/api/v1/test-login/2", nil)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected login 200, got %d", resp.StatusCode)
	}
	if len(resp.Cookies()) == 0 {
		t.Fatal("expected login cookie")
	}
	return resp.Cookies()[0]
}

func performExtensionRequest(t *testing.T, app *fiber.App, method string, path string, cookie *http.Cookie) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s %s failed: %v", method, path, err)
	}
	return resp
}

type controllerActors struct {
	actors map[int64]identity.Actor
}

func (s controllerActors) LoadActor(_ context.Context, userID int64) (identity.Actor, error) {
	return s.actors[userID], nil
}

type extensionRouteProviderFunc func(api fiber.Router)

func (f extensionRouteProviderFunc) RegisterRoutes(api fiber.Router) {
	f(api)
}

type controllerFakeStore struct {
	items      map[string]extensions.Extension
	enabledID  string
	disabledID string
	events     []extensions.ExtensionEvent
}

func (s *controllerFakeStore) List(context.Context) ([]extensions.Extension, error) {
	items := make([]extensions.Extension, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	return items, nil
}

func (s *controllerFakeStore) Get(_ context.Context, id string) (extensions.Extension, error) {
	item, ok := s.items[id]
	if !ok {
		return extensions.Extension{}, extensions.ErrExtensionNotFound
	}
	return item, nil
}

func (s *controllerFakeStore) SaveInstalled(_ context.Context, input extensions.SaveInstalledInput) (extensions.Extension, error) {
	item := extensions.Extension{
		ID:          input.Manifest.ID,
		Name:        input.Manifest.Name,
		Version:     input.Manifest.Version,
		Type:        input.Manifest.Type,
		Status:      extensions.StatusInstalled,
		Manifest:    input.Manifest,
		PackagePath: input.PackagePath,
		InstalledAt: time.Now(),
		UpdatedAt:   time.Now(),
	}
	s.items[item.ID] = item
	return item, nil
}

func (s *controllerFakeStore) Enable(_ context.Context, id string, _ string) (extensions.Extension, error) {
	item := s.items[id]
	item.Status = extensions.StatusEnabled
	s.items[id] = item
	s.enabledID = id
	return item, nil
}

func (s *controllerFakeStore) Disable(_ context.Context, id string) (extensions.Extension, error) {
	item := s.items[id]
	item.Status = extensions.StatusDisabled
	s.items[id] = item
	s.disabledID = id
	return item, nil
}

func (s *controllerFakeStore) CreateEvent(_ context.Context, input extensions.EventInput) (extensions.ExtensionEvent, error) {
	event := extensions.ExtensionEvent{ID: int64(len(s.events) + 1), ExtensionID: input.ExtensionID, ActorUserID: input.ActorUserID, Action: input.Action, Message: input.Message, CreatedAt: time.Now()}
	s.events = append(s.events, event)
	return event, nil
}

func (s *controllerFakeStore) ListEvents(context.Context, string, int) ([]extensions.ExtensionEvent, error) {
	return s.events, nil
}
