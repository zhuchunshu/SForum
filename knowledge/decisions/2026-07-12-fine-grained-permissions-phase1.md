# Decision: Fine-Grained Global Permissions (Phase 1)

## Status

Accepted

## Context

The core permission catalog was too coarse for safe role design:

- `settings.manage` covered site, mail, avatar, appearance, and forum runtime
  options.
- `extension.manage` covered listing, plugin lifecycle, themes, and web releases.
- `user.manage` covered listing users, role assignment, and permission overrides.
- Author topic edit/delete reused `post.edit_own` / `post.delete_own`, which
  made labels and capability boundaries unclear.

Category/topic scoped ACL is still deferred until multi-moderator workflows
require it.

## Decision

Keep global action-level RBAC. Split high-risk parent permissions into
grantable child keys, and fix author topic ownership keys.

### New keys

- Forum: `topic.edit_own`, `topic.delete_own`
- Settings: `settings.site.manage`, `settings.mail.manage`,
  `settings.avatar.manage`, `settings.appearance.manage`,
  `forum.settings.manage`
- Users: `user.view`, `user.permission_override`
- Extensions: `extension.view`, `extension.plugin.manage`,
  `extension.theme.manage`, `extension.release.manage`

### Compatibility

- Legacy parents remain in the catalog:
  `settings.manage`, `extension.manage`, `user.manage`.
- Effective permission expansion is one-way: holding a parent also satisfies
  child checks and expands the session permission list with children.
- Migration grants children to roles/users that already hold the parent, and
  grants `topic.edit_own` / `topic.delete_own` to roles that hold
  `post.edit_own` / `post.delete_own`.
- `member` receives the new topic own keys by default.

### Deferred

- Category-scoped moderator ACL
- Built-in `moderator` / `operator` role templates (next iteration)
- `topic.move` / independent hide keys

## Consequences

- Operators can separate content moderation, site ops, and technical admin.
- Existing deployments do not suddenly lose access after upgrade.
- Frontend menus and option `managePermission` bindings must use child keys.
- New permissions require seed + migration + zh-CN/en-US catalog text + tests.
