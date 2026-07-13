package extensionscontroller

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

func TestTrustedRuntimeControllerUsesImmediateStatusResponses(t *testing.T) {
	frontend := &fakeTrustedFrontendHTTPService{status: extensions.FrontendStatus{
		ExtensionID: "demo.plugin", Kind: extensions.AdminFrontendKindPrebuiltComponent,
		TrustState: extensions.FrontendTrustRequired,
	}}
	app, manager := newTrustedRuntimeTestApp(t, frontend)
	managerCookie := loginExtensionUser(t, app, manager, 2)
	superCookie := loginExtensionUser(t, app, manager, 1)

	resp := performExtensionRequest(t, app, http.MethodGet, "/api/v1/admin/extensions/demo.plugin/frontend", managerCookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("frontend status: got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = performExtensionJSONRequest(t, app, http.MethodPost, "/api/v1/admin/extensions/demo.plugin/frontend/trust", managerCookie, `{"digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("manager grant: got %d", resp.StatusCode)
	}
	resp.Body.Close()

	frontend.grant = extensions.FrontendStatus{ExtensionID: "demo.plugin", Kind: extensions.AdminFrontendKindPrebuiltComponent, TrustState: extensions.FrontendTrustTrusted}
	resp = performExtensionJSONRequest(t, app, http.MethodPost, "/api/v1/admin/extensions/demo.plugin/frontend/trust", superCookie, `{"digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("super-admin grant: got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = performExtensionJSONRequest(t, app, http.MethodPost, "/api/v1/admin/extensions/demo.plugin/frontend/trust", superCookie, `{"digest":"not-a-digest"}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("invalid digest: got %d", resp.StatusCode)
	}
	resp.Body.Close()

	frontend.revoke = extensions.FrontendStatus{ExtensionID: "demo.plugin", Kind: extensions.AdminFrontendKindPrebuiltComponent, TrustState: extensions.FrontendTrustRevoked}
	resp = performExtensionRequest(t, app, http.MethodDelete, "/api/v1/admin/extensions/demo.plugin/frontend/trust", superCookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("immediate revoke: got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = performExtensionRequest(t, app, http.MethodGet, "/api/v1/admin/web-releases", superCookie)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("removed release route must be 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestTrustedRuntimeControllerServesOnlyImmutableAdminAssets(t *testing.T) {
	frontend := &fakeTrustedFrontendHTTPService{asset: extensions.FrontendAsset{
		Body: []byte("export const apiVersion = 1\n"), ContentType: "application/javascript; charset=utf-8", ETag: `"abc"`,
	}}
	app, manager := newTrustedRuntimeTestApp(t, frontend)
	cookie := loginExtensionUser(t, app, manager, 2)
	digest := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	resp := performExtensionRequest(t, app, http.MethodGet, "/api/v1/admin/extensions/demo.plugin/frontend/assets/"+digest+"/entry", cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected asset 200, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var body bytes.Buffer
	_, _ = body.ReadFrom(resp.Body)
	if body.String() != "export const apiVersion = 1\n" || resp.Header.Get("Cache-Control") != "private, max-age=31536000, immutable" || resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("unexpected immutable response: headers=%v body=%q", resp.Header, body.String())
	}
}

func newTrustedRuntimeTestApp(t *testing.T, frontend TrustedFrontendService) (*fiber.App, *authsession.Manager) {
	t.Helper()
	manager := authsession.NewManager(session.NewStore(), authsession.Config{HashSecret: "test-secret"})
	users := controllerActors{actors: map[int64]identity.Actor{
		1: {ID: 1, Status: identity.UserStatusActive, RoleKeys: []string{identity.RoleSuperAdmin}},
		2: {ID: 2, Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionExtensionManage: true}},
	}}
	store := &controllerFakeStore{items: map[string]extensions.Extension{}}
	controller := NewController(extensions.NewService(store, "storage/extensions"), users, manager).WithTrustedRuntime(frontend)
	loginProvider := extensionRouteProviderFunc(func(api fiber.Router) {
		api.Post("/test-login/:id", func(c fiber.Ctx) error {
			id := int64(1)
			if c.Params("id") == "2" {
				id = 2
			}
			_, err := manager.Start(c, id)
			return err
		})
	})
	app := apphttp.NewApp(config.Config{
		AppName: "SForum", AppEnv: "test", CSRFEnabled: false,
		AppLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"},
	}, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{controller, loginProvider}})
	return app, manager
}

type fakeTrustedFrontendHTTPService struct {
	status    extensions.FrontendStatus
	grant     extensions.FrontendStatus
	revoke    extensions.FrontendStatus
	asset     extensions.FrontendAsset
	challenge extensions.FrontendTrustChallenge
}

func (s *fakeTrustedFrontendHTTPService) Asset(context.Context, identity.Actor, string, string, string) (extensions.FrontendAsset, error) {
	return s.asset, nil
}

func (s *fakeTrustedFrontendHTTPService) Challenge(_ context.Context, actor identity.Actor, _ string) (extensions.FrontendTrustChallenge, error) {
	if !actor.IsSuperAdmin() {
		return extensions.FrontendTrustChallenge{}, identity.ErrPermissionDenied
	}
	return s.challenge, nil
}

func (s *fakeTrustedFrontendHTTPService) Frontend(context.Context, identity.Actor, string) (extensions.FrontendStatus, error) {
	return s.status, nil
}

func (s *fakeTrustedFrontendHTTPService) Grant(_ context.Context, actor identity.Actor, _ string, _ extensions.GrantFrontendInput) (extensions.FrontendStatus, error) {
	if !actor.IsSuperAdmin() {
		return extensions.FrontendStatus{}, identity.ErrPermissionDenied
	}
	return s.grant, nil
}

func (s *fakeTrustedFrontendHTTPService) Revoke(_ context.Context, actor identity.Actor, _ string) (extensions.FrontendStatus, error) {
	if !actor.IsSuperAdmin() {
		return extensions.FrontendStatus{}, identity.ErrPermissionDenied
	}
	return s.revoke, nil
}
