# 2026-07-05 Session Handoff

## Changed

- Added embedded Goose migration files under `database/migrations`.
- Added shared `database/migrator` package with PostgreSQL table-lock guarded
  `Up`.
- API and worker bootstrap now run startup migrations when
  `MIGRATE_ON_STARTUP=true`.
- `cmd/migrate` now reuses the shared migrator.
- Added `MIGRATE_ON_STARTUP` to config, environment examples, and Compose
  API/worker environments.
- Clarified `scripts/dev.sh --no-migrate` as skipping only the dependency-start
  one-shot migration command.
- Updated architecture/development docs and backend knowledge notes.

## Decisions

- Startup migrations are enabled by default and concurrency-safe through
  Goose's PostgreSQL table lock.
- The explicit migration command remains part of deploy flow after backup so
  operators see migration failures before app services update.

## Next

- Keep River internal migrations explicit until the worker has real durable job
  handlers and the project decides whether to wrap River migrations into the
  shared SForum migrator.

## Open Questions

- Should `deploy.sh` eventually skip the explicit migration command once
  release health checks are strong enough, or keep it permanently for
  operational clarity?
