# 2026-07-22 Forum Content Revisions Ledger

## Status

Accepted for V1 implementation. M0 only; no runtime/schema implementation has
started in this checkpoint.

## Context

SForum already has `topics` and `comments` backed by shared `posts`, plus
source-only `post_revisions` rows inserted before an edit overwrites the hot
`posts` row. That baseline is useful for "was edited" markers, but it is not a
restorable version ledger: rows are superseded snapshots, lack positive
per-post revision numbers, do not include version 1/current rows, and record
`edited_by_user_id` as the actor who superseded the stored snapshot rather than
the author of that snapshot.

Current code evidence at M0:

- `UpdateTopic` and `UpdateComment` still call `createPostRevision` before
  `updatePost`; no `currentRevision` or `expectedRevision` contract exists.
- Public edited facts are derived from `EXISTS(post_revisions)`.
- `topic.before_update` and `topic.updated` exist; `comment.before_update` and
  `comment.updated` do not.
- Forum writes already reuse render/sanitize, content post-filter, publication
  policy, attachment reference replacement, cache invalidation, and search
  enqueue/delete behavior.
- Existing permissions include `topic.edit_own`, `topic.edit_any`,
  `post.edit_own`, and `post.edit_any`. The two history-view permissions are
  not present yet.
- The local development database on 2026-07-22 had 257 `posts`, 57 `topics`,
  200 `comments`, and 0 `post_revisions`; the applied Goose version was
  `202607220051`.

## Decision

V1 will evolve `post_revisions` into the authoritative durable content-version
ledger while keeping `posts` as the hot current read model.

- Every accepted committed version, including version 1 and the current
  version, is represented by exactly one `post_revisions` row.
- `posts.current_revision` is the hot CAS/read token. It is transitional
  `0` only during schema/backfill rollout and must not be exposed by final V1
  APIs.
- Revisions store restorable source/editor/render metadata, content hash,
  changed fields, attachment IDs, operation/origin, actor, commit time, restore
  pointer, completeness, and redaction metadata. They do not store derived HTML,
  plain text, excerpts, IP addresses, attachment provider internals, or audit
  payload copies.
- Topic metadata snapshots live in a focused one-to-one
  `topic_revision_snapshots` table keyed by `post_revisions.id`; comment
  revisions need no separate metadata table in V1.
- Runtime code appends revisions. It must not update or delete ordinary
  revision payloads. The sole mutation exception is explicit `super_admin`
  redaction of a non-current revision payload.

## CAS And Write Semantics

Final V1 requires `expectedRevision` on topic/comment PATCH and restore
requests. The service must compare the submitted token before synchronous
extension filters and then recheck under the database lock before side effects.
A mismatch returns `409 forum.revision_conflict`; there is no force-overwrite
path in V1.

The stored revision snapshot is the final accepted state after permission,
edit-window, filters, validation, render/sanitize, content post-filter,
publication policy, category/tag, and attachment validation. A semantic no-op
returns the current resource/current revision without touching timestamps,
history, audit, cache, search, or extension events.

## Restore And Redaction

Restore is append-only. Restoring revision N creates revision current+1 with
operation `restore` and `restored_from_revision_id`; it never mutates the
target/current historical rows and must run through the same current Host
permission, content, moderation, attachment, cache, search, and event pipeline
as ordinary edits.

Historical attachment restore is deliberately narrow: only attachment IDs
stored in the selected immutable revision, still active, and still valid for
the resource author/current visibility policy may be rebound. Missing category,
tag, or attachment state fails closed with stable 422 reasons and no partial
write.

`super_admin` may redact a non-current revision payload only with typed
confirmation and a reason. Redaction preserves the revision header while making
the payload non-previewable and non-restorable. The current revision cannot be
redacted.

## Permission And Audit Boundary

V1 adds `topic.revision.view_any` and `post.revision.view_any`. History is
admin-only; authors' own edits are recorded but not exposed through
self-service history or restore in V1. Restore requires the matching history
permission plus the existing edit-any permission. Redaction is `super_admin`
only.

Revision rows are the long-lived restore source. `audit_events` remains a
short-retention operational log. Cross-author edit, restore, and redaction
append lightweight audit rows in the same transaction, without raw content,
diffs, reason text, attachment URLs, or restrictive FKs from revisions to
audit rows.

## Extension Boundary

Core owns revision persistence, permission checks, accepted content, restore,
redaction, moderation policy, cache/search effects, and history authorization.
Plugins may observe safe metadata. V1 does not expose raw revision query or
mutation providers to plugins.

`comment.before_update` is added as a synchronous content-only filter.
`topic.updated` gains revision metadata and `comment.updated` is added, but
observe payloads never include raw source, reason, IP, file URLs, credentials,
or attachment provider internals.

## Dependency Decision

Use the npm package `diff` for admin text comparison when M6 installs the
frontend dependency. Do not use the npm package named `jsdiff`.

M0 registry evidence captured on 2026-07-22:

- `diff` latest version: `9.0.0`
- Published: 2026-04-13
- Repository: `https://github.com/kpdecker/jsdiff`
- License: `BSD-3-Clause`
- Package metadata keywords include `jsdiff`

The earlier task-book wording that called this option MIT-licensed is corrected:
the acceptable V1 choice is BSD-3-Clause, not MIT. No dependency is installed
in M0.

## Migration And Backfill Boundary

The next Goose migration number reserved for M1 is `202607220052`, because the
latest applied/file prefix at M0 is `202607220051`.

Backfill must be recoverable, reentrant, observable, and batched. The schema
migration remains additive; large payload copying is not performed in one
startup migration transaction. M1 should add a CLI or bounded durable job that:

- claims small batches with `FOR UPDATE SKIP LOCKED`;
- writes missing legacy-numbered rows in stable `(post_id, created_at, id)`
  order;
- inserts exactly one current snapshot after all legacy rows for each post;
- sets `posts.current_revision` only after the post's ledger is complete;
- marks unreconstructable metadata/actor fields as incomplete or unknown rather
  than inventing attribution;
- reports pending, completed, skipped, and failed counts; and
- can be interrupted and rerun without duplicate revisions.

History/restore UI remains disabled until backfill pending count is zero.

## References

- Task book: `knowledge/plans/2026-07-22-forum-content-revisions-v1.md`
- Test matrix: `knowledge/plans/2026-07-22-forum-content-revisions-v1-m0-contract-tests.md`
- Earlier forum model decision:
  `knowledge/decisions/2026-07-06-forum-topics-comments-posts.md`
- Tiptap/content storage decision:
  `knowledge/decisions/2026-07-06-tiptap-editor-content-storage.md`
- Server policy decision:
  `knowledge/decisions/2026-07-12-forum-policy-enforcement.md`
- Fine-grained permissions decision:
  `knowledge/decisions/2026-07-12-fine-grained-permissions-phase1.md`
- Trusted extension platform decision:
  `knowledge/decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`
