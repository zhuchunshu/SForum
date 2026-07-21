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
| Taxonomy | Categories, tags, icons |
| Users / roles / permissions | Members, role templates, matrix, per-user overrides |
| Attachments | Storage provider, governance |
| Mail & notifications | Provider selection, test mail, delivery history |
| Moderation | Reports, pre-publication review |
| SEO | Metadata, robots/sitemap-related controls |
| Extensions | Plugins, themes, settings, event log, trust |

## Permissions

- New admin capabilities should use grantable permission keys  
- Plugins may **declare** keys but must never self-assign them on install/enable  

## Next

- [Forum day-to-day](./forum.md)  
- [Extensions & themes](./extensions.md)  
