# Core Forum V1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. If the worker does not have these skills, treat each task as an independent checklist and stop for review after every task.

**Goal:** Build SForum's first usable forum loop before any SForumData import work: read topics, reply, publish, edit basic content, moderate topics, show public profiles, recover passwords by mail, and review user reports.

**Architecture:** Keep SForum core as the host framework. Forum, identity, moderation, mail, and profile behavior should live behind explicit Go services, permission checks, OpenAPI contracts, Nuxt pages, and stable extension/provider boundaries. Do not clone old SForum features wholesale; implement only the smallest coherent product surface that lets a new forum operate safely.

**Tech Stack:** Go Fiber v3, PostgreSQL/goose/sqlc style stores where already used, Redis-backed sessions, existing RBAC policy helpers, modular OpenAPI, Nuxt 4/Vue 3/Nuxt UI, default built-in Nuxt Layer theme, existing SF component library, existing `web_options`, existing attachment foundation, existing human-verification boundary.

---

## Must Read First

- `AGENTS.md`
- `knowledge/index.md`
- `knowledge/archive/legacy-sforum-feature-gap.md`
- `knowledge/modules/forum.md`
- `knowledge/modules/identity.md`
- `knowledge/modules/frontend.md`
- `knowledge/modules/attachments.md`
- `knowledge/modules/extensions.md`
- `knowledge/decisions/2026-07-06-core-framework-plugin-first-architecture.md`
- `knowledge/decisions/2026-07-06-forum-topics-comments-posts.md`
- `knowledge/decisions/2026-07-06-tiptap-editor-content-storage.md`
- `docs/roadmap.md`
- `contracts/openapi.yaml`
- `apps/api/app/Http/Controllers/Forum/routes.go`
- `apps/api/app/Models/Forum/service.go`
- `apps/api/app/Models/Forum/store.go`
- `apps/api/app/Models/Forum/types.go`
- `apps/web/app/composables/useForumApi.ts`
- `apps/web/app/utils/forumTaxonomy.ts`
- `apps/web/app/components/SFEditor.vue`
- `apps/web/app/components/SFComment.vue`
- `apps/web/app/config/adminModules.ts`
- `extensions/builtin/themes/sforum-default/layer/app/pages/index.vue`

## Hard Scope

Implement these first-version capabilities:

- Topic detail page at `/t/:topicID/:topicSlug`, including sanitized topic content, tags, category link, author link, status badges, comments, reply form, SEO metadata, and canonical slug handling.
- Topic/comment composer flows using the existing `SFEditor`: create topic, reply to topic/comment, edit own topic, edit own comment.
- Basic topic lifecycle operations: soft delete, hide/restore for moderators, lock/unlock, pin/unpin, and permission-aware UI actions.
- Public member profile page and current-user profile settings for basic display fields.
- Password reset by email with a minimal mail provider contract, safe defaults, SMTP/dev providers, and optional human verification for password-reset initiation.
- Report topic/comment and an admin moderation queue for review, status update, and quick jump to the reported resource.

## Non-Goals

- No SForumData importer in this plan.
- No payments, wallet, credits, paid posts, orders, or entitlement system.
- No private messages, follows/fans, check-in/tasks, invitation codes, friend links, polls, awards, phone/SMS, OAuth2, or shortcode renderer.
- No full notification center.
- No Meilisearch search UI or index rebuild.
- No role-scoped category permissions.
- No replacement of the extension runtime.
- No route monkey-patching or plugin override of core routes.

## Global Rules For Every Task

