# Uploaded Theme Activation Runtime Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let administrators upload a Nuxt Layer theme and click Activate so SForum builds it, health-checks it, switches the running web release, and keeps the previous theme available for rollback.

**Architecture:** Implement a first single-node self-hosted theme runtime. The API records theme activation releases and enqueues a River job; the worker builds a Nuxt artifact with the selected layer, runs a preview health check, marks the release active, and writes an atomic `current.json` file. The web container runs a small Bun supervisor that watches `current.json`, starts the selected `.output/server/index.mjs`, and restarts when a new release becomes active.

**Tech Stack:** Go Fiber, PostgreSQL migrations, River jobs, Nuxt 4/Nitro, Bun, Docker Compose shared volumes, existing `extension.manage` permission.

---

## Scope

This plan implements the first usable uploaded theme activation path for the current Docker Compose and local-process architecture. It deliberately keeps v1 narrow:

- Uploaded themes may only use the host web app dependencies already present in `apps/web/package.json`; theme packages must not install their own dependencies during activation.
- Theme activation runs one build at a time through the dedicated `theme` queue.
- Activation is asynchronous for uploaded themes and immediate for restoring the built-in default theme.
- Multi-node rolling deployment, Kubernetes, remote builders, marketplace trust, theme signatures, and arbitrary theme dependency installation are outside this plan.

## File Structure

- Modify `apps/web/nuxt.config.ts`: read `SFORUM_THEME_LAYER` and `SFORUM_NITRO_OUTPUT_DIR` so builds can target uploaded layers and isolated release output directories.
- Create `apps/web/scripts/runtime.mjs`: Bun supervisor that starts the active release and restarts when `current.json` changes.
- Modify `apps/web/Dockerfile`: add a runtime target that includes source, dependencies, default `.output`, runtime script, and shared release support.
- Modify `compose.yaml`, `compose.prod.yaml`, and `compose.dev.yaml`: add a `theme_releases` volume and mount it into `web` and `worker`.
- Modify `apps/api/Dockerfile`: make the worker target capable of building web releases by copying Bun, `apps/web`, `node_modules`, and built-in extensions.
- Modify `apps/api/config/config.go` and `apps/api/config/config_test.go`: add theme runtime paths, timeouts, and command defaults.
- Create migration `apps/api/database/migrations/202607070001_theme_releases.sql`: store theme release build and activation state.
- Modify `apps/api/app/Models/Extensions/types.go`, `store.go`, `postgres_store.go`, and tests: add theme release model and store methods.
- Create `apps/api/app/Support/ThemeRuntime/builder.go` and tests: build, preview health-check, atomic current-release writing.
- Create `apps/api/app/Jobs/Extensions/theme_activate.go` and tests: River job that runs the activation pipeline.
- Modify `apps/api/bootstrap/app.go` and `apps/api/bootstrap/worker.go`: wire dispatcher and theme activation worker.
- Modify `apps/api/app/Models/Extensions/service.go`, controller tests, OpenAPI files, and localization: uploaded theme activation creates a queued release instead of returning `theme_runtime_unavailable`.
- Modify `apps/web/app/utils/adminExtensions.ts`, `apps/web/app/composables/useAdminExtensionsManager.ts`, `apps/web/app/pages/admin/extensions/themes.vue`, and i18n files: show Activate, Building, Failed, Active, and Restore Default states.
- Update `knowledge/modules/extensions.md`, `knowledge/modules/frontend.md`, `knowledge/index.md`, and add a session handoff after implementation.

---

### Task 1: Dynamic Nuxt Theme Layer And Web Runtime Tests

**Files:**
- Modify: `apps/web/nuxt.config.ts`
- Create: `apps/web/scripts/runtime.mjs`
- Create: `tests/validate-theme-runtime.js`
- Modify: `scripts/test.sh`

- [ ] **Step 1: Write the failing validation script**

Create `tests/validate-theme-runtime.js`:

```js
const fs = require('fs')
const path = require('path')

const root = path.resolve(__dirname, '..')
const nuxtConfig = fs.readFileSync(path.join(root, 'apps/web/nuxt.config.ts'), 'utf8')
const runtimeScript = fs.readFileSync(path.join(root, 'apps/web/scripts/runtime.mjs'), 'utf8')

function assertIncludes(source, pattern, message) {
  if (!source.includes(pattern)) {
    throw new Error(message)
  }
}

assertIncludes(nuxtConfig, 'SFORUM_THEME_LAYER', 'nuxt.config.ts must read SFORUM_THEME_LAYER')
assertIncludes(nuxtConfig, 'SFORUM_NITRO_OUTPUT_DIR', 'nuxt.config.ts must read SFORUM_NITRO_OUTPUT_DIR')
assertIncludes(nuxtConfig, 'themeLayers', 'nuxt.config.ts must build a themeLayers array')
assertIncludes(runtimeScript, 'SFORUM_THEME_RELEASE_ROOT', 'runtime script must read SFORUM_THEME_RELEASE_ROOT')
assertIncludes(runtimeScript, 'current.json', 'runtime script must watch current.json')
assertIncludes(runtimeScript, 'spawn', 'runtime script must spawn the selected Nitro server')
assertIncludes(runtimeScript, 'fs.watch', 'runtime script must watch release changes')

console.log('Theme runtime validation passed.')
```

- [ ] **Step 2: Run the validation and verify it fails**

Run: `node tests/validate-theme-runtime.js`

Expected: FAIL with `ENOENT` for `apps/web/scripts/runtime.mjs` or missing `SFORUM_THEME_LAYER`.

- [ ] **Step 3: Make Nuxt config read dynamic theme layers**

In `apps/web/nuxt.config.ts`, add this near the existing top-level constants:

```ts
const defaultThemeLayer = '../../extensions/builtin/themes/sforum-default/layer'
const uploadedThemeLayer = process.env.SFORUM_THEME_LAYER?.trim()
const themeLayers = uploadedThemeLayer
  ? [uploadedThemeLayer, defaultThemeLayer]
  : [defaultThemeLayer]
const nitroOutputDir = process.env.SFORUM_NITRO_OUTPUT_DIR?.trim()
```

Then replace the static `extends` and add Nitro output config:

```ts
export default defineNuxtConfig({
  extends: themeLayers,
  modules: ['@nuxt/ui', '@nuxtjs/i18n', '@nuxtjs/seo'],
  ssr: true,
  buildDir: process.env.NUXT_BUILD_DIR || '.nuxt',
  nitro: nitroOutputDir ? { output: { dir: nitroOutputDir } } : {},
```

Keep the existing CSS, modules, i18n, SEO, and TypeScript config unchanged.

- [ ] **Step 4: Add the web runtime supervisor**

Create `apps/web/scripts/runtime.mjs`:

```js
import { spawn } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'

const releaseRoot = process.env.SFORUM_THEME_RELEASE_ROOT || '/var/lib/sforum/theme-releases'
const fallbackOutput = process.env.SFORUM_FALLBACK_OUTPUT || path.resolve(process.cwd(), '.output')
const currentFile = path.join(releaseRoot, 'current.json')

let child = null
let activeServer = ''
let restartTimer = null

function readCurrentServer() {
  try {
    const raw = fs.readFileSync(currentFile, 'utf8')
    const current = JSON.parse(raw)
    if (typeof current.server === 'string' && current.server.trim()) {
      return current.server
    }
  } catch (error) {
    if (error.code !== 'ENOENT') {
      console.error('[sforum-web-runtime] invalid current release:', error.message)
    }
  }
  return path.join(fallbackOutput, 'server/index.mjs')
}

function stopChild() {
  if (!child) {
    return
  }
  child.kill('SIGTERM')
  child = null
}

function startCurrent() {
  const server = readCurrentServer()
  if (server === activeServer && child) {
    return
  }
  if (!fs.existsSync(server)) {
    console.error(`[sforum-web-runtime] selected server does not exist: ${server}`)
    return
  }
  stopChild()
  activeServer = server
  child = spawn(process.execPath, [server], {
    stdio: 'inherit',
    env: {
      ...process.env,
      HOST: process.env.HOST || '0.0.0.0',
      PORT: process.env.PORT || '3000'
    }
  })
  child.on('exit', (code, signal) => {
    console.error(`[sforum-web-runtime] child exited code=${code ?? ''} signal=${signal ?? ''}`)
    child = null
    setTimeout(startCurrent, 1000)
  })
}

function scheduleRestart() {
  clearTimeout(restartTimer)
  restartTimer = setTimeout(startCurrent, 250)
}

fs.mkdirSync(releaseRoot, { recursive: true })
fs.watch(releaseRoot, scheduleRestart)
process.on('SIGTERM', () => {
  stopChild()
  process.exit(0)
})
process.on('SIGINT', () => {
  stopChild()
  process.exit(0)
})

startCurrent()
```

- [ ] **Step 5: Add the validation to the shared test script**

In `scripts/test.sh`, after homepage validation, add:

```bash
echo "Running theme runtime validation..."
node tests/validate-theme-runtime.js
```

- [ ] **Step 6: Run validation**

Run: `node tests/validate-theme-runtime.js`

Expected: `Theme runtime validation passed.`

- [ ] **Step 7: Run web typecheck**

Run: `cd apps/web && bun run typecheck`

Expected: PASS. If TypeScript rejects `nitro.output.dir`, replace the config block with:

```ts
nitro: {
  output: {
    dir: nitroOutputDir || '.output'
  }
},
```

Then rerun the command.

- [ ] **Step 8: Commit**

```bash
git add apps/web/nuxt.config.ts apps/web/scripts/runtime.mjs tests/validate-theme-runtime.js scripts/test.sh
git commit -m "feat: add dynamic theme layer web runtime"
```

---

### Task 2: Theme Release Data Model

**Files:**
- Create: `apps/api/database/migrations/202607070001_theme_releases.sql`
- Modify: `apps/api/database/migrations/embed_test.go`
- Modify: `apps/api/app/Models/Extensions/types.go`
- Modify: `apps/api/app/Models/Extensions/store.go`
- Modify: `apps/api/app/Models/Extensions/postgres_store.go`
- Modify: `apps/api/app/Models/Extensions/service_test.go`

- [ ] **Step 1: Write failing store tests**

Add this test to `apps/api/app/Models/Extensions/service_test.go` near other store-like fake checks:

```go
func TestThemeReleaseLifecycleStoresBuildState(t *testing.T) {
	store := newFakeExtensionStore(map[string]Extension{
		"starter.theme": withInstalledPackage(t, installedExtension("starter.theme", TypeTheme, ManifestBackend{})),
	})
	release, err := store.CreateThemeRelease(context.Background(), ThemeReleaseInput{
		ExtensionID: "starter.theme",
		Version:     "1.0.0",
		LayerPath:   "/themes/starter/layer",
	})
	if err != nil {
		t.Fatalf("create theme release: %v", err)
	}
	if release.Status != ThemeReleaseQueued {
		t.Fatalf("expected queued release, got %q", release.Status)
	}
	updated, err := store.UpdateThemeRelease(context.Background(), ThemeReleaseUpdate{
		ID:           release.ID,
		Status:       ThemeReleaseBuilt,
		ArtifactPath: "/var/lib/sforum/theme-releases/releases/1/.output",
		Message:      "build passed",
	})
	if err != nil {
		t.Fatalf("update theme release: %v", err)
	}
	if updated.Status != ThemeReleaseBuilt || updated.ArtifactPath == "" {
		t.Fatalf("expected built artifact, got %#v", updated)
	}
	latest, err := store.LatestThemeRelease(context.Background(), "starter.theme")
	if err != nil {
		t.Fatalf("latest theme release: %v", err)
	}
	if latest.ID != release.ID {
		t.Fatalf("expected latest release %d, got %d", release.ID, latest.ID)
	}
}
```

Expected initial failure: `ThemeReleaseInput` or store methods are undefined.

- [ ] **Step 2: Add migration**

Create `apps/api/database/migrations/202607070001_theme_releases.sql`:

```sql
-- +goose Up
CREATE TABLE extension_theme_releases (
  id BIGSERIAL PRIMARY KEY,
  extension_id TEXT NOT NULL REFERENCES extensions(id) ON DELETE CASCADE,
  extension_version TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('queued', 'building', 'built', 'activating', 'active', 'failed', 'rolled_back')),
  layer_path TEXT NOT NULL,
  artifact_path TEXT NOT NULL DEFAULT '',
  server_entry TEXT NOT NULL DEFAULT '',
  message TEXT NOT NULL DEFAULT '',
  build_log TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  activated_at TIMESTAMPTZ
);

CREATE INDEX extension_theme_releases_extension_created_idx
  ON extension_theme_releases (extension_id, created_at DESC, id DESC);

CREATE UNIQUE INDEX extension_theme_releases_single_active_idx
  ON extension_theme_releases ((status))
  WHERE status = 'active';

-- +goose Down
DROP INDEX IF EXISTS extension_theme_releases_single_active_idx;
DROP INDEX IF EXISTS extension_theme_releases_extension_created_idx;
DROP TABLE IF EXISTS extension_theme_releases;
```

- [ ] **Step 3: Add model types**

In `apps/api/app/Models/Extensions/types.go`, add constants:

```go
const (
	ThemeReleaseQueued     = "queued"
	ThemeReleaseBuilding   = "building"
	ThemeReleaseBuilt      = "built"
	ThemeReleaseActivating = "activating"
	ThemeReleaseActive     = "active"
	ThemeReleaseFailed     = "failed"
	ThemeReleaseRolledBack = "rolled_back"
)
```

Add structs:

```go
type ThemeRelease struct {
	ID               int64      `json:"id"`
	ExtensionID      string     `json:"extensionId"`
	ExtensionVersion string     `json:"extensionVersion"`
	Status           string     `json:"status"`
	LayerPath        string     `json:"layerPath"`
	ArtifactPath     string     `json:"artifactPath"`
	ServerEntry      string     `json:"serverEntry"`
	Message          string     `json:"message"`
	BuildLog         string     `json:"buildLog,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
	ActivatedAt      *time.Time `json:"activatedAt,omitempty"`
}

