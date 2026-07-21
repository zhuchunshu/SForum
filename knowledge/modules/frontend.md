# Frontend Module

## Accepted V3 Target (P0–P12 Complete; P13 LTS Residual)

The accepted target, including the canonical template comparison and detailed
architecture mind map, is documented in
`../decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`; its phased task
book is `../plans/2026-07-13-trusted-plugin-theme-platform-v3.md`. Progress:
`../plans/2026-07-13-trusted-plugin-theme-platform-v3-progress.md` (**~99.7%**).

**Live program state (2026-07-21):** P0–P12 closed. Public presentation is thin
Nuxt shells + Host body islands + theme L1 (`data-theme-owned=presentation`,
`sf-navbar`/`sf-footer`). Trusted public L2 is implemented and production-
default **off** via `SFORUM_V3_PUBLIC_L2` (not “missing”). Admin registry-
catalogs UI and editor L2 load paths are gated in `scripts/test.sh`. Remaining
V3 residual is LTS-blocked loader/Protocol V1 deletion only.

The remainder of this module note describes the current frontend unless a
section is explicitly labeled as target behavior.

- Themes become complete buildless public view packages compiled with Go
  `html/template` at activation into immutable, digest-keyed runtime snapshots.
  The contract includes `if`, `range`, `with`, `template`, and `block` with a
  restricted FuncMap.
- Nuxt remains the SSR shell and typed host-island runtime. Primary page content,
  metadata, canonical links, and structured data are present in the initial
  HTML response and never depend on L2 or client hydration.
- Trusted public L2 uses author-prebuilt package-local ESM after exact-artifact
  confirmation; operators do not install dependencies or build extension code.
- Component Registry supports add, before, after, wrap, replace, hide, and
  prop/result filters. Themes may override plugin templates through versioned
  template ids and deterministic conflict rules, but cannot alter plugin
  business data contracts.
- Admin Surface Registry covers navigation, dashboards, lists, filters,
  row/bulk actions, forms, notices, editor/detail regions, import, and export.
  Navigation/Region Registry covers public menus, breadcrumbs, headers, footers,
  sidebars, and theme-defined widget regions.
- L2/component failure preserves the SSR/L1 fallback and can quarantine the
  failing component without breaking navigation or indexable content.
- With JavaScript disabled, body content, lists, comments, links, and pagination
  remain complete and usable; only interactive L2 enhancement is absent.

P1 uses one shared Host-owned impact dialog for plugin and theme enable flows.
It presents every canonical impact category, keeps blocking errors visible,
allows delegated managers to preview without executing, and requires an active
`super_admin` to issue and consume the exact-artifact challenge. Success Toasts
follow the active appearance tokens and dismiss after 10 seconds. Prebuilt admin
frontend loading now requires the whole-artifact grant; legacy frontend-only
grants do not satisfy V3 trust.

## Trusted Admin Runtime

There is one full-trust client-only extension path:

- Admin Micro-frontend API v1 (preferred for complex settings): author-prebuilt
  package-local `.mjs`/`.css`, authenticated immutable digest URL, explicit
  actor-bound confirmation, version/API/component/digest grant, framework-
  neutral `mount(target, bridge)`, cleanup, and Schema fallback. No Nuxt build.

`SFExtensionSettingsRenderer` is the normal plugin/theme path and handles
Schema form/tabs/groups/columns/callouts plus Settings Actions without author
JavaScript. Trusted component code is client-only; SSR renders host metadata
and fallback. Error boundaries and third-failure session quarantine prevent one
component from breaking navigation or other admin pages.

## Purpose

Owns the Nuxt web application, SSR pages, UI composition, frontend routing,
metadata, and browser-side interactions.

## Runtime themes / Page Registry (live)

Public theming is runtime Page Registry + L0/L1 + optional trusted L2
(security-remediated 2026-07-13; P9 L2 matrix closed 2026-07-21):

- **Page Registry** + stable page ids (`forum.home`, …)
- **L0** CSS/assets without rebuild
- **L1** sandboxed templates composing host SF islands; theme owns presentation
  chrome (`sf-navbar`/`sf-footer`) on replaceable pages
- **L2** author-prebuilt package-local ESM after digest-bound trust; production
  default **off** (`SFORUM_V3_PUBLIC_L2`); fail-closed quarantine on mismatch
