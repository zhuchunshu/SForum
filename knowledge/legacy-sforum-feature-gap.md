# Legacy SForum Feature Gap

This note tracks features that exist in `/Users/inkedus/Code/github/SForum-old`
but are not yet implemented, or are only partially implemented, in the current
SForum rewrite. It is intended to guide implementation before importing
SForumData export packages.

## Source Review

Reviewed on 2026-07-07:

- Old route map and domain notes:
  `/Users/inkedus/Code/github/SForum-old/docs/knowledge-base/routes.md`,
  `domain-model.md`, and `plugins-and-extension-points.md`.
- Old plugin code under `app/Plugins/Core`, `User`, `Topic`, `Comment`,
  `Mail`, `Search`, and `SForumData`.
- Old migrations under `/Users/inkedus/Code/github/SForum-old/migrations`.
- Current SForum knowledge notes under `knowledge/modules/` and current route
  controllers under `apps/api/app/Http/Controllers`.

Reviewed on 2026-07-10:

- Old user auth/session implementation:
  `app/Plugins/User/src/AuthGuard.php`, `src/Auth.php`,
  `src/Models/UsersAuth.php`, `src/Controller/Api/AuthOffline*`, and
  `app/Themes/CodeFec/resources/views/user/setting/auth.blade.php`.
- Old admin account-security settings:
  `resources/views/setting/user/core.blade.php` and
  `resources/views/setting/user/sign.blade.php`.

## Current Rewrite Baseline

Already implemented or mostly covered in the current SForum:

- One user system with registration, login/logout, Redis browser sessions,
  first-user `super_admin`, roles, permissions, per-user permission overrides,
  admin user management, role management, and permission matrix.
- ALTCHA-backed human verification for registration when enabled.
- Forum category groups, categories, tags, topic-tag joins, topic list/detail
  API, comment tree API, topic/comment creation API, comment edit/delete API,
  and admin taxonomy/settings pages.
- Runtime web options, personalization, footer settings, SEO settings, and
  public runtime option reads.
- Attachment foundation with provider settings, upload API, admin governance,
  local/OSS/COS/FTP/SFTP adapters, and orphan-cleanup boundaries.
- Extension foundation with upload, manifest validation, plugin enable/disable,
  theme activation, extension settings, event delivery tracking, provider-slot
  direction, and developer scaffolding.
- Read-only admin database table manager.

The gaps below should not be copied blindly from the old PHP architecture. Use
the current permission-aware, plugin-first, beginner-friendly defaults when
implementing them.

## Migration-Blocking Gaps

These affect whether imported legacy data can be displayed or operated on
without large silent loss.

| Area | Old SForum capability | Current gap | Migration impact | Suggested landing |
| --- | --- | --- | --- | --- |
| Topic detail UI | Public `/{id}.html` topic detail with comments and right-side widgets | API exists, but default theme has no topic detail page wired to real comments | Imported topics would list but not have a full reading page | Core forum/theme |
| Composer UI | `/topic/create`, edit page, preview, upload integration, rich toolbar | `SFEditor` exists, topic create API exists, but no public create/edit page flow | Imported users can read but cannot continue normal posting workflow from UI | Core forum/theme |
| Topic lifecycle actions | Edit, delete, restore, lock/unlock, pin/top, essence, admin/author/moderator checks | Schema and permissions exist for some states; endpoints/UI are mostly deferred | Legacy topic status/options cannot be faithfully operated after import | Core forum + moderation |
| Comment operations | Reply, edit, delete, like, star/favorite, adopt/best answer, report | Create/edit/delete API exists; like/star/adopt/report UI and data model missing | Legacy comment likes/adopted state/report refs would be dropped or parked | Core forum + moderation |
| User public profiles | User list, user homepage, topic/comment/follow/fans/collection/order tabs | Admin user management exists; public profile and user content pages missing | Imported authors lack profile pages and profile metadata display | Identity/profile module |
| User profile data | Avatar, signature, background image, custom settings, login/session views | Only core user account/session exists; no profile/settings UI beyond auth pages | Legacy `users_options`, avatars, background images, settings need targets | Identity/profile + attachments |
| Attachments in content | Old uploads tied to users/posts; editor upload endpoints | Attachment system exists, but post/avatar/reference writes are not wired | Files can be imported as attachments but references may not update automatically | Attachments + forum/profile |
| Legacy user roles | `user_class` permissions, hierarchy `permission-value`, black-room behavior | New RBAC exists; hierarchy/scoped old semantics not mapped | Import needs a deterministic user-class to role/status mapping | Identity migration mapping |
| Password reset/email verification | Forgot-password code flow, email verification, optional cleanup for unverified users | Registration exists; password reset and email verification are planned but absent | Legacy password-reset records should not import, but users need recovery path | Identity + mail provider |
| Mail sending | Mail plugin and admin mail configuration | `mail.provider` is planned but no real mail provider slice exists | Password reset, email verification, notifications cannot ship cleanly | Plugin/provider slot |

