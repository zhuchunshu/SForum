# Extension Plugin Runtime Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make enabled plugins actually usable through declared routes, lifecycle hooks, and controlled provider slots while keeping core API routes authoritative.

**Architecture:** Keep install/state management in `app/Models/Extensions`, add runtime process/route/hook/provider infrastructure under `app/Support/Extensions`, and let the controller enforce host-side route authorization before proxying. Uploaded themes remain outside this plan.

**Tech Stack:** Go Fiber v3, fasthttp, HashiCorp go-plugin, PostgreSQL-backed extension store, Nuxt/Vue admin UI, Bun tests, modular OpenAPI.

---

## File Structure

- Modify `apps/api/app/Models/Extensions/types.go`: manifest route/provider fields, backend protocol version, runtime status shape, new error codes.
- Modify `apps/api/app/Models/Extensions/service.go`: manifest validation, route matching, runtime lifecycle calls from enable/disable/list/get, lifecycle hook emission.
- Modify `apps/api/app/Models/Extensions/store.go`: keep current store interface; use `List` for startup reconciliation.
- Modify `apps/api/app/Models/Extensions/service_test.go`: model-level TDD for manifest validation and lifecycle runtime behavior.
- Create `apps/api/app/Support/Extensions/types.go`: runtime interfaces shared by manager, gateway, hooks, and provider registry.
- Create `apps/api/app/Support/Extensions/route_gateway.go`: fasthttp localhost proxy gateway.
- Create `apps/api/app/Support/Extensions/route_gateway_test.go`: gateway proxy and timeout tests.
- Create `apps/api/app/Support/Extensions/manager.go`: runtime manager with status, start/stop/restart, and startup reconciliation.
- Create `apps/api/app/Support/Extensions/manager_test.go`: process-independent manager tests.
- Create `apps/api/app/Support/Extensions/protocol.go`: HashiCorp go-plugin protocol and RPC client wrapper.
- Create `apps/api/app/Support/Extensions/protocol_test.go`: helper plugin handshake test.
- Create `apps/api/app/Support/Extensions/hooks.go`: typed hook bus.
- Create `apps/api/app/Support/Extensions/hooks_test.go`: observe/validate/mutate policy tests.
- Create `apps/api/app/Support/Extensions/providers.go`: provider slot registry with default/select/restore behavior.
- Create `apps/api/app/Support/Extensions/providers_test.go`: provider registry tests.
- Modify `apps/api/app/Http/Controllers/Extensions/controller.go`: route proxy handler, route actor loading, runtime error mapping.
- Modify `apps/api/app/Http/Controllers/Extensions/routes.go`: replace unavailable route handler with proxy handler.
- Modify `apps/api/app/Http/Controllers/Extensions/controller_test.go`: route authorization tests.
- Modify `apps/api/app/Providers/extensions.go`: inject extension service/runtime/gateway.
- Modify `apps/api/bootstrap/app.go`: construct runtime manager, reconcile enabled plugins, close runtime on API shutdown.
- Modify `contracts/openapi/paths/extensions.yaml`: document plugin route namespace.
- Modify `contracts/openapi/schemas/extensions.yaml`: document manifest additions and runtime status.
- Modify `apps/web/app/utils/adminExtensions.ts`: route/hook/provider counts and runtime status helpers.
- Modify `apps/web/tests/adminExtensions.test.ts`: frontend helper tests.
- Modify `apps/web/app/pages/admin/extensions/plugins.vue`: runtime health and restart button.
- Modify `apps/web/app/composables/useAdminExtensionsManager.ts`: restart action call.
- Modify `apps/web/i18n/locales/zh-CN.json` and `apps/web/i18n/locales/en-US.json`: runtime labels.
- Modify `knowledge/modules/extensions.md` and add `knowledge/sessions/2026-07-06-extension-plugin-runtime.md`.

## Task 1: Manifest Schema And Validation

**Files:**
- Modify: `apps/api/app/Models/Extensions/types.go`
- Modify: `apps/api/app/Models/Extensions/service.go`
- Test: `apps/api/app/Models/Extensions/service_test.go`
- Modify: `contracts/openapi/schemas/extensions.yaml`

- [ ] **Step 1: Write failing manifest validation tests**

Add these tests after `TestServiceInstallArchiveRejectsReservedDefaultThemeID` in `apps/api/app/Models/Extensions/service_test.go`:

```go
func TestServiceInstallArchiveValidatesRuntimeManifestDeclarations(t *testing.T) {
	service := NewService(&fakeExtensionStore{}, t.TempDir())
	actor := extensionManager()

	valid := validManifest("runtime.plugin", TypePlugin)
	_, err := service.InstallArchive(context.Background(), actor, ArchiveInput{
		FileName: "runtime.zip",
		Data: extensionArchive(t, valid,
			zipFile{name: "backend/plugin", body: "#!/bin/sh\n"},
		),
	})
	if err != nil {
		t.Fatalf("expected runtime manifest to install, got %v", err)
	}

	cases := []struct {
		name     string
		manifest string
	}{
		{
			name: "unsafe route path",
			manifest: `{
				"id":"bad.route","name":"Bad Route","version":"1.0.0","type":"plugin","sforumVersion":"^1.0.0",
				"routes":[{"path":"../escape","methods":["GET"]}]
			}`,
		},
		{
			name: "public write route",
			manifest: `{
				"id":"bad.public","name":"Bad Public","version":"1.0.0","type":"plugin","sforumVersion":"^1.0.0",
				"routes":[{"path":"/write","methods":["POST"],"access":"public"}]
			}`,
		},
		{
			name: "unknown hook",
			manifest: `{
				"id":"bad.hook","name":"Bad Hook","version":"1.0.0","type":"plugin","sforumVersion":"^1.0.0",
				"hooks":[{"name":"topic.destroyed"}]
			}`,
		},
		{
			name: "unknown provider",
			manifest: `{
				"id":"bad.provider","name":"Bad Provider","version":"1.0.0","type":"plugin","sforumVersion":"^1.0.0",
				"providers":[{"slot":"unknown.provider","label":"Unknown"}]
			}`,
		},
		{
			name: "permission route without manifest permission",
			manifest: `{
				"id":"bad.permission","name":"Bad Permission","version":"1.0.0","type":"plugin","sforumVersion":"^1.0.0",
				"routes":[{"path":"/admin","methods":["POST"],"access":"permission","permission":"extension.bad.manage"}]
			}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.InstallArchive(context.Background(), actor, ArchiveInput{
				FileName: "bad.zip",
				Data:     extensionArchive(t, tc.manifest),
			})
			if !errors.Is(err, ErrInvalidManifest) {
				t.Fatalf("expected invalid manifest, got %v", err)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd apps/api
go test ./app/Models/Extensions -run TestServiceInstallArchiveValidatesRuntimeManifestDeclarations -count=1
```

Expected: FAIL because `providers`, `access`, `permission`, `timeoutMs`, and `protocolVersion` are not modeled or validated yet.

- [ ] **Step 3: Extend manifest and runtime types**

In `apps/api/app/Models/Extensions/types.go`, add constants:

```go
	RuntimeStopped  = "stopped"
	RuntimeStarting = "starting"
	RuntimeRunning  = "running"
	RuntimeFailed   = "failed"

	RouteAccessPublic     = "public"
	RouteAccessLogin      = "login"
	RouteAccessPermission = "permission"

	CodeRouteNotFound         = "extension.route_not_found"
	CodeRouteMethodNotAllowed = "extension.route_method_not_allowed"
	CodeRuntimeUnavailable    = "extension.runtime_unavailable"
	CodeRuntimeFailed         = "extension.runtime_failed"
```

Extend the structs:

```go
type Manifest struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	Version       string              `json:"version"`
	Type          string              `json:"type"`
	SForumVersion string              `json:"sforumVersion"`
	Permissions   []string            `json:"permissions"`
	Settings      []ManifestSetting   `json:"settings"`
	Migrations    []ManifestMigration `json:"migrations"`
	Backend       ManifestBackend     `json:"backend"`
	Frontend      ManifestFrontend    `json:"frontend"`
	AdminPages    []ManifestAdminPage `json:"adminPages"`
	Routes        []ManifestRoute     `json:"routes"`
	Hooks         []ManifestHook      `json:"hooks"`
	Jobs          []ManifestJob       `json:"jobs"`
	Providers     []ManifestProvider  `json:"providers"`
}

