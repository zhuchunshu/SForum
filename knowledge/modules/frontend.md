# Frontend Module

## Purpose

Owns the Nuxt application, SSR shells, routing, metadata, browser interactions,
admin UI, reusable `SF*` components, and runtime theme presentation bridge.
Business rules, persistence, authorization, and search indexing remain API
responsibilities.

## Stack And Boundaries

- Nuxt 4, Vue 3, Nuxt UI 4, Tailwind CSS, and Bun.
- SSR-first public pages; `zh-CN` default and `en-US` secondary locale.
- Public pages render typed Fiber API read models. Nuxt server middleware may
  proxy same-origin/SSR requests but does not own domain policy.
- Same-origin API proxy retries failed `GET`/`HEAD` requests at most nine times
  after the initial attempt (500 ms apart) so an API Air reload does not drop
  immutable theme assets. Unsafe requests are never retried.
- Host resource delivery keeps public immutable theme bytes under
  `/_sforum/assets`; trusted admin component bytes load through the
  digest-bound `/api/v1/admin/extensions/{id}/frontend/assets/{digest}/{asset}`
  endpoint.
- User-visible copy uses i18n keys. Public web options contain only frontend-
  safe state; secret metadata and writes use admin-only endpoints.
- Build/typecheck use `.nuxt-build` and `.nuxt-typecheck` so they do not disturb
  the user's port-3000 dev server.

## Active Work

### Current HEAD regression remediation

Plan: `../plans/2026-07-22-current-head-regression-remediation.md`.

- Status: closed 2026-07-23. Search/frontend/Page Registry and extension-gate
  regressions from the current-HEAD book are remediated.
- `forum.topic.reply` has production Page ViewModel/controller resolve coverage,
  not only catalog/template assertions.
- Query-bearing `forum.home` and replaceable `system.not_found` remain selected
  theme surfaces; Core remains a bounded emergency fallback.
- Shared Page Registry and error-flow files are released to the focused
  selected-theme public 404 task.

### Theme-consistent public resource 404

Plan: `../plans/2026-07-22-theme-consistent-public-resource-404.md` (**completed
2026-07-23**).

- Missing/hidden/deleted public resources and unknown routes resolve
  `system.not_found` while retaining the selected theme's navbar, responsive
  sidebar/body structure, and footer.
- Semantic 404 is attempted once and preserves its reason. Hard SSR returns
  404 with `no-store` and `noindex,nofollow`, without success canonical or
  structured data.
- `SFSystemThemeTemplate` renders reviewed L0/L1 error AST through statically
  compiled Host islands so production SSR and hydration agree. Core is a
  complete local emergency page only for actual theme/runtime/API failure.
- The final 2026-07-23 gate passed all Bun tests (`546/546`, `3303` assertions),
  Nuxt typecheck and production build, full Go tests, OpenAPI validation,
  repository validators, HTTP/cache probes, and the development/production
  desktop/mobile browser matrix.
- A 2026-07-23 follow-up removed the remaining hard-refresh half-state: the
  error boundary now fetches L1/L0/options/session through the explicit SSR
  internal API path, stages exact-artifact L0 and L1 candidates, and commits
  them together only when extension/version/digest/node revision all agree.
  Any request, identity, revision, or renderer-source failure commits a complete
  Core emergency payload and clears theme identity plus CSS links. Hydration
  reuses that serialized decision without a second startup fetch.
- Error-boundary Host islands use direct component imports. General Page
  Registry islands use explicit `defineAsyncComponent(() => import(...))`
  wrappers rather than runtime `resolveComponent('Lazy...')` registry lookups,
  so HMR cannot hand `undefined` to vnode creation.
- Nuxt payload extraction is disabled. Request-local `no-store` policy and
  Nuxt 4.4 extracted payload routing can otherwise turn `_payload.json` into an
  HTML response and cancel client navigation. Inline SSR payload keeps
  development and production behavior aligned without a second cache surface.
- Default-theme 404 always keeps its footer: current source artifacts scope the
  homepage footer suppression away from `.sf-page--not-found`, and Host CSS
  provides the same compatibility for installed older artifacts. In
  development, only the Nuxt overlay entry attached to an ordinary body-marked
  404 is hidden; after successful home/back/retry recovery its resolved entry
  is removed. Real runtime and 5xx diagnostics remain visible while active.
