# Performance-First Jobs And Queues Design

## Context

SForum already targets Go Fiber, PostgreSQL, Redis, Meilisearch, and a separate
`cmd/worker` process. The next architecture decision is how background jobs and
queues should work.

The desired developer experience is Laravel-inspired: modules should be able to
define jobs, dispatch them from service code, run workers by queue, retry failed
work, delay jobs, and keep operational behavior visible. The runtime model
should still be Go-native, explicit, and performance-first.

## Decision Summary

Use River as the primary durable job queue for SForum, backed by PostgreSQL.

SForum will wrap River with a small `internal/platform/jobs` layer so module
code uses stable project-owned concepts while still benefiting from River's
typed workers, PostgreSQL persistence, transactional enqueueing, retries, queue
configuration, and unique-job support.

Redis remains the cache, session, and rate-limit store. It should not be the
primary durable queue for the first jobs framework. Redis-backed queues may be
added later for clearly non-critical fast-lane work, but durable forum jobs
should start in PostgreSQL.

Temporal and workflow engines are deferred. They solve long-running workflow
orchestration well, but they would add too much operational weight before
SForum has complex multi-step workflows.

## Library Survey

### River

River fits SForum's current architecture best because the system already uses
PostgreSQL as the source of truth. Jobs can be inserted from the same database
transaction that writes forum data, which is the key requirement for reliable
search indexing, notification fanout, and email triggers.

Use River for:

- Durable background jobs.
- Transactional enqueueing after domain writes.
- Queue-specific worker concurrency.
- Delayed jobs and retryable work.
- Unique jobs where duplicate work would waste resources or cause drift.

Tradeoff: PostgreSQL is doing more work. This is acceptable for SForum because
forum workloads usually benefit more from consistency and operational
simplicity than from introducing another primary durable system on day one.

### Asynq

Asynq is a mature Redis-backed Go task queue. It is attractive for very fast
ephemeral background tasks, but it would make the first reliable job path span
PostgreSQL and Redis. That split requires an additional outbox or recovery
design to avoid business data committing without the corresponding job.

Keep Asynq as a later option for non-critical fast-lane jobs only after the
durable PostgreSQL queue is established.

### Watermill

Watermill is useful for message routing and event-driven applications. It is a
better fit if SForum later grows into multiple independent services or needs a
broader event bus. For the current monolith-plus-worker architecture, it is more
surface area than the first queue framework needs.

### Temporal

Temporal is a strong choice for durable workflows with activities, timers,
compensation, and long-running state. SForum may evaluate it later for complex
moderation or billing-like workflows. It is not the right first jobs foundation
for a forum MVP.

## Architecture

### Package Shape

Target package layout:

```text
apps/api/internal/platform/jobs/
|-- config.go
|-- dispatcher.go
|-- registry.go
|-- runtime.go
|-- types.go
`-- testing.go

apps/api/internal/modules/search/jobs/
|-- index_post.go
|-- index_topic.go
`-- rebuild_index.go

apps/api/internal/modules/notifications/jobs/
|-- fanout_reply.go
`-- send_email.go
```

`platform/jobs` owns infrastructure concerns only:

- River client construction.
- Queue configuration.
- Worker registration.
- Dispatch helpers.
- Transaction-aware enqueue helpers.
- Job middleware for logging, timeouts, and error classification.
- Test helpers for module job tests.

Domain modules own their job argument structs and handlers. A module registers
its workers during application bootstrap, but it does not reach into another
module's private job internals.

### Public Project API

The SForum wrapper should be small and boring:

```go
type Dispatcher interface {
    Enqueue(ctx context.Context, args JobArgs, opts EnqueueOptions) error
    EnqueueTx(ctx context.Context, tx pgx.Tx, args JobArgs, opts EnqueueOptions) error
}

