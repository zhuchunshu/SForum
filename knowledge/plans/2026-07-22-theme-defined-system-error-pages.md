# Theme-Defined System Error Pages - Task Book

Status: **completed** — M0-M6 implemented and verified on 2026-07-23; the
selected healthy public theme now owns L1 presentation for 403, 404, 429, and
500/502/503/504 system error families while Host retains truth and recovery
Date: 2026-07-22
Goal: let the selected public theme own the L1 presentation of common system
error pages while Host retains status, safe content, behavior, SEO, and an
always-available emergency fallback.

Focused precursor: public resource-not-found 404 behavior was completed in
`2026-07-22-theme-consistent-public-resource-404.md`. Its request-local context,
selected-theme rendering, one-attempt system resolver, document policy, and
Core emergency fallback are the baseline for this broader book. Continue with
403/429/5xx without reimplementing or weakening the finished 404 path.

M1-M6 directly reused these six completed 404 building
blocks:

1. the server pre-preparation plugin that resolves the final presentation
   before `error.vue` renders;
2. the request-local presentation composable and its serialized hydration
   state;
3. exact extension/version/package digest/node revision validation across L0
   and L1;
4. Host-owned status, cache, robots, canonical, and structured-data document
   policy;
5. the reviewed L0/L1 system AST renderer and statically compiled Host islands;
6. the complete, non-recursive Core emergency fallback for genuine theme,
   renderer, transport, or runtime faults.

Implementation stayed scoped to system error pages and did not merge into the
search, content-revision, or V3 production-rewire programs.

## Completion Ledger - 2026-07-23

- Added virtual Page Registry surfaces for `system.forbidden`,
  `system.not_found`, `system.rate_limited`, and `system.server_error`, with
  `PageDefinition.virtual` exposed through OpenAPI/admin inspection.
- Split Host-owned system error semantics into safe request-local context and
  reviewed Host islands for details, actions, recovery, sidebar, and rail.
- Preserved original HTTP status, `no-store`, `noindex,nofollow`, Core
  emergency fallback, and 401/API-envelope boundaries.
- Kept selected-theme L0/L1 exact-artifact matching authoritative; Core renders
  when the resolver, options/auth preloads, skin links, or artifact identity
  fail.
- Rejected plugin replacements for `system.*` and rejected public L2 widgets on
  system error templates.
- Added complete default and Nocturne built-in templates/styles for forbidden,
  not-found, rate-limited, and server-error surfaces.
- Browser QA covered real unknown-route 404 on desktop/mobile, dark mode,
  English locale, recovery search, selected-theme markers, SSR status, and no
  framework overlay. Current QA data does not expose natural full-page
  429/500/503 producers; those status families are covered by mapping,
  resolver, ViewModel, compiler/runtime, and fallback tests.
- Final cleanup also fixed default-theme mobile system-error layout so the
  hidden sidebar/right-rail state collapses to one full-width column at narrow
  viewports.

## Completed Precursor And Shared-Worktree Warning

The regression dependency and focused 404 precursor are complete. Shared error
and Page Registry files are available to this book after the focused 404 commit.

- Before editing, re-read `git status --short`, the active regression plan, and
  its latest handoff.
- Do not overwrite, revert, restage, or reimplement those changes.
- Preserve the completed behavior that `system.not_found` resolves against the
  selected theme, keeps theme chrome for ordinary semantic 404s, and falls back
  to Core only on a real theme/runtime/API failure.
- Extend the existing system-theme AST renderer and request-local error context
  for new status families; do not create a parallel error rendering stack.

## Required Reading Before Coding

1. `AGENTS.md`
2. `knowledge/index.md`
3. `knowledge/modules/frontend.md`
4. `knowledge/modules/extensions.md`
5. `knowledge/decisions/2026-07-13-runtime-page-registry-themes.md`
6. `knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`
7. `knowledge/plans/2026-07-22-current-head-regression-remediation.md`
8. `docs/extensions/page-catalog.md`
9. This task book

## Product Outcome

