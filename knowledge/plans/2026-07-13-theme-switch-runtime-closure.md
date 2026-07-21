# Theme Switch Runtime Closure — Implementation Task Book

Status: **superseded** — do not implement  
Date: 2026-07-13  
Superseded by: synchronous Page Registry activation, removal of runtime theme
release/supervisor paths, and Trusted Plugin/Theme Platform V3 theme work
(`plans/2026-07-13-trusted-plugin-theme-platform-v3.md`,
`decisions/2026-07-13-runtime-page-registry-themes.md`,
`decisions/2026-07-13-remove-legacy-web-release.md`).

Audience: historical only

Related accepted architecture:

- `knowledge/decisions/2026-07-13-runtime-page-registry-themes.md`
- `knowledge/plans/2026-07-13-runtime-page-registry-themes.md`
- `knowledge/sessions/archive/2026-07/2026-07-13-runtime-page-registry-round2-remediation.md`

## Review Gate

This task book is documentation only. Do **not** implement any phase until the
user explicitly approves this plan, including the four product choices under
“Decisions For Review”.

After approval, execute phases in order. Do not combine all work into one large
change: every phase has a separate test and rollback boundary.

## Execution Interlock With Concurrent Work

Codex task `019f58a7-127d-7d52-9ef1-013893d9eb27` was still active during the
technical review. It is removing legacy Web Release code and changing extension
settings/frontend contracts in the same checkout. P0 must not start until that
task is idle/completed and its final handoff and working-tree diff have been
read.

Before any production-code or test edit:

1. Re-run `git status --short --branch` and capture the final diff/name-status.
2. Re-read the concurrent task's final handoff and confirm its final synchronous
   extension lifecycle response shapes.
3. Re-check all interfaces and files listed under “Concurrent overlap and
   mandatory re-check” below. Do not infer final contracts from the current
   intermediate tree.
4. If overlapping changes continue to appear, stop the affected phase. Do not
   merge, revert, format, or repair the other task's work.
5. Establish a clean ownership boundary for commits before staging anything;
   this plan does not authorize committing another task's changes.
6. Re-list database migrations after the concurrent task completes and allocate
   a new unused sequence after its final migration; do not reuse its current
   `202607130005` slot.

### Concurrent overlap and mandatory re-check

- Extension HTTP lifecycle:
  `apps/api/app/Http/Controllers/Extensions/{controller.go,controller_test.go,frontend.go,routes.go}`
- Theme selection and Page Registry bridge:
  `apps/api/app/Models/Extensions/{service.go,service_test.go,store.go,postgres_store.go,page_registry_adapter.go,types.go}`
- Page APIs and bindings:
  `apps/api/app/Http/Controllers/Pages/{controller.go,controller_test.go}` and
  `apps/api/app/Support/Pages/{registry.go,postgres_store.go,memory_store.go,extension_bridge.go}`
- Manifest/package contracts:
  `apps/api/app/Support/ExtensionManifest/**`
- Bootstrap/worker/provider assembly:
  `apps/api/bootstrap/{app.go,app_test.go,worker.go}` and
  `apps/api/app/Providers/{extensions.go,jobs.go}`
- Admin extension UI and shared types:
  `apps/web/app/composables/useAdminExtensionsManager.ts`,
  `apps/web/app/utils/adminExtensions.ts`, extension admin pages/components,
  and both locale catalogs
- Theme SSR/runtime/cache:
  `apps/web/nuxt.config.ts`, `apps/web/package.json`, `apps/web/app/app.vue`,
  public/auth/admin layouts, `useActiveThemeSkin.ts`,
  `useActiveThemeSettings.ts`, `SFPageOutlet.vue`, and both dynamic Page
  Registry route pages
- Built-in theme packages:
  `extensions/builtin/themes/{sforum-default,sforum-nocturne}/**`
- OpenAPI:
  `contracts/openapi.yaml`, `contracts/openapi/paths/{extensions,pages}.yaml`,
  and `contracts/openapi/schemas/{extensions,pages}.yaml`
- Knowledge:
  `knowledge/index.md`, `knowledge/modules/{extensions,frontend}.md`, and the
  concurrent task's decision/plan/handoff documents

Deleted Web Release types, jobs, routes, permissions, options, or UI branches
must stay deleted. This program may adapt to the final buildless settings/admin
frontend contract, but must not recreate a compatibility wrapper around the
removed release lifecycle.

## Technical Review Invariants

The following are required for approval; they are not optional implementation
details:

1. **Preview-bound mutation.** A theme activation by id alone is insufficient.
   The mutation must be conditional on the target version/package digest and the
   active-theme state shown in preview. A stale preview returns conflict before
   any DB, Registry, binding, event, or audit mutation.
2. **Exactly one active theme and one sync owner.** Concurrent API calls and
   built-in sync must not leave two enabled themes. The API host owns built-in
   theme catalog sync + Registry restore; a standalone worker must not update
   theme rows, artifact digests, or selection behind the API's in-memory
   Registry. Serialize relevant Postgres theme writes and enforce selection at
   the database layer; a process-local mutex is not sufficient.