- Host stays Nuxt; public themes are not full Nuxt apps and do not rebuild Nitro
- Dynamic add routes: `pages/[...sfRegistryPage].vue` → `GET /pages/resolve-path`
- Plugin page data: **SSR-only** via API `loaderData` (no client plugin route fetch)
- Fail-closed `SFPageOutlet` / `SFHostPublicChrome` retained forever

**ADR:** `../decisions/2026-07-13-runtime-page-registry-themes.md`  
**Plan:** `../plans/2026-07-13-runtime-page-registry-themes.md`  
**Round-2:** `../sessions/2026-07-13-runtime-page-registry-round2-remediation.md`  
**Page catalog:** `../../docs/extensions/page-catalog.md`

## Current Status

Foundation scaffold exists under `apps/web`.
The web container now passes `APP_URL` into Nuxt, and Nuxt uses it for the
site config and Nuxt i18n SEO `baseUrl`.
Generated output directories are ignored by Nuxt/Vite development watchers, and
`bun run build`/`bun run typecheck` use sibling Nuxt temporary directories
(`.nuxt-build` and `.nuxt-typecheck`) so they do not disturb the active dev
server state.
Nuxt DevTools is opt-in with `NUXT_DEVTOOLS=true`; the default dev path avoids
its dependency scan and resident runtime cost. Development also disables Nuxt
payload extraction so HMR cannot leave stale SWR `/_payload.json` responses
that change Page Registry ownership during client navigation; production keeps
payload extraction enabled. `SFThemeTemplate` resolves page-level Host islands
through Nuxt lazy components, while the always-present navbar/footer stay
eager. Tag detail SSR normalizes every AsyncData list, starts its three
independent reads concurrently, and keeps `.length` access out of the render
function so HMR transitions cannot reject SSR.
`bun run preview` starts `scripts/preview.mjs`, prints an SForum Web Preview
startup banner, then imports the generated Nitro server entry at
`.output/server/index.mjs`; this keeps local preview aligned with the root
`.env` API target while avoiding edits to generated output. The installed
`nuxi preview` command has no `--host` flag, and
`nuxt preview --host 0.0.0.0` treats `0.0.0.0` as `ROOTDIR`.
During development startup and API hot reloads, the global site-options read
uses a short timeout and falls back to local defaults so SSR can render the page
while the API process is still compiling.
App startup keeps anonymous SSR cache-safe while avoiding a delayed login-state
swap for authenticated visitors. Requests carrying `sforum_session` restore
`/auth/session` during SSR, and `public-session-cache.ts` marks their HTML and
Nuxt payload responses `no-store` with Nitro cache/SWR disabled. Anonymous
public pages remain share-cacheable and restore auth in browser `onMounted`.
Transient API failures mark auth as temporarily unavailable without clearing
the cached user state.
Guest middleware passes its `to.query.redirect` into
`useAuthReturnNavigation`; the composable calls `useRoute()` only for component
callers, matching Nuxt's middleware route contract.
Nuxt top-level ignores stay scoped to app-local generated output so Nuxt UI
components under `node_modules/@nuxt/ui/dist` are still auto-imported.
Nuxt UI remote font integration is disabled for now to avoid build-time network
provider retries while the theme uses local/system fonts.
The forum UI component library now lives in `apps/web/app/components/` with
uppercase `SF` component names. The first component set is backed by
`apps/web/app/assets/css/sforum-components.css` and previewed on the dev-only
`/components` route. That preview page now shows the components in expanded
forum scenarios: publishing, moderation, member profile, feedback, lists, and
state handling.
`SFAvatar` is the required first-party user avatar renderer. Real user chrome,
topic lists, topic detail authors, and comments should pass `AvatarView`
through `:avatar` instead of using `UAvatar`, hand-written initials, or raw URL
props. Name-only `SFAvatar` usage is kept for component demos and generic
fallback examples.
`SFEditor` is now backed by Tiptap rather than a plain textarea. It keeps the
existing `v-model` as Markdown for simple parent integration, emits a
`content-change` payload containing HTML, Markdown, native Tiptap JSON, text,
character count, word count, and empty state, and includes toolbar controls,
custom emoji nodes, preview, Markdown source, and JSON inspection modes. The
client HTML is only for preview and must be regenerated/sanitized by the API
before storage.
The topic composer sends `SFEditor` Markdown through `useForumApi.createTopic`,
which wraps editor fields under the backend `content` contract. Composer tag
input accepts Unicode letters/numbers plus hyphens, matching backend tag slug
validation for Chinese tag names.
Public and admin Vue pages are owned by `apps/web`. Installable themes provide
runtime `theme.json`, templates, and assets through Page Registry/L0/L1; they do
not extend Nuxt or replace the host deployment artifact. Core keeps auth/session
logic, API clients, i18n catalogs, SEO helpers, permissions, and reusable
component/composable infrastructure.
The default-theme public forum uses a **full-width three-column flat shell**
(demo `tmp/demos/grok/forum-fullwidth-3col/`). Layering:

