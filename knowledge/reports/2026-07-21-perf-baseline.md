# 2026-07-21 Perf Baseline (M0, pre-M1)

Status: **measurement only** — current `main` read path, **no** M1+ ListTopics/view/hot_score/comment-bound rewrites.

Task book: `knowledge/plans/archive/2026-07/2026-07-21-million-scale-read-path.md` (M0).

## Hardware / environment

| Item | Value |
| --- | --- |
| Host | macOS, x86_64, ~16 GiB RAM |
| Postgres | Docker `postgres:17-alpine` (`sforum-postgres-1`), **shm 64 MiB** |
| Redis | Docker Compose (shared with dev) |
| Perf DB | `sforum_perf` on port `15432` (dedicated; not the casual `sforum` DB) |
| Seed | `cmd/sforum seed:perf --confirm-perf-db` → **1,000,000** topics, **50,000** comments on `perf-hot-thread`, 20 categories, 200 users |
| Seed wall time | **5m26s** bulk path (~3266 topics/s peak) |
| DB size after seed | **~2.1 GiB** |
| API under test | Second process on **:8082** pointed at `sforum_perf` + `BUILTIN_EXTENSION_ROOT=storage/builtin-dev` |
| Main API / web | Left alone on **:8081** / **:3000** |

## Seed how-to (reproducible)

```bash
docker exec sforum-postgres-1 psql -U sforum -d postgres -c "CREATE DATABASE sforum_perf;"
export DATABASE_URL='postgres://sforum:sforum@127.0.0.1:15432/sforum_perf?sslmode=disable'
cd apps/api && go run ./cmd/migrate
go run ./cmd/sforum seed:forum --profile=perf-1m --dry-run   # plan only
go run ./cmd/sforum seed:perf --confirm-perf-db --database-url="$DATABASE_URL"
```

Isolation: writes require `--confirm-perf-db`. Do not seed 1e6 into a shared casual dev DB.

## Load scripts

- Location: `tests/perf/` (k6)
- Scenarios: home, category, topic-by-slug, comments flat/tree, mixed, view flood
- Prefer **`LIGHT=1`** against Compose PG (default `run-all.sh`). Full 50 VU stages can exhaust Docker **shm=64m** under ListTopics `COUNT(*)` + sort and take the API down (observed: `SQLSTATE 53100 could not resize shared memory segment`).

## Measured numbers

### A) Sequential curl (single-flight, after restart; cold then warm)

| Scenario | cold | warm (median of next 4) | Notes |
| --- | --- | --- | --- |
| Home `GET /topics?page=1` | **2.77 s** | **~10–20 ms** | Redis list cache warms after first hit |
| Category `?categorySlug=general` | **0.70 s** | **~13–31 ms** | 200k topics in `general` |
| Topic by slug `perf-hot-thread` | **44 ms** | **~17–22 ms** | slug unique index |
| Comments flat p1 (50k thread) | **116 ms** | **~38–54 ms** | |
| Comments tree p1 | **122 ms** | **~86–106 ms** | unbounded descendants risk remains (M3) |
| View seq burst ×20 by-slug | — | **~15–27 ms** each | v1 has **no** Redis view path; each hit still eligible for PG view update when product implements it — today view_count stays 0 on seed data |

### B) k6 `LIGHT=1` (≤5 VUs, warm-ish)

| Scenario | p50 | p95 | p99 | max | error rate | throughput |
| --- | --- | --- | --- | --- | --- | --- |
| home_topics | 13.9 ms | 42.4 ms | **163 ms** | 16.4 s | 0.47% | ~32 rps |
| category_topics | 14.3 ms | 38.4 ms | **108 ms** | 3.5 s | 0.00% | ~21 rps |
| topic_by_slug / comments / mixed / view flood | — | — | — | — | **API died mid-suite** | — |

Mid-suite failure cause: concurrent load + residual PG pressure after home/category work; HTTP listener drained (`api runtime authority lost`). **Not** a k6 script bug. Prefer sequential probes + `LIGHT=1` on this Docker shm size.

### C) Earlier aggressive run (50 VUs) — diagnostic only

- Home: first ~2.5k successes while cache warm, then **92% errors** as PG shared memory exhausted.
- Subsequent scenarios: **100% connection refused** after API drain.
- Do **not** treat those p99/error rates as product targets; they measure infrastructure collapse under ListTopics cost.

## Top 5 slow queries (EXPLAIN ANALYZE on `sforum_perf`)

Evidence: implementer scratch `explain-slow.log` (cold-ish, concurrent noise possible).

| Rank | Query shape | Actual time | Diagnosis |
| --- | --- | --- | --- |
| 1 | **ListTopics home**: join categories + **posts**, `ORDER BY is_pinned, last_activity_at, id`, `LIMIT 20` | **~15.4 s** | Parallel seq scan of **1e6** topics + **external merge Disk sort ~109 MB**; posts index only for 20 rows after sort |
| 2 | **ListTopics tree roots filter** on 50k-comment topic (`parent IS NULL` via path index filter) | **~4.8 s** | Index scan with heavy filter / row removal (tree path still expensive) |
| 3 | **Category list** general: bitmap on `topics_category_activity_idx` then sort 200k | **~2.0 s** | Index helps filter category but still materializes large set before top-N |
| 4 | **Public total `COUNT(*)`** on topics `status IN (active,locked)` | **~1.7 s** | **Full table sequential scan** — forbidden on hot path after M1 (D1) |
| 5 | Comments flat `path_key` p1 | **~17 ms** | Healthy index path (`comments_topic_path_idx`) |

**Must-call-out for M1:** ListTopics **count + posts join/sort** dominate homepage cold path. Slug detail is already fine (~1 ms SQL).

## Gaps vs Success Metrics defaults (plan table)

| Target (API, warm, single process) | Baseline reality |
| --- | --- |
| Home p99 ≤ 50 ms, ≥ 200 rps | Warm p99 ~163 ms at ~32 rps (`LIGHT=1`); cold multi-second; 200 rps not sustainable on this Docker PG without M1 |
| Category p99 ≤ 50 ms | Warm p99 ~108 ms at ~21 rps; cold ~0.7 s |
| Topic by slug p99 ≤ 40 ms | Sequential warm ~20 ms (meets single-flight); concurrent suite incomplete |
| Comments flat/tree | Sequential warm OK-ish; tree still unbounded product risk |
| View flood: zero per-request `UPDATE topics.view_count` | **Not implemented** (Iteration A / M2); baseline does not claim pass |

## Product implications (for M1+, not done here)

1. Kill public home **full-table COUNT** (D1).
2. Stop joining full post bodies for list; use slim select / excerpt strategy (M1).
3. Prefer index-ordered scans over disk sort of 1e6 rows for p1.
4. Raise Docker Postgres `shm_size` for serious load re-runs (ops note, not product).
5. Cap tree descendants (D2 / M3); cache ListComments (M3).
6. View count Redis INCR (D3 / Iteration A / M2).

## Artifacts

| Artifact | Path |
| --- | --- |
| Seed CLI | `apps/api/cmd/sforum` `seed:forum --profile=perf-1m` / `seed:perf` |
| k6 suite | `tests/perf/` |
| This report | `knowledge/reports/2026-07-21-perf-baseline.md` |

## Explicit non-claims

- This report does **not** claim M1 wins.
- This report does **not** claim “million-row second open” marketing readiness.
- Aggressive 50 VU failure is an environment + cold-path cost finding, not a reason to skip M1.
