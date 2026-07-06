# Extension Admin Manifest v2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Support the v2 extension admin manifest shape (`admin.entry`, `admin.pages[]`, explicit `menu: true`) while preserving legacy `adminPages` compatibility and keeping `Manage` inside the admin shell.

**Architecture:** The backend manifest package becomes the source of truth for normalizing v2 admin declarations, deriving effective pages, deriving menu pages, and resolving the default Manage path. The extension service uses only explicit menu pages for sidebar navigation, while the frontend uses the same rules for list-row Manage links and dynamic page lookup. The scaffold generator and OpenAPI contract are updated so new extension packages are created with the v2 shape.

**Tech Stack:** Go 1.25, Fiber service layer, SForum extension manifest package, Nuxt 4/Vue 3/TypeScript, Bun test, modular OpenAPI YAML.

---

## Scope

This plan implements the first Extension Platform v2 slice:

- Parse and validate `admin.entry`.
- Parse and validate `admin.pages[]`.
- Preserve legacy `adminPages` as a compatibility input.
- Add `menu` to admin page declarations with default `false`.
- Keep generated `/about` available as a management page.
- Resolve Manage route as `admin.entry`, then `/settings`, then first declared page, then `/about`.
- Expose sidebar navigation only for enabled plugins or active themes and only for pages with `menu: true`.
- Update CLI scaffold output to write the v2 `admin` object.
- Update OpenAPI and focused tests.

This plan does not add extension-owned frontend assets, Markdown rendering, iframes, remote components, plugin logs, failed-enable rollback, or `mail.provider`. Those are separate v2 slices.

## File Structure

- `apps/api/app/Support/ExtensionManifest/manifest.go`: add `ManifestAdmin`, `Manifest.Admin`, `ManifestAdminPage.Menu`, v2 normalization, validation, and exported helper functions.
- `apps/api/app/Support/ExtensionManifest/manifest_test.go`: new focused tests for v2 admin declarations and legacy compatibility.
- `apps/api/app/Models/Extensions/service.go`: use manifest helpers for sidebar navigation and management pages.
- `apps/api/app/Models/Extensions/service_test.go`: update navigation expectations so only `menu: true` pages appear.
- `apps/api/app/Http/Controllers/Extensions/controller_test.go`: update controller navigation test for explicit menu pages.
- `apps/api/cmd/sforum/generator.go`: scaffold v2 `admin` instead of legacy `adminPages`.
- `apps/api/cmd/sforum/generator_test.go`: assert scaffolded manifests use `admin.entry` and `admin.pages`.
- `apps/web/app/utils/adminExtensions.ts`: add v2 admin types, effective page helpers, and Manage-route resolution.
- `apps/web/tests/adminExtensions.test.ts`: add frontend helper tests for v2 admin pages and legacy fallback.
- `apps/web/app/pages/admin/extensions/index.vue`: use `extensionManageRoute(item)` for list-row Manage.
- `apps/web/app/pages/admin/extensions/plugins.vue`: use `extensionManageRoute(item)` for plugin Manage.
- `apps/web/app/pages/admin/extensions/themes.vue`: use `extensionManageRoute(item)` for theme Manage.
- `contracts/openapi/schemas/extensions.yaml`: document `admin`, `admin.pages[].menu`, and legacy `adminPages`.
- `contracts/openapi/paths/extensions.yaml`: update navigation description to explicit menu pages.
- `knowledge/modules/extensions.md`: mark the admin manifest v2 slice implemented after code lands.
- `knowledge/sessions/2026-07-07-extension-admin-manifest-v2.md`: session handoff for the shipped slice.

---

### Task 1: Backend Manifest v2 Types, Normalization, And Validation

**Files:**
- Create: `apps/api/app/Support/ExtensionManifest/manifest_test.go`
- Modify: `apps/api/app/Support/ExtensionManifest/manifest.go`

- [ ] **Step 1: Write failing manifest tests**

Create `apps/api/app/Support/ExtensionManifest/manifest_test.go` with:

