# Theme-Consistent Public Resource 404 - Task Book

Status: **completed** - G0 and M0-M6 closed on 2026-07-23; ordinary public
resource 404s keep the healthy selected theme and Core remains emergency-only
Date: 2026-07-22
Goal: public resources that do not exist or are not publicly visible must keep
the selected theme's navbar, sidebar/body layout, and footer while Host preserves
the real HTTP 404, safe copy, SEO/cache policy, and emergency fallback.

Implement this book milestone by milestone. Do not turn it into a CSS patch and
do not expand it to the complete 403/429/5xx program.

## Relationship To Existing Work

This is the approved, narrow precursor to
`2026-07-22-theme-defined-system-error-pages.md`:

- this book owns public **resource-not-found 404** behavior only;
- the broader book remains the owner of complete 403/429/500/502/503/504
  coverage and the final general-purpose error component architecture;
- both books share `error.vue`, `SFPageOutlet.vue`, Page Registry Controller /
  ViewModel code, `system.not_found`, built-in theme templates, and tests;
- do not run both implementations concurrently;
- G0 closed `2026-07-22-current-head-regression-remediation.md` M7 first; the
  next implementation may edit shared 404/Page Registry files under this book;
- an explicit written handoff remains an acceptable substitute when another
  task closes M7 before implementation begins.

The focused book does not supersede the broader system-error book. When this
book closes, the broader book must consume its 404 behavior instead of
reimplementing it.

## Required Reading Before Coding

1. `AGENTS.md`
2. `knowledge/index.md`
3. `knowledge/modules/frontend.md`
4. `knowledge/modules/extensions.md`
5. `knowledge/plans/2026-07-22-current-head-regression-remediation.md`
6. `knowledge/plans/2026-07-22-theme-defined-system-error-pages.md`
7. `knowledge/decisions/2026-07-13-runtime-page-registry-themes.md`
8. `docs/extensions/page-catalog.md`
9. This task book

## User-Visible Outcome

For a healthy selected theme:

```text
open a missing/deleted/hidden public resource
-> Host confirms that it is not publicly available
-> system.not_found renders through the selected theme
-> the selected theme's navbar/sidebar/body layout/footer stay consistent
-> the response remains a real, private HTTP 404
```

Core public chrome must not appear for an ordinary business 404. Core remains
an emergency-only path for an unavailable or invalid theme/runtime.

For the built-in default theme specifically:

- topbar spacing, search placement, utility/session controls, and active accent
  match normal default-theme public pages;
- desktop keeps the normal left public navigation and its configured category
  links/counts;
- the 404 content occupies the normal center content area; a right rail may be
  intentionally absent because there is no resource metadata;
- the left navigation collapses at the same breakpoint as normal public pages;
- footer presentation and active L0 tokens remain theme-owned;
- no nested or duplicate navbar/sidebar/footer may appear.

Nocturne and future themes retain their own L1 structure. Do not force every
theme to copy the default three-column layout.

## M0 Baseline Evidence (Planning Session)

Live local evidence on 2026-07-22:

| Probe | Current result |
| --- | --- |
| Missing topic | `/t/60/topic-17vm0aw` |
| Topic API | HTTP 404, `forum.topic_not_found` |
| Page resolve API | HTTP 404, `pages.data_not_found` |
| Rendered outlet | `provider=core`, `template=0`, `hostChrome=1` |
| Missing-page document | HTTP 200 with `s-maxage=60, stale-while-revalidate` |
| Visible chrome | `SFHostPublicChrome`, not selected-theme L1 |
| Active theme CSS | present; this is not a missing-stylesheet bug |
| Existing topic control | `/t/59/topic-nwlc1n-4` -> default theme, template=1, hostChrome=0 |
| Browser console | no visible warning/error for the final client page |
| SSR payload | includes a Nuxt composable-context error after the resolve retry |

Current causal chain:

1. `CorePageViewModelSource.populateTopicDetail` maps a missing topic to
   `ErrCorePageDataNotFound`.
2. the Page Registry Controller returns the `pages.data_not_found` 404 envelope;
3. `SFPageOutlet` retries every thrown resolve error, including semantic 4xx;
4. it catches the final error as generic `transport_unavailable`;
5. `useHostPublicChrome` becomes true and renders the Core navbar/footer shell;
6. `SFTopicShowPage` separately renders an inline not-found card, so Nuxt never
   owns the document as an error response and SSR remains 200/cacheable.

