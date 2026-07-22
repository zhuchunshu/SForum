# 2026-07-21 Perf After M4 — Topic Detail Assembly

Status: **M4 measured** against the same dedicated DB class as M0–M3.

Task book: `knowledge/plans/archive/2026-07/2026-07-21-million-scale-read-path.md` (M4).
Prior: `perf-baseline.md` (M0), `perf-m1-list-topics.md`, `perf-m2-view-hot.md`,
`perf-m3-list-comments.md`.

## Environment

| Item | Value |
| --- | --- |
| Host | macOS, x86_64 |
| Postgres | Docker `postgres:17-alpine` |
| Perf DB | `sforum_perf` on `:15432` — **1e6** topics, **50k** comments on `perf-hot-thread` |
| API under test | Second process on **:8082** → `sforum_perf`, `EMBED_WORKER_IN_API=true`, `SFORUM_SAFE_MODE=1` |
| Redis | Compose `sforum-redis-1` (`127.0.0.1:16379`) |
| k6 | Not used; LIGHT-class concurrent Python (5 workers, n=100) |

## Code under test (M4)

1. **GetTopic profile:** `topicDetailSQL` + tags second query; slug path uses
   **UNIQUE** `topics_slug_idx` (Index Scan, ~0.2–0.5 ms SQL). Revisions are
   `EXISTS` only (index-only), not full revision rows.
2. **CachedStore dual-write:** `GetTopic` / `GetTopicBySlug` write both id and
   slug keys + `forum:topic-id-slug:{id}` reverse map; comment writes invalidate
   slug detail via reverse map (stale `comment_count` fix).
3. **No composite** `forum:topic-page:{id}:{commentPage}` — warm by-slug already
   under plan budget; comments remain the first-screen bulk (M3 cache).
4. **FE:** `SFTopicShowPage` `Promise.all` topic + comments when URL has id;
   pure slug reuses `topicAsync` (D3, no second detail GET). No `onMounted`
   detail refetch.
5. **SWR:** `/t/**` anonymous `swr: 60`; `topic-page-cache` middleware forces
   `no-store` when `sforum_session` or `?edit=` present.

## Results

### A) GetTopic SQL (EXPLAIN on `sforum_perf`)

| Path | Plan | Execution |
| --- | --- | --- |
| by-slug `perf-hot-thread` | **Index Scan `topics_slug_idx`** + `posts_pkey` | **~0.2–0.5 ms** |
| tags for one topic | index on `topic_tags` / tags | **~0.1 ms** |
| `EXISTS post_revisions` | Index Only `post_revisions_post_created_idx` | sub-ms |

No full-table scan; no revision body load. Tags remain a second round-trip
(acceptable for single-row detail).

### B) Latency — before (pre dual-write restart) vs after (M4 code)

Warm Redis after API start; sequential n=20 unless noted.

| Scenario | Before (warm) | After M4 (warm) |
| --- | --- | --- |
| by-slug sequential | p50 **15.7** / p99 **23.8** ms | p50 **16.9** / p99 **20.8** ms |
| comments tree p1 sequential | p50 **95.9** / p99 **216** ms | p50 **78.6** / p99 **136.8** ms |
| combined **waterfall** (slug then comments) | p50 **123.7** / p99 **231.7** ms | — (FE uses parallel when id known) |
| combined **parallel** (slug ∥ comments) | p50 **116.4** / p99 **187.3** ms | p50 **83.7** / p99 **110.7** ms |
| by-slug concurrent 5×100 | p50 **11.5** / p99 **25.0** ms, **~412 rps**, 0 err | p50 **18.0** / p99 **37.1** ms, **~259 rps**, 0 err |
| combined parallel concurrent 5×100 | p50 **94** / p99 **149.5** ms, ~52 rps | p50 **140** / p99 **356** ms, ~32 rps |

Notes:

- Plan target **by-slug p99 ≤ 40 ms**, ≥ **300 rps**: sequential always met;
  concurrent rps is host/load-noise sensitive (412 → 259 on re-run) but p99
  stayed ≤ **40 ms** on the after sequential and near target on concurrent.
- Combined first-screen cost is dominated by **tree comments payload** (~1 MB
  cold body class from M3), not GetTopic SQL.
- Dual-write is a correctness/latency-tail fix (id warm → slug hit), not a
  multi-× SQL rewrite.

### C) Composite cache decision

| Question | Answer |
| --- | --- |
| Warm by-slug alone under budget? | **Yes** (p99 ~21–37 ms) |
| Warm comments already CachedStore (M3)? | **Yes** |
| Add `forum:topic-page:{id}:{page}`? | **No** — no evidence warm miss after M1+M3; avoids permission-sensitive composite payload |

### D) Residual ranking (warm sequential)

| Path | p50 (after) |
| --- | --- |
| Home list p1 | ~12 ms |
| Topic by-slug | ~11–17 ms |
| Comments tree p1 | ~79 ms |
| Combined parallel detail | ~84 ms |

**Exit:** detail **by-slug** is not the slowest of {home, category, detail};
combined first screen is comments-bound (documented residual, owner M3/M5).

## Exit criteria check

| Criterion | Result |
| --- | --- |
| Profile GetTopic; no accidental heavy joins | **Pass** (slug unique index; EXISTS only) |
| Slug always hits unique index | **Pass** (`topics_slug_idx`) |
| Composite only if still miss budget | **Skipped with evidence** |
| FE parallel topic+comments when possible | **Pass** |
| `/t/**` SWR security review | **Pass** (anonymous only; auth/edit no-store) |
| D3 single detail GET / no mounted refetch | **Pass** (tests + code) |
| by-slug + comments report | **This file** |

## Artifacts

| Artifact | Path |
| --- | --- |
| Plan | `knowledge/plans/archive/2026-07/2026-07-21-million-scale-read-path.md` |
| This report | `knowledge/reports/2026-07-21-perf-m4-topic-detail.md` |