type Registry interface {
    Add(worker WorkerRegistration)
}
```

Exact River API calls can stay inside `platform/jobs`. Module service code
should dispatch jobs through the project dispatcher. This gives the codebase a
Laravel-like mental model without coupling every module to River setup details.

### Queue Names

Start with these named queues:

- `critical`: small durable jobs that unblock user-visible consistency.
- `default`: ordinary background work.
- `search`: Meilisearch indexing and rebuild slices.
- `mail`: SMTP or provider-backed email delivery.
- `notifications`: mention, reply, and digest fanout.
- `maintenance`: cleanup, pruning, and scheduled maintenance.

Each queue gets independent concurrency. Slow external I/O must not starve
search indexing or critical jobs.

### Dispatch Rules

Use transactional enqueueing whenever a job represents a side effect of a
database write:

- Topic created -> enqueue topic indexing job in the same transaction.
- Post created or edited -> enqueue post indexing and notification jobs in the
  same transaction.
- User requests email verification -> persist token and enqueue email job in
  the same transaction.

Handlers should receive IDs and compact metadata, not full snapshots of domain
objects. The handler should reload current state from PostgreSQL when it runs.
This keeps payloads small and lets jobs naturally ignore deleted, hidden, or
superseded data.

### Idempotency

Every job must be safe to retry.

Rules:

- Job args include stable IDs, not mutable display data.
- External delivery jobs store provider attempts and idempotency keys.
- Indexing jobs can safely upsert or delete derived Meilisearch documents.
- Notification fanout jobs deduplicate by recipient, source object, and event
  type.
- Maintenance jobs process bounded batches and enqueue follow-up work when
  needed.

Use River unique jobs for cases where duplicate queued work is wasteful, such
as "index topic 123" or "rebuild search slice 4". Do not use uniqueness to hide
bugs in business logic; the domain service should still make explicit decisions
about what it dispatches.

### Error Handling And Retries

Classify job failures into:

- Retryable: temporary database, Meilisearch, SMTP, network, or rate-limit
  errors.
- Permanent: invalid job payloads, missing required configuration, unsupported
  enum values, or objects that can never be processed.
- Benign no-op: the referenced record was deleted, hidden, superseded, or no
  longer requires the work.

Retryable failures use bounded retries with backoff and jitter. Permanent
failures should stop retrying and log enough structured context for operators
to inspect the failed job. Benign no-ops should return success after logging at
debug or info level.

### Performance Rules

The jobs framework should optimize for predictable throughput before clever
abstraction:

- Keep payloads small.
- Prefer IDs and database reloads over serialized entity snapshots.
- Batch reads and writes inside handlers when processing fanout or indexing.
- Keep handler transactions short.
- Set per-queue concurrency explicitly.
- Use separate PostgreSQL pool sizing for API and worker processes.
- Use timeouts for every handler.
- Chunk large rebuilds into bounded jobs rather than one long job.
- Avoid dispatching unbounded fanout directly from HTTP request handlers.

The worker process should be horizontally scalable. Multiple worker containers
may run against the same PostgreSQL database as long as queue concurrency and
database pool settings are sized deliberately.

### Scheduling

Delayed jobs are part of the first framework. Periodic jobs should be introduced
only when the first maintenance use case needs them.

Preferred periodic pattern:

1. A small scheduler registers named periodic tasks in the worker process.
2. Each periodic task enqueues ordinary durable jobs.
3. The durable job does bounded work and can be retried independently.

This keeps scheduled triggers separate from job execution and avoids
long-running maintenance loops hidden inside one worker function.

### Observability

Every job execution log should include:

- `job_kind`
- `job_id`
- `queue`
- `attempt`
- `duration_ms`
- `result`
- `error_kind` when failed

Use `log/slog` first. Add metrics after the first real jobs exist, with at
least queue latency, execution duration, retry count, failure count, and
in-flight job count.

An admin jobs dashboard is not required for the first implementation, but the
data model and logging should leave room for one.

### Security And Authorization

Jobs must not trust stale authorization decisions when acting on mutable user
or forum state.

Rules:

- Store actor IDs when useful for audit, not as proof that an action is still
  allowed.
- Re-read current records before applying side effects.
- Do not put secrets in job args.
- Do not send user-visible notifications for deleted, hidden, or inaccessible
  content.
- Staff or admin jobs should write audit records through the owning module.

### First Job Families

The first useful job families should be:

1. Search indexing jobs for topics and posts.
2. Search rebuild jobs that chunk PostgreSQL rows into Meilisearch updates.
3. Email delivery jobs for verification, password reset, and notifications.
4. Notification fanout jobs for replies and mentions.
5. Maintenance jobs for pruning expired tokens, old sessions, and stale
   temporary records.

Search should probably be implemented first because it is already identified as
derived state and naturally benefits from transactional enqueueing.

## Documentation And Implementation Follow-Up

Update architecture and knowledge-base documentation to record:

- River/PostgreSQL is the chosen durable queue foundation.
- Redis is not the first durable queue store.
- Modules define their own jobs and register handlers through `platform/jobs`.
- Worker concurrency is per queue.
- Jobs must be idempotent and use small payloads.

When implementation begins, create a separate implementation plan for:

1. Adding River dependencies and migrations.
2. Building `internal/platform/jobs`.
3. Updating `cmd/worker`.
4. Adding the first search indexing job.
5. Adding integration tests for transactional enqueueing.

## Sources Checked

- River documentation: https://riverqueue.com/docs
- River transactional enqueueing: https://riverqueue.com/docs/transactional-enqueueing
- River unique jobs: https://riverqueue.com/docs/unique-jobs
- Asynq repository: https://github.com/hibiken/asynq
- Watermill documentation: https://watermill.io/docs/
- Temporal Go SDK documentation: https://docs.temporal.io/develop/go/core-application
