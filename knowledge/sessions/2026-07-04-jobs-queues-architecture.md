# 2026-07-04 Jobs Queues Architecture

## Changed

- Added the performance-first jobs and queues design spec.
- Recorded the accepted River/PostgreSQL queue decision.
- Added a `jobs` module note.
- Updated backend, architecture, deployment, and research docs with the queue
  direction.

## Decisions

- River backed by PostgreSQL is the primary durable job framework.
- Redis remains for sessions, cache, and rate limits, not the first durable
  queue store.
- SForum will wrap River with `internal/platform/jobs` so module code gets a
  Laravel-like dispatch/register model without hiding Go dependencies.
- First queue names are `critical`, `default`, `search`, `mail`,
  `notifications`, and `maintenance`.

## Next

- Have the user review the written jobs and queues spec.
- After approval, create an implementation plan for River dependency setup,
  queue migrations, `internal/platform/jobs`, worker runtime, and the first
  search indexing job.

## Open Questions

- Exact queue concurrency environment variable names.
- Whether the first periodic scheduler uses River-native periodic features or a
  small SForum scheduler that enqueues durable jobs.
- Which queue metrics must be present before production launch.
