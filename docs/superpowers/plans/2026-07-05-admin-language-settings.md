# Admin Language Settings Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an admin language settings page that uploads runtime ZIP language packs, stores package files outside Git, and enables frontend-only runtime locale messages.

**Architecture:** Add a backend `Localization` domain module with ZIP validation, package metadata tables, public locale/message APIs, and `locale.manage` admin APIs. Keep source-controlled `zh-CN`/`en-US` catalogs as the baseline; uploaded packages live under `LOCALE_PACK_ROOT` and are served through the API for Vue I18n runtime merging. Move default/supported locale management from the generic settings page into a dedicated admin `/locales` page.

**Tech Stack:** Go/Fiber v3, PostgreSQL migrations, pgx, Nuxt 4/Vue 3, @nuxtjs/i18n, vue-i18n runtime APIs, Nuxt UI, OpenAPI 3.1 modular contract.

---

## File Structure

Backend domain:

- Create `apps/api/app/Models/Localization/types.go` for package DTOs, manifest structs, errors, constants, and service inputs.
- Create `apps/api/app/Models/Localization/store.go` for the storage interface.
- Create `apps/api/app/Models/Localization/service.go` for permission checks, ZIP parsing, validation, install/enable/disable, locale settings, and message reads.
- Create `apps/api/app/Models/Localization/postgres_store.go` for PostgreSQL persistence.
- Create `apps/api/app/Models/Localization/service_test.go` for ZIP, manifest, override, conflict, and state tests.

Backend HTTP:

- Create `apps/api/app/Http/Controllers/Localization/routes.go`.
- Create `apps/api/app/Http/Controllers/Localization/controller.go`.
- Create `apps/api/app/Http/Controllers/Localization/controller_test.go`.
- Create `apps/api/app/Providers/localization.go`.
- Modify `apps/api/bootstrap/app.go` to wire the store/provider.
- Modify `apps/api/config/config.go` and `apps/api/config/config_test.go` for `LOCALE_PACK_ROOT`.

Database and permissions:

- Create `apps/api/database/migrations/202607050006_locale_packs.sql`.
- Modify `apps/api/app/Models/Identity/seeds.go`.
- Modify `apps/api/app/Models/Identity/seeds_test.go` only if the current assertions require exact permission coverage.
- Modify `apps/api/app/Models/Options/service.go` and `apps/api/app/Models/Options/service_test.go` so locale settings can validate against enabled runtime locales.

OpenAPI:

- Create `contracts/openapi/paths/localization.yaml`.
- Create `contracts/openapi/schemas/localization.yaml`.
- Modify `contracts/openapi.yaml`.

Frontend runtime:

- Create `apps/web/app/composables/useRuntimeLocales.ts`.
- Create `apps/web/app/plugins/runtime-locales.client.ts`.
- Modify `apps/web/app/components/SFNavbar.vue`.
- Modify `apps/web/app/composables/useWebOptions.ts` only to stop owning locale-setting admin behavior if needed.

Frontend admin:

- Create `apps/web/app/pages/admin/locales.vue`.
- Modify `apps/web/app/config/adminModules.ts`.
- Modify `apps/web/app/pages/admin/settings/index.vue` to remove locale controls from the basic tab.
- Modify `apps/web/i18n/locales/zh-CN.json` and `apps/web/i18n/locales/en-US.json` for labels and messages.
- Create `apps/web/tests/runtimeLocales.test.ts`.
- Update or create admin validation tests under `tests/validate-admin-framework.ts`.

Operations and docs:

- Modify `.gitignore`.
- Modify `compose.yaml`, `compose.dev.yaml`, and `compose.prod.yaml`.
- Modify `.env.example` and `.env.production.example`.
- Modify `knowledge/modules/localization.md`, `knowledge/modules/options.md`, and `knowledge/index.md`.
- Add a session handoff after implementation.

---

## Task 1: Add Permission, Config, Storage Ignore, And Migration

**Files:**
- Modify: `apps/api/app/Models/Identity/seeds.go`
- Modify: `apps/api/config/config.go`
- Modify: `apps/api/config/config_test.go`
- Create: `apps/api/database/migrations/202607050006_locale_packs.sql`
- Modify: `.gitignore`
- Modify: `compose.yaml`
- Modify: `compose.dev.yaml`
- Modify: `compose.prod.yaml`
- Modify: `.env.example`
- Modify: `.env.production.example`

- [ ] **Step 1: Write the config test for `LOCALE_PACK_ROOT`**

Add this assertion to the default config test in `apps/api/config/config_test.go`:

```go
if cfg.LocalePackRoot != "/var/lib/sforum/locale-packs" {
	t.Fatalf("expected default LocalePackRoot, got %q", cfg.LocalePackRoot)
}
```

Add this assertion to the environment override test:

```go
t.Setenv("LOCALE_PACK_ROOT", "/srv/sforum/locale-packs")
cfg := Load()
if cfg.LocalePackRoot != "/srv/sforum/locale-packs" {
	t.Fatalf("expected overridden LocalePackRoot, got %q", cfg.LocalePackRoot)
}
```

- [ ] **Step 2: Run the config test and verify it fails**

Run:

```bash
go test ./config -run TestLoad -count=1
```

Expected: FAIL because `Config.LocalePackRoot` does not exist.

- [ ] **Step 3: Add the config field**

In `apps/api/config/config.go`, add the field:

```go
LocalePackRoot string
```

In `Load()`, set:

```go
LocalePackRoot: env("LOCALE_PACK_ROOT", "/var/lib/sforum/locale-packs"),
```

- [ ] **Step 4: Add the `locale.manage` seed permission**

In `apps/api/app/Models/Identity/seeds.go`, add:

```go
PermissionLocaleManage = "locale.manage"
```

Add this seed permission:

```go
{Key: PermissionLocaleManage, Module: "localization", Description: "Upload and manage runtime language packs."},
```

- [ ] **Step 5: Add the locale pack migration**

Create `apps/api/database/migrations/202607050006_locale_packs.sql`:

```sql
-- +goose Up
CREATE TABLE locale_packs (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'installed' CHECK (status IN ('installed', 'enabled', 'disabled')),
  active_version_id BIGINT,
  allow_builtin_override BOOLEAN NOT NULL DEFAULT false,
  installed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE locale_pack_versions (
  id BIGSERIAL PRIMARY KEY,
  pack_id TEXT NOT NULL REFERENCES locale_packs(id) ON DELETE CASCADE,
  version TEXT NOT NULL,
  manifest JSONB NOT NULL,
  package_path TEXT NOT NULL,
  messages_path TEXT NOT NULL,
  installed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (pack_id, version)
);

ALTER TABLE locale_packs
  ADD CONSTRAINT locale_packs_active_version_fk
  FOREIGN KEY (active_version_id) REFERENCES locale_pack_versions(id) ON DELETE SET NULL;

CREATE TABLE locale_pack_locales (
  id BIGSERIAL PRIMARY KEY,
  version_id BIGINT NOT NULL REFERENCES locale_pack_versions(id) ON DELETE CASCADE,
  locale_code TEXT NOT NULL,
  display_name TEXT NOT NULL,
  frontend_path TEXT NOT NULL,
  backend_path TEXT NOT NULL DEFAULT '',
  message_key_count INTEGER NOT NULL DEFAULT 0,
  missing_key_count INTEGER NOT NULL DEFAULT 0,
  overrides_builtin BOOLEAN NOT NULL DEFAULT false,
  UNIQUE (version_id, locale_code)
);

CREATE TABLE locale_pack_events (
  id BIGSERIAL PRIMARY KEY,
  pack_id TEXT NOT NULL REFERENCES locale_packs(id) ON DELETE CASCADE,
  actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  action TEXT NOT NULL,
  message TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX locale_packs_status_idx ON locale_packs (status);
CREATE INDEX locale_pack_locales_locale_idx ON locale_pack_locales (locale_code);
CREATE INDEX locale_pack_events_pack_created_idx ON locale_pack_events (pack_id, created_at DESC, id DESC);

INSERT INTO permissions (key, module, description)
VALUES ('locale.manage', 'localization', 'Upload and manage runtime language packs.')
ON CONFLICT (key) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key)
SELECT roles.id, 'locale.manage'
FROM roles
WHERE roles.key = 'super_admin'
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_permissions WHERE permission_key = 'locale.manage';
DELETE FROM permissions WHERE key = 'locale.manage';

DROP TABLE IF EXISTS locale_pack_events;
DROP TABLE IF EXISTS locale_pack_locales;
ALTER TABLE locale_packs DROP CONSTRAINT IF EXISTS locale_packs_active_version_fk;
DROP TABLE IF EXISTS locale_pack_versions;
DROP INDEX IF EXISTS locale_packs_status_idx;
DROP TABLE IF EXISTS locale_packs;
```

