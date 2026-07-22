# Knowledge Index

Project memory entry point for humans and AI sessions. Prefer this file, the
latest hot handoff, and the relevant module note—do not re-read the full
session archive.

## Latest Handoff

- **2026-07-22 Site search CJK + fuzzy matching + index repair**
  - Default PostgreSQL engine: `simple` FTS + Unicode Han n-grams + indexed
    `pg_trgm` title/excerpt typo tolerance + weighted author/category/tag/slug
    metadata; relevance-first stable ranking
  - Fixed River completed-job suppression and false-success worker behavior;
    full rebuild now always schedules every public topic
  - Docker DB migrated and genuinely rebuilt: 57/57 public topics indexed;
    `小明` returns topic 60 in API and browser
  - Decision: `decisions/2026-07-22-default-site-search-cjk-fuzzy.md`
  - Handoff: `sessions/2026-07-22-site-search-cjk-fuzzy-handoff.md`
  - Module: `modules/search.md`

- **2026-07-22 Theme-defined system error pages task book**
  - Ready M0-M6 plan: selected-theme L0/L1 for 403/404/429/5xx, narrow Host
    error islands, no L2/plugin replacement, and non-recursive emergency fallback
  - Wait for or coordinate with overlapping current-HEAD regression work before
    editing shared Page Registry/error files
  - Plan: `plans/2026-07-22-theme-defined-system-error-pages.md`
  - Handoff: `sessions/2026-07-22-theme-defined-system-error-pages-plan-handoff.md`
  - Modules: `modules/frontend.md` · `modules/extensions.md`

- **2026-07-22 Forum content revisions V1 task book**
  - Ready M0–M7 plan: admin edit-any workbench, immutable self/staff history,
    optimistic concurrency, diff, safe restore, and super-admin redaction
  - Plan: `plans/2026-07-22-forum-content-revisions-v1.md`
  - Handoff: `sessions/2026-07-22-forum-content-revisions-v1-plan-handoff.md`
  - Module: `modules/forum.md`

- **2026-07-22 Current HEAD regression remediation plan**
  - Ready task book for invalid PG FTS config, frontend typecheck, advanced
    reply Page Registry closure, stable search pagination/single hydration, and
    Extensions Controller gate stability
  - Plan: `plans/2026-07-22-current-head-regression-remediation.md`
  - Handoff: `sessions/2026-07-22-current-head-regression-plan-handoff.md`
  - Modules: `modules/search.md` · `modules/frontend.md`

- **2026-07-22 Advanced reply page**
  - Comment input top-right「高级回复」→ `/topics/reply` full SFEditor
  - Page Registry `forum.topic.reply`; draft handoff via sessionStorage
  - Handoff: `sessions/2026-07-22-advanced-reply-page-handoff.md`
  - Module: `modules/frontend.md`

- **2026-07-22 Topic reply always expanded**
  - Content-page comment input always open; no collapse/expand toggle
  - Handoff: `sessions/2026-07-22-topic-reply-always-expanded-handoff.md`
  - Module: `modules/frontend.md`

- **2026-07-22 Hybrid topic typography 100% fidelity**
  - Demo 实测对齐：标题/正文/h2/pre/byline/按钮/搜索/评论标题；分类圆点；
    顶栏 active 主色无下划线；本地 accent `custom:#d94763`
  - Handoff: `sessions/2026-07-22-hybrid-topic-typography-fidelity-handoff.md`
  - Module: `modules/frontend.md`

- **2026-07-22 Content policy badge settings instant effect (P0+P1)**
  - `site.public_surface_revision` bump on extension settings that affect
    public contributions; Nuxt `/t/**` SWR varies revision header
  - Admin Toast: refresh topic page, no theme reactivation
  - Handoff: `sessions/2026-07-22-content-policy-badge-settings-instant-handoff.md`
  - Module: `modules/extensions.md`

- **2026-07-22 Sticky theme resolve resilience**
  - Removed unsafe last-good L1 shell reuse; resolve retries once and binds
    async data to page/path/query/locale/actor
  - Fail-closed SSR is `no-store` with Nitro SWR disabled; skin/settings
    last-good caches require exact theme identity + short TTL
  - Handoff: `sessions/2026-07-22-sticky-theme-resolve-resilience-handoff.md`
  - Module: `modules/frontend.md`

- **2026-07-22 i18n no_prefix seamless locale**
  - Strategy `no_prefix`: no `/en` routes; `setLocale` + cookie only
  - Legacy `/en/*` 301 strip; non-default locale bypasses shared SWR
  - Handoff: `sessions/2026-07-22-i18n-no-prefix-seamless-handoff.md`
  - Decision: `decisions/2026-07-22-i18n-no-prefix-seamless.md`
  - Module: `modules/localization.md`

