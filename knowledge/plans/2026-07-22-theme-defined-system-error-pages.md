# Theme-Defined System Error Pages - Task Book

Status: **ready** - approved implementation checklist; start code changes only
after the overlapping current-HEAD regression work is closed or handed off  
Date: 2026-07-22  
Goal: let the selected public theme own the L1 presentation of common system
error pages while Host retains status, safe content, behavior, SEO, and an
always-available emergency fallback.

Implement this book milestone by milestone. Each milestone must leave the
repository buildable and must record exact verification output. Do not merge
this work into the search, content-revision, or V3 production-rewire programs.

## Dependency And Shared-Worktree Warning

The current worktree already contains overlapping edits in `apps/web/app/error.vue`,
`SFPageOutlet.vue`, Page Registry route matching, Page ViewModels, tests, and
knowledge files for `2026-07-22-current-head-regression-remediation.md`.

- Before editing, re-read `git status --short`, the active regression plan, and
  its latest handoff.
- Do not overwrite, revert, restage, or reimplement those changes.
- Start implementation only when the overlapping owner has completed M7 or has
  explicitly handed the files over. M0 inspection may run earlier because it is
  read-only.
- Preserve the repaired behavior that `system.not_found` resolves against the
  selected theme and falls back to Core on resolver failure.

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

- [ ] Confirm the current-HEAD regression plan has released the overlapping
      files; record current branch, HEAD, and `git status --short`.
- [ ] Read the actual error flow from `createError`/middleware through
      `error.vue`, `SFPageOutlet`, `/pages/resolve`, compiled theme snapshot,
      render-output parsing, and Core fallback.
- [ ] Enumerate all current browser-facing error producers and distinguish
      redirect/inline/API-envelope behavior from full-page Nuxt errors.
- [ ] Measure current 404 resolver/fallback timing and verify whether active L0
      CSS links are present in error SSR HTML.
- [ ] Confirm no mature library is needed and record the framework-native choice.
- [ ] Add an ADR for D1-D7, including the theme-only replacement boundary and
      5xx best-effort fallback rationale.
- [ ] Freeze the Page ID/status matrix, virtual-page representation, component
      IDs, and safe ViewModel fields before implementation.

**Exit:** no production behavior change; ADR accepted; exact file/contract map
and baseline evidence recorded in this plan or the first implementation PR.

## M1 - Catalog, Contracts, And Theme Ownership Policy

### Tasks

- [ ] Add `system.forbidden`, `system.rate_limited`, and `system.server_error`;
      preserve `system.not_found` and its contract unchanged.
- [ ] Add the explicit virtual-page property to `PageDefinition`; update catalog
      invariants, copies, JSON, admin frontend typing, and
      `contracts/openapi/schemas/pages.yaml`.
- [ ] Generalize core-path matching for declared virtual pages and delete the
      hard-coded `system.not_found` path exception.
- [ ] Register each typed error Page ViewModel and update the Core factory/data
      source exhaustively so catalog-count tests cannot drift.
- [ ] Set safe localized defaults and `noindex,nofollow` for every error model.
- [ ] Enforce theme-only replace ownership for system error targets during
      contribution validation; reject plugins with a stable validation reason.
- [ ] Keep provider selection inspectable and preserve normal Core fallback when
      the active theme omits a surface.
- [ ] Update OpenAPI examples/descriptions and run
      `ruby scripts/validate-openapi-refs.rb`.

### Required tests

- [ ] Catalog validity and unique contracts.
- [ ] Virtual pages accept the current request path but never participate in
      ordinary public route matching.
- [ ] All catalog entries have a production Core ViewModel.
- [ ] Theme contribution allowed; plugin contribution denied.
- [ ] Missing theme contribution resolves to Core without an error.
- [ ] Controller/runtime resolve returns the selected exact theme for every new
      Page ID.

**Exit:** backend and OpenAPI tests pass; no frontend status is routed to the
wrong Page ID.

## M2 - Host Error Context And Nuxt Rendering

### Tasks

- [ ] Add one typed request-local system-error context carrying only normalized
      status, public content keys, retry policy, and Host actions.