- [ ] **Step 6: Add runtime storage ignores and env defaults**

In `.gitignore`, add under persisted runtime data:

```gitignore
storage/
```

In `.env.example` and `.env.production.example`, add:

```dotenv
LOCALE_PACK_ROOT=../../storage/locale-packs
```

For production examples, use:

```dotenv
LOCALE_PACK_ROOT=/var/lib/sforum/locale-packs
```

- [ ] **Step 7: Add Compose volume wiring**

In `compose.yaml`, add `LOCALE_PACK_ROOT` to `api` and `worker` environment and mount the volume:

```yaml
      LOCALE_PACK_ROOT: ${LOCALE_PACK_ROOT:-/var/lib/sforum/locale-packs}
    volumes:
      - attachment_uploads:/app/storage/app/attachments
      - locale_packs:/var/lib/sforum/locale-packs
```

Add to top-level volumes:

```yaml
  locale_packs:
```

In `compose.dev.yaml`, add the same volume to `api` and `worker` services:

```yaml
      - locale_packs:/var/lib/sforum/locale-packs
```

Add to its volumes section:

```yaml
  locale_packs:
```

- [ ] **Step 8: Run focused verification**

Run:

```bash
go test ./config -count=1
go test ./database/migrations -count=1
```

Expected: both commands PASS.

- [ ] **Step 9: Commit**

```bash
git add .gitignore .env.example .env.production.example compose.yaml compose.dev.yaml compose.prod.yaml apps/api/config/config.go apps/api/config/config_test.go apps/api/app/Models/Identity/seeds.go apps/api/database/migrations/202607050006_locale_packs.sql
git commit -m "feat: add locale pack storage foundation"
```

---

## Task 2: Implement Backend Localization Domain Service

**Files:**
- Create: `apps/api/app/Models/Localization/types.go`
- Create: `apps/api/app/Models/Localization/store.go`
- Create: `apps/api/app/Models/Localization/service.go`
- Create: `apps/api/app/Models/Localization/postgres_store.go`
- Create: `apps/api/app/Models/Localization/service_test.go`

- [ ] **Step 1: Write service tests first**

Create `apps/api/app/Models/Localization/service_test.go` with tests covering permission, unsafe ZIP paths, manifest validation, install event, built-in override denial, locale conflict, and disable cleanup:

```go
package localizationmodel

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

func TestServiceInstallArchiveRequiresLocaleManagePermission(t *testing.T) {
	service := NewService(&fakeStore{}, t.TempDir())
	actor := identity.Actor{ID: 7, Status: identity.UserStatusActive, Permissions: map[string]bool{}}

	_, err := service.InstallArchive(context.Background(), actor, ArchiveInput{
		FileName: "sample.zip",
		Data:     localeArchive(t, validManifest("ja.pack", "ja-JP", false), zipFile{name: "messages/ja-JP.json", body: `{"nav":{"home":"ホーム"}}`}),
	})
	if !errors.Is(err, identity.ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestServiceInstallArchiveValidatesManifestMessagesAndSafeZipPaths(t *testing.T) {
	service := NewService(&fakeStore{}, t.TempDir())
	actor := localeManager()

	_, err := service.InstallArchive(context.Background(), actor, ArchiveInput{FileName: "missing.zip", Data: localeArchive(t, "")})
	if !errors.Is(err, ErrInvalidArchive) {
		t.Fatalf("expected invalid archive for missing manifest, got %v", err)
	}

	_, err = service.InstallArchive(context.Background(), actor, ArchiveInput{
		FileName: "unsafe.zip",
		Data: localeArchive(t, validManifest("ja.pack", "ja-JP", false),
			zipFile{name: "../escape.json", body: `{}`},
			zipFile{name: "messages/ja-JP.json", body: `{}`},
		),
	})
	if !errors.Is(err, ErrInvalidArchive) {
		t.Fatalf("expected invalid archive for unsafe path, got %v", err)
	}

	_, err = service.InstallArchive(context.Background(), actor, ArchiveInput{
		FileName: "bad-json.zip",
		Data: localeArchive(t, validManifest("ja.pack", "ja-JP", false),
			zipFile{name: "messages/ja-JP.json", body: `[]`},
		),
	})
	if !errors.Is(err, ErrInvalidMessages) {
		t.Fatalf("expected invalid messages, got %v", err)
	}
}

func TestServiceInstallArchiveStoresPackVersionLocalesAndEvent(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, t.TempDir())

	installed, err := service.InstallArchive(context.Background(), localeManager(), ArchiveInput{
		FileName: "ja.zip",
		Data: localeArchive(t, validManifest("ja.pack", "ja-JP", false),
			zipFile{name: "messages/ja-JP.json", body: `{"nav":{"home":"ホーム"},"auth":{"login":"ログイン"}}`},
		),
	})
	if err != nil {
		t.Fatalf("InstallArchive returned error: %v", err)
	}
	if installed.ID != "ja.pack" || installed.Version != "1.0.0" || installed.Status != StatusInstalled {
		t.Fatalf("unexpected installed pack: %#v", installed)
	}
	if len(installed.Locales) != 1 || installed.Locales[0].MessageKeyCount != 2 {
		t.Fatalf("expected locale key count, got %#v", installed.Locales)
	}
	if len(store.events) != 1 || store.events[0].Action != EventInstalled {
		t.Fatalf("expected install event, got %#v", store.events)
	}
}

func TestServiceEnableRequiresDoubleOptInForBuiltinOverride(t *testing.T) {
	store := &fakeStore{items: map[string]LocalePack{
		"zh.override": installedPack("zh.override", "zh-CN", true),
	}}
	service := NewService(store, t.TempDir())

	_, err := service.Enable(context.Background(), localeManager(), EnableInput{
		PackID:               "zh.override",
		Version:              "1.0.0",
		AllowBuiltinOverride: false,
	})
	if !errors.Is(err, ErrOverrideDenied) {
		t.Fatalf("expected override denied, got %v", err)
	}

	enabled, err := service.Enable(context.Background(), localeManager(), EnableInput{
		PackID:               "zh.override",
		Version:              "1.0.0",
		AllowBuiltinOverride: true,
	})
	if err != nil {
		t.Fatalf("Enable returned error: %v", err)
	}
	if enabled.Status != StatusEnabled || !enabled.AllowBuiltinOverride {
		t.Fatalf("expected enabled override pack, got %#v", enabled)
	}
}

func TestServiceEnableRejectsRuntimeLocaleConflict(t *testing.T) {
	store := &fakeStore{items: map[string]LocalePack{
		"ja.one": installedPack("ja.one", "ja-JP", false),
		"ja.two": installedPack("ja.two", "ja-JP", false),
	}}
	store.items["ja.one"].Status = StatusEnabled
	service := NewService(store, t.TempDir())

	_, err := service.Enable(context.Background(), localeManager(), EnableInput{
		PackID:  "ja.two",
		Version: "1.0.0",
	})
	if !errors.Is(err, ErrLocaleConflict) {
		t.Fatalf("expected locale conflict, got %v", err)
	}
}

func TestServiceRuntimeMessagesReadsEnabledFrontendMessages(t *testing.T) {
	root := t.TempDir()
	store := &fakeStore{}
	service := NewService(store, root)
	installed, err := service.InstallArchive(context.Background(), localeManager(), ArchiveInput{
		FileName: "ja.zip",
		Data: localeArchive(t, validManifest("ja.pack", "ja-JP", false),
			zipFile{name: "messages/ja-JP.json", body: `{"nav":{"home":"ホーム"}}`},
		),
	})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	store.items[installed.ID].Status = StatusEnabled

	messages, err := service.RuntimeMessages(context.Background(), "ja-JP")
	if err != nil {
		t.Fatalf("RuntimeMessages returned error: %v", err)
	}
	if messages.Locale != "ja-JP" || messages.Messages["nav"].(map[string]any)["home"] != "ホーム" {
		t.Fatalf("unexpected runtime messages: %#v", messages)
	}
}

func localeManager() identity.Actor {
	return identity.Actor{ID: 42, Status: identity.UserStatusActive, Permissions: map[string]bool{identity.PermissionLocaleManage: true}}
}

func validManifest(id string, locale string, allowOverride bool) string {
	override := "false"
	if allowOverride {
		override = "true"
	}
	return `{"id":"` + id + `","name":"Demo Locale Pack","version":"1.0.0","sforumVersion":"^1.0.0","allowBuiltinOverride":` + override + `,"locales":[{"code":"` + locale + `","name":"Demo","frontend":"messages/` + locale + `.json","backend":"backend/` + locale + `.json"}]}`
}

type zipFile struct {
	name string
	body string
}

func localeArchive(t *testing.T, manifest string, files ...zipFile) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	if manifest != "" {
		writeZipFile(t, writer, ManifestFileName, manifest)
	}
	for _, file := range files {
		writeZipFile(t, writer, file.name, file.body)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buffer.Bytes()
}

func writeZipFile(t *testing.T, writer *zip.Writer, name string, body string) {
	t.Helper()
	file, err := writer.Create(name)
	if err != nil {
		t.Fatalf("create zip file %s: %v", name, err)
	}
	if _, err := io.WriteString(file, body); err != nil {
		t.Fatalf("write zip file %s: %v", name, err)
	}
}

func installedPack(id string, locale string, allowOverride bool) LocalePack {
	return LocalePack{
		ID:                   id,
		Name:                 "Demo Locale Pack",
		Version:              "1.0.0",
		Status:               StatusInstalled,
		AllowBuiltinOverride: allowOverride,
		Manifest: Manifest{
			ID:                   id,
			Name:                 "Demo Locale Pack",
			Version:              "1.0.0",
			SForumVersion:        "^1.0.0",
			AllowBuiltinOverride: allowOverride,
			Locales: []ManifestLocale{{Code: locale, Name: "Demo", Frontend: "messages/" + locale + ".json"}},
		},
		Locales: []LocalePackLocale{{LocaleCode: locale, DisplayName: "Demo", FrontendPath: "messages/" + locale + ".json", OverridesBuiltin: locale == "zh-CN" || locale == "en-US"}},
		InstalledAt: time.Now(),
		UpdatedAt:   time.Now(),
	}
}
```

