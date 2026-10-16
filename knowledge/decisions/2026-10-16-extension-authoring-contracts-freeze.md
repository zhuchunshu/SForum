# Extension Authoring Contracts: M0 Freeze

Status: **frozen**. All seven contract areas are written from traced production
code, and the two areas that were deliberately left blank in an earlier revision
(plugin session state, local dev grant) are now frozen in sections 5 and 6.

Two corrections were applied after independent verification, both recorded inline
in section 3: the custom-guard runtime is **live** rather than closed, and
`core.guard.raw_request` grants raw credentials with **zero** plugin guard code on
the path. The second contradicts a code comment that an earlier revision of this
document trusted; behaviour was re-verified directly and wins over the comment.

Scope: milestone M0 of
`knowledge/plans/2026-08-22-extension-authoring-speed-and-controlled-power.md`.
Corresponding inventory: `knowledge/sessions/2026-10-16-m0-threat-and-inventory-report.md`.

## Context

The task book assumed several contracts were open. Reading the code showed one of
them was already implemented (custom guards) and one was mis-modelled (what the
descriptor contribution points can carry). This record freezes what was verified
and refuses to freeze what was not.

## 1. `layout: site|bare` On Public Plugin Pages

Decision: add a `Layout` field to the page contribution with two values, `site`
and `bare`.

- **`site` is the default.** The Host guarantees exactly one navbar and one footer
  around the plugin body, using the shared public geometry.
- **`bare` is explicit opt-out**, for an author who owns their own chrome.

**REVISION (same day, after measuring).** This section originally froze **`bare`**
as the default, on the reasoning that contributions bind exact artifacts and that
defaulting to `site` would retroactively add chrome - risking a double navbar on a
template that already mounts chrome islands. Measuring the tree disproved the
risk and showed the original default was the wrong call:

- **0 of 5** plugin page templates in the repository mount `sf-navbar` or
  `sf-footer` (`extensions/fixtures/plugins/page-registry-demo/templates/*.html`,
  `.../sforum-plugin-page-business-e2e/templates/article.html`, and the override
  fixture theme). Control: **34 of 34** theme templates do.
- So no existing artifact can be double-chromed, and the real inconsistency was
  the opposite one: every plugin public page was the only page on the site with no
  chrome at all, while `AGENTS.md` requires public pages to share one geometry
  contract.
- The exact-artifact argument does not forbid the change either: **the shell is
  presentation**, and the platform already re-renders public-page presentation on
  theme activation without re-approving any plugin artifact. Treating
  presentation as artifact-bound was the mistaken premise.
- Two independent sources also point this way: this program's M2 exit gate ("a
  newly added plugin page inherits the active site shell by default") and
  `AGENTS.md`'s beginner-friendly-defaults rule (a safe working default without
  the author having to know the field exists).

Unknown values are still rejected at package preflight (`pages: invalid page
layout: "fancy"`), so the renderer only ever sees `site` or `bare`. The
controller's defensive mapping keeps an invalid value on the no-chrome side,
because a value that bypassed admission is an anomaly that should not be masked by
adding chrome.

Consequence to accept: "exactly one navbar and footer in every path" is a
guarantee for `layout: site` pages, not a universal invariant. `bare` pages
remain author-owned by definition. If a future milestone wants `site` to become
the norm, that is a migration with its own artifact and review, not a default flip.

## 2. Selected-Theme Shell, Body Outlet, And Host Fallback

Decision: reuse the **existing** shell machinery rather than introducing a second
shell or a second template engine (`AGENTS.md`).

What already exists and must be the foundation:

- `apps/web/app/components/SFPageOutletRender.vue` is the shared shell for
  registry pages. It selects `SFSystemThemeTemplate` (system pages),
  `SFThemeTemplate` (theme path, forwarding `regionWidgetRefs` into the single CSP
  aggregation), `SFHostPublicChrome` (fallback chrome), or a bare slot, and
  publishes the evidence attributes `data-page`, `data-provider`,
  `data-template`, `data-host-chrome`. It suppresses host chrome for `auth.*`,
  `system.*`, and `dev.components`.
- `apps/web/app/components/SFHostPublicChrome.vue` is the existing host chrome
  shell, with `data-layout="fullwidth-3col"` and navbar/announcement/footer.
- The body render source already has a reviewed candidate order:
  `ThemeRenderSourceActiveOverride` -> `ThemeRenderSourcePlugin` ->
  `ThemeRenderSourceActiveTheme` -> `ThemeRenderSourceDefaultTheme` ->
  `ThemeRenderSourceEmergency`
  (`apps/api/app/Support/Pages/theme_fallback.go:15`).

**The gap, precisely:** a plugin `action=add` page bypasses all of it. Its route
shell `apps/web/app/pages/[...sfRegistryPage].vue` renders the body inside a
literal `<main class="sf-public-page sf-registry-add mx-auto w-full max-w-4xl px-4 py-8">`
containing only `SFThemeTemplate` (or an alert/empty state). So chrome exists only
if the plugin template happens to mount it, and the geometry is a hardcoded
`max-w-4xl` rather than the shared public geometry contract.

Decisions:

1. Plugin `action=add` pages must render through the same shell path as other
   registry pages. The shell becomes a **second, independent axis** from the body
   render source: (body source) x (shell source). Today only the body axis exists,
   and the package has no `shell` concept at all (verified: no `shell` symbol in
   `app/Support/Pages/*.go`).
2. The shell source resolves to exactly one provider: the selected theme's shell
   artifact when present, otherwise the **Host fallback** (`SFHostPublicChrome`).
   "Exactly one navbar and footer" is enforced by there being exactly one shell,
   not by asking authors to be careful.
3. The plugin body is mounted through **one reviewed outlet** with a typed
   ViewModel boundary. The typed boundary already exists conceptually as the
   ViewModel contract carried by `ThemeRuntimeProviderBinding` (`pageId`,
   `contributionId`, `templateId`, `template`, `contractVersion`, `viewModelId`,
   `viewModelSchema`, `schemaDigest`, `themeOverrideKey`).
