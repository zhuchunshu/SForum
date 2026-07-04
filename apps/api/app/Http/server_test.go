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
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	identitycontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Identity"
	optionscontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Options"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	humanverify "github.com/zhuchunshu/sforum/apps/api/app/Support/HumanVerify"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

type apiEnvelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type apiErrorData struct {
	Reason string              `json:"reason"`
	Fields map[string][]string `json:"fields"`
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
	assertFieldMessage(t, body.Data.Fields, identity.FieldHumanVerification, "请先完成人机验证。")
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

func TestRegisterEndpointDoesNotConsumeHumanVerificationForInvalidFields(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	provider := &countingHumanProvider{result: humanverify.VerifyResult{Verified: true, Code: humanverify.CodeOK}}
	verifier := humanverify.NewService(
		humanverify.ServiceConfig{Enabled: true, ChallengeTTL: time.Minute},
		provider,
		humanverify.NewMemoryStore(),
	)
	identityController := identitycontroller.NewControllerWithVerifier(identity.NewService(store), session.NewStore(), verifier)
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})

	requestBody := []byte(`{"username":" ","email":"not-an-email","password":"short","displayName":"Admin","locale":"zh-CN","humanVerification":{"provider":"altcha","token":"valid-token"}}`)
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
	if provider.verifies != 0 {
		t.Fatalf("expected verifier not to run for invalid fields, got %d calls", provider.verifies)
	}
}

func TestRegisterEndpointDoesNotConsumeHumanVerificationForDuplicateIdentity(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	if _, err := identity.NewService(store).Register(context.Background(), identity.RegisterInput{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "correct horse battery staple",
	}); err != nil {
		t.Fatalf("seed register returned error: %v", err)
	}

	provider := &countingHumanProvider{result: humanverify.VerifyResult{Verified: true, Code: humanverify.CodeOK}}
	verifier := humanverify.NewService(
		humanverify.ServiceConfig{Enabled: true, ChallengeTTL: time.Minute},
		provider,
		humanverify.NewMemoryStore(),
	)
	identityController := identitycontroller.NewControllerWithVerifier(identity.NewService(store), session.NewStore(), verifier)
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})

	requestBody := []byte(`{"username":"ADMIN","email":"ADMIN@example.com","password":"correct horse battery staple","displayName":"Admin","locale":"zh-CN","humanVerification":{"provider":"altcha","token":"valid-token"}}`)
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
	if provider.verifies != 0 {
		t.Fatalf("expected verifier not to run for duplicate identity, got %d calls", provider.verifies)
	}
}

func TestRegisterEndpointMapsSessionSaveFailure(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	sessionStore := session.NewStore(session.Config{Storage: failingSessionStorage{err: errors.New("session storage unavailable")}})
	identityController := identitycontroller.NewController(identity.NewService(store), sessionStore)
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})

	requestBody := []byte(`{"username":"admin","email":"admin@example.com","password":"correct horse battery staple","displayName":"Admin","locale":"zh-CN"}`)
	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/auth/register", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("register request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.StatusCode)
	}
	var body apiEnvelope[apiErrorData]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if body.Data.Reason != identity.CodeSessionUnavailable {
		t.Fatalf("expected session unavailable reason, got %q", body.Data.Reason)
	}
	if body.Message != "账号已创建，但自动登录失败，请直接登录。" {
		t.Fatalf("expected session unavailable message, got %q", body.Message)
	}
	if len(store.users) != 1 {
		t.Fatalf("expected account to be created before session failure, got %d users", len(store.users))
	}
}

func TestRegisterEndpointMapsAuditFailureAfterAccountCreated(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	store.loginAuditErr = errors.New("audit unavailable")
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

	if resp.StatusCode != nethttp.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.StatusCode)
	}
	var body apiEnvelope[apiErrorData]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if body.Data.Reason != identity.CodeSessionUnavailable {
		t.Fatalf("expected session unavailable reason, got %q", body.Data.Reason)
	}
	if body.Message != "账号已创建，但自动登录失败，请直接登录。" {
		t.Fatalf("expected session unavailable message, got %q", body.Message)
	}
	if len(store.users) != 1 {
		t.Fatalf("expected account to be created before audit failure, got %d users", len(store.users))
	}
}

