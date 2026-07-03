# Research Notes

Use this file to capture short comparisons before choosing third-party
libraries, frameworks, or services.

## Frontend Rendering And UI

- Problem: forum categories, topics, and posts need crawler-friendly first-load
  HTML, metadata, canonical URLs, and a polished Vue UI.
- Options: plain Vue 3 + Vite SPA, Nuxt 4 SSR, custom server rendering.
- Recommendation: Nuxt 4 with Vue 3 and Nuxt UI; Bun manages scripts/packages.
- Reason: Nuxt keeps Vite-backed development while adding SSR, routing, data
  fetching, route rules, and metadata conventions. A plain SPA would push SEO
  work onto custom infrastructure.
- Follow-up: use `@nuxtjs/seo` once page routes exist.
- Sources: https://nuxt.com/docs/4.x/getting-started/introduction,
  https://ui.nuxt.com/docs/getting-started/installation/nuxt,
  https://nuxtseo.com/, https://bun.sh/docs

## Backend HTTP Framework

- Problem: expose a maintainable JSON API for forum reads, writes, sessions,
  moderation, and search.
- Options: Go standard `net/http`, Gin, Echo, Go Fiber v3.
- Recommendation: Go Fiber v3.
- Reason: it matches the requested stack and provides a broad middleware
  ecosystem. Fiber v3 currently requires Go 1.25+, which the local environment
  satisfies with Go 1.26.3.
- Follow-up: keep handlers thin and put domain logic in modules/services.
- Sources: https://docs.gofiber.io/next/

## PostgreSQL Access

- Problem: use PostgreSQL safely without hiding important forum queries behind
  hard-to-debug ORM behavior.
- Options: GORM, Ent, handwritten `pgx`, `pgx + sqlc`.
- Recommendation: `pgx/v5 + sqlc`.
- Reason: `pgx` is the PostgreSQL driver/toolkit; `sqlc` generates type-safe Go
  from SQL while keeping queries reviewable and database-specific.
- Follow-up: isolate generated code under `internal/store/sqlc`.
- Sources: https://pkg.go.dev/github.com/jackc/pgx/v5,
  https://docs.sqlc.dev/en/latest/

## Database Migrations

- Problem: evolve schema in a conventional, reviewable way.
- Options: custom migrations, `golang-migrate/migrate`, `goose`.
- Recommendation: `goose`.
- Reason: simple SQL migration files, common Go ecosystem usage, and easy local
  and CI execution.
- Follow-up: migrations live under `apps/api/internal/store/migrations`.
- Sources: https://github.com/pressly/goose

## Browser Sessions

- Problem: authenticate browser users with low operational complexity and good
  revocation behavior.
- Options: stateless JWT in browser storage, signed cookies only, Redis-backed
  server sessions.
- Recommendation: secure HTTP-only cookie with Redis-backed Fiber sessions.
- Reason: forums need reliable logout, revocation, rate-limiting integration,
  and role changes. Server sessions keep browser auth simpler than JWT refresh
  flows.
- Follow-up: enable CSRF protection for cookie-authenticated writes.
- Sources: https://docs.gofiber.io/next/middleware/session/,
  https://github.com/redis/go-redis

## Search

- Problem: provide fast topic/post search without making the search engine the
  source of truth.
- Options: PostgreSQL full-text search only, Meilisearch, OpenSearch.
- Recommendation: Meilisearch as a derived index fed from PostgreSQL.
- Reason: Meilisearch is simpler to operate than larger search clusters and
  fits forum search/discovery needs well. The index can be rebuilt from
  PostgreSQL.
- Follow-up: use an outbox or durable queue to keep writes and indexing events
  consistent.
- Sources: https://www.meilisearch.com/docs,
  https://github.com/meilisearch/meilisearch-go

## User Content Rendering

- Problem: render user-authored posts safely.
- Options: raw HTML, WYSIWYG HTML plus sanitizer, Markdown plus sanitizer.
- Recommendation: Markdown source rendered with `goldmark`, then sanitized with
  `bluemonday`.
- Reason: Markdown is easier to moderate, diff, store, and render consistently
  for a forum MVP. Sanitization remains required even when Markdown is used.
- Follow-up: choose allowed extensions and link policies before implementation.
- Sources: https://github.com/yuin/goldmark,
  https://github.com/microcosm-cc/bluemonday
