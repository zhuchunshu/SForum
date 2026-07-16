# Architecture

Status: proposed on 2026-07-03 and updated as the scaffold evolves. This
document defines the target shape for ongoing implementation.

## Architecture Goals

- Build an SEO-friendly forum with server-rendered public pages.
- Support multilingual product features from the first implementation, with
  Simplified Chinese as the default language.
- Keep the frontend and backend independently understandable, testable, and
  deployable.
- Use mature framework-native or ecosystem libraries before creating custom
  infrastructure.
- Keep SForum core focused on the host framework, shared contracts, permissions,
  extension runtime, stable forum primitives, and first-class extension
  interfaces instead of bundling every optional product vertical.
- Deliver deployment-specific systems such as payments, outbound mail delivery,
  notification channels, analytics, and external integrations through plugins or
  explicit provider slots by default.
- Keep PostgreSQL as the source of truth; treat Redis and Meilisearch as
  supporting systems with rebuildable state.
- Use Laravel-inspired organization where it helps readability, but keep Go
  dependency wiring explicit and simple.

## System Shape

The default deployment shape should be same-origin:

```text
browser
  -> local browser or host reverse proxy
       -> ordinary HTTP -> Nuxt at 127.0.0.1:${WEB_PORT}
            -> pages at /
            -> /api/v1/* proxy route -> Fiber API at api:8080
       -> WebSocket Upgrade -> Fiber API at 127.0.0.1:${API_PORT}
            -> route ingress + Host guards + Registry Dispatcher
       -> Fiber API
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

## Core Framework And Plugin-First Boundary

SForum core is the host framework. It should provide the stable surface that a
forum and its extensions need, while keeping optional business verticals outside
the core runtime unless they are generic primitives.

Core owns:

- HTTP/SSR composition, API envelopes, OpenAPI contracts, localization, jobs,
  migrations, options, health checks, and deployment conventions.
- Identity, sessions, RBAC, API-authoritative policy checks, and permission
  catalog plumbing.
- Forum content primitives that define SForum as a forum: categories, topics,
  comments, posts, revisions, visibility, moderation boundaries, and search
  indexing contracts.
- The extension manager, plugin runtime, event catalog, filter/validate points,
  provider-slot registry, plugin route gateway, extension settings, and
  developer scaffolding.
- Small default or development adapters when they keep first-run behavior safe,
  such as no-op providers, local test providers, or protected built-in plugins
  that use the same extension APIs as uploaded plugins.

Core must not grow full vertical implementations for capabilities that vary by
deployment, vendor, or business model. These are plugin territory by default:

- Payment provider integrations, provider-specific subscriptions, refunds,
  invoice rendering, vendor ledgers, and vendor webhook parsing.
- Outbound mail provider implementations, SMTP/vendor credentials, digest
  delivery, campaign delivery, and mail-specific retry policy.
- Notification channels such as email, SMS, web push, chat integrations, and
  product-specific notification fanout rules.
- Analytics, third-party integrations, risk scoring providers, and alternative
  provider implementations for search, storage, human verification, or
  sanitization.

When a new capability is needed, design the host framework contract first:

1. Define the actor, action, protected resource, and API policy boundary.
2. Decide whether the capability needs a shared core model or only a lightweight
   extension point. Shared models are appropriate when SForum must coordinate
   state across plugins, such as payment intents, transaction records,
   entitlement checks, webhook idempotency, or notification preferences.
3. Choose the narrowest host contract: observe event, validate/filter event,
   provider slot, plugin route namespace, extension-owned admin page, or a
   small core framework module with provider interfaces.
4. Keep plugin routes under `/api/v1/extensions/{extensionId}/*`; plugins cannot
   override core routes or receive raw session cookies as authority.
5. Store extension-owned configuration in `extension_settings`. Store active
   provider selection in the owning core module only when host behavior depends
   on that selection, and provide one-click restore to the built-in default.
6. Put real provider or vendor logic in a plugin package. If SForum ships a
   default implementation, ship it as a protected built-in plugin rather than a
   hard-coded core service whenever practical.
7. Record architectural choices in `knowledge/decisions/` when the boundary will
   matter to future sessions.

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
- Human verification: disabled by default, with ALTCHA as the first supported
  self-hosted provider when deployments enable it. Verification remains
  server-side in Go with Redis-backed replay protection/rate limits; keep a
  narrow provider interface so Cloudflare Turnstile can be added later for
  deployments that want managed bot detection.
- Durable jobs and queues: River backed by PostgreSQL, wrapped by a small
  project-owned `app/Support/Jobs` package.
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

Target layout as implementation grows:

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
|       |-- app/
|       |   |-- Http/
|       |   |   |-- Controllers/
|       |   |   |   |-- Identity/
|       |   |   |   |-- Forum/
|       |   |   |   `-- Admin/
|       |   |   |-- Middleware/
|       |   |   |-- errors.go
|       |   |   `-- server.go
|       |   |-- Jobs/
|       |   |-- Models/
|       |   |   |-- Identity/
|       |   |   |-- Forum/
|       |   |   |-- Moderation/
|       |   |   `-- Search/
|       |   |-- Policies/
|       |   |-- Providers/
|       |   `-- Support/
|       |       |-- Jobs/
|       |       |-- Localization/
|       |       |-- Meili/
|       |       |-- Postgres/
|       |       |-- Redis/
|       |       `-- Storage/
|       |-- bootstrap/
|       |   `-- app.go
|       |-- config/
|       |   `-- config.go
|       |-- database/
|       |   |-- migrations/
|       |   |-- queries/
|       |   `-- sqlc/
|       |-- go.mod
|       `-- sqlc.yaml
|-- contracts/
|   |-- openapi.yaml
|   `-- openapi/
|       |-- components/
|       |-- paths/
|       `-- schemas/
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
- `apps/api` is a Go module named `github.com/zhuchunshu/sforum/apps/api`.
- `apps/api/bootstrap` is the API composition layer. It wires config, logging,
  database clients, session stores, module providers, HTTP routes, and worker
  dependencies. It should be the Go equivalent of Laravel's `bootstrap/app.php`:
  explicit and easy to read, not a hidden dependency container.
- `apps/api/app/Http` is the HTTP kernel. Controllers live under
  `app/Http/Controllers/<Area>` and stay thin: parse requests, call services,
  map domain errors, and return stable responses.
- `apps/api/app/Providers` contains explicit service providers that assemble
  controllers, services, stores, policies, jobs, and route providers for each
  area.
- `apps/api/app/Models/<Domain>` contains domain-facing Go packages: types,
  services, policies, repository interfaces, and persistence adapters for one
  domain. These are not Laravel Eloquent models; the name is a directory
  convention for SForum's Laravel-like project shape.
- `apps/api/app/Jobs` contains queued job definitions and handlers owned by the
  application. Shared River runtime, dispatch helpers, and infrastructure glue
  live under `apps/api/app/Support/Jobs`.
- `apps/api/app/Support/*` wraps external systems and reusable infrastructure
  clients such as PostgreSQL, Redis, Meilisearch, localization, storage, and
  queue support.
- `apps/api/config` contains application configuration loading.
- `apps/api/database/*` contains migrations, handwritten SQL, and generated
  `sqlc` code. Generated code should stay isolated from handwritten domain
  logic.
- `apps/web/i18n/locales/*` contains frontend message catalogs. Start with
  `zh-CN` and `en-US`, and keep Simplified Chinese complete before adding or
  changing user-facing features.
- `contracts/openapi.yaml` is the API contract entrypoint. Module-owned path
  items, schemas, shared parameters, and shared responses live under
  `contracts/openapi/`. Generate TypeScript types or a client for `apps/web`
  from the entrypoint once endpoints exist.
- The earlier top-level `src/` placeholder was retired once the `apps/`
  structure was created.

## Laravel-Inspired Backend Composition

SForum should borrow Laravel's application organization where it improves
readability: a small process entry, a visible bootstrap layer, explicit service
providers, route files, middleware groups, and thin controllers. It should not
try to recreate Laravel's dynamic container or PHP runtime behavior in Go.

Recommended mapping:

- `cmd/api/main.go`: process entry only. Load config, create the logger, call
  `bootstrap.NewAPI(...)`, start Fiber, and handle graceful shutdown.
- `bootstrap`: application assembly. Open PostgreSQL/Redis clients,
  build session stores, instantiate module providers, collect route providers,
  and return the HTTP app plus cleanup hooks.
- `config`: environment parsing and typed application settings.
- `app/Http`: HTTP kernel. Own Fiber configuration, global middleware,
  `/api/v1` grouping, health/system routes, centralized JSON error handling,
  and route-provider interfaces.
- `app/Http/Controllers/<Area>`: request/response DTOs, route declarations,
  and controller methods. Controllers parse input, call services, map module
  errors to stable API codes, and return responses.
- `app/Providers`: module composition. Build each area's store, service,
  policies, controllers, route provider, jobs, and seeds using dependencies
  passed from bootstrap.
- `app/Models/<Domain>`: domain logic and persistence boundaries for one area.
- `app/Support`: infrastructure wrappers and shared adapters.
- `database`: schema migrations, handwritten SQL queries, and generated SQL
  access code.

Route registration should be explicit and ordered. Prefer a small
`http.RouteProvider` interface and an explicit provider list over package-level
side effects, init-time registration, or filesystem scanning. A new module
should become reachable only after bootstrap adds its provider to the list.

Middleware belongs at the narrowest useful layer:

- Global middleware: request IDs, panic recovery, logging, and safe defaults.
- API group middleware: JSON/API conventions such as content negotiation,
  stable error shape, rate limiting, CSRF for cookie-authenticated writes, and
  locale negotiation when needed.
- Route group middleware: authentication, permission checks, human verification,
  and risk controls for specific capabilities.

Do not register routes from `cmd/*`, `app/Models/*`, `app/Support/*`, service
constructors, or database stores. Those packages may provide dependencies, but
route shape belongs to `app/Http` and controller route files.

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
- API and worker processes run embedded Goose migrations at startup when
  `MIGRATE_ON_STARTUP=true`. The shared migrator uses a PostgreSQL Goose lock,
  so parallel process starts serialize safely.
- Development Compose and production deploys may still run the same migration
  binary explicitly as a visible pre-start check; startup migration should then
  be an idempotent no-op.
- Local and production environment files should provide first-run fallback
  values for site URL, default locale, supported locales, and CAPTCHA settings;
  operators manage the runtime values from the admin site settings page.
- Production Compose publishes Nuxt and a dedicated Host API WebSocket ingress
  on `127.0.0.1:${WEB_PORT}` and `127.0.0.1:${API_PORT}` respectively. The API
  ingress is not a second public origin; Caddy sends only Upgrade traffic to it.
  PostgreSQL, Redis, and Meilisearch stay on the Compose network.
- Same-origin `/api/v1/*` requests should enter through Nuxt and proxy to Fiber
  internally at `api:8080`.
- Same-origin WebSocket Upgrade requests should bypass Nitro and enter Fiber
  directly so arbitrary Registry paths retain Host guard and Safe Mode authority.

## Backend Module Boundaries

### `identity`

Owns users, credentials, sessions, registration, email verification, password
reset, profile identity fields, roles, permissions, policy helpers, and
login/logout flows.

Recommended MVP auth model:

- Browser sessions with secure, HTTP-only cookies.
- Redis-backed Fiber sessions.
- Session ID regeneration on login and privilege escalation.
- Password hashing with Argon2id from `golang.org/x/crypto`.
- CSRF protection for cookie-authenticated writes.
- ALTCHA-backed human verification for registration and password-reset
  initiation, with risk-based challenges for suspicious login attempts.
- Redis-backed rate limits and single-use challenge tracking for
  anti-automation flows.
- One `users` table for regular members, moderators, and administrators.
- Open registration by default after bootstrapping.
- The first registered user becomes the protected initial `super_admin`.
- Later registered users receive the system `member` role by default.
- `member` can have a custom display alias, but its role key is stable and the
  role is not deletable while it remains the default registration role.
- Admin-managed custom roles/user groups are supported.
- Use database-backed RBAC and Go policy helpers first; keep the policy
  interface narrow enough to introduce Casbin later if the permission matrix
  becomes complex.

### `forum`

Owns categories, topics, posts, revisions, topic visibility, pin/lock/archive
states, slug rules, and read models used by public pages.

Recommended URL pattern:

- Category: `/c/:categorySlug`
- Topic: `/t/:topicID/:topicSlug`

Keep the numeric topic ID in the URL for stable lookup and allow slug changes to
redirect to the canonical URL.

Forum write flows should be able to call the shared human-verification boundary
for risk-based actions such as rapid replies, first posts from new users, or
posts containing links. The first posting milestone can start with rate limits
and email-verification gates; do not duplicate CAPTCHA logic inside forum
handlers.

### `moderation`

Owns reports, moderation actions, audit trail, soft deletion, staff notes, and
role-sensitive actions. Moderation must consume the identity module's actor and
policy helpers instead of duplicating permission checks in handlers. Start with
in-code policies for simple roles; evaluate Casbin only if the permission matrix
grows beyond ordinary forum roles.

### `jobs`

Owns the shared background job framework, worker runtime, queue configuration,
retry conventions, and dispatch API. SForum uses River with PostgreSQL as the
primary durable queue foundation. Redis is not the first durable job store; it
remains focused on sessions, cache, and rate limiting.

The framework should feel Laravel-inspired at the module boundary: application
packages define typed jobs, dispatch jobs from services, run workers by queue,
retry failed work, delay jobs, and keep failures visible. The implementation
remains Go-native and explicit through a small `app/Support/Jobs` wrapper
around River.

Initial queue names:

- `critical`
- `default`
- `search`
- `mail`
- `notifications`
- `maintenance`

Rules:

- Enqueue durable jobs in the same PostgreSQL transaction as the domain write
  whenever the job represents a side effect of that write.
- Keep job payloads compact and ID-based.
- Make handlers idempotent and retry-safe.
- Re-read current state inside handlers before applying side effects.
- Set worker concurrency per queue so slow external I/O cannot starve indexing
  or critical jobs.
- Chunk large rebuilds and fanout work into bounded jobs.

### `search`

Owns Meilisearch index settings, indexing jobs, rebuilds, and search endpoints.
Search indexes are derived data. PostgreSQL remains authoritative.

Use the River-backed durable job framework so post/topic changes can be
committed with indexing jobs in the same PostgreSQL transaction and processed
by `cmd/worker`.

### `notifications`

Core notification work is framework-only unless a product milestone explicitly
accepts more. Core may define notification events, preference contracts,
delivery-attempt records, queue names, and provider slots. Channel
implementations, digest policy, outbound mail delivery, SMS, web push, and
third-party messaging integrations should be plugins, including protected
built-in plugins if SForum later needs a bundled default.

### `payments`

Payments need a core framework contract before any provider plugin is useful.
If paid memberships, subscriptions, donations, orders, invoices, or other
monetization features enter scope, SForum core should define the provider-neutral
payment architecture:

- Permission keys, entitlement checks, and policy helpers.
- Provider-slot interfaces for creating payment intents, confirming/canceling
  payments, querying provider status, and requesting refunds when the product
  accepts refunds.
- Canonical payment intent, transaction, refund, and webhook-delivery records
  when SForum must reason about state across providers.
- Idempotent webhook gateway rules, event publication, retry visibility, and
  audit boundaries.
- Admin provider selection, reset-to-default behavior, and operator-safe
  configuration surfaces.

Plugins extend the payment framework. They implement provider adapters,
provider-specific transaction mapping, hosted checkout/session behavior,
refund support, invoice rendering, webhook payload verification/parsing, and
vendor-specific settings. Payment plugins must use the core provider slots and
canonical records instead of creating parallel payment state that core cannot
authorize, audit, or reconcile.

### `localization`

Owns backend locale negotiation, locale-aware email/notification templates,
supported-locale configuration, and translation keys for server-originated
messages. Backend API responses should include localized `message` values, and
Nuxt should display those messages first for API-originated prompts while using
frontend catalogs for frontend-owned UI states.

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
  reset attempts, human-verification challenge state, replay-protection keys,
  and short-lived cache entries.
- Meilisearch owns public search documents that can be rebuilt from PostgreSQL.
- Object storage should be S3-compatible when uploads enter scope. Use local
  MinIO in development and a hosted S3/R2-compatible service in production.

## API And SSR Contract

- Public read routes should be optimized for SSR page needs: category lists,
  topic lists, topic detail, and user profile summaries.
- Mutating routes should be under `/api/v1/*`, require session auth, and return
  the unified JSON API envelope.
- Nuxt admin routes live in the same web app under protected routes such as
  `/admin/*`; the Fiber API remains the source of truth for permission checks.
- API JSON responses must include integer `code`, localized `message`, and
  `data`. `code` equals the HTTP status code. Stable machine-readable error
  reasons live under `data.reason`.
- Keep OpenAPI modular: `contracts/openapi.yaml` should stay a small entrypoint,
  with route operations in `contracts/openapi/paths/`, reusable schemas in
  `contracts/openapi/schemas/`, and shared parameters/responses in
  `contracts/openapi/components/`.
- Validate split contract references with `ruby scripts/validate-openapi-refs.rb`
  after changing OpenAPI files.
- The frontend should display API `message` first for API-originated prompts and
  use `data.reason` for control flow or fallback behavior.
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
- Implement the identity foundation: open registration, first-user
  `super_admin` bootstrapping, Redis-backed sessions, default `member`
  assignment, and initial RBAC migrations.

## Open Architecture Questions

- Production host target: which single VPS/container host should the Docker
  Compose deployment optimize for first?
- Which built-in plugin or provider slot should be implemented first for
  transactional mail, if email verification or notification delivery enters the
  MVP path?
- Whether payments enter SForum scope at all, and if so which host-owned
  entitlement or provider-slot contracts are required before a payment plugin
  can be built safely.
- Upload support and object storage provider.
- Production backup destination and retention policy.
- Exact MVP role-management screens and permission seed list.
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