type ManifestBackend struct {
	Entry           string `json:"entry"`
	RPC             string `json:"rpc"`
	ProtocolVersion int    `json:"protocolVersion,omitempty"`
}

type ManifestRoute struct {
	Path       string   `json:"path"`
	Methods    []string `json:"methods"`
	Access     string   `json:"access,omitempty"`
	Permission string   `json:"permission,omitempty"`
	TimeoutMS  int      `json:"timeoutMs,omitempty"`
}

type ManifestProvider struct {
	Slot      string `json:"slot"`
	Label     string `json:"label"`
	TimeoutMS int    `json:"timeoutMs,omitempty"`
}

type RuntimeStatus struct {
	State       string     `json:"state"`
	LastError   string     `json:"lastError,omitempty"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	RouteCount  int        `json:"routeCount"`
	HookCount   int        `json:"hookCount"`
	ProviderCount int      `json:"providerCount"`
}
```

Add `Runtime *RuntimeStatus ` + "`json:\"runtime,omitempty\"`" + ` to `Extension`.

- [ ] **Step 4: Implement normalization and validation**

In `apps/api/app/Models/Extensions/service.go`, extend `normalizeManifest`:

```go
	manifest.Backend.RPC = strings.TrimSpace(manifest.Backend.RPC)
	if manifest.Backend.ProtocolVersion == 0 && manifest.Backend.RPC != "" {
		manifest.Backend.ProtocolVersion = 1
	}
	for index := range manifest.Routes {
		manifest.Routes[index].Path = normalizeRoutePath(manifest.Routes[index].Path)
		manifest.Routes[index].Access = strings.ToLower(strings.TrimSpace(manifest.Routes[index].Access))
		manifest.Routes[index].Permission = strings.TrimSpace(manifest.Routes[index].Permission)
		for methodIndex := range manifest.Routes[index].Methods {
			manifest.Routes[index].Methods[methodIndex] = strings.ToUpper(strings.TrimSpace(manifest.Routes[index].Methods[methodIndex]))
		}
	}
	for index := range manifest.Hooks {
		manifest.Hooks[index].Name = strings.TrimSpace(manifest.Hooks[index].Name)
	}
	for index := range manifest.Providers {
		manifest.Providers[index].Slot = strings.TrimSpace(manifest.Providers[index].Slot)
		manifest.Providers[index].Label = strings.TrimSpace(manifest.Providers[index].Label)
	}
```

Add helpers near `safeArchivePath`:

```go
func normalizeRoutePath(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return path.Clean(value)
}

func knownHookPoint(name string) bool {
	switch name {
	case "extension.enabled", "extension.disabled", "user.registered", "topic.before_create", "topic.created", "attachment.uploaded":
		return true
	default:
		return false
	}
}

func knownProviderSlot(slot string) bool {
	switch slot {
	case "search.provider", "attachment.storage.provider", "human_verification.provider", "auth.risk.provider", "editor.sanitizer.provider":
		return true
	default:
		return false
	}
}

func manifestHasPermission(manifest Manifest, permission string) bool {
	for _, item := range manifest.Permissions {
		if strings.TrimSpace(item) == permission {
			return true
		}
	}
	return false
}
```

Extend `validateManifest`:

```go
	if manifest.Backend.RPC != "" && manifest.Backend.RPC != "hashicorp-go-plugin" {
		return ErrInvalidManifest
	}
	if manifest.Backend.ProtocolVersion < 0 || manifest.Backend.ProtocolVersion > 1 {
		return ErrInvalidManifest
	}
	for _, route := range manifest.Routes {
		if route.Path == "" || !strings.HasPrefix(route.Path, "/") || strings.Contains(route.Path, "..") {
			return ErrInvalidManifest
		}
		access := route.Access
		if access == "" {
			access = RouteAccessLogin
		}
		if access != RouteAccessPublic && access != RouteAccessLogin && access != RouteAccessPermission {
			return ErrInvalidManifest
		}
		if len(route.Methods) == 0 {
			return ErrInvalidManifest
		}
		for _, method := range route.Methods {
			switch method {
			case "GET", "HEAD", "OPTIONS", "POST", "PUT", "PATCH", "DELETE":
			default:
				return ErrInvalidManifest
			}
			if access == RouteAccessPublic && method != "GET" && method != "HEAD" && method != "OPTIONS" {
				return ErrInvalidManifest
			}
		}
		if access == RouteAccessPermission && (route.Permission == "" || !manifestHasPermission(manifest, route.Permission)) {
			return ErrInvalidManifest
		}
		if route.TimeoutMS < 0 {
			return ErrInvalidManifest
		}
	}
	for _, hook := range manifest.Hooks {
		if !knownHookPoint(hook.Name) {
			return ErrInvalidManifest
		}
	}
	for _, provider := range manifest.Providers {
		if provider.Label == "" || !knownProviderSlot(provider.Slot) || provider.TimeoutMS < 0 {
			return ErrInvalidManifest
		}
	}
```

- [ ] **Step 5: Update OpenAPI schema**

In `contracts/openapi/schemas/extensions.yaml`, add route fields `access`, `permission`, `timeoutMs`, backend `protocolVersion`, `providers`, and `runtime`.

- [ ] **Step 6: Run test to verify it passes**

Run:

```bash
cd apps/api
go test ./app/Models/Extensions -run TestServiceInstallArchiveValidatesRuntimeManifestDeclarations -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/api/app/Models/Extensions/types.go apps/api/app/Models/Extensions/service.go apps/api/app/Models/Extensions/service_test.go contracts/openapi/schemas/extensions.yaml
git commit -m "feat(extensions): validate runtime manifest declarations"
```

## Task 2: Runtime Lifecycle Contract In The Extension Service

**Files:**
- Modify: `apps/api/app/Models/Extensions/store.go`
- Modify: `apps/api/app/Models/Extensions/service.go`
- Test: `apps/api/app/Models/Extensions/service_test.go`

- [ ] **Step 1: Write failing lifecycle tests**

Add tests near `TestServiceEnableRunsPluginPreflightBeforeStatusChange`:

```go
func TestServiceEnableStartsRuntimeAndRollsBackOnStartFailure(t *testing.T) {
	expected := errors.New("bind failed")
	store := &fakeExtensionStore{items: map[string]Extension{
		"demo.plugin": installedExtension("demo.plugin", TypePlugin, ManifestBackend{Entry: "backend/plugin"}),
	}}
	runtime := &fakeRuntimeManager{startErr: expected}
	service := NewServiceWithRuntime(store, t.TempDir(), runtime, nil)

	_, err := service.Enable(context.Background(), extensionManager(), "demo.plugin")
	if !errors.Is(err, ErrRuntimeFailed) {
		t.Fatalf("expected runtime failure, got %v", err)
	}
	if store.enabledID != "demo.plugin" || store.disabledID != "demo.plugin" {
		t.Fatalf("expected enable then rollback disable, enabled=%q disabled=%q", store.enabledID, store.disabledID)
	}
	if len(runtime.started) != 1 || runtime.started[0] != "demo.plugin" {
		t.Fatalf("expected runtime start attempt, got %#v", runtime.started)
	}
	if last := store.events[len(store.events)-1]; last.Action != EventEnableFailed || last.Message == "" {
		t.Fatalf("expected enable failure event, got %#v", store.events)
	}
}

func TestServiceDisableStopsRuntimeAndListDecoratesRuntimeStatus(t *testing.T) {
	store := &fakeExtensionStore{items: map[string]Extension{
		"demo.plugin": installedExtension("demo.plugin", TypePlugin, ManifestBackend{Entry: "backend/plugin"}),
	}}
	store.items["demo.plugin"] = extensionWithStatus(store.items["demo.plugin"], StatusEnabled)
	runtime := &fakeRuntimeManager{statuses: map[string]RuntimeStatus{
		"demo.plugin": {State: RuntimeRunning, RouteCount: 1, HookCount: 1, ProviderCount: 1},
	}}
	service := NewServiceWithRuntime(store, t.TempDir(), runtime, nil)

	items, err := service.List(context.Background(), extensionManager())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if items[0].Runtime == nil || items[0].Runtime.State != RuntimeRunning {
		t.Fatalf("expected decorated runtime status, got %#v", items[0].Runtime)
	}

	_, err = service.Disable(context.Background(), extensionManager(), "demo.plugin")
	if err != nil {
		t.Fatalf("Disable returned error: %v", err)
	}
	if len(runtime.stopped) != 1 || runtime.stopped[0] != "demo.plugin" {
		t.Fatalf("expected runtime stop, got %#v", runtime.stopped)
	}
}
```

Add helpers at the bottom:

```go
func extensionWithStatus(item Extension, status string) Extension {
	item.Status = status
	return item
}

type fakeRuntimeManager struct {
	err      error
	startErr error
	started  []string
	stopped  []string
	statuses map[string]RuntimeStatus
}

func (r *fakeRuntimeManager) Check(context.Context, Extension) error {
	return r.err
}

func (r *fakeRuntimeManager) Start(_ context.Context, extension Extension) error {
	r.started = append(r.started, extension.ID)
	return r.startErr
}

func (r *fakeRuntimeManager) Stop(_ context.Context, extension Extension) error {
	r.stopped = append(r.stopped, extension.ID)
	return nil
}

func (r *fakeRuntimeManager) Status(_ context.Context, extension Extension) RuntimeStatus {
	if r.statuses != nil {
		if status, ok := r.statuses[extension.ID]; ok {
			return status
		}
	}
	return RuntimeStatus{State: RuntimeStopped}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd apps/api
go test ./app/Models/Extensions -run 'TestServiceEnableStartsRuntimeAndRollsBackOnStartFailure|TestServiceDisableStopsRuntimeAndListDecoratesRuntimeStatus' -count=1
```

Expected: FAIL because `NewServiceWithRuntime`, `ErrRuntimeFailed`, and lifecycle calls do not exist.

- [ ] **Step 3: Add runtime manager contract**

In `apps/api/app/Models/Extensions/store.go`, add:

```go
type RuntimeManager interface {
	RuntimePreflight
	Start(ctx context.Context, extension Extension) error
	Stop(ctx context.Context, extension Extension) error
	Status(ctx context.Context, extension Extension) RuntimeStatus
}
```

In `types.go`, add:

```go
ErrRuntimeFailed = errors.New("extensions: runtime failed")
```

- [ ] **Step 4: Wire runtime manager into service**

In `service.go`, change the service field from `runtime RuntimePreflight` to `runtime RuntimeManager`. Add:

```go
func NewServiceWithRuntime(store Store, extensionRoot string, runtime RuntimeManager, themeBuilder ThemeBuilder) *Service {
	return NewServiceWithBuiltinsAndRuntime(store, extensionRoot, "", runtime, themeBuilder)
}

func NewServiceWithBuiltinsAndRuntime(store Store, extensionRoot string, builtinRoot string, runtime RuntimeManager, themeBuilder ThemeBuilder) *Service {
	if strings.TrimSpace(extensionRoot) == "" {
		extensionRoot = "storage/extensions"
	}
	if runtime == nil {
		runtime = LocalRuntimeManager{}
	}
	if themeBuilder == nil {
		themeBuilder = LocalThemeBuilder{}
	}
	return &Service{store: store, extensionRoot: extensionRoot, builtinRoot: strings.TrimSpace(builtinRoot), runtime: runtime, themeBuilder: themeBuilder}
}
```

Keep old constructors by routing through the new one. Add:

```go
type LocalRuntimeManager struct {
	LocalRuntimePreflight
}

func (LocalRuntimeManager) Start(context.Context, Extension) error {
	return nil
}

func (LocalRuntimeManager) Stop(context.Context, Extension) error {
	return nil
}

func (LocalRuntimeManager) Status(_ context.Context, extension Extension) RuntimeStatus {
	return RuntimeStatus{
		State:         RuntimeStopped,
		RouteCount:    len(extension.Manifest.Routes),
		HookCount:     len(extension.Manifest.Hooks),
		ProviderCount: len(extension.Manifest.Providers),
	}
}
```

- [ ] **Step 5: Start/rollback/stop and decorate**

In `Enable`, after `store.Enable`, call `runtime.Start`. If start fails, call `store.Disable`, record `EventEnableFailed`, and return `ErrRuntimeFailed` wrapped with the runtime error. In `Disable`, call `runtime.Stop` after the store disables the plugin. In `List`, call a helper:

```go
func (s *Service) decorateRuntime(ctx context.Context, item Extension) Extension {
	if item.Type == TypePlugin && s.runtime != nil {
		status := s.runtime.Status(ctx, item)
		item.Runtime = &status
	}
	return item
}
```

Apply the helper to `List`, `Enable`, `Disable`, `VerifyExtension`, and `InstallArchive` return values where the item is a plugin.

- [ ] **Step 6: Run tests**

```bash
cd apps/api
go test ./app/Models/Extensions -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/api/app/Models/Extensions/store.go apps/api/app/Models/Extensions/types.go apps/api/app/Models/Extensions/service.go apps/api/app/Models/Extensions/service_test.go
git commit -m "feat(extensions): connect plugin runtime lifecycle"
```

## Task 3: Route Matching And Controller Authorization

**Files:**
- Modify: `apps/api/app/Models/Extensions/service.go`
- Modify: `apps/api/app/Models/Extensions/types.go`
- Modify: `apps/api/app/Http/Controllers/Extensions/controller.go`
- Modify: `apps/api/app/Http/Controllers/Extensions/routes.go`
- Test: `apps/api/app/Http/Controllers/Extensions/controller_test.go`

- [ ] **Step 1: Write failing controller route tests**

Add this test to `controller_test.go`:

```go
func TestControllerProxiesOnlyDeclaredPluginRoutesAfterHostAuthorization(t *testing.T) {
	app, manager, store := newExtensionTestApp()
	store.items["demo.plugin"] = extensions.Extension{
		ID:      "demo.plugin",
		Name:    "Demo Plugin",
		Version: "1.0.0",
		Type:    extensions.TypePlugin,
		Status:  extensions.StatusEnabled,
		Manifest: extensions.Manifest{
			ID:            "demo.plugin",
			Name:          "Demo Plugin",
			Version:       "1.0.0",
			Type:          extensions.TypePlugin,
			SForumVersion: "^1.0.0",
			Permissions:   []string{"extension.demo.manage"},
			Routes: []extensions.ManifestRoute{
				{Path: "/hello", Methods: []string{"GET"}, Access: extensions.RouteAccessPublic},
				{Path: "/profile", Methods: []string{"GET"}, Access: extensions.RouteAccessLogin},
				{Path: "/admin/reindex", Methods: []string{"POST"}, Access: extensions.RouteAccessPermission, Permission: "extension.demo.manage"},
			},
		},
		InstalledAt: time.Now(),
		UpdatedAt:   time.Now(),
	}

	resp := performExtensionRequest(t, app, http.MethodGet, "/api/v1/extensions/demo.plugin/hello", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected public route 200, got %d", resp.StatusCode)
	}

	resp = performExtensionRequest(t, app, http.MethodGet, "/api/v1/extensions/demo.plugin/profile", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected login route 401, got %d", resp.StatusCode)
	}

	ordinaryCookie := loginExtensionUser(t, app, manager, 2)
	resp = performExtensionRequest(t, app, http.MethodPost, "/api/v1/extensions/demo.plugin/admin/reindex", ordinaryCookie)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected permission route 403, got %d", resp.StatusCode)
	}

	managerCookie := loginExtensionUser(t, app, manager, 1)
	resp = performExtensionRequest(t, app, http.MethodPost, "/api/v1/extensions/demo.plugin/admin/reindex", managerCookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected permission route 200, got %d", resp.StatusCode)
	}

	resp = performExtensionRequest(t, app, http.MethodDelete, "/api/v1/extensions/demo.plugin/hello", managerCookie)
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected undeclared method 405, got %d", resp.StatusCode)
	}

	resp = performExtensionRequest(t, app, http.MethodGet, "/api/v1/extensions/demo.plugin/missing", managerCookie)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected undeclared path 404, got %d", resp.StatusCode)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd apps/api
go test ./app/Http/Controllers/Extensions -run TestControllerProxiesOnlyDeclaredPluginRoutesAfterHostAuthorization -count=1
```

Expected: FAIL because plugin route dispatch still returns `extension.route_unavailable`.

- [ ] **Step 3: Add route match result types**

In `types.go`, add:

```go
type MatchedRoute struct {
	Extension Extension
	Route     ManifestRoute
	Path      string
}

var (
	ErrRouteNotFound         = errors.New("extensions: route not found")
	ErrRouteMethodNotAllowed = errors.New("extensions: route method not allowed")
	ErrRuntimeUnavailable    = errors.New("extensions: runtime unavailable")
)
```

- [ ] **Step 4: Add route matching helper**

In `service.go`, add:

```go
func (s *Service) MatchRoute(ctx context.Context, extensionID string, method string, routePath string) (MatchedRoute, error) {
	extension, err := s.store.Get(ctx, normalizeID(extensionID))
	if err != nil {
		return MatchedRoute{}, err
	}
	if extension.Type != TypePlugin || extension.Status != StatusEnabled {
		return MatchedRoute{}, ErrRouteNotFound
	}
	normalizedPath := normalizeRoutePath(routePath)
	pathExists := false
	for _, route := range extension.Manifest.Routes {
		if normalizeRoutePath(route.Path) != normalizedPath {
			continue
		}
		pathExists = true
		for _, allowed := range route.Methods {
			if strings.EqualFold(allowed, method) {
				return MatchedRoute{Extension: extension, Route: route, Path: normalizedPath}, nil
			}
		}
	}
	if pathExists {
		return MatchedRoute{}, ErrRouteMethodNotAllowed
	}
	return MatchedRoute{}, ErrRouteNotFound
}
```

- [ ] **Step 5: Replace unavailable route handler**

In `routes.go`, change:

```go
api.All("/extensions/:extensionId/*", h.routeUnavailable)
```

to:

```go
api.All("/extensions/:extensionId/*", h.proxyExtensionRoute)
```

In `controller.go`, replace `routeUnavailable` with policy checks. For this task, return `apphttp.OK(c, fiber.Map{"ok": true})` after authorization; Task 4 replaces it with real proxying.

```go
func (h *Controller) proxyExtensionRoute(c fiber.Ctx) error {
	routePath := "/" + c.Params("*")
	matched, err := h.service.MatchRoute(c.Context(), c.Params("extensionId"), c.Method(), routePath)
	if err != nil {
		return mapExtensionError(err)
	}
	actor, hasActor, err := h.optionalActor(c)
	if err != nil {
		return err
	}
	access := matched.Route.Access
	if access == "" {
		access = extensions.RouteAccessLogin
	}
	switch access {
	case extensions.RouteAccessLogin:
		if !hasActor {
			return fiber.NewError(fiber.StatusUnauthorized, "auth.required")
		}
	case extensions.RouteAccessPermission:
		if !hasActor {
			return fiber.NewError(fiber.StatusUnauthorized, "auth.required")
		}
		if !actor.Can(matched.Route.Permission) {
			return fiber.NewError(fiber.StatusForbidden, "permission.denied")
		}
	}
	return apphttp.OK(c, fiber.Map{"ok": true})
}

func (h *Controller) optionalActor(c fiber.Ctx) (identity.Actor, bool, error) {
	userID, ok, err := h.sessions.CurrentUserID(c)
	if err != nil || !ok {
		return identity.Actor{}, false, err
	}
	actor, err := h.users.LoadActor(c.Context(), userID)
	if err != nil {
		return identity.Actor{}, false, err
	}
	return actor, true, nil
}
```

Extend `mapExtensionError`:

```go
case errors.Is(err, extensions.ErrRouteNotFound):
	return fiber.NewError(fiber.StatusNotFound, extensions.CodeRouteNotFound)
case errors.Is(err, extensions.ErrRouteMethodNotAllowed):
	return fiber.NewError(fiber.StatusMethodNotAllowed, extensions.CodeRouteMethodNotAllowed)
case errors.Is(err, extensions.ErrRuntimeUnavailable):
	return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeRuntimeUnavailable)
