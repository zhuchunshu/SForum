# 2026-07-14 Trusted Plugin And Theme Platform V3 P4 Progress

## Status

- Overall V3: **27%**.
- P0-P3: **100%**.
- P4: **47%**, active (7 of 15 task/test rows complete).
- Branch: `main`; last implementation commit: `04c8b5d75`.

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
- Added real protocol-v2 execution for exact-artifact River plugin jobs. The
  worker resolves live package/trust state, cancels stale or incompatible rows,
  and rechecks both the running and startup-frozen Manifest before dispatch.
- Added deterministic execute/drain/migrate/cancel upgrade decisions and closed
  the old-version fail-closed execution matrix. A Host-owned exact-identity
  replacement ledger and transactional planner now preserve attempts, queue,
  priority, schedule, and tags without updating River's private args column.
- Added the crash-resumable coordinator for all six operations and eleven
  actions. Stable attempts inherit opaque checkpoints, persist every validated
  progress update, use a detached five-second terminal write window, and
  reconcile every terminal/recovery crash window without repeating a completed
  action.
- Added 34 real PostgreSQL crash/retry boundary scenarios and formally closed
  the P4 boundary-recovery test row.
- Added multi-node step lease persistence and repository CAS. Exact owner,
  monotonically increasing revision, database statement time, expiry takeover,
  heartbeat fencing, progress writes, and terminal lease clearing are covered
  against real PostgreSQL.
- Added an additive `LifecycleRequest.forced` field and connected the durable
  coordinator to protocol-v2. Forced authority stays distinct from `dry_run`
  and plugin-owned input; all eleven actions, live progress, result JSON, and
  typed remote failures cross the frozen subprocess transport.
- Added the independent `host.gate` lifecycle-step identity needed for Host
  safety gates to use lease fencing without masquerading as plugin code. Its
  additive migration preserves historical rows across rollback.
- Added the real PostgreSQL/River queued-job reconciliation adapter. The full
  extension-scoped River snapshot, exact migration ledger claim/link, public
  transactional replacement insert, and old-row cancellation share one
  database transaction; no River payload column is rewritten in place.
- Added lease ownership to coordinator execution for plugin actions, Host
  gates, and forced skips. Heartbeats, progress, success/failure/cancellation,
  takeover, Host terminal replay, detached writes, and lease-loss cancellation
  are fenced by exact revision.
- Added additive staged-version persistence and store-level active/candidate
  separation. Static upgrade replay reuses the immutable candidate row and
  cannot change type, active identity, or enabled status.
- Removed legacy static-upload side effects: no runtime stop, provider reset,
  trust revocation, status reset, executable hook, or migration-ledger write.
  Staged trust preview/challenge binds the candidate without invalidating the
  active artifact grant.
- Added the instance-bound runtime admission primitive with atomic drain
  closure, explicit lifecycle cleanup exemption, inflight wait, force cancel,
  and per-class residual counts.
- Added exact staged promotion/discard CAS and immutable active/candidate
  retention. Candidate id, digest, extension ownership, row locks, and one-winner
  concurrency fence stale or foreign writes while preserving the old version.
- Bound lifecycle authority snapshots to actor, exact live grant, TrustImpact,
  artifact digests, canonical action inputs, and a semantic SHA-256 request
  fingerprint.
- Added Manager exact-instance accounting and admission lookup. Active pointer
  switches, stale stop/drain/remove fencing, lifecycle cleanup admission, and
  residual counters are instance-bound; physical ProtocolStarter retention is
  still in progress and V1 remains a hard replacement.
- Updated OpenAPI and Nuxt management surfaces for inert candidates. Extension
  resources expose `stagedVersion` without database identity, upload results
  expose `activationPending`, and list/details/Toast copy accurately state that
  the current artifact continues running.

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
- Plugin-job focused repetition, relevant package tests, race, vet, full Go
  tests, and full Go build passed.
- Coordinator focused tests passed with `-count=100`; all 34 real PostgreSQL
  crash windows passed repeatedly and under race detection.
- Step lease concurrent-claim/takeover/heartbeat/progress/completion tests
  passed repeatedly, under race detection, and in the full Extensions package.
- Protocol generation was reproducible; Buf lint/breaking checks, SDK tests,
  runtime-adapter repeated/subprocess/race tests, and full API tests passed.
- The plugin-job ledger passed an isolated PostgreSQL `Up -> Down -> Up` cycle.
- The Host-gate migration passed focused migration and embedded migrator
  rollback/forward tests.
- PostgreSQL/River reconciliation passed atomic rollback, retry, migration
  identity, extension scoping, and real River-row integration coverage.
- Coordinator lease execution passed focused repetition, full Models tests,
  race, vet, real PostgreSQL fencing, and full API tests.
- Staged version migration passed full migration/migrator tests against real
  PostgreSQL; store staging, trust targeting, and inert Service behavior passed
  focused repetition and race coverage plus the full Models package.
- Runtime admission passed focused repetition/race, full Support/Extensions,
  and vet.
- Exact staged promotion/discard passed real PostgreSQL concurrent CAS,
  rollback-on-write-failure, migration/migrator, Models race, vet, and full API
  tests.
- Manager exact-instance tests and real protocol-v2 handshake passed focused
  tests and race detection.
- OpenAPI references, staged management contract validation, bilingual JSON,
  Nuxt typecheck, and Nuxt production build passed.

## Active Ownership

- V2-only retained physical runtime handles and exact stop/discard are in
  flight under `Support/Extensions`.
- Initial/final Host-gate topology, exact source/target request binding, and
  durable Host result/checkpoint propagation are in flight under
  `Models/Extensions` lifecycle coordinator files.
- Historical exact-version lookup and rollback/upgrade CAS strengthening are in
  flight under `Models/Extensions` version repository files.

## Next

1. Land retained ProtocolStarter instances, corrected Host-gate paths, and exact
   historical-version transaction primitives as independent commits.
2. Wire the lifecycle state machine and first-trusted-enable transaction to the
   durable ledger, exact-artifact trust, frozen runtime, drain, audit, and
   recovery contracts.
3. Wire route/job/provider/schedule admission barriers, then implement upgrade,
   rollback, uninstall removal modes, recovery HTTP/UI, and the complete P3/P4
   repository gate.

## Open Questions

- None. The accepted V3 ADR defines the current product boundary.
