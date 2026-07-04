# Identity And Permissions Design

## Goal

Design the SForum identity and permissions foundation for a community forum:
one user system for public users and staff users, open registration by default,
safe first-user bootstrapping, and admin-managed custom roles.

## Context

SForum is a forum product, not a separate front-office/back-office business
system. Administrators, moderators, and regular members are all forum users.
They should share one profile, one login session, one audit trail, and one
permission model. Admin screens are protected areas of the same product rather
than a second account system.

The existing architecture uses:

- Nuxt 4 for the web application.
- Go Fiber v3 for the API.
- PostgreSQL as the source of truth.
- Redis-backed browser sessions.
- `pgx + sqlc + goose` for database access and migrations.

## Library Survey

The permission model should start with a database-backed RBAC model and small
Go policy helpers.

Options considered:

- In-code policies over database roles and permissions.
- Casbin for RBAC/ABAC policy enforcement.
- OPA/Cedar-style policy engines.
- Relationship-based authorization systems such as SpiceDB.

Recommendation: use database-backed RBAC plus Go policy functions for the MVP.
This keeps the first forum implementation simple, inspectable, and easy to
test. The policy interface should be narrow enough that Casbin can replace or
augment the policy implementation later if category-scoped roles or complex
permission rules grow beyond ordinary forum needs.

## Core Decisions

- Use one `users` table for all people.
- Open registration is the default product behavior after bootstrapping.
- The first successfully registered user becomes the initial super
  administrator.
- Later registered users receive the system `member` role by default.
- Admin and moderation access come from roles and permissions, not from a
  separate admin user table.
- Role permissions are additive. MVP authorization does not need explicit deny
  rules.
- The Fiber API is the final authorization authority.
- Nuxt may hide admin navigation and protect `/admin/*` routes for user
  experience, but every protected API action must still check permissions.

## System Roles

SForum ships with at least two system roles:

| Role key | Purpose | Editable alias | Deletable | Removable from protected users |
| --- | --- | --- | --- | --- |
| `super_admin` | Owns all permissions and system recovery power. | Yes | No | No for the initial super admin |
| `member` | Default role for open registration. | Yes | No | No while it remains the configured default role |

Role keys are stable identifiers used by code and policies. Administrators may
change role aliases/display names, such as renaming `member` to "普通会员",
but they may not change the role key or delete system roles.

The initial super administrator cannot be deleted, disabled, or stripped of the
`super_admin` role. This protects system recovery even if later role changes
are misconfigured.

## Custom Roles

The admin area should support custom role management, also presented as user
groups in the UI.

Administrators with role-management permission can:

- Create custom roles.
- Set a role alias/display name and description.
- Grant or revoke permissions on custom roles.
- Assign roles to users.
- Disable a custom role when it should no longer grant permissions.
- Delete a custom role only when it is not a system role and has no assignments,
  or after assignments are explicitly migrated.

Users can have multiple roles. Effective permissions are the union of all
enabled role permissions. This keeps common forum cases simple: a moderator can
remain a member while also receiving moderation abilities.

## Suggested Permission Names

Use stable action-style permission keys. Initial examples:

- `admin.access`
- `role.manage`
- `user.manage`
- `user.ban`
- `category.manage`
- `topic.create`
- `topic.edit_any`
- `topic.delete_any`
- `topic.lock`
- `topic.pin`
- `post.create`
- `post.edit_own`
- `post.edit_any`
- `post.delete_own`
- `post.delete_any`
- `moderation.report_review`
- `settings.manage`

The exact seed list can grow with feature implementation. Avoid adding
permissions before an endpoint or workflow needs them.

## Data Model

Recommended tables:

- `users`: canonical account, profile, status, locale preference, and protected
  bootstrap flags.
- `user_credentials`: password hash and password metadata.
- `roles`: stable role key, alias/display name, description, system flags, and
  enabled status.
- `permissions`: stable permission key, module, description.
- `role_permissions`: many-to-many relation from roles to permissions.
- `user_roles`: many-to-many relation from users to roles.
- `sessions`: Redis-owned session data, with PostgreSQL audit records only when
  required later.
- `audit_events`: important identity and permission changes.

Recommended `roles` fields:

- `id`
- `key`
- `alias`
- `description`
- `is_system`
- `is_default`
- `is_deletable`
- `is_enabled`
- `created_at`
- `updated_at`

Recommended `users` fields for protection:

- `id`
- `username`
- `email`
- `display_name`
- `locale`
- `status`
- `is_initial_super_admin`
- `created_at`
- `updated_at`

`is_initial_super_admin` should be true for exactly one user: the first
registered account. Deletion, disabling, and role-removal flows must reject
changes that would compromise this account.

## Registration Flow

Registration should be open by default.

1. User submits username, email, password, and locale preference.
2. API validates input, rate limits the attempt, and verifies the required
   human-verification token.
3. API starts a PostgreSQL transaction.
4. API takes a transaction-scoped bootstrap lock to prevent two concurrent
   first registrations.
