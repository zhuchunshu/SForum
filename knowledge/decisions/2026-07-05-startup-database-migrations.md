# Decision: Startup Database Migrations

## Status

Accepted

## Context

SForum already uses Goose SQL migrations and a standalone `cmd/migrate`
binary. Local development and production deploys could run migrations
explicitly, but direct API or worker starts still depended on an operator or
script having run the migration step first.

The production model can have API and worker processes starting near the same
time, so startup migrations need to be idempotent and concurrency-safe.

## Decision

API and worker processes run SForum Goose migrations during startup when
`MIGRATE_ON_STARTUP=true`, which is the default.

Migration SQL files are embedded into the API, worker, and `sforum-migrate`
binaries through `go:embed`. Runtime containers do not need a separate
`database/migrations` directory.

All entrypoints use the shared `database/migrator` package. The migrator uses
Goose's PostgreSQL table lock so parallel startup checks serialize safely.

`cmd/migrate`, `scripts/dev.sh`, and `deploy.sh` remain useful explicit
migration paths. Production deploys should still run the explicit migration
command after backup and before updating services so failures are visible
before traffic reaches a new release. Startup migration then acts as an
idempotent safety net.

## Consequences

- Direct `air`, API, or worker starts are schema-safe by default.
- The API can create/update runtime option defaults after migrations have
  ensured `web_options` exists.
- Multiple app processes may start together without racing the same migration.
- Operators can set `MIGRATE_ON_STARTUP=false` only for special maintenance
  cases where migrations are intentionally managed outside process startup.
