# Million-Scale Read Path — Task Book

Status: **in progress** — M0–**M5 complete** (keyset pagination; after report);
next M6  
Date: 2026-07-21  
Last decision pass: 2026-07-21 (four open questions → resolved defaults)  
Goal: make public forum **read paths** safe for ~1M topics / large hot
threads with second-class list/detail latency on a single-node PG + Redis
deployment, with reproducible load evidence.

**Not a marketing claim.** Exit is measured p99/QPS under seeded data, not
“Xiuno-level” branding.

## Related Plans And Decisions

| Doc | Relationship |
| --- | --- |
| `plans/2026-07-12-iteration-a-engagement-loop.md` | **Owns product view-count** (INCR + flush + dedup). This plan **requires** that workstream and adds load acceptance + `hot_score` coupling. Do not dual-implement view count. |
| `decisions/2026-07-08-search-cache-deep-pagination.md` | Existing cache / page clamp / search split — extend, do not rip out. |
| `decisions/2026-07-08-performance-hardening.md` | Comment SQL pagination, pools, SWR — baseline. |
| `architecture-maturity-audit.md` Part D | Highest-value performance gaps; this plan operationalizes them. |
| `plans/2026-07-12-development-directions.md` | Strategy: capacity proof after daily community paths; this is the capacity track. |

## Product Goal

Under a seeded dataset of approximately:

- **1,000,000** public topics (across many categories)
- **≥1** hot topic with **≥50,000** comments
- Redis available; search via site FTS or Meili (both must not regress)

Public operators should observe:

1. Homepage / category **first page** list p99 within an agreed budget (see M0)
2. Topic detail **first screen** (topic body + first comment page) p99 within budget
3. Unique topic views **do not** write PG on every request
4. Tree view of a hot thread **never** loads unbounded descendants
5. A checked-in or documented **k6 (or vegeta) suite** that can re-run the proof

## Current Baseline (do not rebuild)

| Area | Status | Evidence |
| --- | --- | --- |
| Redis `CachedStore` on ListTopics / GetTopic / taxonomy | **Done** | `forum/cached_store.go`, gen invalidation |
| `maxTopicPage` ≈ 200 clamp | **Done** | `normalizePage` / OpenAPI max |
| Keyword list vs search split (service-level) | **Done (M1)** | Service `ErrUseSearchEndpoint`; store list has **no ILIKE** |
| Comment flat SQL LIMIT/OFFSET | **Done** | `listCommentsFlat` |
| Comment tree roots page + all descendants | **Done (M3 / D2)** | cap default 50 + `hasMoreChildren` |
| `topics.view_count` column + display | **Done** | schema + UI |
| View count increment / Redis flush | **Done (M2 / D3)** | Iteration A WS1 + flush job |
| Keyset / cursor public pagination | **Done (M5)** | `after` + hasMore/nextCursor |
| Approximate / denormalized list totals | **Done (M1 / D1)** | cat/tag `topic_count`; home sum + `totalApproximate` |
| `hot_score` / popular precompute | **Done (M2)** | column + indexes; list hot sort |
| ListComments in CachedStore | **Done (M3)** | topic gen + short TTL; skip viewer-scoped |
| Load-test suite / capacity numbers | **Missing** | audit B6 |
| Read replicas / multi-node | **Out of scope** | later plan if needed |

## Out Of Scope

- Horizontal scale runbooks, sticky sessions across web nodes, theme multi-node
- Read replicas / write splitting
- Shard / partition topics tables
- Payments, OAuth, likes, bookmarks (likes/bookmarks stay on Iteration A)
- Rewriting the extension platform or plugin RPC for throughput
- Claiming “million rows second open” in public docs without M0 numbers
- Replacing Nuxt SSR with a pure static PHP-style stack

## Success Metrics (agree before coding M1+)

Record chosen numbers in the first M0 PR description; defaults below are
**starting recommendations**, not sacred.

| Scenario | Seed | Default target (API, cache warm, single API process) |
| --- | --- | --- |
| `GET` topics home p1 | 1M topics | p99 ≤ **50ms**, sustained ≥ **200 rps** |
| `GET` topics category p1 | 200k in one category | p99 ≤ **50ms**, ≥ **200 rps** |
| `GET` topic by slug | random active | p99 ≤ **40ms**, ≥ **300 rps** |
| `GET` comments flat p1 | 50k-comment topic | p99 ≤ **40ms**, ≥ **200 rps** |
| `GET` comments tree p1 | same, max descendants/root | p99 ≤ **80ms**, response body bounded |
| Mixed 90% read / 10% write | concurrent posters | PG CPU stable; no connection pool exhaustion |
| View flood 1k rps on one topic | — | **zero** per-request `UPDATE topics.view_count` |