4. Theme override stays presentation-only. `compatibleThemeOverride`
   (`theme_fallback.go:264`) requires equal `ViewModelID`/`ViewModelSchema`, and a
   matching non-empty `ThemeOverrideKey` once a `SchemaDigest` is present. A theme
   may therefore replace body presentation without touching plugin data or access
   contracts; this gateway must not be relaxed by the shell work.
5. CSP ownership is unchanged. The theme template path aggregates exactly once in
   `SFThemeTemplate` (`SFThemeTemplate.vue:149,159`); the native path aggregates in
   `SFPageOutletResolver`. The shell must not add a second aggregation.
6. Special ownership is preserved: `auth.*`, `system.*`, and `dev.components` keep
   their current chrome suppression; virtual system error pages keep their
   existing owners.
7. Any new shell or layout artifact must be declared exactly like other
   artifacts - package file entry + digest, agreed across
   `theme.json.pages[].template`, the Manifest V3 `templates[]` declaration, and
   `packageFiles[]` - because that three-way fail-closed identity contract already
   exists for theme templates.

Exit conditions for M1 are the task book's, unchanged: focused Go tests, manifest
validation, both built-in themes' digest/validate/test, catalog generation check,
architecture gate, and exactly-one-chrome evidence in Browser for both the
success and fallback paths.

### 2.1 Shell ownership: two routes, and which to take first

Delivered so far: `layout: site|bare` (default `bare`) with the **Host fallback
shell** (`SFHostPublicChrome`), verified in Browser. The remaining question is how
the **selected theme** owns the shell. Two routes exist, and they are not
equivalent:

**Route A - a theme-authored shell template.** The theme ships a shell artifact
(declared with the existing three-way identity contract) containing chrome plus a
plugin-body island, and the Host composes the plugin body into it. This requires
real cross-runtime work: `ThemeRenderedPage` is
`HTMLSegments []string` + `Islands []themecompiler.IslandDescriptor`, so composing
means splicing a nested render output at the shell's island position, plus a new
allowed island tag in `allowedHostIslands`/`productionThemeIslandBindings`, plus
the web-side island binding, plus declarations and digests in both built-in
themes. It buys themes the ability to control the shell's **markup**.

**Route B - reuse the existing theme parameterization.** `SFHostPublicChrome`
already reads `useActiveThemeSettings()` for `layout.show_footer` and
`layout.show_announcements`, and the theme already declares its chrome islands in
`theme.json.navigationLocations` through
`themeNavigationLocationBindings` (`public.topbar.primary` -> `sf-navbar`,
`public.footer.primary` -> `sf-footer`, `public.mobile.primary` -> `sf-navbar`,
`public.sidebar.primary` -> `sf-home-navigation`). What is **not** theme-driven
today is the navbar's geometry: `SFHostPublicChrome` hardcodes
`<SFNavbar layout="fullwidth-3col" />`. Route B therefore means letting the host
shell honour the theme's declared navigation bindings and geometry setting. No new
artifact type, no composition, no new island.

Recommendation: **take Route B first.** It delivers the outcome the milestone is
actually for - the selected theme owns the chrome around a plugin body, with one
navbar and one footer - using contracts that already exist, and it is verifiable
without touching the compiler/runtime seam. Route A remains the right answer for a
theme that wants to own the shell's markup, and can be added later as a distinct
capability rather than as the only way to get any theme involvement.

This supersedes this document's earlier phrasing ("the selected theme's shell
artifact when present"), which assumed Route A as the sole mechanism. Recorded
rather than silently changed, because the choice changes M1's remaining scope
materially.

### 2.2 The shell does not pad the body, and must not start doing so

Recorded because it is a tempting and wrong "fix". A `layout: site` plugin page
whose own template has no container renders flush to the viewport edge, which
looks like a missing inset in the shell. It is not:

- The theme's per-page shell is `<div class="sf-page sf-theme--default">` +
  `<sf-navbar>` + `<div class="sf-theme-chrome-body" data-theme-region="body">` +
  the body island + `<sf-footer>` (`extensions/builtin/themes/sforum-default/templates/category-index.html`).
  `.sf-theme-chrome-body` carries only `flex: 1 0 auto; background:
  var(--sf-public-bg)` (`apps/web/app/assets/css/sforum-theme.css:671`).
- The horizontal inset lives in the **body component itself** (`.sforum-home`,
  `.sforum-topic-page`, …). The shell only *adjusts* those insets through the
  `:is(...)` lists that cover known core page classes, in both paths
  (`sforum-theme.css:796` for the theme shell, `:1048` for
  `.sf-host-public-chrome--fullwidth-3col .sf-host-public-chrome__body`).
- So "the body island owns its inner layout" is the existing contract, and the
  reference fixtures behave the same way
  (`extensions/fixtures/plugins/page-registry-demo/templates/members.html` is a
  bare `<main>`).

Consequence: the shell must **not** add a padding or container wrapper of its own
for plugin bodies. Doing so would introduce a second layout for public pages,
which `AGENTS.md` explicitly forbids, and would double-inset any plugin body that
does bring its own container. A plugin that wants the shared container geometry
declares it in its own template, exactly as the built-in theme's pages do.

## 3. Authorization Guard V1 Contract (frozen from traced code)

**Correction recorded here: the production binding already exists.** The task book
treated it as unimplemented. Traced chain:

- Manifest admission accepts `guards[]` with `kind` of `"custom"` or
  `"raw_request"` (`app/Support/ExtensionManifest/v3_validate.go:106`); the
  declaration type is `ManifestGuard` (`v3_registry_types.go:39`: id,
  contractVersion, kind, entry, digest, permissions).
- Production assembly injects the evaluator:
  `bootstrap/api_assembly_domains.go:618` ->
  `NewRuntimePluginRouteGuardEvaluator(lifecycleStack.RuntimeManager, extensionGuardPolicy)`.
- The authorizer dispatches to it:
  `app/Support/Routes/core_guard_authorizer.go:229`
  (`CoreGuardAuthorizer.authorizePluginGuard`), after validating the binding via
  `validPluginGuardBinding` (`:236`).
