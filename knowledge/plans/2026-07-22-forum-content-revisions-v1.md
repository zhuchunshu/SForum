# Forum Content Editing And Revisions V1 — Task Book

Status: **active** — M4 restore, attachment safety, and redaction complete; M5 admin content management next
Date: 2026-07-22
Goal: allow authorized staff to edit any topic or comment from the admin area,
record every effective self/staff edit as an immutable revision, prevent stale
overwrites, and provide safe history inspection, comparison, and restore.

This is the V1 content-governance foundation, not the collaboration feature.
Implement it milestone by milestone. Every milestone must leave the repository
buildable and must report the exact tests that passed.

## M0 Freeze Evidence (2026-07-22)

- ADR accepted:
  `knowledge/decisions/2026-07-22-forum-content-revisions-ledger.md`.
- Contract-test matrix and fixture plan:
  `knowledge/plans/2026-07-22-forum-content-revisions-v1-m0-contract-tests.md`.
- Current branch/worktree at takeover: `main`, clean `git status --short`.
- Current latest Goose version and next migration number: latest applied/file
  prefix is `202607220051`; M1 must start at `202607220052`.
- Generated catalog commands confirmed:
  `cd apps/api && go run ./cmd/sforum extension docs generate`,
  `cd apps/api && go run ./cmd/sforum extension docs generate --check`,
  `node scripts/v3-catalog/generate.mjs`, and
  `node scripts/v3-catalog/generate.mjs --check`.
- Local development DB read-only baseline:
  257 `posts`, 57 `topics`, 200 `comments`, 0 `post_revisions`,
  0 posts with legacy revisions. Current `post_revisions` columns are
  `id`, `post_id`, `edited_by_user_id`, `raw_content`, `source_format`,
  `editor_type`, `editor_version`, `render_version`, `content_hash`, `reason`,
  and `created_at`.
- Code audit result: task-book baseline still matches current code. Not yet
  implemented: `currentRevision`, mandatory `expectedRevision`, numbered
  current-version ledger rows, history-view permissions, comment update hook/
  event, admin content workbench, restore, and redaction.
- Dependency audit corrected the license wording: npm `diff` 9.0.0 was
  published 2026-04-13 from `https://github.com/kpdecker/jsdiff` and is
  `BSD-3-Clause`, not MIT. The npm package literally named `jsdiff` is a
  different stale 1.1.1 ISC package and must not be installed for V1.

## Required Reading Before Coding

1. `AGENTS.md`
2. `knowledge/index.md`
3. `knowledge/modules/forum.md`
4. `knowledge/modules/moderation.md`
5. `knowledge/modules/identity.md`
6. `knowledge/modules/attachments.md`
7. `knowledge/modules/extensions.md`
8. `knowledge/decisions/2026-07-06-forum-topics-comments-posts.md`
9. `knowledge/decisions/2026-07-06-tiptap-editor-content-storage.md`
10. `knowledge/decisions/2026-07-12-forum-policy-enforcement.md`
11. `knowledge/decisions/2026-07-12-fine-grained-permissions-phase1.md`
12. `knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`
13. This task book

## Product Outcome

After V1:

- a staff member with `topic.edit_any` or `post.edit_any` can find and edit the
  corresponding content from `/admin/forum/content`;
- every effective topic/comment edit creates exactly one durable revision,
  regardless of whether the editor is the author or another actor;
- authorized staff can inspect a revision timeline, compare source text, preview
  a historical version, and restore it;
- a restore creates a new revision and never rewrites/deletes history;
- stale browser forms receive `409 forum.revision_conflict` instead of silently
  overwriting a newer edit;
- the existing public `edited` marker remains operator-configurable and is based
  on the revision number, not timestamps or `EXISTS(post_revisions)`;
- Core remains authoritative for permissions, accepted content, revision
  persistence, restore, moderation policy, cache invalidation, and search effects.

## Current Baseline — Reuse, Do Not Rebuild

| Area | Current evidence | V1 treatment |
| --- | --- | --- |
| Edit authority | `topic.edit_own`, `topic.edit_any`, `post.edit_own`, `post.edit_any` | Reuse; do not add duplicate edit permissions |
| Moderator defaults | built-in moderator template contains both `*_edit_any` keys | Keep and add history-view keys |
| Topic edit | `Service.UpdateTopic` + `PostgresStore.UpdateTopic` | Extend with CAS, revision metadata, no-op detection |
| Comment edit | `Service.UpdateComment` + `PostgresStore.UpdateComment` | Extend with CAS, update hooks/events, revision metadata |
| Existing history | `post_revisions` stores superseded source snapshots | Migrate into a numbered accepted-version ledger |
| Edited marks | read paths use `EXISTS(post_revisions)` | Change to `currentRevision > 1` |
| Content pipeline | render + sanitize + content post-filter + publication decision | Reuse for edit and restore; snapshot final accepted source |
| Topic filter/events | `topic.before_update`, `topic.updated` | Preserve and add revision metadata to observe payload |
| Comment filter/events | create hooks exist; update hooks/events are absent | Add `comment.before_update`, `comment.updated` |
| Attachments | `attachment_references` + transactional replacement/counting | Snapshot IDs; use a dedicated safe restore path |
| Cache/search | `CachedStore` invalidation + topic index enqueue/delete | Reuse on edit and restore |
| Generic audit | `audit_events`, default cleanup after 90 days | Correlate staff actions only; never use as restore source |
| Public editor | `SFTopicEditor`, inline comment `SFEditor` | Reuse; add revision token/conflict handling |
| Admin registry | `adminModules.ts` + generated V3 catalogs | Add one Core content-management page and regenerate catalogs |

## Frozen V1 Scope

### Versioned fields

