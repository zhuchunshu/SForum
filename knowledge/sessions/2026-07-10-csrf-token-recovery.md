# 2026-07-10 Session Handoff

## Changed

- `useApiClient().request` now primes the browser CSRF cookie before unsafe
  requests when `csrf_` is missing.
- Unsafe requests still use the double-submit `X-Csrf-Token` header, but now
  retry once after `csrf.invalid` by refreshing the token through
  `GET /api/v1/health`.
- Added web regression tests for first-submit CSRF priming and stale-token
  recovery.
- Updated `knowledge/modules/frontend.md` with the safe request pattern.

## Decisions

- Keep the fix in the shared API client instead of patching individual forms.
- Use the existing safe `/api/v1/health` route to mint/refresh the token; no new
  backend endpoint was needed.

## Next

- Continue routing cookie-authenticated writes through `useApiClient().request`.
- If a future page must use raw `fetch` for uploads/streams, mirror the same
  CSRF priming and one-time retry behavior there.

## Open Questions

- None.
