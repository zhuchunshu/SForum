# 2026-07-10 Session Handoff

## Changed

- Updated `apps/web/app/middleware/admin.ts` so a missing admin session after a
  transient auth-service failure redirects to login instead of aborting
  navigation with a Nuxt 503 error.
- Added `apps/web/tests/adminRouteRendering.test.ts` coverage that executes the
  admin middleware and verifies the unavailable-auth path does not call
  `createError`/`abortNavigation`.
- Updated frontend knowledge notes to reflect the no-Nuxt-error admin fallback.

## Decisions

- Keep API policy checks authoritative. The frontend guard remains an operator
  experience helper: cached current users continue through the existing
  `admin.access` permission check, while missing users go to login.

## Verification

- `bun test tests/adminRouteRendering.test.ts`
- `bun test tests/adminRouteRendering.test.ts tests/protectedRouteRendering.test.ts tests/useApiClient.test.ts tests/appStartup.test.ts`
- `bun run typecheck`
- Browser QA: `http://127.0.0.1:3000/control-panel` redirected to login without
  a Nuxt error overlay or console errors.

## Next

- If a future admin shell wants to keep users on the admin URL while the API is
  down, add a first-class recoverable admin unavailable state rather than
  reintroducing Nuxt fatal errors.

## Open Questions

- None.
