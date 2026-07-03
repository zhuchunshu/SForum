# Backend Module

## Purpose

Owns the Fiber API, domain rules, persistence, sessions, search indexing, and
background work.

## Current Status

Foundation scaffold exists under `apps/api`.

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

## Planned Boundaries

- `identity`: users, credentials, sessions, profiles, password reset, email
  verification.
- `forum`: categories, topics, posts, revisions, visibility, slugs.
- `moderation`: reports, staff actions, audit trail, soft deletion.
- `search`: Meilisearch settings, indexing jobs, rebuilds, search endpoints.
- `localization`: locale negotiation, supported locale config, server-owned
  localized templates, and translation key conventions.
- `notifications`: deferred unless MVP requires it.

## Open Questions

- Whether auth ships in the first executable milestone or follows the read-only
  forum foundation.
- Whether search indexing uses a PostgreSQL outbox, a Redis-backed queue, or a
  Postgres-native job library.
- Final deployment target and runtime process model.
- Whether backend emails and notifications need full English translation in MVP.

## Next Steps

- Add PostgreSQL/Redis/Meilisearch connectivity after the health-check
  foundation.
- Add supported-locale config and a user locale preference field during identity
  schema design.
- Define the first OpenAPI contract and schema migrations.