- API policy checks are authoritative. Frontend guards and hidden buttons are only UX helpers.
- Use the existing API envelope: `{"code": <http status>, "message": "...", "data": ...}`.
- Update OpenAPI for every new or changed endpoint.
- Add allowed and denied tests for unsafe endpoints and admin operations.
- Keep public pages in `extensions/builtin/themes/sforum-default/layer/app/pages`.
- Keep reusable app-level components, composables, utilities, i18n, and admin pages in `apps/web/app`.
- Do not grow `apps/web/app/pages/admin/settings/index.vue`; it is already near 1000 lines. Add a separate admin page or focused components for mail settings.
- Use Nuxt Icon/Lucide icons. Do not use emoji as icons or status markers.
- Use translation keys for user-facing strings.
- Keep comments useful and prefer Chinese comments in project code.
- Do not kill the user's port 3000 dev server.
- Before network-dependent dependency commands, use the project proxy environment from `AGENTS.md`.

## Suggested API Shape

Use these endpoint names unless local code review reveals a stronger existing convention:

- `PATCH /api/v1/topics/{topicID}` updates title/category/tags/content.
- `DELETE /api/v1/topics/{topicID}` soft-deletes a topic.
- `POST /api/v1/topics/{topicID}/hide`
- `POST /api/v1/topics/{topicID}/restore`
- `POST /api/v1/topics/{topicID}/lock`
- `POST /api/v1/topics/{topicID}/unlock`
- `POST /api/v1/topics/{topicID}/pin`
- `POST /api/v1/topics/{topicID}/unpin`
- `GET /api/v1/profiles/{username}` reads a public profile.
- `GET /api/v1/profile` reads the current user's editable profile.
- `PUT /api/v1/profile` updates the current user's editable profile.
- `POST /api/v1/auth/password-reset/request`
- `POST /api/v1/auth/password-reset/confirm`
- `POST /api/v1/admin/mail/test`
- `POST /api/v1/moderation/reports`
- `GET /api/v1/admin/moderation/reports`
- `PATCH /api/v1/admin/moderation/reports/{reportID}`

## Permission Model

- Topic create: existing `topic.create`.
- Comment create: existing `post.create`.
- Own topic edit: author plus existing `post.edit_own`.
- Any topic edit: existing `topic.edit_any`.
- Own topic delete: author plus existing `post.delete_own`.
- Any topic delete, hide, and restore: existing `topic.delete_any`.
- Lock/unlock: existing `topic.lock`.
- Pin/unpin: existing `topic.pin`.
- Own comment edit/delete: existing `post.edit_own` and `post.delete_own`.
- Any comment edit/delete: existing `post.edit_any` and `post.delete_any`.
- Profile update: login-required current user only; no new permission.
- Mail settings and test email: existing `settings.manage`.
- Password reset request/confirm: public, rate-limited, and optionally human-verification protected.
- Report create: login-required active user.
- Report review: existing `moderation.report_review`.

## Implementation Tasks

### Task 0: Preflight And Guardrails

**Files:**
- Read: files listed in "Must Read First"
- Inspect: `git status --short`

- [ ] Confirm there are unrelated dirty files before editing. Do not revert unrelated work.
- [ ] Confirm current migrations end at `202607070003_forum_taxonomy.sql`; if new migration filenames below already exist, use the next monotonically increasing timestamp.
- [ ] Run `rg -n "topic.edit_any|topic.delete_any|topic.lock|topic.pin|moderation.report_review|post.edit_own|post.delete_own" apps/api` and confirm existing permission keys before adding any new key.
- [ ] Keep implementation commits task-sized. Suggested commit messages are listed at the end of each task.

### Task 1: Topic Lifecycle Backend

**Files:**
- Modify: `apps/api/app/Models/Forum/types.go`
- Modify: `apps/api/app/Models/Forum/store.go`
- Modify: `apps/api/app/Models/Forum/service.go`
- Modify: `apps/api/app/Models/Forum/postgres_store.go`
- Modify: `apps/api/app/Http/Controllers/Forum/routes.go`
- Modify: `apps/api/app/Http/Controllers/Forum/controller.go`
- Modify: `apps/api/app/Support/Events/catalog.go`
- Modify: `contracts/openapi.yaml`
- Modify: `contracts/openapi/paths/forum.yaml`
- Modify: `contracts/openapi/schemas/forum.yaml`
- Test: `apps/api/app/Models/Forum/service_test.go`
- Test: `apps/api/app/Http/Controllers/Forum/controller_test.go`