3. **Crash convergence.** Theme selection, in-memory Registry replacement, and
   provider-binding cleanup cannot be one transaction. Startup restore must
   reconcile stale bindings from inactive/invalid themes so a crash between
   steps converges on the selected DB theme and core fallback.
4. **Shared-cache coherence.** Clearing browser `useAsyncData` does not purge
   Nitro SWR or CDN `s-maxage` HTML. Theme-sensitive SSR must not be shared-cached
   until a theme-revision-aware cache key or reliable purge exists. The active
   skin descriptor is non-cacheable; digest assets remain immutable.
5. **Exact-artifact approvals.** A theme may remain selected after its package
   changes, but bindings for an old version/digest/contract must be revoked and
   explicitly approved again.
6. **Single-node runtime boundary.** The current Page Registry is in-memory.
   P0 must confirm that the supported deployment for this program is one API
   Registry process. If multi-replica API runtime is already supported, durable
   cross-node revision propagation becomes a blocking prerequisite rather than
   an implicit promise of this task.

## Problem Statement And Evidence

The runtime-theme architecture is present, but activation does not currently
converge all persisted, in-memory, and rendered state.

Observed on 2026-07-13:

1. `sforum.nocturne-theme` activation succeeded and emitted
   `theme_activated` events.
2. While active, `/site/active-theme/skin` returned Nocturne’s digest-bound CSS
   and token URLs, proving L0 activation was real.
3. `page_provider_bindings` contained no `forum.home` approval, so
   `/pages/resolve?id=forum.home` returned `provider=core`; the Nocturne L1 hero
   could not render.
4. Subsequent API/Air restarts emitted `builtin_synced` and switched the active
   theme back to `sforum.default-theme`.
5. The cause is deterministic: `SyncBuiltins` always calls
   `EnsureDefaultThemeActive`, and that helper activates the default whenever
   the current active theme is not already the default.
6. Admin activation refreshes the extension list only. It does not preview page
   impact, request selected page approvals, refresh active-theme settings/skin,
   or invalidate Page Registry async data.
7. The working tree is concurrently removing the legacy Web Release lifecycle.
   In the current intermediate state the Go activate controller returns a plain
   `Extension`, while the Nuxt manager still expects `ExtensionOperation`.
   Execution must first reconcile this response contract after the concurrent
   cleanup settles; the task book must not restore deleted Web Release coupling.
8. The preview endpoint currently reports version/digest, but the activation
   endpoint accepts only the extension id. An upgrade between preview and
   confirmation can therefore activate unreviewed bytes and page impacts.
9. `PostgresStore.ActivateTheme` has no cross-request serialization or database
   uniqueness constraint for enabled themes. Concurrent activations can violate
   the exactly-one-active-theme invariant.
10. Public routes currently use Nitro SWR or CDN `s-maxage` caching. Browser
    async-data invalidation cannot make cached SSR HTML, L1 output, or old skin
    links immediately current.
11. Both protected default and Nocturne currently declare a `forum.home`
    replace. “Preselect every replace” and “default activation restores core”
    are contradictory unless the protected-default recovery behavior is stated
    explicitly.

The user-visible result is therefore accurate but incomplete: some L0 visual
tokens may change, while L1 page chrome stays core; after an API restart even
the selected active theme is lost.

## Desired Outcome

When an operator activates a valid runtime theme:

1. The chosen theme remains active across API restart, Air rebuild, worker
   startup, and built-in resync, with exactly one active theme even under
   concurrent activation.
2. Before activation, the UI clearly lists L0 skin effects and every L1 page
   add/replace impact for the exact package digest that will be activated.
3. A `super_admin` can explicitly apply the theme’s recommended page replaces
   in the same guided flow without visiting a second screen.
4. Non-super-admin theme managers may activate L0, but the UI must clearly state
   that protected core-page replaces were not approved.
5. Theme CSS, theme settings, and page resolution converge immediately after a
   successful operation without rebuilding Nuxt.
6. Switching back to the default theme restores core page providers and does
   not leave stale bindings to the previous theme.
7. Failure is honest and recoverable: core pages remain available, blocking
   errors stay visible, and the UI never claims a complete switch after a
   partial result.
8. New SSR requests do not serve an old theme through Nitro/CDN shared caches;
   the initiating browser converges without a hard reload.
9. Package updates keep a valid selected theme usable at L0, but stale L1
   approvals never silently carry to different bytes.

## Decisions For Review

The recommended choices below require user approval before implementation.

### D1 — Preserve the selected theme; validate it in API restore

**Recommended:** API-host built-in sync selects the default only when no active
theme row exists. It preserves any selected theme row without attempting
package preflight. Package/template/CSS validity remains the responsibility of
the API host's `RestoreActiveThemeRegistry`, which preflights and falls back to
the protected default when the selected package is invalid. A standalone worker
does not sync or mutate theme rows/artifact identity/selection; it consumes the
settled DB state. If it still needs plugin catalog sync, split that operation so
theme writes remain API-owned.