## Short Library Survey

No new dependency is justified.

- Nuxt `createError` / `showError` / `error.vue` already provide SSR and client
  error-boundary behavior.
- ofetch errors already expose HTTP status and API envelopes.
- `apiErrorReason` already extracts stable Host reason codes.
- Page Registry, L0/L1 templates, `system.not_found`, `SFHomeNavigation`, and
  built-in theme runtime snapshots already cover the presentation lifecycle.

Do not run `bun add`, `bun install`, `go get`, or `go mod tidy` for this work.

## Frozen Product And Architecture Decisions

### D1. Semantic 404 is not a theme transport failure

`pages.data_not_found`, `forum.topic_not_found`, and equivalent reviewed public
resource-not-found reasons are semantic outcomes. They must not become
`transport_unavailable`, must not be retried, and must not select Core chrome.

Network failures, timeouts, malformed render output, runtime/artifact mismatch,
and unavailable ViewModel infrastructure remain technical failures and keep
the existing fail-closed behavior.

### D2. Resource 404 enters `system.not_found`

When Page Registry cannot build a selected-theme resource page because the
resource is not publicly available, the Nuxt route must raise a sanitized 404
and let `error.vue` render `system.not_found` against the selected theme.

Do not render the original resource page template with fake or partial resource
data. Do not redirect to `/`. Do not add a second public `/404` route.

### D3. Theme owns chrome; Host owns truth and actions

The selected theme owns navbar/footer placement, body width, sidebar presence,
spacing, responsive behavior, and visual grouping.

Host owns:

- status code and whether the condition is 404;
- localized generic title/description;
- home/back actions and `clearError` behavior;
- safe public navigation/category data;
- disclosure policy for hidden/deleted/private resources.

Use one request-local typed error context. Do not store the current error in a
global mutable singleton.

For the focused implementation, replace the current whole-page
`system.component.not_found` Host slot with a dedicated, self-contained 404 body
island. The island may reuse `SFHomeNavigation` and Host error primitives, while
the L1 template mounts `sf-navbar`, the body island, and `sf-footer`. Keep a
complete local Host emergency page that never calls Page Registry.

### D4. Default theme uses its real public shell

The default `not-found.html` must use the same outer theme classes/tokens and
desktop left-navigation geometry as its normal public surfaces. Reuse existing
components and CSS contracts; do not copy a second sidebar or navbar
implementation into the error component.

The default theme may omit the right information rail on 404. That absence must
be deliberate and responsive, not leftover empty space or a broken grid.

### D5. Core fallback is emergency-only

Core chrome remains mandatory for:

- Page Registry transport timeout/unavailability;
- missing or corrupt exact theme artifact;
- theme render/composition failure;
- safe mode and other Host-owned recovery states.

Do not remove `SFHostPublicChrome` or the local error emergency page. The goal
is to stop ordinary business 404s from being misclassified as emergencies.

### D6. Retry only transient failures and keep Nuxt context valid

The Page Registry retry helper must accept an explicit retry decision or use a
small reviewed classifier:

- do not retry any 4xx semantic/API contract response;
- allow the existing bounded retry only for transient transport failures and
  reviewed retryable 5xx responses;
- preserve the original error when no retry is allowed;
- make a real retry safe for SSR Nuxt request context, preferably by capturing
  request-scoped headers/cookies when the composable is created rather than
  invoking Nuxt composables after an awaited delay;
- never retry a mutation (this helper is GET-only and must stay that way).

### D7. HTTP, privacy, SEO, and cache behavior are Host policy

Every covered not-found document must:

- return HTTP 404 on hard SSR navigation;
- stay 404 after hydration and client navigation;
- send `Cache-Control: no-store` and disable Nitro cache/SWR for the document
  and payload;
- emit `noindex,nofollow` consistently in header and meta;
- omit success canonical URLs and success structured data;
- use generic copy that does not reveal whether a resource was deleted, hidden,
  private, never existed, or was denied by moderation policy;
- keep API JSON envelopes unchanged.

### D8. No new operator setting or permission

Theme selection immediately determines the 404 presentation. Resetting to the
recommended theme restores its 404 presentation through the existing theme
reset flow. Do not add an "error theme" selector.

