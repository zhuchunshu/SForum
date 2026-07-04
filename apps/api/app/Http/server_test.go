package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	identitycontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Identity"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	humanverify "github.com/zhuchunshu/sforum/apps/api/app/Support/HumanVerify"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

type apiEnvelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type apiErrorData struct {
	Reason string `json:"reason"`
}

type healthResponse struct {
	Name             string    `json:"name"`
	Status           string    `json:"status"`
	Environment      string    `json:"environment"`
	Locale           string    `json:"locale"`
	SupportedLocales []string  `json:"supportedLocales"`
	Time             time.Time `json:"time"`
}

func TestNewAppRegistersRouteProviders(t *testing.T) {
	cfg := testConfig()
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{
		RouteProviders: []apphttp.RouteProvider{routeProviderFunc(func(api fiber.Router) {
			api.Get("/probe", func(c fiber.Ctx) error {
				return c.JSON(fiber.Map{"status": "registered"})
			})
		})},
	})

	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/probe", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("probe request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHealthEndpoint(t *testing.T) {
	cfg := config.Config{
		AppName:          "SForum",
		AppEnv:           "test",
		AppLocale:        "zh-CN",
		SupportedLocales: []string{"zh-CN", "en-US"},
	}

	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{})
	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/health", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body apiEnvelope[healthResponse]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}

	if body.Code != nethttp.StatusOK {
		t.Fatalf("expected envelope code 200, got %d", body.Code)
	}
	if body.Message != "OK" {
		t.Fatalf("expected OK message, got %q", body.Message)
	}
	if body.Data.Status != "ok" {
		t.Fatalf("expected ok status, got %q", body.Data.Status)
	}
	if body.Data.Locale != "zh-CN" {
		t.Fatalf("expected zh-CN locale, got %q", body.Data.Locale)
	}
	if len(body.Data.SupportedLocales) != 2 {
		t.Fatalf("expected two supported locales, got %v", body.Data.SupportedLocales)
	}
}

func TestRegisterEndpointCreatesSession(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	identityController := identitycontroller.NewController(identity.NewService(store), session.NewStore())
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})

	requestBody := []byte(`{"username":"admin","email":"admin@example.com","password":"correct horse battery staple","displayName":"Admin","locale":"zh-CN"}`)
	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/auth/register", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("register request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	if cookies := resp.Cookies(); len(cookies) == 0 {
		t.Fatal("expected session cookie")
	}
	var body apiEnvelope[identity.CurrentUser]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if body.Code != nethttp.StatusCreated {
		t.Fatalf("expected envelope code 201, got %d", body.Code)
	}
	if body.Message != "OK" {
		t.Fatalf("expected OK message, got %q", body.Message)
	}
	if body.Data.Username != "admin" {
		t.Fatalf("expected admin username, got %q", body.Data.Username)
	}
	if body.Data.Locale != "zh-CN" {
		t.Fatalf("expected zh-CN locale, got %q", body.Data.Locale)
	}
}

func TestRegisterEndpointRequiresHumanVerification(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	verifier := humanverify.NewService(
		humanverify.ServiceConfig{Enabled: true, ChallengeTTL: time.Minute},
		httpFakeHumanProvider{result: humanverify.VerifyResult{Verified: true, Code: humanverify.CodeOK}},
		humanverify.NewMemoryStore(),
	)
	identityController := identitycontroller.NewControllerWithVerifier(identity.NewService(store), session.NewStore(), verifier)
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})

	requestBody := []byte(`{"username":"admin","email":"admin@example.com","password":"correct horse battery staple","displayName":"Admin","locale":"zh-CN"}`)
	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/auth/register", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("register request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
	var body apiEnvelope[apiErrorData]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if body.Code != nethttp.StatusUnprocessableEntity {
		t.Fatalf("expected envelope code 422, got %d", body.Code)
	}
	if body.Message != "请先完成人机验证。" {
		t.Fatalf("expected human verification message, got %q", body.Message)
	}
	if body.Data.Reason != humanverify.CodeRequired {
		t.Fatalf("expected required reason, got %q", body.Data.Reason)
	}
}