```go
package extensionmanifest

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestAdminManifestV2NormalizeValidateAndResolveManagePath(t *testing.T) {
	body := []byte(`{
		"id":"demo.plugin",
		"name":"Demo Plugin",
		"description":"Demo plugin.",
		"url":"https://example.com/demo",
		"author":{"name":"Demo Studio"},
		"version":"1.0.0",
		"type":"plugin",
		"sforumVersion":"^1.0.0",
		"admin":{
			"entry":"settings",
			"pages":[
				{"path":"settings","label":"Settings","view":"settings"},
				{"path":"/dashboard","label":"Dashboard","view":"about","menu":true,"icon":"i-lucide-layout-dashboard","order":20}
			]
		}
	}`)

	var manifest Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	normalized := Normalize(manifest)
	if normalized.Admin.Entry != "/settings" {
		t.Fatalf("expected normalized admin entry, got %q", normalized.Admin.Entry)
	}
	pages := EffectiveAdminPages(normalized)
	if len(pages) != 2 {
		t.Fatalf("expected two effective pages, got %#v", pages)
	}
	if pages[0].Path != "/settings" || pages[0].Menu {
		t.Fatalf("settings page should normalize with menu false: %#v", pages[0])
	}
	if pages[1].Path != "/dashboard" || !pages[1].Menu {
		t.Fatalf("dashboard page should be an explicit menu page: %#v", pages[1])
	}
	if AdminManagePath(normalized) != "/settings" {
		t.Fatalf("expected entry to drive manage path, got %q", AdminManagePath(normalized))
	}
	menuPages := MenuAdminPages(normalized)
	if len(menuPages) != 1 || menuPages[0].Path != "/dashboard" {
		t.Fatalf("expected only explicit menu page, got %#v", menuPages)
	}
	if err := Validate(manifest); err != nil {
		t.Fatalf("v2 admin manifest should validate: %v", err)
	}
}

func TestAdminManifestV2RejectsBrokenEntryAndExternalPaths(t *testing.T) {
	base := Manifest{
		ID:            "demo.plugin",
		Name:          "Demo Plugin",
		Description:   "Demo plugin.",
		URL:           "https://example.com/demo",
		Author:        ManifestAuthor{Name: "Demo Studio"},
		Version:       "1.0.0",
		Type:          TypePlugin,
		SForumVersion: "^1.0.0",
	}

	cases := []struct {
		name     string
		manifest Manifest
	}{
		{
			name: "entry must target declared page or about",
			manifest: func() Manifest {
				next := base
				next.Admin.Entry = "/missing"
				next.Admin.Pages = []ManifestAdminPage{{Path: "/settings", Label: "Settings", View: "settings"}}
				return next
			}(),
		},
		{
			name: "entry cannot be external url",
			manifest: func() Manifest {
				next := base
				next.Admin.Entry = "https://example.com/settings"
				next.Admin.Pages = []ManifestAdminPage{{Path: "/settings", Label: "Settings", View: "settings"}}
				return next
			}(),
		},
		{
			name: "page cannot contain traversal",
			manifest: func() Manifest {
				next := base
				next.Admin.Pages = []ManifestAdminPage{{Path: "/../settings", Label: "Settings", View: "settings"}}
				return next
			}(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := Validate(tc.manifest); !errors.Is(err, ErrInvalidManifest) {
				t.Fatalf("expected invalid manifest, got %v", err)
			}
		})
	}
}

func TestLegacyAdminPagesRemainCompatible(t *testing.T) {
	manifest := Manifest{
		ID:            "legacy.plugin",
		Name:          "Legacy Plugin",
		Description:   "Legacy plugin.",
		URL:           "https://example.com/legacy",
		Author:        ManifestAuthor{Name: "Demo Studio"},
		Version:       "1.0.0",
		Type:          TypePlugin,
		SForumVersion: "^1.0.0",
		AdminPages: []ManifestAdminPage{
			{Path: "/settings", Label: "Settings", View: "settings", Menu: true},
		},
	}

	if err := Validate(manifest); err != nil {
		t.Fatalf("legacy adminPages should validate: %v", err)
	}
	normalized := Normalize(manifest)
	if AdminManagePath(normalized) != "/settings" {
		t.Fatalf("expected legacy settings page as manage path, got %q", AdminManagePath(normalized))
	}
	menuPages := MenuAdminPages(normalized)
	if len(menuPages) != 1 || menuPages[0].Path != "/settings" {
		t.Fatalf("expected legacy explicit menu page, got %#v", menuPages)
	}
}
```

- [ ] **Step 2: Run manifest tests to verify they fail**

Run:

```bash
cd apps/api && go test ./app/Support/ExtensionManifest -run 'TestAdminManifestV2|TestLegacyAdminPages' -count=1
```

Expected: FAIL with undefined names such as `ManifestAdmin`, `EffectiveAdminPages`, `AdminManagePath`, `MenuAdminPages`, or missing field `Menu`.

- [ ] **Step 3: Add backend manifest v2 implementation**

Modify `apps/api/app/Support/ExtensionManifest/manifest.go`.

Add `Admin ManifestAdmin` to `Manifest` after `Frontend`:

```go
	Admin        ManifestAdmin      `json:"admin"`
```

Add this type after `ManifestFrontend`:

```go
type ManifestAdmin struct {
	Entry string              `json:"entry,omitempty"`
	Pages []ManifestAdminPage `json:"pages,omitempty"`
}
```

Add `Menu` to `ManifestAdminPage`:

```go
	Menu        bool   `json:"menu,omitempty"`
```

