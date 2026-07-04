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

## Product Internationalization

- Problem: all user-facing features must support multiple languages, with
  Simplified Chinese as the default and SEO-friendly localized public pages.
- Options: custom translation helpers, Vue I18n directly, Nuxt i18n.
- Recommendation: Nuxt i18n for the frontend, with backend stable error codes
  and a small backend localization module for emails/notifications.
- Reason: Nuxt i18n integrates Vue I18n with Nuxt routing and SEO metadata
  helpers. Keeping backend APIs code-based avoids coupling English or Chinese
  prose to API contracts.
- Follow-up: configure `zh-CN` as the default locale and `en-US` as the first
  secondary locale when `apps/web` is scaffolded.
- Sources: https://i18n.nuxtjs.org/docs/getting-started,
  https://i18n.nuxtjs.org/docs/guide/seo

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

## Authorization And Roles

- Problem: support one forum user system with regular members, moderators,
  administrators, open registration, protected first-user bootstrapping, and
  admin-managed custom roles/user groups.
- Options: in-code policy helpers over database RBAC, Casbin, OPA/Cedar-style
  policy engines, relationship-based authorization systems such as SpiceDB.
- Recommendation: start with database-backed RBAC plus small Go policy helpers.
- Reason: the MVP permission model is ordinary forum RBAC with a few protected
  system invariants. A narrow policy interface can adopt Casbin later if
  category-scoped roles or complex policies become necessary.
- Follow-up: seed system `super_admin` and `member` roles; keep role keys stable
  while allowing editable display aliases.

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

## Development And Deployment Orchestration

- Problem: run the full local stack with hot reload and deploy production with
  one repeatable command.
- Options: raw shell scripts only, Docker Compose, Kubernetes, platform-specific
  PaaS configuration.
- Recommendation: Docker Compose with dev/prod override files, plus
  `scripts/dev.sh` and a bilingual interactive `deploy.sh`.
- Reason: Compose is enough for a maintainable single-host forum deployment and
  also works well for local dependencies such as PostgreSQL, Redis, and
  Meilisearch. It avoids Kubernetes complexity while keeping services explicit.
- Follow-up: use Compose Watch when available, but provide a normal
  `docker compose up --build` fallback.
- Sources: https://docs.docker.com/compose/how-tos/file-watch/,
  https://docs.docker.com/compose/how-tos/profiles/,
  https://docs.docker.com/compose/how-tos/production/

## Go Hot Reload

- Problem: reload the Fiber API and worker when Go code changes during local
  development.
- Options: manual `go run`, `air`, `CompileDaemon`, custom watcher.
- Recommendation: `air` in development containers.
- Reason: `air` is a mature Go live reload tool and keeps local backend
  iteration close to the Nuxt dev-server experience.
- Follow-up: add `.air.toml` files once `apps/api` commands exist.
- Sources: https://github.com/air-verse/air
