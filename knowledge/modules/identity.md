# Identity Module

## Purpose

Owns users, credentials, sessions, registration, login/logout, roles,
permissions, human-verification requirements for identity flows, and policy
helpers.

## Current Status

Designed. No identity implementation exists yet.

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

## Planned Tables

- `users`
- `user_credentials`
- `roles`
- `permissions`
- `role_permissions`
- `user_roles`
- `audit_events`

## Planned Boundaries

- Fiber API owns registration, login/logout, session loading, permission
  checks, human-verification enforcement, protected-user invariants, and audit
  writes.
- Nuxt owns login/register pages, admin role-management UI, route guards, and
  localized permission-denied and verification-failure messages.
- Nuxt route guards are user-experience helpers only. API policy checks remain
  authoritative.

## Open Questions

- Which exact username, email, and password rules should MVP registration use?
- Should email verification be required before posting, or only before
  sensitive account recovery flows?
- What ALTCHA challenge expiration and work cost should be the production
  default?
- Which role-management screens are required in the first admin milestone?

## Next Steps

- Add migrations for identity and RBAC tables.
- Seed `super_admin`, `member`, and the initial permission list.
- Implement concurrent-safe first-user registration.
- Add ALTCHA challenge generation, server verification, Redis replay
  protection, and registration rate limits.
- Add session middleware and current-user endpoint.
- Add admin role-management API endpoints and UI after the identity foundation
  works.
