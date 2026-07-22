# Decision: Read Replica Thresholds And API Horizontal Scale

## Status

Accepted (2026-07-21) — **documentation only** (million-scale plan M7).  
No read-replica routing, no dual `DATABASE_URL` reader pool, and no multi-primary
code ships under this decision until the thresholds below are met **and** a
follow-up implementation plan is opened.

## Context

The million-scale read-path task book
(`plans/archive/2026-07/2026-07-21-million-scale-read-path.md`) completed **M0–M6** on a single
Postgres primary + shared Redis:

| Milestone | Outcome (summary) |
| --- | --- |
| M0 | 1e6-topic seed + `tests/perf` harness on dedicated `sforum_perf` |
| M1 | ListTopics slim + D1 totals (no public full-table COUNT) |
| M2 | View Redis INCR + flush; `hot_score` sort |
| M3 | Tree descendant cap + ListComments cache |
| M4 | Detail dual-write cache; no composite page cache |
| M5 | Keyset `after` / `hasMore` / `nextCursor` |
| M6 | Topics list gen shard global/cat/tag |

Warm public list/detail p99 on the measured Docker-class host is **second-class
tens of milliseconds** under LIGHT load; cold paths and concurrent pressure are
dominated by **Postgres CPU / shared memory / connection pool**, not by missing
application fan-out. Architecture audit Part D already deferred read replicas;
M7 records **when** to revisit and what must stay true for multi-API nodes.

## Decision

### 1. Default topology remains single-node Postgres + shared Redis

- One **primary** Postgres for all core reads and writes.
- One **shared Redis** for sessions, CachedStore gens, view counters, rate
  limits, and other host-owned keys.
- One or more **stateless API** processes (and optional standalone worker) that
  only hold ephemeral process memory (Fiber, plugin children, in-process
  registries rebuilt on boot).
- Nuxt SSR may scale separately; it must treat the API as the authority and must
  not assume sticky routing to a particular API instance for session validity.

**Do not** implement application-level read/write split until §3 thresholds fire.

### 2. API process is horizontally scalable (already assumed)

These properties are **host contracts**, not optional niceties:

| Concern | Assumption |
| --- | --- |
| Browser session | Redis-backed server session (`sforum_session`); any API node can load the same session id |
| Session revocation | Immediate via Redis / session store; no local-only session map |
| Forum list/detail cache | `CachedStore` keys + generation counters in **shared** Redis; multi-API writers must bump the same gen keys |
| View count | Redis INCR + River flush job; flush may run embedded or on worker, but delta keys are shared |
| Rate limits | Redis-backed when Redis is configured (multi-instance safe) |
| Local disk | Not required for correctness of public reads; attachment/local storage providers may bind a node — that is a **storage** concern, not session affinity |
| Plugin children | Process-local; enable/trust state is DB-backed; a new API node must re-attach/start plugins via host lifecycle, not sticky peer memory |
| Safe mode / migrations | Host-owned; one leader-style migrator at boot is enough; do not run divergent schema writers |

**Explicit non-goals for sticky sessions:** operators should not rely on LB
session stickiness to fix “wrong user” or “stale permission” bugs. If stickiness
is used, it is only an optimization for connection reuse.

### 3. When to add a Postgres read replica (metrics thresholds)

Open a **new** implementation plan (not silent code under M7) when **any** of
the following holds on a production-like primary for **≥ 7 days** (or a
controlled load test that reproduces the same shape):

| Signal | Threshold (starting recommendation) | Why |
| --- | --- | --- |
| Primary CPU | Sustained **> 70%** during peak read hours | Headroom for writes + vacuum |
| Read latency | Public warm list/detail **p99 > 2×** plan budget (e.g. home/category **> 100 ms** API p99) **while** Redis hit rate for list keys stays **> 80%** | Means cache is working but primary still overloaded on miss / uncached paths |
| Connections | `pg_stat_activity` near pool ceiling; API `MaxConns` wait / checkout errors | Scale-out reads before raising pool blindly |
| IO | Disk util / `IOPS` saturated; checkpoint / WAL write pressure with mostly-read traffic | Replica absorbs sequential read storms |
| QPS | Sustained public read QPS **> ~3–5×** what a single primary + warm Redis can hold after M1–M6 (operator-measured; not a fixed marketing number) | Capacity proof, not branding |

**Do not** add a replica solely because:

- Cold cache after deploy is slow (fix TTL / warm-up / longer page-1 TTL first).
- One category write storm invalidates home (expected; M6 already isolates other cats).
- Tree comment cold miss is heavy (payload + joins; cap/cache already in M3).
- Docker Compose `shm_size` is too small in dev (ops knob, not product replica).

### 4. If/when replica work starts (future plan constraints)

A future implementation plan must specify:

1. **Read-only DSN** (e.g. `DATABASE_READ_URL`) separate from primary `DATABASE_URL`.
2. **Which queries** may use the reader: public ListTopics / GetTopic /
   ListComments / taxonomy / search FTS candidates — **never** auth session
   mutations, moderation writes, River job claims, or gen-bumping write paths.
3. **Replication lag policy:** if lag > N seconds (suggest start at **2–5 s**),
   fall back to primary for that request class or serve slightly stale lists
   with documented UX (D1 already allows stale totals).
4. **No dual-write** application logic; logical replication / managed replica
   only.
5. **Worker** stays on primary (River + advisory locks).
6. Tests: allowed/denied routing matrix; lag inject; pool exhaustion on primary
   with reader still serving public GET.

### 5. Explicit no code under this decision / M7

- No `DATABASE_READ_URL` wiring in bootstrap.
- No query router, no sqlc dual pool, no “read replica” feature flag.
- No change to CachedStore gen keys beyond M6.
- M7 exit is **this note + plan/module handoff updates** only.

## Consequences

- Operators can run **N API replicas** today behind a load balancer as long as
  Redis + Postgres primary are shared and plugin/storage side effects are
  understood.
- Capacity work after M0–M6 should prefer: raise PG resources / `shm_size`,
  tune pools, keep Redis healthy, re-run `tests/perf` — **before** replica
  topology complexity.
- When thresholds trip, work is a **new plan**, not a silent follow-on PR to the
  million-scale task book.
- Multi-region active-active, PG partitioning, and cross-node theme build
  supervisors remain out of scope (see task book Out Of Scope).

## Evidence Links

- Plan: `knowledge/plans/archive/2026-07/2026-07-21-million-scale-read-path.md` (M0–M6 reports)
- Baseline: `knowledge/reports/2026-07-21-perf-baseline.md`
- After M6: `knowledge/reports/2026-07-21-perf-m6-cache-sharding.md`
- Sessions: `knowledge/decisions/2026-07-05-browser-session-jwt-strategy.md`
- Prior hardening: `knowledge/decisions/2026-07-08-performance-hardening.md`
- Audit deferral: `knowledge/archive/architecture-maturity-audit.md` Part D
