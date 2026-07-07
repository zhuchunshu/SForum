# Theme Runtime Convergence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 后台激活或恢复主题后，生产前台同一个入口自动呈现当前主题；本地 `cd apps/web && bun run dev` 也跟随后台主题切换自动重启生效。

**Architecture:** `theme-releases/current.json` 是唯一运行时主题选择信号。生产 `web` 由 `apps/web/scripts/runtime.mjs` 托管并切换已构建 Nitro release；本地 `bun run dev` 变为主题感知 supervisor，读取同一个 `current.json.layerPath` 后重启 Nuxt dev。

**Tech Stack:** Go theme activation worker, Go extension service, Nuxt 4, Node/Bun scripts, Docker Compose.

---

## Context And Constraints

- 用户目标是“部署后后台切换主题，刷新同一个前台 URL 就能看到新主题”；不要求强制刷新所有已打开访客页面。
- 不再引导用户运行 `bun run preview` 或额外访问单独预览端口。`preview` 只保留为固定 `.output` 的普通预览。
- 当前生产 Compose 的 `web`、`api`、`worker` 已共享 `theme_releases` volume；继续沿用这个共享目录。
- 当前本地 API/worker 默认使用 `../../storage/theme-releases`，本地 Nuxt dev 应读取同一个目录。
- 主题切换 v1 仍是单节点切换；多节点滚动发布、CDN 缓存刷新、浏览器自动 reload 以后再做。
- 遵守仓库约定：先写/更新测试，再改实现；改 OpenAPI 时才跑 OpenAPI 校验。本计划不需要新增 API contract。

## Diagnosis

- 上传主题激活完成后，worker 会写 `theme-releases/current.json`，但文件里只有 `server` 等生产 release 信息，没有 `layerPath`，本地 Nuxt dev 无法知道该使用哪个 Nuxt Layer。
- 恢复内置默认主题目前只更新数据库状态，不写明确的 default `current.json`，所以生产 runtime 和未来 dev supervisor 都缺少“切回默认主题”的文件信号。
- `apps/web/package.json` 里的 `dev` 仍是裸 `nuxt dev --host 0.0.0.0 --dotenv ../../.env`，不会监听 `current.json`。
- `apps/web/scripts/runtime.mjs` 能看 `current.json.server`，但还没有明确 default 状态、`layerPath` 兼容策略、相对路径解析、Node runtime 配置和更稳的候选 server 切换保护。
- 之前手动验证发现：直接用 Node 跑已构建 Nitro release 正常；用 Bun 直接跑 Nitro server 可能出现端口/监听不确定性。因此生产托管和 worker 健康检查都应向 Node 收敛。

## Target `current.json` Contract

上传主题激活后写入：

```json
{
  "releaseId": 16,
  "extensionId": "sforum.signal-garden",
  "mode": "uploaded",
  "server": "/absolute/path/to/theme-releases/releases/16/.output/server/index.mjs",
  "layerPath": "/absolute/path/to/storage/extensions/sforum.signal-garden/current/layer",
  "activatedAt": "2026-07-07T12:34:56Z"
}
```

恢复默认主题后写入：

```json
{
  "extensionId": "sforum.default-theme",
  "mode": "default",
  "activatedAt": "2026-07-07T12:34:56Z"
}
```

旧格式继续兼容：

```json
{
  "server": "/absolute/path/to/theme-releases/releases/16/.output/server/index.mjs"
}
```

读取规则：

- `mode === "default"` 或没有 `current.json`：使用默认 `.output/server/index.mjs`，本地 dev 不设置 `SFORUM_THEME_LAYER`。
- `server` 非空：生产 runtime 启动这个 Nitro server；如果是相对路径，以 `SFORUM_THEME_RELEASE_ROOT` 为基准解析。
- `layerPath` 非空：本地 dev supervisor 设置 `SFORUM_THEME_LAYER=<layerPath>` 后启动 Nuxt dev；如果是相对路径，以仓库根目录为基准解析。
- JSON 损坏或选中的上传 release 缺文件：记录清晰错误。不要先停掉正在工作的旧 child；如果当前没有 child，回退默认 `.output` 保持站点可用。

## Files