- The evaluator invokes plugin code:
  `app/Http/plugin_guard_runtime.go:32`
  (`RuntimePluginRouteGuardEvaluator.EvaluatePluginGuard`).

Frozen inputs:

| Input | Source |
| --- | --- |
| Plan identity | `plan.Revision()`, `plan.Method()`, `plan.Path()` |
| Step identity | `step.Guard`, `step.PluginGuard{ID, ContractVersion, Kind, Entry, Digest}`, `step.RequestSchema` |
| Request | `evaluation.Request.Query`, `.Body`, `.Authority{Mode, GuardKind}` |
| Exact artifact | `step.Provider.Artifact{ExtensionID, ExtensionVersion, PackageDigest, RuntimeInstanceID}` |

Frozen gates, all required: policy `Lookup` must be `Found` with non-zero
`Revision`; `SafeMode` must be off; entry must be a `plugin` and `enabled`; entry
version and package digest must equal the artifact's; when authority is raw or
`CurrentTrustRequired` is set, `CurrentArtifactTrusted` must be true. Then the
exact runtime instance must be `Active` with matching identity, version, and
artifact digest, and the call must acquire a `RuntimeCallGuard` lease.

Frozen failure policy: **fail-closed, unconditionally.**
`preInvocationPluginGuardFailure` (`plugin_guard_runtime.go:130`) never returns
nil - it maps to `PluginGuardFailureTimeout`, `Canceled`, or `Unavailable`.
`ErrRouteGuardUnavailable` (`app/Http/route_dispatcher.go:33`, returned at
`route_dispatcher.go:87,89,92` and
`app/Http/core_guard_authorizer.go:104,118,133,145`) covers the closed core guard
ids.

Two states that documentation currently collapses into one, and must stay
separate in the contract:

- `custom` - implemented, production-wired, trust-gated. `kind: custom` guard
  invocations receive `RequestAuthorityFiltered`, so credential headers are
  stripped.
- `kind: raw_request` - implemented and production-wired, and it **does** receive
  forwarded credentials (a production-chain test asserts the guard errors if they
  are absent).

**CORRECTION (independent verification, same day).** An earlier revision of this
section asserted that raw-session credential forwarding is "closed", citing the
comment at `app/Http/plugin_guard_runtime.go:53-54`. That was wrong: the comment
contradicts the code. A separate read-only trace caught it and the behaviour was
re-verified directly:

- `app/Http/plugin_guard_runtime.go:55-57` returns **nil (allow)** when
  `step.PluginGuard.ID == ""` - after only the policy/trust gate. So
  `core.guard.raw_request`, which by construction requires an **empty** binding
  (`app/Support/Routes/request_authority.go:189-191` demands
  `equalPluginGuardBinding(step.PluginGuard, PluginGuardBinding{})`), runs **no
  plugin guard code at all**.
- That same empty binding mints `RequestAuthorityRaw`
  (`request_authority.go:103-105`).
- Credential headers are skipped only when the authority is *not* raw
  (`app/Support/Extensions/protocol_v2_request_authority.go:129-131`:
  `if protocolV2RequestCredentialHeader(canonical) && authority.Mode != ProtocolV2RequestAuthorityRaw { continue }`).

Therefore `core.guard.raw_request` grants the plugin **handler** raw credential
authority (cookie / authorization / x-api-key / x-auth-token) with zero plugin
decision code on the path. This is recorded as an **open decision, not a frozen
contract**: the repository contains no decision record, no test pinning the allow
as intentional, and a comment asserting the opposite, so the intent cannot be
inferred from the code. It must be resolved by a human before M4 freezes the
guard contract.

Also corrected: an earlier revision cited `app/Http/route_dispatcher.go:33,87,89,92`
as production evidence that the core guard ids are closed. That file's
`HostRouteGuardAuthorizer` is referenced **only from `_test.go` files** and is
therefore unreachable from production. `ErrRouteGuardUnavailable` is real
production behaviour, but as the fail-closed mapping of non-denied guard failures
(`app/Http/core_guard_authorizer.go:133`) and of params-mismatch /
registry-unavailable (`:104,118,145`) - not as a "closed core guard id" path.

## 4. Guards Do Not Issue Or Manage Identity

Decision: guards may never issue, renew, rotate, or revoke authentication state,
and this is already true of the implementation.

Evidence: `app/Http/plugin_guard_runtime.go` and
`app/Http/core_guard_authorizer.go` contain **zero** session-issuing or
cookie-setting calls (searched for issue/renew/rotate/set-cookie: no matches).
A guard receives a *resolved* authority (`ResolvedRequestAuthority{Mode,
GuardKind}`) and returns only an error. Authentication remains entirely owned by
AuthSession; a guard is an authorization decision point on an already-identified
actor.

The inverse also holds: "unavailable" is not "deny". A guard that cannot run
fails closed with `Unavailable`, and the contract must not let a plugin turn a
runtime failure into an allow.

## 5. Plugin Session State

Frozen from the traced AuthSession lifecycle. The decisive finding is that the
**obvious** design - add plugin keys to the session payload - violates an existing
invariant. The contract below is shaped by that.

### 5.1 What exists today

- One Redis key per session, key = the raw session id, **no prefix**, DB 0
  (`apps/api/bootstrap/api_assembly_infrastructure.go:214`, storage
  `gofiber/storage/redis/v3`, `app/Support/Redis/session.go:41`).
  This is the **only unprefixed key** in the app; every other key follows
  `sforum:<domain>:…`.
- Payload is gob `map[any]any` (`middleware/session/session.go:586,608`), keys:
  `user_id`, `created_at`, `renewed_at`, `login_ip`, `login_user_agent`,
  `token_version`, `sid`, `last_seen_touched` (`AuthSession/manager.go:28-37`).
  New types must be `RegisterType`d and keys must be comparable.
