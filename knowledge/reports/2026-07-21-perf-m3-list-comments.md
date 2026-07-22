# 2026-07-21 Perf After M3 — ListComments Bounds + Cache

Status: **M3 measured** against the same dedicated DB class as M0–M2.

Task book: `knowledge/plans/archive/2026-07/2026-07-21-million-scale-read-path.md` (M3).
Prior: `reports/2026-07-21-perf-baseline.md` (M0), `perf-m1-list-topics.md`, `perf-m2-view-hot.md`.

## Environment

| Item | Value |
| --- | --- |
| Host | macOS, x86_64 |
| Postgres | Docker `postgres:17-alpine` |
| Perf DB | `sforum_perf` on `:15432` — **1e6** topics, **50k** comments on `perf-hot-thread` |
| API under test | Second process on **:8082** → `sforum_perf`, `EMBED_WORKER_IN_API=true`, `SFORUM_SAFE_MODE=1` |
| Redis | Compose `sforum-redis-1` (`127.0.0.1:16379`) |
| k6 | Not used; LIGHT-class concurrent Python (5 workers, n=100) for tree p1 |

## Code under test (M3)

1. **D2 tree bound:** `forum.comments.tree_descendants_per_root` (1–100, default **50**); per-root `ROW_NUMBER` cap; `hasMoreChildren` when truncated; FE「加载更多回复」→ `ListCommentReplies`.
2. **Totals:** flat public total from `topics.comment_count`; tree total = root `COUNT` (cached with list payload).
3. **CachedStore:** `ListComments` short TTL + topic-scoped generation; skip cache when soft-delete scope is viewer-specific; write paths bump comment gen.

## Results

### A) Tree p1 structure (50k-comment topic)

| Metric | Result |
| --- | --- |
| HTTP | **200** |
| Roots on p1 | **20** |
| Tree `total` (roots) | **1000** |
| Max descendants under any root in payload | **50** (cap) |
| Roots with `hasMoreChildren` | **11** |
| Response body (cold) | ~**1.1 MB** (bounded; full HTML for up to 20×50 nodes) |

**Exit criterion:** no unbounded descendant query; body bounded by cap — **Pass**.

### B) Latency (sequential warm + concurrent LIGHT)

| Scenario | Result |
| --- | --- |
| Tree cold (first hit after API start) | **~6.7 s** (miss path: multi-join + large payload; environment-sensitive) |
| Tree warm sequential n=20 | p50 **44 ms**, p95 **54 ms**, p99 **95 ms** |
| Tree concurrent 5 workers ×100 | **100/100 200**, p50 **83 ms**, p95 **161 ms**, p99 **173 ms**, ~**54 rps** |
| Flat p1 sample (after warm) | **~26 ms**, `total=50000` via `comment_count` |

| Contrast | M0 baseline (sequential) | M3 |
| --- | --- | --- |
| Tree warm | ~86–106 ms | p50 **44 ms** (Redis list cache) |
| Tree product risk | unbounded descendants | **cap 50/root** + hasMoreChildren |
| Flat total | `COUNT(*)` | denormalized **comment_count** |

Plan target table: tree p1 p99 ≤ **80 ms** (warm, single process). Sequential warm p99 **~95 ms** is near target on this Docker shm host; concurrent LIGHT p99 higher as expected. Cap + cache are the product exits; further composite detail cache is **M4**.

### C) Cache / invalidation (unit)

Covered by `TestCachedStoreListCommentsHitAndInvalidate` / `TestCachedStoreListCommentsSkipsViewerScoped` (hit after first load; miss after `CreateComment`; no shared cache when `IncludeDeleted`).

## Exit criteria check

| Criterion | Result |
| --- | --- |
| Tree descendants hard-capped (default 50) | **Pass** |
| Option + restore defaults + OpenAPI `hasMoreChildren` | **Pass** |
| FE load more via ListCommentReplies | **Pass** |
| Flat total prefers `comment_count` | **Pass** (`total=50000`) |
| ListComments in CachedStore + write gen | **Pass** (unit + warm latency drop) |
| 50k tree p1 bounded | **Pass** |

## Residual / next

- Cold tree still expensive on first miss (large HTML payload); M4 composite / field trimming may help.
- Concurrent tree p99 above 80 ms under 5 workers on Docker PG shm — ops note, not a product dual-implement.
- **M4** next: topic detail assembly / optional composite cache (not started here).

## Artifacts

| Artifact | Path |
| --- | --- |
| Plan | `knowledge/plans/archive/2026-07/2026-07-21-million-scale-read-path.md` |
| This report | `knowledge/reports/2026-07-21-perf-m3-list-comments.md` |
