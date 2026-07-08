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
  listing, role creation/update/delete, role permission replacement, permission
  catalog/matrix reads, admin user listing/detail, user role replacement, and
  user direct permission override replacement.
- The permission catalog includes `database.manage` for the read-only admin
  database table manager. `super_admin` receives it by migration and policy as
  part of the protected all-permissions role.
- The permission catalog includes `tag.manage` for forum tag creation,
  approval, disabling, and policy management. Existing deployments receive it
  through the forum taxonomy migration, and `super_admin` receives it by
  default.
- API exposes `/api/v1/auth/registration-status` so the registration page can
  show when the next successful registration will become the initial
  `super_admin`.
- Registration human verification is supported but disabled by default. When
  the admin CAPTCHA settings `human_verification.provider=altcha` and
  `human_verification.scenarios.register=enabled` are enabled,
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
- Login now treats only an explicit missing credential as
  `auth.invalid_credentials`; internal credential-loading errors, such as a
  missing permission table after code/schema drift, bubble up instead of being
  misreported as a wrong password.
- Password policy is now runtime configurable through public
  `identity.password.*` options. Registration and password reset confirmation
  share the same backend `PasswordPolicy` validator; password hashing only owns
  Argon2id hashing and no longer hard-codes product policy.
- Nuxt has login/register pages, an admin route middleware, an admin overview,
  user management, editable user-group management, and a permission matrix. The
  matrix is an audit/comparison view rather than the primary editor: it caps the
  default displayed user groups, supports search and explicit comparison
  selection, and can show only permissions that differ inside the current
  comparison scope.
- Role/user-group creation and updates now trim role fields and reject blank
  keys or aliases. Role keys are limited to stable ASCII path-safe identifiers;
  the roles admin form shows visible field labels and blocks empty submissions
  before calling the API. Migration `202607060002_role_input_constraints`
  removes historical blank custom roles caused by the earlier missing
  validation and adds database non-blank checks.

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
  For non-`super_admin` users this is now extended by direct user permission
  overrides: enabled role permissions plus direct allows minus direct denies.
- Start with database-backed RBAC and Go policy helpers; keep room to adopt
  Casbin if permissions become substantially more complex.
- Keep resource-scoped ACL out of the first admin permissions release. Forum
  category/topic scoped rules should be added only when concrete forum
  workflows require them.
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
- `user_permission_overrides`
- `audit_events`

## Current Boundaries

- Fiber API owns registration, login/logout, session loading, permission
  checks, human-verification enforcement, protected-user invariants, and audit
  writes.
- Nuxt owns login/register pages, the first admin user-group UI shell, route
  guards, and localized permission-denied messages.
- Nuxt route guards are user-experience helpers only. API policy checks remain
  authoritative.

## Permission-Aware Development Rules

- Treat authorization as part of feature design. Before adding a route,
  mutation, admin screen, data export, moderation action, background action, or
  setting update, identify the actor, action, protected resource, and required
  permission boundary.
- Prefer existing permission keys and policy helpers. Add a new permission only
  when it maps to a distinct admin-grantable capability, then update seed data,
  permission catalog text, API contracts when relevant, and frontend permission
  labels.
- Keep permission checks on the API side for every protected operation. Nuxt
  middleware, hidden menu items, disabled buttons, and localized denial messages
  are helpful UI affordances, not security boundaries.
- Cover both allowed and denied paths in tests for unsafe endpoints and admin
  operations. Include direct user allow/deny behavior when a feature depends on
  effective permissions.
- Continue to preserve `super_admin` invariants: active super administrators
  pass all policy checks, and direct permission overrides cannot be edited for
  current `super_admin` users.

## Implementation Notes

- `apps/api/app/Models/Identity/service.go` owns registration, login,
  registration status, current-user loading, actor loading, role-management
  checks, permission catalog/matrix reads, admin user reads, user role
  replacement, user direct permission override replacement, and configurable
  password policy enforcement for registration.
- `apps/api/app/Models/Identity/password.go` owns Argon2id hashing plus the
  shared `PasswordPolicy` model used before password creation/update.
- `apps/api/app/Models/Identity/policy.go` keeps permission checks small:
  `super_admin` receives all permissions while active, and other users rely on
  enabled role permissions plus user direct allows minus direct denies.
- `apps/api/app/Http/Controllers/Identity/controller.go` maps stable API error
  codes such as `auth.required`, `permission.denied`, and
  `role.default_role_locked`; permission management adds stable reasons such as
  `permission.invalid`, `permission.override_conflict`, `role.invalid`,
  `role.invalid_input`, and `user.super_admin_permissions_locked`.
  Registration field errors use backend-localized messages in `data.fields`.
- `apps/api/app/Support/AuthSession` owns authenticated browser session
  lifecycle: login session reset, current-user lookup, idle TTL refresh,
  periodic session-id renewal, logout destruction, and salted session-id hashes
  for audit correlation.
- `apps/api/app/Support/HumanVerify` owns the provider boundary, ALTCHA v2
  challenge/verification adapter, Redis-backed replay/rate-limit store, and
  in-memory test/local store.
- If correct credentials return the generic login-failed message after identity
  or permissions work, check `goose_db_version` and PostgreSQL logs first. A
  local schema missing `202607050002_user_permission_overrides` caused
  permission loading during login to fail before password verification results
  could be surfaced accurately.
- `apps/api/app/Providers/identity.go` wires the identity store, service, and
  controller into the ordered route-provider list.
- `apps/api/bootstrap/app.go` wires a runtime human-verification service that
  reads provider, ALTCHA secret, TTL, and cost from Options on each
  challenge/verify request. Environment values remain first-run fallbacks for
  seeding missing options.
- The default theme registration page
  (`extensions/builtin/themes/sforum-default/layer/app/pages/register.vue`)
  renders the ALTCHA widget client-side only when public option
  `human_verification.provider` is `altcha`, reads the public ALTCHA widget
  type/auto/display/worker/min-duration settings, and maps
  `human_verification.*`, `rate_limit.exceeded`, and `auth.session_unavailable`
  API error codes to localized messages. It also reads
  `/api/v1/auth/registration-status`, shows a first-user super-admin notice
  while no users exist, blocks repeated submit attempts while a request is in
  flight, and resets the ALTCHA widget after verification failures.
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

- Add CSRF protection for cookie-authenticated unsafe requests.
- Tune production ALTCHA challenge cost, expiration, and per-IP limits after
  testing on expected low-end client devices.
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
