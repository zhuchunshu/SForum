# 2026-07-10 Session Handoff

## Changed

- Added a global frontend API connectivity error state in
  `apps/web/app/composables/useApiConnectionError.ts`.
- `useApiClient().request` now opens that state when a client-side request hits
  backend API connectivity failures such as 502/503/504, `server.unavailable`,
  browser fetch failures, or timeouts.
- Added `SFApiConnectionModal` to the root app shell so all pages can show one
  persistent modal when the backend API is unreachable.
- Added Chinese and English modal copy under `errors.apiConnection`.

## Decisions

- Keep business errors local to pages. Auth redirects, field validation, CSRF
  recovery, and normal API envelopes should not trigger the global modal.
- Keep the modal persistent with manual dismiss and refresh-retry actions,
  matching SForum's rule that blocking errors should not auto-close.
- Do not show raw API paths in the modal; store the path in state for tests and
  debugging only.

## Verification

- `bun test tests/useApiClient.test.ts tests/appStartup.test.ts`
- `bun run typecheck`

## Next

- Continue routing browser API calls through `useApiClient().request` so the
  global connectivity handling is applied consistently.

## Open Questions

- None.