Add fake store methods at the bottom:

```go
type fakeStore struct {
	items  map[string]LocalePack
	events []LocalePackEvent
}

func (s *fakeStore) List(context.Context) ([]LocalePack, error) {
	items := make([]LocalePack, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	return items, nil
}

func (s *fakeStore) ListEnabled(context.Context) ([]LocalePack, error) {
	items := []LocalePack{}
	for _, item := range s.items {
		if item.Status == StatusEnabled {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *fakeStore) Get(_ context.Context, id string) (LocalePack, error) {
	if item, ok := s.items[id]; ok {
		return item, nil
	}
	return LocalePack{}, ErrPackNotFound
}

func (s *fakeStore) SaveInstalled(_ context.Context, input SaveInstalledInput) (LocalePack, error) {
	item := LocalePack{
		ID:                   input.Manifest.ID,
		Name:                 input.Manifest.Name,
		Version:              input.Manifest.Version,
		Status:               StatusInstalled,
		AllowBuiltinOverride: input.Manifest.AllowBuiltinOverride,
		Manifest:             input.Manifest,
		PackagePath:          input.PackagePath,
		MessagesPath:         input.MessagesPath,
		Locales:              input.Locales,
		InstalledAt:          time.Now(),
		UpdatedAt:            time.Now(),
	}
	if s.items == nil {
		s.items = map[string]LocalePack{}
	}
	s.items[item.ID] = item
	return item, nil
}

func (s *fakeStore) Enable(_ context.Context, input EnableInput) (LocalePack, error) {
	item, ok := s.items[input.PackID]
	if !ok {
		return LocalePack{}, ErrPackNotFound
	}
	item.Status = StatusEnabled
	item.AllowBuiltinOverride = input.AllowBuiltinOverride
	item.UpdatedAt = time.Now()
	s.items[input.PackID] = item
	return item, nil
}

func (s *fakeStore) Disable(_ context.Context, id string) (LocalePack, error) {
	item, ok := s.items[id]
	if !ok {
		return LocalePack{}, ErrPackNotFound
	}
	item.Status = StatusDisabled
	item.UpdatedAt = time.Now()
	s.items[id] = item
	return item, nil
}

func (s *fakeStore) CreateEvent(_ context.Context, input EventInput) (LocalePackEvent, error) {
	event := LocalePackEvent{ID: int64(len(s.events) + 1), PackID: input.PackID, ActorUserID: input.ActorUserID, Action: input.Action, Message: input.Message, CreatedAt: time.Now()}
	s.events = append(s.events, event)
	return event, nil
}

func (s *fakeStore) ListEvents(context.Context, string, int) ([]LocalePackEvent, error) {
	return s.events, nil
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
go test ./app/Models/Localization -count=1
```

Expected: FAIL because the package and types do not exist.

- [ ] **Step 3: Create domain types**

Create `apps/api/app/Models/Localization/types.go`:

```go
package localizationmodel

import (
	"errors"
	"time"
)

const (
	ManifestFileName = "sforum.locale.json"

	StatusInstalled = "installed"
	StatusEnabled   = "enabled"
	StatusDisabled  = "disabled"

	EventInstalled    = "installed"
	EventEnabled      = "enabled"
	EventEnableFailed = "enable_failed"
	EventDisabled     = "disabled"

	CodeInvalidArchive  = "locale_pack.archive_invalid"
	CodeInvalidManifest = "locale_pack.manifest_invalid"
	CodeInvalidMessages = "locale_pack.messages_invalid"
	CodeNotFound        = "locale_pack.not_found"
	CodeOverrideDenied  = "locale_pack.override_denied"
	CodeLocaleConflict  = "locale_pack.locale_conflict"
)

var (
	ErrInvalidArchive  = errors.New("locale packs: invalid archive")
	ErrInvalidManifest = errors.New("locale packs: invalid manifest")
	ErrInvalidMessages = errors.New("locale packs: invalid messages")
	ErrPackNotFound    = errors.New("locale packs: not found")
	ErrOverrideDenied  = errors.New("locale packs: builtin override denied")
	ErrLocaleConflict  = errors.New("locale packs: locale conflict")
)

type Manifest struct {
	ID                   string           `json:"id"`
	Name                 string           `json:"name"`
	Version              string           `json:"version"`
	SForumVersion        string           `json:"sforumVersion"`
	Locales              []ManifestLocale `json:"locales"`
	AllowBuiltinOverride bool             `json:"allowBuiltinOverride"`
}

type ManifestLocale struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Frontend string `json:"frontend"`
	Backend  string `json:"backend"`
}

type LocalePack struct {
	ID                   string             `json:"id"`
	Name                 string             `json:"name"`
	Version              string             `json:"version"`
	Status               string             `json:"status"`
	AllowBuiltinOverride bool               `json:"allowBuiltinOverride"`
	Manifest             Manifest           `json:"manifest"`
	PackagePath          string             `json:"packagePath"`
	MessagesPath         string             `json:"messagesPath"`
	Locales              []LocalePackLocale `json:"locales"`
	InstalledAt          time.Time          `json:"installedAt"`
	UpdatedAt            time.Time          `json:"updatedAt"`
}

type LocalePackLocale struct {
	LocaleCode      string `json:"localeCode"`
	DisplayName     string `json:"displayName"`
	FrontendPath    string `json:"frontendPath"`
	BackendPath     string `json:"backendPath"`
	MessageKeyCount  int    `json:"messageKeyCount"`
	MissingKeyCount  int    `json:"missingKeyCount"`
	OverridesBuiltin bool   `json:"overridesBuiltin"`
}

type LocalePackEvent struct {
	ID          int64     `json:"id"`
	PackID      string    `json:"packId"`
	ActorUserID int64     `json:"actorUserId"`
	Action      string    `json:"action"`
	Message     string    `json:"message"`
	CreatedAt   time.Time `json:"createdAt"`
}

type ArchiveInput struct {
	FileName string
	Data     []byte
}

type SaveInstalledInput struct {
	Manifest     Manifest
	PackagePath  string
	MessagesPath string
	Locales       []LocalePackLocale
}

type EnableInput struct {
	PackID               string `json:"packId"`
	Version              string `json:"version"`
	AllowBuiltinOverride bool   `json:"allowBuiltinOverride"`
}

type EventInput struct {
	PackID      string
	ActorUserID int64
	Action      string
	Message     string
}

type LocaleSummary struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Source  string `json:"source"`
	Runtime bool   `json:"runtime"`
	Route   bool   `json:"route"`
}

type LocaleSettings struct {
	DefaultLocale string          `json:"defaultLocale"`
	Enabled       []string        `json:"enabledLocales"`
	RouteLocales  []LocaleSummary `json:"routeLocales"`
	RuntimeLocales []LocaleSummary `json:"runtimeLocales"`
}

type RuntimeMessages struct {
	Locale   string         `json:"locale"`
	Version  string         `json:"version"`
	Messages map[string]any `json:"messages"`
}
```

