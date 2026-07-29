# Moderation Module

Configurable pre-publication review, user reports, admin management, and the
frontend moderator workbench.

## Permissions And Entry Points

- `moderation.manage` controls the admin **Moderation management** page under
  the Forum management sidebar folder. It can read/update/reset policy settings
  and read the complete audit history, but it does not grant review actions.
- `moderation.review` controls the frontend `/moderation` workbench and all
  review actions. It does not grant policy management.
- `moderation.view_ip` reveals full client IP fields (`ipAddress`,
  `lastEditIp`) on workbench pending items, report items, and review context.
  Without it, the API strips those fields even for reviewers. The moderator
  role template includes it by default.
- The review and policy permissions are intentionally independent.
  `super_admin` still passes them through the existing super-admin policy.

## Backend

- `moderation_reports` stores individual user reports. The same reporter cannot
  create multiple open reports for one target.
- `moderation_settings` is a singleton policy record. Modes are `off`,
  `rules`, and `all`; `off` is the upgrade-safe stored default while the admin
  UI recommends `rules` for ordinary operation.
- Built-in rule mode can review new-user content and explicit external HTTP(S)
  links. The site hostname is excluded. More advanced risk/sensitive-content
  detection belongs behind stable filters/events or plugins.
- Topics and comments support `pending` and `rejected`. Neither status is
  returned by public reads, counted in public statistics, or indexed. Approval
  updates counters and search derivatives after the decision transaction.
- Forum content revisions V1 is complete: topic/comment creation writes version
  1; authorized history remains limited to `topic.revision.view_any` /
  `post.revision.view_any`; edits use CAS and accepted snapshots; restore is
  append-only. The moderation boundary remains unchanged: restore can re-render
  and re-evaluate the current pipeline but must not publish or change `pending`,
  `rejected`, `hidden`, or `deleted` lifecycle state. Rejected/failed edits
  create no revision or audit success record.
- `moderation_decisions` stores immutable action, reviewer, note, trigger
  snapshot, target, source, and timestamp audit records.

### Endpoints

- `POST /api/v1/moderation/reports` remains the login-required report entry.
- `/api/v1/admin/moderation/settings`, `/settings/reset`, and `/decisions`
  require `moderation.manage`.
- `/api/v1/moderation/workbench/*` exposes counts, pending content, enriched
  reports, history, complete safe-rendered context, and explicit decisions;
  all endpoints require `moderation.review`.
- `GET /api/v1/me/content-review` is login-required and returns only the
  current author's pending/rejected topics and comments with rejection notes.

## Frontend

- `moderation.review` remains a Core-owned, non-replaceable workbench. It is
  separately `themeable`: the active theme may provide the public navbar/body/
  footer L1 shell only when it embeds the required `sf-moderation-review` Host
  island. Middleware, permissions, state, API calls, rendering safety, and
  decisions stay inside `SFModerationReviewPage` and Core APIs.
- Admin sidebar places **Moderation management** under the Forum management
  folder (`admin.nav.forum`), alongside categories, tags, and forum settings.
- The admin page is policy management plus a read-only complete audit table.
- The frontend `/moderation` workbench has **Pending publication**, **User
  reports**, and **History** sources. Layout reuses the public three-column
  chrome from home/notifications: left shell is `SFHomeNavigation` (route mode)
  plus workbench sources/type filters in `#after-navigation`; right rail uses
  the same section stack language (large overview number + `dl` stats + help
  copy in queue mode; decision rail in review mode). Mobile drawers share
  `forum-mobile-menu-open` / `forum-mobile-info-open` with other public pages.
  Queue mode is URL-backed by source, content type, and page; review mode adds
  stable item query fields and keeps navigation inside the current
  source/filter/page, including history items whose source is a report. Rows
  are fully clickable, context uses the existing safe-rendered HTML path, and
  history remains read-only.
- The workbench center column uses `--sf-public-surface`; the surrounding page
  canvas keeps `--sf-public-bg`. This matches profile settings and preserves
  the active theme's intended foreground/background contrast.
- Workbench CSS is registered in Nuxt's initial global stylesheet set rather
  than the async theme island, so a hard refresh cannot briefly inherit the
  generic oversized `h1` rule before the moderation styles arrive.
- Destructive actions require a review note. Successful actions use the active
  theme color and auto-dismiss after 10 seconds; blocking errors stay visible.
- Pending topic creation shows a toast and returns the author home; review
  outcomes arrive via notifications. `GET /api/v1/me/content-review` remains
  available for API/clients (no first-party `/my` page). Pending comments show
  review-submitted feedback and are not inserted into the public comment list
  before approval.
- The public report dialog remains available on topics and comments.

## Non-Goals

- No task claiming, assignment, internal moderator discussion, SLA, custom
  sensitive-word engine, machine-learning risk score, or rate-limit auto-hide.