case errors.Is(err, extensions.ErrRuntimeFailed):
	return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeRuntimeFailed)
```

- [ ] **Step 6: Run controller tests**

```bash
cd apps/api
go test ./app/Http/Controllers/Extensions -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/api/app/Models/Extensions/types.go apps/api/app/Models/Extensions/service.go apps/api/app/Http/Controllers/Extensions/controller.go apps/api/app/Http/Controllers/Extensions/routes.go apps/api/app/Http/Controllers/Extensions/controller_test.go
git commit -m "feat(extensions): authorize declared plugin routes"
```

## Task 4: fasthttp Route Gateway

**Files:**
- Create: `apps/api/app/Support/Extensions/types.go`
- Create: `apps/api/app/Support/Extensions/route_gateway.go`
- Test: `apps/api/app/Support/Extensions/route_gateway_test.go`
- Modify: `apps/api/app/Http/Controllers/Extensions/controller.go`

- [ ] **Step 1: Write failing route gateway tests**

Create `apps/api/app/Support/Extensions/route_gateway_test.go`:

```go
package extensionsruntime

import (
	"net"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
)

func TestRouteGatewayProxiesRequestAndTrustedHeaders(t *testing.T) {
	var receivedExtensionID string
	server := fasthttp.Server{Handler: func(ctx *fasthttp.RequestCtx) {
		receivedExtensionID = string(ctx.Request.Header.Peek("X-SForum-Extension-ID"))
		ctx.SetStatusCode(fasthttp.StatusCreated)
		ctx.SetBodyString("plugin-ok")
	}}
	listener := listenLocalhost(t)
	defer listener.Close()
	go server.Serve(listener)
	defer server.Shutdown()

	gateway := NewRouteGateway()
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.Header.SetMethod(fasthttp.MethodPost)
	req.SetRequestURI("/api/v1/extensions/demo.plugin/hello")
	req.SetBodyString("payload")

	err := gateway.Proxy(&ProxyInput{
		Request:     req,
		Response:    resp,
		ExtensionID: "demo.plugin",
		TargetBase:  "http://" + listener.Addr().String(),
		TargetPath:  "/hello",
		Timeout:     time.Second,
	})
	if err != nil {
		t.Fatalf("Proxy returned error: %v", err)
	}
	if resp.StatusCode() != fasthttp.StatusCreated || string(resp.Body()) != "plugin-ok" {
		t.Fatalf("unexpected proxy response status=%d body=%q", resp.StatusCode(), string(resp.Body()))
	}
	if receivedExtensionID != "demo.plugin" {
		t.Fatalf("expected trusted extension header, got %q", receivedExtensionID)
	}
}
```

Add local listener helper:

```go
func listenLocalhost(t *testing.T) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen localhost: %v", err)
	}
	return listener
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd apps/api
go test ./app/Support/Extensions -run TestRouteGatewayProxiesRequestAndTrustedHeaders -count=1
```

Expected: FAIL because `NewRouteGateway`, `ProxyInput`, and `Proxy` do not exist.

- [ ] **Step 3: Add runtime gateway types**

Create `apps/api/app/Support/Extensions/types.go`:

```go
package extensionsruntime

