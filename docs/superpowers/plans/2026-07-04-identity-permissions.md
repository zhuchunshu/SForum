# Identity Permissions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the SForum identity foundation: open registration, first-user `super_admin` bootstrap, default `member` role, Redis-backed browser sessions, API authorization, and basic admin role management.

**Architecture:** Keep one user system for regular members, moderators, and administrators. PostgreSQL owns users, roles, permissions, role assignments, and audit events; Redis owns browser sessions; Go service/policy code owns authorization decisions; Nuxt consumes the current-session API and protects admin routes for user experience only.

**Tech Stack:** Go Fiber v3, PostgreSQL, `pgx/v5`, `sqlc`, `goose`, Redis/Fiber sessions, Argon2id from `golang.org/x/crypto`, Nuxt 4, Vue 3, Nuxt i18n, Bun.

---

## Source Notes

- Approved spec: `docs/superpowers/specs/2026-07-04-identity-permissions-design.md`
- Existing API scaffold: `apps/api/internal/http/server.go`
- Existing OpenAPI contract: `contracts/openapi.yaml`
- Existing Nuxt proxy: `apps/web/server/routes/api/v1/[...path].ts`
- Official docs referenced during planning:
  - Fiber session middleware: https://docs.gofiber.io/next/middleware/session/
  - Fiber CSRF middleware: https://docs.gofiber.io/next/middleware/csrf/
  - `pgx`: https://pkg.go.dev/github.com/jackc/pgx/v5
  - `sqlc`: https://docs.sqlc.dev/en/latest/
  - `goose`: https://github.com/pressly/goose

## File Map

Backend files to create:

- `apps/api/cmd/migrate/main.go`: run goose migrations against `DATABASE_URL`.
- `apps/api/internal/platform/postgres/pool.go`: create and close `pgxpool.Pool`.
- `apps/api/internal/platform/redis/session.go`: create Redis-backed Fiber session storage.
- `apps/api/internal/store/migrations/202607040001_identity_rbac.sql`: identity and RBAC schema plus seed roles/permissions.
- `apps/api/internal/store/queries/identity.sql`: SQLC queries for registration, session loading, permissions, roles, and role assignment.
- `apps/api/internal/modules/identity/types.go`: role keys, permission keys, statuses, DTOs, and domain errors.
- `apps/api/internal/modules/identity/policy.go`: actor permission helpers.
- `apps/api/internal/modules/identity/password.go`: Argon2id password hashing and verification.
- `apps/api/internal/modules/identity/service.go`: registration, login, session summary, and role-management service methods.
- `apps/api/internal/modules/identity/store.go`: store interfaces used by services.
- `apps/api/internal/modules/identity/postgres_store.go`: PostgreSQL implementation over sqlc queries.
- `apps/api/internal/modules/identity/http.go`: Fiber handlers and route registration.
- `apps/api/internal/modules/identity/*_test.go`: unit tests for policy, passwords, registration, and role invariants.

Backend files to modify:

- `apps/api/go.mod`
- `apps/api/go.sum`
- `apps/api/internal/config/config.go`
- `apps/api/internal/http/errors.go`
- `apps/api/internal/http/server.go`
- `apps/api/cmd/api/main.go`
- `apps/api/sqlc.yaml` only if generated-package settings need adjustment after `sqlc generate`.
- `contracts/openapi.yaml`

Frontend files to create:

- `apps/web/app/composables/useAuthSession.ts`: fetch and cache current session state.
- `apps/web/app/middleware/admin.ts`: redirect non-admin users away from `/admin/*`.
- `apps/web/app/pages/register.vue`: open registration page.
- `apps/web/app/pages/login.vue`: login page.
- `apps/web/app/pages/admin/index.vue`: protected admin overview.
- `apps/web/app/pages/admin/roles.vue`: first role-management screen shell.
- `tests/validate-identity-ui.js`: repository-level validation for i18n keys and required Nuxt auth/admin files.

Frontend files to modify:

- `apps/web/i18n/locales/zh-CN.json`
- `apps/web/i18n/locales/en-US.json`
- `scripts/test.sh`

---

### Task 1: Add Backend Database, Migration, And Session Infrastructure

**Files:**
- Modify: `apps/api/go.mod`
- Modify: `apps/api/go.sum`
- Create: `apps/api/cmd/migrate/main.go`
- Create: `apps/api/internal/platform/postgres/pool.go`
- Create: `apps/api/internal/platform/redis/session.go`
- Test: `apps/api/internal/platform/redis/session_test.go`

- [ ] **Step 1: Write Redis address parsing tests**

Create `apps/api/internal/platform/redis/session_test.go`:

```go
package redis

import "testing"

func TestParseAddr(t *testing.T) {
	host, port, err := ParseAddr("redis:6379")
	if err != nil {
		t.Fatalf("ParseAddr returned error: %v", err)
	}
	if host != "redis" {
		t.Fatalf("expected host redis, got %q", host)
	}
	if port != 6379 {
		t.Fatalf("expected port 6379, got %d", port)
	}
}

func TestParseAddrRejectsMissingPort(t *testing.T) {
	_, _, err := ParseAddr("redis")
	if err == nil {
		t.Fatal("expected error for missing port")
	}
}
```

- [ ] **Step 2: Run the new test to verify it fails**

Run:

```bash
cd apps/api
go test ./internal/platform/redis -run TestParseAddr -count=1
```

Expected: FAIL because package `internal/platform/redis` or `ParseAddr` does not exist.

- [ ] **Step 3: Add backend dependencies**

Run:

```bash
cd apps/api
go get github.com/jackc/pgx/v5 github.com/pressly/goose/v3 github.com/gofiber/storage/redis/v3
```

Expected: `go.mod` records direct requirements for `pgx/v5`, `goose/v3`, and Fiber Redis storage.

- [ ] **Step 4: Create Redis session storage helper**

Create `apps/api/internal/platform/redis/session.go`:

```go
package redis

import (
	"fmt"
	"net"
	"strconv"

	redisstorage "github.com/gofiber/storage/redis/v3"
)

func ParseAddr(addr string) (string, int, error) {
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, fmt.Errorf("parse redis addr %q: %w", addr, err)
	}

	port, err := strconv.Atoi(portText)
	if err != nil {
		return "", 0, fmt.Errorf("parse redis port %q: %w", portText, err)
	}
	if host == "" || port <= 0 {
		return "", 0, fmt.Errorf("invalid redis addr %q", addr)
	}

	return host, port, nil
}

func NewStorage(addr string) (*redisstorage.Storage, error) {
	host, port, err := ParseAddr(addr)
	if err != nil {
		return nil, err
	}

	return redisstorage.New(redisstorage.Config{
		Host: host,
		Port: port,
	}), nil
}
```

- [ ] **Step 5: Create PostgreSQL pool helper**

Create `apps/api/internal/platform/postgres/pool.go`:

```go
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}
```

- [ ] **Step 6: Create migrate command**

Create `apps/api/cmd/migrate/main.go`:

```go
package main

import (
	"database/sql"
	"log/slog"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/inkedus/sforum/apps/api/internal/config"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		logger.Error("open database failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		logger.Error("database ping failed", "error", err)
		os.Exit(1)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		logger.Error("set goose dialect failed", "error", err)
		os.Exit(1)
	}

	if err := goose.Up(db, "internal/store/migrations"); err != nil {
		logger.Error("migrations failed", "error", err)
		os.Exit(1)
	}

	logger.Info("migrations complete")
}
```

- [ ] **Step 7: Run tests and build**

Run:

```bash
cd apps/api
go test ./internal/platform/redis -count=1
go test ./...
go build ./cmd/migrate
```

Expected: all tests pass and the migrate command builds.

- [ ] **Step 8: Commit**

```bash
git add apps/api/go.mod apps/api/go.sum apps/api/cmd/migrate/main.go apps/api/internal/platform/postgres/pool.go apps/api/internal/platform/redis/session.go apps/api/internal/platform/redis/session_test.go
git commit -m "feat(api): add database and session infrastructure"
```

---

### Task 2: Add Identity And RBAC Schema