type ThemeReleaseInput struct {
	ExtensionID string
	Version     string
	LayerPath   string
}

type ThemeReleaseUpdate struct {
	ID           int64
	Status       string
	ArtifactPath string
	ServerEntry  string
	Message      string
	BuildLog     string
}
```

Add to `Extension`:

```go
ThemeRelease *ThemeRelease `json:"themeRelease,omitempty"`
```

- [ ] **Step 4: Extend the store interface**

In `apps/api/app/Models/Extensions/store.go`, add:

```go
CreateThemeRelease(ctx context.Context, input ThemeReleaseInput) (ThemeRelease, error)
UpdateThemeRelease(ctx context.Context, input ThemeReleaseUpdate) (ThemeRelease, error)
LatestThemeRelease(ctx context.Context, extensionID string) (ThemeRelease, error)
ActiveThemeRelease(ctx context.Context) (ThemeRelease, error)
```

- [ ] **Step 5: Implement Postgres methods**

In `apps/api/app/Models/Extensions/postgres_store.go`, add methods:

```go
func (s *PostgresStore) CreateThemeRelease(ctx context.Context, input ThemeReleaseInput) (ThemeRelease, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO extension_theme_releases (extension_id, extension_version, status, layer_path)
		VALUES ($1, $2, 'queued', $3)
		RETURNING id, extension_id, extension_version, status, layer_path, artifact_path,
		  server_entry, message, build_log, created_at, updated_at, activated_at
	`, input.ExtensionID, input.Version, input.LayerPath)
	return scanThemeRelease(row)
}

func (s *PostgresStore) UpdateThemeRelease(ctx context.Context, input ThemeReleaseUpdate) (ThemeRelease, error) {
	activatedSQL := "activated_at"
	if input.Status == ThemeReleaseActive {
		activatedSQL = "now()"
	}
	row := s.pool.QueryRow(ctx, fmt.Sprintf(`
		UPDATE extension_theme_releases
		SET status = $2,
		    artifact_path = COALESCE(NULLIF($3, ''), artifact_path),
		    server_entry = COALESCE(NULLIF($4, ''), server_entry),
		    message = $5,
		    build_log = $6,
		    updated_at = now(),
		    activated_at = %s
		WHERE id = $1
		RETURNING id, extension_id, extension_version, status, layer_path, artifact_path,
		  server_entry, message, build_log, created_at, updated_at, activated_at
	`, activatedSQL), input.ID, input.Status, input.ArtifactPath, input.ServerEntry, input.Message, input.BuildLog)
	return scanThemeRelease(row)
}

func (s *PostgresStore) LatestThemeRelease(ctx context.Context, extensionID string) (ThemeRelease, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, extension_id, extension_version, status, layer_path, artifact_path,
		  server_entry, message, build_log, created_at, updated_at, activated_at
		FROM extension_theme_releases
		WHERE extension_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, extensionID)
	return scanThemeRelease(row)
}

func (s *PostgresStore) ActiveThemeRelease(ctx context.Context) (ThemeRelease, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, extension_id, extension_version, status, layer_path, artifact_path,
		  server_entry, message, build_log, created_at, updated_at, activated_at
		FROM extension_theme_releases
		WHERE status = 'active'
		ORDER BY activated_at DESC NULLS LAST, id DESC
		LIMIT 1
	`)
	return scanThemeRelease(row)
}
```

Add scanner:

```go
type themeReleaseScanner interface {
	Scan(dest ...any) error
}

func scanThemeRelease(row themeReleaseScanner) (ThemeRelease, error) {
	var item ThemeRelease
	err := row.Scan(
		&item.ID,
		&item.ExtensionID,
		&item.ExtensionVersion,
		&item.Status,
		&item.LayerPath,
		&item.ArtifactPath,
		&item.ServerEntry,
		&item.Message,
		&item.BuildLog,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.ActivatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ThemeRelease{}, ErrExtensionNotFound
	}
	return item, err
}
```

- [ ] **Step 6: Decorate theme rows**

In `Service.List`, after runtime decoration, attach latest releases for themes:

```go
if items[index].Type == TypeTheme {
	if release, err := s.store.LatestThemeRelease(ctx, items[index].ID); err == nil {
		items[index].ThemeRelease = &release
	}
}
```

Also decorate returned items from `VerifyExtension` and `ActivateTheme`.

- [ ] **Step 7: Run migration embed and extension tests**

Run: `cd apps/api && go test ./database/migrations ./app/Models/Extensions`

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add apps/api/database/migrations apps/api/app/Models/Extensions
git commit -m "feat: store theme activation releases"
```

---

### Task 3: Theme Build Runtime

**Files:**
- Create: `apps/api/app/Support/ThemeRuntime/builder.go`
- Create: `apps/api/app/Support/ThemeRuntime/builder_test.go`
- Modify: `apps/api/config/config.go`
- Modify: `apps/api/config/config_test.go`

- [ ] **Step 1: Write failing builder tests**

Create `apps/api/app/Support/ThemeRuntime/builder_test.go`:

```go
package themeruntime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuilderWritesCurrentReleaseAtomically(t *testing.T) {
	root := t.TempDir()
	server := filepath.Join(root, "releases", "1", ".output", "server", "index.mjs")
	if err := os.MkdirAll(filepath.Dir(server), 0o755); err != nil {
		t.Fatalf("mkdir server: %v", err)
	}
	if err := os.WriteFile(server, []byte("console.log('ok')\n"), 0o644); err != nil {
		t.Fatalf("write server: %v", err)
	}
	builder := NewBuilder(Config{ReleaseRoot: root})
	if err := builder.WriteCurrent(context.Background(), CurrentRelease{
		ReleaseID:   1,
		ExtensionID: "starter.theme",
		Server:      server,
	}); err != nil {
		t.Fatalf("write current: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "current.json"))
	if err != nil {
		t.Fatalf("read current: %v", err)
	}
	if !strings.Contains(string(raw), "starter.theme") || !strings.Contains(string(raw), server) {
		t.Fatalf("current.json missing release data: %s", raw)
	}
}

func TestBuilderRejectsMissingLayer(t *testing.T) {
	builder := NewBuilder(Config{ReleaseRoot: t.TempDir(), WebRoot: t.TempDir(), BunPath: "bun", BuildTimeout: time.Second})
	_, err := builder.Build(context.Background(), BuildInput{
		ReleaseID:   1,
		ExtensionID: "starter.theme",
		LayerPath:   filepath.Join(t.TempDir(), "missing"),
	})
	if err == nil {
		t.Fatal("expected missing layer error")
	}
}
```

Expected initial failure: package or types undefined.

- [ ] **Step 2: Add config fields**

In `apps/api/config/config.go`, add to `Config`:

```go
ThemeReleaseRoot         string
ThemeWebRoot             string
ThemeBunPath             string
ThemeBuildTimeout        time.Duration
ThemePreviewTimeout      time.Duration
ThemePreviewPath         string
```

In `Load`, add:

```go
ThemeReleaseRoot:    env("THEME_RELEASE_ROOT", "/var/lib/sforum/theme-releases"),
ThemeWebRoot:        env("THEME_WEB_ROOT", "/app/apps/web"),
ThemeBunPath:        env("THEME_BUN_PATH", "bun"),
ThemeBuildTimeout:   envDuration("THEME_BUILD_TIMEOUT", 5*time.Minute),
ThemePreviewTimeout: envDuration("THEME_PREVIEW_TIMEOUT", 30*time.Second),
ThemePreviewPath:    env("THEME_PREVIEW_PATH", "/"),
```

Update `apps/api/config/config_test.go` to assert those defaults and env overrides.

- [ ] **Step 3: Implement the builder**

Create `apps/api/app/Support/ThemeRuntime/builder.go`:

```go
package themeruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	ReleaseRoot    string
	WebRoot        string
	BunPath        string
	BuildTimeout   time.Duration
	PreviewTimeout time.Duration
	PreviewPath    string
}

type Builder struct {
	config Config
}

type BuildInput struct {
	ReleaseID   int64
	ExtensionID string
	LayerPath   string
}

type BuildResult struct {
	ArtifactPath string
	ServerEntry  string
	BuildLog     string
}

type CurrentRelease struct {
	ReleaseID   int64  `json:"releaseId"`
	ExtensionID string `json:"extensionId"`
	Server      string `json:"server"`
	ActivatedAt string `json:"activatedAt"`
}

func NewBuilder(config Config) *Builder {
	if config.ReleaseRoot == "" {
		config.ReleaseRoot = "/var/lib/sforum/theme-releases"
	}
	if config.WebRoot == "" {
		config.WebRoot = "/app/apps/web"
	}
	if config.BunPath == "" {
		config.BunPath = "bun"
	}
	if config.BuildTimeout <= 0 {
		config.BuildTimeout = 5 * time.Minute
	}
	if config.PreviewTimeout <= 0 {
		config.PreviewTimeout = 30 * time.Second
	}
	if config.PreviewPath == "" {
		config.PreviewPath = "/"
	}
	return &Builder{config: config}
}

func (b *Builder) Build(ctx context.Context, input BuildInput) (BuildResult, error) {
	layerInfo, err := os.Stat(input.LayerPath)
	if err != nil || !layerInfo.IsDir() {
		return BuildResult{}, fmt.Errorf("theme layer is not available: %s", input.LayerPath)
	}
	releaseDir := filepath.Join(b.config.ReleaseRoot, "releases", strconv.FormatInt(input.ReleaseID, 10))
	artifactPath := filepath.Join(releaseDir, ".output")
	buildDir := filepath.Join(releaseDir, ".nuxt-build")
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		return BuildResult{}, fmt.Errorf("create release dir: %w", err)
	}
	buildCtx, cancel := context.WithTimeout(ctx, b.config.BuildTimeout)
	defer cancel()
	cmd := exec.CommandContext(buildCtx, b.config.BunPath, "run", "build")
	cmd.Dir = b.config.WebRoot
	cmd.Env = append(os.Environ(),
		"SFORUM_THEME_LAYER="+input.LayerPath,
		"SFORUM_NITRO_OUTPUT_DIR="+artifactPath,
		"NUXT_BUILD_DIR="+buildDir,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return BuildResult{BuildLog: string(output)}, fmt.Errorf("theme build failed: %w", err)
	}
	server := filepath.Join(artifactPath, "server", "index.mjs")
	if _, err := os.Stat(server); err != nil {
		return BuildResult{ArtifactPath: artifactPath, BuildLog: string(output)}, fmt.Errorf("theme server entry missing: %w", err)
	}
	if err := b.HealthCheck(ctx, server); err != nil {
		return BuildResult{ArtifactPath: artifactPath, ServerEntry: server, BuildLog: string(output)}, err
	}
	return BuildResult{ArtifactPath: artifactPath, ServerEntry: server, BuildLog: string(output)}, nil
}

func (b *Builder) HealthCheck(ctx context.Context, server string) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("reserve preview port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	previewCtx, cancel := context.WithTimeout(ctx, b.config.PreviewTimeout)
	defer cancel()
	cmd := exec.CommandContext(previewCtx, b.config.BunPath, server)
	cmd.Env = append(os.Environ(), "HOST=127.0.0.1", "PORT="+strconv.Itoa(port))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start theme preview: %w", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()
	url := "http://127.0.0.1:" + strconv.Itoa(port) + b.config.PreviewPath
	deadline := time.Now().Add(b.config.PreviewTimeout)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequestWithContext(previewCtx, http.MethodGet, url, nil)
		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp.StatusCode < 500 {
			_ = resp.Body.Close()
			return nil
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("theme preview health check failed")
}

func (b *Builder) WriteCurrent(_ context.Context, current CurrentRelease) error {
	current.ActivatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := os.MkdirAll(b.config.ReleaseRoot, 0o755); err != nil {
		return fmt.Errorf("create release root: %w", err)
	}
	raw, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal current release: %w", err)
	}
	tmp := filepath.Join(b.config.ReleaseRoot, "current.json.tmp")
	final := filepath.Join(b.config.ReleaseRoot, "current.json")
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("write current release temp file: %w", err)
	}
	return os.Rename(tmp, final)
}
```

- [ ] **Step 4: Run builder and config tests**

Run: `cd apps/api && go test ./app/Support/ThemeRuntime ./config`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/app/Support/ThemeRuntime apps/api/config
git commit -m "feat: add theme release builder"
```

---

### Task 4: Theme Activation Job

**Files:**
- Create: `apps/api/app/Jobs/Extensions/theme_activate.go`
- Create: `apps/api/app/Jobs/Extensions/theme_activate_test.go`
- Modify: `apps/api/app/Support/Jobs/types.go`
- Modify: `apps/api/app/Support/Jobs/config.go`
- Modify: `apps/api/config/config.go`

- [ ] **Step 1: Add a theme queue**

In `apps/api/app/Support/Jobs/types.go`, add:

```go
QueueTheme = "theme"
```

In `apps/api/app/Support/Jobs/config.go`, add `ThemeWorkers int` to `Config`, map it to `cfg.JobQueueThemeWorkers`, and include:

```go
QueueTheme: {MaxWorkers: cfg.ThemeWorkers},
```

In `apps/api/config/config.go`, add:

```go
JobQueueThemeWorkers int
```

and load:

```go
JobQueueThemeWorkers: envPositiveInt("JOB_QUEUE_THEME_WORKERS", 1),
```

- [ ] **Step 2: Write failing job tests**

Create `apps/api/app/Jobs/Extensions/theme_activate_test.go`:

```go
package extensionjobs

