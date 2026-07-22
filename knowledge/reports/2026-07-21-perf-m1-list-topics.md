# 2026-07-21 Perf After M1 — ListTopics Cold Path

Status: **M1 measured** against the same dedicated DB and hardware class as M0.

Task book: `knowledge/plans/archive/2026-07/2026-07-21-million-scale-read-path.md` (M1).
Baseline: `knowledge/reports/2026-07-21-perf-baseline.md` (M0).

## Environment (same class as M0)

| Item | Value |
| --- | --- |
| Host | macOS, x86_64, ~16 GiB RAM |
| Postgres | Docker `postgres:17-alpine`, shm 64 MiB |
| Perf DB | `sforum_perf` on `:15432` — **1e6** topics (unchanged seed) |
| API under test | Second process on **:8082** → `sforum_perf` |
| Redis | Compose (list cache flushed before cold sequential probes) |
| k6 | Not available in this environment (brew bottle 404); used sequential curl + **LIGHT-equivalent** concurrent Python (5 workers) |

## Code changes under test (M1)

1. **Slim list**: page CTE selects topic IDs only; `posts.plain_text` via `left(..., 2000)` only for the page rows (no full body/raw/html on list).
2. **D1 totals**: category/tag → `topic_count`; home → `SUM(categories.topic_count)` + `totalApproximate=true`; multi-filter → `min` of counters + approximate; **no** public `COUNT(*)`.
3. **ILIKE removed** from store list SQL (service still returns `ErrUseSearchEndpoint` for non-empty `query`).
4. **Indexes**:
   - Category default sort: **Index Only Scan** on `topics_category_activity_idx` with **LIMIT 20** (stops after 20).
   - Home default sort: new partial index `topics_public_activity_idx` on `(is_pinned DESC, last_activity_at DESC, id DESC)` WHERE active/locked.
5. CachedStore: cache key includes `sort`; page=1 TTL 45s.
6. OpenAPI + FE: `totalApproximate`; UI “约 N” only when approximate.

## Before / after (M0 baseline → M1)

### A) Sequential curl (cold then warm)

| Scenario | M0 cold | M1 cold | M0 warm (med) | M1 warm |
| --- | --- | --- | --- | --- |
| Home `GET /topics?page=1` | **2.77 s** | **0.24 s** (~**11.5×**) | ~10–20 ms | **~9–11 ms** |
| Category `?categorySlug=general` | **0.70 s** | **0.10 s** (~**7×**) | ~13–31 ms | **~10–18 ms** |

API body checks after M1:

- Home: `total=1000000`, `totalApproximate=true`, 20 items, excerpt present (~180 runes).
- Category: `total=200000`, `totalApproximate` omitted/false.

### B) Concurrent LIGHT-class load (5 workers, n=80, warm)

| Scenario | M0 k6 LIGHT p50 / p99 / rps / err | M1 concurrent p50 / p99 / rps / err |
| --- | --- | --- |
| home_topics | 13.9 ms / **163 ms** / ~32 rps / 0.47% | **15.7 ms / 28.7 ms** / **~287 rps** / **0%** |
| category_topics | 14.3 ms / **108 ms** / ~21 rps / 0% | **18.3 ms / 27.2 ms** / **~258 rps** / **0%** |

Notes:

- M1 concurrent tool is not k6; stages differ slightly from `tests/perf/*`. Direction and order-of-magnitude match LIGHT=1 intent (≤5 concurrent).
- Warm home p99 **28.7 ms** is under the plan’s **50 ms** starting target; sustained ~287 rps at 5 workers exceeds the **200 rps** recommendation for this single process + Docker PG.

### C) EXPLAIN (ANALYZE) on `sforum_perf`

| Query | M0 (baseline top slow) | M1 |
| --- | --- | --- |
| Home list p1 (id page) | ~**15.4 s** parallel seq scan + disk sort of 1e6 | **~0.4 ms** `Index Scan topics_public_activity_idx` + LIMIT 20 |
| Category general p1 | ~**2.0 s** bitmap + sort 200k | **~0.16 ms** `Index Only Scan topics_category_activity_idx` rows=**20** |
| Public total | ~**1.7 s** full-table `COUNT(*)` | **~0.06 ms** `SUM(topic_count)` on 20 categories |
| Posts join | In sort set (heavy) | Only after page CTE (20 rows) via `posts_pkey` |

**No sequential scan of `posts` for default list.** Category path uses `topics_category_activity_idx` as required.

## Exit criteria check

| Criterion | Result |
| --- | --- |
| Warm home p1 ≤ 50 ms p99 **or** cold ≥2× vs baseline | **Both**: cold ~11.5×; warm p99 ~29 ms |
| EXPLAIN: no seq scan of posts for default list | **Pass** |
| D1: no public full-table COUNT | **Pass** |
| ILIKE gone from list store path | **Pass** |

## Residual (not M1)

- `hot` sort still expression-based (M2 `hot_score`).
- Tag-only list still uses EXISTS (acceptable; counters for total).
- Deep OFFSET still clamped at page 200 (M5 keyset).
- View count / ListComments cache (M2/M3).
- Re-run with real k6 when binary is available; numbers should be re-captured into a short addendum if they diverge.

## Artifacts

| Artifact | Path |
| --- | --- |
| ListTopics rewrite | `apps/api/app/Models/Forum/list_topics.go` |
| Global activity index | `apps/api/database/migrations/202607210046_topics_public_activity_idx.sql` |
| This report | `knowledge/reports/2026-07-21-perf-m1-list-topics.md` |
