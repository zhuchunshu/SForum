# 2026-07-12 Session Handoff — Schedule ops workbench

## Changed

- Runtime enable/disable for core schedules via `web_options`
  (`jobs.schedule.<id>.enabled`); worker constructors skip insert when disabled
- `GET /admin/jobs/schedules` now returns `lastRunAt` / `nextRunAt`
- `POST /admin/jobs/schedules/{id}/enable|disable|trigger` (`jobs.manage`)
- Dedicated admin page `/schedules` under 运维管理; Jobs page links there
- OpenAPI paths/schemas updated; Go + frontend tests green

## Decisions

- Enable state is option-backed, not a separate table (small key set, no migration)
- Next run is estimated from last River job + interval (River has no public next-fire store)
- Manual trigger clears Unique opts so operators always get a fresh enqueue attempt
- Disabled schedules cannot be triggered (409 `jobs.schedule_disabled`)

## Next

- Optional: show recent runs per schedule on the schedules page
- Plugin-declared schedules still deferred to F2 capabilities

## Open Questions

- Whether next-run estimate should use worker start epoch when no history exists
  (currently `now + interval`)
