# Knowledge Index

Project memory entry point for humans and AI sessions. Read this file, the one
hot handoff for the current workstream, and the relevant module note. Do not
load archived sessions or completed plans as current context.

## Active Workstreams

### Tri-State Color Mode Reliability

- Status: **ready**; Automatic/Light/Dark product semantics are approved, the
  `localhost` versus `127.0.0.1` persistence split is reproduced, and M0-M5
  implementation/verification remain.
- Plan: `plans/2026-07-27-tristate-color-mode-reliability.md`
- Handoff:
  `sessions/2026-07-27-tristate-color-mode-plan-handoff.md`
- Decision:
  `decisions/2026-07-27-tristate-color-mode-preference.md`
- Module: `modules/frontend.md`

### Configurable Public Navigation Platform

- Status: **ready**; topbar operator rows and plugin append are present, while
  shared topbar/sidebar/mobile/footer placement, accessible sorting, defaults,
  snapshots, backup, and full V3 plugin/theme lifecycle wiring remain open.
- Plan: `plans/2026-07-27-configurable-public-navigation-platform.md`
- Handoff:
  `sessions/2026-07-27-public-navigation-platform-plan-handoff.md`
- Decision:
  `decisions/2026-07-27-operator-owned-public-navigation.md`
- Modules: `modules/options.md`, `modules/frontend.md`,
  `modules/extensions.md`

### Notification Platform V2

- Status: **ready**; V1 inbox/mail projection is present, but direct topic
  replies, approval-time reply/mention fanout, user preferences, plugin
  notification contracts, realtime refresh, and external channels remain open.
- Plan: `plans/2026-07-27-notification-platform-v2.md`
- Handoff: `sessions/2026-07-27-notification-platform-plan-handoff.md`
- Decision: `decisions/2026-07-27-notification-platform-v2.md`
- Module: `modules/notifications.md`

### V3 production rewire honesty remediation

- Status: **ready**; eight production-call-chain findings remain. Support-only
  evidence is not production closure.
- Plan: `plans/2026-07-22-v3-production-rewire-honesty-remediation.md`
- Prior partial evidence: `sessions/2026-07-22-p11-p12-p13-production-rewire-handoff.md`
- Module: `modules/extensions.md`

### Built-in GitHub social login

- Status: **整改完成，等待独立复审** (2026-07-27). R1-R7 now have focused,
  isolated runtime, and Browser evidence; the generic Host/plugin architecture
  and protected built-in reference remain subject to independent acceptance.
- Plan: `plans/2026-07-27-github-social-login-builtin-plugin.md`
- Remediation:
  `plans/2026-07-27-external-auth-core-plugin-review-remediation.md`
- Handoff:
  `sessions/2026-07-28-github-plugin-restart-handoff.md`
- Prior remediation handoff:
  `sessions/2026-07-27-external-auth-review-remediation-handoff.md`
- Evidence matrix:
  `reports/2026-07-27-external-auth-r1-r7-requirements-evidence-matrix.md`
- Decision: `decisions/2026-07-27-github-social-login-builtin-v1.md`
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
  optional external plugin; Host-ledger reconciliation repairs missing, stale,
  and obsolete documents automatically.
- **Extensions:** Manifest V3, exact-artifact trust, lifecycle, Host API v2,
  registries, Page Registry themes, theme-defined virtual system error pages,
  and buildless settings UI are present; extension-owned permission
  localization is present; production-rewire honesty findings remain open.
- **Dev:** Compose owns PostgreSQL, Redis, and Mailpit. The user owns the web
  dev server on port 3000; do not kill it.

## Latest Handoff

- GitHub built-in restart repair: dedicated Host restart orchestration,
  correct built-in trust preview semantics, exact staged-target recovery,
  legacy-to-Lifecycle-V2 bridge, and successful browser/database evidence
  without 409 or exact-fence conflict:
  `sessions/2026-07-28-github-plugin-restart-handoff.md`
- Architecture boundary guardrails: future feature work is now constrained by
  domain placement, fixed Core Tab components, Go package responsibility, and
  baseline-enforced non-growth for large files/flat packages/God objects:
  `sessions/2026-07-28-architecture-boundary-guardrails-handoff.md`
- Runtime public URL override: admin `site.url` may be cleared to inherit
  environment `APP_URL`; public consumers receive the resolved value while
  OAuth/CSRF/cookie security stays environment-owned:
  `sessions/2026-07-27-app-url-runtime-fallback-handoff.md`
