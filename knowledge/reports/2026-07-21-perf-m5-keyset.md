# 2026-07-21 Perf After M5 — Keyset Pagination

Status: **M5 measured** against the same dedicated DB class as M0–M4.

Task book: `knowledge/plans/2026-07-21-million-scale-read-path.md` (M5).  
Prior: `perf-m4-topic-detail.md`.

## Environment

| Item | Value |
| --- | --- |
| Host | macOS, x86_64 |
| Postgres | Docker `postgres:17-alpine` |
| Perf DB | `sforum_perf` on `:15432` — **1e6** topics, **50k** comments on `perf-hot-thread` |
| API under test | Second process on **:8082** → `sforum_perf`, `EMBED_WORKER_IN_API=true`, `SFORUM_SAFE_MODE=1` |
| Redis | Compose `sforum-redis-1` (`127.0.0.1:16379`) |
| k6 | Not required; LIGHT-class sequential + concurrent Python |

## Code under test (M5)

1. **OpenAPI:** `after` query; response `hasMore` + `nextCursor`; **cursor wins over page**.
2. **ListTopics keyset:** opaque base64url cursor `(sort, pin, sortKey, id)`; `is_pinned` first dimension; seek-friendly `col <= $k AND (col < $k OR id < $id)`.
3. **ListComments flat keyset:** `path_key` + `id`; tree keeps page + `hasMore` from root total.
4. **CachedStore:** key includes `after`; page=1 longer TTL only when no cursor.
5. **FE home:** infinite scroll prefers `nextCursor` / `after`; taxonomy「约」only when `totalApproximate`.
6. **Perf script:** `tests/perf/deep_scroll.js` (cursor vs page modes).

## Results

### A) SQL EXPLAIN (category `general`, active sort)

| Query | Plan | Execution |
| --- | --- | --- |
| Keyset after ~offset 2000 (seek form) | **Index Only Scan** `topics_category_activity_idx`; Index Cond includes `last_activity_at <= …` | **~0.14–0.4 ms** |
| OFFSET 4000 LIMIT 21 | Index Only Scan walks **4021** rows | **~2.2 ms** |
| OFFSET 3600 (earlier sample) | Index Only Scan rows=3620 | **~24 ms** (cold buffers) |

Keyset stays near-constant SQL cost; deep OFFSET cost grows with skipped rows.

### B) API latency (LIGHT-class)

Warm Redis after first hits; sequential unless noted.

| Scenario | p50 | p99 | Notes |
| --- | --- | --- | --- |
| Cursor deep scroll **100 steps** cold-ish | **12.8** ms | **18.8** ms | step50≈12 ms, last≈15 ms — **flat with depth** |
| OFFSET pages 50–74 cold | 13.7 ms | 17.9 ms | Still OK on category activity index |
| OFFSET pages 180–200 cold | 15.1 ms | 19.4 ms | Clamp region; SQL walk cost higher than keyset at same depth |
| Cursor warm re-hit 25 steps | **5.6** ms | **8.5** ms | CachedStore hit |
| Concurrent cursor 5×10 | **9.8** ms | **21.8** ms | 0 errors |
| Flat comments cursor 20 steps (hot thread) | 21.5 ms | 29.6 ms | path_key keyset |

**Before M5 (deep OFFSET only):** scroll past page clamp required large OFFSET; home infinite scroll used `page++` and degraded toward clamp.  
**After M5:** primary feed uses `after`/`nextCursor`; 100-step category scroll p99 **≤ 20 ms** cold, **≤ 9 ms** warm.

### C) Correctness probes

| Check | Result |
| --- | --- |
| 100-step cursor walk duplicates | **0** |
| `after` + `page=99` vs `after` only | **Same items** (cursor precedence) |
| Invalid `after` | **400** `forum.cursor_invalid` |
| Pinned dimension | Documented: pins first via `is_pinned DESC` in keyset |

## Exit criteria check

| Criterion | Result |
| --- | --- |
| OpenAPI hasMore/nextCursor + cursor > page | **Pass** |
| ListTopics keyset + stable pin algorithm | **Pass** |
| ListComments flat keyset | **Pass** |
| FE home cursor / infinite scroll | **Pass** |
| Deep scroll p99 stable (not OFFSET-bound) | **Pass** (report) |

## Artifacts

| Artifact | Path |
| --- | --- |
| Plan | `knowledge/plans/2026-07-21-million-scale-read-path.md` |
| This report | `knowledge/reports/2026-07-21-perf-m5-keyset.md` |
| Perf script | `tests/perf/deep_scroll.js` |