## High-Value Product Gaps

These are visible forum features that should likely be rebuilt before or soon
after the first serious data migration.

| Area | Old SForum capability | Current gap | Suggested direction |
| --- | --- | --- | --- |
| Search | `/search` searches topic titles and post bodies | Search module is planned; no Meilisearch indexing or search UI | Build Meilisearch-backed public search and rebuild workflow |
| Notifications | Interactive, system, and private-message notices with read/clear APIs | No notification model or delivery/fanout yet | Core notification contracts plus plugin channels |
| Private messages | Private conversations, unread counts, WebSocket route, retention cleanup | No PM module | Treat as a core social feature or a protected plugin with explicit permissions |
| Follow/fans | Follow/unfollow, follower/following pages and counts | No social graph | Add only if user profiles need it for migration parity |
| Collections/favorites | Topic/comment collections and remove APIs | No favorites module | Implement as simple core bookmark/favorite model before importing collections |
| Likes/reactions | Topic likes and comment likes | Feed row has UI affordance, but no persisted reaction model | Define a generic reaction model or narrow topic/comment likes |
| Reports/moderation center | Report topic/comment, report center, admin report review/update/remove | `moderation.report_review` exists, but moderation module is not implemented | Build report workflow before importing `report` rows |
| Scoped moderators | Per-board moderators through `moderator` table | Only global roles/permissions exist | Add resource-scoped moderator assignments if category moderation is required |
| Topic keywords | `/keywords` pages and topic keyword links | Current tags may cover some use cases; no separate keywords | Decide whether to merge old keywords into tags or add a lightweight alias/search feature |
| Polls/votes | `topic_vote` table | No poll/vote module | Defer unless legacy sites depend on it heavily |
| Topic options | Disable comments, only-author view, topic unlock exemptions | No equivalent runtime model | Rebuild only after topic lifecycle and permissions are stable |

## Legacy Auth And Session Lessons

Old SForum has a useful product shape for account security, but its code should
not be copied into the rewrite.

What is worth learning:

- Every successful login creates a durable `users_auth` row with user id,
  random token, login IP, User-Agent, and timestamps. This doubles as login
  history and the active-device list.
- `core_user_session_num` is an admin setting for how many devices may remain
  online at once. Login keeps the newest N records and removes older ones.
- The user account page lists login time, browser, operating system, masked IP,
  IP region, and marks the current device.
- Users can take a single device offline, or take every other device offline
  while preserving the current one.
- Offline actions require an email verification code valid for 10 minutes.
- Admin settings also expose switches around registration/login verification,
  third-party login, email verification, max online devices, and optional IP/UA
  binding checks.

What the rewrite should avoid:

- Do not expose raw session tokens in HTML or emails. Use opaque session ids on
  the server and return only short display fingerprints to the UI.
- Do not import old `users_auth` rows as active security state. Treat them as
  historical audit data at most.
- Do not bind normal sessions strictly to IP by default; mobile networks and
  proxies make that fragile. Prefer risk signals, recent-login notices, and
  reauthentication for sensitive actions.
- Do not let naming drift make settings ambiguous. Use positive option names
  such as `identity.session.bind_user_agent` or `identity.session.max_devices`.

Suggested rewrite direction:

- Add a first-class `user_sessions` or `identity_sessions` model tied to the
  existing Redis browser session flow, storing only a salted session hash,
  display fingerprint, IP prefix, User-Agent summary, created/last-seen times,
  revoked time, and revoke reason.
- Keep Redis-backed browser sessions as the authority for first-party auth, but
  index sessions by user id so the API can revoke a single device, all other
  devices, or all sessions after password reset/security events.
- Move max-device behavior into beginner-friendly runtime options, with safe
  defaults and one-click reset on the admin settings page.
- Add user-facing account security UI before data migration: login history,
  active devices, current-device marker, revoke-device action, and revoke-other
  devices action.
- Add admin-facing audit/search later under a protected permission boundary
  such as `user.manage` or a narrower future account-security permission.

## Plugin-Or-Framework Gaps

Old SForum implemented these in core-ish PHP plugins, but the rewrite should
use provider slots, explicit events, and extension packages.

| Area | Old SForum capability | Current gap | Suggested direction |
| --- | --- | --- | --- |
| Payments | WeChat Pay, Alipay, balance pay, order admin, notify callbacks | No payment core/framework yet | Core payment intents/transactions/entitlements plus provider plugins |
| Wallet/wealth | Balance, golds, credits, exp, recharge, exchange, awards | No wallet/points module | Plugin or optional core framework after payment contract is accepted |
| Paid posts | Paid-content shortcodes and purchase records | No entitlement/paid-content model | Depends on payment framework and content visibility contract |
| OAuth2 login | Admin third-party login settings and OAuth2 service middleware | No external auth providers | Auth provider plugin slot after core account security rules settle |
| Mail | Configurable mail sender plugin | No real mail provider | First full provider-slot slice, as already planned in extensions module |
| Search replacement | Old search was direct DB LIKE; current wants Meilisearch | Search still planned | Use `search.provider`/Meilisearch first, not old LIKE behavior |
| Attachment storage providers | Old local file-store extension points | Current attachment providers exist in core | Future external providers should use `attachment.storage.provider` |

