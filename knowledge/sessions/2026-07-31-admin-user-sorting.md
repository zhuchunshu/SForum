# 2026-07-31 Admin User Sorting Handoff

## Changed

- Added server-side `sortBy` / `sortOrder` support to `GET /users` for joined,
  updated, username, display name, email, and account status ordering.
- Added a whitelist-only PostgreSQL order mapper with `id` as the stable
  pagination tiebreaker and focused injection-regression tests.
- Added bilingual field/direction selectors to `/control-panel/users`; changing
  either resets the list to page 1 and refreshes from the API.
- Extracted `SFAdminUserListToolbar`; lowered the users route architecture cap
  from 1595 to 1570 and Identity service from 1091 to 1079.

## Decisions

- Default order is registration time descending (newest users first).
- Invalid API sort values fail safely to the default instead of reaching SQL.
- Sorting remains server-side because the list is paginated.

## Verification

- `cd apps/api && go test -count=1 ./...`
- `cd apps/web && bun run typecheck`
- Focused Bun sorting/list-surface tests: 8 passed.
- OpenAPI reference validation and architecture boundary validation passed.
- Rendered Browser QA is not complete: the in-app browser redirected as guest,
  and Browser timed out twice while claiming the existing logged-in Chrome tab.

## Next

- Re-run `/control-panel/users` desktop and `390x844` Browser QA when Chrome
  control is stable; verify selector changes, row order, console health, and no
  horizontal overflow.

## Open Questions

- None.