Tune targets after first baseline if hardware differs; **never** ship M* without
before/after numbers for that milestone.

---

## Milestone Map

```text
M0  Baseline harness + 1M seed     (measurement first)
M1  ListTopics cold-path           (P0-1)
M2  View count + hot_score         (P0-4; share Iteration A)
M3  ListComments bounds + cache    (P0-2)
M4  Topic detail assembly          (P0-3)
M5  Keyset pagination + total UX   (P0-1 deep)
M6  Cache sharding + COUNT cleanup (P1)
M7  Optional: replica design only  (P2 doc, no code unless needed)
```

Implement **M0 → M1 → M2 → M3 → M4** in order. M5 may start after M1 if
API contract review is ready. M6 after M3+M5. M7 is documentation only.

---

## Resolved Product Defaults (2026-07-21)

These four items were open at plan creation; **accepted as implementation law**
for this task book. Do not re-litigate in PR review without a new plan note.

### D1 — Public list `total` semantics

| Surface | Rule |
| --- | --- |
| Single category filter | Use denormalized **`categories.topic_count`** (already maintained on write). UI shows the number **without** “约”; short stale window after writes is OK. |
| Single tag filter | Use denormalized **`tags.topic_count`** the same way. |
| Unfiltered home / multi-filter / hard-to-count combos | Prefer **approximate or long-TTL cached total**, or de-emphasize total in UI. If estimate is used, OpenAPI must say `total` may be approximate; UI may show “约 N” only for true estimates (not for taxonomy counters). |
| Admin / moderation lists | Keep **exact `COUNT(*)`** (low QPS). |
| Infinite scroll / keyset (M5) | **`hasMore` (+ `nextCursor`) is required**; `total` is secondary and may be omitted or approximate on public feeds. |

**Forbidden on public hot paths:** request-time full-table `COUNT(*)` just to
paint an exact homepage total.

### D2 — Tree descendant cap

| Item | Decision |
| --- | --- |
| Default | **50** descendants per root comment on `view=tree` |
| Runtime option | `forum.comments.tree_descendants_per_root` (or equivalent under existing forum options), range **1–100**, default **50**, restore-defaults supported |
| Overflow | Set **`hasMoreChildren`** (name may match existing FE field) on the root; load more via **`ListCommentReplies`** |
| Mobile | Do **not** lower API default to 20 for mobile; fold in UI if needed. Optional later product: prefer `view=flat` on small screens (out of M3 scope unless free). |

### D3 — View count when / how

| Item | Decision |
| --- | --- |
| Trigger | Successful **public** `GET` topic detail (**by id and by slug**), after resolve — **not** “SSR only” |
| Who | Anonymous + logged-in (Iteration A) |
| Dedup | **30 minutes** per topic per visitor key (session id if logged in; else server visitor cookie or IP+UA hash) |
| Write path | Redis `INCR` delta + River flush to PG; **never** per-request `UPDATE topics.view_count` |
| Redis down | Skip count; detail still **200** |
| Do not count | List, search, sitemap, admin preview, non-public statuses |
| Double-count control | FE must reuse payload so one navigation hits detail API **once**; no extra `onMounted` detail refetch; **no** separate public `POST /view` unless a later proof shows unavoidable double GET |
| Owner | Product path = Iteration A WS1; this plan M2 adds load proof + `hot_score` |

### D4 — Seed and perf tooling layout

| Artifact | Location |
| --- | --- |
| Scale seed | **`cmd/sforum`**: extend `seed:forum` with profile flag **or** `seed:perf` subcommand — same CLI family, append-only, `DATABASE_URL` |
| Load scripts | **`tests/perf/`** (k6 preferred) + short README how to run against Compose + `api-dev` |
| Baseline / after reports | **`knowledge/reports/YYYY-MM-DD-perf-*.md`** |
| Thin shell wrapper | Optional only; must call `sforum`, not reimplement SQL |

**Forbidden:** a second divergent seed implementation under `scripts/` that
bypasses `cmd/sforum` domain rules.