- [ ] **Step 4: Create the store interface**

Create `apps/api/app/Models/Localization/store.go`:

```go
package localizationmodel

import "context"

type Store interface {
	List(ctx context.Context) ([]LocalePack, error)
	ListEnabled(ctx context.Context) ([]LocalePack, error)
	Get(ctx context.Context, id string) (LocalePack, error)
	SaveInstalled(ctx context.Context, input SaveInstalledInput) (LocalePack, error)
	Enable(ctx context.Context, input EnableInput) (LocalePack, error)
	Disable(ctx context.Context, id string) (LocalePack, error)
	CreateEvent(ctx context.Context, input EventInput) (LocalePackEvent, error)
	ListEvents(ctx context.Context, packID string, limit int) ([]LocalePackEvent, error)
}
```

- [ ] **Step 5: Implement the service**

Create `apps/api/app/Models/Localization/service.go` with these functions and constants:

```go
package localizationmodel

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

const maxArchiveBytes = 10 * 1024 * 1024

var packIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,80}$`)
var localeCodePattern = regexp.MustCompile(`^[a-z]{2,3}-[A-Z][A-Za-z]{1,3}$`)
var builtInLocaleSet = map[string]bool{"zh-CN": true, "en-US": true}

type Service struct {
	store        Store
	localeRoot   string
	builtinKeys  map[string]bool
}

func NewService(store Store, localeRoot string) *Service {
	if strings.TrimSpace(localeRoot) == "" {
		localeRoot = "storage/locale-packs"
	}
	return &Service{store: store, localeRoot: localeRoot, builtinKeys: map[string]bool{}}
}

func NewServiceWithBuiltinKeys(store Store, localeRoot string, builtinKeys []string) *Service {
	service := NewService(store, localeRoot)
	for _, key := range builtinKeys {
		service.builtinKeys[key] = true
	}
	return service
}
```

Implement methods `List`, `InstallArchive`, `Enable`, `Disable`, `Events`, `PublicLocales`, and `RuntimeMessages` in the same file. The implementation must keep all filesystem writes under `s.localeRoot`, normalize manifest fields before validation, and return the domain errors from `types.go` instead of raw JSON or ZIP parsing errors. Include helper functions `readArchive`, `readZipFile`, `safeArchivePath`, `normalizeManifest`, `validateManifest`, `flattenMessageKeys`, `isBuiltinLocale`, and `localeDisplayName`.

The important behavior is:

```go
func (s *Service) InstallArchive(ctx context.Context, actor identity.Actor, input ArchiveInput) (LocalePack, error) {
	if !actor.Can(identity.PermissionLocaleManage) {
		return LocalePack{}, identity.ErrPermissionDenied
	}
	if len(input.Data) == 0 || len(input.Data) > maxArchiveBytes {
		return LocalePack{}, ErrInvalidArchive
	}
	manifest, files, err := readArchive(input.Data)
	if err != nil {
		return LocalePack{}, err
	}
	if err := validateManifest(manifest); err != nil {
		return LocalePack{}, err
	}
	locales, err := s.validateMessageFiles(manifest, files)
	if err != nil {
		return LocalePack{}, err
	}
	versionDir := filepath.Join(s.localeRoot, manifest.ID, manifest.Version)
	if err := os.MkdirAll(filepath.Join(versionDir, "files"), 0o755); err != nil {
		return LocalePack{}, err
	}
	packagePath := filepath.Join(versionDir, "package.zip")
	if err := os.WriteFile(packagePath, input.Data, 0o600); err != nil {
		return LocalePack{}, err
	}
	if err := writeManifest(versionDir, manifest); err != nil {
		return LocalePack{}, err
	}
	if err := extractArchiveFiles(versionDir, files); err != nil {
		return LocalePack{}, err
	}
	installed, err := s.store.SaveInstalled(ctx, SaveInstalledInput{Manifest: manifest, PackagePath: packagePath, MessagesPath: filepath.Join(versionDir, "files"), Locales: locales})
	if err != nil {
		return LocalePack{}, err
	}
	_, _ = s.store.CreateEvent(ctx, EventInput{PackID: installed.ID, ActorUserID: actor.ID, Action: EventInstalled, Message: "Locale pack installed."})
	return installed, nil
}
```

Implement `Enable` so it checks permission, loads the target pack, rejects built-in overrides without double opt-in, rejects locale conflicts against other enabled packs, then calls `store.Enable`.

Use this exact conflict rule in `Enable`: for every locale in the target pack, if the locale is not built-in and any other enabled pack already provides the same locale code, return `ErrLocaleConflict` before changing state.

Use this exact `RuntimeMessages` rule: load enabled packs from `store.ListEnabled(ctx)`, find the first enabled pack locale with a matching `LocaleCode`, read `filepath.Join(pack.MessagesPath, locale.FrontendPath)`, decode it into `map[string]any`, and return `ErrPackNotFound` if no enabled pack provides the locale.

- [ ] **Step 6: Implement PostgreSQL store**

Create `apps/api/app/Models/Localization/postgres_store.go` with a `PostgresStore` struct containing `pool *pgxpool.Pool`. Use explicit SQL in this file and scan `manifest JSONB` into `Manifest` with `encoding/json`. Include:

```go
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore
func (s *PostgresStore) List(ctx context.Context) ([]LocalePack, error)
func (s *PostgresStore) ListEnabled(ctx context.Context) ([]LocalePack, error)
func (s *PostgresStore) Get(ctx context.Context, id string) (LocalePack, error)
func (s *PostgresStore) SaveInstalled(ctx context.Context, input SaveInstalledInput) (LocalePack, error)
func (s *PostgresStore) Enable(ctx context.Context, input EnableInput) (LocalePack, error)
func (s *PostgresStore) Disable(ctx context.Context, id string) (LocalePack, error)
func (s *PostgresStore) CreateEvent(ctx context.Context, input EventInput) (LocalePackEvent, error)
func (s *PostgresStore) ListEvents(ctx context.Context, packID string, limit int) ([]LocalePackEvent, error)
```

`SaveInstalled` must insert/update `locale_packs`, insert the version, replace `locale_pack_locales` rows for that version, set `active_version_id`, and return `Get(ctx, id)`.

- [ ] **Step 7: Run backend domain tests**

Run:

```bash
gofmt -w apps/api/app/Models/Localization
go test ./app/Models/Localization -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add apps/api/app/Models/Localization
git commit -m "feat: add locale pack domain service"
```

---

## Task 3: Add HTTP Controller, Provider, And Bootstrap Wiring

**Files:**
- Create: `apps/api/app/Http/Controllers/Localization/routes.go`
- Create: `apps/api/app/Http/Controllers/Localization/controller.go`
- Create: `apps/api/app/Http/Controllers/Localization/controller_test.go`
- Create: `apps/api/app/Providers/localization.go`
- Modify: `apps/api/bootstrap/app.go`

- [ ] **Step 1: Write controller tests first**

Create `apps/api/app/Http/Controllers/Localization/controller_test.go` with an in-memory Fiber app, a session login route, a fake `identity.ActorStore`, and a fake localization store. Cover:

```go
func TestControllerRequiresLoginAndLocaleManagePermission(t *testing.T)
func TestControllerListsAndEnablesLocalePacksForManager(t *testing.T)
func TestControllerPublicLocalesAndRuntimeMessages(t *testing.T)
```

Use actors:

```go
Permissions: map[string]bool{identity.PermissionLocaleManage: true}
```

Expected assertions:

```go
if resp.StatusCode != http.StatusUnauthorized { t.Fatalf(...) }
if resp.StatusCode != http.StatusForbidden { t.Fatalf(...) }
if body.Data.Reason != "permission.denied" { t.Fatalf(...) }
if listBody.Data[0].ID != "ja.pack" { t.Fatalf(...) }
if messagesBody.Data.Locale != "ja-JP" { t.Fatalf(...) }
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
go test ./app/Http/Controllers/Localization -count=1
```

Expected: FAIL because the controller package does not exist.

- [ ] **Step 3: Create routes**

Create `apps/api/app/Http/Controllers/Localization/routes.go`:

```go
package localizationcontroller