Every new revision is a full snapshot of the restorable authored fields.

Topic revisions contain:

- title;
- accepted raw body plus source/editor/render metadata and content hash;
- category slug;
- ordered normalized tag slugs;
- ordered unique attachment IDs referenced by the topic body.

Comment revisions contain:

- accepted raw body plus source/editor/render metadata and content hash;
- ordered unique attachment IDs referenced by the comment body.

The following are deliberately not restored by content history:

- original author identity;
- topic/comment status (`active`, `pending`, `rejected`, `hidden`, `deleted`);
- lock/pin state and timestamps;
- moderation decisions and trigger history;
- counters, hot score, last activity, view count, reply count;
- comment parent/root/path/depth;
- creation IP or last-edit IP.

Those remain lifecycle/moderation/audit data and keep their current authority.

### V1 history visibility

- History is not public.
- Authors' own edits are always recorded, but V1 does not expose self-service
  history or self-service restore.
- Topic history requires `topic.revision.view_any`.
- Comment history requires `post.revision.view_any`.
- Restore requires the matching history-view permission plus the existing
  `topic.edit_any` or `post.edit_any` permission.
- The admin page also requires the existing `admin.access` middleware contract.
- `super_admin` keeps the existing bypass semantics.

### Non-goals

- real-time multi-user editing, CRDT transport, presence, or cursors;
- per-topic collaborators or scoped editor ACLs;
- shared drafts, suggestions, inline review comments, or approval workflow;
- a revision for every autosave or keystroke;
- delta/patch chains as canonical storage;
- versioning topic/comment lifecycle state;
- public attribution of the staff editor;
- operator-configurable revision retention or partitioning before metrics justify it;
- notification delivery to the original author (defer to V1.1 until a reliable
  author-access route exists for hidden/pending/rejected content);
- Tiptap native JSON acceptance; preserve the current source-format contract.

## Frozen Architecture Decisions

### A. `post_revisions` becomes the durable version ledger

- Reuse and evolve `post_revisions`; do not introduce an event-sourcing system
  or a second generic content-history stack.
- A row represents one accepted committed version, including version 1 and the
  current version.
- `posts` remains the hot current-content read model. Duplicating only the
  current raw source in the revision ledger is an accepted bounded cost.
- Revisions store source/editor/render metadata, not derived HTML, plain text, or
  excerpts. Historical preview is rendered/sanitized on read with the current
  safe renderer; restore is re-rendered before commit.
- History is ordered by `(post_id, revision_no)`, never by wall-clock time alone.
- Runtime code must not update/delete ordinary revision payloads. The only
  mutation exception is explicit `super_admin` redaction described below.

### B. Save final accepted state, not request state

The snapshot must be created after:

1. authority and edit-window checks;
2. `before_update` filters;
3. normalization and limits;
4. rendering/sanitization;
5. content post-filters;
6. publication policy evaluation;
7. category/tag/attachment validation.

Plugin-patched or sanitized content is therefore what history and restore see.
Rejected/failed requests create no revision and no audit success record.

### C. One effective request equals one revision

- A topic request that changes title, body, category, tags, and attachments
  creates one revision, not one row per field.
- Compare the normalized final snapshot to the current snapshot in the locked
  transaction.
- A semantic no-op returns the current resource and current revision without
  touching `updated_at`, counters, caches, search, audit, or history.
- `changedFields` is server-computed from the allowlist:
  `title|content|category|tags|attachments`.

### D. Optimistic concurrency is mandatory at V1 completion

- Topic detail and comment responses expose `currentRevision >= 1`.
- Both PATCH requests require `expectedRevision` at final acceptance.
- The store locks the owning topic/comment and post, compares the submitted
  revision, and returns `ErrRevisionConflict` before any side effect on mismatch.
- HTTP maps this to `409` with reason `forum.revision_conflict`; clients refetch
  current content. Do not add a one-click "force overwrite" path in V1.
- During staged rollout only, the server may temporarily accept an omitted token
  while first-party clients are upgraded. The final gate must make it required in
  OpenAPI, Go request types, frontend types, and tests.

### E. Staff reason rules

- When actor and original author differ, `reason` is required, trimmed, and
  limited to 500 Unicode runes.
- Restore always requires a reason, even if actor and author happen to match.
- Self-edit reasons are optional.
- Reason is visible only through authorized history/admin surfaces.
- Do not emit reason or raw content in extension-event payloads or logs.

### F. Restore creates a new version

- Restore input is `{ expectedRevision, reason }` plus the target revision in the
  route.
- Restore copies all restorable fields available in the target snapshot and
  creates revision `currentRevision + 1` with operation `restore` and
  `restoredFromRevisionId`.
- It never changes or deletes the target/current historical rows.
- It runs the same current permission, filter, content, moderation, attachment,
  cache, search, and event pipeline as a normal edit.
- Restore is atomic and all-or-nothing. Missing/disabled category/tag data or
  unavailable attachment references return stable `422` errors with no partial
  write.
- A legacy snapshot marked incomplete restores only the fields declared by its
  `restorableFields` (normally body only) and keeps current metadata.
- Deleted resources are read-only in the content editor. Pending, rejected, and
  hidden resources may be edited by authorized staff, but content restore never
  changes their status; publication/visibility changes must still use the
  authoritative moderation or lifecycle workflow.

### G. Historical attachment restore is narrow

- A revision stores IDs, never file bytes, URLs, credentials, or attachment
  provider internals.
- Normal edit attachment ownership checks remain unchanged.
- Historical restore may rebind only IDs present in the selected immutable
  revision, still active, and valid for the resource's original author/current
  visibility policy. It must not become a generic "attach another user's file"
  capability.
