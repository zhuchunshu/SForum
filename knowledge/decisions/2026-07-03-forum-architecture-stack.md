# Decision: Forum Architecture Stack

## Status

Proposed

## Context

SForum needs an SEO-friendly forum architecture before any application code is
added. The requested direction is Bun, Vue 3, Vite, Nuxt UI, Go Fiber v3,
PostgreSQL, Redis, and Meilisearch.

The repository is documentation-first until the stack, module boundaries, and
first milestone are agreed.

## Decision

Use a two-application monorepo:

- `apps/web`: Nuxt 4 + Vue 3 + Nuxt UI, managed with Bun, rendered with SSR for
  public forum pages.
- `apps/api`: Go Fiber v3 API, using PostgreSQL as the source of truth, Redis
  for sessions/cache/rate limits, and Meilisearch for rebuildable search
  indexes.

Use these backend support libraries by default:

- `pgx/v5` for PostgreSQL connectivity.
- `sqlc` for type-safe SQL access.
- `goose` for SQL migrations.
- `redis/go-redis/v9` for Redis access.
- `meilisearch-go` for Meilisearch access.
- `go-playground/validator/v10` for request validation.
- `goldmark` plus `bluemonday` for Markdown rendering and HTML sanitization.

Use same-origin routing in production where possible:

- Nuxt serves public pages at `/`.
- Fiber serves JSON APIs at `/api/v1/*`.

## Consequences

- SEO-critical pages can ship as server-rendered HTML instead of SPA-only
  content.
- The API remains the single owner of domain logic and writes.
- PostgreSQL stays authoritative; Redis and Meilisearch can be rebuilt or
  repopulated from durable data.
- `sqlc` requires careful SQL design up front but avoids ORM behavior that is
  hard to inspect.
- Fiber v3 currently requires Go 1.25 or newer; the local environment has Go
  1.26.3.
- Search indexing needs a durable outbox or queue so topic/post writes and index
  updates do not drift silently.

## Follow-up

- Confirm MVP scope and whether Meilisearch ships in the first executable
  milestone or immediately after core forum reads/writes.
- Confirm deployment target and same-origin reverse proxy strategy.
- Confirm registration policy, email provider, and upload/object storage plan.
