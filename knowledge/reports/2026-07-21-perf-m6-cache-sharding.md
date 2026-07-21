# 2026-07-21 Perf After M6 — Cache Sharding + COUNT Cleanup

Status: **M6 measured** against the same dedicated DB class as M0–M5.

Task book: `knowledge/plans/2026-07-21-million-scale-read-path.md` (M6).  
Prior: `perf-m5-keyset.md`.

## Environment

| Item | Value |
| --- | --- |
| Host | macOS, x86_64 |
| Postgres | Docker `postgres:17-alpine` |
| Perf DB | `sforum_perf` on `:15432` — **1e6** topics, **50k** comments on `perf-hot-thread` |
| API under test | Second process on **:8082** → `sforum_perf`, `EMBED_WORKER_IN_API=true`, `SFORUM_SAFE_MODE=1` |
| Redis | Compose `sforum-redis-1` (`127.0.0.1:16379`) |
| k6 | Not required; LIGHT-class sequential + concurrent Python + unit tests |

## Code under test (M6)

1. **Topics list gen sharding:**  
   - `forum:gen:topics:global` — unfiltered home  
   - `forum:gen:topics:cat:{slug}` — category filter  
   - `forum:gen:topics:tag:{slug}` — tag filter  
   - cat+tag dual filter embeds both gens in the cache key  
2. **Write path:** bump only global + affected cat/tag scopes (not all categories).  
   Comment / lifecycle paths resolve scopes from detail cache (or one `GetTopic` fallback).  
3. **COUNT audit:** public ListTopics still D1 (no full-table COUNT); flat comments use `comment_count`; tree root COUNT only on ListComments cache miss; author rate-limit COUNTs are write-path only.  
4. **Tests:** cat A write does not miss cat B list cache; tag-x write does not miss tag-y; comment in cat A leaves cat B warm.

## Results

### A) Correctness — scoped invalidation (unit)

| Check | Result |
| --- | --- |
| `TestCachedStoreTopicsScopedInvalidationByCategory` | **Pass** — cat B list store calls unchanged after cat A `CreateTopic` |
| `TestCachedStoreTopicsScopedInvalidationByTag` | **Pass** |
| `TestCachedStoreCommentWriteInvalidatesOnlyTopicCategory` | **Pass** |
| Home list after any topic write | **Miss** (global gen bumped — expected) |
| Package `go test ./app/Models/Forum/` | **Pass** |

### B) Redis key shape (live API on sforum_perf)

After warm multi-cat reads, list keys are scope-prefixed (gen value `0` until first write bump):

```text
forum:topics:g0:::latest:1:20:                 # home
forum:topics:c0:general::latest:1:20:          # category general
forum:topics:c0:perf-cat-01::latest:1:20:      # category perf-cat-01
forum:topics:c0:perf-cat-02::latest:1:20:      # category perf-cat-02
```

Pre-M6 single gen would have been `forum:topics:{n}:…` shared across all filters; one bump invalidated every category.

### C) API latency (LIGHT-class, warm Redis)

Sequential n=30 after warm-up; concurrent 5 workers × 200 mixed home/cat reads.

| Scenario | p50 | p99 | Notes |
| --- | --- | --- | --- |
| Home p1 warm | **5.4** ms | **11.5** ms | global gen scope |
| Category `general` p1 warm | **6.0** ms | **8.6** ms | cat gen scope |
| Category `perf-cat-01` p1 warm | **7.0** ms | **9.5** ms | independent of general |
| Category `perf-cat-02` p1 warm | **7.4** ms | **18.6** ms | independent shard |
| Topic by-slug warm | **6.7** ms | **10.9** ms | detail path unchanged |
| Concurrent multi-cat 200 ok | **13.9** ms | **25.1** ms | 0 errors |

**Before M6 (M5 warm category cursor re-hit):** p50 ~5.6 ms / p99 ~8.5 ms — same class.  
**After M6:** multi-category warm lists stay second-class; sharding does not add measurable overhead on read path (one extra Redis GET per scope, already needed for gen).

### D) COUNT leftover matrix (public hot path)

| Path | COUNT(*)? | Disposition |
| --- | --- | --- |
| ListTopics home/cat/tag | **No** | D1 denormalized / sum |
| ListComments flat public | **No** | `topics.comment_count` |
| ListComments tree total | Root `count(*)` on **cache miss only** | Kept; topic-scoped + CachedStore |
| ListComments hasMoreChildren | Per-page root group count | Necessary; not full-table |
| Author daily/cooldown | COUNT since | Write path only — kept |
| IncludeDeleted comment total | COUNT | Admin/author rare — kept |

No new public full-table `COUNT(*)` removed in this milestone because M1/M3 already cleared the hot leftovers; M6 audit confirms and documents residuals.

## Exit criteria check

| Criterion | Result |
| --- | --- |
| CachedStore topics gen split by cat/tag/global | **Pass** |
| Write bumps only affected scopes | **Pass** (unit) |
| Cat A write does not miss cat B list cache | **Pass** (unit) |
| Public hot path COUNT audit | **Pass** (matrix above) |
| Mixed multi-cat warm lists stay fast | **Pass** (LIGHT report) |

## Residual / next

- Home still invalidates on every public topic write (by design — activity feed).  
- Tree root COUNT remains on ListComments miss; optional later `root_comment_count` denorm if miss rate hurts.  
- **M7** is doc-only (read replica threshold) — no code in this plan unless single-node ceiling is proven.

## Artifacts

| Artifact | Path |
| --- | --- |
| Plan | `knowledge/plans/2026-07-21-million-scale-read-path.md` |
| This report | `knowledge/reports/2026-07-21-perf-m6-cache-sharding.md` |
| Code | `apps/api/app/Models/Forum/cached_store.go` (+ tests) |