import "github.com/gofiber/fiber/v3"

func (h *Controller) RegisterRoutes(api fiber.Router) {
	api.Get("/locales", h.listPublicLocales)
	api.Get("/locale-messages/:locale", h.runtimeMessages)
	api.Get("/admin/locale-packs", h.listPacks)
	api.Post("/admin/locale-packs", h.install)
	api.Post("/admin/locale-packs/:id/versions/:version/enable", h.enable)
	api.Post("/admin/locale-packs/:id/disable", h.disable)
	api.Get("/admin/locale-packs/:id/events", h.events)
	api.Put("/admin/locales/settings", h.updateSettings)
}
```

- [ ] **Step 4: Create controller**

Create `apps/api/app/Http/Controllers/Localization/controller.go`:

```go
package localizationcontroller

import (
	"errors"
	"io"
	"strconv"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	localizationmodel "github.com/zhuchunshu/sforum/apps/api/app/Models/Localization"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
)

const maxUploadedArchiveBytes = 12 * 1024 * 1024

type Controller struct {
	service  *localizationmodel.Service
	users    identity.ActorStore
	sessions *authsession.Manager
}

func NewController(service *localizationmodel.Service, users identity.ActorStore, sessions *authsession.Manager) *Controller {
	return &Controller{service: service, users: users, sessions: sessions}
}
```

Implement handlers:

```go
func (h *Controller) listPublicLocales(c fiber.Ctx) error {
	settings, err := h.service.PublicLocales(c.Context())
	if err != nil {
		return mapLocaleError(err)
	}
	return apphttp.OK(c, settings)
}
```

For `install`, mirror the extension upload controller and call `InstallArchive`. For `enable`, bind JSON body into `EnableInput`, set `PackID` and `Version` from route params, and call `service.Enable`.

Implement `actor(c fiber.Ctx)` exactly like extension controller, using `sessions.CurrentUserID` and `users.LoadActor`.

Map errors:

```go
case errors.Is(err, identity.ErrPermissionDenied):
	return fiber.NewError(fiber.StatusForbidden, "permission.denied")
case errors.Is(err, localizationmodel.ErrInvalidArchive):
	return fiber.NewError(fiber.StatusUnprocessableEntity, localizationmodel.CodeInvalidArchive)
case errors.Is(err, localizationmodel.ErrInvalidManifest):
	return fiber.NewError(fiber.StatusUnprocessableEntity, localizationmodel.CodeInvalidManifest)
case errors.Is(err, localizationmodel.ErrInvalidMessages):
	return fiber.NewError(fiber.StatusUnprocessableEntity, localizationmodel.CodeInvalidMessages)
case errors.Is(err, localizationmodel.ErrOverrideDenied):
	return fiber.NewError(fiber.StatusConflict, localizationmodel.CodeOverrideDenied)
case errors.Is(err, localizationmodel.ErrLocaleConflict):
	return fiber.NewError(fiber.StatusConflict, localizationmodel.CodeLocaleConflict)
case errors.Is(err, localizationmodel.ErrPackNotFound):
	return fiber.NewError(fiber.StatusNotFound, localizationmodel.CodeNotFound)
```

- [ ] **Step 5: Create provider and wire bootstrap**

Create `apps/api/app/Providers/localization.go`:

```go
package providers

import (
	"github.com/gofiber/fiber/v3"

	localizationcontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/Localization"
	localizationmodel "github.com/zhuchunshu/sforum/apps/api/app/Models/Localization"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
)

type LocalizationProvider struct {
	controller *localizationcontroller.Controller
}

func NewLocalizationProvider(store localizationmodel.Store, users identity.ActorStore, sessions *authsession.Manager, localeRoot string) *LocalizationProvider {
	return &LocalizationProvider{
		controller: localizationcontroller.NewController(localizationmodel.NewService(store, localeRoot), users, sessions),
	}
}

func (p *LocalizationProvider) RegisterRoutes(api fiber.Router) {
	p.controller.RegisterRoutes(api)
}
```

In `apps/api/bootstrap/app.go`, import the model and create:

```go
localizationStore := localizationmodel.NewPostgresStore(pool)
localizationProvider := providers.NewLocalizationProvider(localizationStore, identityStore, authSessions, cfg.LocalePackRoot)
```

Add `localizationProvider` to `RouteProviders`.

- [ ] **Step 6: Run controller and bootstrap tests**

Run:

```bash
gofmt -w apps/api/app/Http/Controllers/Localization apps/api/app/Providers/localization.go apps/api/bootstrap/app.go
go test ./app/Http/Controllers/Localization -count=1
go test ./bootstrap -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/api/app/Http/Controllers/Localization apps/api/app/Providers/localization.go apps/api/bootstrap/app.go
git commit -m "feat: expose locale pack APIs"
```

---

## Task 4: Connect Locale Settings Validation To Runtime Locales

**Files:**
- Modify: `apps/api/app/Models/Options/service.go`
- Modify: `apps/api/app/Models/Options/service_test.go`
- Modify: `apps/api/app/Http/Controllers/Localization/controller.go`
- Modify: `apps/api/app/Providers/localization.go`

- [ ] **Step 1: Write Options service tests for runtime locale validation**

Add to `apps/api/app/Models/Options/service_test.go`:

```go
func TestServiceUpdateManyAllowsConfiguredRuntimeLocales(t *testing.T) {
	service := NewServiceWithAllowedLocales(&fakeStore{}, []string{"zh-CN", "en-US", "ja-JP"}, time.Minute)
	actor := settingsActor()

	updated, err := service.UpdateMany(context.Background(), actor, []UpdateInput{
		{Name: NameSiteSupportedLocales, Value: "zh-CN,ja-JP"},
		{Name: NameSiteDefaultLocale, Value: "ja-JP"},
	})
	if err != nil {
		t.Fatalf("UpdateMany returned error: %v", err)
	}
	if value := adminValue(updated, NameSiteDefaultLocale); value != "ja-JP" {
		t.Fatalf("expected runtime default locale, got %q", value)
	}
}

