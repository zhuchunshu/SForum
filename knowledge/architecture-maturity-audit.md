# Architecture Maturity Audit: Modularization & Performance

Status: living audit  
Last reviewed: 2026-07-12  
Audience: humans and AI sessions deciding whether SForum has “already achieved”
the modular, performance-first architecture described in docs and decisions.

## How To Read This

This is **not** a product feature checklist. It answers:

1. Is the **modular host/plugin architecture** real, partial, or aspirational?
2. Is **performance engineering** implemented, validated, or only designed?

### Status legend

| Status | Meaning |
| --- | --- |
| **Done** | Implemented in code with a clear owner path; tests or known usage exist |
| **Partial** | Skeleton, one vertical, or core-only adapter exists; not the full target |
| **Planned** | Documented decision/roadmap; little or no production runtime |
| **Missing** | Explicitly out of scope or not started |

### Score legend (subjective, for prioritization only)

| Score | Meaning |
| --- | --- |
| 0–2 | Design only or absent |
| 3–5 | Partial / one path / unproven |
| 6–8 | Production-usable for current scale assumptions |
| 9–10 | Mature, multi-provider or scale-validated |

Scores are **not** marketing grades. They help compare gaps inside this repo.

---

## Executive Snapshot

| Dimension | Score | One-line verdict |
| --- | --- | --- |
| Modular host framework | **7.5 / 10** | Real boundaries and extension runtime; not a full plugin marketplace |
| Plugin-first verticals | **5 / 10** | Mail vertical is real; most other slots are declared or still core-owned |
| Code / contract modularity | **7 / 10** | Clear monorepo modules; several oversized domain files |
| Performance engineering | **6.5 / 10** | Targeted hardening landed; no capacity proof or horizontal scale story |
| Performance validation | **2 / 10** | No published load baseline / multi-node soak evidence |

**Overall:** architecture direction is **implemented as foundation**, not
finished as a complete modular or high-performance product platform.

Primary sources:

- `docs/architecture.md`
- `docs/roadmap.md`
- `docs/extension-platform-v2.md`
- `knowledge/decisions/2026-07-06-core-framework-plugin-first-architecture.md`
- `knowledge/decisions/2026-07-04-performance-first-jobs-queues.md`
- `knowledge/decisions/2026-07-08-search-cache-deep-pagination.md`
- `knowledge/decisions/2026-07-08-performance-hardening.md`
- `knowledge/modules/extensions.md`, `backend.md`, `forum.md`, `mail.md`

---

## Part A — Modularization Scorecard

### A1. Repository And Process Boundaries

| Item | Target (from docs) | Evidence | Status | Score |
| --- | --- | --- | --- | --- |
| Monorepo split | web / api / contracts / extensions / knowledge / tests | `apps/*`, `contracts/`, `extensions/`, `knowledge/`, `tests/` | **Done** | 9 |
| API Laravel-style layout | cmd, bootstrap, Http, Models, Providers, Support, database | `apps/api` tree matches decision | **Done** | 8 |
| Explicit bootstrap wiring | No hidden DI container; providers assemble runtime | `bootstrap/app.go`, `bootstrap/worker.go` | **Done** | 8 |
| Domain module packages | identity, forum, extensions, jobs, … | `app/Models/*`, `app/Providers/*`, `app/Jobs/*` | **Done** | 8 |
| Modular OpenAPI | index + paths/schemas per module | `contracts/openapi/{paths,schemas,components}/` + `validate-openapi-refs.rb` | **Done** | 9 |
| Knowledge-base memory | decisions, modules, sessions for future work | `knowledge/` actively maintained | **Done** | 8 |
| File size discipline | Prefer split before ~1000 lines | Several hotspots >1000 lines (see A5) | **Partial** | 4 |

### A2. Core Host Framework Contracts

