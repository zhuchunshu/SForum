# Agent Guide

This file is the shared working agreement for SForum. Read it before starting any new session.

## Project Intent

SForum is a maintainable, plugin-first, open-source forum framework. Core is
the host framework; product verticals and provider/vendor behavior live in
extensions. The repository is past scaffolding and actively implementing core
forum, identity, admin, and extension systems.

## Working Principles

- Prefer mature third-party libraries before building custom infrastructure. Before starting a module or feature, briefly check whether established libraries already solve the core problem well.
- Avoid messy, tightly coupled code. Keep responsibilities clear, name things plainly, and make the easy path maintainable.
- Split long implementations across multiple focused files. Avoid placing more than 1000 lines in one file unless there is a strong, documented reason.
- Keep changes scoped to the current task. Do not refactor unrelated areas just because they are nearby.
- Record decisions in the knowledge base when they will matter to future sessions.

## Stack And Directory Map

Monorepo with a Go API plus a Nuxt web app, plus shared contracts, extensions,
knowledge base, and tests.

- `apps/api` — Go 1.25 API + background worker. Module path
  `github.com/zhuchunshu/sforum/apps/api`. Laravel-style Go layout:
  - `cmd/` — `api` (HTTP server), `worker` (River queue worker), `migrate`
    (embedded Goose migrator), `sforum` (developer CLI; scaffolds
    plugins/themes, runs `seed:forum`).
  - `bootstrap/` — runtime assembly; `NewWorker` for the worker process.
  - `app/Http/` — HTTP kernel, controllers, and route registration
    (`app/Http/Controllers/*`).
  - `app/Models/` — domain logic and services.
  - `app/Providers/` — provider wiring (mail provider slot, attachment
    storage adapters, search, cache, etc.).
  - `app/Support/` — cross-cutting infra: `Jobs` (River durable queue),
    `Search` (Meilisearch), `Cache` (Redis CachedStore decorator).
  - `database/` — `migrations/` (Goose SQL), `migrator/` (shared embedded
    migrator), `queries/` + `sqlc/` (sqlc-generated code; config in
    `sqlc.yaml`).
  - `config/` — runtime config loaders. `config.Load` does NOT read `.env`;
    the dev scripts source `.env` and export vars explicitly.
- `apps/web` — Nuxt 4 / Vue 3 / Nuxt UI 4 frontend (package name
  `@sforum/web`). SSR-first (no `ssr: false` remains anywhere). Key deps:
  Bun, Tailwind, `@nuxtjs/i18n`, Tiptap, DOMPurify, altcha,
  `@iconify-json/tabler` + `@iconify-json/lucide`. App code lives under
  `apps/web/app/{components,composables,pages,layouts,middleware,plugins,config}`
  with the uppercase `SF` prefix component library, and admin pages under
  `apps/web/app/pages/admin`. i18n default locale `zh-CN`, secondary `en-US`.
- `contracts/` — modular OpenAPI contract. `openapi.yaml` is the entrypoint
  index only; paths in `openapi/paths/<module>.yaml`, schemas in
  `openapi/schemas/<module>.yaml`, shared components in `openapi/components/`.
- `extensions/` — `builtin/` (protected built-in plugins/themes, including
  the default theme `sforum-default`) and `dev/` (development extensions).
  Container images copy built-in themes to `/app/extensions/builtin`.
- `knowledge/` — project memory: `index.md`, `modules/`, `decisions/`,
  `sessions/`, `glossary.md`, `research.md`. Read before sensitive work.
- `tests/` — repo-level validation scripts (Playwright/Node) that hit the
  running dev servers.
- `scripts/` — dev/test orchestration shell scripts (see Commands below).
- `docs/`, `deploy/`, `compose*.yaml`, `deploy.sh` — deployment and ops.
- Runtime deps via Compose: PostgreSQL, Redis, Meilisearch, Mailpit.

## Commands

Run from the repo root unless noted. Before any `go get`/`go mod tidy`/
`bun install`/`bun add` network command, set the local proxy:
`export https_proxy=http://127.0.0.1:7897 http_proxy=http://127.0.0.1:7897 all_proxy=socks5://127.0.0.1:7897`

