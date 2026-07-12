package pagescontroller

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Pages"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

type pagesEnvelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type pagesActors struct {
	actors map[int64]identity.Actor
}

func (s pagesActors) LoadActor(_ context.Context, userID int64) (identity.Actor, error) {
	a, ok := s.actors[userID]
	if !ok {
		return identity.Actor{}, identity.ErrUserNotFound
	}
	return a, nil
}

type pagesThemeStore struct {
	active extensions.Extension
	items  map[string]extensions.Extension
}

func (s *pagesThemeStore) Get(_ context.Context, id string) (extensions.Extension, error) {
	if e, ok := s.items[id]; ok {
		return e, nil
	}
	return extensions.Extension{}, extensions.ErrExtensionNotFound
}

func (s *pagesThemeStore) ActiveTheme(context.Context) (extensions.Extension, error) {
	if s.active.ID == "" {
		return extensions.Extension{}, extensions.ErrExtensionNotFound
	}
	return s.active, nil
}

type auditSink struct {
	events []audit.Event
}

func (a *auditSink) Append(_ context.Context, e audit.Event) error {
	a.events = append(a.events, e)
	return nil
}

type fakeRouteTargets struct {
	bases map[string]string
}

func (f fakeRouteTargets) RouteTargetBase(id string) (string, bool) {
	b, ok := f.bases[id]
	return b, ok
}

func fixtureDemoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	// Controllers/Pages → Controllers → Http → app → api → apps → repo root
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../../../"))
	return filepath.Join(root, "extensions/fixtures/plugins/page-registry-demo")
}

type pagesRouteProvider func(api fiber.Router)

func (f pagesRouteProvider) RegisterRoutes(api fiber.Router) { f(api) }

type roundTripFunc func(*nethttp.Request) (*nethttp.Response, error)

func (f roundTripFunc) Do(_ context.Context, req *nethttp.Request) (*nethttp.Response, error) {
	return f(req)
}

