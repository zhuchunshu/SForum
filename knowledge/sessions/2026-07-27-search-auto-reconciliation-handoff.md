# 2026-07-27 Search Auto-Reconciliation Handoff

## Changed

- Added Host ledger `search_index_state` and success-only state commits from
  search index/delete workers.
- Added enabled River schedule `search.reconcile`: run on worker start and every
  15 minutes, with bounded 500 index + 500 delete repair batches.
- Wired schedule catalog, admin manual trigger, standalone/embedded workers,
  generated extension catalogs, focused tests, and migration `202607270062`.
- Real dev runtime first pass removed 92 historical site-search ghosts.

## Decisions

- Core ledger is provider-neutral and avoids a new mandatory provider list API.
- Manual rebuild remains for schema changes and fast provider-switch backfill.

## Next

- Monitor maintenance/search queue volume on a large active site; adjust the
  fixed batch or interval only with production evidence.
- Consider an optional provider document-list/reset capability if operators
  need cleanup of external ghosts that predate the Host ledger.

## Open Questions

- None blocking.