import (
	"time"

	"github.com/valyala/fasthttp"
)

type ProxyInput struct {
	Request     *fasthttp.Request
	Response    *fasthttp.Response
	ExtensionID string
	ActorID     string
	Locale      string
	TargetBase  string
	TargetPath  string
	Timeout     time.Duration
}

type RouteTarget struct {
	BaseURL string
}
```

- [ ] **Step 4: Implement gateway**

Create `apps/api/app/Support/Extensions/route_gateway.go`:

```go
package extensionsruntime

import (
	"net/url"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

type RouteGateway struct{}

func NewRouteGateway() *RouteGateway {
	return &RouteGateway{}
}

func (g *RouteGateway) Proxy(input *ProxyInput) error {
	target, err := url.Parse(input.TargetBase)
	if err != nil {
		return err
	}
	timeout := input.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	request := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(request)
	input.Request.CopyTo(request)
	request.SetRequestURI(input.TargetPath)
	request.URI().SetScheme(target.Scheme)
	request.URI().SetHost(target.Host)
	request.Header.Del("Cookie")
	request.Header.Set("X-SForum-Extension-ID", input.ExtensionID)
	if input.ActorID != "" {
		request.Header.Set("X-SForum-Actor-ID", input.ActorID)
	}
	if input.Locale != "" {
		request.Header.Set("X-SForum-Locale", input.Locale)
	}
	for _, header := range []string{"Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization", "TE", "Trailer", "Transfer-Encoding", "Upgrade"} {
		request.Header.Del(header)
	}
	client := fasthttp.HostClient{Addr: target.Host}
	if strings.EqualFold(target.Scheme, "https") {
		client.IsTLS = true
	}
	return client.DoTimeout(request, input.Response, timeout)
}
```

- [ ] **Step 5: Run gateway tests**

```bash
cd apps/api
go test ./app/Support/Extensions -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/api/app/Support/Extensions/types.go apps/api/app/Support/Extensions/route_gateway.go apps/api/app/Support/Extensions/route_gateway_test.go
git commit -m "feat(extensions): add plugin route proxy gateway"
```

## Task 5: Runtime Manager With Route Targets And Startup Reconciliation

**Files:**
- Create: `apps/api/app/Support/Extensions/manager.go`
- Test: `apps/api/app/Support/Extensions/manager_test.go`
- Modify: `apps/api/app/Models/Extensions/store.go`
- Modify: `apps/api/app/Models/Extensions/service.go`

- [ ] **Step 1: Write failing manager tests**

Create `apps/api/app/Support/Extensions/manager_test.go`:

```go
package extensionsruntime

import (
	"context"
	"errors"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

func TestManagerTracksStartStopStatusAndRouteTargets(t *testing.T) {
	manager := NewManager(ManagerConfig{})
	extension := runtimeExtension("demo.plugin")

	if err := manager.Start(context.Background(), extension); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	status := manager.Status(context.Background(), extension)
	if status.State != extensions.RuntimeRunning || status.RouteCount != 1 {
		t.Fatalf("unexpected running status: %#v", status)
	}
	target, ok := manager.RouteTarget("demo.plugin")
	if !ok || target.BaseURL == "" {
		t.Fatalf("expected route target, got %#v ok=%v", target, ok)
	}

	if err := manager.Stop(context.Background(), extension); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	status = manager.Status(context.Background(), extension)
	if status.State != extensions.RuntimeStopped {
		t.Fatalf("expected stopped status, got %#v", status)
	}
}

func TestManagerRecordsStartFailure(t *testing.T) {
	manager := NewManager(ManagerConfig{Starter: fakeStarter{err: errors.New("start failed")}})
	extension := runtimeExtension("broken.plugin")
	err := manager.Start(context.Background(), extension)
	if err == nil {
		t.Fatal("expected start failure")
	}
	status := manager.Status(context.Background(), extension)
	if status.State != extensions.RuntimeFailed || status.LastError == "" {
		t.Fatalf("expected failed status, got %#v", status)
	}
}
```

Add helpers in the same file:

```go
func runtimeExtension(id string) extensions.Extension {
	return extensions.Extension{
		ID:     id,
		Type:   extensions.TypePlugin,
		Status: extensions.StatusEnabled,
		Manifest: extensions.Manifest{
			ID:            id,
			Name:          "Demo Plugin",
			Version:       "1.0.0",
			Type:          extensions.TypePlugin,
			SForumVersion: "^1.0.0",
			Backend:       extensions.ManifestBackend{Entry: "backend/plugin", RPC: "hashicorp-go-plugin", ProtocolVersion: 1},
			Routes:        []extensions.ManifestRoute{{Path: "/hello", Methods: []string{"GET"}, Access: extensions.RouteAccessPublic}},
		},
	}
}

type fakeStarter struct {
	err error
}

func (s fakeStarter) Start(context.Context, extensions.Extension) (RouteTarget, error) {
	if s.err != nil {
		return RouteTarget{}, s.err
	}
	return RouteTarget{BaseURL: "http://127.0.0.1:43210"}, nil
}

func (s fakeStarter) Stop(context.Context, extensions.Extension) error {
	return nil
}
```

- [ ] **Step 2: Run tests to verify failure**

```bash
cd apps/api
go test ./app/Support/Extensions -run 'TestManagerTracksStartStopStatusAndRouteTargets|TestManagerRecordsStartFailure' -count=1
```

Expected: FAIL because `Manager` does not exist.

- [ ] **Step 3: Implement manager**

Create `apps/api/app/Support/Extensions/manager.go`:

```go
package extensionsruntime

import (
	"context"
	"sync"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

type Starter interface {
	Start(ctx context.Context, extension extensions.Extension) (RouteTarget, error)
	Stop(ctx context.Context, extension extensions.Extension) error
}

type ManagerConfig struct {
	Starter Starter
}

type Manager struct {
	mu       sync.RWMutex
	starter  Starter
	statuses map[string]extensions.RuntimeStatus
	targets  map[string]RouteTarget
}

func NewManager(config ManagerConfig) *Manager {
	starter := config.Starter
	if starter == nil {
		starter = localStarter{}
	}
	return &Manager{starter: starter, statuses: map[string]extensions.RuntimeStatus{}, targets: map[string]RouteTarget{}}
}

func (m *Manager) Check(context.Context, extensions.Extension) error {
	return nil
}

func (m *Manager) Start(ctx context.Context, extension extensions.Extension) error {
	m.setStatus(extension, extensions.RuntimeStatus{State: extensions.RuntimeStarting, RouteCount: len(extension.Manifest.Routes), HookCount: len(extension.Manifest.Hooks), ProviderCount: len(extension.Manifest.Providers)})
	target, err := m.starter.Start(ctx, extension)
	if err != nil {
		m.setStatus(extension, extensions.RuntimeStatus{State: extensions.RuntimeFailed, LastError: err.Error(), RouteCount: len(extension.Manifest.Routes), HookCount: len(extension.Manifest.Hooks), ProviderCount: len(extension.Manifest.Providers)})
		return err
	}
	now := time.Now().UTC()
	m.mu.Lock()
	m.targets[extension.ID] = target
	m.statuses[extension.ID] = extensions.RuntimeStatus{State: extensions.RuntimeRunning, StartedAt: &now, RouteCount: len(extension.Manifest.Routes), HookCount: len(extension.Manifest.Hooks), ProviderCount: len(extension.Manifest.Providers)}
	m.mu.Unlock()
	return nil
}

func (m *Manager) Stop(ctx context.Context, extension extensions.Extension) error {
	err := m.starter.Stop(ctx, extension)
	m.mu.Lock()
	delete(m.targets, extension.ID)
	m.statuses[extension.ID] = extensions.RuntimeStatus{State: extensions.RuntimeStopped, RouteCount: len(extension.Manifest.Routes), HookCount: len(extension.Manifest.Hooks), ProviderCount: len(extension.Manifest.Providers)}
	m.mu.Unlock()
	return err
}

func (m *Manager) Status(_ context.Context, extension extensions.Extension) extensions.RuntimeStatus {
	m.mu.RLock()
	status, ok := m.statuses[extension.ID]
	m.mu.RUnlock()
	if ok {
		return status
	}
	return extensions.RuntimeStatus{State: extensions.RuntimeStopped, RouteCount: len(extension.Manifest.Routes), HookCount: len(extension.Manifest.Hooks), ProviderCount: len(extension.Manifest.Providers)}
}

func (m *Manager) RouteTarget(extensionID string) (RouteTarget, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	target, ok := m.targets[extensionID]
	return target, ok
}

func (m *Manager) Reconcile(ctx context.Context, items []extensions.Extension) {
	for _, item := range items {
		if item.Type == extensions.TypePlugin && item.Status == extensions.StatusEnabled && item.Manifest.Backend.Entry != "" {
			_ = m.Start(ctx, item)
		}
	}
}

func (m *Manager) setStatus(extension extensions.Extension, status extensions.RuntimeStatus) {
	m.mu.Lock()
	m.statuses[extension.ID] = status
	m.mu.Unlock()
}

type localStarter struct{}

func (localStarter) Start(context.Context, extensions.Extension) (RouteTarget, error) {
	return RouteTarget{}, nil
}

func (localStarter) Stop(context.Context, extensions.Extension) error {
	return nil
}
```

- [ ] **Step 4: Run manager tests**

```bash
cd apps/api
go test ./app/Support/Extensions -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/app/Support/Extensions/manager.go apps/api/app/Support/Extensions/manager_test.go
git commit -m "feat(extensions): add plugin runtime manager"
```

## Task 6: Controller Proxy Integration

**Files:**
- Modify: `apps/api/app/Http/Controllers/Extensions/controller.go`
- Modify: `apps/api/app/Http/Controllers/Extensions/controller_test.go`
- Modify: `apps/api/app/Providers/extensions.go`

- [ ] **Step 1: Write failing controller proxy body test**

Extend `TestControllerProxiesOnlyDeclaredPluginRoutesAfterHostAuthorization` so public and permission route calls assert the response body contains `plugin-ok`, not `{"ok":true}`. Add a fake gateway to `newExtensionTestApp` that returns `plugin-ok`.

```go
type controllerFakeGateway struct{}

func (controllerFakeGateway) Proxy(c fiber.Ctx, input ProxyInput) error {
	c.Status(http.StatusAccepted)
	return c.SendString("plugin-ok")
}
```

- [ ] **Step 2: Run test to verify failure**

```bash
cd apps/api
go test ./app/Http/Controllers/Extensions -run TestControllerProxiesOnlyDeclaredPluginRoutesAfterHostAuthorization -count=1
```

Expected: FAIL because the controller returns a JSON placeholder from Task 3.

- [ ] **Step 3: Add controller gateway abstraction**

In `controller.go`, add:

```go
type ProxyInput struct {
	Matched extensions.MatchedRoute
	Actor   identity.Actor
	HasActor bool
}

type RouteGateway interface {
	Proxy(c fiber.Ctx, input ProxyInput) error
}
```

Change controller struct:

```go
type Controller struct {
	service  *extensions.Service
	users    identity.ActorStore
	sessions *authsession.Manager
	gateway  RouteGateway
}
```

Add constructor:

```go
func NewControllerWithGateway(service *extensions.Service, users identity.ActorStore, sessions *authsession.Manager, gateway RouteGateway) *Controller {
	return &Controller{service: service, users: users, sessions: sessions, gateway: gateway}
}
```

Keep `NewController` delegating to `NewControllerWithGateway(..., nil)`.

- [ ] **Step 4: Call gateway after authorization**

At the end of `proxyExtensionRoute`, replace placeholder OK with:

```go
	if h.gateway == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, extensions.CodeRuntimeUnavailable)
	}
	return h.gateway.Proxy(c, ProxyInput{Matched: matched, Actor: actor, HasActor: hasActor})
```

Use the fake gateway in controller tests.

- [ ] **Step 5: Run controller tests**

```bash
cd apps/api
go test ./app/Http/Controllers/Extensions -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/api/app/Http/Controllers/Extensions/controller.go apps/api/app/Http/Controllers/Extensions/controller_test.go apps/api/app/Providers/extensions.go
git commit -m "feat(extensions): proxy authorized plugin routes"
```

## Task 7: HashiCorp go-plugin Protocol Starter

**Files:**
- Modify: `apps/api/go.mod`
- Modify: `apps/api/go.sum`
- Create: `apps/api/app/Support/Extensions/protocol.go`
- Test: `apps/api/app/Support/Extensions/protocol_test.go`
- Modify: `apps/api/app/Support/Extensions/manager.go`

- [ ] **Step 1: Add dependency with the configured proxy**

Run:

```bash
cd apps/api
export https_proxy=http://127.0.0.1:7897 http_proxy=http://127.0.0.1:7897 all_proxy=socks5://127.0.0.1:7897
go get github.com/hashicorp/go-plugin@latest
go mod tidy
```

Expected: `go.mod` includes `github.com/hashicorp/go-plugin`.

- [ ] **Step 2: Write failing protocol unit test**

Create `apps/api/app/Support/Extensions/protocol_test.go`:

```go
package extensionsruntime

import (
	"context"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

func TestProtocolStarterRejectsUnsupportedRPC(t *testing.T) {
	starter := NewProtocolStarter(ProtocolStarterConfig{})
	extension := runtimeExtension("bad.protocol")
	extension.Manifest.Backend.RPC = "custom"
	_, err := starter.Start(context.Background(), extension)
	if err == nil {
		t.Fatal("expected unsupported protocol error")
	}
}

func TestProtocolStarterRequiresBackendEntry(t *testing.T) {
	starter := NewProtocolStarter(ProtocolStarterConfig{})
	extension := runtimeExtension("missing.entry")
	extension.Manifest.Backend.Entry = ""
	_, err := starter.Start(context.Background(), extension)
	if err == nil {
		t.Fatal("expected missing entry error")
	}
}

var _ Starter = (*ProtocolStarter)(nil)
var _ = extensions.TypePlugin
```

- [ ] **Step 3: Run test to verify failure**

```bash
cd apps/api
go test ./app/Support/Extensions -run 'TestProtocolStarterRejectsUnsupportedRPC|TestProtocolStarterRequiresBackendEntry' -count=1
```

Expected: FAIL because `ProtocolStarter` does not exist.

- [ ] **Step 4: Implement protocol starter skeleton**

Create `apps/api/app/Support/Extensions/protocol.go`:

```go
package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
	"os"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

var ErrUnsupportedProtocol = errors.New("unsupported plugin protocol")

type ProtocolStarterConfig struct{}

type ProtocolStarter struct{}

func NewProtocolStarter(ProtocolStarterConfig) *ProtocolStarter {
	return &ProtocolStarter{}
}

func (s *ProtocolStarter) Start(ctx context.Context, extension extensions.Extension) (RouteTarget, error) {
	if extension.Manifest.Backend.RPC != "" && extension.Manifest.Backend.RPC != "hashicorp-go-plugin" {
		return RouteTarget{}, ErrUnsupportedProtocol
	}
	entry := extension.Manifest.Backend.Entry
	if entry == "" {
		return RouteTarget{}, fmt.Errorf("backend entry is required")
	}
	path, ok := installedFilePathForRuntime(extension, entry)
	if !ok {
		return RouteTarget{}, extensions.ErrInvalidManifest
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return RouteTarget{}, fmt.Errorf("backend entry %s is not available", entry)
	}
	return RouteTarget{}, fmt.Errorf("hashicorp plugin handshake did not return a route target")
}

func (s *ProtocolStarter) Stop(context.Context, extensions.Extension) error {
	return nil
}

func installedFilePathForRuntime(extension extensions.Extension, manifestPath string) (string, bool) {
	return extensions.InstalledFilePathForRuntime(extension, manifestPath)
}
```

Expose `InstalledFilePathForRuntime` from `apps/api/app/Models/Extensions/service.go` as a small wrapper around the existing unexported `installedFilePath`.

- [ ] **Step 5: Extend protocol to perform real handshake**

Replace the skeleton `ProtocolStarter` with a client holder and protocol types:

```go
type ProtocolStarter struct {
	mu      sync.Mutex
	clients map[string]*plugin.Client
}

type PluginProtocol interface {
	Health() (PluginHealth, error)
	RouteTarget() (PluginRouteTarget, error)
	InvokeHook(PluginHookRequest) (PluginHookResponse, error)
}

type PluginHealth struct {
	OK bool
}

type PluginRouteTarget struct {
	BaseURL string
}

type PluginHookRequest struct {
	Name    string
	Payload map[string]any
}

type PluginHookResponse struct {
	OK      bool
	Reason  string
	Message string
}

var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "SFORUM_PLUGIN",
	MagicCookieValue: "sforum-plugin-v1",
}
```

Use this start flow:

```go
func (s *ProtocolStarter) Start(ctx context.Context, extension extensions.Extension) (RouteTarget, error) {
	if extension.Manifest.Backend.RPC != "" && extension.Manifest.Backend.RPC != "hashicorp-go-plugin" {
		return RouteTarget{}, ErrUnsupportedProtocol
	}
	path, ok := extensions.InstalledFilePathForRuntime(extension, extension.Manifest.Backend.Entry)
	if !ok {
		return RouteTarget{}, extensions.ErrInvalidManifest
	}
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return RouteTarget{}, fmt.Errorf("backend entry %s is not available", extension.Manifest.Backend.Entry)
	}
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: handshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"sforum-plugin-v1": &netRPCPlugin{},
		},
		Cmd:              exec.CommandContext(ctx, path),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolNetRPC},
	})
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return RouteTarget{}, err
	}
	raw, err := rpcClient.Dispense("sforum-plugin-v1")
	if err != nil {
		client.Kill()
		return RouteTarget{}, err
	}
	protocol := raw.(PluginProtocol)
	health, err := protocol.Health()
	if err != nil || !health.OK {
		client.Kill()
		if err != nil {
			return RouteTarget{}, err
		}
		return RouteTarget{}, fmt.Errorf("plugin health check failed")
	}
	target, err := protocol.RouteTarget()
	if err != nil || target.BaseURL == "" {
		client.Kill()
		if err != nil {
			return RouteTarget{}, err
		}
		return RouteTarget{}, fmt.Errorf("plugin route target is empty")
	}
	s.mu.Lock()
	s.clients[extension.ID] = client
	s.mu.Unlock()
	return RouteTarget{BaseURL: target.BaseURL}, nil
}
```

Use this stop flow:

```go
func (s *ProtocolStarter) Stop(_ context.Context, extension extensions.Extension) error {
	s.mu.Lock()
	client := s.clients[extension.ID]
	delete(s.clients, extension.ID)
	s.mu.Unlock()
	if client != nil {
		client.Kill()
	}
	return nil
}
```

Add a handshake test in `protocol_test.go` that builds a helper plugin binary under `t.TempDir()` and points `extension.PackagePath` at a fake uploaded package whose `files/backend/plugin` is that binary. The helper plugin implements `Health` and `RouteTarget` and returns `http://127.0.0.1:43123`. The assertion is:

```go
target, err := starter.Start(context.Background(), extension)
if err != nil {
	t.Fatalf("Start returned error: %v", err)
}
if target.BaseURL != "http://127.0.0.1:43123" {
	t.Fatalf("unexpected route target: %#v", target)
}
```

- [ ] **Step 6: Run support tests**

```bash
cd apps/api
go test ./app/Support/Extensions -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/api/go.mod apps/api/go.sum apps/api/app/Models/Extensions/service.go apps/api/app/Support/Extensions/protocol.go apps/api/app/Support/Extensions/protocol_test.go apps/api/app/Support/Extensions/manager.go
git commit -m "feat(extensions): add hashicorp plugin starter"
```

## Task 8: Startup Reconciliation And Shutdown Cleanup

**Files:**
- Modify: `apps/api/bootstrap/app.go`
- Modify: `apps/api/app/Providers/extensions.go`
- Test: `apps/api/bootstrap/app_test.go`

- [ ] **Step 1: Write failing bootstrap lifecycle test**

Add a small factory seam test in `apps/api/bootstrap/app_test.go`:

```go
func TestExtensionRuntimeFactoryCanBeReplacedForBootstrapTests(t *testing.T) {
	original := newExtensionRuntimeManager
	defer func() { newExtensionRuntimeManager = original }()

	called := false
	newExtensionRuntimeManager = func() extensionRuntime {
		called = true
		return fakeBootstrapExtensionRuntime{}
	}

	runtime := newExtensionRuntimeManager()
	runtime.Reconcile(context.Background(), []extensions.Extension{{
		ID:     "demo.plugin",
		Type:   extensions.TypePlugin,
		Status: extensions.StatusEnabled,
	}})
	runtime.Close(context.Background())

	if !called {
		t.Fatal("expected runtime factory replacement to be called")
	}
}

type fakeBootstrapExtensionRuntime struct{}

func (fakeBootstrapExtensionRuntime) Check(context.Context, extensions.Extension) error { return nil }
func (fakeBootstrapExtensionRuntime) Start(context.Context, extensions.Extension) error { return nil }
func (fakeBootstrapExtensionRuntime) Stop(context.Context, extensions.Extension) error { return nil }
func (fakeBootstrapExtensionRuntime) Status(context.Context, extensions.Extension) extensions.RuntimeStatus {
	return extensions.RuntimeStatus{State: extensions.RuntimeStopped}
}
func (fakeBootstrapExtensionRuntime) RouteTarget(string) (extensionsruntime.RouteTarget, bool) {
	return extensionsruntime.RouteTarget{}, false
}
func (fakeBootstrapExtensionRuntime) Reconcile(context.Context, []extensions.Extension) {}
func (fakeBootstrapExtensionRuntime) Close(context.Context) {}
```

