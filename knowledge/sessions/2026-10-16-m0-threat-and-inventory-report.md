# M0 Threat And Inventory Report - Extension Authoring Speed And Controlled Power

Scope: milestone M0 of `knowledge/plans/2026-08-22-extension-authoring-speed-and-controlled-power.md`.
This report names real production paths and open residuals. It changes no production
code. Claims are backed by file/symbol references gathered by reading the code, not
by assuming the plan's premises.

Status of this report: **complete for M0**. All four areas were traced from
production code. The decisions that follow from them live in
`knowledge/decisions/2026-10-16-extension-authoring-contracts-freeze.md`.

Two corrections were made after independent verification, and they matter more
than the milestone: (1) the custom-guard runtime is **live**, not closed, and
(2) `core.guard.raw_request` grants raw credentials with **zero** guard code on the
path - an earlier revision of this report trusted a code comment that states the
opposite of the behaviour. See sections 2.2b and 2.3.

## 1. Public Plugin Page Chain (M1-critical) - COMPLETE

### 1.1 Production call chain

Traced end to end. Every step below exists in production code.

1. **Declaration.** A plugin/theme page contribution is a `PageContribution`
   (`apps/api/app/Support/Pages/registry.go:36`) carrying `ID`, `Action`
   (`add`|`replace`), `Target`, `Path`, `Template`, `Contract`, `Access`,
   `Permission`, `DataSource` (`core`|`plugin`), `DataRoute`, `DataSchema`,
   `ExtensionID`, `Version`, `PackageDigest`, `RuntimeInstanceID`,
   `RouteSignature`.
2. **Route admission.** `GET /pages/resolve-path` is registered at
   `apps/api/app/Http/Controllers/Pages/controller.go:107`, handler
   `Controller.resolvePath` (`controller.go:680`).
3. **Reserved-path guard.** `pages.IsReservedPath(path)` -> 404
   (`controller.go:687`; reserved list in `app/Support/Pages/reserved.go`).
4. **Path resolution.** `registry.ResolveAddedPathMatch(path)` ->
   match with `Contribution` + `Params` (`controller.go:693`). Matching is
   implemented in `app/Support/Pages/route_matcher.go`.
5. **Access authority (fail closed).** `enforcePageAccess(c, contrib.Access,
   contrib.Permission)` (`controller.go:700`; `enforcePageAccess` at
   `controller.go:~834`). Unknown access values are rejected; the web shell only
   *reacts* to 401/403/404, it does not decide.
6. **Provider/artifact binding.** The response carries `Provider`,
   `ExtensionID`, `ContributionID`, `SelectedVersion`,
   `SelectedPackageDigest`, `SelectedRuntimeInstanceID`, `TemplatePath`,
   `Contract` (`controller.go:713-737`). The admin binding record is
   `ProviderBinding` (`registry.go:57`): pageId + extensionId + contributionId +
   version + packageDigest + contractVersion + approvedBy + templatePath.
7. **Runtime snapshot lookup.** `runtime.Resolve(RuntimeArtifact{ExtensionID,
   ExtensionVersion, PackageDigest, RuntimeInstanceID}, contributionID,
   contributionID)` (`controller.go:749`;
   `app/Support/Pages/theme_runtime.go:714`). `RuntimeArtifact` is defined at
   `registry.go:~70`. `runtime.Claims(...)` distinguishes "artifact mismatch"
   (`controller.go:783`) from "not covered".
8. **Loader Gateway + plugin render.** When the snapshot declares a
   `PluginDataContract` (`theme_runtime.go:443`), the Loader Gateway runs
   `loader.LoadExactForContribution(...)` and the snapshot renders via
   `snapshot.RenderPluginData(...)` (`theme_runtime.go:505`), producing
   `TemplateHTML` + `RenderOutput` (`controller.go:760-777`). A runtime-applied
   render is the only path that sets `LoaderData`.
9. **Theme-baseline template render.** When no snapshot covers the contribution
   and `contrib.Template != ""`, the template is loaded from the contribution's
   own package content root and rendered with route params as variables:
   `pages.LoadTemplate(root, contrib.Template)` + `pages.RenderTemplate(html,
   vars)` (`controller.go:790-808`; `app/Support/Pages/template.go`). This path
   is instrumented through `apilts.Process().RecordShimCall(...)` as an LTS
   baseline shim.