The rendered 404 is public. Theme activation/trust continues to use existing
`super_admin` controls; no new RBAC key is required.

## Resource Matrix

Initial required coverage:

| Public surface | Page ID | Not-found source | Expected result |
| --- | --- | --- | --- |
| Topic detail | `forum.topic.show` | missing/hidden/deleted topic | themed `system.not_found`, HTTP 404 |
| Category detail | `forum.category.show` | unknown/private category | themed `system.not_found`, HTTP 404 |
| Tag detail | `forum.tag.show` | unknown/unavailable tag | themed `system.not_found`, HTTP 404 |
| Public profile | `forum.profile.show` | unknown/non-public profile | themed `system.not_found`, HTTP 404 |
| Unknown Nuxt route | `system.not_found` | unmatched route | preserve themed 404, remove current inconsistencies |

M0 must inventory any other public `*.show` surface returning
`ErrCorePageDataNotFound`. Add it only when it uses the same public semantic and
requires no new product decision.

Out of scope:

- API JSON error redesign;
- login redirect/401 behavior;
- complete 403/429/500/502/503/504 theme coverage;
- CDN/edge errors before Nuxt;
- maintenance scheduling or custom redirects;
- L2 execution on error pages;
- revealing moderation/private-resource state;
- unrelated Page Registry refactoring.

## Expected Flow

```text
public route shell
  -> SFPageOutlet resolves selected theme + resource ViewModel
     -> healthy resource: render selected page L1
     -> semantic resource 404: throw sanitized Nuxt 404 (no retry)
        -> error.vue -> SFPageOutlet(system.not_found)
           -> healthy selected theme: theme L1 + dedicated 404 body island
           -> theme/runtime failure: local Core emergency page, no recursion
     -> technical resolve failure: existing Core fail-closed public chrome
```

The `system.not_found` resolver must never recurse into the failing resource
resolver or fetch the missing resource again.

## Likely File Ownership

Read the actual code again before editing. Expected overlap includes:

- `apps/web/app/components/SFPageOutlet.vue`
- `apps/web/app/utils/pageResolve.ts`
- `apps/web/app/composables/useApiClient.ts` (only if required for retry context)
- `apps/web/app/error.vue`
- `apps/web/app/components/SFErrorPageContent.vue`
- `apps/web/app/components/SFThemeTemplate.vue`
- new focused Host 404 body/error-context files under `apps/web/app/`
- affected public resource page/body islands
- `apps/api/app/Http/Controllers/Pages/controller.go`
- `apps/api/app/Models/PageViewModels/source.go`
- ThemeCompiler / Component Catalog only if the reviewed island contract needs
  a new versioned entry
- `extensions/builtin/themes/sforum-default/templates/not-found.html`
- `extensions/builtin/themes/sforum-default/assets/hybrid-forum.css`
- `extensions/builtin/themes/sforum-nocturne/templates/not-found.html`
- focused Go, Bun, validation, and browser tests
- OpenAPI only if the external resolve/error schema changes

Avoid touching files that are not necessary after M0 proves the smallest path.

## Milestones

### G0 - Close Current-HEAD Regression M7 And Release Shared Files

Do this first in the implementation session. It is a prerequisite gate, not a
reason to stop and wait for another task.

- [x] Re-read `2026-07-22-current-head-regression-remediation.md`, its latest
      handoff, and current dirty-worktree ownership.
- [x] Run every M7 focused test, race test, full Go test, Bun test, typecheck,
      build, OpenAPI validation, repository gate, and browser smoke exactly as
      listed in that book.
- [x] Fix failures only when they belong to the regression book's existing
      M0-M6 scope; preserve unrelated user work and do not absorb new product
      programs.
- [x] Update its search/frontend module notes, knowledge index, plan status,
      and completion handoff with exact evidence.
- [x] Mark the regression plan completed only when every M7 exit is genuinely
      green or an environment-only skip has exact unaffected substitute proof.
- [x] Record that Page Registry/error files are released to this focused 404
      book.

Exit: regression M7 is closed and the shared files have an explicit owner. If a
real failure outside the regression plan blocks the gate, record the exact
failure and obtain direction instead of weakening tests or silently expanding
scope.

### M0 - Ownership, Baseline, And Contract Freeze

