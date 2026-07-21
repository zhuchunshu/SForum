# 2026-07-21 M7 — Horizontal Scale (Doc Only)

Status: **decision recorded; no code**.

Task book: `knowledge/plans/2026-07-21-million-scale-read-path.md` (M7).  
Decision: `knowledge/decisions/2026-07-21-read-replica-and-api-horizontal-scale.md`.

## Summary

M0–M6 leave public read paths on a **single Postgres primary + shared Redis**
with measured warm list/detail latency in the second-class tens of milliseconds
(LIGHT class on Docker PG). M7 does **not** implement read replicas.

| Item | Outcome |
| --- | --- |
| API multi-instance | **Documented as supported** via Redis sessions + shared CachedStore gens |
| Read replica | **Deferred** until CPU / p99 / pool / IO thresholds in the decision |
| Code changes | **None** |
| New plan required | Yes, before any `DATABASE_READ_URL` or query router |

## Single-node ceiling (from M0–M6, not re-measured)

Replica is **not** justified by M0–M6 alone: residual pain is cache miss cost,
payload size (tree comments), and ops limits (Docker shm), not a proven need for
read/write split. Re-run `tests/perf` (prefer `LIGHT=1` on small shm) after
hardware upgrades before opening a replica implementation plan.

## Exit criteria check

| Criterion | Result |
| --- | --- |
| Decision note: when to add read replica | **Pass** (thresholds in decision §3) |
| Session/Redis shared; API stateless assumptions | **Pass** (decision §2) |
| Explicitly no code until single-node ceiling | **Pass** (decision §5; this report) |

## Artifacts

| Artifact | Path |
| --- | --- |
| Decision | `knowledge/decisions/2026-07-21-read-replica-and-api-horizontal-scale.md` |
| Plan | `knowledge/plans/2026-07-21-million-scale-read-path.md` |
| This report | `knowledge/reports/2026-07-21-perf-m7-horizontal-scale.md` |
