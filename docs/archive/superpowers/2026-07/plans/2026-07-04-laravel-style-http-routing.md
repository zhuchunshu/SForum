# Laravel-Style HTTP Routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor the Go/Fiber API so route registration follows the documented Laravel-inspired bootstrap/provider/routes structure.

**Architecture:** Keep `cmd/api/main.go` process-focused, add `internal/bootstrap` for application assembly, make `internal/http` accept an ordered list of route providers, and move identity route declarations into a module-owned `routes.go`. Dependencies remain explicit Go constructors.

**Tech Stack:** Go 1.25+, Fiber v3, pgx, Fiber session middleware, Redis storage.

---

### Task 1: HTTP Route Providers

**Files:**
- Modify: `apps/api/internal/http/server.go`
- Modify: `apps/api/internal/http/server_test.go`

- [ ] **Step 1: Write the failing test**

Add a test proving `NewApp` can register a generic route provider without a module-specific dependency field.

- [ ] **Step 2: Run the focused HTTP test**

Run: `go test ./internal/http -run TestNewAppRegistersRouteProviders -count=1`

Expected: FAIL because `Dependencies` does not yet expose `RouteProviders`.

- [ ] **Step 3: Implement the minimal HTTP route-provider interface**

Add `RouteProvider`, change `Dependencies` to hold `[]RouteProvider`, and register each provider under `/api/v1`.

- [ ] **Step 4: Run HTTP tests**

Run: `go test ./internal/http -count=1`

Expected: PASS.

### Task 2: Identity Provider And Routes

**Files:**
- Create: `apps/api/internal/modules/identity/provider.go`
- Create: `apps/api/internal/modules/identity/routes.go`
- Modify: `apps/api/internal/modules/identity/http.go`

- [ ] **Step 1: Move route declarations**

Move `Handler.RegisterRoutes` from `http.go` into `routes.go` without changing endpoint paths.

- [ ] **Step 2: Add identity module provider**

Create `NewProvider(store Store, sessions *session.Store) *Provider` and make `Provider.RegisterRoutes(api fiber.Router)` delegate to the module handler.

- [ ] **Step 3: Run identity and HTTP tests**

Run: `go test ./internal/modules/identity ./internal/http -count=1`

Expected: PASS.

### Task 3: API Bootstrap

**Files:**
- Create: `apps/api/internal/bootstrap/app.go`
- Modify: `apps/api/cmd/api/main.go`

- [ ] **Step 1: Add bootstrap constructor**

Create `bootstrap.NewAPI(ctx, cfg, logger)` that wires Postgres, Redis session storage, identity provider, and the Fiber app.

- [ ] **Step 2: Slim the process entrypoint**

Replace infrastructure wiring in `cmd/api/main.go` with the bootstrap call and defer cleanup.

- [ ] **Step 3: Run full API tests**

Run: `go test ./...`

Expected: PASS.

### Task 4: Final Verification

**Files:**
- All touched Go files

- [ ] **Step 1: Format Go code**

Run: `gofmt -w apps/api/cmd/api/main.go apps/api/internal/bootstrap/app.go apps/api/internal/http/server.go apps/api/internal/http/server_test.go apps/api/internal/modules/identity/http.go apps/api/internal/modules/identity/provider.go apps/api/internal/modules/identity/routes.go`

- [ ] **Step 2: Run full tests**

Run: `go test ./...`

Expected: PASS.