This preserves both uploaded themes and alternate built-in themes such as
Nocturne. It removes the obsolete assumption that only the protected default
may survive startup. All selection writes are serialized and DB-enforced so
there is exactly one enabled theme. If built-in sync changes the selected
theme's package digest, the theme stays selected only if preflight succeeds;
old digest-bound L1 bindings are removed and require reapproval.

### D2 — Keep page replacement approval explicit

**Recommended:** do not silently auto-approve a theme’s core-page replaces.
The activation modal lists each replace and, for `super_admin`, preselects all
final-state-valid replaces as the recommended path. There is no new manifest
“recommended” flag in this task. Confirmation is bound to the target
version/digest and the active-theme state displayed by preview, and requires an
explicit final button: “激活主题并应用已选页面布局”.

The protected default is the recovery path: its recommended selection is core
fallback, so its declared `forum.home` replace is visible but not preselected.
An operator may still explicitly select it. Switching away permanently removes
the previous theme's provider bindings; switching back requires a new explicit
approval, except that exact valid bindings are retained for a same-theme,
same-version, same-digest reactivation.

The modal is a desired-state editor for `super_admin`, not merely a list of new
approvals. It shows each page's current and predicted provider. Deselecting an
exact binding already owned by the target theme explicitly calls
`restore-core`; a valid provider owned by an enabled plugin is retained unless
the operator selects the target theme to replace it. Non-super-admin users see
these protected states read-only.

This remains consistent with the accepted digest/version/contract-bound
approval model while making the beginner-friendly path obvious.

### D3 — Keep L0 active if a later page mutation fails

**Recommended:** activation and approvals reuse the existing endpoints in a
deterministic sequence. Theme selection + Registry replacement + old-binding
cleanup is the activation boundary and must compensate or fail as one operation.
Only after that boundary succeeds may selected page approvals begin. If one
approval later fails, keep the valid L0 theme active, retain any approvals that
already succeeded, keep failed/unselected pages on their predicted valid
provider or core fallback, and show a per-page persistent partial-success alert
with retry/link to Page Registry.

Do not automatically roll the whole theme back: L0 is safe and usable, core is
the designed L1 fallback, and a hidden compensating rollback would be more
surprising than an explicit partial state.

A stale-preview conflict, activation preflight failure, active-theme CAS
conflict, or old-binding cleanup failure is **not** partial success: it must not
begin approvals. Invalid-theme startup fallback is asymmetric: never roll back
to an invalid theme merely because stale-binding cleanup fails; keep the safe
default/core runtime, record a diagnostic, and retry reconciliation.

### D4 — Stabilize the current L0/L1 contract; do not expand L2 in this task

**Recommended:** this program closes switching and rendering lifecycle gaps.
It does not enable arbitrary JavaScript widgets, add new executable theme
surfaces, or redesign the host island catalog. L2 remains disabled. Correctness
does include removing shared SSR caching from theme-sensitive public/auth routes
until a later reviewed theme-revision cache design exists; vendor-specific CDN
purge infrastructure is not added here. A separate reviewed program may later
split the monolithic `sf-home-page` island, add structural islands, or restore
shared caching with revision-aware keys.

## Scope

### In scope

- Active-theme persistence across bootstrap and built-in sync
- API-owned built-in theme sync; standalone worker read-only theme startup
- Serialized/DB-enforced exactly-one active-theme selection
- Invalid active-theme fallback to protected default
- Old/stale-theme Page Registry binding cleanup and startup crash reconciliation
- Activation preview and explicit selected approvals
- Preview-bound activation preconditions for target digest and current theme
- Canonical synchronous theme-activation response typing after Web Release removal
- Super-admin and non-super-admin UX
- SSR-safe/reactive L0 stylesheet ownership
- Active theme settings and page-resolve invalidation after switch
- Theme-sensitive Nitro/CDN HTML cache correctness
- Clear Toast/alert semantics and appearance-preset distinction
- API/OpenAPI/frontend typings, unit/integration/browser tests, knowledge notes

### Out of scope

- L2 runtime widgets or arbitrary uploaded JavaScript
- Runtime compilation of Vue SFCs
- Core API/security route replacement
- Multi-active themes
- Theme marketplace/install flow redesign
- New provider slots, events, or unrelated extension refactors
- A broad public-page redesign or new Nocturne visual direction
- Vendor-specific CDN purge APIs
- Live push into unrelated already-open browser tabs/sessions; they converge on
  their next navigation/request, while the initiating SPA and new SSR requests
  converge immediately
- Multi-replica Page Registry propagation unless P0 finds multi-replica API is
  already a supported deployment contract

## Library And Framework Survey

No new dependency is expected.

- Use existing Go extension/page services and Postgres stores.
- Use a Postgres transaction lock/advisory lock plus a database uniqueness
  invariant for theme selection; do not use a process-local mutex as authority.
- Use the existing `bluemonday` L1 sanitization and Page Registry contracts.
- Use Nuxt-native `useAsyncData`, `useHead`, `refreshNuxtData`, and
  `clearNuxtData` for runtime convergence.
- Use Nuxt/Nitro route rules and HTTP cache headers for correctness. Do not add
  an application cache-purge framework in this program.
