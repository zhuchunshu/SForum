# 2026-07-14 Trusted Plugin And Theme Platform V3 P4 Progress

## Status

- Overall V3: **27%**.
- P0-P3: **100%**.
- P4: **47%**, active (7 of 15 task/test rows complete).
- Branch: `main`; last committed slice: `74fd5f367`.

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
- Protocol V2 now owns real staged, published, and retained physical processes.
  Exact lifecycle calls work on candidates and retained rollback instances;
  readiness is deferred until after lifecycle preparation, stale stop cannot
  unregister a replacement, and V1 retains hard-replacement semantics.
- Plugin job enqueue now acquires the active exact Manager instance and holds a
  `RuntimeCallJob` lease through the River insert. Stale/draining callers fail
  closed, force-drain cancellation stays distinct from caller cancellation,
  and bootstrap binds the production adapter.
- Added exact-runtime schedule publish/acquire/drain/wait admission with atomic
  failed-activation compensation and retained rollback. No production schedule
  trigger owner exists yet, so this prerequisite does not close the drain row.
- Bound each Service Registry winner to its exact Manager runtime identity.
  Unary provider invocation and the full bidirectional stream hold a
  `RuntimeCallService` lease, use its cancellation context, distinguish caller
  cancellation from forced drain, and fail closed on stale/draining winners
  without selecting a fallback provider.
- Added Manager-owned staged runtime orchestration over ProtocolStarter:
  candidate start, exact health/readiness, old-instance drain/wait, publication,
  retained stop, and unpublished discard. Both source and target are fenced
  during publication, the old instance must be idle, failed transitions are
  compensated, and retained instances can be republished for rollback.
- Added the exact `LifecycleCoordinatorRuntime` adapter. It validates canonical
  step identity, operation role, exact source/target artifact bindings, frozen
  authority, plan version, removal mode, and uninstall-only force before
  acquiring a Manager lifecycle-cleanup lease and dispatching to the persisted
  `InstanceID`; stale identities never fall back to the active process.
- Corrected coordinator recovery ordering, database-time lease claims,
  multi-role revalidation, canonical Host markers, and side-effectful skipped
  gates; Host gates now receive independently reconstructed durable action
  results rather than overloading their previous Host result.
- Added the production exact Host dispatcher for all 32 gate positions across
  six operations. Every returned runtime/admission snapshot is rechecked for
  exact identity, artifact, health, and readiness; missing composed boundaries
  fail closed.
- Added durable audit identities, scoped lifecycle history reads, and safe
  inspection DTOs that exclude authority, checkpoints, raw action documents,
  error metadata, and lease ownership.
- Added P6 immutable API-route snapshot foundations with revision CAS,
  deterministic matching, exact runtime instance bindings, conflict inspection,
  Safe Mode, and inherited-core-guard declarations. This is not production
  route execution and does not close a P6 row yet.
- Added the P8 bounded contextual Theme Compiler with immutable compiled/runtime
  identities, tokenizer-backed static HTML checks, passive ViewModels, standard
  template control actions, restricted helpers, and zero hot-path filesystem
  access. It is not connected to Page runtime publication and does not close a
  P8 row yet.

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
- Protocol V2 exact-instance focused/repeated/race tests and vet passed,
  including staged readiness, retained lifecycle, rollback, stale stop, and
  V1/V2 transition protection.
- HostAPI, Jobs, Extensions, and bootstrap focused tests passed after job and
  schedule admission; HostAPI/Jobs/bootstrap race and all four package vet
  gates passed.
- Exact service admission passed unary and bidirectional-stream lease coverage,
  stale-winner/no-fallback checks, forced-drain cancellation, bootstrap adapter
  tests, focused package tests, race detection, and vet.
- Manager staged publication passed focused and repeated tests, race detection,
  vet, and real protocol-v2 subprocess coverage for healthy publication,
  transition compensation, retained rollback, exact stop, and discard.
- Exact lifecycle runtime execution passed every action/role mapping,
  cross-version source cleanup, drift rejection, typed cancellation, lease
  fencing, active/retained instance tests, repeated/race/vet gates, and real
  subprocess verification that stale identities cannot reach the active runtime.

## Active Ownership

- Composed lifecycle publication/compensation is in flight under
  `app/Support/Extensions/lifecycle_composed_boundary*.go`; it must prove exact
  reverse compensation for runtime, database, and registry failure windows.
- Lifecycle inspection controller/OpenAPI work is in flight. Service DTOs are
  committed, but the HTTP contract remains uncommitted until leak and ref gates
  pass.
- Bootstrap construction, first trusted enable, disable/upgrade/rollback,
  uninstall removal modes, recovery mutations/UI, and exact route/theme
  publication remain open.

## Next

1. Land composed publication/compensation and lifecycle inspection HTTP after
   their focused gates pass.
2. Construct the real repository/runtime/Host/coordinator in bootstrap.
3. Wire the lifecycle state machine and first-trusted-enable transaction to the
   durable ledger, exact-artifact trust, frozen runtime, drain, audit, and
   recovery contracts.
4. Implement upgrade, rollback, uninstall removal modes, recovery HTTP/UI, and
   the complete P3/P4 repository gate.

## Open Questions

- None. The accepted V3 ADR defines the current product boundary.

## Inspection, Catalog, SafeHTML, And Boundary Audit Checkpoint

- Committed lifecycle inspection HTTP in `74c13e64f` and its OpenAPI/stable
  route catalog contract in `ce30b306c`. Focused controller tests, race, vet,
  1,636 OpenAPI references, and the 209-route/115-UI/99-row P0 validator passed.
- Committed `2fe465eea`, which makes the reviewed 209-route catalog available as
  immutable caller-owned Go data generated by the existing P0 drift gate.
- Committed `9100b078c`, which adds Host-produced `SafeHTML`, an explicit
  `safeHTML` template helper, forged-value rejection, contextual URL/attribute
  tests, and compiler cache invalidation through version `@2`. Normal, race,
  and vet gates passed.
- P4 remains **47% (7/15)** and overall remains **27%**. These commits close
  prerequisites, not the production lifecycle or Page/Route publication rows.
- The composed boundary is still uncommitted. Mandatory audit expanded it to
  cover drained target publication, jobs/schedules/route admission at the
  source drain gate, a durable exact-operation publication journal, and
  terminal-after-commit uninstall finalization. Process-local reverse closures
  are insufficient after a crash and must not be presented as atomic recovery.
- Recovery repository/coordinator wiring is also uncommitted. It must preserve
  the original authority actor and audit while each recovery attempt uses and
  persists its own actor/audit pair and forced-escalation reason.
- Next remains: land those two audited slices, add the durable publication
  journal/finalizer migration if required, then construct the real lifecycle
  stack in bootstrap and route Service mutations through it.