**Files:**
- Create: `apps/api/internal/store/migrations/202607040001_identity_rbac.sql`
- Create: `apps/api/internal/modules/identity/seeds.go`
- Test: `apps/api/internal/modules/identity/seeds_test.go`

- [ ] **Step 1: Write seed constant tests**

Create `apps/api/internal/modules/identity/seeds_test.go`:

```go
package identity

import "testing"

func TestSystemRoles(t *testing.T) {
	if RoleSuperAdmin != "super_admin" {
		t.Fatalf("unexpected super admin role key: %q", RoleSuperAdmin)
	}
	if RoleMember != "member" {
		t.Fatalf("unexpected member role key: %q", RoleMember)
	}
}

func TestDefaultPermissionsContainAdminAccess(t *testing.T) {
	found := false
	for _, permission := range SeedPermissions {
		if permission.Key == PermissionAdminAccess {
			found = true
		}
		if permission.Key == "" || permission.Module == "" {
			t.Fatalf("permission must have key and module: %#v", permission)
		}
	}
	if !found {
		t.Fatal("expected admin.access seed permission")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run:

```bash
cd apps/api
go test ./internal/modules/identity -run 'TestSystemRoles|TestDefaultPermissionsContainAdminAccess' -count=1
```

Expected: FAIL because package files and constants do not exist.

- [ ] **Step 3: Create seed constants**

Create `apps/api/internal/modules/identity/seeds.go`:

```go
package identity

const (
	RoleSuperAdmin = "super_admin"
	RoleMember     = "member"

	PermissionAdminAccess            = "admin.access"
	PermissionRoleManage             = "role.manage"
	PermissionUserManage             = "user.manage"
	PermissionUserBan                = "user.ban"
	PermissionCategoryManage         = "category.manage"
	PermissionTopicCreate            = "topic.create"
	PermissionTopicEditAny           = "topic.edit_any"
	PermissionTopicDeleteAny         = "topic.delete_any"
	PermissionTopicLock              = "topic.lock"
	PermissionTopicPin               = "topic.pin"
	PermissionPostCreate             = "post.create"
	PermissionPostEditOwn            = "post.edit_own"
	PermissionPostEditAny            = "post.edit_any"
	PermissionPostDeleteOwn          = "post.delete_own"
	PermissionPostDeleteAny          = "post.delete_any"
	PermissionModerationReportReview = "moderation.report_review"
	PermissionSettingsManage         = "settings.manage"
)

type SeedPermission struct {
	Key         string
	Module      string
	Description string
}

var SeedPermissions = []SeedPermission{
	{Key: PermissionAdminAccess, Module: "admin", Description: "Access the admin area."},
	{Key: PermissionRoleManage, Module: "identity", Description: "Create and update roles and role permissions."},
	{Key: PermissionUserManage, Module: "identity", Description: "Manage user accounts and assignments."},
	{Key: PermissionUserBan, Module: "identity", Description: "Ban users from participating."},
	{Key: PermissionCategoryManage, Module: "forum", Description: "Create and update categories."},
	{Key: PermissionTopicCreate, Module: "forum", Description: "Create topics."},
	{Key: PermissionTopicEditAny, Module: "forum", Description: "Edit any topic."},
	{Key: PermissionTopicDeleteAny, Module: "forum", Description: "Delete any topic."},
	{Key: PermissionTopicLock, Module: "forum", Description: "Lock or unlock topics."},
	{Key: PermissionTopicPin, Module: "forum", Description: "Pin or unpin topics."},
	{Key: PermissionPostCreate, Module: "forum", Description: "Create posts."},
	{Key: PermissionPostEditOwn, Module: "forum", Description: "Edit own posts."},
	{Key: PermissionPostEditAny, Module: "forum", Description: "Edit any post."},
	{Key: PermissionPostDeleteOwn, Module: "forum", Description: "Delete own posts."},
	{Key: PermissionPostDeleteAny, Module: "forum", Description: "Delete any post."},
	{Key: PermissionModerationReportReview, Module: "moderation", Description: "Review moderation reports."},
	{Key: PermissionSettingsManage, Module: "admin", Description: "Manage system settings."},
}
```

- [ ] **Step 4: Create identity migration**

Create `apps/api/internal/store/migrations/202607040001_identity_rbac.sql`:

```sql
-- +goose Up
CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  username TEXT NOT NULL,
  username_lower TEXT NOT NULL UNIQUE,
  email TEXT NOT NULL,
  email_lower TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  locale TEXT NOT NULL DEFAULT 'zh-CN',
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'banned')),
  is_initial_super_admin BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX users_one_initial_super_admin
  ON users (is_initial_super_admin)
  WHERE is_initial_super_admin;