func TestRegisterEndpointReturnsFieldErrors(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	identityController := identitycontroller.NewController(identity.NewService(store), session.NewStore())
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})

	requestBody := []byte(`{"username":" ","email":"not-an-email","password":"short","displayName":"Admin","locale":"zh-CN"}`)
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
	if body.Data.Reason != identity.CodeRegisterInvalid {
		t.Fatalf("expected register invalid reason, got %q", body.Data.Reason)
	}
	if body.Message != "注册失败：请按标出的提示修改后再提交。" {
		t.Fatalf("expected actionable register message, got %q", body.Message)
	}
	assertFieldMessage(t, body.Data.Fields, identity.FieldUsername, "请填写用户名。")
	assertFieldMessage(t, body.Data.Fields, identity.FieldEmail, "邮箱格式不正确，请填写可接收邮件的地址。")
	assertFieldMessage(t, body.Data.Fields, identity.FieldPassword, "密码至少需要 12 个字符。")
}

func TestRegisterEndpointReturnsDuplicateFieldErrors(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	identityController := identitycontroller.NewController(identity.NewService(store), session.NewStore())
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})

	registerHTTPUser(t, app, "admin", "admin@example.com")

	requestBody := []byte(`{"username":"ADMIN","email":"ADMIN@example.com","password":"correct horse battery staple","displayName":"Admin","locale":"zh-CN"}`)
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
	if body.Data.Reason != identity.CodeRegisterInvalid {
		t.Fatalf("expected register invalid reason, got %q", body.Data.Reason)
	}
	assertFieldMessage(t, body.Data.Fields, identity.FieldUsername, "这个用户名已被使用，请换一个。")
	assertFieldMessage(t, body.Data.Fields, identity.FieldEmail, "这个邮箱已经注册过，请直接登录或换一个邮箱。")
}

func TestLoginEndpointReturnsActionableInvalidCredentials(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	identityController := identitycontroller.NewController(identity.NewService(store), session.NewStore())
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})

	registerHTTPUser(t, app, "admin", "admin@example.com")

	requestBody := []byte(`{"login":"admin","password":"wrong password"}`)
	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/auth/login", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
	var body apiEnvelope[apiErrorData]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if body.Data.Reason != "auth.invalid_credentials" {
		t.Fatalf("expected invalid credentials reason, got %q", body.Data.Reason)
	}
	if body.Message != "登录失败：请检查用户名/邮箱和密码；如果还没有账号，请先注册。" {
		t.Fatalf("expected actionable login message, got %q", body.Message)
	}
}

func TestLoginEndpointRecordsAudit(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	identityController := identitycontroller.NewController(identity.NewService(store), session.NewStore())
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})

	registerHTTPUser(t, app, "admin", "admin@example.com")

	requestBody := []byte(`{"login":"admin","password":"correct horse battery staple"}`)
	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/auth/login", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SForum Test Browser")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(store.loginAudits) != 2 {
		t.Fatalf("expected register and login audits, got %#v", store.loginAudits)
	}
	audit := store.loginAudits[1]
	if audit.Action != identity.AuditActionLogin {
		t.Fatalf("expected login audit action, got %#v", audit)
	}
	if audit.UserID == 0 || audit.IPAddress == "" || audit.UserAgent != "SForum Test Browser" || audit.SessionHash == "" {
		t.Fatalf("expected populated login audit, got %#v", audit)
	}
}