func TestServiceUpdateManyRejectsUnknownRuntimeLocale(t *testing.T) {
	service := NewServiceWithAllowedLocales(&fakeStore{}, []string{"zh-CN", "en-US"}, time.Minute)
	actor := settingsActor()

	_, err := service.UpdateMany(context.Background(), actor, []UpdateInput{
		{Name: NameSiteSupportedLocales, Value: "zh-CN,ja-JP"},
		{Name: NameSiteDefaultLocale, Value: "ja-JP"},
	})
	if !errors.Is(err, ErrInvalidOption) {
		t.Fatalf("expected invalid option for unknown locale, got %v", err)
	}
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
go test ./app/Models/Options -run RuntimeLocales -count=1
```

Expected: FAIL because `NewServiceWithAllowedLocales` does not exist.

- [ ] **Step 3: Add allowed locale support**

In `apps/api/app/Models/Options/service.go`, add `allowedLocales []string` to `Service`.

Add constructor:

```go
func NewServiceWithAllowedLocales(store Store, allowedLocales []string, cacheTTL time.Duration) *Service {
	service := NewServiceWithDefaultsAndCacheTTL(store, Defaults{}, cacheTTL)
	service.allowedLocales = normalizeLocaleListForOptions(allowedLocales)
	return service
}
```

Replace uses of `builtInLocales` for site default/supported locale normalization with:

```go
func (s *Service) localeChoices() []string {
	if len(s.allowedLocales) == 0 {
		return builtInLocales
	}
	return s.allowedLocales
}
```

Update `normalizeOptionValue`, `parseStoredLocales`, and validation helpers to accept allowed locale choices as parameters where needed.

- [ ] **Step 4: Add localization settings handler**

In the localization controller, make `PUT /admin/locales/settings` call a new service method that receives:

```go
type LocaleSettingsInput struct {
	DefaultLocale  string   `json:"defaultLocale"`
	EnabledLocales []string `json:"enabledLocales"`
}
```

The service should:

1. Require `locale.manage`.
2. Build allowed locales from built-ins plus enabled packs.
3. Update `site.default_locale` and `site.supported_locales` through the `RuntimeOptionUpdater` interface shown below.

Create this narrow interface in `apps/api/app/Models/Localization/types.go` and make the localization service depend on it instead of importing the Options service directly:

```go
type RuntimeOptionUpdater interface {
	UpdateMany(ctx context.Context, actor identity.Actor, inputs []options.UpdateInput) ([]options.AdminOption, error)
}
```

- [ ] **Step 5: Run focused tests**

Run:

```bash
gofmt -w apps/api/app/Models/Options apps/api/app/Http/Controllers/Localization apps/api/app/Providers/localization.go
go test ./app/Models/Options -count=1
go test ./app/Http/Controllers/Localization -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/api/app/Models/Options apps/api/app/Http/Controllers/Localization apps/api/app/Providers/localization.go
git commit -m "feat: validate runtime locale settings"
```

---

## Task 5: Add OpenAPI Contract

**Files:**
- Create: `contracts/openapi/paths/localization.yaml`
- Create: `contracts/openapi/schemas/localization.yaml`
- Modify: `contracts/openapi.yaml`

- [ ] **Step 1: Add localization schemas**

Create `contracts/openapi/schemas/localization.yaml` with:

```yaml
LocaleSource:
  type: string
  enum: [builtin, runtime]
LocaleSummary:
  type: object
  required: [code, name, source, runtime, route]
  properties:
    code: { type: string, example: ja-JP }
    name: { type: string, example: 日本語 }
    source:
      "$ref": "#/LocaleSource"
    runtime: { type: boolean }
    route: { type: boolean }
LocaleSettings:
  type: object
  required: [defaultLocale, enabledLocales, routeLocales, runtimeLocales]
  properties:
    defaultLocale: { type: string, example: zh-CN }
    enabledLocales:
      type: array
      items: { type: string }
    routeLocales:
      type: array
      items:
        "$ref": "#/LocaleSummary"
    runtimeLocales:
      type: array
      items:
        "$ref": "#/LocaleSummary"
RuntimeLocaleMessages:
  type: object
  required: [locale, version, messages]
  properties:
    locale: { type: string, example: ja-JP }
    version: { type: string, example: ja.pack@1.0.0 }
    messages:
      type: object
      additionalProperties: true
LocalePackStatus:
  type: string
  enum: [installed, enabled, disabled]
LocalePackLocale:
  type: object
  required: [localeCode, displayName, frontendPath, messageKeyCount, missingKeyCount, overridesBuiltin]
  properties:
    localeCode: { type: string }
    displayName: { type: string }
    frontendPath: { type: string }
    backendPath: { type: string }
    messageKeyCount: { type: integer }
    missingKeyCount: { type: integer }
    overridesBuiltin: { type: boolean }
LocalePack:
  type: object
  required: [id, name, version, status, allowBuiltinOverride, locales, installedAt, updatedAt]
  properties:
    id: { type: string }
    name: { type: string }
    version: { type: string }
    status:
      "$ref": "#/LocalePackStatus"
    allowBuiltinOverride: { type: boolean }
    locales:
      type: array
      items:
        "$ref": "#/LocalePackLocale"
    installedAt: { type: string, format: date-time }
    updatedAt: { type: string, format: date-time }
LocalePackEnableInput:
  type: object
  properties:
    allowBuiltinOverride: { type: boolean, default: false }
LocaleSettingsInput:
  type: object
  required: [defaultLocale, enabledLocales]
  properties:
    defaultLocale: { type: string }
    enabledLocales:
      type: array
      items: { type: string }
LocalePackEvent:
  type: object
  required: [id, packId, action, message, createdAt]
  properties:
    id: { type: integer, format: int64 }
    packId: { type: string }
    actorUserId: { type: integer, format: int64 }
    action: { type: string }
    message: { type: string }
    createdAt: { type: string, format: date-time }
ApiResponseLocaleSettings:
  allOf:
  - "$ref": "./common.yaml#/ApiResponseBase"
  - type: object
    properties:
      data:
        "$ref": "#/LocaleSettings"
ApiResponseRuntimeLocaleMessages:
  allOf:
  - "$ref": "./common.yaml#/ApiResponseBase"
  - type: object
    properties:
      data:
        "$ref": "#/RuntimeLocaleMessages"
ApiResponseLocalePack:
  allOf:
  - "$ref": "./common.yaml#/ApiResponseBase"
  - type: object
    properties:
      data:
        "$ref": "#/LocalePack"
ApiResponseLocalePackList:
  allOf:
  - "$ref": "./common.yaml#/ApiResponseBase"
  - type: object
    properties:
      data:
        type: array
        items:
          "$ref": "#/LocalePack"
ApiResponseLocalePackEventList:
  allOf:
  - "$ref": "./common.yaml#/ApiResponseBase"
  - type: object
    properties:
      data:
        type: array
        items:
          "$ref": "#/LocalePackEvent"
```

- [ ] **Step 2: Add paths**

Create `contracts/openapi/paths/localization.yaml`:

```yaml
locales:
  get:
    summary: List available locales
    operationId: listLocales
    responses:
      '200':
        description: Locale settings and available locale sources.
        content:
          application/json:
            schema:
              "$ref": "../schemas/localization.yaml#/ApiResponseLocaleSettings"
localeMessages:
  get:
    summary: Get runtime locale messages
    operationId: getRuntimeLocaleMessages
    parameters:
    - name: locale
      in: path
      required: true
      schema: { type: string }
    responses:
      '200':
        description: Runtime frontend message overlay.
        content:
          application/json:
            schema:
              "$ref": "../schemas/localization.yaml#/ApiResponseRuntimeLocaleMessages"
      '404':
        "$ref": "../components/responses.yaml#/NotFound"
      '503':
        "$ref": "../components/responses.yaml#/ServiceUnavailable"
adminLocalePacks:
  get:
    summary: List locale packs
    operationId: listLocalePacks
    responses:
      '200':
        description: Locale packs visible to administrators.
        content:
          application/json:
            schema:
              "$ref": "../schemas/localization.yaml#/ApiResponseLocalePackList"
      '401':
        "$ref": "../components/responses.yaml#/Unauthorized"
      '403':
        "$ref": "../components/responses.yaml#/Forbidden"
  post:
    summary: Upload a locale pack ZIP
    operationId: uploadLocalePack
    requestBody:
      required: true
      content:
        multipart/form-data:
          schema:
            type: object
            required: [file]
            properties:
              file:
                type: string
                format: binary
    responses:
      '201':
        description: Locale pack installed.
        content:
          application/json:
            schema:
              "$ref": "../schemas/localization.yaml#/ApiResponseLocalePack"
      '401':
        "$ref": "../components/responses.yaml#/Unauthorized"
      '403':
        "$ref": "../components/responses.yaml#/Forbidden"
      '422':
        "$ref": "../components/responses.yaml#/UnprocessableEntity"
adminLocalePackEnable:
  post:
    summary: Enable a locale pack version
    operationId: enableLocalePack
    requestBody:
      required: false
      content:
        application/json:
          schema:
            "$ref": "../schemas/localization.yaml#/LocalePackEnableInput"
    responses:
      '200':
        description: Locale pack enabled.
        content:
          application/json:
            schema:
              "$ref": "../schemas/localization.yaml#/ApiResponseLocalePack"
      '409':
        "$ref": "../components/responses.yaml#/Conflict"
adminLocalePackDisable:
  post:
    summary: Disable a locale pack
    operationId: disableLocalePack
    responses:
      '200':
        description: Locale pack disabled.
        content:
          application/json:
            schema:
              "$ref": "../schemas/localization.yaml#/ApiResponseLocalePack"
adminLocalePackEvents:
  get:
    summary: List locale pack events
    operationId: listLocalePackEvents
    responses:
      '200':
        description: Locale pack event list.
        content:
          application/json:
            schema:
              "$ref": "../schemas/localization.yaml#/ApiResponseLocalePackEventList"
adminLocaleSettings:
  put:
    summary: Update language settings
    operationId: updateLocaleSettings
    requestBody:
      required: true
      content:
        application/json:
          schema:
            "$ref": "../schemas/localization.yaml#/LocaleSettingsInput"
    responses:
      '200':
        description: Language settings updated.
        content:
          application/json:
            schema:
              "$ref": "../schemas/localization.yaml#/ApiResponseLocaleSettings"
      '401':
        "$ref": "../components/responses.yaml#/Unauthorized"
      '403':
        "$ref": "../components/responses.yaml#/Forbidden"
      '422':
        "$ref": "../components/responses.yaml#/UnprocessableEntity"
```

- [ ] **Step 3: Wire root OpenAPI refs**

In `contracts/openapi.yaml`, add paths:

```yaml
  "/locales":
    "$ref": "./openapi/paths/localization.yaml#/locales"
  "/locale-messages/{locale}":
    "$ref": "./openapi/paths/localization.yaml#/localeMessages"
  "/admin/locale-packs":
    "$ref": "./openapi/paths/localization.yaml#/adminLocalePacks"
  "/admin/locale-packs/{packID}/versions/{version}/enable":
    "$ref": "./openapi/paths/localization.yaml#/adminLocalePackEnable"
  "/admin/locale-packs/{packID}/disable":
    "$ref": "./openapi/paths/localization.yaml#/adminLocalePackDisable"
  "/admin/locale-packs/{packID}/events":
    "$ref": "./openapi/paths/localization.yaml#/adminLocalePackEvents"
  "/admin/locales/settings":
    "$ref": "./openapi/paths/localization.yaml#/adminLocaleSettings"
```

Add schemas in `components.schemas` for the localization response and object schemas.

- [ ] **Step 4: Validate OpenAPI refs**

Run:

```bash
ruby scripts/validate-openapi-refs.rb
```

Expected: PASS with no unresolved references.

- [ ] **Step 5: Commit**

```bash
git add contracts/openapi.yaml contracts/openapi/paths/localization.yaml contracts/openapi/schemas/localization.yaml
git commit -m "docs: add localization api contract"
```

---

## Task 6: Add Frontend Runtime Locale Loader

**Files:**
- Create: `apps/web/app/composables/useRuntimeLocales.ts`
- Create: `apps/web/app/plugins/runtime-locales.client.ts`
- Create: `apps/web/tests/runtimeLocales.test.ts`
- Modify: `apps/web/app/components/SFNavbar.vue`

- [ ] **Step 1: Write frontend runtime locale tests**

Create `apps/web/tests/runtimeLocales.test.ts`:

```ts
import { describe, expect, it } from 'bun:test'
import {
  mergeRuntimeMessages,
  resolveLocaleTarget,
  type LocaleSummary
} from '../app/composables/useRuntimeLocales'

describe('runtime locales', () => {
  it('distinguishes route locales from runtime-only locales', () => {
    const locales: LocaleSummary[] = [
      { code: 'zh-CN', name: '简体中文', source: 'builtin', runtime: false, route: true },
      { code: 'en-US', name: 'English', source: 'builtin', runtime: false, route: true },
      { code: 'ja-JP', name: '日本語', source: 'runtime', runtime: true, route: false }
    ]

    expect(resolveLocaleTarget('en-US', locales)).toEqual({ code: 'en', route: true })
    expect(resolveLocaleTarget('ja-JP', locales)).toEqual({ code: 'ja-JP', route: false })
  })

  it('deep-merges runtime messages over built-in messages', () => {
    const merged = mergeRuntimeMessages(
      { nav: { home: '首页', login: '登录' }, auth: { login: '登录' } },
      { nav: { home: 'ホーム' } }
    )

    expect(merged.nav.home).toBe('ホーム')
    expect(merged.nav.login).toBe('登录')
    expect(merged.auth.login).toBe('登录')
  })
})
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
cd apps/web && bun test tests/runtimeLocales.test.ts
```

Expected: FAIL because `useRuntimeLocales` does not exist.

- [ ] **Step 3: Implement composable**

Create `apps/web/app/composables/useRuntimeLocales.ts`:

```ts
export type LocaleSource = 'builtin' | 'runtime'

export type LocaleSummary = {
  code: string
  name: string
  source: LocaleSource
  runtime: boolean
  route: boolean
}

export type LocaleSettings = {
  defaultLocale: string
  enabledLocales: string[]
  routeLocales: LocaleSummary[]
  runtimeLocales: LocaleSummary[]
}

export type RuntimeLocaleMessages = {
  locale: string
  version: string
  messages: Record<string, unknown>
}

export function resolveLocaleTarget(locale: string, locales: LocaleSummary[]) {
  const item = locales.find((entry) => entry.code === locale)
  if (item?.route) {
    return { code: locale === 'en-US' ? 'en' : locale, route: true }
  }
  return { code: locale, route: false }
}

export function mergeRuntimeMessages(base: Record<string, any>, overlay: Record<string, any>) {
  const output = { ...base }
  for (const [key, value] of Object.entries(overlay)) {
    if (isPlainObject(value) && isPlainObject(output[key])) {
      output[key] = mergeRuntimeMessages(output[key], value)
      continue
    }
    output[key] = value
  }
  return output
}

function isPlainObject(value: unknown): value is Record<string, any> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}
```

Add the composable function:

```ts
export const useRuntimeLocales = () => {
  const { request } = useApiClient()
  const settings = useState<LocaleSettings>('runtime-locales', () => ({
    defaultLocale: 'zh-CN',
    enabledLocales: ['zh-CN', 'en-US'],
    routeLocales: [
      { code: 'zh-CN', name: '简体中文', source: 'builtin', runtime: false, route: true },
      { code: 'en-US', name: 'English', source: 'builtin', runtime: false, route: true }
    ],
    runtimeLocales: []
  }))
  const loadedVersions = useState<Record<string, string>>('runtime-locale-message-versions', () => ({}))

  const allLocales = computed(() => [...settings.value.routeLocales, ...settings.value.runtimeLocales])

  async function refresh() {
    settings.value = await request<LocaleSettings>('/locales')
    return settings.value
  }

  async function loadMessages(locale: string) {
    const payload = await request<RuntimeLocaleMessages>(`/locale-messages/${encodeURIComponent(locale)}`)
    loadedVersions.value[locale] = payload.version
    return payload
  }

  return { settings, allLocales, loadedVersions, refresh, loadMessages }
}
```

- [ ] **Step 4: Add client plugin**

Create `apps/web/app/plugins/runtime-locales.client.ts`:

```ts
export default defineNuxtPlugin(async () => {
  const runtimeLocales = useRuntimeLocales()
  try {
    await runtimeLocales.refresh()
  } catch {
    // 运行时语言包不可用时保留内置语言，避免启动阶段阻塞页面。
  }
})
```

- [ ] **Step 5: Update navbar language picker**

In `apps/web/app/components/SFNavbar.vue`, use `useRuntimeLocales()` and merge built-in i18n locales with runtime locales. For runtime-only selections, call `loadMessages(localeCode)` and then set the Vue I18n locale without route navigation. Keep route locales on `switchLocalePath`.

Use this click handler:

```ts
async function selectLanguage(code: string) {
  const target = resolveLocaleTarget(code, runtimeLocaleOptions.value)
  langMenuOpen.value = false
  if (target.route) {
    await navigateTo(switchLocalePath(target.code))
    return
  }
  try {
    const payload = await runtimeLocales.loadMessages(code)
    const i18n = useNuxtApp().$i18n as any
    i18n.global?.mergeLocaleMessage?.(code, payload.messages)
    i18n.mergeLocaleMessage?.(code, payload.messages)
    locale.value = code
  } catch {
    // 失败时保留当前语言，避免导航栏切换造成空白文案。
  }
}
```

- [ ] **Step 6: Run frontend tests**

Run:

```bash
cd apps/web && bun test tests/runtimeLocales.test.ts
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/web/app/composables/useRuntimeLocales.ts apps/web/app/plugins/runtime-locales.client.ts apps/web/app/components/SFNavbar.vue apps/web/tests/runtimeLocales.test.ts
git commit -m "feat: load runtime locale messages"
```

---

## Task 7: Build Admin Language Settings Page

**Files:**
- Create: `apps/web/app/pages/admin/locales.vue`
- Modify: `apps/web/app/config/adminModules.ts`
- Modify: `apps/web/app/pages/admin/settings/index.vue`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`
- Modify: `tests/validate-admin-framework.ts`

