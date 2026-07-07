# 2026-07-07 Core Forum V1 — Session Handoff

## Changed

Implemented the full Core Forum V1 plan
(`docs/superpowers/plans/2026-07-07-core-forum-v1-task-brief.md`) on branch
`feature/core-forum-v1`. All 8 tasks complete:

- **Task 1 — Topic Lifecycle Backend**: topic update/delete/hide/restore/
  lock/unlock/pin/unpin in `app/Models/Forum`; triple-storage preserved on
  content edit (`post_revisions`); 8 new observe events; OpenAPI + service +
  HTTP tests (allowed/denied).
- **Task 2 — Topic Detail + Comments UI**: `/t/:topicID/:topicSlug` page with
  SSR `useAsyncData`, canonical slug redirect, sanitized HTML, tree/flat
  comment views, permission-aware action UI. `usePermissions` composable,
  `ForumComment` types, `SFComment` HTML rendering + nested slot.
- **Task 3 — Composer/Reply/Edit Flows**: `/topics/new`, `/t/:id/:slug/edit`,
  top-level + nested reply, comment edit/delete. `SFEditor` submits markdown
  (`sourceFormat=markdown`, `editorType=tiptap`, `editorVersion=sf-editor-v1`).
- **Task 4 — Public Profiles**: `user_profiles` migration; `Profile` model
  (public by username, current-user get/update, stats, recent topics);
  `/u/:username` + `/settings/profile` pages; exported `ScanTopicSummary`/
  `RowScanner` for cross-model reuse.
- **Task 5 — Mail + Password Reset**: `Support/Mail` package (noop/dev_log/
  smtp providers); `password_reset_tokens` migration; `PasswordResetService`
  (generic request, single-use confirm); mail options in `web_options`;
  `/forgot-password` + `/reset-password` + `/admin/settings/mail` pages.
- **Task 6 — Reports + Moderation Queue**: `moderation_reports` migration with
  duplicate-open guard; `Moderation` model with forum target validation;
  `/moderation/reports` (public) + `/admin/moderation/reports` (review);
  `/admin/moderation` page + report dialog on topic detail.
- **Task 7 — Verification + Docs**: all tests pass, OpenAPI validated, module
  notes + decision record + this handoff written.

Migrations added: `202607070004_user_profiles.sql`,
`202607070005_password_reset_and_mail_options.sql`,
`202607070006_moderation_reports.sql`.

## Decisions

- Mail provider uses standard-library `net/smtp` (no new dependency) for V1;
  recorded in `knowledge/decisions/2026-07-07-mail-provider-contract.md`.
- `ScanTopicSummary`/`RowScanner` exported from Forum so Profile reuses the
  same SELECT column layout (avoids duplicating topic-summary scanning).
- `GetTopicForAction` loads topic without public visibility filter for
  permission checks (update/delete/lifecycle); public reads still filter.
- Password-reset request never reveals email existence (always 200); mail-send
  failures in that path are swallowed.
- `topics.hidden_at` column was NOT added (hidden status tracked via
  `status='hidden'` + `deleted_at`/`locked_at` only) to preserve the planned
  migration numbering.

## Verification (all pass)

- Backend: `cd apps/api && go test ./...` — 31 packages, 0 failures.
- Frontend tests: `cd apps/web && bun test` — forum tests pass (12/12 in
  forumTopic/forumTaxonomy/adminForum). Pre-existing failures in
  `useApiClient.test.ts` (login/register) and `themeProxy.test.ts`
  (network-dependent, timeouts) are unrelated to this work and present on
  the clean tree.
- Frontend typecheck: `cd apps/web && bun run typecheck` — 0 errors.
- OpenAPI: `ruby scripts/validate-openapi-refs.rb` — 797 refs across 23 files.

## Next

- Manual browser smoke (dev servers): register/login → create topic → reply →
  edit own → lock as admin → confirm reply blocked → report → review in admin
  → request password reset → confirm test mail via dev_log → open/edit profile.
- Wire `attachment_references` for profile avatar uploads (V1 left the
  attachment-ownership check as a documented follow-up).
- Meilisearch indexer integration for topic/comment writes (deferred per plan).
- Search UI and notification center remain non-goals for this release.

## Open Questions

- Should email verification be required before posting in a follow-up?
- When should the legacy SForumData importer land (explicitly out of scope
  here per the plan)?
- Should `net/smtp` be replaced by a richer provider plugin (DKIM, pooling)
  before production, or is V1 SMTP sufficient?