- Reuse the existing API connection modal, Toast conventions, permission
  helpers, audit writer, and admin Page Registry approval endpoint.

Do not add a state-management library, CSS loader, event bus package, or custom
retry framework for this work.

## Target Lifecycle

```text
Admin clicks Activate
        │
        ▼
GET activation preview
  - L0 skin identity
  - L1 add/replace impacts
  - approval/conflict/access notes
  - target version + package digest
  - current active theme id/revision
        │
        ▼
Explicit confirmation
        │
        ▼
POST theme activate
  - require previewed target digest/current-theme state
  - stale state -> 409, no mutation
  - preflight package
  - serialize and switch DB active theme (exactly one)
  - atomically replace in-memory contributions
  - clear bindings owned by previous theme
  - compensate and reconcile to DB truth on failure
        │
        ├── selected replace approvals (super_admin only)
        │
        ▼
Refresh runtime state
  - active skin
  - active theme settings
  - replace and add-page async data
  - theme-sensitive SSR cache policy
        │
        ▼
Public SSR/client render
  - L0 stylesheet links
  - approved L1 templates
  - retained valid plugin provider or core fallback for unapproved/failed pages
```

## Phase Overview

| Phase | Name | Exit criterion |
| --- | --- | --- |
| P0 | Final-state audit and regression harness | Concurrent work is settled; lifecycle, cache, concurrency, and crash baselines are captured |
| P1 | Backend lifecycle convergence | Exactly one valid active theme survives restart; stale bindings converge after switch/crash |
| P2 | Preview-bound activation contract | Backend/controller/OpenAPI reject stale previews; the old id-only UI action is safely interlocked |
| P3 | Render and cache convergence | SSR/client CSS/settings/all page resolves refresh without rebuild, hard reload, or stale shared HTML |
| P4 | Guided activation and operator acceptance | Preview/confirm uses P2+P3; partial state and Nocturne/default behavior are understandable |
| P5 | Full gates and knowledge handoff | All automated gates and browser scenarios pass; documentation reflects final behavior |

## P0 — Settle Concurrent Work And Freeze Regressions

**Goal:** Re-audit the settled post-Web-Release tree and encode the observed
failures before production behavior changes.

### Tasks

- [ ] T0.0 Satisfy the execution interlock: confirm concurrent task
      `019f58a7-127d-7d52-9ef1-013893d9eb27` is idle/completed, read its final
      handoff, snapshot the working-tree diff, and re-check every mandatory
      overlap listed above. Do not start tests while those interfaces move.
- [ ] T0.1 Confirm the supported deployment boundary (single API Registry
      process vs multi-replica). If multi-replica is supported, stop and add a
      durable cross-node theme revision/Registry reload design before P1.
- [ ] T0.2 Confirm the final synchronous activate request/response and the
      removal of release types/routes/permissions/options. Do not resurrect
      deleted release types merely to satisfy stale frontend typing.
- [ ] T0.3 Inventory every theme-sensitive public/auth route rule and every
      async-data key (`site-active-theme-*`, replace resolves, both dynamic add
      route families). Record which SSR responses currently carry SWR or
      `s-maxage`.
- [ ] T0.4 Add a service test where an alternate built-in theme is active,
      `SyncBuiltins` runs, and that theme must remain active.
- [ ] T0.5 Add the same preservation test for a valid uploaded runtime theme.
- [ ] T0.6 Preserve the no-active-theme case: built-in sync/bootstrap selects
      `sforum.default-theme`.
- [ ] T0.7 Add bootstrap/restore coverage: invalid active theme package falls
      back to default and registers default contributions.
- [ ] T0.8 Add Page Registry coverage for bindings owned by a previous/inactive
      theme, including a simulated crash after DB selection but before cleanup.
- [ ] T0.9 Add a real-Postgres concurrency regression proving two simultaneous
      activations cannot leave more than one enabled theme; inventory/repair any
      duplicate enabled-theme rows before adding the DB invariant.
- [ ] T0.10 Add contract characterization for the settled synchronous activate
      response and centralize the frontend activation entry points. Do not add
      temporary assertions that encode the known missing preview/refresh UX.
- [ ] T0.11 Add tests proving an unchanged same-theme artifact preserves an
      exact binding while a version/digest/contract change invalidates it.
- [ ] T0.12 Record exact baseline commands, failures, route-cache inventory,
      async-data key inventory, and the concurrent-task final revision in the
      session handoff before production edits.

### Verification

- Focused Go service, Postgres store, bootstrap, and Page Registry packages
- Focused Bun tests for extension manager and theme runtime composables
- Static route-rule/async-data inventory assertions

### Rollback

Tests/docs-only phase; revert without runtime impact. Do not revert the
concurrent task's settled changes.

## P1 — Backend Active Theme And Binding Convergence

**Goal:** Make bootstrap preserve valid selections and make theme transitions
leave persistent Page Registry state honest.

### Tasks

- [ ] T1.1 Replace the misleading `EnsureDefaultThemeActive` sync semantics with a
      plainly named helper such as `EnsureThemeSelected`:
  - return any existing active theme unchanged;
  - activate the protected default only for `ErrExtensionNotFound`;
  - propagate other store errors;
  - do not perform package preflight in built-in sync.