10. **Fallback.** `setRespFallback(reason)` (`controller.go:738`) sets
    `Fallback = true` plus a reason. Reasons observed: view-model unavailable,
    render failed, artifact mismatch, runtime unavailable. Additionally
    `resp.Fallback` is forced when `TemplateHTML == ""` or the render output
    source is `ThemeRenderSourceEmergency` (`controller.go:773`).
11. **Response envelope.** `{page, provider, extensionId, contributionId,
    action, fallback, reason, templateHtml, renderOutput, nodeRevision,
    dataSource, dataRoute, loaderData, loaderError, selectedVersion,
    selectedPackageDigest, selectedRuntimeInstanceId, templatePath, routeParams,
    contract}`; response headers forced by `setPrivatePageResponse`
    (`controller.go:830`): `Cache-Control: private, no-store` and
    `Vary: Cookie, Authorization, Accept-Language`.
12. **SSR catch-all.** `apps/web/app/pages/[...sfRegistryPage].vue` calls
    `/pages/resolve-path` via `useAsyncData` and maps 401 -> `/login`,
    403 -> 403 error, 404 -> 404 error.
13. **Body render.** The same file renders the plugin body inside
    `<main class="sf-public-page sf-registry-add mx-auto w-full max-w-4xl px-4 py-8">`
    containing only `SFThemeTemplate` (or an alert/empty state).
14. **Theme override of presentation.**
    `compatibleThemeOverride(plugin, override)`
    (`app/Support/Pages/theme_fallback.go:264`) requires equal `ViewModelID` and
    `ViewModelSchema`, and, when `SchemaDigest != ""`, a matching non-empty
    `ThemeOverrideKey`. Candidate assembly is `themeRenderPlan.addCandidate(...)`
    (`theme_fallback.go:239`) with `emergencyThemeOutput(...)`
    (`theme_fallback.go:134`) as the emergency output.

### 1.2 Chrome ownership: three render paths, one of them broken

`apps/web/app/layouts/default.vue` is a **pass-through** (`<slot />`). Its comment
states the contract explicitly: public chrome is owned by theme L1
(`sf-navbar`/`sf-footer`) or by the `SFPageOutlet` fail-closed host shell. Nuxt's
layout deliberately does not mount chrome, to avoid double navigation.

That leaves three ownership paths:

| Path | Owner | Evidence |
| --- | --- | --- |
| Theme L1 template | the theme's own `templates/<page>.html` mounts `<sf-navbar>` / `<sf-footer>` | `extensions/builtin/themes/sforum-default/templates/*.html` (e.g. `category-index.html:3,7`) |
| Registry/core fallback | `SFHostPublicChrome` | `apps/web/app/components/SFPageOutletRender.vue:64`; component defines `data-layout="fullwidth-3col"` |
| Plugin `action=add` page | **none** | `apps/web/app/pages/[...sfRegistryPage].vue` renders a bare `<main class="… max-w-4xl …">` |

`SFPageOutletRender.vue` is the existing shared shell for registry pages. It
selects among `SFSystemThemeTemplate` (system pages), `SFThemeTemplate` (theme
path, and it forwards `regionWidgetRefs` into the single CSP aggregation),
`SFHostPublicChrome` (fallback chrome), or a bare slot, and it publishes
`data-page`, `data-provider`, `data-template`, `data-host-chrome` for evidence.
`useHostPublicChrome` is false for `auth.*`, `system.*` and `dev.components`.

**Finding (the M1 gap, precisely):** plugin `action=add` pages bypass
`SFPageOutletRender` entirely. Therefore:

- A plugin page has chrome only if the plugin author remembered to write
  `<sf-navbar></sf-navbar>` and `<sf-footer></sf-footer>` into their own template
  file. There is no shell that guarantees it.
- A plugin page uses a hardcoded `max-w-4xl px-4 py-8` geometry instead of the
  shared public geometry contract, so it does not match either the theme path or
  `SFHostPublicChrome`'s `fullwidth-3col`.
