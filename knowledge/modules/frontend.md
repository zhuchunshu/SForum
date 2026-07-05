# Frontend Module

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
App startup also attempts to restore the current browser session from
`/auth/session`; transient API failures mark auth as temporarily unavailable
without clearing the cached user state.
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
`SFIconPicker` is available for future admin/user setting forms that need an
icon field. It supports Tabler Icons and the existing Nuxt Icon/Lucide naming,
stores plain `i-tabler-*` or `i-lucide-*` strings, and `nuxt.config.ts`
explicitly includes the local `lucide` and `tabler` icon collections.
SF inputs/search and the standalone login/register auth inputs now override
WebKit browser autofill styling so saved credentials keep the intended white
input surface, dark text, caret color, and focus ring instead of the default
browser fill background.
The registration page reads backend `data.fields` errors and shows field-level
messages next to username, email, password, and human verification while keeping
login failures as a single actionable top-level message.
The registration password input now always shows the current rule
(`>= 12` characters) before submission. Login and registration success handlers
store the `CurrentUser` returned by the API directly before navigating, so a
successful account creation is not reclassified as a form failure if a later
refresh/navigation step has trouble. `useApiClient()` reads locale from the
Nuxt app i18n runtime instead of calling `useI18n()`, keeping it safe for route
middleware such as the admin guard.
Admin pages use a dedicated `admin` Nuxt layout built from Nuxt UI Dashboard
components (`UDashboardGroup`, `UDashboardSidebar`, `UDashboardPanel`,
`UDashboardNavbar`, `UDashboardToolbar`) and Nuxt Icon lucide icons. The source
directory remains `apps/web/app/pages/admin`, while Nuxt `pages:extend`
rewrites the public URL prefix to `NUXT_PUBLIC_ADMIN_ROUTE_PREFIX`, with
`/control-panel` as the default.
Admin modules now use a low-code registry in
`apps/web/app/config/adminModules.ts`: sidebar entries, tab labels/icons,
keep-alive component names, badges, and frontend-visible permission
requirements are centralized there. Page components call `useAdminPage('/id')`
instead of hand-writing `useAdminTabs().openTab(...)` metadata.
The public forum navbar user dropdown no longer exposes the admin entry link,
so the configurable admin prefix is not revealed from the regular logged-in UI.
Runtime site options are read through `useWebOptions()`. Public options now
include site name, site URL, default locale, enabled locales, and the public
human-verification provider. `site.name` drives the navbar, auth pages, admin
shell, and browser title template, with `SForum` as the fallback product name.
Admin-only option reads and batch saves power the settings page tabs; ALTCHA
secret values are never exposed to public frontend state.
Personalization now reads `appearance.theme` and footer options from the same
runtime option layer. The root app sets `data-sforum-theme` on `<html>`, CSS
variables switch between preset themes or controlled `custom:#rrggbb` colors,
and the admin personalization page edits the theme plus footer copyright/link
content. Nuxt UI's generated `--ui-color-primary-*` and `--ui-primary` tokens
are bridged to the same runtime theme variables so admin sidebar highlights and
`color="primary"` controls do not keep Nuxt UI's default green.
Admin route middleware distinguishes real unauthenticated responses from
temporary auth-service failures. A 401 or `auth.required` redirects to login;
API restart/502/timeout cases show a temporary unavailable error instead of
forcing the user to sign in again.

## Regression Notes

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