- [x] Record branch, HEAD, `git status --short`, and overlapping user changes.
- [x] Confirm G0 closed regression-remediation M7 or record an equivalent
      explicit handoff of every overlapping file before production edits.
- [x] Reproduce one healthy topic and one missing topic through API, hard SSR,
      client navigation, DOM markers, headers, payload, and console.
- [x] Inventory every current public resource `ErrCorePageDataNotFound` producer.
- [x] Freeze the semantic-error classifier using existing HTTP status/envelope
      fields; do not invent a parallel error protocol.
- [x] Freeze the request-local 404 context and reviewed body-island/component
      IDs against the real Component Catalog.
- [x] Confirm no dependency is needed.

Exit: ownership is explicit, baseline evidence is recorded, and the component /
error contract is unambiguous. If the gate is closed, stop after read-only M0.

### M1 - Semantic Error And Retry Classification

- [x] Add focused helpers that distinguish semantic 404, retryable transient
      failure, and non-retryable technical failure.
- [x] Ensure 4xx Page Registry responses are attempted exactly once.
- [x] Preserve the original reason/status instead of flattening it to
      `transport_unavailable`.
- [x] Make delayed SSR retries Nuxt-context-safe without a global request store.
- [x] Add tests for 404/no retry, transient success on retry, retry exhaustion,
      and preserved error identity.

Exit: semantic 404 cannot become Core fallback merely because the resolver
returned a non-200 response.

### M2 - Route Error Handoff And HTTP Policy

- [x] Map reviewed public resource-not-found resolve errors to a sanitized Nuxt
      404 before the resource body island mounts or performs duplicate reads.
- [x] Preserve login/permission behavior; do not turn every 401/403 into 404.
- [x] Enforce 404, `no-store`, disabled Nitro SWR, and `noindex,nofollow` for
      SSR document and payload.
- [x] Remove success canonical/structured-data output from the error document.
- [x] Prove client navigation reaches the same error boundary and can navigate
      home/back without stale page state.

Exit: missing resource requests are genuine private 404 documents, not inline
200 cards.

### M3 - Theme-Owned 404 Chrome

- [x] Introduce the request-local Host 404 context and dedicated body island.
- [x] Keep status/copy/actions Host-owned and disclosure-safe.
- [x] Change `system.component.not_found` from a whole-page Host slot to the
      reviewed focused body-island contract.
- [x] Update default `not-found.html` to mount selected-theme navbar, normal
      public shell/body, 404 island, and footer without duplication.
- [x] Reuse `SFHomeNavigation` data/presentation for the default desktop left
      sidebar; do not clone it.
- [x] Update Nocturne with its own theme-consistent L1 structure.
- [x] Keep a complete no-recursion Core emergency page for theme failure.
- [x] Ensure error templates are L0/L1 only and cannot execute public L2.

Exit: normal business 404 uses the active theme's real chrome; forced theme
failure still renders an usable local emergency page.

### M4 - Resource Coverage And Security

- [x] Cover topic, category, tag, profile, and unmatched-route cases from the
      matrix.
- [x] Cover hidden/deleted resources for guest and privileged actors without
      leaking their existence in public copy.
- [x] Verify healthy resources still render their original selected-theme L1.
- [x] Verify technical resolve failure still selects Core fallback.
- [x] Add allowed/denied tests where authorization participates in visibility.

Exit: every listed public resource uses one consistent semantic 404 flow and no
permission boundary regresses.

### M5 - Browser And Production-Path Verification

Required desktop checks for default theme and Nocturne:

- [x] hard-load a healthy resource;
- [x] click from the homepage into a healthy resource;
- [x] hard-load each missing resource type;
- [x] navigate client-side to a missing resource;
- [x] use home/back actions and confirm error state clears;
- [x] compare navbar/sidebar/footer markers and geometry with a healthy page;
- [x] verify desktop plus one mobile viewport, light/dark, zh-CN/en-US;
- [x] verify page identity, nonblank SSR, no framework overlay, console health,
      screenshot evidence, and at least one real interaction;
- [x] force Page Registry/theme failure and verify emergency Core without
      recursive errors.

Header checks:

```bash
curl -i -sS -H 'Accept: text/html' \
  'http://127.0.0.1:3000/t/<missing-id>/<slug>'
```