- Modify: `apps/api/config/config.go`
- Modify/Test: `apps/api/config/config_test.go`
- Modify: `apps/api/app/Support/ThemeRuntime/builder.go`
- Modify/Test: `apps/api/app/Support/ThemeRuntime/builder_test.go`
- Modify: `apps/api/app/Jobs/Extensions/theme_activate.go`
- Modify/Test: `apps/api/app/Jobs/Extensions/theme_activate_test.go`
- Modify: `apps/api/app/Models/Extensions/service.go`
- Modify: `apps/api/app/Models/Extensions/store.go`
- Modify/Test: `apps/api/app/Models/Extensions/service_test.go`
- Modify: `apps/api/app/Providers/extensions.go`
- Modify: `apps/api/bootstrap/app.go`
- Modify: `apps/api/bootstrap/worker.go`
- Modify: `apps/api/Dockerfile`
- Modify: `compose.yaml`
- Modify: `apps/web/scripts/runtime.mjs`
- Create: `apps/web/scripts/dev-theme-runtime.mjs`
- Modify: `apps/web/package.json`
- Modify: `apps/web/Dockerfile`
- Modify/Test: `tests/validate-theme-runtime.js`
- Modify/Test: `tests/validate-theme-activation-progress.js` if copy validation depends on exact strings
- Modify/Test: `apps/web/tests/adminExtensions.test.ts` if release progress copy assertions require updates
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`
- Modify: `README.md`
- Modify: `knowledge/modules/frontend.md`
- Modify: `knowledge/modules/extensions.md`
- Create: `knowledge/sessions/2026-07-07-theme-runtime-convergence.md`

---

## Task 1: Lock The Backend `current.json` Contract With Tests

**Files:**

- Modify/Test: `apps/api/app/Support/ThemeRuntime/builder_test.go`
- Modify/Test: `apps/api/app/Jobs/Extensions/theme_activate_test.go`
- Modify/Test: `apps/api/app/Models/Extensions/service_test.go`
- Modify/Test: `apps/api/config/config_test.go`

- [ ] **Step 1: Update `TestBuilderWritesCurrentReleaseAtomically` to decode JSON and assert `server` plus `layerPath`**

Use a typed decode instead of `strings.Contains`, so missing fields fail clearly:

```go
var current CurrentRelease
if err := json.Unmarshal(raw, &current); err != nil {
	t.Fatalf("decode current: %v", err)
}
if current.ExtensionID != "starter.theme" || current.Server != server || current.LayerPath != layer {
	t.Fatalf("unexpected current release: %#v", current)
}
if current.Mode != "uploaded" {
	t.Fatalf("expected uploaded mode, got %q", current.Mode)
}
```

Expected first run: FAIL because `CurrentRelease` has no `LayerPath` or `Mode`.

- [ ] **Step 2: Add a builder test for default current state**

Add a test that calls:

```go
err := builder.WriteCurrent(context.Background(), CurrentRelease{
	ExtensionID: DefaultThemeExtensionID,
	Mode:        CurrentModeDefault,
})
```

Assert `current.json` has `extensionId: "sforum.default-theme"`, `mode: "default"`, a non-empty `activatedAt`, and empty `server`/`layerPath`.

Expected first run: FAIL until constants and struct fields exist.

- [ ] **Step 3: Update `TestActivateThemeWorkerMarksReleaseActive`**

Assert the worker writes uploaded release data:

```go
if builder.current.Server != "/tmp/out/server/index.mjs" {
	t.Fatalf("expected current server entry, got %#v", builder.current)
}
if builder.current.LayerPath != "/tmp/layer" {
	t.Fatalf("expected current layer path, got %#v", builder.current)
}
if builder.current.Mode != themeruntime.CurrentModeUploaded {
	t.Fatalf("expected uploaded mode, got %#v", builder.current)
}
```

Expected first run: FAIL because the worker does not pass `LayerPath` or `Mode`.

- [ ] **Step 4: Add a service test for restoring the default theme writing current state**

Add a fake writer near existing fakes:

```go
type fakeThemeCurrentWriter struct {
	current themeruntime.CurrentRelease
	err     error
}