Replace the admin-page validation loop inside `Validate` with:

```go
	if err := validateAdminDeclaration(manifest); err != nil {
		return err
	}
```

In `Normalize`, after `manifest.Frontend.Layer = strings.TrimSpace(manifest.Frontend.Layer)`, add:

```go
	manifest.Admin.Entry = NormalizeRoutePath(manifest.Admin.Entry)
	normalizeAdminPageSlice(manifest.Admin.Pages)
	normalizeAdminPageSlice(manifest.AdminPages)
```

Remove the existing inline loop that normalizes `manifest.AdminPages`.

Add these helper functions after `Normalize`:

```go
func normalizeAdminPageSlice(pages []ManifestAdminPage) {
	for index := range pages {
		pages[index].Path = NormalizeRoutePath(pages[index].Path)
		pages[index].Label = strings.TrimSpace(pages[index].Label)
		pages[index].Description = strings.TrimSpace(pages[index].Description)
		pages[index].Icon = strings.TrimSpace(pages[index].Icon)
		pages[index].View = strings.ToLower(strings.TrimSpace(pages[index].View))
		if pages[index].View == "" {
			pages[index].View = "about"
		}
		pages[index].Permission = strings.TrimSpace(pages[index].Permission)
	}
}

func validateAdminDeclaration(manifest Manifest) error {
	pages := EffectiveAdminPages(manifest)
	for _, page := range pages {
		if page.Path == "" || !strings.HasPrefix(page.Path, "/") || strings.Contains(page.Path, "..") || page.Label == "" {
			return ErrInvalidManifest
		}
		if page.View != "" && page.View != "about" && page.View != "settings" {
			return ErrInvalidManifest
		}
		if page.Order < 0 {
			return ErrInvalidManifest
		}
	}
	if manifest.Admin.Entry == "" {
		return nil
	}
	if strings.Contains(manifest.Admin.Entry, "://") || !strings.HasPrefix(manifest.Admin.Entry, "/") || strings.Contains(manifest.Admin.Entry, "..") {
		return ErrInvalidManifest
	}
	if manifest.Admin.Entry == "/about" {
		return nil
	}
	for _, page := range pages {
		if page.Path == manifest.Admin.Entry {
			return nil
		}
	}
	return ErrInvalidManifest
}

func EffectiveAdminPages(manifest Manifest) []ManifestAdminPage {
	manifest = Normalize(manifest)
	if len(manifest.Admin.Pages) > 0 {
		return manifest.Admin.Pages
	}
	return manifest.AdminPages
}

func MenuAdminPages(manifest Manifest) []ManifestAdminPage {
	pages := EffectiveAdminPages(manifest)
	menuPages := make([]ManifestAdminPage, 0, len(pages))
	for _, page := range pages {
		if page.Menu {
			menuPages = append(menuPages, page)
		}
	}
	return menuPages
}

func AdminManagePath(manifest Manifest) string {
	manifest = Normalize(manifest)
	pages := EffectiveAdminPages(manifest)
	if manifest.Admin.Entry != "" {
		return manifest.Admin.Entry
	}
	for _, page := range pages {
		if page.Path == "/settings" {
			return page.Path
		}
	}
	if len(pages) > 0 {
		return pages[0].Path
	}
	return "/about"
}
```

- [ ] **Step 4: Run manifest tests to verify they pass**

Run:

