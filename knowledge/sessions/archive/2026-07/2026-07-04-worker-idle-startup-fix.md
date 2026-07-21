# 2026-07-04 Worker Idle Startup Fix

## Changed

- Fixed `cmd/worker` startup while no concrete job handlers are registered.
- `bootstrap.NewWorker` now returns an idle worker when the registry is empty,
  so the development worker process can stay alive before search, mail, or
  notification workers are wired.
- `app/Support/Jobs.Start` and `Stop` now treat a nil River client as a no-op
  for that idle path.
- Added a regression test for the empty-registry worker startup path.

## Decisions

- Do not register a fake River worker just to satisfy River startup.
- Keep idle startup as a temporary but explicit bridge until modules inject
  real job registrations.

## Next

- Wire concrete module workers into `bootstrap.NewWorker` when their services
  exist.
- Run River migrations before enabling real River workers against a fresh
  database.

## Open Questions

- Whether River migrations should be wrapped by the future SForum migrate
  command.
