# Frontend Module

## Trusted Admin Runtime

Admin extension components are client-only static imports generated into an
immutable Web Release. SSR renders validated metadata/placeholders only. The
host provides independent error boundaries, third-failure session quarantine,
extension-scoped localization/API/navigation/Toast capabilities, and a stale
admin-tab release monitor. Jobs is the first production consumer through its
column, row-action, and detail-section slots.

## Purpose

Owns the Nuxt web application, SSR pages, UI composition, frontend routing,
metadata, and browser-side interactions.

## Current Status

Foundation scaffold exists under `apps/web`.
The web container now passes `APP_URL` into Nuxt, and Nuxt uses it for the
site config and Nuxt i18n SEO `baseUrl`.
Generated output directories are ignored by Nuxt/Vite development watchers, and
`bun run build`/`bun run typecheck` use sibling Nuxt temporary directories
(`.nuxt-build` and `.nuxt-typecheck`) so they do not disturb the active dev
server state.
`bun run preview` starts `scripts/preview.mjs`, prints an SForum Web Preview
startup banner, then imports the generated Nitro server entry at
`.output/server/index.mjs`; this keeps local preview aligned with the root
`.env` API target while avoiding edits to generated output. The installed
`nuxi preview` command has no `--host` flag, and
`nuxt preview --host 0.0.0.0` treats `0.0.0.0` as `ROOTDIR`.
During development startup and API hot reloads, the global site-options read
uses a short timeout and falls back to local defaults so SSR can render the page
while the API process is still compiling.
App startup splits cache-safe SSR work from browser-only auth restoration:
SSR refreshes frontend-safe web options only, while browser `onMounted`
restores the current session from `/auth/session`. Transient API failures mark
auth as temporarily unavailable without clearing the cached user state.
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
Public, non-admin UI is now owned by the protected built-in default theme layer
at `extensions/builtin/themes/sforum-default/layer`. The root Nuxt app extends
that fallback layer and can prepend an uploaded Nuxt Layer during release builds
through `SFORUM_THEME_LAYER`. The layer owns the homepage, default layout, auth
layout/pages, public navbar/footer, and public/auth chrome CSS. Core keeps admin
pages/layout, auth/session logic, API clients, i18n catalogs, SEO helpers,
permissions, and reusable component/composable infrastructure.
The default-theme public forum follows V32 暖橙左栏 (demo
`tmp/demo/grok/1/v32-right-sidebar/`, content is left-nav despite the folder
name). Layout:
- Sticky topbar (`SFNavbar`): logo, Latest → `/`, Categories → `/categories`,
  Tags → `/tags` (hidden when `forum.tags.public_pages` is off), search,
  compose, session controls. Density ~52px.
- Public taxonomy list pages (default theme): `/tags` T01 weight cloud and
  `/categories` C04 grouped tile grid; styles in `sforum-taxonomy.css`.
- Homepage: sticky 240px left nav (`SFHomeNavigation`) with compose, all
  topics, and category color dots + counts; main column notice + latest feed
  tab + dense topic table (`SFHomeTopicRow` without excerpt cards). Author
  avatar column is honest (author only; no fabricated participant stacks).
- Infinite scroll still SSR-loads page 1 via topic/search APIs, hydrates through
  Nuxt `useState`, and appends with `IntersectionObserver`. URL-backed filters
  and stale-response guards are unchanged. Missing API capabilities (unread,
  ranking, mine-only feed, likes, bookmarks) are not rendered.
- Topic detail: main article + comment tree + composer; sticky 280px
  `SFTopicSideCard` (status/category/replies/views, author as participant,
  tags). Share copies the URL; no fake like/bookmark. Comment stream remains
  tree/flat via `SFCommentStreamControls` (backend has no relevance sort).
  `SFTopicProgressRail` is retained in the theme package but no longer mounted
  on the default detail route.
Public surface tokens live in the theme layer (`sforum-theme.css` etc.);
`--sf-accent*` still come from runtime appearance. Dark mode uses the existing
`.dark` public variables.

