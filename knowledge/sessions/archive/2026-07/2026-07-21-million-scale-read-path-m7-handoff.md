# 2026-07-21 Session Handoff — Million-scale M7 (close-out)

## Changed

- **Decision:** `decisions/2026-07-21-read-replica-and-api-horizontal-scale.md`
  — when to add a PG read replica; API multi-instance / shared Redis assumptions;
  **no code** under M7.
- **Report:** `reports/2026-07-21-perf-m7-horizontal-scale.md`
- **Plan:** M0–M7 complete; status **completed**.

## Decisions

- Single primary + shared Redis remains default after M0–M6 evidence.
- N API processes are OK if sessions and CachedStore gens stay on shared Redis.
- Replica work needs a **new** plan when metrics thresholds fire.

## Next

- None for this task book. Optional later: production soak with full k6, or a
  separate replica implementation plan if thresholds are met.

## Open Questions

- None for M7.