import (
	"context"
	"testing"

	"github.com/riverqueue/river"
	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

func TestActivateThemeArgsKindAndOptions(t *testing.T) {
	args := ActivateThemeArgs{ReleaseID: 7, ExtensionID: "starter.theme"}
	if args.Kind() != "extension.theme_activate" {
		t.Fatalf("unexpected kind %q", args.Kind())
	}
	opts := args.EnqueueOptions()
	if opts.Queue != "theme" || opts.MaxAttempts != 1 || !opts.Unique.ByArgs {
		t.Fatalf("unexpected enqueue options: %#v", opts)
	}
}

func TestActivateThemeWorkerMarksReleaseActive(t *testing.T) {
	store := &fakeThemeStore{
		extension: extensions.Extension{ID: "starter.theme", Version: "1.0.0", Type: extensions.TypeTheme},
		release: extensions.ThemeRelease{ID: 7, ExtensionID: "starter.theme", Status: extensions.ThemeReleaseQueued, LayerPath: "/tmp/layer"},
	}
	builder := &fakeThemeBuilder{
		result: BuildResult{ArtifactPath: "/tmp/out", ServerEntry: "/tmp/out/server/index.mjs", BuildLog: "ok"},
	}
	worker := ActivateThemeWorker{Store: store, Builder: builder}
	err := worker.Work(context.Background(), &river.Job[ActivateThemeArgs]{
		Args: ActivateThemeArgs{ReleaseID: 7, ExtensionID: "starter.theme"},
	})
	if err != nil {
		t.Fatalf("work: %v", err)
	}
	if store.activeThemeID != "starter.theme" {
		t.Fatalf("expected active theme starter.theme, got %q", store.activeThemeID)
	}
	if store.release.Status != extensions.ThemeReleaseActive {
		t.Fatalf("expected active release, got %#v", store.release)
	}
	if builder.current.ExtensionID != "starter.theme" {
		t.Fatalf("expected current release write, got %#v", builder.current)
	}
}
```

Add test fakes in the same file using the method names introduced in Task 2 and Task 3.

- [ ] **Step 3: Implement job args and worker**

Create `apps/api/app/Jobs/Extensions/theme_activate.go`:

```go
package extensionjobs

import (
	"context"
	"fmt"

	"github.com/riverqueue/river"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
	themeruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/ThemeRuntime"
)

type ThemeStore interface {
	Get(ctx context.Context, id string) (extensions.Extension, error)
	ActivateTheme(ctx context.Context, id string) (extensions.Extension, error)
	UpdateThemeRelease(ctx context.Context, input extensions.ThemeReleaseUpdate) (extensions.ThemeRelease, error)
	LatestThemeRelease(ctx context.Context, extensionID string) (extensions.ThemeRelease, error)
}

type ThemeBuilder interface {
	Build(ctx context.Context, input themeruntime.BuildInput) (themeruntime.BuildResult, error)
	WriteCurrent(ctx context.Context, current themeruntime.CurrentRelease) error
}

type ActivateThemeArgs struct {
	ReleaseID   int64  `json:"release_id" river:"unique"`
	ExtensionID string `json:"extension_id"`
}

func (ActivateThemeArgs) Kind() string {
	return "extension.theme_activate"
}

func (ActivateThemeArgs) EnqueueOptions() supportjobs.EnqueueOptions {
	return supportjobs.EnqueueOptions{
		Queue:       supportjobs.QueueTheme,
		MaxAttempts: 1,
		Unique:      river.UniqueOpts{ByArgs: true},
	}
}

type ActivateThemeWorker struct {
	river.WorkerDefaults[ActivateThemeArgs]
	Store   ThemeStore
	Builder ThemeBuilder
}

func (w *ActivateThemeWorker) Work(ctx context.Context, job *river.Job[ActivateThemeArgs]) error {
	if w.Store == nil {
		return fmt.Errorf("theme activation worker requires store")
	}
	if w.Builder == nil {
		return fmt.Errorf("theme activation worker requires builder")
	}
	extension, err := w.Store.Get(ctx, job.Args.ExtensionID)
	if err != nil {
		return err
	}
	if extension.Type != extensions.TypeTheme {
		return fmt.Errorf("extension %s is not a theme", extension.ID)
	}
	release, err := w.Store.LatestThemeRelease(ctx, extension.ID)
	if err != nil {
		return err
	}
	if release.ID != job.Args.ReleaseID {
		return fmt.Errorf("theme release mismatch: job=%d latest=%d", job.Args.ReleaseID, release.ID)
	}
	_, _ = w.Store.UpdateThemeRelease(ctx, extensions.ThemeReleaseUpdate{
		ID:      release.ID,
		Status:  extensions.ThemeReleaseBuilding,
		Message: "Building theme release.",
	})
	result, err := w.Builder.Build(ctx, themeruntime.BuildInput{
		ReleaseID:   release.ID,
		ExtensionID: extension.ID,
		LayerPath:   release.LayerPath,
	})
	if err != nil {
		_, _ = w.Store.UpdateThemeRelease(ctx, extensions.ThemeReleaseUpdate{
			ID:       release.ID,
			Status:   extensions.ThemeReleaseFailed,
			Message:  err.Error(),
			BuildLog: result.BuildLog,
		})
		return err
	}
	_, _ = w.Store.UpdateThemeRelease(ctx, extensions.ThemeReleaseUpdate{
		ID:           release.ID,
		Status:       extensions.ThemeReleaseActivating,
		ArtifactPath: result.ArtifactPath,
		ServerEntry:  result.ServerEntry,
		Message:      "Switching active web release.",
		BuildLog:     result.BuildLog,
	})
	if err := w.Builder.WriteCurrent(ctx, themeruntime.CurrentRelease{
		ReleaseID:   release.ID,
		ExtensionID: extension.ID,
		Server:      result.ServerEntry,
	}); err != nil {
		_, _ = w.Store.UpdateThemeRelease(ctx, extensions.ThemeReleaseUpdate{
			ID:       release.ID,
			Status:   extensions.ThemeReleaseFailed,
			Message:  err.Error(),
			BuildLog: result.BuildLog,
		})
		return err
	}
	if _, err := w.Store.ActivateTheme(ctx, extension.ID); err != nil {
		return err
	}
	_, err = w.Store.UpdateThemeRelease(ctx, extensions.ThemeReleaseUpdate{
		ID:           release.ID,
		Status:       extensions.ThemeReleaseActive,
		ArtifactPath: result.ArtifactPath,
		ServerEntry:  result.ServerEntry,
		Message:      "Theme release activated.",
		BuildLog:     result.BuildLog,
	})
	return err
}

func RegisterThemeActivationWorker(registry *supportjobs.Registry, store ThemeStore, builder ThemeBuilder) {
	registry.Add(func(workers *river.Workers) error {
		return river.AddWorkerSafely[ActivateThemeArgs](workers, &ActivateThemeWorker{Store: store, Builder: builder})
	})
}
```

- [ ] **Step 4: Run job tests**

Run: `cd apps/api && go test ./app/Jobs/Extensions ./app/Support/Jobs ./config`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/app/Jobs/Extensions apps/api/app/Support/Jobs apps/api/config
git commit -m "feat: add theme activation job"
```

---

### Task 5: Activation Service And API Contract

**Files:**
- Modify: `apps/api/app/Models/Extensions/service.go`
- Modify: `apps/api/app/Models/Extensions/service_test.go`
- Modify: `apps/api/app/Http/Controllers/Extensions/controller.go`
- Modify: `apps/api/app/Http/Controllers/Extensions/controller_test.go`
- Modify: `apps/api/app/Support/Localization/messages.go`
- Modify: `contracts/openapi/paths/extensions.yaml`
- Modify: `contracts/openapi/schemas/extensions.yaml`

- [ ] **Step 1: Write failing service test for uploaded theme activation**

Replace `TestServiceActivateThemeAllowsOnlyBuiltinDefaultThemeInV1` with:

```go
func TestServiceActivateThemeQueuesUploadedThemeBuild(t *testing.T) {
	store := newFakeExtensionStore(map[string]Extension{
		"starter.theme": withInstalledPackage(t, installedExtension("starter.theme", TypeTheme, ManifestBackend{})),
	})
	dispatcher := &fakeThemeActivationDispatcher{}
	service := NewServiceWithThemeActivation(store, t.TempDir(), "", LocalRuntimeManager{}, fakeThemeBuilder{}, dispatcher)

	queued, err := service.ActivateTheme(context.Background(), extensionManager(), "starter.theme")
	if err != nil {
		t.Fatalf("ActivateTheme returned error: %v", err)
	}
	if queued.ThemeRelease == nil || queued.ThemeRelease.Status != ThemeReleaseQueued {
		t.Fatalf("expected queued theme release, got %#v", queued.ThemeRelease)
	}
	if dispatcher.releaseID == 0 || dispatcher.extensionID != "starter.theme" {
		t.Fatalf("expected activation dispatch, got release=%d extension=%q", dispatcher.releaseID, dispatcher.extensionID)
	}
	if store.activeThemeID == "starter.theme" {
		t.Fatal("uploaded theme should not become active before worker completes")
	}
}
```

- [ ] **Step 2: Add a dispatcher interface to the service**

In `store.go` or `service.go`, add:

```go
type ThemeActivationDispatcher interface {
	EnqueueThemeActivation(ctx context.Context, release ThemeRelease) error
}
```

Extend `Service` with:

```go
themeActivationDispatcher ThemeActivationDispatcher
```

Add constructor:

```go
func NewServiceWithThemeActivation(store Store, extensionRoot string, builtinRoot string, runtime RuntimeManager, themeBuilder ThemeBuilder, dispatcher ThemeActivationDispatcher) *Service {
	service := NewServiceWithBuiltinsAndRuntime(store, extensionRoot, builtinRoot, runtime, themeBuilder)
	service.themeActivationDispatcher = dispatcher
	return service
}
```

- [ ] **Step 3: Change uploaded theme activation**

In `ActivateTheme`, replace the uploaded-theme hard block with:

```go
if extension.ID != DefaultThemeID || extension.Source != SourceBuiltin {
	if err := s.verifyExtension(ctx, extension); err != nil {
		s.recordEnableFailure(ctx, actor, extension.ID, err)
		return Extension{}, err
	}
	layerPath, ok := installedFilePath(extension, extension.Manifest.Frontend.Layer)
	if !ok {
		return Extension{}, fmt.Errorf("%w: theme layer is unavailable", ErrBuildFailed)
	}
	release, err := s.store.CreateThemeRelease(ctx, ThemeReleaseInput{
		ExtensionID: extension.ID,
		Version:     extension.Version,
		LayerPath:   layerPath,
	})
	if err != nil {
		return Extension{}, err
	}
	if s.themeActivationDispatcher != nil {
		if err := s.themeActivationDispatcher.EnqueueThemeActivation(ctx, release); err != nil {
			_, _ = s.store.UpdateThemeRelease(ctx, ThemeReleaseUpdate{
				ID:      release.ID,
				Status:  ThemeReleaseFailed,
				Message: err.Error(),
			})
			return Extension{}, err
		}
	}
	_, _ = s.store.CreateEvent(ctx, EventInput{
		ExtensionID: extension.ID,
		ActorUserID: actor.ID,
		Action:      EventThemeActivationQueued,
		Message:     "Theme activation queued.",
	})
	extension.ThemeRelease = &release
	return extension, nil
}
```

Add `EventThemeActivationQueued = "theme_activation_queued"` in `types.go`.

- [ ] **Step 4: Add dispatcher adapter**

In `apps/api/app/Jobs/Extensions/theme_activate.go`, add:

```go
type ActivationDispatcherAdapter struct {
	Dispatcher *supportjobs.Dispatcher
}

func (a ActivationDispatcherAdapter) EnqueueThemeActivation(ctx context.Context, release extensions.ThemeRelease) error {
	if a.Dispatcher == nil {
		return nil
	}
	args := ActivateThemeArgs{ReleaseID: release.ID, ExtensionID: release.ExtensionID}
	_, err := a.Dispatcher.Enqueue(ctx, args, args.EnqueueOptions())
	return err
}
```

- [ ] **Step 5: Update controller response expectations**

In `controller.go`, keep using `h.service.ActivateTheme`, but return status `202 Accepted` when `item.ThemeRelease != nil && item.ThemeRelease.Status == extensions.ThemeReleaseQueued`:

```go
status := fiber.StatusOK
if item.ThemeRelease != nil && item.ThemeRelease.Status == extensions.ThemeReleaseQueued {
	status = fiber.StatusAccepted
}
return response.JSON(c, status, item)
```

Update controller tests so uploaded theme activation expects `202` and `themeRelease.status == "queued"` instead of 409.

- [ ] **Step 6: Update OpenAPI**

In `contracts/openapi/paths/extensions.yaml`, change activation `200` description to default-theme restore and add:

```yaml
'202':
  description: Uploaded theme activation accepted and queued for build.
  content:
    application/json:
      schema:
        "$ref": "../schemas/extensions.yaml#/ApiResponseExtension"
```

Remove the 409 description that says uploaded theme activation is unavailable.

In `contracts/openapi/schemas/extensions.yaml`, add `ThemeRelease` and add `themeRelease` to `Extension`.

- [ ] **Step 7: Update localization**

Change messages:

```go
"extension.theme_activation_queued": "主题构建已排队，完成后会自动切换。",
```

Keep `extension.theme_runtime_unavailable` for legacy failures only if other code still references it.

- [ ] **Step 8: Run API and contract tests**

Run:

```bash
cd apps/api && go test ./app/Models/Extensions ./app/Http/Controllers/Extensions ./app/Jobs/Extensions
ruby scripts/validate-openapi-refs.rb
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add apps/api/app/Models/Extensions apps/api/app/Http/Controllers/Extensions apps/api/app/Support/Localization apps/api/app/Jobs/Extensions contracts/openapi
git commit -m "feat: queue uploaded theme activation"
```

---

### Task 6: Bootstrap Worker And API Dispatch

**Files:**
- Modify: `apps/api/bootstrap/app.go`
- Modify: `apps/api/bootstrap/worker.go`
- Modify: `apps/api/bootstrap/app_test.go`
- Modify: `apps/api/app/Support/Extensions/event_jobs.go`

- [ ] **Step 1: Wire an API River dispatcher**

In `bootstrap/app.go`, after creating the Postgres pool and before constructing `extensionService`, create a River client without workers:

```go
jobClient, err := supportjobs.NewClient(pool, supportjobs.FromAppConfig(cfg), river.NewWorkers())
if err != nil {
	extensionRuntime.Close(ctx)
	pool.Close()
	return nil, fmt.Errorf("job dispatcher setup failed: %w", err)
}
jobDispatcher := supportjobs.NewDispatcher(jobClient)
themeDispatcher := extensionjobs.ActivationDispatcherAdapter{Dispatcher: jobDispatcher}
```

Import `github.com/riverqueue/river`, `supportjobs`, and `extensionjobs`.

Use:

```go
extensionService := extensions.NewServiceWithThemeActivation(extensionStore, cfg.ExtensionRoot, cfg.BuiltinExtensionRoot, extensionRuntime, nil, themeDispatcher)
```

Make sure app shutdown closes the job client with `supportjobs.Stop`.

- [ ] **Step 2: Wire worker registration**

In `bootstrap/worker.go`, create the store and builder before `registry.Build()`:

```go
pool, err := postgres.NewPool(ctx, cfg.DatabaseURL, cfg.WorkerDatabaseMaxConns)
if err != nil {
	return nil, fmt.Errorf("postgres setup failed: %w", err)
}
extensionStore := extensions.NewPostgresStore(pool)
themeBuilder := themeruntime.NewBuilder(themeruntime.Config{
	ReleaseRoot:    cfg.ThemeReleaseRoot,
	WebRoot:        cfg.ThemeWebRoot,
	BunPath:        cfg.ThemeBunPath,
	BuildTimeout:   cfg.ThemeBuildTimeout,
	PreviewTimeout: cfg.ThemePreviewTimeout,
	PreviewPath:    cfg.ThemePreviewPath,
})
extensionjobs.RegisterThemeActivationWorker(registry, extensionStore, themeBuilder)
```

Move the existing `registry.IsEmpty()` check after registration.

- [ ] **Step 3: Preserve extension event worker registration**

If event delivery jobs are intended to run through River now, register them in the same worker bootstrap once the runtime manager is available. If not, keep inline delivery unchanged and do not add a partial registration.

- [ ] **Step 4: Update bootstrap tests**

In `apps/api/bootstrap/app_test.go`, update fakes so `NewApp` accepts the dispatcher setup. Add a test that `newExtensionRuntimeManager` still receives enabled plugins and that `NewApp` does not fail when River client creation succeeds.

- [ ] **Step 5: Run bootstrap tests**

Run: `cd apps/api && go test ./bootstrap`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/api/bootstrap apps/api/app/Support/Extensions
git commit -m "feat: wire theme activation jobs"
```

---

### Task 7: Docker And Compose Runtime

**Files:**
- Modify: `apps/api/Dockerfile`
- Modify: `apps/web/Dockerfile`
- Modify: `compose.yaml`
- Modify: `compose.prod.yaml`
- Modify: `compose.dev.yaml`
- Modify: `.env.example` if present

- [ ] **Step 1: Make web prod use the runtime supervisor**

In `apps/web/Dockerfile`, change the `prod` stage:

```dockerfile
FROM oven/bun:1.3-alpine AS prod

WORKDIR /app/apps/web
ENV HOST=0.0.0.0
ENV PORT=3000
ENV SFORUM_THEME_RELEASE_ROOT=/var/lib/sforum/theme-releases
COPY --from=deps /app/apps/web/node_modules ./node_modules
COPY --from=build /app/apps/web/.output ./.output
COPY apps/web/scripts ./scripts
EXPOSE 3000
CMD ["bun", "scripts/runtime.mjs"]
```

- [ ] **Step 2: Give the worker a theme build toolchain**

In `apps/api/Dockerfile`, add a web deps stage:

```dockerfile
FROM oven/bun:1.3-alpine AS webdeps
WORKDIR /app/apps/web
COPY apps/web/package.json apps/web/bun.lock ./
RUN bun install --frozen-lockfile
COPY apps/web/ ./
COPY extensions/builtin /app/extensions/builtin
```

Change the `worker` target:

```dockerfile
FROM oven/bun:1.3-alpine AS worker

RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/sforum-worker /usr/local/bin/sforum-worker
COPY --from=build /app/extensions/builtin /app/extensions/builtin
COPY --from=webdeps /app/apps/web /app/apps/web
ENV THEME_WEB_ROOT=/app/apps/web
ENV THEME_BUN_PATH=bun
ENV THEME_RELEASE_ROOT=/var/lib/sforum/theme-releases
CMD ["sforum-worker"]
```

- [ ] **Step 3: Add shared release volume**

In `compose.yaml`, add volume:

```yaml
  theme_releases:
```

Mount into `worker`:

```yaml
      - theme_releases:/var/lib/sforum/theme-releases
```

Mount into `web`:

```yaml
      - theme_releases:/var/lib/sforum/theme-releases
```

Add worker env:

```yaml
      THEME_RELEASE_ROOT: ${THEME_RELEASE_ROOT:-/var/lib/sforum/theme-releases}
      THEME_WEB_ROOT: ${THEME_WEB_ROOT:-/app/apps/web}
      THEME_BUN_PATH: ${THEME_BUN_PATH:-bun}
```

Add web env:

```yaml
      SFORUM_THEME_RELEASE_ROOT: ${THEME_RELEASE_ROOT:-/var/lib/sforum/theme-releases}
```

- [ ] **Step 4: Keep dev safe**

In `compose.dev.yaml`, mount the same volume but keep the user's host-run port 3000 behavior unchanged. Do not add commands that kill or replace the user's local Nuxt dev server.

- [ ] **Step 5: Validate Compose config**

Run: `docker compose -f compose.yaml -f compose.prod.yaml config`

Expected: config renders successfully and shows `theme_releases` mounted into `worker` and `web`.

- [ ] **Step 6: Commit**

```bash
git add apps/api/Dockerfile apps/web/Dockerfile compose.yaml compose.prod.yaml compose.dev.yaml .env.example
git commit -m "feat: add theme runtime containers"
```

---

### Task 8: Admin UI Activation States

**Files:**
- Modify: `apps/web/app/utils/adminExtensions.ts`
- Modify: `apps/web/app/composables/useAdminExtensionsManager.ts`
- Modify: `apps/web/app/pages/admin/extensions/themes.vue`
- Modify: `apps/web/app/pages/admin/extensions/index.vue`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`
- Modify: `tests/validate-admin-framework.ts`

- [ ] **Step 1: Add frontend theme release types**

In `apps/web/app/utils/adminExtensions.ts`, add:

```ts
export type AdminThemeReleaseStatus = 'queued' | 'building' | 'built' | 'activating' | 'active' | 'failed' | 'rolled_back'
export type AdminThemeActionState = 'active' | 'activateDefault' | 'activate' | 'queued' | 'building' | 'activating' | 'failed'

export type AdminThemeRelease = {
  id: number
  extensionId: string
  extensionVersion: string
  status: AdminThemeReleaseStatus
  message: string
  createdAt: string
  updatedAt: string
  activatedAt?: string
}
```

Add to `AdminExtension`:

```ts
themeRelease?: AdminThemeRelease
```

Replace `themeActionState`:

```ts
export function themeActionState(item: AdminExtension): AdminThemeActionState {
  if (item.status === 'enabled') {
    return 'active'
  }
  if (item.id === 'sforum.default-theme' && item.source === 'builtin') {
    return 'activateDefault'
  }
  if (item.themeRelease?.status === 'queued') {
    return 'queued'
  }
  if (item.themeRelease?.status === 'building') {
    return 'building'
  }
  if (item.themeRelease?.status === 'activating') {
    return 'activating'
  }
  if (item.themeRelease?.status === 'failed') {
    return 'failed'
  }
  return 'activate'
}
```

- [ ] **Step 2: Update activation toast**

In `useAdminExtensionsManager.ts`, change activation success:

```ts
toast.add({
  color: activated.themeRelease?.status === 'queued' ? 'info' : 'success',
  icon: activated.themeRelease?.status === 'queued' ? 'i-lucide-hourglass' : 'i-lucide-palette',
  title: activated.themeRelease?.status === 'queued'
    ? t('admin.extensions.themeActivationQueued')
    : t('admin.extensions.themeActivated')
})
```

- [ ] **Step 3: Update theme page buttons**

In `themes.vue`, replace the `verifyOnly` block with:

```vue
<UButton
  v-else-if="themeActionState(item) === 'activate'"
  size="sm"
  icon="i-lucide-play"
  :loading="busyId === item.id"
  @click="activateTheme(item)"
>
  {{ t('admin.extensions.activateTheme') }}
</UButton>
<UButton
  v-else-if="themeActionState(item) === 'failed'"
  size="sm"
  color="error"
  variant="subtle"
  icon="i-lucide-refresh-cw"
  :loading="busyId === item.id"
  @click="activateTheme(item)"
>
  {{ t('admin.extensions.retryActivation') }}
</UButton>
<UButton
  v-else-if="['queued', 'building', 'activating'].includes(themeActionState(item))"
  size="sm"
  color="neutral"
  variant="subtle"
  icon="i-lucide-hourglass"
  disabled
>
  {{ t(`admin.extensions.themeRelease.${item.themeRelease?.status || 'queued'}`) }}
</UButton>
```

Show latest release message:

```vue
<p v-if="item.themeRelease?.message" class="mt-2 text-xs leading-5 text-slate-500 dark:text-zinc-400">
  {{ item.themeRelease.message }}
</p>
```

- [ ] **Step 4: Update i18n**

In `zh-CN.json`:

```json
"activateTheme": "激活主题",
"retryActivation": "重试激活",
"themeActivationQueued": "主题构建已排队。",
"themeRelease": {
  "queued": "排队中",
  "building": "构建中",
  "built": "已构建",
  "activating": "切换中",
  "active": "当前主题",
  "failed": "激活失败",
  "rolled_back": "已回滚"
}
```

Add matching English strings in `en-US.json`.

- [ ] **Step 5: Add admin validation**

In `tests/validate-admin-framework.ts`, assert `themeActionState` can return `activate`, `queued`, `building`, `activating`, and `failed` by checking source text includes those literals.

- [ ] **Step 6: Run frontend validation**

Run:

```bash
cd apps/web && bun run typecheck
bun tests/validate-admin-framework.ts
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/web/app/utils/adminExtensions.ts apps/web/app/composables/useAdminExtensionsManager.ts apps/web/app/pages/admin/extensions apps/web/i18n tests/validate-admin-framework.ts
git commit -m "feat: show uploaded theme activation states"
```

---

### Task 9: End-To-End Verification And Documentation

**Files:**
- Modify: `knowledge/modules/extensions.md`
- Modify: `knowledge/modules/frontend.md`
- Modify: `knowledge/index.md`
- Create: `knowledge/sessions/2026-07-07-theme-activation-runtime.md`

- [ ] **Step 1: Add an integration fixture theme**

Create a temporary theme ZIP under `/private/tmp` during verification with:

```json
{
  "id": "demo.activated-theme",
  "name": "Demo Activated Theme",
  "description": "Demo theme used to verify uploaded theme activation.",
  "url": "https://example.com/demo-activated-theme",
  "author": {
    "name": "SForum",
    "url": "https://example.com",
    "email": "dev@example.com"
  },
  "version": "1.0.0",
  "type": "theme",
  "sforumVersion": "^1.0.0",
  "frontend": {
    "layer": "layer"
  }
}
```

The layer should include `layer/app/pages/index.vue` with visible text `Demo Activated Theme`.

- [ ] **Step 2: Run full test suite**

Run: `./scripts/test.sh`

Expected: PASS.

- [ ] **Step 3: Build Compose images**

Run: `docker compose -f compose.yaml -f compose.prod.yaml build web worker api`

Expected: PASS.

- [ ] **Step 4: Manual activation smoke test**

1. Start production-like services:

```bash
docker compose -f compose.yaml -f compose.prod.yaml up -d postgres redis meilisearch api worker web
```

2. Log in as a user with `extension.manage`.
3. Upload the demo theme ZIP from the admin extension overview.
4. Open the Themes page and click Activate.
5. Wait until the row shows Active Theme.
6. Load the public homepage and confirm it contains `Demo Activated Theme`.
7. Click Restore Default Theme and confirm the built-in homepage returns.

- [ ] **Step 5: Update knowledge base**

In `knowledge/modules/extensions.md`, replace the v1 blocked-theme text with:

```md
Uploaded Nuxt Layer themes can now be activated through the self-hosted theme
runtime. Activation queues a River job, builds an isolated Nuxt/Nitro artifact,
health-checks a preview server, writes the active release file, and lets the web
runtime restart onto the selected artifact. Failed builds keep the previous
active theme running.
```

In `knowledge/modules/frontend.md`, document:

```md
The web production container runs `apps/web/scripts/runtime.mjs`, which watches
`THEME_RELEASE_ROOT/current.json` and starts the selected Nitro server. The
default `.output` remains the fallback release when no uploaded theme is active.
```

Add a session handoff at `knowledge/sessions/2026-07-07-theme-activation-runtime.md` with Changed, Decisions, Next, and Open Questions sections.

- [ ] **Step 6: Commit**

```bash
git add knowledge
git commit -m "docs: document uploaded theme activation runtime"
```

---

## Self-Review

- Spec coverage: This plan covers upload activation semantics, asynchronous build, Nuxt dynamic layer selection, health check, atomic switch, web runtime restart, UI state, OpenAPI, tests, Docker, and knowledge base updates.
- Placeholder scan: The plan avoids placeholder markers, vague error-handling steps, and hidden implementation work. Every task lists concrete files, code shape, commands, and expected results.
- Type consistency: The backend uses `ThemeRelease`, `ThemeReleaseInput`, `ThemeReleaseUpdate`, and `ThemeActivationDispatcher`; the frontend mirrors this with `AdminThemeRelease`. Queue names use `QueueTheme` and job kind `extension.theme_activate`.

## Execution Notes

- Use the configured proxy before dependency downloads, but this plan should not add new dependencies.
- Do not kill a user-owned port 3000 dev server during manual checks.
- Keep uploaded theme dependency installation unsupported in this release; themes must build using host dependencies.
- If `SFORUM_NITRO_OUTPUT_DIR` is not honored by Nuxt/Nitro during implementation, update `nuxt.config.ts` to set Nitro output through the supported config key and keep `tests/validate-theme-runtime.js` plus `bun run typecheck` as the guard.