- [ ] **Step 1: Add admin registry entry**

In `apps/web/app/config/adminModules.ts`, add:

```ts
{
  id: '/locales',
  labelKey: 'admin.nav.locales',
  icon: 'i-lucide-languages',
  componentName: 'AdminLocales',
  requiredPermissions: ['locale.manage']
}
```

Add `{ type: 'page', pageId: '/locales' }` inside the system folder.

- [ ] **Step 2: Add locale page translations**

In `apps/web/i18n/locales/zh-CN.json`, add keys under `admin`:

```json
"locales": {
  "metaTitle": "语言设置",
  "title": "语言设置",
  "recommended": "推荐默认设置",
  "enabledLanguages": "启用语言",
  "languagePacks": "语言包",
  "builtinOverrides": "内置语言覆盖",
  "upload": "上传语言包",
  "restoreDefaults": "恢复推荐默认值",
  "save": "保存语言设置",
  "saved": "语言设置已保存",
  "saveFailed": "语言设置保存失败",
  "uploadFailed": "语言包上传失败",
  "enable": "启用",
  "disable": "禁用",
  "coverage": "覆盖 {count} 个文案键，缺失 {missing} 个"
}
```

In `apps/web/i18n/locales/en-US.json`, add matching English values.

- [ ] **Step 3: Create admin page**