| Item | Target | Evidence | Status | Score |
| --- | --- | --- | --- | --- |
| Core owns identity/RBAC/sessions | Host framework primitives | Identity module + Redis sessions + policy helpers | **Done** | 8 |
| Core owns forum primitives | categories, topics, comments, posts, moderation boundaries | Forum + moderation modules | **Done** | 8 |
| Core owns extension manager | install, enable/disable, verify, settings, events | `Models/Extensions`, admin APIs, admin UI | **Done** | 8 |
| Plugin subprocess runtime | HashiCorp go-plugin, health, route target | `Support/Extensions/protocol.go`, `manager.go` | **Done** | 8 |
| Plugin route namespace only | `/api/v1/extensions/:id/*`, no core route override | Route gateway + controller proxy | **Done** | 9 |
| Explicit events / filters | Catalog, observe + sync filter, delivery tracking | Event catalog, hook bus, deliveries, River plumbing | **Done** | 7 |
| Provider slot registry | First-class slots with selection/reset | Manifest allowlist exists; **runtime depth varies by slot** | **Partial** | 5 |
| Declarative contributions | Host-owned points, no executable descriptors | `contributions[]`, admin inspection, topic actions / jobs slots | **Partial** | 6 |
| Trusted admin frontend | Digest trust + WebRelease, not arbitrary SSR inject | Frontend trust + web release pipeline | **Partial** | 6 |
| Extension settings isolation | `extension_settings`, reset to defaults | Settings APIs + secret masking for mail | **Done** | 8 |
| Permission-aware extension ops | `extension.manage` authoritative on API | Controllers + seeds | **Done** | 8 |

### A3. Plugin-First Verticals (Product Boundary)

Architecture says vendor/deployment systems should be plugins by default.

| Vertical | Declared target | Runtime reality | Status | Score |
| --- | --- | --- | --- | --- |
| Outbound mail transport | `mail.provider` plugin | Built-in `sforum.smtp`; core owns outbox/records/selection | **Done** | 8 |
| In-app notifications | Core records + channel plugins | Core inbox/fanout/mail projection implemented; extra channels not pluginized | **Partial** | 6 |
| Attachment storage | `attachment.storage.provider` slot | Manifest slot **known**; adapters live in **core** `Support/Storage` | **Partial** | 4 |
| Search engine | `search.provider` slot | Meilisearch wired in **core**; slot allowlisted only | **Planned** | 2 |
| Human verification | `human_verification.provider` | ALTCHA in core with provider-shaped interface; not extension package | **Partial** | 4 |
| Auth risk scoring | `auth.risk.provider` | Allowlisted in manifest only | **Planned** | 1 |
| Editor sanitizer | `editor.sanitizer.provider` | Core goldmark/bluemonday path; slot allowlisted only | **Planned** | 1 |
| Notification channel (SMS/push/chat) | `notification.channel` (roadmap) | Not a full slot runtime | **Missing** | 1 |
| Payments | Core intents + `payment.provider` plugins | Not implemented | **Missing** | 0 |
| Analytics / third-party integrations | Plugins by default | Not implemented as platform | **Missing** | 0 |

Manifest known slots (validation allowlist only — **not** equal to full runtime):

```text
mail.provider
search.provider
attachment.storage.provider
human_verification.provider
auth.risk.provider
editor.sanitizer.provider
```

Source: `apps/api/app/Support/ExtensionManifest/manifest.go` (`knownProviderSlot`).

### A4. Extension Lifecycle Completeness

| Capability | Status | Notes |
| --- | --- | --- |
| ZIP upload + manifest validation | **Done** | Unsafe path rejection, size limit, digest |
| Install → list → verify | **Done** | Admin API + UI |
| Plugin enable/disable + runtime start/stop | **Done** | Failed enable rolls back disable |
| API startup reconcile enabled plugins | **Done** | `extensionRuntime.Reconcile` |
| Theme activate (uploaded) | **Done** | Queued build + health + `current.json` |
| Theme restore default (builtin) | **Done** | Synchronous path |
| Web release build / rollback / cleanup | **Partial** | Release rollback exists; plugin package upgrade/uninstall incomplete |
| Plugin package upgrade | **Missing** | Backlog in extensions module |
| Plugin uninstall + data retention policy | **Missing** | Backlog |
| Dependency / version compatibility gates | **Partial** | Manifest fields exist; full solver not productized |
| Signatures / trusted marketplace | **Missing** | Explicit future work |
| Plugin migrations runner | **Missing / Planned** | Manifest can declare migrations; full execution story incomplete |
| Dev scaffolding (`make:plugin` / `make:theme`) | **Done** | `cmd/sforum` |