- **AuthSession is the only writer** (no other `sess.Set`/`sess.Get` in the repo).
- TTL is the whole-key `IdleTimeout`, and `Save()` runs on **every** request
  (`AuthSession/renewal.go:110`), not only on the 24h rotation. Absolute lifetime
  is enforced by `absExpirationKey` in the payload, which `Reset()`s the whole
  session (`middleware/session/store.go:66-69`).
- `Regenerate()` (rotation) deletes the old Redis key, mints a new session id, and
  **preserves the in-memory payload** (`renewal.go:104`); `sid` stays stable.
- Revocation is **not** a Redis delete: it is a per-request PostgreSQL check
  (`Models/Identity/sessions.go:42 IsSessionRevoked`, hot path at
  `AuthSession/manager.go:346-351`). `EnforceMaxSessions`
  (`sessions.go:218`, default 5, clamped 1..20) marks the *other* sids revoked
  while their Redis keys live on.
- `clearSessionData` (`AuthSession/pending.go:245-252`) deletes **all** payload
  keys unconditionally on issue/renew failure.
- Three derived values that must never be conflated or exposed as credentials:
  `sid` (32B hex, not a credential), `session_hash` =
  HMAC-SHA256(sessionID, `SESSION_HASH_SECRET`) (`manager.go:430`), and
  `SessionFingerprint` = SHA-256(sid) (`external_auth_store.go:32`).
- Privacy has `Register`/`EraseUser` hooks (`Support/Privacy/hooks.go:201,343`)
  with **no session-state consumer** today.
- The wire protocol declares `Actor.session_id`
  (`contracts/proto/sforum/protocol/v2/common.proto:22`) but the production path
  **never populates it**, and Host API v2 has **no** session-state service at all.

### 5.2 Decisions

1. **Plugin session state must NOT live in the session payload.** Adding a key
   there has two unacceptable consequences, both proven above: it makes every
   request reset the whole-key TTL (so plugin state would *extend the Core session
   lifetime*, violating an existing invariant), and `clearSessionData` would
   silently wipe plugin state on unrelated issue/renew failures.
2. It must be a **separate Redis key** under the established namespace, i.e.
   `sforum:<plugin-scoped-domain>:…`, never the unprefixed session key. The
   existing Host-derived-scope pattern to reuse for the key shape is
   `HostAPI/cache_registry_service.go:254,335,430-440`
   (`sforum:host-cache:v1:installation:<hash>:contract:<hash>:segment:<hash>:…`)
   with its isolation digest over actor/permission/locale plus the exact artifact
   (`Support/CacheRegistry/validation.go:252`).
3. Because rotation preserves the payload but **replaces the Redis key**, plugin
   state must be keyed independently of the session id, or explicitly migrated on
   rotation. Keying directly by `sid` is not sufficient, because a session can be
   revoked while its key survives (5.1 above).
4. It needs its **own invalidation path**. Revocation, device-cap eviction, and
   logout do not delete Redis state today, so M6 must add an explicit hook rather
   than assume piggybacking.
5. It must be covered by the privacy erase path: the plugin state needs a
   consumer registered with `Support/Privacy` so user erasure reaches it.
6. Limits must be announced by the Host, not chosen by the plugin. Adopt the
   existing cache-registry ceilings as the starting envelope (key ≤512 bytes,
   value ≤1 MiB, ≤64 tags, TTL ≤24h) rather than inventing new numbers.
7. `Actor.session_id` must stay **unpopulated** until this contract lands. Exposing
   a session identifier to plugins before the state contract exists would hand out
   a correlation handle with no lifecycle guarantees.

### 5.3 Remaining open

The precise key derivation, the invalidation hook shape, and whether TTL should be
bounded by the session's idle timeout are **not** frozen here - they are M6's
design work under the constraints above. What is frozen is that the payload is not
the place, the namespace is, invalidation and erasure are requirements, and the
limits come from the Host.

## 6. Local Dev Grant And Reload Safety

Frozen from the traced CLI and dev-loop inventory.

### 6.1 What exists today

- Real `sforum` subcommands: `version`, `make:plugin`/`make:theme`,
  `seed:forum`, `seed:perf`, `users:reset-password`, `revisions backfill`,
  `dev:cleanup-orphan-plugins`, and `extension` with `validate`, `build`,
  `digest`, `test`, `package`, `docs generate`, `list`, `disable`,
  `disable-all`, `quarantine`, `command list`, `command run`, `api-lts`,
  `system-tier …`.
- **No `extension dev`**, and no `extension discovery` / `trust` / `enable` /
  `restart` CLI entries. Those exist only as HTTP admin routes
  (`app/Http/Controllers/Extensions/routes.go:48,49`).
- **No file-watcher dependency exists**: `fsnotify` is absent from every
  `go.mod`/`go.sum`. `air` is an externally installed global tool, is not a module
  dependency, watches only `app|bootstrap|cmd|config|database`, and therefore does
  **not** watch `extensions/`. `scripts/build-builtin-plugins.sh` rebuilds a
  **hardcoded list of 8** built-in ids by hand.
- The only existing environment-gated dev-only mechanism is the orphan-plugin
  reaper: `Support/DevHygiene/reaper.go:30 ShouldEnableDevelopmentOrphanReaper`
  (enabled only when `APP_ENV=development`, wired at
  `bootstrap/api_assembly_http.go:229`, with tests asserting a no-op in
  production and test).
- There is **no dev-only trust bypass**: `ExecutableTrustService` has no `APP_ENV`
  branch and is always exact-artifact + `super_admin` + 5-minute challenge.
- Admin/public plugin bundles are served from immutable digest paths
  (`/_sforum/assets/extensions/<id>/<digest>/<path>`), so any change requires a new
  digest and a fresh trust confirmation. There is no dev side channel.
- Dead code found while tracing: `newExtensionRecoveryStaleBuiltinCommand`
  (`recovery_stale_builtin.go:24`) is referenced nowhere, is registered under no
  parent command, and is not covered by the docs gate's file list. It has neither a
  runtime entry nor a gate.

### 6.2 Decisions

1. **Reuse the established env gate, do not invent a trust model.** A dev loop
   must be gated exactly like `DevHygiene/reaper.go`: `APP_ENV=development` only,
   with a bootstrap-time refusal in production and a test asserting the
   production/test no-op. That is the one pattern this repo already trusts.