func (w *fakeThemeCurrentWriter) WriteCurrent(_ context.Context, current themeruntime.CurrentRelease) error {
	w.current = current
	return w.err
}
```

Create `TestServiceActivateThemeRestoresBuiltinDefaultThemeWritesCurrentFile`:

```go
writer := &fakeThemeCurrentWriter{}
service := NewServiceWithThemeActivation(store, t.TempDir(), "", LocalRuntimeManager{}, fakeThemeBuilder{}, nil)
service.themeCurrentWriter = writer

active, err := service.ActivateTheme(context.Background(), extensionManager(), DefaultThemeID)
```

Assert `active.ID == DefaultThemeID`, `writer.current.ExtensionID == DefaultThemeID`, `writer.current.Mode == themeruntime.CurrentModeDefault`, `writer.current.Server == ""`, and `writer.current.LayerPath == ""`.

Expected first run: FAIL because service has no current writer.

- [ ] **Step 5: Add config tests for `THEME_NODE_PATH`**

In `apps/api/config/config_test.go`, assert default `ThemeNodePath == "node"` and env override `THEME_NODE_PATH=/usr/local/bin/node`.

Expected first run: FAIL because config has no `ThemeNodePath`.

- [ ] **Step 6: Run focused failing tests**

Run:

```sh
cd apps/api && go test ./app/Support/ThemeRuntime ./app/Jobs/Extensions ./app/Models/Extensions ./config -count=1
```

Expected: FAIL only for the new expectations.

---

## Task 2: Implement Backend Current Writer Semantics

**Files:**

- Modify: `apps/api/config/config.go`
- Modify: `apps/api/app/Support/ThemeRuntime/builder.go`
- Modify: `apps/api/app/Jobs/Extensions/theme_activate.go`
- Modify: `apps/api/app/Models/Extensions/service.go`
- Modify: `apps/api/app/Models/Extensions/store.go`
- Modify: `apps/api/app/Providers/extensions.go`
- Modify: `apps/api/bootstrap/app.go`
- Modify: `apps/api/bootstrap/worker.go`

- [ ] **Step 1: Extend API config**

Add:

```go
ThemeNodePath string
```

Load it with:

```go
ThemeNodePath: env("THEME_NODE_PATH", "node"),
```

- [ ] **Step 2: Extend `themeruntime.Config` and `CurrentRelease`**

In `apps/api/app/Support/ThemeRuntime/builder.go`, add constants and fields:

```go
const (
	CurrentModeUploaded         = "uploaded"
	CurrentModeDefault          = "default"
	DefaultThemeExtensionID      = "sforum.default-theme"
)

type Config struct {
	ReleaseRoot    string
	WebRoot        string
	BunPath        string
	NodePath       string
	BuildTimeout   time.Duration
	PreviewTimeout time.Duration
	PreviewPath    string
}

