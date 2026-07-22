# 2026-07-21 Perf After M2 — View Count + Hot Score

Status: **M2 measured** against the same dedicated DB class as M0/M1.

Task book: `knowledge/plans/archive/2026-07/2026-07-21-million-scale-read-path.md` (M2).
Related product path: `knowledge/plans/2026-07-12-iteration-a-engagement-loop.md` WS1.
Prior: `reports/2026-07-21-perf-baseline.md` (M0), `reports/2026-07-21-perf-m1-list-topics.md` (M1).

## Environment

| Item | Value |
| --- | --- |
| Host | macOS, x86_64 |
| Postgres | Docker `postgres:17-alpine` |
| Perf DB | `sforum_perf` on `:15432` — **1e6** topics (unchanged seed) + migration `202607210047` hot_score backfill |
| API under test | Second process on **:8082** → `sforum_perf`, `EMBED_WORKER_IN_API=true`, `SFORUM_SAFE_MODE=1` (theme island allowlist noise on this DB; JSON API unaffected) |
| Redis | Compose `sforum-redis-1` (shared with main dev) |
| k6 | Not used; LIGHT-class concurrent Python (5 workers, n=100) for view flood |

## Code under test (M2)

1. **D3 view count (Iteration A WS1 once):**
   - Public `GET` detail by id + by-slug → `RecordTopicView` after resolve
   - Visitor key: `u:{id}` / `s:{sid}` / `a:{sha256(ip+ua)[:16]}`
   - Redis `SETNX` dedup 30m + `INCR` delta + dirty set
   - River schedule `forum.flush_view_counts` every **45s** → `ApplyViewCountDeltas` (`view_count += δ`, `hot_score += δ`)
   - Redis down / nil recorder → skip count, detail still 200
   - **No** public `POST /view`
2. **hot_score:**
   - Column + backfill `comment_count*5+view_count`
   - Indexes `topics_public_hot_idx`, `topics_category_hot_idx`
   - Comment create/delete/moderation maintain score; list `sort=hot` uses column (not expression)
3. **FE:** `SFTopicShowPage` single `useAsyncData` detail load per navigation (payload reuse)

## Results

### A) View flood (LIGHT-class)

| Metric | Result |
| --- | --- |
| Requests | 100 concurrent (5 workers) `GET /topics/by-slug/perf-hot-thread` |
| HTTP | **100/100 status 200**, 0 errors |
| Latency | p50 **18.3 ms**, p95 **45.2 ms**, p99 **56.4 ms**, ~**226 rps** |
| PG `view_count` during flood | stayed **0** (pre-flush) |
| Redis after flood | dirty member topic `1`, delta key **51** (50 unique UA + 1 prior curl) |
| PG after ~50s flush window | `view_count=51`, `hot_score=250051` (from `250000+51`; comment_count 50k → base hot 250000) |

**Exit criterion:** zero per-request `UPDATE topics.view_count` during flood — **Pass** (counter only moved after River flush).

### B) popular / hot list EXPLAIN

| Query | Plan | Time |
| --- | --- | --- |
| Home hot p1 | **Index Scan `topics_public_hot_idx`** + LIMIT 20 | ~**1.2 ms** exec |
| Category general hot p1 | **Index Only Scan `topics_category_hot_idx`** rows=20 | ~**2.9 ms** exec |
| API `GET /topics?sort=hot&page=1` | 200, 20 items, total≈1e6 approximate | ~55 ms cold-ish |

**Exit criterion:** popular list no longer sorts by live expression; uses hot_score index — **Pass**.

### C) Contrast to M0/M1 (detail path)

| Scenario | M0 / M1 note | M2 |
| --- | --- | --- |
| Detail GET | No view side effect; expression hot sort | View via Redis; hot via column+index |
| View flood | Would require per-request UPDATE if naive | **No** PG write per request |

## Exit criteria check

| Criterion | Result |
| --- | --- |
| Iteration A WS1 product path (dedup, flush, GET side effect) | **Pass** |
| No public POST /view | **Pass** |
| View flood: zero per-request view_count UPDATE | **Pass** |
| hot list EXPLAIN uses hot_score index | **Pass** |

## Residual / next

- Search index `view_count` reindex on flush remains optional (not every view).
- Theme watcher on `sforum_perf` needs `SFORUM_SAFE_MODE=1` until `sf-my-home-page` island allowlist is aligned (unrelated to M2 read path).
- Builtin plugin package digests may drift after local `backend/plugin` rebuilds — refresh `sforum.extension.json` digests before SyncBuiltins.
- **M3** next: ListComments bounds + cache (D2 tree cap).