## Lower-Priority Legacy Features

These may be useful later, but should not block the first import unless a
specific old deployment depends on them.

| Area | Old SForum capability | Current gap | Notes |
| --- | --- | --- | --- |
| Invitation codes | Admin create/export and registration gating via invitation codes | No invitation-code module | Could be a registration policy plugin or core option |
| Friend links | Admin friend-link CRUD and shortcode/rendered widget | Footer links exist, but no friend-link model | Import only if public link directories matter |
| Shortcodes | login/reply/password/user/comment/topic-tag/invitation/media/buy/only-author/code/friend-links and alert variants | Tiptap/editor exists, no shortcode renderer or policy | Rebuild allowlisted shortcodes carefully; client HTML remains untrusted |
| Emoji/OwO | Emoji data route and editor integrations | Custom emoji node exists, no legacy emoji catalog endpoint | Can be part of editor polish |
| Daily tasks/check-in | User daily/system tasks and rewards | No tasks/check-in module | Likely optional gamification plugin |
| User awards | `users_award` records | No award/badge model | Could become badges/reputation later |
| Username change log | Username change history | No username change flow yet | Add when account settings support username changes |
| Phone verification | Phone field and verification timestamp | No phone/SMS provider | Defer until SMS provider architecture exists |
| Traditional Chinese locale | Old has `zh_CHT` resources | Current locales are `zh-CN` and `en-US` | Add through runtime language-pack track or built-in locale |
| Custom user code | User custom code setting | No equivalent | Security-sensitive; avoid until a clear sandbox/CSP design exists |

## Operations And Admin Gaps

| Area | Old SForum capability | Current status | Suggested direction |
| --- | --- | --- | --- |
| Admin logs | Admin log viewer and filesystem log cleanup | No admin log UI | Add structured audit/log viewer after audit events are expanded |
| Backups | Admin backup page | `deploy.sh` has backup/restore direction, no web UI | Keep production backup in deployment tooling first |
| Menu management | Admin-managed header/menu entries | Current admin registry and footer links exist, but no public nav editor | Build runtime nav links only if operators need it |
| Custom admin hook components | UI hook/component admin pages | Extension admin pages exist; no arbitrary hook component editor | Prefer extension admin pages and provider slots |
| Plugin update/version check | Old plugin upload, removal, version API checks | Current extension lifecycle lacks upgrade/update checks | Extension v2 backlog already covers upgrade/rollback/uninstall/signatures |
| Theme resource migration | Old theme upload/resource migration | Current Nuxt Layer activation exists | Current approach supersedes old resource migration |
| Maintenance jobs | Auto-lock old topics, auto-delete unverified users, cleanup orders/PM/logs/topic history | Jobs foundation exists, but these jobs are absent | Implement as domain jobs with safe defaults and admin reset controls |
| Data export/import | SForumData export admin page | Current has no importer | Add importer only after target modules above are ready enough |

## Suggested Build Order Before Data Migration

1. **Reader and posting parity**: topic detail page, comments UI, topic
   composer, attachment references, topic edit/delete/lock/pin/hide endpoints.
2. **Profiles and account recovery**: public user profiles, profile/avatar
   settings, password reset, email verification, user login history/offline
   sessions, and mail provider.
3. **Interactions and moderation**: likes/reactions, favorites, reports,
   scoped moderator assignments, notification contracts.
4. **Search**: Meilisearch indexing, rebuild command/job, and public search UI.
5. **Social extras**: follows/fans, private messages, notification center.
6. **Payment/entitlement track**: payment core framework, provider plugins,
   wallet/credits if still desired, paid-content entitlement checks.
7. **Legacy extras**: invitation codes, shortcodes, friend links, tasks,
   awards, OAuth2 login, phone/SMS, Traditional Chinese locale.
8. **Migration tooling**: SForumData package validator, dry-run report,
   staged importer, ID mapping tables, unsupported-data parking report, and
   post-import rebuild jobs.

## Import Policy Notes

- Do not import old session tokens, password-reset tokens, raw authorization
  records, or IP-heavy records as active security state.
- Password hashes need an explicit compatibility decision. Prefer reset-on-login
  or a hash-upgrade adapter rather than assuming the old hash algorithm is safe
  to reuse.
- Old payments, wallet balances, paid-post orders, private messages, and
  notifications should be imported only after their new module/plugin contracts
  exist. Until then, the importer should report them as unsupported instead of
  silently discarding them.
- `topic_keywords` can likely be mapped to tags or search metadata, but this
  needs a product decision before import.
- Old `user_class` should map to current roles and statuses through an explicit
  operator-reviewed mapping file. The old "小黑屋" behavior should map to
  `banned` or a locked-down role, not to display-name hacks.