`SFComment` has an explicit `presentation`, `depth`, and
`collapseFromDepth` contract. Tree mode renders one branch rail/inset on
desktop, collapses depth-two descendants once at the boundary, and preserves a
direct non-interactive reply reference. Mobile clears every recursive inset;
flat mode never recurses. Rich content containers, code, and images must remain
bounded so no comment depth can widen the document viewport.
Uploaded themes are incremental overlays. When `SFORUM_THEME_LAYER` is set,
`apps/web/nuxt.config.ts` extends `[uploadedThemeLayer, defaultThemeLayer]` so
the uploaded layer can override public pages, layouts, components, and assets,
while missing files continue to resolve from `sforum.default-theme`. The
declared layer directory itself must still exist; only files inside it may be
omitted and inherited from the default theme.
Production and development web Docker images build from the repository root and
copy `extensions/builtin` into `/app/extensions/builtin`, while keeping the web
workdir at `/app/apps/web`; this preserves the static Nuxt layer reference
`../../extensions/builtin/themes/sforum-default/layer` inside containers.
The web production container runs `apps/web/scripts/runtime.mjs`, which watches
`THEME_RELEASE_ROOT/current.json` through `SFORUM_THEME_RELEASE_ROOT` and starts
the selected Nitro server. `current.json` carries `mode` (`uploaded` or
`default`), an absolute `server` path for uploaded releases, and a `layerPath`
for local dev. `runtime.mjs` resolves relative `server` paths against the
release root, uses blue-green Nitro switching, keeps the old child running when
a candidate is missing or unhealthy, and falls back to the default `.output`
when `mode === 'default'` or the file is absent.
Locally, `bun run dev` runs `apps/web/scripts/dev-theme-runtime.mjs`, a
theme-aware supervisor that reads the same `current.json`, injects
`SFORUM_THEME_LAYER` from `layerPath`, and owns exactly one inner `nuxt dev`
(spawned via `bun run dev:plain`). On a selection change it clears the proxy
target, stops and waits for the old process group, then starts the latest layer
and restores traffic after the child is healthy. This local serial restart has
a brief development-only outage: parallel Nuxt dev instances cannot safely
share their build lock, generated output, cache, and HMR resources. The
supervisor loads the repository root `.env`, uses `PORT` or `WEB_PORT` for its
fixed public proxy port, prints that public URL, and suppresses Nuxt
child-process Local/Network lines so internal random `PORT=0` addresses are not
mistaken for the frontend access port.
`bun run dev:plain` runs raw `nuxt dev` as an escape hatch for troubleshooting.
`bun run preview` only serves the fixed `.output` build and does not follow
admin theme switching.
The production Docker build creates a build-local `.nuxt -> .nuxt-build`
symlink before `bun run build` because `tsconfig.json` still extends
`./.nuxt/tsconfig.json` while the build script uses `NUXT_BUILD_DIR=.nuxt-build`.
Layer-owned global CSS is registered from the layer's own directory with an
absolute `import.meta.url`-based path; do not use `~/assets/...` inside a layer
config for theme assets because Nuxt resolves `~` against the host app.
Layer pages that import package types may need host-provided type paths in
`apps/web/nuxt.config.ts` because TypeScript resolves modules from the layer
file location and will not naturally climb into `apps/web/node_modules`.
`SFIconPicker` is available for future admin/user setting forms that need an
icon field. It supports Tabler Icons and the existing Nuxt Icon/Lucide naming,
stores plain `i-tabler-*` or `i-lucide-*` strings, and `nuxt.config.ts`
explicitly includes the local `lucide` and `tabler` icon collections. The
picker loads the full local Tabler/Lucide catalog through the Nuxt server route
`/api/icon-collections/:collection`, returns names in pages, and registers only
the visible page's icon data with Iconify before rendering so thousands of
icons are available without putting every SVG into the first client bundle.
SF inputs/search and the standalone login/register auth inputs now override
WebKit browser autofill styling so saved credentials keep the intended white
input surface, dark text, caret color, and focus ring instead of the default
browser fill background.
The registration page reads backend `data.fields` errors and shows field-level
messages next to username, email, password, and human verification while keeping
login failures as a single actionable top-level message.
The registration password input now always shows the current rule
(`>= 12` characters) before submission. Login and registration success handlers
store the `CurrentUser` returned by the API directly, show a 10-second success
Toast, then navigate, so a successful account creation is not reclassified as a
form failure if a later refresh/navigation step has trouble. `useApiClient()`
reads locale from the Nuxt app i18n runtime instead of calling `useI18n()`,
keeping it safe for route middleware such as the admin guard.
Password readiness progress now uses gradual length scoring against the active
password policy, so the recommended length-only default shows intermediate
progress instead of jumping from 0% to 100%; backend policy validation remains
authoritative.
Login, registration, forgot-password, reset-password, and ordinary protected
user workflows (`/settings/**`, `/topics/new`,
`/t/:topicID/:topicSlug/edit`, plus English prefixes) remain server-rendered
instead of SPA-only and explicitly disable route cache. A global
`auth.global.ts` middleware consumes `requiresAuth` page metadata and redirects
missing users to the locale-aware login page; if the auth API is temporarily
unavailable and there is no cached user, these ordinary user pages still
degrade to login rather than a 503 shell. The root app still waits for startup
web options during SSR, but it does not refresh auth during SSR because public
SWR pages must not cache user-specific payload. Browser startup refresh runs on
mount so SPA admin/component-preview routes do not hold the first client render
behind API calls and cached public pages can still restore valid sessions after
hydration.
Admin pages use a dedicated `admin` Nuxt layout built from Nuxt UI Dashboard
components (`UDashboardGroup`, `UDashboardSidebar`, `UDashboardPanel`,
`UDashboardNavbar`, `UDashboardToolbar`) and Nuxt Icon lucide icons. The source
directory remains `apps/web/app/pages/admin`, while Nuxt `pages:extend`
rewrites the public URL prefix to `NUXT_PUBLIC_ADMIN_ROUTE_PREFIX`, with
`/control-panel` as the default.
The admin layout now renders a dedicated `SFAdminFooter` inside the main
content scroll area. It splits the footer into left-side SForum copyright and
right-side official product summary copy, separate from the public/theme-layer
`SFFooter` and its configurable operator footer links.
Admin modules now use a low-code registry in
`apps/web/app/config/adminModules.ts`: sidebar entries, tab labels/icons,
keep-alive component names, badges, and frontend-visible permission
requirements are centralized there. Page components call `useAdminPage('/id')`
instead of hand-writing `useAdminTabs().openTab(...)` metadata.
The admin index route now renders the "均衡指挥台 / Balanced Command Center"
from `GET /api/v1/admin/overview` rather than faning out to several admin
module endpoints. It keeps `useAdminPage('/')` and `UDashboardToolbar`, shows
API memory, posts, users, action summaries, CSS-only 7-day trend bars, runtime
status, content health, top categories, and quick module links. Formatting
helpers in `app/utils/adminOverview.ts` intentionally avoid locale-dependent
date/number output so SSR and client hydration stay stable.
Admin sidebar parent folders derive active/open state from the current admin
route: only the matching parent opens initially, inactive folders stay
collapsed by default, and the sidebar body scrolls independently when the menu
list grows.
The admin personalization page remains a registered `/personalization` page,
but its sidebar entry now lives under the System configuration folder instead
of the top-level admin group.
The admin shell now includes a `database.manage`-gated database table manager
under the System folder. It uses the existing admin registry, Nuxt UI controls,
native dense tables, masked sensitive cells with per-cell reveal, and CSV
export that keeps sensitive values masked.
Dynamic extension admin pages under `/extensions/{id}/pages/*` are treated as
route-backed custom admin tabs. The admin layout creates a temporary tab from
the route when needed, activates existing custom tabs on route changes, and
keeps the Extensions sidebar folder open/active for those dynamic pages until
the page component replaces the temporary label with manifest metadata.
UI feedback should favor Toasts for user-triggered success and completion
states: authentication success, create/update/delete success, saved settings,
restored defaults, uploads, exports, copied values, queued jobs, and similar
actions should normally show a short Toast. Non-error alerts/toasts should
auto-dismiss after 10 seconds. Blocking errors, field-level validation, and
guidance the user must act on should remain visible near the relevant form or
page state; error Toasts may be used for non-blocking failures but must not
replace field-level messages or auto-close. Success Toast styling should follow
the active SForum appearance/theme tokens and admin personalization settings.
The public forum navbar user dropdown no longer exposes the admin entry link,
so the configurable admin prefix is not revealed from the regular logged-in UI.
The public forum navbar now includes a client-rendered Light/Dark mode toggle
that uses Nuxt Color Mode's `.dark` class. The default theme and SF component
CSS define dark semantic variables for public chrome, cards, search, feed rows,
tabs, pagination, forms, and editor surfaces so the forum home page responds to
the same color-mode state as the admin shell.
Runtime site options are read through `useWebOptions()`. Public options now
include site name, site URL, default locale, enabled locales, and the public
human-verification provider. `site.name` drives the navbar, auth pages, admin
shell, and browser title template, with `SForum` as the fallback product name.
Admin-only option reads and batch saves power the settings page tabs; ALTCHA
secret values are never exposed to public frontend state.
Personalization now reads `appearance.theme` and footer options from the same
runtime option layer. `appearance.theme` remains the stored option key, but UI
language calls it an appearance preset / 配色预设 to avoid confusing color
presets with installable Nuxt Layer themes. The root app sets
`data-sforum-theme` on `<html>`, CSS variables switch between preset colors or
controlled `custom:#rrggbb` colors, and the admin personalization page edits
the appearance preset plus footer copyright/link content. Nuxt UI's generated
`--ui-color-primary-*` and `--ui-primary` tokens are bridged to the same
runtime variables so admin sidebar highlights and `color="primary"` controls do
not keep Nuxt UI's default green. Nuxt UI's `success` token family is also
bridged to the active SForum primary color so success Toasts and other
`color="success"` UI feedback follow the selected appearance preset/custom
color.
Recommended personalization defaults are shared from `useWebOptions()`:
`appearance.theme=pine_teal`, the default bilingual footer copyright, and the
Terms/Privacy/Guidelines footer links. The personalization reset action restores
these recommended defaults instead of only reloading the last saved snapshot.
SEO now reads runtime `seo.*` options through `useWebOptions()` and public pages
should use `useSForumSeo()` for title templates, descriptions, canonical URLs,
robots meta, Open Graph/Twitter tags, verification tags, and minimal JSON-LD.
The Nuxt sitemap module uses a dynamic server source and robots.txt is extended
through a Nitro hook. Local and preview URLs are always noindex.
Admin route middleware distinguishes real unauthenticated responses from
temporary auth-service failures through `useAuthSession()`. A missing user
after refresh redirects to the locale-aware login page even when the auth API is
temporarily unavailable, so API restart/502/timeout cases do not render a Nuxt
503 error page. Cached current users are preserved and continue through the
admin permission check.
Client-side API requests made through `useApiClient().request` now detect
backend API connectivity failures globally. Gateway/runtime failures such as
502/503/504, `server.unavailable`, browser `Failed to fetch`, and timeout-style
network errors open a persistent `SFApiConnectionModal` from the root app shell.
Business errors such as 401, 422, field validation, CSRF recovery, and other
backend envelopes remain owned by the calling page so field-level guidance and
auth redirects keep their existing behavior.
Nuxt now owns a project-specific global error page at `app/error.vue`. The
first release uses the shared public SForum chrome for both forum and admin
routes, renders the selected community empty-state style for `404`, `403`,
`500`, and `503`, and keeps error pages `noindex` through the existing SEO
helper.

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
- Safe pattern: `app.vue` refreshes only web options during SSR and restores
  auth in browser `onMounted`; protected/admin route middleware remains
  responsible for server-side auth checks on cache-disabled routes.
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