- [ ] Add forum domain inputs for topic update and topic lifecycle actions:
  - update fields: `topicID`, `title`, `categorySlug`, `tagSlugs`, `content`
  - lifecycle action fields: `topicID`, `action`, `actorUserID`
  - accepted lifecycle actions: `hide`, `restore`, `lock`, `unlock`, `pin`, `unpin`
- [ ] Add store methods for topic update, soft delete, status update, and pin update.
- [ ] Preserve `posts` triple-storage rules when topic content changes: re-render/sanitize with `RenderContent`, update the existing content record, and store the previous content in `post_revisions`.
- [ ] Authorize own-topic update through author ownership plus `post.edit_own`; authorize any-topic update through `topic.edit_any`.
- [ ] Authorize own-topic delete through author ownership plus `post.delete_own`; authorize any-topic delete/hide/restore through `topic.delete_any`.
- [ ] Authorize lock/unlock with `topic.lock`; authorize pin/unpin with `topic.pin`.
- [ ] Keep public reads limited to `active` and `locked` topics. Hidden and deleted topics must not appear in public lists or public detail.
- [ ] Locked topics remain readable but reject new comments with `forum.topic_closed`.
- [ ] Emit explicit topic events for updated, deleted, hidden, restored, locked, unlocked, pinned, and unpinned actions.
- [ ] Add HTTP routes using the suggested API shape.
- [ ] Add OpenAPI path items, request schemas, response schemas, 401/403/404/409/422 responses, and permission notes.
- [ ] Add service tests for allowed and denied author/moderator paths.
- [ ] Add HTTP tests for unauthenticated, denied, and allowed lifecycle actions.
- [ ] Run `cd apps/api && go test ./app/Models/Forum ./app/Http/Controllers/Forum`.
- [ ] Run `ruby scripts/validate-openapi-refs.rb`.
- [ ] Suggested commit: `feat: add topic lifecycle api`

### Task 2: Topic Detail And Comments UI

**Files:**
- Create: `extensions/builtin/themes/sforum-default/layer/app/pages/t/[topicID]/[topicSlug].vue`
- Modify: `apps/web/app/composables/useForumApi.ts`
- Modify: `apps/web/app/utils/forumTaxonomy.ts`
- Modify: `apps/web/app/components/SFComment.vue`
- Modify: `apps/web/app/locales/zh-CN.ts`
- Modify: `apps/web/app/locales/en-US.ts`
- Test: `apps/web/tests/forumTopic.test.ts`

- [ ] Add frontend types for `ForumComment`, `ForumCommentList`, `ForumReplyReference`, and topic lifecycle responses.
- [ ] Extend `useForumApi()` with `listTopicComments(topicID, view, page, perPage)`, `createTopicComment`, `updateComment`, `deleteComment`, and the topic lifecycle methods added in Task 1.
- [ ] Update `SFComment` so it can render API-sanitized HTML from `content.htmlContent` while still supporting plain text for component previews.
- [ ] Build `/t/:topicID/:topicSlug` in the default theme layer.
- [ ] Fetch topic detail and comments with SSR-friendly `useAsyncData`.
- [ ] If the route slug differs from `topic.slug`, redirect to `forumTopicPath(topic)` with `navigateTo(canonicalPath, { redirectCode: 301 })` during SSR and replace the route on the client.
- [ ] Render category, tags, author profile link, created/updated dates, comment count, view count, locked/hidden/deleted labels where visible to authorized actors, and sanitized topic HTML.
- [ ] Render desktop comments as a readable tree using depth indentation and mobile comments as a flatter list with `replyTo` context when API returns flat view.
- [ ] Add empty, pending, error, and permission-denied states using existing SF feedback components.
- [ ] Add SEO metadata with `useSForumSeo()` and noindex behavior for unavailable resources.
- [ ] Add translation keys in Simplified Chinese and English.
- [ ] Add a focused frontend test for `forumTopicPath`, comment query building, and comment type helpers.
- [ ] Run `cd apps/web && bun test`.
- [ ] Run `cd apps/web && bun run typecheck`.
- [ ] Suggested commit: `feat: add topic detail page`

