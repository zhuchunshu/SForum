# Knowledge Index

This is the entry point for project memory.

## Current Project State

- Repository initialized on 2026-07-03.
- Basic documentation and knowledge-base skeleton created.
- First application scaffold has been added under `apps/web` and `apps/api`.
- Forum architecture stack has been proposed and foundation scaffolding has
  started.
- Proposed stack: Nuxt 4/Vue 3/Nuxt UI/Bun frontend; Go Fiber v3,
  PostgreSQL, Redis, and Meilisearch backend.
- Development/deployment workflow has been proposed: Docker Compose for
  production orchestration and local dependency services, local `bun run dev`
  plus `air` for frontend/API hot reload, and bilingual `deploy.sh` for
  production operations.
- `scripts/dev.sh` now starts only development dependencies: PostgreSQL, Redis,
  Meilisearch, and Mailpit. It stops old Compose-managed frontend/backend
  containers, waits for dependencies, and runs migrations by default; use
  `--no-migrate` only when testing dependency startup.
- Local frontend and backend processes read the repository root `.env`
  directly: Nuxt dev uses `--dotenv ../../.env`, and Air uses
  `env_files = ["../../.env"]`.
- Frontend build/typecheck commands use sibling Nuxt temporary directories
  (`.nuxt-build` and `.nuxt-typecheck`) instead of nesting under the dev
  server's `.nuxt`, and generated output is ignored by dev watchers to avoid
  repeated reloads.
- Frontend production preview runs the generated Nitro server directly through
  `HOST=0.0.0.0 bun --env-file=../../.env .output/server/index.mjs`, because
  the installed `nuxi preview` command does not support `--host` and misreads
  the host value as `ROOTDIR`.
- Nuxt top-level `ignore` rules are intentionally narrower than Vite watcher
  ignores so dependency packages such as `@nuxt/ui/dist` remain discoverable by
  Nuxt component auto-imports.
- Development Compose publishes dependency services to loopback-only host ports
  so local `air` and Nuxt can connect. Production Compose still publishes only
  the `web` service to `127.0.0.1:${WEB_PORT}`, with API and internal services
  staying on the Compose network.
- Product internationalization is required from the first implementation.
  Default locale is Simplified Chinese (`zh-CN`); first secondary locale is
  English (`en-US`).
- The standalone `forum-components.html` demo uses a Pine Teal clean forum UI
  direction: teal primary actions, light surfaces, thin borders, reduced
  radius, no gradient backgrounds, no emoji icons, and a dedicated
  status/feedback component section.
- The Pine Teal demo has been split into a reusable Nuxt component library under
  `apps/web/app/components/` using the uppercase `SF` prefix. The first library
  slice includes buttons, cards, inputs, toggles, avatars, feed rows, comments,
  search, editor, pagination, progress, skeleton, empty state, alerts, badges,
  toasts, and tabs, with a dev-only `/components` preview page. The preview page
  now covers seven forum-oriented sections: foundations, feedback, forum list,
  composer flow, moderation, member profile, and loading/empty states.
- The SForum homepage (`apps/web/app/pages/index.vue`) has been optimized with a wider max-w-[1376px] container and explicit column widths (Left 270px, Middle flexible up to 720px, Right 290px) on desktop to improve readability and breathing room.
- The thread feed row component (`SFFeedRow.vue`) has been redesigned using a compact no-excerpt layout (Left author avatar, Right title and upvote/reply actions inline, and bottom row metadata/views), doubling the layout information density.
- Sidebar accessibility was improved by fixing double padding via the `flush` property and updating text colors to `slate-500` and `slate-600` for higher contrast.
- The admin foundation now uses a dedicated Nuxt UI Dashboard shell with Nuxt
  Icon lucide icons. Source files stay under `apps/web/app/pages/admin`, while
  Nuxt rewrites the public route prefix from `NUXT_PUBLIC_ADMIN_ROUTE_PREFIX`,
  defaulting to `/control-panel`.
- The public forum navbar no longer shows an admin entry in the logged-in user
  dropdown, avoiding direct exposure of the configurable admin route prefix.
