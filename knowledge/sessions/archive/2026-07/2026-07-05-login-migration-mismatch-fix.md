# 2026-07-05 Login Migration Mismatch Fix

## Changed

- Applied the missing local database migration
  `202607050002_user_permission_overrides.sql`; `goose_db_version` now includes
  `202607050002`.
- Login credential lookup now maps only explicit missing credentials to
  `auth.invalid_credentials`; internal store errors bubble up instead of being
  disguised as a wrong-password response.
- Added identity service coverage for credential-store errors during login.

## Root Cause

- Correct-password login was failing because current identity code loaded user
  permissions during credential lookup, but the local database did not yet have
  the `user_permission_overrides` table.
- PostgreSQL logged `relation "user_permission_overrides" does not exist`.
  Before the code hardening, `Service.Login()` converted that store error into
  `ErrInvalidCredentials`, which produced the generic frontend login-failed
  message.

## Verification

- `go test ./app/Models/Identity ./app/Http`
- HTTP probe through Nuxt dev proxy:
  - `POST /api/v1/auth/register` created
    `codex_login_fix_20260705_0434`.
  - `POST /api/v1/auth/login` with the same password returned `200 OK`.
- PostgreSQL logs showed no new missing-table errors after the migration.

## Next

- Keep running migrations after permission/schema changes before testing auth
  flows.
- Consider surfacing unexpected login-time service errors as a temporary service
  failure message in the frontend if that becomes a recurring local-dev pain.