func newPagesTestApp(t *testing.T) (*fiber.App, *pages.Registry, *pagesThemeStore, *auditSink) {
	t.Helper()
	manager := authsession.NewManager(session.NewStore(), authsession.Config{HashSecret: "test-secret"})
	users := pagesActors{actors: map[int64]identity.Actor{
		1: {
			ID: 1, Status: identity.UserStatusActive,
			RoleKeys:    []string{identity.RoleSuperAdmin},
			Permissions: map[string]bool{identity.PermissionExtensionView: true, identity.PermissionExtensionManage: true},
		},
		2: {
			ID: 2, Status: identity.UserStatusActive,
			RoleKeys:    []string{identity.RoleMember},
			Permissions: map[string]bool{},
		},
		3: {
			ID: 3, Status: identity.UserStatusActive,
			RoleKeys:    []string{identity.RoleModerator},
			Permissions: map[string]bool{identity.PermissionModerationReview: true},
		},
		4: {
			ID: 4, Status: identity.UserStatusActive,
			RoleKeys:    []string{identity.RoleMember},
			Permissions: map[string]bool{identity.PermissionExtensionView: true},
		},
	}}
	store := pages.NewMemoryStore()
	reg := pages.NewRegistry(store)
	demoRoot := fixtureDemoRoot(t)
	themeStore := &pagesThemeStore{
		items: map[string]extensions.Extension{
			"sforum.page-registry-demo": {
				ID: "sforum.page-registry-demo", Type: extensions.TypePlugin,
				Version: "1.0.0", PackageDigest: "digest-v1", PackagePath: demoRoot,
				Status: extensions.StatusEnabled,
			},
			"demo.theme": {
				ID: "demo.theme", Type: extensions.TypeTheme,
				Version: "1.0.0", PackageDigest: "theme-d", PackagePath: demoRoot,
				Status: extensions.StatusEnabled,
			},
		},
		active: extensions.Extension{
			ID: "demo.theme", Type: extensions.TypeTheme,
			Version: "1.0.0", PackageDigest: "theme-d", PackagePath: demoRoot,
		},
	}
	bridge := pages.NewExtensionBridge(reg)
	_ = bridge.RegisterPluginPackage(context.Background(), pages.ThemeExtension{
		ID: "sforum.page-registry-demo", Version: "1.0.0",
		PackagePath: demoRoot, PackageDigest: "digest-v1",
	})

	loader := pages.NewPageDataLoader(roundTripFunc(func(r *nethttp.Request) (*nethttp.Response, error) {
		return &nethttp.Response{
			StatusCode: 200,
			Header:     nethttp.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"title":"hello","slug":"` + r.URL.Query().Get("param.slug") + `"}`)),
		}, nil
	}))
	gw := pages.NewLoaderGateway(loader, fakeRouteTargets{
		bases: map[string]string{"sforum.page-registry-demo": "http://127.0.0.1:19999"},
	})

	sink := &auditSink{}
	ctrl := NewControllerWithThemes(reg, users, manager, themeStore).
		WithAuditor(sink).
		WithLoader(gw)

	loginProvider := pagesRouteProvider(func(api fiber.Router) {
		api.Post("/test-login/:id", func(c fiber.Ctx) error {
			id := c.Params("id")
			var userID int64
			for _, ch := range id {
				userID = userID*10 + int64(ch-'0')
			}
			_, err := manager.Start(c, userID)
			return err
		})
	})
	cfg := config.Config{AppName: "SForum", AppEnv: "test", CSRFEnabled: false, AppLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{
		RouteProviders: []apphttp.RouteProvider{loginProvider, pagesRouteProvider(func(api fiber.Router) {
			ctrl.RegisterRoutes(api)
		})},
	})
	return app, reg, themeStore, sink
}

func performPages(t *testing.T, app *fiber.App, method, path string, body []byte, cookie *nethttp.Cookie) *nethttp.Response {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	// 读完 body 缓冲，避免 app.Test 连接复用时污染后续请求
	raw, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(raw))
	return resp
}

func loginPagesUser(t *testing.T, app *fiber.App, userID int64) *nethttp.Cookie {
	t.Helper()
	resp := performPages(t, app, nethttp.MethodPost, "/api/v1/test-login/"+itoa(userID), nil, nil)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("login status %d", resp.StatusCode)
	}
	for _, c := range resp.Cookies() {
		return c
	}
	t.Fatal("no session cookie")
	return nil
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func TestResolveCore(t *testing.T) {
	app, _, _, _ := newPagesTestApp(t)
	resp := performPages(t, app, nethttp.MethodGet, "/api/v1/pages/resolve?id=forum.home", nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var body pagesEnvelope[map[string]any]
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body.Data["provider"] != "core" {
		t.Fatalf("%#v", body.Data)
	}
}

func TestResolveAddedStaticAndParams(t *testing.T) {
	app, _, _, _ := newPagesTestApp(t)
	resp := performPages(t, app, nethttp.MethodGet, "/api/v1/pages/resolve-path?path=/demo-docs/hello", nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var body pagesEnvelope[map[string]any]
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body.Data["action"] != "add" {
		t.Fatalf("%#v", body.Data)
	}
	params, _ := body.Data["routeParams"].(map[string]any)
	if params["slug"] != "hello" {
		t.Fatalf("params %#v", params)
	}
}

func TestResolvePath404(t *testing.T) {
	app, _, _, _ := newPagesTestApp(t)
	resp := performPages(t, app, nethttp.MethodGet, "/api/v1/pages/resolve-path?path=/no-such-page", nil, nil)
	if resp.StatusCode != 404 {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestResolvePathLogin401(t *testing.T) {
	app, _, _, _ := newPagesTestApp(t)
	resp := performPages(t, app, nethttp.MethodGet, "/api/v1/pages/resolve-path?path=/demo-members", nil, nil)
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestResolvePathLoginOK(t *testing.T) {
	app, _, _, _ := newPagesTestApp(t)
	cookie := loginPagesUser(t, app, 2)
	resp := performPages(t, app, nethttp.MethodGet, "/api/v1/pages/resolve-path?path=/demo-members", nil, cookie)
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestApproveNonSuperAdmin403(t *testing.T) {
	app, _, _, _ := newPagesTestApp(t)
	cookie := loginPagesUser(t, app, 4)
	body := []byte(`{"extensionId":"sforum.page-registry-demo","contributionId":"sforum.page-registry-demo.home","version":"1.0.0","packageDigest":"digest-v1","contractVersion":"sforum.page.home@1"}`)
	resp := performPages(t, app, nethttp.MethodPost, "/api/v1/admin/pages/forum.home/approve", body, cookie)
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestApproveAndResolveReplace(t *testing.T) {
	app, reg, _, sink := newPagesTestApp(t)
	cookie := loginPagesUser(t, app, 1)
	body := []byte(`{"extensionId":"sforum.page-registry-demo","contributionId":"sforum.page-registry-demo.home","version":"1.0.0","packageDigest":"digest-v1","contractVersion":"sforum.page.home@1","templatePath":"templates/evil.html"}`)
	resp := performPages(t, app, nethttp.MethodPost, "/api/v1/admin/pages/forum.home/approve", body, cookie)
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("approve %d %s", resp.StatusCode, b)
	}
	if len(sink.events) == 0 {
		t.Fatal("expected audit")
	}
	rDirect, err := reg.Resolve(context.Background(), "forum.home")
	if err != nil || rDirect.Provider != "sforum.page-registry-demo" {
		t.Fatalf("registry resolve after approve: %#v err=%v", rDirect, err)
	}
	if strings.Contains(rDirect.TemplatePath, "evil") {
		t.Fatalf("forged template path stored: %s", rDirect.TemplatePath)
	}
	resp = performPages(t, app, nethttp.MethodGet, "/api/v1/pages/resolve?id=forum.home", nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("resolve status %d", resp.StatusCode)
	}
	var env pagesEnvelope[map[string]any]
	_ = json.NewDecoder(resp.Body).Decode(&env)
	if env.Data["provider"] != "sforum.page-registry-demo" {
		t.Fatalf("public resolve provider=%v fallback=%v raw=%#v", env.Data["provider"], env.Data["fallback"], env.Data)
	}
	if html, _ := env.Data["templateHtml"].(string); html == "" {
		t.Fatal("expected rendered template html")
	}
}

func TestApproveBadContract(t *testing.T) {
	app, _, _, _ := newPagesTestApp(t)
	cookie := loginPagesUser(t, app, 1)
	body := []byte(`{"extensionId":"sforum.page-registry-demo","contributionId":"sforum.page-registry-demo.home","version":"1.0.0","packageDigest":"digest-v1","contractVersion":"wrong@1"}`)
	resp := performPages(t, app, nethttp.MethodPost, "/api/v1/admin/pages/forum.home/approve", body, cookie)
	if resp.StatusCode != 422 {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestRestoreAudit(t *testing.T) {
	app, reg, _, sink := newPagesTestApp(t)
	_ = reg.ApproveReplace(context.Background(), pages.ProviderBinding{
		PageID: "forum.home", ExtensionID: "sforum.page-registry-demo",
		ContributionID: "sforum.page-registry-demo.home", Version: "1.0.0",
		PackageDigest: "digest-v1", ContractVersion: "sforum.page.home@1", ApprovedBy: 1,
	})
	cookie := loginPagesUser(t, app, 1)
	before := len(sink.events)
	resp := performPages(t, app, nethttp.MethodPost, "/api/v1/admin/pages/forum.home/restore-core", nil, cookie)
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if len(sink.events) <= before {
		t.Fatal("expected restore audit")
	}
}

func TestThemeAssetActiveOnly(t *testing.T) {
	app, _, themes, _ := newPagesTestApp(t)
	root := themes.active.PackagePath
	cssDir := filepath.Join(root, "assets")
	_ = os.MkdirAll(cssDir, 0o755)
	cssPath := filepath.Join(cssDir, "controller-test.css")
	_ = os.WriteFile(cssPath, []byte("body{color:red}"), 0o644)
	t.Cleanup(func() { _ = os.Remove(cssPath) })

	resp := performPages(t, app, nethttp.MethodGet, "/api/v1/site/theme-assets/demo.theme/assets/controller-test.css?v=theme-d", nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("active asset %d", resp.StatusCode)
	}
	resp = performPages(t, app, nethttp.MethodGet, "/api/v1/site/theme-assets/sforum.page-registry-demo/templates/docs.html", nil, nil)
	if resp.StatusCode != 404 {
		t.Fatalf("inactive should 404, got %d", resp.StatusCode)
	}
	resp = performPages(t, app, nethttp.MethodGet, "/api/v1/site/theme-assets/demo.theme/assets/controller-test.css?v=wrong", nil, nil)
	if resp.StatusCode != 404 {
		t.Fatalf("digest mismatch %d", resp.StatusCode)
	}
}

func TestGuestAccessConflictWhenLoggedIn(t *testing.T) {
	app, reg, _, _ := newPagesTestApp(t)
	_ = reg.RegisterContributions("guest.plug", []pages.PageContribution{{
		ID: "g.login", Action: pages.ActionAdd, Path: "/guest-only-page",
		Access: pages.AccessGuest, ExtensionID: "guest.plug", Version: "1", PackageDigest: "d",
		Template: "templates/docs.html",
	}})
	resp := performPages(t, app, nethttp.MethodGet, "/api/v1/pages/resolve-path?path=/guest-only-page", nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("anon guest %d", resp.StatusCode)
	}
	cookie := loginPagesUser(t, app, 2)
	resp = performPages(t, app, nethttp.MethodGet, "/api/v1/pages/resolve-path?path=/guest-only-page", nil, cookie)
	if resp.StatusCode != 409 {
		t.Fatalf("logged-in guest page should 409, got %d", resp.StatusCode)
	}
}

func TestModerationAccess(t *testing.T) {
	app, reg, _, _ := newPagesTestApp(t)
	_ = reg.RegisterContributions("mod.plug", []pages.PageContribution{{
		ID: "m.x", Action: pages.ActionAdd, Path: "/mod-only",
		Access: pages.AccessModeration, ExtensionID: "mod.plug", Version: "1", PackageDigest: "d",
	}})
	cookie := loginPagesUser(t, app, 2)
	resp := performPages(t, app, nethttp.MethodGet, "/api/v1/pages/resolve-path?path=/mod-only", nil, cookie)
	if resp.StatusCode != 403 {
		t.Fatalf("member %d", resp.StatusCode)
	}
	cookie = loginPagesUser(t, app, 3)
	resp = performPages(t, app, nethttp.MethodGet, "/api/v1/pages/resolve-path?path=/mod-only", nil, cookie)
	if resp.StatusCode != 200 {
		t.Fatalf("mod %d", resp.StatusCode)
	}
}

func TestPermissionAccess(t *testing.T) {
	app, reg, _, _ := newPagesTestApp(t)
	_ = reg.RegisterContributions("perm.plug", []pages.PageContribution{{
		ID: "p.x", Action: pages.ActionAdd, Path: "/perm-page",
		Access: pages.AccessPermission, Permission: identity.PermissionExtensionView,
		ExtensionID: "perm.plug", Version: "1", PackageDigest: "d",
	}})
	cookie := loginPagesUser(t, app, 2)
	resp := performPages(t, app, nethttp.MethodGet, "/api/v1/pages/resolve-path?path=/perm-page", nil, cookie)
	if resp.StatusCode != 403 {
		t.Fatalf("deny %d", resp.StatusCode)
	}
	cookie = loginPagesUser(t, app, 4)
	resp = performPages(t, app, nethttp.MethodGet, "/api/v1/pages/resolve-path?path=/perm-page", nil, cookie)
	if resp.StatusCode != 200 {
		t.Fatalf("allow %d", resp.StatusCode)
	}
}

func TestLoaderWiredOnResolvePath(t *testing.T) {
	app, reg, _, _ := newPagesTestApp(t)
	reg.ClearExtension("sforum.page-registry-demo")
	_ = reg.RegisterContributions("sforum.page-registry-demo", []pages.PageContribution{{
		ID: "sforum.page-registry-demo.docs", Action: pages.ActionAdd, Path: "/demo-docs/:slug",
		Template: "templates/docs.html", Access: pages.AccessPublic,
		DataSource: "plugin", DataRoute: "/docs/data",
		ExtensionID: "sforum.page-registry-demo", Version: "1.0.0", PackageDigest: "digest-v1",
		Contract: "sforum.page.plugin_docs@1",
	}})
	resp := performPages(t, app, nethttp.MethodGet, "/api/v1/pages/resolve-path?path=/demo-docs/hello", nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var env pagesEnvelope[map[string]any]
	_ = json.NewDecoder(resp.Body).Decode(&env)
	ld, _ := env.Data["loaderData"].(map[string]any)
	if ld == nil || ld["title"] != "hello" {
		t.Fatalf("loaderData %#v", env.Data["loaderData"])
	}
	if ld["slug"] != "hello" {
		t.Fatalf("slug param not passed: %#v", ld)
	}
}

func TestLoaderUnavailableWhenPluginRuntimeMissing(t *testing.T) {
	manager := authsession.NewManager(session.NewStore(), authsession.Config{HashSecret: "test-secret"})
	users := pagesActors{actors: map[int64]identity.Actor{}}
	reg := pages.NewRegistry(pages.NewMemoryStore())
	_ = reg.RegisterContributions("offline", []pages.PageContribution{{
		ID: "o.x", Action: pages.ActionAdd, Path: "/offline-docs",
		Access: pages.AccessPublic, DataSource: "plugin", DataRoute: "/d",
		ExtensionID: "offline", Version: "1", PackageDigest: "d",
	}})
	gw := pages.NewLoaderGateway(pages.NewPageDataLoader(nil), fakeRouteTargets{bases: map[string]string{}})
	ctrl := NewController(reg, users, manager).WithLoader(gw)
	cfg := config.Config{AppName: "SForum", AppEnv: "test", CSRFEnabled: false, AppLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{
		RouteProviders: []apphttp.RouteProvider{pagesRouteProvider(func(api fiber.Router) {
			ctrl.RegisterRoutes(api)
		})},
	})
	resp := performPages(t, app, nethttp.MethodGet, "/api/v1/pages/resolve-path?path=/offline-docs", nil, nil)
	var env pagesEnvelope[map[string]any]
	_ = json.NewDecoder(resp.Body).Decode(&env)
	if env.Data["loaderError"] == nil || env.Data["loaderError"] == "" {
		t.Fatalf("expected loader error: %#v", env.Data)
	}
}

func TestLoaderRejectsNonLoopbackAndRedirect(t *testing.T) {
	l := pages.NewPageDataLoader(nil)
	res := l.Fetch(context.Background(), pages.LoaderRequest{
		ExtensionID: "p", Route: "/d", TargetBase: "https://evil.example",
	})
	if !res.Fallback {
		t.Fatalf("%#v", res)
	}
	l = pages.NewPageDataLoader(roundTripFunc(func(r *nethttp.Request) (*nethttp.Response, error) {
		return &nethttp.Response{
			StatusCode: 302,
			Header:     nethttp.Header{"Location": []string{"http://127.0.0.1/x"}},
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}))
	res = l.Fetch(context.Background(), pages.LoaderRequest{
		ExtensionID: "p", Route: "/d", TargetBase: "http://127.0.0.1:1",
	})
	if res.Error == "" || !strings.Contains(res.Error, "redirect") {
		t.Fatalf("%#v", res)
	}
}



func TestThemeAssetRejectsSVGAndJS(t *testing.T) {
	app, _, themes, _ := newPagesTestApp(t)
	root := themes.active.PackagePath
	assets := filepath.Join(root, "assets")
	_ = os.MkdirAll(assets, 0o755)
	svg := filepath.Join(assets, "evil.svg")
	js := filepath.Join(assets, "evil.js")
	html := filepath.Join(assets, "evil.html")
	_ = os.WriteFile(svg, []byte("<svg xmlns='http://www.w3.org/2000/svg'></svg>"), 0o644)
	_ = os.WriteFile(js, []byte("alert(1)"), 0o644)
	_ = os.WriteFile(html, []byte("<html></html>"), 0o644)
	t.Cleanup(func() {
		_ = os.Remove(svg)
		_ = os.Remove(js)
		_ = os.Remove(html)
	})
	for _, name := range []string{"evil.svg", "evil.js", "evil.html"} {
		resp := performPages(t, app, nethttp.MethodGet, "/api/v1/site/theme-assets/demo.theme/assets/"+name+"?v=theme-d", nil, nil)
		if resp.StatusCode != 404 && resp.StatusCode != 422 && resp.StatusCode != 403 {
			t.Fatalf("%s should be rejected, got %d", name, resp.StatusCode)
		}
	}
}

func TestThemeAssetRejectsPathTraversal(t *testing.T) {
	app, _, _, _ := newPagesTestApp(t)
	resp := performPages(t, app, nethttp.MethodGet, "/api/v1/site/theme-assets/demo.theme/assets/../../etc/passwd?v=theme-d", nil, nil)
	if resp.StatusCode == 200 {
		t.Fatal("path traversal must not succeed")
	}
}

func TestLoaderDoesNotForwardAuthHeaders(t *testing.T) {
	var sawCookie, sawAuth, sawCSRF bool
	var sawActorHint bool
	manager := authsession.NewManager(session.NewStore(), authsession.Config{HashSecret: "test-secret"})
	users := pagesActors{actors: map[int64]identity.Actor{}}
	reg := pages.NewRegistry(pages.NewMemoryStore())
	_ = reg.RegisterContributions("hdr.plug", []pages.PageContribution{{
		ID: "h.x", Action: pages.ActionAdd, Path: "/hdr-docs",
		Access: pages.AccessPublic, DataSource: "plugin", DataRoute: "/d",
		ExtensionID: "hdr.plug", Version: "1", PackageDigest: "d",
	}})
	loader := pages.NewPageDataLoader(roundTripFunc(func(r *nethttp.Request) (*nethttp.Response, error) {
		if r.Header.Get("Cookie") != "" {
			sawCookie = true
		}
		if r.Header.Get("Authorization") != "" {
			sawAuth = true
		}
		if r.Header.Get("X-Csrf-Token") != "" {
			sawCSRF = true
		}
		if r.Header.Get("X-SForum-Actor-ID-Hint") != "" || r.Header.Get("X-SForum-Actor-ID") != "" {
			sawActorHint = true
		}
		return &nethttp.Response{
			StatusCode: 200,
			Header:     nethttp.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		}, nil
	}))
	gw := pages.NewLoaderGateway(loader, fakeRouteTargets{bases: map[string]string{"hdr.plug": "http://127.0.0.1:19999"}})
	ctrl := NewController(reg, users, manager).WithLoader(gw)
	cfg := config.Config{AppName: "SForum", AppEnv: "test", CSRFEnabled: false, AppLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{
		RouteProviders: []apphttp.RouteProvider{pagesRouteProvider(func(api fiber.Router) {
			ctrl.RegisterRoutes(api)
		})},
	})
	// 即便客户端带 Cookie/Authorization，loader 也不得转发
	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/pages/resolve-path?path=/hdr-docs", nil)
	req.Header.Set("Cookie", "session=secret")
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("X-Csrf-Token", "tok")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d %s", resp.StatusCode, raw)
	}
	if sawCookie || sawAuth || sawCSRF {
		t.Fatalf("loader must not forward cookie/auth/csrf: cookie=%v auth=%v csrf=%v", sawCookie, sawAuth, sawCSRF)
	}
	// actor hint 对匿名请求应为 false
	if sawActorHint {
		t.Fatalf("anonymous request should not send actor hint")
	}
}