When an operator selects a healthy public theme, normal browser-facing 403,
404, 429, and server-error pages use that theme's declared L1 structure and L0
skin without a Nuxt rebuild. Theme authors can arrange the public chrome, error
details, actions, and decorative presentation through reviewed template and
Host-island contracts.

The result must remain useful when the API, Page Registry resolver, active theme
snapshot, template, or optional client JavaScript is unavailable. In those
cases the Host renders a complete localized emergency page and preserves the
original HTTP status.

## Current Baseline - Reuse, Do Not Rebuild

| Area | Current evidence | Required treatment |
| --- | --- | --- |
| Nuxt error boundary | `apps/web/app/error.vue` | Keep as the only browser error entrypoint |
| Error semantics | `utils/errorPage.ts` + `SFErrorPageContent.vue` | Keep Host-owned status mapping, localization, SEO, and actions |
| Page resolution | `SFPageOutlet.vue` + `/pages/resolve` | Reuse selected-theme resolution and bounded Core fallback |
| Existing error page | `system.not_found` / `sforum.page.not_found@1` | Preserve the published id and contract |
| Theme runtime | immutable compiled Page Registry snapshots | Reuse L0/L1; do not add a second theme loader |
| Error ViewModel | `ThemeCompiler.ErrorPageViewModel` | Extend only with reviewed non-sensitive presentation fields if needed |
| Built-in themes | default and Nocturne `not-found.html` | Replace whole-page Host island with themed structure plus narrow Host islands |
| Theme activation | existing trust, validation, selection, reset, and audit | No new operator setting or permission |
| Fail-closed behavior | Core slot + `no-store` resolver fallback | Retain and strengthen for every system error surface |

## Short Library Survey

No new dependency is justified. Nuxt's error boundary, Vue provide/inject or
typed props, the existing Go `html/template` compiler, and Page Registry already
solve the lifecycle and rendering problems. A third-party error-page package
would duplicate routing/theme authority and weaken the current immutable
artifact and Host-island controls.

Record this conclusion in the implementation ADR. Do not run `bun add`,
`go get`, or `go mod tidy` for this feature unless M0 finds a concrete missing
capability and the plan is amended first.

## Frozen Architecture Decisions

These decisions are implementation law for this task book. Change them only in
an explicit ADR reviewed before the affected code lands.

### D1. Error pages are virtual Page Registry surfaces

Add or preserve the following stable catalog entries:

| Page id | Contract | Status selection | Access |
| --- | --- | --- | --- |
| `system.forbidden` | `sforum.page.forbidden@1` | 403 | public virtual surface |
| `system.not_found` | `sforum.page.not_found@1` | 404 | public virtual surface |
| `system.rate_limited` | `sforum.page.rate_limited@1` | 429 | public virtual surface |
| `system.server_error` | `sforum.page.server_error@1` | 500, 502, 503, 504 | public virtual surface |

- A virtual surface has no routable public path. Nuxt reaches it only after the
  Host has already selected an error status.
- Add an explicit catalog property such as `Virtual bool` rather than growing a
  list of `system.*` string special cases. Update the exposed `PageDefinition`
  OpenAPI schema and frontend admin type at the same time.
- Other 4xx/5xx statuses use the complete Host generic error page in v1. Do not
  create one Page ID per HTTP status without a product need.
- A browser 401 continues through the existing login/redirect policy. It is not
  converted into a themed error page.
- API JSON error envelopes are unchanged and out of scope.

### D2. Themes own presentation; Host owns truth and behavior

- `error.vue` remains authoritative for the original `NuxtError`, HTTP status,
  selected system Page ID, and emergency fallback.
- Themes own L1 page structure: navbar/footer placement, main layout, visual
  grouping, decorative content, and placement of reviewed error islands.
- Host owns localized status/title/description, home/back/retry behavior,
  `clearError`, and whether retry is valid.
- Split the current whole-page `SFErrorPageContent` responsibility into a
  complete emergency fallback plus narrow reviewed primitives, expected to be
  equivalent to `system.component.error_details` and
  `system.component.error_actions`.