2. **No trust bypass.** A dev loop may shorten the *loop* (watch, rebuild,
   re-digest), but it may not weaken exact-artifact trust: the digest still
   changes, and the operator still confirms. Any convenience must be an
   automation of the existing CLI steps, not a hole in them.
3. A watcher is a **new module dependency** and therefore a decision, not an
   implementation detail: `fsnotify` must be added to `apps/api/go.mod` explicitly
   (today it is only an indirect dependency of the external `air` binary).
4. The dev loop must not vendor a second build pipeline. It orchestrates the
   existing `extension build`, `digest --write`, `validate`, `test` sequence.
5. The hardcoded 8-id built-in list in `scripts/build-builtin-plugins.sh` must
   stop being duplicated in shell. The dev loop should discover built-ins from the
   manifest tree, which also removes the documented drift between the script, the
   README, and the Dockerfile's backend list.
6. `extensions/dev/` is a scaffold output directory and is never scanned; the dev
   loop must therefore publish through the normal staging path rather than relying
   on a dev-only extension root.

### 6.3 Remaining open

The precise watch-set, debounce/coalescing policy, and whether the loop restarts
the API are M8's design work. What is frozen: the gate pattern, the no-bypass
rule, the explicit dependency decision, the reuse of existing CLI steps, and the
removal of the hardcoded id list.


## 7. Versioning, LTS, And Rollback

Frozen from traced code:

- Every page and contribution carries a versioned contract string
  (`PageContribution.Contract`, e.g. `sforum.page.outbound@1`), and page↔contract
  pairing is asserted by the PageViewModels registry
  (`coreViewModel("site.outbound", "sforum.page.outbound@1", …)`).
- Runtime rendering carries `RenderOutput.NodeRevision`, and the response exposes
  `selectedVersion`, `selectedPackageDigest`, `selectedRuntimeInstanceId`,
  `templatePath`.
- The baseline template path is instrumented as an LTS shim:
  `apilts.Process().RecordShimCall(apilts.ThemeRequestTimeLoaderContractID)`
  (`app/Http/Controllers/Pages/controller.go`, template branch). This is how the
  LTS baseline is measured rather than assumed.
- Rollback is a re-binding operation on `ProviderBinding`
  (extensionId + contributionId + version + packageDigest + contractVersion +
  approvedBy), not an in-place mutation of a running artifact.
- Responses on this path are `Cache-Control: private, no-store` with
  `Vary: Cookie, Authorization, Accept-Language`
  (`setPrivatePageResponse`), so per-actor page state is never shared-cached.

Policy decision: adding a contract-capable field (such as `layout`) must be
additive and must not change rendering for artifacts that do not use it, per
section 1.

## Consequences

- M1 may proceed on the shell contract: it is fully traced and does not depend on
  the two open sections.
- M4 must be re-scoped from "implement" to "verify, correct the stale disclosure,
  and close allowed/denied coverage" - the runtime exists.
- M6 and M8 stay blocked on their sections. This record deliberately does not
  guess them.
- The stale docs (`docs/extensions/routes.md:283,284,399`,
  `docs/extensions/authoring-guide.md:741`) understate a live, request-path,
  high-risk power and must be corrected as part of M4, not silently.

## Revision 1 (2026-10-16): human decisions on the six open items

The operator resolved every open item this program had accumulated. Two of them are
**contract changes** and therefore belong in this record; the other four settle
acceptance wording, scope, or naming and do not change any runtime contract.

### R1. `core.guard.raw_request` credential forwarding is a vulnerability - tighten it

Decision: **the current behaviour is a hole, not a design.** An empty-binding
`core.guard.raw_request` currently allows with zero plugin guard code and mints
`RequestAuthorityRaw`, under which `cookie`, `authorization`, `x-api-key` and
`x-auth-token` are forwarded to the plugin handler
(`app/Support/Extensions/protocol_v2_request_authority.go:129-131`), while the comment
directly above claims credential forwarding stays closed.

Consequence for the contract: raw request authority must no longer be granted
implicitly. The implementation obligation is to make credential forwarding
**explicit and declared** (a package that wants it must say so, and the Host must
disclose it at trust time), and to keep the failing side covered by tests. The
comment and every doc that repeats it must be corrected in the same change. This
supersedes the "intent unknown" wording in section 3 of this record: the intent is now
declared by the operator, so the trace stands and the behaviour changes.

### R2. Plugin page data gets a protocol method (Loader Gateway transport)

Decision: **extend Protocol V2 with a data-fetch method** rather than exposing a
loopback HTTP base to plugin runtimes or dropping the capability. This unblocks the
Loader Gateway for `dataSource: plugin` pages, M10 row 2, and the `view: schema`
form-submit path.

Consequence for the contract: the protocol surface grows by one request/response pair
with the same exact-artifact, trust, and Safe-Mode gates as every other runtime call;
the capability stays **closed** until that method exists, so no page may claim it
earlier. The existing pinned test
(`app/Support/Extensions/IntegrationTests/page_loader_transport_integration_test.go`)
describes the gap that this decision closes and must be updated together with the
implementation, not deleted ahead of it.

### R3. Per-request plugin-guard denials are not audited (confirmed)

Decision: **keep the current behaviour.** `PluginGuardFailureDenied`
(`app/Http/plugin_guard_runtime.go:110-125`) writes no audit row: the audit ledger is
an admin-facing, low-volume record and the request path must not gain a durable write.
This closes the M5 judgement call as a decision rather than a gap.

### R4. The `challenge` clause leaves M10 row 4

Decision: **remove the clause from the acceptance row** instead of changing the guard
contract. Guard outcomes are allow or `denied|crash|timeout|protocol|canceled|unavailable`
(`app/Http/plugin_guard_runtime.go:32-113`); no `challenge` outcome exists and adding one
is not wanted.

### R5. M10 row 1 validates the existing built-in theme

Decision: **reword the row to "the existing protected built-in theme"**. The repository
contains exactly one (`extensions/builtin/themes/sforum-default`); adding a second is not
part of this program.

