# Usage

[← English docs home](../README.md)

For **operators**: registration, admin configuration, forum operations, extensions.  
For contributors, see [Development](../development/README.md).

## Chapters

| Doc | Topic |
| --- | --- |
| [First registration & super admin](./first-login.md) | First user, `super_admin`, sessions |
| [Admin control panel](./admin.md) | Entry, settings UX, permissions |
| [Forum day-to-day](./forum.md) | Taxonomy, posting, moderation |
| [Search](./search.md) | Site search vs optional Meilisearch |
| [Extensions & themes](./extensions.md) | Install, enable, trust, activate |

## Operator principles

1. **Recommended defaults first**, with one-click restore where configurable.  
2. **API authorization is authoritative**; hidden UI is UX only.  
3. **Plugin-first** for mail transport, optional search engines, storage vendors, etc.  
4. **Auditable trust** for executable plugins (exact-artifact super-admin confirmation).

## Common entry points (dev defaults)

| Entry | Path |
| --- | --- |
| Public site | `/` |
| Login / register | theme routes (typically `/login`, `/register`) |
| Admin | `/control-panel` (`NUXT_PUBLIC_ADMIN_ROUTE_PREFIX`) |
| Public profile | `/u/:user` |
| Profile settings | `/settings/profile` |
