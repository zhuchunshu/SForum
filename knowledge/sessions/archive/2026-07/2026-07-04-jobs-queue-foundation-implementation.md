# 2026-07-04 Jobs Queue Foundation Implementation

## Changed

- Added River-backed queue configuration, dispatch helpers, worker registry,
  and runtime helpers under `apps/api/app/Support/Jobs`.
- Added worker/database pool configuration to `apps/api/config`.
- Added testable PostgreSQL pool config with explicit max connection support.
- Replaced the placeholder worker process with `bootstrap.NewWorker` and a
  graceful River runtime lifecycle.
- Added the first typed search job contract, `search.index_topic`.
- Documented queue environment variables and River migration commands.

## Decisions

- The implementation follows the current Laravel-style Go layout:
  shared queue infrastructure lives in `app/Support/Jobs`, and application
  job definitions live in `app/Jobs/*`.
- The first search job depends on a `TopicIndexer` interface because forum
  topic writes and the Meilisearch indexer are not implemented yet.

## Next

- Run River migrations before starting `cmd/worker` against a fresh database.
- Inject real job registrations into `bootstrap.NewWorker` once modules have
  concrete indexers, mailers, or notification fanout services.
- Dispatch `search.index_topic` from future topic write transactions.

## Open Questions

- Should River migrations remain an explicit CLI operation, or should the
  future SForum migration command wrap them alongside Goose migrations?
- Which queue metrics are required before production launch?