Create `apps/web/app/pages/admin/locales.vue`:

```vue
<script setup lang="ts">
import { useAdminPage } from '~/composables/useAdminPage'
import { enabledOptionValue } from '~/composables/useWebOptions'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminLocales' })

const { t } = useI18n()
const toast = useToast()
const adminPage = useAdminPage('/locales')
const runtimeLocales = useRuntimeLocales()
const saving = ref(false)
const uploading = ref(false)

const form = reactive({
  defaultLocale: 'zh-CN',
  enabledLocales: ['zh-CN', 'en-US'] as string[],
  allowBuiltinOverride: false
})

const { request } = useApiClient()

const { data: packs, refresh: refreshPacks } = await useAsyncData('admin-locale-packs', () => request<any[]>('/admin/locale-packs'), { default: () => [] })

await useAsyncData('admin-runtime-locales', async () => {
  const settings = await runtimeLocales.refresh()
  form.defaultLocale = settings.defaultLocale
  form.enabledLocales = [...settings.enabledLocales]
  return settings
})

const localeChoices = computed(() => runtimeLocales.allLocales.value)

async function saveSettings() {
  saving.value = true
  try {
    const settings = await request('/admin/locales/settings', {
      method: 'PUT',
      body: {
        defaultLocale: form.defaultLocale,
        enabledLocales: form.enabledLocales
      }
    })
    await runtimeLocales.refresh()
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('admin.locales.saved') })
    return settings
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.locales.saveFailed') })
  } finally {
    saving.value = false
  }
}

function restoreDefaults() {
  form.defaultLocale = 'zh-CN'
  form.enabledLocales = ['zh-CN', 'en-US']
}
</script>
```

Use Nuxt UI controls and existing `SFAdminFormFooter`. Keep layout dense and admin-like. Include file input upload using `FormData`:

```ts
async function uploadPack(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  uploading.value = true
  try {
    const formData = new FormData()
    formData.append('file', file)
    await request('/admin/locale-packs', { method: 'POST', body: formData })
    await refreshPacks()
    await runtimeLocales.refresh()
  } catch (error) {
    toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.locales.uploadFailed') })
  } finally {
    uploading.value = false
    input.value = ''
  }
}
```

- [ ] **Step 4: Move locale controls out of settings page**

In `apps/web/app/pages/admin/settings/index.vue`:

- Remove `defaultLocale` and `supportedLocales` from the basic settings form state.
- Remove locale checkbox UI.
- Remove locale updates from `saveBasicSettings`.
- Keep site name, site URL, and verification settings unchanged.

- [ ] **Step 5: Update admin framework validation**

In `tests/validate-admin-framework.ts`, ensure `/locales` is accepted as a registered admin page and that `AdminLocales` is a valid component name.

- [ ] **Step 6: Run frontend validation**

Run:

```bash
cd apps/web && bun test tests/runtimeLocales.test.ts
cd ../.. && bun run --cwd apps/web typecheck
```

Expected: tests and typecheck PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/web/app/pages/admin/locales.vue apps/web/app/config/adminModules.ts apps/web/app/pages/admin/settings/index.vue apps/web/i18n/locales/zh-CN.json apps/web/i18n/locales/en-US.json tests/validate-admin-framework.ts
git commit -m "feat: add admin language settings page"
```

---

## Task 8: Knowledge Base, End-To-End Verification, And Final Commit

**Files:**
- Modify: `knowledge/index.md`
- Modify: `knowledge/modules/localization.md`
- Modify: `knowledge/modules/options.md`
- Create: `knowledge/sessions/2026-07-05-admin-language-settings-implementation.md`

- [ ] **Step 1: Update knowledge base**

Add implementation status to `knowledge/modules/localization.md`:

```md
- Runtime language pack implementation is now available: `locale.manage`
  protects admin package upload/enable/disable APIs, packages are stored under
  `LOCALE_PACK_ROOT`, and frontend runtime messages load through
  `/api/v1/locale-messages/:locale`.
```

Add a note to `knowledge/modules/options.md`:

```md
- Locale runtime options now validate against built-in locales plus enabled
  runtime language pack locales. Language settings are managed from the admin
  language settings page rather than the generic site settings tab.
```

Create the session handoff:

```md
# 2026-07-05 Admin Language Settings Implementation Handoff

## Changed

- Added runtime ZIP language pack storage, validation, APIs, and admin UI.
- Added `locale.manage`.
- Added frontend runtime locale message loading.

## Decisions

- Runtime languages do not add route prefixes in this release.
- Backend message files remain reserved but inactive.

## Next

- Write a separate design before activating backend message pack files.

## Open Questions

- None.
```

- [ ] **Step 2: Run contract validation**

Run:

```bash
ruby scripts/validate-openapi-refs.rb
```

Expected: PASS.

- [ ] **Step 3: Run focused backend tests**

Run:

```bash
go test ./app/Models/Localization ./app/Http/Controllers/Localization ./app/Models/Options ./config ./database/migrations -count=1
```

Expected: PASS.

- [ ] **Step 4: Run frontend tests and typecheck**

Run:

```bash
cd apps/web && bun test tests/runtimeLocales.test.ts && bun run typecheck
```

Expected: PASS.

- [ ] **Step 5: Run full test script**

Run:

```bash
./scripts/test.sh
```

Expected: PASS. If it fails for an unrelated dirty-worktree issue, capture the failing command and error, then run the focused tests from Steps 2-4 again before reporting.

- [ ] **Step 6: Commit knowledge updates**

```bash
git add knowledge/index.md knowledge/modules/localization.md knowledge/modules/options.md knowledge/sessions/2026-07-05-admin-language-settings-implementation.md
git commit -m "docs: document language settings implementation"
```

---

## Implementation Notes

- Do not delete or overwrite existing dirty worktree changes. Stage only files touched by the current task.
- Use `apply_patch` for manual edits.
- Use Chinese comments for non-obvious project code comments.
- Keep uploaded runtime language files out of Git-tracked directories.
- Do not add route-prefix SEO support for runtime languages in this implementation.
- Keep backend message override inactive even if the uploaded ZIP contains `backend/<locale>.json`.

## Self-Review Checklist

- Spec storage boundary is covered by Tasks 1 and 2.
- Manifest, ZIP, override, conflict, and message validation are covered by Task 2.
- API, permission, and provider wiring are covered by Tasks 1, 3, 4, and 5.
- Admin menu/page behavior is covered by Task 7.
- Frontend runtime message loading and route/runtime language distinction are covered by Task 6.
- OpenAPI validation and tests are covered by Tasks 5 and 8.
- Operations and knowledge-base updates are covered by Tasks 1 and 8.