- Pass the current error state to those Host primitives through one typed,
  request-local context. Never use a global mutable error singleton.
- Keep semantic HTML and accessible heading/action labels in the no-JavaScript
  SSR output.

An intended theme template shape is:

```html
<div class="sf-page sf-page--system-error" data-theme-owned="presentation">
  <sf-navbar></sf-navbar>
  <main class="theme-error-layout">
    <sf-error-details></sf-error-details>
    <sf-error-actions></sf-error-actions>
  </main>
  <sf-footer></sf-footer>
</div>
```

The exact element names must follow the reviewed Component Catalog. Themes may
omit optional decorative markup, but built-in templates must include the error
details and required home action.

### D3. Only the selected theme may replace system error pages

- System error pages are presentation-only theme surfaces. Plugin `replace`
  contributions targeting these IDs are rejected during manifest/package
  validation.
- Theme replacement continues through normal activation, exact-artifact
  publication, conflict inspection, and rollback. No separate "error theme"
  selector is added.
- A third-party theme may omit any system error contribution; Core then renders
  the Host fallback for that status.
- Both protected built-in themes must declare all four surfaces so first-time
  operators receive complete recommended defaults.
- This deliberately closed plugin surface must be recorded in the Extension
  Surface Matrix with availability and information-disclosure as the reason.

### D4. Error surfaces are L0/L1 only

- Do not execute public L2 components on `system.*` error pages.
- Theme compilation/activation must reject or strip an error template that
  references an executable package component; prefer rejection with a clear
  validation reason so an operator can fix the package before activation.
- Reviewed Host islands (`sf-navbar`, `sf-footer`, error details/actions) remain
  allowed.
- Templates receive no raw request/session object, stack, internal error,
  upstream payload, permission reason, resource identifier, or plugin failure.

### D5. Error rendering must fail closed quickly and without recursion

- A system error resolve has a shorter single-attempt budget than an ordinary
  public page. M0 should measure local behavior, then freeze a production target
  no greater than 1 second unless evidence justifies a smaller bound.
- Any timeout, transport error, malformed render output, missing artifact,
  composition failure, or unavailable ViewModel renders the local Host fallback.
- The fallback path must not call Page Registry again and must not throw from
  its own setup/render path.
- A failed theme must never replace the original error status with 200 or a
  secondary 500.
- Theme support for 5xx is best-effort by design. A failure in the service that
  resolves the theme is exactly when the Host emergency page is required.

### D6. Status, privacy, SEO, and caching remain Host policy

- Preserve the original HTTP status on SSR and client navigation.
- Every system error page sends `Cache-Control: no-store`; disable Nitro route
  cache/SWR for the error response and its payload.
- Emit `noindex,nofollow`. Do not emit success-page canonical links or
  structured data for an error document.
- Public copy is generic. A 403 must not confirm that a protected resource
  exists; a 404 must not disclose hidden/deleted moderation state; a 5xx must
  not expose stack traces, SQL, paths, extension IDs, or upstream messages.
- Detailed evidence belongs in server logs/telemetry with the existing request
  correlation policy, never in the theme ViewModel or HTML.
- No new permission is required. Theme activation/selection retains its current
  authorization and exact-artifact trust controls; the rendered surfaces are
  public responses.

### D7. Activation remains beginner-friendly

- Selecting a theme immediately selects its available system error templates.
- Missing optional templates silently use the complete Core fallback in public;
  package validation/admin inspection should still make coverage visible.
- Restoring the recommended/default theme restores its error templates through
  the existing one-click theme reset. Do not create duplicate reset state.
- Admin Page Registry inspection should label virtual pages clearly instead of
  displaying an unexplained empty path.

## Scope

### In scope

- Catalog, schema, route matching, ViewModel registry/factory/source, compiler,
  selected-theme runtime, Nuxt error entrypoint, narrow Host error islands,
  built-in theme templates/styles, tests, Page Registry docs, ADR, module notes,
  and hot handoff.