type CurrentRelease struct {
	ReleaseID   int64  `json:"releaseId,omitempty"`
	ExtensionID string `json:"extensionId"`
	Mode        string `json:"mode,omitempty"`
	Server      string `json:"server,omitempty"`
	LayerPath   string `json:"layerPath,omitempty"`
	ActivatedAt string `json:"activatedAt"`
}
```

Default `NodePath` to `"node"`. Keep `BunPath` for `bun run build`.

- [ ] **Step 3: Normalize mode before writing current**

In `WriteCurrent`, before setting `ActivatedAt`, normalize:

```go
if current.ExtensionID == "" {
	current.ExtensionID = DefaultThemeExtensionID
}
if current.Mode == "" {
	if current.Server != "" || current.LayerPath != "" {
		current.Mode = CurrentModeUploaded
	} else {
		current.Mode = CurrentModeDefault
	}
}
```

Do not require `server` for default mode. Keep atomic `current.json.tmp` then `os.Rename`.

- [ ] **Step 4: Run theme release health checks with Node**

In `HealthCheck`, replace preview command startup with `b.config.NodePath`:

```go
cmd := exec.CommandContext(previewCtx, b.config.NodePath, server)
```

Keep build command on `b.config.BunPath`.

- [ ] **Step 5: Pass `LayerPath` and mode from theme worker**

In `apps/api/app/Jobs/Extensions/theme_activate.go`, write:

```go
if err := w.Builder.WriteCurrent(ctx, themeruntime.CurrentRelease{
	ReleaseID:   release.ID,
	ExtensionID: extension.ID,
	Mode:        themeruntime.CurrentModeUploaded,
	Server:      result.ServerEntry,
	LayerPath:   release.LayerPath,
}); err != nil {
	return w.failThemeRelease(ctx, release.ID, err, result)
}
```

- [ ] **Step 6: Add a current writer to the extension service**

In `apps/api/app/Models/Extensions/store.go`, import `themeruntime` and add:

```go
type ThemeCurrentWriter interface {
	WriteCurrent(ctx context.Context, current themeruntime.CurrentRelease) error
}
```

In `Service`, add:

```go
themeCurrentWriter ThemeCurrentWriter
```

Update `NewServiceWithThemeActivation` or add `NewServiceWithThemeActivationAndCurrentWriter` so the provider can pass the writer without directly assigning private fields. Existing constructors should keep working with `nil`.

- [ ] **Step 7: Write default current when restoring the built-in default theme**

After `s.store.ActivateTheme(ctx, extension.ID)` succeeds and before the activated event is recorded:

```go
if s.themeCurrentWriter != nil {
	if err := s.themeCurrentWriter.WriteCurrent(ctx, themeruntime.CurrentRelease{
		ExtensionID: DefaultThemeID,
		Mode:        themeruntime.CurrentModeDefault,
	}); err != nil {
		return Extension{}, err
	}
}
```

This makes the admin success response depend on the runtime selection file being written.

- [ ] **Step 8: Wire the writer into API provider/bootstrap**

In `bootstrap/app.go`, create:

```go
themeCurrentWriter := themeruntime.NewBuilder(themeruntime.Config{
	ReleaseRoot: cfg.ThemeReleaseRoot,
	NodePath:    cfg.ThemeNodePath,
})
```

Pass it into both places that construct the extension service used for sync/routes:

- the `extensionService` used for `SyncBuiltins`
- `providers.NewExtensionsProviderWithRuntimeAndThemeActivation(...)`

Update `apps/api/app/Providers/extensions.go` to accept and forward `ThemeCurrentWriter`.

- [ ] **Step 9: Wire Node path into worker builder**

In `bootstrap/worker.go`, pass:

```go
NodePath: cfg.ThemeNodePath,
```

- [ ] **Step 10: Run backend tests**

Run:

```sh
cd apps/api && go test ./app/Support/ThemeRuntime ./app/Jobs/Extensions ./app/Models/Extensions ./config -count=1
```

Expected: PASS.

---

## Task 3: Make Production Runtime Node-Based And Safer

**Files:**

- Modify/Test: `apps/web/scripts/runtime.mjs`
- Modify/Test: `tests/validate-theme-runtime.js`
- Modify: `apps/web/Dockerfile`
- Modify: `apps/api/Dockerfile`
- Modify: `compose.yaml`

- [ ] **Step 1: Update Node validation expectations first**

In `tests/validate-theme-runtime.js`, add checks that `runtime.mjs` includes:

- `SFORUM_NODE_PATH`
- `SFORUM_FALLBACK_OUTPUT`
- `mode === 'default'` or equivalent default-mode handling
- relative server path resolution against `releaseRoot`
- child candidate existence check before stopping the old process

Also assert `apps/web/package.json` later exposes `theme:runtime`.

Expected first run: FAIL.

- [ ] **Step 2: Refactor `runtime.mjs` selection logic**

Implement helpers with these responsibilities:

```js
function fallbackServer() {
  return path.join(fallbackOutput, 'server/index.mjs')
}

function resolveServer(server) {
  return path.isAbsolute(server) ? server : path.resolve(releaseRoot, server)
}