---

## M0 — Capacity Baseline And Seed (do first)

### Design decisions

| Decision | Rule (resolved) | Why |
| --- | --- | --- |
| Tool | k6 (preferred) or vegeta | Scriptable, CI-friendly |
| Load scripts | **`tests/perf/`** + README | D4 |
| Seed CLI | **`cmd/sforum`** profile `perf-1m` (or `seed:perf`) | D4; reuse seed:forum conventions |
| Reports | `knowledge/reports/` dated perf notes | Discoverable baselines |
| Isolation | dedicated DB / docker volume | Never seed 1M into a shared dev DB by accident |
| Proxy | document proxy env for any network installs | Agents.md network rule |

### 0.1 Seed profile

- [x] Add scale profile on **`cmd/sforum`** (flag or subcommand per D4):
  - `small` (existing / quick)
  - `perf-1m`: ~1e6 topics, realistic category distribution, ≥1 topic with 5e4 comments
- [x] Seed is **append-only**, skips domain events, uses bulk inserts / COPY where practical
- [x] Print ETA and row counts; allow resume or clearly fail if partial
- [x] Document disk/time expectations (order-of-magnitude)
- [x] Do **not** put bulk seed SQL only in `scripts/` without going through `sforum`

### 0.2 Load scripts

- [x] Place scripts under **`tests/perf/`** (D4); scenarios match Success Metrics table (home, category, topic, comments, mixed, view flood)
- [x] Warm-up phase + measure phase; report p50/p95/p99, error rate, throughput
- [x] Capture PG pool / Redis hit rate if admin metrics exist; else `EXPLAIN (ANALYZE)` samples for cold ListTopics
- [x] README in `tests/perf/`: how to run against local `api-dev` + Compose PG/Redis

### 0.3 Baseline capture

- [x] Run once on **current main** before M1 code; store results under
  `knowledge/reports/2026-07-21-perf-baseline.md` (or dated successor)
- [x] List top 5 slow queries from baseline (must include ListTopics count + join if present)

**Exit criteria:** any engineer can re-seed and re-run; baseline numbers exist for
comparison. **No production code change required** except seed tooling.

---

## M1 — ListTopics Cold Path (P0-1, highest QPS)

### Design decisions

| Decision | Rule (resolved) | Why |
| --- | --- | --- |
| List body join | **Stop joining full `posts.plain_text` for list** | I/O killer at 1M |
| Excerpt source | Prefer existing short field / `left(plain_text, N)` only if needed; better: denormalized `topics.excerpt` or first-N chars at write time | Stable list width |
| Keyword on list | **Hard-reject** non-empty query in store or service only via search | Remove dead ILIKE branch risk |
| Total | **D1**: taxonomy counters for cat/tag; home approximate or long-TTL cache; no public full-table COUNT | Drop request-path full scans |
| Popular sort | Defer full fix to M2 `hot_score`; M1 may document popular as “best effort” | Avoid two sort rewrites |
| Cache TTL | Keep gen invalidation; optional longer TTL for page=1 only | Homepage stability |

### 1.1 SQL / store

- [x] Rewrite `ListTopics` select list to avoid heavy post body columns
- [x] Ensure default sort `last_activity_at DESC, id DESC` uses
  `topics_category_activity_idx` (verify with EXPLAIN on seeded DB)
- [x] Remove or dead-code-eliminate list `ILIKE` path; service already has
  `ErrUseSearchEndpoint` — store must not reintroduce scan
- [x] Tag filter: keep EXISTS; confirm index `topic_tags (tag_id, topic_id)`
- [x] Optional: covering-friendly column order for index-only scans where easy
  (home: `topics_public_activity_idx` migration `202607210046`)

### 1.2 Total cost (implements D1)

- [x] Single **category** list: set `total` from **`categories.topic_count`** (no `COUNT(*)`)
- [x] Single **tag** list: set `total` from **`tags.topic_count`**
- [x] Unfiltered home: long-TTL cached total and/or PG estimate (`reltuples` or equivalent); **never** full-table count on hot path
  (implemented as `SUM(public categories.topic_count)` + `totalApproximate`; list page=1 Redis TTL 45s)
