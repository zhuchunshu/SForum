# Trusted Admin Plugin Runtime Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the approved trusted, build-time Vue component runtime for core-owned admin slots while preserving the currently served site across package, build, activation, and process failures.

**Architecture:** Make extension package versions immutable by content digest, persist exact super-administrator trust grants, and plan one deterministic Web Release containing the active theme plus every enabled trusted frontend plugin. River workers produce verified immutable artifacts only through `ready`; an API-owned coordinator prepares plugin runtimes and writes desired state, while the Node web supervisor owns proxy switching and durable `active.json` or failure acknowledgements. Nuxt SSR consumes manifest-only metadata; a generated client registry lazily loads approved Vue components through an extension-scoped Admin SDK.

**Tech Stack:** Go 1.25+, Fiber v3, pgx/PostgreSQL, River v0.40, Goose, Bun 1.3, Nuxt 4, Vue 3, Nuxt UI 4, TypeScript, Node supervisor scripts, modular OpenAPI, Bun tests, Go tests, and existing browser QA tooling.

**Approved spec:** `docs/superpowers/specs/2026-07-10-trusted-admin-plugin-runtime-design.md`

---

## Execution Rules On Shared `main`

The user explicitly requires implementation on `main`. Other tasks may edit and commit concurrently, so every task follows this protocol:

```bash
git status --short
git log -1 --oneline --decorate
```

- Re-read every file immediately before editing it.
- If another task has an unstaged or untracked change in a required path, wait for that task to commit, then re-read and rebase the planned edit on the new `HEAD`.
- Never use `git add -A`, `git add .`, `git stash`, `git reset`, `git checkout --`, or amend another task's commit.
- Stage only the explicit paths listed in each commit step. Run `git diff --cached --check` and `git diff --cached --name-status` before every commit.
- The supervisor lifecycle files were recently edited by a background task. Task 8 must re-read their latest committed versions and must not touch either path while another unstaged change is present.
- Before creating a migration, run `rg --files apps/api/database/migrations | sort`. This plan reserves `202607100004` and `202607100005`; if a concurrent commit takes either number, rename this plan's migration to the next free number and update the embedded-migration assertion in the same commit.

## Library Decisions

- Keep Nuxt/Vite static imports and generated modules; do not add Module Federation or a runtime ESM loader.
- Use `github.com/tailscale/hujson@v0.0.0-20260302212456-ecc657c15afd` to standardize Bun's JSON-with-comments lockfile before `encoding/json` decoding. It is a focused, maintained BSD-3-Clause parser and avoids ad hoc comment/trailing-comma stripping.
- Use Vue's built-in `onErrorCaptured`, async components, provide/inject, and the existing browser tooling. Do not add a second component framework or registry library.
- Keep River's public worker APIs and the existing serialized `theme` queue. Do not use unstable `riverdriver` APIs.

## File Map

| Boundary | Responsibility | Primary paths |
| --- | --- | --- |
| Package input | Immutable snapshots, canonical digest, Bun lock inspection | `apps/api/app/Support/ExtensionPackage/` |
| Manifest | `frontend.admin` DTO and catalog-aware validation | `apps/api/app/Support/ExtensionManifest/` |
| Domain | Grants, release state, snapshots, effects, planning | `apps/api/app/Models/Extensions/` |
| Persistence | Two Goose migrations and focused pgx stores | `apps/api/database/migrations/`, Extensions postgres files |
| Build | Sanitized isolated workspace, generated registry, artifact verification | `apps/api/app/Support/WebReleaseRuntime/`, `apps/api/app/Jobs/Extensions/` |
| Activation | API-owned runtime effects, pointers, acknowledgement reconciliation | `apps/api/app/Support/WebReleaseCoordinator/`, `apps/api/bootstrap/` |
| HTTP contract | Trust/release controllers, routes, localized reasons, OpenAPI | `apps/api/app/Http/Controllers/Extensions/`, `contracts/openapi/` |
| Admin SDK | Exact public package plus host-only injection contract | `apps/web/packages/admin-sdk/` |
| Nuxt host | Slot registry/rendering, trust/releases UI, stale-tab monitor | `apps/web/app/`, `apps/web/server/`, `apps/web/nuxt.config.ts` |
| Supervisor | Desired/active/failure file protocol and proxy orchestration | `apps/web/scripts/`, `apps/web/tests/` |
| Deployment | Shared volume/env compatibility and worker/web images | Compose files, Dockerfiles, env examples, `deploy/volumes/README.md` |
| Verification | Shared plugin/registry fixtures, repo gates, author docs, memory | `tests/`, `docs/extensions/`, `knowledge/` |

Task-local file lists below are authoritative and provide every exact create/modify path.

## Stable Contracts Used Throughout The Plan

```go
type Dependency struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Integrity string `json:"integrity,omitempty"`
}

type DependencySummary struct {
	Direct     []Dependency `json:"direct"`
	Resolved   []Dependency `json:"resolved"`
	LockDigest string       `json:"lockDigest"`
}

type FrontendStatus struct {
	ExtensionID  string                 `json:"extensionId"`
	Declaration *ManifestAdminFrontend `json:"declaration,omitempty"`
	TrustState  string                 `json:"trustState"`
	Digest      string                 `json:"digest,omitempty"`
	Dependencies DependencySummary     `json:"dependencies"`
}

type WebReleaseSummary struct {
	ID              int64            `json:"id"`
	Status          WebReleaseStatus `json:"status"`
	CompositionHash string           `json:"compositionHash"`
	ReloadMode      string           `json:"reloadMode"`
}

type WebReleaseStatus string
const (
	WebReleaseQueued WebReleaseStatus = "queued"
	WebReleaseResolving WebReleaseStatus = "resolving"
	WebReleaseInstalling WebReleaseStatus = "installing"
	WebReleaseBuilding WebReleaseStatus = "building"
	WebReleaseVerifying WebReleaseStatus = "verifying"
	WebReleaseReady WebReleaseStatus = "ready"
	WebReleaseActivating WebReleaseStatus = "activating"
	WebReleaseActive WebReleaseStatus = "active"
	WebReleaseInactive WebReleaseStatus = "inactive"
	WebReleaseFailed WebReleaseStatus = "failed"
	WebReleaseSuperseded WebReleaseStatus = "superseded"
	WebReleaseRolledBack WebReleaseStatus = "rolled_back"
)
type ExtensionOperation struct {
	Extension  Extension           `json:"extension"`
	Frontend   *FrontendStatus     `json:"frontend,omitempty"`
	WebRelease *WebReleaseSummary  `json:"webRelease,omitempty"`
	Queued     bool                `json:"queued"`
}
```

```ts
import type { Ref } from 'vue'

export interface AdminSlotContextMap {}
export interface AdminSlotOptionsMap {}

export type SForumAdminHost = {
  extensionId: string
  locale: Readonly<Ref<string>>
  t: (key: string, params?: Record<string, unknown>) => string
  navigate: (adminPath: string) => Promise<void>
  toast: (input: { title: string, description?: string, kind?: 'success' | 'error' }) => void
  extensionRequest: <T>(path: string, options?: Record<string, unknown>) => Promise<T>
}

export type AdminSlotPoint = keyof AdminSlotContextMap & string
export type AdminSlotProps<P extends AdminSlotPoint> = Readonly<{
  context: AdminSlotContextMap[P]
  options: AdminSlotOptionsMap[P]
  extensionId: string
  contributionId: string
}>

export const ADMIN_SDK_API_VERSION = 1 as const
export const useSForumAdminHost: () => SForumAdminHost
```

The infrastructure project does not register `admin.jobs.*`. A build-only `admin.test.fixture` declaration is supplied by the integration fixture and omitted from ordinary production catalog generation.

---

### Task 1: Make Installed Extension Versions Immutable

