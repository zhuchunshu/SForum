package extensionscontroller

import (
	"context"
	"encoding/json"
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

func TestTrustedRuntimeControllerUsesExplicitSyncAndQueuedStatuses(t *testing.T) {
	frontend := &fakeTrustedFrontendHTTPService{status: extensions.FrontendStatus{
		ExtensionID: "demo.plugin", TrustState: extensions.FrontendTrustRequired,
	}}
	releases := &fakeWebReleaseHTTPService{}
	app, manager := newTrustedRuntimeTestApp(t, frontend, releases)
	managerCookie := loginExtensionUser(t, app, manager, 2)
	superCookie := loginExtensionUser(t, app, manager, 1)

	resp := performExtensionRequest(t, app, http.MethodGet, "/api/v1/admin/extensions/demo.plugin/frontend", managerCookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected manager frontend read 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = performExtensionJSONRequest(t, app, http.MethodPost, "/api/v1/admin/extensions/demo.plugin/frontend/trust", managerCookie, `{"packageDigest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected non-super-admin grant 403, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	frontend.grant = extensions.ExtensionOperation{Frontend: &extensions.FrontendStatus{TrustState: extensions.FrontendTrustTrusted}}
	resp = performExtensionJSONRequest(t, app, http.MethodPost, "/api/v1/admin/extensions/demo.plugin/frontend/trust", superCookie, `{"packageDigest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected disabled grant 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	frontend.grant.Queued = true
	frontend.grant.WebRelease = &extensions.WebReleaseSummary{ID: 8, Status: extensions.WebReleaseQueued}
	resp = performExtensionJSONRequest(t, app, http.MethodPost, "/api/v1/admin/extensions/demo.plugin/frontend/trust", superCookie, `{"packageDigest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected enabled grant 202, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = performExtensionJSONRequest(t, app, http.MethodPost, "/api/v1/admin/extensions/demo.plugin/frontend/trust", superCookie, `{"packageDigest":"not-a-digest"}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected invalid digest 422, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	frontend.revoke = extensions.ExtensionOperation{Queued: true, WebRelease: &extensions.WebReleaseSummary{ID: 9}}
	resp = performExtensionRequest(t, app, http.MethodDelete, "/api/v1/admin/extensions/demo.plugin/frontend/trust", superCookie)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected active revoke 202, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestTrustedRuntimeControllerListsAndCommandsWebReleases(t *testing.T) {
	frontend := &fakeTrustedFrontendHTTPService{}
	releases := &fakeWebReleaseHTTPService{
		page:     extensions.WebReleasePage{Items: []extensions.WebRelease{{ID: 7}}, Total: 1, Page: 2, PerPage: 5},
		detail:   extensions.WebReleaseDetail{WebRelease: extensions.WebRelease{ID: 7}},
		rebuild:  extensions.WebReleaseOperation{WebRelease: extensions.WebRelease{ID: 11}, Queued: true},
		retry:    extensions.WebReleaseOperation{WebRelease: extensions.WebRelease{ID: 8}, Queued: true},
		rollback: extensions.WebReleaseOperation{WebRelease: extensions.WebRelease{ID: 9}, Queued: true},
	}
	app, manager := newTrustedRuntimeTestApp(t, frontend, releases)
	cookie := loginExtensionUser(t, app, manager, 2)

	resp := performExtensionRequest(t, app, http.MethodGet, "/api/v1/admin/web-releases?page=2&perPage=5&status=failed", cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected release list 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	if releases.listInput.Page != 2 || releases.listInput.PerPage != 5 || releases.listInput.Status != extensions.WebReleaseFailed {
		t.Fatalf("unexpected list input: %#v", releases.listInput)
	}

	resp = performExtensionRequest(t, app, http.MethodGet, "/api/v1/admin/web-releases/7", cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected release detail 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = performExtensionRequest(t, app, http.MethodPost, "/api/v1/admin/web-releases/rebuild", cookie)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected rebuild 202, got %d", resp.StatusCode)
	}
	var rebuildBody testEnvelope[extensions.WebReleaseOperation]
	if err := json.NewDecoder(resp.Body).Decode(&rebuildBody); err != nil {
		t.Fatalf("decode rebuild response: %v", err)
	}
	resp.Body.Close()
	if rebuildBody.Data.WebRelease.ID != 11 || !rebuildBody.Data.Queued {
		t.Fatalf("unexpected rebuild body: %#v", rebuildBody.Data)
	}

	for path, wantID := range map[string]int64{
		"/api/v1/admin/web-releases/7/retry":    8,
		"/api/v1/admin/web-releases/7/rollback": 9,
	} {
		resp = performExtensionRequest(t, app, http.MethodPost, path, cookie)
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("expected %s 202, got %d", path, resp.StatusCode)
		}
		var body testEnvelope[extensions.WebReleaseOperation]
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode %s response: %v", path, err)
		}
		resp.Body.Close()
		if body.Data.WebRelease.ID != wantID || body.Data.WebRelease.ID == 7 {
			t.Fatalf("command reused terminal release: %#v", body.Data)
		}
	}

	frontend.restore = extensions.ExtensionOperation{Queued: true, WebRelease: &extensions.WebReleaseSummary{ID: 10}}
	superCookie := loginExtensionUser(t, app, manager, 1)
	resp = performExtensionRequest(t, app, http.MethodPost, "/api/v1/admin/web-releases/restore-defaults", superCookie)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected restore defaults 202, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func newTrustedRuntimeTestApp(
	t *testing.T,
	frontend TrustedFrontendService,
	releases WebReleaseAdminService,
) (*fiber.App, *authsession.Manager) {
	t.Helper()
	manager := authsession.NewManager(session.NewStore(), authsession.Config{HashSecret: "test-secret"})
	users := controllerActors{actors: map[int64]identity.Actor{
		1: {ID: 1, Status: identity.UserStatusActive, RoleKeys: []string{identity.RoleSuperAdmin}},
		2: {ID: 2, Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionExtensionManage: true}},
	}}
	store := &controllerFakeStore{items: map[string]extensions.Extension{}}
	controller := NewController(extensions.NewService(store, "storage/extensions"), users, manager).WithTrustedRuntime(frontend, releases)
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
	status  extensions.FrontendStatus
	grant   extensions.ExtensionOperation
	revoke  extensions.ExtensionOperation
	restore extensions.ExtensionOperation
}

func (s *fakeTrustedFrontendHTTPService) Frontend(context.Context, identity.Actor, string) (extensions.FrontendStatus, error) {
	return s.status, nil
}

func (s *fakeTrustedFrontendHTTPService) Grant(_ context.Context, actor identity.Actor, _ string, _ extensions.GrantFrontendInput) (extensions.ExtensionOperation, error) {
	if !actor.IsSuperAdmin() {
		return extensions.ExtensionOperation{}, identity.ErrPermissionDenied
	}
	return s.grant, nil
}

func (s *fakeTrustedFrontendHTTPService) Revoke(_ context.Context, actor identity.Actor, _ string) (extensions.ExtensionOperation, error) {
	if !actor.IsSuperAdmin() {
		return extensions.ExtensionOperation{}, identity.ErrPermissionDenied
	}
	return s.revoke, nil
}

func (s *fakeTrustedFrontendHTTPService) RestoreDefaults(_ context.Context, actor identity.Actor) (extensions.ExtensionOperation, error) {
	if !actor.IsSuperAdmin() {
		return extensions.ExtensionOperation{}, identity.ErrPermissionDenied
	}
	return s.restore, nil
}

type fakeWebReleaseHTTPService struct {
	page      extensions.WebReleasePage
	detail    extensions.WebReleaseDetail
	rebuild   extensions.WebReleaseOperation
	retry     extensions.WebReleaseOperation
	rollback  extensions.WebReleaseOperation
	listInput extensions.WebReleaseListInput
}

func (s *fakeWebReleaseHTTPService) List(_ context.Context, _ identity.Actor, input extensions.WebReleaseListInput) (extensions.WebReleasePage, error) {
	s.listInput = input
	return s.page, nil
}

func (s *fakeWebReleaseHTTPService) Detail(context.Context, identity.Actor, int64) (extensions.WebReleaseDetail, error) {
	return s.detail, nil
}

func (s *fakeWebReleaseHTTPService) Rebuild(context.Context, identity.Actor) (extensions.WebReleaseOperation, error) {
	return s.rebuild, nil
}

func (s *fakeWebReleaseHTTPService) Retry(context.Context, identity.Actor, int64) (extensions.WebReleaseOperation, error) {
	return s.retry, nil
}

func (s *fakeWebReleaseHTTPService) Rollback(context.Context, identity.Actor, int64) (extensions.WebReleaseOperation, error) {
	return s.rollback, nil
}