### A5. Code-Level Modularity Health

| Check | Observation | Status |
| --- | --- | --- |
| Module folders by domain | Strong: Models/Controllers/Providers/Jobs aligned | **Done** |
| Shared Support packages | Cache, Jobs, Search, Extensions, Storage, etc. | **Done** |
| sqlc adoption | Present for some identity paths; not universal | **Partial** |
| Oversized handwritten files | `Forum/postgres_store.go` ~2.0k, `Options/service.go` ~1.7k, `Forum/service.go` ~1.3k, `Extensions/service.go` ~1.1k | **Partial** |
| Built-in extension packages | SMTP plugin + default theme (+ dev themes) | **Partial** |
| Frontend modular admin registry | `adminModules.ts` low-code shell | **Done** |
| Frontend theme layering | Builtin default + uploaded Nuxt Layer pipeline | **Done** |

### A6. Modularization Score Summary

| Area | Score |
| --- | --- |
| Repo / process modularity | 8 |
| Host contracts & runtime | 7.5 |
| Plugin-first vertical coverage | 5 |
| Lifecycle completeness | 5.5 |
| Code hygiene (file/sqlc boundaries) | 5.5 |
| **Weighted modularization** | **~7 / 10** |

**Verdict:** Modularization is **real and enforced for the host/extension
boundary**, especially routes, events, and mail. It is **not complete** as a
multi-vendor, multi-slot, marketplace-ready plugin platform.

---

## Part B — Performance Checklist (Code vs Claim)

### B1. Data Plane: Reads

| Check | Target | Implementation | Status | Score |
| --- | --- | --- | --- | --- |
| Full-text search off PG `ILIKE` | Meilisearch path | `Support/Search`, `/search`, index jobs | **Done** | 8 |
| Topic list query separation | Keyword search not via ListTopics scan | `ErrUseSearchEndpoint` when query set | **Done** | 8 |
| Public read cache | Redis decorator on hot forum reads | `forum.CachedStore` + generation invalidation | **Done** | 7 |
| Deep OFFSET protection | Clamp page | `maxTopicPage` (~200) + OpenAPI max | **Done** | 8 |
| Comment list memory safety | SQL pagination, not load-all | flat LIMIT/OFFSET; tree roots + children batch | **Done** | 7 |
| Comment tree mega-thread guard | Bound descendants per root | Still can load all children of a hot root | **Partial** | 4 |
| Topic detail HTTP cache | Safe caching | `/t/**` deliberately `cache: false` | **Partial** | 4 |
| Homepage / taxonomy SWR | routeRules swr | `/`, `/c/**`, `/tags/**`, `/u/**` configured | **Done** | 7 |
| Query homepage no-store | Avoid payload key collisions | home query middleware disables cache/swr | **Done** | 7 |
| View count write path | Cheap increment | Column exists; **no Redis counter + batch flush** called out as gap | **Missing** | 2 |

### B2. Data Plane: Writes And Side Effects

| Check | Target | Implementation | Status | Score |
| --- | --- | --- | --- | --- |
| Durable async jobs | River + PG | `Support/Jobs`, worker process, enqueue API | **Done** | 8 |
| Search index async | Post-write enqueue | `search.index_topic` / delete jobs | **Done** | 8 |
| Mail delivery async | River + durable deliveries | `Notifications/deliver_mail` jobs | **Done** | 8 |
| Transactional enqueue | Same TX as domain write where needed | Mail/notification design uses River TX insert pattern | **Partial** | 6 |
| Job payload discipline | ID-based small payloads | Generally followed in job args | **Done** | 7 |
| Idempotent handlers | Safe retry | Required by design; coverage varies by job | **Partial** | 6 |
| Orphan / session cleanup jobs | Maintenance queues | Attachment orphan + session cleanup jobs exist | **Done** | 7 |