**Files:**

- Create: `apps/api/database/migrations/202607100004_immutable_extension_versions.sql`
- Create: `apps/api/app/Support/ExtensionPackage/digest.go`
- Create: `apps/api/app/Support/ExtensionPackage/digest_test.go`
- Create: `apps/api/app/Support/ExtensionPackage/snapshot.go`
- Create: `apps/api/app/Support/ExtensionPackage/snapshot_test.go`
- Modify: `apps/api/app/Models/Extensions/types.go`
- Modify: `apps/api/app/Models/Extensions/postgres_store.go`
- Modify: `apps/api/app/Models/Extensions/service.go`
- Modify: `apps/api/app/Models/Extensions/service_test.go`
- Modify: `apps/api/database/migrations/embed_test.go`

- [ ] **Step 1: Write failing canonical digest and snapshot tests**

Cover sorted relative paths, normalized file modes, byte changes, mode changes, symlink rejection, two archives with identical normalized content producing one digest, and two packages with the same ID/version but different content remaining in separate digest directories.

```go
func TestDigestTreeRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "component.vue"), []byte("<template />"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("component.vue", filepath.Join(root, "alias.vue")); err != nil {
		t.Fatal(err)
	}
	_, err := extensionpackage.DigestTree(root)
	if !errors.Is(err, extensionpackage.ErrSymlink) {
		t.Fatalf("expected ErrSymlink, got %v", err)
	}
}
```

- [ ] **Step 2: Run the focused tests and verify RED**

Run: `cd apps/api && go test ./app/Support/ExtensionPackage ./app/Models/Extensions`

Expected: FAIL because the package and `PackageDigest` persistence do not exist.

- [ ] **Step 3: Implement the canonical digest and immutable snapshot contract**

```go
type Snapshot struct {
	Root       string
	Manifest   string
	Digest     string
}

type File struct {
	Path string
	Mode fs.FileMode
	Body []byte
}

func DigestTree(root string) (string, error)
func SnapshotUploaded(root string, manifest []byte, files []File) (Snapshot, error)
func SnapshotBuiltin(root string, destinationRoot string) (Snapshot, error)
```

Hash each regular file as `relative path + NUL + normalized mode + NUL + length + NUL + bytes`, sorted by slash-normalized path. Exclude storage wrappers such as the original ZIP. Reject every symlink before copying or hashing. Write to a staging directory, `fsync` files, then atomically rename to `<EXTENSION_ROOT>/<id>/<version>/<digest>`.

- [ ] **Step 4: Persist immutable version identity**

Add `package_digest TEXT NOT NULL DEFAULT ''`, replace `UNIQUE(extension_id, version)` with `UNIQUE(extension_id, version, package_digest)`, and make `SaveInstalled`/`SaveBuiltin` insert a new version row instead of updating an old one. Add `PackageDigest` to `Extension`, `SaveInstalledInput`, and `SaveBuiltinInput`. Preserve legacy rows with an empty digest and preserve legacy `package.zip` path resolution.

- [ ] **Step 5: Reject ZIP symlink entries before extraction**

In `readArchive`, reject `file.Mode()&os.ModeSymlink != 0` before reading bytes. Install into staging, compute the canonical digest, atomically publish the snapshot, and only then update `active_version_id`.

- [ ] **Step 6: Run tests and commit**

Run: `cd apps/api && go test ./app/Support/ExtensionPackage ./app/Models/Extensions ./database/migrations`

```bash
git add apps/api/database/migrations/202607100004_immutable_extension_versions.sql apps/api/database/migrations/embed_test.go apps/api/app/Support/ExtensionPackage apps/api/app/Models/Extensions/types.go apps/api/app/Models/Extensions/postgres_store.go apps/api/app/Models/Extensions/service.go apps/api/app/Models/Extensions/service_test.go
git diff --cached --check
git commit -m "feat: make extension package versions immutable"
```

### Task 2: Validate Admin Frontend Manifests And Locked Dependencies

**Files:**

- Create: `apps/api/app/Support/ExtensionManifest/admin_frontend.go`
- Create: `apps/api/app/Support/ExtensionManifest/admin_frontend_test.go`
- Create: `apps/api/app/Support/ExtensionPackage/frontend.go`
- Create: `apps/api/app/Support/ExtensionPackage/frontend_test.go`
- Modify: `apps/api/app/Support/ExtensionManifest/manifest.go`
- Modify: `apps/api/app/Support/ExtensionManifest/manifest_test.go`
- Modify: `apps/api/app/Models/Extensions/types.go`
- Modify: `apps/api/go.mod`
- Modify: `apps/api/go.sum`

- [ ] **Step 1: Add failing table-driven manifest tests**

Cover plugin-only `frontend.admin`, safe root/components/locales, positive supported API version, required `zh-CN`/`en-US`, one component per trusted contribution, unknown point rejection, duplicate IDs, escaping paths, missing files, symlinks, forbidden `workspace:`/`file:`/`link:` dependencies, incompatible host peer ranges, plugin-supplied Nuxt/Vite/Nitro hooks, and attempts to relax host CSP. Pass a test catalog containing only `admin.test.fixture`; keep the production catalog empty for trusted component points.

- [ ] **Step 2: Verify RED**

Run: `cd apps/api && go test ./app/Support/ExtensionManifest ./app/Support/ExtensionPackage`

Expected: FAIL because `ManifestFrontend.Admin` and catalog-aware validation are absent.

- [ ] **Step 3: Add the manifest contracts**

```go
type ManifestAdminFrontend struct {
	Root       string            `json:"root"`
	APIVersion int               `json:"apiVersion"`
	Components map[string]string `json:"components"`
	Locales    map[string]string `json:"locales"`
}

type AdminComponentContributionPayload struct {
	Component string `json:"component"`
}

func ValidateWithContributionPoints(manifest Manifest, points []ContributionPointDefinition) error
```

Keep `Validate(manifest)` as the compatibility entrypoint using `ContributionPointDefinitions()`. Descriptor points and trusted component points remain different `Kind` values; a component payload may have slot-owned extra fields, but the host always validates its `component` binding.

- [ ] **Step 4: Add the maintained JSONC parser and package inspection**

Run with the repository proxy:

```bash
cd apps/api
https_proxy=http://127.0.0.1:7897 http_proxy=http://127.0.0.1:7897 all_proxy=socks5://127.0.0.1:7897 GOPROXY=https://goproxy.cn,direct go get github.com/tailscale/hujson@v0.0.0-20260302212456-ecc657c15afd
```

Standardize `bun.lock` with `hujson.Standardize`, decode it with `encoding/json`, and return sorted direct/resolved dependency summaries plus the lockfile digest. Validate `package.json`, the frozen lock, locale JSON string leaves, required paths, allowed protocols, and host peer declarations without executing plugin code.

- [ ] **Step 5: Run tests and commit**

Run: `cd apps/api && go test ./app/Support/ExtensionManifest ./app/Support/ExtensionPackage ./app/Models/Extensions`

```bash
git add apps/api/app/Support/ExtensionManifest/admin_frontend.go apps/api/app/Support/ExtensionManifest/admin_frontend_test.go apps/api/app/Support/ExtensionManifest/manifest.go apps/api/app/Support/ExtensionManifest/manifest_test.go apps/api/app/Support/ExtensionPackage/frontend.go apps/api/app/Support/ExtensionPackage/frontend_test.go apps/api/app/Models/Extensions/types.go apps/api/go.mod apps/api/go.sum
git diff --cached --check
git commit -m "feat: validate trusted admin frontend packages"
```

### Task 3: Add Trust And Web Release Persistence

**Files:**

