# 2026-07-22 Forum Content Revisions V1 M0 Handoff

## Changed

- Completed M0 contract freeze for
  `knowledge/plans/2026-07-22-forum-content-revisions-v1.md`.
- Added ADR:
  `knowledge/decisions/2026-07-22-forum-content-revisions-ledger.md`.
- Added concrete contract-test matrix and fixture strategy:
  `knowledge/plans/2026-07-22-forum-content-revisions-v1-m0-contract-tests.md`.
- Updated forum, identity, moderation, attachments, and extensions module notes
  with the frozen boundaries relevant to later milestones.

## Decisions

- Evolve `post_revisions` into the accepted-version ledger; keep `posts` as the
  hot current read model with `posts.current_revision`.
- Final V1 requires optimistic concurrency via `expectedRevision` on edit and
  restore; conflicts are `409 forum.revision_conflict` with no force-overwrite
  path.
- Restore is append-only and reruns the current Host content, moderation,
  attachment, cache/search, and event pipeline.
- Redaction is a narrow `super_admin` exception for non-current payloads only.
- History and restore are admin-only in V1; self-service history, collaboration,
  notifications, retention controls, and CRDT drafts remain deferred.
- Use npm `diff` (9.0.0, BSD-3-Clause, `kpdecker/jsdiff`) for M6 diff UI; do
  not install npm `jsdiff`.

## Evidence

- Takeover branch/worktree: `main`, clean `git status --short`.
- Latest Goose version in files and local DB: `202607220051`; next M1 migration
  number: `202607220052`.
- Local DB read-only counts: 257 `posts`, 57 `topics`, 200 `comments`,
  0 `post_revisions`, 0 posts with legacy revisions.
- Current code audit confirmed no runtime implementation exists yet for
  `currentRevision`, mandatory `expectedRevision`, numbered current-version
  ledger rows, history-view permissions, comment update hook/event, admin
  content workbench, restore, or redaction.

## Verification

- `ruby scripts/validate-openapi-refs.rb`
- `cd apps/api && go test ./app/Models/Forum ./app/Support/Events ./app/Models/Identity ./database/migrator`
- `node tests/validate-identity-ui.js`

## Next

- Start M1 only: additive schema migration `202607220052`, idempotent batched
  backfill, new revision insertion helper, create-path version 1 rows, and
  mixed-version edited-marker compatibility.
- Do not start M2+ permissions/routes/UI until M1 leaves PostgreSQL tests green.

## Open Questions

- None blocking M1. The only corrected scope wording is the diff dependency
  license: `diff` is BSD-3-Clause, not MIT.