### B3. Network, Process, And Connection Layers

| Check | Target | Implementation | Status | Score |
| --- | --- | --- | --- | --- |
| Fiber timeouts | Read/Write/Idle | Config + `server.go` | **Done** | 8 |
| Body limit | Bound request size | `HTTPBodyLimit` default 4MB | **Done** | 8 |
| Response compression | brotli/gzip | Fiber compress middleware | **Done** | 8 |
| Write rate limit | IP-based on mutating methods | Fiber limiter + Redis storage | **Done** | 7 |
| Redis pool hygiene | Shared client + pool knobs | humanverify+cache merged; session storage separate | **Done** | 7 |
| PG pool hygiene | MinConns, idle, lifetime, connect timeout | `postgres.NewPoolWithOptions` for API/worker | **Done** | 8 |
| Meili client timeout | Avoid hung SSR/API | `NewClientWithTimeout` | **Done** | 8 |
| Multi-instance limiter | Shared Redis | When Redis storage present | **Done** | 7 |

### B4. Frontend Performance

| Check | Target | Implementation | Status | Score |
| --- | --- | --- | --- | --- |
| SSR-first public HTML | SEO / first paint | Nuxt SSR; admin no longer `ssr: false` shell | **Done** | 8 |
| Public hybrid caching | SWR/ISR-like | nitro `routeRules` swr/cache | **Done** | 7 |
| Asset compression | brotli+gzip public assets | `compressPublicAssets` | **Done** | 8 |
| Long-cache hashed assets | immutable cache headers | SEO middleware static branch | **Done** | 8 |
| Heavy component lazy load | Editor / icon picker | `LazySFEditor` / `LazySFIconPicker` | **Done** | 7 |
| Image optimization | Nuxt Image / webp | `@nuxt/image` + `SFAvatar` NuxtImg | **Done** | 7 |
| Client data cache reuse | getCachedData / payload reuse | Explicit follow-up in performance decision | **Missing** | 2 |
| Infinite scroll vs deep page | Reduce page churn | Homepage infinite scroll implemented | **Done** | 7 |

### B5. Deploy / Runtime Elasticity

| Check | Target | Implementation | Status | Score |
| --- | --- | --- | --- | --- |
| Separate API vs worker | Process split | `cmd/api`, `cmd/worker`; prod default split | **Done** | 8 |
| Dev embed worker option | Faster local loop | `EMBED_WORKER_IN_API` | **Done** | 7 |
| Theme switch zero-downtime | Blue-green Nitro | Production supervisor + health + atomic proxy swap | **Done** | 8 |
| Multi-node theme rollout | Shared release coordination across nodes | Still largely single-node / shared volume assumptions | **Partial** | 3 |
| Read replicas / write splitting | Scale reads | Explicitly deferred | **Missing** | 0 |
| Horizontal API scale story | Stateless API + shared Redis/PG | Sessions in Redis help; no full runbook/proof | **Partial** | 4 |
| Plugin runtime cost model | Isolation vs throughput | Subprocess RPC prioritizes isolation | **Partial** | 5 |

### B6. Performance Validation And Operability

| Check | Target | Implementation | Status | Score |
| --- | --- | --- | --- | --- |
| Load test suite / capacity numbers | Documented QPS/p99 | Not present as project baseline | **Missing** | 1 |
| Chaos / Meili/Redis failure modes | Graceful degrade | Search 503 path designed; not fully soak-tested here | **Partial** | 4 |
| Admin runtime observability | Memory, pool, trends | Admin overview command center aggregates runtime + pool stats | **Done** | 7 |
| Job operator workbench | Inspect/retry/pause | Admin jobs UI + permissions | **Done** | 7 |
| Published SLOs | Latency/error budgets | Not defined as product SLOs | **Missing** | 1 |

### B7. Performance Score Summary

