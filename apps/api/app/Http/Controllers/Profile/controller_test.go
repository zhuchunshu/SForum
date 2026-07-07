package profilecontroller

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	profile "github.com/zhuchunshu/sforum/apps/api/app/Models/Profile"
	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

type profileEnvelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func TestControllerPublicProfileRead(t *testing.T) {
	app, _, _ := newProfileTestApp()
	resp := performProfileRequest(t, app, nethttp.MethodGet, "/api/v1/profiles/alice", nil, nil)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var body profileEnvelope[profile.PublicProfile]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Data.Username != "alice" {
		t.Fatalf("expected username alice, got %q", body.Data.Username)
	}
}

func TestControllerMyProfileRequiresLogin(t *testing.T) {
	app, _, _ := newProfileTestApp()
	resp := performProfileRequest(t, app, nethttp.MethodGet, "/api/v1/profile", nil, nil)
	if resp.StatusCode != nethttp.StatusUnauthorized {
		t.Fatalf("expected 401 without login, got %d", resp.StatusCode)
	}
}

func TestControllerUpdateProfileRequiresLogin(t *testing.T) {
	app, _, _ := newProfileTestApp()
	body := []byte(`{"bio":"hello"}`)
	resp := performProfileRequest(t, app, nethttp.MethodPut, "/api/v1/profile", body, nil)
	if resp.StatusCode != nethttp.StatusUnauthorized {
		t.Fatalf("expected 401 without login, got %d", resp.StatusCode)
	}
}

func TestControllerUpdateProfileAllowsCurrentUser(t *testing.T) {
	app, _, store := newProfileTestApp()
	cookie := loginProfileUser(t, app, 7)
	body := []byte(`{"bio":"updated bio"}`)
	resp := performProfileRequest(t, app, nethttp.MethodPut, "/api/v1/profile", body, cookie)
	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200 update, got %d", resp.StatusCode)
	}
	if store.upserted.Bio != "updated bio" {
		t.Fatalf("expected upserted bio, got %q", store.upserted.Bio)
	}
	// 只能写自己：upsert 目标必须是当前登录用户。
	if store.upserted.UserID != 7 {
		t.Fatalf("expected upsert target user 7, got %d", store.upserted.UserID)
	}
}

func TestControllerUpdateProfileRejectsInvalidWebsite(t *testing.T) {
	app, _, _ := newProfileTestApp()
	cookie := loginProfileUser(t, app, 7)
	body := []byte(`{"websiteUrl":"not-a-url"}`)
	resp := performProfileRequest(t, app, nethttp.MethodPut, "/api/v1/profile", body, cookie)
	if resp.StatusCode != nethttp.StatusUnprocessableEntity {
		t.Fatalf("expected 422 invalid website, got %d", resp.StatusCode)
	}
}

func newProfileTestApp() (*fiber.App, *authsession.Manager, *profileFakeStore) {
	manager := authsession.NewManager(session.NewStore(), authsession.Config{HashSecret: "test-secret"})
	users := profileFakeActors{actors: map[int64]identity.Actor{
		7: {ID: 7, Status: identity.UserStatusActive, Permissions: map[string]bool{}},
	}}
	store := &profileFakeStore{
		user:   profile.UserProfileSummary{UserID: 7, Username: "alice", DisplayName: "Alice"},
		profile: profile.Profile{UserID: 7, Bio: "hello"},
		stats:   profile.ProfileStats{TopicCount: 3, CommentCount: 12},
	}
	controller := NewController(profile.NewService(store), users, manager)
	loginProvider := profileRouteProviderFunc(func(api fiber.Router) {
		api.Post("/test-login/:id", func(c fiber.Ctx) error {
			userID, err := strconv.ParseInt(c.Params("id"), 10, 64)
			if err != nil || userID == 0 {
				userID = 7
			}
			_, err = manager.Start(c, userID)
			return err
		})
	})
	app := apphttp.NewApp(config.Config{AppName: "SForum", AppEnv: "test", AppLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}, slog.Default(), apphttp.Dependencies{
		RouteProviders: []apphttp.RouteProvider{controller, loginProvider},
	})
	return app, manager, store
}

func loginProfileUser(t *testing.T, app *fiber.App, userID int64) *nethttp.Cookie {
	t.Helper()
	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/test-login/"+strconv.FormatInt(userID, 10), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != nethttp.StatusOK || len(resp.Cookies()) == 0 {
		t.Fatalf("expected login cookie, got %d", resp.StatusCode)
	}
	return resp.Cookies()[0]
}

func performProfileRequest(t *testing.T, app *fiber.App, method, path string, body []byte, cookie *nethttp.Cookie) *nethttp.Response {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s %s failed: %v", method, path, err)
	}
	return resp
}

type profileFakeActors struct {
	actors map[int64]identity.Actor
}

func (a profileFakeActors) LoadActor(_ context.Context, userID int64) (identity.Actor, error) {
	return a.actors[userID], nil
}

type profileRouteProviderFunc func(api fiber.Router)

func (f profileRouteProviderFunc) RegisterRoutes(api fiber.Router) { f(api) }

type profileFakeStore struct {
	user     profile.UserProfileSummary
	profile  profile.Profile
	stats    profile.ProfileStats
	recent   []forum.TopicSummary
	upserted profile.Profile
}

func (s *profileFakeStore) GetProfile(context.Context, int64) (profile.Profile, error) {
	return s.profile, nil
}

func (s *profileFakeStore) UpsertProfile(_ context.Context, input profile.Profile) (profile.Profile, error) {
	s.upserted = input
	return input, nil
}

func (s *profileFakeStore) GetUserSummaryByUsername(context.Context, string) (profile.UserProfileSummary, error) {
	return s.user, nil
}

func (s *profileFakeStore) GetUserSummaryByID(context.Context, int64) (profile.UserProfileSummary, error) {
	return s.user, nil
}

func (s *profileFakeStore) GetProfileStats(context.Context, int64) (profile.ProfileStats, error) {
	return s.stats, nil
}

func (s *profileFakeStore) ListRecentTopics(context.Context, int64, int) ([]forum.TopicSummary, error) {
	return s.recent, nil
}