- If any required attachment is unavailable, fail closed with
  `forum.revision_attachment_unavailable`; V1 has no partial "skip missing"
  restore button.

### H. Revision history and generic audit are separate

- Revision rows are the long-lived rollback source.
- `audit_events` remains a short-retention operational log and may be cleaned.
- Cross-author edit, restore, and redaction append lightweight audit actions in
  the same database transaction as the content revision. Extend the shared audit
  writer with a narrow transaction-aware entry point instead of accepting drift.
- Audit metadata contains target type/id, author id, revision id/no, operation,
  and changed fields. It must not duplicate raw content, diffs, attachment URLs,
  or the free-form reason.
- Use stable actions `forum.topic.edit_any`, `forum.comment.edit_any`,
  `forum.topic.revision_restore`, `forum.comment.revision_restore`,
  `forum.topic.revision_redact`, and `forum.comment.revision_redact`.
- Never add a restrictive FK from revisions to `audit_events`; audit retention
  must not make revision cleanup fail.

### I. Privacy redaction is an explicit exception

- Normal users and grantable roles cannot delete or mutate history.
- `super_admin` may permanently redact a non-current revision payload with a
  typed confirmation and mandatory reason.
- Redaction preserves the revision header, actor/time/version/operation, but
  clears raw source, topic metadata, and attachment IDs and marks the row
  non-previewable/non-restorable.
- The current revision cannot be redacted; edit or hard-delete the live resource
  first.
- Redaction is auditable and must be covered by allowed/denied tests.
- Hard deletion/privacy erasure of the whole post continues to cascade through
  its revisions. Soft deletion retains history.

### J. Collaboration remains a later layer

`currentRevision`, immutable attribution, conflict detection, and restore are
the prerequisites for future asynchronous collaboration. A later collaboration
track may add `topic_collaborators`, drafts, suggestions, review, and publish.
Real-time editing must use a mature Tiptap-compatible CRDT stack (for example
Yjs/Hocuspocus after a fresh license/maintenance review) in a separate draft
channel; CRDT updates must not become durable published revisions one keystroke
at a time.

## Dependency And Library Survey

- Revision persistence is domain-specific and already exists in PostgreSQL.
  Temporal-table extensions, generic event sourcing, and JSON Patch chains add
  operational/restore complexity without solving SForum's permission and
  content-pipeline rules. Do not adopt them for V1.
- For the admin line diff, prefer the mature `diff` package (project commonly
  known as JsDiff) after verifying the current release and maintenance status.
  M0 verified npm `diff` 9.0.0, published 2026-04-13, repository
  `kpdecker/jsdiff`, license `BSD-3-Clause`. Do not install the stale npm
  package named `jsdiff` (1.1.1, ISC) for V1. Do not hand-roll a diff
  algorithm. Install with the repository proxy environment during M6, not M0.
- `github.com/pmezard/go-difflib` is currently only an indirect Go dependency;
  do not make it a production API dependency unless the frontend option proves
  insufficient and the dependency is promoted/reviewed explicitly.
- Yjs/Hocuspocus is research for the later collaboration track only.

## Target Data Model

Exact migration numbering follows the next available Goose sequence. Keep the
schema additive/online-compatible until backfill is complete.

### `posts`

Add:

- `current_revision BIGINT NOT NULL DEFAULT 0` during migration;
- final constraint `current_revision > 0` only after backfill reaches zero
  pending rows.

`0` is a transitional legacy sentinel and must never be returned by the final
API. At final acceptance every post has a complete current revision row.

### Evolved `post_revisions`

Retain the existing source fields and add/evolve:

| Column | Contract |
| --- | --- |
| `revision_no BIGINT` | positive, unique per `post_id` |
| `actor_user_id BIGINT NULL` | author of this accepted version; FK users `SET NULL` |
| `operation TEXT` | `create|edit|restore|migration` |
| `origin TEXT` | server-derived `self|staff|migration` |
| `changed_fields TEXT[]` | normalized allowlist, sorted and unique |
| `attachment_ids BIGINT[]` | sorted/unique snapshot; no FK because history may outlive files |
| `committed_at TIMESTAMPTZ` | when this version became current |
| `restored_from_revision_id BIGINT NULL` | self-FK; present only for restore |
| `snapshot_complete BOOLEAN` | false for legacy rows lacking metadata |
| `redacted_at TIMESTAMPTZ NULL` | payload redaction time |
| `redacted_by_user_id BIGINT NULL` | `super_admin` actor, FK `SET NULL` |
| `redaction_reason TEXT` | mandatory only when redacted |

Reuse existing `reason`. Its service/API limit is 500 Unicode runes. Existing
`edited_by_user_id` currently means "actor who superseded the stored old
snapshot", not author of that snapshot. Migrate it honestly:

- preserve it temporarily as legacy evidence (rename to
  `superseded_by_user_id` if migration safety permits);
- backfill `actor_user_id` using post creator/current updater plus ordered legacy
  rows;
- new runtime code must never depend on the legacy column;
- document any actor/time that cannot be reconstructed as unknown rather than
  inventing attribution.

Required indexes/constraints:

- unique `(post_id, revision_no)`;
- list index `(post_id, revision_no DESC)` (replace the old created-at index when
  safe);
- index on `actor_user_id, committed_at DESC` only if the admin query actually
  uses it; do not index source text;
- checks for positive revision, allowed operation/origin/changed fields, restore
  pointer consistency, and redaction field consistency.

### `topic_revision_snapshots`

Add a focused one-to-one table instead of stuffing arbitrary resource metadata
into a generic JSON blob:

| Column | Contract |
| --- | --- |
| `post_revision_id BIGINT PRIMARY KEY` | FK `post_revisions(id)` cascade |
| `topic_id BIGINT` | owning topic identity |
| `title TEXT` | accepted title snapshot |
| `category_slug TEXT` | historical stable lookup value |
| `tag_slugs TEXT[]` | normalized sorted snapshot |