5. API inserts the user and password hash.
6. If no user existed before this transaction, assign `super_admin` and
   `member`, and mark `is_initial_super_admin = true`.
7. Otherwise assign the current default role, initially `member`.
8. API creates a Redis-backed browser session.
9. API returns the current user summary and effective permissions needed by the
   frontend shell.

Email verification can be added later without changing the role model. If email
verification is required before posting, the account may still be created with
`member`, while post-creation permissions check verification status.

## Human Verification And Anti-Automation

SForum keeps human verification disabled by default and supports ALTCHA as the
first self-hosted provider for open registration and password-reset initiation
when a deployment explicitly enables it. Verification is not a replacement for
rate limiting or moderation; it is one layer that increases the cost of
automated abuse.

Backend responsibilities:

- Generate fresh ALTCHA challenges from the Fiber API.
- Verify submitted ALTCHA payloads on the server.
- Store short-lived challenge/replay state in Redis.
- Rate limit challenge generation and protected form submissions by IP,
  account identifier, session, and action.
- Return stable error codes such as `human_verification.required`,
  `human_verification.invalid`, `human_verification.expired`,
  `human_verification.replayed`, and `rate_limit.exceeded`.

Frontend responsibilities:

- Render the ALTCHA widget on registration when the public provider is
  `altcha`.
- Render it on password-reset initiation when that flow exists and verification
  is enabled.
- Render it on login only after the API reports a challenge is required for the
  current risk state.
- Localize all verification and rate-limit errors in Simplified Chinese and
  English.

Keep the backend provider interface narrow so a deployment can add Cloudflare
Turnstile later without rewriting identity handlers.

## Authorization Flow

For authenticated requests:

1. Session middleware loads the user ID from Redis.
2. API loads user status, roles, and effective permissions.
3. Policy helpers evaluate the required permission and any resource-specific
   ownership rules.
4. Handlers call the relevant policy before executing writes or protected reads.

Recommended policy shape:

```go
type Actor struct {
	ID          int64
	Status      UserStatus
	RoleKeys    []string
	Permissions map[string]bool
}

func (a Actor) IsSuperAdmin() bool
func (a Actor) Can(permission string) bool
func CanEditPost(actor Actor, post PostSummary) bool
```

`IsSuperAdmin()` should short-circuit permission checks. Ownership-sensitive
rules such as `post.edit_own` should live in named policy helpers instead of
being repeated in handlers.

## Admin UI

The admin interface lives in the same Nuxt application under protected routes,
for example `/admin/*`.

Expected first admin pages:

- Role list.
- Role create/edit form.
- Permission assignment matrix.
- User role assignment.
- Audit log for identity and permission changes.

Nuxt should request the current user and effective permissions during app
initialization or route middleware. It may hide navigation items the actor
cannot use. It must not be trusted as the only permission check.

## API Surface

Initial endpoints can be added under `/api/v1`:

- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/logout`
- `GET /auth/session`
- `GET /roles`
- `POST /roles`
- `PATCH /roles/{roleKey}`
- `DELETE /roles/{roleKey}`
- `PUT /roles/{roleKey}/permissions`
- `PUT /users/{userID}/roles`

Protected endpoints should return stable error codes such as:

- `auth.required`
- `auth.invalid_credentials`
- `permission.denied`
- `role.system_role_locked`
- `role.default_role_locked`
- `user.initial_super_admin_locked`

Frontend translations should map these codes to Simplified Chinese and English
messages.

## Error Handling

- Return `401` when no valid session exists.
- Return `403` when the actor is authenticated but lacks permission.
- Return `409` when a request conflicts with system invariants, such as
  deleting `member` or disabling the initial super administrator.
- Return `422` for validation errors.
- Audit every successful role, permission, and protected-user change.
- Audit rejected attempts to modify the initial super administrator when the
  actor was authenticated.

## Testing Strategy

Backend tests should cover:

- First registration receives `super_admin` and `member`.
- Second registration receives only the configured default role.
- Concurrent first-registration attempts produce exactly one initial super
  administrator.
- Registration requires a valid single-use human-verification token.
- Replayed, expired, missing, or invalid human-verification tokens fail with
  stable error codes.
- Challenge generation and registration submission are rate limited.
- `member` alias can change while `member` key cannot change.
- `member` cannot be deleted while it is the default registration role.
- The initial super administrator cannot be deleted, disabled, or stripped of
  `super_admin`.
- Custom roles can grant permissions additively.
- `super_admin` passes every permission check.
- Non-admin users cannot access role-management endpoints.

Frontend tests should cover:

- Anonymous users can reach registration.
- Members do not see admin navigation.
- Users with `admin.access` can enter `/admin/*`.
- API `403` responses are shown as localized permission-denied messages.

## Out Of Scope For The First Identity Milestone

- Category-scoped moderator assignments.
- Explicit deny permissions.
- Organization or tenant boundaries.
- Social login.
- Two-factor authentication.
- Third-party CAPTCHA provider implementation.
- Full notification/email delivery.
- Casbin or another policy engine integration.

These can be introduced later without changing the one-user-system principle.