- [ ] T1.2 Keep `RestoreActiveThemeRegistry` as the authoritative package/L0/L1
      preflight and invalid-theme fallback path.
- [ ] T1.3 Make the API host the sole owner of built-in theme catalog sync and
      Registry restore. Standalone worker startup must not call a path that
      changes theme rows, package version/digest, or selection. If plugin sync is
      still required there, split it from theme sync. Embedded worker startup
      reuses the API's already-settled extension state.
- [ ] T1.4 Serialize every Postgres transaction that can change theme selection
      or an active/target theme's artifact identity, and add a DB invariant
      permitting at most one `type=theme,status=enabled` row. The migration must
      deterministically normalize any pre-existing duplicate to the same winner
      used by `ActiveTheme`, then create the invariant.
- [ ] T1.5 Extend the Page Registry store with one-statement atomic binding
      cleanup by extension (Postgres + memory store), returning enough deleted
      binding metadata for diagnostics/audit.
- [ ] T1.6 After a successful switch to a different theme, clear bindings owned
      by the previous theme before returning activation success.
- [ ] T1.7 Make compensation explicit: if Registry replacement or old-binding
      cleanup fails after DB selection, serialize rollback, re-read the DB
      winner, and rebuild in-memory theme contributions to that truth. Return a
      diagnosable failure if compensation is incomplete; do not silently rely
      on the current best-effort rollback helper.
- [ ] T1.8 On every API startup restore, reconcile theme-owned bindings against
      the selected active theme and exact registered version/digest/contract.
      Remove bindings owned by inactive/missing/invalid themes. This is the
      crash-recovery path for interruption between activation steps.
- [ ] T1.9 When invalid-theme bootstrap fallback activates the default, attempt
      the same cleanup. If cleanup fails, keep default/core active, record the
      failure, and retry on the next reconciliation; never reactivate the
      invalid package.
- [ ] T1.10 Preserve same-theme reactivation only for an exact valid binding.
      Package version/digest/contract changes keep L0 selected after successful
      preflight but remove stale L1 bindings for explicit reapproval.
- [ ] T1.11 Add automatic binding-revocation diagnostics/audit containing the
      previous theme id, page ids, artifact identity, reason, and count without
      forging a human `restore-core` action.

### Required tests

- Alternate built-in survives `SyncBuiltins`
- Uploaded runtime theme survives `SyncBuiltins`
- Missing selection activates default
- Invalid package falls back to default
- Concurrent activations leave exactly one enabled theme
- Standalone worker startup before/after API cannot mutate theme rows or selection
- Restart restores active theme contributions and approved provider
- Switching Nocturne → default clears Nocturne bindings
- Crash after DB switch converges on restart and removes inactive-theme bindings
- Switching theme with binding-store failure restores one DB winner and matching Registry
- Same-theme exact artifact keeps its valid binding
- Same theme id with changed digest keeps valid L0 but revokes stale L1 binding
- Invalid fallback cleanup failure keeps default/core available and is retried

### Rollback

- Revert P1 code commits. Apply the migration's explicit down path to remove the
  uniqueness index only when rollback is intentionally requested; rows disabled
  while normalizing duplicates are not silently re-enabled.
- Existing core fallback still protects rendering if bindings become offline.
- This phase now includes a schema migration because exactly-one active theme
  must be database-enforced, not merely process-conventional.

## P2 — Preview-Bound Activation Contract

**Goal:** Stabilize the backend/controller/OpenAPI contract before exposing the
guided frontend flow.

### Tasks

- [ ] T2.0 Normalize theme activation to one synchronous, non-Web-Release
      response contract across Go controller, OpenAPI, and Nuxt after the
      concurrent cleanup settles. Prefer a plain activated extension response;
      the request must carry `expectedVersion`, `expectedPackageDigest`, and the
      previewed active-theme id/version/digest tuple (or explicit no-active
      state). Remove stale
      `queued`/`webRelease` theme branches; do not reintroduce them.
- [ ] T2.1 Add typed frontend models for activation preview impacts; remove
      `any`-shaped candidate handling before P4 consumes the contract.
- [ ] T2.2 Add the missing OpenAPI path/schema for
      `GET /admin/pages/activate-preview/{extensionId}` including permission and
      error responses. The response must include current active theme identity,
      target version/digest, L0 skin metadata, typed L1 impacts, per-impact
      approval eligibility, current/predicted provider, final-state conflicts,
      and blocking preflight reasons.
- [ ] T2.3 Move preview construction behind a service/shared validator so it
      uses the same package-content root, theme type check, final-state route
      conflict rules, and preflight semantics as activation. Exclude the theme
      being replaced from false conflict reporting.
- [ ] T2.4 Under the same Postgres theme lock used by P1, lock/read the target
      artifact and current active tuple, compare both to the request, and return
      HTTP 409 before mutation on mismatch. Theme install/upgrade/builtin-sync
      writers must participate in that lock so the comparison cannot race a
      package update.