| Area | Score |
| --- | --- |
| Read path engineering | 7 |
| Async write side effects | 7 |
| Network / pools / limits | 8 |
| Frontend caching & assets | 7 |
| Scale-out architecture | 3 |
| Measurement & proof | 2 |
| **Weighted performance** | **~6 / 10** |

**Verdict:** SForum has a **performance-aware foundation** with concrete
hardening. It does **not** yet justify marketing language like “high-performance
at scale” without load evidence and horizontal design.

---

## Part C — Cross-Cutting Matrix (Claim → Reality)

| Public / docs claim style | Safer accurate claim | Reality |
| --- | --- | --- |
| “Highly modular plugin platform” | “Host framework with controlled extension runtime” | True for contracts; incomplete vertical ecosystem |
| “Upload plugin and it just works” | “Upload installs; enable starts backend; frontend/theme may queue builds” | Correct operator model |
| “Plugin-first architecture” | “Plugin-first **policy** + first verticals” | Mail is the proof slice |
| “Performance-first stack” | “Performance-hardened defaults for single-node / modest multi-process deploy” | Accurate |
| “Zero-downtime themes” | “Production theme supervisor blue-green on one web host” | Done for that scope |
| “Meilisearch-powered search” | “Primary keyword search path is Meilisearch” | Done |
| “Redis-cached forum reads” | “Selected public forum reads cached with generation TTL” | Done |
| “Scale to millions of rows” | “Risks addressed by design; not capacity-certified” | Design yes, proof no |

---

## Part D — Highest-Value Gaps (Ordered)

Use this list when choosing the next architecture investment.

### Modularization gaps

1. **Promote remaining provider slots beyond allowlist**  
   Especially `attachment.storage.provider` and `search.provider` if those should
   truly be swappable without core forks.
2. **Finish extension lifecycle**  
   Upgrade, uninstall, migration runner, retention, compatibility checks.
3. **Split oversized domain files**  
   Forum store/service, options service, extensions service.
4. **Payment framework only when product needs it**  
   Core intents first, providers as plugins — do not hard-code gateways in core.
5. **More built-in example plugins**  
   SDK docs + one more vertical reduce “framework without ecosystem” risk.

### Performance gaps

1. **Establish a capacity baseline**  
   Even a small k6/vegeta suite on homepage, topic list, topic detail, search,
   and write paths would turn “hardening” into evidence.
2. **View count path**  
   Redis increment + batch flush (called out in decisions as missing).
3. **Hot comment-thread descendant bounds**  
   Prevent pathological tree loads under a single root.
4. **Frontend data-layer reuse**  
   `getCachedData` / payload reuse follow-up from performance decision.
5. **Horizontal scale design**  
   Only when multi-node is a real operator requirement: sticky-free sessions
   already lean Redis; still need theme release, web-release, and worker
   coordination runbooks.

---

## Part E — Suggested Re-Audit Cadence

Re-run this audit when any of the following lands:

- A new provider slot becomes end-to-end (not just allowlisted)
- Extension upgrade/uninstall ships
- Load test baseline is checked into `tests/` or `docs/`
- Horizontal deployment of web/API/worker is supported in `deploy/`

Update:

1. Status cells in Parts A–B  
2. Executive snapshot scores  
3. `knowledge/index.md` link text if the overall verdict changes  
4. A short `knowledge/sessions/YYYY-MM-DD-*.md` handoff if scores move materially

---

## Part F — One-Page Cheat Sheet

```text
Modularization .............. ~7/10   host+runtime real; verticals incomplete
Plugin-first policy ......... real    mail is the reference implementation
Provider slots .............. partial allowlist > runtime depth
Extension lifecycle ......... mid+    install/enable/theme + upgrade/uninstall/ledger; SQL runner later
Performance engineering ..... ~6/10   search/cache/pools/jobs/limits done
Performance proof ........... low     no published capacity baseline
Scale-out ................... early   single-node assumptions still dominate
Safe public wording ......... "modular host framework with performance-aware defaults"
Unsafe public wording ....... "fully modular high-performance plugin marketplace"
```
