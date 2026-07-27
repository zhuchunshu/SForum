# 2026-07-27 Search Index Automatic Reconciliation

## Status

Accepted.

## Context

Normal forum writes enqueue search index/delete jobs, and River retries an
enqueued failure. This did not repair an enqueue failure, an exhausted job, or
data written outside the Forum service. Periodic full reindexing would repair
missing documents but wastefully rewrite healthy indexes and still could not
identify deletions in an external provider.

Alternatives considered:

- periodic full reindex: simple, but unbounded and unable to prove deletes;
- require every provider to list all document IDs: stronger remote inspection,
  but expands the provider RPC contract and is not supported by current plugins;
- Host-owned synchronization ledger: provider-neutral, bounded, and compatible
  with existing `index` / `delete` transport operations.

## Decision

1. Core owns `search_index_state`, keyed by `provider_id + topic_id`. A worker
   updates it only after the selected provider confirms an idempotent index or
   delete operation.
   The protected PostgreSQL provider additionally compares the actual
   `search_documents` rows, so an out-of-band missing/obsolete site document is
   repairable even when the ledger previously recorded success.
2. River schedule `search.reconcile` runs on worker start and every 15 minutes.
   Each run enqueues at most 500 stale/missing index jobs and 500 obsolete
   delete jobs. Active-state uniqueness prevents overlap with normal writes.
3. Staleness compares the ledger source timestamp with authoritative
   `topics.updated_at`. This favors correctness; popular topics whose view count
   flush advances `updated_at` may receive a harmless extra upsert.
4. Migration backfills the protected site-search ledger from
   `search_documents`, including historical ghosts so the first reconciliation
   can delete them.
5. Manual full reindex remains available for schema derivation changes and fast
   bulk backfill after selecting a new provider.

## Consequences

- Lost or exhausted incremental jobs are repaired without operator action.
- Hidden, deleted, and physically missing topics tracked by Host are eventually
  removed from the current provider.
- The repair rate is bounded and observable through the existing Jobs schedule
  catalog; operators may disable or manually trigger it.
- An external provider ghost created before this ledger existed and never
  recorded by Host cannot be enumerated through the current transport contract.
  The same applies if an external provider loses data out of band without a
  failed Host operation. Public live hydration still filters ghosts; provider
  reset/list support can be a future optional capability.