- [ ] T2.5 Keep preview available to the existing extension-view/theme-manage
      authorities, activation to theme managers, and replace/restore to
      `super_admin`. Add allowed/denied coverage at each API boundary.
- [ ] T2.6 In the same change that makes preview preconditions required, disable
      the old id-only Activate UI action with concise non-error guidance. Do not
      leave a button that sends an empty body or silently previews on the
      operator's behalf. P4 replaces this temporary safe interlock.

### Required tests

- Theme manager may preview and activate the exact previewed L0 artifact
- Extension viewer may preview but cannot activate
- Non-super-admin cannot approve or restore replace via API
- Upgrade/digest change between preview and activation returns 409 with no mutation
- Concurrent current-theme change returns 409 and requires a new preview
- Preview final-state conflicts match activation preflight outcomes
- Preview rejects non-theme/invalid package without mutation
- No frontend action can send the obsolete id-only/empty-body activation

### Rollback

Revert the preview/CAS contract and temporary UI interlock together. Do not
restore Web Release response types. P4 must not land against an id-only
activation endpoint.

## P3 — SSR-Safe L0 Skin And Immediate Runtime Refresh

**Goal:** Remove the one-shot client DOM injection gap and make browser and SSR
theme state coherent through Nuxt-native primitives and explicit cache policy.

### Tasks

- [ ] T3.1 Refactor `useActiveThemeSkin` into a setup-time composable backed by
      `useAsyncData('site-active-theme-skin', ...)`.
- [ ] T3.2 Render digest-bound CSS/token URLs through reactive `useHead` link
      entries with stable keys instead of manual `document.createElement`
      management. Preserve token-before-theme CSS order.
- [ ] T3.3 Include active skin links in SSR for public/default and auth layouts,
      preventing flash-of-default-theme and ensuring a usable first render.
- [ ] T3.4 Scope public theme CSS away from the admin layout. Admin
      personalization/color mode remains independent from the active public
      theme. Remove the app-global client injection path so client navigation
      away from public/auth layouts also removes theme links.
- [ ] T3.5 Make `GET /site/active-theme/skin` explicitly
      `Cache-Control: no-store`. Keep only digest-bound theme assets immutable.
- [ ] T3.6 Disable Nitro SWR/CDN `s-maxage` for routes that render public/auth
      layouts, active theme settings, or Page Registry output. Correctness is
      the interim policy; revision-aware shared caching is a separate program.
- [ ] T3.7 Centralize theme-runtime async-data keys under one shared helper or
      prefix so replace resolves and both dynamic add route families can be
      invalidated without duplicated string conventions.
- [ ] T3.8 Add one shared `refreshActiveThemeRuntime` helper that:
  - refreshes `site-active-theme-skin`;
  - refreshes `site-active-theme-settings`;
  - clears replace resolves (`page-resolve:*`);
  - clears both add-route families (`page-resolve-path:*` and
    `page-resolve-path-x:*`), preferably through the centralized prefix;
  - optionally refreshes the currently rendered public page when the switch is
    initiated from a public-capable surface.
- [ ] T3.9 Expose the helper with focused tests for full and partial final-state
      refresh. Do not wire it into the stale id-only activation action; P4 owns
      integration with the preview-bound flow.
- [ ] T3.10 Keep safe host/default CSS when the API is unavailable and reuse the
      existing API connection modal for operator retry; do not invent an
      unbounded background retry loop. Preserve a diagnosable fetch error rather
      than swallowing it into an indistinguishable empty success response.
- [ ] T3.11 Ensure old digest links are absent after the new head state settles;
      a bounded stylesheet handoff may overlap only while the new link loads.
- [ ] T3.12 Refresh active theme settings so default-theme-only values do not
      leak into Nocturne and vice versa.

### Required tests

- SSR response for a public page contains active theme links
- Admin response does not contain public theme skin links
- Active-skin descriptor is `no-store`; digest assets are immutable
- Theme-sensitive public/auth HTML is not Nitro/CDN shared-cached
- A pre-switch SSR request followed by activation cannot serve old L0/L1 HTML
  on the next SSR request
- Theme switch replaces old link digests with new digests
- Active theme settings change with theme id
- Cleared `page-resolve:*` data re-resolves the approved provider
- Both dynamic add-route async-data key families are cleared and re-resolved
- API unavailable renders safe host fallback and exposes retry UX
- No duplicate stylesheet links across navigation/HMR

### Rollback

Revert the reactive head/client changes and cache-rule commit together. Core
CSS remains bundled, so public pages stay usable; do not restore shared HTML
caching while SSR still embeds active digest/provider state without coherent
invalidation.

## P4 — Guided Activation, Operator Clarity, And Reference Acceptance

**Goal:** Expose the P2 contract through the P3 runtime primitives, with honest
partial-state UX and clear capability boundaries.

### Tasks

- [ ] T4.1 Make every theme Activate button use one manager entry point that
      fetches P2 preview before mutation.
