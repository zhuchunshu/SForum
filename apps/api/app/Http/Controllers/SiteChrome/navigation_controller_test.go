package sitechromecontroller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	sitechrome "github.com/zhuchunshu/sforum/apps/api/app/Models/SiteChrome"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

func TestPublicNavigationRejectsInvalidLocationAndUsesOptionalActor(t *testing.T) {
	app, _ := newNavigationControllerApp(identity.Actor{}, controllerNavigationStore{document: sitechrome.NavigationDocument{Revision: 4}})
	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/site/navigation?locations=not-a-location", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("invalid location status=%d", response.StatusCode)
	}
}

func TestAccountSettingsNavigationRequiresAuthentication(t *testing.T) {
	app, _ := newNavigationControllerApp(identity.Actor{}, controllerNavigationStore{document: sitechrome.NavigationDocument{Revision: 1}})
	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/site/account-navigation", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous account navigation status=%d", response.StatusCode)
	}
	actorApp, cookie := newNavigationControllerApp(identity.Actor{ID: 12, Status: identity.UserStatusActive}, controllerNavigationStore{document: sitechrome.NavigationDocument{Revision: 1}})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/site/account-navigation", nil)
	request.AddCookie(cookie)
	response, err = actorApp.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("authenticated account navigation status=%d", response.StatusCode)
	}
}

func TestPublicNavigationIsPrivateAndVariesByActor(t *testing.T) {
	document := sitechrome.NavigationDocument{
		Revision: 4,
		Definitions: []sitechrome.NavigationDefinition{{
			SourceKey: "operator.members", SourceKind: sitechrome.NavigationSourceOperator,
			LinkKind: sitechrome.NavigationLinkInternal, LabelEnUS: "Members", Href: "/members",
		}},
		Placements: []sitechrome.NavigationPlacement{{
			SourceKey: "operator.members", Location: sitechrome.NavigationLocationTopbar,
			Order: 1, Enabled: true, Visibility: sitechrome.NavigationVisibilityAuthenticated,
		}},
	}
	actor := identity.Actor{ID: 22, Status: identity.UserStatusActive}
	app, cookie := newNavigationControllerApp(actor, controllerNavigationStore{document: document})

	guestResponse, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/site/navigation?locations=public.topbar.primary", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer guestResponse.Body.Close()
	assertPrivateNavigationHeaders(t, guestResponse)
	var guestEnvelope struct {
		Data sitechrome.ResolvedNavigation `json:"data"`
	}
	if err := json.NewDecoder(guestResponse.Body).Decode(&guestEnvelope); err != nil {
		t.Fatal(err)
	}
	if resolvedNavigationHasSource(guestEnvelope.Data, "operator.members") {
		t.Fatalf("guest received authenticated item: %#v", guestEnvelope.Data)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/site/navigation?locations=public.topbar.primary", nil)
	request.AddCookie(cookie)
	memberResponse, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer memberResponse.Body.Close()
	assertPrivateNavigationHeaders(t, memberResponse)
	var memberEnvelope struct {
		Data sitechrome.ResolvedNavigation `json:"data"`
	}
	if err := json.NewDecoder(memberResponse.Body).Decode(&memberEnvelope); err != nil {
		t.Fatal(err)
	}
	if !resolvedNavigationHasSource(memberEnvelope.Data, "operator.members") {
		t.Fatalf("member navigation omitted authenticated item: %#v", memberEnvelope.Data)
	}
}

func assertPrivateNavigationHeaders(t *testing.T, response *http.Response) {
	t.Helper()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", response.StatusCode)
	}
	if got := response.Header.Get("Cache-Control"); got != "private, no-store" {
		t.Fatalf("Cache-Control=%q", got)
	}
	vary := map[string]bool{}
	for _, field := range strings.Split(response.Header.Get("Vary"), ",") {
		vary[strings.ToLower(strings.TrimSpace(field))] = true
	}
	for _, field := range []string{"cookie", "authorization", "accept-language"} {
		if !vary[field] {
			t.Fatalf("Vary=%q missing %s", response.Header.Get("Vary"), field)
		}
	}
}

