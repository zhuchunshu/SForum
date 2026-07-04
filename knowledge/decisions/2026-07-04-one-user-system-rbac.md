# Decision: One User System With Forum RBAC

## Status

Accepted

## Context

SForum is a forum program. Public users, moderators, and administrators should
belong to one community identity system instead of being split into front-office
and back-office accounts.

The product needs open registration after the first administrator exists. It
also needs custom roles, called user groups in the admin UI, without making the
MVP authorization model too heavy.

## Decision

Use one `users` table for all accounts.

The first successfully registered user becomes the initial `super_admin`. This
user has all permissions and cannot be deleted, disabled, or stripped of the
`super_admin` role.

After bootstrapping, registration stays open. Newly registered users receive the
system `member` role by default. The `member` role key is stable and
undeletable while it is the default registration role. Administrators may change
its display alias.

Support custom roles/user groups in the admin area. Users can have multiple
roles, and effective permissions are the union of enabled role permissions.

Start with database-backed RBAC plus Go policy helpers. Keep the policy
interface small enough to adopt Casbin later if the permission model becomes
more complex.

## Consequences

- Administrators and moderators remain normal forum users with extra powers.
- Profiles, sessions, audit trails, and moderation history stay unified.
- Open registration matches the expected behavior of a community forum.
- System roles need database invariants and service-layer checks.
- Category-scoped moderation and explicit deny rules are deferred until product
  requirements justify them.