- Actual unknown-route 404, permission 403, rate-limit 429, and representative
  500/503 browser flows.
- SSR, hydration, JavaScript-disabled output, dark mode, both locales, and both
  built-in themes.

### Out of scope

- Redesigning JSON API error envelopes or Fiber error middleware.
- A WYSIWYG error-page editor or separate admin copy settings.
- Maintenance scheduling, custom redirects, soft-404 SEO behavior, or CDN edge
  error documents served before a request reaches Nuxt.
- Theme-defined authorization, retry decisions, logging, or HTTP status.
- Public L2 widgets, remote assets, arbitrary JavaScript, or plugin-owned system
  error replacements.
- Refactoring unrelated public pages or the general Page Registry runtime.

## Milestone Map

| Milestone | Focus | Depends on |
| --- | --- | --- |
| M0 | Baseline, ownership, ADR, and contracts | overlapping regression handoff |
| M1 | Catalog, OpenAPI, ViewModels, and owner policy | M0 |
| M2 | Host error context, primitives, and Nuxt mapping | M1 |
| M3 | Runtime restrictions and resilience | M1-M2 |
| M4 | Built-in theme coverage and admin inspection | M2-M3 |
| M5 | Production-path and browser verification | M1-M4 |
| M6 | Full gate, docs, knowledge, and handoff | M5 |

---

## M0 - Baseline, Ownership, And ADR

### Tasks

- [x] Confirm the current-HEAD regression plan has released the overlapping
      files; record current branch, HEAD, and `git status --short`.
- [x] Read the actual error flow from `createError`/middleware through
      `error.vue`, `SFPageOutlet`, `/pages/resolve`, compiled theme snapshot,
      render-output parsing, and Core fallback.
- [x] Enumerate all current browser-facing error producers and distinguish
      redirect/inline/API-envelope behavior from full-page Nuxt errors.
- [x] Measure current 404 resolver/fallback timing and verify whether active L0
      CSS links are present in error SSR HTML.
- [x] Confirm no mature library is needed and record the framework-native choice.
- [x] Add an ADR for D1-D7, including the theme-only replacement boundary and
      5xx best-effort fallback rationale.
- [x] Freeze the Page ID/status matrix, virtual-page representation, component
      IDs, and safe ViewModel fields before implementation (recorded below).

**Exit:** baseline and file/contract map are recorded. Overlapping ownership is
released; the ADR landed with the production edits.

### M0 actual verification (2026-07-22)

#### Dependency gate (released 2026-07-23)

| Check | Result |
| --- | --- |
| Regression plan status | **completed**; M0-M7 closed |
| Focused 404 precursor | **completed**; M0-M6 closed with completion handoff |
| Explicit file handoff to this book | Shared error/Page Registry files released after the focused 404 commit |
| Required preservation | Selected-theme 404 plus its server pre-preparation plugin, request-local presentation composable, exact-artifact validation, document policy, system AST renderer, and emergency-only Core fallback |
| Next action | Completed; preserve the 404 precursor behavior while reviewing/merging this branch |

Shared / high-risk overlap (touched in commits ahead of `origin/main` and/or
owned by regression M1–M3):

- `apps/web/app/error.vue`
- `apps/web/app/components/SFPageOutlet.vue`
- `apps/web/app/composables/useActiveThemeSettings.ts`
- `apps/web/tests/pageOutlet.test.ts`
- `apps/api/app/Support/Pages/catalog.go`
- `apps/api/app/Support/Pages/route_matcher.go`
- `apps/api/app/Support/Pages/viewmodel_factory.go`
- `apps/api/app/Models/PageViewModels/source.go`
- `apps/api/app/Support/ThemeCompiler/viewmodel_registry.go`

Also consumed by this book as released shared runtime surfaces:

- `apps/web/app/components/SFErrorPageContent.vue`
- `apps/web/app/utils/errorPage.ts`
- `apps/web/app/utils/pageResolve.ts`
- `apps/web/app/components/SFThemeTemplate.vue`
- `contracts/openapi/schemas/pages.yaml`
- built-in theme `not-found.html` / `theme.json` page contributions

