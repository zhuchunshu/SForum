# Knowledge Index

Project memory entry point for humans and AI sessions. Read this file, the one
hot handoff for the current workstream, and the relevant module note. Do not
load archived sessions or completed plans as current context.

## Active Workstreams

### Configurable Public Navigation Platform

- Status: **active, M0-M6 complete**; all four public locations share the
  canonical actor-sensitive navigation authority, sidebar taxonomy expands
  through `core.dynamic.categories`, stored placements now match the admin
  document (footer navigation defaults empty), and exact active-theme capability
  is projected from immutable runtime state. M7 is the final lifecycle and
  release gate.
- Plan: `plans/2026-07-27-configurable-public-navigation-platform.md`
- Handoff:
  `sessions/2026-07-28-public-navigation-platform-handoff.md`
- Decision:
  `decisions/2026-07-27-operator-owned-public-navigation.md`
- Modules: `modules/options.md`, `modules/frontend.md`,
  `modules/extensions.md`

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
- **Notifications:** V2 is complete: transactional reply/mention/moderation
  fanout, layered policy and own-user preferences, exact-artifact plugin
  emission, durable-revision SSE with REST/reconnect fallback, generic channel
  delivery, and the protected Web Push reference provider are shipped.
- **Dev:** Compose owns PostgreSQL, Redis, and Mailpit. The user owns the web
  dev server on port 3000; do not kill it.

## Latest Handoff

- Lifecycle V2 settings restart: enabled plugins such as
  `sforum.auth-github` now preflight before persistence and restart the exact
  active artifact through Host lifecycle orchestration; stable recovery errors
  replace the prior generic 500, while operator verification remains:
  `sessions/2026-07-29-lifecycle-v2-settings-restart-fix.md`
- Stale extension permission suggestions: approval remains exact-artifact
  fail-closed, while authorized rejection can now close pending records after
  disable, uninstall, missing-artifact cleanup, or replacement:
  `sessions/2026-07-29-stale-extension-role-suggestion-rejection.md`
- Default site brand assets: the supplied V3 mark now ships at
  `/brand/sforum-logo.svg`; empty runtime Logo/favicon options resolve to it
  without storing the fallback, while operator URLs remain authoritative:
  `sessions/2026-07-29-default-site-brand-assets.md`
- Admin attachment navigation: Attachment Configuration and Attachment
  Management are independent permission-aware routes; Attachment Configuration
  contains Basic Configuration and Compression Configuration tabs; Basic
  Configuration now includes persistent field guidance and units, while the old
  URL redirects compatibly and the governance extension placement follows
  Attachment Management; operator verification remains:
  `sessions/2026-07-29-admin-attachment-submenus-handoff.md`
- Personalization brand assets: compact click/drag uploads now auto-fill URL
  plus attachment ID and show previews; brand SVG is safely rasterized to PNG
  without weakening the ordinary attachment denylist:
  `sessions/2026-07-29-personalization-brand-asset-upload.md`,
  `decisions/2026-07-29-brand-svg-rasterization.md`
- Category directory layout: removed the duplicate desktop top offset, moved
  group focus into the main toolbar, and reduced both side rails; immutable
  theme refresh and desktop/mobile manual verification remain:
  `sessions/2026-07-29-category-directory-layout-handoff.md`
- Forum content relative time: topic/comment publish and accepted-edit times
  now use a one-month relative window with site-timezone `Y-m-d H:i:s`
  fallback; automated and manual verification remain for the user:
  `sessions/2026-07-29-forum-content-relative-time-handoff.md`
- Canonical public search route: `/search?q=...` now owns a complete
  `forum.search` Page Registry/theme surface; old homepage search URLs redirect,
  and operator runtime verification remains:
  `sessions/2026-07-29-public-search-route-handoff.md`
- Announcement authoring: labeled bilingual create form, field-oriented
  SForum editor preset, time-window controls, local validation, and
  server-sanitized Markdown presentation; operator Browser QA remains:
  `sessions/2026-07-29-announcement-editor-handoff.md`
- Missing-artifact uninstall routing: per-row cleanup now targets only the
  selected missing extension, ordinary lifecycle DELETE fails closed, and
  immutable publication history no longer causes refresh-time reappearance:
  `sessions/2026-07-29-extension-missing-artifact-uninstall-routing.md`
