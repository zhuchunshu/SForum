# 2026-07-23 Forum Content Revisions V1 M7 Handoff (Archived)

## Changed

- Ran local development ledger backfill to zero pending: 259 posts all have
  positive `current_revision`; 264 revision rows all have positive numbers.
- Added concurrent admin content-list indexes in
  `202607231000_forum_admin_content_indexes.sql`, with a migration contract test.
- Added PostgreSQL coverage proving audit cleanup can remove an expired
  cross-author edit audit event without altering its revision ledger rows.
- Added API-localized messages for every stable forum revision error, bilingual
  operator/admin documentation, module-boundary notes, extension-author guidance,
  and the M7 rollout/performance report.

## Decisions

- Do not add a production `posts.current_revision > 0` check constraint until
  each production database has recorded a zero-pending backfill result. The
  mixed-rollout compatibility read remains a deployment compatibility path only.
- Existing dedicated million-scale public-read reports remain the capacity
  evidence: revisions do not enter public list/detail cache values. The local
  development EXPLAIN is recorded only for revision/admin query shape, not a
  capacity claim.

## Next

- Repair the current `ExtensionManifest.LocalizedText` compilation errors in
  `v3_platform_normalize.go` and `v3_validate_platform.go`, then rerun
  `./scripts/test.sh`.
- If the gate passes, mark the M7 checklist and plan complete, archive the M6/M7
  hot handoffs, and move the plan to `knowledge/plans/archive/2026-07/`.

## Open Questions

- `k6` is absent locally. Re-run `tests/perf/` only when public read SQL/cache
  behavior changes, and only against the documented dedicated perf database.