For redacted revisions, clear these payload fields while retaining the row as a
tombstone. Comment revisions need no separate metadata table in V1.

### Backfill rules

- Number existing legacy rows by stable `(post_id, created_at, id)` order.
- Reconstruct version actor/commit time only from durable evidence; otherwise
  leave actor null and mark `snapshot_complete=false`.
- Insert one current snapshot for every post after its historical rows, with
  `revision_no = legacy_count + 1` and current topic/comment metadata.
- Unedited posts become version 1; edited posts become `N + 1`.
- Legacy rows normally expose `restorableFields=[content]`; do not pretend old
  category/tag/attachment snapshots exist.
- Backfill must be idempotent, resumable, batched, and observable. Do not place a
  million-row payload copy inside one startup migration transaction.
- Provide a focused CLI/backfill command or durable bounded job using
  `FOR UPDATE SKIP LOCKED`; report pending/completed/error counts.
- Mixed-version runtime during rollout must read safely, but history/restore UI
  is not enabled until backfill reports zero pending rows.

## API Contract Target

Names may follow established controller conventions, but semantics and access
must not be weakened.

### Existing write routes

| Method | Route | Final request change |
| --- | --- | --- |
| `PATCH` | `/api/v1/topics/{topicID}` | require `expectedRevision`; optional `reason` |
| `PATCH` | `/api/v1/comments/{commentID}` | require `expectedRevision`; optional `reason` |

Topic detail and every editable comment response expose `currentRevision`.
Admin list/detail rows also expose it. Public topic list summaries need not add
the field unless an existing consumer requires it.

### Revision routes

| Method | Route | Permission | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/topics/{topicID}/revisions` | `topic.revision.view_any` | Keyset-paged summaries, newest first |
| `GET` | `/api/v1/topics/{topicID}/revisions/{revisionNo}` | same | Authorized source + safe preview metadata |
| `POST` | `/api/v1/topics/{topicID}/revisions/{revisionNo}/restore` | view-any + `topic.edit_any` | Create a new current revision |
| `POST` | `/api/v1/topics/{topicID}/revisions/{revisionNo}/redact` | `super_admin` | Permanent payload tombstone |
| `GET` | `/api/v1/comments/{commentID}/revisions` | `post.revision.view_any` | Keyset-paged summaries, newest first |
| `GET` | `/api/v1/comments/{commentID}/revisions/{revisionNo}` | same | Authorized source + safe preview metadata |
| `POST` | `/api/v1/comments/{commentID}/revisions/{revisionNo}/restore` | view-any + `post.edit_any` | Create a new current revision |
| `POST` | `/api/v1/comments/{commentID}/revisions/{revisionNo}/redact` | `super_admin` | Permanent payload tombstone |

List responses never include raw source. Detail responses include source only
after the same history permission check. Pagination uses an opaque revision/id
cursor; cap `perPage` at 100 and default to 20.

Suggested stable models:

- `ForumRevisionSummary`: id, revisionNo, current, actor summary, operation,
  origin, reason, changedFields, committedAt, restoredFromRevisionNo,
  snapshotComplete, restorableFields, redacted;
- `ForumRevisionDetail`: summary + rawContent/source/editor/render metadata,
  contentHash, attachment availability summary, safe rendered preview, and topic
  metadata when applicable;
- `RestoreRevisionRequest`: expectedRevision, reason;
- `RedactRevisionRequest`: expectedRevision, reason, confirmation;
- `AdminForumContentRow/List`: target type/id, topic context, author, status,
  excerpt, currentRevision, updatedAt.

### Admin content routes

| Method | Route | Access | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/admin/forum/content/topics` | any topic edit/history permission | Filtered topic management list |
| `GET` | `/api/v1/admin/forum/content/topics/{topicID}` | same, action-specific checks | Non-public-aware edit/detail model |
| `GET` | `/api/v1/admin/forum/content/comments` | any comment edit/history permission | Filtered comment management list |
| `GET` | `/api/v1/admin/forum/content/comments/{commentID}` | same, action-specific checks | Non-public-aware edit/detail model |

Do not duplicate mutation logic under `/admin`: the admin UI uses the canonical
PATCH/restore routes so policy remains single-source.

Admin list filters:

- common: opaque cursor, `perPage<=100`, status, author username/id, updated date;
- topics: topic id, title prefix, category slug;
- comments: comment id, topic id;
- no unindexed `%query%` scan over topic/comment bodies;
- sort by `updated_at DESC, id DESC` with a matching index where needed.

### Errors

Add localized stable reasons and OpenAPI responses:

- `forum.revision_conflict` → `409`;
- `forum.revision_not_found` → `404`;
- `forum.revision_reason_required` → `422` with `reason` field message;
- `forum.revision_not_restorable` → `422`;
- `forum.revision_attachment_unavailable` → `422`;
- `forum.revision_category_unavailable` → `422`;
- `forum.revision_tag_unavailable` → `422`;
- `forum.revision_redacted` → `422`;
- `forum.revision_redaction_forbidden` → `403`.

Do not broaden the global error envelope only to return the latest revision on
conflict. The client refetches the resource after the stable 409 reason.

## Permission And Role Work

Add:

- `topic.revision.view_any` — inspect any topic's historical authored content;
- `post.revision.view_any` — inspect any comment's historical authored content.

Update all of:

- Go permission constants and seed catalog;
- additive database migration for existing installations;
- grants to `super_admin` and the built-in `moderator` template;
- frontend permission constants and role-template catalog;
- zh-CN/en-US permission labels/descriptions;
- admin module navigation gating;
- allowed and denied service/controller tests;
- OpenAPI security/permission descriptions;
- knowledge module/decision notes.

