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
- Use ALTCHA as the default human-verification provider for open registration
  and password-reset initiation.
- Do not challenge every login by default; require human verification for login
  only after suspicious failure patterns.
- Store challenge replay protection and rate-limit state in Redis.

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
  current-user loading, actor loading, and role-management service checks.
- `apps/api/app/Models/Identity/policy.go` keeps permission checks small:
  `super_admin` receives all permissions while active, and other users rely on
  the union of enabled role permissions.
- `apps/api/app/Http/Controllers/Identity/controller.go` maps stable API error
  codes such as `auth.required`, `permission.denied`, and
  `role.default_role_locked`.
- `apps/api/app/Providers/identity.go` wires the identity store, service, and
  controller into the ordered route-provider list.
- Registration responses reload the current user after the bootstrap
  transaction so `roleKeys` and `permissions` serialize as arrays.
- `contracts/openapi.yaml` documents the current auth and role endpoints.

## Open Questions

- Which exact username, email, and password rules should MVP registration use?
- Should email verification be required before posting, or only before
  sensitive account recovery flows?
- What ALTCHA challenge expiration and work cost should be the production
  default?
- Which role-management screens are required in the first admin milestone?

## Next Steps

- Add ALTCHA challenge generation, server verification, Redis replay
  protection, and registration rate limits.
- Add CSRF protection for cookie-authenticated unsafe requests.
- Add account deletion/disable flows while preserving the initial
  `super_admin` invariant.
- Expand the admin role-management UI from list shell to editable role and
  permission screens.
- Decide exact username, email, password, and email-verification policies.