func TestRegistrationStatusEndpointTracksBootstrapUser(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	identityController := identitycontroller.NewController(identity.NewService(store), session.NewStore())
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})

	assertRegistrationStatus(t, app, true)
	registerHTTPUser(t, app, "admin", "admin@example.com")
	assertRegistrationStatus(t, app, false)
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
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode challenge response: %v", err)
	}
	if _, ok := body["code"]; ok {
		t.Fatalf("challenge endpoint should return raw ALTCHA payload, got envelope-like response: %#v", body)
	}
	if body["challenge"] != "fake" {
		t.Fatalf("expected fake challenge, got %v", body)
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

func TestPermissionsEndpointRequiresAuth(t *testing.T) {
	cfg := testConfig()
	identityController := identitycontroller.NewController(identity.NewService(newHTTPFakeStore()), session.NewStore())
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})
	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/permissions", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("permissions request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestPermissionMatrixEndpointAllowsSuperAdmin(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	identityController := identitycontroller.NewController(identity.NewService(store), session.NewStore())
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})
	adminCookie := registerHTTPUser(t, app, "admin", "admin@example.com")

	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/permissions/matrix", nil)
	req.AddCookie(adminCookie)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("permission matrix request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body apiEnvelope[identity.PermissionMatrix]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode permission matrix response: %v", err)
	}
	if len(body.Data.Permissions) == 0 || len(body.Data.Roles) == 0 {
		t.Fatalf("expected permissions and roles in matrix, got %#v", body.Data)
	}
}

func TestUsersEndpointRejectsMember(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	identityController := identitycontroller.NewController(identity.NewService(store), session.NewStore())
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})
	registerHTTPUser(t, app, "admin", "admin@example.com")
	memberCookie := registerHTTPUser(t, app, "member1", "member1@example.com")

	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/users", nil)
	req.AddCookie(memberCookie)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("users request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestReplaceUserRolesEndpointAllowsSuperAdmin(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	store.roles["moderator"] = identity.Role{ID: 3, Key: "moderator", Alias: "版主", IsEnabled: true, IsDeletable: true}
	identityController := identitycontroller.NewController(identity.NewService(store), session.NewStore())
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})
	adminCookie := registerHTTPUser(t, app, "admin", "admin@example.com")
	registerHTTPUser(t, app, "member1", "member1@example.com")

	requestBody := []byte(`{"roleKeys":["member","moderator"]}`)
	req := httptest.NewRequest(nethttp.MethodPut, "/api/v1/users/2/roles", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(adminCookie)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("replace user roles request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body apiEnvelope[identity.AdminUserDetail]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode user roles response: %v", err)
	}
	if !slices.Contains(body.Data.RoleKeys, "moderator") {
		t.Fatalf("expected moderator role in response, got %v", body.Data.RoleKeys)
	}
}

func TestReplaceUserPermissionOverridesEndpointAllowsSuperAdmin(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	identityController := identitycontroller.NewController(identity.NewService(store), session.NewStore())
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})
	adminCookie := registerHTTPUser(t, app, "admin", "admin@example.com")
	registerHTTPUser(t, app, "member1", "member1@example.com")

	requestBody := []byte(`{"allow":["admin.access"],"deny":["topic.create"]}`)
	req := httptest.NewRequest(nethttp.MethodPut, "/api/v1/users/2/permission-overrides", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(adminCookie)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("replace user permission overrides request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body apiEnvelope[identity.AdminUserDetail]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode user permission overrides response: %v", err)
	}
	if !slices.Contains(body.Data.PermissionOverrides.Allow, identity.PermissionAdminAccess) {
		t.Fatalf("expected admin access direct allow, got %#v", body.Data.PermissionOverrides)
	}
	if !slices.Contains(body.Data.Permissions, identity.PermissionAdminAccess) || slices.Contains(body.Data.Permissions, identity.PermissionTopicCreate) {
		t.Fatalf("expected effective permissions to reflect allow/deny, got %v", body.Data.Permissions)
	}
}

func TestInitialSuperAdminRoleLockEndpointReturnsConflict(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	identityController := identitycontroller.NewController(identity.NewService(store), session.NewStore())
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})
	adminCookie := registerHTTPUser(t, app, "admin", "admin@example.com")

	req := httptest.NewRequest(nethttp.MethodPut, "/api/v1/users/1/roles", bytes.NewReader([]byte(`{"roleKeys":["member"]}`)))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(adminCookie)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("replace initial super admin roles request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	var body apiEnvelope[apiErrorData]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode initial super admin lock response: %v", err)
	}
	if body.Data.Reason != "user.initial_super_admin_locked" {
		t.Fatalf("expected initial super admin lock reason, got %q", body.Data.Reason)
	}
}