Expected: `404`, `Cache-Control: no-store`, `noindex,nofollow`, selected-theme
L1 markers for a healthy theme, no successful-topic structured data.

### M6 - Full Gate, Documentation, And Handoff

Run the focused tests first, then:

```bash
git diff --check

cd apps/api
go test ./app/Models/PageViewModels ./app/Http/Controllers/Pages
go test ./...

cd ../web
bun test
bun run typecheck
bun run build

cd ../..
ruby scripts/validate-openapi-refs.rb
./scripts/test.sh
```

- [x] Update `knowledge/modules/frontend.md` with final behavior and evidence.
- [x] Update the broader system-error task book so its 404 milestones consume,
      rather than duplicate, this completed slice.
- [x] Update `knowledge/plans/README.md` and `knowledge/index.md`.
- [x] Replace the planning handoff with a completion handoff containing exact
      commands, browser routes, headers, and remaining risks.

Exit: all in-scope gates are green, the unrelated full-gate failure is recorded
without weakening its contract, public 404 behavior is documented, and no
broader 403/429/5xx completion claim is made.

## Acceptance Checklist

- [x] Ordinary resource 404 never renders `data-provider="core"` while the
      selected theme is healthy.
- [x] Default-theme 404 navbar/sidebar/footer match normal default-theme public
      pages and do not duplicate.
- [x] Nocturne keeps its own chrome rather than inheriting default-theme layout.
- [x] Hard navigation returns 404; no missing resource returns cacheable 200.
- [x] Semantic 404 is not retried and is not labeled transport failure.
- [x] Hidden/deleted/private resources share generic public copy.
- [x] Healthy resources retain selected-theme L1 and current performance.
- [x] Forced theme/runtime failure retains a bounded Core emergency page.
- [x] No new dependency, setting, permission, or public L2 execution is added.
- [x] Focused tests, typecheck, build, and browser matrix pass with exact
      evidence; the full Bun/repository gate reaches only the unrelated stale
      `prebuiltSettingsComponent.test.ts` asset-path assertion recorded below.

## Completion Evidence (2026-07-23)

- Focused web suites: `50 pass, 0 fail, 251 expect()` across Page Outlet,
  Page Resolve, and default-theme navbar behavior. An earlier wider focused run
  also passed `75` tests.
- `cd apps/web && bun run typecheck`: passed.
- `cd apps/web && bun run build`: passed immediately before the final
  HMR/loading-state guard.
- `cd apps/api && go test ./...`: passed; focused Page ViewModel/controller
  coverage passed in the same implementation cycle.
- `ruby scripts/validate-openapi-refs.rb`: passed, `2165` references across
  `54` files.
- `git diff --check`: passed before handoff.
- Full Bun/repository gate: `542 pass, 1 fail`; the only failure is the
  pre-existing `apps/web/tests/prebuiltSettingsComponent.test.ts` assertion
  for the obsolete `/_sforum/private-assets/extensions/...` path. Production
  intentionally uses the trusted digest-bound admin endpoint; this 404 task did
  not weaken or alter that contract.
- HTTP/cache/SEO probes returned real `404`, `Cache-Control: no-store`, and
  `noindex,nofollow`, with no success canonical or JSON-LD on the error
  document.
- Browser verification covered healthy and missing public resources, default
  and Nocturne themes, desktop/mobile, light/dark, and `zh-CN`/`en-US`.
  Theme navbar/sidebar/body/footer persisted after hydration, logged-in chrome
  stayed correct, home/back cleared the error state, and forced runtime failure
  retained the bounded Core emergency page. A later user report found an HMR
  component-alias warning and a transient unresolved Core shell; the final
  change uses exact `SF*` component imports, a resolve-pending skeleton, and
  client-only Reka dropdowns with stable SSR placeholders.
  Per user direction, that small final guard was not followed by another
  browser/build cycle.
- No dependency, operator setting, permission, API schema, 403/429/5xx behavior,
  or public L2 execution was added.

## Original Implementation Prompt

Use a fresh task and start with:

> Implement `knowledge/plans/2026-07-22-theme-consistent-public-resource-404.md`
> milestone by milestone. Start at G0 by closing the current-head regression
> M7 gate, then execute M0-M6. Respect the dirty worktree, keep the feature
> scope to public resource 404s, and do not stop until the required tests,
> browser evidence, and knowledge handoff are complete.