- [ ] **Step 2: Run test to verify failure**

```bash
cd apps/api
go test ./bootstrap -run TestNewAPIReconcilesEnabledPluginRuntime -count=1
```

Expected: FAIL because bootstrap has no runtime factory seam.

- [ ] **Step 3: Add bootstrap runtime factory**

In `bootstrap/app.go`, add a small package-level factory:

```go
type extensionRuntime interface {
	extensions.RuntimeManager
	RouteTarget(extensionID string) (extensionsruntime.RouteTarget, bool)
	Reconcile(ctx context.Context, items []extensions.Extension)
	Close(ctx context.Context)
}

var newExtensionRuntimeManager = func() *extensionsruntime.Manager {
	return extensionsruntime.NewManager(extensionsruntime.ManagerConfig{
		Starter: extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{}),
	})
}
```

After `extensionService.SyncBuiltins(ctx)`, load extensions and reconcile:

```go
extensionRuntime := newExtensionRuntimeManager()
if items, err := extensionStore.List(ctx); err == nil {
	extensionRuntime.Reconcile(ctx, items)
}
```

Pass `extensionRuntime` into `providers.NewExtensionsProvider`.

- [ ] **Step 4: Close runtime on API close**

Add `Close(ctx)` to `Manager` that stops running plugins. Call it from `API.Close` before closing Redis/PostgreSQL resources.