- Create: `apps/api/database/migrations/202607100005_trusted_admin_web_releases.sql`
- Create: `apps/api/app/Models/Extensions/frontend_types.go`
- Create: `apps/api/app/Models/Extensions/frontend_store.go`
- Create: `apps/api/app/Models/Extensions/frontend_postgres.go`
- Create: `apps/api/app/Models/Extensions/frontend_postgres_test.go`
- Create: `apps/api/app/Models/Extensions/web_release_types.go`
- Create: `apps/api/app/Models/Extensions/web_release_state.go`
- Create: `apps/api/app/Models/Extensions/web_release_state_test.go`
- Create: `apps/api/app/Models/Extensions/web_release_store.go`
- Create: `apps/api/app/Models/Extensions/web_release_postgres.go`
- Create: `apps/api/app/Models/Extensions/web_release_postgres_test.go`
- Modify: `apps/api/database/migrations/embed_test.go`

- [ ] **Step 1: Write failing state-machine tests**

Assert every allowed transition, reject skips such as `queued -> active`, distinguish normal `active -> inactive` from compensation `active -> rolled_back`, keep final states immutable, and require retry/rollback to create a new release ID.

```go
func TestWebReleaseTransitionRejectsFinalStateMutation(t *testing.T) {
	for _, state := range []WebReleaseStatus{WebReleaseFailed, WebReleaseSuperseded, WebReleaseInactive, WebReleaseRolledBack} {
		require.Error(t, ValidateWebReleaseTransition(state, WebReleaseBuilding))
	}
}
```

- [ ] **Step 2: Verify RED**

Run: `cd apps/api && go test ./app/Models/Extensions ./database/migrations`

- [ ] **Step 3: Create the schema exactly around the approved ownership model**

Create `extension_frontend_trust_grants`, `web_releases`, `web_release_extensions`, `web_release_extension_effects`, and `web_release_events`. Add a monotonic release generation sequence, partial unique indexes for one active Web Release and one live exact grant, foreign keys with safe delete behavior, JSONB immutable snapshots, timestamps, `reload_mode`, activation checkpoint, artifact fields, public reason/message, and `web_release_id` on legacy `extension_theme_releases`.

- [ ] **Step 4: Implement focused stores instead of growing `postgres_store.go`**

```go
type FrontendTrustStore interface {
	FrontendGrant(context.Context, string, string, string) (FrontendTrustGrant, error)
	CreateFrontendGrant(context.Context, FrontendTrustGrantInput) (FrontendTrustGrant, error)
	RequestFrontendRevocation(context.Context, FrontendRevocationInput) (FrontendTrustGrant, error)
	FinalizeFrontendRevocations(context.Context, int64) error
}

type WebReleaseStore interface {
	CreateWebRelease(context.Context, WebReleaseCreateInput) (WebRelease, error)
	TransitionWebRelease(context.Context, WebReleaseTransitionInput) (WebRelease, error)
	WebRelease(context.Context, int64) (WebReleaseDetail, error)
	ListWebReleases(context.Context, WebReleaseListInput) (WebReleasePage, error)
}
```

All transition methods append `web_release_events` in the same transaction. Paginate with `page/perPage` and return `{items,total,page,perPage}`.

- [ ] **Step 5: Run tests and commit**

Run: `cd apps/api && go test ./app/Models/Extensions ./database/migrations`

```bash
git add apps/api/database/migrations/202607100005_trusted_admin_web_releases.sql apps/api/database/migrations/embed_test.go apps/api/app/Models/Extensions/frontend_types.go apps/api/app/Models/Extensions/frontend_store.go apps/api/app/Models/Extensions/frontend_postgres.go apps/api/app/Models/Extensions/frontend_postgres_test.go apps/api/app/Models/Extensions/web_release_types.go apps/api/app/Models/Extensions/web_release_state.go apps/api/app/Models/Extensions/web_release_state_test.go apps/api/app/Models/Extensions/web_release_store.go apps/api/app/Models/Extensions/web_release_postgres.go apps/api/app/Models/Extensions/web_release_postgres_test.go
git diff --cached --check
git commit -m "feat: persist trusted frontend web releases"
```

### Task 4: Plan Deterministic Compositions And Trust Lifecycle

**Files:**

- Create: `apps/api/app/Models/Extensions/web_release_planner.go`
- Create: `apps/api/app/Models/Extensions/web_release_planner_test.go`
- Create: `apps/api/app/Models/Extensions/frontend_service.go`
- Create: `apps/api/app/Models/Extensions/frontend_service_test.go`
- Create: `apps/api/app/Models/Extensions/web_release_service.go`
- Create: `apps/api/app/Models/Extensions/web_release_service_test.go`
- Create: `apps/api/app/Jobs/Extensions/web_release_build.go`
- Create: `apps/api/app/Jobs/Extensions/web_release_build_test.go`
- Modify: `apps/api/app/Models/Extensions/service.go`
- Modify: `apps/api/app/Models/Extensions/store.go`

- [ ] **Step 1: Write failing planner and authorization tests**

Cover deterministic sorting, composition hash stability, enabled/trusted inclusion, disabled/untrusted/invalidated/pending-revocation exclusion, protected built-in source trust, duplicate desired composition idempotency, exact digest recheck at grant time, inactive super-admin denial, extension manager grant denial, disabled-plugin grant returning immediate state, enabled-plugin grant creating a release, inactive revoke returning immediate state, active revoke creating a safe release, ordinary changes using `prompt`, revocation/security rollback using `force`, and restore-defaults preserving backend status/settings.

- [ ] **Step 2: Verify RED**

Run: `cd apps/api && go test ./app/Models/Extensions`

- [ ] **Step 3: Implement the deterministic composition input**

```go
type WebComposition struct {
	Theme      WebThemeSnapshot       `json:"theme"`
	Extensions []WebExtensionSnapshot `json:"extensions"`
	WebSource  string                 `json:"webSource"`
	WebLock    string                 `json:"webLock"`
	SDKVersion int                    `json:"sdkVersion"`
	BunVersion string                 `json:"bunVersion"`
	Contract   int                    `json:"contract"`
}

type PlanWebReleaseInput struct {
	TriggerKind        string
	TriggerExtensionID string
	TargetThemeID      string
	RequestedBy        int64
	ReloadMode         string
}

type PlannedWebRelease struct {
	Composition WebComposition
	Hash        string
	Existing    *WebRelease
}

type WebReleaseBuildEnqueuer interface {
	EnqueueWebReleaseBuildTx(context.Context, pgx.Tx, int64) error
}

func (p *WebReleasePlanner) Plan(context.Context, PlanWebReleaseInput) (PlannedWebRelease, error)
```

Canonicalize JSON after sorting contributions by `order`, extension ID, and contribution ID. Hash the canonical bytes with SHA-256. Recalculate each package digest from its immutable snapshot before accepting it into a plan.

- [ ] **Step 4: Implement super-administrator trust rules and two-phase revocation**

Use `actor.IsSuperAdmin()` in service methods. Require `GrantFrontendInput.PackageDigest` to equal the freshly calculated digest. Set `revocation_pending` immediately, exclude it from all new and rollback plans, and finalize only when the safe release becomes active. Built-ins use source trust and never receive interactive grant rows.

- [ ] **Step 5: Create queued release plus River job atomically**

Use the existing `Dispatcher.EnqueueTx` through a narrow `WebReleaseBuildEnqueuer` adapter. The transaction inserts the release, immutable extension snapshots, effect rows, initial event, and `extension.web_release_build` job. If the exact composition is active or non-final, return that release without inserting a duplicate.

Define `WebReleaseBuildArgs`, `Kind`, queue/max-attempt/unique options, and the transaction-aware dispatcher adapter in `web_release_build.go`; Task 7 adds the worker implementation to the same file.

- [ ] **Step 6: Run tests and commit**

Run: `cd apps/api && go test ./app/Models/Extensions ./app/Support/Jobs`