- [ ] T4.2 Add a focused confirmation modal showing:
  - current theme and target theme;
  - L0 skin summary;
  - every add/replace target, route, access class, contract, and conflict;
  - current provider and predicted provider after the chosen operation;
  - whether each replace requires `super_admin`;
  - that unselected/unapproved pages retain a valid non-old-theme provider when
    one exists, otherwise use core fallback.
- [ ] T4.3 For `super_admin`, preselect all target-theme replaces valid in the
      computed final state; no manifest recommendation field is added. For
      `sforum.default-theme`, preselect core fallback while leaving its declared
      replace available for explicit selection.
- [ ] T4.4 For non-super-admin theme managers, render protected provider choices
      read-only and explain that activation applies L0/add pages while existing
      valid protected provider state remains API-controlled.
- [ ] T4.5 On confirm, execute deterministically:
  1. POST theme activation with the previewed target and active-theme tuples;
  2. after success, for `super_admin`, POST `restore-core` for deselected pages
     currently bound to the same target artifact, then POST selected approvals
     sequentially with exact id/version/digest/contract;
  3. leave valid plugin-owned providers untouched when their target-theme
     replacement is unselected;
  4. stop on the first page mutation failure and report successful, failed, and
     pending page ids without forged/stale recovery.
- [ ] T4.6 Do not run page mutations if activation fails or returns 409. Refresh
      preview and require a new confirmation after conflicts.
- [ ] T4.7 Invoke `refreshActiveThemeRuntime` only after the mutation sequence
      reaches its final state, including partial page success, then refresh the
      extension and Page Registry admin lists.
- [ ] T4.8 Full success: one 10-second success Toast using SForum appearance
      tokens. Partial success: persistent inline warning plus non-auto-dismiss
      error Toast with retry and link to “扩展 → 页面”.
- [ ] T4.9 Treat deliberate unselected layouts as a successful operator choice,
      not an error, while summarizing predicted retained/core providers.
- [ ] T4.10 Add i18n in `zh-CN` and `en-US`; no emoji, use approved icons.
- [ ] T4.11 On the Themes page, distinguish:
  - active public theme;
  - L0 skin applied;
  - exact-artifact L1 page layouts applied/remaining on core;
  - approvals revoked because version/digest/contract changed.
- [ ] T4.12 Add concise helper text that “配色预设 / appearance preset” is
      separate from an installable public theme.
- [ ] T4.13 Verify Nocturne’s current contract remains intentionally small:
      L0 skin plus `forum.home` L1 shell around the host `sf-home-page` island.
- [ ] T4.14 Verify the protected-default recommended flow restores the normal
      core homepage and removes the Nocturne binding. Also verify a super-admin
      may explicitly choose the default package's listed L1 `forum.home`
      contribution instead of core fallback.
- [ ] T4.15 Verify light/dark modes remain readable and that theme CSS does not
      override admin personalization.
- [ ] T4.16 Update runtime-theme author documentation with the activation,
      explicit approval, fallback, and restart-persistence lifecycle.
- [ ] T4.17 Record the current structural limit: the host island catalog is
      intentionally small and `sf-home-page` remains monolithic. Put expansion
      in follow-up backlog rather than silently broadening this task.

### Required tests

- Theme manager sees preview and may activate L0/add pages
- Non-super-admin cannot approve/restore through UI or API
- Super-admin explicitly confirms recommended replaces
- Protected default recommends core fallback though its L1 replace is listed
- Activation failure/conflict prevents page mutation requests
- Digest/version/contract approval race fails closed with retry guidance
- One page mutation failure produces partial state, not false success
- Earlier successful page mutations are reported and retained
- Same-theme deselection explicitly restores core for that bound page
- Unselected target replace retains a valid plugin-owned provider
- Default activation clears the old theme provider without extra approval
- Final-state runtime refresh runs after full and partial sequences

### Acceptance boundary

This phase proves a complete switch for the capabilities Nocturne actually
declares. It does not claim that every public page has a distinct Nocturne
layout.

### Rollback

Revert the guided modal/status commits and return to P2's safely disabled
activation action; the separate Page Registry screen remains available. Do not
re-enable the obsolete id-only request or revert the P2/P3 safety contracts as
part of a P4 UI rollback.

## P5 — Full Verification, Contracts, And Handoff

**Goal:** Close the program with evidence proportionate to a lifecycle change.

### Automated gates

- [ ] T5.0 Re-check that the concurrent task remains completed and no
      overlapping unowned edits appeared during P0-P4.
- [ ] T5.1 `cd apps/api && go test ./...`
- [ ] T5.2 `cd apps/api && go build ./...`
- [ ] T5.3 `cd apps/web && bun test`
- [ ] T5.4 `cd apps/web && bun run typecheck`
- [ ] T5.5 `cd apps/web && bun run build`
- [ ] T5.6 `ruby scripts/validate-openapi-refs.rb`
- [ ] T5.7 `./scripts/test.sh`

### Browser/live scenarios

Prefer isolated API/database ports. Never kill the user’s manually started web
server on port 3000. If validation must use the current development database,
snapshot the original active theme and page bindings and restore them exactly.

- [ ] T5.8 Super-admin: default → Nocturne preview → activate + approve
      `forum.home` → Nocturne skin and hero render.
