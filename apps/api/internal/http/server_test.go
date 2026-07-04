package http

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

	"github.com/gofiber/fiber/v3"
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

func TestRolesEndpointRequiresAuth(t *testing.T) {
	cfg := testConfig()
	identityHandler := identity.NewHandler(identity.NewService(newHTTPFakeStore()), session.NewStore())
	app := NewApp(cfg, slog.Default(), Dependencies{IdentityHandler: identityHandler})
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
	identityHandler := identity.NewHandler(identity.NewService(store), session.NewStore())
	app := NewApp(cfg, slog.Default(), Dependencies{IdentityHandler: identityHandler})
	adminCookie := registerHTTPUser(t, app, "admin", "admin@example.com")

	body := []byte(`{"key":"moderator","alias":"版主","description":"管理内容"}`)
	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/roles", bytes.NewReader(body))
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
	var role identity.Role
	if err := json.NewDecoder(resp.Body).Decode(&role); err != nil {
		t.Fatalf("decode role response: %v", err)
	}
	if role.Key != "moderator" || role.Alias != "版主" {
		t.Fatalf("unexpected role response: %#v", role)
	}
}

func TestCreateRoleEndpointRejectsMember(t *testing.T) {
	cfg := testConfig()
	store := newHTTPFakeStore()
	identityHandler := identity.NewHandler(identity.NewService(store), session.NewStore())
	app := NewApp(cfg, slog.Default(), Dependencies{IdentityHandler: identityHandler})
	registerHTTPUser(t, app, "admin", "admin@example.com")
	memberCookie := registerHTTPUser(t, app, "member1", "member1@example.com")

	body := []byte(`{"key":"moderator","alias":"版主","description":"管理内容"}`)
	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/roles", bytes.NewReader(body))
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