```bash
git add apps/api/app/Models/Extensions/web_release_planner.go apps/api/app/Models/Extensions/web_release_planner_test.go apps/api/app/Models/Extensions/frontend_service.go apps/api/app/Models/Extensions/frontend_service_test.go apps/api/app/Models/Extensions/web_release_service.go apps/api/app/Models/Extensions/web_release_service_test.go apps/api/app/Jobs/Extensions/web_release_build.go apps/api/app/Jobs/Extensions/web_release_build_test.go apps/api/app/Models/Extensions/service.go apps/api/app/Models/Extensions/store.go
git diff --cached --check
git commit -m "feat: plan trusted frontend compositions"
```

### Task 5: Expose Trust And Web Release API Contracts

**Files:**

- Create: `apps/api/app/Http/Controllers/Extensions/frontend.go`
- Create: `apps/api/app/Http/Controllers/Extensions/web_releases.go`
- Modify: `apps/api/app/Http/Controllers/Extensions/controller.go`
- Modify: `apps/api/app/Http/Controllers/Extensions/controller_test.go`
- Modify: `apps/api/app/Http/Controllers/Extensions/routes.go`
- Modify: `apps/api/app/Support/Localization/messages.go`
- Modify: `contracts/openapi.yaml`
- Modify: `contracts/openapi/paths/extensions.yaml`
- Modify: `contracts/openapi/schemas/extensions.yaml`
- Modify: `contracts/openapi/components/parameters.yaml`

- [ ] **Step 1: Write failing controller tests for every policy and status branch**

Test `extension.manage` reads; active-super-admin-only grant/revoke/restore; CSRF-protected unsafe routes; grant disabled `200`, grant enabled `202`, revoke absent `200`, revoke active `202`, restore already-default `200`, restore queued `202`; paginated list/detail; retry and rollback creating new IDs; ineligible rollback; and stable localized reasons.

- [ ] **Step 2: Verify RED**

Run: `cd apps/api && go test ./app/Http/Controllers/Extensions`

- [ ] **Step 3: Register the approved endpoints**

```text
GET    /api/v1/admin/extensions/{extensionID}/frontend
POST   /api/v1/admin/extensions/{extensionID}/frontend/trust
DELETE /api/v1/admin/extensions/{extensionID}/frontend/trust
GET    /api/v1/admin/web-releases
GET    /api/v1/admin/web-releases/{releaseID}
POST   /api/v1/admin/web-releases/{releaseID}/retry
POST   /api/v1/admin/web-releases/{releaseID}/rollback
POST   /api/v1/admin/web-releases/restore-defaults
```

The trust request body is `{ "packageDigest": "<64 lowercase hex characters>" }`. Response data includes explicit `queued` and optional `webRelease`; frontend code never has to infer HTTP status from shape.

- [ ] **Step 4: Update modular OpenAPI without registering Jobs slots**

Add `WebReleaseID`, `FrontendStatus`, `FrontendTrustOperation`, `WebReleaseSummary`, `WebReleaseDetail`, `WebReleasePage`, request schemas, transition events, stable error descriptions, and both `200`/`202` responses. Extend `frontend.admin` manifest schema. Do not add `admin.jobs.*` to the contribution catalog.

- [ ] **Step 5: Verify and commit**

Run: `cd apps/api && go test ./app/Http/Controllers/Extensions`

Run: `ruby scripts/validate-openapi-refs.rb`

```bash
git add apps/api/app/Http/Controllers/Extensions/frontend.go apps/api/app/Http/Controllers/Extensions/web_releases.go apps/api/app/Http/Controllers/Extensions/controller.go apps/api/app/Http/Controllers/Extensions/controller_test.go apps/api/app/Http/Controllers/Extensions/routes.go apps/api/app/Support/Localization/messages.go contracts/openapi.yaml contracts/openapi/paths/extensions.yaml contracts/openapi/schemas/extensions.yaml contracts/openapi/components/parameters.yaml
git diff --cached --check
git commit -m "feat: expose trusted frontend release APIs"
```

### Task 6: Add The Admin SDK And Empty Host Runtime

**Files:**

- Create: `apps/web/packages/admin-sdk/package.json`
- Create: `apps/web/packages/admin-sdk/src/index.ts`
- Create: `apps/web/packages/admin-sdk/src/host.ts`
- Create: `apps/web/packages/admin-sdk/src/internal.ts`
- Create: `apps/web/app/runtime/admin-extensions/types.ts`
- Create: `apps/web/app/runtime/admin-extensions/catalog.ts`
- Create: `apps/web/app/runtime/admin-extensions/empty-metadata.ts`
- Create: `apps/web/app/runtime/admin-extensions/empty-registry.client.ts`
- Create: `apps/web/app/runtime/admin-extensions/quarantine.ts`
- Create: `apps/web/app/composables/useAdminExtensionRegistry.ts`
- Create: `apps/web/app/components/SFAdminExtensionSlot.vue`
- Create: `apps/web/app/components/SFAdminExtensionContribution.vue`
- Create: `apps/web/tests/adminSdk.test.ts`
- Create: `apps/web/tests/adminExtensionRegistry.test.ts`
- Modify: `apps/web/package.json`
- Modify: `apps/web/bun.lock`
- Modify: `apps/web/nuxt.config.ts`
- Modify: `apps/web/Dockerfile`
- Modify: `apps/api/Dockerfile`

- [ ] **Step 1: Write failing SDK and registry tests**

Assert the exact public exports, API version `1`, declaration-merging types, no internal injection key in the public entrypoint, deterministic metadata order, lazy loader lookup, owner-scoped host injection, extension-prefixed translation/request paths, independent retry state, and quarantine on the third failure for `release + extension + contribution`.

- [ ] **Step 2: Verify RED**

Run: `cd apps/web && bun test tests/adminSdk.test.ts tests/adminExtensionRegistry.test.ts`

- [ ] **Step 3: Create a real local workspace package**

Set `apps/web/package.json` workspaces to `packages/*` and add `"@sforum/admin-sdk": "workspace:*"`. Give the SDK package semantic version `1.0.0` and API generation `1`; its `exports` map exposes `.` and a host-only `./internal` subpath. Docker dependency stages copy `packages/admin-sdk/package.json` before `bun install --frozen-lockfile`.

- [ ] **Step 4: Implement the host runtime with empty production registries**

```ts
export type AdminComponentLoader = () => Promise<{ default: Component }>

export type AdminComponentMetadata = {
  point: string
  extensionId: string
  contributionId: string
  componentId: string
  order: number
  label: Record<string, string>
  options: Record<string, unknown>
}
```

`SFAdminExtensionSlot` renders stable metadata placeholders during SSR. On the client it delegates each contribution to `SFAdminExtensionContribution`, which creates an extension-scoped host, lazily loads one module, contains errors with `onErrorCaptured`, and exposes a retry button. Map app locale `en` to manifest locale `en-US`; prefix all translations and extension requests by owner; route navigation through `useAdminRoutes`; apply 10-second success and persistent error Toast behavior. Store failure counts in `sessionStorage`; a new embedded release ID naturally changes the key.

- [ ] **Step 5: Configure aliases and host peer deduplication**

`nuxt.config.ts` selects generated metadata/registry paths from `SFORUM_ADMIN_REGISTRY_ROOT` or the empty modules. Alias `@sforum/admin-sdk` to the workspace source and dedupe `vue`, `vue-router`, `nuxt`, `@nuxt/ui`, and the SDK. The server graph may import metadata only; client loaders stay in a `.client.ts` module.

- [ ] **Step 6: Verify and commit**

Run: `cd apps/web && bun test tests/adminSdk.test.ts tests/adminExtensionRegistry.test.ts`

Run: `cd apps/web && bun run typecheck && bun run build`