- [x] Multi-filter edge cases: approximate or cached; document in OpenAPI
- [x] Verify Redis list cache hit does not recompute count
- [x] UI: taxonomy totals display as normal numbers; “约” **only** when total is a true estimate (home), not for cat/tag counters
- [x] Admin paths unchanged (exact count OK)

### 1.3 Contract / frontend

- [x] OpenAPI: document `total` may be denormalized/stale (cat/tag) or approximate (home); public clients should prefer shallow pages / future `hasMore`
- [x] Frontend: tolerate stale total; shallow pagination still works
- [x] i18n: add “约 {n}” (or equivalent) only for approximate-home display if shown
- [x] `ruby scripts/validate-openapi-refs.rb` after contract edits

### 1.4 Tests

- [x] Unit/store tests: list without posts join fields still returns required summary fields
- [x] Service test: non-empty query still rejected / routed
- [x] Regression: pin order + activity order
- [x] Re-run M0 scripts; attach before/after in report
  (`knowledge/reports/2026-07-21-perf-m1-list-topics.md`; k6 binary missing → concurrent LIGHT-class probe)

**Exit criteria:** warm cache home p1 meets target **or** cold path improved ≥2×
vs baseline on same hardware; EXPLAIN shows no sequential scan of posts for default list.
**Met:** cold home ~11.5×; warm p99 ~29 ms; EXPLAIN index-only/limit 20; no posts seq scan.

---

## M2 — View Count + Hot Score (P0-4)

### Ownership split

| Concern | Owner plan |
| --- | --- |
| Product: who counts, dedup 30m, GET side effect, flush job, tests | **Iteration A WS1** — implement there first or same PR series |
| Perf: no per-view PG write; load scenario; couple to popular sort | **This plan M2** |

### Design decisions

| Decision | Rule (resolved) | Why |
| --- | --- | --- |
| When to count | **D3**: successful public detail GET (id + slug), not SSR-only | Avoid undercount on client navigations |
| Counter | Redis INCR delta + SETNX visitor key (30m) | Matches Iteration A + D3 |
| Flush | River job 30–60s | Write batching |
| Display | PG value (+ optional live Redis delta later) | Simple v1 |
| No public view POST | Unless later proof forces it (D3) | Prevent double-count API surface |
| Popular | Add `topics.hot_score` BIGINT maintained on flush / comment write | Indexable sort |
| Formula | e.g. `comment_count * 5 + view_count` stored; recompute on flush and comment count change | Same product meaning as current expression |
| Failure | Redis down → skip view (page still 200) | Read > metrics |

### 2.1 Complete Iteration A view path (D3)

- [x] Implement all checkboxes under Iteration A Workstream 1 (schema/jobs/API/dedup/tests)
- [x] Count on **both** `GET /topics/:id` and `GET /topics/by-slug/:slug` after public resolve
- [x] Dedup 30m; skip list/search/admin; skip non-public statuses
- [x] FE: one detail API per navigation (payload reuse); no mounted re-fetch that double-counts
- [x] No separate `POST /topics/:id/view` in v1
- [x] Cross-link commits in both plan files when done

### 2.2 hot_score

- [x] Migration: `topics.hot_score BIGINT NOT NULL DEFAULT 0`
- [x] Backfill: `hot_score = comment_count * 5 + view_count` (batched)
- [x] Index: e.g. `(is_pinned DESC, hot_score DESC, id DESC)` partial active/locked — validate cardinality
  (`topics_public_hot_idx` + `topics_category_hot_idx`)
- [x] Update on: view flush batch, comment create/delete (and moderation count paths)
- [x] Replace SQL expression sort with `ORDER BY is_pinned DESC, hot_score DESC, id DESC`
- [ ] Search index field: optional sync on flush batch (not every view) — deferred optional

### 2.3 Perf acceptance

- [x] k6 view flood: assert no row-level view update storm (pg_stat_statements or log probe)
  (LIGHT-class concurrent probe; PG view_count flat during flood — see report)
- [x] popular list EXPLAIN uses hot_score index

**Exit criteria:** Iteration A view exit criteria met **and** popular list no longer
sorts by live expression; view flood scenario passes.
**Met:** `knowledge/reports/2026-07-21-perf-m2-view-hot.md`.

---

## M3 — ListComments Bounds + Cache (P0-2)

### Design decisions

