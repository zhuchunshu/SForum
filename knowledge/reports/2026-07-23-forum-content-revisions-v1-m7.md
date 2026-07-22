# 2026-07-23 Forum Content Revisions V1 M7

Status: rollout and closure evidence.

## Development Rollout

- Ran `sforum revisions backfill --batch=100 --loop` against the local
  development database: `100 -> 157 pending`, `100 -> 57 pending`, then
  `57 -> 0 pending`.
- Post-run verification: 259 `posts`, zero `current_revision < 1`; 264
  `post_revisions`, zero null/non-positive `revision_no` values.
- A production hard `CHECK (current_revision > 0)` is intentionally deferred:
  no production backfill-zero proof is available. The existing mixed-rollout
  compatibility read remains until an operator records zero pending posts on
  every production database and adds/validates the constraint in a dedicated
  migration. This is a deployment prerequisite, not a client-visible fallback.

## Query Evidence

Local PostgreSQL 17 development data is too small for capacity claims, but
`EXPLAIN (ANALYZE, BUFFERS)` confirms the revision list uses
`post_revisions_post_revision_desc_idx` (0.275 ms, 10 shared-buffer hits).
The unindexed admin default list measured 0.906 ms on 57 topics but scanned the
small `topics` and `posts` tables before sorting. Migration
`202607231000_forum_admin_content_indexes.sql` adds concurrent updated/status/
topic ordering indexes without indexing body or revision payloads.

The existing dedicated million-scale evidence remains applicable because public
list/detail SQL and cache values do not include revision payloads:

- `2026-07-21-perf-m1-list-topics.md`: warm home p99 28.7 ms, no public
  `posts` sequential scan.
- `2026-07-21-perf-m4-topic-detail.md`: by-slug detail uses `topics_slug_idx`;
  history payloads are not loaded in the hot detail read.
- `2026-07-21-perf-m6-cache-sharding.md`: warm multi-category read p99 25.1
  ms, with cache generation values rather than revision bodies in public cache.

`k6` is not installed in this environment, so no new capacity numbers are
claimed. Re-run `tests/perf/` against the documented dedicated `sforum_perf`
database after any public list/detail query change.

## Audit Retention

`TestRevisionLedgerAuditCleanupDoesNotBreakHistoryPostgres` creates a
cross-author edit, expires and removes its audit event, then proves both ledger
rows and accepted source remain. Revisions are the restore authority; audit
cleanup must never be treated as history retention.
