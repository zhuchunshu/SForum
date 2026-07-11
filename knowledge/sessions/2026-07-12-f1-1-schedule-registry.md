# 2026-07-12 Session Handoff — F1.1 Schedule Registry

## Changed

- `apps/api/app/Support/Jobs/schedule.go` — `ScheduleDefinition`,
  `ScheduleRegistry`, `BuildPeriodicJobs`, admin `ScheduleView`
- `apps/api/app/Support/Jobs/core_schedules.go` — three core definitions +
  `NewCoreScheduleRegistry`
- `apps/api/bootstrap/worker.go` — periodics only via registry; registered
  `attachments.cleanup_orphans` worker + daily schedule
- `GET /api/v1/admin/jobs/schedules` (`jobs.view`) + OpenAPI
- Admin Jobs workbench read-only “Scheduled jobs” section (zh-CN / en-US)
- Knowledge: `modules/jobs.md`, F1.1 checkboxes in framework waves plan

## Commits (main)

1. `feat(jobs): add Schedule Registry catalog for host periodics`
2. `feat(jobs): migrate periodics onto Schedule Registry in worker bootstrap`
3. `feat(jobs): expose read-only schedule catalog on admin Jobs workbench`
4. docs/knowledge updates (this handoff + plan checkboxes)

## Decisions

- River remains execution authority; SForum owns schedule **catalog**.
- F1 interval-only; cron expressions rejected at Build with a clear error.
- Plugin-declared schedules deferred to F2.
- Admin list is read-only (no enable/disable, no last/next run).

## Verification

- `go test ./app/Support/Jobs/ ./app/Models/Jobs/`
- `ruby scripts/validate-openapi-refs.rb`
- `bun test tests/adminJobs.test.ts`

## Next

1. **F1.2** Ready endpoint + worker heartbeat + admin stale signal
2. F1.3 / F1.4 as time allows
3. Product tracks (Iteration A, settings Wave 3) remain parallel

## Open Questions

- Unchanged from framework plan defaults: Meili degraded-ready, Redis
  heartbeat TTL, schedules stay on Jobs workbench.