function readSelection() {
  // returns { kind: 'default', server } or { kind: 'uploaded', server }
}
```

Behavior:

- Missing `current.json` returns default fallback.
- `current.mode === 'default'` returns default fallback.
- Non-empty `current.server` returns uploaded server, preserving old `{ "server": "..." }` compatibility.
- Invalid JSON logs `[sforum-web-runtime] invalid current release: ...`; if a child is already running, keep it. If no child is running, use fallback.

- [ ] **Step 3: Start Nitro with Node**

Use:

```js
const nodePath = process.env.SFORUM_NODE_PATH || 'node'
child = spawn(nodePath, [server], {
  stdio: 'inherit',
  env: {
    ...process.env,
    HOST: process.env.HOST || '0.0.0.0',
    PORT: process.env.PORT || '3000'
  }
})
```

Do not use `process.execPath`, because `runtime.mjs` might be launched by Bun in old deployments.

- [ ] **Step 4: Do not stop the old child until the candidate server exists**

In `startCurrent`, compute candidate first:

```js
const selection = readSelection()
if (!fs.existsSync(selection.server)) {
  console.error(`[sforum-web-runtime] selected server does not exist: ${selection.server}`)
  if (child) return
  selection.server = fallbackServer()
}
```

Only after a valid candidate is known should `stopChild()` run.

- [ ] **Step 5: Watch the release root with debounce**

Keep `fs.watch(releaseRoot, scheduleRestart)` and debounce around 250-300ms. Restart only when the resolved selected server path changes.

- [ ] **Step 6: Update web Docker prod runtime**

Change `apps/web/Dockerfile` prod stage to a Node-capable image or install Node in the existing image. Preferred simple path:

```dockerfile
FROM node:24-alpine AS prod

WORKDIR /app/apps/web
ENV HOST=0.0.0.0
ENV PORT=3000
ENV SFORUM_THEME_RELEASE_ROOT=/var/lib/sforum/theme-releases
ENV SFORUM_NODE_PATH=node
COPY --from=deps /app/apps/web/node_modules ./node_modules
COPY --from=build /app/apps/web/.output ./.output
COPY apps/web/scripts ./scripts
EXPOSE 3000
CMD ["node", "scripts/runtime.mjs"]
```

If the repository standardizes on `oven/bun`, the alternative is `apk add --no-cache nodejs` and `CMD ["node", "scripts/runtime.mjs"]`.

- [ ] **Step 7: Ensure worker image has Node for release health checks**

In `apps/api/Dockerfile` worker stage, install Node:

```dockerfile
RUN apk add --no-cache ca-certificates nodejs
ENV THEME_NODE_PATH=node
```

Keep `THEME_BUN_PATH=bun` because builds still use Bun.

- [ ] **Step 8: Add Compose env for Node paths**

In `compose.yaml`:

- web service: `SFORUM_NODE_PATH: ${SFORUM_NODE_PATH:-node}`
- worker service: `THEME_NODE_PATH: ${THEME_NODE_PATH:-node}`

- [ ] **Step 9: Run validation**

Run:

```sh
node tests/validate-theme-runtime.js
```

Expected: PASS after Task 4 updates package scripts too. If it fails only because `dev-theme-runtime.mjs` does not exist yet, continue to Task 4 and rerun.

---

## Task 4: Add Theme-Aware Local Dev Supervisor

**Files:**

- Create/Test: `apps/web/scripts/dev-theme-runtime.mjs`
- Modify/Test: `apps/web/package.json`
- Modify/Test: `tests/validate-theme-runtime.js`

- [ ] **Step 1: Extend validation for local dev scripts**

In `tests/validate-theme-runtime.js`, assert:

```js
assertIncludes(webPackage.scripts.dev, 'scripts/dev-theme-runtime.mjs', 'web dev must use theme-aware supervisor')
assertIncludes(webPackage.scripts['dev:plain'], 'nuxt dev --host 0.0.0.0 --dotenv ../../.env', 'web dev:plain must keep raw Nuxt dev')
assertIncludes(webPackage.scripts['theme:runtime'], 'scripts/runtime.mjs', 'theme:runtime must run production theme supervisor')
```

Read `apps/web/scripts/dev-theme-runtime.mjs` and assert it includes `SFORUM_THEME_LAYER`, `current.json`, `fs.watch`, `dev:plain`, and `SIGTERM`.

Expected first run: FAIL.

- [ ] **Step 2: Create `apps/web/scripts/dev-theme-runtime.mjs`**

Implement as a Node script. Required behavior:

- `releaseRoot = process.env.SFORUM_THEME_RELEASE_ROOT || path.resolve(process.cwd(), '../../storage/theme-releases')`
- `currentFile = path.join(releaseRoot, 'current.json')`
- read `current.layerPath`
- if `mode === "default"` or file missing, start Nuxt dev without `SFORUM_THEME_LAYER`
- if `layerPath` exists, start Nuxt dev with `SFORUM_THEME_LAYER=<resolved layerPath>`
- spawn only its own child: `bun run dev:plain`
- on current layer change, `SIGTERM` old child and restart
- do not inspect or kill unrelated port 3000 processes
- debounce watch events around 300ms
- print logs like:
  - `[sforum-dev-runtime] starting Nuxt dev with default theme`
  - `[sforum-dev-runtime] starting Nuxt dev with theme layer: ...`
  - `[sforum-dev-runtime] current.json changed; restarting Nuxt dev`

Suggested structure:

```js
import { spawn } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'