- **L1** page templates: shell + which host islands to mount
- **L0 tokens**: `--sf-public-*` only (no list-row BEM mirror in theme CSS)
- **Host `SF*`**: data + Tailwind presentation (`SFHomeTopicRow`, `SFAvatar`, …)
- **Component Registry / L2**: theme wrap/replace of registered targets for deep forks

Hooks: `data-sf-region="topic-list"`, `data-sf-component="forum.topic_list_row"`,
root class `sf-avatar`. After package edits in dev, rsync to
`storage/builtin-dev`, restart API (`SyncBuiltins` stages a new digest), then
super_admin theme activate with `approveCoreReplacements` so L1 bindings and
L0 skin track the new digest. Layout:
- Sticky topbar (`SFNavbar`): logo, Latest → `/`, Categories → `/categories`,
  Tags → `/tags` (hidden when `forum.tags.public_pages` is off), search,
  compose, session controls. Density ~52px.
- Public taxonomy list pages (default theme): `/tags` T01 weight cloud and
  `/categories` C04 grouped tile grid; styles in `sforum-taxonomy.css`.
- Homepage: full-bleed grid — sticky ~220px left nav (`SFHomeNavigation`) with
  compose, all topics, and category color dots + counts; main column notice +
  latest feed tab + dense topic table (`SFHomeTopicRow` without excerpt cards);
  optional sticky right rail (`SFHomeRightRail`). Author avatar column is honest
  (author only; no fabricated participant stacks).
- Infinite scroll still SSR-loads page 1 via topic/search APIs, hydrates through
  Nuxt `useState`, and appends with `IntersectionObserver`. URL-backed filters
  and stale-response guards are unchanged. Missing API capabilities (unread,
  ranking, mine-only feed, likes, bookmarks) are not rendered.
- Topic detail: same full-width three-column shell — left `SFHomeNavigation`
  (route mode), center article + comment tree + composer, sticky right
  `SFTopicSideCard` (status/category/replies/views, author as participant,
  tags). Share copies the URL; no fake like/bookmark. Comment stream remains
  tree/flat via `SFCommentStreamControls` (backend has no relevance sort).
  Collapse: hide right rail first (≤1180px), then left nav (≤960px).
  `SFTopicProgressRail` is retained in the theme package but no longer mounted
  on the default detail route.
Public surface tokens live in the theme layer (`sforum-default` skin + host
baseline `sforum-theme.css` etc.); `--sf-accent*` still come from runtime
appearance. Dark mode uses the existing `.dark` public variables.

`SFComment` has an explicit `presentation`, `depth`, and
`collapseFromDepth` contract. Tree mode renders one branch rail/inset on
desktop, collapses depth-two descendants once at the boundary, and preserves a
direct non-interactive reply reference. Mobile clears every recursive inset;
flat mode never recurses. Rich content containers, code, and images must remain
bounded so no comment depth can widen the document viewport.
Comment floor badges are display-only list positions (`#1`, `#2`, …) computed
from the comment list page/perPage, while anchors keep the real `comment.id`
target (`#comment-<id>`) for stable deep links. Do not render database IDs as
visible floor numbers.
Public themes no longer ship as Nuxt Layers selected at runtime. Host
`apps/web` owns public pages/components/layouts/CSS. Themes are L0/L1 packages
(`theme.json` + `assets/` + `templates/`) activated through the Page Registry
without rebuilding Nuxt or restarting Nitro.

- `bun run dev` starts Nuxt directly; production starts
  `.output/server/index.mjs` directly.
- Active theme skin URLs come from `GET /api/v1/site/active-theme/skin` and are
  emitted through root `useHead` during SSR, so the first HTML already includes
  the theme-assets stylesheets before hydration.
- Prebuilt settings components load only through the authenticated immutable API
  digest endpoint after trust. There is no runtime SFC compilation, admin
  registry, host-peer resolver, dev compose, release supervisor, or extension
  frontend dependency install.

## Regression Notes

### CSRF Token Bootstrapping

- Symptom to watch for: the first form submit on login, registration, password
  reset, or admin pages returns `csrf.invalid` even though the page was loaded
  from the same site.