- GitHub Actions and security remediation: reusable CI/security checks, four
  multi-platform GHCR images, version-pinned deployment, gRPC remediation, and
  Fiber `v3.4.0` across all 18 source Go modules with session failure cleanup;
  the Buf tool graph now uses patched CEL, text, and compression dependencies;
  redirect targets, database pool integers, and stored Argon2 verification now
  reject unsafe input before crossing their runtime boundaries; the unpatched
  imaging dependency is gone and JPEG/PNG transforms now explicitly reject
  TIFF; uploaded extension ZIP paths now have a CodeQL-visible strict source
  guard plus independent snapshot containment checks; external-auth runtime
  evidence now logs only credential-neutral primitive projections; release
  tags now inject one shared Core/Web/API/worker/migrator/CLI build identity,
  shown once beside the SForum admin brand, while the protected overview shows
  build and runtime diagnostics; the bilingual `scripts/release.sh` validates
  and pushes the annotated release tag without building locally; repository
  licensing is now explicitly MIT under Inkedus while separately licensed
  third-party material retains its own terms:
  `sessions/2026-07-29-github-actions-release-pipeline-handoff.md`,
  `decisions/2026-07-29-mit-project-license.md`
- Notification email opt-in defaults: admin per-type email control, Core
  transaction-scoped user preference enforcement, hard site gate, safe default
  migration, personal managed state, and desktop/mobile Browser QA:
  `sessions/2026-07-28-notification-email-opt-in-handoff.md`
- Notification stream lifecycle hotfix: downstream disconnect now destroys the
  Nuxt upstream SSE socket, API subscriptions have one-minute leases, and the
  client uses bounded backoff plus HMR cleanup instead of native retry storms:
  `sessions/2026-07-28-notification-stream-lifecycle-hotfix-handoff.md`
- Notification detail type navigation: list/detail now share the same type
  rail, loaded-scope counts, current-type highlight, and filter-state return;
  focused verification passed while rendered Browser QA remains to repeat
  after the Chrome control timeout:
  `sessions/2026-07-28-notification-detail-type-nav-handoff.md`
- Notification list source identity: reply/mention rows now use the actor's
  configured avatar, system rows use Tabler icons, hidden targets scrub actor
  summaries, and desktop/mobile Browser QA is complete:
  `sessions/2026-07-28-notification-source-avatar-handoff.md`
- Topic detail stale-comment repair: `/t/**` whole-page SWR removed, explicit
  anonymous/session cache-control policies, API generation caches retained,
  and direct anchor/runtime proof with all current comments:
  `sessions/2026-07-28-topic-page-cache-correctness-handoff.md`
- Unified Admin Mail and Notification settings: one permission-aware fixed-tab
  surface, compact shared settings geometry, legacy notification-route redirect,
  and desktop/mobile Browser QA:
  `sessions/2026-07-28-admin-mail-notification-settings-parity-handoff.md`
- Notification detail preview and realtime badge recovery: independent
  authenticated list/detail routes, recipient-authoritative preview, selected
  theme proof, single-owner public chrome, immediate auth subscription, and
  SSE error/reconnect/visible-page REST reconciliation:
  `sessions/2026-07-28-notification-detail-preview-handoff.md`
- Architecture boundary debt M0-M12 completed: fixed tabs, domain placement,
  backend splits, focused collaborators, stable extension contracts, import
  ratchets, ADR, and resumed final gate evidence:
  `sessions/2026-07-28-architecture-debt-m12-handoff.md`
- Architecture boundary debt M10: runtime `Manager` is now a 72-method
  compatibility facade over four focused collaborators with single owners for
  admission and event/provider state; M11 stable packages are next:
  `sessions/2026-07-28-architecture-debt-m10-handoff.md`
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
- Configurable public navigation M0-M6: revisioned four-location editor,
  recovery/backup, canonical topbar/sidebar/mobile/footer SSR authority,
  dynamic taxonomy, exact theme capabilities, and selected-theme
  desktop/mobile Browser QA are complete; M7 final gate is next:
  `sessions/2026-07-28-public-navigation-platform-handoff.md`
- Notification Platform V2 completed M0-M7 with exact-artifact Web Push,
  real PostgreSQL multi-node/fixture evidence, hidden-target scrubbing, and a
  green full repository gate; use this handoff only for independent review:
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

- Tri-state color mode reliability: shared Automatic/Light/Dark authority and
  public/admin menus, resolved-only extension bridge, cache-neutral SSR, and
  safe local canonical origin completed. See
  `reports/2026-07-27-tristate-color-mode-reliability-final.md`.
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