- [ ] T5.9 Restart API/Air → Nocturne remains active; skin digest and approved
      provider remain unchanged.
- [ ] T5.10 Nocturne → default → core homepage returns; old binding is absent.
- [ ] T5.11 Theme manager without `super_admin` → L0 activation allowed,
      protected replace unavailable, denied API path verified.
- [ ] T5.12 Approval failure injection → L0 remains active, core fallback
      remains usable, persistent partial-state feedback shown.
- [ ] T5.13 Desktop and one mobile viewport, light and dark, no relevant console
      errors or framework overlay.
- [ ] T5.14 Stale preview injection: change target digest or active theme between
      preview and confirm → 409, no activation/event/audit/binding mutation.
- [ ] T5.15 Cache regression: request themed SSR, switch theme, request again →
      no old digest/provider and no shared-cache response header.
- [ ] T5.16 Same-theme package digest change → L0 remains usable after preflight,
      stale L1 approval is removed, and UI requires reapproval.

### Documentation close

- [ ] T5.17 Update this checklist and status.
- [ ] T5.18 Update `knowledge/modules/extensions.md` and
      `knowledge/modules/frontend.md` with final lifecycle ownership.
- [ ] T5.19 Add an ADR addendum only if implementation changes an accepted
      decision; ordinary bugfix details belong in the session handoff.
- [ ] T5.20 Add a final session handoff with changed files, decisions, gates,
      remaining risks, and exact next step.
- [ ] T5.21 Update `knowledge/index.md` from “plan ready” to “completed”.

## Permission And Security Matrix

| Action | Authority |
| --- | --- |
| View theme list/preview | Existing `extension.view`, theme-manage, or compatibility parent permission; preview exposes no secrets |
| Activate public theme L0/add pages | `extension.theme.manage` or existing extension manage fallback; request is preview-bound |
| Approve core page replace | `super_admin` only; API authoritative |
| Restore core provider | `super_admin` only |
| Automatically revoke stale theme binding | Internal lifecycle/reconciler only; reason and artifact identity audited |
| Read active skin descriptor | Public GET, active theme only, `no-store` |
| Serve active theme assets | Public GET, active theme only, digest-bound, allowlisted type, immutable when digest is present |
| Resolve public page | Existing catalog access + API policy; core fallback on mismatch |

Frontend hiding/disabled states are UX only. Existing API permission checks,
contract validation, digest validation, and safe template paths remain the
authority.

## Commit Discipline

Only commit when the user authorizes commits. Keep changes small and
revert-friendly.

Suggested boundaries:

1. `test(themes): cover active theme persistence and stale bindings`
2. `fix(themes): enforce one active selection and restart reconciliation`
3. `fix(pages): revoke stale theme provider bindings on switch and restore`
4. `fix(themes): bind synchronous activation to previewed artifact state`
5. `feat(themes): preview activation and apply explicit page approvals`
6. `fix(web): make theme SSR head and async data reactive`
7. `fix(web): prevent stale shared-cache theme HTML`
8. `docs(themes): explain runtime activation approval and fallback`
9. `test(themes): verify restart-safe and cache-safe complete switch`

Do not mix unrelated identity/admin-user work currently present in the working
tree with this program.

## Definition Of Done

The program is complete only when all statements are true:

- [ ] A valid non-default theme survives API and worker built-in sync.
- [ ] An invalid/missing active theme safely restores the protected default.
- [ ] Concurrent activation and startup paths cannot leave multiple enabled themes.
- [ ] Old-theme provider bindings do not remain after switching away.
- [ ] Crash/startup reconciliation removes bindings owned by inactive/invalid themes.
- [ ] Theme activation always shows exact-artifact page impact before mutation.
- [ ] Stale target/current-theme preview state returns conflict with no mutation.
- [ ] Core replace approval stays explicit and `super_admin`-only.
- [ ] A complete Nocturne activation visibly applies L0 and approved L1 home.
- [ ] Switching to default restores core home without stale Nocturne state.
- [ ] Same-id package changes revoke old digest/contract approvals.
- [ ] CSS/settings/page resolution update without Nuxt rebuild or hard reload.
- [ ] Replace and both dynamic add-route async-data families are invalidated.
- [ ] Theme-sensitive SSR is not stale through Nitro/CDN shared caching.
- [ ] Admin styling is not controlled by the public theme skin.
- [ ] Partial failures are visible, persistent, and leave core fallback usable.
- [ ] Concurrent task final state and overlap ownership were rechecked before P0 and P5.
- [ ] OpenAPI, i18n, tests, browser evidence, modules, index, and handoff agree.

## Reviewer Checklist

Please explicitly approve or change:

- [ ] D1 preserve the selected theme and validate/fallback in API restore
- [ ] D2 explicit confirmation with recommended page replaces preselected
- [ ] D3 keep L0 active on page-mutation partial failure
- [ ] D4 no L2/new island expansion; correctness-first shared-cache removal is in scope
- [ ] Phase order P0 → P5
- [ ] Preview-bound mutation, DB singleton, crash reconciliation, and cache invariants
- [ ] Commit boundaries and Definition of Done