func TestRegisterEndpointAcceptsHumanVerificationToken(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	verifier := humanverify.NewService(
		humanverify.ServiceConfig{Enabled: true, ChallengeTTL: time.Minute},
		httpFakeHumanProvider{result: humanverify.VerifyResult{Verified: true, Code: humanverify.CodeOK}},
		humanverify.NewMemoryStore(),
	)
	identityController := identitycontroller.NewControllerWithVerifier(identity.NewService(store), session.NewStore(), verifier)
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})

	requestBody := []byte(`{"username":"admin","email":"admin@example.com","password":"correct horse battery staple","displayName":"Admin","locale":"zh-CN","humanVerification":{"provider":"altcha","token":"valid-token"}}`)
	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/auth/register", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("register request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	if cookies := resp.Cookies(); len(cookies) == 0 {
		t.Fatal("expected session cookie")
	}
	var body apiEnvelope[identity.CurrentUser]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if body.Code != nethttp.StatusCreated {
		t.Fatalf("expected envelope code 201, got %d", body.Code)
	}
	if body.Message != "OK" {
		t.Fatalf("expected OK message, got %q", body.Message)
	}
	if body.Data.Username != "admin" {
		t.Fatalf("expected admin username, got %q", body.Data.Username)
	}
	if body.Data.Locale != "zh-CN" {
		t.Fatalf("expected zh-CN locale, got %q", body.Data.Locale)
	}
}

func TestHumanVerificationChallengeEndpoint(t *testing.T) {
	cfg := testConfig()
	verifier := humanverify.NewService(
		humanverify.ServiceConfig{Enabled: true, ChallengeTTL: time.Minute},
		httpFakeHumanProvider{challenge: map[string]any{"challenge": "fake"}},
		humanverify.NewMemoryStore(),
	)
	identityController := identitycontroller.NewControllerWithVerifier(identity.NewService(newHTTPFakeStore()), session.NewStore(), verifier)
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})

	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/human-verification/challenge?purpose=register", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("challenge request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body apiEnvelope[map[string]string]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode challenge response: %v", err)
	}
	if body.Code != nethttp.StatusOK || body.Message != "OK" {
		t.Fatalf("unexpected envelope: %#v", body)
	}
	if body.Data["challenge"] != "fake" {
		t.Fatalf("expected fake challenge, got %v", body.Data)
	}
}

func TestSessionEndpointRequiresAuth(t *testing.T) {
	cfg := testConfig()
	identityController := identitycontroller.NewController(identity.NewService(newHTTPFakeStore()), session.NewStore())
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})
	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/auth/session", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("session request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestLogoutEndpointReturnsNoDataEnvelope(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	identityController := identitycontroller.NewController(identity.NewService(store), session.NewStore())
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})
	cookie := registerHTTPUser(t, app, "admin", "admin@example.com")

	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(cookie)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("logout request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body apiEnvelope[json.RawMessage]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode logout envelope: %v", err)
	}
	if body.Code != nethttp.StatusOK || body.Message != "OK" {
		t.Fatalf("unexpected envelope: %#v", body)
	}
	if string(body.Data) != "null" {
		t.Fatalf("expected null data, got %s", string(body.Data))
	}
}

func TestSessionEndpointReturnsLocalizedEnvelopeError(t *testing.T) {
	cfg := testConfig()
	identityController := identitycontroller.NewController(identity.NewService(newHTTPFakeStore()), session.NewStore())
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})
	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/auth/session", nil)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("session request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}

	var body apiEnvelope[apiErrorData]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if body.Code != nethttp.StatusUnauthorized {
		t.Fatalf("expected envelope code 401, got %d", body.Code)
	}
	if body.Message != "Please sign in first." {
		t.Fatalf("expected localized message, got %q", body.Message)
	}
	if body.Data.Reason != "auth.required" {
		t.Fatalf("expected auth.required reason, got %q", body.Data.Reason)
	}
}

func TestErrorEnvelopeFallsBackToDefaultLocale(t *testing.T) {
	cfg := testConfig()
	identityController := identitycontroller.NewController(identity.NewService(newHTTPFakeStore()), session.NewStore())
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})
	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/auth/session", nil)
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("session request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}

	var body apiEnvelope[apiErrorData]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if body.Message != "请先登录。" {
		t.Fatalf("expected Chinese fallback message, got %q", body.Message)
	}
}

func TestMissingRouteReturnsNotFoundEnvelope(t *testing.T) {
	cfg := testConfig()
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{})
	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/missing-route", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("missing route request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	var body apiEnvelope[apiErrorData]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if body.Code != nethttp.StatusNotFound {
		t.Fatalf("expected envelope code 404, got %d", body.Code)
	}
	if body.Data.Reason != "not_found" {
		t.Fatalf("expected not_found reason, got %q", body.Data.Reason)
	}
	if body.Message != "请求的资源不存在。" {
		t.Fatalf("expected not found message, got %q", body.Message)
	}
}

func TestHealthEndpointRejectsUnsupportedMethod(t *testing.T) {
	cfg := testConfig()
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{})
	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/health", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("health method request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.StatusCode)
	}

	var body apiEnvelope[apiErrorData]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if body.Code != nethttp.StatusMethodNotAllowed {
		t.Fatalf("expected envelope code 405, got %d", body.Code)
	}
	if body.Data.Reason != "method_not_allowed" {
		t.Fatalf("expected method_not_allowed reason, got %q", body.Data.Reason)
	}
	if body.Message != "不支持当前请求方法。" {
		t.Fatalf("expected method not allowed message, got %q", body.Message)
	}
}

