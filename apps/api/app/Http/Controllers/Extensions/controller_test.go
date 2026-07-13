package extensionscontroller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
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

func TestMapExtensionSettingsRollbackFailure(t *testing.T) {
	mapped := mapExtensionError(errors.Join(
		extensions.ErrSettingsRollbackFailed,
		errors.New("restore database unavailable"),
	))
	fiberErr, ok := mapped.(*fiber.Error)
	if !ok {
		t.Fatalf("expected fiber error, got %T", mapped)
	}
	if fiberErr.Code != http.StatusServiceUnavailable || fiberErr.Message != extensions.CodeSettingsRollbackFailed {
		t.Fatalf("unexpected mapping: %#v", fiberErr)
	}
}

func TestControllerRequiresLoginAndExtensionManagePermission(t *testing.T) {
	app, manager, _ := newExtensionTestApp(t)

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
	app, manager, store := newExtensionTestApp(t)
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

func TestControllerListsNavigationAndManagesExtensionSettings(t *testing.T) {
	app, manager, store := newExtensionTestApp(t)
	cookie := loginExtensionUser(t, app, manager, 1)
	plugin := store.items["demo.plugin"]
	plugin.Status = extensions.StatusEnabled
	plugin.Manifest.Admin = extensions.ManifestAdmin{
		Entry: "/settings",
		Pages: []extensions.ManifestAdminPage{
			{Path: "/settings", Label: "Settings", View: "settings", Icon: "i-lucide-settings", Order: 10},
			{Path: "/dashboard", Label: "Dashboard", View: "about", Icon: "i-lucide-layout-dashboard", Order: 5, Menu: true},
		},
	}
	plugin.Manifest.Settings = []extensions.ManifestSetting{{Key: "demo.title", Label: extensionmanifest.LocalizedText{Default: "Title"}, Type: "text", Default: "Hello"}}
	store.items[plugin.ID] = plugin
	theme := store.items["demo.theme"]
	theme.Status = extensions.StatusEnabled
	store.items[theme.ID] = theme

	resp := performExtensionRequest(t, app, http.MethodGet, "/api/v1/admin/extensions/navigation", cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 navigation, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var navigation testEnvelope[[]extensions.ExtensionAdminNavigationItem]
	if err := json.NewDecoder(resp.Body).Decode(&navigation); err != nil {
		t.Fatalf("decode navigation: %v", err)
	}
	if !controllerNavigationContains(navigation.Data, "demo.plugin", "/dashboard") || controllerNavigationContains(navigation.Data, "demo.plugin", "/settings") || controllerNavigationContains(navigation.Data, "demo.theme", "/about") {
		t.Fatalf("unexpected navigation items: %#v", navigation.Data)
	}

	resp = performExtensionRequest(t, app, http.MethodGet, "/api/v1/admin/extensions/demo.plugin/settings", cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 settings, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var settings testEnvelope[extensions.ExtensionSettings]
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if controllerSettingValue(settings.Data, "demo.title") != "Hello" {
		t.Fatalf("expected default setting, got %#v", settings.Data)
	}

	resp = performExtensionJSONRequest(t, app, http.MethodPut, "/api/v1/admin/extensions/demo.plugin/settings", cookie, `{"values":{"demo.title":"Updated"}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 update settings, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		t.Fatalf("decode updated settings: %v", err)
	}
	if controllerSettingValue(settings.Data, "demo.title") != "Updated" {
		t.Fatalf("expected updated setting, got %#v", settings.Data)
	}

	resp = performExtensionRequest(t, app, http.MethodPost, "/api/v1/admin/extensions/demo.plugin/settings/reset", cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 reset settings, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		t.Fatalf("decode reset settings: %v", err)
	}
	if controllerSettingValue(settings.Data, "demo.title") != "Hello" {
		t.Fatalf("expected default after reset, got %#v", settings.Data)
	}
}

func TestControllerProxiesOnlyDeclaredPluginRoutesAfterHostAuthorization(t *testing.T) {
	app, manager, store := newExtensionTestApp(t)
	store.items["demo.plugin"] = extensions.Extension{
		ID:      "demo.plugin",
		Name:    "Demo Plugin",
		Version: "1.0.0",
		Type:    extensions.TypePlugin,
		Status:  extensions.StatusEnabled,
		Manifest: extensions.Manifest{
			ID:            "demo.plugin",
			Name:          "Demo Plugin",
			Description:   "Demo plugin for controller tests.",
			URL:           "https://example.com/demo-plugin",
			Author:        extensions.ManifestAuthor{Name: "SForum Team", URL: "https://example.com", Email: "dev@example.com"},
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
	app, manager, store := newExtensionTestApp(t)
	cookie := loginExtensionUser(t, app, manager, 1)

	resp := performExtensionRequest(t, app, http.MethodPost, "/api/v1/admin/extensions/demo.theme/verify", cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 verify, got %d", resp.StatusCode)
	}
	if store.verifiedID != "demo.theme" {
		t.Fatalf("expected store verify call for demo.theme, got %q", store.verifiedID)
	}

	resp = performExtensionRequest(t, app, http.MethodPost, "/api/v1/admin/extensions/demo.theme/activate", cookie)
	// Runtime Page Registry：主题激活同步完成，不触发 Nuxt 构建。
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 runtime theme activation, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var body testEnvelope[extensions.Extension]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode activation response envelope: %v", err)
	}
	if body.Data.ID != "demo.theme" {
		t.Fatalf("expected demo.theme activated, got %#v", body.Data)
	}
}

func TestControllerListsExtensionEventDefinitionsAndDeliveries(t *testing.T) {
	app, manager, store := newExtensionTestApp(t)
	cookie := loginExtensionUser(t, app, manager, 1)
	store.deliveries = []extensions.ExtensionEventDelivery{{
		ID:            7,
		ExtensionID:   "demo.plugin",
		EventName:     appevents.TopicCreated,
		EventKind:     appevents.KindObserve,
		Status:        extensions.DeliverySucceeded,
		CorrelationID: "corr-1",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}}

	resp := performExtensionRequest(t, app, http.MethodGet, "/api/v1/admin/extensions/event-definitions", cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 event definitions, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var definitions testEnvelope[[]appevents.Definition]
	if err := json.NewDecoder(resp.Body).Decode(&definitions); err != nil {
		t.Fatalf("decode event definitions: %v", err)
	}
	if !eventDefinitionListContains(definitions.Data, appevents.TopicBeforeCreate) {
		t.Fatalf("expected topic.before_create definition, got %#v", definitions.Data)
	}
	if !eventDefinitionListContains(definitions.Data, appevents.CommentBeforeCreate) {
		t.Fatalf("expected comment.before_create definition, got %#v", definitions.Data)
	}
	if !eventDefinitionListContains(definitions.Data, appevents.TopicBeforeUpdate) {
		t.Fatalf("expected topic.before_update definition, got %#v", definitions.Data)
	}
	if !eventDefinitionListContains(definitions.Data, appevents.UserBeforeRegister) {
		t.Fatalf("expected user.before_register definition, got %#v", definitions.Data)
	}
	if !eventDefinitionListContains(definitions.Data, appevents.AttachmentBeforeUpload) {
		t.Fatalf("expected attachment.before_upload definition, got %#v", definitions.Data)
	}

	resp = performExtensionRequest(t, app, http.MethodGet, "/api/v1/admin/extensions/event-deliveries?extensionId=demo.plugin", cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 event deliveries, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var deliveries testEnvelope[[]extensions.ExtensionEventDelivery]
	if err := json.NewDecoder(resp.Body).Decode(&deliveries); err != nil {
		t.Fatalf("decode event deliveries: %v", err)
	}
	if len(deliveries.Data) != 1 || deliveries.Data[0].EventName != appevents.TopicCreated {
		t.Fatalf("unexpected deliveries: %#v", deliveries.Data)
	}
}

func TestControllerListsContributionPointsAndContributions(t *testing.T) {
	app, manager, store := newExtensionTestApp(t)
	cookie := loginExtensionUser(t, app, manager, 1)
	plugin := store.items["demo.plugin"]
	plugin.Status = extensions.StatusEnabled
	plugin.Manifest.Contributions = []extensions.ManifestContribution{{
		Point: "forum.topic.actions",
		ID:    "demo.bookmark",
		Order: 100,
		Label: map[string]string{
			"zh-CN": "收藏",
			"en-US": "Bookmark",
		},
		Icon:    "i-lucide-bookmark",
		Payload: json.RawMessage(`{"type":"extensionRoute","method":"POST","path":"/topic-actions/bookmark"}`),
	}}
	store.items[plugin.ID] = plugin

	resp := performExtensionRequest(t, app, http.MethodGet, "/api/v1/admin/extensions/contribution-points", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without session, got %d", resp.StatusCode)
	}

	ordinaryCookie := loginExtensionUser(t, app, manager, 2)
	resp = performExtensionRequest(t, app, http.MethodGet, "/api/v1/admin/extensions/contributions", ordinaryCookie)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 without extension.manage, got %d", resp.StatusCode)
	}

	resp = performExtensionRequest(t, app, http.MethodGet, "/api/v1/admin/extensions/contribution-points", cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 contribution points, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var points testEnvelope[[]extensions.ContributionPointDefinition]
	if err := json.NewDecoder(resp.Body).Decode(&points); err != nil {
		t.Fatalf("decode contribution points: %v", err)
	}
	pointIDs := make(map[string]bool, len(points.Data))
	for _, point := range points.Data {
		pointIDs[point.ID] = true
	}
	// F4.3 + E2 公开贡献点 + jobs + extension settings。
	requiredPoints := []string{
		"forum.topic.actions",
		"forum.topic.sidebar",
		"forum.topic.badges",
		"forum.comment.actions",
		"forum.nav.items",
		"forum.topic.list.badges",
		"forum.composer.toolbar",
		"forum.profile.tabs",
		"admin.dashboard.widgets",
		"system.health.checks",
	}
	if len(points.Data) != len(requiredPoints) {
		t.Fatalf("unexpected contribution points count %d: %#v", len(points.Data), points.Data)
	}
	for _, id := range requiredPoints {
		if !pointIDs[id] {
			t.Fatalf("missing contribution point %q in %#v", id, points.Data)
		}
	}

	resp = performExtensionRequest(t, app, http.MethodGet, "/api/v1/admin/extensions/contributions", cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 contributions, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var contributions testEnvelope[[]extensions.EffectiveContribution]
	if err := json.NewDecoder(resp.Body).Decode(&contributions); err != nil {
		t.Fatalf("decode contributions: %v", err)
	}
	if len(contributions.Data) != 1 || contributions.Data[0].ExtensionID != "demo.plugin" || contributions.Data[0].ID != "demo.bookmark" {
		t.Fatalf("unexpected contributions: %#v", contributions.Data)
	}
}

func newExtensionTestApp(t *testing.T) (*fiber.App, *authsession.Manager, *controllerFakeStore) {
	t.Helper()
	manager := authsession.NewManager(session.NewStore(), authsession.Config{HashSecret: "test-secret"})
	users := controllerActors{actors: map[int64]identity.Actor{
		1: {
			ID:          1,
			Status:      identity.UserStatusActive,
			Permissions: map[string]bool{identity.PermissionExtensionManage: true, "extension.demo.manage": true},
		},
		2: {ID: 2, Status: identity.UserStatusActive, Permissions: map[string]bool{}},
	}}
	plugin := extensions.Extension{
		ID:      "demo.plugin",
		Name:    "Demo Plugin",
		Version: "1.0.0",
		Type:    extensions.TypePlugin,
		Status:  extensions.StatusInstalled,
		Source:  extensions.SourceUploaded,
		Manifest: extensions.Manifest{
			ID:            "demo.plugin",
			Name:          "Demo Plugin",
			Description:   "Demo plugin for route tests.",
			URL:           "https://example.com/demo-plugin",
			Author:        extensions.ManifestAuthor{Name: "SForum Team", URL: "https://example.com", Email: "dev@example.com"},
			Version:       "1.0.0",
			Type:          extensions.TypePlugin,
			SForumVersion: "^1.0.0",
		},
		InstalledAt: time.Now(),
		UpdatedAt:   time.Now(),
	}
	plugin.PackagePath = controllerInstalledPackage(t, plugin.Manifest)
	theme := extensions.Extension{
		ID:      "demo.theme",
		Name:    "Demo Theme",
		Version: "1.0.0",
		Type:    extensions.TypeTheme,
		Status:  extensions.StatusInstalled,
		Source:  extensions.SourceUploaded,
		Manifest: extensions.Manifest{
			ID:            "demo.theme",
			Name:          "Demo Theme",
			Description:   "Demo theme for controller tests.",
			URL:           "https://example.com/demo-theme",
			Author:        extensions.ManifestAuthor{Name: "SForum Team", URL: "https://example.com", Email: "dev@example.com"},
			Version:       "1.0.0",
			Type:          extensions.TypeTheme,
			SForumVersion: "^1.0.0",
		},
		InstalledAt: time.Now(),
		UpdatedAt:   time.Now(),
	}
	theme.PackagePath = controllerInstalledPackage(t, theme.Manifest)
	store := &controllerFakeStore{items: map[string]extensions.Extension{
		plugin.ID: plugin,
		theme.ID:  theme,
	}}
	controller := NewControllerWithGateway(extensions.NewServiceWithHooks(store, "storage/extensions", nil), users, manager, controllerFakeGateway{})
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
	app := apphttp.NewApp(config.Config{AppName: "SForum", AppEnv: "test", CSRFEnabled: false, AppLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}, slog.Default(), apphttp.Dependencies{
		RouteProviders: []apphttp.RouteProvider{controller, loginProvider},
	})
	return app, manager, store
}

func controllerInstalledPackage(t *testing.T, manifest extensions.Manifest) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), manifest.ID, manifest.Version)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("create extension package root: %v", err)
	}
	packagePath := filepath.Join(root, "package.zip")
	if err := os.WriteFile(packagePath, []byte("zip"), 0o600); err != nil {
		t.Fatalf("write extension archive: %v", err)
	}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("marshal extension manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, extensions.ManifestFileName), body, 0o600); err != nil {
		t.Fatalf("write extension manifest: %v", err)
	}
	if manifest.Type == extensions.TypeTheme {
		if err := os.WriteFile(filepath.Join(root, "theme.json"), []byte(`{"schemaVersion":1,"styles":{"tokens":{}}}`), 0o600); err != nil {
			t.Fatalf("write theme contract: %v", err)
		}
	}
	return packagePath
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

func performExtensionJSONRequest(t *testing.T, app *fiber.App, method string, path string, cookie *http.Cookie, body string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
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

func eventDefinitionListContains(items []appevents.Definition, name string) bool {
	for _, item := range items {
		if item.Name == name {
			return true
		}
	}
	return false
}

func controllerNavigationContains(items []extensions.ExtensionAdminNavigationItem, extensionID string, pagePath string) bool {
	for _, item := range items {
		if item.ExtensionID == extensionID && item.Path == pagePath {
			return true
		}
	}
	return false
}

func controllerSettingValue(settings extensions.ExtensionSettings, key string) string {
	for _, item := range settings.Items {
		if item.Key == key {
			return item.Value
		}
	}
	return ""
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

type controllerFakeStore struct {
	items      map[string]extensions.Extension
	enabledID  string
	disabledID string
	verifiedID string
	settings   map[string]map[string]string
	events     []extensions.ExtensionEvent
	deliveries []extensions.ExtensionEventDelivery
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

func (s *controllerFakeStore) PromoteStagedVersion(_ context.Context, input extensions.StagedVersionCASInput) (extensions.Extension, error) {
	item, ok := s.items[input.ExtensionID]
	if !ok {
		return extensions.Extension{}, extensions.ErrExtensionNotFound
	}
	if item.StagedVersion == nil {
		return extensions.Extension{}, extensions.ErrStagedVersionNotFound
	}
	if item.StagedVersion.ID != input.ExpectedStagedVersionID || item.StagedVersion.PackageDigest != input.ExpectedPackageDigest {
		return extensions.Extension{}, extensions.ErrStagedVersionConflict
	}
	staged := item.StagedVersion
	item.Version, item.Manifest = staged.Version, staged.Manifest
	item.PackageDigest, item.AdminFrontendDigest = staged.PackageDigest, staged.AdminFrontendDigest
	item.PackagePath, item.ActiveVersionID = staged.PackagePath, staged.ID
	item.InstalledAt, item.StagedVersion = staged.InstalledAt, nil
	s.items[item.ID] = item
	return item, nil
}

func (s *controllerFakeStore) DiscardStagedVersion(_ context.Context, input extensions.StagedVersionCASInput) (extensions.Extension, error) {
	item, ok := s.items[input.ExtensionID]
	if !ok {
		return extensions.Extension{}, extensions.ErrExtensionNotFound
	}
	if item.StagedVersion == nil {
		return extensions.Extension{}, extensions.ErrStagedVersionNotFound
	}
	if item.StagedVersion.ID != input.ExpectedStagedVersionID || item.StagedVersion.PackageDigest != input.ExpectedPackageDigest {
		return extensions.Extension{}, extensions.ErrStagedVersionConflict
	}
	item.StagedVersion = nil
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

func (s *controllerFakeStore) PruneMissingBuiltins(_ context.Context, activeIDs []string) error {
	active := map[string]bool{}
	for _, id := range activeIDs {
		active[id] = true
	}
	for id, item := range s.items {
		if item.Source == extensions.SourceBuiltin && !active[id] {
			delete(s.items, id)
		}
	}
	return nil
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

func (s *controllerFakeStore) ListSettings(_ context.Context, extensionID string) (map[string]string, error) {
	if s.settings == nil {
		return map[string]string{}, nil
	}
	values := map[string]string{}
	for key, value := range s.settings[extensionID] {
		values[key] = value
	}
	return values, nil
}

func (s *controllerFakeStore) ReplaceSettings(_ context.Context, extensionID string, values map[string]string) error {
	if s.settings == nil {
		s.settings = map[string]map[string]string{}
	}
	next := map[string]string{}
	for key, value := range values {
		next[key] = value
	}
	s.settings[extensionID] = next
	return nil
}

func (s *controllerFakeStore) CompareAndSwapSetting(_ context.Context, extensionID, name, oldValue, newValue string) (bool, error) {
	if s.settings == nil || s.settings[extensionID][name] != oldValue {
		return false, nil
	}
	s.settings[extensionID][name] = newValue
	return true, nil
}

func (s *controllerFakeStore) ResetSettings(_ context.Context, extensionID string) error {
	if s.settings != nil {
		delete(s.settings, extensionID)
	}
	return nil
}

func (s *controllerFakeStore) Delete(_ context.Context, id string) error {
	if _, ok := s.items[id]; !ok {
		return extensions.ErrExtensionNotFound
	}
	delete(s.items, id)
	if s.settings != nil {
		delete(s.settings, id)
	}
	return nil
}

func (s *controllerFakeStore) ListMigrationLedger(context.Context, string) ([]extensions.MigrationRecord, error) {
	return []extensions.MigrationRecord{}, nil
}

func (s *controllerFakeStore) RecordMigration(context.Context, string, extensions.MigrationRecord) error {
	return nil
}

func (s *controllerFakeStore) CreateEventDelivery(_ context.Context, input extensions.EventDeliveryInput) (extensions.ExtensionEventDelivery, error) {
	delivery := extensions.ExtensionEventDelivery{
		ID:            int64(len(s.deliveries) + 1),
		ExtensionID:   input.ExtensionID,
		EventName:     input.EventName,
		EventKind:     input.EventKind,
		Status:        input.Status,
		Reason:        input.Reason,
		Message:       input.Message,
		CorrelationID: input.CorrelationID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	s.deliveries = append(s.deliveries, delivery)
	return delivery, nil
}

func (s *controllerFakeStore) UpdateEventDelivery(_ context.Context, input extensions.EventDeliveryUpdateInput) error {
	for index := range s.deliveries {
		if s.deliveries[index].ID == input.ID {
			s.deliveries[index].Status = input.Status
			s.deliveries[index].Reason = input.Reason
			s.deliveries[index].Message = input.Message
			s.deliveries[index].AttemptCount = input.AttemptCount
			s.deliveries[index].UpdatedAt = time.Now()
			if input.Completed {
				completedAt := time.Now()
				s.deliveries[index].CompletedAt = &completedAt
			}
			return nil
		}
	}
	return extensions.ErrExtensionNotFound
}

func (s *controllerFakeStore) ListEventDeliveries(_ context.Context, input extensions.EventDeliveryListInput) ([]extensions.ExtensionEventDelivery, error) {
	items := []extensions.ExtensionEventDelivery{}
	for _, delivery := range s.deliveries {
		if input.ExtensionID != "" && delivery.ExtensionID != input.ExtensionID {
			continue
		}
		if input.EventName != "" && delivery.EventName != input.EventName {
			continue
		}
		if input.Status != "" && delivery.Status != input.Status {
			continue
		}
		items = append(items, delivery)
	}
	return items, nil
}