func TestUserPermissionOverrideConflictEndpointReturnsUnprocessable(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	identityController := identitycontroller.NewController(identity.NewService(store), session.NewStore())
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController}})
	adminCookie := registerHTTPUser(t, app, "admin", "admin@example.com")
	registerHTTPUser(t, app, "member1", "member1@example.com")

	req := httptest.NewRequest(nethttp.MethodPut, "/api/v1/users/2/permission-overrides", bytes.NewReader([]byte(`{"allow":["post.create"],"deny":["post.create"]}`)))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(adminCookie)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("replace conflicting user permission overrides request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
	var body apiEnvelope[apiErrorData]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode permission override conflict response: %v", err)
	}
	if body.Data.Reason != "permission.override_conflict" {
		t.Fatalf("expected permission override conflict reason, got %q", body.Data.Reason)
	}
}

func TestWebOptionsEndpointReturnsPublicOptions(t *testing.T) {
	cfg := testConfig()
	optionStore := newHTTPFakeOptionStore()
	optionsController := optionscontroller.NewControllerWithStore(options.NewService(optionStore), newHTTPFakeStore(), session.NewStore())
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{optionsController}})

	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/web-options", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("web options request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body apiEnvelope[[]options.Option]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode web options response: %v", err)
	}
	if len(body.Data) != 5 {
		t.Fatalf("unexpected web options response: %#v", body.Data)
	}
	if optionValue(body.Data, options.NameSiteName) != "SForum" {
		t.Fatalf("expected site name in public options, got %#v", body.Data)
	}
	if optionValue(body.Data, options.NameAltchaSecret) != "" {
		t.Fatalf("public web options should not expose altcha secret: %#v", body.Data)
	}
}

func TestWebOptionsUpdateRequiresAuth(t *testing.T) {
	cfg := testConfig()
	optionStore := newHTTPFakeOptionStore()
	optionsController := optionscontroller.NewControllerWithStore(options.NewService(optionStore), newHTTPFakeStore(), session.NewStore())
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{optionsController}})

	req := httptest.NewRequest(nethttp.MethodPut, "/api/v1/web-options", bytes.NewReader([]byte(`{"name":"site.name","value":"Example Forum"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("update web option request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestWebOptionsUpdateAllowsSuperAdmin(t *testing.T) {
	cfg := testConfig()
	userStore := newHTTPFakeStore()
	optionStore := newHTTPFakeOptionStore()
	sessionStore := session.NewStore()
	identityController := identitycontroller.NewController(identity.NewService(userStore), sessionStore)
	optionsController := optionscontroller.NewControllerWithStore(options.NewService(optionStore), userStore, sessionStore)
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController, optionsController}})
	adminCookie := registerHTTPUser(t, app, "admin", "admin@example.com")

	req := httptest.NewRequest(nethttp.MethodPut, "/api/v1/web-options", bytes.NewReader([]byte(`{"name":"site.name","value":"Example Forum"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(adminCookie)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("update web option request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body apiEnvelope[options.Option]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode web option response: %v", err)
	}
	if body.Data.Name != options.NameSiteName || body.Data.Value != "Example Forum" {
		t.Fatalf("unexpected updated web option: %#v", body.Data)
	}
}

func TestAdminWebOptionsRequireAuth(t *testing.T) {
	cfg := testConfig()
	optionStore := newHTTPFakeOptionStore()
	optionsController := optionscontroller.NewControllerWithStore(options.NewService(optionStore), newHTTPFakeStore(), session.NewStore())
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{optionsController}})

	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/admin/web-options", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("admin web options request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAdminWebOptionsMaskSecretAndSaveBatch(t *testing.T) {
	cfg := testConfig()
	userStore := newHTTPFakeStore()
	optionStore := newHTTPFakeOptionStore()
	optionStore.items[options.NameAltchaSecret] = "existing-secret"
	sessionStore := session.NewStore()
	identityController := identitycontroller.NewController(identity.NewService(userStore), sessionStore)
	optionsController := optionscontroller.NewControllerWithStore(options.NewService(optionStore), userStore, sessionStore)
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{RouteProviders: []apphttp.RouteProvider{identityController, optionsController}})
	adminCookie := registerHTTPUser(t, app, "admin", "admin@example.com")

	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/admin/web-options", nil)
	req.AddCookie(adminCookie)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("admin web options request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var listBody apiEnvelope[[]options.AdminOption]
	if err := json.NewDecoder(resp.Body).Decode(&listBody); err != nil {
		t.Fatalf("decode admin options response: %v", err)
	}
	secret := adminOption(listBody.Data, options.NameAltchaSecret)
	if !secret.Secret || !secret.SecretSet || secret.Value != "" {
		t.Fatalf("expected masked secret in admin response, got %#v", secret)
	}

	body := []byte(`{"options":[{"name":"site.name","value":"Example Forum"},{"name":"site.default_locale","value":"en"},{"name":"site.supported_locales","value":"en-US"},{"name":"human_verification.provider","value":"altcha"},{"name":"human_verification.altcha.secret","value":""},{"name":"human_verification.altcha.challenge_ttl","value":"2m"},{"name":"human_verification.altcha.cost","value":"2000"}]}`)
	updateReq := httptest.NewRequest(nethttp.MethodPut, "/api/v1/admin/web-options", bytes.NewReader(body))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.AddCookie(adminCookie)
	updateResp, err := app.Test(updateReq)
	if err != nil {
		t.Fatalf("update admin web options request failed: %v", err)
	}
	defer updateResp.Body.Close()

	if updateResp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", updateResp.StatusCode)
	}
	if optionStore.items[options.NameAltchaSecret] != "existing-secret" {
		t.Fatalf("expected blank secret update to keep existing secret, got %q", optionStore.items[options.NameAltchaSecret])
	}
	var updateBody apiEnvelope[[]options.AdminOption]
	if err := json.NewDecoder(updateResp.Body).Decode(&updateBody); err != nil {
		t.Fatalf("decode updated admin options response: %v", err)
	}
	if got := adminOption(updateBody.Data, options.NameSiteDefaultLocale).Value; got != "en-US" {
		t.Fatalf("expected normalized default locale, got %q", got)
	}
	if got := adminOption(updateBody.Data, options.NameAltchaSecret); !got.SecretSet || got.Value != "" {
		t.Fatalf("expected masked kept secret, got %#v", got)
	}
}