func TestRolesEndpointRequiresAuth(t *testing.T) {
	cfg := testConfig()
	identityController := identitycontroller.NewController(identity.NewService(newHTTPFakeStore()), session.NewStore())
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})
	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/roles", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("roles request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestCreateRoleEndpointAllowsSuperAdmin(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	identityController := identitycontroller.NewController(identity.NewService(store), session.NewStore())
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})
	adminCookie := registerHTTPUser(t, app, "admin", "admin@example.com")

	requestBody := []byte(`{"key":"moderator","alias":"版主","description":"管理内容"}`)
	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/roles", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(adminCookie)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("create role request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var body apiEnvelope[identity.Role]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode role response: %v", err)
	}
	if body.Code != nethttp.StatusCreated || body.Message != "OK" {
		t.Fatalf("unexpected envelope: %#v", body)
	}
	if body.Data.Key != "moderator" || body.Data.Alias != "版主" {
		t.Fatalf("unexpected role response: %#v", body.Data)
	}
}

func TestCreateRoleEndpointRejectsMember(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	identityController := identitycontroller.NewController(identity.NewService(store), session.NewStore())
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})
	registerHTTPUser(t, app, "admin", "admin@example.com")
	memberCookie := registerHTTPUser(t, app, "member1", "member1@example.com")

	requestBody := []byte(`{"key":"moderator","alias":"版主","description":"管理内容"}`)
	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/roles", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(memberCookie)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("create role request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}

	var body apiEnvelope[apiErrorData]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if body.Code != nethttp.StatusForbidden {
		t.Fatalf("expected envelope code 403, got %d", body.Code)
	}
	if body.Message != "没有权限执行此操作。" {
		t.Fatalf("expected permission denied message, got %q", body.Message)
	}
	if body.Data.Reason != "permission.denied" {
		t.Fatalf("expected permission.denied reason, got %q", body.Data.Reason)
	}
}

func testConfig() config.Config {
	return config.Config{
		AppName:          "SForum",
		AppEnv:           "test",
		AppLocale:        "zh-CN",
		SupportedLocales: []string{"zh-CN", "en-US"},
	}
}

func registerHTTPUser(t *testing.T, app *fiber.App, username, email string) *nethttp.Cookie {
	t.Helper()

	body := []byte(fmt.Sprintf(
		`{"username":%q,"email":%q,"password":"correct horse battery staple","displayName":%q,"locale":"zh-CN"}`,
		username,
		email,
		username,
	))
	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("register request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusCreated {
		t.Fatalf("expected register 201, got %d", resp.StatusCode)
	}
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected register session cookie")
	}
	return cookies[0]
}

type httpFakeStore struct {
	nextUserID  int64
	nextRoleID  int64
	users       map[int64]identity.CurrentUser
	credentials map[int64]string
	roles       map[string]identity.Role
	userRoleIDs map[int64][]int64
}

func newHTTPFakeStore() *httpFakeStore {
	return &httpFakeStore{
		nextUserID:  1,
		nextRoleID:  100,
		users:       map[int64]identity.CurrentUser{},
		credentials: map[int64]string{},
		roles: map[string]identity.Role{
			identity.RoleSuperAdmin: {ID: 1, Key: identity.RoleSuperAdmin, Alias: "超级管理员", IsEnabled: true},
			identity.RoleMember:     {ID: 2, Key: identity.RoleMember, Alias: "普通会员", IsDefault: true, IsEnabled: true},
		},
		userRoleIDs: map[int64][]int64{},
	}
}

func (s *httpFakeStore) WithBootstrapTx(ctx context.Context, fn func(context.Context, identity.TxStore) error) error {
	return fn(ctx, s)
}

func (s *httpFakeStore) AnyUserExists(context.Context) (bool, error) {
	return len(s.users) > 0, nil
}

func (s *httpFakeStore) CreateUser(_ context.Context, input identity.CreateUserInput) (identity.CurrentUser, error) {
	user := identity.CurrentUser{
		ID:                  s.nextUserID,
		Username:            input.Username,
		DisplayName:         input.DisplayName,
		Locale:              input.Locale,
		Status:              identity.UserStatusActive,
		IsInitialSuperAdmin: input.IsInitialSuperAdmin,
	}
	s.nextUserID++
	s.users[user.ID] = user
	return user, nil
}

func (s *httpFakeStore) CreateCredential(_ context.Context, userID int64, passwordHash string) error {
	s.credentials[userID] = passwordHash
	return nil
}