Development (the user runs `apps/web` dev server manually on port 3000 — do
not kill it):

- `./scripts/dev.sh` — start only dev dependencies (PostgreSQL, Redis,
  Meilisearch, Mailpit) via Compose and run migrations. `--build` rebuilds,
  `--no-migrate` skips migrations. Stops old Compose-managed frontend/backend
  containers first.
- `cd apps/web && bun run dev` — theme-aware Nuxt dev supervisor
  (`scripts/dev-theme-runtime.mjs`) that restarts `nuxt dev` with the active
  theme layer when `theme-releases/current.json` changes.
- `./scripts/api-dev.sh` — run the API with `air` (hot reload). Refuses to
  start if the API port is already in use; never stops a user process. In
  dev the API embeds the worker (`EMBED_WORKER_IN_API=true`).
- `./scripts/worker-dev.sh` — standalone worker via `.air.worker.toml`; only
  needed when `EMBED_WORKER_IN_API=false`.

Build, typecheck, lint, test:

- `cd apps/web && bun run build` — Nuxt build (uses `NUXT_BUILD_DIR=.nuxt-build`).
- `cd apps/web && bun run typecheck` — Nuxt typecheck (uses
  `NUXT_BUILD_DIR=.nuxt-typecheck`).
- `cd apps/web && bun run dev:plain` — plain `nuxt dev` without theme-layer
  supervisor; still acknowledges Web Releases via `active.json` so plugin
  enable/disable can finish. `dev:nuxt` is the absolute bare Nuxt process.
- `cd apps/api && go build ./...` / `go test ./...` — Go build and tests.
- `./scripts/test.sh` — full repo test gate: `go test ./...`, OpenAPI ref
  validation, Nuxt typecheck, and all `tests/validate-*.js|.ts` scripts.
- `ruby scripts/validate-openapi-refs.rb` — validate OpenAPI `$ref`s after
  editing any contract file.

CLI:

- `cd apps/api && go run ./cmd/sforum` — developer console: scaffold
  plugins/themes, `seed:forum` fake data. `seed:forum` is append-only,
  triggers no events, reads `DATABASE_URL` from env or `--database-url`.

## Core Framework And Plugin-First Development

SForum core is the host framework, not a place to accumulate every optional
product vertical. New capabilities must be designed around stable framework
contracts first.

- Treat payments, outbound mail delivery, notification channels, analytics,
  external integrations, provider-specific search/storage/security behavior,
  and similar deployment-specific systems as plugins by default.
- Core should expose the stable contracts that make plugins easy to build:
  explicit events, provider slots, typed payloads, permission checks, admin
  selection/reset UI, SDK helpers, scaffolding, tests, no-op defaults, and
  development adapters.
- When a product area needs shared semantics, such as payments, core may define
  the framework model and lifecycle: payment intents, transaction records,
  entitlement checks, webhook idempotency, provider interfaces, events, and
  admin configuration contracts. Provider/vendor behavior still belongs in
  plugins.
- Real provider or vendor logic should live in an extension package, including
  protected built-in plugins when SForum needs a bundled default.
- Do not let plugins override arbitrary core routes, monkey-patch core services,
  read raw session cookies as authority, or bypass API policy checks. Core-owned
  routes, events, filters, and provider slots are the only supported extension
  points.
- Before adding a core module for a new product area, write down why a plugin,
  provider slot, or event is insufficient. Record architectural choices in
  `knowledge/decisions/`.

## Beginner-Friendly Defaults

Every new feature, admin screen, configuration flow, and user-facing workflow
must be friendly to first-time operators and non-expert users from the first
version.

- Provide safe, working recommended defaults instead of requiring users to
  understand every technical setting before the feature can be used.
- Make the recommended path visually obvious in the UI with plain language,
  concise helper text, and familiar controls.
- Support one-click restoration to the recommended defaults for configurable
  features. If restoring defaults would preserve secrets, credentials, or other
  sensitive state, state that clearly in the UI.
- Avoid empty required configuration screens unless the feature truly cannot
  work without external credentials. In that case, explain the missing
  credential and keep non-credential defaults filled in.
