# Frontend Module

## Purpose

Owns the Nuxt web application, SSR pages, UI composition, frontend routing,
metadata, and browser-side interactions.

## Current Status

Foundation scaffold exists under `apps/web`.
The web container now passes `APP_URL` into Nuxt, and Nuxt uses it for the
site config and Nuxt i18n SEO `baseUrl`.
Generated output directories are ignored by Nuxt/Vite development watchers, and
`bun run build`/`bun run typecheck` use isolated Nuxt temporary directories so
they do not disturb the active dev server state.
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
SF inputs/search and the standalone login/register auth inputs now override
WebKit browser autofill styling so saved credentials keep the intended white
input surface, dark text, caret color, and focus ring instead of the default
browser fill background.
Admin pages use a dedicated `admin` Nuxt layout built from Nuxt UI Dashboard
components (`UDashboardGroup`, `UDashboardSidebar`, `UDashboardPanel`,
`UDashboardNavbar`, `UDashboardToolbar`) and Nuxt Icon lucide icons. The source
directory remains `apps/web/app/pages/admin`, while Nuxt `pages:extend`
rewrites the public URL prefix to `NUXT_PUBLIC_ADMIN_ROUTE_PREFIX`, with
`/control-panel` as the default.

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
  management, moderation, audit, and site settings.
- Add SEO metadata conventions before real pages proliferate.
- Start replacing static forum page sketches with the reusable `SF*` components.
