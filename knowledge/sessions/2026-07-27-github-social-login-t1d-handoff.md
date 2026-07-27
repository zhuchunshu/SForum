# 2026-07-27 GitHub Social Login T1D Handoff

## Status

**T1D complete.** M1R remaining: T1E only. Do not start M2 or the
GitHub plugin package / UI work.

Prior: `sessions/2026-07-27-github-social-login-t1c-handoff.md`.

## Changed

- Session-bound recent-auth:
  `user_recent_auth` PK is `(user_id, session_fingerprint)`; fingerprint is
  non-reversible SID SHA-256 (`SessionFingerprint`). Mark after successful
  password login and external login/registration; empty fingerprint fail
  closed. Cross-session isolation tested.
- Unlink: load target link first; verify actor ownership, active status, and
  expected revision; last-login-method check and unlink mutation share one
  transaction (`FOR UPDATE` + TransitionTx). Idempotency key is
  `unlink:{user}:{link}:r{rev}:{requestId}` (never client IP).
- Password credentials: `password_hash` remains NOT NULL; external-only users
  have no `user_credentials` row. Migration revised + repair
  `202607270057_external_auth_t1d_session_credentials.sql`. Upsert path for
  self-service setup, password-reset confirm, and admin set-password.
- `POST /api/v1/auth/password` (login + session-bound recent-auth) creates or
  updates local password.
- External registration reuses `Service.ValidateExternalRegister` (username/
  email/reserved/hooks/registration-mode; no password) and human verification;
  editable fields validated **before** opaque ticket consume; policy re-checked
  inside creation TX; default-role assignment requires `RowsAffected == 1` or
  full rollback; post-create reload via canonical `GetCurrentUser` before
  session issue.
- Zero-user bootstrap still blocks external registration; password bootstrap
  remains open. Non-enumerating email behavior preserved on password reset
  (external-only lookup by email is silent when missing).

## Decisions

- Recent-auth is durable in PostgreSQL (session fingerprint columns) rather
  than Redis-only so tests and multi-node share the same gate without a second
  store contract.
- Unlink request body may omit `expectedRevision`/`requestId`; revision
  defaults to the loaded active link revision; request id is Host-generated
  opaque when absent (still not IP-bound).
- Password setup for external-only does not require a current password; it
  requires session-bound recent-auth only.

## Verification

Focused:

```text
cd apps/api && go test ./app/Models/Identity/ ./app/Http/Controllers/Identity/ ./app/Support/Localization/ -count=1
# all ok
```

Full:

```text
cd apps/api && go test ./... -count=1
# all packages ok (exit 0)
```

T1D-specific coverage: session fingerprint non-reversible, cross-session
isolation, authorize-link session gate, unlink idempotency key shape,
shared identity field validators for external registration.

## Next

Start a fresh conversation for **T1E only**: OpenAPI/Core Route Catalog,
controller HTTP allowed/denied tests, callback replay/exact-artifact,
database transition, two-provider, and M1 re-review report. Do not start M2,
plugin, or UI work.

## Open Questions

- None for T1E entry. If implementation evidence disproves a frozen rule, stop
  for a user decision.