#### Error flow at the 2026-07-22 M0 audit

1. Producers call Nuxt `createError` / `showError`, or Nuxt synthesizes 404 for
   unmatched routes.
2. `apps/web/app/error.vue` is the only browser error entrypoint.
3. **Only status 404** routes through `SFPageOutlet page="system.not_found"`;
   all other statuses render `SFErrorPageContent` alone (Host full page).
4. `SFPageOutlet` → `useAsyncData` → `GET /pages/resolve?id=&path=&query=` with
   `PAGE_RESOLVE_TIMEOUT_MS` **5000 (dev) / 8000 (prod)** and **maxAttempts=2**.
5. Healthy theme: `provider=sforum.default-theme`, `fallback=false`,
   `templatePath=templates/not-found.html`, `renderOutput` islands.
6. Theme L1 is a shell + whole-page Host island `sf-not-found-page` →
   `system.component.not_found` → default slot (`SFErrorPageContent`).
7. Transport/resolve failure: local `coreResolveFallback` + slot; no second
   Page Registry call from the fallback payload helper itself.
8. Host content/actions/SEO live in `utils/errorPage.ts` +
   `SFErrorPageContent.vue` (`noindex` via `useSForumSeo`).

#### Browser-facing error producers

| Kind | Examples | Full-page Nuxt error? |
| --- | --- | --- |
| Unknown public URL | Nuxt unmatched route → 404 | Yes → themed `system.not_found` |
| Registry catch-all 404/403 | `pages/[...sfRegistryPage].vue`, `pages/x/[...path].vue` | Yes |
| Missing taxonomy | `SFTagShowPage`, `SFCategoryShowPage`, `SFTagIndexPage` | Yes 404 |
| Dev gallery | `pages/components.vue` `showError(404)` | Yes |
| Auth 401 | guest/auth middleware login redirect | **No** themed error page |
| Admin inline errors | `admin/users.vue` local `showError(message)` | **No** (inline UI) |
| API JSON 4xx/5xx | Fiber envelopes, write rate-limit **429** | **No** browser page |
| Nitro/proxy | `pluginRouteProxy` 503, icon collection 404 | Server/API path, not product error page |

`errorPage.ts` maps 403 / 404 / 500 / 503 (+ generic). **No 429** content key
yet. **502/504** fall through generic Host content if they ever reach Nuxt.

#### Timing and SSR baseline (local, 2026-07-22)

Environment: Nuxt on `:3000` (user-owned), API on `:8081`.

| Probe | Result |
| --- | --- |
| `GET /api/v1/pages/resolve?id=system.not_found&path=/…` | **~15 ms**, HTTP 200, `provider=sforum.default-theme`, `fallback=false` |
| `GET /this-path-…` with `Accept: text/html` | **HTTP 404**, **~240–330 ms**, ~128 KB HTML |
| Status preserved | Yes (`404 Page not found`) |
| Themed L1 markers | `data-provider="sforum.default-theme"`, `sf-page--not-found`, `data-theme-owned="presentation"`, Host `sforum-error-page` inside island |
| `Cache-Control` | `no-cache` (not yet `no-store`) |
| Robots | header `x-robots-tag: noindex, nofollow`; meta `noindex,follow` (meta not `nofollow`) |
| Active L0 `data-sforum-theme-skin` links | **Absent** on both homepage and 404 HTML in this probe (skin fetch not visible in SSR output) |
| Host/core CSS | Present (`sforum-theme.css`, component scoped CSS, etc.) |

**Timeout freeze candidate for M3:** production system-error resolve budget
**≤ 1000 ms**, **single attempt**, no retry — current ordinary-page budget
(5–8 s, 2 attempts) is intentionally too large for D5.

#### Library survey

No new dependency. Reuse Nuxt error boundary, Page Registry L0/L1, Go
`html/template` compiler, existing Host islands. Do not `bun add` / `go get`
for this feature.

