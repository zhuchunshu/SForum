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

## Runtime themes / Page Registry (live)

Public theming is runtime Page Registry + L0/L1 (P0–P5 complete):

- **Page Registry** + stable page ids (`forum.home`, …)
- **L0** CSS/assets without rebuild
- **L1** sandboxed templates composing host SF islands
- **L2** author-prebuilt widgets for heavy UI
- Host stays Nuxt; public themes are not full Nuxt apps and do not rebuild Nitro

**ADR:** `../decisions/2026-07-13-runtime-page-registry-themes.md`  
**Plan:** `../plans/2026-07-13-runtime-page-registry-themes.md`  
**Page catalog:** `../../docs/extensions/page-catalog.md`

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
Public themes no longer ship as Nuxt Layers selected at runtime. Host
`apps/web` owns public pages/components/layouts/CSS. Themes are L0/L1 packages
(`theme.json` + `assets/` + `templates/`) activated through the Page Registry
without rebuilding Nuxt or restarting Nitro.

- `bun run dev` / production `theme:runtime` (`scripts/runtime-plain.mjs`) start
  plain Nuxt/Nitro.
- Active theme skin CSS is injected client-side from
  `GET /api/v1/site/active-theme/skin` + theme-assets routes.
- Trusted **admin** plugin frontends may still use Web Release / dev-compose
  (`bun run dev:compose`, `SFORUM_ADMIN_REGISTRY_ROOT`).
- Optional legacy scripts (`dev-theme-runtime.mjs`, `runtime.mjs`) remain for
  admin composition experiments and historical Web Release contracts; they are
  not the public theme activation path.

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
