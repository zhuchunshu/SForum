# Jobs Module

## Purpose

Owns SForum's background job framework, queue runtime, job dispatch API, worker
configuration, retry behavior, and operational conventions.

Domain modules own their individual jobs. This module owns the shared platform
surface that makes those jobs consistent.

## Current Status

Architecture accepted on 2026-07-04. Foundation implementation started on
2026-07-04.

The selected durable queue foundation is River backed by PostgreSQL.

Implemented so far:

- `apps/api/app/Support/Jobs` wraps River queue config, dispatching, worker
  registration, and runtime startup.
- `apps/api/bootstrap.NewWorker` opens the worker PostgreSQL pool, builds the
  worker registry, and creates the River client when at least one module has
  registered job handlers.
- `apps/api/cmd/worker` starts and gracefully stops the River-backed worker
  runtime.
- `apps/api/app/Jobs/Search` defines the first typed job contract,
  `search.index_topic`, against a narrow `TopicIndexer` interface.
- Until concrete module workers are injected, `cmd/worker` intentionally starts
  in idle mode. This avoids passing an empty worker bundle to River, which
  rejects startup with `at least one Worker must be added to the Workers
  bundle`.

## Planned Stack

- River for durable PostgreSQL-backed jobs.
- PostgreSQL as the authoritative job store.
- `pgx/v5` and the existing database pool for transactional enqueueing.
- `log/slog` for worker logs.
- Redis only for sessions/cache/rate limits and possible later non-critical
  fast-lane jobs.

## Planned Boundaries

- `apps/api/app/Support/Jobs`: queue configuration, River client setup,
  dispatch helpers, worker registry, runtime startup, logging middleware,
  timeout handling, and test helpers.
- `apps/api/cmd/worker`: process entrypoint for consuming jobs.
- `apps/api/app/Jobs/*`: module-owned job args and handlers.

## Queue Names

- `critical`: small durable jobs needed for user-visible consistency.
- `default`: ordinary background work.
- `search`: Meilisearch indexing and rebuild work.
- `mail`: email delivery.
- `notifications`: reply, mention, and digest fanout.
- `maintenance`: cleanup and scheduled maintenance work.

## Rules

- Enqueue jobs in the same PostgreSQL transaction as the domain write whenever
  the job represents a side effect of that write.
- Keep job payloads compact and ID-based.
- Re-read current state inside handlers.
- Make every job idempotent and retry-safe.
- Set queue concurrency deliberately so slow external I/O cannot block search
  or critical jobs.
- Chunk large rebuilds into bounded jobs.
- Do not put secrets in job args.

## Open Questions

- Whether the first scheduler uses River-native periodic features or a small
  SForum scheduler that enqueues ordinary durable jobs.
- Which observability metrics are required before production launch.
- Whether River migrations should stay as an explicit CLI command or be wrapped
  by the future SForum migrate command.

## Next Steps

- Run River migrations before enabling concrete workers against a fresh
  database.
- Wire real module registrations into `bootstrap.NewWorker` as domain jobs
  become available.
- Implement the actual Meilisearch topic indexer and dispatch
  `search.index_topic` transactionally from future topic writes.
- Add integration tests for transactional enqueueing once forum write services
  exist.
