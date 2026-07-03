# Architecture

Status: proposed on 2026-07-03. This document defines the target shape before
application code is added.

## Architecture Goals

- Build an SEO-friendly forum with server-rendered public pages.
- Support multilingual product features from the first implementation, with
  Simplified Chinese as the default language.
- Keep the frontend and backend independently understandable, testable, and
  deployable.
- Use mature framework-native or ecosystem libraries before creating custom
  infrastructure.
- Keep PostgreSQL as the source of truth; treat Redis and Meilisearch as
  supporting systems with rebuildable state.
- Use Laravel-inspired organization where it helps readability, but keep Go
  dependency wiring explicit and simple.

## System Shape

The default deployment shape should be same-origin:

```text
browser
  -> Nuxt web app at /
  -> Fiber API at /api/v1/*
       -> PostgreSQL
       -> Redis
       -> Meilisearch
       -> background worker
```

Nuxt renders forum pages on the server for first-load HTML, metadata, canonical
URLs, and crawler-friendly content. The Fiber API owns business rules, writes,
sessions, authorization, search indexing, and background work.

Nuxt server routes may proxy or aggregate API calls for SSR ergonomics, but they
must not become a second backend for forum domain logic.

## Chosen Stack

### Frontend

- Runtime/package manager: Bun.
- App framework: Nuxt 4 with Vue 3.
- Build layer: Vite through Nuxt's default builder.
- UI system: Nuxt UI with Tailwind CSS.
- Internationalization: Nuxt i18n with Simplified Chinese (`zh-CN`) as the
  default product locale and English (`en-US`) as the first secondary locale.
- SEO: SSR enabled by default, `useSeoMeta`, canonical URLs, route rules for
  hybrid rendering, and `@nuxtjs/seo` for robots, sitemap, OG image, and
  schema.org helpers.
- Testing: Vitest for unit/component tests and Playwright for critical user
  flows once UI exists.

Why: a plain Vue/Vite SPA would need extra work to make topics, categories, and
posts consistently indexable. Nuxt gives the project SSR, file-based routing,
data fetching, and Vite-backed development without splitting the frontend stack.

### Backend

- Language/runtime: Go 1.25+.
- HTTP framework: Go Fiber v3.
- Database: PostgreSQL.
- Database driver: `pgx/v5`.
- SQL access: `sqlc` generated query layer.
- Migrations: `goose`.
- Cache/session/rate-limit store: Redis.
- Redis client: `redis/go-redis/v9`; Fiber storage adapters may be used where
  they fit Fiber middleware.
- Search: Meilisearch with `meilisearch-go`, fed from PostgreSQL through a
  rebuildable indexer.
- Validation: `go-playground/validator/v10` through Fiber's struct validator.
- User content rendering: Markdown via `goldmark`; sanitize rendered HTML with
  `bluemonday`.
- Logging/observability: Go `log/slog` first; add OpenTelemetry when external
  tracing is introduced.

Why: Fiber matches the requested stack and provides middleware for common web
concerns. `pgx + sqlc` keeps database access explicit and type-checked without
the hidden behavior of an ORM. `goose` keeps schema evolution conventional and
easy to run in CI.

## Repository Layout

Target layout when implementation starts:

```text
.
|-- apps/
|   |-- web/
|   |   |-- app/
|   |   |   |-- assets/
|   |   |   |-- components/
|   |   |   |-- composables/
|   |   |   |-- layouts/
|   |   |   |-- middleware/
|   |   |   |-- pages/
|   |   |   |-- plugins/
|   |   |   `-- utils/
|   |   |-- public/
|   |   |-- server/
|   |   |   |-- api-proxy/
|   |   |   `-- middleware/
|   |   |-- i18n/
|   |   |   `-- locales/
|   |   |-- shared/
|   |   |-- nuxt.config.ts
|   |   `-- package.json
|   `-- api/
|       |-- cmd/
|       |   |-- api/
|       |   |-- migrate/
|       |   `-- worker/
|       |-- internal/
|       |   |-- bootstrap/
|       |   |-- config/
|       |   |-- http/
|       |   |   |-- middleware/
|       |   |   |-- routes.go
|       |   |   `-- errors.go
|       |   |-- modules/
|       |   |   |-- identity/
|       |   |   |-- forum/
|       |   |   |-- moderation/
|       |   |   |-- notifications/
|       |   |   |-- localization/
|       |   |   `-- search/
|       |   |-- platform/
|       |   |   |-- meili/
|       |   |   |-- postgres/
|       |   |   |-- redis/
|       |   |   `-- storage/
|       |   |-- store/
|       |   |   |-- migrations/
|       |   |   |-- queries/
|       |   |   `-- sqlc/
|       |   `-- testing/
|       |-- go.mod
|       `-- sqlc.yaml
|-- contracts/
|   `-- openapi.yaml
|-- compose.yaml
|-- compose.dev.yaml
|-- compose.prod.yaml
|-- deploy.sh
|-- deploy/
|-- docs/
|-- knowledge/
|-- scripts/
`-- tests/
```