const releaseRoot = process.env.SFORUM_THEME_RELEASE_ROOT || path.resolve(process.cwd(), '../../storage/theme-releases')
const currentFile = path.join(releaseRoot, 'current.json')
const bunPath = process.env.SFORUM_BUN_PATH || 'bun'

let child = null
let activeLayer = null
let restartTimer = null

function readLayerPath() {
  // return null for default/missing current; return resolved layer path for uploaded current
}

function startDev(reason = 'startup') {
  // compare next layer with activeLayer; restart only when changed or child missing
}

function stopChild() {
  // SIGTERM only the child created by this script
}
```

- [ ] **Step 3: Update package scripts**

In `apps/web/package.json`:

```json
{
  "scripts": {
    "dev": "node scripts/dev-theme-runtime.mjs",
    "dev:plain": "nuxt dev --host 0.0.0.0 --dotenv ../../.env",
    "build": "NUXT_BUILD_DIR=${NUXT_BUILD_DIR:-.nuxt-build} nuxt build",
    "preview": "HOST=0.0.0.0 bun --env-file=../../.env scripts/preview.mjs",
    "theme:runtime": "node scripts/runtime.mjs",
    "typecheck": "NUXT_BUILD_DIR=.nuxt-typecheck nuxt typecheck"
  }
}
```

Do not change `preview` behavior except documentation.

- [ ] **Step 4: Run validation**

Run:

```sh
node tests/validate-theme-runtime.js
```

Expected: PASS.

---

## Task 5: Fix Admin Messaging

**Files:**

- Modify/Test: `apps/web/i18n/locales/zh-CN.json`
- Modify/Test: `apps/web/i18n/locales/en-US.json`
- Modify/Test: `tests/validate-theme-activation-progress.js`
- Modify/Test: `apps/web/tests/adminExtensions.test.ts` if existing tests assert exact translation behavior

- [ ] **Step 1: Update Chinese strings**

Use copy that reflects runtime behavior:

```json
{
  "themeActivated": "主题已激活。前台运行时正在使用该主题；本地 dev 会自动重启后生效。",
  "themeActivationQueued": "主题构建已排队。构建完成后前台运行时会自动切换；本地 dev 会自动重启后生效。"
}
```

For `admin.extensions.themeProgress.active`, use:

```json
"active": "前台运行时正在使用该主题；本地 dev 会自动重启后生效。"
```

- [ ] **Step 2: Update English strings**

Use equivalent copy:

```json
{
  "themeActivated": "Theme activated. The frontend runtime is using this theme; local dev will apply it after the automatic restart.",
  "themeActivationQueued": "Theme build queued. The frontend runtime will switch automatically after the build finishes; local dev will apply it after the automatic restart."
}
```

For `admin.extensions.themeProgress.active`, use:

```json
"active": "The frontend runtime is using this theme; local dev will apply it after the automatic restart."
```

- [ ] **Step 3: Keep alert behavior unchanged**

Do not make error alerts auto-dismiss. Non-error toasts may continue using the existing 10-second auto-dismiss behavior.

- [ ] **Step 4: Run frontend/admin copy validation**

Run:

```sh
node tests/validate-theme-activation-progress.js
cd apps/web && bun test tests/adminExtensions.test.ts
```

Expected: PASS.

---

## Task 6: Documentation And Knowledge Base

**Files:**

- Modify: `README.md`
- Modify: `knowledge/modules/frontend.md`
- Modify: `knowledge/modules/extensions.md`
- Create: `knowledge/sessions/2026-07-07-theme-runtime-convergence.md`

- [ ] **Step 1: Update README local development section**

Document:

- `cd apps/web && bun run dev` is theme-aware and follows backend theme activation by restarting Nuxt dev.
- `cd apps/web && bun run dev:plain` runs raw Nuxt dev for troubleshooting.
- `cd apps/web && bun run preview` previews the fixed `.output` build and does not follow admin theme switching.

- [ ] **Step 2: Update README production section**

Document:

- production `web` service runs `apps/web/scripts/runtime.mjs`
- production should not use `bun run preview` as the web server
- `web`, `api`, and `worker` must share `theme_releases`
- theme switch visibility is “next request/refresh on the same public URL”

- [ ] **Step 3: Update `knowledge/modules/frontend.md`**

Record:

- `bun run dev` is now the theme-aware dev supervisor
- `bun run dev:plain` is the raw Nuxt dev escape hatch
- production runtime uses Node to run selected Nitro server
- `preview` is fixed-build only

- [ ] **Step 4: Update `knowledge/modules/extensions.md`**

Record:

- `current.json` contains uploaded/default modes
- uploaded activation writes `server` and `layerPath`
- default restore writes explicit default state
- local and production runtimes consume the same selection file

- [ ] **Step 5: Add session handoff**

Create `knowledge/sessions/2026-07-07-theme-runtime-convergence.md`:

```md
# 2026-07-07 Session Handoff