```bash
cd apps/api && go test ./app/Support/ExtensionManifest -run 'TestAdminManifestV2|TestLegacyAdminPages' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit backend manifest helpers**

Run:

```bash
git add apps/api/app/Support/ExtensionManifest/manifest.go apps/api/app/Support/ExtensionManifest/manifest_test.go
git commit -m "feat: support extension admin manifest v2"
```

---

### Task 2: Backend Navigation Uses Explicit Menu Pages

**Files:**
- Modify: `apps/api/app/Models/Extensions/service.go`
- Modify: `apps/api/app/Models/Extensions/service_test.go`
- Modify: `apps/api/app/Http/Controllers/Extensions/controller_test.go`

- [ ] **Step 1: Update service navigation tests first**

In `apps/api/app/Models/Extensions/service_test.go`, replace `TestServiceNavigationUsesEnabledPluginsAndActiveTheme` with:

```go
func TestServiceNavigationUsesOnlyExplicitMenuPagesFromEnabledPluginsAndActiveTheme(t *testing.T) {
	enabledPlugin := installedExtension("enabled.plugin", TypePlugin, ManifestBackend{})
	enabledPlugin.Status = StatusEnabled
	enabledPlugin.Manifest.Admin = ManifestAdmin{
		Entry: "/settings",
		Pages: []ManifestAdminPage{
			{Path: "/settings", Label: "Settings", View: "settings", Icon: "i-lucide-settings", Order: 20},
			{Path: "/dashboard", Label: "Dashboard", View: "about", Icon: "i-lucide-layout-dashboard", Order: 10, Menu: true},
		},
	}
	disabledPlugin := installedExtension("disabled.plugin", TypePlugin, ManifestBackend{})
	disabledPlugin.Status = StatusDisabled
	disabledPlugin.Manifest.Admin = ManifestAdmin{
		Pages: []ManifestAdminPage{{Path: "/hidden", Label: "Hidden", View: "about", Menu: true}},
	}
	activeTheme := installedExtension("active.theme", TypeTheme, ManifestBackend{})
	activeTheme.Status = StatusEnabled
	activeTheme.Manifest.Admin = ManifestAdmin{
		Pages: []ManifestAdminPage{{Path: "/theme", Label: "Theme", View: "about", Order: 30, Menu: true}},
	}
	store := &fakeExtensionStore{items: map[string]Extension{
		enabledPlugin.ID:  enabledPlugin,
		disabledPlugin.ID: disabledPlugin,
		activeTheme.ID:    activeTheme,
	}}
	service := NewService(store, t.TempDir())

	items, err := service.Navigation(context.Background(), extensionManager())
	if err != nil {
		t.Fatalf("Navigation returned error: %v", err)
	}
	if navigationContains(items, "enabled.plugin", "/about") || navigationContains(items, "enabled.plugin", "/settings") {
		t.Fatalf("generated and non-menu pages should not inject sidebar navigation: %#v", items)
	}
	if !navigationContains(items, "enabled.plugin", "/dashboard") {
		t.Fatalf("expected enabled plugin menu page, got %#v", items)
	}
	if !navigationContains(items, "active.theme", "/theme") {
		t.Fatalf("expected active theme explicit menu page, got %#v", items)
	}
	if navigationContains(items, "disabled.plugin", "/hidden") {
		t.Fatalf("disabled plugin should not inject sidebar navigation: %#v", items)
	}
}
```

In `apps/api/app/Http/Controllers/Extensions/controller_test.go`, update the setup in `TestControllerListsNavigationAndManagesExtensionSettings`:

```go
	plugin.Manifest.Admin = extensions.ManifestAdmin{
		Entry: "/settings",
		Pages: []extensions.ManifestAdminPage{
			{Path: "/settings", Label: "Settings", View: "settings", Icon: "i-lucide-settings", Order: 10},
			{Path: "/dashboard", Label: "Dashboard", View: "about", Icon: "i-lucide-layout-dashboard", Order: 5, Menu: true},
		},
	}
```

Update the navigation assertion in the same test:

```go
	if !controllerNavigationContains(navigation.Data, "demo.plugin", "/dashboard") || controllerNavigationContains(navigation.Data, "demo.plugin", "/settings") || controllerNavigationContains(navigation.Data, "demo.theme", "/about") {
		t.Fatalf("unexpected navigation items: %#v", navigation.Data)
	}
