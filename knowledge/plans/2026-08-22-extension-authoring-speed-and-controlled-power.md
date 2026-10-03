# Extension Authoring Speed And Controlled Power - Task Book

Status: **ready** - product direction approved; implementation has not begun

Date: 2026-08-22
Last updated: 2026-08-22 - initial task book

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
| M0 | Freeze contracts, inventory real call chains, write ADR and threat model | **Codex** | not started |
| M1 | Implement selected-theme public site-shell contract and fallback | **Codex** | not started |
| M2 | Wire plugin add pages through the shared shell and complete Browser evidence | **Codex** | not started |
| M3 | Add page/middleware scaffolds and focused fixture docs | **OpenCode + DeepSeek V4 Flash**, Codex review | blocked by M2 |
| M4 | Freeze and implement custom authorization guard V1 production binding | **Codex** | blocked by V3 route residual inventory |
| M5 | Complete guard trust, inspector, audit, lifecycle, and recovery UX | **Codex** | blocked by M4 |
| M6 | Add plugin-namespaced session state | **Codex** | blocked by M0 session ADR |
| M7 | Add narrow typed request conveniences selected from real use cases | mixed; contract by **Codex**, mechanical SDK work delegable | blocked by M4 |
| M8 | Implement `extension dev` secure watch/build/reload loop | **Codex** core, delegable CLI help/docs slices | blocked by M1-M7 contracts |
| M9 | Publish reference plugins, bilingual author docs, and migration guide | **OpenCode + DeepSeek V4 Flash**, Codex review | blocked by M8 |
| M10 | Joined security, upgrade, multi-node, Browser, and release gate | **Codex** | blocked by M1-M9 |

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
