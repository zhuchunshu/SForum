# Decision: User-Level Permission Overrides For Admin Management

## Status

Accepted

## Context

SForum already uses one user system with database-backed RBAC. User groups are
good for common access patterns, but the admin area also needs precise control
over what a specific user can do without creating one-off groups for every
exception.

Resource-scoped access control for categories, topics, or posts is not needed
yet because the forum domain workflows are still early.

## Decision

Keep the existing global RBAC model and add user-level permission overrides.

For active non-super-admin users, effective permissions are:

`enabled role permissions ∪ direct user allow - direct user deny`.

Active `super_admin` users continue to pass every permission check through the
Go policy helper. Direct permission overrides cannot be edited while a user
currently has the `super_admin` role.

This remains a global action-level permission model. Category/topic scoped ACL
is deferred until the forum module has concrete resource workflows.

## Consequences

- Administrators can precisely grant or remove individual global abilities for
  a user.
- Common access remains managed through user groups.
- Direct denies can override accidental group grants for ordinary users.
- The API remains the final authorization boundary; the Nuxt admin UI only
  improves visibility and ergonomics.
- The policy surface stays small enough to migrate to Casbin or a scoped ACL
  engine later if resource-level rules become necessary.
