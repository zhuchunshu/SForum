# 2026-10-16 Session Handoff — M5 Guard Authorization Disclosure

Plan: `knowledge/plans/2026-08-22-extension-authoring-speed-and-controlled-power.md`
(milestone **M5**, in progress — task 1 delivered, tasks 2-8 audited with gaps).

## Changed

- **Authorization ownership is now disclosed to operators** (M5 task 1 + part of
  task 3). Everything the milestone asks for already existed *inside* the
  digested impact document, but scattered across three sections (`routes[]` for
  route scope/fallback/mutable fields, `guardDeclarations[]` for guard
  kind/entry/digest/permissions, `binaries` for executable digests) and without
  a risk classification. It is now joined in one place:
  - `apps/api/app/Support/ExtensionManifest/guard_risk.go` — the risk
    vocabulary (`host_owned` / `extension_code` / `raw_request`), the guard kind
    constants, `GuardRuntimeRisk`, `GuardRequiresExactArtifactTrust`.
  - `apps/api/app/Models/Extensions/guard_policy.go` —
    `TrustGuardAuthorizationView` + `GuardAuthorizationViews(impact)`.
  - `apps/api/app/Models/Extensions/trust_types.go` +
    `trust_service.go` — `ExecutableTrustStatus.GuardAuthorization` filled on
    every status read.
  - `apps/api/app/Support/Routes/inspector.go` + `plugin_guard.go` —
    `InspectorStep.guardRisk` and `.guardRequiresTrust` on every inspected step.
  - Web: `SFAdminExecutableTrustImpact.vue` renders an `授权归属` section
    (`data-testid="extension-guard-authorization"`) with per-route owner, guard
    kind, executable digest, fallback and requested fields, plus a red alert
    (`data-testid="extension-guard-raw-request"`) when a route hands
    authorization to raw request authority; the enable/activate/upgrade dialog
    passes `trustStatus.guardAuthorization`; `extensionTrust.ts` carries
    `ExecutableGuardAuthorization`; zh-CN/en-US keys added under
    `admin.extensions.trust.guardAuthorization.*`.
- `v3_validate.go` now uses `GuardKindCustom` / `GuardKindRawRequest` instead of
  duplicated string literals, so the admission whitelist and the risk
  vocabulary cannot drift apart.

**Why a derived view instead of new impact fields — read this before "fixing"
it:** see `knowledge/decisions/2026-10-16-trust-impact-digest-is-authority-view-is-derivation.md`.
Adding fields to `TrustImpact`/`TrustGuard` changes `canonicalTrustImpactDigest`
for *every* extension, and that digest is the grant identity in
`extension_trust_grants` (`HasLiveGrant` matches on `impact_digest`), while
`IdentityRegistry/legacy_adoption.go` admits only `sforum.trust-impact@2`. A
digested change therefore un-trusts every existing grant and needs an impact
schema bump plus an operator re-confirmation path.

## Evidence

Go (all green, `-count=1`): `app/Models/Extensions`, `app/Support/Routes`,
`app/Support/ExtensionManifest`, `app/Http/Controllers/Extensions`; new tests
`app/Support/ExtensionManifest/guard_risk_test.go` (internal, reuses the
validator fixture) and `app/Models/Extensions/guard_authorization_view_test.go`
(join, raw-request disclosure, detachment from the impact, and an assertion that
the view never leaks into the digested document), plus extended
`TestInspectorDisclosesExactCustomGuardBinding` and a new
`TestInspectorMarksHostOwnedGuardRisk`.

Runtime (HTTP, real dev instance, authenticated super_admin):

- Installed probe fixture `acme.guardprobe` (custom guard + a
  `core.guard.raw_request` route, packaged in `/tmp` and uploaded; left
  **installed and disabled**).
- `GET /api/v1/admin/extensions/acme.guardprobe/trust` →
  `guardAuthorization: [{route: acme.guardprobe.route.before, guardKind: custom,
  risk: extension_code, requiresExactTrust: true, fallback: closed,
  mutableRequestFields: ["/query/acme-guardprobe-trace"], digest: 913a39cd…},
  {route: acme.guardprobe.route.raw, guardId: core.guard.raw_request,
  risk: raw_request, rawRequest: true, requiresExactTrust: true, digest: ""}]`.