- [ ] Split `SFErrorPageContent.vue` into reusable details/actions primitives and
      a complete emergency composition. Avoid duplicating status mapping or
      navigation behavior.
- [ ] Register reviewed Host components in `SFThemeTemplate.vue`, the Go theme
      island bindings, and the Component Catalog/generation source.
- [ ] Map 403/404/429/500-family statuses to the stable Page IDs in `error.vue`;
      keep other statuses on the generic Host fallback.
- [ ] Route selected IDs through `SFPageOutlet` without changing the original
      Nuxt status or clearing the error during SSR.
- [ ] Apply `no-store`, no-SWR, and error SEO policy before themed rendering.
- [ ] Ensure the emergency fallback does not depend on Page Registry, optional
      theme settings, session restore, or a successful API read.
- [ ] Preserve accessible headings, focus order, responsive button layout,
      locale-aware home path, browser-back behavior, and retry behavior.

### Required tests

- [ ] Pure status-to-Page-ID mapping, including unknown and malformed status.
- [ ] SSR render of themed and fallback paths with the same original status.
- [ ] Hydration of details/actions without mismatch.
- [ ] No stack/internal message reaches rendered props or HTML.
- [ ] 401 remains redirect/login behavior rather than a themed error surface.

**Exit:** all four status families render complete Host fallback content before
built-in templates are changed.

## M3 - Runtime Restrictions And Resilience

### Tasks

- [ ] Add the system-error resolve policy with one short bounded attempt and no
      stale shell reuse.
- [ ] Prove malformed theme output, missing snapshot, resolver timeout, and
      transport failure all return the emergency composition without recursion.
- [ ] Prohibit public L2 descriptors/components on every system error page at
      package compile/activation time.
- [ ] Keep reviewed Host islands and L0 assets allowed under exact-artifact
      validation.
- [ ] Verify failed resolution marks the outer response and payload `no-store`
      and cannot enter shared Nitro SWR.
- [ ] Confirm active theme identity/digest is not reused across a theme switch,
      locale change, actor change, path change, or error status change.
- [ ] Add bounded telemetry for selected-theme success and fallback reason
      without recording request paths or private error detail unnecessarily.

### Required tests

- [ ] Timeout and unavailable-API fallback completes inside the frozen budget.
- [ ] Render-parser and runtime errors cannot cause an error-boundary loop.
- [ ] L2 declaration on a system error page is rejected.
- [ ] Stale or mismatched artifact never renders.
- [ ] Error output is not shared across status, locale, path, or actor.

**Exit:** fault-injection tests pass and the emergency path is independent of
the failed runtime.

## M4 - Built-In Themes And Operator Inspection

### Tasks

- [ ] Add forbidden, rate-limited, and server-error declarations/templates to
      both `sforum-default` and `sforum-nocturne`.
- [ ] Rewrite both not-found templates so the theme owns chrome/layout and uses
      narrow error details/actions Host islands instead of the whole-page slot.
- [ ] Add cohesive responsive/dark styles in each theme's existing asset files;
      do not duplicate the Host fallback stylesheet.
- [ ] Keep icons from the approved icon integration; no emoji, inline SVG, or
      remote decoration.
- [ ] Ensure status/title/actions fit mobile and desktop containers in Chinese
      and English.
- [ ] Update built-in completeness tests to require all system error templates,
      reviewed islands, presentation ownership markers, and no L2 references.
- [ ] Update `/admin/extensions/pages` so virtual system pages have a clear
      localized label instead of an empty-path dash; retain current approval,
      reset, and inspection controls.
- [ ] Verify default/theme reset restores recommended error coverage and does
      not preserve a stale template digest.

**Exit:** both built-in themes provide materially distinct but accessible error
presentation and pass completeness/activation tests.

## M5 - Production-Path And Browser Verification

### Automated matrix

- [ ] Go unit tests for Catalog, route matcher, ViewModel registry/factory/source,
      contribution validation, compiler, runtime snapshot, and controller resolve.
- [ ] Web unit tests for mapping, context, Page Outlet, render output, hydration,
      SEO, cache headers, actions, and fallback.
