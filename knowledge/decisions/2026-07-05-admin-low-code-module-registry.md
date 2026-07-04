# 2026-07-05 Admin Low-Code Module Registry

## Status

Accepted.

## Context

The admin shell was still spreading the same page metadata across multiple
places: sidebar menu items, tab definitions, route-to-tab sync, page header
icons, and per-page `openTab(...)` calls. This made every new admin page easy
to wire inconsistently.

## Decision

Admin pages are registered through
`apps/web/app/config/adminModules.ts`.

- `adminPageDefinitions` owns each page's stable id, translation key, Lucide
  icon, keep-alive component name, badge key, and optional frontend-visible
  permission requirement.
- `adminSidebarNavigation` owns the sidebar tree and can reference registered
  page ids instead of duplicating route labels/icons.
- `useAdminTabs()` opens tabs by page id only and derives tab metadata from the
  registry.
- Admin pages call `useAdminPage('/page-id')` and keep `defineOptions({ name:
  '...' })` aligned with the registry component name.
- `admin.vue` builds the sidebar from the registry and hides entries whose
  registered permissions are not present on the current user. API policies
  remain authoritative.
- Route-to-tab synchronization uses `useAdminRoutes().routeId(route.path)` plus
  the page registry, not local string parsing in the layout.

## Consequences

- Adding a normal admin page now requires adding one page definition, optionally
  adding it to `adminSidebarNavigation`, and calling `useAdminPage('/id')` from
  the page.
- Sidebar labels, page header icons, tab labels, closability, and keep-alive
  names no longer need to be repeated in every page component.
- New admin modules should not add hard-coded menu arrays to `admin.vue` or
  call `useAdminTabs().openTab(...)` directly from page components.
