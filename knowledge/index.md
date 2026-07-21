# Knowledge Index

Project memory entry point for humans and AI sessions. Prefer this file, the
latest hot handoff, and the relevant module note—do not re-read the full
session archive.

## Latest Handoff

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

- **Iteration A** engagement loop still open: view-count increment, likes/reactions, bookmarks — `plans/2026-07-12-iteration-a-engagement-loop.md`
- Admin settings richness: Waves 1–2 landed; later waves remain blueprint — `plans/2026-07-12-admin-settings-richness.md`
- Extension surface density: E1–E6 largely landed; product north-star slots continue under V3 provider model — `plans/2026-07-12-extension-surface-density.md`
- V3 P13: **do not** delete LTS-gated shims early — wait for APILTS window

### Explicitly deferred / not current focus

- Horizontal scale / multi-node capacity proof
- Payments, marketplace, OAuth social login (unless product prioritizes)
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
- `plans/2026-07-12-iteration-a-engagement-loop.md` — product engagement
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