- Identity and permissions architecture is accepted: SForum uses one user
  system for regular users, moderators, and administrators; the first registered
  user becomes the protected initial `super_admin`; later open registrations
  receive the undeletable default `member` role; admin-managed custom
  roles/user groups are supported.
- Security verification architecture is accepted: SForum keeps human
  verification disabled by default, with ALTCHA as the first supported
  self-hosted provider for registration, password-reset initiation, and later
  risk-based actions when deployments enable it, paired with Redis-backed rate
  limits and single-use challenge tracking.
- ALTCHA-backed registration human verification is implemented as an opt-in
  identity slice. The API exposes `/api/v1/human-verification/challenge`,
  verifies ALTCHA payloads before account creation when enabled, stores
  replay/rate-limit state in Redis, and the Nuxt registration page sends the
  widget token through `humanVerification` only when the public runtime provider
  is `altcha`.
- The registration page now reads `/api/v1/auth/registration-status` and, when
  no user exists yet, warns that the first registered user will become the
  super administrator.
- Login/register error feedback is now user-actionable: login failures keep a
  single generic `auth.invalid_credentials` reason for safety, while
  registration validation returns localized `data.fields` messages for
  username, email, password, and human verification.
- Registration now validates editable fields and username/email conflicts before
  consuming ALTCHA tokens, constructs the returned current-user access inside
  the bootstrap transaction, and reports post-create session failures as
  `auth.session_unavailable` so users are guided to log in instead of retrying
  registration.
- Login and registration pages now hydrate the frontend auth state directly
  from the successful API response before navigating, avoiding an extra session
  refresh and keeping admin-route middleware from misreporting a successful
  registration as a form failure. The registration page also shows the current
  password rule below the password input.
- Browser authentication uses Redis-backed server sessions rather than
  JWT-first auth. Sessions now have a 30-day idle timeout, 180-day absolute
  timeout, 24-hour session-id renewal, login-time session reset, production
  Secure cookies, and login audit records with IP, User-Agent, time, and salted
  session hash.
- Frontend auth refresh now preserves the current user state during transient
  API restart/timeout/gateway failures, restores browser sessions during app
  startup when the API is available, and only redirects to login on confirmed
  401/`auth.required` responses.
- Backend API code has migrated to a Laravel-style directory shape while
  staying Go-explicit: `cmd/api` is process-focused, `bootstrap` assembles the
  runtime, `app/Http` owns the HTTP kernel, `app/Http/Controllers/*` owns
  controllers and routes, `app/Providers` owns provider wiring,
  `app/Models/*` owns domain logic, and `database/*` owns migrations, SQL, and
  generated `sqlc` code.
- Goose migrations now run automatically from `scripts/dev.sh` through a
  one-shot `migrate` service after dependency startup. Production deploys run
  the same migration binary explicitly from `deploy.sh` after a PostgreSQL
  backup.
- Jobs and queues foundation implementation has started: River-backed durable
  queue support now lives under `apps/api/app/Support/Jobs`, `cmd/worker` uses
  `bootstrap.NewWorker`, and the first search job contract is
  `search.index_topic`.
- Backend API responses now have an accepted envelope design: every API JSON
  response must include integer `code`, localized `message`, and `data`; `code`
  equals the HTTP status code, and stable machine-readable reasons live under
  `data.reason`.
- Runtime web options are now introduced through `web_options(name, value)`.
  `site.name` defaults to `SForum`, is cached in the backend Options service,
  is readable by the frontend through `useWebOptions().webOption()`, and can be
  edited from the admin site settings page by users with `settings.manage`.

## Navigation

- `decisions/` - decision records for architecture, product, and process choices.
- `modules/` - notes for each feature area or system module.
- `sessions/` - short handoffs from previous work sessions.
- `glossary.md` - shared terms and domain language.
- `research.md` - library and ecosystem research notes.
- `../docs/architecture.md` - proposed technical architecture and directory
  layout.
- `modules/identity.md` - identity, registration, sessions, roles, permissions,
  human verification, and policy notes.