- [ ] **Step 5: Run bootstrap tests**

```bash
cd apps/api
go test ./bootstrap -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/api/bootstrap/app.go apps/api/bootstrap/app_test.go apps/api/app/Providers/extensions.go apps/api/app/Support/Extensions/manager.go
git commit -m "feat(extensions): reconcile plugin runtime on startup"
```

## Task 9: Lifecycle Hook Bus

**Files:**
- Create: `apps/api/app/Support/Extensions/hooks.go`
- Test: `apps/api/app/Support/Extensions/hooks_test.go`
- Modify: `apps/api/app/Models/Extensions/service.go`
- Modify: `apps/api/app/Support/Extensions/manager.go`

- [ ] **Step 1: Write failing hook bus tests**

Create `apps/api/app/Support/Extensions/hooks_test.go`:

```go
package extensionsruntime

import (
	"context"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

func TestHookBusInvokesOnlyEnabledDeclaredHooks(t *testing.T) {
	calls := []string{}
	bus := NewHookBus(HookBusConfig{
		Invoker: HookInvokerFunc(func(_ context.Context, extension extensions.Extension, input HookInput) HookResult {
			calls = append(calls, extension.ID+":"+input.Name)
			return HookResult{OK: true}
		}),
	})
	bus.Register(runtimeExtension("demo.plugin"))
	bus.Emit(context.Background(), HookInput{Name: "extension.enabled"})
	bus.Emit(context.Background(), HookInput{Name: "topic.created"})
	if len(calls) != 1 || calls[0] != "demo.plugin:extension.enabled" {
		t.Fatalf("unexpected hook calls: %#v", calls)
	}
}
```

- [ ] **Step 2: Run test to verify failure**

```bash
cd apps/api
go test ./app/Support/Extensions -run TestHookBusInvokesOnlyEnabledDeclaredHooks -count=1
```

Expected: FAIL because `HookBus` does not exist.

- [ ] **Step 3: Implement hook bus**

Create `hooks.go`:

```go
package extensionsruntime

import (
	"context"
	"sync"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

type HookInput struct {
	Name    string
	Payload map[string]any
}

type HookResult struct {
	OK      bool
	Reason  string
	Message string
}

type HookInvoker interface {
	InvokeHook(ctx context.Context, extension extensions.Extension, input HookInput) HookResult
}

type HookInvokerFunc func(context.Context, extensions.Extension, HookInput) HookResult

func (fn HookInvokerFunc) InvokeHook(ctx context.Context, extension extensions.Extension, input HookInput) HookResult {
	return fn(ctx, extension, input)
}

type HookBusConfig struct {
	Invoker HookInvoker
}

type HookBus struct {
	mu       sync.RWMutex
	invoker  HookInvoker
	plugins  map[string]extensions.Extension
}

func NewHookBus(config HookBusConfig) *HookBus {
	return &HookBus{invoker: config.Invoker, plugins: map[string]extensions.Extension{}}
}

func (b *HookBus) Register(extension extensions.Extension) {
	b.mu.Lock()
	b.plugins[extension.ID] = extension
	b.mu.Unlock()
}

func (b *HookBus) Unregister(extensionID string) {
	b.mu.Lock()
	delete(b.plugins, extensionID)
	b.mu.Unlock()
}

func (b *HookBus) Emit(ctx context.Context, input HookInput) []HookResult {
	b.mu.RLock()
	plugins := make([]extensions.Extension, 0, len(b.plugins))
	for _, plugin := range b.plugins {
		plugins = append(plugins, plugin)
	}
	b.mu.RUnlock()
	results := []HookResult{}
	for _, plugin := range plugins {
		if !declaresHook(plugin, input.Name) || b.invoker == nil {
			continue
		}
		results = append(results, b.invoker.InvokeHook(ctx, plugin, input))
	}
	return results
}

func declaresHook(extension extensions.Extension, name string) bool {
	for _, hook := range extension.Manifest.Hooks {
		if hook.Name == name {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Emit lifecycle hooks from service**

After runtime start succeeds in `Enable`, emit `extension.enabled`. After runtime stop in `Disable`, emit `extension.disabled`. If the manager owns the hook bus, expose `EmitHook(ctx, input)` on the runtime manager contract.

- [ ] **Step 5: Run tests**

```bash
cd apps/api
go test ./app/Support/Extensions ./app/Models/Extensions -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/api/app/Support/Extensions/hooks.go apps/api/app/Support/Extensions/hooks_test.go apps/api/app/Support/Extensions/manager.go apps/api/app/Models/Extensions/service.go
git commit -m "feat(extensions): emit plugin lifecycle hooks"
```

## Task 10: Provider Registry, Admin UI, Contracts, And Knowledge

**Files:**
- Create: `apps/api/app/Support/Extensions/providers.go`
- Test: `apps/api/app/Support/Extensions/providers_test.go`
- Modify: `apps/web/app/utils/adminExtensions.ts`
- Modify: `apps/web/tests/adminExtensions.test.ts`
- Modify: `apps/web/app/pages/admin/extensions/plugins.vue`
- Modify: `apps/web/app/composables/useAdminExtensionsManager.ts`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`
- Modify: `contracts/openapi/paths/extensions.yaml`
- Modify: `contracts/openapi/schemas/extensions.yaml`
- Modify: `knowledge/modules/extensions.md`
- Add: `knowledge/sessions/2026-07-06-extension-plugin-runtime.md`