Do not grant these permissions to `member`, `operator`, or `tech_admin` by
default. Plugin install/enable paths must never grant them.

## Extension Surface Contract

V1 must update the Forum Extension Surface Matrix explicitly:

| Surface | V1 contract |
| --- | --- |
| Routes | Core revision list/detail/restore/redact routes with authoritative policy |
| Hooks | retain `topic.before_update`; add `comment.before_update` with content-only patch allowlist |
| Observe events | enrich `topic.updated`; add `comment.updated`; include revision/operation/changedFields/restoredFrom, never raw content/reason |
| Queries | no raw revision mutation/query provider in V1; closed for history integrity and sensitive-content ownership |
| Admin/public UI | Core admin page registered; no public history; trusted presentation surfaces follow existing Admin/Page Registry rules |
| Identity/permissions | two host permissions above; plugins may declare their own collaboration keys later but cannot assign Host grants |
| Media | attachment ID snapshots; Host validates restore; no file bytes in events/history contracts |
| Navigation/regions | `/admin/forum/content` in Forum folder; no public navigation item |
| Cache/search | edit/restore invalidate the same scopes and enqueue the same index effects as canonical writes |
| Jobs | bounded idempotent backfill only; no per-edit background revision persistence |
| Lifecycle/privacy | soft delete retains; hard delete cascades; super-admin redaction tombstones payload |

Update `app/Support/Events/catalog.go`, generated catalogs, extension docs, and
contract tests together. `comment.before_update` runs after authority/edit-window
checks and before final validation/render, matching topic update ordering.

Event payload names are fixed for V1:

- `topic.updated` adds `revisionNo`, `operation`, `changedFields`, and optional
  `restoredFromRevisionNo` to its existing safe metadata;
- `comment.updated` contains `commentId`, `topicId`, `actorUserId`, `revisionNo`,
  `operation`, `changedFields`, and optional `restoredFromRevisionNo`;
- `comment.before_update` contains `actorUserId`, `commentId`, `topicId`, and
  `content`, with only `content` patchable.

Host performs an early expected-revision comparison before invoking a synchronous
update filter and repeats the comparison under the database lock. Unauthorized or
already-stale requests must not invoke plugin filters.

## Admin UX Contract

Add `/admin/forum/content` under the Forum navigation group. Register it through
the existing admin module/surface system and regenerate V3 catalogs; do not
hand-edit generated inventory output.

### Content list

- Use a dense work-oriented table, not decorative cards.
- Tabs: **Topics** and **Comments**; hide a tab when the actor has neither the
  corresponding edit nor history permission.
- Columns: content/topic, author, status, current version, updated time, actions.
- Filters use familiar inputs/selects and server-side cursor pagination.
- Actions use Nuxt/Iconify Lucide or Tabler icons with tooltips.
- Edit action requires `*_edit_any`; history action requires
  `*.revision.view_any`.
- Deleted content is inspection-only; no edit/restore shortcut.

### Edit flow

- Reuse `SFTopicEditor`/`SFEditor` through focused shared props/components; do
  not build a second editor stack.
- Show the original author and an explicit "editing another user's content"
  state before the form.
- Require reason for cross-author edits and keep field errors beside the form.
- Submit `expectedRevision` from the loaded detail.
- Success uses a theme-aware success Toast that auto-dismisses within 10 seconds.
- A 409 shows a persistent conflict alert with **Reload latest** and **View
  history** actions. V1 has no force-save action.

### History and diff

- A revision timeline shows version, editor, time, operation, reason, and changed
  fields; current/restored/redacted/legacy states are explicit.
- Fetch raw source only when an authorized user opens a detail/compare view.
- Compare any selected historical revision against current or another selected
  revision.
- Desktop uses a stable side-by-side line diff; mobile uses a single-column
  unified diff. Long unbroken lines wrap without horizontal page overflow.
- Metadata changes (title/category/tags/attachments) appear as structured rows;
  do not force them through the text diff.
- Historical HTML preview is server-sanitized and clearly separate from raw diff.

### Restore and redaction

- Restore opens a confirmation modal with target/current version, changed-field
  summary, attachment/category/tag availability, required reason, and the latest
  `expectedRevision`.
- Restore success Toast identifies the new version number.
- Validation/conflict errors remain visible until resolved or dismissed.
- Redaction is visible only to `super_admin`, requires typed confirmation and a
  reason, and warns that the payload cannot be recovered.
- Never put raw revision content, IP addresses, or attachment provider data into
  Toasts, URLs, analytics, or client logs.

## Milestone Task Book

### M0 — Contract Freeze And ADR

- [x] Re-read required files and inspect the current dirty worktree; preserve
      unrelated user changes.
- [x] Add an ADR for the accepted ledger/current-read-model/CAS/restore/redaction
      decisions. Reference this task book.
- [x] Confirm exact next Goose migration number and generated-catalog commands.
- [x] Verify the current `diff` package release/license before any dependency add.
- [x] Write the named contract-test matrix and fixture strategy for revision
      numbering, conflict, restore, cross-author reason, history permission, and
      redaction. Do not commit a permanently failing test checkpoint.
- [x] Record baseline `post_revisions` row semantics and representative counts.

Acceptance:

- ADR is accepted and no frozen decision remains implicit.
- The test design names the production boundary and expected failure before
  implementation; any executable tests added in M0 remain green.
- No production behavior changes in M0.

M0 validation:

- `ruby scripts/validate-openapi-refs.rb`
- `cd apps/api && go test ./app/Models/Forum ./app/Support/Events ./app/Models/Identity ./database/migrator`
- `node tests/validate-identity-ui.js`
- Full gate intentionally deferred to later executable milestones unless M1+
  changes runtime/schema/frontend behavior.

