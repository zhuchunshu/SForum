# 2026-07-12 Session Handoff

## Goal for next session

Implement **Phase 1 follow-up: built-in role templates** so operators can apply
safe permission packs without hand-picking dozens of fine-grained keys.

Do **not** start category-scoped ACL yet.

## Already done (on `main`, 4 commits)

```
1ae6b42e4 fix(extensions): require release permission when plugin enable queues web release
0db8b705e feat(web): wire admin UI to fine-grained permissions
b4315ef72 feat(api): enforce fine-grained permission checks
aa0bd2cdd feat(identity): add fine-grained permission catalog phase 1
```

Working tree should be clean on `main`.

### What shipped

1. **Fine-grained global permissions (Phase 1)**
   - Decision: `knowledge/decisions/2026-07-12-fine-grained-permissions-phase1.md`
   - Catalog: `apps/api/app/Models/Identity/seeds.go`
   - Compat expand parent→child: `apps/api/app/Models/Identity/permission_compat.go`
   - `Actor.Can` uses expansion; session/admin lists expand via
     `ExpandEffectivePermissions` in `postgres_store.go`
   - Migration: `apps/api/database/migrations/202607120001_fine_grained_permissions_phase1.sql`
   - Forum author topic edit/delete now uses `topic.edit_own` / `topic.delete_own`
     (not `post.edit_own` / `post.delete_own`)
   - Options `managePermission` rebound to child keys (site/mail/avatar/appearance/forum)
   - Extension surfaces split: view / plugin / theme / release
   - User list/detail: `user.view`; overrides: `user.permission_override`;
     role assign / force logout still `user.manage`
   - Web menus: `apps/web/app/config/adminModules.ts`
   - Frontend can() legacy fallback: `apps/web/app/composables/usePermissions.ts`
   - i18n catalog updated for zh-CN/en-US
   - Identity UI validator requires every SeedPermission key to have i18n

### Compatibility rules (do not break)

- Legacy parents remain: `settings.manage`, `user.manage`, `extension.manage`
- Holding parent ⇒ `Can(child)` true and session list includes children
- Holding only child does **not** imply parent
- Migration backfills children for roles/users that already hold parents
- member gets `topic.edit_own` + `topic.delete_own` by default

### Local ops note

After pull, run migrations and **re-login** so session permissions expand.

```bash
./scripts/dev.sh   # or existing migrate path
```

## Next: built-in role templates

### Product intent

Ship system (or seed) roles that map cleanly to common operator jobs:

| Role key (suggested) | Purpose | Permission pack (suggested) |
|----------------------|---------|-----------------------------|
| `member` | already exists | create + own topic/post edit/delete (already mostly done) |
| `moderator` | content ops | `moderation.review`, `topic.lock`, `topic.pin`, `topic.edit_any`, `topic.delete_any`, `post.edit_any`, `post.delete_any`, optional `user.ban` |
| `operator` | site ops, no tech root | `user.view`, `user.manage` (maybe no override), `settings.site/mail/avatar/appearance`, `forum.settings.manage`, `seo.manage`, `category.manage`, `tag.manage`, `attachment.manage`, `attachment.settings.manage` optional, **no** extension.release / database / jobs.manage |
| `tech_admin` (optional) | technical | `extension.view/plugin/theme/release`, `jobs.view/manage`, `search.manage`, `database.manage`, `attachment.settings.manage` |

`super_admin` stays protected all-permissions; do not make it editable.

### Implementation guidance

1. Prefer **seed roles via migration** (like other permissions), not one-off UI-only packs.
2. Roles should be system-ish:
   - not deletable, or clearly marked system
   - alias/description localized in admin UI if possible
3. Admin UX (roles page):
   - show templates as selectable packs, or “apply template” when creating a role
   - still allow custom roles
4. Do **not** remove fine keys; templates are convenience on top of Phase 1.
5. Update:
   - `seeds.go` / migration
   - zh-CN/en-US role display if needed
   - `knowledge/modules/identity.md`
   - tests for default role permission sets
6. Keep permission-aware discipline: API remains authority.

### Explicitly deferred (later sessions)

- `topic.move` / independent hide
- Category-scoped moderator ACL
- Casbin / ABAC
- Per-page `*.view` permissions for every admin menu

## Key files

| Area | Path |
|------|------|
| Decision | `knowledge/decisions/2026-07-12-fine-grained-permissions-phase1.md` |
| Seeds | `apps/api/app/Models/Identity/seeds.go` |
| Compat | `apps/api/app/Models/Identity/permission_compat.go` |
| Policy | `apps/api/app/Models/Identity/policy.go` |
| Migration | `apps/api/database/migrations/202607120001_fine_grained_permissions_phase1.sql` |
| Forum own | `apps/api/app/Models/Forum/service.go` (`canEditTopic` / `canDeleteTopic`) |
| Options perms | `apps/api/app/Models/Options/service.go` + community/site option files |
| Extensions | `apps/api/app/Models/Extensions/service.go` helpers `canViewExtensions` / `canManagePlugins` / `canManageThemes` / `canManageReleases` |
| Admin nav | `apps/web/app/config/adminModules.ts` |
| Frontend can | `apps/web/app/composables/usePermissions.ts` |
| i18n catalog | `apps/web/i18n/locales/zh-CN.json` / `en-US.json` → `admin.permissionCatalog` |
| Identity module | `knowledge/modules/identity.md` |
| Agents rules | `Agents.md` |

## Suggested new-session prompt

```text
继续 SForum 权限 Phase 1 后续：实现内置角色模板（moderator / operator，可选 tech_admin）。

先读：
- knowledge/sessions/2026-07-12-fine-grained-permissions-phase1-handoff.md
- knowledge/decisions/2026-07-12-fine-grained-permissions-phase1.md
- apps/api/app/Models/Identity/seeds.go
- knowledge/modules/identity.md

约束：
- 在 main 上做，拆成可回滚的 git commit
- 不要做版块级 ACL
- 保留 settings.manage / user.manage / extension.manage 兼容父权限
- 模板角色用 migration seed；super_admin 仍不可编辑权限
- 更新 i18n、测试、knowledge

做完后跑相关 go test 与 tests/validate-identity-ui.js。
```

## Open questions (ask user if unclear)

1. `moderator` 是否默认包含 `user.ban`？
2. `operator` 是否包含 `user.permission_override`？（建议 **否**）
3. 是否需要独立 `tech_admin`，还是 operator + 手工加 tech keys？
4. 模板角色是否允许站点改其权限集合，还是固定 system pack + 复制为自定义角色？

## Verification already run

- `go test` Identity / Forum / Options / Extensions / Forum controller / Identity controller — pass
- `node tests/validate-identity-ui.js` — pass
- `bun test` adminMailNotifications + adminForum — pass