- [ ] Repo validation scripts updated so old assertions about a whole-page
      `HostPageIsland` cannot hide a regression.
- [ ] OpenAPI reference validation passes.

### Browser matrix

Use the in-app Browser skill for rendered UI verification. Do not kill the
user-owned Nuxt server on port 3000.

- [ ] Default theme: actual unknown URL returns 404, themed marker and active
      CSS in SSR; home/back actions work.
- [ ] Nocturne: the same URL remains 404 and visibly uses Nocturne structure.
- [ ] Representative 403, 429, 500, and 503 flows keep their exact status and
      correct action set.
- [ ] Disable JavaScript and verify status, title, description, home link,
      navbar/footer, and theme structure remain usable.
- [ ] Force Page Registry/API failure and verify prompt complete Host fallback,
      no blank page, no loop, no console error cascade, and no leaked detail.
- [ ] Check desktop and mobile in both locales and light/dark appearance; capture
      screenshots as evidence.
- [ ] Switch themes without rebuilding and verify the next error navigation uses
      the new immutable digest with no stale presentation.

### Acceptance invariants

- [ ] Error documents never return HTTP 200.
- [ ] Error pages never become indexable or share-cacheable.
- [ ] Themes cannot alter authorization or retry policy.
- [ ] Plugins cannot replace a system error surface.
- [ ] Theme/runtime failure never prevents the Host emergency page.
- [ ] No theme template or browser payload contains internal error detail.

**Exit:** focused automated suites and the browser matrix are green with saved
evidence paths/results.

## M6 - Full Gate, Documentation, And Handoff

### Tasks

- [ ] Run formatting for touched Go/Vue/Markdown files as applicable.
- [ ] Run `cd apps/api && go test ./...`.
- [ ] Run `cd apps/web && bun test`.
- [ ] Run `cd apps/web && bun run typecheck`.
- [ ] Run `cd apps/web && bun run build`.
- [ ] Run `ruby scripts/validate-openapi-refs.rb`.
- [ ] Run `./scripts/test.sh` and distinguish unrelated pre-existing failures
      with evidence; do not claim completion while an in-scope gate fails.
- [ ] Run `git diff --check` and inspect `git status --short` without touching
      unrelated user changes.
- [ ] Update `docs/extensions/page-catalog.md` and the bilingual theme-author
      docs with virtual/error surfaces, contracts, allowed Host islands,
      theme-only ownership, fallback, and no-L2 rule.
- [ ] Update the Extension Surface Matrix for route, template, component,
      identity/permission, navigation, cache, and lifecycle treatment.
- [ ] Update `knowledge/modules/frontend.md`, `knowledge/modules/extensions.md`,
      this plan status, `knowledge/plans/README.md`, `knowledge/index.md`, and one
      hot handoff. Archive intermediate handoffs if the work spans sessions.

**Exit:** full gate green, docs and knowledge truthful, plan marked completed,
and no required work remains.

## Completion Definition

Do not mark this plan completed until all of the following are true:

- All four stable system Page IDs resolve through the selected healthy theme.
- Theme templates own page structure and use narrow Host semantic/action islands.
- Original status, privacy, SEO, caching, and authorization remain Host-owned.
- System error pages cannot execute L2 or be replaced by plugins.
- Both built-in themes provide complete recommended coverage.
- Resolver/theme/runtime failure produces a prompt, complete, non-recursive Host
  fallback with the same HTTP status.
- Backend, frontend, OpenAPI, browser, and full repository gates pass.
- The ADR, author docs, Extension Surface Matrix, module notes, plan ledger, and
  hot handoff agree with production behavior.

## Open Questions To Resolve In M0

These questions do not change the frozen security/ownership boundaries:

- Whether 502/504 are currently emitted as full Nuxt pages or only through API
  envelopes; unused mappings may remain supported but need not gain fixtures.
- The exact sub-1-second system resolve timeout after measuring local and
  production-like behavior.
- Whether the existing generated Component Catalog command can register both
  new Host islands without a manual compatibility alias for
  `system.component.not_found`.