- **2026-07-22 Locale switch setLocale fix**
  - Language menu uses `setLocale` (cookie + path); fixes en→zh stuck on `/`
    and false “zh active” under `prefix_except_default`
  - Handoff: `sessions/2026-07-22-locale-switch-setlocale-fix.md`
  - Module: `modules/frontend.md`

- **2026-07-22 Default theme appearance + dark mode**
  - Default theme L0 no longer hard-locks rose accent; follows site
    `appearance.theme` (`--sf-accent*`) and `.dark` public surface tokens
  - Handoff: `sessions/2026-07-22-default-theme-appearance-dark-mode-handoff.md`
  - Module: `modules/frontend.md`

- **2026-07-22 Topbar session align + locale/theme**
  - User avatar/name column width matches right rail; language + day/night
    toggles restored on public topbar
  - Handoff: `sessions/2026-07-22-topbar-session-align-locale-theme-handoff.md`
  - Module: `modules/frontend.md`

- **2026-07-22 Topic detail navigation experience**
  - Topic content no longer waits for client-side comment/category completion
  - Core topic island is eager; global progress feedback and click-to-load reply editor added
  - SSR still contains complete topic, comments, and navigation data
  - Handoff: `sessions/2026-07-22-topic-detail-navigation-experience.md`
  - Module: `modules/frontend.md`

- **2026-07-22 Comment floor display**
  - Topic detail comment floor badges now show list positions (`#1`, `#2`, …)
    instead of database comment IDs
  - Anchors still target stable `#comment-<id>` deep links
  - Handoff: `sessions/2026-07-22-comment-floor-display-handoff.md`
  - Module: `modules/frontend.md`

- **2026-07-22 Hybrid demo fidelity polish (default theme UI)**
  - Fonts/weights, topbar active underline, topic list 最近活动, left-nav colored category icons
  - Demo: `tmp/demos/sforum-hybrid-topic-list/`
  - Handoff: `sessions/2026-07-22-hybrid-demo-fidelity-polish-handoff.md`
  - Module: `modules/frontend.md`

- **2026-07-22 extension.hook_failed on post/comment**
  - Host Protocol V2 now JSON-normalizes hook payloads before `structpb`
  - Content-policy reads content object `plainText`/`rawContent`
  - Handoff: `sessions/2026-07-22-hook-payload-structpb-encode-fix.md`
  - Module: `modules/extensions.md`

- **2026-07-22 Web dev startup and tag SSR stability**
  - Tag detail no longer rejects SSR during AsyncData/HMR transitions
  - Page Host islands load lazily; DevTools is opt-in; dev payload extraction is off
  - Guest middleware uses its incoming `to` route without `useRoute()` warnings
  - Handoff: `sessions/2026-07-22-web-dev-startup-tag-ssr-handoff.md`
  - Module: `modules/frontend.md`

- **2026-07-22 V3 production rewire honesty remediation (plan ready)**
  - Acceptance review: 5 findings still open, 3 partial; Support green ≠ production
  - New task book (M0–M8):  
    `plans/2026-07-22-v3-production-rewire-honesty-remediation.md` (**ready**)
  - Do **not** claim rewire closed or V3 100%; LTS residual still open
  - Prior rewire handoff retained as evidence of partial work:  
    `sessions/2026-07-22-p11-p12-p13-production-rewire-handoff.md`
  - Module: `modules/extensions.md`

- **2026-07-22 P11/P12/P13 production rewire (partial — honesty reopened)**
  - SettingsLifecycle bind + dual-PG CAS; RuntimeRollout CAS; CompatFarm process;
    formal ZIP; commerce add-via-Dispatcher — **not** full ops honesty
  - Open: `enc::` migrate, real multi-node rollout, SystemTier order, Marketplace/
    Privacy consumers, deploy key, farm/commerce residual
  - Handoff: `sessions/2026-07-22-p11-p12-p13-production-rewire-handoff.md`
  - Module: `modules/extensions.md`

- **2026-07-22 Public session first-render stability**
  - Active theme CSS links now ship in SSR HTML
  - Session-bearing pages use `no-store` SSR auth; anonymous SWR remains
  - Handoff: `sessions/2026-07-22-public-session-first-render-handoff.md`
  - Module: `modules/frontend.md`