### R6. `view: schema` declaration naming is confirmed

Decision: **keep `schema.descriptor` (the `kind: schema` package file holding the bounded
descriptor) and `schema.contract` (the exact `<schemaFileId>@<version>` reference the
descriptor must satisfy)**, as implemented by the M9 `--schema-page` scaffold and the
shared preflight. The adjacent plan's "Interpretation of the frozen contract" section is
therefore confirmed rather than merely assumed.

## Revision 2 (2026-10-16): the plugin session-state contract is authorised — M6 decisions

**Status: authorised by the operator, not yet implemented.** Section 5 fixed the
*constraints* of plugin session state and its §5.3 deliberately left four things open as
"M6's design work". The operator has now authorised adding the contract, and this
revision decides those four open items so M6 can start. **No production code changed**;
nothing here is observable at runtime yet, and every decision below is individually
vetoable by the operator before implementation begins.

This is a **new extension ABI**, so it is recorded as a contract change rather than
slipped into an implementation commit — that is the rule the plan set and the reason
M6/row 5/M9's last item were held instead of guessed.

### 2.1 Verified inputs (measured, not recalled)

| Fact | Where |
| --- | --- |
| Session Redis key is the **raw session id with no prefix**, DB 0 — the only unprefixed key in the app | `bootstrap/api_assembly_infrastructure.go:214`, `Support/Redis/session.go:41` |
| **Login issues a fresh `sid`** (`generateSID()` per issue; the previous `sid` is captured as `previousSID`) | `Support/AuthSession/manager.go:198,222` |
| **Only `Regenerate()`** (24 h rotation) changes the key while preserving the in-memory payload, and `sid` is stable across it | `Support/AuthSession/renewal.go:104` |
| Revocation is a **per-request PostgreSQL check**, not a Redis delete; `EnforceMaxSessions` revokes other sids **while their Redis keys live on** | `Models/Identity/sessions.go:42,218` |
| Namespace pattern to mirror | `sforum:host-cache:v1:installation:<id>:…` — `Support/HostAPI/cache_registry_service.go:39,254` |
| The envelope §5.2(6) cites **is real** | `Support/HostAPI/cache_registry_service.go:27-31` (`HostCacheMaximumKeyBytes=512`, `HostCacheMaximumValueBytes=1<<20`, `HostCacheMaximumTTL=24h`) and `CacheRegistry/validation.go:22` (`maxTagsPerCache=64`) |
| Privacy consumers register with an export **and** an erase hook | `Support/Privacy/hooks.go:201,343` |
| Protocol V2 `Actor.session_id` is declared but a populated value is **rejected** | `sdk/plugin/v2/identity_runtime.go:235` |

### 2.2 Decisions (the four open §5.3 items, plus what §5.2 requires of them)

**S1 — Key derivation.** One Redis key per `(installation, extension, session, namespace key)`
under the established prefixed namespace:

```text
sforum:plugin-session:v1:installation:<installationID>
  :extension:<sha256hex(extensionID)>
  :session:<sha256hex(sid)>
  :key:<sha256hex(namespaceKey)>
```

Every variable part is hashed, so no raw extension id, session id, or plugin key ever
appears in a key name and each component's length is bounded by construction. The session
component uses the existing `SessionFingerprint` notion (`sha256(sid)`) rather than the raw
id: it is stable across rotation (S4) and it is not a credential — the raw sid is never
stored outside the Core session key.

**S2 — Value shape: opaque bytes, bounded, loss-tolerant.** V1 stores the plugin's own
bytes (≤ 1 MiB) with **no Host-side schema** and **no migration guarantee**: the contract
states that this state is *ephemeral workflow state*, a plugin must tolerate a miss, and a
different value shape is the plugin's own versioning concern (it may put a version byte in
its payload). Rationale: a Host-typed document would force a new manifest declaration plus
a digest contract for every small workflow, which is exactly the authoring overhead this
program exists to remove; the §5.2(7)-style guarantees plugins actually need (isolation,
bounds, invalidation, erasure) are unaffected by this choice.

**S3 — TTL: the effective TTL is the minimum of four bounds**, so plugin state can never
outlive — let alone extend — the Core session:

```text
effective = min(requested, 24h /* S6 */, remaining idle lifetime, remaining absolute lifetime)
```

A missing/zero `requested` means "session-scoped" (idle bound). Absolute expiry is enforced
host-side at read time as well, not only by Redis TTL, because Redis TTL is the mechanism
and not the authority.

**S4 — Invalidation hook, with per-session and per-extension indexes.** Redis has no
efficient "delete by prefix", and a revoked sid's key survives (verified above), so the
contract adds two bounded index sets and one narrow Host-internal interface:

- `sforum:plugin-session:v1:installation:<id>:session:<sha256(sid)>:keys` — a SET of that
  session's namespace-key hashes.
- `sforum:plugin-session:v1:installation:<id>:extension:<sha256(extensionID)>:sessions` —
  a SET of session fingerprints holding state for that extension.