- [ ] **Step 1: Write failing provider registry test**

Create `providers_test.go`:

```go
package extensionsruntime

import "testing"

func TestProviderRegistryKeepsSelectsAndRestoresDefault(t *testing.T) {
	registry := NewProviderRegistry()
	if selected := registry.Selected("search.provider"); selected.ExtensionID != "" || selected.Label != "Built-in search" {
		t.Fatalf("expected built-in default, got %#v", selected)
	}
	registry.Select("search.provider", ProviderSelection{ExtensionID: "demo.plugin", Label: "Demo Search"})
	if selected := registry.Selected("search.provider"); selected.ExtensionID != "demo.plugin" {
		t.Fatalf("expected plugin provider, got %#v", selected)
	}
	registry.RestoreDefault("search.provider")
	if selected := registry.Selected("search.provider"); selected.ExtensionID != "" || selected.Label != "Built-in search" {
		t.Fatalf("expected restored built-in default, got %#v", selected)
	}
}
```

- [ ] **Step 2: Run test to verify failure**

```bash
cd apps/api
go test ./app/Support/Extensions -run TestProviderRegistryKeepsSelectsAndRestoresDefault -count=1
```

Expected: FAIL because provider registry does not exist.

- [ ] **Step 3: Implement provider registry**

Create `providers.go`:

```go
package extensionsruntime

import "sync"

type ProviderSelection struct {
	ExtensionID string
	Label       string
}

type ProviderRegistry struct {
	mu         sync.RWMutex
	defaults   map[string]ProviderSelection
	selections map[string]ProviderSelection
}

func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		defaults: map[string]ProviderSelection{
			"search.provider":             {Label: "Built-in search"},
			"attachment.storage.provider": {Label: "Built-in attachment storage"},
			"human_verification.provider": {Label: "Built-in human verification"},
			"auth.risk.provider":          {Label: "Built-in auth risk checks"},
			"editor.sanitizer.provider":   {Label: "Built-in sanitizer"},
		},
		selections: map[string]ProviderSelection{},
	}
}

func (r *ProviderRegistry) Selected(slot string) ProviderSelection {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if selected, ok := r.selections[slot]; ok {
		return selected
	}
	return r.defaults[slot]
}

func (r *ProviderRegistry) Select(slot string, selection ProviderSelection) {
	r.mu.Lock()
	r.selections[slot] = selection
	r.mu.Unlock()
}

func (r *ProviderRegistry) RestoreDefault(slot string) {
	r.mu.Lock()
	delete(r.selections, slot)
	r.mu.Unlock()
}
```

- [ ] **Step 4: Add frontend helper tests**

In `apps/web/tests/adminExtensions.test.ts`, add:

```ts
test('summarizes runtime declarations and running state', () => {
  const item = extension({
    id: 'runtime.plugin',
    name: 'Runtime Plugin',
    type: 'plugin',
    status: 'enabled',
    manifest: {
      routes: [{ path: '/hello', methods: ['GET'], access: 'public' }],
      hooks: [{ name: 'extension.enabled' }],
      providers: [{ slot: 'search.provider', label: 'Demo Search' }]
    },
    runtime: {
      state: 'running',
      routeCount: 1,
      hookCount: 1,
      providerCount: 1
    }
  })

  expect(runtimeStatusLabelKey(item)).toBe('admin.extensions.runtime.running')
  expect(runtimeCapabilitySummary(item)).toEqual({ routes: 1, hooks: 1, providers: 1 })
})
```

- [ ] **Step 5: Update frontend helpers**

In `apps/web/app/utils/adminExtensions.ts`, add provider and runtime types plus:

```ts
export function runtimeStatusLabelKey(item: AdminExtension) {
  return `admin.extensions.runtime.${item.runtime?.state || 'stopped'}`
}

export function runtimeCapabilitySummary(item: AdminExtension) {
  return {
    routes: item.runtime?.routeCount ?? item.manifest.routes?.length ?? 0,
    hooks: item.runtime?.hookCount ?? item.manifest.hooks?.length ?? 0,
    providers: item.runtime?.providerCount ?? item.manifest.providers?.length ?? 0
  }
}

export function canRestartPlugin(item: AdminExtension) {
  return item.type === 'plugin' && item.status === 'enabled' && Boolean(item.runtime)
}
```

- [ ] **Step 6: Update admin plugin UI**

In `plugins.vue`, show runtime badge and counts using `runtimeStatusLabelKey`, `runtimeCapabilitySummary`, and `canRestartPlugin`. Enable restart only when `canRestartPlugin(item)` returns true and wire it to `restartExtension(item)`.

- [ ] **Step 7: Update contracts and i18n**

Add `runtime`, `providers`, and plugin route namespace descriptions to OpenAPI. Add `admin.extensions.runtime.*` and `admin.extensions.capability.providers` labels in both locale files.

- [ ] **Step 8: Update knowledge**

Update `knowledge/modules/extensions.md` so it says plugin route proxying, lifecycle hooks, and provider registry are implemented for plugin runtime v1, while uploaded theme activation remains separate. Add session handoff:

```md
# 2026-07-06 Extension Plugin Runtime

## Changed

- Added plugin runtime planning and implementation for declared plugin routes, lifecycle hooks, and provider slot registry.

## Decisions

- Plugins cannot override arbitrary core API routes.
- Provider replacement is allowed only through named core-owned slots.

## Next

- Design uploaded theme activation worker separately.

## Open Questions

- None.
```

- [ ] **Step 9: Run verification**

```bash
ruby scripts/validate-openapi-refs.rb
cd apps/api
go test ./app/Models/Extensions ./app/Http/Controllers/Extensions ./app/Support/Extensions -count=1
cd ../web
bun test tests/adminExtensions.test.ts
```

Expected: all commands pass.

- [ ] **Step 10: Commit**

```bash
git add apps/api/app/Support/Extensions/providers.go apps/api/app/Support/Extensions/providers_test.go apps/web/app/utils/adminExtensions.ts apps/web/tests/adminExtensions.test.ts apps/web/app/pages/admin/extensions/plugins.vue apps/web/app/composables/useAdminExtensionsManager.ts apps/web/i18n/locales/zh-CN.json apps/web/i18n/locales/en-US.json contracts/openapi/paths/extensions.yaml contracts/openapi/schemas/extensions.yaml knowledge/modules/extensions.md knowledge/sessions/2026-07-06-extension-plugin-runtime.md
git commit -m "feat(extensions): surface plugin runtime status and providers"
```

## Final Verification

Run from repository root:

```bash
ruby scripts/validate-openapi-refs.rb
./scripts/test.sh
```

Expected: OpenAPI references validate, and the project test script exits 0.

## Self-Review

- Spec coverage: manifest runtime declarations are Task 1; lifecycle runtime is Task 2; route authorization is Task 3; route proxying is Tasks 4 and 6; subprocess protocol is Task 7; startup reconciliation is Task 8; lifecycle hooks are Task 9; provider slots/admin/docs are Task 10.
- Theme boundary: theme activation is not included, and knowledge updates keep it separate.
- Permission boundary: plugin routes are always under `/api/v1/extensions/{extensionId}/*`, and controller policy checks run before proxying.
- Beginner defaults: provider registry starts with built-in defaults and has restore behavior.