- There is no `layout: site|bare` selector on `PageContribution`
  (`registry.go:36`). The only `layout` field in the extension manifest today is
  the admin settings UI layout (`form`|`tabs`) in
  `app/Support/ExtensionManifest/settings_document.go:58`. So "site shell" is
  not expressible today and a backward-compatible default must be defined.

### 1.3 CSP and SEO ownership on the plugin page path

- CSP: `SFThemeTemplate.vue` imports `applyPublicPageDocumentPolicy`
  (`SFThemeTemplate.vue:9`) and performs exactly one aggregation, merging region
  widget refs with theme island refs (`SFThemeTemplate.vue:149,159`). The region
  path aggregates in `SFPageOutletResolver`; the theme path aggregates in
  `SFThemeTemplate`. The plugin page reuses `SFThemeTemplate`, so it inherits the
  theme-path aggregation.
- SEO: the plugin page hardcodes `useSForumSeo({ title: () => page.id || path,
  type: 'website', noindex: true })` (`[...sfRegistryPage].vue:74-78`). Plugin
  pages are unconditionally `noindex` today.

### 1.4 Exact-artifact and trust inputs on this path

Trust inputs observed on the page path: `ExtensionID`, `Version`,
`PackageDigest`, `RuntimeInstanceID` (`RuntimeArtifact`), `ProviderBinding`
(approvedBy + contractVersion + templatePath), and contribution
`Contract`. The response is `private, no-store` and varies on
Cookie/Authorization/Accept-Language. Access is decided only by
`enforcePageAccess`.

### 1.5 Reserved routes

`pages.IsReservedPath` is enforced before resolution
(`controller.go:687`); the list lives in `app/Support/Pages/reserved.go`. The
report does not restate the list; the source of truth is that file.

### 1.6 Current tests vs missing tests (page chain)

Present and relevant:

- `app/Support/Pages/route_matcher_test.go`, `route_matcher_decode_test.go` -
  path matching and decoding.
- `app/Support/Pages/theme_runtime_test.go`,
  `theme_runtime_hot_path_test.go`, `theme_runtime_inspection_test.go` -
  runtime snapshot resolution and hot path.
- `app/Support/Pages/theme_package_test.go`,
  `builtin_theme_completeness_test.go` - theme package identity and completeness.
- `app/Support/Pages/plugin_lifecycle_test.go`,
  `registry_lifecycle_publication_test.go`,
  `plugin_theme_override_fixture_test.go` - lifecycle publication and override
  fixtures.
- `app/Support/Pages/provider_snapshot_test.go`,
  `registry_provider_management.go` - provider binding management.
- `apps/web/tests/framework/pageRegions.test.ts` - region render-site contract
  (both directions), including the generated matrix.
- `apps/web/tests/themes/*` - theme template and CSP aggregation contracts.

