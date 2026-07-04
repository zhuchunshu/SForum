# Backend Module

## Purpose

Owns the Fiber API, domain rules, persistence, sessions, search indexing, and
background work.

## Current Status

Foundation scaffold exists under `apps/api`.
Jobs and queues architecture has been accepted. River backed by PostgreSQL is
the first durable queue foundation.

## Planned Stack

- Go 1.25+.
- Go Fiber v3.
- PostgreSQL with `pgx/v5`.
- `sqlc` for query generation.
- `goose` for migrations.
- Redis through `redis/go-redis/v9`.
- Meilisearch through `meilisearch-go`.
- `go-playground/validator/v10` for validation.
- `log/slog` for structured logging.
- Backend locale configuration for `zh-CN` default and `en-US` support.
- River for durable PostgreSQL-backed jobs and worker queues.

## Planned Boundaries

- `identity`: users, credentials, sessions, profiles, registration, password
  reset, email verification, roles, permissions, and policy helpers.
- `forum`: categories, topics, posts, revisions, visibility, slugs.
- `moderation`: reports, staff actions, audit trail, soft deletion.
- `search`: Meilisearch settings, indexing jobs, rebuilds, search endpoints.
- `jobs`: River-backed durable queue framework, dispatcher, worker runtime,
  retry behavior, and shared job conventions.
- `localization`: locale negotiation, supported locale config, server-owned
  localized templates, and translation key conventions.
- `notifications`: deferred unless MVP requires it.

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
- Define the first OpenAPI contract and schema migrations.
- Add River and `internal/platform/jobs` after the jobs design is reviewed.