```bash
git add apps/web/packages/admin-sdk apps/web/app/runtime/admin-extensions apps/web/app/composables/useAdminExtensionRegistry.ts apps/web/app/components/SFAdminExtensionSlot.vue apps/web/app/components/SFAdminExtensionContribution.vue apps/web/tests/adminSdk.test.ts apps/web/tests/adminExtensionRegistry.test.ts apps/web/package.json apps/web/bun.lock apps/web/nuxt.config.ts apps/web/Dockerfile apps/api/Dockerfile
git diff --cached --check
git commit -m "feat: add trusted admin component sdk"
```

### Task 7: Build Verified Web Releases In River

**Files:**

- Create: `apps/api/app/Support/WebReleaseRuntime/types.go`
- Create: `apps/api/app/Support/WebReleaseRuntime/environment.go`
- Create: `apps/api/app/Support/WebReleaseRuntime/environment_test.go`
- Create: `apps/api/app/Support/WebReleaseRuntime/registry.go`
- Create: `apps/api/app/Support/WebReleaseRuntime/registry_test.go`
- Create: `apps/api/app/Support/WebReleaseRuntime/builder.go`
- Create: `apps/api/app/Support/WebReleaseRuntime/builder_test.go`
- Create: `apps/api/app/Support/WebReleaseRuntime/pointers.go`
- Create: `apps/api/app/Support/Postgres/advisory_lock.go`
- Create: `apps/api/app/Support/Postgres/advisory_lock_test.go`
- Create: `apps/web/build/admin-extension-guard.ts`
- Create: `apps/web/tests/adminExtensionBuildGuard.test.ts`
- Modify: `apps/api/app/Jobs/Extensions/web_release_build.go`
- Modify: `apps/api/app/Jobs/Extensions/web_release_build_test.go`
- Modify: `apps/api/app/Support/ThemeRuntime/builder.go`
- Modify: `apps/api/bootstrap/worker.go`
- Modify: `apps/api/bootstrap/worker_test.go`
- Modify: `apps/api/config/config.go`
- Modify: `apps/api/config/config_test.go`
- Modify: `apps/web/nuxt.config.ts`

- [ ] **Step 1: Write failing environment, generator, builder, and worker tests**

Cover install/build environment allowlists, secret redaction, bounded logs, copied-package digest recheck, frozen installs with `--ignore-scripts`, forbidden lifecycle sentinel, static literal lazy imports, locale namespace generation, module-root escape rejection, peer dedupe, deterministic output, typecheck before build, preview failure logs, artifact digest/manifest, stale generation supersede checks before and after build, advisory build lock, and worker stopping at `ready` without pointer/runtime writes.

- [ ] **Step 2: Verify RED**

Run: `cd apps/api && go test ./app/Support/WebReleaseRuntime ./app/Jobs/Extensions ./bootstrap ./config`

Run: `cd apps/web && bun test tests/adminExtensionBuildGuard.test.ts`

- [ ] **Step 3: Generalize configuration with legacy fallback**

Resolve `WEB_RELEASE_ROOT`, then `THEME_RELEASE_ROOT`, then `../../storage/theme-releases`. Resolve `WEB_RELEASE_WEB_ROOT -> THEME_WEB_ROOT -> ../web`, `WEB_RELEASE_BUN_PATH -> THEME_BUN_PATH -> bun`, `WEB_RELEASE_BUILD_TIMEOUT -> THEME_BUILD_TIMEOUT -> 5m`, and `WEB_RELEASE_PREVIEW_TIMEOUT -> THEME_PREVIEW_TIMEOUT -> 30s`. Retain existing `Theme*` accessors during compatibility. Keep queue name `theme` and `JOB_QUEUE_THEME_WORKERS`.

- [ ] **Step 4: Build in an isolated release workspace**

Copy host web source without generated outputs or `node_modules`; link the host dependency directory read-only; copy each immutable extension snapshot into the release workspace; verify its digest; inspect its locked dependencies; run `bun install --frozen-lockfile --ignore-scripts` in each admin root; persist the lockfile and resolved-dependency snapshot digests; generate metadata, lazy loaders, locales, and build input; run Nuxt typecheck/build; preview; hash `.output`; write immutable `release.json`; preserve `dev-input` for local Nuxt; then transition only to `ready`.

`release.json` contains schema version, release ID, composition hash, artifact path/digest, server entry, active theme identity/layer, dev-input path, generated registry root, reload mode, and build timestamp. The worker acquires `sforum.web_release.build` through a pinned PostgreSQL advisory-lock connection before entering `resolving` and releases it after the final `ready`/failure transition.

Install commands receive only path/home/temp/proxy/registry variables. Build commands receive those plus approved public `NUXT_PUBLIC_*`, locale/site values, output/build directories, theme layer, registry root, and release identity. Explicit tests prove database URL, session secrets, Redis password, option key, signing material, and unrelated variables are absent.

- [ ] **Step 5: Add a Vite module-boundary guard**

Generate allowed plugin roots and declared dependency names. Reject plugin-source imports using `~/`, `@/`, private `#build`, absolute core paths, undeclared packages, or relative paths escaping the frontend root. Allow host peer aliases and locked dependency trees. Treat unknown slots, unused/missing component mappings, and duplicate contribution IDs as build failures.

- [ ] **Step 6: Register the new River job and retain a legacy adapter**

`WebReleaseBuildArgs.Kind()` returns `extension.web_release_build`, uses queue `theme`, disables River's one-minute timeout, and is unique by release ID. Modify the old theme worker into a compatibility adapter for previously queued legacy rows; it must enqueue or associate a Web Release and must no longer write current state or mark a release active.

- [ ] **Step 7: Verify and commit**

Run: `cd apps/api && go test ./app/Support/WebReleaseRuntime ./app/Support/ThemeRuntime ./app/Jobs/Extensions ./bootstrap ./config`

Run: `cd apps/web && bun test tests/adminExtensionBuildGuard.test.ts && bun run typecheck`

```bash
git add apps/api/app/Support/WebReleaseRuntime apps/api/app/Support/Postgres/advisory_lock.go apps/api/app/Support/Postgres/advisory_lock_test.go apps/api/app/Support/ThemeRuntime/builder.go apps/api/app/Jobs/Extensions/web_release_build.go apps/api/app/Jobs/Extensions/web_release_build_test.go apps/api/app/Jobs/Extensions/theme_activate.go apps/api/app/Jobs/Extensions/theme_activate_test.go apps/api/bootstrap/worker.go apps/api/bootstrap/worker_test.go apps/api/config/config.go apps/api/config/config_test.go apps/web/build/admin-extension-guard.ts apps/web/tests/adminExtensionBuildGuard.test.ts apps/web/nuxt.config.ts
git diff --cached --check
git commit -m "feat: build immutable web releases"
```

### Task 8: Add Supervisor Desired/Active Acknowledgements

**Prerequisite:** The background changes to `apps/web/scripts/dev-theme-lifecycle.mjs` and `apps/web/tests/devThemeLifecycle.test.ts` are committed. Re-read both from the latest `main` before editing.

**Files:**

- Create: `apps/web/scripts/web-release-contract.mjs`
- Create: `apps/web/tests/webReleaseContract.test.ts`
- Modify: `apps/web/scripts/runtime.mjs`
- Modify: `apps/web/scripts/dev-theme-runtime.mjs`
- Modify: `apps/web/scripts/dev-theme-lifecycle.mjs`
- Modify: `apps/web/tests/devThemeLifecycle.test.ts`
- Modify: `apps/web/tests/devRuntimeStartup.test.ts`
- Modify: `apps/web/tests/themeProxy.test.ts`
- Modify: `apps/web/scripts/theme-proxy.mjs`