- Handoff:
  `../sessions/2026-07-22-theme-consistent-public-resource-404-plan-handoff.md`.

### Theme-defined system error pages

Plan: `../plans/2026-07-22-theme-defined-system-error-pages.md`
(**completed**). Completion handoff:
`../sessions/2026-07-23-theme-defined-system-error-pages-completion-handoff.md`.

- `error.vue` maps 403, 404, 429, and 500/502/503/504 to virtual Page Registry
  surfaces (`system.forbidden`, `system.not_found`, `system.rate_limited`,
  `system.server_error`). Other statuses stay on the generic Host fallback;
  browser 401 remains login/redirect behavior.
- `useSystemErrorPageResolve` performs one short attempt
  (800 ms dev / 1000 ms prod) and `useSystemErrorPagePresentation` commits L0
  skin links plus L1 render output only after exact extension/version/package
  digest/node revision identity matches. Transport/options/auth/artifact
  failure clears active theme identity and renders the Core emergency page.
- Reviewed Host islands own safe semantics and recovery behavior: details,
  actions, recovery search, sidebar, and rail. Theme L1 owns layout/chrome;
  plugins and public L2 remain closed on `system.*`.
- Default and Nocturne built-in themes now provide themed forbidden,
  not-found, rate-limited, and server-error templates/styles. A final default
  theme mobile CSS fix collapses the hidden-sidebar system-error layout to one
  full-width column below 960 px.
- Focused browser QA covered real unknown-route 404, selected-theme markers,
  status preservation, recovery search, dark mode, mobile, and English locale.
  Current QA data has no natural full-page 429/5xx producer, so those families
  are covered by mapping/resolver/fallback tests rather than committed
  test-only routes.

## Runtime Themes And Page Registry

- Nuxt remains the SSR host. Themes are buildless runtime packages; they do not
  extend Nuxt or replace the deployment artifact.
- Stable Page Registry IDs select page providers and ViewModel contracts.
- L0 supplies CSS/tokens/fonts/images/locales. L1 supplies reviewed
  server-rendered templates. Trusted author-prebuilt L2 is optional,
  digest-authorized, contract-bound, and production-default off through
  `SFORUM_V3_PUBLIC_L2`.
- Public body content, metadata, canonical links, structured data, links, and
  pagination must be complete in initial HTML without L2 or hydration.
- Dynamic add routes enter through `pages/[...sfRegistryPage].vue` and
  `/pages/resolve-path`. Plugin page loader data is SSR-only.
- `SFPageOutlet` and `SFHostPublicChrome` are Host emergency surfaces. Component
  failure preserves L1/SSR content and may quarantine only the failing L2.
- Public chrome islands such as `SFNavbar` and `SFFooter` must stay statically
  reachable from runtime theme templates so critical scoped CSS is present
  before browser back/forward restores the page.
- Activation prewarms and atomically swaps exact theme runtime state. Active
  skin links are emitted by root `useHead` during SSR.

Architecture sources:

- `../decisions/2026-07-13-runtime-page-registry-themes.md`
- `../decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`
- `../plans/2026-07-13-trusted-plugin-theme-platform-v3.md`
- `../../docs/extensions/page-catalog.md`

## Resolve And Cache Invariants

- Page resolve state is keyed by page ID, path, query, locale, and actor. Never
  reuse rendered output across those boundaries.
- Transient `/pages/resolve` failure may use one bounded retry. Semantic 404
  must not retry.
- Any fail-closed/runtime fallback response disables Nitro cache/SWR and sets
  `Cache-Control: no-store` so anonymous cache cannot share Core emergency
  HTML.
- Resolve payloads preserve provider/artifact/revision identity and distinguish
  authoritative Core ownership from transient runtime failure.
- Last-good theme skin/settings cache is allowed only for the exact
  `extensionId + packageDigest + nodeRevision`, with a short TTL and validated
  same-origin asset URLs. A known theme change cannot reuse prior-theme state.
- Theme asset URLs place `packageDigest` in the path rather than a query so CSS
  relative fonts/images retain the same immutable identity.
- Anonymous public pages may be shared/SWR cached. Requests carrying
  `sforum_session` restore auth during SSR and must disable HTML/payload cache.
  Browser `onMounted` still revalidates session state.
- Public contributions controlled by extension settings vary topic SWR through
  `site.public_surface_revision`.

## Public Forum UI