### Task 3: Composer, Reply, And Edit Flows

**Files:**
- Create: `extensions/builtin/themes/sforum-default/layer/app/pages/topics/new.vue`
- Create: `extensions/builtin/themes/sforum-default/layer/app/pages/t/[topicID]/[topicSlug]/edit.vue`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/t/[topicID]/[topicSlug].vue`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/index.vue`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/c/[categorySlug].vue`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/tags/[tagSlug].vue`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/components/SFNavbar.vue`
- Modify: `apps/web/app/composables/useForumApi.ts`
- Modify: `apps/web/app/utils/forumTaxonomy.ts`
- Modify: `apps/web/app/locales/zh-CN.ts`
- Modify: `apps/web/app/locales/en-US.ts`

- [ ] Add `createTopic` and `updateTopic` helpers that submit `ContentInput` as Markdown:
  - `rawContent`: `SFEditor` payload markdown
  - `sourceFormat`: `markdown`
  - `editorType`: `tiptap`
  - `editorVersion`: `sf-editor-v1`
- [ ] Build `/topics/new` with category select, tag input using existing tag policy, title input, `SFEditor`, draft-safe disabled states, and API field error display.
- [ ] Link the homepage, category page, tag page, and navbar to `/topics/new` only for authenticated users who can create topics.
- [ ] Build `/t/:topicID/:topicSlug/edit` for author/moderator edit. Preload current topic content, preserve category/tags, and redirect back to canonical topic path after save.
- [ ] Add reply composer to the topic detail page for top-level replies.
- [ ] Add nested reply action on comments. The first version may use one inline editor at a time.
- [ ] Add comment edit/delete UI using existing comment APIs and permission-aware action visibility.
- [ ] Add topic action UI for edit, delete, hide/restore, lock/unlock, and pin/unpin. Use icon buttons with tooltips for compact actions.
- [ ] Do not implement likes, votes, favorites, best-answer, or comment adoption in this task.
- [ ] If editor upload is touched, use the existing attachment upload API and record references only through an explicit backend method; do not add a new storage provider.
- [ ] Run `cd apps/web && bun test`.
- [ ] Run `cd apps/web && bun run typecheck`.
- [ ] Suggested commit: `feat: add forum composer flows`

### Task 4: Public Profiles And Profile Settings

**Files:**
- Create: `apps/api/database/migrations/202607070004_user_profiles.sql`
- Create: `apps/api/app/Models/Profile/types.go`
- Create: `apps/api/app/Models/Profile/store.go`
- Create: `apps/api/app/Models/Profile/service.go`
- Create: `apps/api/app/Models/Profile/postgres_store.go`
- Create: `apps/api/app/Http/Controllers/Profile/routes.go`
- Create: `apps/api/app/Http/Controllers/Profile/controller.go`
- Create: `apps/api/app/Providers/profile.go`
- Modify: `apps/api/bootstrap/app.go`
- Modify: `contracts/openapi.yaml`
- Create: `contracts/openapi/paths/profile.yaml`
- Create: `contracts/openapi/schemas/profile.yaml`
- Create: `extensions/builtin/themes/sforum-default/layer/app/pages/u/[username].vue`
- Create: `extensions/builtin/themes/sforum-default/layer/app/pages/settings/profile.vue`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/components/SFNavbar.vue`
- Modify: `apps/web/app/locales/zh-CN.ts`
- Modify: `apps/web/app/locales/en-US.ts`
- Test: `apps/api/app/Models/Profile/service_test.go`
- Test: `apps/api/app/Http/Controllers/Profile/controller_test.go`