CREATE TABLE user_credentials (
  user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  password_hash TEXT NOT NULL,
  password_changed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE roles (
  id BIGSERIAL PRIMARY KEY,
  key TEXT NOT NULL UNIQUE,
  alias TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  is_system BOOLEAN NOT NULL DEFAULT FALSE,
  is_default BOOLEAN NOT NULL DEFAULT FALSE,
  is_deletable BOOLEAN NOT NULL DEFAULT TRUE,
  is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX roles_one_default_role
  ON roles (is_default)
  WHERE is_default;

CREATE TABLE permissions (
  key TEXT PRIMARY KEY,
  module TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE role_permissions (
  role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  permission_key TEXT NOT NULL REFERENCES permissions(key) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (role_id, permission_key)
);

CREATE TABLE user_roles (
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, role_id)
);

CREATE TABLE audit_events (
  id BIGSERIAL PRIMARY KEY,
  actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  target_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  action TEXT NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO roles (key, alias, description, is_system, is_default, is_deletable)
VALUES
  ('super_admin', '超级管理员', '拥有所有权限并用于系统恢复。', TRUE, FALSE, FALSE),
  ('member', '普通会员', '开放注册用户的默认用户组。', TRUE, TRUE, FALSE);

INSERT INTO permissions (key, module, description)
VALUES
  ('admin.access', 'admin', 'Access the admin area.'),
  ('role.manage', 'identity', 'Create and update roles and role permissions.'),
  ('user.manage', 'identity', 'Manage user accounts and assignments.'),
  ('user.ban', 'identity', 'Ban users from participating.'),
  ('category.manage', 'forum', 'Create and update categories.'),
  ('topic.create', 'forum', 'Create topics.'),
  ('topic.edit_any', 'forum', 'Edit any topic.'),
  ('topic.delete_any', 'forum', 'Delete any topic.'),
  ('topic.lock', 'forum', 'Lock or unlock topics.'),
  ('topic.pin', 'forum', 'Pin or unpin topics.'),
  ('post.create', 'forum', 'Create posts.'),
  ('post.edit_own', 'forum', 'Edit own posts.'),
  ('post.edit_any', 'forum', 'Edit any post.'),
  ('post.delete_own', 'forum', 'Delete own posts.'),
  ('post.delete_any', 'forum', 'Delete any post.'),
  ('moderation.report_review', 'moderation', 'Review moderation reports.'),
  ('settings.manage', 'admin', 'Manage system settings.');

INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, permissions.key
FROM roles
CROSS JOIN permissions
WHERE roles.key = 'super_admin';

INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, permissions.key
FROM roles
JOIN permissions ON permissions.key IN ('topic.create', 'post.create', 'post.edit_own', 'post.delete_own')
WHERE roles.key = 'member';

-- +goose Down
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS user_credentials;
DROP INDEX IF EXISTS users_one_initial_super_admin;
DROP TABLE IF EXISTS users;
```

- [ ] **Step 5: Run tests**

Run:

```bash
cd apps/api
go test ./internal/modules/identity -run 'TestSystemRoles|TestDefaultPermissionsContainAdminAccess' -count=1
go test ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/api/internal/store/migrations/202607040001_identity_rbac.sql apps/api/internal/modules/identity/seeds.go apps/api/internal/modules/identity/seeds_test.go
git commit -m "feat(api): add identity rbac schema"
```

---

### Task 3: Add Identity Domain Types And Policy Helpers

**Files:**
- Create: `apps/api/internal/modules/identity/types.go`
- Create: `apps/api/internal/modules/identity/policy.go`
- Test: `apps/api/internal/modules/identity/policy_test.go`

- [ ] **Step 1: Write policy tests**

Create `apps/api/internal/modules/identity/policy_test.go`:

```go
package identity

import "testing"

func TestSuperAdminCanUseEveryPermission(t *testing.T) {
	actor := Actor{
		ID:       1,
		Status:   UserStatusActive,
		RoleKeys: []string{RoleSuperAdmin},
	}

	if !actor.Can("made.up.permission") {
		t.Fatal("expected super admin to pass any permission")
	}
}

func TestInactiveActorCannotUsePermissions(t *testing.T) {
	actor := Actor{
		ID:          2,
		Status:      UserStatusDisabled,
		Permissions: map[string]bool{PermissionAdminAccess: true},
	}

	if actor.Can(PermissionAdminAccess) {
		t.Fatal("expected disabled actor to fail permission check")
	}
}

func TestMemberCanEditOwnPost(t *testing.T) {
	actor := Actor{
		ID:          3,
		Status:      UserStatusActive,
		Permissions: map[string]bool{PermissionPostEditOwn: true},
	}
	post := PostSummary{ID: 10, AuthorUserID: 3}

	if !CanEditPost(actor, post) {
		t.Fatal("expected actor to edit own post")
	}
}

func TestMemberCannotEditOtherPostWithoutAnyPermission(t *testing.T) {
	actor := Actor{
		ID:          3,
		Status:      UserStatusActive,
		Permissions: map[string]bool{PermissionPostEditOwn: true},
	}
	post := PostSummary{ID: 10, AuthorUserID: 4}

	if CanEditPost(actor, post) {
		t.Fatal("expected actor not to edit someone else's post")
	}
}
```

- [ ] **Step 2: Run policy tests to verify they fail**

Run:

```bash
cd apps/api
go test ./internal/modules/identity -run 'Test.*Actor|TestMember' -count=1
```

Expected: FAIL because `Actor`, `UserStatusActive`, and policy helpers do not exist.

- [ ] **Step 3: Add domain types**

Create `apps/api/internal/modules/identity/types.go`:

```go
package identity

import "errors"

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
	UserStatusBanned   UserStatus = "banned"
)

var (
	ErrInvalidCredentials        = errors.New("identity: invalid credentials")
	ErrPermissionDenied          = errors.New("identity: permission denied")
	ErrSystemRoleLocked          = errors.New("identity: system role is locked")
	ErrDefaultRoleLocked         = errors.New("identity: default role is locked")
	ErrInitialSuperAdminLocked   = errors.New("identity: initial super admin is locked")
	ErrUsernameOrEmailNotUnique  = errors.New("identity: username or email is not unique")
	ErrPasswordDoesNotMeetPolicy = errors.New("identity: password does not meet policy")
)

type Actor struct {
	ID          int64
	Status      UserStatus
	RoleKeys    []string
	Permissions map[string]bool
}

type PostSummary struct {
	ID           int64
	AuthorUserID int64
}

type CurrentUser struct {
	ID                   int64      `json:"id"`
	Username             string     `json:"username"`
	DisplayName          string     `json:"displayName"`
	Locale               string     `json:"locale"`
	Status               UserStatus `json:"status"`
	IsInitialSuperAdmin  bool       `json:"isInitialSuperAdmin"`
	RoleKeys             []string   `json:"roleKeys"`
	Permissions          []string   `json:"permissions"`
}
```

- [ ] **Step 4: Add policy helpers**

Create `apps/api/internal/modules/identity/policy.go`:

```go
package identity

func (a Actor) IsActive() bool {
	return a.Status == UserStatusActive
}

func (a Actor) IsSuperAdmin() bool {
	if !a.IsActive() {
		return false
	}
	for _, roleKey := range a.RoleKeys {
		if roleKey == RoleSuperAdmin {
			return true
		}
	}
	return false
}

func (a Actor) Can(permission string) bool {
	if !a.IsActive() {
		return false
	}
	if a.IsSuperAdmin() {
		return true
	}
	if a.Permissions == nil {
		return false
	}
	return a.Permissions[permission]
}

func CanEditPost(actor Actor, post PostSummary) bool {
	if actor.Can(PermissionPostEditAny) {
		return true
	}
	return post.AuthorUserID == actor.ID && actor.Can(PermissionPostEditOwn)
}
```

- [ ] **Step 5: Run tests**

Run:

```bash
cd apps/api
go test ./internal/modules/identity -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/api/internal/modules/identity/types.go apps/api/internal/modules/identity/policy.go apps/api/internal/modules/identity/policy_test.go
git commit -m "feat(api): add identity policy helpers"
```

---

### Task 4: Add Argon2id Password Hashing

**Files:**
- Create: `apps/api/internal/modules/identity/password.go`
- Test: `apps/api/internal/modules/identity/password_test.go`

- [ ] **Step 1: Write password tests**

Create `apps/api/internal/modules/identity/password_test.go`:

```go
package identity

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == "" {
		t.Fatal("expected password hash")
	}

	ok, err := VerifyPassword("correct horse battery staple", hash)
	if err != nil {
		t.Fatalf("VerifyPassword returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected password to verify")
	}
}

func TestVerifyPasswordRejectsWrongPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	ok, err := VerifyPassword("wrong password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword returned error: %v", err)
	}
	if ok {
		t.Fatal("expected wrong password to fail")
	}
}

func TestHashPasswordRejectsShortPassword(t *testing.T) {
	_, err := HashPassword("short")
	if err != ErrPasswordDoesNotMeetPolicy {
		t.Fatalf("expected password policy error, got %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
cd apps/api
go test ./internal/modules/identity -run Test.*Password -count=1
```

Expected: FAIL because password helpers do not exist.

- [ ] **Step 3: Implement password hashing**

Create `apps/api/internal/modules/identity/password.go`:

```go
package identity

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	passwordSaltBytes = 16
	passwordTime      = uint32(1)
	passwordMemory    = uint32(64 * 1024)
	passwordThreads   = uint8(4)
	passwordKeyBytes  = uint32(32)
)

func HashPassword(password string) (string, error) {
	if len([]rune(password)) < 12 {
		return "", ErrPasswordDoesNotMeetPolicy
	}

	salt := make([]byte, passwordSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read password salt: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, passwordTime, passwordMemory, passwordThreads, passwordKeyBytes)
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedKey := base64.RawStdEncoding.EncodeToString(key)

	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", passwordMemory, passwordTime, passwordThreads, encodedSalt, encodedKey), nil
}

func VerifyPassword(password string, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false, fmt.Errorf("invalid password hash format")
	}

	params := strings.Split(parts[3], ",")
	if len(params) != 3 {
		return false, fmt.Errorf("invalid password hash params")
	}

	memory, err := parseParam(params[0], "m")
	if err != nil {
		return false, err
	}
	time, err := parseParam(params[1], "t")
	if err != nil {
		return false, err
	}
	threads, err := parseParam(params[2], "p")
	if err != nil {
		return false, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("decode password salt: %w", err)
	}
	expectedKey, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("decode password key: %w", err)
	}

	actualKey := argon2.IDKey([]byte(password), salt, uint32(time), uint32(memory), uint8(threads), uint32(len(expectedKey)))
	return subtle.ConstantTimeCompare(actualKey, expectedKey) == 1, nil
}

