# 2026-07-10 Session Handoff - Homepage Auth Cache State

## Changed

- Fixed root app startup so SSR refreshes only public web options and does not
  embed `auth:user` into cacheable public page payloads.
- Browser startup now restores `/auth/session` from `onMounted`, avoiding reuse
  of the SSR `app-startup` async-data payload that could leave the homepage in
  guest UI after refresh.
- Added `apps/web/tests/appStartup.test.ts` coverage for both constraints.

## Decisions

- Public SWR pages must not write user-specific auth state into SSR payloads.
- Admin/protected routes keep using route middleware as the server-side auth
  authority because those routes are cache-disabled.

## Next

- Browser-check the exact logged-in homepage refresh flow when a browser
  session with a valid SForum login is available.

## Open Questions

- None.