- [ ] **Step 1: Write failing pointer and acknowledgement tests**

Cover new pointer parsing, legacy theme fallback, release manifest matching, identical Go/Node artifact digest vectors, invalid/missing artifacts, atomic JSON writes, stable readiness, successful `active.json`, candidate failure acknowledgement, old target preservation, watcher filtering to `current.json(.tmp)`, latest desired convergence, and dev selection derived from immutable `release.json` rather than a new pointer-only `layerPath`.

- [ ] **Step 2: Verify RED**

Run: `cd apps/web && bun test tests/webReleaseContract.test.ts tests/themeProxy.test.ts tests/devThemeLifecycle.test.ts tests/devRuntimeStartup.test.ts`

- [ ] **Step 3: Implement the shared file contract**

```js
export function readDesiredRelease({ releaseRoot, legacyRoot, fallback })
export async function verifyReleaseArtifact(selection)
export async function writeActiveAcknowledgement(releaseRoot, active)
export async function writeFailureAcknowledgement(releaseRoot, failure)
export function watchableReleaseFile(filename) {
  return filename === 'current.json' || filename === 'current.json.tmp'
}
```

`active.json` contains release ID, composition hash, artifact digest, server entry, theme identity, reload mode, and switched timestamp. `failures/{releaseId}.json` contains a stable reason, scrubbed message, and timestamp. All writes use temporary files plus atomic rename.

New `current.json` contains `schemaVersion: 1`, release ID, composition hash, artifact path/digest, server entry, active theme identity, reload mode, and request timestamp. Development-only layer/registry paths are read from verified `release.json`, never added as a second desired-state contract.

- [ ] **Step 4: Adapt production and development supervisors**

Production verifies manifest and full artifact digest, starts a candidate, requires consecutive healthy probes across the stabilization window, switches the proxy, then writes `active.json`. A failure writes the release-specific acknowledgement and leaves the old target serving. Development keeps one serial Nuxt process, but its selection key includes theme layer, generated registry/dev-input root, release ID, and composition hash; it acknowledges only after the new dev process is healthy.

- [ ] **Step 5: Verify and commit**

Run: `cd apps/web && bun test tests/webReleaseContract.test.ts tests/themeProxy.test.ts tests/devThemeLifecycle.test.ts tests/devRuntimeStartup.test.ts`

```bash
git add apps/web/scripts/web-release-contract.mjs apps/web/tests/webReleaseContract.test.ts apps/web/scripts/runtime.mjs apps/web/scripts/dev-theme-runtime.mjs apps/web/scripts/dev-theme-lifecycle.mjs apps/web/tests/devThemeLifecycle.test.ts apps/web/tests/devRuntimeStartup.test.ts apps/web/tests/themeProxy.test.ts apps/web/scripts/theme-proxy.mjs
git diff --cached --check
git commit -m "feat: acknowledge active web releases"
```

### Task 9: Coordinate Activation And Crash Recovery In The API

**Files:**

- Create: `apps/api/app/Support/WebReleaseCoordinator/coordinator.go`
- Create: `apps/api/app/Support/WebReleaseCoordinator/effects.go`
- Create: `apps/api/app/Support/WebReleaseCoordinator/reconcile.go`
- Create: `apps/api/app/Support/WebReleaseCoordinator/coordinator_test.go`
- Create: `apps/api/app/Support/WebReleaseCoordinator/reconcile_test.go`
- Modify: `apps/api/app/Support/Postgres/advisory_lock.go`
- Modify: `apps/api/bootstrap/app.go`
- Modify: `apps/api/bootstrap/app_test.go`
- Modify: `apps/api/app/Providers/extensions.go`

- [ ] **Step 1: Write failing checkpoint and crash-injection tests**

Test crashes before runtime preparation, after preparation, after effective commit, before pointer write, after pointer write, after supervisor switch, and before final DB transition. Test invalid/mismatched acknowledgements, failure compensation exactly once, safe-build failure leaving revocation pending, pending revocation finalization, previous release inactivation, API restart runtime reconciliation, stale desired supersede, and advisory activation lock exclusion.

- [ ] **Step 2: Verify RED**

Run: `cd apps/api && go test ./app/Support/WebReleaseCoordinator ./bootstrap`

- [ ] **Step 3: Implement the coordinator ownership boundary**

```go
type Store interface {
	NextActivation(context.Context) (extensions.WebReleaseDetail, error)
	Transition(context.Context, extensions.WebReleaseTransitionInput) error
	CommitEffect(context.Context, extensions.WebReleaseEffectInput) error
	FinalizeRevocations(context.Context, int64) error
}
type RuntimeManager interface {
	Start(context.Context, extensions.Extension) error
	Stop(context.Context, extensions.Extension) error
	Reconcile(context.Context, []extensions.Extension)
}
type PointerStore interface {
	WriteCurrent(context.Context, webruntime.CurrentRelease) error
	ReadActive(context.Context) (webruntime.ActiveRelease, error)
	ReadFailure(context.Context, int64) (webruntime.Failure, error)
	RestorePrevious(context.Context, int64) error
}
type AdvisoryLocker interface {
	WithLock(context.Context, string, func(context.Context) error) error
}
type Coordinator struct {
	store    Store
	runtime  RuntimeManager
	pointers PointerStore
	lock     AdvisoryLocker
}

func (c *Coordinator) Reconcile(ctx context.Context) error
func (c *Coordinator) Start(ctx context.Context) error
func (c *Coordinator) Stop(ctx context.Context) error
```

Only the coordinator calls plugin `Start/Stop`, changes effective extension/theme state for release-backed operations, writes `current.json`, accepts `active.json`, restores a previous pointer, compensates effects, finalizes revocations, and marks active/inactive/rolled-back states. Every phase persists a checkpoint before the next side effect.

- [ ] **Step 4: Wire one shared Extensions service and coordinator into API bootstrap**

Remove the duplicate service construction in `NewAPI`. Construct one service, one runtime manager, one coordinator, and inject the same service into the provider. Start periodic reconciliation after built-in sync/runtime reconciliation and stop it before closing runtime/database resources.

- [ ] **Step 5: Verify and commit**

Run: `cd apps/api && go test ./app/Support/WebReleaseCoordinator ./app/Providers ./bootstrap`

```bash
git add apps/api/app/Support/WebReleaseCoordinator apps/api/app/Support/Postgres/advisory_lock.go apps/api/bootstrap/app.go apps/api/bootstrap/app_test.go apps/api/app/Providers/extensions.go
git diff --cached --check
git commit -m "feat: coordinate web release activation"
```

### Task 10: Route Plugin And Theme Lifecycle Through Web Releases

**Files:**

- Modify: `apps/api/app/Models/Extensions/service.go`
- Modify: `apps/api/app/Models/Extensions/service_test.go`
- Modify: `apps/api/app/Http/Controllers/Extensions/controller.go`
- Modify: `apps/api/app/Http/Controllers/Extensions/controller_test.go`
- Modify: `apps/api/app/Models/AdminOverview/postgres_store.go`
- Create: `apps/api/app/Models/AdminOverview/postgres_store_test.go`
- Modify: `apps/web/app/utils/adminExtensions.ts`
- Modify: `apps/web/app/composables/useAdminExtensionsManager.ts`
- Modify: `apps/web/app/pages/admin/extensions/index.vue`
- Modify: `apps/web/app/pages/admin/extensions/themes.vue`
- Modify: `apps/web/tests/adminExtensions.test.ts`
- Modify: `tests/validate-theme-activation-progress.js`
- Modify: `contracts/openapi/paths/extensions.yaml`
- Modify: `contracts/openapi/schemas/extensions.yaml`

- [ ] **Step 1: Write failing operation-order tests**