- [ ] Add a `user_profiles` table:
  - `user_id` primary key referencing `users(id)` with cascade delete
  - `bio` text default empty
  - `signature` text default empty
  - `location` text default empty
  - `website_url` text default empty
  - `avatar_attachment_id` nullable reference to `attachments(id)` with `ON DELETE SET NULL`
  - `created_at`, `updated_at`
- [ ] Keep the profile table sparse and safe. Do not add background images, custom code, birthday, phone, follow counts, or gamification fields.
- [ ] Add public profile read by username. Return user summary, profile fields, topic count, comment count, and recent public topics.
- [ ] Add current-user profile read/update. Validate length and URL shape without overbuilding validation.
- [ ] Make profile update login-required and current-user only. Do not add an admin profile editor in this task.
- [ ] If avatar upload is wired, require an existing attachment owned by the actor and write an `attachment_references` row for `resource_type = 'avatar'`.
- [ ] Build `/u/:username` public profile page with profile summary and recent public topics.
- [ ] Build `/settings/profile` current-user settings page with safe defaults and a clear save flow.
- [ ] Add navbar user menu links to public profile and profile settings.
- [ ] Add OpenAPI coverage for public profile and current-user profile endpoints.
- [ ] Add service and HTTP tests for public read, own update, unauthenticated update denial, and cross-user update denial.
- [ ] Run `cd apps/api && go test ./app/Models/Profile ./app/Http/Controllers/Profile`.
- [ ] Run `cd apps/web && bun run typecheck`.
- [ ] Run `ruby scripts/validate-openapi-refs.rb`.
- [ ] Suggested commit: `feat: add public profiles`

### Task 5: Mail Provider Contract And Password Reset

