# Laravel-Style Database Migrations Design

## Context

SForum already targets PostgreSQL, Go Fiber, `pgx/v5`, `sqlc`, and `goose`.
The project now needs a stricter development rule for database work: schema
changes should feel like Laravel migrations, remain reviewable, and support
automatic execution during startup or deployment.

## Decision Summary

Use `goose` SQL migrations as the only authoritative schema evolution
mechanism. Keep `pgx/v5 + sqlc` as the default data access path. Do not use
GORM `AutoMigrate` for production schema changes.

GORM may be considered later for limited data-access cases, but it must not
modify the database structure outside the migration system.

## Migration Rules

- Store migration files under `apps/api/internal/store/migrations`.
- Every database schema change must be represented by a new migration file.
- A migration that has been merged, deployed, or shared with another developer
  is immutable history. Do not edit it; add a new migration that corrects or
  extends it.
- Migrations must include both up and down steps unless a rollback is genuinely
  impossible. Irreversible migrations must explain why in the migration file.
- Prefer migrations for data backfills and data corrections that affect
  application behavior. One-off operational scripts are allowed only when they
  are clearly documented and not part of normal schema evolution.
- Do not use application startup code, repositories, or model definitions to
  create or alter tables implicitly.

## Laravel-Inspired Developer Experience

The project should eventually expose commands with clear Laravel-like intent:

- Create a new migration.
- Run all pending migrations.
- Roll back the latest migration batch when safe.
- Show migration status.

The implementation can be Go-native, but the mental model should be familiar:
append-only migration files, ordered execution, explicit rollback, and visible
status.

## Automatic Migration Strategy

Local development should support automatic migrations so `./scripts/dev.sh`
can bring a developer to a runnable database without manual steps.

Production should keep migrations as a first-class deploy step. `deploy.sh`
should run migrations after backing up PostgreSQL and before starting the app
services that depend on the new schema.

API startup may support automatic migrations behind configuration. The default
should be conservative:

- Development: automatic migrations may be enabled by default.
- Production: migrations should normally run through `deploy.sh` or an
  explicit migration command, not every app process startup.

This avoids multiple production API instances racing to migrate the same
database and keeps destructive or long-running migrations visible to operators.

## Documentation Updates

Update the following project documents:

- `docs/architecture.md`: clarify that `goose` migrations are the only schema
  evolution source and that `GORM AutoMigrate` is not allowed for production
  schema management.
- `docs/development-and-deployment.md`: describe automatic local migrations and
  the guarded production deployment flow.
- `knowledge/modules/backend.md`: record the backend module rule so future
  sessions find it quickly.
- `knowledge/decisions/2026-07-04-laravel-style-database-migrations.md`: add
  the decision record.
- `knowledge/index.md`: add a short navigation/status note if useful.

## Testing And Verification

For this documentation-only change:

- Verify the edited Markdown reads consistently with the existing architecture
  decision.
- Check that no existing rule still suggests ad hoc schema changes or GORM
  `AutoMigrate` as a default.
- Leave application code untouched.

## Open Questions

- Exact migration command names can be chosen when the API CLI is implemented.
- Whether rollback is enabled in production operations or limited to local/CI
  remains a deployment policy decision.