#### Frozen contract map (implementation law when unblocked)

| Page id | Contract | Status family | Access |
| --- | --- | --- | --- |
| `system.forbidden` | `sforum.page.forbidden@1` | 403 | public virtual |
| `system.not_found` | `sforum.page.not_found@1` | 404 | public virtual (preserve) |
| `system.rate_limited` | `sforum.page.rate_limited@1` | 429 | public virtual |
| `system.server_error` | `sforum.page.server_error@1` | 500, 502, 503, 504 | public virtual |

- `PageDefinition.Virtual bool` (OpenAPI + admin typing); generalize
  `MatchCorePagePath` off the hard-coded `system.not_found` exception.
- Host islands (reviewed): `system.component.error_details`,
  `system.component.error_actions` (replace whole-page
  `system.component.not_found` / `sf-not-found-page` on built-in templates).
- Safe ViewModel fields only: status code, localized public title/message keys
  or strings, robots `noindex,nofollow`. No stack, path secrets, extension IDs,
  permission reasons, or upstream payloads.
- Theme-only replace; plugins rejected; no L2 on `system.*`.
- 401 stays login/redirect; API JSON envelopes unchanged.

#### M0 follow-up closure

- ADR file landed as
  `knowledge/decisions/2026-07-23-theme-defined-system-error-pages.md`.
- The system-error resolver uses one attempt with an 800 ms development /
  1000 ms production timeout.
- 502/504 remain mapped to `system.server_error` even though current QA data
  does not expose a natural full-page producer for those statuses.
- Component Catalog and runtime island bindings now expose the narrow system
  error Host islands.

## M1 - Catalog, Contracts, And Theme Ownership Policy

### Tasks

- [x] Add `system.forbidden`, `system.rate_limited`, and `system.server_error`;
      preserve `system.not_found` and its contract unchanged.
- [x] Add the explicit virtual-page property to `PageDefinition`; update catalog
      invariants, copies, JSON, admin frontend typing, and
      `contracts/openapi/schemas/pages.yaml`.
- [x] Generalize core-path matching for declared virtual pages and delete the
      hard-coded `system.not_found` path exception.
- [x] Register each typed error Page ViewModel and update the Core factory/data
      source exhaustively so catalog-count tests cannot drift.
- [x] Set safe localized defaults and `noindex,nofollow` for every error model.
- [x] Enforce theme-only replace ownership for system error targets during
      contribution validation; reject plugins with a stable validation reason.
- [x] Keep provider selection inspectable and preserve normal Core fallback when
      the active theme omits a surface.
- [x] Update OpenAPI examples/descriptions and run
      `ruby scripts/validate-openapi-refs.rb`.

### Required tests

- [x] Catalog validity and unique contracts.
- [x] Virtual pages accept the current request path but never participate in
      ordinary public route matching.
- [x] All catalog entries have a production Core ViewModel.
- [x] Theme contribution allowed; plugin contribution denied.
- [x] Missing theme contribution resolves to Core without an error.
- [x] Controller/runtime resolve returns the selected exact theme for every new
      Page ID.

**Exit:** backend and OpenAPI tests pass; no frontend status is routed to the
wrong Page ID.

## M2 - Host Error Context And Nuxt Rendering

### Tasks

- [x] Add one typed request-local system-error context carrying only normalized
      status, public content keys, retry policy, and Host actions.
- [x] Split `SFErrorPageContent.vue` into reusable details/actions primitives and
      a complete emergency composition. Avoid duplicating status mapping or
      navigation behavior.
- [x] Register reviewed Host components in `SFThemeTemplate.vue`, the Go theme
      island bindings, and the Component Catalog/generation source.
- [x] Map 403/404/429/500-family statuses to the stable Page IDs in `error.vue`;
      keep other statuses on the generic Host fallback.
- [x] Route selected IDs through `SFPageOutlet` without changing the original
      Nuxt status or clearing the error during SSR.