- The default theme uses a flat, responsive three-column shell: navigation,
  primary content, and contextual right rail. It collapses the right rail
  before the left navigation.
- Native scrollbars are themed globally from `main.css` with neutral
  `--sf-scrollbar-*` tokens derived from public surface, text-muted, and border
  colors. Keep them quiet rather than accent-colored; page-local scroll
  containers should use the global contract unless they deliberately use
  `.no-scrollbar`.
- `/categories` uses the default-theme grouped directory surface: real
  `ForumCategoryGroup`/`ForumCategory` DTOs, URL-backed group focus via
  `?group=<group.id>`, page-local category filtering, group-local sorting, and
  a derived right rail. Topbar `SFSearch` remains global forum search.
- The notifications page follows the same default-theme shell and shared
  mobile drawer state as the forum homepage/navbar. Its filters are client-side
  over loaded notifications only; the global unread total remains API-owned.
  Desktop-hidden right rails must explicitly opt back into display inside a
  mobile drawer.
- `forum.topic.create` now uses the same default-theme three-column native flow:
  left rail reuses `SFHomeNavigation`, the center column keeps the real
  `SFTopicComposerPage` form and `SFEditor`, and the right rail derives publish
  summary/settings/checks from live form state. Desktop uses a fixed bottom
  publish dock with extra content padding; mobile collapses to a single column
  and keeps category, tags, draft, errors, and publish controls available.
- `SFHomeNavigation` owns shared public left-rail links and footer links. Its
  host CSS must style the footer too, because homepage, topic, and taxonomy
  pages reuse the component inside desktop sidebars and mobile left drawers.
- Default-theme public pages must still render the global `SFFooter` from L1
  chrome; the left-rail legal links are navigation shortcuts, not a replacement
  for the site footer.
- On desktop, the default-theme full-width three-column shell fixes the navbar,
  left navigation, and right rail in the viewport; only the center content
  column scrolls. The rail divider lines touch the topbar and viewport bottom
  directly; spacing belongs inside each column, not on the outer layout. Mobile
  keeps ordinary document scrolling plus drawer scrolling to avoid scroll traps.
- Homepage, topic detail, and notifications place the public footer inside the
  center content column through `SFContentColumnFooter`; full-width L1 footer is
  hidden for `fullwidth-3col` shells so the footer scrolls with content instead
  of occupying global chrome.
- `/tags` is the `forum.tag.index` Page Registry body island. It renders the
  default-theme three-column heat overview from real `listTags` and
  `listCategoryGroups` API data, with all/hot/week/A-Z filters, localized empty
  states, and the shared navbar mobile drawers for left navigation and tag
  context.
- The `/moderation` Host body island mirrors that flat shell with scoped
  `sforum-moderation` CSS instead of changing default-theme shared CSS; its
  source/filter/page/review state is query-backed for SSR/client recovery, and
  review actions move into mobile drawers as soon as the right rail collapses.
- Homepage/topic/taxonomy data is real API state. Do not render fabricated
  likes, bookmarks, unread state, participant stacks, or ranking.
- `/u/{username}` is also part of the default-theme three-column public shell:
  left rail reuses `SFHomeNavigation`, center renders member summary plus
  locale-aware daily public activity groups, and right rail renders real public
  profile details, public stats, and recent topics. The same rail is reused in
  the mobile information drawer. Keep social/portfolio/gamified actions out
  until the API owns those contracts. Public-reply activity links use
  `<NuxtLink :to>` (not `<a href>`) so in-app navigation preserves the
  `#comment-{id}` hash and the topic page can resolve the page client-side;
  same-page comment anchors elsewhere stay native `<a href="#comment-{id}">`.
- Homepage and eligible lists SSR page 1 and continue with keyset infinite
  scroll. URL-backed filters and stale-response guards remain authoritative.
- Topic detail ships complete topic/comments/navigation in initial SSR HTML;
  client navigation may let secondary data fill through explicit pending
  states.
- Reply editor mounts after explicit interaction; advanced reply uses
  `forum.topic.reply` and draft handoff.
- Topic editing is a standalone page `forum.topic.edit` at
  `/topics/:topicId/edit` (id-based because editing may change the slug);
  `SFTopicShowPage` only navigates there — the old `?edit=1` inline mode is
  removed. Save returns to `forumTopicPath`; canonical redirect fixes slug.
- Topic create draft persistence is currently local `sessionStorage` under the
  composer page because there is no create-topic draft API. Publishing still
  uses the canonical `POST /topics` flow and keeps API authorization
  authoritative.