- Extension artifact-presence reconciliation: DB-retained plugin/theme records
  expose `artifactState`; missing packages fail closed and super admins can
  explicitly batch-uninstall eligible records through an atomic, data-aware
  confirmation flow:
  `sessions/2026-07-27-extension-artifact-presence-handoff.md`
- Search automatic reconciliation: provider-neutral Host ledger, startup + 15m
  bounded repair schedule, and real-runtime cleanup of 92 historical ghosts:
  `sessions/2026-07-27-search-auto-reconciliation-handoff.md`
- Tri-state color-mode task book: shared preference authority, explicit
  Automatic/Light/Dark UI, canonical local origin, cache-safe persistence, and
  M0-M5 one-conversation milestones:
  `sessions/2026-07-27-tristate-color-mode-plan-handoff.md`
- Configurable public navigation task book: Core-owned placement/defaults/
  backup, theme-owned presentation, bounded V3 plugin injection, and M0-M7
  one-conversation milestones:
  `sessions/2026-07-27-public-navigation-platform-plan-handoff.md`
- Notification Platform V2 task book: Core recipient correctness, layered
  admin/user preferences, namespaced plugin emission, durable-revision SSE, and
  a Web Push reference channel:
  `sessions/2026-07-27-notification-platform-plan-handoff.md`
- Built-in GitHub social login: R1-R7 remediation packet completed with
  isolated HTTP/Browser artifacts; independent review remains required:
  `sessions/2026-07-27-external-auth-review-remediation-handoff.md`
- Topic create/edit selected-theme and Core-fallback shell parity, exact
  artifact activation, CSS validator boundary fix, and editor Host Island
  allowlist:
  `sessions/2026-07-27-topic-composer-shell-parity-handoff.md`
- Topic side card **贡献者**: author+editors avatar group (max 5), public
  contribution timeline modal, `TopicDetail.contributors` +
  `GET /topics/{id}/contribution-timeline` (no body/reason; staff exposed):
  `sessions/2026-07-27-topic-contributors-handoff.md`
- Editor-document edit load fix: topic/comment editors restore native Tiptap
  JSON via `forumEditorInitialContent` → `SFEditor.initialContent`, never seed
  Markdown v-model with `rawContent`; save uses `forumContentFromEditorPayload`:
  `sessions/2026-07-27-editor-document-edit-load-fix.md`
- Comment stream visual refresh: deep-link highlight now light-sweep animation;
  header meta regrouped (time · floor right-aligned, OP badge, one-line reply
  quote): `sessions/2026-07-26-comment-stream-visual-refresh-handoff.md`
- `/settings/profile` + `/settings/security` three-column chrome extracted into
  shared `SFSettingsShell` (slots: main / rail / head-actions; supersedes the
  07-24 "no SettingsShell yet" decision):
  `sessions/2026-07-26-settings-shell-refactor-handoff.md`
- Profile Replies tab deep links now include `/page/N#comment-{id}` via API
  `commentPage` (SSR cannot see URL hash alone):
  `sessions/2026-07-24-profile-reply-comment-page-links.md`
- Comment cross-page anchor SSR resolve (topic page) + path-style `/page/N`:
  `sessions/2026-07-24-comment-cross-page-anchor-handoff.md`
- `/moderation` host chrome sidebar tokens aligned with default-theme hybrid:
  `sessions/2026-07-24-moderation-sidebar-token-parity-handoff.md`

## Recently Completed

- Settings chrome: shared `sforum-settings` CSS shell, `SFSettingsAccountNav`,
  profile preview rail, security summary rail; fullwidth theme templates;
  chrome now componentized as `SFSettingsShell` (2026-07-26).
- `/moderation` workbench sidebars: reuse `SFHomeNavigation`, notifications rail
  section language, shared mobile drawer keys; queue overview large count;
  decision rail restyle; typecheck + workbench unit tests pass.
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
- Theme-defined system error pages: completed 2026-07-23. The selected public
  theme can own L0/L1 presentation for 403, 404, 429, and 500/502/503/504
  virtual surfaces while Host preserves status, safe copy/actions, no-store,
  noindex/nofollow, and Core emergency fallback. See
  `plans/2026-07-22-theme-defined-system-error-pages.md`.
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
