# Decision: Built-in Role Templates (Phase 1 Follow-up)

## Status

Accepted

## Context

Fine-grained global permissions (Phase 1) make safe role design possible, but
operators still need ready-made packs for common jobs. Hand-picking dozens of
keys for every site is error-prone.

Category-scoped ACL remains deferred.

## Decision

Ship three **system** roles via migration seed, with source-of-truth packs in
`apps/api/app/Models/Identity/seeds.go` (`SeedRoleTemplates`):

| Role key | Purpose | Notable grants | Explicitly excluded |
|----------|---------|----------------|---------------------|
| `moderator` | Content ops | `moderation.review`, topic/post any edit/delete, lock/pin, `user.ban`, `admin.access` | Settings, extensions, role.manage, permission overrides |
| `operator` | Site ops | `user.view` + `user.manage`, settings children, forum/SEO/category/tag/attachment manage, `admin.access` | `user.permission_override`, extension release, database, jobs, role.manage |
| `tech_admin` | Technical ops | extension view/plugin/theme/release, jobs view/manage, search, database, attachment settings, `admin.access` | user.manage, moderation, site settings |

Also keep existing:

- `super_admin` — all permissions; **not** permission-editable; not deletable
- `member` — default registration group; narrow create/own edit-delete pack

### System role policy

- Templates are `is_system=true`, `is_deletable=false`.
- Alias/description remain editable (same as `member`).
- Permission sets for templates **are** editable so operators can tune packs;
  only `super_admin` stays permission-locked.
- `DeleteRole` rejects all built-in system keys via `IsBuiltInSystemRole`.
- Migration uses `ON CONFLICT DO NOTHING` for role_permissions so upgrades
  **add missing** template grants without stripping site customizations.
- Role row conflict updates only `is_system` / `is_deletable` / `is_enabled`,
  not alias/description (preserve operator renames).

### Admin UX

- Roles admin page can **apply template** packs when creating or editing a
  custom (or non-super-admin) group: fills checkboxes; on create also prefills
  localized alias/description (not the system key).
- Frontend pack copy lives in `apps/web/app/config/roleTemplates.ts` and
  `admin.roleCatalog.*` i18n; must stay aligned with `SeedRoleTemplates`.

### Compatibility

- Legacy parents `settings.manage` / `user.manage` / `extension.manage` remain.
- Templates grant **child** keys (or explicit manage keys), not parent bags,
  so operators do not accidentally inherit override/release powers via parent
  expansion unless they later grant the parent themselves.

## Consequences

- New installs get assignable moderator/operator/tech packs immediately after
  migrate.
- Existing sites gain the three roles without wiping custom role permissions.
- Operators still create custom groups; templates are convenience, not the only
  path.
- Category-scoped moderator ACL remains a later phase.
