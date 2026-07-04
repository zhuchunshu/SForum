# Identity Module

## Purpose

Owns users, credentials, sessions, registration, login/logout, roles,
permissions, human-verification requirements for identity flows, and policy
helpers.

## Current Status

Initial identity foundation is implemented.

- PostgreSQL migrations create users, credentials, roles, permissions, role
  assignments, and audit events.
- Seed data includes `super_admin`, the default `member` role, and initial
  permission keys.
- Registration is open. The first registered user becomes the protected initial
  `super_admin`; later registrations receive `member`.
- Browser sessions are backed by Redis through Fiber sessions.
- API endpoints exist for registration, login, logout, current session, role
  listing, role creation/update/delete, and role permission replacement.
- API exposes `/api/v1/auth/registration-status` so the registration page can
  show when the next successful registration will become the initial
  `super_admin`.
- Registration human verification is supported but disabled by default. When
  `HUMAN_VERIFICATION_PROVIDER=altcha` is set,
  `/api/v1/human-verification/challenge?purpose=register` returns an ALTCHA v2
  challenge, and `/api/v1/auth/register` verifies the submitted
  `humanVerification` token only after editable registration fields and
  username/email conflicts pass validation.
- Registration validation now returns actionable field-level API errors under
  `data.fields` for `username`, `email`, `password`, and `humanVerification`.
  The stable reason for editable registration fields is
  `auth.register_invalid`; login failures still use one generic
  `auth.invalid_credentials` reason to avoid account enumeration.
- If account creation succeeds but the browser session cannot be saved,
  registration returns `auth.session_unavailable` with the localized message
  "账号已创建，但自动登录失败，请直接登录。"; the user should log in rather than retry
  registration or human verification.
- Browser auth remains Redis-backed server session, not JWT-first. Sessions use
  HTTP-only SameSite=Lax cookies, secure cookies in production, 30-day idle
  timeout, 180-day absolute timeout, and 24-hour session-id renewal by default.
- Nuxt treats only 401/`auth.required` from `/auth/session` as logged out.
  Transient API restart, timeout, or gateway failures keep the existing
  frontend user state and surface auth service unavailability instead of
  redirecting to login.
- Successful registration auto-login and every successful login write
  `audit_events` records with user id, action, IP address, User-Agent, and a
  salted session-id hash. The first version stores this for security/admin
  review and does not expose it to users yet.
- Nuxt has login/register pages, an admin route middleware, an admin overview,
  and a first user-group list shell.

## Architecture Decisions

- Use one `users` table for public users, moderators, and administrators.
- The first registered user becomes the initial `super_admin`.
- The initial super administrator cannot be deleted, disabled, or stripped of
  the `super_admin` role.
- Registration remains open after bootstrapping.
- Later registrations receive the system `member` role by default.
- `member` can have a custom display alias, but its role key cannot change and
  it cannot be deleted while it is the default registration role.
- Admin-managed custom roles are supported and can be presented as user groups.
- Effective permissions are the union of all enabled roles assigned to a user.
- Start with database-backed RBAC and Go policy helpers; keep room to adopt
  Casbin if permissions become substantially more complex.
- Keep human verification disabled by default; use ALTCHA as the first
  supported self-hosted provider for deployments that enable registration and
  password-reset checks.
- Do not challenge every login by default; require human verification for login
  only after suspicious failure patterns.
- Store challenge replay protection and rate-limit state in Redis.
- Do not introduce access/refresh JWT for first-party browser forum sessions.
  If SForum later ships mobile apps or third-party API access, use short-lived
  access tokens and persisted rotating refresh tokens with reuse detection.

## Implemented Tables

- `users`
- `user_credentials`
- `roles`
- `permissions`
- `role_permissions`
- `user_roles`
- `audit_events`

## Current Boundaries

- Fiber API owns registration, login/logout, session loading, permission
  checks, human-verification enforcement, protected-user invariants, and audit
  writes.
- Nuxt owns login/register pages, the first admin user-group UI shell, route
  guards, and localized permission-denied messages.
- Nuxt route guards are user-experience helpers only. API policy checks remain
  authoritative.

## Implementation Notes

- `apps/api/app/Models/Identity/service.go` owns registration, login,
  registration status, current-user loading, actor loading, and role-management
  service checks.
- `apps/api/app/Models/Identity/policy.go` keeps permission checks small:
  `super_admin` receives all permissions while active, and other users rely on
  the union of enabled role permissions.
- `apps/api/app/Http/Controllers/Identity/controller.go` maps stable API error
  codes such as `auth.required`, `permission.denied`, and
  `role.default_role_locked`; registration field errors use backend-localized
  messages in `data.fields`.
- `apps/api/app/Support/AuthSession` owns authenticated browser session
  lifecycle: login session reset, current-user lookup, idle TTL refresh,
  periodic session-id renewal, logout destruction, and salted session-id hashes
  for audit correlation.
- `apps/api/app/Support/HumanVerify` owns the provider boundary, ALTCHA v2
  challenge/verification adapter, Redis-backed replay/rate-limit store, and
  in-memory test/local store.
- `apps/api/app/Providers/identity.go` wires the identity store, service, and
  controller into the ordered route-provider list.
- `apps/api/bootstrap/app.go` wires the configured human-verification provider.
  `HUMAN_VERIFICATION_PROVIDER=disabled` is the default; set it to `altcha` to
  require ALTCHA.
- `apps/web/app/pages/register.vue` renders the ALTCHA widget client-side only
  when the public runtime provider is `altcha`, and maps
  `human_verification.*`, `rate_limit.exceeded`, and
  `auth.session_unavailable` API error codes to localized messages. It also
  reads `/api/v1/auth/registration-status`, shows a first-user super-admin
  notice while no users exist, blocks repeated submit attempts while a request
  is in flight, and resets the ALTCHA widget after verification failures.
- Registration builds and loads the returned current-user access inside the
  bootstrap transaction so response construction failures roll back account
  creation instead of leaving a created user behind a 500 response.
- `contracts/openapi.yaml` documents the current auth and role endpoints.

## Open Questions

- Which exact username, email, and password rules should MVP registration use?
- Should email verification be required before posting, or only before
  sensitive account recovery flows?
- What ALTCHA challenge expiration and work cost should be the production
  default?
- Which role-management screens are required in the first admin milestone?

## Next Steps

- Tune production ALTCHA challenge cost, expiration, and per-IP limits after
  testing on expected low-end client devices.
- Add CSRF protection for cookie-authenticated unsafe requests.
- Add admin/user-facing login history views when the account/security UI is
  built.
- Add risk-based controls for new device/IP patterns, including optional
  reauthentication or human verification.
- Extend the same human-verification boundary to password-reset initiation and
  risk-based login/posting checks when those flows are implemented.
- Add account deletion/disable flows while preserving the initial
  `super_admin` invariant.
- Expand the admin role-management UI from list shell to editable role and
  permission screens.
- Decide exact username, email, password, and email-verification policies.