### M1 — Additive Schema And Online Backfill

- [x] Add `posts.current_revision` transitional column.
- [x] Evolve `post_revisions` with version/actor/operation/origin/fields/
      attachments/commit/restore/completeness/redaction columns and constraints.
- [x] Add `topic_revision_snapshots`.
- [x] Add the final list/unique indexes without an unbounded startup lock; use
      Goose no-transaction/concurrent index patterns where required.
- [x] Implement a transaction helper that inserts one accepted revision snapshot.
- [x] Insert version 1 transactionally on new topic/comment creation.
- [x] Implement idempotent batched backfill with progress and retry visibility.
- [x] Support mixed legacy rows without inventing metadata/attribution.
- [x] Change edited-fact reads from `EXISTS` to effective revision `>1` while the
      migration is mixed.
- [x] Add migration/migrator/store integration tests against PostgreSQL.

Acceptance:

- New content always has revision 1 and matching `posts.current_revision`.
- Existing edited content retains every old raw snapshot in stable order.
- Re-running backfill creates no duplicates and resumes after interruption.
- No single migration transaction copies every post payload.
- Public topic/comment reads remain within their existing performance envelope.

M1 validation (2026-07-22):

- `GOCACHE=/private/tmp/sforum-gocache go test ./app/Models/Forum ./app/Http/Controllers/Forum ./app/Support/Events ./database/migrator ./database/migrations ./cmd/sforum ./app/Models/Profile`
  - `cmd/sforum` needed one sandbox-escalated rerun because its existing
    orphan-plugin dry-run test executes `/bin/ps`; the package passed after
    permission was granted.
- `set -a; . ../../.env; set +a; GOCACHE=/private/tmp/sforum-gocache go test ./app/Models/Forum -run 'TestRevisionLedger.*Postgres' -count=1`
- `GOCACHE=/private/tmp/sforum-gocache go test ./...`
- `ruby scripts/validate-openapi-refs.rb`

M1 implementation notes:

- Goose migration `202607220052_forum_content_revision_ledger.sql` is additive
  and `NO TRANSACTION`; it adds concurrent revision indexes and does not bulk
  copy post payloads.
- `sforum revisions backfill --batch=N [--loop]` claims `posts.current_revision
  = 0` batches using `FOR UPDATE SKIP LOCKED`, numbers legacy rows by
  `(post_id, created_at, id)`, inserts one current snapshot, and only then sets
  `posts.current_revision`.
- Legacy rows keep incomplete body-only semantics; unreconstructable metadata
  is left unknown instead of invented.

### M2 — Revision Read Models And Permissions

- [x] Add the two permission keys, seeds, migration grants, role templates, and
      translations.
- [x] Add Forum store/service types for current revision, summaries, details,
      keyset lists, and non-public-aware admin detail.
- [x] Implement topic/comment revision list/detail with history permissions.
- [x] Render safe historical preview on demand; never persist new derived HTML.
- [x] Implement admin topic/comment cursor lists without body substring scans.
- [x] Add routes/controllers and modular OpenAPI schemas/path items.
- [x] Add allowed/denied/not-found/redacted/legacy tests.

Acceptance:

- Unauthorized users cannot infer whether a hidden resource or revision exists.
- List responses contain no raw source.
- Detail source is available only after the correct history permission.
- Moderator template, role UI, API checks, and translations agree.
- `ruby scripts/validate-openapi-refs.rb` passes.

M2 validation (2026-07-22):

- `GOCACHE=/private/tmp/sforum-gocache go test ./app/Models/Identity ./database/migrations ./app/Models/Forum ./app/Http/Controllers/Forum`
- `set -a; . ../../.env; set +a; GOCACHE=/private/tmp/sforum-gocache go test ./app/Models/Forum -run 'TestRevision(ReadModels|Ledger).*Postgres' -count=1`
- `ruby scripts/validate-openapi-refs.rb`
- `node tests/validate-identity-ui.js`

M2 implementation notes:

- Goose migration `202607220053_forum_revision_view_permissions.sql` adds
  `topic.revision.view_any` and `post.revision.view_any`, granting only
  `super_admin` and the built-in `moderator` template by default.
- Revision list/detail reads are history-permission gated in the service; lists
  return summary headers only, and detail renders a safe preview on demand after
  source authorization.
- Admin content topic/comment list/detail reads require `admin.access` plus the
  matching edit-any or history-view permission. M2 adds no admin mutation route.
- M3 remains responsible for mandatory `expectedRevision`, CAS, reason rules,
  accepted edit snapshots, comment update hooks/events, and write semantics.

### M3 — Versioned Edit Writes And CAS

- [x] Add `expectedRevision`/`reason` to topic/comment service and HTTP inputs.
- [x] Lock resource + post, compare revision, and fail before side effects.
- [x] Reject an already-stale request before synchronous plugin filters and
      recheck the same expected revision under the transaction lock.
- [x] Run final normalized no-op detection inside the transaction.
- [x] Replace the old "snapshot before overwrite" helper with "append final
      accepted version" semantics.
- [x] Snapshot full topic/comment V1 fields and increment current revision once.
- [x] Require reason for cross-author edits.
- [x] Keep edit windows for author-only edits; `*_edit_any` remains exempt.
- [x] Add transaction-aware generic audit append for cross-author successful edits.
- [x] Add `comment.before_update` and `comment.updated`; enrich `topic.updated`.
- [x] Preserve moderation requeue, cache invalidation, search indexing, and
      attachment counts exactly once.
- [x] Make canonical edit/readback work for every non-deleted status without
      accidentally publishing rejected/hidden content; status changes remain in
      moderation/lifecycle services.