- **2026-07-22 P13 reference-plugin honesty**
  - Commerce/Custom-content/Media e2e execute declared surfaces (not Manifest-only)
  - Formal ZIP + honesty gates; superseded for rewire detail by production-rewire handoff
  - Handoff: `sessions/2026-07-22-p13-reference-plugin-honesty-handoff.md`
  - LTS residual: `sessions/2026-07-21-trusted-plugin-theme-platform-v3-p13-lts-residual-handoff.md`
  - Module: `modules/extensions.md`

- **2026-07-22 Social login provider task book**
  - Ready plan: `plans/2026-07-22-social-login-provider-plugins.md`
  - Core-owned accounts/sessions + plugin-owned GitHub/Google/Discord/Telegram
  - Handoff: `sessions/2026-07-22-social-login-provider-plan-handoff.md`
  - Module: `modules/identity.md`

- **2026-07-21 Default theme full-width 3-col flat shell**
  - Demo: `tmp/demos/grok/forum-fullwidth-3col/`
  - Theme: `extensions/builtin/themes/sforum-default/` (tokens + L1 home/topic)
  - Host: `SFTopicShowPage` left nav; home/topic CSS baselines
  - Handoff: `sessions/2026-07-21-default-theme-fullwidth-3col-handoff.md`
  - Module: `modules/frontend.md`

- **2026-07-21 Million-scale read path — M7 complete (task book closed)**
  - Plan: `plans/2026-07-21-million-scale-read-path.md` (**completed**, M0–M7)
  - Decision: `decisions/2026-07-21-read-replica-and-api-horizontal-scale.md`
  - Report: `reports/2026-07-21-perf-m7-horizontal-scale.md` (doc only; no code)
  - Multi-API + shared Redis OK; read replica deferred to metrics thresholds
  - Handoff: `sessions/2026-07-21-million-scale-read-path-m7-handoff.md`
  - Module: `modules/forum.md`

- **2026-07-21 Theme staged startup repair (API boot)**
  - Active builtin theme stuck on old digest with retired `sf-my-home-page`
  - Startup promotes healthy staged package + new theme runtime publication
  - Handoff: `sessions/2026-07-21-theme-staged-startup-repair-handoff.md`

- **2026-07-21 Million-scale read path — M6 complete**
  - After: `reports/2026-07-21-perf-m6-cache-sharding.md` (scoped gen; multi-cat warm)
  - Handoff: `sessions/2026-07-21-million-scale-read-path-m6-handoff.md`

- **2026-07-21 Million-scale read path — M5 complete**
  - After: `reports/2026-07-21-perf-m5-keyset.md` (100-step cursor p99 ~19 ms)
  - Handoff: `sessions/2026-07-21-million-scale-read-path-m5-handoff.md`

- **2026-07-21 Million-scale read path — M4 complete**
  - After: `reports/2026-07-21-perf-m4-topic-detail.md` (by-slug warm p99 ~21 ms)
  - Handoff: `sessions/2026-07-21-million-scale-read-path-m4-handoff.md`

- **2026-07-21 Million-scale read path — M3 complete**
  - After: `reports/2026-07-21-perf-m3-list-comments.md` (tree cap 50; warm p50 ~44 ms)
  - Handoff: `sessions/2026-07-21-million-scale-read-path-m3-handoff.md`

- **2026-07-21 Million-scale read path — M2 complete**
  - After: `reports/2026-07-21-perf-m2-view-hot.md` (view flood 0 per-req UPDATE; hot index)
  - Handoff: `sessions/2026-07-21-million-scale-read-path-m2-handoff.md`

- **2026-07-21 Remove `/my` pages**
  - Handoff: `sessions/2026-07-21-remove-my-pages-handoff.md`
  - Deleted `/my` + `/my/content-review` routes/entry points; API review list kept
  - Module: `modules/moderation.md` · `modules/profile.md`

- **2026-07-21 Million-scale read path — M1 complete**
  - After: `reports/2026-07-21-perf-m1-list-topics.md` (home cold ~11.5×, warm p99 ~29 ms)
  - Handoff: `sessions/2026-07-21-million-scale-read-path-m1-handoff.md`

- **2026-07-21 Forum settings Search provider tab**
  - Handoff: `sessions/2026-07-21-forum-settings-search-provider-tab-handoff.md`
  - `/admin/forum/settings?tab=search` + search.provider list/select/reset APIs
  - Module: `modules/search.md`

- **2026-07-21 Plugin process RSS on admin lists**
  - Handoff: `sessions/2026-07-21-plugin-process-rss-admin-list-handoff.md`
  - `runtime.memoryBytes` on extension List/Detail; Plugins list badge
  - Shared: `apps/api/app/Support/ProcessMemory`