- `GET /api/v1/admin/extensions/route-inspector?method=GET&path=/api/v1/topics`
  → `guardRisk: "host_owned"`, `guardRequiresTrust: false` on the core step.

Browser (real carrier, logged-in super_admin):

- Desktop 1313x930: the review dialog renders `授权归属` with 2 rows, both risk
  badges (`插件代码`, `原始请求`), the raw-request alert, 0 clipped controls, no
  horizontal overflow. Screenshot:
  `storage/qa/2026-10-16-guard-authorization-dialog-1313x930.png` (gitignored).
- Narrow: the only narrow carrier in this session reports **500x846, not 390**
  (its `Emulation.setDeviceMetricsOverride` is accepted but ignored; recorded
  honestly, never presented as 390). At that width the section renders, the
  alert is visible, 0 clipped controls, 0 elements overflowing their own box, no
  document horizontal overflow, dialog box spans 16 → 484.

Web: `bun test` 1207 pass / 0 fail (two new tests in
`tests/admin/executableTrustUi.test.ts`, including a per-locale key/label check
for every risk the API can emit); `bun run typecheck` exit 0.
Repo: architecture validation passed (1780 production files, no baseline
raised), `scripts/v3-catalog/generate.mjs --check` current (16 files),
`validate-docs.mjs` passed, `validate-openapi-refs.rb` OK (2790 refs).

## Round 2 (same day): task 6 gap closed, tasks 2/4/5 verified

Second pass read the actual emitters and tests instead of stopping at "not
found". Results, with the evidence that settles each one:

- **Task 2 — separate short-lived actor-bound super_admin confirmation:
  VERIFIED.** `challengeForTarget` requires `actor.IsSuperAdmin()`
  (`trust_service.go:309-311`); the challenge row is actor-bound
  (`trust_postgres.go:212-237` compares `actorUserID`, expiry stored) and
  short-lived (`DefaultTrustChallengeTTL = 5 * time.Minute`,
  `trust_service.go:24`, capped by `WithTTL`). Emissions:
  `extension.trust_challenge` / `extension.trust_denied` / `extension.trust_grant`
  at `trust_service.go:335/379/383/419/422`; a denied confirm carries
  `reason = trustErrorCode(err)`, which is the "deny reason category".
- **Task 4 — audit coverage: VERIFIED except one deliberate gap.** The audit
  vocabulary in `app/Support/Audit/audit.go` already names every required
  action: `extension.trust_grant`, `extension.trust_revoke`,
  `extension.trust_denied`, `extension.trust_challenge`, `extension.enable`,
  `extension.disable`, `extension.upgrade`, `extension.rollback`,
  `extension.uninstall`, `routes.provider_select`, `routes.provider_reset`,
  `providers.slot_select`, `pages.conflict_select`, `routes.runtime_incident`,
  `routes.committed_after_failure`, `extension.activation_failed`,
  `extension.lifecycle_recovery`, `extension.safe_mode_boot`,
  `extension.cli_recovery`; quarantine is audited through
  `routes.committed_after_failure` with `quarantineResult` in the metadata
  (`app/Http/route_failure_recorder.go:311-338`) plus the durable incident row.
  The one gap is a **judgement call, not an oversight**: a per-request
  *plugin-guard denial* (`PluginGuardFailureDenied`,
  `app/Http/plugin_guard_runtime.go:110-125`) writes no audit row, because the
  audit ledger is an admin-facing, low-volume record and the request path must
  not gain a durable write. Needs a human decision if the milestone intends
  per-denial auditing.
- **Task 5 — disable/upgrade close admission and drain guard calls: VERIFIED.**
  Guard invocations acquire a `RuntimeCallGuard` lease
  (`app/Http/plugin_guard_runtime.go:66`); the admission gate has
  `BeginDrain` / `Quarantine` / `Wait` / `ForceCancel` / `Resume` with per-class
  bookkeeping (`app/Support/ExtensionRuntime/admission.go:145-240`), and the
  lifecycle/publication paths call them
  (`manager_runtime_instances.go:131/166/188/243/268/414/444`,
  `manager_staged_runtime.go:257/312/448`, `plugin_runtime_full_set.go:693`).
  Integration coverage exists:
  `protocol_v2_routes_integration_test.go:205-225` and
  `commerce_workflow_reference_plugin_integration_test.go:198-208` acquire a
  guard-class call and then assert drain behaviour.
