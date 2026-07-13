# 2026-07-13 Session Handoff

## Changed

- Admin users list UI now paginates at **20 per page** (`UPagination` +
  `DEFAULT_PER_PAGE = 20`).
- Admin manage drawer is comprehensive:
  - Account: username, email, displayName, locale, status (+ id/timestamps)
  - Public profile: bio, signature, location, websiteUrl
  - Security: revoke all sessions, clear stored client IPs
  - Roles + permission overrides (existing)
- Backend:
  - `AdminUserSummary` adds `createdAt`/`updatedAt`
  - `AdminUserDetail` adds `profile`
  - New `PATCH /users/{userID}` → `Service.UpdateAdminUser` +
    `PostgresStore.UpdateAdminUser` (account + profile, audit
    `user.admin_update`)
- OpenAPI: schema + path for update; list still documents page/perPage
- Tests: identity service cases for manage/ban/self-status; UI validate script
  checks pagination + PATCH + profile save

## Decisions

- Reuse existing `user.manage` / `user.ban` rather than a new permission key.
- Partial PATCH (omit = leave unchanged); separate save buttons for account vs
  profile in the UI for clearer operator feedback.
- Super-admin account edits restricted to super_admin actors (same family of
  protection as session revoke).

## Next

- Optional: admin reset password for a user (not in this change).
- Optional: show avatar attachment management in admin drawer.

## Open Questions

- None for this scope.