func resolvedNavigationHasSource(resolved sitechrome.ResolvedNavigation, sourceKey string) bool {
	for _, location := range resolved.Locations {
		for _, item := range location.Items {
			if item.SourceKey == sourceKey {
				return true
			}
		}
	}
	return false
}

func TestAdminNavigationReadRequiresAuthenticationAndPermission(t *testing.T) {
	store := controllerNavigationStore{document: sitechrome.NavigationDocument{Revision: 4}}
	app, _ := newNavigationControllerApp(identity.Actor{}, store)
	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/admin/site/navigation", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous admin navigation status=%d", response.StatusCode)
	}

	denied, cookie := newNavigationControllerApp(identity.Actor{ID: 9, Status: identity.UserStatusActive}, store)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/site/navigation", nil)
	request.AddCookie(cookie)
	response, err = denied.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("denied admin navigation status=%d", response.StatusCode)
	}

	allowed, allowedCookie := newNavigationControllerApp(identity.Actor{ID: 10, Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionSettingsSiteManage: true}}, store)
	request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/site/navigation", nil)
	request.AddCookie(allowedCookie)
	response, err = allowed.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("allowed admin navigation status=%d", response.StatusCode)
	}
}

func TestAdminNavigationCommandsRequireSiteManagePermission(t *testing.T) {
	app, cookie := newNavigationControllerApp(identity.Actor{ID: 9, Status: identity.UserStatusActive}, controllerNavigationStore{document: sitechrome.NavigationDocument{Revision: 4}})
	for _, test := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/v1/admin/site/navigation/apply", `{}`},
		{http.MethodPost, "/api/v1/admin/site/navigation/defaults/preview", `{}`},
		{http.MethodPost, "/api/v1/admin/site/navigation/defaults/apply", `{}`},
		{http.MethodGet, "/api/v1/admin/site/navigation/snapshots", ""},
		{http.MethodGet, "/api/v1/admin/site/navigation/snapshots/1", ""},
		{http.MethodPost, "/api/v1/admin/site/navigation/snapshots/1/restore", `{}`},
		{http.MethodGet, "/api/v1/admin/site/navigation/export", ""},
		{http.MethodPost, "/api/v1/admin/site/navigation/import/preview", `{}`},
		{http.MethodPost, "/api/v1/admin/site/navigation/import/apply", `{}`},
	} {
		request := httptest.NewRequest(test.method, test.path, bytes.NewBufferString(test.body))
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(cookie)
		response, err := app.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusForbidden {
			t.Fatalf("%s %s status=%d, want forbidden", test.method, test.path, response.StatusCode)
		}
	}
}

func newNavigationControllerApp(actor identity.Actor, store controllerNavigationStore) (*fiber.App, *http.Cookie) {
	sessions := authsession.NewManager(session.NewStore(), authsession.Config{})
	controller := NewController(sitechrome.NewService(store), controllerNavigationActorStore{actor: actor}, sessions)
	app := apphttp.NewApp(config.Config{AppName: "SForum", AppEnv: "test", CSRFEnabled: false, AppLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}, nil, apphttp.Dependencies{
		RouteProviders: []apphttp.RouteProvider{
			navigationControllerRouteProvider(func(api fiber.Router) {
				api.Post("/test-login", func(c fiber.Ctx) error { _, err := sessions.Start(c, actor.ID); return err })
			}),
			controller,
		},
	})
	response, err := app.Test(httptest.NewRequest(http.MethodPost, "/api/v1/test-login", nil))
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()
	if actor.ID <= 0 || len(response.Cookies()) == 0 {
		return app, nil
	}
	return app, response.Cookies()[0]
}

type navigationControllerRouteProvider func(fiber.Router)

func (f navigationControllerRouteProvider) RegisterRoutes(api fiber.Router) { f(api) }

type controllerNavigationStore struct {
	sitechrome.Store
	document sitechrome.NavigationDocument
}

func (s controllerNavigationStore) ReadNavigationDocument(context.Context) (sitechrome.NavigationDocument, error) {
	return s.document, nil
}

type controllerNavigationActorStore struct{ actor identity.Actor }

func (s controllerNavigationActorStore) LoadActor(context.Context, int64) (identity.Actor, error) {
	return s.actor, nil
}