- Known cause: SSR-side API reads may not deliver the backend `Set-Cookie`
  header to the browser before the first client-side unsafe request, and a
  long-lived page may hold an expired `csrf_` cookie.
- Safe pattern: route all unsafe API requests through `useApiClient().request`.
  It now primes `csrf_` with `GET /api/v1/health` when the browser has no token,
  adds `X-Csrf-Token`, and retries once after a backend `csrf.invalid` by
  refreshing the token. Do not bypass it with raw `$fetch` for cookie-authenticated
  writes.
- Required verification after touching CSRF/request plumbing: run
  `bun test tests/useApiClient.test.ts`, `bun run typecheck`, and the backend
  CSRF tests in `go test ./app/Http`.

### Auth Success Navigation And Route Middleware

- Symptom to watch for: login/register API returns success, the form stops
  submitting, but the page does not navigate and no user-facing error appears.
- Known cause: Nuxt route middleware can run outside the same component setup
  context as pages. Do not add `useI18n()` or other component-context-sensitive
  composables directly inside `apps/web/app/middleware/admin.ts`; doing so once
  blocked post-login navigation into the admin route.
- Safe pattern: keep route middleware narrow. Use `useLocalePath()`,
  `useAuthSession()`, and plain fallback text/errors there; put localized UI
  messaging in pages/layouts/components where i18n context is stable.
- Required verification after touching auth pages, `useAuthSession()`, or admin
  middleware: run `bun test tests/useApiClient.test.ts`, `bun run typecheck`,
  and browser-check both successful register and successful login navigation.

### Public SWR Pages And Auth State

- Symptom to watch for: after logging in, refreshing a public cached page such
  as `/` appears logged out, while visiting the admin area still shows the
  session correctly.
- Known cause: public routeRules such as `swr: 600` can serve a cached Nuxt
  payload. Never write `auth:user` into root-app SSR payload on public pages,
  or a cached guest payload can hide a valid browser session and a cached user
  payload risks leaking user-specific chrome.
- Safe pattern: anonymous public SSR refreshes only cache-safe global state.
  When the request has `sforum_session`, `public-session-cache.ts` must disable
  HTML/payload cache and SWR before `app.vue` restores auth during SSR. Browser
  `onMounted` still revalidates the session; protected/admin route middleware
  remains responsible for authorization.
- Required verification after touching app startup/auth cache behavior: run
  `bun test apps/web/tests/appStartup.test.ts apps/web/tests/useApiClient.test.ts
  apps/web/tests/protectedRouteRendering.test.ts apps/web/tests/adminRouteRendering.test.ts`,
  then browser-check a public cached page and an authenticated protected route
  when a logged-in browser session is available.

### SSR Directives For Rendered Content

- Symptom to watch for: SSR topic/comment pages fail with
  `Cannot read properties of undefined (reading 'getSSRProps')` from Vue
  server renderer.
- Known cause: a template uses a custom directive such as `v-highlight`, but
  the directive is registered only by a `.client.ts` plugin. SSR still compiles
  the directive usage and expects a server-side directive object.
- Safe pattern: keep `apps/web/app/plugins/highlight.client.ts` for real
  highlight.js DOM scanning and `apps/web/app/plugins/highlight.server.ts` as
  the no-op SSR placeholder with `getSSRProps`.
- Required verification after touching rendered-content directives: run
  `bun test tests/defaultThemeTopicPage.test.ts`, `bun run typecheck`, and an
  SSR smoke request such as `curl -sS --max-time 20 -o /tmp/sforum-topic.html
  -w '%{http_code} %{size_download}\n' http://127.0.0.1:3000/t/5999`.

## Planned Stack

- Bun for package management and scripts.
- Nuxt 4 with Vue 3.
- Nuxt UI with Tailwind CSS.
- Nuxt i18n with `zh-CN` as default and `en-US` as the first secondary locale.
- Nuxt SSR by default for public pages.
- `@nuxtjs/seo` for sitemap, robots, structured data helpers, and social
  metadata helpers.
- Vitest and Playwright once UI exists.

## Planned Boundaries

- Pages render forum read models from the Fiber API.
- Nuxt server routes may proxy API calls for SSR and same-origin cookies.
- Nuxt must not own forum business logic, persistence, authorization, or search
  indexing.
- Nuxt owns UI translation catalogs, localized routes, and localized metadata.
- User-facing strings must use translation keys instead of inline prose.

## Open Questions

- Final visual direction and density for forum pages.
- Whether English translations are mandatory for MVP launch or can lag during
  internal development.