**Files:**
- Create: `apps/api/database/migrations/202607070005_password_reset_and_mail_options.sql`
- Create: `apps/api/app/Support/Mail/types.go`
- Create: `apps/api/app/Support/Mail/service.go`
- Create: `apps/api/app/Support/Mail/smtp.go`
- Create: `apps/api/app/Support/Mail/dev_log.go`
- Create: `apps/api/app/Support/Mail/noop.go`
- Create: `apps/api/app/Providers/mail.go`
- Modify: `apps/api/app/Models/Options/types.go`
- Modify: `apps/api/app/Models/Options/service.go`
- Modify: `apps/api/app/Models/Identity/types.go`
- Modify: `apps/api/app/Models/Identity/store.go`
- Modify: `apps/api/app/Models/Identity/service.go`
- Modify: `apps/api/app/Models/Identity/postgres_store.go`
- Modify: `apps/api/app/Http/Controllers/Identity/routes.go`
- Modify: `apps/api/app/Http/Controllers/Identity/controller.go`
- Modify: `apps/api/app/Support/Localization/messages.go`
- Modify: `contracts/openapi.yaml`
- Modify: `contracts/openapi/paths/identity.yaml`
- Modify: `contracts/openapi/schemas/identity.yaml`
- Create: `contracts/openapi/paths/mail.yaml`
- Create: `contracts/openapi/schemas/mail.yaml`
- Create: `apps/web/app/pages/admin/settings/mail.vue`
- Modify: `apps/web/app/config/adminModules.ts`
- Create: `extensions/builtin/themes/sforum-default/layer/app/pages/forgot-password.vue`
- Create: `extensions/builtin/themes/sforum-default/layer/app/pages/reset-password.vue`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/login.vue`
- Modify: `apps/web/app/locales/zh-CN.ts`
- Modify: `apps/web/app/locales/en-US.ts`
- Test: `apps/api/app/Support/Mail/service_test.go`
- Test: `apps/api/app/Models/Identity/service_test.go`
- Test: `apps/api/app/Http/Controllers/Identity/controller_test.go`

- [ ] Add `password_reset_tokens` table:
  - `id` bigserial primary key
  - `user_id` references users with cascade delete
  - `token_hash` unique text
  - `expires_at` timestamp
  - `consumed_at` nullable timestamp
  - `created_at` timestamp
  - `request_ip_hash` text default empty
- [ ] Add mail runtime options with safe defaults:
  - `mail.provider`: `dev_log`, `smtp`, or `noop`; development default should work with Mailpit when configured, production without SMTP must not pretend delivery succeeded to operators
  - `mail.from_address`
  - `mail.from_name`
  - `mail.smtp.host`
  - `mail.smtp.port`
  - `mail.smtp.username`
  - `mail.smtp.password` secret
  - `mail.smtp.encryption`: `none`, `starttls`, or `tls`
- [ ] Add a focused mail provider interface with `Send(ctx, Message) error`.
- [ ] Implement `noop`, `dev_log`, and SMTP providers. If an external SMTP library is needed, do a short library survey and record the decision in `knowledge/decisions/`.
- [ ] Add admin mail settings page at `/settings/mail` under the System folder, protected by `settings.manage`. Do not extend `apps/web/app/pages/admin/settings/index.vue`.
- [ ] Add one-click restore to recommended mail defaults. If secret fields are preserved on reset, state that clearly in the UI.
- [ ] Add `POST /api/v1/admin/mail/test` to send a test email to the current admin or a submitted recipient.
- [ ] Add `POST /api/v1/auth/password-reset/request`. The response must be generic whether the email exists or not.
- [ ] Reuse the existing human-verification purpose `password_reset` when enabled by runtime options.
- [ ] Add `POST /api/v1/auth/password-reset/confirm` with token, new password, token expiry check, single-use consumption, password hash update, and active session invalidation where existing session APIs make that possible.
- [ ] Add localized password reset emails in backend messages/templates. Include site name and reset URL from runtime options.
- [ ] Build `/forgot-password` and `/reset-password` pages in the default theme layer.
- [ ] Link the login page to `/forgot-password`.
- [ ] Add OpenAPI coverage for password reset and mail test endpoints.
- [ ] Add tests for generic request response, missing account privacy, token expiry, single-use token, password policy, successful reset, and mail provider failure handling.
- [ ] Run `cd apps/api && go test ./app/Support/Mail ./app/Models/Identity ./app/Http/Controllers/Identity`.
- [ ] Run `cd apps/web && bun run typecheck`.
- [ ] Run `ruby scripts/validate-openapi-refs.rb`.
- [ ] Suggested commit: `feat: add password reset mail flow`

### Task 6: Reports And Moderation Queue

**Files:**
- Create: `apps/api/database/migrations/202607070006_moderation_reports.sql`
- Create: `apps/api/app/Models/Moderation/types.go`
- Create: `apps/api/app/Models/Moderation/store.go`
- Create: `apps/api/app/Models/Moderation/service.go`
- Create: `apps/api/app/Models/Moderation/postgres_store.go`
- Create: `apps/api/app/Http/Controllers/Moderation/routes.go`
- Create: `apps/api/app/Http/Controllers/Moderation/controller.go`
- Create: `apps/api/app/Providers/moderation.go`
- Modify: `apps/api/bootstrap/app.go`
- Modify: `contracts/openapi.yaml`
- Create: `contracts/openapi/paths/moderation.yaml`
- Create: `contracts/openapi/schemas/moderation.yaml`
- Create: `apps/web/app/pages/admin/moderation.vue`
- Modify: `apps/web/app/config/adminModules.ts`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/pages/t/[topicID]/[topicSlug].vue`
- Modify: `apps/web/app/composables/useForumApi.ts`
- Modify: `apps/web/app/locales/zh-CN.ts`
- Modify: `apps/web/app/locales/en-US.ts`
- Test: `apps/api/app/Models/Moderation/service_test.go`
- Test: `apps/api/app/Http/Controllers/Moderation/controller_test.go`

