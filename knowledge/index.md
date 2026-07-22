# Knowledge Index

Project memory entry point for humans and AI sessions. Read this file, the one
hot handoff for the current workstream, and the relevant module note. Do not
load archived sessions or completed plans as current context.

## Active Workstreams

### Forum content revisions V1

- Status: **complete**; M6 authenticated browser QA and allowed/denied API
  coverage are complete. M7 is ready to start from the frozen M0-M6 decisions.
- Plan: `plans/2026-07-22-forum-content-revisions-v1.md`
- Decision: `decisions/2026-07-22-forum-content-revisions-ledger.md`
- Handoff: `sessions/2026-07-23-forum-content-revisions-v1-m6-handoff.md`
- Module: `modules/forum.md`

### Current HEAD regression remediation

- Status: **active**; M0 baseline frozen. Owns search, frontend typecheck,
  Page Registry, pagination/hydration, and extension gate regressions.
- Plan: `plans/2026-07-22-current-head-regression-remediation.md`
- Handoff: `sessions/2026-07-22-current-head-regression-plan-handoff.md`
- Modules: `modules/search.md`, `modules/frontend.md`

### Theme-consistent public resource 404

- Status: **ready**; close regression M7 before editing shared Page Registry
  and error-flow files.
- Plan: `plans/2026-07-22-theme-consistent-public-resource-404.md`
- Handoff: `sessions/2026-07-22-theme-consistent-public-resource-404-plan-handoff.md`
- Module: `modules/frontend.md`

### Theme-defined system error pages

- Status: **blocked after M0 audit**; M1+ waits for regression M7 or an explicit
  overlapping-file handoff. The focused public-resource 404 plan is its
  precursor.
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

## Current Project State

- **Web:** Nuxt 4, Vue 3, Nuxt UI 4, Bun, SSR-first, `zh-CN` default and
  `en-US` secondary locale.
- **API:** Go Fiber v3, PostgreSQL, Redis, River, Goose, and sqlc.
- **Forum:** taxonomy, topics/comments, moderation lifecycle, configurable
  policy, million-scale read-path work, and content-revision M2 are shipped.
- **Identity:** Redis sessions, RBAC, permission overrides, first-user
  `super_admin`, and account-session management are shipped.
- **Search:** protected PostgreSQL site search is the default; Meilisearch is an
  optional external plugin.
- **Extensions:** Manifest V3, exact-artifact trust, lifecycle, Host API v2,
  registries, Page Registry themes, and buildless settings UI are present;
  production-rewire honesty findings remain open.
- **Dev:** Compose owns PostgreSQL, Redis, and Mailpit. The user owns the web
  dev server on port 3000; do not kill it.

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
