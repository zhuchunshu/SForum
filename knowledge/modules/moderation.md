# Moderation Module

User reports and admin moderation queue.

## Backend

- `apps/api/app/Models/Moderation` package: types, store interface, service,
  postgres store, and `ForumTargetValidator` (reuses forum store to check
  target existence/reportability).
- `moderation_reports` table (migration `202607070006`):
  - `reporter_user_id` references `users(id)` set null on delete.
  - `target_type` checked `topic`/`comment`; `target_id` bigint.
  - `reason_code` checked `spam`/`abuse`/`illegal`/`off_topic`/`other`.
  - `status` checked `open`/`reviewing`/`resolved`/`rejected`.
  - `reviewer_user_id` nullable; `review_note` text.
  - Unique index prevents the same reporter from having multiple open reports
    for the same target.

### Endpoints

- `POST /api/v1/moderation/reports` public report creation (login-required
  active user). Validates target exists and is publicly reportable; duplicate
  open report returns 409.
- `GET /api/v1/admin/moderation/reports` admin list with filters for status,
  target type, reporter, and pagination (requires
  `moderation.report_review`).
- `PATCH /api/v1/admin/moderation/reports/{reportID}` update status and review
  note (requires `moderation.report_review`). Resolved/rejected set
  `resolved_at`.

## Frontend

- `/admin/moderation` page with status/type filters, status badges, quick
  actions (mark reviewing/resolved/rejected), review notes, and target
  quick-jump links.
- Report dialog on topic detail page: report buttons on the topic and each
  comment. The dialog is small, clear, uses reason-code chips, and no emoji.
- `useModerationApi` composable + `ModerationReport` types.

## Non-Goals (V1)

- No full notification center or private moderator discussion.
- No automated moderation rules or rate-limit-based auto-hide.