| Decision | Rule (resolved) | Why |
| --- | --- | --- |
| Tree descendants | **D2**: hard cap default **50** per root | Prevent mega-thread blowups |
| Config | `forum.comments.tree_descendants_per_root` range 1–100, default 50, restore defaults | Beginner-friendly operator control |
| Overflow | `hasMoreChildren` + existing `ListCommentReplies` | Already partially present |
| Mobile API | Same default 50; no mobile-only lower API cap | D2 |
| Total flat | Prefer `topics.comment_count` when no deleted-inclusion filters | Drop COUNT(*) |
| Total tree | Cache root count per topic gen; or maintain `root_comment_count` | Avoid count each request |
| Cache | Add CachedStore for ListComments / replies with topic-scoped gen | Detail page QPS |
| Default view | Keep API default; optional later flat-on-mobile product choice | No UX fight in M3 |

### 3.1 Tree bound (implements D2)

- [x] Cap descendants per root at configured N (default **50**)
- [x] Add runtime option + recommended default + one-click restore (forum options surface)
- [x] Set `hasMoreChildren` when truncated; OpenAPI documents field + option
- [x] Frontend tree: “加载更多回复” via `ListCommentReplies` with limit
- [x] Unit test: root with N+10 children returns N + hasMore; N respects option

### 3.2 Total

- [x] flat + public active only: use topic.comment_count when safe
- [x] tree: count roots only on miss; cache with comment generation
- [x] IncludeDeleted paths may keep exact COUNT (admin/author rare)

### 3.3 CachedStore

- [x] Override `ListComments` (and optionally `ListCommentReplies`) with short TTL
  (ListComments only; replies remain direct-store — load-more is rarer)
- [x] Bump comment gen on create/update/delete/moderation status change for that topic
- [x] Key includes topicID, view, page/cursor, perPage, includeDeleted flags
  (plus tree cap; IncludeDeleted/author scope skips cache)
- [x] Tests: hit/miss/invalidate (mirror topics tests)

### 3.4 Perf acceptance

- [x] 50k-comment topic tree p1 bounded memory; p99 within budget
  (`knowledge/reports/2026-07-21-perf-m3-list-comments.md`)
- [x] Detail+comments warm path improved vs baseline
  (tree warm p50 ~44 ms vs M0 ~86–106 ms)

**Exit criteria:** no unbounded descendant query; comment list participates in Redis
cache; OpenAPI + public tree UX for “more replies”.
**Met:** report + unit tests; max descendants/root = 50; 11 roots `hasMoreChildren` on hot seed.

---

## M4 — Topic Detail Assembly (P0-3)

### Design decisions

| Decision | Recommendation | Why |
| --- | --- | --- |
| Payload | Keep single GetTopic for now; strip unused fields only if measured | Avoid big contract churn |
| Composite cache | Optional key `forum:topic-page:{id}:{commentPage}` for anonymous | One round-trip for SSR |
| HTTP cache | Public anonymous HTML SWR longer; authenticated no-store | SEO + personalization |
| First paint | Ensure Lazy editor stays; comments request parallel, not waterfall if easy | FE |
| View count | **D3**: one public detail GET per navigation | Avoid double INCR |

### 4.1 API / cache

- [x] Profile GetTopic SQL (joins, revisions); remove accidental heavy joins
  (slug Index Scan ~0.2–0.5 ms; revisions EXISTS only; tags second light query)
- [x] Ensure slug path always hits unique index (`topics_slug_idx`)
- [x] Optional composite cache only if M3+M1 still miss budget after warm cache
  (**skipped**: warm by-slug p99 ~21–37 ms under budget; see M4 report)
- [x] Document cache TTL vs permission-sensitive fields (never cache private)
  (detail dual-write public only; `/t/**` SWR anonymous-only via middleware)

### 4.2 Frontend

- [x] Topic page: parallel fetch topic + comments where not already
  (`Promise.all` when URL has id; slug reuses `topicAsync`)
- [x] Review routeRules for `/t/**` — enable safe SWR for anonymous if security review OK
  (`swr: 60` + `topic-page-cache` for session/`?edit=`)
- [x] **D3**: payload reuse so SSR+hydration do not double-hit detail GET; no mounted re-fetch for view

### 4.3 Perf acceptance

- [x] Topic-by-slug + comments p1 combined scenario meets budget
  (`knowledge/reports/2026-07-21-perf-m4-topic-detail.md`)