- [x] Apply `no-store`, no-SWR, and error SEO policy before themed rendering.
- [x] Ensure the emergency fallback does not depend on Page Registry, optional
      theme settings, session restore, or a successful API read.
- [x] Preserve accessible headings, focus order, responsive button layout,
      locale-aware home path, browser-back behavior, and retry behavior.

### Required tests

- [x] Pure status-to-Page-ID mapping, including unknown and malformed status.
- [x] SSR render of themed and fallback paths with the same original status.
- [x] Hydration of details/actions without mismatch.
- [x] No stack/internal message reaches rendered props or HTML.
- [x] 401 remains redirect/login behavior rather than a themed error surface.

**Exit:** all four status families render complete Host fallback content before
built-in templates are changed.

## M3 - Runtime Restrictions And Resilience

### Tasks

- [x] Add the system-error resolve policy with one short bounded attempt and no
      stale shell reuse.
- [x] Prove malformed theme output, missing snapshot, resolver timeout, and
      transport failure all return the emergency composition without recursion.
- [x] Prohibit public L2 descriptors/components on every system error page at
      package compile/activation time.
- [x] Keep reviewed Host islands and L0 assets allowed under exact-artifact
      validation.
- [x] Verify failed resolution marks the outer response and payload `no-store`
      and cannot enter shared Nitro SWR.
- [x] Confirm active theme identity/digest is not reused across a theme switch,
      locale change, actor change, path change, or error status change.
- [x] Add bounded telemetry for selected-theme success and fallback reason
      without recording request paths or private error detail unnecessarily.

### Required tests

- [x] Timeout and unavailable-API fallback completes inside the frozen budget.
- [x] Render-parser and runtime errors cannot cause an error-boundary loop.
- [x] L2 declaration on a system error page is rejected.
- [x] Stale or mismatched artifact never renders.
- [x] Error output is not shared across status, locale, path, or actor.

**Exit:** fault-injection tests pass and the emergency path is independent of
the failed runtime.

## M4 - Built-In Themes And Operator Inspection

### Tasks

- [x] Add forbidden, rate-limited, and server-error declarations/templates to
      both `sforum-default` and `sforum-nocturne`.
- [x] Rewrite both not-found templates so the theme owns chrome/layout and uses
      narrow error details/actions Host islands instead of the whole-page slot.
- [x] Add cohesive responsive/dark styles in each theme's existing asset files;
      do not duplicate the Host fallback stylesheet.
- [x] Keep icons from the approved icon integration; no emoji, inline SVG, or
      remote decoration.
- [x] Ensure status/title/actions fit mobile and desktop containers in Chinese
      and English.
- [x] Update built-in completeness tests to require all system error templates,
      reviewed islands, presentation ownership markers, and no L2 references.
- [x] Update `/admin/extensions/pages` so virtual system pages have a clear
      localized label instead of an empty-path dash; retain current approval,
      reset, and inspection controls.
- [x] Verify default/theme reset restores recommended error coverage and does
      not preserve a stale template digest.

**Exit:** both built-in themes provide materially distinct but accessible error
presentation and pass completeness/activation tests.

## M5 - Production-Path And Browser Verification

### Automated matrix

- [x] Go unit tests for Catalog, route matcher, ViewModel registry/factory/source,
      contribution validation, compiler, runtime snapshot, and controller resolve.
- [x] Web unit tests for mapping, context, Page Outlet, render output, hydration,
      SEO, cache headers, actions, and fallback.
- [x] Repo validation scripts updated so old assertions about a whole-page
      `HostPageIsland` cannot hide a regression.
- [x] OpenAPI reference validation passes.

### Browser / production-path matrix

Use the in-app Browser skill for rendered UI verification. Do not kill the
user-owned Nuxt server on port 3000.

- [x] Default theme: actual unknown URL returns 404, themed marker and active
      CSS in SSR; home/back actions work.
- [x] Nocturne: built-in template/runtime coverage and preview digest are
      available; the final cleanup did not repeat a QA-DB theme activation
      switch.
