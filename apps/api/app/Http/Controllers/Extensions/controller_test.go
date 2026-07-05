package extensionscontroller

import (
	"context"
	"encoding/json"
	"io"
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

	resp = performExtensionRequest(t, app, http.MethodPost, "/api/v1/admin/extensions/demo.theme/verify", cookie)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 verifying theme without extension.manage, got %d", resp.StatusCode)
	}

	resp = performExtensionRequest(t, app, http.MethodPost, "/api/v1/admin/extensions/demo.theme/activate", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 activating theme without session, got %d", resp.StatusCode)
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
	if len(listBody.Data) != 2 || !extensionListContains(listBody.Data, "demo.plugin") || !extensionListContains(listBody.Data, "demo.theme") {
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

func TestControllerProxiesOnlyDeclaredPluginRoutesAfterHostAuthorization(t *testing.T) {
	app, manager, store := newExtensionTestApp()
	store.items["demo.plugin"] = extensions.Extension{
		ID:      "demo.plugin",
		Name:    "Demo Plugin",
		Version: "1.0.0",
		Type:    extensions.TypePlugin,
		Status:  extensions.StatusEnabled,
		Manifest: extensions.Manifest{
			ID:            "demo.plugin",
			Name:          "Demo Plugin",
			Version:       "1.0.0",
			Type:          extensions.TypePlugin,
			SForumVersion: "^1.0.0",
			Permissions:   []string{"extension.demo.manage"},
			Routes: []extensions.ManifestRoute{
				{Path: "/hello", Methods: []string{"GET"}, Access: extensions.RouteAccessPublic},
				{Path: "/profile", Methods: []string{"GET"}, Access: extensions.RouteAccessLogin},
				{Path: "/admin/reindex", Methods: []string{"POST"}, Access: extensions.RouteAccessPermission, Permission: "extension.demo.manage"},
			},
		},
		InstalledAt: time.Now(),
		UpdatedAt:   time.Now(),
	}

	resp := performExtensionRequest(t, app, http.MethodGet, "/api/v1/extensions/demo.plugin/hello", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected public route 200, got %d", resp.StatusCode)
	}
	if body := responseBody(t, resp); body != "plugin-ok" {
		t.Fatalf("expected plugin proxy body, got %q", body)
	}

	resp = performExtensionRequest(t, app, http.MethodGet, "/api/v1/extensions/demo.plugin/profile", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected login route 401, got %d", resp.StatusCode)
	}

	ordinaryCookie := loginExtensionUser(t, app, manager, 2)
	resp = performExtensionRequest(t, app, http.MethodPost, "/api/v1/extensions/demo.plugin/admin/reindex", ordinaryCookie)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected permission route 403, got %d", resp.StatusCode)
	}

	managerCookie := loginExtensionUser(t, app, manager, 1)
	resp = performExtensionRequest(t, app, http.MethodPost, "/api/v1/extensions/demo.plugin/admin/reindex", managerCookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected permission route 200, got %d", resp.StatusCode)
	}
	if body := responseBody(t, resp); body != "plugin-ok" {
		t.Fatalf("expected plugin proxy body, got %q", body)
	}

	resp = performExtensionRequest(t, app, http.MethodDelete, "/api/v1/extensions/demo.plugin/hello", managerCookie)
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected undeclared method 405, got %d", resp.StatusCode)
	}

	resp = performExtensionRequest(t, app, http.MethodGet, "/api/v1/extensions/demo.plugin/missing", managerCookie)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected undeclared path 404, got %d", resp.StatusCode)
	}
}

func TestControllerVerifiesAndActivatesThemesForManager(t *testing.T) {
	app, manager, store := newExtensionTestApp()
	cookie := loginExtensionUser(t, app, manager, 1)

	resp := performExtensionRequest(t, app, http.MethodPost, "/api/v1/admin/extensions/demo.theme/verify", cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 verify, got %d", resp.StatusCode)
	}
	if store.verifiedID != "demo.theme" {
		t.Fatalf("expected store verify call for demo.theme, got %q", store.verifiedID)
	}

	resp = performExtensionRequest(t, app, http.MethodPost, "/api/v1/admin/extensions/demo.theme/activate", cookie)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 uploaded theme activation unavailable, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var body testEnvelope[testErrorData]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode activation error envelope: %v", err)
	}
	if body.Data.Reason != extensions.CodeThemeRuntimeUnavailable {
		t.Fatalf("expected theme runtime unavailable reason, got %q", body.Data.Reason)
	}
}