Notes:

- `apps/web` is the Nuxt application. Keep forum business rules out of Nuxt
  server routes.
- `apps/api/internal/modules/*` are vertical domain modules. A module can own
  handlers, request/response DTOs, service methods, policies, and repository
  interfaces for one domain area.
- `apps/api/internal/platform/*` wraps external systems and infrastructure
  clients.
- `apps/api/internal/store/*` contains migrations, handwritten SQL, and
  generated `sqlc` code. Generated code should stay isolated from handwritten
  domain logic.
- `apps/web/i18n/locales/*` contains frontend message catalogs. Start with
  `zh-CN` and `en-US`, and keep Simplified Chinese complete before adding or
  changing user-facing features.
- `contracts/openapi.yaml` is the API contract. Generate TypeScript types or a
  client for `apps/web` from this file once endpoints exist.
- The earlier top-level `src/` placeholder was retired once the `apps/`
  structure was created.

## Development And Deployment

Development and deployment are part of the architecture, not afterthoughts. The
target workflow is documented in `docs/development-and-deployment.md`.

Summary:

- Local development should start with one command: `./scripts/dev.sh`.
- The development environment should include Nuxt, Fiber, the worker,
  PostgreSQL, Redis, and Meilisearch.
- Nuxt should use Vite HMR; Go services should use `air`; Docker Compose Watch
  should sync source changes into containers when available.
- Production should deploy with Docker Compose and `./deploy.sh`.
- `deploy.sh` should support English and Simplified Chinese prompts, first-time
  setup, deploy/update, migrations, backups, restore, logs, status, restart,
  stop, and rollback.
- Local and production environment files should default `APP_LOCALE` to
  `zh-CN` and list supported locales explicitly.
- Production should use same-origin routing through a reverse proxy, with `/`
  serving Nuxt and `/api/v1/*` serving Fiber.

## Backend Module Boundaries

### `identity`

Owns users, credentials, sessions, email verification, password reset, profile
identity fields, and login/logout flows.

Recommended MVP auth model:

- Browser sessions with secure, HTTP-only cookies.
- Redis-backed Fiber sessions.
- Session ID regeneration on login and privilege escalation.
- Password hashing with Argon2id from `golang.org/x/crypto`.
- CSRF protection for cookie-authenticated writes.

### `forum`

Owns categories, topics, posts, revisions, topic visibility, pin/lock/archive
states, slug rules, and read models used by public pages.

Recommended URL pattern:

- Category: `/c/:categorySlug`
- Topic: `/t/:topicID/:topicSlug`

Keep the numeric topic ID in the URL for stable lookup and allow slug changes to
redirect to the canonical URL.

### `moderation`

Owns reports, moderation actions, audit trail, soft deletion, staff notes, and
role-sensitive actions. Start with in-code policies for simple roles; evaluate
Casbin only if the permission matrix grows beyond ordinary forum roles.

### `search`

Owns Meilisearch index settings, indexing jobs, rebuilds, and search endpoints.
Search indexes are derived data. PostgreSQL remains authoritative.

Use an outbox-style table or durable job queue so post/topic changes can be
committed with an indexing event and processed by `cmd/worker`.

### `notifications`

Deferred for MVP unless needed early. Should own mention notifications, reply
notifications, email preferences, digest jobs, and delivery attempts.

### `localization`

Owns backend locale negotiation, locale-aware email/notification templates,
supported-locale configuration, and translation keys for server-originated
messages. The API should return stable machine-readable error codes; Nuxt should
localize normal UI and validation display from those codes.

Default behavior:

- Default locale: `zh-CN`.
- First secondary locale: `en-US`.
- Anonymous locale preference comes from the localized route, then locale cookie,
  then `Accept-Language`, then `zh-CN`.
- Signed-in users can store a preferred locale on their profile.
- Server logs and internal error details remain English-friendly for operations,
  while user-facing responses and templates are localizable.

## Data Ownership

- PostgreSQL owns canonical users, categories, topics, posts, revisions,
  reports, moderation events, and search outbox rows.
- Redis owns ephemeral sessions, rate limits, temporary verification/password
  reset attempts, and short-lived cache entries.
- Meilisearch owns public search documents that can be rebuilt from PostgreSQL.
- Object storage should be S3-compatible when uploads enter scope. Use local
  MinIO in development and a hosted S3/R2-compatible service in production.

