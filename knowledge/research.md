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

## Backend Composition And Routing

- Problem: keep Fiber route registration, middleware, service construction, and
  module boundaries understandable as identity, forum, moderation, search, and
  jobs grow.
- Options: keep route registration directly in `app/Http`, add a full DI
  container, use package-level auto-registration, or adopt a Laravel-inspired
  explicit bootstrap/provider/routes pattern.
- Recommendation: use a Laravel-inspired organization with Go-explicit
  dependencies. Keep `cmd/api` small, assemble shared infrastructure in
  `bootstrap`, let `app/Providers` expose route providers, and keep route
  declarations in `app/Http/Controllers/*/routes.go`.
- Reason: Laravel's route files, middleware groups, and service providers are
  familiar and readable, but Go should keep dependencies visible instead of
  relying on runtime magic. This avoids route sprawl without introducing a
  general-purpose dependency container too early.
- Follow-up: when backend edits settle, refactor route registration to an
  explicit provider list and a small `http.RouteProvider` interface.
- Sources: https://laravel.com/docs/13.x/routing,
  https://laravel.com/docs/13.x/providers,
  https://laravel.com/docs/13.x/lifecycle,
  https://laravel.com/docs/13.x/middleware

## PostgreSQL Access

- Problem: use PostgreSQL safely without hiding important forum queries behind
  hard-to-debug ORM behavior.
- Options: GORM, Ent, handwritten `pgx`, `pgx + sqlc`.
- Recommendation: `pgx/v5 + sqlc`.
- Reason: `pgx` is the PostgreSQL driver/toolkit; `sqlc` generates type-safe Go
  from SQL while keeping queries reviewable and database-specific.
- Follow-up: isolate generated code under `database/sqlc`.
- Sources: https://pkg.go.dev/github.com/jackc/pgx/v5,
  https://docs.sqlc.dev/en/latest/

## Database Migrations

- Problem: evolve schema in a conventional, reviewable way.
- Options: custom migrations, `golang-migrate/migrate`, `goose`.
- Recommendation: `goose`.
- Reason: simple SQL migration files, common Go ecosystem usage, and easy local
  and CI execution.
- Follow-up: migrations live under `apps/api/database/migrations`.
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

## Human Verification And Anti-Automation

- Problem: open forum registration needs protection against automated account
  creation, password-reset abuse, credential-stuffing pressure, and early spam
  without making normal users solve hostile puzzles.
- Options: ALTCHA, Cloudflare Turnstile, hCaptcha, reCAPTCHA, or custom
  honeypots/rate limits only.
- Recommendation: ALTCHA by default, with Redis-backed rate limits and
  single-use challenge tracking. Keep a small provider interface so Turnstile
  can be added later for deployments that already use Cloudflare and accept a
  third-party managed service.
- Reason: ALTCHA is self-hostable, privacy-focused, has official server-side
  integration libraries including Go, supports server-generated challenges, and
  can verify payloads without normal API calls to an external CAPTCHA service.
  ALTCHA's own recommendations also call for single-use challenge tracking,
  expiration, and rate limiting, which fits the existing Redis plan.
- Follow-up: configure provider mode, HMAC secret, challenge expiration, and
  work cost during identity implementation.
- Sources: https://altcha.org/docs/v2/,
  https://altcha.org/docs/v2/server-integration/,
  https://altcha.org/docs/v2/security-recommendations/,
  https://github.com/altcha-org/altcha-lib-go,
  https://developers.cloudflare.com/turnstile/get-started/server-side-validation/,
  https://docs.hcaptcha.com/,
  https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html

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

## Jobs And Queues

- Problem: run search indexing, email delivery, notifications, cleanup, and
  maintenance work with Laravel-like ergonomics while keeping Go runtime
  behavior explicit and performance-first.
- Options: River with PostgreSQL, Asynq with Redis, Watermill as a message
  router, Temporal for durable workflows.
- Recommendation: River backed by PostgreSQL as the primary durable queue.
- Reason: PostgreSQL is already the source of truth, and River supports the
  important first requirement: enqueueing jobs from the same transaction as
  domain writes. That keeps post/topic writes and derived work such as search
  indexing consistent without introducing Redis as a second durable system.
- Follow-up: wrap River behind `app/Support/Jobs`; keep Redis available
  for sessions/cache/rate limits and possible later non-critical fast-lane
  jobs.
- Sources: https://riverqueue.com/docs,
  https://riverqueue.com/docs/transactional-enqueueing,
  https://riverqueue.com/docs/unique-jobs,
  https://github.com/hibiken/asynq,
  https://watermill.io/docs/,
  https://docs.temporal.io/develop/go/core-application

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
