# 2026-07-05 Registration Success Navigation Handoff

## Changed

- Added a persistent registration password hint that mirrors the current
  backend rule: at least 12 characters.
- Login and registration pages now store the `CurrentUser` returned by the API
  directly before navigation instead of doing a separate `/auth/session`
  refresh.
- `useApiClient()` no longer calls `useI18n()`; it reads the runtime locale from
  Nuxt app i18n state so it can be used safely by route middleware.
- Updated the lightweight frontend auth test harness for runtime config and
  direct auth-state hydration.

## Decisions

- Keep the password policy unchanged for this fix.
- Prefer successful API response data as the immediate frontend auth source of
  truth for login/register flows.

## Verification

- `bun test tests/useApiClient.test.ts`
- `bun run typecheck`
- `go test ./app/Http ./app/Models/Identity ./app/Support/AuthSession`
- Manual HTTP smoke through Nuxt dev proxy on port 3100:
  registration returned 201, `/auth/session` with the registration cookie
  returned 200, and login with the same password returned 200.

## Next

- Add browser-level auth flow coverage when Playwright/Browser tooling is
  available without registry download issues.

## Open Questions

- Should successful registration show a short toast before redirect, or is
  immediate navigation enough for MVP?
