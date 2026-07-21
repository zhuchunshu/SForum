# 2026-07-04 Identity Permissions Implementation

## Changed

- Implemented the identity/RBAC foundation for SForum: PostgreSQL schema,
  seed roles/permissions, Redis-backed browser sessions, registration/login
  endpoints, current-session loading, role/user-group management service and
  API, OpenAPI documentation, Nuxt auth pages, admin middleware, and admin
  user-group shell.
- Added repository validation for required identity UI files and locale keys.
- Fixed smoke behavior so registration responses reload the current user and
  return `roleKeys`/`permissions` as arrays instead of `null`.

## Decisions

- Keep one user system for public members, moderators, and administrators.
- First registered user becomes the protected initial `super_admin`.
- Later open registrations default to `member`.
- `member` alias can change, but `member` cannot be deleted as the default
  registration role.
- API permission checks are authoritative; Nuxt route middleware is only a
  user-experience guard.

## Verification

- `go test ./...` in `apps/api` passed.
- `bun run typecheck` in `apps/web` passed.
- `./scripts/test.sh` passed.
- Compose migration completed with `migrations complete`.
- Smoke registration through `http://127.0.0.1:3000/api/v1/auth/register`
  created the first `super_admin` and second `member` user.
- Login smoke for the first admin returned a non-null permissions array.

## Next

- Add CSRF protection for cookie-authenticated unsafe requests.
- Add ALTCHA verification and Redis-backed rate limiting for registration.
- Expand admin user-group UI from list shell to create/edit/delete and
  permission replacement workflows.
- Add account status management while protecting the initial `super_admin`.

## Open Questions

- Exact username, email, and password rules for MVP.
- Whether email verification is required before posting or only before account
  recovery.
- Production ALTCHA challenge cost and expiration.
