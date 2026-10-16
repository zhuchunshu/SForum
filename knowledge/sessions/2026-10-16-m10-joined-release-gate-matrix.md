# M10 Joined Release Gate — Acceptance Matrix (Honest Status)

- Date: 2026-10-17 (**Revision 5** — row 7 closed with real two-node evidence:
  two API nodes from the published `v3.0.9` images sharing one durable
  PostgreSQL + Redis, run on this machine; plus two measured cold-start
  operational findings with mitigations recorded in the runbook)
- Status: **closed**. All ten rows are evidenced. Row 7 (true multi-node) was
  closed on 2026-10-17 with two real nodes (see "Revision 5: row 7 closed" below).
  The plan can move to `archive/` — nothing in this program remains open.
- Owner: closed. The former gap (row 7) was resolved with local two-node
  evidence; its durable per-node machinery (leases, acks, poll/LISTEN
  convergence) was verified against published images.

This matrix is the M10 deliverable: for every required row it states the evidence
that exists, the evidence that does not, and why. Rows are never marked complete
from source-only or Support-only tests.

## Second scoring pass (2026-10-16, later same day): what changed and why

Revision 1 was written from a partial evidence set and went stale. Revision 2
re-measured every row against the running dev instance and the current tree.
The differences, with their cause, because a matrix that silently changes its
mind is worthless:

| Row | Revision 1 said | Revision 2 says | Why it moved |
| --- | --- | --- | --- |
| 1 | partially evidenced; one clause contradicts the repo | **evidenced** | Decision **R5** reworded the row to "the existing protected built-in theme". The repo has exactly one, so the row is now satisfiable as written and is satisfied by the runtime evidence below. |
| 2 | **blocked** for the data half | **evidenced** (both halves) | Decision **R2** replaced the missing loopback transport with a Protocol V2 call, and the A1 reference package proves it end to end on the live instance. Revision 2 also found that the override half already had a real-HTTP test and cited it. |
| 4 | partially evidenced; the `challenge` clause does not exist | **evidenced** | Decision **R4** removed the `challenge` clause from the row rather than changing the guard contract. Every remaining clause is cited and was re-run green. |
| 8 | mobile width blocked by the carrier | **evidenced** | The real-Chrome CDP harness measures 390x844 directly (the "carrier" limit was the `ego` sidebar only). Revision 2 additionally fixed a **harness defect** that had produced a false negative on the `view: component` page (see "Harness defect" below) and now measures the component mount at 390x844. |
| 9 | "was not run" | **evidenced** | Correction: it **had** been run (api/migrate/web candidate targets, isolated database, `APP_ENV=production`, `/api/v1/health` → 200, plus seeded and uploaded-package runs). Revision 1 misreported it as "needs external conditions". |
| 5, 7 | blocked / partially evidenced | row 5's **blocker type changed**, row 7 unchanged | Row 5 is still unevidenced (the capability does not exist), but the contract it needs was authorised later the same day and decided in Revision 2 of the frozen contract record — so the remaining work is building M6, not making a decision. Row 7 still has no multi-node environment. Neither is a test gap. |

Two further corrections inside Revision 2:

1. **Row 2's override half is in-process real HTTP, not a live theme switch.** `TestResolvePathUsesActiveThemeOverrideForPluginBusinessPage` runs the production controller through a real HTTP request (`Controller.RegisterRoutes` + `app.Test`, real theme runtime registry stage/activate). It is production-path evidence but it is *not* a live dev-instance theme activation; that residual is stated in the row rather than glossed.
2. **The row-2 block name is gone.** Revision 1 titled the section "Plugin body data" because it had no data to show. It now shows the live resolve response and the browser DOM.

### Harness defect found and fixed in this pass (public 390x844)

`tests/browser-public-390x844.mjs` decided the page had "settled" by waiting for
`document.body.innerText.length` to stop changing. **That condition is satisfied
before hydration**: the SSR text is already stable, so the two-stable-polls exit
fired at ~2 s. On the `view: component` page the client mount happens later, so
the harness measured *before* the component mounted and reported "no descriptor
request was issued at all".

That observation had previously been recorded (in the adjacent task book) as
possibly a real client gap, and raising `SFORUM_SETTLE_MS` appeared not to help —
correctly, because the exit condition was "2 stable polls", not the budget.

Measured discriminator (same context, same page, 390 wide):

```text
harness settle exit (old logic)      ≈ 1 982 ms   → l2State=idle,  0 descriptor requests
component actually mounts            at 5 427 ms  → l2State=mounted, kids=1
descriptor requests observed         at 5 427 ms  → 200 /api/v1/extensions/runtime/…/components/…
```

So it was a **measurement artefact, not a client-side gap** — and the earlier
"not proven" note in the adjacent task book is now resolved as such rather than
left standing. The harness now (a) waits for real hydration
(`#__nuxt.__vue_app__`), (b) requires a terminal public-L2 state
(`mounted|fallback|quarantined`, never `idle|loading`) before it accepts
stability, and (c) can assert a mount outright via `SFORUM_L2_PATHS`, failing
loudly when hydration never happens instead of silently passing. A negative
control (`SFORUM_L2_PATHS=/probe-site`) was run and does FAIL, so the assertion
is not vacuous.

## Evidence collected in Revision 1's pass (2026-10-16)

Runtime, against the live dev instance (web `:3000`, API `:8080`):

| Probe | Command / method | Result |
| --- | --- | --- |
| SSR chrome, `site` | `curl :3000/probe-site` | `<nav>` ×2 (desktop + mobile, one navbar component, shared scope `data-v-79847b00`), `<footer class="sf-footer">` ×1 |
| SSR chrome, `bare` | `curl :3000/probe-bare` | 0 `<nav>`, 0 `<footer>` |
| SSR chrome, default layout | `curl :3000/probe-default` | same as `site` (layout omitted → platform default) |
| Visible chrome at 500px | `ego` browser, `getComputedStyle` + `getBoundingClientRect` | `/probe-site`: visibleNavs=1, visibleFooters=1, `scrollWidth == innerWidth == 500` (no overflow) |
| Visible chrome, bare | same | `/probe-bare`: visibleNavs=0, visibleFooters=0, `data-registry-shell-source="none"`, no overflow |
| Selected provider/digest | `GET /api/v1/pages/resolve-path?path=/probe-site` | `layout=site`, `selectedProvider=acme.iconprobe`, `selectedPackageDigest=9dfe2359…`, `templatePath=templates/probe-site.html`, `shellProvider=sforum.default-theme`, `shellVersion=1.0.0`, `shellPackageDigest=85ddda6a…`, `fallback=false` |
| Bare page identity | same, `path=/probe-bare` | `layout=bare`, `selectedProvider=acme.iconprobe`, `shellHtml` empty, no shell identity (correct: bare has no theme shell) |
| Active theme artifact | `GET /api/v1/site/active-theme/skin` | `extensionId=sforum.default-theme`, `version=1.0.0`, `packageDigest=85ddda6a…` — **agrees** with `resolve-path`'s `shellPackageDigest` |
| Compatibility farm | `./scripts/test-api-isolated.sh --compat` | `exit=0`, `[pass] current-host-v2-manifest-v3`, `[pass] manifest-v3-contract`, `ok: true` |
| APILTS | `sforum extension api-lts` | `sforum.host.v2` current, `sforum.manifest.v3` current, `sforum.protocol.v2` current, `sforum.theme.l1.request-time-loader` deprecated **with** shim and replacement `sforum.theme.runtime-snapshot` |
| Gate suites | `go test ./app/Support/Pages/... ./app/Support/ThemeCompiler/...`, `./app/Support/Extensions/...`, `./app/Support/AuthSession/... ./app/Support/HostAPI/...` | all `ok` (Extensions incl. IntegrationTests 59.8s) |

## Evidence collected in Revision 2's pass (2026-10-16, re-run)