- [x] Update first-party public topic/comment editors to submit the loaded token.

Acceptance:

- Self edit and staff edit each create one correctly attributed version.
- No-op, validation failure, filter rejection, stale revision, or transaction
  rollback creates no version/audit/cache/search side effect.
- Two concurrent edits with the same expected revision produce one success and
  one deterministic 409.
- Cross-author edit without reason is denied; self edit may omit it.
- The final API contract requires `expectedRevision`.

### M4 — Restore, Attachment Safety, And Redaction

- [x] Implement topic/comment restore service methods through the current write
      pipeline, not direct SQL copying.
- [x] Resolve legacy `restorableFields` honestly.
- [x] Validate current category/tag policy and fail atomically when unavailable.
- [x] Implement the narrow historical attachment rebind validator.
- [x] Create new `restore` revision with `restoredFromRevisionId`.
- [x] Emit canonical updated events and run cache/search effects once.
- [x] Append restore audit in the content transaction.
- [x] Implement `super_admin` payload redaction and audit tombstone behavior.
- [x] Cover current-revision redaction denial and hard-delete cascade.

Acceptance:

- Restore never mutates prior rows and always advances by exactly one version.
- Restore cannot reintroduce an attachment outside the selected historical
  snapshot or bypass current publication/content policy.
- Lifecycle fields remain byte-for-byte unchanged except normal `updated_at`/
  moderation effects explicitly caused by the current content pipeline.
- Redacted revisions cannot be previewed, diffed, or restored.

### M5 — Admin Content Management

- [ ] Add `/admin/forum/content` and register it under the Forum nav group.
- [ ] Add permission-aware Topics/Comments tabs, filters, cursor pagination, and
      empty/loading/error states.
- [ ] Add non-public-aware admin detail loading.
- [ ] Reuse the existing editor components with author/reason/revision props.
- [ ] Add successful edit Toast and persistent field/operation errors.
- [ ] Add conflict state with reload/history actions and no force overwrite.
- [ ] Add focused component/composable tests and i18n in zh-CN/en-US.
- [ ] Regenerate and validate Admin Surface catalogs.

Acceptance:

- A moderator can find and edit active/locked/pending/rejected/hidden content
  allowed by the service; deleted content is inspection-only.
- A history-only actor can inspect but cannot edit.
- A permission denied by direct user override removes the matching UI/actions and
  remains API-authoritative.
- Layout works at desktop and mobile widths without overlap or horizontal page
  overflow.

### M6 — Timeline, Diff, Restore, And Redaction UX

- [ ] Add revision summary timeline and lazy detail loading.
- [ ] Integrate the reviewed diff library for raw source line comparison.
- [ ] Add structured metadata comparison.
- [ ] Add sanitized historical preview.
- [ ] Add restore confirmation, reason, availability checks, conflict handling,
      and success Toast with new revision number.
- [ ] Add `super_admin` redaction confirmation and persistent irreversible warning.
- [ ] Cover legacy incomplete and redacted states explicitly.
- [ ] Add browser tests for topic edit/history/restore and comment edit/history/
      restore, including stale concurrent tabs.

Acceptance:

- Operators can identify who changed what and why without loading every payload.
- Diff remains usable for long Markdown/code lines and mobile.
- Restore/redaction controls never appear without authority and API denial tests
  prove the boundary independently.

### M7 — Rollout, Performance, Documentation, And Closure

- [ ] Complete backfill in the test/development database and prove zero pending.
- [ ] Enforce final `current_revision > 0` invariant when deployment sequencing is
      safe; otherwise retain an explicit compatibility check with a removal task.
- [ ] Measure revision-list/detail/admin-list queries with representative data;
      run `EXPLAIN (ANALYZE, BUFFERS)` and record results.
- [ ] Confirm public list/detail regressions remain within existing million-scale
      report thresholds; revision payloads must not enter hot list cache values.
- [ ] Verify audit cleanup does not delete/break revision rows.
- [ ] Update forum/moderation/identity/attachments/extensions module notes, event
      catalogs, extension authoring docs, OpenAPI, and bilingual user/admin docs.
- [ ] Run the full repository gate and browser QA.
- [ ] Mark this plan `completed`, update `knowledge/plans/README.md`, and archive
      intermediate hot handoffs.

Acceptance:

- All Definition of Done items below pass.
- No transitional optional `expectedRevision` remains in first-party clients or
  the final OpenAPI contract.
- Backfill/rollback/operator instructions are documented and reproducible.

## Required Test Matrix

### Service/store

- create topic/comment writes revision 1;
- self edit, staff edit, multi-field topic edit, body-only comment edit;
- no-op saves no revision and does not touch timestamps/effects;
- stale expected revision under sequential and concurrent requests;
- staff reason required, bounded, trimmed; self reason optional;
- title/category/tag/body/attachment changed-field calculation;
- plugin patch saved as accepted state; plugin reject saves nothing;
- moderation pending/rejected transitions remain authoritative;
- current revision and `edited` presentation setting interaction;
- legacy incomplete list/detail/restore;
- restore current/old/missing/redacted revision;
- restore unavailable category/tag/attachment all fail atomically;
- restore creates new revision and preserves lifecycle fields;
- hard delete cascade, user deletion actor anonymization, soft-delete retention;
- super-admin redaction allowed; every other actor denied.

### HTTP/OpenAPI

- authentication, CSRF, each allow/deny permission path;
- hidden target non-enumeration;
- list payload excludes raw source;
- PATCH required revision/reason behavior;
- 409/404/422 stable reasons and localized messages;
- revision pagination bounds/cursor validation;
- modular refs pass validation.

### Frontend

