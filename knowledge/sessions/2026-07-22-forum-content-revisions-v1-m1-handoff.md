# 2026-07-22 Forum Content Revisions V1 M1 Handoff

## Changed

- Added Goose migration `202607220052_forum_content_revision_ledger.sql`.
- Added `posts.current_revision` as transitional `0` sentinel during rollout.
- Renamed legacy `post_revisions.edited_by_user_id` to
  `superseded_by_user_id` and added nullable accepted-ledger metadata:
  `revision_no`, `actor_user_id`, `operation`, `origin`, `changed_fields`,
  `attachment_ids`, `committed_at`, restore pointer, completeness, and redaction
  columns.
- Added `topic_revision_snapshots` for topic title/category/tag snapshots.
- Added accepted revision insert helper and wired topic/comment creation to
  transactionally insert revision 1 and set `posts.current_revision=1`.
- Added mixed read support: public topic/comment read models expose effective
  `currentRevision >= 1`; edited markers derive from effective revision `> 1`.
- Added idempotent `PostgresStore.BackfillContentRevisions` plus CLI:
  `sforum revisions backfill --batch=N [--loop]`.
- Added migration/static tests and PostgreSQL integration tests for create
  revision 1 and batched/resumable/idempotent backfill.

## Decisions

- M1 intentionally leaves existing edit writes on the old legacy snapshot
  behavior. M3 owns CAS, expectedRevision, no-op detection, reason rules,
  comment update hooks/events, and replacing edit snapshots with final accepted
  version rows.
- The schema migration is additive and `NO TRANSACTION`; concurrent indexes are
  created in the migration, but payload copying is only done by the backfill.
- Backfill updates legacy rows only as a migration/backfill exception, marks
  them `snapshot_complete=false`, and does not invent missing metadata.

## Verification

- `GOCACHE=/private/tmp/sforum-gocache go test ./app/Models/Forum ./app/Http/Controllers/Forum ./app/Support/Events ./database/migrator ./database/migrations ./cmd/sforum ./app/Models/Profile`
- `GOCACHE=/private/tmp/sforum-gocache go test ./cmd/sforum` with sandbox
  escalation for the existing `/bin/ps` dry-run test.
- `set -a; . ../../.env; set +a; GOCACHE=/private/tmp/sforum-gocache go test ./app/Models/Forum -run 'TestRevisionLedger.*Postgres' -count=1`
- `GOCACHE=/private/tmp/sforum-gocache go test ./...`
- `ruby scripts/validate-openapi-refs.rb`

## Next

- Start M2 only: add `topic.revision.view_any` and `post.revision.view_any`,
  permission catalog/role-template/frontend labels, revision list/detail store
  read models, admin content list/detail read models, routes/controllers, and
  modular OpenAPI.
- Keep M3+ deferred: do not require final `expectedRevision` or alter edit
  write semantics until CAS and accepted edit snapshots are implemented
  together.

## Open Questions

- None blocking M2.