## Changed

- Theme runtime selection converged on `theme-releases/current.json`.
- Production web runtime and local `bun run dev` now follow backend theme activation.

## Decisions

- `bun run preview` remains a fixed-build preview and is not part of admin theme switching.
- Nitro release servers run under Node instead of Bun.

## Next

- Manual smoke test uploaded theme activation and default restore on the same frontend URL.

## Open Questions

- Multi-node rollout and CDN cache invalidation remain future work.
```

---

## Final Verification

Run in this order:

```sh
cd apps/api && go test ./app/Support/ThemeRuntime ./app/Jobs/Extensions ./app/Models/Extensions ./config -count=1
node tests/validate-theme-runtime.js
node tests/validate-theme-activation-progress.js
cd apps/web && bun test tests/adminExtensions.test.ts
cd apps/web && bun run typecheck
./scripts/test.sh
```

Manual smoke, if time permits:

```sh
cd apps/web
bun run dev
```

Then:

1. In admin, activate an uploaded theme such as Signal Garden.
2. Watch the dev terminal log an automatic restart.
3. Refresh `http://127.0.0.1:3000/` and confirm the uploaded theme is visible.
4. Restore the built-in default theme in admin.
5. Watch the dev terminal restart again.
6. Refresh `http://127.0.0.1:3000/` and confirm the default UI returns.

Production smoke, if a local Compose stack is available:

```sh
docker compose up -d --build web api worker
docker compose logs -f web worker
```

Then activate an uploaded theme in admin and confirm the same public `WEB_PORT` shows the new theme after refresh.

## Acceptance Criteria

- Production Compose `web` entrypoint is the theme runtime, not `bun run preview`.
- Uploaded theme activation writes `current.json` with `server`, `layerPath`, `mode`, `extensionId`, and `activatedAt`.
- Built-in default theme restore writes explicit default `current.json`.
- `apps/web/scripts/runtime.mjs` supports uploaded/default/legacy current states and runs selected Nitro servers with Node.
- `cd apps/web && bun run dev` follows backend theme activation without extra commands.
- `cd apps/web && bun run dev:plain` remains available for raw Nuxt dev troubleshooting.
- `cd apps/web && bun run preview` is documented as fixed-build preview only.
- Admin success copy no longer implies “database status changed” equals visible runtime switch unless the current file was written.
- Focused Go tests, Node validations, web tests, typecheck, and project test script pass.

