# Jobs Module

## Purpose

Owns SForum's background job framework, queue runtime, job dispatch API, worker
configuration, retry behavior, and operational conventions.

Domain modules own their individual jobs. This module owns the shared platform
surface that makes those jobs consistent.

## Current Status

Architecture accepted on 2026-07-04. No application code has been added yet.

The selected durable queue foundation is River backed by PostgreSQL.

## Planned Stack

- River for durable PostgreSQL-backed jobs.
- PostgreSQL as the authoritative job store.
- `pgx/v5` and the existing database pool for transactional enqueueing.
- `log/slog` for worker logs.
- Redis only for sessions/cache/rate limits and possible later non-critical
  fast-lane jobs.

## Planned Boundaries

- `apps/api/internal/platform/jobs`: queue configuration, River client setup,
  dispatch helpers, worker registry, runtime startup, logging middleware,
  timeout handling, and test helpers.
- `apps/api/cmd/worker`: process entrypoint for consuming jobs.
- `apps/api/internal/modules/*/jobs`: module-owned job args and handlers.

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

- Exact environment variable names for queue concurrency and worker pool sizing.
- Whether the first scheduler uses River-native periodic features or a small
  SForum scheduler that enqueues ordinary durable jobs.
- Which observability metrics are required before production launch.

## Next Steps

- Add River dependency and queue migrations after the design is reviewed.
- Implement `internal/platform/jobs`.
- Replace the placeholder worker with the River runtime.
- Implement the first search indexing job and transactional enqueue test.