Test backend-only plugins remain immediate `200`; trusted frontend enable prepares/commits backend before switch and compensates on failure; disable switches UI out before stopping backend; theme changes commit after acknowledgement; built-in default restore is queued when composition changes; old tabs receive disabled route responses; and legacy ThemeRelease views mirror linked Web Release progress during compatibility.

- [ ] **Step 2: Verify RED**

Run: `cd apps/api && go test ./app/Models/Extensions ./app/Http/Controllers/Extensions ./app/Models/AdminOverview`

- [ ] **Step 3: Return explicit operation results**

Change enable, disable, and activate mutations to return `ExtensionOperation`. Controllers choose `200` or `202` from `Queued`. Frontend DTOs consume `{extension, frontend?, webRelease?, queued}` and replace the embedded extension without raw response-shape guessing. Update the three existing OpenAPI operations to declare both status branches and `ApiResponseExtensionOperation`.

- [ ] **Step 4: Preserve existing theme UI while changing canonical storage**

Map linked Web Release states into the old ThemeRelease progress DTO until the new Releases page ships. Update AdminOverview to count canonical `web_releases` after bootstrap migration, while retaining a legacy fallback until an initial active Web Release exists.

- [ ] **Step 5: Verify and commit**

Run: `cd apps/api && go test ./app/Models/Extensions ./app/Http/Controllers/Extensions ./app/Models/AdminOverview`

Run: `cd apps/web && bun test tests/adminExtensions.test.ts && bun run typecheck`

Run: `ruby scripts/validate-openapi-refs.rb`

```bash
git add apps/api/app/Models/Extensions/service.go apps/api/app/Models/Extensions/service_test.go apps/api/app/Http/Controllers/Extensions/controller.go apps/api/app/Http/Controllers/Extensions/controller_test.go apps/api/app/Models/AdminOverview/postgres_store.go apps/api/app/Models/AdminOverview/postgres_store_test.go apps/web/app/utils/adminExtensions.ts apps/web/app/composables/useAdminExtensionsManager.ts apps/web/app/pages/admin/extensions/index.vue apps/web/app/pages/admin/extensions/themes.vue apps/web/tests/adminExtensions.test.ts tests/validate-theme-activation-progress.js contracts/openapi/paths/extensions.yaml contracts/openapi/schemas/extensions.yaml
git diff --cached --check
git commit -m "feat: unify extension web lifecycle"
```

### Task 11: Build Trusted Frontend And Web Release Admin UI

**Files:**