func TestRuntimeOptionsAffectHealthAndLocalizedResponses(t *testing.T) {
	cfg := testConfig()
	optionStore := newHTTPFakeOptionStore()
	optionStore.items[options.NameSiteName] = "Runtime Forum"
	optionStore.items[options.NameSiteDefaultLocale] = "en-US"
	optionStore.items[options.NameSiteSupportedLocales] = "en-US"
	optionsService := options.NewService(optionStore)
	optionsController := optionscontroller.NewController(optionsService, newHTTPFakeStore(), authsession.NewManager(session.NewStore(), authsession.Config{}))
	app := apphttp.NewApp(cfg, slog.Default(), apphttp.Dependencies{
		RouteProviders: []apphttp.RouteProvider{optionsController},
		Options:        optionsService,
	})

	healthReq := httptest.NewRequest(nethttp.MethodGet, "/api/v1/health", nil)
	healthResp, err := app.Test(healthReq)
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	defer healthResp.Body.Close()

	var health apiEnvelope[healthResponse]
	if err := json.NewDecoder(healthResp.Body).Decode(&health); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if health.Data.Name != "Runtime Forum" || health.Data.Locale != "en-US" || len(health.Data.SupportedLocales) != 1 {
		t.Fatalf("expected runtime health settings, got %#v", health.Data)
	}

	req := httptest.NewRequest(nethttp.MethodPut, "/api/v1/web-options", bytes.NewReader([]byte(`{"name":"site.name","value":"Example Forum"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("localized unauthorized request failed: %v", err)
	}
	defer resp.Body.Close()

	var body apiEnvelope[apiErrorData]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode localized unauthorized response: %v", err)
	}
	if body.Message != "Please sign in first." {
		t.Fatalf("expected runtime default locale to localize auth error in English, got %q", body.Message)
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

func assertRegistrationStatus(t *testing.T, app *fiber.App, expected bool) {
	t.Helper()

	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/auth/registration-status", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("registration status request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected registration status 200, got %d", resp.StatusCode)
	}
	var body apiEnvelope[identity.RegistrationStatus]
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode registration status response: %v", err)
	}
	if body.Code != nethttp.StatusOK || body.Message != "OK" {
		t.Fatalf("unexpected registration status envelope: %#v", body)
	}
	if body.Data.NextUserIsInitialSuperAdmin != expected {
		t.Fatalf("expected nextUserIsInitialSuperAdmin=%v, got %v", expected, body.Data.NextUserIsInitialSuperAdmin)
	}
}

func assertFieldMessage(t *testing.T, fields map[string][]string, field string, expected string) {
	t.Helper()
	messages := fields[field]
	if len(messages) != 1 || messages[0] != expected {
		t.Fatalf("expected field %s message %q, got %#v", field, expected, messages)
	}
}

func optionValue(items []options.Option, name string) string {
	for _, item := range items {
		if item.Name == name {
			return item.Value
		}
	}
	return ""
}

func adminOption(items []options.AdminOption, name string) options.AdminOption {
	for _, item := range items {
		if item.Name == name {
			return item
		}
	}
	return options.AdminOption{}
}

type httpFakeStore struct {
	nextUserID    int64
	nextRoleID    int64
	users         map[int64]identity.CurrentUser
	userEmails    map[int64]string
	credentials   map[int64]string
	loginIndex    map[string]int64
	roles         map[string]identity.Role
	userRoleIDs   map[int64][]int64
	rolePerms     map[int64][]string
	userOverrides map[int64]identity.PermissionOverrides
	loginAudits   []identity.LoginAudit
	loginAuditErr error
}

func newHTTPFakeStore() *httpFakeStore {
	return &httpFakeStore{
		nextUserID:  1,
		nextRoleID:  100,
		users:       map[int64]identity.CurrentUser{},
		userEmails:  map[int64]string{},
		credentials: map[int64]string{},
		loginIndex:  map[string]int64{},
		roles: map[string]identity.Role{
			identity.RoleSuperAdmin: {ID: 1, Key: identity.RoleSuperAdmin, Alias: "超级管理员", IsEnabled: true},
			identity.RoleMember:     {ID: 2, Key: identity.RoleMember, Alias: "普通会员", IsDefault: true, IsEnabled: true},
		},
		userRoleIDs:   map[int64][]int64{},
		rolePerms:     map[int64][]string{2: {identity.PermissionTopicCreate, identity.PermissionPostCreate}},
		userOverrides: map[int64]identity.PermissionOverrides{},
	}
}

func (s *httpFakeStore) WithBootstrapTx(ctx context.Context, fn func(context.Context, identity.TxStore) error) error {
	return fn(ctx, s)
}

func (s *httpFakeStore) AnyUserExists(context.Context) (bool, error) {
	return len(s.users) > 0, nil
}

func (s *httpFakeStore) FindRegistrationConflicts(_ context.Context, username string, email string) (identity.RegistrationConflicts, error) {
	return identity.RegistrationConflicts{
		UsernameTaken: s.loginIndex[strings.ToLower(username)] != 0,
		EmailTaken:    s.loginIndex[strings.ToLower(email)] != 0,
	}, nil
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
	s.userEmails[user.ID] = input.Email
	s.loginIndex[strings.ToLower(input.Username)] = user.ID
	s.loginIndex[strings.ToLower(input.Email)] = user.ID
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

func (s *httpFakeStore) LoadCurrentUserAccess(_ context.Context, current *identity.CurrentUser) error {
	*current = s.withAccess(*current)
	return nil
}

func (s *httpFakeStore) GetCurrentUser(_ context.Context, userID int64) (identity.CurrentUser, error) {
	user, ok := s.users[userID]
	if !ok {
		return identity.CurrentUser{}, errors.New("user not found")
	}
	return s.withAccess(user), nil
}

func (s *httpFakeStore) GetCredentialByLogin(ctx context.Context, login string) (identity.CredentialUser, error) {
	userID, ok := s.loginIndex[strings.ToLower(login)]
	if !ok {
		return identity.CredentialUser{}, identity.ErrCredentialNotFound
	}
	user, err := s.GetCurrentUser(ctx, userID)
	if err != nil {
		return identity.CredentialUser{}, err
	}
	return identity.CredentialUser{CurrentUser: user, PasswordHash: s.credentials[userID]}, nil
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

func (s *httpFakeStore) ListPermissions(context.Context) ([]identity.Permission, error) {
	permissions := make([]identity.Permission, 0, len(identity.SeedPermissions))
	for _, permission := range identity.SeedPermissions {
		permissions = append(permissions, identity.Permission{
			Key:         permission.Key,
			Module:      permission.Module,
			Description: permission.Description,
		})
	}
	return permissions, nil
}

func (s *httpFakeStore) ListPermissionMatrix(context.Context) (identity.PermissionMatrix, error) {
	permissions, _ := s.ListPermissions(context.Background())
	roles := make([]identity.RolePermissionSet, 0, len(s.roles))
	for _, role := range s.roles {
		roles = append(roles, identity.RolePermissionSet{
			RoleKey:        role.Key,
			PermissionKeys: slices.Clone(s.rolePerms[role.ID]),
		})
	}
	return identity.PermissionMatrix{Permissions: permissions, Roles: roles}, nil
}

func (s *httpFakeStore) ListUsers(_ context.Context, input identity.UserListInput) (identity.AdminUserList, error) {
	items := make([]identity.AdminUserSummary, 0, len(s.users))
	for _, user := range s.users {
		items = append(items, s.adminSummary(user))
	}
	return identity.AdminUserList{Items: items, Total: int64(len(items)), Page: input.Page, PerPage: input.PerPage}, nil
}

func (s *httpFakeStore) GetAdminUser(_ context.Context, userID int64) (identity.AdminUserDetail, error) {
	user, ok := s.users[userID]
	if !ok {
		return identity.AdminUserDetail{}, errors.New("user not found")
	}
	user = s.withAccess(user)
	return identity.AdminUserDetail{
		AdminUserSummary:    s.adminSummary(user),
		Permissions:         slices.Clone(user.Permissions),
		PermissionOverrides: s.cloneOverrides(userID),
	}, nil
}

func (s *httpFakeStore) ListRoles(context.Context) ([]identity.Role, error) {
	roles := make([]identity.Role, 0, len(s.roles))
	for _, role := range s.roles {
		role.PermissionKeys = slices.Clone(s.rolePerms[role.ID])
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

func (s *httpFakeStore) ReplaceRolePermissions(_ context.Context, _ int64, roleKey string, permissions []string) error {
	role := s.roles[roleKey]
	s.rolePerms[role.ID] = slices.Clone(permissions)
	return nil
}

func (s *httpFakeStore) ReplaceUserRoles(_ context.Context, _ int64, targetUserID int64, roleKeys []string) (identity.AdminUserDetail, error) {
	roleIDs := make([]int64, 0, len(roleKeys))
	for _, roleKey := range roleKeys {
		role, ok := s.roles[roleKey]
		if !ok {
			return identity.AdminUserDetail{}, errors.New("role not found")
		}
		roleIDs = append(roleIDs, role.ID)
	}
	s.userRoleIDs[targetUserID] = roleIDs
	return s.GetAdminUser(context.Background(), targetUserID)
}

func (s *httpFakeStore) ReplaceUserPermissionOverrides(_ context.Context, _ int64, targetUserID int64, overrides identity.PermissionOverrides) (identity.AdminUserDetail, error) {
	s.userOverrides[targetUserID] = identity.PermissionOverrides{
		Allow: slices.Clone(overrides.Allow),
		Deny:  slices.Clone(overrides.Deny),
	}
	return s.GetAdminUser(context.Background(), targetUserID)
}

func (s *httpFakeStore) RecordLoginAudit(_ context.Context, input identity.LoginAudit) error {
	if s.loginAuditErr != nil {
		return s.loginAuditErr
	}
	s.loginAudits = append(s.loginAudits, input)
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
			permissions[identity.PermissionSettingsManage] = true
		}
		for _, permission := range s.rolePerms[role.ID] {
			permissions[permission] = true
		}
	}
	overrides := s.userOverrides[user.ID]
	for _, permission := range overrides.Allow {
		permissions[permission] = true
	}
	for _, permission := range overrides.Deny {
		delete(permissions, permission)
	}

	user.RoleKeys = roleKeys
	user.Permissions = make([]string, 0, len(permissions))
	for permission := range permissions {
		user.Permissions = append(user.Permissions, permission)
	}
	return user
}

func (s *httpFakeStore) adminSummary(user identity.CurrentUser) identity.AdminUserSummary {
	user = s.withAccess(user)
	return identity.AdminUserSummary{
		ID:                  user.ID,
		Username:            user.Username,
		Email:               s.userEmails[user.ID],
		DisplayName:         user.DisplayName,
		Locale:              user.Locale,
		Status:              user.Status,
		IsInitialSuperAdmin: user.IsInitialSuperAdmin,
		RoleKeys:            slices.Clone(user.RoleKeys),
	}
}

func (s *httpFakeStore) cloneOverrides(userID int64) identity.PermissionOverrides {
	overrides := s.userOverrides[userID]
	return identity.PermissionOverrides{
		Allow: slices.Clone(overrides.Allow),
		Deny:  slices.Clone(overrides.Deny),
	}
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

type httpFakeOptionStore struct {
	items map[string]string
}

func newHTTPFakeOptionStore() *httpFakeOptionStore {
	return &httpFakeOptionStore{items: map[string]string{options.NameSiteName: "SForum"}}
}

func (s *httpFakeOptionStore) List(context.Context) ([]options.Option, error) {
	items := make([]options.Option, 0, len(s.items))
	for name, value := range s.items {
		items = append(items, options.Option{Name: name, Value: value})
	}
	return items, nil
}

func (s *httpFakeOptionStore) InsertMissing(_ context.Context, input options.UpdateInput) error {
	if s.items == nil {
		s.items = map[string]string{}
	}
	if _, ok := s.items[input.Name]; !ok {
		s.items[input.Name] = input.Value
	}
	return nil
}

func (s *httpFakeOptionStore) Upsert(_ context.Context, input options.UpdateInput) (options.Option, error) {
	s.items[input.Name] = input.Value
	return options.Option{Name: input.Name, Value: input.Value}, nil
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

type countingHumanProvider struct {
	result   humanverify.VerifyResult
	verifies int
}

func (p *countingHumanProvider) Challenge(context.Context, humanverify.Purpose, humanverify.Subject) (humanverify.Challenge, error) {
	return humanverify.Challenge{
		Provider: humanverify.ProviderAltcha,
		Purpose:  humanverify.PurposeRegister,
		Payload:  map[string]any{"challenge": "fake"},
	}, nil
}

func (p *countingHumanProvider) Verify(context.Context, humanverify.VerifyRequest) (humanverify.VerifyResult, error) {
	p.verifies++
	if p.result.Code == "" {
		return humanverify.VerifyResult{Verified: true, Code: humanverify.CodeOK}, nil
	}
	return p.result, nil
}

type failingSessionStorage struct {
	err error
}

func (s failingSessionStorage) GetWithContext(context.Context, string) ([]byte, error) {
	return nil, nil
}

func (s failingSessionStorage) Get(string) ([]byte, error) {
	return nil, nil
}

func (s failingSessionStorage) SetWithContext(context.Context, string, []byte, time.Duration) error {
	return s.err
}

func (s failingSessionStorage) Set(string, []byte, time.Duration) error {
	return s.err
}

func (s failingSessionStorage) DeleteWithContext(context.Context, string) error {
	return nil
}

func (s failingSessionStorage) Delete(string) error {
	return nil
}

func (s failingSessionStorage) ResetWithContext(context.Context) error {
	return nil
}

func (s failingSessionStorage) Reset() error {
	return nil
}

func (s failingSessionStorage) Close() error {
	return nil
}