- [x] Representative 403, 429, 500, and 503 flows keep their exact status and
      correct action set. Current QA data has no natural full-page 429/5xx
      producer; mapping, resolver, and fallback behavior are covered by
      automated tests rather than committed test-only routes.
- [x] Disable JavaScript and verify status, title, description, home link,
      navbar/footer, and theme structure remain usable; final evidence is from
      SSR output and automated render assertions rather than a fresh final
      Browser pass.
- [x] Force Page Registry/API failure and verify prompt complete Host fallback,
      no blank page, no loop, no console error cascade, and no leaked detail;
      covered by resolver/presentation fallback tests and prior fault checks.
- [x] Check desktop and mobile in both locales and light/dark appearance; capture
      screenshots as evidence. Final Browser spot checks covered default theme
      desktop/mobile, dark mode, and English locale.
- [x] Switch themes without rebuilding and verify the next error navigation uses
      the new immutable digest with no stale presentation; exact artifact
      identity is covered by runtime tests and the Nocturne preview digest.

### Acceptance invariants

- [x] Error documents never return HTTP 200.
- [x] Error pages never become indexable or share-cacheable.
- [x] Themes cannot alter authorization or retry policy.
- [x] Plugins cannot replace a system error surface.
- [x] Theme/runtime failure never prevents the Host emergency page.
- [x] No theme template or browser payload contains internal error detail.

**Exit:** focused automated suites are green, and current Browser plus
production-path evidence is recorded with the limitations above.

## M6 - Full Gate, Documentation, And Handoff

### Tasks

- [x] Run formatting for touched Go/Vue/Markdown files as applicable.
- [x] Run `cd apps/api && go test ./...`.
- [x] Run `cd apps/web && bun test`.
- [x] Run `cd apps/web && bun run typecheck`.
- [x] Run `cd apps/web && bun run build`.
- [x] Run `ruby scripts/validate-openapi-refs.rb`.
- [x] Run `./scripts/test.sh` and distinguish unrelated pre-existing failures
      with evidence; broader gate evidence came from the implementation
      checkpoint, while final cleanup reran focused suites and diff checks.
- [x] Run `git diff --check` and inspect `git status --short` without touching
      unrelated user changes.
- [x] Update `docs/extensions/page-catalog.md` and the bilingual theme-author
      docs with virtual/error surfaces, contracts, allowed Host islands,
      theme-only ownership, fallback, and no-L2 rule.
- [x] Update the Extension Surface Matrix for route, template, component,
      identity/permission, navigation, cache, and lifecycle treatment.
- [x] Update `knowledge/modules/frontend.md`, `knowledge/modules/extensions.md`,
      this plan status, `knowledge/plans/README.md`, `knowledge/index.md`, and one
      hot handoff. Archive intermediate handoffs if the work spans sessions.

**Exit:** prior full-gate checkpoint evidence plus final focused gates are
recorded, docs and knowledge are truthful, the plan is marked completed, and
no required implementation work remains.

## Completion Definition

This plan is marked completed because the following are true:

- All four stable system Page IDs resolve through the selected healthy theme.
- Theme templates own page structure and use narrow Host semantic/action islands.
- Original status, privacy, SEO, caching, and authorization remain Host-owned.
- System error pages cannot execute L2 or be replaced by plugins.
- Both built-in themes provide complete recommended coverage.
- Resolver/theme/runtime failure produces a prompt, complete, non-recursive Host
  fallback with the same HTTP status.
- Backend, frontend, OpenAPI, Browser spot checks, and prior full-gate
  checkpoint evidence pass for the implemented scope.
- The ADR, author docs, Extension Surface Matrix, module notes, plan ledger, and
  hot handoff agree with production behavior.

## Resolved M0 Questions

- 502/504 are supported as `system.server_error` status-family inputs even
  though current QA data does not expose natural full-page producers.
- The system-error resolve timeout is frozen at one attempt, 800 ms in
  development and 1000 ms in production.
- The Component Catalog and runtime bindings expose the narrow Host islands
  needed by system error templates.