func (s *httpFakeStore) GetDefaultRole(context.Context) (identity.Role, error) {
	return s.roles[identity.RoleMember], nil
}

func (s *httpFakeStore) GetRole(_ context.Context, roleKey string) (identity.Role, error) {
	return s.roles[roleKey], nil
}

func (s *httpFakeStore) AssignRole(_ context.Context, userID int64, roleID int64) error {
	s.userRoleIDs[userID] = append(s.userRoleIDs[userID], roleID)
	return nil
}

func (s *httpFakeStore) GetCurrentUser(_ context.Context, userID int64) (identity.CurrentUser, error) {
	user, ok := s.users[userID]
	if !ok {
		return identity.CurrentUser{}, errors.New("user not found")
	}
	return s.withAccess(user), nil
}

func (s *httpFakeStore) GetCredentialByLogin(context.Context, string) (identity.CredentialUser, error) {
	return identity.CredentialUser{}, errors.New("not implemented")
}

func (s *httpFakeStore) LoadActor(ctx context.Context, userID int64) (identity.Actor, error) {
	user, err := s.GetCurrentUser(ctx, userID)
	if err != nil {
		return identity.Actor{}, err
	}
	permissions := map[string]bool{}
	for _, permission := range user.Permissions {
		permissions[permission] = true
	}
	return identity.Actor{ID: user.ID, Status: user.Status, RoleKeys: user.RoleKeys, Permissions: permissions}, nil
}

func (s *httpFakeStore) ListRoles(context.Context) ([]identity.Role, error) {
	roles := make([]identity.Role, 0, len(s.roles))
	for _, role := range s.roles {
		roles = append(roles, role)
	}
	return roles, nil
}

func (s *httpFakeStore) CreateRole(_ context.Context, input identity.RoleInput) (identity.Role, error) {
	role := identity.Role{
		ID:          s.nextRoleID,
		Key:         input.Key,
		Alias:       input.Alias,
		Description: input.Description,
		IsDeletable: true,
		IsEnabled:   true,
	}
	s.nextRoleID++
	s.roles[role.Key] = role
	return role, nil
}

func (s *httpFakeStore) UpdateRole(_ context.Context, roleKey string, input identity.RoleInput) (identity.Role, error) {
	role, ok := s.roles[roleKey]
	if !ok {
		return identity.Role{}, errors.New("role not found")
	}
	role.Alias = input.Alias
	role.Description = input.Description
	s.roles[roleKey] = role
	return role, nil
}

func (s *httpFakeStore) DeleteRole(_ context.Context, roleKey string) error {
	delete(s.roles, roleKey)
	return nil
}

func (s *httpFakeStore) ReplaceRolePermissions(context.Context, int64, string, []string) error {
	return nil
}

func (s *httpFakeStore) withAccess(user identity.CurrentUser) identity.CurrentUser {
	roleIDs := s.userRoleIDs[user.ID]
	roleKeys := make([]string, 0, len(roleIDs))
	permissions := map[string]bool{}
	for _, roleID := range roleIDs {
		role, ok := s.roleByID(roleID)
		if !ok {
			continue
		}
		roleKeys = append(roleKeys, role.Key)
		if role.Key == identity.RoleSuperAdmin {
			permissions[identity.PermissionAdminAccess] = true
			permissions[identity.PermissionRoleManage] = true
		}
	}

	user.RoleKeys = roleKeys
	user.Permissions = make([]string, 0, len(permissions))
	for permission := range permissions {
		user.Permissions = append(user.Permissions, permission)
	}
	return user
}

func (s *httpFakeStore) roleByID(roleID int64) (identity.Role, bool) {
	for _, role := range s.roles {
		if role.ID == roleID {
			return role, true
		}
	}
	return identity.Role{}, false
}

type routeProviderFunc func(api fiber.Router)

func (fn routeProviderFunc) RegisterRoutes(api fiber.Router) {
	fn(api)
}

type httpFakeHumanProvider struct {
	challenge map[string]any
	result    humanverify.VerifyResult
}

func (p httpFakeHumanProvider) Challenge(context.Context, humanverify.Purpose, humanverify.Subject) (humanverify.Challenge, error) {
	payload := p.challenge
	if payload == nil {
		payload = map[string]any{"challenge": "fake"}
	}
	return humanverify.Challenge{
		Provider: humanverify.ProviderAltcha,
		Purpose:  humanverify.PurposeRegister,
		Payload:  payload,
	}, nil
}

func (p httpFakeHumanProvider) Verify(context.Context, humanverify.VerifyRequest) (humanverify.VerifyResult, error) {
	if p.result.Code == "" {
		return humanverify.VerifyResult{Verified: true, Code: humanverify.CodeOK}, nil
	}
	return p.result, nil
}