**Exit criteria:** detail path no longer the slowest of {home, category, detail}
under warm cache, or documented residual with owner.
**Met:** warm by-slug faster than tree comments; home/detail same class (~10–20 ms);
combined first screen comments-bound (M3 residual).

---

## M5 — Keyset Pagination + Total UX (P0-1 deep)

### Design decisions

| Decision | Rule (resolved) | Why |
| --- | --- | --- |
| Public default | Cursor-based (`after` / `before` opaque token) | Stable cost at depth |
| Compatibility | Keep `page` for page ≤ maxTopicPage; deep page discouraged | SEO + old clients |
| Token | HMAC or signed payload (sort keys + id) — reuse Query Registry cursor ideas if fit | Tamper resistance |
| Total + hasMore | **D1 + M5**: `hasMore` / `nextCursor` **required** on public cursor feeds; `total` follows D1 (denormalized / approximate / optional) | Infinite scroll friendly |
| Comments | Same pattern for flat comments by path_key | Consistency |

### 5.1 Contract

- [x] OpenAPI: `cursor` / `after` query params; response `nextCursor`, `hasMore`
- [x] Document `total` per D1 (not guaranteed exact on public home)
- [x] Document precedence: cursor wins over page when both sent
- [x] Version note in path description; no silent break of page for p1–p200

### 5.2 Store

- [x] ListTopics keyset for default and category sorts (pinned bucket carefully)
- [x] ListComments flat keyset on path_key
- [x] Pinned topics: define stable algorithm (pins first page only vs interleaved)
  (pins as first keyset dimension `is_pinned DESC`, not first-page-only)

### 5.3 Frontend

- [x] Home infinite scroll uses cursor if available
- [x] SFPagination: either shallow page only or hybrid
  (home keeps shallow page links + cursor infinite load)
- [x] i18n: “约 {n}” only for approximate totals (D1); taxonomy counts stay plain

### 5.4 Tests

- [x] Deterministic cursor continuation; no dup/skip under concurrent insert (document limitation)
  (100-step walk 0 dups; concurrent insert may skip/dup — documented limitation)
- [x] k6 deep scroll scenario (many pages) p99 stable
  (`tests/perf/deep_scroll.js` + LIGHT Python report)

**Exit criteria:** scrolling far into a large category does not use large OFFSET;
contract validated; FE uses cursor on primary surfaces.
**Met:** `knowledge/reports/2026-07-21-perf-m5-keyset.md` (100-step cursor p99 ~19 ms).

---

## M6 — Cache Sharding + COUNT Cleanup (P1)

### Design decisions

| Decision | Recommendation | Why |
| --- | --- | --- |
| Topics gen | Split gen by `categoryID` / `tagID` / global | High-traffic reply must not bust all list caches |
| Global home | Separate gen or longer TTL | Home is hottest |
| COUNT leftovers | Grep request path for `count(*)` on topics/comments | Kill remaining hot counts |

### 6.1 Work

- [ ] Refactor CachedStore generation keys to scoped gens
- [ ] Write paths bump only affected scopes
- [ ] Audit and remove unnecessary COUNT in public handlers
- [ ] Tests for scoped invalidation (write in cat A does not miss cat B list cache)

### 6.2 Perf acceptance

- [ ] Under mixed write load, list cache hit rate stays high for untouched categories

**Exit criteria:** write storm in one category does not invalidate whole-site topic lists.

---

## M7 — Horizontal Scale (doc only unless product demands)

- [ ] Decision note: when to add read replica (metrics threshold)
- [ ] Session/Redis already shared — document API stateless assumptions
- [ ] Explicitly **no code** until M0–M6 numbers show single-node ceiling

---

## Permission And Safety Notes

- Public list/detail remain public; no new permission keys for read path work
- Cursor tokens must not leak internal IDs of hidden/pending content
- Cached responses must not include permission-elevated deleted comments for wrong user
- View counting must not become an unauthenticated write amplification vector (dedup + rate limit)
- Load tests only against environments the operator owns

## Testing Gate Per Milestone

Minimum before merging a milestone:

1. Focused `go test` for touched forum packages  
2. OpenAPI ref validation if contracts changed  
3. Relevant FE typecheck if UI changed  
4. M0 perf script subset for that path with before/after note in PR or report  
5. Update this plan checkboxes + `knowledge/modules/forum.md`  

Full `./scripts/test.sh` when milestone is large or contract-heavy.

## Knowledge Updates When Finishing A Milestone

