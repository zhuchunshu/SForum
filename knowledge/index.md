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
- Development/deployment workflow has been proposed: Docker Compose for local
  and production orchestration, `scripts/dev.sh` for one-command hot-reload
  development, and bilingual `deploy.sh` for production operations.
- `scripts/dev.sh` now defaults to a faster bind-mount hot-reload loop without
  forced rebuilds or Compose Watch; use `--build` or `--watch` explicitly when
  needed.
- Frontend build/typecheck commands use isolated Nuxt temporary directories and
  generated output is ignored by dev watchers to avoid repeated reloads.
- Nuxt top-level `ignore` rules are intentionally narrower than Vite watcher
  ignores so dependency packages such as `@nuxt/ui/dist` remain discoverable by
  Nuxt component auto-imports.
- Docker Compose development and production now publish only the `web` service
  to `127.0.0.1:${WEB_PORT}`. API, PostgreSQL, Redis, Meilisearch, and support
  services stay on the Compose network, with `/api/v1/*` proxied through Nuxt.
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
- Identity and permissions architecture is accepted: SForum uses one user
  system for regular users, moderators, and administrators; the first registered
  user becomes the protected initial `super_admin`; later open registrations
  receive the undeletable default `member` role; admin-managed custom
  roles/user groups are supported.
- Security verification architecture is accepted: SForum uses ALTCHA as the
  default self-hosted human-verification provider for registration,
  password-reset initiation, and later risk-based actions, paired with
  Redis-backed rate limits and single-use challenge tracking.
- ALTCHA-backed registration human verification is implemented in the first
  identity slice. The API exposes `/api/v1/human-verification/challenge`,
  verifies ALTCHA payloads before account creation, stores replay/rate-limit
  state in Redis, and the Nuxt registration page sends the widget token through
  `humanVerification`.
- Backend API code has migrated to a Laravel-style directory shape while
  staying Go-explicit: `cmd/api` is process-focused, `bootstrap` assembles the
  runtime, `app/Http` owns the HTTP kernel, `app/Http/Controllers/*` owns
  controllers and routes, `app/Providers` owns provider wiring,
  `app/Models/*` owns domain logic, and `database/*` owns migrations, SQL, and
  generated `sqlc` code.
- Goose migrations now run automatically in the development Compose stack
  through a one-shot `migrate` service before API/worker startup. Production
  deploys run the same migration binary explicitly from `deploy.sh` after a
  PostgreSQL backup.
- Jobs and queues foundation implementation has started: River-backed durable
  queue support now lives under `apps/api/app/Support/Jobs`, `cmd/worker` uses
  `bootstrap.NewWorker`, and the first search job contract is
  `search.index_topic`.
- Backend API responses now have an accepted envelope design: every API JSON
  response must include integer `code`, localized `message`, and `data`; `code`
  equals the HTTP status code, and stable machine-readable reasons live under
  `data.reason`.

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
- `decisions/2026-07-04-laravel-style-http-routing.md` - accepted backend
  composition, route registration, and Laravel-style API directory decision.
- `decisions/2026-07-04-altcha-human-verification.md` - accepted ALTCHA human
  verification decision.
- `decisions/2026-07-04-api-response-envelope-localized-message.md` - accepted
  backend API envelope and localized message decision.
- `decisions/2026-07-04-configurable-admin-control-panel.md` - accepted
  configurable admin route prefix and Nuxt UI dashboard shell decision.
- `sessions/2026-07-04-altcha-human-verification-implementation.md` - ALTCHA
  implementation handoff.
- `sessions/2026-07-04-admin-foundation.md` - admin foundation implementation
  handoff.
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
- What default ALTCHA challenge expiration and work cost should production use?