Missing (in scope for M1's exit gate, named here so they are not re-derived):

- No test asserts **exactly one navbar and one footer** on the plugin
  `action=add` path in its success and fallback paths.
- No test asserts the plugin page path shares the public geometry contract.
- No test covers `layout: site|bare` because the field does not exist yet.
- No test asserts plugin-page chrome survives a theme switch, a missing shell
  artifact, digest drift, disable, rollback, or Safe Mode.
- No paired-surface completeness test proving a plugin page and its sibling
  (e.g. index/detail) both resolve through the same shell.

### 1.7 Source-of-truth owners

| Concern | Owner |
| --- | --- |
| Contribution shape and resolution | `apps/api/app/Support/Pages/registry.go` |
| Core page catalog | `apps/api/app/Support/Pages/catalog.go` |
| Reserved paths | `apps/api/app/Support/Pages/reserved.go` |
| Path matching | `apps/api/app/Support/Pages/route_matcher.go` |
| Loader Gateway | `apps/api/app/Support/Pages/loader_gateway.go`, `loader.go` |
| Template load/render (baseline) | `apps/api/app/Support/Pages/template.go` |
| Active theme runtime, override compatibility | `apps/api/app/Support/Pages/theme_runtime.go`, `theme_fallback.go` |
| Page resolve transport | `apps/api/app/Http/Controllers/Pages/controller.go` |
| Shared web shell | `apps/web/app/components/SFPageOutletRender.vue`, `SFPageOutlet.vue` |
| Host chrome shell | `apps/web/app/components/SFHostPublicChrome.vue` |
| Theme template render + island map + CSP | `apps/web/app/components/SFThemeTemplate.vue` |
| Plugin page route shell | `apps/web/app/pages/[...sfRegistryPage].vue` |

## 2. Custom Guards: Documentation Contradicts Live Production Code - CRITICAL

### 2.1 What the documentation claims

- `docs/extensions/routes.md:283` - `core.guard.raw_request`: "**Closed in the
  current host release** (`ErrRouteGuardUnavailable`)."
- `docs/extensions/routes.md:284` - custom guard ids: "the executable guard
  runtime is a later wave; **closed today**."
- `docs/extensions/routes.md:399` - "do not rely on `inherit`, `raw_request`, or
  custom guards in the current [release]".
- `docs/extensions/authoring-guide.md:741` - "Guards | `core.guard.public` /
  `core.guard.login` only (custom/raw guard runtime is a later wave)".

### 2.2 What production actually does

The custom-guard runtime is **implemented, wired into production assembly, and
trust-gated**. The chain is live end to end:

1. **Admission accepts it.** `app/Support/ExtensionManifest/v3_validate.go:97-116`
   validates `guards[]` and explicitly permits `kind` of `"custom"` or
   `"raw_request"` (`v3_validate.go:106`), tracking raw-request guards in a map.
   The declaration type is `ManifestGuard` (`v3_registry_types.go:39`): id,
   contractVersion, kind, entry, digest, permissions.
2. **Production injects a real evaluator.**
   `bootstrap/api_assembly_domains.go:618` constructs
   `PluginGuards: httpserver.NewRuntimePluginRouteGuardEvaluator(lifecycleStack.RuntimeManager, extensionGuardPolicy)`.
   This is the same typed-policy wrapper described at
   `app/Http/core_guard_authorizer.go:83,93`
   (`ProductionRouteGuardAuthorizer`).
3. **The authorizer dispatches to it.**
   `app/Support/Routes/core_guard_authorizer.go:215`
   (`CoreGuardAuthorizer.authorizePluginGuard`) validates the binding
   (id/contractVersion/kind/entry/digest via `validPluginGuardBinding`, `:235`)
   and then **calls `a.PluginGuards.EvaluatePluginGuard(...)`** (`:229`).
   It only returns `ErrCoreGuardEvaluatorUnavailable` when `PluginGuards` is nil
   or the binding is invalid - and in production `PluginGuards` is not nil.
4. **The evaluator really runs plugin code with gates.**
   `app/Http/plugin_guard_runtime.go:32`
   (`RuntimePluginRouteGuardEvaluator.EvaluatePluginGuard`) requires: a live
   `policy.Lookup` with non-zero revision, `Found`, not `SafeMode`, matching
   extension id/type (`plugin`)/status (`enabled`)/version/package digest, and
   for raw authority or `CurrentTrustRequired`, `CurrentArtifactTrusted`; then it
   inspects the exact runtime instance (`Active`, matching identity/version/digest)
   and acquires a `RuntimeCallGuard` lease before invoking.
5. **`ErrRouteGuardUnavailable` is real production behaviour, but not as a
   "closed core guard id" path.** It is defined at
   `app/Http/route_dispatcher.go:33` and is the fail-closed mapping of non-denied
   guard failures (`app/Http/core_guard_authorizer.go:133`) and of
   params-mismatch / registry-unavailable (`:104,118,145`).
   **Correction:** an earlier revision of this report cited
   `route_dispatcher.go:87,89,92` as production evidence that the core guard ids
   are closed. That was wrong. Those returns live in `HostRouteGuardAuthorizer`,
   which is referenced **only from `_test.go` files** and is unreachable from
   production. The tests asserting "raw closed" therefore exercise a
   configuration production never builds.

### 2.2b R3 - `core.guard.raw_request` grants raw credentials with zero guard code

Found by an independent read-only trace and re-verified directly. This is the most
security-significant fact on this path:

- `app/Http/plugin_guard_runtime.go:55-57` returns **nil (allow)** when
  `step.PluginGuard.ID == ""`, after only the policy/trust gate.
- `core.guard.raw_request` **requires** exactly that empty binding
  (`app/Support/Routes/request_authority.go:189-191` demands
  `equalPluginGuardBinding(step.PluginGuard, PluginGuardBinding{})`), so it runs no
  plugin guard code at all.
- The same binding mints `RequestAuthorityRaw` (`request_authority.go:103-105`).
- Credential headers are stripped only when the authority is not raw
  (`app/Support/Extensions/protocol_v2_request_authority.go:129-131`), so under
  raw authority the plugin **handler** receives cookie / authorization /
  x-api-key / x-auth-token.

The comment immediately above the short-circuit
(`plugin_guard_runtime.go:53-54`, "Credential forwarding remains closed until
raw-session authority is frozen") states the **opposite** of the behaviour. An
earlier revision of this report trusted that comment; the behaviour wins.

Intent is **not determinable from the repository**: there is no decision record,
no test pinning the allow as intentional, and the adjacent comment contradicts it.
This is recorded as an open decision requiring a human, not as a frozen contract.

### 2.3 Reconciliation

The task book's M0 task 2 assumed the risk was "custom-guard code/tests exist
while the docs mark the runtime closed, and Support-only tests are not production
evidence". The direction is the opposite and more serious:

- The tests are **not** Support-only fiction. `plugin_guard_runtime_test.go`
  exercises a type that production assembly actually constructs.
- The **documentation is stale in the unsafe direction**: it tells authors and
  operators that a high-risk power (plugin-authored authorization code running on
  the request path) is unavailable, while it is implemented and reachable behind
  trust gates. It also frames `core.guard.raw_request` as needing "a separately
  trusted raw guard runtime" (R11), when that path needs no guard runtime at all.

That is a documentation/honesty defect with security-disclosure impact, not merely
a coverage gap. It must be corrected as part of M4, and M4 must be re-scoped from
"implement guard V1" to "freeze and verify the existing V1 contract, correct the
disclosure, and close allowed/denied coverage".

Treat `custom` and `kind: raw_request` as two different states, and see 2.2b for
what `core.guard.raw_request` actually does. The docs collapse all three.

### 2.4 Open for the ADR (guard section)

- **R3 intent** (see 2.2b) - deliberate, or a hole? Not inferable; needs a human.
- The exact allow/deny outcome mapping to an HTTP status, and whether
  `authorizeInherited` behaves as the docs claim for `inherit`.
- Whether any additional gate (trust confirmation flow, inspector, audit) is
  missing relative to the frozen V1 contract.

Additional residuals from the same independent trace, to be addressed in M4/M5 and
not re-derived: R5 there is no `challenge` outcome (contract is 204 / 401-403 /
protocol-failure, and 401 vs 403 collapse client-side) although M4 task 5 asks for
allow/deny/challenge; R6 guard `permissions[]` are validated and disclosed but
never enforced; R7 no guard id, denial category, failure kind, or quarantine event
in trace/incident, and no guard-scoped audit action (M5 tasks 1/4/5 need new
fields); R8 the Host hard-codes a 3s guard timeout while the 16-concurrent limit is
declared only to the plugin at handshake, and the admission gate has no guard cap
or circuit breaker (M4 task 7 has no Host-side implementation); R9
`RequiresExecutableTrust` is false for non-uploaded sources, so a built-in plugin
declaring a guard would run with no grant obligation; R10 the task book's sample
guard JSON omits `digest` and the permission-declaration prerequisite that the
implemented schema requires; R12 the recovery-only first-refresh window (guard
routes 502 until recovery clears) is undocumented.


## 2b. AuthSession And Plugin Session State (traced)

### 2b.1 Lifecycle, as it exists

Single production assembly: `bootstrap/api_assembly_infrastructure.go:214`
(`session.NewStore` over `gofiber/storage/redis/v3`, cookie `sforum_session`) and
`:235` (`authsession.NewManager` with `RenewalEffectGate`). Defaults:
`SESSION_IDLE_TIMEOUT` 30d, `SESSION_ABSOLUTE_TIMEOUT` 180d,
`SESSION_RENEWAL_INTERVAL` 24h (`config/config.go:130,131,208`). Fiber's session
middleware is **not** installed, so AuthSession drives `Save()` directly.

- Issue: `Manager.Start`/`Begin`/`BeginWithContext`/`BeginWithAuthorityVersion`
  (`AuthSession/manager.go:148-181`); `Pending` is single-use
  (`pending.go:65`) and directory-write failure rolls the credential back
  (`abortIssue:113`).
- Renew/rotate: `renewal.go:14 refresh`, once per request (`manager.go:359-362`),
  rotation via `sess.Regenerate()` (`renewal.go:104`) which deletes the old Redis
  key, mints a new id, and **keeps the in-memory payload**.
- `Save()` runs on **every** request (`renewal.go:110`), so the whole-key TTL
  slides continuously.
- Logout: `Manager.Destroy` (`manager.go:390`); failure path scrubs **all** payload
  keys (`pending.go:245-252`).
- Revocation is a per-request PostgreSQL check (`Identity/sessions.go:42`, hot path
  `manager.go:346-351`), not a Redis delete. `EnforceMaxSessions`
  (`sessions.go:218`, default 5, clamp 1..20) revokes other sids while their Redis
  keys survive.
- Token version: payload key vs `GetUserTokenVersion` (`manager.go:328-342`).
- Recent auth: `user_recent_auth` keyed `(user_id, session_fingerprint)` with
  `SessionFingerprint = sha256(sid)` (`external_auth_store.go:32`), 5-minute TTL,
  enforced at unlink / password change / external link. **No cleanup job exists**,
  unlike sessions (`Jobs/Identity/cleanup_sessions.go:45`).
- External-auth fencing: `SessionPolicyEvaluator.RequireAllowAndRun` for issue and
  `sessionPolicyRenewalGate.Evaluate` for renew
  (`bootstrap/session_policy_evaluation.go:32`); `ValidateLoginEffect`
  (`external_auth_service.go:285`) re-checks live artifact/activation/SafeMode.
  Host-local revocation bypasses the renewal gate for exactly four paths
  (`bootstrap/session_policy_route_actor.go:28`).

### 2b.2 Constraints a plugin session-state feature must satisfy

These are the findings that shaped the frozen contract, and each one rules out an
obvious design:

1. The payload is written **only** by AuthSession, and `clearSessionData` removes
   every key unconditionally -> plugin state in the payload would be wiped by
   unrelated issue/renew failures.
2. Any payload key resets the whole-key TTL on every request -> plugin state in the
   payload would **extend the Core session lifetime**, violating an existing
   invariant.
3. `Regenerate` preserves the payload but replaces the Redis key -> keying plugin
   state by `sid` is not sufficient.
4. A session can be revoked while its Redis key survives -> plugin state needs its
   own invalidation hook, which **does not exist today**.
5. Payload serialization is gob `map[any]any`; unknown types fail to decode, so new
   types must be `RegisterType`d and keys must be comparable.
6. Three derived values must not be conflated: `sid` (not a credential),
   `session_hash` = HMAC-SHA256(sessionID, `SESSION_HASH_SECRET`)
   (`manager.go:430`), and `SessionFingerprint` = SHA-256(sid).
7. Privacy has `Register`/`EraseUser` hooks (`Support/Privacy/hooks.go:201,343`)
   with **no** session-state consumer; there is no user-deletion endpoint.
8. Role/permission changes are resolved per request and do **not** rotate or revoke
   sessions.
9. The session key is the **only** unprefixed Redis key; every other key is
   `sforum:<domain>:…`.
10. `Actor.session_id` is declared in the wire protocol
    (`contracts/proto/sforum/protocol/v2/common.proto:22`) but **never populated**
    in production, and Host API v2 has **no** session-state service at all.

### 2b.3 Tests

`AuthSession/` has 51 tests across 5 files, **all in-package and all on the
in-memory store**. `Models/Identity` adds session-service, policy-evaluation,
step-up, and gate suites, with real-PostgreSQL integration tests for policy
selection/effect/race - but the session store remains in-memory even there.
`scripts/test-identity-session-policy-lifecycle.sh` pins 22+8 test names.

Gaps: no test uses the real Redis session store (zero hits for the redis storage
package in any `*_test.go`); absolute-timeout behaviour has no Go test;
`expireSessionTransport`/`scrubSessionCredentialWithin` are only exercised on the
in-memory store; `user_recent_auth` has no cleanup/concurrency test.

### 2b.4 Doc-vs-code

`knowledge/modules/identity.md:667-672` describes "idle TTL refresh" without the
two facts that matter for M6: the TTL reset is **whole-key**, and `Save()` happens
on **every** request, not only at the 24h rotation.

### 2b.5 Could not determine

Whether other domains share the session Redis database: `REDIS_ADDR` is single and
no database index is configured. If shared, `session.Store.Reset` is a `FlushDB`
(`gofiber/storage/redis redis.go:136-143`) affecting the whole DB - currently no
production `Reset` call site exists. Also unverified: any out-of-repo cleanup of
`user_recent_auth`.

## 3. Extension CLI And Dev Loop (traced)

### 3.1 Real command inventory

Root: `version`, `make:plugin`/`make:theme`, `seed:forum`, `seed:perf`,
`users:reset-password` (alias `user:reset-password`), `revisions backfill`,
`extension`, `dev:cleanup-orphan-plugins` (`command.go:40-50`, `seed.go`,
`reset_password.go:23`, `revisions.go:20`, `validate.go:17`,
`orphan_plugins.go:9`).

`extension` subcommands: `validate` (`validate.go:38`), `build`
(`build_extension.go:24`), `digest` (`manifest_digest.go:22`), `test`
(`test_extension.go:13`), `package` (`package_extension.go:21`),
`docs generate` (`docs.go:12`), `list`/`disable`/`disable-all`/`quarantine`
(`recovery.go:65,98,119,143`), `command list`/`command run`
(`plugin_command.go:59,71`), `api-lts` (`api_lts.go:13`), `system-tier …`
(`system_tier.go:25,46,73`).

**Dead code:** `newExtensionRecoveryStaleBuiltinCommand`
(`recovery_stale_builtin.go:24`) is referenced nowhere, is added to no parent
command, and is absent from the docs gate's file list
(`tests/validate-docs.mjs:282`). Neither a runtime entry nor a gate covers it.

**Does not exist:** `extension dev`, `extension discovery`, `extension trust`,
`extension enable/restart/activate`. Those capabilities live only in HTTP admin
routes (`Controllers/Extensions/routes.go:48,49,53,57`) or in runtime services
(`CatalogService.SyncExternalSources`, `ExecutableTrustService`).

### 3.2 Dev loop, as it exists

- Go API hot reload: `scripts/api-dev.sh` + `apps/api/.air.toml`. `air` is an
  **externally installed global tool** (`api-dev.sh:19`), is in **no** `go.mod`,
  and its `include_dir` does **not** cover `extensions/`.
- `scripts/build-builtin-plugins.sh` rsyncs to `storage/builtin-dev` and builds a
  **hardcoded list of 8** built-in ids, then runs `digest --write` + `test` inside
  staging and writes `air.env`.
- One-shot author pipeline: `extension build` = `bun install --frozen-lockfile &&
  bun run build` -> `digest --write` -> `validate` -> `test`
  (`build_extension.go:52-78`).
- Startup scanning: `SyncBuiltins` (`host_inventory_adapter.go:208`),
  `SyncExternalSources` (`:92`); immutable snapshot layout
  `<EXTENSION_ROOT>/<id>/<version>/<digest>/` (`Support/ExtensionPackage/snapshot.go:101,128,268`).
- The **only** existing environment-gated dev-only behaviour is the orphan-plugin
  reaper: `DevHygiene/reaper.go:30 ShouldEnableDevelopmentOrphanReaper`, enabled
  only under `APP_ENV=development`, wired at `bootstrap/api_assembly_http.go:229`,
  with `bootstrap/orphan_plugin_reaper_test.go` asserting a production/test no-op.

**Does not exist:** `extension dev`; **any file-watcher dependency**
(`fsnotify` has zero hits across all `go.mod`/`go.sum`); a dev-only trust bypass
(`ExecutableTrustService` has no `APP_ENV` branch); admin-frontend bundle hot
reload (assets are served from immutable digest paths,
`Models/Extensions/public_frontend.go:495`); automatic plugin-backend rebuild
beyond the hardcoded 8; debounce/coalescing; a CLI rollback to the last healthy
artifact; SIGINT/subprocess cleanup; API-restart orchestration; any dev
extension-root setting (`extensions/dev/` is a scaffold output directory and is
**never** scanned).

### 3.3 Tests and gates

Real-path CLI tests: `test_extension_test.go:41,96` (real `go build` of plugin
backends plus real digest/validate/test), `build_extension_test.go` (executed via
`newRootCommand()`, bun replaced by a shell stub so the real Bun never runs),
`package_extension_test.go`, `docs_test.go`, `recovery_test.go` +
`recovery_integration_test.go` (real PostgreSQL), `orphan_plugins_test.go`,
`api_lts_test.go`, `generator_test.go`.

Gaps: no `extension dev` test; **no CLI command-tree completeness test** - the only
completeness gate is the JS `tests/validate-docs.mjs` with hardcoded mappings, which
is why the dead `recovery_stale_builtin.go` went unnoticed;
`TestDockerBuildsProtectedBuiltinBackendsAndValidatesV3Digest`
(`test_extension_test.go:143`) requires only 7 backends while
`apps/api/Dockerfile:13-33` builds 8, so the gate lags the implementation.

### 3.4 Doc-vs-code conflicts

1. `docs/extensions/build-and-load.md:254-255`,
   `docs/zh-CN/extensions/build-and-load.md:241`, and
   `docs/extensions/authoring-guide.md:90` claim `worker-dev.sh` stages built-in
   plugins; it only prints an error and `exit 1`. `docs/en-US/development/workflow.md:21`
   still lists a "Standalone worker", contradicting `setup.md:43`.
2. `extensions/README.md:86-97` lists 5 built-ins "today" while the script builds 8.
3. `extensions/README.md:139` still refers to a standalone `worker` service.

### 3.5 Could not determine

Cobra runtime behaviour for hidden commands/aliases (the inventory comes from
static `Use:`/`AddCommand` parsing); whether `air` restarts on
`storage/builtin-dev/air.env` changes; and a field-level audit of
`docs/{zh-CN,en-US}/development/cli.md` against the implementation.

## 4. Reconciliation With Open V3 Residuals (M0 task 3)

`knowledge/plans/2026-07-13-trusted-plugin-theme-platform-v3-progress.md` carries
the open V3 residuals:

| V3 | Status | Relationship to this program |
| --- | --- | --- |
| M3 | runtime rollout before active publication; real multi-node gate open | **Upstream.** This program must not implement a second publication path. M1's "selected-theme shell provider" publishes through the same active-theme publication, so V3-M3 owns the rollout-ordering fix. |
| M5 | Marketplace/Privacy production consumers open | Independent. Marketplace/commerce does not gate authoring. |
| M6 | CompatFarm RPC failure fixed; matrix/run consolidation open | Independent. |
| M7 | Commerce workflow via Dispatcher open | Independent. |
| M8 | "local gates and overview QA green; release image and Page Registry smoke pending" | **Downstream consumer of this program.** The Page Registry smoke that V3-M8 still owes is exactly the surface M1/M2 build. This program should supply the shell + Browser evidence; V3-M8 keeps ownership of the release-image gate. Do not duplicate a second ledger. |

Ownership decision recorded here: **V3 owns distribution/release gating; this
program owns the authoring/runtime contract.** The only shared artifact is Page
Registry Browser evidence, where this program produces it and V3-M8 consumes it.

## 5. Unresolved Questions That Can Still Move A Contract

1. **R3 intent** (section 2.2b): is the zero-code raw-credential allow deliberate
   or a hole? Not inferable - no decision record, no test, and the adjacent comment
   asserts the opposite. Needs a human.
2. The intended `challenge` outcome for plugin guards: M4 task 5 asks for
   allow/deny/challenge, but no type, field, status, or proto message exists.
3. Whether guard `permissions[]` were meant to be Host-enforced: validated and
   disclosed, never enforced, no ADR statement.
4. Whether `HostRouteGuardAuthorizer` is a deliberately retained legacy test double
   or orphaned dead code: only test references exist.
5. Whether other domains share the session Redis database (section 2b.5).

The M1 shell schema is **not** blocked by any of these: its inputs are fully traced
in section 1 and frozen in the decision record.
