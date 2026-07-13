# 2026-07-14 Trusted Plugin And Theme Platform V3 P4 Progress

## Status

- Overall V3: **25%**.
- P0-P3: **100%**.
- P4: **27%**, active (4 of 15 task/test rows complete).
- Branch: `main`; last implementation commit: `a3c4f75dc`.

## Changed

- Closed the remaining P3 runtime streaming, service discovery, compatibility
  matrix, and v1 built-in package rows.
- Added activation dependency preflight before runtime start for required,
  optional, conflict, and provides relationships.
- Added typed Host lifecycle invocation for all eleven P4 actions over the real
  protocol-v2 subprocess transport while keeping protocol v1 explicit.
- Frozen lifecycle declarations are deep-cloned at runtime Start and again at
  client construction. A stale or forged caller manifest cannot authorize an
  undeclared action or alter the frozen plan/progress/checkpoint contracts.
- Progress streams validate exact step and runtime identity, monotonic counters,
  terminal state, typed failure/cancellation, and the declared result schema.
  The checkpoint schema association is returned, but the current wire
  checkpoint remains an opaque string and is not structurally validated.
- Added the Host-owned lifecycle state machine with the exact ten authoritative
  states, six operations, eleven actions, recommended safety gates, terminal
  rules, uninstall-only force escalation, and failed/cancelled recovery.
- Added an additive PostgreSQL operation/step ledger plus repository. Exact
  artifact and authority snapshots, idempotency fingerprints, stable step ids,
  checkpoints, progress, typed failure, retries, removal modes, actor/audit
  snapshots, and uninstall history survive process and extension deletion.

## Verification

- `go test ./app/Support/Extensions -count=1` passed.
- Focused real-subprocess lifecycle tests passed with `-count=10`.
- Focused lifecycle tests passed with `-race`.
- `go vet ./app/Support/Extensions` passed.
- `git diff --check` passed for the lifecycle slice.
- Real PostgreSQL migration Down/Up and migration/migrator tests passed.
- Repository concurrent acquire/CAS/stable-step, restart, resume, retry, audit
  retention, and extension-deletion history tests passed repeatedly and with
  race detection.

## Active Ownership

- Versioned plugin-job execution and drain compatibility are an independent
  in-flight slice sharing selected runtime files.
- A crash-resumable coordinator is in flight in independent new files.
- Exhaustive durable crash/retry boundary coverage is in flight in an
  independent integration test file.

## Next

1. Review, test, and commit the job, coordinator, and boundary-recovery slices.
2. Wire the lifecycle state machine and first-trusted-enable transaction to the
   durable ledger, exact-artifact trust, frozen runtime, drain, audit, and
   recovery contracts.
3. Run the complete P3/P4 repository gate and record the result.

## Open Questions

- None. The accepted V3 ADR defines the current product boundary.