- Add or update tests for default resolution and reset behavior when the
  feature stores runtime options, preferences, or other configurable state.

## Open-Source Framework Defaults

SForum is an open-source forum framework for different operators. Core forum
features must provide safe recommended defaults, but product behavior should
remain configurable unless a security or integrity rule requires a hard
boundary. Do not hard-code deployment-specific category names, tag policies,
theme assumptions, provider choices, or public-page availability into services.
Expose stable settings, events, provider slots, or admin controls instead, and
support one-click restoration to recommended defaults for operator-facing
configuration.

## Permission-Aware Development

The permission system is now part of the baseline architecture. When developing
any new feature, keep authorization in mind from the design stage instead of
adding it after the UI or endpoint is already complete.

- Identify the actor, action, and protected resource for every new route,
  mutation, admin screen, background action, or data export.
- Decide whether the behavior is public, login-required, role/permission
  protected, or reserved for `super_admin`.
- Reuse existing permission keys and policy helpers before adding new
  permissions. Add a new permission only when it represents a distinct product
  capability that admins should be able to grant or deny.
- Keep API policy checks authoritative. Frontend route guards, hidden buttons,
  disabled controls, and permission-aware navigation are only user-experience
  helpers.
- For unsafe requests and admin operations, add or update tests that cover both
  allowed and denied access paths.
- When a feature adds new permission keys, update seed data, permission catalog
  display text, OpenAPI/contracts when relevant, and the knowledge base.

## API Contract Workflow

OpenAPI is the shared contract between the Go API, Nuxt consumers, tests, and
future generated clients. Keep it modular as the product surface grows.

- Treat `contracts/openapi.yaml` as the entrypoint and index only. Do not grow
  it back into one giant handwritten contract file.
- Put route operations in `contracts/openapi/paths/<module>.yaml`, reusable
  schemas in `contracts/openapi/schemas/<module>.yaml`, and shared parameters
  or responses in `contracts/openapi/components/`.
- Split contract files by product/module ownership, such as identity, options,
  attachments, extensions, and system health.
- Use relative `$ref` values from the file that owns the reference. When moving
  a schema or path item, update references at the same time instead of relying
  on a later cleanup pass.
- For every new or changed endpoint, update the OpenAPI path, request/response
  schemas, error responses, permission/security notes when relevant, and any
  frontend API typing or tests that depend on the shape.
- Run `ruby scripts/validate-openapi-refs.rb` after editing OpenAPI files. Run
  `./scripts/test.sh` when the contract change is part of feature work.

## Frontend UI Conventions

- Do not use emoji as UI icons, decorative symbols, status markers, or action indicators.
- Use icons from an icon library whenever an icon is needed. Current approved choices are Tabler Icons and Nuxt Icon.
- Prefer the project's existing icon integration before adding a new icon package. Do not hand-roll inline SVG icons when an approved library icon exists.
- Use Toast feedback generously for user-triggered actions that succeed,
  complete, start a background task, copy data, reset settings, or save
  changes. Authentication success, create/update/delete success, restore
  defaults, uploads, exports, and queued jobs should normally show a Toast.
- Toast success styling must follow the active SForum appearance/theme tokens
  and admin personalization settings instead of Nuxt UI's default success
  green.
- Keep blocking errors, field-level validation, and guidance that the user must
  read next to the relevant form or page state. Error Toasts are appropriate for
  non-blocking operation failures, but must not replace field-level messages.
- Alerts and toast-style feedback must support automatic dismissal after 10
  seconds for non-error states. Error alerts and error Toasts must not
  auto-close; keep them visible until the user dismisses them or resolves the
  blocking issue.

## AI Working Discipline

The following rules are repository-level instructions for AI coding agents:

- 以瞎猜接口为耻，以认真查询为荣。
- 以模糊执行为耻，以寻求确认为荣。
- 以臆想业务为耻，以人类确认为荣。
- 以创造接口为耻，以复用现有为荣。
- 以跳过验证为耻，以主动测试为荣。
- 以破坏架构为耻，以遵循规范为荣。
- 以假装理解为耻，以诚实无知为荣。
- 以盲目修改为耻，以谨慎重构为荣。
- 以遗忘权限为耻，以主动建模为荣。

