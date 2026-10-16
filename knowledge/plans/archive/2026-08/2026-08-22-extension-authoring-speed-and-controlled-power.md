Status: **active** - M0-M9 complete; **M10 closed except row 7** (2026-10-17, Revision 4: row 5 closed with per-clause evidence — the compiled-fixture joined test, the real-PostgreSQL erasure matrix, and the live `privacy:erase-user` erase; the anonymous clause closed by decision D1 as unavailable-by-construction; row 9's two residuals closed on the real `v3.0.9` registry tag and the production Caddy/TLS layer; a production wiring gap in the S9 mint — the Gateway did not implement the issuer — was found and fixed). Row 7 (true multi-node) is operator-owned environment work and is the only open item.

Status: **active** - M0/M1/M2/M3/M4/M5/M7/M8 complete; **M9 complete** (the session-state workflow reference package landed 2026-10-17 as `sforum-session-checkout-reference`); **M10 closed except row 7** — see the new status line above and the Revision 4 closeout block. **M6 is unblocked by Revision 2 of the frozen contract record and largely built: the store (S1-S3/S6), the six erasure hooks (now actually injected at assembly), the S9 Host-signed single-invocation session delegation, the registered `PluginSessionService` (Safe Mode + exact-runtime admission), the plugin SDK helper `Host.SessionState(ctx)` and the bilingual/API docs are implemented, wired and green (see `sessions/2026-10-16-m6-s9-session-delegation-handoff.md`). A joined end-to-end test also landed (dispatch → mint → real-gRPC plugin echo → real service → real store → retire → replay denied). What is still missing: M9's reference fixture and anonymous session state (unavailable because Core mounts no global session middleware — ADR Revision 3 §3.4).** See the closure-pass block below the milestone table

Date: 2026-08-22
Last updated: 2026-10-17 - M10 Revision 4: row 5 closed (per-clause evidence), row 9 residuals closed (real tag + TLS layer), row 7 recorded as operator-owned; S9 production mint gap fixed; M9 complete with the session-checkout reference package

Goal: restore a fast, familiar extension-authoring loop for public pages,
admin pages, route middleware, authorization, and request-scoped state without
exposing raw Fiber context, raw Core sessions, Host-private Nuxt APIs, or
pre-plugin recovery surfaces to installable extensions.

## Execution Rule

Execute one milestone at a time. Do not combine this program into one patch or
one agent conversation. Every milestone must leave the repository buildable,
update its durable evidence, and satisfy its exit gate before the next starts.

The assignment labels in this task book are deliberate:

- **Codex** means start a new conversation with Codex. The task changes a
  security boundary, cross-runtime contract, active-theme ownership, lifecycle
  authority, or joined production behavior.
- **OpenCode + DeepSeek V4 Flash** means the task is sufficiently mechanical
  only after its prerequisite contract is frozen and merged. The delegated
  agent must stay inside the named files and acceptance criteria.
- **Codex review required** means delegation is acceptable for implementation,
  but the milestone cannot close until a new Codex conversation reviews the
  resulting diff and runs the stated gates.

Do not use a delegated agent to make architecture choices left open by an
earlier milestone. When its prompt encounters an unresolved choice, it must
stop and report the question instead of inventing a contract.

## Required Reading

Before every milestone, read:

1. `AGENTS.md`
2. `knowledge/index.md`
3. `knowledge/modules/extensions.md`
4. `knowledge/modules/frontend.md`
5. `knowledge/modules/identity.md` for M4-M7
6. `knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`
7. `knowledge/plans/2026-07-13-trusted-plugin-theme-platform-v3-progress.md`
8. `knowledge/plans/2026-07-22-v3-production-rewire-honesty-remediation.md`
9. `docs/extensions/authoring-guide.md`
10. `docs/extensions/routes.md`
11. `docs/extensions/page-catalog.md`
12. `docs/extensions/trusted-admin-components.md`
13. the current hot handoff for this workstream, once one exists
14. this task book

For public-shell work, also read the active built-in theme manifests and every
public chrome ownership test. For session work, read the full AuthSession and
external-auth decisions rather than relying on the summary in this plan.

## Problem Statement

SForum V3 has stronger extension primitives than SForum-old, but a simple
extension now pays too much authoring overhead:

- a public page must understand `theme.json`, exact templates, package files,
  Page Registry, Loader Gateway, Host islands, active-theme ownership, and
  digests before it can render inside a coherent site shell;
- a backend change often requires separate build, digest, validation, API
  restart, trust, enable, and runtime inspection steps;
- route `before`, `after`, `filter`, `wrap`, and `global_middleware` exist, but
  there is no beginner-facing middleware scaffold or small handler API;
- custom guard execution exists in lower-level contracts and tests, while the
  current documented production Host keeps custom/raw guards closed;
- plugins cannot hold small request-session workflow state without either
  inventing persistence or requesting authority they should not receive;
- admin page inheritance works, but the public page path does not provide an
  equally obvious default shell contract.

The answer is not to return to in-process arbitrary code. The platform must
make safe paths short and common while preserving explicit trust for genuinely
high-risk powers.

## Confirmed Product Decisions

- Installable plugins never receive `*fiber.Ctx` or an equivalent Host-private
  object.
- Installable plugins never read or mutate the raw Core session map, cookie,
  session identifier, CSRF state, actor id, role, or permission snapshot.
- Installable plugins cannot register pre-plugin boot middleware, replace Safe
  Mode, intercept health/recovery CLI, or load arbitrary Nuxt Layers.
- Custom route middleware remains a versioned Route Registry capability using
  `global_middleware`, `before`, `after`, `filter`, and `wrap`.
- Custom authorization guards are a supported high-risk extension capability,
  but must be exact-artifact, route-scoped, inspectable, separately confirmed,
  timeout-bounded, and fail-closed.
- Authentication and authorization remain separate contracts. A custom guard
  may allow, deny, or challenge an already resolved Host actor. It does not mint
  a user, role, permission, browser session, or actor assertion in V1.
- External login and credential-to-identity flows continue through the
  Identity Registry and Host-owned session issuance.
- Plugin session state, if implemented, is a Host-owned namespaced state store.
  It is not an escape hatch into Core authentication state.
- Plugin-added public pages default to a stable site-shell contract. Authors
  explicitly opt into `bare`; they do not copy navbar/footer markup into every
  page.
- The selected theme may present a plugin-owned business page, but it cannot
  alter the plugin's versioned data schema, access policy, or business route.
- Admin pages continue to inherit the Host admin shell and use only public
  Admin/Plugin UI SDK contracts.
- Development automation may reduce commands but may not weaken production
  trust. Any local development grant is visibly local, non-exportable,
  environment-gated, and rejected by production configuration.
- A future compile-time kernel module tier, if ever approved, is a separate
  product from installable plugins and is outside this task book.

## Non-Goals

- Reintroducing SForum-old's in-process PHP execution model.
- Exposing Fiber, pgx pools, Redis clients, Nuxt internals, raw cookies, raw
  Authorization headers, or Core session structures as an extension ABI.
- Letting a plugin change the current actor, grant itself permissions, issue a
  browser session, or bypass CSRF.
- Adding a generic reverse proxy or arbitrary native shared-library loader.
- Making public L2 code a sandbox; it remains fully trusted browser code after
  exact approval.
- Closing unrelated Marketplace, Privacy, compatibility-farm, commerce, or
  rollout residuals by assertion.
- Adding framework-specific conveniences before the stable wire and ownership
  contracts are frozen.

## Current Baseline And Gaps

| Area | Current baseline | Gap this plan closes |
| --- | --- | --- |
| Public pages | Plugins may add paths through `theme.json`, exact templates, data routes, and schemas | No automatic active-site shell; authors must understand and duplicate chrome ownership |
| Public chrome | Core pages use `SFPageOutlet`; selected L1 templates own navbar/footer and Core fallback uses `SFHostPublicChrome` | Catch-all plugin add pages call `SFThemeTemplate` directly and do not share the complete ownership contract |
| Theme compiler | Package-local Go templates support `define`, `template`, and `block` in tested paths | No frozen cross-package selected-theme shell ABI for plugin page bodies |
| Admin pages | `admin.pages[].view=component` inherits the Host admin layout and stable bridge | Scaffold coverage and examples can be made faster; no architecture replacement needed |
| Route middleware | V3 declares every route action; Protocol V2 request/response patch stages exist | No focused author scaffold, debugger, or single-purpose reference package |
| Custom guards | Exact guard request/response, manager transport, trust vocabulary, and integration tests exist | Public author docs say the current Host route is closed; production binding and joined evidence are absent |
| Request authority | Filtered headers and Host actor context are default; raw request is a separate high-risk grant | No narrow declared-header convenience contract for common authz use cases |
| Sessions | AuthSession is Host-owned and hardened; Identity providers return assertions | No plugin-namespaced request-session state contract; raw access must remain closed |
| CLI | Build, digest, validate, test, package, scaffold, and restart commands exist | Edit-build-digest-restart-trust-enable remains a long manual loop |
| Recovery | Safe Mode, CLI disable/quarantine, immutable revisions, and exact trust exist | New convenience paths must prove they do not weaken these controls |

## Target Author Experience

### Public page

The minimum target is one scaffold command and one declaration:

```bash
sforum make:plugin --id acme.orders --backend --public-page orders
sforum extension dev ../../extensions/dev/plugins/acme.orders
```

```json
{
  "id": "acme.orders.page.index",
  "path": "/orders",
  "layout": "site",
  "access": "login",
  "template": "templates/orders.html",
  "dataSource": "plugin",
  "dataRoute": "/page-data/orders",
  "dataSchema": "schemas/orders.json"
}
```

`layout: site` must render exactly one selected-theme/Host public chrome around
the plugin body. `layout: bare` must render no implicit chrome.

### Route middleware

The minimum target is a generated Protocol V2 handler with typed stages:

```bash
sforum make:plugin --id acme.policy --backend --route-middleware
```

The generated example must cover one `before` request patch and one `after`
response patch. It must not use raw request authority.

### Authorization guard

The minimum target is an explicit route-scoped declaration:

```json
{
  "id": "acme.policy.guard.subscription",
  "contractVersion": "acme.policy.guard.subscription@1",
  "kind": "custom",
  "entry": "backend/plugin",
  "permissions": ["orders.view"]
}
```

The final schema may differ after M0, but V1 must return only structured
allow/deny/challenge output for the Host-resolved actor.

### Plugin session state

The target SDK is intentionally narrow:

```text
GetPluginSessionState(namespaceKey)
SetPluginSessionState(namespaceKey, value, ttl)
DeletePluginSessionState(namespaceKey)
```

The Host derives extension id and session identity from the runtime-scoped
request context. Callers never submit either authority as trusted input.

## Architecture Invariants

### Public shell ownership

- Exactly one component owns global navbar and footer for every successful
  public render path.
- `layout: site` resolves a shell from the selected exact theme artifact, with
  a Host fallback that preserves the shared geometry contract.
- The plugin owns only its body template/data contract; the selected theme owns
  shell presentation and may provide an exact `themeOverrideKey` body override.
- The shell must expose one reviewed body outlet, not arbitrary executable
  slots.
- SSR, locale, SEO, CSP, page regions, and L2 reference aggregation occur once.
- A missing, disabled, stale, or invalid theme/plugin artifact fails to the
  correct prior/Core presentation without duplicate chrome.
- Auth and system pages keep their existing intentional shell rules.

### Middleware and guard ownership

- Route middleware composes only through the versioned Route Registry.
- `global_middleware` means Registry-managed routes, never pre-plugin health or
  out-of-band recovery.
- Request and response mutation remains limited by frozen RFC 6901 allowlists.
- Host guards run by default. A custom authorization guard requires an exact
  route binding and separate high-risk grant.
- Custom guards cannot create actors or sessions in V1.
- Credential authentication belongs to a separately versioned Identity
  provider contract.
- Raw request authority is not part of the first deliverable. M4 must record a
  separate decision before any raw credential field is admitted.

### Session state ownership

- Core authentication claims and plugin state use separate keyspaces and APIs.
- The Host derives the current session and extension identity; the plugin never
  supplies a trusted session id or extension id.
- Plugin state has bounded key count, key length, value bytes, and TTL.
- Values are opaque bytes or a versioned typed document; the ADR must choose one
  and define migration behavior before implementation.
- Anonymous state, authenticated state, session rotation, logout, disable,
  uninstall, privacy erase, and Safe Mode each have an explicit policy.
- State writes cannot extend the Core session lifetime.
- Session replacement cannot silently transfer stale plugin state unless the
  ADR explicitly defines a safe migration rule.

### Development mode ownership

- Development automation works only for configured dev roots and a development
  environment marker.
- Production refuses development grants and dev-only command endpoints.
- Every rebuilt artifact receives a new digest and runtime identity.
- Reload uses the normal disable/drain/stage/enable state machine or a formally
  equivalent dev-only transaction; it never mutates bytes beneath a live
  digest.
- Errors remain actionable: build, digest, schema, trust, start, readiness, and
  route publication failures must be distinguishable.

## Milestone Summary

| Milestone | Outcome | Assignment | Status |
| --- | --- | --- | --- |
| M0 | Freeze contracts, inventory real call chains, write ADR and threat model | **Codex** | **complete** - all four areas traced from production code; ADR frozen. Five residual questions (report section 5, incl. R3 intent) handed to M4 / a human decision; none of them block M1 |
| M1 | Implement selected-theme public site-shell contract and fallback | **Codex** | **complete for the contract; one evidence item parked** - `layout: site\|bare` (default `site`), Host fallback, and the theme-authored shell (Route A) layers 1-5 are verified on the real path; layer 6 now carries automated assertions (composed shell counts, geometry classes, bare isolation, resolve-time fail-open). Open: a web-side `.vue` mount harness for the shell/island branch and a genuine 390x844 pass (carrier-limited) |
| M2 | Wire plugin add pages through the shared shell and complete Browser evidence | **Codex** | **complete (2026-10-16)** - all three exit gates verified at runtime, and every remaining item is now closed: task 5 (executable plugin page with typed loader data) via the new reference package + real Protocol V2 transport (A1); task 6's missing-runtime/disabled behaviour by citation of three existing tests re-run green (A2); task 7 at a **genuine 390x844** on real Chrome 154 over CDP (`tests/browser-public-390x844.mjs`: one visible header + one footer on shell pages, zero on `bare`, no overflow, no clipped controls, no console errors) (A3); and outer artifact identity (A4: `/site/active-theme/skin` and the plugin page's shell identity agree on `sforum.default-theme` / `85ddda6a...`, while the plugin body artifact is reported separately, with DOM markers stated per page kind and no fake attribution). Two follow-up gaps found on the way are recorded below: the enable-time runtime-instance binding (fixed) and the CLI not building the route schema catalog (pinned by a build-time test, deeper fix outstanding) |
`extensions/fixtures/plugins/sforum-page-loader-reference` was installed through the real admin
flow and the page verified at runtime (resolve: `fallback=false`, `loaderData` = the plugin's
typed document, `selectedRuntimeInstanceId` present; SSR HTML contains the rendered data; browser:
one visible header, one visible footer, no overflow). See the A1 section below for the evidence
and for the two follow-up gaps it exposed (the enable path drops the runtime instance id, and the
CLI does not build the route schema catalog so a bad OpenAPI fragment passes the CLI but breaks
API startup). A2 (missing-runtime/disabled behaviour tests) is **closed by citation** - the required
behaviours are already pinned by `TestEnableDisableSyncsPageRegistry`,
`TestLifecyclePagePublicationUsesExactRevisionAndRuntimeAdmission` and
`TestResolvePathReportsLayoutRegardlessOfBodyRender`, all re-run green in this pass rather than
duplicated. A4 (outer artifact identity evidence) is **complete**: the active-theme skin endpoint and the
plugin page's `shellProvider`/`shellPackageDigest` agree on
`sforum.default-theme` / `85ddda6a…`, while the plugin's own body artifact is reported separately
(`sforum.page-loader-reference-live` / `53af8e1c…`); DOM markers are stated per page kind without
fake attribution. A3 (390x844) is **complete** - see the run above (real Chrome 154 at 390x844 dpr3, harness
`tests/browser-public-390x844.mjs`). With A1, A2 and A4 also closed, **M2 has no open items**. Previously: decision R2 replaced the loopback-only transport with a Protocol V2 call, and that landed this round - the Loader Gateway now fetches plugin page data through `InvokeRouteInstance` for a **declared `core.guard.public`** data route, proven by a rewritten integration test (real V2 subprocess returns the typed document; undeclared/non-public/drifted/unwired cases fail closed). What remains for task 5 is the **fixture package plus browser evidence**, not a production gap |
| M3 | Add page/middleware scaffolds and focused fixture docs | **OpenCode + DeepSeek V4 Flash**, Codex review | **complete** - `--public-page`, `--public-page-layout site\|bare`, `--route-middleware` implemented in one composed generator path (Vue admin page flag reused, not duplicated); fresh scaffolds pass `extension build --allow-scaffold`, validate, contract test and exact-template runtime preflight without hand-editing; generated middleware only inherits the target guard and declares only its own mutable pointers; bilingual CLI docs and the authoring guide updated. The plugin-data page path is deliberately not generated (blocked by the loader transport gap) |
| M4 | Freeze and implement custom authorization guard V1 production binding | **Codex** | **complete as re-scoped** - the production binding was already live and is now verified end to end (assembly, authorizer dispatch, exact-artifact/trust/Safe-Mode runtime gate) with the stale "later wave / closed" disclosure corrected in all four docs. Allowed/denied coverage already existed and was mapped rather than duplicated. R3 (`core.guard.raw_request` intent) is **resolved 2026-10-16**: the operator judged it a vulnerability and it was tightened (decision R1, contract record Revision 1) - the guard id is refused at validation and fails closed at runtime, raw authority now requires a package-declared `kind: raw_request` guard. Two smaller guard-contract questions stay open for a human |
| M5 | Guard trust, inspector, lifecycle, and recovery UX | **Codex in a new conversation** | **complete** - task 1 authorization-ownership view; tasks 2/4/5 verified against emitters and tests; task 6 exact guard publication pinned by `lifecycle_route_guard_policy_test.go`; task 7 delivered as the durable quarantine read path (`GET /admin/extensions/route-incidents`) + the Route Inspector panel with loading/error/stale/empty/quarantined/recovery, rendered at 1313x930. Task 8's admin 390-wide QA is **verified** with `tests/browser-admin-390x844.mjs` (real Chrome 154 over CDP at 390x844 dpr3 with an injected session cookie: route inspector + incidents panel present, no overflow, no clipped controls, no console errors; screenshot `storage/qa/2026-10-16-admin-390x844-control-panel-extensions-route-inspector.png`). Remaining: the human call on auditing per-request guard denials |
| M6 | Add plugin-namespaced session state | **Codex** | **in progress — contract exists (Revision 2), first slices implemented and green, wire surface blocked on S9** | The blocker was that no session contract existed anywhere in the tree. That was verified 2026-10-16: Manifest V3 exposes `session` **only** as the Host-invoked identity-provider policy hook (`identityProviders[].kind: "session"`, `operations: ["session.evaluate"]`, `sessionPolicy`; `v3_validate_platform.go:740` binds it to `identity.SessionPolicy`), Protocol V2 declares `Actor.session_id` but the projection **rejects** a populated value (`sdk/plugin/v2/identity_runtime.go:235`), and Host API v2 has no session-state service. The operator then authorised adding the contract, and the four items §5.3 left open are **decided in Revision 2** of `knowledge/decisions/2026-10-16-extension-authoring-contracts-freeze.md` (S1 key derivation, S2 opaque-bytes/loss-tolerant value shape, S3 the four-way TTL bound, S4 invalidation hooks + per-session/per-extension indexes, S5 the full lifecycle policy matrix, S6 Host-owned limits, S7 authority derivation, S8 `Actor.session_id` stays unpopulated). **Implemented and green (2026-10-16):** the new `app/Support/PluginSessionState` package (S1 key derivation, S2 opaque bounded values, S3 the four-way TTL bound, S6 caps, plus the three index sets and idempotent `InvalidateSession`/`InvalidateExtension`) with `go test -race` green; the **logout** and **login-replacement** erasure hooks in `Support/AuthSession` (5 new tests); the **revoke / device-cap eviction / password reset** hooks in `Models/Identity` (with `RETURNING sid`, existing suite green); and the **disable / uninstall** erasure in `Models/Extensions` (post-commit best-effort, `service_lifecycle.go` left at exactly its 1000-line ceiling). **Blocked on a newly discovered contract item, S9:** the plugin→Host call channel is runtime-scoped, so the session is **not** derivable from it; the session scope must travel as a Host-signed single-invocation delegation (the `HostCommandDelegation`/`HostQueryDelegation` pattern). **Consequently `PluginSessionService` is deliberately NOT registered in `HostAPI/v2.go` and deliberately absent from the Host API v2 docs gate and `docs/extensions/host-api-v2.md`** — a declared-but-unserved capability is exactly what this program forbids. **S9 is now implemented (2026-10-16, later):** `HostSessionDelegation` + `RequestContext.host_session_delegation = 11`; the Host mints a single-invocation delegation on the plugin invocations it initiates inside an HTTP request (unary route, streaming route, admin surface), retires it when that invocation settles, and resolves the session scope from its own mint ledger — the token carries the runtime binding only, never the session fingerprint or its deadlines. `PluginSessionService` is **registered** on the production gateway with Safe Mode (reads miss, writes refused) and exact-runtime admission, it is documented in the Host API v2 docs gate + `docs/extensions/host-api-v2.md`, and the SDK ships `Host.SessionState(ctx)`. The three erasure consumers, which had no production caller, are now injected from `bootstrap`. Evidence: `go test ./app/Support/HostAPI/ -run Session` (11 tests / 16 subtests over real gRPC), `go test ./app/Support/AuthSession/ -run PluginSessionScope`, `go test ./app/Support/Extensions/ -run 'SessionDelegation|HostContextAcceptsInbound'`, plus the descriptor-level S7 pin in `sdk/plugin/v2/session_contract_test.go`. The **privacy-erase consumer** also landed (`bootstrap/plugin_session_state_privacy.go`: the repository's first `Support/Privacy` contribution, whose erase hook walks the user's sessions and invalidates each, counts-only, no values anywhere). **Still to do:** M9's reference fixture and anonymous session state (unavailable on today's session layer). **M10 row 5 stays open.** Each S-item is individually vetoable. |
| M7 | Narrow typed request conveniences | contract and policy by **Codex**; mechanical SDK work delegable | **complete (contract decision)** - inventory found every admitted capability already implemented with real consumers (header projection, locale/trace/deadline/idempotency, Host-validated response headers, lossless query values); all six speculative candidates rejected with evidence; **no proto change**, which M7 task 4 permits once the ADR is updated; CRLF/invalid-name and proxy-identity-spoofing tests added. See `knowledge/decisions/2026-10-16-m7-narrow-typed-request-conveniences-contract.md`. Residual: no field-level count/size cap inside the projection (transport-bounded at 4 MiB and now tested in-process with production limits) |
| M8 | Secure `extension dev` loop | core design/runtime by **Codex**; help/docs slices delegable | **complete (2026-10-16)** - the four exit-gate items are each evidenced: (1) source edit → build/digest/validate/test → staging happens in one command, and the activation step is automated **within the ADR** by `tests/admin-extension-activation.sh` (the loop itself must not bypass trust); (2) failure keeps the last healthy artifact (`TestExtensionDevLoopKeepsLastHealthyDigestOnFailure` + `tests/dev-loop-failure-scenarios.sh`); (3) a production-mode smoke now exists on both sides — the command refuses non-development environments (`TestExtensionDevRefusesNonDevelopmentEnvironments`) **and** production config refuses to scan the development scaffold tree at load time (`TestProductionRejectsDevelopmentScaffoldExternalRoots`, added 2026-10-16 to satisfy ADR 6.2.6 / M8 task 7's "reject at parsing/bootstrap, not merely hide the command"); there is no dev trust grant to activate, by ADR 6.2.2; (4) no watcher or build child survives exit (`TestExtensionDevChildProcessKillsWholeProcessGroupOnCancel` + SIGINT path). Earlier detail retained: `extension dev <package-root>` ships with the M0 gate (development-only, refused at entry), fsnotify as a direct dependency, source-input-only watch set, debounce/coalesce/single-transaction, direct reuse of `runExtensionBuild`, machine-readable mode, SIGINT cleanup, and last-healthy preservation; two real defects found by the e2e run are fixed (self-triggering loop via content stamps; preflight refusing stale-digest packages). Built-in discovery (ADR 6.2.5) is **done**: `sforum extension builtins` is the single source of truth and the script no longer hardcodes any id. Frontend build path and child-process reclamation are verified (a real defect — surviving grandchildren after SIGINT — was found and fixed with process-group kill). Admin-activation automation is now **delivered within the ADR**: `tests/admin-extension-activation.sh` drives the approved HTTP flow (install → exact-artifact trust challenge → enable with the confirmation token), refuses to continue for an executable package without a token, accepts the server's `trust_not_required` verdict for packages that ship no executable artifact, and can uninstall afterwards; verified green against the dev API (201 inert → enable 200 → disable/uninstall 200, ledger clean). Rename and API restart are verified in `tests/dev-loop-failure-scenarios.sh`; the runtime-crash clause is bounded rather than faked — a **build child** crash keeps the last healthy artifact (scenario script) and a **plugin runtime** crash is the API's concern, covered by the typed `crash` failure kind failing closed to `ErrCoreGuardEvaluatorUnavailable` without leaking the secret (`app/Http/plugin_guard_runtime_test.go`) |
| M9 | Publish reference plugins, bilingual author docs, and migration guide | **OpenCode + DeepSeek V4 Flash**, Codex review | **complete except one item that is not buildable (2026-10-16)** - canonical EN migration page + zh-CN/en-US handbooks (`docs/extensions/migrating-from-sforum-v2.md`) covering the Controller/Middleware/Blade/Session/Itf-hook/plugin-helper mapping, the deliberately closed operations, and quick starts verified in a clean temporary directory; the CLI docs gate now also covers `extension dev` / `extension builtins` in both locales; focused reference packages added (`sforum-page-layout-reference` for bare/explicit-site/default-site, plus the loader, component and schema references delivered with M2/`view: component`/`view: schema`) with fixture-level preflight + three-way-identity tests. Exit gate added 2026-10-16: `scripts/ci/reference-packages-gate.sh` runs all 10 reference packages through `digest --write` + `validate` + `test` on staged temporary copies (stating that a true clean room needs published SDK artifacts) and they all pass; it immediately found and fixed a third CLI-vs-author gap (a package with no `packageFiles` could not run `digest --write`). **The only unmet item is task 1's "plugin session-state workflow reference package".** It could not be built before because the session contract it would exercise did not exist, and building a fixture against an invented contract is exactly what the frozen record forbids — so the gap was recorded rather than filled with a fake package. **Update (2026-10-16, later the same day): the operator authorised the contract, and Revision 2 of `knowledge/decisions/2026-10-16-extension-authoring-contracts-freeze.md` decides its open items. M6's S9 delegation, service, SDK helper and docs then landed (see the M6 row), so the fixture is now buildable against a real, served contract — but it has not been built yet. M9 therefore stays "complete except this one item", and no fixture has been fabricated.** |
| M10 | Joined release gate | **Codex** in a new conversation | **open, down to two rows (2026-10-16, Revision 2 re-scoring)** - `knowledge/sessions/2026-10-16-m10-joined-release-gate-matrix.md` was re-scored against the live instance and current tree. **Evidenced: rows 1, 2, 3, 4, 6, 8, 9, 10.** Row 1 per decision **R5** (the repository has exactly one protected built-in theme, so the row validates *the existing* one; site/bare verified by the SSR+resolve runtime check, the 390x844 harness and the JS-disabled pass). Row 2 per decision **R2** (the Loader Gateway fetches plugin page data over Protocol V2; live `resolve-path` shows `fallback=false` + the plugin's typed `loaderData` + a bound runtime instance, and the override half is a real-HTTP controller test proving an override wins while a drifty `themeOverrideKey` soft-skips to the plugin template with the business data intact). Row 3 with the unsafe-method order test. Row 4 per decision **R4** (the non-existent `challenge` clause left the row; CSRF order 5 tests, uninstall 8 and restart 5 re-run green — the restart/uninstall set against a disposable PostgreSQL cluster). Row 6 by a fresh full run of `tests/dev-loop-failure-scenarios.sh` (all scenarios OK, `catalog.go` byte-identical after the restart probe). Row 8 at a real 390x844 on real Chrome 154 over CDP across five public paths including the `view: component` page (which now **is** measured mounting, after a harness settle defect was found and fixed), plus the JS-disabled pass and the admin 390 pass. Row 9 **corrected**: the release-image smoke was already run (three candidate targets, isolated database, `APP_ENV=production` health 200, seeded content, uploaded-package install/enable, executable trust gate, release compose stack); Revision 1's "not run / needs external conditions" was wrong. Row 10 by the compat farm (`ok: true`) and APILTS. **Remaining: row 5 (plugin session state — the contract is now authorised and decided in Revision 2 of the frozen contract record, so the remaining work is building M6, not deciding; nothing is claimed until M6 lands green) and row 7 (true multi-node — needs an operator-provided second node; the Go-level runtime-set barrier/reconcile tests are real but are not multi-node evidence).** The plan stays out of `archive/2026-08/` until those two close |

### M10 Revision 4 — row 5 closed, row 9 residuals closed, row 7 recorded (2026-10-17)

Row 5 is closed with per-clause evidence, and the only remaining row (7) is
operator-owned environment work:

- **T1** — the Identity/Extensions erasure hooks finally have their own tests:
  `app/Models/Identity/plugin_session_state_postgres_integration_test.go`
  (real PostgreSQL on the isolated cluster: exactly-the-ended-sid erasure for
  revoke/other/all/device-cap-eviction/password-reset, idempotent re-revoke,
  failure/panic resilience, no-invalidator resilience, cross-user isolation —
  10 tests) and `app/Models/Extensions/plugin_session_state_lifecycle_test.go`
  (real `Disable`/`Uninstall` erase exactly that extension id; failure and
  no-invalidator resilience — 4 tests).
- **T2/T3** — the M9 session-state workflow reference package exists:
  `extensions/fixtures/plugins/sforum-session-checkout-reference/` (real
  Protocol V2 backend, a two-route checkout workflow over
  `Host.SessionState(ctx)`, request/response schemas + OpenAPI, README).
  `reference-packages-gate.sh` went 10 → **11** packages and passes. The joined
  test `TestPluginSessionStateJoinedHostDispatchToStore` was **rewritten from a
  stub gRPC plugin to the compiled fixture binary**: stage → build → digest
  align → real go-plugin subprocess via `ProtocolStarter`/`Manager` → mint →
  plugin echo over real gRPC → real service → real store → settle → retire →
  replay denied → cross-session isolation → live erasure. This replaced the
  stub (correction recorded in the handoff).
- **T4 (D2)** — `privacy:erase-user` CLI: the production entry for
  `privacy.Registry.EraseUser`, gated by the real RBAC check, with the
  plugin-session-state contribution registered through the same bootstrap path
  as the API assembly (Redis unreachable = named fail-closed). Live evidence on
  a disposable migrated+seeded database with real Redis: a planted
  plugin-session key was erased, the report showed
  `core.host.plugin-session-state erased=true`, and the audit ledger recorded
  `erase|user:1|50|ok`; non-admin actors are denied live; the dev database was
  never touched.
- **T5** — matrix Revision 4: row 5 closed clause-by-clause; the anonymous
  clause closed by decision **D1** (unavailable by construction, fail closed —
  not satisfied behaviour); row 9's residuals closed (below). `validate-docs`
  green with the bilingual CLI docs updated.
- **T6** — row 9 residuals closed on the **real `v3.0.9` registry tag**
  (`pull_policy: always`, clean env file, health 200 production, SSR 200,
  plugin backends from immutable digest paths) and the **production
  `deploy/caddy/Caddyfile` reverse-proxy/TLS layer** (HTTP→HTTPS 308, H4
  security headers on HTTPS, health through the proxy, WebSocket Upgrade
  reaching the Go API, unauth SSE reaching the API). Runbook updated with the
  two newly measured fail-closed config errors (`APP_OPTION_ENC_KEY` and
  `MARKETPLACE_ED25519_PUBLIC_KEY_HEX` must both be 32-byte hex).
- **T7** — row 7 recorded: needs an operator-provided second node; nothing left
  to build in this program.
- **Production correction (recorded as ADR Revision 4.3)** — the Gateway did
  not implement the S9 issuer that `ProtocolStarter` type-asserts, so the mint
  that Revision 3 described as production-wired only worked in tests; every
  scoped Host-initiated invocation failed at mint. Fixed on the Gateway
  (unbound = fail closed; Safe Mode does not block minting) and pinned by two
  new tests plus the production-shape subprocess joined test.

### M10 re-scored, and the M6/M9 close-out (2026-10-16, second scoring pass)

**M10 Revision 2.** The joined acceptance matrix was re-scored against the live dev
instance and the current tree; every row now carries a conclusion plus its citation.
Evidenced: 1 (per **R5**, the existing built-in theme), 2 (per **R2**, both the live
loader-data half and the real-HTTP override half), 3, 4 (per **R4**, the non-existent
`challenge` clause left the row; CSRF/uninstall/restart re-run green — the
restart/uninstall set against a disposable PostgreSQL cluster), 6 (fresh full run of
`tests/dev-loop-failure-scenarios.sh`), 8 (real Chrome 390x844 across five public
paths including the `view: component` page, plus JS-disabled and admin 390), 9
(**corrected**: it had already been run; Revision 1's "not run / needs external
conditions" was wrong), 10 (compat farm `ok: true` + APILTS). Remaining: row 5
(session state — the capability does not exist; needs a session ADR revision) and
row 7 (true multi-node — needs a second node). Full detail and the corrections table:
`knowledge/sessions/2026-10-16-m10-joined-release-gate-matrix.md`.

**A harness defect was found and fixed while doing row 8.** The public 390x844
harness judged "settled" by waiting for `document.body.innerText.length` to stop
changing — a condition the *SSR* text already satisfies before hydration, so it
exited at ~2 s. On the `view: component` page the client mount happens later, so the
harness measured *before* the mount and reported "no descriptor request at all";
that reading had been carried in the adjacent frontend plan as a possibly-real client
gap, and raising the settle budget could not help because the exit condition was
"2 stable polls", not the budget. Measured discriminator: old exit ≈1.98 s vs the
component actually mounting at 5.43 s with the descriptor requests issued then. It was
a **measurement artefact**. The harness now waits for real hydration, requires a
terminal public-L2 state before accepting stability, and can assert a mount outright
(`SFORUM_L2_PATHS`), failing loudly when hydration never happens; a negative control
was run and does fail.

**M6 is recorded as needing a human decision, not as a guess.** No session contract
exists: Manifest V3 exposes `session` only as the Host-invoked identity-provider
policy hook (`identityProviders[].kind: "session"` / `operations: ["session.evaluate"]`
/ `sessionPolicy`), Protocol V2 declares `Actor.session_id` but the projection rejects
a populated value, and Host API v2 has no session-state service. The frozen M0 record
§5 fixes constraints only and §5.3 explicitly leaves key derivation, invalidation and
TTL bounding to M6. Adding the contract is a contract change and needs an ADR revision
first. Same blocker, same wording as M10 row 5.

**M9's close-out is now precise.** Everything M9 lists is delivered and cited except
task 1's "plugin session-state workflow reference package", which is not buildable for
the reason above. A fixture built against an invented contract would be a fake, so the
item is recorded as unmet rather than filled.

**M6 is now unblocked (later the same day).** The operator authorised adding the session
contract, so the milestone no longer waits on a decision: the four items §5.3 had left open
are **decided in Revision 2** of `knowledge/decisions/2026-10-16-extension-authoring-contracts-freeze.md`
— S1 key derivation (`sforum:plugin-session:v1:installation:<id>:extension:<sha256>:session:<sha256(sid)>:key:<sha256>`),
S2 opaque-bytes, loss-tolerant value shape with no migration, S3 the four-way TTL bound
(`min(requested, 24h, remaining idle, remaining absolute)`), S4 invalidation hooks backed by
per-session and per-extension index sets, S5 the full lifecycle policy matrix, S6 Host-owned
limits (the verified host-cache envelope plus ≤64 keys per session and ≤4096 per extension),
S7 authority derivation (the wire request cannot even express a caller-supplied extension or
session id), and S8 `Actor.session_id` staying unpopulated. Every S-item is individually
vetoable before implementation starts. **No code was written for this** — the deliverable was
the decision, and nothing may be reported as working until M6 lands with its allowed/denied,
cross-plugin, cross-session, rotation, logout, revoke, expiry, privacy and concurrency tests.
An implementation guide was written so M6 does not have to re-derive this:
`knowledge/sessions/2026-10-16-m6-session-state-implementation-guide.md`.

**Consequence for M9:** its last unmet item (the session-state workflow reference package)
becomes buildable as soon as M6 exists, and not before.

### M10 rows 4 and 8 closed except the non-existent clause (2026-10-16)

Row 4's uninstall and restart clauses are covered by existing tests, which were run
green in this pass rather than duplicated: uninstall (`TestUninstallRequiresDisableAndDeletesPackage`,
`TestUninstallRejectsBuiltin`, `TestUninstallRetainPackageKeepsDirectory`,
`TestUninstallMissingArtifactRequiresDedicatedCleanup`,
`TestUninstallRejectsProtectedSiteSearchBuiltin`,
`TestLifecycleSkipAndForcedUninstallPolicies`,
`TestLifecycleCoordinatorForcedUninstallCanSkipFailedCleanup`,
`TestTechAdminCannotVerifyOrUninstallUploadedBackend`) and restart
(`TestEnsureInitialPluginRuntimePublicationExactlyOnceAcrossRestart`,
`TestPublicFrontendRebuildsAssetSnapshotAfterHostRestart`,
`TestPublicPagePolicyIsDeterministicAcrossOrderAndRestart`,
`TestThemeActivationRuntimeFailureRestoresExactApprovalAcrossServiceAndRestart`,
`TestServiceSyncBuiltinsPreservesSelectedUploadedThemeAcrossRestart`,
`TestAuthProviderSettingsRestartEnabledLifecycleV2ExactArtifact`,
`TestEnableRestartSkipsConfirmation`). The only unmet clause in row 4 remains
**challenge**, which does not exist in the guard contract.

Row 8's last clause, JS-disabled rendering, is now verified:
`SFORUM_DISABLE_JS=1 node tests/browser-registry-page-shell-390x844.mjs` disables
script execution over CDP and still measures one navbar and one footer on `site` and
`default`, zero on `bare`, correct shell markers, no overflow, and SSR body text.

### M10 row 3 closed; row 4's CSRF clause cited (2026-10-16)

Row 3's missing half is now covered by
`TestExecutionPlanOrdersModifierPhasesForSafeAndUnsafeMethods`
(`app/Support/Routes/execution_plan_method_phases_test.go`): the POST plan is marked
unsafe and carries the same `global → before → filter → wrap → handler → after`
order as the GET plan, a `global_middleware` contribution appears in **both** plans
(it is exempt from method matching), and method-specific modifiers appear only in
their own plan. Writing it surfaced a real rule worth knowing: a modifier's target
must match its method, so a GET `before` must target the GET core route or the
registry rejects the input.

Row 4's CSRF clause is covered by existing tests, so none were added:
`TestCSRFMintsTokenOnSafeRequest`, `TestCSRFRejectsUnsafeRequestWithoutToken`,
`TestCSRFAcceptsUnsafeRequestWithValidToken`,
`TestP6RoutePermissionCSRFLocaleQueryAndBodyMatrix`, and
`TestNewAppArbitraryAdminRoutePreservesCSRFAndPermissionAuthority`.

### M10 row 6 closed: dev-loop failure and restart scenarios (2026-10-16)

`tests/dev-loop-failure-scenarios.sh` is a reproducible scenario script (not a unit
test) because these behaviours depend on a real `bun` child, the real
digest/validate/test implementations, real file events and real restart timing. It
asserts, and passes on a fresh run:

1. startup reconcile builds the package;
2. a package failure (a declared `packageFiles` entry removed) is reported as
   `failed stage=build` with the missing path, and `lastHealthyDigest` is kept;
3. the loop survives it and recovers after the file returns;
4. a frontend **compile failure** (`bun run build` → `exit status 7`) is reported with
   its exit status and the last healthy digest is kept;
5. after an **API restart** (documented comment probe; `catalog.go` verified
   byte-identical afterwards) the loop survives and the next edit rebuilds;
6. `SIGINT` exits cleanly and reports `stopped`.

The script cleans up its probe package, binary and temp files on exit (trap), and was
verified to leave no stray loop process.

"Crash" is covered as a **build** crash. A plugin **runtime** crash is out of the
loop's scope: the loop watches sources and does not supervise the runtime, so it has
no reaction to test; that distinction is recorded rather than glossed over.

### M5 task 8 closed: admin 390-wide QA (2026-10-16)

The last M5 gap is closed with a real browser at a genuine mobile viewport. The
harness `tests/browser-admin-390x844.mjs` launches headless Chrome over CDP, sets
`Emulation.setDeviceMetricsOverride(390x844, dpr 3)`, injects the operator session
cookie through `Network.setCookie` (the cookie is httpOnly, so `document.cookie`
cannot be used, but CDP can set it — the value is read from an environment
variable and never committed), navigates to
`/control-panel/extensions/route-inspector`, and asserts the page rendered, the
route-incidents panel exists, and there is no horizontal overflow, no clipped
control, and no console error.

Fresh run (Chrome 154.0.8037.97):

```text
/control-panel/extensions/route-inspector: 390x844 dpr=3 inspector=true incidentsPanel=true overflow=false clipped=0 consoleErrors=0
all admin 390x844 acceptance assertions passed
```

Screenshot: `storage/qa/2026-10-16-admin-390x844-control-panel-extensions-route-inspector.png`.

So "carrier-blocked" was never a property of the repository — only of the ego
sidebar carrier. Both the public-page pass
(`tests/browser-registry-page-shell-390x844.mjs`) and the admin pass now run on
demand through CDP.

### M10 row 9: candidate release image smoke (2026-10-16)

Row 9 is no longer "not run". Both candidate targets were built and exercised:

- `docker build --network=host --build-arg HTTP_PROXY/HTTPS_PROXY --target api|migrate
  -f apps/api/Dockerfile` → `sforum-api:candidate-smoke` (620MB) and
  `sforum-migrate:candidate-smoke` (29.9MB), both exit 0.
- The migrate image was run against an **isolated** database (`sforum_smoke`,
  created for the smoke and dropped afterwards; the dev database was untouched):
  145 tables plus 7 River migrations.
- The API image was started with `APP_ENV=production` against that database and
  served `GET /api/v1/health` → 200 `{"status":"ok","environment":"production"}`.

The smoke also walked the production fail-closed gates in order: placeholder
secrets panic, a missing `EXTENSION_ROOT` fails, and a missing
`MARKETPLACE_ED25519_PUBLIC_KEY_HEX` is refused. The second one produced a
misleading error (`mkdir /storage: permission denied`), so `config.Load` now
refuses a missing or relative `EXTENSION_ROOT` in production with a message that
names the variable (`TestLoadRejectsRelativeExtensionRootInProduction`); compose and
the image already used an absolute path.

Row 9's residual is closed: the **web** image (`--target prod`, 261MB) was built and
run with the api and migrate images against the isolated database. `GET /` through the
web container returned 200 with 81,120 bytes of SSR HTML (title `SForum`, exactly one
desktop and one mobile navbar — the single navbar owner's responsive pair — and
API-derived forum markup). Control experiment: stopping the api container made the same
request return **000**, proving the web image is live SSR rather than a static shell.
Containers, the smoke database, and the three candidate images were removed afterwards.

Row 9 was then extended to a **seeded** database: the host migrate and seed CLIs
prepared an isolated database (50 users with user#1 as super_admin, 2 groups, 6
categories, 12 tags, 8 topics, 17 comments) and the three candidate images served real
content through web SSR — `/` 200 (127,220 bytes, one desktop + one mobile navbar,
8 seeded topic links, 4 seeded category links), `/c/general` 200, and
`/t/1/topic-ngkxas` 200 with seeded topic content.

The extension half is evidenced as well: the image's boot synced its protected
built-ins into the database (8 plugins + 1 theme, with shortcodes/content-policy/
search-site/smtp/web-push and the default theme enabled, auth-github/storage-fs/
storage-s3 installed), the container ran 5 plugin backends as subprocesses from their
immutable digest paths, and the enabled search plugin indexed the seeded topics.

Installing and enabling an **uploaded** package through the release images is now
verified end to end: non-interactive admin password reset → login → multipart install
(201, `status: installed`, `source: uploaded`) → enable (200, `status: enabled`) →
host CLI confirms the ledger entry → the package's public pages render through the web
image with the layout contract intact (`bare` 0 navbars/0 footers/`shellSource=none`;
`site` and `default` 1/1/`shellSource=theme`).

Correction recorded: the enable path requires the exact-artifact trust challenge only
for packages that **ship an executable artifact**. The fixtures installed here declare
`backend: {entry: "", rpc: ""}` and ship no backend file, so `extension.trust_not_required`
is the correct verdict; the gate itself is on in the smoke (`APP_ENV=production` →
`SFORUM_V3_TRUST_CHALLENGES` defaults true).

The **executable** trust path is verified as well. A probe package shipping a compiled
`backend/plugin` (the `sforum-shortcodes` backend copied under the id `acme.trustprobe`,
every id-bearing field rewritten, module `replace` repointed, built `CGO_ENABLED=0
GOOS=linux`) installed at 201, issued a 64-char exact-artifact challenge with a 5-minute
TTL and a `sforum.trust-impact@2` payload, was refused at enable without the token
(409 `extension.trust_challenge_required`), enabled with it (200), and its runtime
started from the immutable digest path (6 plugin backends in the container). Disabling
and re-enabling without a fresh token returns 200 because the exact-artifact grant
persists; after `DELETE …/trust` (`trustRequired=true, trusted=false`) the consumed
token is refused with 409 `extension.trust_challenge_replayed`.

`compose.release.yaml` is exercised as well (local candidate images standing in for the
registry tag, `pull_policy: never` in a throwaway override): the override fails closed
without `SFORUM_VERSION`, then brings up postgres/redis/api/web where the api runs its
own migrations, `/api/v1/health` returns 200 with `environment: production`, the web
image serves `/` 200 with one navbar and one footer, `/c/general` 200, and the image's
built-in ledger reads 5 enabled / 3 installed / 1 theme. Three operational findings were
measured and written into
`knowledge/runbooks/2026-10-16-release-image-stack-verification.md`: the dev `.env`
leaks dev-only values into the release stack (relative `EXTENSION_ROOT` now fails closed;
`NUXT_API_INTERNAL_BASE_URL=127.0.0.1` breaks container SSR), `down` needs the same `-f`
set and `SFORUM_VERSION`, and seeding after boot needs one API restart so the search
index reconciles.

Still open for row 9: the registry pull of a real published tag, and
`compose.prod.yaml`'s reverse-proxy/TLS layer.

### A2 closed by existing coverage (2026-10-16)

A2 ("runtime missing / plugin disabled behaviour tests") was judged against the
existing suite rather than answered with a new test, as the checklist requires. All
four behaviours it asks about are already pinned:

1. **Layout is independent of body render** —
   `TestResolvePathReportsLayoutRegardlessOfBodyRender`
   (`app/Http/Controllers/Pages/controller_layout_test.go`).
2. **Plugin runtime missing** — `TestLoaderUnavailableWhenPluginRuntimeMissing`
   (`app/Http/Controllers/Pages/controller_test.go`): the page still resolves, the
   layout survives, and the response carries a `loaderError`. This is also the
   *always* path for Protocol V2 plugins (no loopback base exists), pinned by
   `app/Support/Extensions/IntegrationTests/page_loader_transport_integration_test.go`
   and `bootstrap/app_test.go`.
3. **Plugin disabled / contributions removed** —
   `TestLifecyclePagePublicationRetainsExactEmptySet` and
   `TestLifecyclePagePublicationUsesExactRevisionAndRuntimeAdmission`
   (`app/Support/Pages/registry_lifecycle_publication_test.go`): an exact empty
   publication clears the extension's pages, removal is fenced by exact artifact and
   runtime instance, and a drained runtime's page is invisible until admission.
4. **A page whose owner no longer contributes** resolves as not-found —
   `TestResolvePath404`.

Not separately asserted (recorded, not claimed): a single HTTP round trip in which
the owner transitions to disabled *between* two resolve calls. The transition itself
is covered at the registry level (3), and the post-transition HTTP outcome is the
same not-found path as (4).

### C1/C2/C3 closed earlier; this pass adds the conflict control and the copy fix (2026-10-16)

**Correction to an earlier draft of this section:** C1/C2/C3 were closed by an
earlier session the same day, in `apps/api/cmd/sforum/theme_scaffold_activation_test.go`:
the coverage hypothesis was refuted
(`TestActiveThemeDoesNotRequireFullPageCoverage`), the **real** cause was found —
the scaffold used to declare `components[]` whose `propsSchema` pointed at a Host
core-page contract, accepted by the CLI but rejected by the runtime component
registry — and the scaffold fix was applied, with CLI↔runtime parity pinned by
`TestThemeComponentSchemaDeclarationIsEnforcedByCliAndRuntime`. The scaffold also
passes `digest --write` / `validate` / `test` with 0 errors and 0 warnings.

This pass adds what was missing: the **conflict control**
(`TestThemePreflightRejectsSamePathAddOwnedByAnotherExtension`) proving the page
preflight rejects a same-path `add` owned by another extension while accepting the
same package on an unowned path, and the analysis of the two **state-dependent**
preflights that can still reject a valid package — the component registry
transition (a theme registration outside the transition fence →
`ErrComponentRegistryConflict`) and the asset publication transition (a theme
publication matching neither the outgoing nor the incoming theme →
`errAssetPublicationConflict`). Decision and evidence:
`knowledge/decisions/2026-10-16-theme-activation-preflight-truth.md`.

Actionable change made: the `extension.build_failed` copy in both locales now names
both families and states that the error text identifies the failed preflight layer
while the extension event log holds the detail. Before this, an author whose package
was valid had no direction.

Both state-dependent paths are now **observed at unit level**:
`TestComponentRegistryThemeTransitionRejectsThirdThemeOutsideTheFence` (a third
theme runtime outside the fence is rejected and clearing it makes the same switch
pass) and the new
`TestThemeAssetTransitionRejectsStaleDriftAndMultipleThemes/stale_third_theme_publication_blocks_every_switch`
subtest (a stale third-theme publication blocks every switch; clearing it fixes the
same switch). The operator procedure is
`knowledge/runbooks/2026-10-16-theme-activation-rejection-diagnosis.md`.

The improved copy is asserted by `TestThemeActivationFailureMessageNamesBothCauses`
(both locales must name the declaration family, the stale-state family, the event
log, and the failed preflight layer).

Not done: no dev-instance reproduction with live stale state.

### Route A layer 6 closed (2026-10-16, tenth session)

B1/B2/B3 of the checklist are now settled:

- **B2 was already covered** by `TestResolvePathFailsOpenToHostShellWhenDeclaredShellDisappears`
  (page still served 200, `shellHtml` empty, `layout` stays `site` so the host shell
  provides chrome). No duplicate test was added.
- **B1** is now a persistent **runtime** check:
  `tests/registry-page-shell-runtime-evidence.mjs`. It asserts, against the live SSR
  output and the resolve API, that `site` pages render the theme shell
  (`data-registry-shell-source=theme`) with exactly one navbar owner (one desktop
  `<nav>` plus one mobile `<nav>`, the same component's responsive pair) and exactly
  one footer, that `bare` renders `none` with zero chrome, that
  `data-registry-shell` carries the layout while `data-registry-shell-source` carries
  the shell source, and that `shellProvider`/`shellPackageDigest` agree with
  `/api/v1/site/active-theme/skin`. It skips with `SKIP` when the dev servers are not
  running, and fails instead when `SFORUM_EVIDENCE_REQUIRE_LIVE=1`.
  A component-level render test was attempted first and **abandoned**: `bun test` has
  no SFC transform (every existing component test reads source strings), so the
  rendering-layer proof has to be runtime evidence.
- **B3** is `knowledge/runbooks/2026-10-16-theme-switch-shell-verification.md`, a
  reproducible procedure with the expected signals per step, its known limits
  (SSR/API level, no visual proof; the public-page 390x844 pass is verified by
  `tests/browser-registry-page-shell-390x844.mjs`), and the
  requirement to switch back.

Two of my own assumptions were wrong when the script first ran and are recorded
because they were instructive: `data-registry-shell` is not the shell source, and
substring counting of `sf-public-mobile-bottom-nav` also matches its `__link`
children (6 hits instead of 1). The script now classifies `<nav>` elements by exact
class token.

### M10 matrix produced (2026-10-16, ninth session)

M10 cannot be closed honestly, so the joined acceptance matrix was produced
instead: `knowledge/sessions/2026-10-16-m10-joined-release-gate-matrix.md`. It
records, per required row, the runtime evidence collected in this pass, the
evidence that does not exist, and the blocking reason.

Evidenced this pass: the compatibility farm (`./scripts/test-api-isolated.sh
--compat` → `ok: true`, both required cells pass) and APILTS contract statuses
(`host.v2`/`manifest.v3`/`protocol.v2` current, the deprecated L1 contract with
its shim and replacement); SSR chrome counts for `site`/`bare`/default; browser
visible-chrome counts and overflow at the carrier's 500px width; and the selected
provider/digest plus shell provider/digest from `/pages/resolve-path`, which
agrees with `/site/active-theme/skin`.

Still open, with owners: rows 1, 3, 4, 6, 7, 8 need more evidence or a decision;
row 2 (plugin body data) is blocked on the loader-transport decision; row 5
(session state) is blocked because the contract does not exist; row 9 (release
image smoke) was not run.

Two clauses in M10's matrix cannot be satisfied as written and are recorded as
conflicts rather than faked: the repository has exactly **one** protected
built-in theme (`sforum.default-theme`), and guards have **no challenge** outcome
(allow or `denied`/`crash`/`timeout`/`protocol`/`canceled`/`unavailable`).

Correction: the task prompt's baseline digest `4691a60541836c57` is stale — the
active theme artifact at runtime is `85ddda6ad7825adaacfa5f741715b64a09737b41320e1a49848ca3e695edccac`,
reported identically by both identity endpoints.

### M9 reference package: page layout (2026-10-16, eighth session)

`extensions/fixtures/plugins/sforum-page-layout-reference` is a focused reference
package for the layout contract (M9 task 1's "bare public page" bullet). One
package, one template per case, three declarations: `layout: "bare"`,
`layout: "site"`, and no `layout` at all. It passes `extension digest --write`,
`validate` and `test` (0 errors, 0 warnings, 4 checks including
`template.runtime_exact`), and two fixture-level tests pin the author-facing
behaviour:

- `TestPageLayoutReferencePackagePreflightsWithExactLayouts` runs the **real
  activation preflight** and asserts both stages: the preflight preserves the
  declaration (bare / site / empty), and `prepareContributions` normalizes the
  omitted case to the platform default `site`. Asserting only the preflight would
  have missed the default, and asserting only the normalizer would have missed the
  package path.
- `TestPageLayoutReferencePackageKeepsThreeWayTemplateIdentity` verifies
  AGENTS.md hard rule 4 on the committed fixture: every `theme.json` template path
  has exactly one `templates[]` declaration and one `packageFiles[]` entry of kind
  `template`, with both digests matching the files.

Correction recorded while doing this: the `ThemePageDecl.Layout` comment in
`app/Support/Pages/theme_package.go` claimed "empty is equivalent to bare (backward
compatible)", which contradicts `NormalizeLayout` (empty → the platform default
`site`) and the runtime evidence from the M2 probes. The comment was stale; it now
states the real rule and the two-stage behaviour.

Still missing from M9 task 1: a session-state workflow fixture, which cannot be
built because no session contract exists. **Close-out (2026-10-16): this is the
single unmet M9 item, and it is blocked on a human decision (a session ADR
revision), not on effort — see the milestone table and the M10 re-scoring section
above. No fixture was fabricated to fill it.**

### M9 documentation pass (2026-10-16, seventh session)

M9's documentation half is delivered; its reference-package half is an inventory
plus recorded gaps.

Delivered: `docs/extensions/migrating-from-sforum-v2.md` (canonical English) with
the SForum-old → V3 migration table, the deliberately closed operations, and
copy-paste quick starts; `docs/zh-CN/extensions/migrating-from-sforum-v2.md` and
`docs/en-US/extensions/migrating-from-sforum-v2.md` handbooks; links from the docs
index and the authoring guide. The docs gate now derives expected subcommands from
`dev_extension.go` and `dev_builtins.go` too, so `extension dev` and
`extension builtins` must be documented in both locales (they are).

Quick starts were verified in a clean `/tmp` directory outside the repository:
`make:plugin` → `digest --write` → `validate` → `test` (0 errors, 1 scaffold-stub
warning) → `build --allow-scaffold --skip-install` all succeeded.

Two contract conflicts were found and are recorded rather than papered over:

1. **No session contract exists.** The manifest and the Protocol V2 surface expose
   no session read/write, and M6 (plugin-namespaced session state) is still blocked
   on the M0 session ADR. M9 task 1 asks for a "plugin session-state workflow"
   reference package and task 5 lists Core session mutation as closed — the second
   is accurate, the first is not buildable today.
2. **Guards have no challenge outcome.** `RuntimePluginRouteGuardEvaluator`
   either returns nil (allow) or fails with kind `denied`, `crash`, `timeout`,
   `protocol`, `canceled` or `unavailable`. M9 task 1 asks for an
   "allow/deny/challenge" reference package; only allow/deny exist.

Inventory of M9 task 1's seven bullets against the fixture tree: typed plugin page
data (`sforum-plugin-page-business-e2e`), host-inherited admin page
(`sforum-admin-surface-reference`), before/after middleware and custom guard
(`sforum-commerce-workflow`), plus focused query/event/schedule/SEO/region
fixtures already exist. A **bare public page** fixture and a session-state
workflow fixture do not exist. Note also that the repo fixtures are CI fixtures,
not clean-directory author packages: their backends declare
`replace … => ../../../../../apps/api`, so they only compile inside the repository
tree, and they ship no `backend/plugin` binary (the Go tests build it), which is
why `extension digest --write` fails on a plain copy of most of them.

### M8 delivered in part (2026-10-16, sixth session)

`sforum extension dev <package-root>` exists and follows every frozen M0 ADR 6.2
decision: the `DevHygiene` development gate (refused at entry, tested for
production/staging/test/empty), no trust bypass (each cycle reports the new
digest and a `reload_required` next step pointing at the admin confirmation),
`fsnotify` as an explicit direct dependency, no second build pipeline (it calls
`runExtensionBuild` and then reads `DigestTree`), source-input-only watching,
debounce/coalescing with one transaction at a time, `--json` for editor
integrations and `--once` for CI, SIGINT cleanup, and last-healthy preservation
on failure.

The end-to-end run found two real defects, both fixed and regression-tested: the
loop self-triggered because each transaction rewrites the manifest during digest
refresh (fixed with a per-file content-stamp snapshot re-taken after every
transaction), and the preflight rejected packages with stale digests — the exact
state the loop exists to repair (fixed; the loop now reconciles once at startup).

The frontend path was then exercised with a real probe package (`--skip-install`,
`bun run build`, dist artifacts, digest refresh, validate, contract test → `built`)
and it exposed a genuine defect: SIGINT killed the loop and `bun` but left the
build script's grandchild running. `runExtensionBuildProcess` now runs children in
their own process group and kills the group on cancel, with platform-split
implementations and two tests.

Not done: automating the admin activation step (forbidden by the ADR) and
runtime-crash/API-restart reactions.

A later pass in the same session closed ADR 6.2.5: `sforum extension builtins`
discovers packages from the manifest tree (id, relative path, backend entry, Go
backend flag, themes included) and `scripts/build-builtin-plugins.sh` consumes it,
so no id or directory name is duplicated in shell any more. Discovery reads the
manifest shape instead of `LoadPackage`, because the protected source tree has no
backend binary and would fail digest validation.

### M8 closed (2026-10-16, follow-up session)

The M8 body listed two things as not done and the later status update closed both
(admin-activation automation inside the ADR, rename + API-restart scenarios). What was
still genuinely missing against the **exit gate** was its third item: "A production-mode
smoke test proves dev grants/commands cannot activate." The command half already existed
(`TestExtensionDevRefusesNonDevelopmentEnvironments`), but production **config** accepted
an `EXTERNAL_EXTENSION_ROOTS` entry pointing straight at the scaffold tree
(`extensions/dev/...`) and would have scanned it — a convention, not an enforcement, and
exactly what ADR 6.2.6 and M8 task 7 forbid ("Production config must reject it at
parsing/bootstrap, not merely hide the command").

Closed in this session:

- `config.DevelopmentScaffoldRoot(root)` classifies a root as a development scaffold tree
  only when the cleaned path contains an `extensions/dev` segment pair; and
  `validateProductionSecrets` now **panics** on such a root in production, next to the
  existing `EXTENSION_ROOT` absolute-path gate.
- `TestProductionRejectsDevelopmentScaffoldExternalRoots` covers both sides: refused
  (`extensions/dev`, `extensions/dev/<id>`, relative scaffold path, trailing separator)
  and not misclassified (`extensions/optional`, `extensions`, an unrelated `dev`
  directory, empty entry), plus a development non-regression case that still keeps the
  scaffold root.
- No dev trust grant exists to activate (ADR 6.2.2 froze "no trust bypass"), so the
  smoke is exactly these two refusals plus the reaper gate
  (`TestShouldEnableDevelopmentOrphanReaper`).

### M7 delivered (2026-10-16, fifth session)

M7 was resolved by inventory rather than by adding fields. Every capability in
the candidate list that has a real consumer is **already** implemented (filtered
request header projection with credential/hop-by-hop/`x-sforum-*` blocking,
`locale`/`trace`/`deadline`/`idempotency_key`, Host-validated response headers,
lossless query values), and every remaining candidate — client IP/proxy
classification, user-agent/device summary, plugin-owned response cookies,
redirect hints, cache hints, rate-limit hints — is rejected with named evidence
(no consumer; or the capability already has a Host owner: proxy trust config,
the identity-provider redirect contract, the cache registry, the reviewed route
execution policies). No protobuf message was added, which M7 task 4 permits once
the ADR records the decision; the exit gate holds because no new field exists and
no SDK/proto file mentions Fiber.

Security tests added (M7 task 5): CRLF/space/colon in header names, CRLF and NUL
in header values are rejected at the projection and never reach the plugin; and
the proxy/client-identity spoofing boundary is pinned (`x-forwarded-for`,
`x-real-ip`, `forwarded`, `user-agent` are forwarded as the client's own claim,
and the Host vouches for none of them). The transport residual is closed too: a
fourth test assembles the production message limits in-process (server
`MaxRecvMsgSize`, client call options) and asserts an over-cap request document
fails at the RPC boundary and never reaches the plugin.

### M5 progress (2026-10-16, fourth session)

Delivered task 1 (and the visibility half of task 3) without changing any
frozen contract: the guard authorization join is a **derived view**, not new
fields in the digested impact document
(see `knowledge/decisions/2026-10-16-trust-impact-digest-is-authority-view-is-derivation.md`).
`ExecutableTrustStatus.guardAuthorization` and `InspectorStep.guardRisk` /
`guardRequiresTrust` now answer "which trusted artifact owns authorization for
this route" with a risk classification, executable digest, fallback, and the
declared mutable fields; the review dialog renders it with a red alert for
routes that hand authorization to raw request authority.

Verified: focused Go suites, full web suite (1207 pass), architecture gate
(1780 files, baseline untouched), V3 catalog check, docs validation, OpenAPI
refs; HTTP evidence from `GET /admin/extensions/{id}/trust` and
`GET /admin/extensions/route-inspector` on a real installed probe fixture; real
browser evidence at 1313x930 plus a 500-wide narrow pass (the carrier's true
width — never claimed as 390).

A second pass then verified by evidence rather than by assumption: task 2
(actor-bound, 5-minute, super_admin-only challenge), task 4 (the audit
vocabulary already names every required action; quarantine is audited through
`routes.committed_after_failure` + the incident row), and task 5 (guard
invocations take a `RuntimeCallGuard` lease; drain/quarantine/wait are wired and
integration-tested). Task 6's gap was real and is closed: the route-policy
publication tests asserted routes and policies but never guards, so
`app/Support/Extensions/lifecycle_route_guard_policy_test.go` now pins exact
guard publication across enable/upgrade/rollback/authoritative-empty/safe-mode.
Task 7 followed in a third pass: the durable crash-loop quarantine state had no
read path at all, so `GET /admin/extensions/route-incidents` (bounded, redacted,
summary + ledger) and the Route Inspector `route-incidents` panel now disclose
it with loading / error / stale / empty / quarantined / recovery states. A new
read route must enter four reviewed matrices — identities catalog, the
`requireExtensionsReadAuthority` provable-policy list (a real runtime wiring
requirement, not a test detail), the read-guard partition map, and the catalog
route count — and the gates caught every one. Remaining: 390-wide admin QA
(carrier-blocked) and the human call on auditing per-request guard denials.

### M3 delivered (2026-10-16, third session)

`generate`/`make:plugin`/`make:theme` gained three composable flags and no second
generation branch:

- `--public-page <slug>` writes the `theme.json` page mapping (`action: add`,
  `path /<slug>`, `access: public`, `contract sforum.page.<id>_<slug>@1`) and the
  L1 template, declared in both manifest `templates[]` and `packageFiles[]` with
  the page contract as its `viewModelSchema`.
- `--public-page-layout site|bare` (default `site`). Unknown values, a layout
  without a page, and non-slug names are rejected with a named error instead of
  producing a package that fails later.
- `--route-middleware` declares `before` (request pointers) and `after`
  (response pointers) against `core.route.forum.topics` with
  `guard: core.guard.inherit`, `fallback: closed`, and no `access`/`permission`
  restatement, plus a `//go:build ignore` example handler.
- The `--vue-admin-page` flag is reused for admin pages (no new branch); the
  interactive forms now offer these options too.

Exit-gate evidence (fresh scaffold, no hand-editing):
`extension validate` OK; `extension test` 0 errors / 1 expected scaffold-stub
warning / 7 checks including `template.runtime_exact`; `extension build
--allow-scaffold` OK; the theme variant additionally passes
`pages.ExtensionBridge.PreflightThemePackage` for both the fresh-install and
replace-active-theme states. Tests:
`apps/api/cmd/sforum/generator_authoring_workflow_test.go` (9 tests, including
flag-combination coverage with backend/complex/prebuilt/Vue/provider).

Runtime render evidence (not only CLI): the generated theme scaffold was
packaged, installed and activated through the real admin flow, then measured in
real headless Chrome over CDP at 390x844 - the generated `layout: bare` page
(`/notes`) renders with zero chrome, and the theme's own home template owns
exactly one visible navbar and one visible footer. That render exposed and fixed
two scaffold defects: the theme home template previously mounted no chrome at
all, and an island tag inside a template HTML comment breaks the exact-template
preflight because the island extractor scans comments too. The dev instance was
restored afterwards (active theme back to `sforum.default-theme` `85ddda6a…`,
both scratch themes uninstalled).

### Closure pass (2026-10-16, second session)

Four things changed after the first session's checkpoint, all evidence in
`sessions/2026-10-16-route-a-layer6-and-loader-transport-handoff.md`:

1. **A production gap was found that re-scopes M2 task 5.** The Loader Gateway's
   only transport is a plugin loopback HTTP base, and Protocol V2 deliberately
   exposes none (`protocol_v2_client.go:463`; V1 is `ErrUnsupportedProtocol`), so
   `dataSource: plugin` page contributions always fall back with
   `view_model_unavailable`. Pinned by
   `IntegrationTests/TestPageLoaderGatewayHasNoTransportForProtocolV2Runtime`
   (real subprocess + real production target source) and
   `bootstrap.TestPageRouteTargetAdapterDeniesRuntimeWithoutLoopbackTarget`.
   Task 5 and the typed-data Browser evidence are blocked on a human decision
   (see `decisions/2026-10-16-theme-activation-exactness-and-open-questions.md`).
2. **The `make:theme` scaffold defect is fixed.** The generated theme declared a
   component on a Host core page schema without the exact `packageFiles[]`
   schema declaration the component registry compiles, so activation always
   failed while the CLI passed. The scaffold no longer declares theme
   components, manifest validation now enforces the same rule the registry does
   (CLI and activation agree), the fixture set that encoded the bad shape was
   corrected, the localized failure copy names the real checks, and
   `docs/extensions/runtime-themes.md` documents exactly what activation checks.
3. **Route A layer 6 gained automated assertions** - composed shell counts,
   geometry classes, bare isolation, and resolve-time fail-open.
   Round 3 additionally produced the genuine 390x844 Browser pass (real Chrome
   over CDP, not the carrier's 500px floor) and fixed the one acceptance item it
   exposed: plugin pages had no visible footer because the Host stylesheet
   suppresses the default theme's page-level shell footer for pages with a
   content column, and a Page Registry `add` page has none. A complementary
   `:not(:has(.sforum-content-footer))` rule in
   `apps/web/app/assets/css/sforum-theme.css` restores it without duplicating
   chrome on theme pages; the active theme artifact moved `4691a605…` ->
   `85ddda6a…` through rebuild + admin reactivation. Evidence:
   `sessions/2026-10-16-route-a-layer6-and-loader-transport-handoff.md`.
4. **Three earlier claims were corrected.** "A 1-page theme is rejected for
   insufficient page coverage" is false (coverage is not required; uncovered
   pages use the Host fallback). "`data-theme-region` survives shell
   compilation" is false: the shell goes through the legacy L1 string sanitizer,
   which strips theme data attributes while keeping classes. "The M2 Browser
   acceptance is not met on plugin pages" was true when measured in round 2 and
   is obsolete after the round-3 footer fix; both 390x844 and 1440x900 now show
   exactly one visible navbar and footer.

## M0 - Contract Freeze, Inventory, And Decision Record

Assignment: **Codex in a new conversation**

Purpose: remove ambiguity before changing Page Registry, route authority,
AuthSession, or lifecycle code.

Tasks:

1. Trace the production call chain for:
   - plugin Page Registry `action=add` from manifest load through lifecycle
     publication, path resolution, Loader Gateway, SSR catch-all, template
     rendering, active theme, CSP, SEO, regions, navbar, and footer;
   - `global_middleware`, `before`, `after`, `filter`, and `wrap` from manifest
     admission through runtime invocation and response commit;
   - custom guard declaration, trust preview, runtime start, route plan,
     invocation, allow/deny mapping, inspector, audit, disable, upgrade,
     rollback, Safe Mode, and restart;
   - AuthSession issue, renew, rotate, logout, revoke, recent-auth, and external
     auth effect fencing;
   - CLI build/digest/test/discovery/trust/enable/restart operations.
2. Reconcile the custom-guard code/tests with the public documentation that
   marks the current Host guard runtime closed. Do not treat Support-only tests
   as production evidence.
3. Reconcile this program with open V3 remediation M3/M5/M6/M7/M8. Name every
   dependency and decide which plan owns closure; do not duplicate ledgers.
4. Briefly survey mature solutions before inventing new infrastructure:
   - Go template layout inheritance already supported by ThemeCompiler;
   - Nuxt runtime layout constraints;
   - fsnotify or another maintained Go watcher for the dev loop;
   - Redis/session namespace patterns with TTL and rotation semantics.
5. Write one decision record covering:
   - `layout: site|bare` schema and backward-compatible default;
   - selected-theme shell identity, exact artifact ownership, body outlet, and
     Host fallback;
   - authorization guard V1 inputs/outputs/order/failure policy;
   - explicit exclusion of actor/session issuance from guards;
   - plugin session state shape, limits, lifecycle, rotation, and privacy;
   - local dev grant and reload safety model;
   - API/LTS/versioning and rollback policy.
6. Produce a threat/inventory report with exact files, current tests, missing
   tests, reserved routes, trust inputs, and source-of-truth owners.
7. Update this task book if the inspected interfaces invalidate any assumed
   milestone boundary.

Exit gate:

- ADR accepted and linked from `knowledge/modules/extensions.md`.
- Threat/inventory report names the real production paths and open residuals.
- No open question can materially change the M1 shell schema, M4 guard
  authority, M6 session state, or M8 dev trust model.
- No production code is changed except source-derived documentation or tests
  required to prove the baseline.

Prompt:

```text
Read AGENTS.md, knowledge/index.md, knowledge/modules/extensions.md,
knowledge/modules/frontend.md, knowledge/modules/identity.md,
knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md, both active
V3 progress/remediation plans, and
knowledge/plans/2026-08-22-extension-authoring-speed-and-controlled-power.md.
Execute M0 only. Trace the real production call chains for plugin public pages,
route middleware, custom guards, AuthSession, and the extension CLI. Reconcile
Support tests with production bindings and open V3 residuals. Write the required
ADR and threat/inventory report, update durable knowledge, and run focused
documentation/catalog checks. Do not implement M1 or silently invent a missing
contract. Preserve all unrelated dirty-worktree changes.
```

## M1 - Selected-Theme Site Shell Contract

**Progress (2026-10-16): the first increment is implemented and verified.**

Delivered: `PageLayout` (`site`|`bare`) with **`bare` as the backward-compatible
default**, normalized fail-closed (`pages: invalid page layout: "fancy"` is
rejected at package preflight), carried from `theme.json` through
`PageContribution` to the `/pages/resolve-path` response; the plugin `action=add`
route shell now selects between the Host chrome shell and the unchanged bare
`<main>`; and the page body moved into one reviewed outlet
(`components/extensions/public/SFRegistryPageBody.vue`) so the two shells never
duplicate rendering branches.

Rendered evidence, two pages of the same plugin at the same digest:

| Page | `layout` | `header.navbar` | `footer.sf-footer` | Shell |
| --- | --- | --- | --- | --- |
| `/probe-bare` | undeclared -> `bare` | 0 | 0 | `data-registry-shell="bare"`, class byte-identical to pre-change |
| `/probe-site` | `site` | 1 | 1 | `data-registry-shell="site"` inside `sf-host-public-chrome` |

"Exactly one navbar and footer" is enforced by having exactly one shell branch,
pinned by a source contract test, so a body-level fallback cannot affect it.

**Remaining for M1:** two routes were found for theme shell ownership, and the
first is recommended (see ADR section 2.1). Route B: let the host shell honour the
theme's existing `navigationLocations` bindings and a geometry setting - no new
artifact type, no compiler/runtime composition. Route A (theme-authored shell
template composed with the plugin body) additionally buys control of the shell's
*markup* and is the larger cross-runtime change: `ThemeRenderedPage` is
`HTMLSegments` + `Islands`, so composing means splicing a nested render output at
the shell's island, plus a new allowed island, the web island binding, and
declarations/digests in both built-in themes.

Also remaining: the conflict / missing-artifact / digest-drift / disable /
rollback / Safe Mode / active-theme-switch tests. Until the shell ownership work
lands, `layout: site` resolves to the Host fallback shell - the contract's defined
fallback branch, not a stub.



Assignment: **Codex in a new conversation**

Purpose: define and implement one reusable selected-theme shell around
plugin-owned public page bodies without duplicating chrome.

Tasks:

1. Extend the frozen Page Registry/theme schema with the ADR-approved site
   shell identity and `layout` selection.
2. Reuse ThemeCompiler `define`/`template`/`block` support where appropriate;
   do not create a second string-template engine.
3. Add exact package-file declarations and digest validation for any new shell
   or layout artifact.
4. Teach the active theme runtime to publish exactly one shell provider and a
   Host fallback provider.
5. Define one reviewed plugin-body outlet with a typed ViewModel boundary.
6. Preserve theme override semantics: a theme may replace body presentation by
   `themeOverrideKey` without changing plugin data or access contracts.
7. Preserve the special ownership rules for auth, moderation, and virtual
   system error pages.
8. Add conflict, missing artifact, digest drift, disable, rollback, Safe Mode,
   and active-theme switch tests.
9. Update both protected built-in themes and their exact package identities.
10. Regenerate Page/Component/Route catalogs and lower any ratchet reclaimed by
    focused extraction.

Exit gate:

- The compiler/runtime can compose selected-theme shell plus plugin body from
  exact immutable artifacts.
- There is exactly one navbar and footer in every successful/fallback path.
- Theme activation and plugin lifecycle remain independently reversible.
- Focused Go tests, manifest validation, catalog generation check, and
  architecture gate pass.

Prompt:

```text
Continue the approved extension-authoring plan at M1 only. Read the M0 ADR,
threat report, current hot handoff, Page Registry/ThemeCompiler code, both
built-in themes, and all public chrome ownership tests. Implement the frozen
selected-theme site-shell contract with one typed plugin-body outlet,
layout=site|bare, exact artifacts, Host fallback, and no duplicate navbar or
footer. Preserve auth/system special cases and do not change guard/session/dev
contracts. Run focused Go tests, both theme digest/validate/test commands,
catalog checks, and architecture validation. Update durable knowledge and stop
after M1 evidence.
```

## M2 - Public Plugin Page Runtime And Browser Closure

**Progress (2026-10-16): tasks 1-4 are implemented and verified.**

- Task 1 (share one route shell/composable): the duplicated resolution plus the
  401/403/404 mapping moved into `app/composables/pages/useRegistryPageResolve.ts`.
  Both `[...sfRegistryPage].vue` and `/x/[...path].vue` are now thin shells that
  differ only in their path source and cache key; a contract test asserts neither
  route shell re-inlines `/pages/resolve-path` or `createError` again.
- Tasks 2-3 (`site` through the shared shell, `bare` explicit): the shell decision
  moved out of the route page into one shared component,
  `app/components/extensions/public/SFRegistryPageShell.vue`. `site` renders the
  host public chrome, everything else renders an explicit no-chrome container. The
  body stays in the single reviewed outlet (`SFRegistryPageBody.vue`), so the two
  shells cannot drift into duplicated rendering branches.
- Task 4 (preserve behaviour): re-verified end to end after the refactor, not
  assumed - `/probe-site` still yields exactly 1 navbar and 1 footer with the body
  rendered and no `max-w-4xl`; `/probe-bare` still yields 0 navbar, 0 footer and
  the byte-identical `max-w-4xl` container; `/x/*` still returns a clean 404 rather
  than a 5xx.

**Default flipped to `site` (2026-10-16).** The M2 exit gate ("a newly added
plugin page inherits the active site shell by default") contradicted the ADR's
original `bare` default. Measurement settled it: **0 of 5** plugin page templates
mount `sf-navbar`/`sf-footer` (control: **34 of 34** theme templates), so no
existing artifact could be double-chromed, and the real inconsistency was that
every plugin public page was the only page on the site without chrome. The shell
is presentation, which the platform already re-renders on theme activation without
re-approving plugin artifacts. Default is now `site`, `bare` is the explicit
opt-out, and unknown values are still rejected at package preflight.

Verified at runtime with the probe plugin (same digest, three pages):

| Page | declared | shell | navbar | footer |
| --- | --- | --- | --- | --- |
| `/probe-default` | none | `site` | 1 | 1 |
| `/probe-site` | `site` | `site` | 1 | 1 |
| `/probe-bare` | `bare` | `bare` | 0 | 0 |

**Task 6 progress (tests added).** Two invariants are now pinned:

- `TestResolvePathReportsLayoutRegardlessOfBodyRender` (Go, real HTTP path): a
  fixture page that declares no `layout` reports the platform default `site`, and
  it does so **regardless of what the body render did**. This is the load-bearing
  invariant for the shell: chrome selection must not depend on body success, or a
  render failure would drop the plugin page into a chrome-less bare page.
- `chooses the shell from the contribution layout alone, not from theme state`
  (web source contract): the shell component must not read active-theme state, so an
  active-theme change cannot alter whether chrome appears or how many times.

**M2 exit gates - all three now verified.** (1) "A newly added plugin page inherits
the active site shell by default" - `site` is the default, proven with three probe pages
sharing one digest. (2) "`bare` is explicit and tested" - 0 navbar / 0 footer and the
byte-identical container. (3) "An active-theme change updates the shell without
rebuilding Nuxt" - verified by switching to a scratch theme that declares no shell and
watching `data-registry-shell-source` move `theme` -> `host` while the chrome count
stayed at exactly one, with no Nuxt rebuild; then restored and re-verified. See the hot
handoff for the full sequence and the two side findings (the forged-fork conflict, and
the `make:theme` scaffold that passes the CLI but fails the runtime activation
validator).

**Remaining for M2, and what it is blocked on.** The rest is not all independent:

- Task 5 (executable plugin page with typed loader data): needs a plugin backend
  over Protocol V2; the probe has none. Independent of Route A.
- Task 6 remainder (missing runtime / disabled plugin): independent of Route A.
- Task 7: the chrome-count, overflow and console-error checks are done at desktop
  and at the narrowest width this browser carrier allows (**500x846**;
  `Emulation.setDeviceMetricsOverride` does not take effect there, so a genuine
  `390x844` pass is still owed). But task 7's **"selected theme provider/digest"**
  item cannot be produced before Route A: with the Host fallback shell a
  `layout: site` plugin page has no theme artifact in its render path at all - the
  body belongs to the plugin and the shell to the Host - so there is no theme
  provider or digest to report for those pages.
- Task 8: the re-staging step did not apply in this increment because no theme
  source was modified; the active artifact was still verified by identity
  (`sforum.default-theme` 1.0.0, packageDigest `94aec837...`, nodeRevision 3). The
  part that expects the shell to be *attributable to a theme artifact* needs Route A.

### Route A layers, and where it actually stands

Route A (theme-authored shell composed with the plugin body) was started. **Layers 1-5 are done and verified on the real path; layer 6 (tests/QA plus the
remaining M2 matrix) is not started.**

1. **Done - the plugin body island.** `sf-plugin-page-body` is declared on the API
   side in both `allowedHostIslands` (`app/Support/Pages/template.go`) and
   `productionThemeIslandBindings` (`theme_body_island.go`, component id
   `plugin.component.page_body`). A pairing test
   (`TestEveryAllowlistedHostIslandHasAProductionBinding`) now fails if the two
   tables ever drift, plus a focused test for this island. This layer is a
   prerequisite for both the theme-shell and the fallback paths and changes no
   rendering on its own.
2. **Done - the theme declaration.** `theme.json` gains an optional `shell`
   (a relative L1 template path), and `PreflightThemePackage` rejects a declared
   shell whose template cannot be loaded ("pages: shell template ..."). A theme that
   declares nothing still preflights, so the capability is optional.
   `TestThemeShellDeclarationIsPreflighted` builds real temp theme packages and
   covers declared-and-loadable, declared-and-missing, and undeclared.
   **Not yet consumed:** the field is parsed and validated but nothing renders it
   yet, so no author-facing documentation was added for it. The three-way identity
   requirement (`theme.json` path <-> manifest `templates[]` <-> `packageFiles[]`)
   is enforced for *provider* templates by `RequireDeclaredTemplates` in
   `theme_runtime.go`; extending that enforcement to the shell belongs with layer 3,
   where the shell is actually selected.
3. **Done - the API returns the shell.** `resolvePath` now calls
   `applyThemeShell`, which for `layout: site` loads the **active theme's** declared
   shell template and returns `shellHtml` plus the theme's exact identity
   (`shellProvider` / `shellVersion` / `shellPackageDigest`) through
   `resolveThemeShell`. That identity is what unblocks M2 task 7's
   provider/digest evidence. Failure policy is deliberately split from admission:
   admission is fail-closed (layer 2 rejects an unloadable shell), while resolution
   is **fail-open to the Host shell** - a shell that breaks after admission must
   degrade to the Host shell rather than render a mangled plugin page.
   `TestResolvePathLeavesShellEmptyWhenThemeDeclaresNone` pins the fallback path;
   the positive path needs layer 5 (a theme that actually declares a shell).
   Note the architecture gate was hit twice here and both times the response was an
   extraction plus a **lowered** baseline (1075 -> 1027), not a raised one: the
   shell wiring became a one-line call, and the page access-enforcement cluster
   moved to `controller_pages_access.go`.
4. **Done - the web island binding.** `plugin.component.page_body` is registered in
   both web maps (`islandComponents` and `legacyIslandBindings`) in
   `SFThemeTemplate.vue`, bound to `SFPluginPageBodyIsland.vue`, which renders the
   reviewed body outlet. Because a theme shell's island is an independent component
   that cannot see the parent's slot, the body is now passed as **data**:
   `SFRegistryPageShell` was refactored from slot-based to a `body` prop and
   `provide`s it (`useRegistryPageBody`). The shell has three mutually exclusive
   branches - theme shell / Host fallback / explicit `bare` - exposed as
   `data-registry-shell-source="theme"|"host"|"none"`.
   Re-verified after the refactor rather than assumed: `/probe-default` and
   `/probe-site` still report `site` + `host` with exactly 1 navbar and 1 footer, and
   `/probe-bare` still reports `bare` + `none` with 0 and 0 and the unchanged
   `max-w-4xl` container.
5. **Done - built-in theme, activated and verified.** `sforum-default` ships
   `templates/plugin-page-shell.html` (mirroring the theme's own page structure:
   `sf-page sf-theme--default sf-theme-shell--fullwidth-3col` +
   `.sf-theme-chrome-body[data-theme-region=body]` + the `sf-plugin-page-body`
   island), declared with its exact digest in both `templates[]` and
   `packageFiles[]`, and referenced by the new `theme.json` `shell`. The active
   theme artifact was then re-staged and activated through the normal admin
   preview flow, moving the active digest `94aec837...` -> `4691a605...`.

   Verified on the real path (same plugin digest, three pages):

   | Page | declared | source | navbar | footer | body in DOM |
   | --- | --- | --- | --- | --- | --- |
   | `/probe-default` | none | `theme` | 1 | 1 | 1 |
   | `/probe-site` | `site` | `theme` | 1 | 1 | 1 |
   | `/probe-bare` | `bare` | `none` | 0 | 0 | 1 |

   The theme geometry survives compilation (`sf-theme-shell--fullwidth-3col` and
   `.sf-theme-chrome-body` classes are preserved; verified live), and the body
   renders inside `.sf-theme-chrome-body` as intended.
   **Correction (2026-10-16, second session):** the original entry also claimed
   `data-theme-region` survived. It does not - the shell is rendered through
   `LoadTemplate` -> `SanitizeTemplateHTML` (bluemonday theme policy), which
   strips `data-theme-region` / `data-theme-owned` / `data-page` /
   `data-theme-shell`. Class-based geometry and the wrapper's
   `data-extension-id` (which the built-in theme's CSS keys on) are unaffected.
   Pinned by an assertion in `TestResolvePathReturnsExactThemeShellWhenDeclared`. A first count appeared to show the body
   twice; that was a measurement error (the second hit was the SSR payload's copy of
   `templateHtml`), and a DOM-scoped count gives exactly 1.

   **Dev-root finding, recorded because it explains the documented trap.** In this
   dev setup `storage/builtin-dev/air.env` sets
   `BUILTIN_EXTENSION_ROOT=<repo>/storage/builtin-dev`, and `air` loads that env file,
   so the running API syncs built-ins from the **build output**, not from
   `extensions/builtin`. Editing a built-in theme source therefore has no effect until
   it is copied into `storage/builtin-dev` and the API restarts. That is the concrete
   mechanism behind the `AGENTS.md` warning.

   **Correction to an earlier claim in this same entry.** I first wrote that
   `scripts/build-builtin-plugins.sh` "only handles plugins - it has no theme step"
   and that its hardcoded id list disagreed with `extensions/README.md`. Both were
   wrong, and in different ways:

   - The script rsyncs the **whole** `extensions/builtin/` tree, so themes are staged
     by it already; only the `go build` section is plugin-specific, which is correct
     because themes have no backend binary. My grep used the ASCII word `theme`
     against a script whose comment says `主题`, so I concluded "no theme step" from a
     missed match rather than from reading it. Running the script staged the theme
     with the shell template and `extension validate` passed on the staged copy.
   - The eight hardcoded plugin ids **exactly match** the eight directories that
     actually contain `backend/`, so the list is correct and complete. The real
     README/script disagreement is the opposite one: `extensions/README.md` listed
     only **5** protected built-ins, so it was stale by three
     (`sforum-auth-github`, `sforum-web-push`, `sforum-shortcodes`). The README has
     been corrected, including a note that themes are staged by the same whole-tree
     copy and that a theme's source digest must already be correct because themes are
     copied verbatim rather than rebuilt.

   So there is no missing dev step to close in M9 for themes; the remaining doc-level
   work there is only the README drift, which is now fixed.

6. **Partially done - tests for the composed path.** The manual evidence from layer 5
   is now pinned automatically:
   `TestResolvePathReturnsExactThemeShellWhenDeclared` points the harness's active
   theme at a dedicated fixture (`extensions/fixtures/themes/plugin-shell-demo`, which
   declares a shell and embeds the body island) and asserts the returned `shellHtml`
   carries `sf-plugin-page-body` plus the exact identity
   (`demo.shell-theme` / `2.0.0` / `shell-digest`). Its counterpart
   `TestResolvePathLeavesShellEmptyWhenThemeDeclaresNone` keeps the fallback honest.
   A dedicated fixture was needed rather than extending the shared
   `page-registry-demo` one, because that fixture is the harness's default active
   theme and adding a shell to it would have flipped every other resolve test onto the
   theme-shell path.

   **Still open in layer 6 / M2 (updated 2026-10-16):** the executable plugin page with
   typed loader data now has its **production transport** (decision R2: the Loader Gateway
   calls `InvokeRouteInstance` for a declared `core.guard.public` data route, with an
   integration test on a real Protocol V2 subprocess); what is left is the fixture package
   (a V2 plugin that serves the data route and ships the page contribution), its dev
   install, and the browser evidence. Still open as before: tests for missing runtime /
   disabled plugin, a genuine `390x844` Browser pass (this carrier cannot set a 390px
   viewport), and the theme-artifact re-staging step is demonstrated manually but not
   scripted - `scripts/build-builtin-plugins.sh` still has no theme step.

**Therefore the corrected order is: Route A first, then close M2.** Route A is the
larger cross-runtime change (theme-authored shell composed with the plugin body)
and the task book assigns it to a fresh conversation, which is why it was
deliberately not started inside a long-running session.


Assignment: **Codex in a new conversation**

Purpose: make the real Nuxt catch-all use M1 and prove runtime behavior at
desktop/mobile, SSR, active-theme switch, and failure paths.

Tasks:

1. Refactor `[...sfRegistryPage].vue` and `/x/[...path].vue` to share one route
   shell/composable rather than duplicating resolution logic.
2. Route `layout: site` through the shared Page Outlet/shell ownership path.
3. Route `layout: bare` through an explicit no-chrome path.
4. Preserve 401 redirect, 403, 404, SEO, locale, Loader Gateway, noindex policy,
   CSP aggregation, regions, and L2 fail-closed behavior.
5. Add one executable plugin-page reference with typed loader data and one
   static page fixture; neither may rely on test-only shortcuts.
6. Add SSR and behavioral tests for site/bare, selected theme, Host fallback,
   missing plugin runtime, disabled plugin, and theme switch.
7. Run Browser QA on desktop and `390x844`:
   - exactly one navbar/footer;
   - selected theme provider/digest and `data-template=1` where expected;
   - plugin body rendered from typed data;
   - no horizontal overflow, clipped controls, console errors, or duplicate
     chrome;
   - bare page contains no implicit chrome.
8. Validate active immutable theme artifacts, not only source templates.

Exit gate:

- A newly added plugin page inherits the active site shell by default.
- Active-theme change updates the shell without rebuilding Nuxt.
- `bare` is explicit and tested.
- Browser evidence and canvas/page screenshots are recorded in the handoff.
- Full web tests run because public chrome/Page Registry authority changed.

Prompt:

```text
Continue the approved extension-authoring plan at M2 only. Use the merged M1
shell contract. Refactor the dynamic public plugin-page catch-alls into one
shared implementation, wire layout=site through the selected-theme shell and
layout=bare through the explicit bare path, and preserve SSR/access/SEO/CSP/
regions/L2 behavior. Add real plugin-page fixtures and focused tests. Rebuild,
stage, activate, and verify the exact built-in theme artifact. Run full web
tests, typecheck, focused Go tests, architecture validation, and Browser QA at
desktop and 390x844. Record provider/digest/template and single-navbar/footer
evidence. Stop after M2.
```

## M3 - Page And Middleware Scaffolds

Assignment: **OpenCode + DeepSeek V4 Flash**, followed by **Codex review**

Purpose: turn the frozen M1/M2 and existing route middleware contracts into
small, generated author workflows.

Tasks:

1. Add composable generator flags without duplicating generator branches:
   - `--public-page <name>`;
   - `--public-page-layout site|bare`;
   - `--admin-page <name>` or reuse/extend `--vue-admin-page` cleanly;
   - `--route-middleware`.
2. Generate valid Manifest V3/theme/page/template/schema/package-file entries.
3. Generate a backend handler with one safe `before` request patch and one
   `after` response patch using declared mutable fields.
4. Generate source tests and an immediately valid placeholder distribution.
5. Keep flags repeatable only if the generator already has a safe idempotent
   merge path; otherwise reject duplicate output clearly.
6. Extend generator tests for combinations with backend, settings, provider,
   and Vue admin page options.
7. Update CLI help and author docs. Do not add custom guard generation yet.

Exit gate:

- A fresh scaffold passes `extension build --allow-scaffold`, digest,
  validation, and contract test without hand-editing.
- Generated pages use `layout: site` by default.
- Generated middleware never requests raw request authority.
- Existing scaffolds remain byte/contract compatible unless the ADR explicitly
  approved a versioned change.

Delegation prompt:

```text
Read AGENTS.md and M3 of
knowledge/plans/2026-08-22-extension-authoring-speed-and-controlled-power.md.
M1 and M2 contracts are already merged and are authoritative. Implement M3
only in the sforum generator, generator tests, CLI help, and directly related
author docs. Add --public-page, --public-page-layout, and --route-middleware;
extend --vue-admin-page only where the task book explicitly requires it.
Generate valid Manifest V3 exact-artifact entries and safe before/after
middleware examples. Do not change Page Registry runtime, Route Dispatcher,
trust, guards, sessions, or lifecycle. Run focused generator tests,
extension build/digest/validate/test on a temporary scaffold, and architecture
validation. If a contract is unclear, stop and report it instead of inventing
one. Preserve unrelated worktree changes.
```

Codex review prompt:

```text
Review the completed M3 diff against the M0-M2 contracts. Prioritize generator
combination bugs, stale digests, invalid manifests, path traversal, accidental
raw authority, non-idempotent output, and missing real CLI smoke tests. Fix only
M3 defects, run the stated gates, update the plan/handoff, and do not begin M4.
```

## M4 - Custom Authorization Guard V1

**M0 finding (2026-10-16): this milestone's premise is wrong - the production
binding already exists.** `bootstrap/api_assembly_domains.go:618` injects
`NewRuntimePluginRouteGuardEvaluator(...)` into the production guard authorizer;
`app/Support/Routes/core_guard_authorizer.go:229` dispatches to it; and
`app/Http/plugin_guard_runtime.go:32` invokes the plugin runtime behind exact
artifact/version/digest/trust/Safe-Mode gates. Manifest admission already accepts
`kind: custom` (`app/Support/ExtensionManifest/v3_validate.go:106`).

M4 is therefore **not** "productionize the binding". It is:

1. Freeze and verify the existing V1 contract (inputs, outputs, ordering, failure
   policy) against the traced code rather than the plan's assumption.
2. Correct the stale disclosure in the unsafe direction: `docs/extensions/routes.md:283,284,399`
   and `docs/extensions/authoring-guide.md:741` tell authors that a request-path,
   high-risk power is closed while it is live. Keep `custom` and `raw_request` as
   two distinct states - raw-session credential forwarding is genuinely still
   closed, per the evaluator's own comment.
3. Close allowed/denied coverage on the production path.
4. Re-scope only after 1-3; do not write a second guard runtime.

Evidence chain: `knowledge/sessions/2026-10-16-m0-threat-and-inventory-report.md`
section 2.

Assignment: **Codex in a new conversation**

Purpose: productionize route-scoped authorization guards without granting
authentication or raw session authority.

Tasks:

1. Implement the M0 guard V1 schema and generated Protocol/SDK surface.
2. Bind guards to exact route id, action, method/path, contract, artifact,
   runtime epoch, instance, and grant.
3. Freeze invocation order relative to Host authentication, CSRF, permission,
   route middleware, handler, and response commit.
4. Limit V1 inputs to the ADR-approved filtered fields and Host-attested actor.
5. Limit V1 results to stable allow, deny, and challenge shapes. Map errors to
   stable API responses without leaking plugin messages or stderr.
6. Wire the real production dispatcher/startup publication path; no direct
   manager-only success qualifies.
7. Enforce deadline, cancellation, concurrency, payload, and circuit-breaker
   limits.
8. Fail closed on timeout, crash, stale runtime, digest drift, schema mismatch,
   unavailable provider, and malformed response.
9. Keep Core callback/session/recovery routes closed according to their catalog
   policy.
10. Add allowed and denied black-box tests plus unsafe-method/CSRF ordering
    evidence.

Exit gate:

- A real HTTP request reaches a trusted guard through the production route
  path and returns correct allow/deny behavior.
- The same artifact disabled, revoked, stale, crashed, or in Safe Mode cannot
  authorize a request.
- The guard cannot create an actor, change permissions, write a session, or
  observe undeclared credentials.
- Protocol, SDK, OpenAPI, catalogs, compatibility fixtures, and architecture
  gates pass.

Prompt:

```text
Continue the approved extension-authoring plan at M4 only. Read the M0 guard
ADR/threat report, current V3 remediation state, Route Registry dispatcher,
Protocol V2 guard code, trust/lifecycle publication, Host guard ordering, and
reserved route catalog. Implement the frozen custom authorization guard V1 in
the real production path. It may allow/deny/challenge only and must not mint an
actor, issue or mutate a session, or receive undeclared raw credentials. Add
allowed/denied/timeout/crash/stale/revoke/Safe Mode/unsafe-method black-box
tests, update wire/SDK/OpenAPI/catalogs, run focused then broad Go gates and
architecture validation, update durable knowledge, and stop after M4.
```

## M5 - Guard Trust, Inspector, Lifecycle, And Recovery UX

Assignment: **Codex in a new conversation**

Purpose: make M4 operable and recoverable rather than merely callable.

Tasks:

1. Include guard identity, route scope, requested fields, risk, executable
   digest, and fallback in the exact impact document.
2. Require a separate short-lived actor-bound `super_admin` confirmation for a
   new or changed guard grant.
3. Expose selected guard/provider/order/status in Route Inspector and extension
   detail without exposing secrets or raw headers.
4. Audit grant, select, change, deny reason category, revoke, failure,
   quarantine, disable, upgrade, rollback, and uninstall events.
5. Make disable/upgrade close admission and drain guard calls before stopping
   the exact runtime.
6. Prove failed upgrade compensation restores the prior complete route/guard
   publication or remains safely disabled.
7. Add UI loading, empty, error, stale, disabled, and recovery states.
8. Run desktop/mobile Browser QA for the affected admin inspectors.

Exit gate:

- Operators can understand exactly which trusted artifact owns authorization
  for each affected route.
- Revoke/disable/Safe Mode works without invoking the guard.
- Crash loops quarantine the exact surface without blocking Host recovery.
- Admin UI and API report the same durable state after restart.

Prompt:

```text
Continue the approved extension-authoring plan at M5 only. M4 guard execution
is merged. Complete exact trust impact, separate super_admin confirmation,
Route Inspector/detail visibility, audit, admission/drain, disable/upgrade/
rollback/uninstall compensation, quarantine, and recovery UI. Test restart and
failure paths through real PostgreSQL/runtime publication. Run focused and
broad Go/web gates, architecture validation, and desktop/mobile Browser QA.
Do not add raw request authority or session APIs. Update durable knowledge and
stop after M5.
```

## M6 - Plugin-Namespace Session State

Assignment: **Codex in a new conversation**

Purpose: support small browser-workflow state without exposing Core session
authority.

Tasks:

1. Implement the M0-selected storage and typed/wire contract in an owner outside
   the AuthSession God path; AuthSession remains authoritative for identity.
2. Derive current session, actor, extension id, runtime identity, and grant from
   Host-attested context. Reject caller-supplied authority.
3. Enforce the frozen key/value/count/TTL limits and namespacing.
4. Define and implement anonymous/authenticated, rotation, replacement, logout,
   revoke, expiration, disable, uninstall, privacy erase, and Safe Mode policy.
5. Ensure state cannot extend Core session TTL or survive a forbidden
   transition.
6. Add Host API v2 clients and small SDK helpers.
7. Add audit only for policy-sensitive operations; never log values.
8. Add allowed/denied, cross-plugin, cross-session, stale runtime, rotation,
   logout, revoke, expiration, privacy, and concurrency tests.
9. Add one real reference workflow that needs state; do not ship a storage API
   with no product-level consumer evidence.

Exit gate:

- Plugin A cannot read/write Plugin B state.
- A plugin cannot address another session or user.
- Session rotation/logout/revoke follows the ADR exactly.
- Values never appear in audit, public API, logs, or extension inventory.
- Full AuthSession and external-auth suites pass, not only new helper tests.

Prompt:

```text
Continue the approved extension-authoring plan at M6 only. Read the M0 session
ADR/threat report, AuthSession implementation and all external-auth/session
decisions/tests. Implement Host-owned plugin-namespaced session state with the
frozen limits and lifecycle policy. Derive all authority from the attested
runtime/request context; never accept a trusted extension id, user id, or
session id from the plugin. Do not expose or mutate Core session claims and do
not let plugin state extend Core TTL. Add one real reference consumer and full
cross-plugin/session/rotation/logout/revoke/privacy tests. Run AuthSession,
identity, Host API, extension, full Go, OpenAPI, and architecture gates. Update
durable knowledge and stop after M6.
```

## M7 - Narrow Typed Request Conveniences

Assignment: contract and policy by **Codex**; mechanical generated SDK/helper
work may be delegated to **OpenCode + DeepSeek V4 Flash** after the contract is
merged.

Purpose: cover real middleware/guard use cases without exposing Fiber.

Candidate capabilities, admitted only with M0/M4 evidence:

- declared request header projection;
- Host-resolved client IP/proxy classification;
- user agent/device summary;
- locale, trace, deadline, and idempotency context;
- Host-validated response headers;
- Host-mediated response cookie operations for plugin-owned names only;
- redirect, cache, and rate-limit hints where existing owners can enforce them.

Tasks:

1. Inventory actual reference-plugin needs; reject speculative fields.
2. Define field-by-field sensitivity, capability/grant requirements, maximum
   sizes/counts, normalization, spoofing boundary, redaction, and audit.
3. Keep raw Cookie, unrestricted Authorization, connection handles, request
   body streams, Fiber locals, and arbitrary response callbacks closed.
4. Add typed Protobuf messages and Host/SDK helpers only after the ADR is
   updated.
5. Add spoofed proxy/header, CRLF, duplicate header, oversized value, stale
   runtime, and cross-extension tests.

Codex contract prompt:

```text
Continue the approved extension-authoring plan at M7 only. Read the M0 ADR,
M4 guard contract and production evidence, current Host API v2 request/response
projections, Fiber adapter boundary, proxy/IP configuration, header and cookie
owners, and the real M3/M9 reference use cases. Inventory actual consumer needs
before adding fields. Freeze field-by-field source ownership, sensitivity,
normalization, spoofing boundary, limits, grants, redaction, audit, and LTS
policy; keep raw Fiber, Cookie, unrestricted Authorization, streams, locals,
and callbacks closed. Implement the Host/policy/proto portion and its security
tests, or explicitly reject candidates without a justified consumer. Run proto,
Host API, guard, extension, OpenAPI, architecture, and relevant broad Go gates.
Update durable knowledge and stop before delegating mechanical SDK work.
```

Delegable SDK prompt, only after Codex freezes the contract:

```text
Read AGENTS.md, the merged M7 contract commit, and M7 of the extension-authoring
task book. Implement only the named generated Go SDK/helper/documentation slice
for the already frozen request-context fields. Do not change policy, Host
authorization, Fiber adapters, guards, sessions, manifests, or trust. Regenerate
from authoritative proto sources, do not hand-edit gen files, and run proto
check plus focused SDK tests. Stop if the frozen contract and code disagree.
```

Exit gate:

- Every new field has a real consumer, stable source owner, and spoofing test.
- No extension API mentions Fiber types.
- Header/cookie outputs are Host-normalized and cannot inject transport
  delimiters or overwrite Host-reserved security/SEO headers.

## M8 - Secure `extension dev` Loop

Assignment: core design/runtime by **Codex**; isolated CLI help/completion/docs
slices may be delegated after the command contract is merged.

Purpose: reduce the common edit-to-browser loop to one command without
mutating live immutable artifacts or weakening production trust.

Tasks:

1. Add `sforum extension dev <package-root>` using the M0 watcher/library
   decision.
2. Restrict it to development mode and configured dev/external roots.
3. Watch only package-contained source inputs; ignore build outputs, package
   archives, VCS directories, and temp files to prevent rebuild loops.
4. Debounce/coalesce changes and permit one build transaction at a time.
5. Run the existing build, digest, validate, and contract-test implementations
   directly rather than shell-parsing their output.
6. Stage a new immutable snapshot and reload through the approved lifecycle
   path. Never overwrite the active digest in place.
7. Define local dev trust exactly as M0 approved. Production config must reject
   it at parsing/bootstrap, not merely hide the command.
8. Preserve the last healthy runtime when build/preflight fails.
9. Print one concise phase/status stream with actionable errors and exact
   artifact identity.
10. Handle SIGINT, child process cleanup, rapid edits, backend/frontend-only
    changes, deleted files, rename, runtime crash, and API restart.
11. Add a machine-readable mode for editor integrations.

Exit gate:

- One command takes a generated plugin from source edit to refreshed local
  page/route without manual digest or API restart.
- Failure leaves the last healthy artifact active and clearly reports why.
- A production-mode smoke test proves dev grants/commands cannot activate.
- No watcher/build process remains after command exit.

Prompt:

```text
Continue the approved extension-authoring plan at M8 only. Read the M0 dev-mode
ADR, current CLI build/digest/validate/test implementations, extension source
discovery, immutable staging, lifecycle restart, trust, and process cleanup.
Implement `sforum extension dev` as a development-only watch/build/preflight/
immutable-stage/reload loop. Reuse internal command implementations, preserve
the last healthy artifact, reject production dev grants at bootstrap, avoid
watch loops, and clean up all child processes. Add rapid-edit/failure/restart/
SIGINT/production-rejection integration tests. Run CLI, extension, full Go, and
architecture gates. Update durable knowledge and stop after M8.
```

Delegable help/docs prompt, only after the M8 command contract is merged:

```text
Read AGENTS.md, the merged M8 command contract, and M8 of the extension-authoring
task book. Implement only CLI help text, shell completion, and directly related
bilingual documentation for the already working `sforum extension dev` command.
Do not change watcher behavior, lifecycle, staging, trust, development-mode
checks, process control, or runtime code. Derive flags and examples from the
real command, run CLI help/completion snapshots plus documentation validation,
and stop if implementation and the frozen contract disagree.
```

## M9 - Reference Packages, Documentation, And Migration Guide

Assignment: **OpenCode + DeepSeek V4 Flash**, followed by **Codex review**

Purpose: make the completed platform discoverable and prove the beginner path
without adding new architecture.

Tasks:

1. Publish focused fixtures/reference packages for:
   - site-layout public page with typed plugin data;
   - bare public page;
   - host-inherited admin page;
   - before/after route middleware;
   - custom authorization guard allow/deny/challenge;
   - plugin session-state workflow;
   - combined dev-loop smoke.
2. Keep each package focused. Do not create another all-capabilities fixture.
3. Update English canonical extension docs and Chinese handbooks.
4. Add a SForum-old migration table mapping Controller, Middleware, Blade,
   Session, Itf hook, and plugin helper patterns to V3 contracts.
5. Document unsupported operations plainly: raw Fiber, Core session mutation,
   arbitrary Nuxt Layer, pre-plugin middleware, and actor minting.
6. Add copy-paste quick starts that pass in a clean temporary directory.
7. Regenerate catalogs and verify all docs links.

Exit gate:

- Every reference package passes build, digest, validation, and contract tests
  through public author commands in a clean temporary directory.
- English and Chinese documentation describe the same supported powers,
  trust requirements, defaults, recovery path, and deliberately closed APIs.
- The migration table contains no claim that depends on a test-only or
  Support-only runtime path.
- Catalog drift, documentation validation, link checks, and architecture
  validation pass.

Delegation prompt:

```text
Read AGENTS.md and M9 of the completed extension-authoring task book. All M1-M8
contracts are merged and must not be changed. Add focused reference packages,
English canonical docs, Chinese handbook updates, and the SForum-old migration
mapping. Reuse existing SDKs/scaffolds and exact-artifact commands. Do not
change runtime, trust, session, guard, Page Registry, or lifecycle policy. Run
extension build/digest/validate/test for every package, docs/catalog drift
checks, link checks, and architecture validation. Stop and report any contract
gap instead of patching architecture.
```

Codex review prompt:

```text
Review M9 for false capability claims, fixtures that bypass production paths,
stale digests, unsafe examples, missing Chinese/English parity, broken links,
and old-version advice that exposes Fiber/session authority. Fix only M9
defects, rerun every package contract test and documentation gate, update the
handoff, and do not begin M10.
```

## Human decisions resolved (2026-10-16)

The six open items this program had accumulated were decided by the operator. The
contract-affecting two are recorded as **Revision 1** in
`knowledge/decisions/2026-10-16-extension-authoring-contracts-freeze.md`; the other four
settle acceptance wording or naming:

| # | Decision | Effect |
| --- | --- | --- |
| R1 | Empty-binding `core.guard.raw_request` forwarding credentials is a **vulnerability**; tighten it so raw authority is explicit and declared | contract change; the stale comment and docs must be corrected with the implementation |
| R2 | Plugin page data gets a **Protocol V2 data-fetch method** (not a loopback HTTP base, not dropping the capability) | unblocks M2 task 5 / A1, M10 row 2, and the `view: schema` form-submit path |
| R3 | Per-request plugin-guard denials stay **unaudited** | closes the M5 judgement call as a decision |
| R4 | The `challenge` clause **leaves** M10 row 4 | row 4 becomes satisfiable without a contract change |
| R5 | M10 row 1 validates **the existing** built-in theme | no second built-in theme is in scope |
| R6 | `view: schema` keeps `schema.descriptor` + `schema.contract` | the adjacent plan's interpretation is confirmed |

Implementation order agreed with the decisions: R1 (security tightening, with
allowed/denied tests and the corrected comment/docs) first, then R2 (protocol method plus
the pinned transport test rewritten into the new contract), then the acceptance-wording
rows close on their existing evidence.

### R1 implemented (2026-10-16)

Status: **production side done, both sides evidenced through the real author commands.**

- `v3_validate.go` no longer accepts `core.guard.raw_request` as a route guard; a route
  declaring it is refused with an error that names the replacement
  (`declare a guard with kind=raw_request (entry + digest + contractVersion)`).
- Every runtime mapping fails closed: `Routes/request_authority.go`,
  `Routes/prepare.go`, `Routes/core_guard_authorizer.go`,
  `Extensions/protocol_v2_request_authority.go`. `route_dispatcher.go` already refused it.
- Kept deliberately (non-granting, disclosure only): the `GuardCoreRaw` constant,
  `guard_risk.go`'s raw-risk label, and `Models/Extensions/trust_service.go`'s
  `requestsRawRequest`, so a legacy artifact that still carries the id is still
  **disclosed** as high risk even though it can no longer execute.
- The misleading comment in `app/Http/plugin_guard_runtime.go:53-54` was corrected.
- Fixture: `extensions/fixtures/plugins/sforum-commerce-workflow` now declares its own
  `kind: raw_request` guard (`sforum.commerce-workflow.guard.raw`, entry `backend/plugin`)
  for its `create-topic-replace` route, which is exactly the allowed side of the new
  contract. Its guard `entry` is backed by the existing `kind: executable` packageFiles
  entry, which `v3_validate_ui.go` requires.
- Docs corrected: `docs/extensions/routes.md`, `docs/zh-CN/extensions/routes.md`,
  `docs/en-US/extensions/routes.md`, `docs/extensions/authoring-guide.md`, the M7
  convenience-contract table, the module note, and the open-question record (resolved in
  place, evidence retained).

CLI evidence, both sides, through the public author commands on a real package with a
real backend binary (built with the fixture's own go.mod):

```text
# allowed side: the package declares its own kind=raw_request guard
$ go run ./cmd/sforum extension validate /tmp/cwprobe
OK  /tmp/cwprobe
$ go run ./cmd/sforum extension test /tmp/cwprobe
  errors: 0  warnings: 0

# denied side: same package, that one route set back to core.guard.raw_request
$ go run ./cmd/sforum extension validate /tmp/cwdeny
Error: invalid package at /tmp/cwdeny: extensions: invalid manifest: core.guard.raw_request is
  no longer accepted as a route guard; declare a guard with kind=raw_request
  (entry + digest + contractVersion) and reference that guard id instead
$ go run ./cmd/sforum extension test /tmp/cwdeny
extension test failed: 1 error(s), 0 warning(s)
```

Test fallout was repaired in **seven** packages by updating the tests that encoded the old
contract: assertions that "the core raw guard grants raw authority" were inverted into
refusal assertions, and tests that merely used that guard as a vehicle for raw-authority
behaviour were switched to a package-declared `kind: raw_request` guard so the allowed
side stays covered. Packages: `app/Support/ExtensionManifest`, `app/Support/Routes`,
`app/Support/Extensions`, `app/Support/Extensions/IntegrationTests`, `app/Http`,
`app/Support/ExtensionOpenAPI`, and `app/Models/Extensions` (21 failures there came from
one shared helper that already declared its own raw guard but still pointed the route at
the retired id).

Two findings worth keeping:

1. **A test was passing vacuously.** `TestRouteInvocationAuthorityRejectsExactBindingDrift`
   (Routes) put the retired guard on a `RoutePhaseAfter` step; the denial is absorbed by the
   `committedAfterFailure` path, so dispatch returned success and the 13-mutation drift
   matrix never ran. It now uses a declared guard and asserts the invoker was reached, so it
   cannot go vacuous again. Worth checking the same "denial swallowed as
   committed-after-failure" shape wherever a test asserts a *rejection* through the response
   stage.
2. **R1 exposed a real publication bug, now fixed.** `buildLifecycleRegistryMaterial`
   did **not** copy `extension.Manifest.Guards` into the route set, while the startup path
   (`buildStartupRoutePublicationMaterial`) did. With R1 in force, raw authority is only
   reachable through a package-declared guard, so that omission would have made every such
   route fail closed at enable time. The two paths now carry the same contract. The fix grew
   `lifecycle_registry_publication.go` past its 1036-line cap, so the shared material
   identity/digest helpers (`registryMaterialDigest`,
   `registryMaterialCompatibleDigests`) moved to
   `lifecycle_registry_publication_components.go` and the baseline was **lowered** to 1018
   (the package's flat-directory cap forbids adding a file).
3. **A defense-in-depth gap was closed in production code.**
   `app/Http/plugin_guard_runtime.go`'s `EvaluatePluginGuard` returned "allowed" for a step
   carrying `core.guard.raw_request` with an empty binding (`if step.PluginGuard.ID == "" {
   return nil }`). It was unreachable from the production chain once `validPluginGuardBinding`
   rejects the guard, but the evaluator should not rubber-stamp an authority it is handed; it
   now fails closed for that guard **and** for the empty binding.

Also measured while closing this: `app/Http/Controllers/Notifications`'s
`TestRevisionStreamReconcilesMissedWakeAndHeartbeats` fails once under a full `go test ./...`
run and passes 3/3 in isolation - a load-sensitive timing flake, not R1 fallout. Everything
else in `go test ./...` is green.

### R2 in progress (2026-10-16) - the loader transport seam

**Reconnaissance result (measured, not assumed).** The host -> plugin Protocol V2 route
transport already exists and needs **no wire-contract change**:

- `RuntimeInvoker.InvokeRouteInstance(ctx, identity, ProtocolV2RouteRequest)`
  (`Support/Extensions/manager_route_transport.go`), dispatched by
  `ProtocolStarter.InvokeRouteInstance` through the **exact** process (no active-process
  fallback), which is what the page loader needs.
- Production already uses it: `app/Http/route_dispatcher_v2.go:62`.

So R2 is **not** "add a gRPC method". It is "let the Page Loader Gateway use the existing
V2 route transport instead of demanding a loopback `BaseURL`", which also avoids touching
the frozen wire contract.

**Done this round (green):**

- `Pages/loader.go`: the response post-validation (size cap, JSON shape, sensitive-key
  scan, optional schema) was extracted into `validateLoaderDocument`, and a new
  `PageDataLoader.FetchV2(ctx, LoaderRequest, LoaderDocumentFunc)` runs the *same*
  validation on the V2 path. Both transports now share one security boundary instead of
  two copies that could drift. `LoaderDocumentFunc` is a plain function type, so the
  `Pages` package still does not import `Support/Extensions`.
- `Pages/loader_gateway.go`: `LoaderRouteTarget` gained an optional `Invoke
  LoaderDocumentFunc` (mutually exclusive with `BaseURL`); the gateway prefers it when
  present and keeps the loopback HTTP path otherwise. A target with neither stays closed
  (503).
- Tests: allowed (data really reaches the gateway over the V2 closure), validation parity
  across the table of rejected documents, no-transport fail-closed, and a direct
  `FetchV2` schema test. All green.
- **Fixed a real gap found while writing those tests:** `ValidateDataRoute` compared
  reserved prefixes as `bad + "/"`, so the already-slashed `/api/` only ever matched
  `/api//` and `/api/anything` passed validation. The rule now rejects `/admin`, `/api`
  and `/control-panel` **and their sub-paths**, with tests.

**R2 is now complete (same round).** What landed on top of the seam above:

- `bootstrap.pageRouteTargetAdapter` returns a target with `Invoke` when the plugin exposes no
  loopback `BaseURL`, binding the closure to the **exact** runtime instance identity plus the
  admission context the adapter already holds. The loopback HTTP path is untouched.
- `pageDataRouteAdapter` (new, bootstrap) resolves the contribution's `dataRoute` against the
  extension record's manifest and **only accepts `core.guard.public` GET routes**; duplicate
  declarations are refused. It depends on a one-method `Get` interface so it can be stubbed.
- Refusals are typed: `pages.ErrLoaderRouteRefused` lets the gateway answer 503 "plugin page
  data route unavailable" for "this route may not be invoked", while genuine call failures stay
  504. Before that split, a refusal was mislabeled as a transport failure.
- Production wiring switched to `NewPageRouteTargetSourceWithRoutes(extensionRuntime, extensionStore)`.
- `IntegrationTests/page_loader_transport_integration_test.go` was **rewritten** (the pinned
  "no transport exists" test is gone). Allowed side: a real Protocol V2 subprocess serves the
  declared public data route and the gateway receives the plugin's typed document
  (`title`/`slug`/`locale`/`actorHint` asserted), twice, proving the admission lease is released.
  Denied side: undeclared route, non-public guard, artifact digest drift, and an unwired route
  source each fail closed with 503 and no data. The helper subprocess asserts the loader call is
  GET + handler stage + **filtered** authority with **no actor**.
- `go test ./...` (apps/api) is fully green, including `bootstrap` and both integration suites.

**A1 progress (2026-10-16, same round).** The reference package exists:
`extensions/fixtures/plugins/sforum-page-loader-reference/` - a real Protocol V2 backend
(`backend/main.go` serves one public GET data route), a page contribution, a template and the
page-data schema. It passes the author commands on the source tree
(`extension digest --write`, `validate` -> `OK`, `test` -> 0 errors / 6 checks including
`template.runtime_exact`), and
`TestReferencePageLoaderFixtureServesTypedDataThroughProductionChain` runs the whole chain:
stage the fixture in a temp dir, build the backend, align the manifest digest, `LoadPackage`,
start the real V2 subprocess, register the page through the real registry, resolve the path,
then fetch through the production Loader Gateway and assert the plugin's typed document
(`title` = "Loaded from the plugin runtime: hello", `state` = "published").

Three findings from building it, all worth keeping:

1. **The canonical `theme.json` data shape is nested** - `data: {source, route, schema}` - not
   the flat `dataSource`/`dataRoute`/`dataSchema` keys. The older fixture
   `sforum-plugin-page-business-e2e` uses the **flat** keys, which are silently ignored, so
   that fixture never had data wiring at all (consistent with it being static). My first
   attempt copied the flat shape and produced an empty `DataSource`/`DataRoute` on the resolved
   contribution; the nested shape fixes it.
2. **`TypedDocument.SchemaId` must be the id without `@version`**; the version goes in
   `SchemaVersion`, while manifest references use `id@version` (`validateProtocolV2DocumentRef`
   splits them). Getting this wrong surfaces as `protocol v2 route response is invalid`.
3. **The loader's own schema check is lightweight** (`type` + `required` only); `enum`/`format`
   enforcement belongs to the compiled `ThemeRuntimeSnapshot` before template execution. The
   test asserts required-field enforcement and says so, instead of pretending the loader
   enforces enums.

Also: `.gitignore` gained `extensions/fixtures/plugins/*/backend/plugin` (mirroring the
builtin/optional rules) so the locally built fixture binary never lands in the repo.

### M9 exit gate run (2026-10-16): all reference packages pass the author loop

New gate `scripts/ci/reference-packages-gate.sh` runs every package under
`extensions/fixtures/plugins/` through `digest --write` + `validate` + `test` **on a staged copy in a
temporary directory**, building a backend when the package declares one and rewriting the SDK
`replace` to the in-repo `apps/api` path. Its output states the honest boundary: this is a staged
in-tree copy, **not** a clean room - a true clean room needs published SDK artifacts (release-time,
M10 row 9).

```text
$ bash scripts/ci/reference-packages-gate.sh
  page-registry-demo: digest + validate + test passed
  sforum-contract-events: digest + validate + test passed
  sforum-contract-schedules: digest + validate + test passed
  sforum-page-component-reference: digest + validate + test passed
  sforum-page-layout-reference: digest + validate + test passed
  sforum-page-loader-reference: backend built / digest + validate + test passed
  sforum-page-schema-reference: digest + validate + test passed
  sforum-plugin-page-business-e2e: digest + validate + test passed
  sforum-prebuilt-settings: digest + validate + test passed
  sforum-region-demo: digest + validate + test passed
reference packages checked: 10
reference package gate: OK (staged in-tree copies, not clean-room)
```

**The gate immediately found a third CLI-vs-author gap, now fixed.** `sforum-contract-events` and
`sforum-contract-schedules` declare no `packageFiles` at all (they only declare routes/events
contracts), and `extension digest --write` answered
`Manifest V3 packageFiles must be inline for digest refresh` - an error for a package that
`extension validate` / `extension test` accept, i.e. the documented author loop broke on step one
for a perfectly legal package shape. `manifest_digest.go` now treats "no package files" as an
explicit no-op: it validates the package and prints
`no package files declared; <manifest> validated`, while a **present but non-inline** array still
errors (`must be an inline array`). Pinned by
`TestExtensionDigestCommandAcceptsPackageWithoutPackageFiles` plus the pre-existing
`TestExtensionDigestCommandRefreshesGeneratedArtifact`.

**A3 is complete (2026-10-16): a genuine 390x844 pass, on a real-Chrome carrier.** The
`ego_*` carrier cannot apply `Emulation.setDeviceMetricsOverride` (measured `innerWidth` is always
500), so this used the same CDP technique as the admin harness: real headless Chrome, metrics
override, `mobile: true`. New durable harness `tests/browser-public-390x844.mjs` (public pages need
no session), which fails - not silently passes - if the override does not take effect.

```text
$ node tests/browser-public-390x844.mjs        # SFORUM_QA_DIR=storage/qa
public 390x844 browser evidence (web=http://127.0.0.1:3000)
  browser: Chrome/154.0.8037.97 viewport=390x844 dpr 3
  /loader-articles/hello: 390x844 dpr=3 shell=site/theme headers=1 footers=1 overflow=false clipped=0 consoleErrors=0
  /probe-site:            390x844 dpr=3 shell=site/theme headers=1 footers=1 overflow=false clipped=0 consoleErrors=0
  /probe-bare:            390x844 dpr=3 shell=bare/none  headers=0 footers=0 overflow=false clipped=0 consoleErrors=0
  all public 390x844 acceptance assertions passed
screenshots: storage/qa/2026-10-16-public-390x844-{loader-articles-hello,probe-site,probe-bare}.png
```

So every clause of task 7 is now measured at the real 390x844: exactly one visible navbar and one
visible footer on shell pages, **zero** on the explicit `bare` page, no horizontal overflow, no
clipped controls, no console errors. The earlier "carrier blocked / 500x846" note is superseded by
this run rather than left standing.

**A4 is complete (2026-10-16): outer artifact identity, with honest attribution.**

`GET /api/v1/site/active-theme/skin` (the active theme artifact):

```json
{"extensionId":"sforum.default-theme","version":"1.0.0",
 "packageDigest":"85ddda6ad7825adaacfa5f741715b64a09737b41320e1a49848ca3e695edccac"}
```

`GET /api/v1/pages/resolve-path?path=/loader-articles/hello` (the plugin page):

| Field | Value | Meaning |
| --- | --- | --- |
| `selectedProvider` / `extensionId` | `sforum.page-loader-reference-live` | the **plugin** owns the page body |
| `selectedVersion` / `selectedPackageDigest` | `1.0.1` / `53af8e1ce208f72dde18b418a46c7e9b8ec8b02410b1dae439517ea363fc3f14` | the plugin's own exact artifact |
| `shellProvider` / `shellVersion` / `shellPackageDigest` | `sforum.default-theme` / `1.0.0` / `85ddda6ad7825adaacfa5f741715b64a09737b41320e1a49848ca3e695edccac` | the **theme** owns the shell, and this digest is **identical** to the skin endpoint above |
| `layout` / `fallback` | `site` / `false` | theme shell applied, nothing fell back |

So both endpoints agree on the selected theme provider/version/digest, and the plugin's artifact
identity is reported separately rather than conflated with it.

**DOM markers, stated precisely (this is where the task book warned about faked attribution):**

```text
core page   GET http://127.0.0.1:3000/            -> data-provider="sforum.default-theme"  data-template="1"
                                                    data-theme-owned="presentation"  data-page="forum.home"
plugin page GET http://127.0.0.1:3000/loader-articles/hello
                                                 -> data-registry-shell-source="theme"  (no data-provider,
                                                    no data-template, no data-theme-owned)
```

A plugin page's body is the **plugin's** template, not a theme template, so `data-provider` /
`data-template="1"` do not apply to it and are **not** claimed here; the theme's participation is
the shell, proven by `data-registry-shell-source="theme"` plus the matching
`shellPackageDigest`. The core page keeps the theme-runtime markers. Claiming `data-template="1"`
for the plugin page would have been exactly the "fake attribution" the task book forbids.

**A2 is closed by citation (2026-10-16) - no synonymous tests were written.** The requirement
was "behaviour tests for a missing runtime and for a disabled plugin". Both behaviours are already
pinned by tests that exist for their own reasons, so this pass **ran** them and records the result
instead of duplicating them:

| Required behaviour | Existing test | Where | Result |
| --- | --- | --- | --- |
| Plugin disabled -> its page stops being served; re-enable restores it without an API restart | `TestEnableDisableSyncsPageRegistry` | `app/Models/Extensions/page_lifecycle_effect_test.go` | PASS (0.00s) |
| Runtime missing/drained -> the published page is not visible until admission returns true | `TestLifecyclePagePublicationUsesExactRevisionAndRuntimeAdmission` | `app/Support/Pages/registry_lifecycle_publication_test.go` | PASS (0.00s) |
| A failing body render must not change the reported layout (chrome ownership is body-independent) | `TestResolvePathReportsLayoutRegardlessOfBodyRender` | `app/Http/Controllers/Pages` | PASS (0.00s) |

The loader side of "runtime unavailable" is additionally covered by the R2 tests
(`TestPageLoaderGatewayWithoutTransportStaysClosed`, the undeclared/non-public/drift cases in
`TestPageLoaderGatewayRefusesNonPublicDataRoutes`), which assert fail-closed fallback with no data.

The only thing this pass added on top is the enable-time runtime-instance binding test
(`TestPageRegistryAdapterBindsPluginRuntimeInstanceAtEnable`), and that covers a gap the above do
**not**: it is about the contribution carrying the instance id at enable time, not about
admission behaviour.

**A1 is complete (2026-10-16).** The reference package was installed into the dev instance
through the real admin flow (install 201 -> exact-artifact trust challenge -> enable 200) and the
page was verified at runtime:

```text
$ curl 'http://127.0.0.1:8080/api/v1/pages/resolve-path?path=%2Floader-articles%2Fhello'
provider                  = "sforum.page-loader-reference-live"
fallback                  = false
reason                    = null
loaderError               = null
dataSource                = "plugin"      dataRoute = "/page-data/article"   layout = "site"
loaderData                = {"state":"published","title":"Loaded from the plugin runtime: hello"}
selectedRuntimeInstanceId = "5b0cf5669f6ca612e697e8c506fc6f2e"

$ curl -s http://127.0.0.1:3000/loader-articles/hello | grep -o 'plugin-loader: [^<]*'
plugin-loader: Loaded from the plugin runtime: hello / published        <- in the SSR HTML

browser DOM (http://127.0.0.1:3000/loader-articles/hello, 1238px):
shell=site  shellSource=theme  visible header=1  visible footer=1  visible nav=1
overflow=false (scrollWidth == innerWidth == 1238)
pluginText = "plugin-loader: Loaded from the plugin runtime: hello / published"
screenshot: /tmp/a1-loader-page-desktop.png
```

The evidence is a plugin page whose body data is fetched from a **real Protocol V2 subprocess**
through the production Loader Gateway, rendered server-side, inside the theme shell, with exactly
one visible navbar and one footer and no horizontal overflow.

**Two production gaps found while getting that evidence (both need follow-up):**

1. **The enable path did not carry the runtime instance id into page contributions - FIXED
   (2026-10-16).** Only the
   startup restore (`restoreExactPluginPagePublications`) sets
   `contributions[index].RuntimeInstanceID`; `PageRegistryAdapter.RegisterPluginPackage` ->
   `toThemeExt` never does, and `pages.ThemeExtension` has no such field. Result: right after an
   enable, the exact loader refuses with `pages: exact plugin runtime unavailable`
   (`resolved.RuntimeInstanceID` is empty, so `selectedRuntimeInstanceId` is absent from the
   resolve payload), and the page only starts working **after an API restart** - a real defect in
   the author experience: "enable the plugin, page still empty until restart".

   **Fix:** `PageRegistryAdapter` gained a `PageRuntimeInstanceSource` seam
   (`WithRuntimeInstances`) and binds the exact active instance id to plugin page contributions
   **before** the template runtime snapshot is compiled and the contributions are registered,
   mirroring the startup restore path. `bootstrap` injects a resolver over the runtime manager
   that returns an id **only** on an exact version+digest match (otherwise the contribution stays
   unbound and request-time admission decides, instead of faking identity).
   `TestPageRegistryAdapterBindsPluginRuntimeInstanceAtEnable` pins all three sides (exact match
   binds, mismatched artifact stays unbound, unwired adapter stays unbound).

   **Runtime verification of the fix** (dev instance, admin flow with the trust token):
   disable `sforum.page-loader-reference-live` -> enable -> resolve **immediately, without any
   API restart**:

   ```text
   fallback        = false
   loaderError     = null
   loaderData      = {"state":"published","title":"Loaded from the plugin runtime: hello"}
   selectedRuntimeInstanceId = "1141cb0d1e538169c7a23988ddb1cb07"   (fresh instance from the re-enable)
   ```
2. **CLI and runtime disagree about route schemas.** `extension validate` / `extension test`
   reported `0 errors` for a package whose route declared `responseSchema` while its OpenAPI
   fragment could not resolve it; the API then **failed to start**
   (`prepare startup route schema publication: ... route schema is missing`), taking the dev API
   down until the extension was disabled out of band. This is the same class of gap as item C.
   The fragment contract an author must satisfy (none of it checked by the CLI) is:
   `operationId` namespaced under the fragment namespace, `x-sforum-route-id`,
   `x-sforum-contract-version`, `x-sforum-guard`, `x-sforum-response-schema`, a
   `responses.<status>.content.application/json.schema.$ref` into `#/components/schemas/<ref>`,
   and `components.schemas` keyed by **exactly** the manifest's schema reference.
   `TestReferencePageLoaderRouteSchemasResolve`
   (`app/Support/ExtensionOpenAPI/reference_page_loader_catalog_test.go`) now pins this for the
   reference package at build time.

   **Deeper fix shipped (2026-10-16):** the CLI now builds the same catalog. `loadValidatedExtensionPackage`
   (`cmd/sforum/validate.go`, shared by `extension validate` and `extension test`) calls
   `preflightRouteSchemaCatalog`, which runs `extensionopenapi.BuildRouteSchemaCatalog` on the author
   tree for any package that declares `openapi[]` or `routes[]`. The check lives in the CLI rather
   than in the SDK preflight so the runtime keeps a single catalog build on its own path. Verified
   both ways: a route whose `responseSchema` has no resolvable OpenAPI definition now fails
   `extension validate` with the *same* message the API used to fail startup with
   (`route schema catalog: … route schema is missing`), and the same package with a resolvable
   fragment passes. Pinned by `TestExtensionValidateBuildsRouteSchemaCatalog` (denied + allowed).
   One fixture had been carrying exactly this latent landmine: `sforum-contract-events` declared
   route schemas with no OpenAPI fragment at all, so it would have failed at enable/startup; it now
   ships a proper fragment (`openapi/routes.yaml` with the namespaced `operationId`,
   `x-sforum-route-id` / `-contract-version` / `-guard` / `-request-schema` / `-response-schema`,
   `requestBody` + `responses` refs into `components.schemas`) and passes the author commands.

**Dev-instance note:** the canonical `sforum.page-loader-reference` row in the dev database is
left `disabled` with a dangling active version (an earlier uninstall removed the version row), so
it cannot be enabled or cleaned up through the admin API (`extension.artifact_missing` /
`missing_artifact_cleanup_invalid`). The evidence above was therefore taken with a temporary
renamed copy (`sforum.page-loader-reference-live`); the canonical row needs the operator's
"missing artifacts" cleanup (or a manual row removal) before it can be used again.

**Remaining for A1 (original note, kept for the audit trail):** dev install of this package plus the browser DOM evidence.

**A1 is now unblocked.** The web plumbing already carries `loaderData` end to end
(`useRegistryPageResolve` -> `SFPageOutletRender` -> `SFRegistryPageBody`/`SFThemeTemplate`), so
what A1 still needs is a **fixture package** (a small V2 plugin that declares the public data
route, implements `InvokeRoute`, ships a page contribution with `dataSource: plugin`, and renders
the document) plus its dev install and browser evidence. The existing
`extensions/fixtures/plugins/sforum-plugin-page-business-e2e` is static (no backend, no routes),
which is exactly why it could never exercise the loader.

**Original remaining list for R2 (kept for the audit trail):**

1. `bootstrap.pageRouteTargetAdapter` must return a target with `Invoke` when
   `target.Target.BaseURL` is empty but the runtime exposes `InvokeRouteInstance`
   (`bootstrap/app.go` currently rejects that case). The closure binds the exact
   `RuntimeInstanceIdentity` plus the admission context already held by the adapter.
2. The closure needs the **declared route** for the contribution's `dataRoute`: `RouteID`,
   `ContractVersion`, `RequestSchema`, `ResponseSchema` and its guard. `GuardPolicyCatalog
   .LookupDeclaredRoute` only exposes `Access`/`Permission`, so the adapter should read the
   route from the extension record's manifest (`extensions.Store`, already available in
   bootstrap via `pagePackageRootAdapter`) and require an exact version/digest match.
3. **Authorization rule for the V2 loader (fail closed):** the loader carries **no actor
   identity** - `ActorID` is documented as a hint that plugins must not treat as
   authorization, and the V2 request can express permissions, so passing it would silently
   upgrade the hint into authority. Therefore the loader may only invoke a declared route
   whose guard is `core.guard.public`; `login`/`permission` cannot be evaluated without an
   actor, and `custom`/`raw_request` guards own their own authorization, which the Host must
   not evaluate on their behalf. Anything else falls back (503).
4. Rewrite `IntegrationTests/page_loader_transport_integration_test.go` into the new
   contract: it currently pins "no transport exists" (`TestPageLoaderGatewayHasNoTransport
   ForProtocolV2Runtime`). It must become allowed (a V2 runtime with a declared public data
   route returns loader data) plus denied (undeclared route / non-public guard / artifact
   drift -> fallback), which is also the fixture work for A1.
5. Only then are `dataSource: plugin` pages and the `view: schema` form submit buildable
   (adjacent plan M7's remaining piece).

### Original R1 checklist (measured before implementation, kept as the audit trail)

The hole is the **Host-owned** guard id: `core.guard.raw_request` may be referenced by a
route with an **empty** plugin-guard binding (`validPluginGuardBinding`,
`app/Support/Routes/core_guard_authorizer.go:237`) yet it maps to raw request authority.
A package-declared guard with `kind: raw_request` is the explicit form and already
requires entry + digest + contract + trust disclosure, so it stays.

Call sites that currently treat `GuardCoreRaw` as raw authority (all must become
fail-closed; validation must reject the reference so authors get an early error):

| Site | Current behaviour |
| --- | --- |
| `app/Support/ExtensionManifest/v3_validate.go:95,97` | accepts `core.guard.raw_request` as a route guard and lists it in `rawRequestGuards` |
| `app/Support/Extensions/protocol_v2_request_authority.go:79` | `GuardCoreRaw` -> `RequestAuthorityRaw` |
| `app/Support/Routes/request_authority.go:190` | same mapping on the plan path |
| `app/Support/Routes/execution_plan.go:450` | same mapping while building steps |
| `app/Support/Routes/prepare.go:66,217` | treats the declaration as raw authority |
| `app/Http/route_dispatcher.go:88` | dispatches it as raw |
| `app/Support/ExtensionManifest/guard_risk.go:26` | flags it as raw-risk (keep: it is the disclosure signal) |

Test sweep to update in the same change (44 references across ~20 files): the matrix and
dispatcher tests in `app/Http/route_request_authority_matrix_test.go`,
`route_dispatcher_test.go`, `core_guard_authorizer_test.go`,
`route_guard_production_chain_integration_test.go`, and the manifest tests
`v3_manifest_test.go`, `v3_fixtures_test.go`, `v3_route_mutable_fields_test.go`,
`guard_risk_test.go`, plus the `app/Support/Routes` suites. New coverage required:
allowed (a package-declared `kind: raw_request` guard still receives raw authority) and
denied (`core.guard.raw_request` is refused at validation **and** fails closed if it ever
reaches the runtime).

Comment correction already applied (2026-10-16): the stale "credential forwarding remains
closed" note in `app/Http/plugin_guard_runtime.go:53-54` now states the traced behaviour
and the decision.

## M10 - Joined Release Gate

Assignment: **Codex in a new conversation**

Purpose: prove that individual milestone success composes into one recoverable
production system.

Required matrix:

1. Public page `site` and `bare` under both protected built-in themes.
2. Plugin body data plus theme override without business-schema drift.
3. Middleware before/after/filter/wrap/global ordering on safe and unsafe
   methods.
4. Custom guard allow/deny/challenge, CSRF order, timeout, crash, quarantine,
   revoke, disable, upgrade, rollback, uninstall, Safe Mode, and restart.
5. Plugin session state across anonymous/authenticated flows, rotation, logout,
   revoke, expiration, privacy erase, disable, uninstall, and runtime restart.
6. Dev loop success, compile failure, schema failure, crash, rapid edit, API
   restart, and production rejection.
7. Multi-node desired revision, exact artifact, guard provider, shell provider,
   and runtime acknowledgement consistency.
8. Browser desktop/mobile, SSR, JS-disabled where applicable, selected provider
   and digest, single chrome, no overflow, and no console errors.
9. Candidate release image smoke; local source-only evidence is insufficient.
10. Compatibility farm and APILTS evaluation for every changed public contract.

Commands and gates:

```bash
node tests/validate-architecture-boundaries.mjs
cd apps/api && go test ./app/Support/Pages/... ./app/Support/ThemeCompiler/...
cd apps/api && go test ./app/Support/Extensions/...
cd apps/api && go test ./app/Support/AuthSession/... ./app/Support/HostAPI/...
cd apps/api && go test ./...
cd apps/web && bun test
cd apps/web && bun run typecheck
cd apps/web && bun run build
ruby scripts/validate-openapi-refs.rb
./scripts/test.sh
```

Run every reference package through `extension build`, `digest --write`,
`validate`, and `test`. Rebuild and activate exact built-in theme artifacts
before Browser evidence.

Exit gate:

- Every required matrix row has production-path evidence.
- No open severity P0/P1 finding remains in this program.
- Existing V3 remediation rows are updated honestly rather than duplicated.
- Relevant module notes, decisions, docs, plan status, hot handoff, and
  `knowledge/index.md` are synchronized.
- The plan moves to `archive/2026-08/` only after the release gate closes.

Prompt:

```text
Execute M10 only for the extension-authoring speed and controlled-power plan.
Read all milestone handoffs, the durable ADR/threat report, active V3 residual
ledger, and the real current code. Build a joined acceptance matrix covering
public shell, middleware, custom guards, plugin session state, dev reload,
multi-node lifecycle, Safe Mode, rollback, both built-in themes, and release
image behavior. Run the full Go/web/OpenAPI/architecture/repository gates,
every reference package test, exact artifact activation, and desktop/mobile
Browser QA. Fix defects within the approved contracts; reopen the owning
milestone for contract gaps. Do not claim completion from source-only or
Support-only tests. Update all durable knowledge and archive the plan only when
every exit condition is met.
```

## Delegation Safety Matrix

| Work item | OpenCode + DeepSeek V4 Flash | Codex |
| --- | --- | --- |
| Generator flags after contract freeze | recommended | review |
| CLI help, shell completion, bilingual docs | recommended | review |
| Focused fixture manifests/templates after contract freeze | recommended | review |
| Mechanical generated SDK helpers after proto freeze | acceptable | review |
| Page Registry/theme shell ownership | no | required |
| Public chrome/SSR/CSP/SEO integration | no | required |
| Custom guard protocol/policy/production binding | no | required |
| Trust, authorization, actor, CSRF, audit, recovery | no | required |
| AuthSession or plugin session-state semantics | no | required |
| Immutable lifecycle/dev reload transaction | no | required |
| Multi-node/release-image/final acceptance | no | required |

## Required Test Principles

- Prefer black-box external package tests for public contracts.
- Every authorization surface has both allowed and denied tests.
- Every unsafe request proves CSRF and guard ordering.
- Every exact-artifact surface tests stale digest, stale epoch, wrong instance,
  revoke, disable, upgrade, rollback, and Safe Mode.
- Every page path proves SSR and single chrome; source string tests alone do not
  close UI behavior.
- Every dev convenience has a production-rejection test.
- Every session-state test proves cross-plugin and cross-session denial.
- Fixtures must use the same CLI/load/runtime paths available to authors.
- Full web tests are mandatory for public chrome/Page Registry authority.
- Architecture ratchets must decrease when the work extracts responsibility;
  never raise a baseline merely to fit this program.

## Completion Definition

This program is complete only when an extension author can:

1. scaffold a public page that automatically inherits the active site shell;
2. scaffold an admin page that inherits the Host admin shell;
3. add safe route middleware without learning internal dispatcher types;
4. add an explicitly trusted route-scoped authorization guard without touching
   Core sessions or creating actors;
5. store bounded plugin-owned workflow state for the current Host session;
6. run one local command that rebuilds, validates, stages, and reloads an
   immutable development artifact;
7. package the same source for production without a dev-only trust artifact;
8. recover through disable, rollback, Safe Mode, and CLI when plugin code
   fails.

The program is not complete if any success depends on raw Fiber access, raw
Core session mutation, a copied navbar/footer, mutable bytes under an active
digest, a test-only runtime shortcut, or an undocumented production flag.