func newExtensionTestApp() (*fiber.App, *authsession.Manager, *controllerFakeStore) {
	manager := authsession.NewManager(session.NewStore(), authsession.Config{HashSecret: "test-secret"})
	users := controllerActors{actors: map[int64]identity.Actor{
		1: {
			ID:          1,
			Status:      identity.UserStatusActive,
			Permissions: map[string]bool{identity.PermissionExtensionManage: true, "extension.demo.manage": true},
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
		"demo.theme": {
			ID:      "demo.theme",
			Name:    "Demo Theme",
			Version: "1.0.0",
			Type:    extensions.TypeTheme,
			Status:  extensions.StatusInstalled,
			Source:  extensions.SourceUploaded,
			Manifest: extensions.Manifest{
				ID:            "demo.theme",
				Name:          "Demo Theme",
				Version:       "1.0.0",
				Type:          extensions.TypeTheme,
				SForumVersion: "^1.0.0",
				Frontend:      extensions.ManifestFrontend{Layer: "layer"},
			},
			InstalledAt: time.Now(),
			UpdatedAt:   time.Now(),
		},
	}}
	controller := NewControllerWithGateway(extensions.NewServiceWithHooks(store, "storage/extensions", nil, controllerThemeBuilder{}), users, manager, controllerFakeGateway{})
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

func responseBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return string(body)
}

func extensionListContains(items []extensions.Extension, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
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

type controllerFakeGateway struct{}

func (controllerFakeGateway) Proxy(c fiber.Ctx, input ProxyInput) error {
	c.Status(http.StatusOK)
	return c.SendString("plugin-ok")
}

type controllerThemeBuilder struct{}

func (controllerThemeBuilder) Build(context.Context, extensions.Extension) error {
	return nil
}

type controllerFakeStore struct {
	items      map[string]extensions.Extension
	enabledID  string
	disabledID string
	verifiedID string
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
		Source:      extensions.SourceUploaded,
		IsDeletable: true,
		Manifest:    input.Manifest,
		PackagePath: input.PackagePath,
		InstalledAt: time.Now(),
		UpdatedAt:   time.Now(),
	}
	s.items[item.ID] = item
	return item, nil
}

func (s *controllerFakeStore) SaveBuiltin(_ context.Context, input extensions.SaveBuiltinInput) (extensions.Extension, error) {
	item := extensions.Extension{
		ID:          input.Manifest.ID,
		Name:        input.Manifest.Name,
		Version:     input.Manifest.Version,
		Type:        input.Manifest.Type,
		Status:      extensions.StatusEnabled,
		Source:      extensions.SourceBuiltin,
		IsSystem:    true,
		IsDeletable: false,
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

func (s *controllerFakeStore) ActivateTheme(_ context.Context, id string) (extensions.Extension, error) {
	item := s.items[id]
	item.Status = extensions.StatusEnabled
	s.items[id] = item
	return item, nil
}

func (s *controllerFakeStore) ActiveTheme(context.Context) (extensions.Extension, error) {
	for _, item := range s.items {
		if item.Type == extensions.TypeTheme && item.Status == extensions.StatusEnabled {
			return item, nil
		}
	}
	return extensions.Extension{}, extensions.ErrExtensionNotFound
}

func (s *controllerFakeStore) CreateEvent(_ context.Context, input extensions.EventInput) (extensions.ExtensionEvent, error) {
	if input.Action == extensions.EventVerified {
		s.verifiedID = input.ExtensionID
	}
	event := extensions.ExtensionEvent{ID: int64(len(s.events) + 1), ExtensionID: input.ExtensionID, ActorUserID: input.ActorUserID, Action: input.Action, Message: input.Message, CreatedAt: time.Now()}
	s.events = append(s.events, event)
	return event, nil
}

func (s *controllerFakeStore) ListEvents(context.Context, string, int) ([]extensions.ExtensionEvent, error) {
	return s.events, nil
}
