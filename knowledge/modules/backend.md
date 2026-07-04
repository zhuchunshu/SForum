# Backend Module

## Purpose

Owns the Fiber API, domain rules, persistence, sessions, search indexing, and
background work.

## Current Status

Foundation scaffold exists under `apps/api`.
Jobs and queues architecture has been accepted. River backed by PostgreSQL is
the first durable queue foundation.
Backend HTTP composition now has a Laravel-style but Go-explicit
implementation: `bootstrap` assembles the API runtime, `app/Http` registers an
ordered route-provider list, `app/Http/Controllers/*` owns thin controllers and
route declarations, `app/Providers` owns provider wiring, and `app/Models/*`
owns domain logic.
API startup output now keeps Fiber's useful listen metadata but replaces the
default Fiber ASCII banner with an SForum API banner through Fiber's
`OnPreStartupMessage` hook.

## Planned Stack

- Go 1.25+.
- Go Fiber v3.
- PostgreSQL with `pgx/v5`.
- `sqlc` for query generation.
- `goose` for migrations.
- Redis through `redis/go-redis/v9`.
- ALTCHA human verification through the official Go library, wrapped behind a
  small provider interface.
- Meilisearch through `meilisearch-go`.
- `go-playground/validator/v10` for validation.
- `log/slog` for structured logging.
- Backend locale configuration for `zh-CN` default and `en-US` support.
- River for durable PostgreSQL-backed jobs and worker queues.

## Planned Boundaries

- `identity`: users, credentials, sessions, profiles, registration, password
  reset, email verification, roles, permissions, human-verification enforcement,
  and policy helpers.
- `forum`: categories, topics, posts, revisions, visibility, slugs.
- `moderation`: reports, staff actions, audit trail, soft deletion.
- `search`: Meilisearch settings, indexing jobs, rebuilds, search endpoints.
- `jobs`: River-backed durable queue framework, dispatcher, worker runtime,
  retry behavior, and shared job conventions.
- `localization`: locale negotiation, supported locale config, server-owned
  localized templates, and translation key conventions.
- `humanverify`: shared provider boundary for ALTCHA challenge generation,
  server-side verification, stable result codes, and later provider swaps.
- `options`: runtime site-facing settings stored in `web_options`, with typed
  service validation, short-lived backend cache, public read endpoints, and
  permission-protected admin updates.
- `notifications`: deferred unless MVP requires it.

## HTTP Bootstrap And Routing

Borrow Laravel's organization where it helps humans navigate the backend:
small entrypoints, a clear bootstrap layer, service providers, route files,
middleware groups, and thin controllers. Keep the implementation Go-native and
explicit; do not introduce a dynamic dependency container before the codebase
needs one.

Target ownership:

- `cmd/api/main.go` starts and stops the process only.
- `bootstrap` wires config, logging, PostgreSQL, Redis, sessions, providers,
  route providers, jobs, and cleanup hooks.
- `config` owns environment parsing and typed settings.
- `app/Http` owns Fiber app construction, global middleware, `/api/v1`,
  health/system routes, JSON error shape, and route-provider interfaces.
- `app/Http/Controllers/*` declares routes and keeps request DTOs, response
  DTOs, and thin controller methods.
- `app/Providers` builds each module from shared dependencies.
- `app/Models/*` owns domain types, services, policies, repository interfaces,
  and persistence adapters.
- `app/Support/*` wraps external systems and reusable infrastructure clients.
- `database/*` owns migrations, handwritten SQL, generated `sqlc` code, and
  the shared Goose migrator.
- API and worker processes run embedded Goose migrations at startup when
  `MIGRATE_ON_STARTUP=true`. The shared migrator uses Goose's PostgreSQL table
  lock, so parallel process starts serialize safely.
- Development Compose and production deploys may still run the same migration
  binary explicitly as a visible pre-start check; startup migration should then
  be an idempotent no-op.

Route registration rules:

- Use an explicit ordered provider list assembled in bootstrap.
- Prefer a small `http.RouteProvider` interface over one dependency field per
  module.
- A module becomes reachable only when its provider is added to bootstrap.
- Do not register routes from `cmd/*`, platform clients, database stores,
  service constructors, package `init` functions, or filesystem scanning.
- Put middleware at the narrowest useful level: global, API group, or route
  group.

## Jobs And Queues

- Use River with PostgreSQL as the primary durable queue.
- Do not use Redis as the first durable job store.
- Enqueue jobs transactionally with domain writes when the job represents a
  side effect of that write.
- Keep job payloads small and ID-based.
- Application jobs live under `app/Jobs/*`.
- Shared queue runtime and dispatch helpers live under `app/Support/Jobs`.
- Initial queue names are `critical`, `default`, `search`, `mail`,
  `notifications`, and `maintenance`.

## Open Questions

- Final deployment target and runtime process model.
- Whether backend emails and notifications need full English translation in MVP.
- Exact username, email, password, and email-verification rules for open
  registration.

## Next Steps

- Add PostgreSQL/Redis/Meilisearch connectivity after the health-check
  foundation.
- Add supported-locale config and a user locale preference field during identity
  schema design.
- Use one user system with open registration, first-user `super_admin`
  bootstrapping, default `member` assignment, and admin-managed custom
  roles/user groups.
- Use ALTCHA by default for human verification, backed by Redis rate limits and
  single-use challenge tracking.
- Define the first OpenAPI contract and schema migrations.
- Add River and `app/Support/Jobs` after the jobs design is reviewed.
- Implement the accepted API response envelope: every JSON API response uses
  integer `code`, backend-localized `message`, and `data`; `code` equals the
  HTTP status code, and stable machine-readable reason keys live in
  `data.reason`.