When writing code, keep the implementation simple and concise. Prefer built-in
functions and mature existing APIs over custom code. Do not over-validate
parameters, over-abstract, or add wrapper methods for simple behavior unless the
same logic is reused in multiple places. Avoid nested helper chains for similar
features; keep straightforward logic straightforward.

Develop the habit of writing useful comments while coding. Prefer Chinese
comments for project code unless surrounding code or external API conventions
make English clearer. Comments should explain non-obvious intent, constraints,
business rules, or tradeoffs; do not add empty comments that merely restate what
the next line of code already says.

## Network And Dependency Commands

The primary development environment may be in mainland China. The required
local proxy for all network-dependent package commands (`go get`,
`go mod tidy`, `bun install`, `bun add`, etc.) is listed at the top of the
Commands section above — set it before running such commands and re-apply it
when retrying network, DNS, registry, or module-download failures.

## Avoiding Hard-To-Maintain Code

Bad pattern:

- One giant file mixes routing, database queries, validation, rendering, and side effects.
- Feature logic depends on global mutable state without a clear owner.
- Copy-pasted branches handle similar cases with tiny differences.
- A function silently does many things: parses input, checks permission, writes data, sends notifications, and formats a response.

Better pattern:

- Keep routing, business logic, data access, validation, and presentation in separate modules once the codebase needs those boundaries.
- Use small functions with explicit inputs and outputs.
- Extract shared behavior when duplication becomes meaningful, not before.
- Add useful comments, preferably in Chinese, where they clarify non-obvious intent, constraints, business rules, or tradeoffs.

## Library-First Development

Before implementing a feature, do a short library survey:

1. Define the problem precisely.
2. Search for mature libraries or framework-native solutions.
3. Compare maintenance status, documentation quality, license, ecosystem fit, and complexity.
4. Record the chosen option in `knowledge/decisions/` when the choice affects architecture.

Examples:

- Use a proven authentication/session library instead of hand-rolling password storage and session security.
- Use a migration tool from the selected backend ecosystem instead of inventing a migration format.
- Use a mature rich-text/Markdown sanitizer instead of custom HTML filtering.

## File Size And Module Boundaries

- Aim for cohesive files that are easy to scan.
- Treat 500 lines as a prompt to review structure.
- Treat 1000 lines as a hard warning sign.
- Split by responsibility, not by arbitrary size alone.
- If a large generated file is unavoidable, label it clearly and keep handwritten logic elsewhere.

## Knowledge Base Workflow

The `knowledge/` directory is the project memory. It exists so a new AI session or human contributor can quickly understand where the project stands.

When starting work:

1. Read `knowledge/index.md`.
2. Read the relevant module note under `knowledge/modules/`.
3. Read recent handoffs under `knowledge/sessions/`.
4. Read relevant decisions under `knowledge/decisions/`.

When finishing work:

1. Update `knowledge/index.md` if navigation or project status changed.
2. Add or update module notes when a feature area changes.
3. Add a decision record for important technical/product choices.
4. Add a short session handoff when the next session will need context.

Recommended handoff format:

```md
# YYYY-MM-DD Session Handoff

## Changed

- ...

## Decisions

- ...

## Next

- ...

## Open Questions

- ...
```

## Current Status

- Actively implementing core forum, identity/RBAC, admin, attachment,
  extension (plugin/theme), SEO, mail, moderation, and search systems. See
  `knowledge/index.md` for the authoritative, frequently-updated status.
- Stack is decided and in place: Nuxt 4/Vue 3/Nuxt UI/Bun frontend; Go Fiber
  v3, PostgreSQL, Redis, Meilisearch backend; River durable queue; Goose
  migrations; sqlc; Redis-backed server sessions.
- The user manually starts the `apps/web` dev server (port 3000) during
  development. When port 3000 is occupied, assume it is the user's own
  running server — do not kill it without asking.
- Always read `knowledge/index.md` and the relevant `knowledge/modules/`
  note before changing sensitive areas, and update the knowledge base when
  finishing work (see Knowledge Base Workflow).