## API And SSR Contract

- Public read routes should be optimized for SSR page needs: category lists,
  topic lists, topic detail, and user profile summaries.
- Mutating routes should be under `/api/v1/*`, require session auth, and return
  JSON problem-style errors with stable machine-readable codes.
- API errors should include a stable `code` and may include a default
  Simplified Chinese message. The frontend should map known codes to localized
  UI text so English and future languages do not depend on backend prose.
- Requests should carry locale context through route, cookie, profile, or
  `Accept-Language`; responses that depend on locale should set appropriate
  cache variation rules.
- Nuxt should call the same API contract used by browser interactions. During
  SSR, forward relevant cookies to the API.
- Prefer same-origin routing in production to avoid CORS complexity for browser
  sessions.

## Internationalization Rules

- All new user-facing product features must add translations at implementation
  time.
- Simplified Chinese (`zh-CN`) is the default and must be complete before a
  feature is considered done.
- English (`en-US`) is the first secondary locale.
- Public URL strategy should be `prefix_except_default`: Simplified Chinese
  pages use unprefixed canonical paths such as `/t/123/title`; English pages use
  `/en/*`.
- Use stable translation keys; do not hardcode user-facing strings inside Vue
  components, Go handlers, email templates, or deploy/runtime scripts.
- Store user locale preference separately from user-generated content language.
- User-generated forum content is not automatically translated. It may include a
  detected or selected content language later for filtering/search, but source
  content remains authored by the user.
- Seed data, category names, moderation reason labels, notifications, and emails
  must be localizable.
- Tests should include at least one route/page assertion for default
  Simplified Chinese and one for English once UI exists.

## SEO Rules

- Render public category, topic, post, and profile pages with SSR.
- Use canonical topic URLs and redirect stale slugs.
- Generate `robots.txt` and `sitemap.xml`.
- Generate localized `lang`, `hreflang`, canonical, and Open Graph locale tags
  for public pages.
- Use stable title/meta description templates per page type.
- Add structured data for forum discussion pages when page content exists.
- Hide private, deleted, draft, or moderation-only content from sitemap and
  search indexes.
- Keep pagination crawlable with canonical page rules.

## First Milestone Scope

Milestone 1 should establish architecture and tooling, not all forum features:

- Scaffold `apps/web` and `apps/api`.
- Add local development services for PostgreSQL, Redis, and Meilisearch.
- Add `compose.yaml`, `compose.dev.yaml`, `scripts/dev.sh`, and hot-reload
  development containers.
- Add config loading, logging, health checks, migrations, and one smoke test per
  app.
- Add `compose.prod.yaml`, `.env.production.example`, and a first version of
  the interactive bilingual `deploy.sh`.
- Add Nuxt i18n configuration, initial `zh-CN` and `en-US` locale catalogs,
  and backend locale configuration with `zh-CN` as the default.
- Define the initial OpenAPI contract skeleton.
- Create the first PostgreSQL schema migrations for users, categories, topics,
  posts, and post revisions.
- Implement session foundation only if authentication is part of the first
  executable milestone.

## Open Architecture Questions

- Production host target: which single VPS/container host should the Docker
  Compose deployment optimize for first?
- Registration policy: open signup, invite-only, or admin-created accounts?
- Email provider for verification and notifications.
- Upload support and object storage provider.
- Production backup destination and retention policy.
- Exact moderation roles and permission matrix.
- Whether search ships in MVP or follows immediately after core forum reads and
  writes.

## Sources Checked

- Nuxt 4 documentation: https://nuxt.com/docs/4.x/getting-started/introduction
- Nuxt UI documentation: https://ui.nuxt.com/docs/getting-started/installation/nuxt
- Nuxt i18n documentation: https://i18n.nuxtjs.org/docs/getting-started
- Nuxt i18n SEO guide: https://i18n.nuxtjs.org/docs/guide/seo
- Nuxt SEO: https://nuxtseo.com/
- Bun documentation: https://bun.sh/docs
- Fiber v3 documentation: https://docs.gofiber.io/next/
- Fiber session middleware: https://docs.gofiber.io/next/middleware/session/
- Fiber validation guide: https://docs.gofiber.io/next/guide/validation/
- `pgx/v5`: https://pkg.go.dev/github.com/jackc/pgx/v5
- `sqlc`: https://docs.sqlc.dev/en/latest/
- `goose`: https://github.com/pressly/goose
- `go-redis`: https://github.com/redis/go-redis
- Meilisearch documentation: https://www.meilisearch.com/docs
- `meilisearch-go`: https://github.com/meilisearch/meilisearch-go
- `goldmark`: https://github.com/yuin/goldmark
- `bluemonday`: https://github.com/microcosm-cc/bluemonday