- `SFAvatar` is the required first-party avatar renderer. Pass shared
  `AvatarView`; do not fork initials/URL handling.
- Comment tree presentation has bounded indentation; mobile clears recursive
  inset. Visible floor labels are list positions while anchors remain stable
  `#comment-<id>` targets. Cross-page anchors resolve with zero flash: when
  the URL carries `#comment-<id>` and no explicit page, `SFTopicShowPage`
  resolves the page server-side (`resolveCommentPage` API), SSR-renders that
  page, and lets the browser scroll natively to the target; a client-side
  watch also scrolls on client navigations. Comment pagination uses a path
  segment `/page/N` (old `?page=N` still resolves and is normalized
  client-side after hydration).
- `.sf-prose` is the shared sanitized rich-content surface. Code blocks use the
  client highlight directive plus a no-op SSR directive registration.
- Runtime appearance owns accent and dark-mode tokens. The default theme must
  not pin a product-specific accent.

## Editor And Content UI

- `SFEditor` uses Tiptap while preserving Markdown `v-model` integration. It
  emits HTML, Markdown, native JSON, plain text, counts, and empty state.
- Client HTML is preview-only; the API regenerates and sanitizes stored output.
- Topic/comment writes send the backend `content` contract, including optional
  attachment IDs.
- Tag input accepts Unicode letters/numbers plus hyphens, matching backend tag
  validation.

## Admin And Trusted Frontend Runtime

- `SFExtensionSettingsRenderer` is the normal plugin/theme path and renders
  Schema forms, tabs, groups, columns, callouts, and allowlisted actions without
  extension JavaScript.
- Complex settings may load an authenticated immutable `.mjs`/`.css`
  micro-frontend only after whole-artifact trust. It is client-only and falls
  back to Schema UI on import/API/mount/CSS/cleanup/quarantine failure.
- There is no runtime SFC compiler, extension dependency installer, Nuxt build
  supervisor, host-peer resolver, or frontend release supervisor.
- Admin route middleware must stay narrow and avoid component-context-only
  composables such as direct `useI18n()` calls.

## Request And SSR Regression Rules

### CSRF

Cookie-authenticated unsafe calls go through `useApiClient().request`. It primes
the CSRF cookie when absent, adds `X-Csrf-Token`, and retries once after
`csrf.invalid`. Do not use raw `$fetch` for those writes.

### Authentication And SWR

Never serialize user auth into shared anonymous public payloads. Session-bearing
SSR restores auth before render and disables cache; protected/admin middleware
remains a UX guard while API policy is authoritative.

### Rendered-content directives

Every directive used by SSR templates needs a server registration. Keep the
real highlight.js client plugin and no-op `highlight.server.ts` with
`getSSRProps`.

## Important Paths

| Path | Responsibility |
| --- | --- |
| `apps/web/app/app.vue` | Root startup, SSR head, session/theme boot |
| `apps/web/app/components` | Uppercase `SF*` component library |
| `apps/web/app/composables` | API, auth, options, Page Registry clients |
| `apps/web/app/pages` | Public and admin route shells |
| `apps/web/app/middleware` | Route/cache/session behavior |
| `apps/web/app/plugins` | Client/server integrations and directives |
| `apps/web/app/assets/css` | Host component/theme baselines |
| `extensions/builtin/themes/sforum-default` | Protected default L0/L1 package |

## Verification

- Unit tests: `cd apps/web && bun test`
- Typecheck: `cd apps/web && bun run typecheck`
- Build: `cd apps/web && bun run build`
- Use the in-app browser for SSR/interaction/responsive checks. Do not kill the
  user's port-3000 dev server.
- For auth/cache/Page Registry changes, verify hard refresh, client navigation,
  authenticated public SSR, anonymous SSR, response status/cache headers,
  console output, and no-JavaScript content where relevant.

## Next Steps

1. Keep theme-defined system error pages SSR-complete, permission-aware,
   cache-safe, localized, and covered whenever new browser-facing error
   producers are added.
2. Keep new pages SSR-complete, permission-aware, cache-safe, localized, and
   registered through stable Page Registry/component contracts.

## Open Questions

- Whether `en-US` copy must be complete for first public release.
- Which remaining `SF*` wrappers provide enough shared behavior to justify
  wrapping Nuxt UI primitives.
