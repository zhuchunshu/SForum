# Backend Module

## Purpose

Owns the Fiber API, domain rules, persistence, sessions, search indexing, and
background work.

## Current Status

Foundation scaffold exists under `apps/api`.
Jobs and queues architecture has been accepted. River backed by PostgreSQL is
the first durable queue foundation.
Backend HTTP composition now has an initial Laravel-inspired but Go-explicit
implementation: `internal/bootstrap` assembles the API runtime,
`internal/http` registers an ordered route-provider list, and identity owns its
provider and routes files.

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
- `notifications`: deferred unless MVP requires it.

## HTTP Bootstrap And Routing

Borrow Laravel's organization where it helps humans navigate the backend:
small entrypoints, a clear bootstrap layer, service providers, route files,
middleware groups, and thin controllers. Keep the implementation Go-native and
explicit; do not introduce a dynamic dependency container before the codebase
needs one.

Target ownership:

- `cmd/api/main.go` starts and stops the process only.
- `internal/bootstrap` wires config, logging, PostgreSQL, Redis, sessions,
  module providers, route providers, jobs, and cleanup hooks.
- `internal/http` owns Fiber app construction, global middleware, `/api/v1`,
  health/system routes, JSON error shape, and route-provider interfaces.
- `internal/modules/*/provider.go` builds each domain module from shared
  dependencies.
- `internal/modules/*/routes.go` declares that module's routes and route-group
  middleware.
- `internal/modules/*/http.go` keeps request DTOs, response DTOs, and thin
  handlers.

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
- Domain modules own their job handlers under `internal/modules/*/jobs`.
- Shared queue runtime and dispatch helpers live under
  `internal/platform/jobs`.
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
- Add River and `internal/platform/jobs` after the jobs design is reviewed.
