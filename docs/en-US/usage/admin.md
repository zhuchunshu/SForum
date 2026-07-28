# Admin control panel

[← Usage](./README.md)

## Entry

- Default prefix: `/control-panel`  
- Env: `NUXT_PUBLIC_ADMIN_ROUTE_PREFIX`  
- May also be adjustable via runtime site options  

Users without permission cannot call admin APIs successfully; hiding menus is not security.

## UX conventions

| Capability | Notes |
| --- | --- |
| Multi-tabs | Keep context across admin pages |
| Module registry | Sidebar/page metadata is registry-driven; plugins may declare entries |
| Defaults + restore | Configurable groups should expose recommended values and restore |
| Toasts | Success feedback for save/reset/upload; error toasts stay until dismissed |

## Typical areas

Menus evolve with version and extensions. Common groups:

| Area | Purpose |
| --- | --- |
| Overview / ops | Health, queues, schedules |
| Site / personalization | Branding, nav, footer, announcements, colors |
| Forum settings | Length limits, nesting, cooldowns, guest read, tag policy |
| Content management | Filter topics/comments, edit, inspect restricted revision timelines, and restore a prior version with a reason |
| Taxonomy | Categories, tags, icons |
| Users / roles / permissions | Members, role templates, matrix, per-user overrides |
| Attachments | Storage provider, governance |
| Mail | Mail provider selection, test mail, mail delivery history |
| Notifications | Type/channel policy, Web Push provider, self-test, redacted delivery health |
| Moderation | Reports, pre-publication review |
| SEO | Metadata, robots/sitemap-related controls |
| Extensions | Plugins, themes, settings, event log, trust |

## Permissions

- New admin capabilities should use grantable permission keys  
- Plugins may **declare** keys but must never self-assign them on install/enable  

## Content revision permissions

- `topic.revision.view_any` and `post.revision.view_any` allow only the matching
  history read; they do not grant editing.
- Restore also needs `topic.edit_any` or `post.edit_any` for the same resource.
  Conflict state offers reload or history only, never force overwrite.
- Redacting a history payload is irreversible, `super_admin`-only, and cannot
  target the current version. Raw historical source, restore reasons, and
  attachment-provider details do not enter extension events or ordinary audit logs.

## Next

- [Forum day-to-day](./forum.md)  
- [Extensions & themes](./extensions.md)  