func parseParam(value string, key string) (int, error) {
	prefix := key + "="
	if !strings.HasPrefix(value, prefix) {
		return 0, fmt.Errorf("missing password hash param %s", key)
	}
	parsed, err := strconv.Atoi(strings.TrimPrefix(value, prefix))
	if err != nil {
		return 0, fmt.Errorf("parse password hash param %s: %w", key, err)
	}
	return parsed, nil
}
```

- [ ] **Step 4: Run tests**

Run:

```bash
cd apps/api
go test ./internal/modules/identity -run Test.*Password -count=1
go test ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/internal/modules/identity/password.go apps/api/internal/modules/identity/password_test.go apps/api/go.mod apps/api/go.sum
git commit -m "feat(api): add argon2id password hashing"
```

---

### Task 5: Add SQLC Identity Queries And Generated Store

**Files:**
- Create: `apps/api/internal/store/queries/identity.sql`
- Generate: `apps/api/internal/store/sqlc/*`
- Create: `apps/api/internal/modules/identity/store.go`
- Create: `apps/api/internal/modules/identity/postgres_store.go`

- [ ] **Step 1: Add SQL queries**

Create `apps/api/internal/store/queries/identity.sql`:

```sql
-- name: AnyUserExists :one
SELECT EXISTS (SELECT 1 FROM users LIMIT 1)::boolean;

-- name: CreateUser :one
INSERT INTO users (username, username_lower, email, email_lower, display_name, locale, is_initial_super_admin)
VALUES ($1, lower($1), $2, lower($2), $3, $4, $5)
RETURNING id, username, display_name, locale, status, is_initial_super_admin;

-- name: CreateUserCredential :exec
INSERT INTO user_credentials (user_id, password_hash)
VALUES ($1, $2);

-- name: GetUserCredentialByLogin :one
SELECT users.id, users.username, users.display_name, users.locale, users.status, users.is_initial_super_admin, user_credentials.password_hash
FROM users
JOIN user_credentials ON user_credentials.user_id = users.id
WHERE users.username_lower = lower($1) OR users.email_lower = lower($1);

-- name: GetDefaultRole :one
SELECT id, key, alias, description, is_system, is_default, is_deletable, is_enabled
FROM roles
WHERE is_default = TRUE AND is_enabled = TRUE;

-- name: GetRoleByKey :one
SELECT id, key, alias, description, is_system, is_default, is_deletable, is_enabled
FROM roles
WHERE key = $1;

-- name: AssignRoleToUser :exec
INSERT INTO user_roles (user_id, role_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: ListUserRoleKeys :many
SELECT roles.key
FROM user_roles
JOIN roles ON roles.id = user_roles.role_id
WHERE user_roles.user_id = $1 AND roles.is_enabled = TRUE
ORDER BY roles.key;

-- name: ListUserPermissions :many
SELECT DISTINCT permissions.key
FROM user_roles
JOIN roles ON roles.id = user_roles.role_id
JOIN role_permissions ON role_permissions.role_id = roles.id
JOIN permissions ON permissions.key = role_permissions.permission_key
WHERE user_roles.user_id = $1 AND roles.is_enabled = TRUE
ORDER BY permissions.key;

-- name: GetCurrentUser :one
SELECT id, username, display_name, locale, status, is_initial_super_admin
FROM users
WHERE id = $1;

-- name: ListRoles :many
SELECT id, key, alias, description, is_system, is_default, is_deletable, is_enabled
FROM roles
ORDER BY is_system DESC, key ASC;

-- name: CreateRole :one
INSERT INTO roles (key, alias, description, is_system, is_default, is_deletable, is_enabled)
VALUES ($1, $2, $3, FALSE, FALSE, TRUE, TRUE)
RETURNING id, key, alias, description, is_system, is_default, is_deletable, is_enabled;

-- name: UpdateRoleAlias :one
UPDATE roles
SET alias = $2, description = $3, updated_at = now()
WHERE key = $1
RETURNING id, key, alias, description, is_system, is_default, is_deletable, is_enabled;

-- name: DeleteRoleByKey :exec
DELETE FROM roles
WHERE key = $1 AND is_deletable = TRUE AND is_system = FALSE;

-- name: DeleteRolePermissions :exec
DELETE FROM role_permissions
WHERE role_id = $1;

-- name: AddRolePermission :exec
INSERT INTO role_permissions (role_id, permission_key)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: CreateAuditEvent :exec
INSERT INTO audit_events (actor_user_id, target_user_id, action, metadata)
VALUES ($1, $2, $3, $4);
```

- [ ] **Step 2: Generate sqlc code**

Run:

```bash
cd apps/api
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0 generate
```

Expected: generated Go files appear in `apps/api/internal/store/sqlc`.

- [ ] **Step 3: Create store interface**

Create `apps/api/internal/modules/identity/store.go`:

```go
package identity

import "context"

type Store interface {
	WithBootstrapTx(ctx context.Context, fn func(context.Context, TxStore) error) error
	GetCurrentUser(ctx context.Context, userID int64) (CurrentUser, error)
	LoadActor(ctx context.Context, userID int64) (Actor, error)
	ListRoles(ctx context.Context) ([]Role, error)
	CreateRole(ctx context.Context, input RoleInput) (Role, error)
	UpdateRole(ctx context.Context, roleKey string, input RoleInput) (Role, error)
	DeleteRole(ctx context.Context, roleKey string) error
	ReplaceRolePermissions(ctx context.Context, actorUserID int64, roleKey string, permissions []string) error
}

type TxStore interface {
	AnyUserExists(ctx context.Context) (bool, error)
	CreateUser(ctx context.Context, input CreateUserInput) (CurrentUser, error)
	CreateCredential(ctx context.Context, userID int64, passwordHash string) error
	GetDefaultRole(ctx context.Context) (Role, error)
	GetRole(ctx context.Context, roleKey string) (Role, error)
	AssignRole(ctx context.Context, userID int64, roleID int64) error
}

type CreateUserInput struct {
	Username             string
	Email                string
	DisplayName          string
	Locale               string
	IsInitialSuperAdmin  bool
}

type Role struct {
	ID          int64  `json:"id"`
	Key         string `json:"key"`
	Alias       string `json:"alias"`
	Description string `json:"description"`
	IsSystem    bool   `json:"isSystem"`
	IsDefault   bool   `json:"isDefault"`
	IsDeletable bool   `json:"isDeletable"`
	IsEnabled   bool   `json:"isEnabled"`
}

type RoleInput struct {
	Key         string
	Alias       string
	Description string
}
```

- [ ] **Step 4: Implement Postgres store**

Create `apps/api/internal/modules/identity/postgres_store.go` with these required behaviors:

```go
package identity

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	store "github.com/inkedus/sforum/apps/api/internal/store/sqlc"
)

type PostgresStore struct {
	pool    *pgxpool.Pool
	queries *store.Queries
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool, queries: store.New(pool)}
}

func (s *PostgresStore) WithBootstrapTx(ctx context.Context, fn func(context.Context, TxStore) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin identity tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext('sforum.identity.bootstrap'))"); err != nil {
		return fmt.Errorf("lock identity bootstrap: %w", err)
	}

	txStore := &postgresTxStore{queries: s.queries.WithTx(tx)}
	if err := fn(ctx, txStore); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit identity tx: %w", err)
	}
	return nil
}

func (s *PostgresStore) LoadActor(ctx context.Context, userID int64) (Actor, error) {
	current, err := s.GetCurrentUser(ctx, userID)
	if err != nil {
		return Actor{}, err
	}
	roleKeys, err := s.queries.ListUserRoleKeys(ctx, userID)
	if err != nil {
		return Actor{}, fmt.Errorf("list user roles: %w", err)
	}
	permissionKeys, err := s.queries.ListUserPermissions(ctx, userID)
	if err != nil {
		return Actor{}, fmt.Errorf("list user permissions: %w", err)
	}
	permissions := make(map[string]bool, len(permissionKeys))
	for _, key := range permissionKeys {
		permissions[key] = true
	}
	return Actor{ID: userID, Status: current.Status, RoleKeys: roleKeys, Permissions: permissions}, nil
}

func auditMetadata(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		return []byte(`{"error":"metadata_marshal_failed"}`)
	}
	return data
}
```

Implement these concrete `PostgresStore` and `postgresTxStore` methods in the same file:

```go
type postgresTxStore struct {
	queries *store.Queries
}

func (s *PostgresStore) GetCurrentUser(ctx context.Context, userID int64) (CurrentUser, error) {
	row, err := s.queries.GetCurrentUser(ctx, userID)
	if err != nil {
		return CurrentUser{}, fmt.Errorf("get current user: %w", err)
	}
	current := CurrentUser{
		ID: row.ID,
		Username: row.Username,
		DisplayName: row.DisplayName,
		Locale: row.Locale,
		Status: UserStatus(row.Status),
		IsInitialSuperAdmin: row.IsInitialSuperAdmin,
	}
	roleKeys, err := s.queries.ListUserRoleKeys(ctx, userID)
	if err != nil {
		return CurrentUser{}, fmt.Errorf("list current user roles: %w", err)
	}
	permissions, err := s.queries.ListUserPermissions(ctx, userID)
	if err != nil {
		return CurrentUser{}, fmt.Errorf("list current user permissions: %w", err)
	}
	current.RoleKeys = roleKeys
	current.Permissions = permissions
	return current, nil
}

func (s *PostgresStore) ListRoles(ctx context.Context) ([]Role, error) {
	rows, err := s.queries.ListRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	roles := make([]Role, 0, len(rows))
	for _, row := range rows {
		roles = append(roles, mapRole(row.ID, row.Key, row.Alias, row.Description, row.IsSystem, row.IsDefault, row.IsDeletable, row.IsEnabled))
	}
	return roles, nil
}

func (s *PostgresStore) CreateRole(ctx context.Context, input RoleInput) (Role, error) {
	row, err := s.queries.CreateRole(ctx, store.CreateRoleParams{
		Key: input.Key,
		Alias: input.Alias,
		Description: input.Description,
	})
	if err != nil {
		return Role{}, fmt.Errorf("create role: %w", err)
	}
	return mapRole(row.ID, row.Key, row.Alias, row.Description, row.IsSystem, row.IsDefault, row.IsDeletable, row.IsEnabled), nil
}

func (s *PostgresStore) UpdateRole(ctx context.Context, roleKey string, input RoleInput) (Role, error) {
	row, err := s.queries.UpdateRoleAlias(ctx, store.UpdateRoleAliasParams{
		Key: roleKey,
		Alias: input.Alias,
		Description: input.Description,
	})
	if err != nil {
		return Role{}, fmt.Errorf("update role: %w", err)
	}
	return mapRole(row.ID, row.Key, row.Alias, row.Description, row.IsSystem, row.IsDefault, row.IsDeletable, row.IsEnabled), nil
}

func (s *PostgresStore) DeleteRole(ctx context.Context, roleKey string) error {
	if err := s.queries.DeleteRoleByKey(ctx, roleKey); err != nil {
		return fmt.Errorf("delete role: %w", err)
	}
	return nil
}

func (s *PostgresStore) ReplaceRolePermissions(ctx context.Context, actorUserID int64, roleKey string, permissions []string) error {
	role, err := s.queries.GetRoleByKey(ctx, roleKey)
	if err != nil {
		return fmt.Errorf("get role for permissions: %w", err)
	}
	if err := s.queries.DeleteRolePermissions(ctx, role.ID); err != nil {
		return fmt.Errorf("delete role permissions: %w", err)
	}
	for _, permission := range permissions {
		if err := s.queries.AddRolePermission(ctx, store.AddRolePermissionParams{
			RoleID: role.ID,
			PermissionKey: permission,
		}); err != nil {
			return fmt.Errorf("add role permission %s: %w", permission, err)
		}
	}
	metadata := auditMetadata(map[string]any{"roleKey": roleKey, "permissions": permissions})
	return s.queries.CreateAuditEvent(ctx, store.CreateAuditEventParams{
		ActorUserID: actorUserID,
		TargetUserID: 0,
		Action: "role.permissions.replace",
		Metadata: metadata,
	})
}

func (s *postgresTxStore) AnyUserExists(ctx context.Context) (bool, error) {
	return s.queries.AnyUserExists(ctx)
}

func (s *postgresTxStore) CreateUser(ctx context.Context, input CreateUserInput) (CurrentUser, error) {
	row, err := s.queries.CreateUser(ctx, store.CreateUserParams{
		Username: input.Username,
		Email: input.Email,
		DisplayName: input.DisplayName,
		Locale: input.Locale,
		IsInitialSuperAdmin: input.IsInitialSuperAdmin,
	})
	if err != nil {
		return CurrentUser{}, fmt.Errorf("create user: %w", err)
	}
	return CurrentUser{
		ID: row.ID,
		Username: row.Username,
		DisplayName: row.DisplayName,
		Locale: row.Locale,
		Status: UserStatus(row.Status),
		IsInitialSuperAdmin: row.IsInitialSuperAdmin,
	}, nil
}

func mapRole(id int64, key, alias, description string, isSystem, isDefault, isDeletable, isEnabled bool) Role {
	return Role{
		ID: id,
		Key: key,
		Alias: alias,
		Description: description,
		IsSystem: isSystem,
		IsDefault: isDefault,
		IsDeletable: isDeletable,
		IsEnabled: isEnabled,
	}
}
```

Add `CreateCredential`, `GetDefaultRole`, `GetRole`, and `AssignRole` on `postgresTxStore` with the same direct wrapping pattern:

```go
func (s *postgresTxStore) CreateCredential(ctx context.Context, userID int64, passwordHash string) error
func (s *postgresTxStore) GetDefaultRole(ctx context.Context) (Role, error)
func (s *postgresTxStore) GetRole(ctx context.Context, roleKey string) (Role, error)
func (s *postgresTxStore) AssignRole(ctx context.Context, userID int64, roleID int64) error
```

- [ ] **Step 5: Run sqlc and Go tests**

Run:

```bash
cd apps/api
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0 generate
go test ./...
```

Expected: generated query package compiles and all tests pass.

- [ ] **Step 6: Commit**

```bash
git add apps/api/internal/store/queries/identity.sql apps/api/internal/store/sqlc apps/api/internal/modules/identity/store.go apps/api/internal/modules/identity/postgres_store.go
git commit -m "feat(api): add identity store queries"
```

---

### Task 6: Implement Registration And Login Services

**Files:**
- Create: `apps/api/internal/modules/identity/service.go`
- Test: `apps/api/internal/modules/identity/service_test.go`

- [ ] **Step 1: Write service tests with an in-memory fake store**

Create `apps/api/internal/modules/identity/service_test.go` with tests named:

```go
func TestRegisterFirstUserAssignsSuperAdminAndMember(t *testing.T)
func TestRegisterSecondUserAssignsDefaultMember(t *testing.T)
func TestLoginRejectsWrongPassword(t *testing.T)
```

The fake store must:

- Implement `Store` and `TxStore`.
- Keep users, credentials, roles, and user role assignments in memory.
- Serialize `WithBootstrapTx` with a `sync.Mutex`.
- Seed `super_admin` and `member`.

Use these core assertions:

```go
if !slices.Contains(first.RoleKeys, RoleSuperAdmin) {
	t.Fatalf("expected first user to have super_admin, got %v", first.RoleKeys)
}
if !first.IsInitialSuperAdmin {
	t.Fatal("expected first user to be initial super admin")
}
if slices.Contains(second.RoleKeys, RoleSuperAdmin) {
	t.Fatalf("expected second user not to have super_admin, got %v", second.RoleKeys)
}
if !slices.Contains(second.RoleKeys, RoleMember) {
	t.Fatalf("expected second user to have member, got %v", second.RoleKeys)
}
```

- [ ] **Step 2: Run service tests to verify they fail**

Run:

```bash
cd apps/api
go test ./internal/modules/identity -run 'TestRegister|TestLogin' -count=1
```

Expected: FAIL because `Service`, `Register`, and `Login` do not exist.

- [ ] **Step 3: Implement service methods**

Create `apps/api/internal/modules/identity/service.go`:

```go
package identity

import (
	"context"
	"strings"
)

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

type RegisterInput struct {
	Username    string
	Email       string
	Password    string
	DisplayName string
	Locale      string
}

type LoginInput struct {
	Login    string
	Password string
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (CurrentUser, error) {
	passwordHash, err := HashPassword(input.Password)
	if err != nil {
		return CurrentUser{}, err
	}

	username := strings.TrimSpace(input.Username)
	email := strings.TrimSpace(input.Email)
	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = username
	}
	locale := strings.TrimSpace(input.Locale)
	if locale == "" {
		locale = "zh-CN"
	}

	var current CurrentUser
	err = s.store.WithBootstrapTx(ctx, func(ctx context.Context, tx TxStore) error {
		hasAnyUser, err := tx.AnyUserExists(ctx)
		if err != nil {
			return err
		}

		current, err = tx.CreateUser(ctx, CreateUserInput{
			Username:            username,
			Email:               email,
			DisplayName:         displayName,
			Locale:              locale,
			IsInitialSuperAdmin: !hasAnyUser,
		})
		if err != nil {
			return err
		}
		if err := tx.CreateCredential(ctx, current.ID, passwordHash); err != nil {
			return err
		}

		member, err := tx.GetDefaultRole(ctx)
		if err != nil {
			return err
		}
		if err := tx.AssignRole(ctx, current.ID, member.ID); err != nil {
			return err
		}
		current.RoleKeys = append(current.RoleKeys, member.Key)

		if !hasAnyUser {
			superAdmin, err := tx.GetRole(ctx, RoleSuperAdmin)
			if err != nil {
				return err
			}
			if err := tx.AssignRole(ctx, current.ID, superAdmin.ID); err != nil {
				return err
			}
			current.RoleKeys = append(current.RoleKeys, superAdmin.Key)
		}

		return nil
	})
	if err != nil {
		return CurrentUser{}, err
	}

	return current, nil
}
```

Add `Login(ctx, input)` using the store credential lookup, `VerifyPassword`, and `GetCurrentUser`. Keep wrong-login and wrong-password failures mapped to `ErrInvalidCredentials`.

- [ ] **Step 4: Run tests**

Run:

```bash
cd apps/api
go test ./internal/modules/identity -run 'TestRegister|TestLogin' -count=1
go test ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/internal/modules/identity/service.go apps/api/internal/modules/identity/service_test.go
git commit -m "feat(api): add identity registration service"
```

---

### Task 7: Add Auth HTTP Endpoints And Session Wiring

**Files:**
- Create: `apps/api/internal/modules/identity/http.go`
- Modify: `apps/api/internal/http/server.go`
- Modify: `apps/api/internal/http/errors.go`
- Modify: `apps/api/cmd/api/main.go`
- Test: `apps/api/internal/http/server_test.go`

- [ ] **Step 1: Extend HTTP tests**

Modify `apps/api/internal/http/server_test.go` with tests:

```go
func TestRegisterEndpointCreatesSession(t *testing.T)
func TestSessionEndpointRequiresAuth(t *testing.T)
```

Use an in-memory identity service fake or the service fake from Task 6. Assert:

```go
if resp.StatusCode != nethttp.StatusCreated {
	t.Fatalf("expected 201, got %d", resp.StatusCode)
}
if cookies := resp.Cookies(); len(cookies) == 0 {
	t.Fatal("expected session cookie")
}
```

For unauthenticated session:

```go
if resp.StatusCode != nethttp.StatusUnauthorized {
	t.Fatalf("expected 401, got %d", resp.StatusCode)
}
```

- [ ] **Step 2: Run HTTP tests to verify they fail**

Run:

```bash
cd apps/api
go test ./internal/http -run 'TestRegisterEndpointCreatesSession|TestSessionEndpointRequiresAuth' -count=1
```

Expected: FAIL because auth routes are not registered.

- [ ] **Step 3: Create identity HTTP handlers**

Create `apps/api/internal/modules/identity/http.go` with:

```go
package identity

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

const sessionUserIDKey = "user_id"

type Handler struct {
	service *Service
	sessions *session.Store
}

func NewHandler(service *Service, sessions *session.Store) *Handler {
	return &Handler{service: service, sessions: sessions}
}

func (h *Handler) RegisterRoutes(api fiber.Router) {
	auth := api.Group("/auth")
	auth.Post("/register", h.register)
	auth.Post("/login", h.login)
	auth.Post("/logout", h.logout)
	auth.Get("/session", h.session)
}

type registerRequest struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
	Locale      string `json:"locale"`
}

func (h *Handler) register(c fiber.Ctx) error {
	var req registerRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}

	current, err := h.service.Register(c.Context(), RegisterInput{
		Username: req.Username,
		Email: req.Email,
		Password: req.Password,
		DisplayName: req.DisplayName,
		Locale: req.Locale,
	})
	if err != nil {
		return mapIdentityError(err)
	}

	sess, err := h.sessions.Get(c)
	if err != nil {
		return err
	}
	sess.Set(sessionUserIDKey, current.ID)
	if err := sess.Save(); err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(current)
}

func mapIdentityError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		return fiber.NewError(fiber.StatusUnauthorized, "auth.invalid_credentials")
	case errors.Is(err, ErrPermissionDenied):
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	case errors.Is(err, ErrSystemRoleLocked):
		return fiber.NewError(fiber.StatusConflict, "role.system_role_locked")
	case errors.Is(err, ErrDefaultRoleLocked):
		return fiber.NewError(fiber.StatusConflict, "role.default_role_locked")
	case errors.Is(err, ErrInitialSuperAdminLocked):
		return fiber.NewError(fiber.StatusConflict, "user.initial_super_admin_locked")
	case errors.Is(err, ErrPasswordDoesNotMeetPolicy):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "auth.password_policy")
	default:
		return err
	}
}
```

Add `login`, `logout`, and `session` methods in the same file using these exact behaviors:

```go
type loginRequest struct {
	Login string `json:"login"`
	Password string `json:"password"`
}

func (h *Handler) login(c fiber.Ctx) error {
	var req loginRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "validation.invalid")
	}
	current, err := h.service.Login(c.Context(), LoginInput{Login: req.Login, Password: req.Password})
	if err != nil {
		return mapIdentityError(err)
	}
	sess, err := h.sessions.Get(c)
	if err != nil {
		return err
	}
	sess.Set(sessionUserIDKey, current.ID)
	if err := sess.Save(); err != nil {
		return err
	}
	return c.JSON(current)
}

func (h *Handler) logout(c fiber.Ctx) error {
	sess, err := h.sessions.Get(c)
	if err != nil {
		return err
	}
	if err := sess.Destroy(); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) session(c fiber.Ctx) error {
	sess, err := h.sessions.Get(c)
	if err != nil {
		return err
	}
	userID, ok := sess.Get(sessionUserIDKey).(int64)
	if !ok || userID == 0 {
		return fiber.NewError(fiber.StatusUnauthorized, "auth.required")
	}
	current, err := h.service.CurrentUser(c.Context(), userID)
	if err != nil {
		return mapIdentityError(err)
	}
	return c.JSON(current)
}
```

- [ ] **Step 4: Wire dependencies through the API app**

Modify `apps/api/internal/http/server.go` so `NewApp` accepts dependencies:

```go
type Dependencies struct {
	IdentityHandler interface {
		RegisterRoutes(api fiber.Router)
	}
}

func NewApp(cfg config.Config, logger *slog.Logger, deps Dependencies) *fiber.App
```

Keep existing health route behavior. After `api := app.Group("/api/v1")`, call:

```go
if deps.IdentityHandler != nil {
	deps.IdentityHandler.RegisterRoutes(api)
}
```

Modify existing tests to pass `Dependencies{}`.

- [ ] **Step 5: Improve error code mapping**

Modify `apps/api/internal/http/errors.go` so `fiber.NewError(status, "auth.required")` returns:

```json
{"code":"auth.required","message":"auth.required"}
```

instead of using the generic `http_error` code. Keep the current Simplified Chinese fallback only for internal errors.

- [ ] **Step 6: Wire main with Postgres store and Redis sessions**

Modify `apps/api/cmd/api/main.go`:

- Create PostgreSQL pool with `postgres.NewPool`.
- Create Redis storage with `redis.NewStorage`.
- Create Fiber session store using the Redis storage.
- Create `identity.NewService(identity.NewPostgresStore(pool))`.
- Create `identity.NewHandler(service, sessions)`.
- Pass `httpserver.Dependencies{IdentityHandler: identityHandler}` into `NewApp`.
- Close the pool during shutdown.

- [ ] **Step 7: Run tests**

Run:

```bash
cd apps/api
go test ./internal/http -count=1
go test ./...
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add apps/api/internal/modules/identity/http.go apps/api/internal/http/server.go apps/api/internal/http/errors.go apps/api/internal/http/server_test.go apps/api/cmd/api/main.go
git commit -m "feat(api): add auth session endpoints"
```

---

### Task 8: Add Role Management Service And API

**Files:**
- Modify: `apps/api/internal/modules/identity/service.go`
- Modify: `apps/api/internal/modules/identity/http.go`
- Test: `apps/api/internal/modules/identity/roles_test.go`
- Test: `apps/api/internal/http/server_test.go`

- [ ] **Step 1: Write role invariant tests**

Create `apps/api/internal/modules/identity/roles_test.go`:

```go
package identity

import "testing"

func TestMemberAliasCanChangeButRoleCannotBeDeleted(t *testing.T) {
	service, store := newTestService(t)
	admin := Actor{ID: 1, Status: UserStatusActive, RoleKeys: []string{RoleSuperAdmin}}

	role, err := service.UpdateRole(testContext(t), admin, RoleMember, RoleInput{
		Alias: "注册用户",
		Description: "所有开放注册用户",
	})
	if err != nil {
		t.Fatalf("UpdateRole returned error: %v", err)
	}
	if role.Key != RoleMember || role.Alias != "注册用户" {
		t.Fatalf("unexpected role after update: %#v", role)
	}

	err = service.DeleteRole(testContext(t), admin, RoleMember)
	if err != ErrDefaultRoleLocked {
		t.Fatalf("expected default role lock, got %v; store=%#v", err, store)
	}
}

func TestNonAdminCannotManageRoles(t *testing.T) {
	service, _ := newTestService(t)
	member := Actor{ID: 2, Status: UserStatusActive, Permissions: map[string]bool{}}

	_, err := service.CreateRole(testContext(t), member, RoleInput{
		Key: "moderator",
		Alias: "版主",
		Description: "管理内容",
	})
	if err != ErrPermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}
```

`newTestService` and `testContext` can live in `service_test.go` and be reused.

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
cd apps/api
go test ./internal/modules/identity -run 'TestMemberAlias|TestNonAdmin' -count=1
```

Expected: FAIL because role-management service methods do not exist.

- [ ] **Step 3: Implement role service methods**

Add to `apps/api/internal/modules/identity/service.go`:

```go
func (s *Service) CreateRole(ctx context.Context, actor Actor, input RoleInput) (Role, error) {
	if !actor.Can(PermissionRoleManage) {
		return Role{}, ErrPermissionDenied
	}
	return s.store.CreateRole(ctx, input)
}

func (s *Service) UpdateRole(ctx context.Context, actor Actor, roleKey string, input RoleInput) (Role, error) {
	if !actor.Can(PermissionRoleManage) {
		return Role{}, ErrPermissionDenied
	}
	input.Key = roleKey
	return s.store.UpdateRole(ctx, roleKey, input)
}

func (s *Service) DeleteRole(ctx context.Context, actor Actor, roleKey string) error {
	if !actor.Can(PermissionRoleManage) {
		return ErrPermissionDenied
	}
	if roleKey == RoleMember {
		return ErrDefaultRoleLocked
	}
	if roleKey == RoleSuperAdmin {
		return ErrSystemRoleLocked
	}
	return s.store.DeleteRole(ctx, roleKey)
}

func (s *Service) ReplaceRolePermissions(ctx context.Context, actor Actor, roleKey string, permissions []string) error {
	if !actor.Can(PermissionRoleManage) {
		return ErrPermissionDenied
	}
	if roleKey == RoleSuperAdmin {
		return ErrSystemRoleLocked
	}
	return s.store.ReplaceRolePermissions(ctx, actor.ID, roleKey, permissions)
}
```

- [ ] **Step 4: Add role HTTP routes**

In `apps/api/internal/modules/identity/http.go`, register:

```go
api.Get("/roles", h.listRoles)
api.Post("/roles", h.createRole)
api.Patch("/roles/:roleKey", h.updateRole)
api.Delete("/roles/:roleKey", h.deleteRole)
api.Put("/roles/:roleKey/permissions", h.replaceRolePermissions)
```

Each handler must load the actor from session, call the service, and return mapped identity errors.

- [ ] **Step 5: Run tests**

Run:

```bash
cd apps/api
go test ./internal/modules/identity -run 'TestMemberAlias|TestNonAdmin' -count=1
go test ./internal/http -count=1
go test ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/api/internal/modules/identity/service.go apps/api/internal/modules/identity/http.go apps/api/internal/modules/identity/roles_test.go apps/api/internal/http/server_test.go
git commit -m "feat(api): add role management"
```

---

### Task 9: Update OpenAPI Contract

**Files:**
- Modify: `contracts/openapi.yaml`

- [ ] **Step 1: Add auth and role schemas**

Add schemas named:

- `CurrentUser`
- `RegisterRequest`
- `LoginRequest`
- `Role`
- `RoleInput`
- `Problem`

`CurrentUser` must include:

```yaml
id:
  type: integer
  format: int64
username:
  type: string
displayName:
  type: string
locale:
  type: string
status:
  type: string
  enum: [active, disabled, banned]
isInitialSuperAdmin:
  type: boolean
roleKeys:
  type: array
  items:
    type: string
permissions:
  type: array
  items:
    type: string
```

- [ ] **Step 2: Add paths**

Add paths matching the spec:

- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/logout`
- `GET /auth/session`
- `GET /roles`
- `POST /roles`
- `PATCH /roles/{roleKey}`
- `DELETE /roles/{roleKey}`
- `PUT /roles/{roleKey}/permissions`

Use `401`, `403`, `409`, and `422` responses with `Problem`.

- [ ] **Step 3: Validate the contract shape**

Run:

```bash
rg -n "auth/register|auth/session|roles|CurrentUser|Problem" contracts/openapi.yaml
```

Expected: all names appear.

- [ ] **Step 4: Commit**

```bash
git add contracts/openapi.yaml
git commit -m "docs(api): document identity endpoints"
```

---

### Task 10: Add Nuxt Session, Registration, Login, And Admin Route Shells

**Files:**
- Create: `apps/web/app/composables/useAuthSession.ts`
- Create: `apps/web/app/middleware/admin.ts`
- Create: `apps/web/app/pages/register.vue`
- Create: `apps/web/app/pages/login.vue`
- Create: `apps/web/app/pages/admin/index.vue`
- Create: `apps/web/app/pages/admin/roles.vue`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`

- [ ] **Step 1: Add frontend auth/session composable**

Create `apps/web/app/composables/useAuthSession.ts`:

```ts
type CurrentUser = {
  id: number
  username: string
  displayName: string
  locale: string
  status: 'active' | 'disabled' | 'banned'
  isInitialSuperAdmin: boolean
  roleKeys: string[]
  permissions: string[]
}

export const useAuthSession = () => {
  const user = useState<CurrentUser | null>('auth:user', () => null)
  const pending = useState<boolean>('auth:pending', () => false)
  const apiBaseUrl = useRuntimeConfig().public.apiBaseUrl

  const refresh = async () => {
    pending.value = true
    try {
      user.value = await $fetch<CurrentUser>(`${apiBaseUrl}/auth/session`, {
        credentials: 'include'
      })
    } catch {
      user.value = null
    } finally {
      pending.value = false
    }
  }

  const can = (permission: string) => {
    return Boolean(user.value?.permissions.includes(permission) || user.value?.roleKeys.includes('super_admin'))
  }

  return { user, pending, refresh, can }
}
```

- [ ] **Step 2: Add admin middleware**

Create `apps/web/app/middleware/admin.ts`:

```ts
export default defineNuxtRouteMiddleware(async () => {
  const localePath = useLocalePath()
  const { user, refresh, can } = useAuthSession()

  if (!user.value) {
    await refresh()
  }

  if (!user.value) {
    return navigateTo(localePath('/login'))
  }

  if (!can('admin.access')) {
    return navigateTo(localePath('/'))
  }
})
```

- [ ] **Step 3: Add auth pages**

Create `register.vue` and `login.vue` using existing `SFInput`, `SFButton`, and `SFAlert` components. Both pages must:

- Use `useI18n()` for all visible text.
- Submit to `/api/v1/auth/register` or `/api/v1/auth/login`.
- Use `credentials: 'include'`.
- Call `refresh()` after success.
- Navigate to `/admin` when the user has `admin.access`, otherwise `/`.

- [ ] **Step 4: Add admin pages**

Create `apps/web/app/pages/admin/index.vue`:

```vue
<script setup lang="ts">
definePageMeta({ middleware: 'admin' })
const { t } = useI18n()

useSeoMeta({
  title: t('admin.home.metaTitle')
})
</script>

<template>
  <main class="page-shell">
    <section class="forum-board">
      <p class="eyebrow">{{ t('admin.home.eyebrow') }}</p>
      <h1>{{ t('admin.home.title') }}</h1>
      <p class="intro">{{ t('admin.home.intro') }}</p>
    </section>
  </main>
</template>
```

Create `apps/web/app/pages/admin/roles.vue` with the same middleware and a roles table shell. It may fetch `/api/v1/roles` and render role key, alias, and system/default badges.

- [ ] **Step 5: Add i18n keys**

Add Chinese keys under `auth`, `admin`, and `errors`:

```json
"auth": {
  "registerTitle": "注册账号",
  "loginTitle": "登录",
  "username": "用户名",
  "email": "邮箱",
  "password": "密码",
  "displayName": "显示名称",
  "submitRegister": "创建账号",
  "submitLogin": "登录"
},
"admin": {
  "home": {
    "metaTitle": "管理后台",
    "eyebrow": "SForum 管理",
    "title": "管理论坛用户组与权限",
    "intro": "后台使用同一套论坛账号体系，所有管理能力由 API 权限控制。"
  },
  "roles": {
    "metaTitle": "用户组",
    "title": "用户组",
    "key": "标识",
    "alias": "别名",
    "system": "系统",
    "default": "默认"
  }
},
"errors": {
  "permissionDenied": "你没有权限执行此操作。"
}
```

Add matching English keys with natural English copy.

- [ ] **Step 6: Run typecheck**

Run:

```bash
cd apps/web
bun run typecheck
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/web/app/composables/useAuthSession.ts apps/web/app/middleware/admin.ts apps/web/app/pages/register.vue apps/web/app/pages/login.vue apps/web/app/pages/admin/index.vue apps/web/app/pages/admin/roles.vue apps/web/i18n/locales/zh-CN.json apps/web/i18n/locales/en-US.json
git commit -m "feat(web): add auth and admin shells"
```

---

### Task 11: Add Repository-Level Identity Validation

**Files:**
- Create: `tests/validate-identity-ui.js`
- Modify: `scripts/test.sh`

- [ ] **Step 1: Create validation script**

Create `tests/validate-identity-ui.js`:

```js
import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const root = process.cwd()
const requiredFiles = [
  'apps/web/app/composables/useAuthSession.ts',
  'apps/web/app/middleware/admin.ts',
  'apps/web/app/pages/register.vue',
  'apps/web/app/pages/login.vue',
  'apps/web/app/pages/admin/index.vue',
  'apps/web/app/pages/admin/roles.vue'
]

for (const file of requiredFiles) {
  if (!existsSync(resolve(root, file))) {
    throw new Error(`Missing required identity UI file: ${file}`)
  }
}

const zh = JSON.parse(readFileSync(resolve(root, 'apps/web/i18n/locales/zh-CN.json'), 'utf8'))
const en = JSON.parse(readFileSync(resolve(root, 'apps/web/i18n/locales/en-US.json'), 'utf8'))

const requiredKeys = [
  ['auth', 'registerTitle'],
  ['auth', 'loginTitle'],
  ['admin', 'home', 'title'],
  ['admin', 'roles', 'title'],
  ['errors', 'permissionDenied']
]

function valueAt(object, path) {
  return path.reduce((current, key) => current?.[key], object)
}

for (const keyPath of requiredKeys) {
  if (!valueAt(zh, keyPath)) {
    throw new Error(`Missing zh-CN locale key: ${keyPath.join('.')}`)
  }
  if (!valueAt(en, keyPath)) {
    throw new Error(`Missing en-US locale key: ${keyPath.join('.')}`)
  }
}

console.log('Identity UI validation passed.')
```

- [ ] **Step 2: Wire validation into test script**

Modify `scripts/test.sh` after existing component validation scripts or after Go tests:

```bash
echo "Running identity UI validation..."
node tests/validate-identity-ui.js
```

- [ ] **Step 3: Run full test script**

Run:

```bash
./scripts/test.sh
```

Expected: Go tests pass, identity UI validation passes, and Nuxt typecheck runs when `apps/web/node_modules` exists.

- [ ] **Step 4: Commit**

```bash
git add tests/validate-identity-ui.js scripts/test.sh
git commit -m "test: validate identity ui wiring"
```

---

### Task 12: End-To-End Smoke Check

**Files:**
- No source changes expected unless smoke check exposes a defect.

- [ ] **Step 1: Run backend tests**

```bash
cd apps/api
go test ./...
```

Expected: PASS.

- [ ] **Step 2: Run frontend typecheck**

```bash
cd apps/web
bun run typecheck
```

Expected: PASS.

- [ ] **Step 3: Run repository test script**

```bash
./scripts/test.sh
```

Expected: PASS.

- [ ] **Step 4: Run migrations against local Compose database**

Start the stack if needed:

```bash
./scripts/dev.sh --build
```

In another shell:

```bash
cd apps/api
go run ./cmd/migrate
```

Expected: migration command logs `migrations complete`.

- [ ] **Step 5: Smoke register first user**

With the API running through Nuxt proxy or direct API service, run:

```bash
curl -i -X POST http://127.0.0.1:3000/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","email":"admin@example.com","password":"correct horse battery staple","displayName":"Admin","locale":"zh-CN"}'
```

Expected:

- HTTP `201`.
- Response contains `"isInitialSuperAdmin":true`.
- Response role keys contain `super_admin` and `member`.
- Response sets a session cookie.

- [ ] **Step 6: Smoke register second user**

```bash
curl -i -X POST http://127.0.0.1:3000/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"member1","email":"member1@example.com","password":"correct horse battery staple","displayName":"Member One","locale":"zh-CN"}'
```

Expected:

- HTTP `201`.
- Response contains `"isInitialSuperAdmin":false`.
- Response role keys contain `member`.
- Response role keys do not contain `super_admin`.

- [ ] **Step 7: Commit smoke-fix changes only if needed**

If smoke testing exposes source changes, commit only the files changed to fix those defects:

```bash
git add <fixed-files>
git commit -m "fix: complete identity smoke behavior"
```

If no source changes are needed, do not create a commit.

---

## Self-Review Checklist

- Spec coverage:
  - One user system: Task 2 schema and Task 6 service.
  - Open registration: Task 6 service and Task 7 HTTP endpoint.
  - First user becomes protected `super_admin`: Task 2 schema invariants and Task 6 registration tests.
  - Later users receive `member`: Task 6 registration tests.
  - `member` alias editable but undeletable: Task 8 role invariant tests.
  - Custom roles/user groups: Task 8 service/API and Task 10 UI shell.
  - Redis-backed sessions: Task 1 infrastructure and Task 7 HTTP wiring.
  - API is final permission authority: Task 3 policy helpers and Task 8 service checks.
  - Nuxt protected admin routes: Task 10 middleware/pages.
  - Stable OpenAPI contract: Task 9.
- Red-flag scan: passed for unfinished-marker terms and vague deferred steps.
- Type consistency:
  - Role keys are `super_admin` and `member`.
  - Permission keys use the action-style strings from the approved spec.
  - `CurrentUser`, `Actor`, `Role`, and `RoleInput` names are used consistently across backend and frontend.
