# 2026-07-05 Session Handoff

## Changed

- Added `apps/web/app/config/adminModules.ts` as the low-code registry for
  admin pages, sidebar groups, icons, tab metadata, and frontend-visible
  permission requirements.
- Added `useAdminPage('/id')` so admin pages register tabs by id only.
- Updated `useAdminTabs()` to derive tab metadata from the registry.
- Updated `admin.vue` to build the sidebar from `adminSidebarNavigation`, hide
  pages based on registered permissions, and sync route changes through
  `useAdminRoutes().routeId(...)`.
- Migrated dashboard, users, roles, permissions, settings, and personalization
  pages to `useAdminPage(...)`.
- Localized the remaining admin shell labels for control panel title,
  administrator label, theme toggle labels, and system navigation group.
- Expanded `tests/validate-admin-framework.ts` to enforce the registry-driven
  admin-page contract.

## Decisions

- Admin module metadata belongs in `adminModules.ts`; page components should
  not duplicate sidebar/tab label/icon/component metadata.
- Frontend sidebar permission filtering is a usability aid only. Backend API
  policy checks remain the source of truth.

## Next

- When adding the next admin module, start with `adminPageDefinitions` and
  `adminSidebarNavigation`, then call `useAdminPage('/new-id')` in the page.
- Consider extracting common admin page headers later if more screens repeat the
  same title/intro/icon layout.

## Open Questions

- Should direct navigation to a registered but frontend-hidden admin page show a
  dedicated 403 page before its API calls fail, or is the current API-driven
  denial enough for the first milestone?