```

- [ ] **Step 2: Run backend navigation tests to verify they fail**

Run:

```bash
cd apps/api && go test ./app/Models/Extensions ./app/Http/Controllers/Extensions -run 'TestServiceNavigationUsesOnlyExplicitMenuPagesFromEnabledPluginsAndActiveTheme|TestControllerListsNavigationAndManagesExtensionSettings' -count=1
```

Expected: FAIL because `ManifestAdmin` is not aliased in the extensions model or because navigation still includes generated and non-menu pages.

- [ ] **Step 3: Alias v2 manifest admin type**

In `apps/api/app/Models/Extensions/types.go`, add this alias near the other manifest aliases:

```go
type ManifestAdmin = extensionmanifest.ManifestAdmin
```

- [ ] **Step 4: Update service navigation implementation**

In `apps/api/app/Models/Extensions/service.go`, replace the page loop in `Navigation`:

```go
		for _, page := range normalizedAdminPages(item.Manifest) {
```

with:

```go
		for _, page := range normalizedMenuAdminPages(item.Manifest) {
```

Replace `normalizedAdminPages` with:

```go
func normalizedAdminPages(manifest Manifest) []ManifestAdminPage {
	pages := make([]ManifestAdminPage, 0, len(extensionmanifest.EffectiveAdminPages(manifest))+1)
	pages = append(pages, ManifestAdminPage{
		Path:        "/about",
		Label:       manifest.Name,
		Description: manifest.Description,
		Icon:        defaultExtensionIcon(manifest.Type),
		View:        "about",
		Order:       0,
	})
	for _, page := range extensionmanifest.EffectiveAdminPages(manifest) {
		if strings.TrimSpace(page.Path) == "" {
			continue
		}
		pages = append(pages, normalizeAdminPageForDisplay(manifest.Type, page))
	}
	sort.SliceStable(pages, func(left, right int) bool {
		if pages[left].Order == pages[right].Order {
			return pages[left].Path < pages[right].Path
		}
		return pages[left].Order < pages[right].Order
	})
	return pages
}

func normalizedMenuAdminPages(manifest Manifest) []ManifestAdminPage {
	pages := make([]ManifestAdminPage, 0, len(extensionmanifest.MenuAdminPages(manifest)))
	for _, page := range extensionmanifest.MenuAdminPages(manifest) {
		if strings.TrimSpace(page.Path) == "" {
			continue
		}
		pages = append(pages, normalizeAdminPageForDisplay(manifest.Type, page))
	}
	sort.SliceStable(pages, func(left, right int) bool {
		if pages[left].Order == pages[right].Order {
			return pages[left].Path < pages[right].Path
		}
		return pages[left].Order < pages[right].Order
	})
	return pages
}

func normalizeAdminPageForDisplay(extensionType string, page ManifestAdminPage) ManifestAdminPage {
	page.Path = extensionmanifest.NormalizeRoutePath(page.Path)
	page.Label = strings.TrimSpace(page.Label)
	page.Description = strings.TrimSpace(page.Description)
	page.Icon = strings.TrimSpace(page.Icon)
	page.View = strings.TrimSpace(page.View)
	page.Permission = strings.TrimSpace(page.Permission)
	if page.Icon == "" {
		page.Icon = defaultExtensionIcon(extensionType)
	}
	if page.View == "" {
		page.View = "about"
	}
	return page
}
```

Keep `normalizedAdminPages` because later service/controller code and future backend Manage helpers can still use the complete management-page list.

- [ ] **Step 5: Run backend navigation tests to verify they pass**

Run:

```bash
cd apps/api && go test ./app/Models/Extensions ./app/Http/Controllers/Extensions -run 'TestServiceNavigationUsesOnlyExplicitMenuPagesFromEnabledPluginsAndActiveTheme|TestControllerListsNavigationAndManagesExtensionSettings' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit backend navigation behavior**

Run:

```bash
git add apps/api/app/Models/Extensions/types.go apps/api/app/Models/Extensions/service.go apps/api/app/Models/Extensions/service_test.go apps/api/app/Http/Controllers/Extensions/controller_test.go
git commit -m "feat: require explicit extension admin menu entries"
```

---

### Task 3: Frontend Admin Helpers Resolve v2 Pages And Manage Routes

**Files:**
- Modify: `apps/web/app/utils/adminExtensions.ts`
- Modify: `apps/web/tests/adminExtensions.test.ts`

- [ ] **Step 1: Write frontend helper tests first**

Update the import list in `apps/web/tests/adminExtensions.test.ts` to include:

```ts
  extensionAdminPages,
  extensionManageRoute,
  findExtensionAdminPage,
```

Add this test inside the existing `describe('admin extension helpers', () => { })` block:

```ts
  test('resolves v2 admin pages, manage route, and legacy adminPages fallback', () => {
    const item = extension({
      id: 'admin.plugin',
      name: 'Admin Plugin',
      type: 'plugin',
      manifest: {
        admin: {
          entry: '/settings',
          pages: [
            { path: '/settings', label: 'Settings', view: 'settings' },
            { path: '/dashboard', label: 'Dashboard', view: 'about', menu: true, icon: 'i-lucide-layout-dashboard', order: 20 }
          ]
        },
        adminPages: [{ path: '/legacy', label: 'Legacy' }]
      }
    })
    const legacy = extension({
      id: 'legacy.plugin',
      name: 'Legacy Plugin',
      type: 'plugin',
      manifest: {
        adminPages: [{ path: '/settings', label: 'Legacy Settings', view: 'settings', menu: true }]
      }
    })
    const generatedOnly = extension({
      id: 'plain.plugin',
      name: 'Plain Plugin',
      type: 'plugin'
    })

    expect(extensionAdminPages(item).map(page => page.path)).toEqual(['/about', '/settings', '/dashboard'])
    expect(findExtensionAdminPage(item, 'dashboard')?.label).toBe('Dashboard')
    expect(extensionManageRoute(item)).toBe('/extensions/admin.plugin/pages/settings')
    expect(extensionManageRoute(legacy)).toBe('/extensions/legacy.plugin/pages/settings')
    expect(extensionManageRoute(generatedOnly)).toBe('/extensions/plain.plugin/pages/about')
  })
```

- [ ] **Step 2: Run frontend helper test to verify it fails**

Run:

```bash
cd apps/web && bun test tests/adminExtensions.test.ts --filter 'resolves v2 admin pages'
```

Expected: FAIL because `admin` and `menu` types/functions are missing or `extensionManageRoute` is undefined.

- [ ] **Step 3: Add v2 frontend types and helpers**

In `apps/web/app/utils/adminExtensions.ts`, add `menu` to `AdminExtensionAdminPage`:

```ts
  menu?: boolean
```

Add this type after `AdminExtensionAdminPage`:

```ts
export type AdminExtensionAdmin = {
  entry?: string
  pages?: AdminExtensionAdminPage[]
}
```

Add `admin?: AdminExtensionAdmin` to `AdminExtensionManifest`.

Change `capabilityCount` so the admin-page count uses v2 pages when present:

```ts
    effectiveManifestAdminPages(manifest).length,
```

Add this helper near `capabilityCount`:

```ts
function effectiveManifestAdminPages(manifest: AdminExtensionManifest) {
  return manifest.admin?.pages?.length ? manifest.admin.pages : manifest.adminPages || []
}
```

Replace the `for` loop in `extensionAdminPages`:

```ts
  for (const page of item.manifest.adminPages || []) {
```

with:

```ts
  for (const page of effectiveManifestAdminPages(item.manifest)) {
```

Inside that `pages.push` object, add:

```ts
      menu: page.menu === true,
```

Add these exported helpers after `findExtensionAdminPage`:

```ts
export function extensionManagePagePath(item: AdminExtension) {
  const pages = extensionAdminPages(item)
  const entry = normalizeExtensionPagePath(item.manifest.admin?.entry)
  if (item.manifest.admin?.entry && pages.some(page => normalizeExtensionPagePath(page.path) === entry)) {
    return entry
  }

  const settings = pages.find(page => normalizeExtensionPagePath(page.path) === '/settings')
  if (settings) {
    return '/settings'
  }

  const declared = pages.find(page => normalizeExtensionPagePath(page.path) !== '/about')
  return declared?.path || '/about'
}

export function extensionManageRoute(item: AdminExtension) {
  return extensionAdminPageRoute(item.id, extensionManagePagePath(item))
}
```

- [ ] **Step 4: Run frontend helper tests to verify they pass**

Run:

```bash
cd apps/web && bun test tests/adminExtensions.test.ts --filter 'resolves v2 admin pages'
```

Expected: PASS.

- [ ] **Step 5: Commit frontend helper behavior**

Run:

```bash
git add apps/web/app/utils/adminExtensions.ts apps/web/tests/adminExtensions.test.ts
git commit -m "feat: resolve extension admin manage routes"
```

---

### Task 4: Frontend Manage Buttons Use Manifest Entry Resolution

**Files:**
- Modify: `apps/web/app/pages/admin/extensions/index.vue`
- Modify: `apps/web/app/pages/admin/extensions/plugins.vue`
- Modify: `apps/web/app/pages/admin/extensions/themes.vue`

- [ ] **Step 1: Update imports and button targets**

In `apps/web/app/pages/admin/extensions/index.vue`, change the import from:

```ts
import { capabilityCount, extensionAdminPageRoute, extensionAuthorName, extensionAuthorWebsite, extensionEventPage, themeActionState, themeStatusLabelKey } from '~/utils/adminExtensions'
```

to:

```ts
import { capabilityCount, extensionAuthorName, extensionAuthorWebsite, extensionEventPage, extensionManageRoute, themeActionState, themeStatusLabelKey } from '~/utils/adminExtensions'
```

Replace:

```vue
:to="adminRoutes.path(extensionAdminPageRoute(item.id))"
```

with:

```vue
:to="adminRoutes.path(extensionManageRoute(item))"
```

In `apps/web/app/pages/admin/extensions/plugins.vue`, change the import from:

```ts
import { canRestartPlugin, capabilityCount, extensionAdminPageRoute, extensionAuthorName, extensionAuthorWebsite, filterExtensionsByType, runtimeCapabilitySummary, runtimeStatusLabelKey, type AdminRuntimeState } from '~/utils/adminExtensions'
```

to:

```ts
import { canRestartPlugin, capabilityCount, extensionAuthorName, extensionAuthorWebsite, extensionManageRoute, filterExtensionsByType, runtimeCapabilitySummary, runtimeStatusLabelKey, type AdminRuntimeState } from '~/utils/adminExtensions'
```

Replace:

```vue
:to="adminRoutes.path(extensionAdminPageRoute(item.id))"
```

with:

```vue
:to="adminRoutes.path(extensionManageRoute(item))"
```

In `apps/web/app/pages/admin/extensions/themes.vue`, change the import from:

```ts
import { capabilityCount, extensionAdminPageRoute, extensionAuthorName, extensionAuthorWebsite, filterExtensionsByType, themeActionState, themeStatusLabelKey } from '~/utils/adminExtensions'
```

to:

```ts
import { capabilityCount, extensionAuthorName, extensionAuthorWebsite, extensionManageRoute, filterExtensionsByType, themeActionState, themeStatusLabelKey } from '~/utils/adminExtensions'
```

Replace:

```vue
:to="adminRoutes.path(extensionAdminPageRoute(item.id))"
```

with:

```vue
:to="adminRoutes.path(extensionManageRoute(item))"
```

- [ ] **Step 2: Run frontend helper and type checks**

Run:

```bash
cd apps/web && bun test tests/adminExtensions.test.ts
```

Expected: PASS.

Run:

```bash
cd apps/web && bun run typecheck
```

Expected: PASS.

- [ ] **Step 3: Commit Manage button routing**

Run:

```bash
git add apps/web/app/pages/admin/extensions/index.vue apps/web/app/pages/admin/extensions/plugins.vue apps/web/app/pages/admin/extensions/themes.vue
git commit -m "feat: route extension manage actions through manifest entry"
```

---

### Task 5: OpenAPI Contract Documents v2 Admin Shape

**Files:**
- Modify: `contracts/openapi/schemas/extensions.yaml`
- Modify: `contracts/openapi/paths/extensions.yaml`

- [ ] **Step 1: Update OpenAPI schemas**

In `contracts/openapi/schemas/extensions.yaml`, update the `ExtensionManifest` description to mention `admin` and legacy `adminPages`.

Add an `admin` property next to `adminPages`:

```yaml
    admin:
      "$ref": "#/ExtensionManifestAdmin"
```

Update `adminPages` description to:

```yaml
      description: Legacy core-container admin page declarations. Prefer admin.pages for new manifests; legacy pages remain compatibility-mapped.
```

Add this schema before `ExtensionManifestAdminPage`:

```yaml
ExtensionManifestAdmin:
  type: object
  description: Extension admin entry and page declarations. Manage resolves to entry, then settings, then first declared page, then the generated about page.
  properties:
    entry:
      type: string
      description: In-admin extension page path such as /settings. Must point to /about or a declared page.
    pages:
      type: array
      items:
        "$ref": "#/ExtensionManifestAdminPage"
```

Add `menu` to `ExtensionManifestAdminPage.properties`:

```yaml
    menu:
      type: boolean
      default: false
      description: Whether this page appears in admin sidebar navigation. Defaults to false.
```

- [ ] **Step 2: Update navigation path description**

In `contracts/openapi/paths/extensions.yaml`, change the `adminExtensionNavigation` description to:

```yaml
    description: Returns explicit extension admin menu pages declared with menu true by enabled plugins and the active theme. All pages stay under the fixed Extensions namespace.
```

- [ ] **Step 3: Validate OpenAPI refs**

Run:

```bash
ruby scripts/validate-openapi-refs.rb
```

Expected: PASS.

- [ ] **Step 4: Commit OpenAPI updates**

Run:

```bash
git add contracts/openapi/schemas/extensions.yaml contracts/openapi/paths/extensions.yaml
git commit -m "docs: document extension admin manifest v2 contract"
```

---

### Task 6: Developer Console Scaffolds v2 Admin Manifests

**Files:**
- Modify: `apps/api/cmd/sforum/generator.go`
- Modify: `apps/api/cmd/sforum/generator_test.go`

- [ ] **Step 1: Update scaffold tests first**

In `apps/api/cmd/sforum/generator_test.go`, add this assertion to `TestGeneratePluginScaffoldNonInteractive` after reading the manifest:

```go
	if manifest.Admin.Entry != "/settings" || len(manifest.Admin.Pages) != 1 || manifest.Admin.Pages[0].Menu {
		t.Fatalf("expected v2 plugin admin settings page without sidebar menu: %#v", manifest.Admin)
	}
	if len(manifest.AdminPages) != 0 {
		t.Fatalf("new scaffolds should not use legacy adminPages: %#v", manifest.AdminPages)
	}
```

Replace the theme admin assertion:

```go
	if len(manifest.AdminPages) == 0 || len(manifest.Settings) == 0 {
		t.Fatalf("expected theme admin page and settings declarations: %#v", manifest)
	}
```

with:

```go
	if manifest.Admin.Entry != "/settings" || len(manifest.Admin.Pages) != 1 || len(manifest.Settings) == 0 {
		t.Fatalf("expected theme v2 admin page and settings declarations: %#v", manifest)
	}
	if len(manifest.AdminPages) != 0 {
		t.Fatalf("new theme scaffolds should not use legacy adminPages: %#v", manifest.AdminPages)
	}
```

- [ ] **Step 2: Run generator tests to verify they fail**

Run:

```bash
cd apps/api && go test ./cmd/sforum -run 'TestGeneratePluginScaffoldNonInteractive|TestGenerateThemeScaffoldNonInteractive' -count=1
```

Expected: FAIL because the scaffold still writes legacy `adminPages`.

- [ ] **Step 3: Update scaffold manifest struct and builder**

In `apps/api/cmd/sforum/generator.go`, replace the `AdminPages` field in `scaffoldManifest`:

```go
	AdminPages    []extensionmanifest.ManifestAdminPage `json:"adminPages,omitempty"`
```

with:

```go
	Admin         extensionmanifest.ManifestAdmin        `json:"admin,omitempty"`
```

In `buildManifest`, replace the `AdminPages` assignment with:

```go
		Admin: extensionmanifest.ManifestAdmin{
			Entry: "/settings",
			Pages: []extensionmanifest.ManifestAdminPage{{
				Path:        "/settings",
				Label:       "Settings",
				Description: "Configure this extension.",
				Icon:        "i-lucide-settings",
				View:        "settings",
				Order:       100,
			}},
		},
```

- [ ] **Step 4: Run generator tests to verify they pass**

Run:

```bash
cd apps/api && go test ./cmd/sforum -run 'TestGeneratePluginScaffoldNonInteractive|TestGenerateThemeScaffoldNonInteractive' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit scaffold updates**

Run:

```bash
git add apps/api/cmd/sforum/generator.go apps/api/cmd/sforum/generator_test.go
git commit -m "feat: scaffold extension admin manifest v2"
```

---

### Task 7: Knowledge Base Handoff For The Implemented Slice

**Files:**
- Modify: `knowledge/modules/extensions.md`
- Create: `knowledge/sessions/2026-07-07-extension-admin-manifest-v2.md`

- [ ] **Step 1: Update extension module status**

In `knowledge/modules/extensions.md`, add this bullet under Current Status:

```md
- Extension admin manifest v2 is implemented for management entry and sidebar
  behavior: new manifests may declare `admin.entry` and `admin.pages[]`,
  legacy `adminPages` remains compatible, `Manage` resolves inside the admin
  shell, and sidebar injection requires explicit `menu: true`.
```

In the Manifest section, change:

```md
The v2 target admin declaration is an `admin` object.
```

to:

```md
The v2 admin declaration is an `admin` object.
```

- [ ] **Step 2: Add session handoff**

Create `knowledge/sessions/2026-07-07-extension-admin-manifest-v2.md`:

```md
# 2026-07-07 Extension Admin Manifest v2

## Changed

- Added `admin.entry` and `admin.pages[]` support to extension manifests.
- Kept legacy `adminPages` compatible for existing packages.
- Added `menu` to admin page declarations, defaulting to false.
- Updated extension admin navigation to include only explicit `menu: true`
  pages from enabled plugins and the active theme.
- Updated list-row Manage actions to resolve through manifest entry,
  `/settings`, first declared page, or generated `/about`.
- Updated OpenAPI and `sforum make:*` scaffolds for the v2 admin shape.

## Decisions

- New scaffolds should emit `admin.entry` and `admin.pages[]`, not legacy
  `adminPages`.
- Generated `/about` remains a management fallback but is not a sidebar item.
- `admin.entry` must stay inside the host admin shell and target `/about` or a
  declared admin page.

## Next

- Add safe manifest content pages when the host-rendered content model is
  designed.
- Start the `mail.provider` vertical slice after plugin observability and
  failed-enable rollback are planned.

## Open Questions

- Whether legacy `adminPages` should emit a deprecation warning during upload
  once extension package diagnostics exist.
```

- [ ] **Step 3: Run docs checks**

Run:

```bash
git diff --check
```

Expected: PASS with no output.

- [ ] **Step 4: Commit knowledge base update**

Run:

```bash
git add knowledge/modules/extensions.md knowledge/sessions/2026-07-07-extension-admin-manifest-v2.md
git commit -m "docs: record extension admin manifest v2 rollout"
```

---

### Task 8: Full Verification

**Files:**
- No source edits.

- [ ] **Step 1: Run focused Go tests**

Run:

```bash
cd apps/api && go test ./app/Support/ExtensionManifest ./app/Models/Extensions ./app/Http/Controllers/Extensions ./cmd/sforum
```

Expected: PASS.

- [ ] **Step 2: Run focused frontend tests**

Run:

```bash
cd apps/web && bun test tests/adminExtensions.test.ts
```

Expected: PASS.

- [ ] **Step 3: Run OpenAPI reference validation**

Run:

```bash
ruby scripts/validate-openapi-refs.rb
```

Expected: PASS.

- [ ] **Step 4: Run full project test script**

Run:

```bash
./scripts/test.sh
```

Expected: PASS. If this fails because dependency services are not running, start dependencies with:

```bash
./scripts/dev.sh
```

Then rerun:

```bash
./scripts/test.sh
```

- [ ] **Step 5: Inspect final status**

Run:

```bash
git status --short
```

Expected: no unstaged or staged source changes.