- admin navigation and per-tab/action permissions;
- topic/comment filters and pagination;
- editor reason rules and no-op state;
- conflict reload/history flow;
- timeline lazy payload loading;
- text/metadata diff rendering;
- restore success/error/conflict;
- redacted/legacy/unavailable states;
- success Toast follows theme and 10-second dismissal; errors persist;
- zh-CN/en-US strings;
- desktop/mobile no overlap/overflow.

### Extension effects

- unauthorized requests do not invoke update filters/events;
- `comment.before_update` patch/reject behavior;
- topic/comment updated event payload contains revision metadata but no source,
  reason, IP, or attachment provider details;
- restore emits one canonical updated event after commit;
- cache/search invalidation happens once for edit/restore and never for failures.

## Verification Commands

Run focused commands after each milestone and the full gate at closure:

```bash
cd apps/api && go test ./app/Models/Forum ./app/Http/Controllers/Forum ./app/Support/Events ./database/migrator
ruby scripts/validate-openapi-refs.rb
cd apps/web && bun test
cd apps/web && bun run typecheck
./scripts/test.sh
```

Use PostgreSQL integration tests for migration/backfill/CAS/restore correctness.
Use the in-app Browser first for rendered admin flows, then capture desktop and
mobile evidence. Do not kill the user's port-3000 web dev server.

## Likely File Ownership

Keep files focused; do not grow existing service/store files past the repository
warning threshold.

- `apps/api/database/migrations/` — additive schema/permissions/indexes
- `apps/api/cmd/sforum/` — bounded revision backfill command if CLI is chosen
- `apps/api/app/Models/Forum/revisions*.go` — revision types/service/store logic
- `apps/api/app/Models/Forum/admin_content*.go` — admin list/detail query logic
- `apps/api/app/Models/Forum/service*.go` — canonical edit/restore orchestration
- `apps/api/app/Models/Forum/cached_store.go` — existing invalidation integration
- `apps/api/app/Http/Controllers/Forum/` — routes/controllers/error mapping
- `apps/api/app/Models/Identity/seeds.go` — permission catalog/templates
- `apps/api/app/Support/Audit/` — transaction-aware append if needed
- `apps/api/app/Support/Events/catalog.go` — comment update + revision metadata
- `contracts/openapi.yaml`, `contracts/openapi/paths/forum.yaml`,
  `contracts/openapi/schemas/forum.yaml` — modular API contract
- `apps/web/app/pages/admin/forum/content.vue` — admin content workbench
- `apps/web/app/components/admin/forum/` — list/editor/history/diff focused parts
- `apps/web/app/composables/useForumApi.ts` or focused admin/revision composables
- `apps/web/app/config/adminModules.ts`, permission config, generated catalogs
- `apps/web/i18n/locales/{zh-CN,en-US}.json`
- `docs/`, `docs/extensions/`, `knowledge/` — user/admin/extension/project memory

## Risks And Required Mitigations

| Risk | Required mitigation |
| --- | --- |
| History stores removed secrets/abuse | non-public permissions, no payload in list/log/events, super-admin redaction, hard-delete cascade |
| Stale editors overwrite each other | mandatory `expectedRevision`, locked CAS, no force-save |
| Restore bypasses current rules | route through current filters/render/moderation/media policy |
| Moderator reattaches another user's file | selected-snapshot-only restore validator |
| Legacy rows imply false metadata/actor | `snapshotComplete/restorableFields`, unknown rather than invented data |
| Initial revision makes every post look edited | `edited = currentRevision > 1`, never `EXISTS` |
| Backfill blocks startup | additive schema, batched resumable backfill, gated UI enablement |
| Revision table bloats hot reads | source-only snapshots, no body indexes, no revision payload in hot caches |
| Generic audit cleanup breaks restore | revision is authoritative, no restrictive audit FK |
| Plugin surface leaks content | metadata-only observe payloads; Host-owned history authorization |
| Admin content search scans 1M bodies | ID/status/author/category/date/title-prefix filters only in V1 |

## Definition Of Done

V1 is complete only when all are true:

- every new topic/comment and every effective edit has a numbered revision;
- all existing snapshots survive migration and backfill completes/idempotently
  reports zero pending;
- self/staff attribution and reason rules are correct;
- required CAS prevents lost updates in real concurrent PostgreSQL tests;
- history list/detail permissions and hidden-target non-enumeration pass;
- restore is append-only, policy-complete, attachment-safe, and atomic;
- super-admin redaction leaves a non-restorable audit tombstone;
- admin topic/comment management, timeline, diff, conflict, restore, and redaction
  flows work in zh-CN/en-US at desktop/mobile sizes;
- public edited markers and current edit flows remain correct;
- cache, search, moderation, attachment counts, audit, and extension events have
  allowed/denied/success/failure tests;
- OpenAPI refs, Go tests/build, Bun tests/typecheck, product validators, and
  `./scripts/test.sh` pass;
- knowledge base, event catalogs, extension docs, and bilingual handbook are
  updated;
- collaboration, drafts, notifications, retention controls, and real-time CRDT
  remain explicitly deferred rather than partially implemented.

## New-Conversation Start Point

The next implementing conversation must start with **M3 only**:

1. inspect the current worktree and reread the required files plus M1/M2 notes;
2. verify whether newer code or migrations changed the M2 baseline;
3. implement mandatory `expectedRevision`, transaction-locked CAS, reason/no-op
   rules, accepted edit snapshots, and comment update hooks/events;
4. keep M4–M7 deferred unless the task book is explicitly updated with evidence;
5. preserve the M0–M2 decisions around permissions, read privacy, CAS, audit,
   restore, redaction, privacy clearing, and plugin boundaries.

Any newly discovered conflict with current code should update this task book and
record the reason before implementation proceeds.
