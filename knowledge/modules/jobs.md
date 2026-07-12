# Jobs Module

## Purpose

Owns SForum's background job framework, queue runtime, job dispatch API, worker
configuration, retry behavior, operational conventions, and admin observability.

Domain modules own their individual jobs. This module owns the shared platform
surface that makes those jobs consistent.

## Current Status

Architecture accepted on 2026-07-04. Foundation implementation started on
2026-07-04; the operator workbench and trusted component slots were completed
on 2026-07-11.

The selected durable queue foundation is River backed by PostgreSQL.

Implemented platform foundation:

- `apps/api/app/Support/Jobs` wraps River queue config, dispatching, worker
  registration, and runtime startup.
- SForum's database migrator runs River's official migrator after Goose, so a
  fresh database receives River tables before API or worker enqueueing.
- API processes create an insert-only River client. Development embeds workers
  by default through `EMBED_WORKER_IN_API=true`; production uses the standalone
  worker process by default.
- `apps/api/bootstrap.NewWorker` opens the worker PostgreSQL pool, builds the
  worker registry, and creates the River client when handlers are registered.
- `apps/api/cmd/worker` starts and gracefully stops the River runtime. An empty
  registry intentionally stays idle because River rejects an empty bundle.
- `apps/api/app/Jobs/Search` owns `search.index_topic`; other domain modules own
  their typed job contracts and handlers.
- `scripts/worker-dev.sh` supports an intentional local API/worker split.

## Operator Workbench

- Admin routes (respecting the configurable admin prefix):
  - `/control-panel/jobs` — queue health, job list, retry/cancel, pause/resume
  - `/control-panel/schedules` — periodic catalog, enable/disable, last/next
    run estimates, manual trigger
- API routes under `/api/v1/admin/jobs` expose overview, schedules, a bounded
  newest-job list, detail, retry, cancel, queue pause/resume, and schedule
  enable/disable/trigger.
- `jobs.view` protects read operations; `jobs.manage` protects mutations. Both
  are granted to `super_admin` by default. API policy checks are authoritative.
- The Jobs UI shows state counts, queue backlog/running/failure counts, filters,
  attempts, arguments, errors, and permission-aware controls.
- River's official client performs mutations; SForum does not implement a
  competing queue state machine.
- Read queries are deliberately bounded to the newest 100 matching jobs.

## Trusted Component Slots

Jobs owns the first production trusted admin component points:

- `admin.jobs.table.columns`
- `admin.jobs.row.actions`
- `admin.jobs.detail.sections`

Plugins may render digest-approved client components there, but cannot bypass
`jobs.manage`, override core routes, or mutate River tables directly.

## Queue Names

- `critical`: small durable jobs needed for user-visible consistency.
- `default`: ordinary background work.
- `search`: Meilisearch indexing and rebuild work.
- `mail`: email delivery.
- `notifications`: reply, mention, and digest fanout.
- `maintenance`: cleanup and scheduled maintenance work.

## Platform Boundaries

- Planned and implemented stack: River, PostgreSQL, `pgx/v5`, and `log/slog`.
- River and PostgreSQL are the durable queue and authoritative job store.
- `pgx/v5` and the existing database pool support transactional enqueueing.
- `apps/api/app/Support/Jobs` owns shared config, dispatch, worker registration,
  runtime startup, queue configuration, timeout behavior, and test helpers.
- `apps/api/cmd/worker` owns the standalone consumer process.
- `apps/api/app/Jobs/*` contains module-owned args and handlers.
- Redis remains for sessions/cache/rate limits, not durable job authority.

## Rules

- Enqueue jobs in the same PostgreSQL transaction as the domain write whenever
  the job represents a side effect of that write.
- Keep job payloads compact and ID-based; do not put secrets in job args.
- Re-read current state inside handlers and make jobs idempotent/retry-safe.
- Set queue concurrency deliberately and chunk large rebuilds into bounded jobs.
- The admin list is capped at the 100 newest matches. Add cursor UI only when
  operator demand justifies a larger history surface.

## Maintenance And Retention

- **Schedule Registry (F1.1):** host-owned catalog in
  `apps/api/app/Support/Jobs` (`ScheduleDefinition`, `ScheduleRegistry`,
  `CoreScheduleDefinitions`). River remains the execution authority;
  `bootstrap.NewWorker` builds `river.PeriodicJob`s only via
  `ScheduleRegistry.BuildPeriodicJobs` (no scattered `NewPeriodicJob` in
  bootstrap).
- Core schedules (daily unless noted):
  - `identity.cleanup_sessions` → queue `default`
  - `extension.web_release_cleanup` → queue `maintenance`
  - `attachments.cleanup_orphans` → queue `maintenance` (handler pre-existed;
    F1.1 registered the periodic)
  - `audit.cleanup_events` → queue `maintenance` (F1.4; default retain 90 days)
- Web Release cleanup always retains the active artifact, its rollback target,
  and the five newest successful artifacts. Failed and superseded artifacts are
  eligible after seven days; build logs after thirty days. Release rows, events,
  and immutable extension snapshots remain as durable history.
- Admin schedule APIs (`jobs.view` / `jobs.manage`):
  - `GET /api/v1/admin/jobs/schedules` — catalog + runtime `enabled`,
    `lastRunAt` (max `river_job.created_at` per kind), estimated `nextRunAt`
  - `POST .../schedules/{id}/enable|disable` — persist flag in
    `web_options` key `jobs.schedule.<id>.enabled` (missing = enabled)
  - `POST .../schedules/{id}/trigger` — enqueue one run now (blocked when
    disabled; clears Unique opts so operators get a fresh job)
- Worker constructors wrap each periodic with an option check: disabled
  schedules return `(nil, nil)` so River skips insert without restart.
- Dedicated admin page: `/control-panel/schedules` (nav: 运维管理 → 定时任务).
  Jobs workbench keeps a short link to that page.

## Worker Heartbeat And Queue Lag (F1.2)

- Worker (standalone `cmd/worker`, or API when `EMBED_WORKER_IN_API=true`)
  publishes Redis key `sforum:worker:heartbeat` every 10s with TTL 45s.
- Admin overview `runtime.worker` exposes `status` (`ok`|`stale`|`unknown`),
  `stale`, `lastSeenAt`, `ageSeconds`.
- Admin overview `runtime.queueLag` cheaply counts River jobs:
  waiting (available/scheduled/retryable/pending), running, discarded.
- API probes: `/api/v1/health` (liveness) vs `/api/v1/ready` (PG required;
  Redis/Meili degraded-ready). Implementation: `app/Support/Health`.

## Resolved Questions

- The first scheduler uses River-native periodic jobs rather than a competing
  SForum scheduler; SForum owns the **catalog**, River owns **when/how** jobs
  insert and run.
- The first operator observability surface is the Jobs workbench. Metrics and
  export formats remain demand-driven follow-up work.
- Worker heartbeat uses Redis (not PG) with short TTL; Meilisearch readiness is
  degraded-ready, not hard-fail.

## Next Steps

- **Wave F1 complete** for F1.1–F1.4. Next framework work is Wave F2 (capability
  grants / Host API). See
  `knowledge/plans/2026-07-12-framework-hardening-waves.md`.
- Wire additional domain job **handlers** into the worker `Registry`; add new
  maintenance **schedules** only through `CoreScheduleDefinitions` (or later
  plugin schedule grants in F2).
- Keep transactional enqueue integration coverage alongside domain writes.
- Add operational metrics/export only after stable self-hosted semantics are
  established.
- Later waves (not F1): job kind `schemaVersion`, plugin-declared schedules,
  outbox alignment, deeper metrics/export.