- **Task 6 — upgrade compensation restores the prior complete route/guard
  publication: gap CLOSED this round.** The route-policy publication tests
  asserted routes and policies but **never guards** (no guard declaration in any
  of them). New `app/Support/Extensions/lifecycle_route_guard_policy_test.go`
  covers it end to end on the authoritative-publication path: a custom-guard
  route publishes its exact binding (id / kind `custom` / entry `bin/guard` /
  digest / permissions); the upgrade to a second version leaves the new digest
  and **no residue** of the old one while the old route stops resolving; the
  rollback restores the **prior** binding field-for-field; and both an
  authoritative empty publication and Safe Mode publish **zero** guard authority.
  Note the manifest rule this fixture exposed: a guard `entry`+`digest` must
  match an `executable` entry in `packageFiles[]`
  (`v3_validate_ui.go:59-63`), so a guard executable is its own exact artifact.
- **Task 7 — UI loading/empty/error/stale/disabled/recovery states: still
  open.** `SFAdminExtensionLifecycleDialog.vue` and
  `useAdminExtensionsManager.ts` carry recovery/quarantine handling, but the six
  states are not individually confirmed and no rendered evidence was collected
  for each one.

## Round 3 (same day): task 7 delivered — the quarantine state is now visible

M5 task 7's missing half was not a missing spinner: the durable crash-loop
quarantine state had **no admin read path at all**. The incident ledger
(`extension_route_runtime_incidents`) was write-only
(`CreatePending`/`Resolve`) and the only audit of it was the
`routes.committed_after_failure` event, so an operator could not see — after a
restart or at all — which exact runtime surface the Host had closed. That is
precisely M5's exit gate ("Admin UI and API report the same durable state after
restart").

Delivered:

- `app/Http/route_runtime_incident_reader.go` — read-only boundary
  (`RouteRuntimeIncidentReader`, deliberately separate from the write interface so
  the recorder's test doubles are unaffected), a bounded read
  (`RouteRuntimeIncidentReadMaximum = 200`, newest first), a redacted view
  (`RouteRuntimeIncidentView`) and the status summary. Note the semantics fixed
  here: `local_quarantine_result` is the *quarantine decision* while
  `resolved_at` only closes the ledger row, so a resolved `quarantined` row still
  counts as quarantined, not pending.
- `GET /api/v1/admin/extensions/route-incidents` (controller handler + route +
  bootstrap/provider wiring), gated like the other inspectors
  (`extension.view` or legacy `extension.manage`); unconfigured reader fails
  closed with 503; an empty ledger is a 200 with `incidents: []`, never a 404.
- OpenAPI: `contracts/openapi/paths/extension-route-incidents.yaml` + schemas in
  `extension-route-inspector.yaml`, indexed in `contracts/openapi.yaml`.
- UI: the Route Inspector page gained the `route-incidents` panel with all six
  states — loading, error, **stale** (a failed refresh keeps the last good ledger
  and warns instead of replacing it with an error), empty, **quarantined**
  (red alert + per-row result badges) and **recovery** (Safe Mode / CLI recovery
  guidance) — plus a bounded-read notice, a refresh action and zh-CN/en-US copy.
  The composable gained `loadIncidents` + a strict ledger parser; its read-only
  guarantee (no provider select/reset) is still asserted.
- Reviewed matrices that a new read route must enter (the gates caught all three):
  `docs/extensions/v3/catalog-identities.json` (stable route identity),
  `app/Http/core_guard_authorizer.go` `requireExtensionsReadAuthority` (the
  provable-policy list — without it the production authorizer answers "guard
  unavailable", so this was a real runtime wiring gap, not a test detail), the
  read-guard partition test map, and the catalog route count (362 → 363).

Evidence:

- HTTP on the live instance: empty ledger → `200 {limit: 200, summary: all 0,
  incidents: []}`.
- HTTP + browser with a **seeded fixture row**: `summary {quarantined: 1}` and a
  full view record; the Route Inspector panel rendered
  `已隔离 1 · 待处理 0 · 无需隔离 0`, the quarantine alert
  (`有 1 个精确运行时表面因崩溃循环被关闭…`), the recovery alert and the row
  (`GET /api/v1/topics`, `acme.iconprobe@0.1.0 · fixture-instance-m5`, `已隔离`,
  `transport_failed`, `runtime_transport`). Screenshot:
  `storage/qa/2026-10-16-route-incidents-quarantined-1313x930.png` (gitignored).
  The fixture row and its audit row were then removed and the panel rendered the
  empty state (`已隔离 0 …`, `没有崩溃循环事故…`, 0 rows).

Two durable-DB facts worth remembering (both discovered by writing the fixture):

1. An incident row must be backed by an audit event with action
   `routes.runtime_incident` and `metadata.incidentKey` equal to the row's key
   (`validate_route_runtime_incident_audit` trigger).
2. The incident table is **append-only** (`resolve_once` + `no_truncate`
   triggers). Removing the fixture required disabling that trigger inside one
   transaction and re-enabling it; the trigger state was verified as enabled
   afterwards and the table is empty again. Also `local_quarantine_result` and
   `resolved_at` must agree (`pending` ⇔ NULL) and `state_check` pins the exact
   phase/invocation-stage/failure/cause/commit combinations.

Rendered-evidence coverage for task 7: loading, empty, quarantined and recovery
were rendered in a real browser at 1313x930; **stale and error are covered by the
web contract tests and the code path, not by a rendered screenshot** (the stale
state needs a failing refresh while a ledger is already loaded).

## Round 4 (same day): M8 `extension dev` loop delivered

`sforum extension dev <package-root>` now exists, built to the M0 ADR 6.2
decisions rather than to convenience:

- **Gate**: the exact `DevHygiene.ShouldEnableDevelopmentOrphanReaper` predicate
  (`APP_ENV=development` only), refused at command entry with an actionable
  message; tests cover production/staging/test/empty/`PROD`.
- **No trust bypass**: the loop only automates the existing CLI steps. Every
  cycle prints the new package digest and a `reload_required` phase whose `next`
  points at the admin exact-artifact confirmation; nothing stages or activates
  behind the operator.
- **Dependency decision honoured**: `github.com/fsnotify/fsnotify v1.10.1` is now
  a **direct** module dependency (it was previously only an indirect dependency of
  the external `air` binary).
- **No second build pipeline**: the runner calls `runExtensionBuild` directly
  (which already performs digest refresh, validate and contract tests) and then
  reads the package digest from `extensionpackage.DigestTree`.
- **Root policy**: allowed under `extensions/dev`, `extensions/optional`,
  `extensions/fixtures`; protected `extensions/builtin` trees and anything outside
  the repo are refused with the reason and the correct path named.
- **Watch set**: package-contained source inputs only — build outputs
  (`frontend/**/dist`, `node_modules`, `.git`, `.nuxt`, `vendor`, `tmp`), archives
  (`*.sforum.zip`, `*.zip`), editor swap files, `*.tmp`, `*.map`, `*.log` are
  excluded by directory and suffix.
- **Debounce + coalescing + one transaction at a time**, with an overflow counter
  when an event storm exceeds the coalescing cap.
- **Machine-readable mode** (`--json`) emits one JSON object per phase for editor
  integrations, plus `--once` for CI.

**Two real defects the end-to-end run exposed (and fixed):**

1. **The loop was self-triggering.** The first real run produced ~13,000 phase
   lines in seconds: each build transaction rewrites `sforum.extension.json`
   during digest refresh, and that write is a watched source input, so the loop
   rebuilt forever. Watch-set filtering alone cannot fix this — the fix is a
   **content stamp snapshot** (`devFileStamp` per watched file, re-taken after
   every transaction): an event whose stamp equals the last snapshot is our own
   write and is ignored, while a genuine author edit or a deletion still
   triggers. Regression test:
   `TestExtensionDevLoopIgnoresItsOwnBuildWrites` (asserts no extra transaction
   after a build that rewrites the manifest, and that a second author edit still
   triggers) and `TestExtensionDevLoopRebuildsOnDeletedSourceInput`.
2. **The preflight refused exactly the state it exists to fix.** Requiring
   `LoadPackage` to succeed meant a package with stale digests — the normal state
   right after an author edit — was rejected before the loop could refresh it.
   The command now only requires a manifest to exist, and the loop runs an
   explicit `trigger: startup` transaction so starting it reconciles the package.

**End-to-end evidence** (real binary, real package under `extensions/dev`, then
removed):

```
$ APP_ENV=development sforum-dev extension dev extensions/dev/acme.devloop-e2e --json --debounce 150ms
{"phase":"watching","root":".../acme.devloop-e2e"}
{"phase":"building","trigger":"startup"}
  … real build: "refreshing exact package digests" → backend/plugin 2c479a34… →
    "validating extension package" OK → "running extension contract tests" PASS (0 errors, 1 stub warning)
{"phase":"built","packageDigest":"11c1bea19e582a5593228ca7843c2fcc7ba9acff06ad7c3f0a5107c78585afe6"}
{"phase":"reload_required","next":"confirm the exact artifact in the admin extensions flow to stage and activate it"}
# one author edit to backend/plugin
{"phase":"building","trigger":"change","changed":1}
{"phase":"built","packageDigest":"55fccd4cf56a0c436ec4a24ee7743f87741030ff625566118df85db326161bd8"}
{"phase":"reload_required", …}
```

Exactly 7 events total (no self-triggering), the declared manifest digest matched
the edited file afterwards (`74f20395…`), `SIGINT` produced
`{"phase":"stopped", …}` with **no leftover process**, and the probe package was
deleted afterwards.

**M8 items not done (recorded, not claimed):** the loop does not automate the
admin activation step (the ADR forbids a trust bypass, so it prints the exact
next step instead); built-in discovery from the manifest tree (ADR 6.2.5) is not
implemented — `scripts/build-builtin-plugins.sh` still carries its hardcoded id
list; and runtime-crash/API-restart reactions are not tested (a rebuild is
triggered by source edits, not by runtime failure).

## Round 5 (same day): M8 sub-item — built-in discovery replaces the hardcoded list

ADR 6.2.5 is now satisfied. `scripts/build-builtin-plugins.sh` no longer contains
any built-in id or directory name; it consumes discovery:

- `sforum extension builtins [--root <dir>] [--json]` walks the manifest tree and
  reports `id`, `relativePath`, `backendEntry` and `hasGoBackend` per package
  (`apps/api/cmd/sforum/dev_builtins.go`). Text mode is tab-separated so the shell
  script needs no `jq`.
- Discovery deliberately does **not** use `LoadPackage`: the protected source tree
  has no `backend/plugin` binary (it is built into staging), so full package
  validation would fail on digest mismatch. It reads the manifest shape, follows a
  `includes.backend` shard when the root manifest has no backend, requires
  `manifestVersion == 3`, and rejects duplicate ids.
- The script loops the discovery output, builds only packages with a Go backend,
  refreshes digests in staging, runs each contract test, and reports
  `built=<n> skipped=<n>`; themes are skipped and still rsynced. `ROOT_DIR` remains
  the override point (`builtin_source_root()` is resolved at call time), which keeps
  the pre-existing staging test working.

Evidence (real run into a probe staging root, then removed):

```
$ SFORUM_BUILTIN_DEV_ROOT="$PWD/storage/builtin-dev-probe2" bash scripts/build-builtin-plugins.sh
Building protected built-in plugin: sforum-shortcodes
Building protected built-in plugin: sforum.auth-github
… (8 plugins, each followed by "errors: 0" contract tests)
Built-in plugins staged at: …/storage/builtin-dev-probe2 (built=8 skipped=1)
Source tree manifests under extensions/builtin were not modified.
```

`exit=0`, 8 × contract-test PASS, 8 staged binaries, and the staged manifest digest
matched the built binary (`22b3ec3a…` for `sforum.smtp`, while the source manifest
kept its stable `ce3ee1e7…`). New test
`TestBuiltinDiscoveryIsTheOnlyBuiltinList` asserts discovery finds ≥8 backend
built-ins plus at least one theme, and that the script contains neither the ids nor
the directories it used to hardcode.

Note for future sessions: the `SFORUM_BUILTIN_DEV_ROOT` override only works at the
same directory depth as `storage/builtin-dev`, because built-in `go.mod` files use a
relative `replace` pointing at `apps/api`. That is now documented in the script.

## Round 6 (same day): M8 frontend path + child-process cleanup defect

The last doable M8 sub-item was the frontend build path and child-process
cleanup. Both were exercised with a real probe package under `extensions/dev`
(generated with `make:plugin --backend`, plus a hand-written
`frontend/admin/package.json` and two `packageFiles` entries), then removed.

- **Frontend path works.** `extension dev --once --skip-install` entered the
  frontend branch (`running bun run build in …/frontend/admin`), the build script
  produced `dist/dashboard.mjs` + `dist/dashboard.css`, digest refresh listed
  both, `validate` passed and the contract test reported 0 errors, ending in
  `built digest=5461df1c…`. `--skip-install` is a new flag on `extension dev`
  (it forwards `SkipInstall` into the existing build options), which is what makes
  an offline/repeat loop possible.
- **Real defect found and fixed: grandchild processes survived SIGINT.** With the
  build script set to `sleep 45`, SIGINT killed the loop and `bun` but left
  `sleep 45` running — `exec.CommandContext` only kills the direct child.
  `runExtensionBuildProcess` now calls `devConfigureChildProcess`, which puts the
  child in its **own process group** (`Setpgid`) and installs a `Cancel` that
  kills the whole group (`syscall.Kill(-pid, SIGKILL)`, treating `ESRCH` as
  success). Platform split: `dev_child_process_unix.go` (`!windows`) and
  `dev_child_process_windows.go` (default semantics; the repo does not cross-build
  for Windows anyway — `app/Support/ProcessMemory` and `app/Support/DevHygiene`
  already fail `GOOS=windows`).
  Re-run after the fix: the in-flight `sleep 45` was gone after SIGINT, and the
  loop reported `failed: bun [run build] failed: signal: killed` followed by
  `stopped` — an accurate report of a cancelled transaction.
  Tests: `TestDevChildProcessKillsWholeProcessGroupOnCancel` (group exists before
  cancel, `ESRCH` after) and `TestDevChildProcessCancelIsIdempotentAfterExit`.

M8 now stands at: delivered core loop + gate + watch set + debounce/coalesce +
single transaction + machine-readable mode + SIGINT cleanup + child-process-group
reclamation + built-in discovery. Still open (recorded, not claimed):
admin-activation automation (ADR forbids the bypass) and runtime-crash/API-restart
reactions.

## Round 7 (same day): M9 documentation half

Delivered the discoverability half of M9 (no architecture change):

- `docs/extensions/migrating-from-sforum-v2.md` — canonical English page with the
  SForum-old → V3 migration table, the deliberately closed operations, and quick
  starts. Mapping rows are contract-accurate: Controller → declared `routes[]`;
  route middleware → `action: "before"`/`"after"` entries with
  `guard: "core.guard.inherit"` and explicit
  `mutableRequestFields`/`mutableResponseFields`; Blade → Page Registry
  `theme.json.pages[]` + `templates[]` + `packageFiles[]`; `Itf` hook → `hooks[]`
  plus typed events/queries/providers; plugin helpers → the Protocol V2 SDK.
- Bilingual handbooks: `docs/zh-CN/extensions/migrating-from-sforum-v2.md`,
  `docs/en-US/extensions/migrating-from-sforum-v2.md`, plus links from
  `docs/README.md` and the authoring guide's related-docs list.
- The CLI docs gate now derives expected subcommands from `dev_extension.go` and
  `dev_builtins.go`, so both locales must document `extension dev` and
  `extension builtins`; they do.
- Quick starts verified in a clean `/tmp` directory outside the repo:
  `make:plugin` → `digest --write` → `validate` → `test` (0 errors, 1
  `backend.scaffold_stub` warning) → `build --allow-scaffold --skip-install`, all
  OK.

**Two contract conflicts recorded instead of papered over** (the task book asked
for things the code does not have):

1. **No session contract.** Neither the manifest nor Protocol V2 exposes session
   reads/writes, and M6 (plugin-namespaced session state) is still blocked on the
   M0 session ADR. M9 task 1's "plugin session-state workflow" reference package is
   therefore not buildable today; task 5's "Core session mutation is closed" is
   accurate and now documented as such.
2. **No challenge outcome for guards.** `RuntimePluginRouteGuardEvaluator`
   (`app/Http/plugin_guard_runtime.go:32`) returns nil (allow) or fails with kind
   `denied`, `crash`, `timeout`, `protocol`, `canceled`, `unavailable`. M9 task 1's
   "allow/deny/challenge" package can only cover allow/deny.

**Reference-package inventory (M9 task 1, seven bullets).** Already focused in
`extensions/fixtures/plugins/`: typed plugin page data
(`sforum-plugin-page-business-e2e`, `theme.json` with `dataSource: plugin` +
`dataRoute` + `dataSchema`), host-inherited admin page
(`sforum-admin-surface-reference`), before/after middleware and custom guard
(`sforum-commerce-workflow`), plus `sforum-query-reference`,
`sforum-contract-events`, `sforum-contract-schedules`, `sforum-seo-reference`,
`sforum-region-demo`, `sforum-media-optimize`, `sforum-membership-reference`,
`sforum-notification-reference`, `sforum-prebuilt-settings`. **Missing:** a bare
public page fixture and a session-state workflow fixture.

**Important limitation for the M9 exit gate:** the repo fixtures are CI fixtures,
not clean-directory author packages. Running the author chain on copies in `/tmp`
fails for most of them at `digest --write` because they declare `backend/plugin`
but ship no binary (`lstat …/backend/plugin: no such file or directory`) — the Go
integration tests build it — and their backend `go.mod` uses
`replace … => ../../../../../apps/api`, so they only compile at that depth inside
the repository. The exit gate's "clean temporary directory" path is satisfied by
generated scaffolds (verified above), not by copying repo fixtures.

## Round 8 (same day): M9 reference package for the layout contract

`extensions/fixtures/plugins/sforum-page-layout-reference` closes M9 task 1's "bare
public page" bullet with one focused package that declares the same public page
three ways: `layout: "bare"`, `layout: "site"`, and no `layout` at all. It passes
`extension digest --write`, `extension validate` and `extension test`
(0 errors, 0 warnings, 4 checks, including `template.runtime_exact`).

Two fixture-level tests (in `app/Support/Pages/page_layout_reference_fixture_test.go`)
pin the author-facing path rather than re-testing the internal helper:

- real activation preflight preserves the declaration (bare / site / empty), then
  `prepareContributions` normalizes the omitted case to the platform default `site`;
- three-way template identity on the committed fixture (one `templates[]`
  declaration + one `packageFiles[]` kind `template` per `theme.json` template path,
  digests matching the files).

**Correction recorded:** `ThemePageDecl.Layout`'s comment said "empty is equivalent
to bare (backward compatible)", which contradicts `NormalizeLayout` (empty → `site`)
and the M2 runtime probes. The comment was stale and is now corrected; the code was
already right.

Still missing from M9 task 1: a session-state workflow fixture (no session contract
exists, so it cannot be built), and the migration guide's reference-package list
now points at the layout fixture.

## Remaining M5 work

- **Task 7** — complete for the surfaces M5 affects: the quarantine panel ships
  and renders loading / empty / quarantined / recovery, and stale / error are
  contract-tested (see the coverage note above). No other inspector is missing a
  state that its API can disclose (loading/empty/error exist everywhere;
  Safe Mode is rendered by exactly the inspectors whose payload exposes it;
  route-inspector has the only staleness concept).
- **Task 8** — 390-wide admin QA is still carrier-blocked (below); desktop 1313x930
  is done for both the guard-authorization dialog and the quarantine panel.
- **Task 4 (partial)** — decide whether per-request plugin-guard **denials** must
  be audited; today only trust decisions and route runtime incidents are.

## Open Questions / Blockers

- **Carrier limitation (unchanged):** the sidebar carrier clamps the viewport to
  500x846, and the DSH built-in carrier reports 1313x930 with no resize or
  emulation control. Genuine 390x844 admin QA therefore needs an authenticated
  real-Chrome session; cookie export is refused by design and the session cookie
  is httpOnly, so it cannot be transplanted without handling live credentials.
  Public-page 390x844 evidence exists from earlier rounds (real Chrome + CDP).
- R3 (empty `core.guard.raw_request` binding mints raw request authority with
  zero guard code) is still an operator decision; this session only made it
  *visible* in the impact view and the route inspector (risk `raw_request`).
- Stray dev fixtures now installed but inert: `acme.iconprobe` (enabled, user
  asked to keep), `acme.guardprobe` (installed, disabled — this session's probe),
  `acme.switchtest` (disabled leftover theme). Ask before removing any.