- **2026-07-21 Million-scale read path — M0 complete**
  - Plan: `plans/2026-07-21-million-scale-read-path.md`
  - Baseline: `reports/2026-07-21-perf-baseline.md`
  - Handoff: `sessions/2026-07-21-million-scale-read-path-m0-handoff.md`

- **2026-07-21 Bilingual docs handbook**
  - Hub: `docs/README.md`
  - 中文: `docs/zh-CN/` · English: `docs/en-US/`
  - Usage + development + deployment + product/architecture
  - Technical contracts remain path-stable under `docs/extensions/`
  - Legacy root drafts / superpowers plans → `docs/archive/`
  - Handoff: `sessions/2026-07-21-docs-bilingual-handbook-handoff.md`

- **2026-07-21 Knowledge base cleanup**
  - Handoff: `sessions/2026-07-21-knowledge-base-cleanup-handoff.md`
  - Slimmed this index; archived historical sessions; unified plan statuses

- **2026-07-21 Search framework + protected site-search default**
  - Decision: `decisions/2026-07-21-search-framework-site-default.md`
    (supersedes “default no engine → 503” in the optional-Meili decision)
  - Module: `modules/search.md`
  - Handoff: `sessions/2026-07-21-search-framework-site-default-handoff.md`
  - Builtin: `extensions/builtin/plugins/sforum-search-site` (`sforum.search-site`)
  - Optional: `extensions/optional/plugins/sforum-search-meilisearch`
  - Host: `PostgresSiteEngine` + `search_documents`; Meili Compose profile `search`

- **2026-07-21 Trusted Plugin / Theme Platform V3 — P13 LTS residual (~99.7%)**
  - Decision: `decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`
  - Task book: `plans/2026-07-13-trusted-plugin-theme-platform-v3.md`
  - **Durable progress (authoritative %):**  
    `plans/2026-07-13-trusted-plugin-theme-platform-v3-progress.md`
  - Hot handoff:  
    `sessions/2026-07-21-trusted-plugin-theme-platform-v3-p13-lts-residual-handoff.md`
  - Intermediate V3 checkpoints: `sessions/archive/2026-07/`
  - Status: P0–P12 complete. P13 implementable work closed. Open only
    LTS-blocked deletions (request-time loader residual, Protocol V1 paths,
    compatibility removal) after RemoveAfter ≈ **2026-11-28** + zero-shim.
  - Evidence: `docs/extensions/v3/p13-final-gates-evidence.md`
  - Inspect: `cd apps/api && go run ./cmd/sforum extension api-lts`

- **2026-07-21 Extension directory docs**
  - Handoff: `sessions/2026-07-21-extension-directory-docs-handoff.md`

## Current Project State

Short status only. Module detail lives under `modules/`; historical narrative
lives in archived sessions and dated decisions.

### Stack

- **Web:** Nuxt 4 / Vue 3 / Nuxt UI 4 / Bun; SSR-first; i18n `zh-CN` (default) + `en-US`
- **API:** Go Fiber v3, PostgreSQL, Redis, River queue, Goose migrations, sqlc
- **Search default:** protected site PG FTS (`sforum.search-site`); Meilisearch optional plugin + Compose profile `search`
- **Dev:** Compose for PG/Redis/Mailpit; host `bun run dev` + `./scripts/api-dev.sh` (embedded worker)

### Shipped (core product)

- Identity: registration/login, Redis sessions, RBAC, permission overrides, first-user `super_admin`, account sessions
- Forum: categories/tags, topics/comments tree, policy enforcement, admin taxonomy & multi-tab forum settings
- Attachments: provider slot, local + plugin storage path, governance, orphan cleanup
- Mail + in-app notifications: core durable records + builtin SMTP plugin
- Moderation: reports, pre-publication review, workbench
- Options: runtime site/personalization/SEO/chrome; beginner defaults + restore
- Extensions: Manifest V3, trust, lifecycle, Page Registry themes (L0/L1), buildless settings UI, Host API v2 / registries (V3 P0–P12)
- Admin: multitabs, module registry, jobs/schedules, security ops surfaces

### Open / next (product, not V3 LTS)

- **Forum content revisions V1 (ready):** admin edit-any workbench, immutable
  self/staff versions, CAS conflict prevention, diff, restore, and redaction —
  `plans/2026-07-22-forum-content-revisions-v1.md`
- **Current HEAD regression remediation (ready):** search/frontend/Page
  Registry/gate M0–M7 —
  `plans/2026-07-22-current-head-regression-remediation.md`
- **V3 production rewire honesty remediation (ready):** eight call-chain findings
  M0–M8 — `plans/2026-07-22-v3-production-rewire-honesty-remediation.md`
