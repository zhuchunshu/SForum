# Knowledge Index

Project memory entry point for humans and AI sessions. Read this file, the one
hot handoff for the current workstream, and the relevant module note. Do not
load archived sessions or completed plans as current context.

## Active Workstreams

### Theme-defined system error pages

- Status: **ready after M0 audit**; M1+ directly reuses the completed 404 server
  pre-preparation, request-local presentation, exact-artifact validation,
  document policy, system AST renderer, and Core emergency fallback while
  continuing with 403/429/5xx.
- Plan: `plans/2026-07-22-theme-defined-system-error-pages.md`
- Audit handoff: `sessions/2026-07-22-theme-defined-system-error-pages-m0-audit-handoff.md`
- Modules: `modules/frontend.md`, `modules/extensions.md`

### V3 production rewire honesty remediation

- Status: **ready**; eight production-call-chain findings remain. Support-only
  evidence is not production closure.
- Plan: `plans/2026-07-22-v3-production-rewire-honesty-remediation.md`
- Prior partial evidence: `sessions/2026-07-22-p11-p12-p13-production-rewire-handoff.md`
- Module: `modules/extensions.md`

### Social login provider plugins

- Status: **ready**; Core owns accounts/sessions and provider plugins own
  GitHub, Google, Discord, and Telegram integration.
- Plan: `plans/2026-07-22-social-login-provider-plugins.md`
- Handoff: `sessions/2026-07-22-social-login-provider-plan-handoff.md`
- Module: `modules/identity.md`

### V3 P13 LTS residual

- Status: P0-P12 complete; implementable P13 work is closed except honesty
  remediation above and compatibility deletions gated by APILTS.
- Do not remove `sforum.protocol.v1` or
  `sforum.theme.l1.request-time-loader` before RemoveAfter around 2026-11-28
  plus zero-shim evidence.
- Plan: `plans/2026-07-13-trusted-plugin-theme-platform-v3.md`
- Progress ledger: `plans/2026-07-13-trusted-plugin-theme-platform-v3-progress.md`
- Handoff: `sessions/2026-07-21-trusted-plugin-theme-platform-v3-p13-lts-residual-handoff.md`
- Module: `modules/extensions.md`

### Content-addressed asset namespace

- Status: **implemented**; browser-facing theme and extension resources use
  Host-reserved `/_sforum` digest paths outside the JSON API namespace.
- Handoff: `sessions/2026-07-23-content-addressed-assets-handoff.md`.
- Decision: `decisions/2026-07-23-content-addressed-asset-namespace.md`.
- Residual: remove legacy `/api/v1/...assets...` compatibility paths only with
  APILTS/deprecation evidence.

## Current Project State

- **Web:** Nuxt 4, Vue 3, Nuxt UI 4, Bun, SSR-first, `zh-CN` default and
  `en-US` secondary locale.
- **API:** Go Fiber v3, PostgreSQL, Redis, River, Goose, and sqlc.
- **Forum:** taxonomy, topics/comments, moderation lifecycle, configurable
  policy, million-scale read-path work, and content revisions V1 are shipped.
- **Identity:** Redis sessions, RBAC, permission overrides, first-user
  `super_admin`, and account-session management are shipped.
- **Search:** protected PostgreSQL site search is the default; Meilisearch is an
  optional external plugin.
- **Extensions:** Manifest V3, exact-artifact trust, lifecycle, Host API v2,
  registries, Page Registry themes, and buildless settings UI are present;
  extension-owned permission localization is present; production-rewire
  honesty findings remain open.
- **Dev:** Compose owns PostgreSQL, Redis, and Mailpit. The user owns the web
  dev server on port 3000; do not kill it.

## Latest Handoff

- Tags heat overview `/tags` recovery and final QA:
  `sessions/2026-07-23-tags-heat-overview-handoff.md`

## Recently Completed

- Default-theme `/categories` grouped directory: implemented against confirmed
  Demo 01 with real category-group API data, SSR internal API reads, focused
  tests, typecheck, production build, and browser screenshot QA.
- Default-theme public profile B1: `/u/{username}` now uses the shared
  three-column shell with real public activity grouped by date, real public
  stats/recent topics, self-only edit UI, mobile drawers, i18n copy, OpenAPI
  updates, and focused Go/Bun/browser validation.
- Default-theme notifications and topic composer now use the shared responsive
  three-column shell with real API flows, mobile drawers, persistent errors,
  appearance-aware feedback, and focused Browser QA.
- Theme-consistent public resource 404: M0-M6 completed 2026-07-23. Ordinary
  semantic 404s retain the healthy selected theme and real private HTTP 404;
  Core is emergency-only. See
  `plans/2026-07-22-theme-consistent-public-resource-404.md`.

## Other Open Product Tracks

- Iteration A engagement: `plans/2026-07-12-iteration-a-engagement-loop.md`
- Admin settings blueprint: `plans/2026-07-12-admin-settings-richness.md`
- Extension surface density: `plans/2026-07-12-extension-surface-density.md`
- Strategic direction: `plans/2026-07-12-development-directions.md`

## Navigation

| Path | Role |
| --- | --- |
| `modules/` | Living feature state; prefer over dated prose |
| `sessions/` | One concise handoff per active workstream |
| `sessions/archive/YYYY-MM/` | Cold historical checkpoints; do not load by default |
| `plans/` | Active, ready, blocked, and blueprint task books |
| `plans/archive/YYYY-MM/` | Completed, cancelled, and superseded task books |
| `decisions/` | Architecture/product ADRs; keep and mark replacements in-file |
| `reports/` | Point-in-time performance and security evidence |
| `glossary.md` | Shared framework terms |

Historical references that require verification before use:

- `knowledge/archive/architecture-maturity-audit.md` -- 2026-07-12 pre-V3 scorecard
- `knowledge/archive/legacy-sforum-feature-gap.md` -- partially stale old-product comparison
- `knowledge/archive/research.md` -- early stack/library research; accepted ADRs supersede it

## Session Rules

1. Read root `AGENTS.md`, this index, one relevant module note, and only the
   active plan/handoff required for the task.
2. Do not read `sessions/archive/` or `plans/archive/` unless recovering
   historical evidence.
3. Update the living module note instead of appending completed work here.
4. Keep one hot handoff per active workstream and archive it when the track
   closes or a newer handoff supersedes it.
5. For V3 percentages and LTS residuals, trust the active progress ledger, not
   archived session prose.

## Open Questions

- Production backup destination and retention policy for operator deployments.
- Whether `en-US` copy must be complete for the first public release.
- Category-scoped ACL timing relative to global RBAC.
- Timing of the next architecture maturity re-audit after V3 LTS cleanup.