- `InvalidateSession(ctx, sidFingerprint)` and `InvalidateExtension(ctx, extensionID)`
  (both idempotent, both bounded by S6's caps so neither scan is unbounded).

Call sites M6 must wire, each with a test: **logout/destroy**, **revoke** (including
`EnforceMaxSessions` eviction), **the fresh-`sid` issue on login** (invalidate the captured
`previousSID`'s namespace — this is what makes "no silent transfer of stale plugin state"
true rather than assumed), **absolute expiry**, **disable/uninstall**, and **privacy erase**.

**S5 — Lifecycle policy matrix** (each row is a required M6 test):

| Transition | Policy |
| --- | --- |
| Anonymous | Supported, same derivation. Anonymous state becomes unreachable on login (S4 invalidates the previous sid), so it can never be promoted into an authenticated session. |
| Authenticated → anonymous (logout) | Session invalidated; state dropped. |
| Rotation (24 h) | Key is `sid`-derived and `sid` is stable ⇒ state survives; **no migration step exists**, so there is nothing to get wrong. |
| Replacement (new `sid` from a new login) | No transfer; the old namespace is invalidated. |
| Revoke / device-cap eviction | Invalidated explicitly by S4 (do not rely on the Redis key disappearing). |
| Expiration | Own TTL per S3; read returns a miss, never stale bytes. |
| Disable / uninstall of the extension | `InvalidateExtension`; uninstall additionally drops the index sets. |
| Privacy erase | A Privacy contribution (`Register(contribution, exportHook, eraseHook)`) whose erase hook walks the erased user's sessions and calls `InvalidateSession` for each. |
| Safe Mode | Fail closed: reads miss, writes are refused. Safe Mode must never be a window in which plugin state is readable for a quarantined extension. |

**S6 — Limits are Host-owned and announced, not plugin-chosen:** namespace key ≤ 512 B
(before hashing), value ≤ 1 MiB, TTL ∈ [1 ms, 24 h] — the verified §5.2(6) envelope —
plus **≤ 64 keys per (extension, session)** and **≤ 4096 keys per extension installation**,
and the transport's own request-size cap. Exceeding a cap is a typed error, never silent
truncation.

**S7 — Wire surface and authority derivation.** A Host-owned surface
(`get` / `set` / `delete`) whose request carries **only** the namespace key, the value, and
the TTL. The extension identity, the runtime instance, and the session all come from the
Host-attested runtime context; a caller-supplied extension id, user id, or session id is
not merely ignored — it does not exist in the request shape, so it cannot be smuggled in.
Cross-plugin and cross-session access are therefore denied by construction, and M6 must
still prove both with tests.

**S8 — `Actor.session_id` stays unpopulated.** S7 deliberately does not need it, and handing
plugins a correlation handle for which they have no contract remains forbidden
(§5.2(7), unchanged by this revision).

### 2.3 What this authorises, and what it does not

- **Authorised:** M6 may now be implemented against S1–S8, and M9's last item (the
  plugin session-state workflow reference package) becomes buildable against a real
  contract.
- **Not authorised and not claimed:** nothing is implemented, so *no* session-state
  behaviour can be reported as working, and M10 row 5 stays **open** — its blocker changes
  from "no contract exists, needs a human decision" to "contract authorised; implementation
  is M6". Row 5 closes only when M6 lands with its allowed/denied, cross-plugin,
  cross-session, rotation, logout, revoke, expiry, privacy and concurrency tests green.
- **Explicitly vetoable:** each of S1–S8 is a separate decision. If the operator disagrees
  with any one of them (most likely S2's opaque-bytes choice or S6's caps), it can be
  changed before implementation without touching anything else, because no code depends on
  this revision yet.

### S9. The session component is delivered as a Host-signed, single-invocation delegation

**Discovery (measured, and it corrects the "derived from the runtime context" phrasing in S7).** The
plugin→Host call seam attests the **runtime**, not the request: the host-v2 gRPC server is built
per runtime binding (`protocolV2HostBinding{identity, authority, tokenHash}` in
`app/Support/Extensions/protocol_v2_host.go:34-58`) and the interceptor publishes only the runtime
identity into the handler context
(`hostapi.ContextWithProtocolV2RuntimeIdentity`, `protocol_v2_host.go:70`). Plugin runtimes are long-lived
processes, so that channel is **runtime-scoped**: from it alone the Host cannot know which HTTP request —
and therefore which session — a `Set` belongs to. S7 said the session is derived from the
"Host-attested runtime context"; that is true of the *extension identity* and false of the *session*.

**Decision.** The session scope travels the same way Protocol V2 already carries exactly this kind of
authority: a Host-signed, **single-invocation delegation** in the inbound `RequestContext`, mirroring
`HostCommandDelegation` / `HostQueryDelegation` — whose own comments read "a short-lived Host-signed
capability delivered to one exact plugin invocation … Plugins cannot construct or substitute this
token" (`contracts/proto/sforum/protocol/v2/common.proto:70-97`). Concretely:

- the Host mints the delegation only where the Host itself knows the session, i.e. while dispatching a
  plugin invocation that it initiated inside an HTTP request;
- the plugin echoes it on `Get` / `Set` / `Delete`;
- the Host validates signature, binding (actor + runtime + artifact + expiry / one use) and then derives
  the session fingerprint from **its own** record of what it minted.

The plugin therefore still never sees or supplies a session identifier, and `Actor.session_id` stays
unpopulated — S8 is unchanged.

**Consequence: this is a wire addition** (a delegation message plus a `RequestContext` field) with
Host-side minting and validation. Until that exists, the following are **deliberately withheld**, because
shipping them would be precisely the "declared-but-unusable capability" this program forbids (the M4
cancellation set the precedent):

- `PluginSessionService` is **not registered** in `app/Support/HostAPI/v2.go`, so the RPC answers
  `Unimplemented`; the generated types exist as the first slice only;
- it is **not** added to the Host API v2 documentation gate's proto list nor to
  `docs/extensions/host-api-v2.md` — documenting a service the Host does not serve would be a false claim;
- no plugin-facing SDK helper is shipped.

**Implemented and green at the time of writing** (each item verified by the command named with it):
S1–S3 and S6 in a new `app/Support/PluginSessionState` package (`go test … -race` green, including cap,
TTL, isolation, expiry-on-read, invalidation-idempotency and backend-failure tests); the S4/S5 erasure
hooks for **logout** and **login replacement** (`Support/AuthSession`, 5 new tests green);
**revoke / device-cap eviction / password reset** (`Models/Identity`, with `RETURNING sid` so the Host
knows exactly which sessions ended; existing suite green); and **extension disable / uninstall**
(`Models/Extensions`, post-commit best-effort).