- **Social login provider plugins ready:** Core completion + unified admin/user
  surfaces + GitHub/Google/Discord/Telegram —
  `plans/2026-07-22-social-login-provider-plugins.md`
- **Iteration A** engagement loop still open: view-count increment, likes/reactions, bookmarks — `plans/2026-07-12-iteration-a-engagement-loop.md`
- **Million-scale read path** task book (**M0–M5 done**, M6 next): cache sharding — `plans/2026-07-21-million-scale-read-path.md`
- Admin settings richness: Waves 1–2 landed; later waves remain blueprint — `plans/2026-07-12-admin-settings-richness.md`
- Extension surface density: E1–E6 largely landed; product north-star slots continue under V3 provider model — `plans/2026-07-12-extension-surface-density.md`
- V3 P13: **do not** delete LTS-gated shims early — wait for APILTS window

### Explicitly deferred / not current focus

- Horizontal scale / multi-node / read replicas (after M0–M6 single-node proof in million-scale plan)
- Payments, marketplace
- Full re-score of `architecture-maturity-audit.md` (stamped pre-V3 completion)

## Navigation

| Path | Role |
| --- | --- |
| `decisions/` | Architecture and product ADRs (keep; mark superseded in-file) |
| `modules/` | Living notes per feature area — start here for implementation context |
| `sessions/` | **Hot** handoffs only (recent / still actionable) |
| `sessions/archive/` | Historical handoffs by month; do not treat as current status |
| `plans/` | Task books and progress ledgers; see `plans/README.md` for status |
| `glossary.md` | Shared domain and framework terms |
| `research.md` | Early library comparisons (historical; decisions supersede) |
| `architecture-maturity-audit.md` | Scorecard as of **2026-07-12** (pre full V3 close); re-audit before citing scores |
| `legacy-sforum-feature-gap.md` | Gap vs old PHP SForum; partially stale — verify before migration work |
| `reports/` | Point-in-time scans (e.g. security) |

### Active plans (read first)

- `plans/2026-07-13-trusted-plugin-theme-platform-v3.md` + `-progress.md` — V3 residual
- `plans/2026-07-22-forum-content-revisions-v1.md` — ready content editing/revision/restore V1 task book
- `plans/2026-07-22-social-login-provider-plugins.md` — ready social login implementation task book
- `plans/2026-07-21-million-scale-read-path.md` — single-node 1M-class read path (M0 done)
- `plans/2026-07-12-iteration-a-engagement-loop.md` — product engagement (WS1 view count done with M2; likes/bookmarks open)
- `plans/2026-07-12-admin-settings-richness.md` — settings blueprint
- `plans/2026-07-12-development-directions.md` — strategy context
- `plans/2026-07-12-extension-surface-density.md` — remaining density / slots

### Completed or superseded plans (archive reference)

See status table in `plans/README.md`. Do not reopen completed security batches
or superseded theme-switch / web-release plans.

### Key decisions (non-exhaustive)

- Core plugin-first: `decisions/2026-07-06-core-framework-plugin-first-architecture.md`
- V3 platform: `decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`
- Page Registry themes: `decisions/2026-07-13-runtime-page-registry-themes.md`
- Search site default: `decisions/2026-07-21-search-framework-site-default.md`
- Host platform capabilities: `decisions/2026-07-12-host-platform-capabilities.md`
- Modular OpenAPI: `decisions/2026-07-05-openapi-contract-modularization.md`

Full ADR list: `ls decisions/`. Superseded ADRs state the replacement in their header.

## How To Use This In A New Session

1. Read root `AGENTS.md` (or `Agents.md`).
2. Read **this file** (Latest Handoff + Current Project State only).
3. Open the **hot** handoff under `sessions/` that matches the task.
4. Open the relevant `modules/<area>.md` and any **active** plan.
5. For V3 percentage or residual rows, trust `plans/*-v3-progress.md`, not old session prose.
6. Before stopping: update the module note and write a short hot handoff; archive is for cold history only.

## Open Questions

Still genuinely open (do not re-litigate settled stack choices):

- Production backup destination and retention policy for operator deployments
- Whether English (`en-US`) copy must be complete for first public release or may lag
- Category-scoped ACL timing relative to global RBAC (optional product track)
- When to schedule the next full architecture maturity re-audit post-V3 LTS cleanup

Settled (do not reopen without a new ADR):

- Default locale `zh-CN`; stack Nuxt 4 + Go Fiber + PG + Redis
- Search ships with protected site PG FTS; Meilisearch is optional
- Extension platform direction is V3 (Manifest, trust, registries, Page Registry)
- Runtime frontend builds / Web Release path removed before first release
