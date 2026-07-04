# 2026-07-05 Admin Permission Management

## Changed

- Added `user_permission_overrides` for per-user global permission allow/deny.
- Effective permissions for ordinary users are now enabled role permissions plus
  direct allows minus direct denies.
- Added API endpoints for permission catalog/matrix, admin user list/detail,
  user role replacement, and user permission override replacement.
- Expanded the admin UI:
  - Users page manages user groups and direct permission overrides.
  - Roles page creates/edits custom groups and role permissions.
  - Permissions page shows the role permission matrix.
- Updated OpenAPI, backend/frontend localization, and UI validation scripts.

## Decisions

- Keep the first release global action-level only; no category/topic scoped ACL
  yet.
- Do not edit direct permission overrides for current `super_admin` users.
- Keep the API as the final authorization boundary; frontend controls are only
  an admin usability layer.

## Verification

- `go test ./...` in `apps/api` passed.
- `bun run typecheck` in `apps/web` passed.
- `node tests/validate-identity-ui.js` passed.
- `bun tests/validate-admin-framework.ts` passed.

## Next

- Add CSRF protection for cookie-authenticated unsafe requests.
- Consider account status management after the user list is connected to
  disable/ban workflows.
- Add resource-scoped forum ACL only after category/topic workflows exist.

## Open Questions

- Should direct denies be highlighted in future public-facing moderation tools?
- Should permission changes eventually force session refresh or show a live
  admin notification?