- Create: `apps/web/app/utils/adminWebReleases.ts`
- Create: `apps/web/app/composables/useAdminFrontendTrust.ts`
- Create: `apps/web/app/composables/useAdminWebReleases.ts`
- Create: `apps/web/app/components/SFAdminFrontendTrustPanel.vue`
- Create: `apps/web/app/pages/admin/extensions/releases.vue`
- Create: `apps/web/tests/adminWebReleases.test.ts`
- Modify: `apps/web/app/pages/admin/extensions/plugins.vue`
- Modify: `apps/web/app/config/adminModules.ts`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`
- Modify: `tests/validate-admin-framework.ts`

- [ ] **Step 1: Write failing pure helper and static page tests**

Cover all release/final states, progress, eligible retry/rollback, server pagination, non-overlapping polling, polling cleanup, super-admin action visibility distinct from `extension.manage`, 10-second success Toasts, persistent error Toasts, inline blocking failures, and bilingual key parity.

- [ ] **Step 2: Verify RED**

Run: `cd apps/web && bun test tests/adminWebReleases.test.ts tests/adminExtensions.test.ts`

- [ ] **Step 3: Implement the Trusted Frontend panel**

Show declaration/trust/invalidated/pending/build/active/failure states, digest, root, component map, API version, slots, dependency summary, and current release. The grant modal displays author/source/version/digest/module paths/dependencies/disabled scripts and the same-origin authority warning. Only `roleKeys.includes('super_admin')` reveals grant/revoke/restore actions; API policy remains authoritative.

- [ ] **Step 4: Implement the server-paginated Web Releases page**

Register `/extensions/releases` under the existing Extensions folder with `extension.manage`. Show composition, trigger, state, duration, previous release, event timeline, cleaned build log, retry, and eligible rollback. Use a recursive `setTimeout` or explicit in-flight guard; poll only non-final releases and clear timers on unmount.

- [ ] **Step 5: Apply feedback rules exactly**

Queued, successful, retry, rollback, and restore actions use theme-aware Toasts with `duration: 10000`. Errors use `duration: 0`. Trust blockers and build failures remain inline with retry/navigation actions and are never replaced by transient Toasts.

- [ ] **Step 6: Verify and commit**

Run: `cd apps/web && bun test tests/adminWebReleases.test.ts tests/adminExtensions.test.ts`

Run: `cd apps/web && bun run typecheck`

Run: `bun tests/validate-admin-framework.ts`

```bash
git add apps/web/app/utils/adminWebReleases.ts apps/web/app/composables/useAdminFrontendTrust.ts apps/web/app/composables/useAdminWebReleases.ts apps/web/app/components/SFAdminFrontendTrustPanel.vue apps/web/app/pages/admin/extensions/releases.vue apps/web/tests/adminWebReleases.test.ts apps/web/app/pages/admin/extensions/plugins.vue apps/web/app/config/adminModules.ts apps/web/i18n/locales/zh-CN.json apps/web/i18n/locales/en-US.json tests/validate-admin-framework.ts
git diff --cached --check
git commit -m "feat: manage trusted frontend releases"
```

### Task 12: Detect Release Changes In Existing Admin Tabs

**Files:**

- Create: `apps/web/app/composables/useAdminReleaseMonitor.ts`
- Create: `apps/web/app/components/SFAdminReleaseNotice.vue`
- Create: `apps/web/server/routes/__sforum/admin-release.get.ts`
- Modify: `apps/web/app/layouts/admin.vue`
- Modify: `apps/web/app/runtime/admin-extensions/types.ts`
- Modify: `apps/web/tests/adminExtensionRegistry.test.ts`
- Modify: `apps/web/tests/adminRouteRendering.test.ts`

- [ ] **Step 1: Write failing monitor tests**

Test embedded/current equality, prompt change, force change, no-store response, visibility-triggered recheck, serialized requests, interval cleanup, and no polling on public layouts.

- [ ] **Step 2: Verify RED**

Run: `cd apps/web && bun test tests/adminExtensionRegistry.test.ts tests/adminRouteRendering.test.ts`

- [ ] **Step 3: Implement the embedded release endpoint and admin-only monitor**

The built artifact embeds `{ releaseId, reloadMode }`. `GET /__sforum/admin-release` returns that object with `Cache-Control: no-store`. The old admin bundle periodically fetches through the supervisor; a changed `prompt` release shows a persistent reload notice, while a changed `force` release immediately calls `window.location.reload()` after the safe release is active.

- [ ] **Step 4: Verify and commit**

Run: `cd apps/web && bun test tests/adminExtensionRegistry.test.ts tests/adminRouteRendering.test.ts && bun run typecheck`

```bash
git add apps/web/app/composables/useAdminReleaseMonitor.ts apps/web/app/components/SFAdminReleaseNotice.vue apps/web/server/routes/__sforum/admin-release.get.ts apps/web/app/layouts/admin.vue apps/web/app/runtime/admin-extensions/types.ts apps/web/tests/adminExtensionRegistry.test.ts apps/web/tests/adminRouteRendering.test.ts
git diff --cached --check
git commit -m "feat: reload stale admin releases"
```

### Task 13: Migrate Deployment Configuration And Retention

**Files:**

- Modify: `.env.example`
- Modify: `.env.production.example`
- Modify: `compose.yaml`
- Modify: `compose.dev.yaml`
- Modify: `apps/api/Dockerfile`
- Modify: `apps/web/Dockerfile`
- Modify: `deploy/volumes/README.md`
- Create: `apps/api/app/Jobs/Extensions/web_release_cleanup.go`
- Create: `apps/api/app/Jobs/Extensions/web_release_cleanup_test.go`
- Modify: `apps/api/bootstrap/worker.go`
- Modify: `apps/api/bootstrap/app.go`
- Modify: `apps/web/package.json`
- Modify: `tests/validate-theme-runtime.js`

- [ ] **Step 1: Write failing config, cleanup, and runtime assertions**

Test `WEB_RELEASE_ROOT` precedence/fallback, API/worker/web shared path, web artifact-only access, worker immutable package access, initial release creation, active plus rollback-target plus five-success retention, seven-day failed/superseded cleanup, thirty-day log cleanup, and transition metadata preservation.

- [ ] **Step 2: Verify RED**

Run: `cd apps/api && go test ./app/Jobs/Extensions ./bootstrap ./config`

Run: `node tests/validate-theme-runtime.js`

- [ ] **Step 3: Preserve the physical volume while changing the canonical name**

Expose `WEB_RELEASE_ROOT=/var/lib/sforum/theme-releases` to API, worker, and web; keep the existing `theme_releases` named volume and old environment fallbacks. API/worker mount immutable extension packages; web mounts built releases only. Document `current.json`, `active.json`, failure acknowledgements, artifacts, and dev-input contents.

- [ ] **Step 4: Use a Node production supervisor**

Keep the existing `oven/bun:1.3-alpine` production base and its current `sforum` user ownership, install Alpine `nodejs`, and run `node scripts/runtime.mjs`; this provides Node-compatible HTTP upgrade semantics without changing the shared volume UID/GID. Nitro artifacts continue to be produced by Bun and run under Node.

- [ ] **Step 5: Bootstrap and clean releases safely**

If no canonical active/pending release exists, enqueue the current theme plus empty trusted-plugin composition while the legacy/fallback site remains live. Register a daily cleanup job that never deletes the active release, its eligible rollback target, referenced immutable snapshots, or metadata/events.

- [ ] **Step 6: Verify Compose and commit**

Run: `docker compose --env-file .env.example -f compose.yaml -f compose.dev.yaml config --quiet`

Run: `docker compose --env-file .env.production.example -f compose.yaml -f compose.prod.yaml config --quiet`

```bash
git add .env.example .env.production.example compose.yaml compose.dev.yaml apps/api/Dockerfile apps/web/Dockerfile deploy/volumes/README.md apps/api/app/Jobs/Extensions/web_release_cleanup.go apps/api/app/Jobs/Extensions/web_release_cleanup_test.go apps/api/bootstrap/worker.go apps/api/bootstrap/app.go apps/web/package.json tests/validate-theme-runtime.js
git diff --cached --check
git commit -m "ops: deploy unified web release runtime"
```

### Task 14: Add End-To-End Fixture, Regression Gates, And Documentation

**Files:**

- Create: `tests/fixtures/extensions/trusted-admin-plugin/sforum.extension.json`
- Create: `tests/fixtures/extensions/trusted-admin-plugin/frontend/admin/package.json`
- Create: `tests/fixtures/extensions/trusted-admin-plugin/frontend/admin/bun.lock`
- Create: `tests/fixtures/extensions/trusted-admin-plugin/frontend/admin/components/HealthyPanel.vue`
- Create: `tests/fixtures/extensions/trusted-admin-plugin/frontend/admin/components/ThrowOnRender.vue`
- Create: `tests/fixtures/extensions/trusted-admin-plugin/frontend/admin/components/ThrowInSetup.vue`
- Create: `tests/fixtures/extensions/trusted-admin-plugin/frontend/admin/components/ThrowDescendant.vue`
- Create: `tests/fixtures/extensions/trusted-admin-plugin/frontend/admin/components/BrokenChild.vue`
- Create: `tests/fixtures/extensions/trusted-admin-plugin/frontend/admin/components/LazyLoadFailure.vue`
- Create: `tests/fixtures/extensions/trusted-admin-plugin/frontend/admin/locales/zh-CN.json`
- Create: `tests/fixtures/extensions/trusted-admin-plugin/frontend/admin/locales/en-US.json`
- Create: `tests/fixtures/extensions/trusted-admin-plugin/frontend/admin/scripts/postinstall.mjs`
- Create: `tests/fixtures/npm/sforum-fixture-dependency/package.json`
- Create: `tests/fixtures/npm/sforum-fixture-dependency/index.js`
- Create: `tests/helpers/local-package-registry.mjs`
- Create: `apps/api/app/Support/WebReleaseRuntime/builder_integration_test.go`
- Create: `tests/validate-trusted-admin-runtime.js`
- Create: `docs/extensions/trusted-admin-components.md`
- Create: `knowledge/sessions/2026-07-10-trusted-admin-plugin-runtime.md`
- Modify: `scripts/test.sh`
- Modify: `knowledge/index.md`
- Modify: `knowledge/modules/extensions.md`
- Modify: `knowledge/modules/frontend.md`

- [ ] **Step 1: Build the shared fixture**

The fixture declares only the build-only `admin.test.fixture` slot and includes one normal component, one throw-on-render component, one lazy-load failure, bilingual locale JSON, one locked pure-JS dependency, and an install-script sentinel that would create a file if lifecycle scripts ran. The registry helper binds `127.0.0.1:4873` and the committed lock points only there; tests prove the sentinel file is absent.

- [ ] **Step 2: Add repository validation**

`tests/validate-trusted-admin-runtime.js` checks the SDK package, generated fallback modules, client-only registry, Web Release endpoint, supervisor acknowledgement files, API routes, bilingual locale keys, and the absence of production `admin.jobs.*` registration. Wire it explicitly into `scripts/test.sh`; that script does not auto-discover validators.

- [ ] **Step 3: Run the focused full-stack gates**

```bash
cd apps/api && go test -count=1 ./...
ruby scripts/validate-openapi-refs.rb
cd apps/web && bun test
cd apps/web && bun run typecheck
cd apps/web && bun run build
node tests/validate-trusted-admin-runtime.js
./scripts/test.sh
```

Expected: all commands exit `0`. If a pre-existing concurrent change fails a gate, record the exact failing test and verify it independently before changing any unrelated file.

- [ ] **Step 4: Run integration and browser verification**

Use an ephemeral database and the local registry helper. Verify frozen install without public network, disabled lifecycle scripts, host peer dedupe, build/typecheck/preview, shared artifact visibility, supervisor acknowledgement, API reconciliation, and rollback. In a running dev stack, verify desktop and mobile admin views in light/dark mode; normal component rendering; isolated load/render/setup/descendant failure; third-failure quarantine; prompt/force reload; grant/revoke permissions; polling cleanup; and no overlap or text clipping.

- [ ] **Step 5: Update author and project memory**

Document manifest fields, required locales, SDK imports, slot ownership, dependency limits, full-trust warning, grant invalidation, build diagnostics, and the fact that Jobs production slots arrive in the separate queue-monitoring project. Update module status and add the session handoff with changed decisions, verification, next work, and open questions.

- [ ] **Step 6: Commit the completed runtime**

```bash
git add tests/fixtures/extensions/trusted-admin-plugin tests/fixtures/npm/sforum-fixture-dependency tests/helpers/local-package-registry.mjs apps/api/app/Support/WebReleaseRuntime/builder_integration_test.go tests/validate-trusted-admin-runtime.js docs/extensions/trusted-admin-components.md scripts/test.sh knowledge/index.md knowledge/modules/extensions.md knowledge/modules/frontend.md knowledge/sessions/2026-07-10-trusted-admin-plugin-runtime.md
git diff --cached --check
git commit -m "test: verify trusted admin plugin runtime"
```
