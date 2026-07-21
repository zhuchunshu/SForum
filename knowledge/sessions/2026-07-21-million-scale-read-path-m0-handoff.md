# 2026-07-21 Session Handoff — Million-scale M0 complete

## Changed

- **M0 shipped (tooling only, no M1+ forum read-path rewrites):**
  - `apps/api/cmd/sforum`: `seed:forum --profile=perf-1m` + `seed:perf` alias;
    pure plan (`seed_plan.go`), bulk write (`seed_bulk.go`), dry-run +
    `--confirm-perf-db` isolation, unit tests for 1e6 plan shape.
  - `tests/perf/`: k6 scenarios (home/category/slug/comments/mixed/view flood),
    README, `run-all.sh`; prefer `LIGHT=1` on Compose PG (shm 64m).
  - Baseline: `knowledge/reports/2026-07-21-perf-baseline.md`
- Full seed on dedicated DB `sforum_perf`: **1e6 topics**, **50k** comments on
  `perf-hot-thread`, ~2.1 GiB, ~5.5 min bulk.

## Decisions

- D1–D4 unchanged (still law).
- Default regular comments on perf-1m = **0** (hot thread carries 50k); override
  with `--comments-max` if needed.
- Aggressive 50 VU k6 against Docker PG is unsafe until M1; document `LIGHT=1`.

## Next

- **M1** ListTopics slim select, kill list ILIKE risk, D1 totals (cat/tag
  counters; home approximate / no public full COUNT).
- Re-run `tests/perf` with `LIGHT=1` after M1 for before/after in a new report.

## Open Questions

- None for D1–D4. Residual: home “约 N” UI (M1/M5), pin+keyset (M5).
