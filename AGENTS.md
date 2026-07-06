# Agent Guide

This file is the shared working agreement for SForum. Read it before starting any new session.

## Project Intent

SForum is intended to become a maintainable forum project. The current repository is intentionally documentation-first: do not add application code until the stack, module boundaries, and first milestone are agreed.

## Working Principles

- Prefer mature third-party libraries before building custom infrastructure. Before starting a module or feature, briefly check whether established libraries already solve the core problem well.
- Avoid messy, tightly coupled code. Keep responsibilities clear, name things plainly, and make the easy path maintainable.
- Split long implementations across multiple focused files. Avoid placing more than 1000 lines in one file unless there is a strong, documented reason.
- Keep changes scoped to the current task. Do not refactor unrelated areas just because they are nearby.
- Record decisions in the knowledge base when they will matter to future sessions.

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
- Admin alerts and toast-style feedback must support automatic dismissal after
  10 seconds for non-error states. Error alerts must not auto-close; keep them
  visible until the user dismisses them or resolves the blocking issue.

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

The primary development environment may be in mainland China. Before running
network-dependent package commands such as `go get`, `go mod tidy`,
`bun install`, `bun add`, or similar dependency downloads, use the configured
local proxy:

```sh
export https_proxy=http://127.0.0.1:7897 http_proxy=http://127.0.0.1:7897 all_proxy=socks5://127.0.0.1:7897
```

Keep this proxy setting in mind when retrying failed dependency commands that
look like network, DNS, registry, or module download issues.

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

- Git repository initialized.
- Documentation and knowledge-base skeleton created.
- First application scaffold has been added under `apps/web` and `apps/api`.
- The user manually starts the `apps/web` dev server (port 3000) during development. When port 3000 is occupied, assume it is the user's own running server — do not kill it without asking.
