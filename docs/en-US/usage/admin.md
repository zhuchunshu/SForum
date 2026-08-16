# Admin control panel

[← Usage](./README.md)

## Entry

- Default prefix: `/control-panel`
- Env: `NUXT_PUBLIC_ADMIN_ROUTE_PREFIX` (currently env-only; there is no
  runtime site-option override)

Users without permission cannot call admin APIs successfully; hiding menus is not security.

## UX conventions

| Capability | Notes |
| --- | --- |
| Multi-tabs | Keep context across admin pages |
| Module registry | Sidebar/page metadata is registry-driven; plugins may declare entries |
| Defaults + restore | Configurable groups should expose recommended values and restore |
| Toasts | Success feedback for save/reset/upload; error toasts stay until dismissed |

## Users, roles, and permissions

| Area | Notes |
| --- | --- |
| Users (`/control-panel/users`) | Search and pagination; server-side sorting by joined/updated time, username, display name, email, or status; edit, ban/unban, profile and security state |
| Roles / groups (`/control-panel/roles`) | Role templates, permission matrix, membership; extension permission suggestions grantable per role |
| Permissions (`/control-panel/permissions`) | Permission key catalog and grants/revocations. Plugins may **declare** keys but must never self-assign them on install/enable; Host role management stays authoritative |
| Per-user overrides | Additional per-user grants/restrictions layered over role defaults |

Backend policy is authoritative: hiding buttons is only UX. New admin
capabilities should use grantable permission keys rather than hard-coding
"super admin only", unless security requires it.

## Forum settings, categories, and tags

| Area | Notes |
| --- | --- |
| Forum settings (Site Settings) | Topic/comment length, nesting, posting cooldowns, guest read, registration policy, tag policy, search, guidelines/privacy/terms links |
| Categories (`/control-panel/forum/categories`) | Category tree, groups, icons/colors, slugs, visibility |
| Tags | Tag policy (required?, limits), common tags management |
| Content management | Filter/edit topics and comments, inspect restricted revision timelines, restore a prior version with a reason |

## Moderation and reports

- Moderation workbench (`/control-panel/moderation`): report queue,
  pre-publication review, actions (hide/restore, lock, delete, …), and audit
  trail.
- Reports come from users; staff handle them; handling is audited.
- Content revision permissions: `topic.revision.view_any` and
  `post.revision.view_any` allow only the matching history read; they do not
  grant editing. Restore also needs `topic.edit_any` or `post.edit_any` for
  the same resource.
- Redacting a history payload is irreversible, `super_admin`-only, and cannot
  target the current version. Raw historical source, restore reasons, and
  attachment-provider details do not enter extension events or ordinary audit logs.

## Attachments policy and storage

| Area | Notes |
| --- | --- |
| Attachment Configuration | Basic configuration: upload limits, allowed types; compression configuration: image variants and backfill |
| Attachment Management | File governance, orphan cleanup policy, filtering by user/type |
| Storage providers | Built-in filesystem (`sforum.storage-fs`) and S3-compatible storage (`sforum.storage-s3`; AWS S3 / MinIO / R2); multiple instances, probe, and one-click writer selection. The S3 plugin is a protected built-in; multi-instance roots cannot be selected as writers |

Upload eligibility stays RBAC-driven; role/user policies set per-file size
limits and oversized uploads get a specific 413 response.

## SMTP, mail delivery, and testing

| Area | Notes |
| --- | --- |
| Mail Settings | Select the mail provider (built-in SMTP `sforum.smtp`, …), configure host/port/encryption/from, connection probe |
| Test mail | Send a test message to validate the configuration (test mail is excluded from cooldown/rate limits) |
| Delivery history | Delivery records, failure reasons, attempt counts (retries are background/automatic) |
| Notification mail | Per-type (reply/mention/moderation, …) mail policy and user default preferences |

## Site branding, navigation, footer, and announcements

| Area | Notes |
| --- | --- |
| Personalization (`/control-panel/personalization`) | Site name, logo, favicon (brand SVG upload is safely rasterized to PNG), appearance palettes |
| Site chrome | Topbar, sidebar, mobile, and footer navigation configuration and ordering (revisioned) |
| Announcements | Authoring (time windows, pinned) and display |
| Site settings | Site URL, registration policy, account-security policy, etc. |

## SEO

- `/control-panel/seo`: site-level SEO configuration (metadata, robots,
  sitemap-related controls).
- Public page titles/descriptions/canonicals come from the unified SEO
  resolver; plugins and themes can extend page-level metadata.

## Webhooks

- `/control-panel/webhooks`: endpoint management, event subscriptions, and
  delivery records.
- Outbound delivery runs as a background job (River queue); failures retry
  **automatically in the background** per queue policy. The admin page only
  lists deliveries (including attempt counts); there is **no manual retry
  button**.
- Inbound `POST /webhooks/inbound/{source}` is currently a gateway skeleton
  only: it acknowledges non-empty bodies and skips CSRF; plugin verify/parse
  hooks are not wired yet.

## System updates and background jobs

| Area | Notes |
| --- | --- |
| System updates | In-admin discovery with results cached for up to 5 minutes, plus a manual check button, update prompt, and guidance; actual production updates run through `upgrade.sh` / `deploy.sh` (see [deployment](../deployment.md)) |
| Jobs (`/control-panel/jobs`) | River queue state, pause/resume queues, job detail/retry/cancel |
| Schedules (`/control-panel/schedules`) | Scheduled jobs and manual triggers |
| Database (`/control-panel/database`) | Read-only table inspection and metadata (requires `database.manage`) |

## Permissions

- New admin capabilities should use grantable permission keys
- Plugins may **declare** keys but must never self-assign them on install/enable

## Next

- [Account & security](./account-security.md)
- [Forum day-to-day](./forum.md)
- [Extensions & themes](./extensions.md)
