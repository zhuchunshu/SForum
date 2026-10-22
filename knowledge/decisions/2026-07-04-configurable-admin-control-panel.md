# 2026-07-04 Configurable Admin Control Panel

## Status

Accepted.

## Context

SForum needs a backend/admin foundation before broader forum management
features are added. The admin URL prefix must be configurable through `.env`,
default to `/control-panel`, and the UI should follow the Nuxt dashboard
template direction with Nuxt UI and Nuxt Icon.

## Decision

- Keep admin page source files under `apps/web/app/pages/admin` for a stable
  project structure.
- Rewrite the public admin route prefix in `apps/web/nuxt.config.ts` via
  `pages:extend`, using `NUXT_PUBLIC_ADMIN_ROUTE_PREFIX` and defaulting to
  `/control-panel`.
- Use a dedicated `apps/web/app/layouts/admin.vue` shell based on Nuxt UI
  Dashboard components and lucide icons through Nuxt Icon.
- Generate admin links through `useAdminRoutes()` instead of hard-coding
  `/admin` or relying on stale i18n route resources.
- Keep the first admin milestone focused on the shell, overview page, and
  existing user-group list.

## Consequences

- Deployments can move the admin entry point by changing `.env`.
- Future admin pages should live under `apps/web/app/pages/admin` and inherit
  `layout: 'admin'` plus the `admin` middleware.
- Admin navigation should use `useAdminRoutes().path(...)`.
- If Nuxt i18n route generation changes, verify `useAdminRoutes()` still emits
  `/control-panel` for the default locale and locale-prefixed paths for
  secondary locales. (Superseded 2026-10-22 — see Update below.)

## Update 2026-10-22

- `useAdminRoutes().path()` no longer hand-rolls a locale segment; it delegates
  to i18n's `localePath`. The app runs the `no_prefix` strategy, so
  `/<locale><prefix>` is not a registered route: client-side navigation to it
  hit the Page Registry catch-all, while the server-side 301 in
  `server/middleware/locale-prefix-compat.ts` only covers full page loads.
  Legacy bookmarks under `/en/**` still 301 to the unprefixed path.
- The first registered account (initial `super_admin`) lands on
  `useAdminRoutes().path('/')` after registration; every other registration
  keeps the normal auth return. See
  `sessions/2026-10-22-first-admin-admin-landing.md`.
