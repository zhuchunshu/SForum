# 2026-07-12 Session Handoff

## Goal completed

Built-in role templates for Phase 1 follow-up: `moderator`, `operator`,
`tech_admin` (plus existing `member` / `super_admin` behavior preserved).

## Changed

### API

- `apps/api/app/Models/Identity/seeds.go`
  - `RoleModerator` / `RoleOperator` / `RoleTechAdmin`
  - `SeedRoleTemplates`, `SeedMemberPermissions`
  - `IsBuiltInSystemRole`, `RoleTemplateByKey`
- `apps/api/app/Models/Identity/service.go` — `DeleteRole` locks all built-in
  system roles (not only super_admin)
- `apps/api/database/migrations/202607120002_builtin_role_templates.sql`
- Tests: `seeds_test.go`, `roles_test.go`

### Web

- `apps/web/app/config/roleTemplates.ts` — frontend pack mirror
- `apps/web/app/pages/admin/roles.vue` — apply template UI + toast
- zh-CN / en-US: `admin.roles.applyTemplate*`, `admin.roleCatalog.*`

### Validation / knowledge

- `tests/validate-identity-ui.js` — template + migration + i18n checks
- Decision: `knowledge/decisions/2026-07-12-builtin-role-templates.md`
- Module: `knowledge/modules/identity.md`
- Index latest handoff pointer

## Product choices locked this session

| Question | Choice |
|----------|--------|
| moderator includes `user.ban`? | **Yes** |
| operator includes `user.permission_override`? | **No** |
| independent `tech_admin`? | **Yes** |
| can sites edit template permission sets? | **Yes** (delete still locked; super_admin perms locked) |

## Permission packs (summary)

- **moderator**: admin.access, moderation.review, topic lock/pin/edit_any/delete_any, post edit_any/delete_any, user.ban
- **operator**: admin.access, user.view/manage, settings site/mail/avatar/appearance, forum.settings, seo, category, tag, attachment.manage + attachment.settings
- **tech_admin**: admin.access, extension view/plugin/theme/release, jobs view/manage, search, database, attachment.settings

## Explicitly deferred

- Category-scoped ACL
- One-click “restore template permissions to defaults” API action
- `topic.move` / independent hide keys
- Casbin / ABAC

## Local ops

```bash
./scripts/dev.sh   # runs migrations including 202607120002
```

Then assign users to `moderator` / `operator` / `tech_admin` in Admin → Users.
Re-login after role assignment so session permissions refresh.

## Verification

- `cd apps/api && go test ./app/Models/Identity/ -count=1`
- `node tests/validate-identity-ui.js`

## Next session ideas

1. Optional: admin action to restore a template role’s permission set to
   `SeedRoleTemplates` defaults (clear extras, re-apply pack).
2. Category-scoped moderator ACL when multi-moderator workflows need it.
3. Broader admin menu `*.view` keys if operator packs still feel too coarse.