- Which `SF` components should wrap Nuxt UI primitives later, versus staying
  plain Vue/CSS components.
- Which role-management screens are required in the first admin milestone.

## Next Steps

- Expand the initial `zh-CN`/`en-US` catalogs as pages are added.
- Add page skeletons for home, category list, topic detail, login, and profile.
- Add protected admin pages under the configurable control-panel shell for user
  management, moderation, and audit.
- Keep `useWebOptions()` public state limited to frontend-safe settings; use
  admin-only endpoints for masked secret metadata and secret updates.
- Add SEO metadata conventions before real pages proliferate.
- Start replacing static forum page sketches with the reusable `SF*` components.

## Rich-Text Prose And Content Layout (2026-07-08)

Topic body and comment HTML previously rendered with an undefined `.sf-prose`
class, so headings, paragraphs, code blocks, lists, tables, and images fell
back to browser defaults and looked unstyled.

- Installed `@tailwindcss/typography` (dev dependency) and enabled it in
  `main.css` via `@plugin "@tailwindcss/typography";` (Tailwind v4 CSS-first
  syntax; the project has no `tailwind.config`).
- Defined `.sf-prose` in `main.css` as a themed wrapper around `@apply prose`:
  accent-colored links and inline code, dark-mode aware, bordered pre/img,
  accent left-border on blockquotes. The class is shared by the topic detail
  body and `SFComment` HTML.
- Both the topic detail page and the composer page moved from `max-w-3xl` /
  `max-w-4xl` centered single columns to `max-w-[1200px]` matching the navbar's
  `max-width: 1200px`, so page content aligns with the navbar edges on wide
  screens.
- Both pages now use a two-column grid `grid-cols-[1fr_300px]` on `lg+` with a
  `lg:sticky lg:top-20` sidebar. Topic detail sidebar shows author card and
  topic stats; composer sidebar shows writing tips and a Markdown cheat sheet.
  Below `lg` the sidebar hides and content stacks single-column.
- Composer inputs now reuse the global `.sf-input__control` classes and the
  `SFInput` component for the title field; the page-scoped `.sf-input` override
  that hardcoded hex colors (`#0f766e`, `#ffffff`, `#18181b`) was removed so
  inputs follow design tokens and theme switching. The category `<select>`
  stays native because `SFInput` does not support select.

Note: adding the `@plugin` directive to `main.css` requires a dev server
restart to take effect; Vite HMR does not reliably pick up new Tailwind
plugin dependencies. The compiled CSS was verified to contain all `prose`
  rules after restart.

## SEO Workbench V2 P0 (2026-07-11)

- `/control-panel/seo` now separates search appearance and content-type
  policies from the existing robots, Sitemap, schema, permalink, and
  verification settings.
- Homepage SEO title, description, keywords, SEO site name, and social image are
  independent from `site.name`; the page shows a live search-result preview.
- `SFSEOImagePicker` supports drag/drop, file selection, manual public URLs,
  upload progress, load validation, preview, replace, and remove. Uploads call
  the protected SEO asset endpoint and auto-fill the returned public URL.
- `seoResolver.ts` is the single pure metadata resolver. Public home, category,
  tag, topic, and profile pages provide typed contexts; privacy/moderation rules
  can only make indexing stricter.
- `seoStructuredData.ts` builds WebSite, Organization, CollectionPage,
  BreadcrumbList, DiscussionForumPosting, and ProfilePage-compatible graphs
  from the resolved public context.

## V3 P8 SSR And Crawler Checkpoint (2026-07-15)

- Public pagination is anchor-based on home, category, tag, and topic surfaces;
  canonical URLs preserve `page` while page one remains query-free.
- Category and tag detail routes stay SSR but use `cache: false`. Nuxt's payload
  URL omits the pagination query, so shared SWR could hydrate current HTML with
  another page or an old dataset. Caching may return only with a normalized
  query and theme-revision-aware key.
- `SFHomeNavigation` accepts omitted catalog props with safe empty defaults so
  legacy L1/plugin templates can render the host island without Vue warnings.
- In-app browser QA passed category page 2 -> 3 navigation, topic/comments,
  profile, and the plugin add page with no hydration warnings or framework
  overlay in a stable API window.
- JavaScript-disabled Playwright rendered complete home/list/topic/profile/
  plugin content and followed linked pagination. Baiduspider received title,
  content, links, canonical, robots, five hreflang links, and valid JSON-LD.
- The Web gate is 310/310 tests plus Nuxt typecheck and production build.
