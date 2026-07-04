# 2026-07-04 Automatic Migrations

## Changed

- Added a dedicated `sforum-migrate` binary to the API Docker build.
- Added a development Compose `migrate` service that runs Goose migrations
  before API and worker startup.
- Added a production Compose `migrate` tool service and wired `deploy.sh` to run
  it after PostgreSQL backup and before service update.
- Updated development/deployment docs and backend knowledge notes.

## Decisions

- Keep local development automatic so `./scripts/dev.sh` brings the schema up to
  date without manual commands.
- Keep production migrations explicit inside the deploy flow rather than running
  from every app process startup.

## Next

- Decide whether the future SForum migration command should also wrap River
  queue migrations.

## Open Questions

- Should production rollback expose Goose down migrations through `deploy.sh`,
  or stay backup/restore-only until release tagging is added?
