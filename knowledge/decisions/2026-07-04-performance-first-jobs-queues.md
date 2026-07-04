# Decision: Performance-First Jobs And Queues

## Status

Accepted

## Context

SForum needs a queue and jobs framework for search indexing, notifications,
email delivery, cleanup, and later maintenance tasks. The desired developer
experience should feel familiar to Laravel users, but the implementation should
stay Go-native, explicit, and optimized for reliable throughput.

The backend stack already uses PostgreSQL as the source of truth, Redis for
sessions/cache/rate limits, and a separate `cmd/worker` process.

## Decision

Use River with PostgreSQL as the primary durable jobs and queues foundation.

Create a small SForum-owned wrapper under `apps/api/app/Support/Jobs` so
modules can dispatch and register jobs through project-level interfaces. Domain
modules define their own job argument structs and handlers under
`apps/api/app/Jobs`.

Redis will not be used as the first durable queue store. It remains available
for sessions, cache, rate limiting, and later non-critical fast-lane work.

Temporal and other workflow engines are deferred until SForum has workflows
complex enough to justify the extra infrastructure.

## Consequences

- Jobs that follow database writes can be enqueued in the same PostgreSQL
  transaction as the domain change.
- Search indexing and notifications can be reliable without a separate outbox
  table or Redis recovery bridge.
- PostgreSQL carries additional queue load, so worker concurrency and database
  pool sizing must be explicit.
- Job payloads should stay small and ID-based.
- Job handlers must be idempotent and safe to retry.
- Queue names and concurrency should be configured per workload, starting with
  `critical`, `default`, `search`, `mail`, `notifications`, and `maintenance`.

## Follow-Up

- Add River dependencies and migrations when implementation begins.
- Build `app/Support/Jobs` as the stable project API around River.
- Convert `cmd/worker` from a placeholder process into the River worker
  runtime.
- Implement search indexing jobs before email or notification jobs.
- Add integration tests for transactional enqueueing and retry behavior.