- `modules/backend.md` - backend stack, module boundaries, jobs, and the
  Laravel-style Go/Fiber API directory structure.
- `modules/options.md` - runtime site options, `web_options` boundaries, API
  routes, and admin settings notes.
- `decisions/2026-07-04-laravel-style-http-routing.md` - accepted backend
  composition, route registration, and Laravel-style API directory decision.
- `decisions/2026-07-04-altcha-human-verification.md` - accepted ALTCHA human
  verification decision.
- `decisions/2026-07-04-api-response-envelope-localized-message.md` - accepted
  backend API envelope and localized message decision.
- `decisions/2026-07-05-browser-session-jwt-strategy.md` - accepted browser
  session lifetime, renewal, audit, and future JWT/API-token strategy.
- `decisions/2026-07-05-registration-altcha-default-disabled.md` - accepted
  registration ALTCHA default-off runtime behavior.
- `decisions/2026-07-04-configurable-admin-control-panel.md` - accepted
  configurable admin route prefix and Nuxt UI dashboard shell decision.
- `decisions/2026-07-05-local-dev-dependencies-and-processes.md` - accepted
  local development split where Compose starts dependencies and frontend/API
  run as host processes.
- `decisions/2026-07-05-admin-multitabs-and-layout-rules.md` - accepted admin
  multitabs, topbar breadcrumbs, larger tabs, and nested menu rules decision.
- `sessions/2026-07-04-altcha-human-verification-implementation.md` - ALTCHA
  implementation handoff.
- `sessions/2026-07-04-registration-status-notice.md` - first-user
  super-admin notice implementation handoff.
- `sessions/2026-07-04-admin-foundation.md` - admin foundation implementation
  handoff.
- `sessions/2026-07-05-registration-verification-session-failure.md` -
  registration verification ordering and session-failure fix handoff.
- `sessions/2026-07-05-registration-success-navigation.md` - registration
  password hint, success-state hydration, and middleware-safe API locale
  handoff.
- `sessions/2026-07-05-local-dev-dependencies.md` - local dependency startup
  and host-process development handoff.
- `sessions/2026-07-05-registration-altcha-default-disabled.md` -
  registration ALTCHA default-off implementation handoff.
- `sessions/2026-07-05-nuxt-dev-open-delay.md` - Nuxt dev 503 loading page,
  build/typecheck directory isolation, and local API port mismatch handoff.
- `sessions/2026-07-05-nuxt-preview-script.md` - Nuxt production preview script
  fix for the unsupported `nuxi preview --host` flag.
- `sessions/2026-07-05-admin-multitab-layout-upgrades.md` - admin multitabs,
  global topbar, theme adaptive sidebar, and nested menu layout upgrades handoff.
- `sessions/2026-07-05-public-navbar-hide-admin-entry.md` - public navbar admin
  entry removal handoff.
- `sessions/2026-07-05-auth-session-restart-resilience.md` - frontend auth
  refresh behavior for API restart/session recovery resilience.
- `../docs/superpowers/specs/2026-07-04-security-verification-design.md` -
  security verification design.
- `../docs/development-and-deployment.md` - proposed local development,
  hot-reload, Docker Compose, and production deployment workflow.
- `../apps/web` - Nuxt web scaffold with default `zh-CN` localization.
- `../apps/api` - Go Fiber API and worker scaffold.

## How To Use This In A New Session

1. Read `AGENTS.md`.
2. Read this file.
3. Open the latest handoff in `sessions/`.
4. Open related module notes.
5. Continue work and update these notes before stopping.

## Open Questions

- What is the first usable MVP scope?
- Which forum features are required versus later enhancements?
- What deployment target should the architecture optimize for?
- Should Meilisearch ship in the first executable milestone or immediately
  after core forum reads/writes?
- What production backup destination and retention policy should be used?
- Should English translations be mandatory for MVP launch or allowed to lag
  during internal development?
- Should email verification be required before posting in MVP?
- What ALTCHA challenge expiration and work cost should production use when
  human verification is explicitly enabled?
