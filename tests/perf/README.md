# SForum read-path load tests (M0)

k6 scenarios for the million-scale public read path. Pair with:

- Compose PostgreSQL + Redis (`./scripts/dev.sh`)
- Dedicated perf database seeded via `cmd/sforum` (`seed:perf` / `seed:forum --profile=perf-1m`)
- API process on current `main` (`./scripts/api-dev.sh` or a second process pointed at the perf DB)

**Do not** run full `perf-1m` against a casual shared dev database.

## Prerequisites

1. Install [k6](https://k6.io/docs/get-started/installation/) (network installs need the Agents.md proxy in mainland China environments):

   ```bash
   export https_proxy=http://127.0.0.1:7897 http_proxy=http://127.0.0.1:7897 all_proxy=socks5://127.0.0.1:7897
   # example: brew install k6   or download a release binary
   k6 version
   ```

2. Seed a **dedicated** database (example creates `sforum_perf` on the Compose Postgres):

   ```bash
   set -a; . ./.env; set +a
   # create DB once
   docker exec sforum-postgres-1 psql -U sforum -d postgres -c "CREATE DATABASE sforum_perf;"
   export DATABASE_URL='postgres://sforum:sforum@127.0.0.1:15432/sforum_perf?sslmode=disable'
   cd apps/api && go run ./cmd/migrate
   go run ./cmd/sforum seed:perf --confirm-perf-db --database-url="$DATABASE_URL"
   # dry-run only:
   go run ./cmd/sforum seed:forum --profile=perf-1m --dry-run
   ```

   Order of magnitude: ~5–15 GiB disk, ~10–60 minutes for full 1e6 topics + 5e4 hot comments (bulk path).

3. Point an API at that `DATABASE_URL` (and Redis). Example second process on port 8082:

   ```bash
   set -a; . ./.env; set +a
   export DATABASE_URL='postgres://sforum:sforum@127.0.0.1:15432/sforum_perf?sslmode=disable'
   export HTTP_PORT=8082
   cd apps/api && go run ./cmd/api
   ```

## Running scenarios

All scripts read:

| Env | Default | Meaning |
| --- | --- | --- |
| `BASE_URL` | `http://127.0.0.1:8081` | API origin (include host only; paths use `/api/v1/...`) |
| `HOT_SLUG` | `perf-hot-thread` | Hot topic slug from seed |
| `CATEGORY_SLUG` | `general` | Category with large topic share |

Warm-up + measure phases are encoded in each script (`stages` or `setup` + scenario).

```bash
export BASE_URL=http://127.0.0.1:8082

# Prefer LIGHT=1 against Compose Postgres (default shm 64m). Full 50 VU stages can
# exhaust PG shared memory on ListTopics COUNT/sort and take the API down.
export LIGHT=1

# Single scenarios
k6 run tests/perf/home_topics.js
k6 run tests/perf/category_topics.js
k6 run tests/perf/topic_by_slug.js
k6 run tests/perf/comments_flat.js
k6 run tests/perf/comments_tree.js
k6 run tests/perf/mixed_read_write.js
k6 run tests/perf/view_flood.js
# M5 deep scroll: keyset `after` vs deep OFFSET `page` (category list)
MODE=both DEEP_STEPS=25 k6 run tests/perf/deep_scroll.js

# Suite wrapper (sequential)
LIGHT=1 ./tests/perf/run-all.sh
```

Each run prints p50/p95/p99, error rate, and throughput (k6 summary). Capture stdout into `knowledge/reports/` when recording a baseline.

## Scenarios vs success metrics

| Script | Maps to plan Success Metrics |
| --- | --- |
| `home_topics.js` | `GET` topics home p1 |
| `category_topics.js` | `GET` topics category p1 |
| `topic_by_slug.js` | `GET` topic by slug |
| `comments_flat.js` | comments flat p1 (50k thread) |
| `comments_tree.js` | comments tree p1 (bounded body) |
| `mixed_read_write.js` | 90% read / 10% write (write needs auth cookie; skips write if `AUTH_COOKIE` unset) |
| `view_flood.js` | view flood on one topic (baseline records per-request PG updates until M2) |

## Notes

- Scripts are dumb HTTP clients only — no business logic.
- M0 baseline is measurement on **current main** before M1 ListTopics rewrites.
- Targets in the task book (e.g. p99 ≤ 50ms) are starting recommendations; hardware differs.
