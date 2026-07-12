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

- Admin sidebar places **Moderation management** under the Forum management
  folder (`admin.nav.forum`), alongside categories, tags, and forum settings.
- The admin page is policy management plus a read-only complete audit table.
- The frontend `/moderation` workbench has **Pending publication**, **User
  reports**, and **History** tabs. Rows show title/topic, excerpt, author,
  category, time, triggers or report details; expanded context supplies the
  complete content and explicit actions.
- Destructive actions require a review note. Successful actions use the active
  theme color and auto-dismiss after 10 seconds; blocking errors stay visible.
- Pending topic creation routes the author to `/my/content-review`. Pending
  comments show review-submitted feedback and are not inserted into the public
  comment list before approval.
- The public report dialog remains available on topics and comments.

## Non-Goals

- No task claiming, assignment, internal moderator discussion, SLA, custom
  sensitive-word engine, machine-learning risk score, or rate-limit auto-hide.