- [ ] Add `moderation_reports` table:
  - `id` bigserial primary key
  - `reporter_user_id` references users with `ON DELETE SET NULL`
  - `target_type` checked as `topic` or `comment`
  - `target_id` bigint
  - `reason_code` checked against first-version codes: `spam`, `abuse`, `illegal`, `off_topic`, `other`
  - `body` text default empty
  - `status` checked as `open`, `reviewing`, `resolved`, or `rejected`
  - `reviewer_user_id` nullable user reference
  - `review_note` text default empty
  - `created_at`, `updated_at`, `resolved_at`
- [ ] Add duplicate guard so the same reporter cannot create multiple open reports for the same target.
- [ ] Add public report creation endpoint for logged-in active users.
- [ ] Validate that reported topic/comment exists and is publicly reportable.
- [ ] Add admin report list with filters for status, target type, reporter, and page/perPage.
- [ ] Add admin report update for status and review note. Reviewing requires `moderation.report_review`.
- [ ] Add quick actions in the admin moderation page that call Task 1 topic lifecycle endpoints when the reviewer has the needed permission.
- [ ] Add report buttons on topic and comment UI. The report dialog should be small, clear, and not use emoji.
- [ ] Do not implement full notifications or private moderator discussion in this task.
- [ ] Add OpenAPI coverage for report create, list, and update.
- [ ] Add tests for unauthenticated report denial, duplicate report conflict, invalid target, reviewer permission denial, allowed review, and lifecycle quick-action permission boundaries.
- [ ] Run `cd apps/api && go test ./app/Models/Moderation ./app/Http/Controllers/Moderation`.
- [ ] Run `cd apps/web && bun run typecheck`.
- [ ] Run `ruby scripts/validate-openapi-refs.rb`.
- [ ] Suggested commit: `feat: add moderation reports`

### Task 7: Cross-Cut Verification, Docs, And Handoff

**Files:**
- Modify: `knowledge/index.md`
- Modify: `knowledge/modules/forum.md`
- Modify: `knowledge/modules/identity.md`
- Create: `knowledge/modules/profile.md`
- Create: `knowledge/modules/mail.md`
- Create: `knowledge/modules/moderation.md`
- Create: `knowledge/decisions/2026-07-07-mail-provider-contract.md`
- Create: `knowledge/sessions/2026-07-07-core-forum-v1.md`

- [ ] Run backend tests: `cd apps/api && go test ./...`.
- [ ] Run frontend tests: `cd apps/web && bun test`.
- [ ] Run frontend typecheck: `cd apps/web && bun run typecheck`.
- [ ] Run OpenAPI reference validation: `ruby scripts/validate-openapi-refs.rb`.
- [ ] Run a manual browser smoke if dev servers are available:
  - register or log in
  - create topic
  - open topic detail
  - reply
  - edit own topic
  - lock topic as admin
  - confirm reply is blocked while locked
  - report a topic/comment
  - review report in admin
  - request password reset
  - confirm test mail delivery through the configured dev provider
  - open a user profile and edit own profile
- [ ] Update knowledge files with implemented routes, permissions, tables, and remaining open questions.
- [ ] Record unsupported legacy features that remain intentionally outside this release.
- [ ] Confirm no SForumData importer code was added.
- [ ] Suggested commit: `docs: record core forum v1 handoff`

## Final Acceptance

The work is complete only when:

- A normal logged-in member can create a topic, read it, reply, edit own content, and delete own content.
- A moderator/admin with the existing permissions can lock, unlock, pin, unpin, hide, restore, and delete topics.
- Public users can open topic detail pages, category pages, tag pages, and public profile pages.
- Password reset works through the configured mail provider without revealing whether an email exists.
- Users can report topics/comments and reviewers can process reports in admin.
- OpenAPI is updated and reference validation passes.
- Backend tests, frontend tests, and frontend typecheck pass or the handoff records exact failures.
- The implementation does not add the legacy importer and does not silently discard legacy-only feature needs.