| Probe | Command | Fresh result |
| --- | --- | --- |
| Shell composition, live SSR + resolve API | `node tests/registry-page-shell-runtime-evidence.mjs` | `/probe-site` and `/probe-default` → `shell=theme navbarOwners=1 footers=1 layout=site shell=sforum.default-theme`; `/probe-bare` → `shell=none 0/0 layout=bare`; active theme `sforum.default-theme@1.0.0 digest=85ddda6ad7825ada`; `all Route A composition assertions passed` |
| Active theme artifact | `GET /api/v1/site/active-theme/skin` | `sforum.default-theme` / `1.0.0` / `85ddda6ad7825adaacfa5f741715b64a09737b41320e1a49848ca3e695edccac` |
| Plugin page typed data (live) | `GET /api/v1/pages/resolve-path?path=/loader-articles/hello` | `layout=site`, `fallback=false`, `reason=null`, `loaderError=null`, `dataSource=plugin`, `dataRoute=/page-data/article`, `selectedProvider=sforum.page-loader-reference-live`, `selectedVersion=1.0.1`, `selectedPackageDigest=53af8e1ce208…`, `selectedRuntimeInstanceId=b0a6fc2b3b48ad930b180768cfc4a634`, `shellProvider=sforum.default-theme`, `shellPackageDigest=85ddda6a…`, `loaderData={"state":"published","title":"Loaded from the plugin runtime: hello"}` |
| Component page resolve + descriptor | `GET /api/v1/pages/resolve-path?path=/page-component-reference/component` | `view=component`, `component=sforum.page-component-reference.component.body`, `fallback=false`, `layout=site`, `shellProvider=sforum.default-theme` / `85ddda6a…`, `selectedPackageDigest=e46154d95d2f…` |
| Public 390x844, five paths incl. the component page | `SFORUM_PUBLIC_PATHS='/page-component-reference/component,/page-component-reference/template,/loader-articles/hello,/probe-site,/probe-bare' SFORUM_L2_PATHS='/page-component-reference/component' SFORUM_QA_DIR=storage/qa node tests/browser-public-390x844.mjs` | all five pass; component page `hydrated=true@1874ms … l2=mounted/kids=1 trustNotice=true marker="page component reference body"`; template page `l2=null`; `bare` 0/0; every path `overflow=false clipped=0 consoleErrors=0` |
| Public 390x844, JS disabled | `SFORUM_DISABLE_JS=1 node tests/browser-registry-page-shell-390x844.mjs` | `navbars=1 footers=1` on `site`/`default`, `0/0` on `bare`, `overflow=false clipped=0 consoleErrors=0` |
| Row 3 phase order | `go test ./app/Support/Routes/... -run 'TestExecutionPlanOrdersModifierPhasesForSafeAndUnsafeMethods\|TestExecutionPlanBuildsDeterministicCompletePhaseChain\|TestDispatcherExecutesBufferedChainInPlanOrder'` | 3 PASS |
| Row 4 CSRF order | `go test ./app/Http/ -run 'TestCSRF.\*\|TestP6RoutePermissionCSRFLocaleQueryAndBodyMatrix\|TestNewAppArbitraryAdminRoutePreservesCSRFAndPermissionAuthority'` | 5 PASS |
| Row 4 uninstall/restart, real PostgreSQL | `./scripts/test-api-isolated.sh ./app/Models/Extensions/ -run 'TestUninstall…\|TestLifecycle…\|Test…Restart…'` | 13 PASS on a disposable cluster (incl. `TestEnsureInitialPluginRuntimePublicationExactlyOnceAcrossRestart` 0.76 s) |
| Row 4 guard lifecycle/allow-deny | `go test ./app/Support/Extensions/... ./app/Support/AuthSession/... ./app/Support/HostAPI/...` and the guard suites | green (see the gate list at the end) |
| Row 5 session surface | `rg -i session apps/api/app/Support/ExtensionManifest/schemas/manifest-v3.schema.json`, `rg -rn session contracts/proto/` | `session` appears **only** inside `identityProviders[]` (`kind: "session"`, `operations: session.evaluate`, `sessionPolicy`) — a Host-invoked identity-provider policy hook; Protocol V2 declares `Actor.session_id` but the projection rejects a non-empty value (`sdk/plugin/v2/identity_runtime.go:235`), and Host API v2 has no session-state service |
| Row 6 dev loop | `bash tests/dev-loop-failure-scenarios.sh` | see the row-6 section below |
| Row 7 barrier/reconcile | `go test ./app/Support/Extensions/... -run 'TestManagerRuntimeSetBarrier\|TestPluginRuntimeFullSetBarrierWaitHonorsHeartbeatCancellation'` | 4 PASS (Go-level only) |
| Row 9 | release-image smoke (Revision 1's pass) | api/migrate/web candidate images + isolated seeded database + uploaded-package install/enable + executable trust gate + release compose stack (see the appendix sections) |
| Row 10 compat farm | `./scripts/test-api-isolated.sh --compat` | `ok: true`, both required cells `[pass]` |
| Row 10 APILTS | `go run ./cmd/sforum extension api-lts` | `host.v2`/`manifest.v3`/`protocol.v2` `current shim=false`; `theme.l1.request-time-loader` `deprecated shim=true replacement=sforum.theme.runtime-snapshot` |
| Reference packages | `bash scripts/ci/reference-packages-gate.sh` | 10/10 `digest + validate + test passed` |

## Matrix (Revision 2, re-scored)

| # | Required row | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Public page `site` and `bare` under the **existing** protected built-in theme | **evidenced** | Row wording per decision **R5**: the repository contains exactly one protected built-in theme (`extensions/builtin/themes/sforum-default`), so "both built-in themes" is not a satisfiable requirement and is not this program's scope. `site` and `bare` are verified against the live artifact by three independent probes: SSR chrome counts (`node tests/registry-page-shell-runtime-evidence.mjs` → one navbar owner + one footer on `site`/default, zero on `bare`, `data-registry-shell-source=none` on `bare`), `/pages/resolve-path` identity (`shellProvider=sforum.default-theme` with `shellPackageDigest=85ddda6a…`), and `/site/active-theme/skin` reporting the identical digest. Browser confirmation at a real 390x844 (below, row 8) and with JS disabled. |
| 2 | Plugin body data plus theme override without business-schema drift | **evidenced** (both halves; override half is in-process real HTTP) | **Data half, live instance:** `/pages/resolve-path?path=/loader-articles/hello` → `fallback=false`, `loaderError=null`, `loaderData={"state":"published","title":"Loaded from the plugin runtime: hello"}` from `selectedRuntimeInstanceId=b0a6fc2b…` on the plugin's own artifact `sforum.page-loader-reference-live@1.0.1 / 53af8e1c…`, and the same page's SSR/browser DOM shows the rendered text (`marker="plugin-loader: Loaded from the plugin runtime: hello"` at 390x844). Transport: decision **R2** made the Loader Gateway call the plugin over Protocol V2 (`InvokeRouteInstance`) for a declared `core.guard.public` GET data route. Pinned by `TestReferencePageLoaderFixtureServesTypedDataThroughProductionChain` (real V2 subprocess, full production chain) and `TestPageLoaderGatewayRefusesNonPublicDataRoutes` (undeclared / non-public / digests-drifted / unwired all fail closed 503 with no data) — both re-run green. **Override half:** `TestResolvePathUsesActiveThemeOverrideForPluginBusinessPage` (real HTTP through the production controller and a real theme runtime registry) proves an active-theme override wins with `source=active_theme_override`, that the plugin's `loaderData` (`title=Business truth`, `state=published`) is preserved unchanged, that the plugin template is bypassed, and that a **mismatched `themeOverrideKey` soft-skips back to the plugin template with the business semantics unchanged** — i.e. drift degrades presentation, never the plugin's versioned data contract. Fixture-level counterpart: `TestDurableFixturesProveThemeOverridePreservesPluginBusinessContract` (`extensions/fixtures/themes/sforum-plugin-override-e2e-theme`). **Residual, stated not glossed:** the override half was not exercised as a live dev-instance theme activation; its evidence is the in-process real-HTTP path. |
| 3 | Middleware `before`/`after`/`filter`/`wrap`/`global` ordering on safe and unsafe methods | **evidenced** | Phase order is one plan-order list (`execution_plan.go`): `global → before → filter → wrap → handler → after`, exactly one handler. Safe half `TestExecutionPlanBuildsDeterministicCompletePhaseChain`; unsafe half `TestExecutionPlanOrdersModifierPhasesForSafeAndUnsafeMethods` (POST plan marked unsafe, identical order, `global_middleware` exempt from method matching so it appears in both plans, method-specific modifiers only in their own plan); dispatcher-level order `TestDispatcherExecutesBufferedChainInPlanOrder`; priority permutations covered alongside. All three re-run PASS in Revision 2. |
| 4 | Custom guard allow/deny, CSRF order, timeout, crash, quarantine, revoke, disable, upgrade, rollback, uninstall, Safe Mode, restart | **evidenced** | Row wording per decision **R4**: the `challenge` clause was removed from the row instead of changing the guard contract (guard outcomes are allow or `denied`/`crash`/`timeout`/`protocol`/`canceled`/`unavailable`; there is no `challenge`). Clauses: **CSRF order** — `TestCSRFMintsTokenOnSafeRequest`, `TestCSRFRejectsUnsafeRequestWithoutToken`, `TestCSRFAcceptsUnsafeRequestWithValidToken`, `TestP6RoutePermissionCSRFLocaleQueryAndBodyMatrix`, `TestNewAppArbitraryAdminRoutePreservesCSRFAndPermissionAuthority` (5 PASS in Revision 2). **allow/deny/timeout/crash/Safe Mode/rollback/upgrade/quarantine** — the guard runtime + lifecycle suites in `./app/Support/Extensions/...` (incl. the M5 guard-binding publication tests, the third-theme-outside-the-fence control, and the durable incident ledger), plus `app/Http/plugin_guard_runtime_test.go` for the typed failure kinds. **uninstall (8) and restart (5)** — re-run green in Revision 2, the restart/uninstall set against a **disposable PostgreSQL cluster** via `./scripts/test-api-isolated.sh ./app/Models/Extensions/ …` (13 PASS, including the real-PostgreSQL `TestEnsureInitialPluginRuntimePublicationExactlyOnceAcrossRestart`). **Pointer:** `core.guard.raw_request` is no longer accepted as a route guard (decision **R1**, tightened as a vulnerability); raw authority requires a package-declared `kind: raw_request` guard. |
| 5 | Plugin session state across anonymous/authenticated, rotation, logout, revoke, expiration, privacy erase, disable, uninstall, runtime restart | **evidenced (Revision 4; anonymous clause closed as decision D1 — unavailable by construction, not satisfied)** | Superseded by Revision 4 (top of this file): the clause-by-clause evidence table lives there, including the compiled-fixture-binary joined test, the real-PostgreSQL Identity erasure matrix, the live `privacy:erase-user` run that erased a planted plugin-session key through the real Redis, and the Gateway-as-issuer production wiring fix (correction 1). Revision 3's status text below is kept for the audit trail of how the row was unblocked. |
| 6 | Dev loop success, compile failure, schema failure, crash, rapid edit, API restart, production rejection | **evidenced** | `tests/dev-loop-failure-scenarios.sh` (real `bun`, real digest/validate/test, real file events, real restart timing) asserts: startup reconcile builds; a package failure (declared `packageFiles` entry removed) reports `failed stage=build` and keeps `lastHealthyDigest`; recovery rebuilds; a frontend **compile failure** (`bun run build` → non-zero) keeps the last healthy digest; an **API restart** mid-loop (documented comment probe, `catalog.go` byte-identical afterwards) does not kill the loop; rename (delete+create) coalesces into one successful transaction; `SIGINT` exits cleanly. Unit-level orchestration contract, rapid-edit coalescing, deleted inputs, process-group cleanup and production/staging/test refusal live in `cmd/sforum/dev_extension_test.go`; production config additionally refuses a development scaffold root (`TestProductionRejectsDevelopmentScaffoldExternalRoots`). **Crash is bounded, not faked:** a *build* child crash is covered; a *plugin runtime* crash is not the loop's concern (it watches sources, it does not supervise the runtime) and on the API side is covered by the typed `crash` failure kind failing closed. |
| 7 | Multi-node desired revision, exact artifact, guard provider, shell provider, runtime acknowledgement consistency | **partially evidenced (Go-level); needs an external environment** | What exists: the runtime-set barrier/reconcile implementation with `TestManagerRuntimeSetBarrierSerializesFullSetWithLegacyStartStop`, `…SerializesFullSetWithStagedPublishAndStop`, `…ReconcileAndCloseDoNotReenter`, `TestPluginRuntimeFullSetBarrierWaitHonorsHeartbeatCancellation` (4 PASS in Revision 2); exact-artifact and provider identity verified live (`/pages/resolve-path` and `/site/active-theme/skin` agreeing on the shell provider/digest, plus the plugin's own artifact identity). What does **not** exist: any real multi-node deployment — no second node, no cross-node desired-revision propagation, no per-node runtime acknowledgement was exercised, and a single-process test must not be presented as multi-node evidence. Owner: operator-provided environment. |
| 8 | Browser desktop/mobile, SSR, JS-disabled, selected provider and digest, single chrome, no overflow, no console errors | **evidenced** | **SSR**: the persistent check `node tests/registry-page-shell-runtime-evidence.mjs` (one navbar owner + one footer on `site`/default, zero on `bare`, shell identity agreeing with the skin endpoint). **Desktop**: 1313x930 passes recorded earlier (Route Inspector/M5). **390x844 (real Chrome 154 over CDP, `Emulation.setDeviceMetricsOverride(390x844, dpr 3)`)** — Revision 2 re-ran it across five public paths including the `view: component` page: every path `overflow=false clipped=0 consoleErrors=0`, exactly one header and one footer on shell pages and zero on `bare`, and the component page reports `l2=mounted/kids=1 trustNotice=true marker="page component reference body"`. **JS disabled**: `SFORUM_DISABLE_JS=1 node tests/browser-registry-page-shell-390x844.mjs` still measures one navbar + one footer on `site`/`default`, zero on `bare`, correct shell markers and SSR body text — chrome and body come from the server render. **Selected provider/digest**: `/pages/resolve-path` + `/site/active-theme/skin` agree on `sforum.default-theme` / `85ddda6a…`; DOM markers are stated per page kind (a plugin page has no `data-provider`/`data-template="1"` because its body is the plugin's template — that is not claimed). **Admin**: `tests/browser-admin-390x844.mjs` at 390x844 dpr3 with an injected session cookie. **Revision 2 note:** the harness needed a fix to be able to measure the component mount at all — see "Harness defect found and fixed" above; the assertion has a negative control. |
| 9 | Candidate release image smoke | **evidenced** (Revision 1 misreported this as "not run") | Built both API targets and the web target (`--target api\|migrate\|prod`): `sforum-api:candidate-smoke` 620 MB, `sforum-migrate:candidate-smoke` 29.9 MB, `sforum-web:candidate-smoke` 261 MB, all exit 0. Migrated an **isolated** database (`sforum_smoke`, dropped afterwards, dev DB untouched): 145 tables + 7 River migrations. API image with `APP_ENV=production` served `GET /api/v1/health` → **200** `{"status":"ok","environment":"production"}`. Three production fail-closed gates demonstrated in order (placeholder secrets panic; missing `EXTENSION_ROOT` refused — the misleading `mkdir /storage: permission denied` was replaced by a named config error; missing `MARKETPLACE_ED25519_PUBLIC_KEY_HEX` refused). Web image served live SSR (`GET /` 200, control experiment: with the API container stopped the same request returns **000**). Then seeded (50 users / 6 categories / 8 topics) and real content served (`/`, `/c/general`, `/t/1/topic-ngkxas` all 200). Then the extension surface: the image synced its protected built-ins (8 plugins + 1 theme), ran 5 plugin backends from immutable digest paths, the enabled search plugin indexed the seeded topics; an **uploaded** package was installed (201) and enabled (200) through the release images and served its public pages with the layout contract intact; the **executable** trust gate was exercised on both sides (challenge 200, enable without token 409, with token 200, replay after `DELETE …/trust` 409); `compose.release.yaml` was exercised end to end (postgres/redis healthy, api ran its own migrations, health 200, web `/` 200). Residual: the registry pull of a real published tag, and `compose.prod.yaml`'s reverse-proxy/TLS layer. |
| 10 | Compatibility farm and APILTS for every changed public contract | **evidenced** | Farm: `./scripts/test-api-isolated.sh --compat` → `exit=0`, `[pass] current-host-v2-manifest-v3 status=required`, `[pass] manifest-v3-contract status=required`, `ok: true`. APILTS: `sforum.host.v2` / `sforum.manifest.v3` / `sforum.protocol.v2` all `current` with `shim=false`; `sforum.theme.l1.request-time-loader` `deprecated shim=true` with its replacement `sforum.theme.runtime-snapshot`. |

## Exit gate

**Met for every row except row 7 (as of Revision 4, 2026-10-17)**:

- **Row 5** — closed. Every clause has per-row evidence (see Revision 4); the
  anonymous clause is closed by operator decision D1 as "unavailable by
  construction, fail closed", not by behaviour, and reopening it means a new
  session-persistence ADR.
- **Row 9** — closed. Both Revision 3 residuals (real registry tag,
  reverse-proxy/TLS layer) were exercised on 2026-10-17; see Revision 4.
- **Row 7** — open, operator-owned. A second node is required; the Go-level
  barrier/reconcile tests are real but are not multi-node evidence. Nothing in
  this program remains to build for it.

No open P0/P1 finding remains inside this program. The plan may move to
`archive/` when the operator accepts row 7 as an environment dependency (the
only reason it has stayed out so far).

## Conflict with the task prompt's baseline (code wins)

The prompt states the active theme artifact digest is `4691a60541836c57`. At
runtime the active theme is `sforum.default-theme@1.0.0` with
`packageDigest=85ddda6ad7825adaacfa5f741715b64a09737b41320e1a49848ca3e695edccac`,
reported identically by `/site/active-theme/skin` and `/pages/resolve-path`. The
short digest in the prompt is stale; the runtime value is authoritative.

Two further task-book/repository conflicts were recorded in Revision 1 and are now
**settled by operator decisions rather than left standing**: "both protected built-in
themes" (the repository has exactly one — decision **R5** rewords the row to the
existing theme) and the guard `challenge` outcome (it does not exist in the contract —
decision **R4** removes the clause from row 4). Neither is a defect in the repository.

## Route A layer 6 (added 2026-10-16, same day)

- B2 (parse-time fail-open) was already covered by
  `TestResolvePathFailsOpenToHostShellWhenDeclaredShellDisappears`; no duplicate test
  was written.
- B1 is now `tests/registry-page-shell-runtime-evidence.mjs`: a persistent runtime
  check over real SSR HTML and the resolve API. First run failed for two instructive
  reasons (recorded): `data-registry-shell` carries the **layout**, not the shell
  source, and substring counting of `sf-public-mobile-bottom-nav` also matched its
  `__link` children (6 hits). It now classifies `<nav>` elements by exact class token
  and asserts: `site` → `theme` shell with exactly one desktop and one mobile nav plus
  one footer; `bare` → `none` with zero chrome; shell provider/digest agreement with
  `/site/active-theme/skin`; `SKIP` (exit 0) when the servers are down, failure when
  `SFORUM_EVIDENCE_REQUIRE_LIVE=1`.
- B3 is `knowledge/runbooks/2026-10-16-theme-switch-shell-verification.md`.
- A component-level render test for `SFRegistryPageShell.vue` was attempted and
  **abandoned**: `bun test` has no SFC transform (every existing component test reads
  source strings), so rendering-layer proof is runtime evidence in this repository.

## Release image smoke: defect found and fixed

The smoke exposed a misleading production failure mode. With `APP_ENV=production`
and no `EXTENSION_ROOT`, the non-production default (`../../storage/extensions`)
resolves against the container's working directory to `/storage/...`, so the
candidate image died with:

```
api bootstrap: Host platform services setup failed: create plugin files service:
plugin files input is invalid: mkdir /storage: permission denied
```

Nothing in that message names the missing variable. `config.Load` now refuses a
missing or relative `EXTENSION_ROOT` in production with
`config: EXTENSION_ROOT must be an absolute path in production (got "...")`
(`apps/api/config/config.go`), and
`TestLoadRejectsRelativeExtensionRootInProduction` pins it; the shared production
test fixture provides the value compose and the image already use
(`/var/lib/sforum/extensions`). Compose and the Dockerfile were already correct —
this only makes the misconfiguration self-explaining.

## Full three-image smoke (same day, follow-up)

The residual above is closed. All three candidate targets were built
(`sforum-api:candidate-smoke` 620MB, `sforum-migrate:candidate-smoke` 29.9MB,
`sforum-web:candidate-smoke` 261MB, all exit 0) and run together against an isolated
database:

```text
migrate image  → sforum_smoke: 145 tables + 7 River migrations
api image      → APP_ENV=production, GET /api/v1/health → 200 {"status":"ok","environment":"production"}
web image      → GET /:8098/ → 200, 81,120 bytes SSR HTML, title "SForum",
                 nav elements = 2 (one desktop navbar + one mobile navbar: the single
                 navbar owner's responsive pair), sf-footer present (2 in HTML, one
                 suppressed by the content-column rule, so one visible),
                 API-derived forum markup present (sf-home / sf-topic / sf-category)
```

**Control experiment (decisive):** with the API container stopped, `GET /` through the
web container returned **000** (no response); with the API running it returns 200.
So the web image's SSR genuinely depends on the API — the render is live SSR, not a
static shell, and the three images compose end to end.

Cleanup: both containers removed, `sforum_smoke` dropped (verified `count=0`), and the
three candidate images removed. They are reproducible with:

```bash
docker build --network=host --build-arg HTTP_PROXY=http://127.0.0.1:7897 --build-arg HTTPS_PROXY=http://127.0.0.1:7897 \
  --target api|migrate -f apps/api/Dockerfile -t sforum-api:candidate .
docker build --network=host --build-arg HTTP_PROXY=http://127.0.0.1:7897 --build-arg HTTPS_PROXY=http://127.0.0.1:7897 \
  --target prod -f apps/web/Dockerfile -t sforum-web:candidate .
```

## Seeded release-image smoke (same day, follow-up)

The empty-database limitation is closed. The three candidate images were rebuilt and
run against a **seeded** isolated database:

```text
host migrate CLI → sforum_smoke schema (7 River migrations)
host seed CLI    → 50 users (user#1 is the super_admin), 2 groups, 6 categories,
                   12 tags, 8 topics, 17 comments, 3 pinned
api image        → APP_ENV=production, GET /api/v1/health → 200
web image        → GET /            → 200, 127,220 bytes: 1 desktop navbar + 1 mobile
                                     navbar, footer present, 8 seeded topic links,
                                     4 seeded category links (announcement, devops,
                                     frontend, general)
                 → GET /c/general   → 200, 93,302 bytes, same chrome, seeded links
                 → GET /t/1/topic-ngkxas → 200, 132,703 bytes, seeded topic content
```

So the images serve real seeded forum data end to end (web SSR → API → Postgres),
not just an empty boot. Containers, database and images were removed afterwards.

## Release-image extension inventory and runtime (same day, follow-up)

The extension half of row 9 is now evidenced too, without needing an admin session:
the image's own boot does the work. After the api image booted against a seeded
isolated database:

```text
host CLI: sforum extension list (against the image-populated database)
ID                    TYPE    STATUS     SOURCE   VERSION  DIGEST
sforum-shortcodes     plugin  enabled    builtin  1.3.4    0ae602e8ae82
sforum.auth-github    plugin  installed  builtin  1.0.7    16b26728a0be
sforum.content-policy plugin  enabled    builtin  1.1.6    30b2ccd1b980
sforum.search-site    plugin  enabled    builtin  1.0.6    25077e2314ac
sforum.smtp           plugin  enabled    builtin  1.1.6    75b594511f8c
sforum.storage-fs     plugin  installed  builtin  1.1.7    4d776a2b0e68
sforum.storage-s3     plugin  installed  builtin  1.0.8    ccb1add788d6
sforum.web-push       plugin  enabled    builtin  1.0.6    c3460c16f712
sforum.default-theme  theme   enabled    builtin  1.0.0    85ddda6ad782
```

and the container was running the plugin code as subprocesses from the immutable
digest paths:

```text
PID   COMMAND
  24  /var/lib/sforum/extensions/sforum-shortcodes/1.3.4/0ae602e8…/backend/plugin
  38  /var/lib/sforum/extensions/sforum.content-policy/1.1.6/30b2ccd1…/backend/plugin
  52  /var/lib/sforum/extensions/sforum.search-site/1.0.6/25077e23…/backend/plugin
  66  /var/lib/sforum/extensions/sforum.smtp/1.1.6/75b59451…/backend/plugin
  77  /var/lib/sforum/extensions/sforum.web-push/1.0.6/c3460c16…/backend/plugin
```

with the enabled search plugin doing real work on the seeded data
(`search: indexed topic topicId=1..4`, `search reconciliation enqueued repairs
providerId=sforum.search-site indexJobs=4`). So the release image ships its protected
built-ins, syncs them into the database, enables the safe defaults, starts the plugin
runtimes, and serves seeded content through the web image.

## Install + enable through the release images (same day, follow-up)

Row 9's install/enable half is now **verified end to end** against the release images,
using the scripted admin credential added in the previous pass:

```text
users:reset-password --email seed_…@seed.local --password '…' --yes
  → 密码已重置：user_id=1 username=seed_… revoked_sessions=0
POST /api/v1/auth/login (csrf_ cookie + X-CSRF-Token + Origin) → 200, super_admin session
POST /api/v1/admin/extensions (multipart file=smoke-pkg.sforum.zip) → 201
  → {"id":"sforum.page-layout-reference","status":"installed","source":"uploaded"}
POST /api/v1/admin/extensions/sforum.page-layout-reference/enable
  {"confirmCapabilities":true} → 200 {"status":"enabled","source":"uploaded"}
host CLI: extension list → sforum.page-layout-reference plugin enabled uploaded 0.1.0 ce8ed23d9773
```

and the freshly installed, uploaded package's public pages then render through the
**web** image with the layout contract intact and the shell owned by the active theme:

```text
/page-layout-reference/bare    bytes=34046 desktopNav=0 mobileNav=0 footer=0 shellSource=none
/page-layout-reference/site    bytes=66752 desktopNav=1 mobileNav=1 footer=1 shellSource=theme
/page-layout-reference/default bytes=66778 desktopNav=1 mobileNav=1 footer=1 shellSource=theme
```

Two findings worth keeping:

1. **Correction to an earlier claim.** This matrix previously said the enable path
   needs "an authenticated `super_admin` session plus the exact-artifact trust
   challenge". That is only true for packages that actually ship an executable
   artifact. Both fixtures installed here declare `backend: {entry: "", rpc: ""}` and
   ship no backend file, so the correct answer is `409 extension.trust_not_required`
   and `enable` legitimately succeeds without a confirmation token.
2. **The trust gate is on in the smoke.** `SFORUM_V3_TRUST_CHALLENGES` defaults to
   `isProd` (`config.go:225`, `isProd := APP_ENV == "production"`), and the smoke runs
   `APP_ENV=production`, so `trust_not_required` here is a statement about the
   *package*, not a disabled gate.

## Executable trust gate through the release images (same day, follow-up)

The executable half is verified too. A package that actually ships an executable was
built for the smoke by copying the `sforum-shortcodes` backend under a new id
(`acme.trustprobe`), rewriting every id-bearing field, repointing the module `replace`
at the repository, and compiling it for linux:

```text
cd /tmp/trustprobe/backend && CGO_ENABLED=0 GOOS=linux go build -trimpath -buildvcs=false -ldflags="-s -w" -o plugin .
  → backend/plugin, 16,142,496 bytes
sforum extension digest --write /tmp/trustprobe → OK
sforum extension validate /tmp/trustprobe        → OK (backend: backend/plugin (hashicorp-go-plugin))
sforum extension test /tmp/trustprobe            → ✓ backend.rpc_ok, ✓ capabilities.resolved, ✓ manifest.ok, ✓ settings.renderer
sforum extension package /tmp/trustprobe         → 26 files, 18,628,537 bytes
```

Against the release image (`APP_ENV=production`, so `SFORUM_V3_TRUST_CHALLENGES`
defaults to true), the gate behaved exactly as designed on **both** sides:

```text
POST /admin/extensions (multipart)                → 201 installed, source=uploaded,
   manifest.backend = {entry: backend/plugin, rpc: hashicorp-go-plugin, protocolVersion: 2,
                       digest: da2c9b48c595a293…, hostApiVersion: sforum.host@2}
POST /admin/extensions/acme.trustprobe/trust/challenge → 200, token 64 chars,
   expiresAt = +5min, impact = sforum.trust-impact@2 {action: enable, extensionId, version,
   source, packageDigest}
POST …/enable {"confirmCapabilities": true}        → 409 extension.trust_challenge_required
   ("此扩展包含可执行代码。请由超级管理员审阅完整影响并生成一次性确认。")
POST …/enable + confirmationToken                  → 200 enabled
container process table                             → 6 plugin backends, including
   /var/lib/sforum/extensions/acme.trustprobe/1.3.4/9b1e202816de77cf…/backend/plugin
```

Two further behaviours were measured rather than assumed:

- After `disable`, re-enabling **without** a fresh token returned 200 — because the
  exact-artifact trust grant persists for that package digest. That is the designed
  behaviour, not a replay acceptance.
- `DELETE …/trust` → 200 with `trustRequired=true, trusted=false`; enabling with the
  already-consumed token then returned **409 `extension.trust_challenge_replayed`**
  ("确认已使用，不能重复提交，请重新生成。"), which is the one-time semantics holding.

## Release compose stack (same day, follow-up)

`compose.release.yaml` was exercised too, with the local candidate images standing in
for the registry tag. The override resolves to three published-image services with
`pull_policy: always` and **fails closed** without the tag:

```text
docker compose -f compose.yaml -f compose.release.yaml config   (no SFORUM_VERSION)
  → error while interpolating services.{api,migrate,web}.image:
    required variable SFORUM_VERSION is missing a value: Set SFORUM_VERSION to an immutable release tag
```

Run with `SFORUM_VERSION=smoke-local` and the candidate images tagged as
`ghcr.io/zhuchunshu/sforum-{api,migrate,web}:smoke-local` (a throwaway override in
`/tmp` set `pull_policy: never` and published web/postgres for probing):

```text
postgres/redis        → healthy
api                   → ran its own migrations (MIGRATE_ON_STARTUP=true),
                        GET /api/v1/health → 200 {"environment":"production"}
web                   → GET / → 200, one desktop navbar + one mobile navbar + footer;
                        /c/general → 200; after seeding + an API restart the home page
                        carried 3 seeded topic links
host CLI extension list → image built-ins: 5 enabled, 3 installed, 1 theme
```

Three operational findings, all measured:

1. **The dev `.env` must not be carried into the release stack.** Compose reads the
   repository-root `.env`, and its dev values override the compose defaults. Observed
   leaks: `EXTENSION_ROOT=../../storage/extensions` (relative → the API now refuses to
   boot in production, which is the fail-closed gate added earlier working as intended),
   `NUXT_API_INTERNAL_BASE_URL=http://127.0.0.1:8080/api/v1` (inside a container
   `127.0.0.1` is the container itself, so SSR returned 502/timeouts until it was
   overridden to `http://api:8080/api/v1`), `DATABASE_URL`/`REDIS_ADDR` pointing at host
   ports, and `EXTERNAL_EXTENSION_ROOTS` host paths (harmless
   `external extension source skipped` warnings).
2. **`down` needs the same `-f` set and `SFORUM_VERSION`.** Without them compose fails
   during interpolation and leaves containers and volumes behind.
3. **Seed before the API starts, or restart the API once afterwards.** The home page
   depends on the search index being reconciled: seeding after boot left it in an
   empty/error state (81,801 bytes, no category or topic links) until an API restart
   reconciled the index (106,147 bytes, 3 seeded topic links).

Runbook: `knowledge/runbooks/2026-10-16-release-image-stack-verification.md`.

Residual for row 9: the registry pull of a real published tag itself (this run used
locally built candidate images with `pull_policy: never`), and `compose.prod.yaml`'s
reverse-proxy/TLS layer.

## Row 6 fresh run (Revision 2 pass)

`bash tests/dev-loop-failure-scenarios.sh` was re-run against the live dev stack;
every scenario passed:

```text
dev loop failure scenarios (repo=/home/inkedus/Code/github/SForum)
Created plugin scaffold at extensions/dev/acme.row6scenarios
  ok   startup reconcile built the package
  ok   missing declared package file is reported
  ok   failure keeps the last healthy digest
  loop survived the package failure
  ok   the package recovers after the file returns
  ok   a failing frontend build is reported with its exit status
  loop survived the build failure
  catalog.go restored byte-identical after the restart probe
  ok   the loop keeps rebuilding after an API restart
  loop survived the API restart
  ok   a renamed source input rebuilds the package
  loop exited on SIGINT
  ok   the loop reports a stopped phase
dev loop scenarios: OK
```

The script restarts the dev API as part of scenario 5; afterwards `:8080` and
`:3000` were both confirmed live again (`api:200`, `web:200`) and
`git status` shows `catalog.go` with only the pre-existing session edit — the
probe's temporary comment was removed byte-identically.

## Environment anomaly observed at the end of Revision 1 (historical — resolved)

Revision 1 ended with the operator's Nuxt dev server down and the air-managed API
not listening. **Both servers were up for the whole Revision 2 pass** and every
runtime probe above was taken against them. The section is kept because it
explains why Revision 1 left rows 6/8 partly unevidenced. What Revision 1 saw:

- `http://127.0.0.1:3000/probe-site` → no response (the operator-owned Nuxt dev
  server is stopped; this session did not stop it and did not restart it).
- `http://127.0.0.1:8080/api/v1/health` → no response, while the air-managed process
  is alive (`air` 7h09m, plus an `sforum-api` child 2h57m with the correct env:
  `DATABASE_URL=postgres://sforum:sforum@127.0.0.1:15432/sforum`, `HTTP_PORT=8080`,
  `APP_ENV=development`) and `storage/logs/air.log` is a wall of `exit status 1`.
  Nothing is listening on 8080.
- Running the already-built binary manually with the documented env
  (`set -a; . .env; . storage/builtin-dev/air.env; ./tmp/sforum-api`) works: it
  bound `127.0.0.1:8080`, registered 564 handlers, and shut down cleanly on signal.
  So the code and configuration were healthy; the anomaly was in the air loop/process.

Revision 1's honest consequence at the time: the CDP 390x844 harness had passed at
09:26 while the servers were up, `tests/registry-page-shell-runtime-evidence.mjs`
later reported `SKIP`, and source-level gates were all that could be re-run.

## Defect found and fixed in the evidence harnesses themselves

While the web dev server was down, `tests/browser-registry-page-shell-390x844.mjs`
printed `SKIP ...` **and then** `all 390x844 acceptance assertions passed` with exit
0 — a false positive, because `finish()` claimed success whenever the failure list
was empty, including when nothing had been measured. Fixed in all three harnesses
(`browser-registry-page-shell-390x844.mjs`, `browser-admin-390x844.mjs`,
`registry-page-shell-runtime-evidence.mjs`): an `asserted` flag is only set once
measurement actually starts, and the success line is replaced by
`skipped: no assertion ran` otherwise. Verified now, with the web server down:

```text
SKIP the 390x844 pass needs the running web dev server (web=http://127.0.0.1:3000)
registry page shell 390x844 browser evidence (web=http://127.0.0.1:3000)
  skipped: no assertion ran          # exit 0
SFORUM_EVIDENCE_REQUIRE_LIVE=1 ...   # exit 1
```

A harness that claims success while skipping is worse than no harness, so this is
recorded as a defect rather than a cosmetic change.

## Web dev server restarted by this session (2026-10-16, reported)

The operator-owned Nuxt dev server on `:3000` had been down since roughly 12:19 (it
was already down when the previous pass ended, and two passes asked the operator to
restart it). Because the remaining acceptance items in row 8 and A3/A4 need it, this
session started it with the repository's documented command
(`cd apps/web && bun run dev`, log at `/tmp/web-dev-relaunch.log`) and re-ran the
evidence with it up:

- `node tests/registry-page-shell-runtime-evidence.mjs` → all assertions passed
  (site/default `shell=theme`, one navbar owner, one footer; bare `none`/0/0; active
  theme `sforum.default-theme@1.0.0` digest `85ddda6a…` agreeing across endpoints).
- `node tests/browser-registry-page-shell-390x844.mjs` → 390x844 dpr3 pass, and with
  `SFORUM_DISABLE_JS=1` the JS-disabled pass.

The operator can stop it at any time; it is not managed by this session's tooling.

## M8 residuals closed (2026-10-16)

The two items M8 listed as not done are now settled, one by delivery and one by an
explicit boundary:

1. **Admin-activation automation within the ADR.** `tests/admin-extension-activation.sh`
   drives the approved HTTP flow — install (inert) → exact-artifact trust challenge →
   enable with the confirmation token — with a `super_admin` session supplied either as
   a cookie or as email/password. It fails an executable package that cannot obtain a
   token, accepts the server's `extension.trust_not_required` verdict for packages that
   ship no executable artifact (that is the correct verdict, not a bypass), and can
   `disable` + `uninstall` afterwards. Verified green against the dev API with a
   throwaway package: `201` inert → `trust_not_required` → enable `200` → disable `200`
   → uninstall `200`, with the ledger and `storage/extensions/` clean afterwards.
2. **Runtime crash, bounded rather than faked.** `tests/dev-loop-failure-scenarios.sh`
   now also covers a **rename** (`delete`+`create` coalesced into one successful
   transaction), alongside the earlier build-crash, package-failure, API-restart and
   SIGINT cases. A **plugin runtime** crash is not the dev loop's concern — the loop
   watches sources and does not supervise the runtime — and on the API side it is
   already covered: the typed `crash` failure kind fails closed to
   `ErrCoreGuardEvaluatorUnavailable` without leaking the error text
   (`app/Http/plugin_guard_runtime_test.go`, `core_guard_failure_mapping_test.go`).## Revision 5 (2026-10-17): row 7 closed with real two-node evidence

The architecture behind row 7 was already implemented and durable: each node
registers a boot lease in shared PostgreSQL
(`plugin_runtime_nodes`: `node_id = hostname`, `process_role`, `boot_id`, a 45s
lease renewed every 10s), the lifecycle publishes immutable full-set
publications with a revision, every node writes a per-node per-revision
acknowledgement (`plugin_runtime_publication_acks`), and each node runs a
coordinator that converges to the latest publication by 5s durable polling plus
a LISTEN wake hint (LISTEN degradation is a warn, polling remains the
correctness path). Two nodes were run for real on this machine from the
published `v3.0.9` images sharing one disposable PostgreSQL + Redis.

Probe table (all measurements against the live two-node topology):

| # | Probe | Measured result |
| --- | --- | --- |
| P1 | Two-node registration | `plugin_runtime_nodes` has both `node-a|api` and `node-b|api` rows with distinct boot ids and live leases; both `lease_live=t` while running. |
| P2 | Identical publication revision acked by both nodes | revision 1 acked `applied` by both (node-a 17:35:20 / node-b 17:37:42). |
| P3 | Cross-node disable | `disable sforum.web-push` via node A's admin API → immutable desired revision 2 → both nodes ack `applied` and node B's `web-push` plugin subprocess is gone (verified with `ps`). |
| P4 | Cross-node enable | re-enable on node A → revision 3 → node B acks `applied` and re-runs the runtime from the **identical immutable artifact path** (`…/sforum.web-push/1.0.2/a982b361…/backend/plugin`) on both nodes — cross-node exact-artifact identity confirmed byte-for-byte by path digest. |
| P5 | Node departs | `docker compose stop api-b` → node B's lease expires within the 45s window (`lease_expired=t`, `last_seen_at` stops advancing); node A keeps serving (`/api/v1/health` 200) throughout. |
| P6 | Node rejoins | node B restarts: fresh boot id registers and immediately acks `applied` at the latest revision; runtimes revived on B; the orphaned boot row is retained until its lease expiry instead of being falsified. |

Two cold-start operational findings were also measured (both are mitigated by
the documented deploy path, and both are now written into the runbook):

1. **Cold-start first-migration deadlock** — two nodes booting simultaneously
   against an empty schema with `MIGRATE_ON_STARTUP=true` can deadlock: node A's
   goose session advisory lock is held by one idle pool connection while its
   `CREATE INDEX CONCURRENTLY` waits for node B's already-open transaction, and
   node B's transaction waits for node A's advisory lock; without timeouts it
   does not resolve on its own. Mitigation: deploy-path order (migrate image
   first, or start one node to `health 200` before the second; or
   `MIGRATE_ON_STARTUP=false` on subsequent nodes).
2. **Simultaneous first-boot order race** — two nodes racing the very first
   boot tripped the legacy-mail adoption precondition
   (`builtin smtp plugin is not synchronized`) before builtin sync completed.
   Sequential first boot avoids it; recorded for the deploy runbook (also
   don't omit `BUILTIN_EXTENSION_ROOT`). The earlier release smoke never hit
   these because the release compose starts `migrate` as an explicit step, and
   neither finding involves the coordinator machinery that row 7 probes.

Runbook with the reproduction steps:
`knowledge/runbooks/2026-10-16-release-image-stack-verification.md`
(“双节点多运行验证（2026-10-17，M10 行 7 的真两节点证据）”).

## Historical revisions (1–3, 2026-10-16) — superseded by Revision 5 above (rows 1–3 audit trail only)26-10-17 (**Revision 5** — row 7 closed with real two-node evidence:
  two API nodes from the published `v3.0.9` images sharing one durable
  PostgreSQL + Redis, run on this machine; plus two measured cold-start
  operational findings with mitigations recorded in the runbook)
- Status: **closed**. All ten rows are evidenced. Row 7 (true multi-node) was
  closed on 2026-10-17 with two real nodes (see "Revision 5: row 7 closed" below).
  The plan can move to `archive/` — nothing in this program remains open.
- Owner: closed. The former gap (row 7) was resolved with local two-node
  evidence; its durable per-node machinery (leases, acks, poll/LISTEN
  convergence) was verified against published images.

This matrix is the M10 deliverable: for every required row it states the evidence
that exists, the evidence that does not, and why. Rows are never marked complete
from source-only or Support-only tests.

## Second scoring pass (2026-10-16, later same day): what changed and why

Revision 1 was written from a partial evidence set and went stale. Revision 2
re-measured every row against the running dev instance and the current tree.
The differences, with their cause, because a matrix that silently changes its
mind is worthless:

| Row | Revision 1 said | Revision 2 says | Why it moved |
| --- | --- | --- | --- |
| 1 | partially evidenced; one clause contradicts the repo | **evidenced** | Decision **R5** reworded the row to "the existing protected built-in theme". The repo has exactly one, so the row is now satisfiable as written and is satisfied by the runtime evidence below. |
| 2 | **blocked** for the data half | **evidenced** (both halves) | Decision **R2** replaced the missing loopback transport with a Protocol V2 call, and the A1 reference package proves it end to end on the live instance. Revision 2 also found that the override half already had a real-HTTP test and cited it. |
| 4 | partially evidenced; the `challenge` clause does not exist | **evidenced** | Decision **R4** removed the `challenge` clause from the row rather than changing the guard contract. Every remaining clause is cited and was re-run green. |
| 8 | mobile width blocked by the carrier | **evidenced** | The real-Chrome CDP harness measures 390x844 directly (the "carrier" limit was the `ego` sidebar only). Revision 2 additionally fixed a **harness defect** that had produced a false negative on the `view: component` page (see "Harness defect" below) and now measures the component mount at 390x844. |
| 9 | "was not run" | **evidenced** | Correction: it **had** been run (api/migrate/web candidate targets, isolated database, `APP_ENV=production`, `/api/v1/health` → 200, plus seeded and uploaded-package runs). Revision 1 misreported it as "needs external conditions". |
| 5, 7 | blocked / partially evidenced | row 5's **blocker type changed**, row 7 unchanged | Row 5 is still unevidenced (the capability does not exist), but the contract it needs was authorised later the same day and decided in Revision 2 of the frozen contract record — so the remaining work is building M6, not making a decision. Row 7 still has no multi-node environment. Neither is a test gap. |

Two further corrections inside Revision 2:

1. **Row 2's override half is in-process real HTTP, not a live theme switch.** `TestResolvePathUsesActiveThemeOverrideForPluginBusinessPage` runs the production controller through a real HTTP request (`Controller.RegisterRoutes` + `app.Test`, real theme runtime registry stage/activate). It is production-path evidence but it is *not* a live dev-instance theme activation; that residual is stated in the row rather than glossed.
2. **The row-2 block name is gone.** Revision 1 titled the section "Plugin body data" because it had no data to show. It now shows the live resolve response and the browser DOM.

### Harness defect found and fixed in this pass (public 390x844)

`tests/browser-public-390x844.mjs` decided the page had "settled" by waiting for
`document.body.innerText.length` to stop changing. **That condition is satisfied
before hydration**: the SSR text is already stable, so the two-stable-polls exit
fired at ~2 s. On the `view: component` page the client mount happens later, so
the harness measured *before* the component mounted and reported "no descriptor
request was issued at all".

That observation had previously been recorded (in the adjacent task book) as
possibly a real client gap, and raising `SFORUM_SETTLE_MS` appeared not to help —
correctly, because the exit condition was "2 stable polls", not the budget.

Measured discriminator (same context, same page, 390 wide):

```text
harness settle exit (old logic)      ≈ 1 982 ms   → l2State=idle,  0 descriptor requests
component actually mounts            at 5 427 ms  → l2State=mounted, kids=1
descriptor requests observed         at 5 427 ms  → 200 /api/v1/extensions/runtime/…/components/…
```

So it was a **measurement artefact, not a client-side gap** — and the earlier
"not proven" note in the adjacent task book is now resolved as such rather than
left standing. The harness now (a) waits for real hydration
(`#__nuxt.__vue_app__`), (b) requires a terminal public-L2 state
(`mounted|fallback|quarantined`, never `idle|loading`) before it accepts
stability, and (c) can assert a mount outright via `SFORUM_L2_PATHS`, failing
loudly when hydration never happens instead of silently passing. A negative
control (`SFORUM_L2_PATHS=/probe-site`) was run and does FAIL, so the assertion
is not vacuous.

## Evidence collected in Revision 1's pass (2026-10-16)

Runtime, against the live dev instance (web `:3000`, API `:8080`):

| Probe | Command / method | Result |
| --- | --- | --- |
| SSR chrome, `site` | `curl :3000/probe-site` | `<nav>` ×2 (desktop + mobile, one navbar component, shared scope `data-v-79847b00`), `<footer class="sf-footer">` ×1 |
| SSR chrome, `bare` | `curl :3000/probe-bare` | 0 `<nav>`, 0 `<footer>` |
| SSR chrome, default layout | `curl :3000/probe-default` | same as `site` (layout omitted → platform default) |
| Visible chrome at 500px | `ego` browser, `getComputedStyle` + `getBoundingClientRect` | `/probe-site`: visibleNavs=1, visibleFooters=1, `scrollWidth == innerWidth == 500` (no overflow) |
| Visible chrome, bare | same | `/probe-bare`: visibleNavs=0, visibleFooters=0, `data-registry-shell-source="none"`, no overflow |
| Selected provider/digest | `GET /api/v1/pages/resolve-path?path=/probe-site` | `layout=site`, `selectedProvider=acme.iconprobe`, `selectedPackageDigest=9dfe2359…`, `templatePath=templates/probe-site.html`, `shellProvider=sforum.default-theme`, `shellVersion=1.0.0`, `shellPackageDigest=85ddda6a…`, `fallback=false` |
| Bare page identity | same, `path=/probe-bare` | `layout=bare`, `selectedProvider=acme.iconprobe`, `shellHtml` empty, no shell identity (correct: bare has no theme shell) |
| Active theme artifact | `GET /api/v1/site/active-theme/skin` | `extensionId=sforum.default-theme`, `version=1.0.0`, `packageDigest=85ddda6a…` — **agrees** with `resolve-path`'s `shellPackageDigest` |
| Compatibility farm | `./scripts/test-api-isolated.sh --compat` | `exit=0`, `[pass] current-host-v2-manifest-v3`, `[pass] manifest-v3-contract`, `ok: true` |
| APILTS | `sforum extension api-lts` | `sforum.host.v2` current, `sforum.manifest.v3` current, `sforum.protocol.v2` current, `sforum.theme.l1.request-time-loader` deprecated **with** shim and replacement `sforum.theme.runtime-snapshot` |
| Gate suites | `go test ./app/Support/Pages/... ./app/Support/ThemeCompiler/...`, `./app/Support/Extensions/...`, `./app/Support/AuthSession/... ./app/Support/HostAPI/...` | all `ok` (Extensions incl. IntegrationTests 59.8s) |

## Evidence collected in Revision 2's pass (2026-10-16, re-run)

| Probe | Command | Fresh result |
| --- | --- | --- |
| Shell composition, live SSR + resolve API | `node tests/registry-page-shell-runtime-evidence.mjs` | `/probe-site` and `/probe-default` → `shell=theme navbarOwners=1 footers=1 layout=site shell=sforum.default-theme`; `/probe-bare` → `shell=none 0/0 layout=bare`; active theme `sforum.default-theme@1.0.0 digest=85ddda6ad7825ada`; `all Route A composition assertions passed` |
| Active theme artifact | `GET /api/v1/site/active-theme/skin` | `sforum.default-theme` / `1.0.0` / `85ddda6ad7825adaacfa5f741715b64a09737b41320e1a49848ca3e695edccac` |
| Plugin page typed data (live) | `GET /api/v1/pages/resolve-path?path=/loader-articles/hello` | `layout=site`, `fallback=false`, `reason=null`, `loaderError=null`, `dataSource=plugin`, `dataRoute=/page-data/article`, `selectedProvider=sforum.page-loader-reference-live`, `selectedVersion=1.0.1`, `selectedPackageDigest=53af8e1ce208…`, `selectedRuntimeInstanceId=b0a6fc2b3b48ad930b180768cfc4a634`, `shellProvider=sforum.default-theme`, `shellPackageDigest=85ddda6a…`, `loaderData={"state":"published","title":"Loaded from the plugin runtime: hello"}` |
| Component page resolve + descriptor | `GET /api/v1/pages/resolve-path?path=/page-component-reference/component` | `view=component`, `component=sforum.page-component-reference.component.body`, `fallback=false`, `layout=site`, `shellProvider=sforum.default-theme` / `85ddda6a…`, `selectedPackageDigest=e46154d95d2f…` |
| Public 390x844, five paths incl. the component page | `SFORUM_PUBLIC_PATHS='/page-component-reference/component,/page-component-reference/template,/loader-articles/hello,/probe-site,/probe-bare' SFORUM_L2_PATHS='/page-component-reference/component' SFORUM_QA_DIR=storage/qa node tests/browser-public-390x844.mjs` | all five pass; component page `hydrated=true@1874ms … l2=mounted/kids=1 trustNotice=true marker="page component reference body"`; template page `l2=null`; `bare` 0/0; every path `overflow=false clipped=0 consoleErrors=0` |
| Public 390x844, JS disabled | `SFORUM_DISABLE_JS=1 node tests/browser-registry-page-shell-390x844.mjs` | `navbars=1 footers=1` on `site`/`default`, `0/0` on `bare`, `overflow=false clipped=0 consoleErrors=0` |
| Row 3 phase order | `go test ./app/Support/Routes/... -run 'TestExecutionPlanOrdersModifierPhasesForSafeAndUnsafeMethods\|TestExecutionPlanBuildsDeterministicCompletePhaseChain\|TestDispatcherExecutesBufferedChainInPlanOrder'` | 3 PASS |
| Row 4 CSRF order | `go test ./app/Http/ -run 'TestCSRF.\*\|TestP6RoutePermissionCSRFLocaleQueryAndBodyMatrix\|TestNewAppArbitraryAdminRoutePreservesCSRFAndPermissionAuthority'` | 5 PASS |
| Row 4 uninstall/restart, real PostgreSQL | `./scripts/test-api-isolated.sh ./app/Models/Extensions/ -run 'TestUninstall…\|TestLifecycle…\|Test…Restart…'` | 13 PASS on a disposable cluster (incl. `TestEnsureInitialPluginRuntimePublicationExactlyOnceAcrossRestart` 0.76 s) |
| Row 4 guard lifecycle/allow-deny | `go test ./app/Support/Extensions/... ./app/Support/AuthSession/... ./app/Support/HostAPI/...` and the guard suites | green (see the gate list at the end) |
| Row 5 session surface | `rg -i session apps/api/app/Support/ExtensionManifest/schemas/manifest-v3.schema.json`, `rg -rn session contracts/proto/` | `session` appears **only** inside `identityProviders[]` (`kind: "session"`, `operations: session.evaluate`, `sessionPolicy`) — a Host-invoked identity-provider policy hook; Protocol V2 declares `Actor.session_id` but the projection rejects a non-empty value (`sdk/plugin/v2/identity_runtime.go:235`), and Host API v2 has no session-state service |
| Row 6 dev loop | `bash tests/dev-loop-failure-scenarios.sh` | see the row-6 section below |
| Row 7 barrier/reconcile | `go test ./app/Support/Extensions/... -run 'TestManagerRuntimeSetBarrier\|TestPluginRuntimeFullSetBarrierWaitHonorsHeartbeatCancellation'` | 4 PASS (Go-level only) |
| Row 9 | release-image smoke (Revision 1's pass) | api/migrate/web candidate images + isolated seeded database + uploaded-package install/enable + executable trust gate + release compose stack (see the appendix sections) |
| Row 10 compat farm | `./scripts/test-api-isolated.sh --compat` | `ok: true`, both required cells `[pass]` |
| Row 10 APILTS | `go run ./cmd/sforum extension api-lts` | `host.v2`/`manifest.v3`/`protocol.v2` `current shim=false`; `theme.l1.request-time-loader` `deprecated shim=true replacement=sforum.theme.runtime-snapshot` |
| Reference packages | `bash scripts/ci/reference-packages-gate.sh` | 10/10 `digest + validate + test passed` |

## Matrix (Revision 2, re-scored)

| # | Required row | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Public page `site` and `bare` under the **existing** protected built-in theme | **evidenced** | Row wording per decision **R5**: the repository contains exactly one protected built-in theme (`extensions/builtin/themes/sforum-default`), so "both built-in themes" is not a satisfiable requirement and is not this program's scope. `site` and `bare` are verified against the live artifact by three independent probes: SSR chrome counts (`node tests/registry-page-shell-runtime-evidence.mjs` → one navbar owner + one footer on `site`/default, zero on `bare`, `data-registry-shell-source=none` on `bare`), `/pages/resolve-path` identity (`shellProvider=sforum.default-theme` with `shellPackageDigest=85ddda6a…`), and `/site/active-theme/skin` reporting the identical digest. Browser confirmation at a real 390x844 (below, row 8) and with JS disabled. |
| 2 | Plugin body data plus theme override without business-schema drift | **evidenced** (both halves; override half is in-process real HTTP) | **Data half, live instance:** `/pages/resolve-path?path=/loader-articles/hello` → `fallback=false`, `loaderError=null`, `loaderData={"state":"published","title":"Loaded from the plugin runtime: hello"}` from `selectedRuntimeInstanceId=b0a6fc2b…` on the plugin's own artifact `sforum.page-loader-reference-live@1.0.1 / 53af8e1c…`, and the same page's SSR/browser DOM shows the rendered text (`marker="plugin-loader: Loaded from the plugin runtime: hello"` at 390x844). Transport: decision **R2** made the Loader Gateway call the plugin over Protocol V2 (`InvokeRouteInstance`) for a declared `core.guard.public` GET data route. Pinned by `TestReferencePageLoaderFixtureServesTypedDataThroughProductionChain` (real V2 subprocess, full production chain) and `TestPageLoaderGatewayRefusesNonPublicDataRoutes` (undeclared / non-public / digests-drifted / unwired all fail closed 503 with no data) — both re-run green. **Override half:** `TestResolvePathUsesActiveThemeOverrideForPluginBusinessPage` (real HTTP through the production controller and a real theme runtime registry) proves an active-theme override wins with `source=active_theme_override`, that the plugin's `loaderData` (`title=Business truth`, `state=published`) is preserved unchanged, that the plugin template is bypassed, and that a **mismatched `themeOverrideKey` soft-skips back to the plugin template with the business semantics unchanged** — i.e. drift degrades presentation, never the plugin's versioned data contract. Fixture-level counterpart: `TestDurableFixturesProveThemeOverridePreservesPluginBusinessContract` (`extensions/fixtures/themes/sforum-plugin-override-e2e-theme`). **Residual, stated not glossed:** the override half was not exercised as a live dev-instance theme activation; its evidence is the in-process real-HTTP path. |
| 3 | Middleware `before`/`after`/`filter`/`wrap`/`global` ordering on safe and unsafe methods | **evidenced** | Phase order is one plan-order list (`execution_plan.go`): `global → before → filter → wrap → handler → after`, exactly one handler. Safe half `TestExecutionPlanBuildsDeterministicCompletePhaseChain`; unsafe half `TestExecutionPlanOrdersModifierPhasesForSafeAndUnsafeMethods` (POST plan marked unsafe, identical order, `global_middleware` exempt from method matching so it appears in both plans, method-specific modifiers only in their own plan); dispatcher-level order `TestDispatcherExecutesBufferedChainInPlanOrder`; priority permutations covered alongside. All three re-run PASS in Revision 2. |
| 4 | Custom guard allow/deny, CSRF order, timeout, crash, quarantine, revoke, disable, upgrade, rollback, uninstall, Safe Mode, restart | **evidenced** | Row wording per decision **R4**: the `challenge` clause was removed from the row instead of changing the guard contract (guard outcomes are allow or `denied`/`crash`/`timeout`/`protocol`/`canceled`/`unavailable`; there is no `challenge`). Clauses: **CSRF order** — `TestCSRFMintsTokenOnSafeRequest`, `TestCSRFRejectsUnsafeRequestWithoutToken`, `TestCSRFAcceptsUnsafeRequestWithValidToken`, `TestP6RoutePermissionCSRFLocaleQueryAndBodyMatrix`, `TestNewAppArbitraryAdminRoutePreservesCSRFAndPermissionAuthority` (5 PASS in Revision 2). **allow/deny/timeout/crash/Safe Mode/rollback/upgrade/quarantine** — the guard runtime + lifecycle suites in `./app/Support/Extensions/...` (incl. the M5 guard-binding publication tests, the third-theme-outside-the-fence control, and the durable incident ledger), plus `app/Http/plugin_guard_runtime_test.go` for the typed failure kinds. **uninstall (8) and restart (5)** — re-run green in Revision 2, the restart/uninstall set against a **disposable PostgreSQL cluster** via `./scripts/test-api-isolated.sh ./app/Models/Extensions/ …` (13 PASS, including the real-PostgreSQL `TestEnsureInitialPluginRuntimePublicationExactlyOnceAcrossRestart`). **Pointer:** `core.guard.raw_request` is no longer accepted as a route guard (decision **R1**, tightened as a vulnerability); raw authority requires a package-declared `kind: raw_request` guard. |
| 5 | Plugin session state across anonymous/authenticated, rotation, logout, revoke, expiration, privacy erase, disable, uninstall, runtime restart | **evidenced (Revision 4; anonymous clause closed as decision D1 — unavailable by construction, not satisfied)** | Superseded by Revision 4 (top of this file): the clause-by-clause evidence table lives there, including the compiled-fixture-binary joined test, the real-PostgreSQL Identity erasure matrix, the live `privacy:erase-user` run that erased a planted plugin-session key through the real Redis, and the Gateway-as-issuer production wiring fix (correction 1). Revision 3's status text below is kept for the audit trail of how the row was unblocked. |
| 6 | Dev loop success, compile failure, schema failure, crash, rapid edit, API restart, production rejection | **evidenced** | `tests/dev-loop-failure-scenarios.sh` (real `bun`, real digest/validate/test, real file events, real restart timing) asserts: startup reconcile builds; a package failure (declared `packageFiles` entry removed) reports `failed stage=build` and keeps `lastHealthyDigest`; recovery rebuilds; a frontend **compile failure** (`bun run build` → non-zero) keeps the last healthy digest; an **API restart** mid-loop (documented comment probe, `catalog.go` byte-identical afterwards) does not kill the loop; rename (delete+create) coalesces into one successful transaction; `SIGINT` exits cleanly. Unit-level orchestration contract, rapid-edit coalescing, deleted inputs, process-group cleanup and production/staging/test refusal live in `cmd/sforum/dev_extension_test.go`; production config additionally refuses a development scaffold root (`TestProductionRejectsDevelopmentScaffoldExternalRoots`). **Crash is bounded, not faked:** a *build* child crash is covered; a *plugin runtime* crash is not the loop's concern (it watches sources, it does not supervise the runtime) and on the API side is covered by the typed `crash` failure kind failing closed. |
| 7 | Multi-node desired revision, exact artifact, guard provider, shell provider, runtime acknowledgement consistency | **partially evidenced (Go-level); needs an external environment** | What exists: the runtime-set barrier/reconcile implementation with `TestManagerRuntimeSetBarrierSerializesFullSetWithLegacyStartStop`, `…SerializesFullSetWithStagedPublishAndStop`, `…ReconcileAndCloseDoNotReenter`, `TestPluginRuntimeFullSetBarrierWaitHonorsHeartbeatCancellation` (4 PASS in Revision 2); exact-artifact and provider identity verified live (`/pages/resolve-path` and `/site/active-theme/skin` agreeing on the shell provider/digest, plus the plugin's own artifact identity). What does **not** exist: any real multi-node deployment — no second node, no cross-node desired-revision propagation, no per-node runtime acknowledgement was exercised, and a single-process test must not be presented as multi-node evidence. Owner: operator-provided environment. |
| 8 | Browser desktop/mobile, SSR, JS-disabled, selected provider and digest, single chrome, no overflow, no console errors | **evidenced** | **SSR**: the persistent check `node tests/registry-page-shell-runtime-evidence.mjs` (one navbar owner + one footer on `site`/default, zero on `bare`, shell identity agreeing with the skin endpoint). **Desktop**: 1313x930 passes recorded earlier (Route Inspector/M5). **390x844 (real Chrome 154 over CDP, `Emulation.setDeviceMetricsOverride(390x844, dpr 3)`)** — Revision 2 re-ran it across five public paths including the `view: component` page: every path `overflow=false clipped=0 consoleErrors=0`, exactly one header and one footer on shell pages and zero on `bare`, and the component page reports `l2=mounted/kids=1 trustNotice=true marker="page component reference body"`. **JS disabled**: `SFORUM_DISABLE_JS=1 node tests/browser-registry-page-shell-390x844.mjs` still measures one navbar + one footer on `site`/`default`, zero on `bare`, correct shell markers and SSR body text — chrome and body come from the server render. **Selected provider/digest**: `/pages/resolve-path` + `/site/active-theme/skin` agree on `sforum.default-theme` / `85ddda6a…`; DOM markers are stated per page kind (a plugin page has no `data-provider`/`data-template="1"` because its body is the plugin's template — that is not claimed). **Admin**: `tests/browser-admin-390x844.mjs` at 390x844 dpr3 with an injected session cookie. **Revision 2 note:** the harness needed a fix to be able to measure the component mount at all — see "Harness defect found and fixed" above; the assertion has a negative control. |
| 9 | Candidate release image smoke | **evidenced** (Revision 1 misreported this as "not run") | Built both API targets and the web target (`--target api\|migrate\|prod`): `sforum-api:candidate-smoke` 620 MB, `sforum-migrate:candidate-smoke` 29.9 MB, `sforum-web:candidate-smoke` 261 MB, all exit 0. Migrated an **isolated** database (`sforum_smoke`, dropped afterwards, dev DB untouched): 145 tables + 7 River migrations. API image with `APP_ENV=production` served `GET /api/v1/health` → **200** `{"status":"ok","environment":"production"}`. Three production fail-closed gates demonstrated in order (placeholder secrets panic; missing `EXTENSION_ROOT` refused — the misleading `mkdir /storage: permission denied` was replaced by a named config error; missing `MARKETPLACE_ED25519_PUBLIC_KEY_HEX` refused). Web image served live SSR (`GET /` 200, control experiment: with the API container stopped the same request returns **000**). Then seeded (50 users / 6 categories / 8 topics) and real content served (`/`, `/c/general`, `/t/1/topic-ngkxas` all 200). Then the extension surface: the image synced its protected built-ins (8 plugins + 1 theme), ran 5 plugin backends from immutable digest paths, the enabled search plugin indexed the seeded topics; an **uploaded** package was installed (201) and enabled (200) through the release images and served its public pages with the layout contract intact; the **executable** trust gate was exercised on both sides (challenge 200, enable without token 409, with token 200, replay after `DELETE …/trust` 409); `compose.release.yaml` was exercised end to end (postgres/redis healthy, api ran its own migrations, health 200, web `/` 200). Residual: the registry pull of a real published tag, and `compose.prod.yaml`'s reverse-proxy/TLS layer. |
| 10 | Compatibility farm and APILTS for every changed public contract | **evidenced** | Farm: `./scripts/test-api-isolated.sh --compat` → `exit=0`, `[pass] current-host-v2-manifest-v3 status=required`, `[pass] manifest-v3-contract status=required`, `ok: true`. APILTS: `sforum.host.v2` / `sforum.manifest.v3` / `sforum.protocol.v2` all `current` with `shim=false`; `sforum.theme.l1.request-time-loader` `deprecated shim=true` with its replacement `sforum.theme.runtime-snapshot`. |

## Exit gate

**Met for every row except row 7 (as of Revision 4, 2026-10-17)**:

- **Row 5** — closed. Every clause has per-row evidence (see Revision 4); the
  anonymous clause is closed by operator decision D1 as "unavailable by
  construction, fail closed", not by behaviour, and reopening it means a new
  session-persistence ADR.
- **Row 9** — closed. Both Revision 3 residuals (real registry tag,
  reverse-proxy/TLS layer) were exercised on 2026-10-17; see Revision 4.
- **Row 7** — open, operator-owned. A second node is required; the Go-level
  barrier/reconcile tests are real but are not multi-node evidence. Nothing in
  this program remains to build for it.

No open P0/P1 finding remains inside this program. The plan may move to
`archive/` when the operator accepts row 7 as an environment dependency (the
only reason it has stayed out so far).

## Conflict with the task prompt's baseline (code wins)

The prompt states the active theme artifact digest is `4691a60541836c57`. At
runtime the active theme is `sforum.default-theme@1.0.0` with
`packageDigest=85ddda6ad7825adaacfa5f741715b64a09737b41320e1a49848ca3e695edccac`,
reported identically by `/site/active-theme/skin` and `/pages/resolve-path`. The
short digest in the prompt is stale; the runtime value is authoritative.

Two further task-book/repository conflicts were recorded in Revision 1 and are now
**settled by operator decisions rather than left standing**: "both protected built-in
themes" (the repository has exactly one — decision **R5** rewords the row to the
existing theme) and the guard `challenge` outcome (it does not exist in the contract —
decision **R4** removes the clause from row 4). Neither is a defect in the repository.

## Route A layer 6 (added 2026-10-16, same day)

- B2 (parse-time fail-open) was already covered by
  `TestResolvePathFailsOpenToHostShellWhenDeclaredShellDisappears`; no duplicate test
  was written.
- B1 is now `tests/registry-page-shell-runtime-evidence.mjs`: a persistent runtime
  check over real SSR HTML and the resolve API. First run failed for two instructive
  reasons (recorded): `data-registry-shell` carries the **layout**, not the shell
  source, and substring counting of `sf-public-mobile-bottom-nav` also matched its
  `__link` children (6 hits). It now classifies `<nav>` elements by exact class token
  and asserts: `site` → `theme` shell with exactly one desktop and one mobile nav plus
  one footer; `bare` → `none` with zero chrome; shell provider/digest agreement with
  `/site/active-theme/skin`; `SKIP` (exit 0) when the servers are down, failure when
  `SFORUM_EVIDENCE_REQUIRE_LIVE=1`.
- B3 is `knowledge/runbooks/2026-10-16-theme-switch-shell-verification.md`.
- A component-level render test for `SFRegistryPageShell.vue` was attempted and
  **abandoned**: `bun test` has no SFC transform (every existing component test reads
  source strings), so rendering-layer proof is runtime evidence in this repository.

## Release image smoke: defect found and fixed

The smoke exposed a misleading production failure mode. With `APP_ENV=production`
and no `EXTENSION_ROOT`, the non-production default (`../../storage/extensions`)
resolves against the container's working directory to `/storage/...`, so the
candidate image died with:

```
api bootstrap: Host platform services setup failed: create plugin files service:
plugin files input is invalid: mkdir /storage: permission denied
```

Nothing in that message names the missing variable. `config.Load` now refuses a
missing or relative `EXTENSION_ROOT` in production with
`config: EXTENSION_ROOT must be an absolute path in production (got "...")`
(`apps/api/config/config.go`), and
`TestLoadRejectsRelativeExtensionRootInProduction` pins it; the shared production
test fixture provides the value compose and the image already use
(`/var/lib/sforum/extensions`). Compose and the Dockerfile were already correct —
this only makes the misconfiguration self-explaining.

## Full three-image smoke (same day, follow-up)

The residual above is closed. All three candidate targets were built
(`sforum-api:candidate-smoke` 620MB, `sforum-migrate:candidate-smoke` 29.9MB,
`sforum-web:candidate-smoke` 261MB, all exit 0) and run together against an isolated
database:

```text
migrate image  → sforum_smoke: 145 tables + 7 River migrations
api image      → APP_ENV=production, GET /api/v1/health → 200 {"status":"ok","environment":"production"}
web image      → GET /:8098/ → 200, 81,120 bytes SSR HTML, title "SForum",
                 nav elements = 2 (one desktop navbar + one mobile navbar: the single
                 navbar owner's responsive pair), sf-footer present (2 in HTML, one
                 suppressed by the content-column rule, so one visible),
                 API-derived forum markup present (sf-home / sf-topic / sf-category)
```

**Control experiment (decisive):** with the API container stopped, `GET /` through the
web container returned **000** (no response); with the API running it returns 200.
So the web image's SSR genuinely depends on the API — the render is live SSR, not a
static shell, and the three images compose end to end.

Cleanup: both containers removed, `sforum_smoke` dropped (verified `count=0`), and the
three candidate images removed. They are reproducible with:

```bash
docker build --network=host --build-arg HTTP_PROXY=http://127.0.0.1:7897 --build-arg HTTPS_PROXY=http://127.0.0.1:7897 \
  --target api|migrate -f apps/api/Dockerfile -t sforum-api:candidate .
docker build --network=host --build-arg HTTP_PROXY=http://127.0.0.1:7897 --build-arg HTTPS_PROXY=http://127.0.0.1:7897 \
  --target prod -f apps/web/Dockerfile -t sforum-web:candidate .
```

## Seeded release-image smoke (same day, follow-up)

The empty-database limitation is closed. The three candidate images were rebuilt and
run against a **seeded** isolated database:

```text
host migrate CLI → sforum_smoke schema (7 River migrations)
host seed CLI    → 50 users (user#1 is the super_admin), 2 groups, 6 categories,
                   12 tags, 8 topics, 17 comments, 3 pinned
api image        → APP_ENV=production, GET /api/v1/health → 200
web image        → GET /            → 200, 127,220 bytes: 1 desktop navbar + 1 mobile
                                     navbar, footer present, 8 seeded topic links,
                                     4 seeded category links (announcement, devops,
                                     frontend, general)
                 → GET /c/general   → 200, 93,302 bytes, same chrome, seeded links
                 → GET /t/1/topic-ngkxas → 200, 132,703 bytes, seeded topic content
```

So the images serve real seeded forum data end to end (web SSR → API → Postgres),
not just an empty boot. Containers, database and images were removed afterwards.

## Release-image extension inventory and runtime (same day, follow-up)

The extension half of row 9 is now evidenced too, without needing an admin session:
the image's own boot does the work. After the api image booted against a seeded
isolated database:

```text
host CLI: sforum extension list (against the image-populated database)
ID                    TYPE    STATUS     SOURCE   VERSION  DIGEST
sforum-shortcodes     plugin  enabled    builtin  1.3.4    0ae602e8ae82
sforum.auth-github    plugin  installed  builtin  1.0.7    16b26728a0be
sforum.content-policy plugin  enabled    builtin  1.1.6    30b2ccd1b980
sforum.search-site    plugin  enabled    builtin  1.0.6    25077e2314ac
sforum.smtp           plugin  enabled    builtin  1.1.6    75b594511f8c
sforum.storage-fs     plugin  installed  builtin  1.1.7    4d776a2b0e68
sforum.storage-s3     plugin  installed  builtin  1.0.8    ccb1add788d6
sforum.web-push       plugin  enabled    builtin  1.0.6    c3460c16f712
sforum.default-theme  theme   enabled    builtin  1.0.0    85ddda6ad782
```

and the container was running the plugin code as subprocesses from the immutable
digest paths:

```text
PID   COMMAND
  24  /var/lib/sforum/extensions/sforum-shortcodes/1.3.4/0ae602e8…/backend/plugin
  38  /var/lib/sforum/extensions/sforum.content-policy/1.1.6/30b2ccd1…/backend/plugin
  52  /var/lib/sforum/extensions/sforum.search-site/1.0.6/25077e23…/backend/plugin
  66  /var/lib/sforum/extensions/sforum.smtp/1.1.6/75b59451…/backend/plugin
  77  /var/lib/sforum/extensions/sforum.web-push/1.0.6/c3460c16…/backend/plugin
```

with the enabled search plugin doing real work on the seeded data
(`search: indexed topic topicId=1..4`, `search reconciliation enqueued repairs
providerId=sforum.search-site indexJobs=4`). So the release image ships its protected
built-ins, syncs them into the database, enables the safe defaults, starts the plugin
runtimes, and serves seeded content through the web image.

## Install + enable through the release images (same day, follow-up)

Row 9's install/enable half is now **verified end to end** against the release images,
using the scripted admin credential added in the previous pass:

```text
users:reset-password --email seed_…@seed.local --password '…' --yes
  → 密码已重置：user_id=1 username=seed_… revoked_sessions=0
POST /api/v1/auth/login (csrf_ cookie + X-CSRF-Token + Origin) → 200, super_admin session
POST /api/v1/admin/extensions (multipart file=smoke-pkg.sforum.zip) → 201
  → {"id":"sforum.page-layout-reference","status":"installed","source":"uploaded"}
POST /api/v1/admin/extensions/sforum.page-layout-reference/enable
  {"confirmCapabilities":true} → 200 {"status":"enabled","source":"uploaded"}
host CLI: extension list → sforum.page-layout-reference plugin enabled uploaded 0.1.0 ce8ed23d9773
```

and the freshly installed, uploaded package's public pages then render through the
**web** image with the layout contract intact and the shell owned by the active theme:

```text
/page-layout-reference/bare    bytes=34046 desktopNav=0 mobileNav=0 footer=0 shellSource=none
/page-layout-reference/site    bytes=66752 desktopNav=1 mobileNav=1 footer=1 shellSource=theme
/page-layout-reference/default bytes=66778 desktopNav=1 mobileNav=1 footer=1 shellSource=theme
```

Two findings worth keeping:

1. **Correction to an earlier claim.** This matrix previously said the enable path
   needs "an authenticated `super_admin` session plus the exact-artifact trust
   challenge". That is only true for packages that actually ship an executable
   artifact. Both fixtures installed here declare `backend: {entry: "", rpc: ""}` and
   ship no backend file, so the correct answer is `409 extension.trust_not_required`
   and `enable` legitimately succeeds without a confirmation token.
2. **The trust gate is on in the smoke.** `SFORUM_V3_TRUST_CHALLENGES` defaults to
   `isProd` (`config.go:225`, `isProd := APP_ENV == "production"`), and the smoke runs
   `APP_ENV=production`, so `trust_not_required` here is a statement about the
   *package*, not a disabled gate.

## Executable trust gate through the release images (same day, follow-up)

The executable half is verified too. A package that actually ships an executable was
built for the smoke by copying the `sforum-shortcodes` backend under a new id
(`acme.trustprobe`), rewriting every id-bearing field, repointing the module `replace`
at the repository, and compiling it for linux:

```text
cd /tmp/trustprobe/backend && CGO_ENABLED=0 GOOS=linux go build -trimpath -buildvcs=false -ldflags="-s -w" -o plugin .
  → backend/plugin, 16,142,496 bytes
sforum extension digest --write /tmp/trustprobe → OK
sforum extension validate /tmp/trustprobe        → OK (backend: backend/plugin (hashicorp-go-plugin))
sforum extension test /tmp/trustprobe            → ✓ backend.rpc_ok, ✓ capabilities.resolved, ✓ manifest.ok, ✓ settings.renderer
sforum extension package /tmp/trustprobe         → 26 files, 18,628,537 bytes
```

Against the release image (`APP_ENV=production`, so `SFORUM_V3_TRUST_CHALLENGES`
defaults to true), the gate behaved exactly as designed on **both** sides:

```text
POST /admin/extensions (multipart)                → 201 installed, source=uploaded,
   manifest.backend = {entry: backend/plugin, rpc: hashicorp-go-plugin, protocolVersion: 2,
                       digest: da2c9b48c595a293…, hostApiVersion: sforum.host@2}
POST /admin/extensions/acme.trustprobe/trust/challenge → 200, token 64 chars,
   expiresAt = +5min, impact = sforum.trust-impact@2 {action: enable, extensionId, version,
   source, packageDigest}
POST …/enable {"confirmCapabilities": true}        → 409 extension.trust_challenge_required
   ("此扩展包含可执行代码。请由超级管理员审阅完整影响并生成一次性确认。")
POST …/enable + confirmationToken                  → 200 enabled
container process table                             → 6 plugin backends, including
   /var/lib/sforum/extensions/acme.trustprobe/1.3.4/9b1e202816de77cf…/backend/plugin
```

Two further behaviours were measured rather than assumed:

- After `disable`, re-enabling **without** a fresh token returned 200 — because the
  exact-artifact trust grant persists for that package digest. That is the designed
  behaviour, not a replay acceptance.
- `DELETE …/trust` → 200 with `trustRequired=true, trusted=false`; enabling with the
  already-consumed token then returned **409 `extension.trust_challenge_replayed`**
  ("确认已使用，不能重复提交，请重新生成。"), which is the one-time semantics holding.

## Release compose stack (same day, follow-up)

`compose.release.yaml` was exercised too, with the local candidate images standing in
for the registry tag. The override resolves to three published-image services with
`pull_policy: always` and **fails closed** without the tag:

```text
docker compose -f compose.yaml -f compose.release.yaml config   (no SFORUM_VERSION)
  → error while interpolating services.{api,migrate,web}.image:
    required variable SFORUM_VERSION is missing a value: Set SFORUM_VERSION to an immutable release tag
```

Run with `SFORUM_VERSION=smoke-local` and the candidate images tagged as
`ghcr.io/zhuchunshu/sforum-{api,migrate,web}:smoke-local` (a throwaway override in
`/tmp` set `pull_policy: never` and published web/postgres for probing):

```text
postgres/redis        → healthy
api                   → ran its own migrations (MIGRATE_ON_STARTUP=true),
                        GET /api/v1/health → 200 {"environment":"production"}
web                   → GET / → 200, one desktop navbar + one mobile navbar + footer;
                        /c/general → 200; after seeding + an API restart the home page
                        carried 3 seeded topic links
host CLI extension list → image built-ins: 5 enabled, 3 installed, 1 theme
```

Three operational findings, all measured:

1. **The dev `.env` must not be carried into the release stack.** Compose reads the
   repository-root `.env`, and its dev values override the compose defaults. Observed
   leaks: `EXTENSION_ROOT=../../storage/extensions` (relative → the API now refuses to
   boot in production, which is the fail-closed gate added earlier working as intended),
   `NUXT_API_INTERNAL_BASE_URL=http://127.0.0.1:8080/api/v1` (inside a container
   `127.0.0.1` is the container itself, so SSR returned 502/timeouts until it was
   overridden to `http://api:8080/api/v1`), `DATABASE_URL`/`REDIS_ADDR` pointing at host
   ports, and `EXTERNAL_EXTENSION_ROOTS` host paths (harmless
   `external extension source skipped` warnings).
2. **`down` needs the same `-f` set and `SFORUM_VERSION`.** Without them compose fails
   during interpolation and leaves containers and volumes behind.
3. **Seed before the API starts, or restart the API once afterwards.** The home page
   depends on the search index being reconciled: seeding after boot left it in an
   empty/error state (81,801 bytes, no category or topic links) until an API restart
   reconciled the index (106,147 bytes, 3 seeded topic links).

Runbook: `knowledge/runbooks/2026-10-16-release-image-stack-verification.md`.

Residual for row 9: the registry pull of a real published tag itself (this run used
locally built candidate images with `pull_policy: never`), and `compose.prod.yaml`'s
reverse-proxy/TLS layer.

## Row 6 fresh run (Revision 2 pass)

`bash tests/dev-loop-failure-scenarios.sh` was re-run against the live dev stack;
every scenario passed:

```text
dev loop failure scenarios (repo=/home/inkedus/Code/github/SForum)
Created plugin scaffold at extensions/dev/acme.row6scenarios
  ok   startup reconcile built the package
  ok   missing declared package file is reported
  ok   failure keeps the last healthy digest
  loop survived the package failure
  ok   the package recovers after the file returns
  ok   a failing frontend build is reported with its exit status
  loop survived the build failure
  catalog.go restored byte-identical after the restart probe
  ok   the loop keeps rebuilding after an API restart
  loop survived the API restart
  ok   a renamed source input rebuilds the package
  loop exited on SIGINT
  ok   the loop reports a stopped phase
dev loop scenarios: OK
```

The script restarts the dev API as part of scenario 5; afterwards `:8080` and
`:3000` were both confirmed live again (`api:200`, `web:200`) and
`git status` shows `catalog.go` with only the pre-existing session edit — the
probe's temporary comment was removed byte-identically.

## Environment anomaly observed at the end of Revision 1 (historical — resolved)

Revision 1 ended with the operator's Nuxt dev server down and the air-managed API
not listening. **Both servers were up for the whole Revision 2 pass** and every
runtime probe above was taken against them. The section is kept because it
explains why Revision 1 left rows 6/8 partly unevidenced. What Revision 1 saw:

- `http://127.0.0.1:3000/probe-site` → no response (the operator-owned Nuxt dev
  server is stopped; this session did not stop it and did not restart it).
- `http://127.0.0.1:8080/api/v1/health` → no response, while the air-managed process
  is alive (`air` 7h09m, plus an `sforum-api` child 2h57m with the correct env:
  `DATABASE_URL=postgres://sforum:sforum@127.0.0.1:15432/sforum`, `HTTP_PORT=8080`,
  `APP_ENV=development`) and `storage/logs/air.log` is a wall of `exit status 1`.
  Nothing is listening on 8080.
- Running the already-built binary manually with the documented env
  (`set -a; . .env; . storage/builtin-dev/air.env; ./tmp/sforum-api`) works: it
  bound `127.0.0.1:8080`, registered 564 handlers, and shut down cleanly on signal.
  So the code and configuration were healthy; the anomaly was in the air loop/process.

Revision 1's honest consequence at the time: the CDP 390x844 harness had passed at
09:26 while the servers were up, `tests/registry-page-shell-runtime-evidence.mjs`
later reported `SKIP`, and source-level gates were all that could be re-run.

## Defect found and fixed in the evidence harnesses themselves

While the web dev server was down, `tests/browser-registry-page-shell-390x844.mjs`
printed `SKIP ...` **and then** `all 390x844 acceptance assertions passed` with exit
0 — a false positive, because `finish()` claimed success whenever the failure list
was empty, including when nothing had been measured. Fixed in all three harnesses
(`browser-registry-page-shell-390x844.mjs`, `browser-admin-390x844.mjs`,
`registry-page-shell-runtime-evidence.mjs`): an `asserted` flag is only set once
measurement actually starts, and the success line is replaced by
`skipped: no assertion ran` otherwise. Verified now, with the web server down:

```text
SKIP the 390x844 pass needs the running web dev server (web=http://127.0.0.1:3000)
registry page shell 390x844 browser evidence (web=http://127.0.0.1:3000)
  skipped: no assertion ran          # exit 0
SFORUM_EVIDENCE_REQUIRE_LIVE=1 ...   # exit 1
```

A harness that claims success while skipping is worse than no harness, so this is
recorded as a defect rather than a cosmetic change.

## Web dev server restarted by this session (2026-10-16, reported)

The operator-owned Nuxt dev server on `:3000` had been down since roughly 12:19 (it
was already down when the previous pass ended, and two passes asked the operator to
restart it). Because the remaining acceptance items in row 8 and A3/A4 need it, this
session started it with the repository's documented command
(`cd apps/web && bun run dev`, log at `/tmp/web-dev-relaunch.log`) and re-ran the
evidence with it up:

- `node tests/registry-page-shell-runtime-evidence.mjs` → all assertions passed
  (site/default `shell=theme`, one navbar owner, one footer; bare `none`/0/0; active
  theme `sforum.default-theme@1.0.0` digest `85ddda6a…` agreeing across endpoints).
- `node tests/browser-registry-page-shell-390x844.mjs` → 390x844 dpr3 pass, and with
  `SFORUM_DISABLE_JS=1` the JS-disabled pass.

The operator can stop it at any time; it is not managed by this session's tooling.

## M8 residuals closed (2026-10-16)

The two items M8 listed as not done are now settled, one by delivery and one by an
explicit boundary:

1. **Admin-activation automation within the ADR.** `tests/admin-extension-activation.sh`
   drives the approved HTTP flow — install (inert) → exact-artifact trust challenge →
   enable with the confirmation token — with a `super_admin` session supplied either as
   a cookie or as email/password. It fails an executable package that cannot obtain a
   token, accepts the server's `extension.trust_not_required` verdict for packages that
   ship no executable artifact (that is the correct verdict, not a bypass), and can
   `disable` + `uninstall` afterwards. Verified green against the dev API with a
   throwaway package: `201` inert → `trust_not_required` → enable `200` → disable `200`
   → uninstall `200`, with the ledger and `storage/extensions/` clean afterwards.
2. **Runtime crash, bounded rather than faked.** `tests/dev-loop-failure-scenarios.sh`
   now also covers a **rename** (`delete`+`create` coalesced into one successful
   transaction), alongside the earlier build-crash, package-failure, API-restart and
   SIGINT cases. A **plugin runtime** crash is not the dev loop's concern — the loop
   watches sources and does not supervise the runtime — and on the API side it is
   already covered: the typed `crash` failure kind fails closed to
   `ErrCoreGuardEvaluatorUnavailable` without leaking the error text
   (`app/Http/plugin_guard_runtime_test.go`, `core_guard_failure_mapping_test.go`).
