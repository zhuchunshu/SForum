package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3/middleware/session"

	"github.com/inkedus/sforum/apps/api/internal/config"
	"github.com/inkedus/sforum/apps/api/internal/modules/identity"
)

func TestHealthEndpoint(t *testing.T) {
	cfg := config.Config{
		AppName:          "SForum",
		AppEnv:           "test",
		AppLocale:        "zh-CN",
		SupportedLocales: []string{"zh-CN", "en-US"},
	}

	app := NewApp(cfg, slog.Default(), Dependencies{})
	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/health", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != nethttp.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body healthResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}

	if body.Status != "ok" {
		t.Fatalf("expected ok status, got %q", body.Status)
	}
	if body.Locale != "zh-CN" {
		t.Fatalf("expected zh-CN locale, got %q", body.Locale)
	}
	if len(body.SupportedLocales) != 2 {
		t.Fatalf("expected two supported locales, got %v", body.SupportedLocales)
	}
}

func TestRegisterEndpointCreatesSession(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	identityHandler := identity.NewHandler(identity.NewService(store), session.NewStore())
	app := NewApp(cfg, slog.Default(), Dependencies{IdentityHandler: identityHandler})

	body := []byte(`{"username":"admin","email":"admin@example.com","password":"correct horse battery staple","displayName":"Admin","locale":"zh-CN"}`)
	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
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
}

func TestSessionEndpointRequiresAuth(t *testing.T) {
	cfg := testConfig()
	identityHandler := identity.NewHandler(identity.NewService(newHTTPFakeStore()), session.NewStore())
	app := NewApp(cfg, slog.Default(), Dependencies{IdentityHandler: identityHandler})
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

func testConfig() config.Config {
	return config.Config{
		AppName:          "SForum",
		AppEnv:           "test",
		AppLocale:        "zh-CN",
		SupportedLocales: []string{"zh-CN", "en-US"},
	}
}

type httpFakeStore struct {
	nextUserID  int64
	users       map[int64]identity.CurrentUser
	credentials map[int64]string
	roles       map[string]identity.Role
	userRoleIDs map[int64][]int64
}

func newHTTPFakeStore() *httpFakeStore {
	return &httpFakeStore{
		nextUserID:  1,
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
	return user, nil
}

func (s *httpFakeStore) GetCredentialByLogin(context.Context, string) (identity.CredentialUser, error) {
	return identity.CredentialUser{}, errors.New("not implemented")
}

func (s *httpFakeStore) LoadActor(context.Context, int64) (identity.Actor, error) {
	return identity.Actor{}, errors.New("not implemented")
}

func (s *httpFakeStore) ListRoles(context.Context) ([]identity.Role, error) {
	return nil, errors.New("not implemented")
}

func (s *httpFakeStore) CreateRole(context.Context, identity.RoleInput) (identity.Role, error) {
	return identity.Role{}, errors.New("not implemented")
}

func (s *httpFakeStore) UpdateRole(context.Context, string, identity.RoleInput) (identity.Role, error) {
	return identity.Role{}, errors.New("not implemented")
}

func (s *httpFakeStore) DeleteRole(context.Context, string) error {
	return errors.New("not implemented")
}

func (s *httpFakeStore) ReplaceRolePermissions(context.Context, int64, string, []string) error {
	return errors.New("not implemented")
}