- [ ] `knowledge/modules/forum.md` — cache, pagination, view, comment bounds
- [ ] This plan status / checkbox progress
- [ ] `knowledge/plans/README.md` if status changes
- [ ] Short hot handoff under `knowledge/sessions/` when a multi-day slice lands
- [ ] No new ADR required for D1–D4 unless implementation diverges; this plan section is the product policy source until then

## Suggested PR Slice Size

| PR | Content |
| --- | --- |
| PR0 | `cmd/sforum` perf seed profile + `tests/perf` k6 harness + baseline report |
| PR1 | ListTopics slim select + ILIKE removal + **D1 totals** |
| PR2 | Iteration A view count + flush job (**D3**) |
| PR3 | hot_score migration + popular sort |
| PR4 | comment descendant cap default 50 + option + OpenAPI + FE more replies (**D2**) |
| PR5 | ListComments CachedStore |
| PR6 | detail FE parallel / SWR + single detail fetch for view dedup |
| PR7 | keyset topics + FE home (`hasMore`) |
| PR8 | keyset comments + total cleanup |
| PR9 | cache sharding |

Prefer this granularity over one mega-PR.

## Open Questions

**None remaining for D1–D4** (resolved 2026-07-21; see **Resolved Product Defaults**).

Still free to decide during implementation (not product blockers):

1. Exact `hot_score` formula weights if product wants time-decay later (M2 ships non-decay formula first).
2. Pinned-topic + keyset interleaving algorithm detail (M5).
3. Whether home approximate total is shown in UI or hidden in favor of infinite scroll only (M1/M5 FE choice; API still documents semantics).

---

## Progress Log

| Date | Note |
| --- | --- |
| 2026-07-21 | Task book created from performance path review; status **ready**. |
| 2026-07-21 | Resolved D1–D4 (total semantics, tree cap 50, view on public GET+30m dedup, seed in `cmd/sforum` + `tests/perf`). Open questions closed; milestones M0–M5 updated to match. |
| 2026-07-21 | **M0 done:** `seed:forum --profile=perf-1m` / `seed:perf` bulk seed; `tests/perf` k6 + `LIGHT=1`; baseline `knowledge/reports/2026-07-21-perf-baseline.md` (1e6 topics + 50k hot comments on `sforum_perf`). Next: M1 ListTopics slim + D1 totals. |
| 2026-07-21 | **M1 done:** ListTopics page-CTE slim select + D1 totals + no list ILIKE; `topics_public_activity_idx`; OpenAPI `totalApproximate` + FE「约」; report `knowledge/reports/2026-07-21-perf-m1-list-topics.md` (home cold ~11.5×, warm p99 ~29 ms). Next: **M2** view count + `hot_score` (Iteration A WS1). |
| 2026-07-21 | **M2 done:** D3 view count (GET detail + 30m Redis dedup + INCR + `forum.flush_view_counts` 45s); `topics.hot_score` + hot indexes; list `sort=hot` column; Iteration A WS1 checkboxes; report `knowledge/reports/2026-07-21-perf-m2-view-hot.md` (flood 0 per-req UPDATE; hot Index Scan). Next: **M3** ListComments bounds + cache. |
| 2026-07-21 | **M3 done:** D2 tree cap (`forum.comments.tree_descendants_per_root` default 50) + `hasMoreChildren` + FE load more; flat total via `comment_count`; ListComments CachedStore topic gen; report `knowledge/reports/2026-07-21-perf-m3-list-comments.md` (warm tree p50 ~44 ms; max desc/root 50). Next: **M4** topic detail assembly. |
| 2026-07-21 | **M4 done:** GetTopic profile (slug unique index); CachedStore id+slug dual-write + reverse-map invalidate; no composite page cache (evidence); FE parallel topic+comments; `/t/**` anonymous SWR + auth/edit gate; report `knowledge/reports/2026-07-21-perf-m4-topic-detail.md` (by-slug warm p99 ~21 ms). Next: **M5** keyset pagination. |
| 2026-07-21 | **M5 done:** public ListTopics / ListComments flat `after` keyset + `hasMore`/`nextCursor`; cursor > page; pin-stable seek SQL; FE home infinite scroll; report `knowledge/reports/2026-07-21-perf-m5-keyset.md` (100-step cursor p99 ~19 ms). Next: **M6** cache sharding. |