**Still to do for M6:** the S9 delegation (mint + validate), the service adapter and its registration,
the privacy-erase consumer, the plugin SDK helper, the reference fixture (also M9's last item), and the
joined acceptance tests. **Nothing session-related may be reported as working until those land**, and
M10 row 5 stays open.

## Revision 3 (2026-10-16): S9 implemented — three clarifications the implementation forced

S9 is now **implemented and served** (see
`sessions/2026-10-16-m6-s9-session-delegation-handoff.md` for the evidence). Three
points S9 left implicit were decided while building it, and one S5 row turned out
not to be satisfiable on today's session layer. Recorded here because each one is
observable at runtime.

### 3.1 The session fingerprint never enters the token, only the Host's mint record

S9 said the Host "derives the session fingerprint from its own record of what it
minted". Implemented literally: the signed token carries the exact runtime binding
(extension id/version, artifact digest, trust grant, runtime epoch, instance id)
and the actor user id — **not** the session fingerprint, and not the session's
deadlines. Those live only in the in-memory mint ledger. A plugin that base64-decodes
its own token therefore learns nothing about the session, which is what keeps S8
(`Actor.session_id` stays unpopulated) honest rather than cosmetic. Test:
`TestProtocolV2SessionDelegationTokenCarriesNoSessionIdentifier`.

### 3.2 "One use" is enforced as *single invocation*, not single RPC

S9's heading says "single-invocation delegation" while its body says "one use". Taken
literally as one RPC per token, a workflow could not even read-modify-write its own
key, so the contract would be unusable. Implemented as: the Host mints the delegation
where it initiates an invocation, the plugin may perform the several operations of
that invocation, and the transport **retires** the token when the invocation settles
(`defer` in the unary, streaming and admin-surface transports). Reuse after
settlement fails closed (`ErrProtocolV2SessionDelegationRetired`), which is the
property the acceptance criteria actually test. The 2-minute TTL remains a second
bound for a stream that never settles.

### 3.3 A streaming invocation keeps its delegation until the stream settles

The stream transport retires the delegation in the same `closeOnce` block that
cancels the child deadline, so a plugin streaming for minutes keeps session access
for exactly as long as its invocation lives — not longer.

### 3.4 Correction: S5's "Anonymous | Supported" row is not satisfiable today

S5 lists anonymous sessions as supported with "the same derivation". Reading the
session layer shows why that cannot hold as written: Core mounts **no global
`session` middleware** (only `Support/AuthSession` calls `Save`, on issue and
renewal), so a brand-new anonymous session is never persisted — its cookie session
id is discarded at the end of the request. Keying plugin state by that id would
leave bytes that no later request can ever reach.

Decision: `PluginSessionScope` returns **no scope** for an unpersisted anonymous
session (`sess.Fresh()` with no `sid`), so the plugin receives
`ErrPluginSessionDelegationUnavailable` — the same ordinary outcome as a background
call. The code path for a *persisted* anonymous session (cookie present, no
`user_id`) is kept, because it becomes reachable the moment anonymous sessions are
persisted; today nothing persists them. Consequence, stated plainly: **anonymous
plugin session state is unavailable**, and M10 row 5's anonymous clause stays open.
Test: `TestPluginSessionScopeRefusesUnpersistedAnonymousSession`.

Also measured while implementing this: Fiber's absolute-expiry key is set when a
session is first loaded and is **wiped by the `Reset()` that login performs**, so a
logged-in session carries no stored absolute boundary at all. `PluginSessionScope`
therefore anchors the absolute bound at `created_at + AbsoluteTimeout` (never at
"now + AbsoluteTimeout", which would extend past the session) and prefers the stored
value whenever it is present. Test:
`TestPluginSessionScopeAnchorsAbsoluteBoundaryAtCreation`.

### 3.5 The session-state service is served by the API process

`PluginSessionService` is bound on the API's Host API gateway before any plugin
broker registers, together with the runtime admission source and the boot-time Safe
Mode flag. A standalone worker is unsupported in this repository (the API always
embeds the worker) and has no session scope to mint from, so it is not wired; a
plugin runtime started there would see `Unimplemented`, which is the honest state.

## Revision 4 (2026-10-17): row-5 closeout decisions and one production correction

Recorded with the M10 matrix Revision 4 and the 2026-10-17 closeout handoff.

### 4.1 D1 — anonymous session state stays unavailable (operator decision)

The operator decided **not** to implement anonymous session persistence. S5's
anonymous row is therefore closed as "unavailable by construction": Core mounts
no global session middleware, anonymous sessions are never persisted, and
`PluginSessionScope` returns no scope for them (`ErrPluginSessionDelegationUnavailable`,
ADR Revision 3 §3.4 unchanged). Reopening this requires a session-persistence
decision (per-visitor cookie + Redis state), which is a new ADR, not a plugin
contract change.

### 4.2 D2 — the privacy-erase entry is a super_admin CLI (operator decision)

`privacy.Registry.EraseUser` gets its first production entry: the CLI command
`privacy:erase-user` (`apps/api/cmd/sforum/privacy_erase_user.go`). The
permission gate is the production RBAC check (actor must be a real
`super_admin` / `user.manage` account, format `user:<id>`), actor and target are
written to the privacy audit ledger, the report carries extension ids, success
and external-retention warnings only (no values, no namespace keys), and the
Host-owned plugin-session-state contribution is registered through the same
`bootstrap` path as the API assembly (Redis unreachable = named fail-closed
error). An admin UI remains a separate, undecided product question.

### 4.3 Correction — the production Gateway was missing the S9 issuer

Revision 3 described the mint points as production-wired. That was true only in
tests, which inject the delegation authority into the client config directly.
Production passes the bare `*hostapi.Gateway` to `ProtocolStarter`, and the S9
issuer is resolved by type assertion on that registrar — which the Gateway did
not implement. Every Host-initiated invocation carrying a session scope failed
at mint. Fixed: `Gateway.MintProtocolV2SessionDelegation` /
`RetireProtocolV2SessionDelegation` delegate to the bound session service's
authority (unbound = named fail-closed; Safe Mode does not block minting — the
boundary stays on the three operations). The compiled-fixture joined test
(`TestPluginSessionStateJoinedHostDispatchToStore`) runs the exact production
shape (Gateway as registrar), so this cannot silently regress. Lesson recorded:
a capability "implemented and served" must be verified through the assembly
path, not only through a test that wires its pieces directly.
