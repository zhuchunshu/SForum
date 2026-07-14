# Trusted Plugin And Theme Platform V3 Progress Ledger

Date: 2026-07-15
Overall progress: **48%**
Active phase: **P6 - Full Route And Middleware Registry V1 (61%, 11 of 18 rows)**

This ledger is the durable percentage and context-compaction checkpoint for the
V3 program. Update it before context compression, at every phase boundary, and
after any commit that materially changes the completion calculation.

## Percentage Model

Progress is weighted by implementation risk and verified phase exits. A phase
may contribute a fraction of its weight while work is in progress, but docs,
scaffolding, or demo-only code cannot satisfy a runtime exit criterion.

| Phase | Weight | Current completion | Earned |
| --- | ---: | ---: | ---: |
| P0 Governance | 3% | 100% | 3% |
| P1 Trust/recovery | 6% | 100% | 6% |
| P2 Manifest/contracts | 7% | 100% | 7% |
| P3 Host API v2 | 8% | 100% | 8% |
| P4 Lifecycle/dependencies | 7% | 100% | 7% |
| P5 Database/commands | 8% | 65% | 5.18% |
| P6 Routes/middleware | 10% | 61% | 6.11% |
| P7 Workflow/admin/query/identity | 10% | 18% | 1.82% |
| P8 Theme compiler/runtime | 8% | 50% | 4% |
| P9 Components/assets/L2 | 8% | 0% | 0% |
| P10 Content/media/data | 8% | 0% | 0% |
| P11 Platform services | 6% | 0% | 0% |
| P12 Operations/ecosystem | 6% | 0% | 0% |
| P13 References/removal/final gates | 5% | 0% | 0% |

Displayed overall progress is the floor of earned weighted progress until the
program reaches 100% and every final gate passes.

## Completed P0 Evidence

- Read all required authority, module notes, and latest handoffs.
- Confirmed clean `main` baseline at `d72c9ac2c` before V3 edits.
- Generated the exact 27-theme + 72-plugin traceability matrix.
- Inventoried 204 API registrations and 113 Nuxt page/component surfaces with
  stable IDs and contract versions.
- Inventoried events, contribution points, provider slots, schedules, job kinds,
  cache, content/data, and 33 admin pages.
- Published the eleven-family Extension Surface Matrix for current core modules.
- Froze namespaces, migration flags, raw DB/custom guard policy, and rollback
  rules.
- Added reproducible v1 enable/theme-resolve/route-proxy/plugin-RPC benchmarks.
- Linked superseded historical decisions to the active V3 target.
- Passed catalog drift, focused Go tests/benchmarks, OpenAPI validation, Nuxt
  typecheck/build, and the full repository gate.
- Closed every P0 task and verification checkbox without changing production
  behavior.

## Compression Checkpoint Protocol

Before expected context compression:

1. Update this file's percentage, evidence, last commit, dirty files, and next
   command.
2. Update the active session handoff under `knowledge/sessions/`.
3. Run the focused tests for the current slice.
4. Inspect `git diff` and `git diff --cached`; stage only V3-owned hunks.
5. Commit every coherent buildable slice. If a slice cannot be committed,
   record the exact dirty files and why.
6. Resume from the recorded next command, not from conversation recollection.

Context usage must be monitored throughout the long-running goal. Do not wait
for an automatic compression warning when a coherent slice can be checkpointed:
write the current evidence and exact resume point here, update the session
handoff, run the focused gate, and commit the checkpoint first. Every progress
update to the user must include the displayed overall percentage and active
phase percentage.

## Completed P1 Evidence

- Added additive trust-recovery persistence, including one-use challenge
  digests, durable exact-artifact grants, revocation, and startup-attempt state.
- Added a default-off exact-artifact trust service. Challenge tokens are stored
  as SHA-256 hashes, returned in plaintext once, bound to the actor and complete
  impact document, and expire after at most five minutes.
- Exact identity covers package, backend, admin frontend, and migration
  digests. A granted unchanged digest survives restart without another prompt.
- Added super-admin trust inspection, challenge, and revoke HTTP contracts;
  delegated extension managers may store inert packages but cannot authorize
  execution.
- Added Host-owned `SFORUM_SAFE_MODE=1` before plugin startup. It blocks plugin
  lifecycle/settings mutations, routes, contributions, navigation, providers,
  Host API capabilities, trusted frontend assets, and non-default theme
  resolution while keeping health and recovery available.
- Added PostgreSQL-only `sforum extension list`, `disable`, and `disable-all`
  recovery commands. They do not load packages, boot HTTP/Nuxt, or initialize
  plugin runtime code.
- Added startup-attempt containment: a failed, incomplete `starting`, or
  `skipped` attempt for the same digest is skipped on the next boot; a manual
  enable or changed digest may retry.
- Unified backend and admin-frontend execution under the whole-artifact grant;
  legacy frontend-only grants cannot bypass V3 exact-artifact checks.
- Added a shared admin impact dialog with every canonical impact category,
  persistent blocking errors, delegated preview-only behavior, a two-step
  challenge/enable flow, and a theme-aware 10-second success Toast.
- Covered package/backend bytes, migration bytes/declarations, admin frontend
  bytes/contracts, routes, permissions, features, authority, Host/Frontend
  contracts, and dependencies in the trust invalidation matrix.
- Covered wrong actor, missing, expired, replayed, and stale challenges through
  the HTTP boundary, plus trust audit and delegated static-preview behavior.
- Verified PostgreSQL challenge concurrency produces one grant and one replay;
  isolated Safe Mode boot left the executable-plugin sentinel absent; malformed
  package recovery disabled the extension; two isolated boots executed a
  failing plugin exactly once (`failed`, then `skipped`).

## Last Durable Checkpoint

### 2026-07-14 P4 Convergence Checkpoint

- Overall remains **27%** and P4 remains **47% (7 of 15)**. The underlying P4
  implementation is substantially ahead of the accepted rows, but no row is
  promoted until the production bootstrap, exact migration engine, cleanup
  finalizer, recovery UI, and repository gates converge.
- `ee5a58fc1`, `41b8dc7b0`, `b9b52caf1`, `c2dd53941`, `af32d68cb`, and
  `af2408f0c` complete authoritative uninstall coordination, the PostgreSQL
  production fence, exact River reconciliation, immutable recovery authority,
  sanitized history failures, and target-admission ordering across the shared
  publication marker.
- `38b84f538` and `9ade60966` add the Database Registry ledger and scoped
  schema/role credentials. `cfa416e7c` synchronizes the Goose parser dependency
  for all three built-in protocol-v1 backend compatibility modules.
- `0cd41344f` exposes V2 uninstall results and retry/skip/forced recovery over
  authenticated HTTP/OpenAPI. Focused Controller, Models, Localization,
  OpenAPI-ref, and staged-contract gates passed.
- `b707fa47f` adds exact-artifact migration source/target proof and static
  preflight. Focused, race, and six real-PostgreSQL scenarios passed.
- Active dirty ownership is isolated: exact migration engine and role-integrity
  work under `app/Support/Extensions/extension_database_migration_*`; production
  assembly under `apps/api/bootstrap/{app.go,extension_lifecycle*}`; and
  management recovery/removal UI under `apps/web`. User-owned `.reasonix`,
  `.zcode`, and `CLAUDE.md` deletions remain untouched.
- Release blockers: migration-role pre-existing membership/settings/ownership
  rejection; failed advisory-unlock connection isolation; non-nil production
  migration engine; exact cleanup finalizer/purger with durable database
  disposition; full bootstrap/service binding; three removal modes plus
  retry/skip/forced UI; real PostgreSQL/race/vet/OpenAPI/Nuxt/browser/full-repo
  gates.
- Exact resume order: land the exact migration engine; land additive database
  disposition migration and API; land bootstrap and cleanup finalizer/purger;
  land the management UI; run the complete P4 gate; then update the 15-row
  ledger and begin P5 without reimplementing already-landed Database Registry
  prerequisites.

### 2026-07-14 P4 Registry And Service Production-Path Checkpoint

- Overall remains **27%** and P4 remains **47% (7 of 15)**. The newly landed
  Registry and Service slices are production prerequisites, but Jobs,
  migrations, bootstrap, uninstall/recovery, and the complete P4 gates have not
  yet converged as one production path.
- `8163f1673`, `02771edc2`, `687d84905`, `efa053d33`, and `acec487f8` bind Page,
  Route, Hook, and Service snapshots to immutable revisions and exact Manager
  runtime admission. A drained or staged target stays invisible, stale Hook or
  Service teardown cannot remove its replacement, and caller-owned inspection
  snapshots cannot mutate live Registry state.
- `392344a17` and `7ed4e15c7` persist the aggregate Registry publication phase
  and deterministically reconcile Page, Route foundation, Hook, and Service
  families behind the shared lifecycle publication fence. The failure test
  covers Route target publication followed by a conflicting Page snapshot;
  ordinary target calls remain closed and frozen-source recovery is required.
- `8dfb53009` and `be73a5eb5` prove and implement idempotent Protocol V2 service
  republication from the startup-frozen handshake after compensation removed
  the process-local Service set.
- `064b24f4b`, `548a4826e`, and `ccaa03c08` route trusted install/enable,
  disable, staged upgrade, and exact historical rollback through the durable
  coordinator with actor-bound stable idempotency keys, static preflight before
  position zero, retained authority for deactivation/rollback, HTTP/OpenAPI
  contracts, and admin client key propagation. A retry after state publication
  resolves the original operation before consulting mutated extension state.
- Clean `git archive HEAD` tests passed Models/Extensions, Pages, Routes,
  HostAPI, and Support/Extensions. Registry focused repetition, race, vet, and
  a real PostgreSQL Registry publication test also passed.
- Active dirty ownership is isolated: Jobs/shared-fence plus composed-boundary
  ordering; preflight/migration engine plus additive migration `013` Database
  Registry; bootstrap production assembly; and Service uninstall/recovery. The
  user-owned `.reasonix`, `.zcode`, and `CLAUDE.md` deletions remain untouched.
- Next: land the Jobs fence and post-marker reconciliation, migration `013`
  independently before its P5 implementation, then migration/preflight and
  bootstrap adapters. Finish uninstall preserve/export/remove, forced recovery,
  retry/skip UI, and denied/allowed tests before recalculating P4 progress.

### 2026-07-14 P4 Cleanup And Exact State Publication Checkpoint

- Overall remains **27%** and P4 remains **47% (7 of 15)**. The production
  cleanup and extension-state delegates are complete, but jobs/schedules,
  aggregate registries, migration proof, bootstrap construction, and Service
  routing are not yet one verified production path.
- `72806f0ad`, `cfdf2b246`, and `83162467b` add the retained cleanup schema,
  exact-artifact staging/finalization contracts, and real PostgreSQL coverage.
  Disable and retired-source recovery records remain retained. Uninstall only
  stages a durable tombstone; physical identity/package/data purge requires a
  terminally succeeded operation plus an idempotent exact receipt. Evidence
  retention is distinct from physical resource presence, and both purge/commit
  crash windows converge without inventing success.
- `341b38f36`, `afa7ea044`, and `71a64dfc8` add the durable extension-state
  publication intent, PostgreSQL transaction, and composed-boundary adapter.
  All six operations use operation-first locking, immutable source/target
  vectors, exact version CAS, marker-controlled restore, and restart-safe
  inspection. Upgrade restores its staged candidate; rollback preserves an
  unrelated staged pointer; a committed publication can never restore source.
- `da13783f4` updates the coordinator lease test to count every canonical Host
  gate. `1e71190f9` makes first trusted install promote a newer never-executed
  staged candidate atomically while retaining the old inert active/staged
  vector for compensation.
- Clean `git archive HEAD` plus staged-patch gates passed full Models and
  Support/Extensions tests, focused repetition, real PostgreSQL, race, and vet.
  Migration `009` passed isolated `Up -> evidence-protected Down refusal ->
  empty Down -> Up` against PostgreSQL.
- Three parallel uncommitted slices are active and isolated by ownership:
  `010` jobs/schedules plus HostAPI expected-plan fencing and post-marker
  reconciliation; `011` aggregate registry publication plus exact Page/Route/
  Hook/Service admission; and `012` static preflight plus durable migration
  source-resume proof. Their temporary shared-package compile failures are not
  committed and must be cleared before review.
- Mandatory review findings already sent to the jobs owner: post-marker River
  reconciliation must run before target admission opens; expected plan
  comparison must occur inside the same River transaction before irreversible
  mutation; crash after River commit but before evidence marking must recover
  from durable facts; arbitrary plugin/SQL error text must not be persisted.
- Next: review and land `010`, `011`, and `012` migrations independently, then
  their adapters; construct the exact Manager/runtime/Host/coordinator stack in
  bootstrap; run the same static preflight before position-zero process staging;
  and route first trusted enable through deferred install plus atomic
  publication. User-owned `.reasonix`, `.zcode`, and `CLAUDE.md` deletions remain
  untouched.

### 2026-07-14 P4 Atomic Lifecycle Publication Boundary Checkpoint

- Latest committed slices are `bd8afbfb8 feat(extensions): compose atomic
  lifecycle boundary` and `c938e74db feat(extensions): persist atomic
  publication decisions`. Overall remains **27%** and P4 remains **47% (7 of
  15)** because the production state/jobs/registry/migration adapters and
  Service/bootstrap path are not yet wired.
- The exact Host now drains job/schedule admission before runtime admission,
  rebuilds process-local source/target instances only during explicit
  revalidation, and reopens a failed early drain only after the canonical
  publication marker and durable migration compatibility proof both allow it.
- The composed boundary covers install, enable, disable, upgrade, rollback,
  and uninstall. Target runtime publication remains drained until durable
  extension state, jobs/schedules, and the aggregate registries all inspect as
  the exact target. The journal marker commits only after that convergence;
  post-marker failures remain closed and converge forward.
- PostgreSQL publication decisions are fenced by operation, canonical Host
  step, mode, exact source/target versions and digests, and attempt. Runtime
  instance ids may be rebound after process restart without changing artifact
  authority. Commit-unknown recovery reads a fresh durable marker, and marker
  evidence survives deletion of the mutable extension row.
- Focused `count=10`, race, vet, migration tests, full Support/Extensions tests,
  and real PostgreSQL concurrency/restart/commit-unknown tests passed. A clean
  `git archive HEAD` test also proved the commits do not depend on parallel
  uncommitted files.
- In-flight files are isolated to three parallel P4 slices: cleanup tombstones
  and finalization (`202607140008`), exact extension-state publication
  (`202607140009`), and production jobs/schedules publication (reserved
  `202607140010` if persistence is required). User-owned `.reasonix`, `.zcode`,
  and `CLAUDE.md` deletions remain untouched.
- Next: review and land migrations `008` and `009` independently, then their
  adapters; finish jobs/schedules and aggregate registry transactions; build
  the production preflight/migration adapter; construct the lifecycle stack in
  bootstrap; and route first trusted enable through deferred install plus
  atomic publication.

### 2026-07-14 P4 Host Gates, P6 Route Snapshot, And P8 Compiler Checkpoint

- Latest committed slice: `74fd5f367 test(themes): cover compiler security
  boundaries`. Overall remains **27%** and P4 remains **47% (7 of 15)**. P6
  and P8 foundations are not wired into production request/activation paths,
  so no authoritative row or displayed percentage is advanced.
- `891bcdbe5`, `110f752c2`, and `2794eb5eb` fixed the coordinator's five
  recovery defects, bound source/target exact runtimes, and exposed durable
  allowlisted lifecycle action results to Host gates.
- `a7a0d80ff` exposes a Manager-owned exact coordinator adapter without leaking
  or mismatching its private `ProtocolStarter`; `0b13bf3e3` dispatches all 32
  Host gate positions and revalidates stage/inspect/health/drain snapshots.
- `6f9dab6eb`, `5418fa1b5`, and `110a6941f` provide durable audit ids, scoped
  lifecycle history queries, and Service allowlist DTOs. Authority snapshots,
  opaque checkpoints, input/result documents, error metadata, and lease tokens
  cannot cross that Service inspection boundary.
- `ec5ed7fee`, `42ff24177`, and `1a37d7f41` add the P6 immutable API-route
  snapshot foundation: deterministic specificity/priority, revision CAS,
  exact runtime-instance artifact bindings, Safe Mode, conflicts, GET-to-HEAD,
  defensive activation validation, and explicit inherited-core-guard syntax.
  It still lacks Nuxt public/admin route defaults, execution/proxy semantics,
  provider selection, guard/schema/fallback enforcement, OpenAPI aggregation,
  Inspector/UI, and production lifecycle publication.
- `4969a6872` and `74fd5f367` add the P8 `html/template` compiler foundation:
  layouts/partials/control actions, restricted helpers, contextual escaping,
  tokenizer-backed static XSS rejection, passive ViewModel enforcement,
  recursion/source/output/deadline limits, immutable binding revisions, and no
  render-time filesystem access. Page ViewModels, explicit SafeHTML, typed
  segments/SEO, fallback, publication/restart convergence, and theme migration
  remain open.
- Focused normal/repeated/race/vet gates passed for Models/Extensions,
  Support/Extensions, Routes, ThemeCompiler, Pages, and Pages controllers.
  User-owned `.reasonix`, `.zcode`, and `CLAUDE.md` deletions remain untouched.
- In-flight dirty files belong to two isolated agent slices:
  `app/Support/Extensions/lifecycle_composed_boundary*.go` and lifecycle
  inspection controller/OpenAPI files. Neither is committable until its own
  failure-injection/contract gates finish.
- Next: commit the composed publication/compensation boundary and safe
  inspection HTTP contract; bootstrap the real repository/runtime/Host/
  coordinator; then route first trusted enable through deferred install and
  atomic activation before disable/upgrade/rollback/uninstall recovery UI.

### 2026-07-14 P4 Exact Runtime Publication And Call Barriers Checkpoint

- Last implementation commit: `11d12ed82 feat(extensions): run lifecycle on
  exact runtime instances`.
- P4 remains 7 of 15 rows complete. These commits close prerequisite runtime
  publication and call-admission contracts, but the lifecycle Service/HTTP path
  does not invoke them yet, so no authoritative task row or percentage was
  advanced.
- `ba0107459` gives Protocol V2 real staged, published, and retained physical
  processes. Unpublished start defers readiness until lifecycle work completes;
  exact health/publish/stop/discard and lifecycle calls never fall back to the
  active extension. Retained instances can be republished for rollback, stale
  stop cannot unregister a replacement, and V1 remains a hard replacement.
- `5d2cb6574` adds a Host-owned exact-runtime schedule admission registry.
  Publish, trigger acquire, drain, wait, failed-activation compensation, and
  retained rollback share one linearization boundary. The repository still has
  no manifest schedule trigger owner, so this is not claimed as production
  schedule integration.
- `3be41740a` binds protocol-v2 job enqueue to the active exact Manager
  instance. The lease spans the River insert, stale/draining identities fail
  closed, forced drain and caller cancellation remain distinguishable, and
  bootstrap installs the production adapter without creating an import cycle.
- `c20a87cde` binds the Service Registry winner's exact extension/instance
  identity to a Manager `RuntimeCallService` lease. Unary invocation and the
  complete bidirectional stream run on the lease context; stale, draining, and
  unavailable winners fail closed without trying a lower-priority provider,
  while forced drain remains distinguishable from caller cancellation.
- `fab179571` adds Manager-owned candidate start, exact health/readiness,
  publish, drain/wait, stop, and discard orchestration. Publication requires
  the old active instance to be drained and idle, fences both sides during the
  transition, fails closed across the ProtocolStarter/Manager switch, and
  preserves retained runtimes for exact rollback publication.
- `11d12ed82` adds the coordinator's exact-runtime adapter. Every lifecycle
  action validates its canonical step, source/target role and binding, frozen
  authority, plan, removal mode, and forced authority before acquiring a
  Manager lifecycle-cleanup lease and calling `RunLifecycleInstance`; a stale
  binding cannot fall back to the active process.
- Focused normal/repeated/race tests and vet passed for ProtocolStarter,
  Manager, HostAPI, Jobs, bootstrap, service admission, and exact lifecycle
  execution. Real subprocess coverage passed for staged publication, active
  plus retained lifecycle calls, stale-binding rejection, and forced-drain
  cancellation.
- The uncommitted Models coordinator slice passed its first normal/race/vet
  gate, then a mandatory self-audit found five recovery defects: final-gate
  success ordering, local-clock lease TOCTOU, incomplete source/target
  revalidation coverage, non-canonical marker ids, and side-effectful skipped
  terminal semantics. It is being corrected before commit.
- The exact-instance coordinator adapter, service-provider admission, and
  Manager staged-runtime API are committed. The Models coordinator corrections
  remain pending and uncommitted; the five self-audit defects must be fixed and
  its normal/race/vet gates rerun before that slice can land. Unrelated
  `.reasonix`, `.zcode`, and `CLAUDE.md` deletions remain untouched.
- Next: land the corrected Models coordinator; implement the production Host
  gate against the committed Manager stage/health/publish/drain/stop API;
  construct the coordinator in bootstrap; then move first trusted enable from
  the legacy `store.Enable -> runtime.Start` path into the durable transaction.

### 2026-07-14 P4 Exact Version And Runtime Instance Foundations Checkpoint

- Last implementation commit: `04c8b5d75 test(extensions): validate staged
  management contracts`.
- P4 is 7 of 15 rows complete. The 34-scenario real PostgreSQL matrix closes
  crash/retry at every lifecycle boundary; the broader idempotency/resume task
  remains open until Service and HTTP invoke the coordinator.
- Commits since the prior checkpoint: `f2d0ba93f` boundary recovery,
  `e01b10e8c` retry checkpoint inheritance, `075145fa7` coordinator,
  `a19ff67bb` lease migration, `397a0667c` forced wire authority,
  `0cb14b9f4` job migration ledger, `16dd33fbe` queued-job planner,
  `0a38bddb8` lease repository CAS, `b3adee77b` protocol-v2 runtime adapter,
  `3a6ccbb8f` leased Host-gate compatibility migration, and `71135e942`
  PostgreSQL/River job reconciliation. Later coherent slices are `dbb8f6a88`
  staged-version schema, `a2966f50c` inert store staging, `e5be7695b` runtime
  admission gate, `870fb4e34` coordinator lease execution, `e62a4c99c` staged
  trust review, `71340249f` inert Service upload semantics, `022fc843e`
  lifecycle authority snapshots, `468a67af1` exact staged promotion/discard,
  `fc0419b48` Manager exact-instance accounting, `125b0bbdd` inert staging
  OpenAPI, `e6fe6b3ef` staged-candidate admin display, and `04c8b5d75` the
  cross-layer management contract validator.
- The coordinator preserves stable steps/attempts/checkpoints/progress and
  detached terminal writes. Protocol-v2 now carries all eleven actions, exact
  forced authority, live progress, result JSON, and typed remote failures.
- Step lease ownership uses owner/revision/expiry CAS and PostgreSQL statement
  time, so lock waits cannot create already-expired grants. Real concurrent
  claims produce one winner; stale owners cannot heartbeat, persist, or close.
- Queued-job migration has an exact source/target/trust ledger, pure
  transactional replacement planner, and real pgx/River adapter. Replacement
  uses River's public `InsertTx` and `JobCancelTx`, conditionally links the Host
  ledger, and never updates River's private args storage directly.
- Host gates now have the independent additive `lifecycle_action = 'host.gate'`
  identity required to share the step-lease path without impersonating plugin
  actions. The reversible Down constraint retains historical Host-gate rows
  while preventing old binaries from writing new ones.
- Coordinator execution now claims and heartbeats every plugin action, Host
  gate, and forced skip. Exact lease revision fences progress and terminal
  writes; blocked heartbeats are cancellable; all terminal writes use a bounded
  detached context; Host failure recovery retains the original typed failure.
- Static uploads now persist immutable staged candidates without stopping the
  active process, changing enabled state, selecting providers, revoking the
  active exact-artifact grant, or writing migration execution history. Trust
  review and challenges bind the staged artifact while the active grant remains
  valid. The first-trusted-enable transaction is not yet wired, so the related
  P4 task remains open.
- An instance-bound admission gate now provides atomic ordinary-call closure,
  lifecycle-cleanup exemption, inflight wait, forced cancellation, and exact
  residual counters. Manager now retains exact instance snapshots and fences
  stale admission/stop bookkeeping, but ProtocolStarter still has one physical
  process slot per extension. Therefore dual-runtime execution, production
  route/job/provider admission, and the drain task remain open.
- Static upload responses and extension resources now expose an immutable
  staged candidate without leaking its database id. The admin list/details and
  bilingual success Toast distinguish staging from activation; OpenAPI refs,
  Nuxt typecheck/build, and the dedicated cross-layer validator pass.
- A P4 audit confirmed that the initial `planned` Host gate is currently
  skipped and disable/upgrade/uninstall lack required final Host gates. The
  accepted boundary requires exact source/target runtime binding, durable Host
  checkpoints, an upgrade activation gate, and final cleanup gates before the
  coordinator can own production lifecycle operations.
- Active parallel work: V2-only physical retained runtimes, lifecycle Host-gate
  path/request corrections, and exact historical-version rollback CAS. Unrelated
  `.reasonix`, `.zcode`, and `CLAUDE.md` deletions remain untouched.
- Next: land those three independent prerequisites, wire real admission and job
  barriers, execute first trusted install/enable through the coordinator, then
  implement upgrade/rollback/uninstall recovery API/UI and run the full P4 gate.

### 2026-07-14 P4 Exact-Artifact Plugin Job Checkpoint

- Last implementation commit: `d38e81d42 feat(jobs): execute exact-artifact
  plugin jobs`.
- P4 is 6 of 15 rows complete. River rows already persist the exact envelope;
  the worker now resolves the live extension and trust grant, rejects legacy or
  incompatible rows permanently, and rechecks the running and startup-frozen
  Manifest immediately before protocol-v2 dispatch.
- Job progress validates response identity, job id, monotonic counters,
  terminal state, typed failure/cancellation, and the absence of an undeclared
  result. A runtime change between resolution and dispatch maps to a permanent
  `runtime_changed` cancellation, so old code cannot receive the job.
- The deterministic upgrade policy defines execute, drain, declared migration,
  and cancel outcomes. Lifecycle-driven enumeration/migration of River rows is
  still part of the broader coordinator/drain work and is not claimed here.
- Verification passed focused repeated tests, relevant package tests, race,
  vet, `go test ./...`, `go build ./...`, staged diff review, and a parent
  targeted rerun.
- Current uncommitted ownership: crash-resumable lifecycle coordinator and the
  exhaustive PostgreSQL boundary recovery matrix. Unrelated `.reasonix`,
  `.zcode`, and `CLAUDE.md` deletions remain untouched.
- Next: review and land the boundary matrix and coordinator independently, then
  wire first trusted enable and lifecycle-driven drain into Service/HTTP.

### 2026-07-14 P4 State Machine And Durable Ledger Checkpoint

- Last implementation commit: `a3c4f75dc feat(extensions): persist lifecycle
  operations`.
- P4 is 4 of 15 rows complete. The Host-owned pure state machine fixes the
  authoritative ten states, six operations, eleven actions, recommended safety
  gates, terminal behavior, and the sole failed/cancelled retry path through
  recovery. Forced execution is uninstall-only; skippable plugin cleanup never
  bypasses Host safety gates.
- Additive PostgreSQL operation and step-attempt ledgers persist the exact
  artifact/authority snapshot, idempotency fingerprint, stable step ids,
  checkpoints, monotonic progress, typed errors, retries, actor/audit snapshots,
  and all three removal modes. Extension and audit retention cannot delete the
  lifecycle history accidentally.
- The repository serializes acquisition per extension, enforces one open
  operation, uses revision/state CAS, reuses an existing idempotency key,
  resumes failed/cancelled operations, and allocates monotonic step attempts.
- Verification passed real PostgreSQL migration Down/Up, migration/migrator
  tests, concurrent acquire/CAS/stable-step tests, restart/resume/retry tests,
  full Models/Extensions race detection, focused repeated tests, and vet.
- Current uncommitted ownership: versioned plugin-job runtime execution;
  crash-resumable lifecycle coordinator; exhaustive repository/state-machine
  boundary recovery tests. Existing `.reasonix`, `.zcode`, and `CLAUDE.md`
  deletions are unrelated and must remain untouched.
- Next: land the job and coordinator slices independently, wire the coordinator
  into exact-artifact first trusted enable, then implement drain/uninstall
  cleanup and operator retry/skip/forced-removal APIs and UI.

### 2026-07-14 P3 Completion And P4 Lifecycle Transport Checkpoint

- Last implementation commit: `e70dd677e feat(extensions): add typed lifecycle
  runtime transport`.
- P3 is 13 of 13 rows complete. Runtime streaming helpers cover route, file,
  lifecycle progress, and job streams; immutable service discovery is exercised
  across two real plugin subprocesses; the complete compatibility matrix and
  SMTP/storage/content-policy v1 package gates pass; transactional Host Commands
  retain rollback coverage.
- P4 is 3 of 15 rows complete. Activation now resolves required/optional/
  conflict/provides relationships before runtime start, with cycle, version,
  ambiguity, stale-candidate, and no-start failure coverage.
- All eleven lifecycle v2 actions cross the real gRPC subprocess transport.
  The Host freezes the lifecycle declaration at Start, rejects a stale or forged
  caller manifest, validates exact request/response runtime identity and result
  schema, preserves typed cancellation/retry metadata, and exposes the declared
  checkpoint schema while treating the current string checkpoint as opaque.
- Lifecycle transport verification passed the complete Extensions package,
  ten repeated real-subprocess runs, race detection, vet, and diff checks.
- Current uncommitted files belong to three active parallel slices: versioned
  plugin-job execution/drain compatibility, the additive lifecycle ledger
  migration, and the PostgreSQL lifecycle operation/step repository. Do not
  stage those groups together.
- Next: review and commit each parallel slice independently, run the P3/P4
  repository gates, then wire the lifecycle state machine and first-trusted-
  enable transaction to the durable ledger and frozen runtime contract.

### 2026-07-14 Service Broker And Reference Checkpoint

- Last implementation commit: `876836f7a fix(deploy): bind built-in plugin
  digest in image`.
- P3 is 9 of 13 rows complete. The dedicated CI drift row and built-in V2
  migration with real V1 rollback are now closed.
- Committed immutable Service Registry snapshots, exact SemVer/build selection,
  conflict inspection, Host List/Resolve/Invoke/Stream, schema enforcement,
  SDK service dispatch, idle timeout, and handshake freeze.
- ProtocolStarter now publishes Manifest-matched handshake services, serializes
  each extension lifecycle, removes exact instances on Stop, replaces instances
  on restart, and reaps registrations after unexpected process exit.
- Host authorization is the only caller-authority decision. Provider runtime
  grants are no longer confused with caller grants. Plugin-supplied Actor is
  rejected/cleared pending a Host-attested delegation contract.
- V2 hooks now bind full Manifest event id/name/kind/contract/input/result and
  derived patch schemas. Contract drift fails closed.
- `sforum.content-policy` defaults to Protocol V2, publishes a typed reusable
  service, enforces typed hook contracts, and retains a buildable/runnable V1
  source and Manifest. Linux image builds refresh and validate the exact binary
  digest after reproducible compilation.
- New focused gates passed: HostAPI and Extensions race/vet, content-policy
  race/vet, real crash/restart/concurrent-start subprocess tests, CLI Linux
  double-build reproducibility, built-in build script, proto drift, and Host API
  V2 documentation drift.
- Full Docker image build remains externally unverified because Docker Hub's
  anonymous-token request timed out over IPv6 before build steps began.
- Current uncommitted ownership: transactional Host Command implementation in
  `apps/api/app/Support/HostAPI/v2{,_command}*.go`; two-plugin E2E and v1 package
  compatibility agents are running in separate test files.
- Next: finish transactional rollback integration, real two-plugin service
  discovery matrix, SMTP/storage V1 package gates, then implement remaining
  route/file/progress/job streaming before the P3 full repository exit gate.

- Last implementation commit: `756c33738 test(extensions): cover protocol v2
  concurrency gate`.
- P3 commits so far: `b4d50005f`, `bda361626`, `1b9923372`, `ff4661103`,
  `7afa0c174`, `063f9897a`, `7320324e6`, `ef3ac6288`, `1bd1988c2`, and
  `01156b709`, `d5eef0127`, `297f6b92f`, `7dcac6eab`, `fc1fef8f4`, and
  `756c33738`.
- The library survey selected latest HashiCorp go-plugin `v1.8.0`, latest
  protobuf-go `v1.36.11`, pinned Buf `v1.71.0`, and protoc-gen-go-grpc `v1.6.2`;
  the isolated `tools/proto` module keeps tool dependencies out of the API
  runtime graph.
- `sforum.protocol.v2`, `sforum.plugin.v2`, and `sforum.host.v2` define 18 gRPC
  services and 147 message/enum declarations. They cover handshake, health,
  readiness, lifecycle, routes, hooks, query/transactional commands, database,
  cache, jobs, schedules, services, secrets, files, HTTP, admin, identity,
  permissions, media, navigation, audit, and tracing.
- Every dynamic document binds a schema id/version. Request context carries
  actor, locale, trace/request ids, deadline, exact extension identity, runtime
  epoch, trust grant, and disclosed authority. Transactional commands bind
  version, idempotency, dry-run impact, policy decisions, atomic outcome, audit,
  revision, and typed result.
- Generated Go code is committed under `apps/api/sdk/plugin/v2/gen`. Descriptor
  tests lock the complete service catalog, envelope fields, command result, and
  twelve required streaming modes. `scripts/test.sh` now runs Buf lint,
  generation, and drift detection.
- Runtime identity now exposes the exact grant, artifact digest, runtime token,
  epoch, and disclosed authority required by the v2 request envelope.
- The generated SDK provides a protocol-v2 plugin server with exact token,
  artifact, and epoch binding; health/readiness; 4 MiB message limits;
  deadlines; and a concurrency gate.
- Runtime startup selects transport exactly from the trusted Manifest:
  protocol v1 uses net/rpc, protocol v2 uses gRPC with AutoMTLS, and neither
  mismatch direction silently downgrades. Real subprocess tests exercise the
  v2 handshake, health, readiness, and hook path.
- Runtime status and the admin plugin list expose protocol version, transport,
  deprecation, start count, RPC call count, and last call. Protocol v1 remains
  operational and visibly deprecated until its P13 removal gate.
- Each v2 process now receives a unique `go-plugin.GRPCBroker` channel. Host
  calls require the exact runtime token in gRPC metadata plus the current
  artifact/grant/epoch/instance identity, disclosed authority, request id, and
  deadline; v2 no longer starts or receives the v1 loopback HTTP gateway.
- The Go SDK exposes all generated Host clients and builds bounded Host request
  contexts that preserve locale and trace while replacing runtime identity and
  authority. Actor is cleared until a Host-attested delegation exists.
- Host Query own-settings (unary and server stream), Permission, safe Identity,
  declared Job enqueue, and namespaced Audit append adapt the existing
  authoritative v1 services. Unsupported resource policy, job options,
  provider calls, cancellation, and list surfaces fail explicitly rather than
  silently discarding fields.
- Real subprocess tests cover Host callbacks, AutoMTLS broker streaming, stale
  identity, forged authority, expired deadline, cancellation, 4 MiB message
  rejection, concurrency saturation, and stop/start broker rebinding. Focused
  race detection passed.
- Verification passed: `go test ./...`, `go build ./...`,
  `./scripts/proto.sh check`, 1,607 OpenAPI refs across 40 files, Nuxt
  typecheck, and all 277 web validation tests.
- Rendered admin-page browser QA remains pending because both available local
  browser sessions were unauthenticated; the attempted route rendered the
  theme 404/dev overlay rather than the protected admin page.
- Working tree was clean at `756c33738` before this documentation checkpoint.
- This older checkpoint's Service Discovery next step is superseded by the
  2026-07-14 checkpoint above.

### 2026-07-14 Lifecycle Inspection, Generated Route Catalog, And SafeHTML Checkpoint

- Latest committed slice: `9100b078c feat(themes): add host-produced safe HTML
  values`. Overall remains **27%** and P4 remains **47% (7 of 15)**. The new
  P6/P8 prerequisites are still not production-published and therefore do not
  advance an authoritative row.
- `74c13e64f` and `ce30b306c` expose allowlisted lifecycle history/detail over
  authenticated `extension.view`, contract it in OpenAPI, and add stable route
  identities. The HTTP response excludes exact authority, idempotency,
  checkpoint, input/result, lease, and opaque error metadata.
- `2fe465eea` generates all 209 reviewed core API route identities into a
  caller-owned Go catalog from the same P0 source generator. Runtime code no
  longer needs to read `docs/` or maintain a handwritten duplicate when the
  Route Registry is production-wired.
- `9100b078c` bumps the Theme Compiler contract to
  `sforum.theme-compiler@2` and adds an opaque Host-produced `SafeHTML` value.
  Only the explicit `safeHTML` helper accepts it; ordinary strings remain
  context-escaped, Go trusted-content aliases remain rejected, and URL or
  attribute contexts retain `html/template` filtering.
- Composed lifecycle review found three release-blocking crash boundaries now
  being corrected before commit: target admission must remain drained until
  DB/jobs/registries are exact; uninstall position 6 may only stage a durable
  pending purge until the operation terminal is committed; and publication
  requires a durable exact-operation journal so restart can converge partial
  DB/jobs/registry writes instead of relying on process-local compensation.
- Source drain must close jobs, schedules, and route-facing admission before
  waiting on the exact runtime. A missing production drainer fails closed.
- Recovery decision persistence is in flight against migration `202607140006`.
  It keeps the original exact-artifact authority immutable while recording the
  actor/audit/reason for every retry, skip, and forced-uninstall escalation.
- User-owned `.reasonix`, `.zcode`, and `CLAUDE.md` deletions remain untouched.

## P4 Exit Checkpoint

- P4 is complete at **15 of 15 rows**. Overall V3 progress is **31%** and P5 is
  active at **0 of 17 rows**.
- Production bootstrap binds Service mutations to the PostgreSQL lifecycle
  repository, exact protocol-v2 runtime, static preflight, migration engine,
  River jobs, schedule admission, route/service/page registries, durable state,
  publication journal, cleanup tombstone, database disposition, and terminal
  physical-purge finalizer.
- Static install remains inert. `install.plan` and `install` first run only in
  the exact-artifact trusted enable transaction. Disable, upgrade, rollback,
  and uninstall retain frozen authority and exact runtimes until their declared
  hooks and Host-owned cleanup boundaries finish.
- Preserve, export-then-remove, and complete removal have real PostgreSQL
  coverage. Repeated uninstall, external cleanup failure, forced skip/recovery,
  original actor/audit retention, crash replay, and incompatible queued jobs
  are covered across Service, coordinator, boundary, and full-chain tests.
- `9895221a8` makes the failed machine snapshot and typed error one durable
  transition before terminal completion. `3b6480fb9` marks a successfully
  claimed step lease `running`, so the exact PostgreSQL fence cannot reject its
  own Host gate. `1982525d6` proves Service -> PostgreSQL coordinator -> real
  protocol-v2 subprocess -> composed deactivation -> cleanup/finalizer.
- Authenticated Chrome QA passed on desktop and 390px mobile for lifecycle
  history, close behavior, long-id wrapping, and all three uninstall modes;
  refreshed console warn/error output was empty.
- Final gates passed: real PostgreSQL lifecycle suites, related five-package
  race detection, five-package vet, and complete `./scripts/test.sh`, including
  1,677 OpenAPI references, Nuxt typecheck, 212 routes, 117 UI surfaces, and 99
  traceability rows.
- P4 owns schedule admission and drain only. Dynamic plugin periodic trigger
  registration remains a P7 deliverable and should use River
  `PeriodicJobs.AddSafely` / `RemoveByID`; no current production trigger caller
  bypasses the P4 admission gate.
- P5 must reuse the already-landed deterministic Database Registry, scoped
  credentials, exact migration engine, migration ledger, and uninstall data
  disposition. Resume by auditing the 17 P5 rows against current production
  code before implementing only the missing Host Query/Command and database
  authority boundaries.

## P5 Implementation Checkpoint

- P5 is **65% complete (11 of 17 rows)**. Weighted overall V3 progress is
  **36%**. This count credits only production behavior or exact integration
  evidence, not protobuf shapes or unwired foundations.
- Completed rows: deterministic safe identifiers; exact migration discovery,
  checksum, lock, read-only dry-run, transaction policy, ledger/progress/failure;
  isolated Goose parsing/history; uninstall database disposition; broad
  migration/credential/disposition tests; own-schema isolation; and
  multi-process migration-once. Core-upgrade compatibility blocking and
  database query tracing/slow-query observability are now also complete.
- `c913127c8` adds a public exact-artifact migration preflight. It uses a
  PostgreSQL read-only transaction and the same Goose parser as execution,
  reports statement/checksum digests, transaction modes, non-transactional
  warnings, and backup strategies, and proves no SQL execution, resource
  provisioning, or ledger writes.
- `d75de7751` runs the same exact migration plan from two independent test
  processes and independent PostgreSQL pools. The durable advisory lock and
  ledger converge to one plan, one applied step, and one state row.
- `dd0e08e74` enforces eight connections, a five-second statement timeout, and
  a fifteen-second idle-transaction timeout on every own-schema runtime role;
  real PostgreSQL tests prove over-budget rejection and slow-query cancellation.
- `c9ca0d797` publishes Host-owned, security-barrier, non-updatable
  `sforum_core_v1` views for safe identities, public forum content, entity
  metadata, and attachment metadata without granting base-table access.
- `801052f2e`, `0c2fcdfec`, and `eedaabb15` add the durable Host Command receipt
  ledger, server-attested exact broker identity, and a PostgreSQL backend that
  keeps domain writes, audit, and replay evidence in one transaction. Concrete
  production domain command definitions remain required before the command
  rows can close.
- `368e48e4d` production-binds an immutable Host Query catalog for safe user,
  public topic, and public attachment reads. Exact enabled artifacts and live
  trust grants are resolved server-side; forged request identity/authority,
  stale artifacts, PII fields, unsafe shapes, and oversized pages fail closed.
- `c142426a2` promotes database authority, compatibility, backup/retention,
  migration digest, and transaction policy in the exact-artifact trust flow.
  Authenticated admin-page QA plus a temporary production-component fixture
  passed on desktop and 390px with no overflow or app console errors; ordinary
  backup guidance expires after ten seconds while high-risk warnings persist.
- `ebc8c3919` runs an exact-trust raw-authority compatibility check before any
  Goose core migration. A real PostgreSQL test proves incompatible target
  versions block while revoked grants no longer do.
- `86ed767d8` and `36e143386` add bounded, redacted Host Query tracing with
  slow classification and document the direct-role PostgreSQL logging boundary.
  Real PostgreSQL, race, vet, and build gates passed.
- `c13da29d1`, `14086dee2`, and `926ed2ff2` add the exact-artifact
  DatabaseService core, real own-schema transaction/isolation/replay/revocation
  proof, and frozen gRPC registration.
- `42a9d895d` adds the Manifest V3 source for that catalog: namespaced positive-
  version operations bind exact `database_operation` package files and digests,
  typed parameter/result schemas, column allowlists, and HostAPI-aligned limits.
  Inline SQL, authority mismatch, query/execute field mixing, undeclared local
  schemas, and JSON Schema drift fail closed.
- `e5bb88c3b` and `4a1fb0fb3` complete the plugin-owned transaction row:
  API and standalone worker bootstrap bind exact SQL catalogs before any
  broker registration; active, disabled/installed, staged, upgrade, and
  rollback artifacts are supported without mutating already-running broker
  snapshots. DatabaseService trace output is bounded and redacted.
- `bc591e691`, `09df90982`, and `dc49dcf6d` add strict modular plugin OpenAPI
  aggregation plus the durable route-provider selection Store/API and V2
  cleanup invalidation. They do not advance P6 yet because production
  bootstrap, admin conflict UI/API, and the Fiber dispatcher remain open.
- `e5eea90ab`, `7dbe3e3c0`, and `43be1b5d5` begin P6 without inflating its
  completion count: exact runtime publication fencing, wildcard conflict
  inspection, the 212-row production core catalog, Safe Mode filtering, and an
  immutable fail-closed execution plan are present, but Fiber has no Registry
  dispatcher yet.
- Partial rows: runtime credential delivery; physical `database.core.full`
  grants; stable views/typed query-command publication; and concrete PostgreSQL
  Host Command atomicity. Missing rows include six concrete Host Commands and
  raw-authority behavior tests.
- The authority audit found a real public-contract conflict. ADR prose says
  database powers are disclosed in tiers, while Manifest, JSON Schema,
  OpenAPI, validation, and the database ledger all store one mutually
  exclusive authority. Do not silently invent cumulative semantics.
- Product decision requested: prefer additive composable database grants plus
  exact-artifact declared operation catalogs and bounded batch transactions.
  Direct per-process credentials remain an alternative, but the current
  one-role-per-extension rotation model terminates old sessions and is unsafe
  for rolling upgrade or multi-node runtime use.
- Additional product boundaries remain for trusted actor delegation into
  actor-scoped Host Commands and the provider-neutral entitlement lifecycle.
  Continue independent stable-view, connection-budget, migration-guidance,
  and backend-ledger work while awaiting those decisions.
- PostgreSQL development dependencies are available. Do not assume frontend or
  API dev servers remain running; start them explicitly before browser QA.

## P6 Route Inspector And Dispatcher Integration Checkpoint

- Weighted V3 progress remains **36%**. P5 remains **65% (11 of 17 rows)** and
  P6 remains **0% by authoritative exit rows**. P6 foundations are materially
  ahead of zero, but no row is promoted until its production transport,
  authority, schema, and fallback boundaries close together.
- Latest committed slice: `738b8a30b feat(routes): inspect exact execution
  chains`. The detached Inspector reports one immutable Registry revision,
  exact execution chain, selected/stale/unselected provider state, guard and
  contract metadata, and relevant bounded timing/fallback traces. It never
  chooses a provider by priority and filters unrelated route conflicts/traces.
- `RouteTraceRing` is concurrency-safe and circular, defaults to 256 records,
  hard-caps at 4096, and accepts no request, response, header, query, body,
  secret, actor, idempotency-key, or raw-error fields. Forged exact-artifact
  attribution is rejected.
- Inspector focused, race, vet, build, and diff checks passed. The committed
  core is not yet exposed through HTTP/UI and the Dispatcher has not yet
  published trace events, so the P6 Inspector task remains open.
- Authenticated Chrome initially rendered
  `/control-panel/extensions/route-providers` with the expected super-admin
  identity, navigation, route-provider copy, counters, and loading state with
  no console warnings/errors. API hot reload temporarily disappeared and the
  page received 502 responses; after API recovery, a full reload found the
  login session expired. Desktop/mobile selection/reset QA therefore remains
  incomplete and must not be reported as passed.
- Active dirty ownership is isolated: Fiber buffered dispatcher and production
  bootstrap in `app/Http`, `Support/Routes/dispatcher*`, and `bootstrap/app.go`;
  Inspector HTTP/OpenAPI/catalog generation in Extensions controllers,
  contracts, and generated route catalogs; Page ViewModel/compiler work in
  `Support/ThemeCompiler`. Do not stage these groups together.
- Dispatcher review found boundaries that must stay explicit: pure core plans
  must bypass buffering to preserve existing streaming/download behavior;
  inherited core guards need executable Host guard metadata; declared schemas
  need an exact-artifact schema catalog; and safe fallback needs Host-observed
  side-effect evidence rather than trusting only a plugin response header.
  SSE, WebSocket, multipart, streaming, and backpressure remain unimplemented.
- P8 review rejected a ThemeCompiler-only `forum.search` identity because the
  current Page Registry has no standalone search page. Search remains state in
  the existing home/list ViewModel until a real core page identity is added.
- Exact resume order: review and land the pure-core-safe Fiber adapter; review
  and land Inspector HTTP/OpenAPI separately; review and land the catalog-only
  Page ViewModel slice; then implement Dispatcher trace publication, exact
  schema catalogs, inherited/custom guard contracts, Host-observed side-effect
  fencing, and the non-buffered transport modes before recalculating P6.

## P6 First Accepted Rows Checkpoint

- P6 is now **33% complete (6 of 18 authoritative rows)** and the weighted V3
  total is **39%**. The active phase is P6; P5 remains partially open at 65%
  only for the three recorded product decisions and their dependent rows.
- Accepted task rows: all 218 core routes have generated stable ids/contracts;
  snapshots are immutable with deterministic specificity/priority; declared
  plugin routes may target arbitrary public/admin/API paths and methods; and
  exact replace-provider selection plus conflict API/UI is production-bound.
- Accepted test rows: Safe Mode excludes all third-party route snapshots while
  preserving Host routes, and strict plugin OpenAPI aggregation rejects
  collisions and unsafe package references.
- `9bcaad539` exposes a permissioned, detached Route Inspector snapshot over
  modular OpenAPI without returning the raw inspected path/query. `ace782c5b`
  production-binds the buffered HTTP dispatcher to the exact lifecycle
  Registry/Runtime Manager while leaving pure core streams untouched and
  filtering request authority, Host headers, and plugin `Set-Cookie` output.
- These commits do not close full action semantics, inherited/custom/raw
  guards, exact schema catalogs, Host-observed side-effect fencing, SEO alias/
  redirect integration, live trace publication, streaming transports, the
  complete action/disconnect/crash matrix, or the new-vs-v1 benchmark row.

## P6 Production Trace Checkpoint

- P6 is now **39% complete (7 of 18 authoritative rows)** and weighted V3 is
  **40%**. The newly accepted row is the production Route Inspector with exact
  chain/provider/guard/contract metadata plus bounded timing and fallback
  traces.
- `3b017173c` injects one concurrency-safe trace ring into both the production
  Dispatcher and permissioned Inspector. Plugin steps publish redacted denied,
  schema-rejected, transport-failed, fallback-used, succeeded, and committed
  outcomes with exact artifact attribution and commit state. Pure core routes
  remain untraced and keep their existing streaming path.
- Independent review found that a non-handler `readonly_core` fallback was
  allowed to continue without a fallback/commit trace. `61da559d5` closes that
  audit gap and adds the before-plugin -> core fallback regression.
- Verification passed 20 repeated Route package runs, Route/HTTP race tests,
  focused vet, bootstrap/controller/provider regressions, `go build ./...`, and
  `git diff --check`.
- Active dirty groups remain isolated: the exact-artifact OpenAPI route schema
  catalog under `Support/ExtensionOpenAPI`, and P8 Page ViewModel/typed render
  work under `Support/ThemeCompiler`. P8 independent review found nested-island
  structure and required Host-form-island blockers, so no P8 row is credited.
- Next: review and land the schema catalog foundation, then production-publish
  it from exact lifecycle artifacts; complete Host-observed side-effect fencing,
  inherited/custom/raw guards, remaining action semantics, streaming transports,
  SEO integration, complete failure matrix, and the v1 comparison benchmark.

## P6 Host-Observed Fallback Fence Checkpoint

- P6 is now **56% complete (10 of 18 authoritative rows)** and weighted V3 is
  **41%**. Newly accepted rows are safe GET/fail-closed unsafe fallback,
  preventing fallback after Host-observed request/response commitment, and the
  unsafe replacement failure test that proves Core is never a second writer.
- `caa158402` removes the plugin-controlled side-effect response header as an
  authority source. The Host transport records request headers/request write
  and first response byte through `net/http/httptrace`; after the request leaves
  the Host, crash, disconnect, cancellation, timeout, oversized response, or
  partial response all remain fail closed.
- A pristine safe-method dial failure may still use declared `not_found` or
  `readonly_core` fallback. Unsafe methods never fallback. Tests cover accepted
  GET crash, partial headers/body, accepted POST, timeout/cancellation, forged
  side-effect headers, exact commit-state precedence, trace publication, and
  zero Core calls after any possible plugin write.
- Verification passed 10 repeated HTTP/Route/bootstrap runs, race detection,
  focused vet, `go build ./...`, and staged diff checks.
- The schema catalog remains uncommitted after a second independent review
  found missing status/media-type identity, duplicate JSON-key rejection, and
  bounded/cancellable validation. P8 typed Page ViewModel output was committed
  in `d9268872a` but remains at 0 authoritative rows until production
  construction and runtime publication are proven.
- Next: close and production-bind exact schemas; implement executable inherited
  and separately trusted custom/raw guards; then complete action protocol,
  alias/rewrite, streaming transports, SEO integration, failure matrix, and
  the v1 comparison benchmark.

## P6 Typed Guard And Exact Schema Foundation Checkpoint

- P6 remains **56% complete (10 of 18 authoritative rows)** and weighted V3
  remains **41%**. Neither foundation is promoted until its production runtime
  and lifecycle publication are complete.
- `1fcfdbf05` adds reviewed typed guards for all 218 Core routes, exact inherited
  guard resolution, fail-closed missing/plugin targets, immutable permission
  slices across Registry/Match/Inspector/plan boundaries, and source-verified
  generated catalog policy. The production Guard Authorizer is still active
  work; custom/raw guard authority remains separate high-risk trust.
- `6667e630f` adds an exact artifact/route/method/contract/action/direction/
  schema/media/status catalog using `jsonschema/v6`. It isolates operation
  variants, rejects duplicate/trailing/deep/oversized JSON, shares compiled
  schemas, and bounds decode plus validation with slots, deadlines, structural
  budgets, and exact HEAD-to-GET response admission.
- Schema focused count-10, split race, vet, and `go build ./...` gates passed.
  Production still injects a nil catalog. Correct publication must join Route
  and Schema restore under the durable lifecycle fence; restart currently
  restores only Core routes.
- Production has no source for the Host-owned OpenAPI route policy tuple
  (security, rate-limit, idempotency). The ADR requires Host authority but does
  not select defaults or a persistence/resolution contract. Do not invent
  plugin-controlled or inferred policy values merely to publish the catalog.
- P5 remains 11/17: its remaining rows depend on composable database grants,
  actor delegation, provider-neutral entitlement semantics, and direct database
  credential rolling-upgrade policy. These do not block independent P6/P8 work.
- Active parallel work: production typed Guard Authorizer, Dispatcher allocation
  regression/benchmark, and P8 production Page ViewModel construction/publication.

## P6 Performance Comparison Checkpoint

- P6 is now **61% complete (11 of 18 authoritative rows)** and weighted V3 is
  **42%**. The accepted row is the reproducible performance comparison against
  the current namespaced proxy baseline; the measured V3 regression remains
  explicit and must be remeasured at P13.
- `e38d91f7a` binds planning to one internal immutable Registry revision instead
  of copying all 218 routes one or two times per request. Public Snapshot,
  Resolve, Match, and plan getters remain caller-owned deep copies with mutation
  and concurrent-publication race coverage.
- Allocation gates pass at Core 21/64, selected HTTP 352/480, and six-step chain
  1,557/2,100 allocations. The same-run medians are v1 198.263 us, Core 117.740
  us, selected 606.905 us, and composed 2.418 ms.
- Selected HTTP is still +206.1% latency, +232.9% bytes, and +208.8%
  allocations versus the comparable v1 fixture. Production PostgreSQL provider
  lookup is excluded, so this is not a no-regression claim.
- `24ae6a3a2` production-binds the typed Core Guard Authorizer with an exact
  plan/step/request fence. Eleven explicit evaluator ids cover 22 contextual
  routes; the remaining 101 contextual routes and custom/raw authority stay
  fail closed and therefore do not close the guard row.
- `b84920a03` separates schema-only aggregation from policy publication. Schema
  compilation strips policy fields; public OpenAPI aggregation still requires
  exact Host-owned security, rate-limit, and idempotency policies.

## P6 V2 Transport And P8 Runtime Checkpoint

- Weighted V3 progress is now **44%**. P6 remains **61% (11 of 18 rows)**;
  Protocol V2 unary transport is a required production slice but does not by
  itself close the complete route-action or streaming rows. P8 is accepted at
  **33% (6 of 18 rows)** after production review.
- `c64fa56f8` adds revision-fenced immutable Route Schema publication. A
  prepared candidate is publishable only when its base revision, caller
  expectation, and current revision all match; stale and foreign writers fail.
- `0a1997578` and `2a180cb07` build and production-bind exact theme runtime
  snapshots for 23 of 23 Core Page ViewModels. Snapshot-covered requests use
  compiled templates without Store or filesystem access, stale artifact
  identity fails closed, and PAT-scoped viewer authority cannot regain full
  permissions from the user projection.
- `d1d42a130` measures production `Snapshot.Render` and compile paths for small
  and large fixtures and adds allocation ceilings. The report explicitly
  defers full request-chain, RSS, and JavaScript-disabled comparison to P13.
- `6a41bbcd9` routes Protocol V2 HTTP-mode steps through real gRPC
  `InvokeRoute`, exact retained instances, one Manager admission lease,
  Host-authored actor authority, frozen route/schema identity, bounded typed
  responses, and fail-closed `stream_follows`. Review removed a lifecycle mutex
  that serialized all same-plugin requests and made trace ids unique per call.
- P8 rows accepted: all catalog Page ViewModel contracts, the bounded compiler,
  standard template control actions, sealed SafeHTML, immutable exact runtime
  snapshots, and the compile/render performance row. Install-time compilation,
  all-catalog zero-I/O, typed frontend consumption, four-level fallback,
  plugin business ViewModels, multi-node convergence, and crawler/JavaScript-
  disabled evidence remain open.
- Verification passed focused repetition, race detection, full Pages,
  Extensions, Pages Controller, HTTP and bootstrap packages, vet, all API tests,
  `go build ./...`, and staged diff checks.
- Active parallel work is isolated: immutable Page provider resolution under
  `Support/Pages`, and production Route Schema lifecycle/bootstrap publication.
  Next integrate Route + Schema restore, then continue contextual/custom/raw
  guards and freeze full action semantics before streaming transports.

## P6 Schema Lifecycle And P8 Admin Isolation Checkpoint

- Weighted V3 progress is now **45%**. P6 remains **61% (11 of 18 rows)**;
  P8 advances to **39% (7 of 18 rows)** after accepting public-theme/admin
  style isolation. P5 remains partially open at **65% (11 of 17 rows)** and
  does not block independent P6/P8 delivery.
- `69ac44074` fences exact Route Schema replacement artifacts, and `69266b051`
  publishes schemas before Route Registry exposure, restores both immutable
  publications after runtime reconciliation, keeps Safe Mode Core-only with an
  empty plugin schema catalog, and injects the live publication into the
  production dispatcher validator.
- `ecf860984` adds notification-recipient contextual guard evaluation. Explicit
  evaluators now cover 26 of 123 contextual routes; the remaining 97 routes,
  custom guards, and raw request/session authority continue to fail closed.
- `6dda8c9e0` resolves Page providers from immutable snapshots without a
  PostgreSQL lookup on the production request path.
- `9e369ffc8` keeps public theme skins out of admin desktop/mobile routes while
  restoring them on SPA navigation back to public pages. Browser QA found two
  exact public links, zero admin links, and no relevant console errors.
- Verification passed focused/race package tests, vet, `go build ./...`, 290
  web tests, Nuxt typecheck, production build, and desktop/mobile browser QA.
- Next close the remaining contextual guard catalog and independently trusted
  custom/raw guards, then implement the frozen route-action semantics and real
  multipart/streaming/SSE/WebSocket transports. P8 next owns typed frontend
  render segments, four-level fallback, and all-catalog zero-I/O evidence.

## P8 Install-Time Template Safety Checkpoint

- Weighted V3 remains **45%** after flooring. P8 advances to **44% (8 of 18
  rows)** by closing install-time static template safety; P6 remains **61% (11
  of 18 rows)** and P5 remains **65% (11 of 17 rows)**.
- `b54b8f541` runs bounded preflight over every uploaded theme L1 page plus its
  shared layouts and partials before the package enters the authoritative
  Store. Static dangerous HTML, forbidden helpers, recursive/deep template
  graphs, and contextual escaping failures are rejected even in unused pages.
- Activation still recompiles the exact artifact. A failed candidate cannot be
  staged or published, and the prior immutable runtime continues serving.
- Focused and repeated package tests, targeted race tests, Models race tests,
  vet, `go build ./...`, and ordinary ThemeCompiler allocation budgets passed.
  Full ThemeCompiler under `-race` exceeds pre-existing allocation ceilings
  because of race instrumentation and is not treated as an allocation result.

## P5 Remaining-Boundary Audit

- P5 remains **65% (11 of 17 rows)**. The six open rows are credential delivery
  to plugin processes; stable views plus typed Query/Command publication; six
  production Host Commands; physical `database.core.full`; raw-authority
  behavior/compatibility; and complete Host Command atomicity evidence.
- The current database registry issues one shared runtime role per extension.
  Password rotation or revocation changes that role and terminates every
  session using it, so wiring the current credential directly into plugin
  startup would break rolling upgrades and multi-node operation.
- Product freeze is required before implementation: map legacy single-choice
  `database.authority` to additive grants, and choose per-runtime lease roles
  versus Host-mediated DatabaseService-only access. The recommended V3 path is
  cumulative compatibility grants plus per-runtime lease roles, with old/new
  credentials coexisting until the old exact runtime drains.
- Migration read-only preflight, retry, advisory locking, and failure recovery
  are already accepted. More tests in those completed areas cannot be used to
  inflate the six remaining rows.

## P6 Guard, P7 Service Dependency, And P8 Typed Render Checkpoint

- Weighted V3 progress is now **46%**. P6 remains **61% (11 of 18 rows)**;
  P7 begins at **5% (1 of 22 rows)**; P8 advances to **50% (9 of 18 rows)**.
- `28b3dbf74`, `40aab8be5`, `e60324989`, `7b93272a0`, `ff85c6fa3`,
  `a8a59df5b`, and `afaaa6eb8` expand exact contextual guard execution from 26
  to 68 of 123 routes. Fifty-five resource- or policy-dependent routes remain
  fail closed; custom and raw request/session authority also remain closed.
- `82a16c65e` closes typed plugin-to-plugin service dependency/version checks.
  Protocol V2 atomically publishes exact caller/provider identity, services,
  required/optional ID or capability SemVer, and provided capabilities. List,
  Resolve, Invoke, and Stream share one immutable snapshot with no database hot
  path; stale, disabled, upgraded, missing, version-drifted, and ambiguous
  runtimes fail closed.
- `615a9496f` closes typed frontend RenderOutput consumption. Public registry
  routes prefer typed HTML segments and island descriptors; parse5 rebuilds the
  same nested HTML5 AST in SSR and hydration, while the legacy L1 compatibility
  path also no longer uses regex or `v-html` island splitting.
- Typed output rejects missing/duplicate/forged placeholders, unknown
  components, cross-type props, dangerous elements/attributes/URLs, and
  correctly restores Go `omitempty` zero values. Focused tests passed 14/14,
  all web tests 304/304, Nuxt typecheck and production build passed, and real
  3002 SSR/reload QA reported no placeholder leakage, alert, or console errors.
- `e11c518bb` fixes startup Route/Schema restoration so legacy provider-only
  plugins are skipped while exact runtimes with declared Routes/OpenAPI remain
  eligible without requiring lifecycle hooks. A real API boot completed
  migrations, runtime reconciliation, publication restore, embedded worker
  startup, and listen on 8081 before a clean shutdown.

## P6 Guard, P7 Hook Registry, And P8 Fallback Checkpoint

- Weighted V3 progress is now **48%**. P6 remains **61% (11 of 18 rows)**;
  P7 advances to **18% (4 of 22 rows)**; P8 remains **50% (9 of 18 rows)**
  until the full theme-switch row, add-page SSR path, and multi-node evidence
  converge.
- `476ca7aca`, `5b96b58f2`, and `24ed8b4d8` production-bind immutable
  exact-artifact extension policy, static option/public policy, and Page
  Registry access policy. Contextual guard execution now covers **101 of 123**
  routes; 22 resource-dependent routes plus custom/raw authority remain closed.
- `ae4ca62fd` implements the compiled active-override -> plugin -> active theme
  -> default theme -> Host emergency render plan, exact template digest and
  ViewModel binding, default-theme prewarm, typed attempt evidence, and zero
  request-time package I/O. The P8 fallback row remains open until plugin add
  pages and the complete switch/convergence contract are verified.
- `124c151dc`, `77674b2fc`, and `4b3f8b82c` close the first three P7 tasks:
  Manifest V3 versioned namespaced action/filter hooks; immutable exact-runtime
  priority/schema/failure-policy composition; and mandatory Host revalidation
  for authoritative filters. Plugin dependency SemVer, optional-provider
  downgrade, River exact delivery, nested payload isolation, lifecycle
  rollback, and Protocol V2 declaration binding fail closed.
- Hook validation rejects async fail-closed contracts because individually
  durable enqueues cannot be rolled back, and bounds Host listener deadlines to
  1-5000 ms. Full ExtensionManifest/Extensions tests, complete Extensions race,
  focused rollback races, vet, and build passed before the three commits.
- Active uncommitted ownership is isolated: P6 continues the remaining guards;
  P8 owns additive active-theme migration, exact-preview/current-theme CAS,
  stale binding reconciliation, runtime skin revision, OpenAPI, and admin UI.
  Do not stage P8 files with guard or documentation commits.
- P5 remains **65% (11 of 17 rows)** pending explicit product freeze for
  additive grants, per-runtime lease roles, actor delegation, and the
  provider-neutral entitlement minimum. Independent P6-P8 work continues.

## P6 Declared Route And Cookie Credential Checkpoint

- Weighted V3 remains **48%**. P6 remains **61% (11 of 18 authoritative
  rows)** because these guard batches do not yet close the complete inherited/
  custom/raw guard row.
- `32d32ac72` authorizes declared extension route guards against immutable
  exact-artifact policy, and `306204f98` adds Host-derived cookie/bearer
  credential provenance for PAT list/create. Client header text is not trusted;
  the dispatcher reads the authenticated PAT context after Bearer middleware.
- Contextual guard execution now covers **105 of 123** routes. The remaining 18
  are five target-dependent identity admin routes, three self resource-dependent
  identity routes, four executable bootstrap flows, two entity-meta value
  routes, two attachment reads, one inbound webhook, and one forum comment
  creation route. Custom/raw request and session authority remain fail closed.
- The cookie-bound slice passed ten focused repetitions, full HTTP race,
  Routes/bootstrap tests, `go vet ./...`, `go build ./...`, gofmt, staged diff,
  and whitespace checks.
- Resume by closing only routes whose ownership/policy can be proven from an
  immutable Host snapshot with zero request-path Store I/O. Do not credit route
  counts as a completed P6 row until the inherited/custom/raw guard exit gate
  passes as a whole.

## P6 Credential, P7 Provider, And P8 Publication Checkpoint

- Weighted V3 remains **48%**. The active phase is **P6 at 61% (11 of 18
  rows)**; P5 is independently open at **65% (11 of 17)**, P7 is **18% (4 of
  22)**, and P8 is **50% (9 of 18)**. Intermediate production slices below do
  not inflate a row until its complete authoritative exit gate passes.
- `9e1b80c35` authorizes the current inert inbound-webhook skeleton from an
  immutable Host policy. It does not claim webhook-signature verification.
  Contextual Core Guard execution is now **106 of 123**; 17 resource-dependent
  routes plus separately frozen custom/raw request and session authority remain.
- `78db19b98`, `9d540fc8e`, `3516bbf7d`, and `c280bbd25` add Manifest-bound
  typed provider slots, immutable exact-runtime publication, package-closure
  schema binding, fixed typed invocation, bounded deadlines, deep cloning, Host
  request/response validation, and explicit fallback. The P7 provider row stays
  open until a real Plugin B -> Host broker -> Plugin A consumption path and
  lifecycle rollback evidence pass.
- `64164cfb2`, `dd7ba898c`, `da90dbf5b`, `f41815ba2`, and `5abdea62b` make theme
  activation swap provider bindings atomically, expose a process-local runtime
  revision, require exact visible-preview CAS, publish the HTTP/OpenAPI contract,
  and bind the admin confirmation flow. Runtime failure restores the previous
  database theme, runtime, Page publication, and exact approval actor.
- `457dac0a9` independently adds the durable PostgreSQL theme-publication
  ledger, boot-scoped node leases, per-node acknowledgements, commit-time wakeup,
  and database-enforced append-only history. Real PostgreSQL tests covered
  constraints, mutation rejection, acknowledgement transitions, retention,
  history-protected Down, and reapply. PostgreSQL remains authoritative;
  `NOTIFY` is only a wakeup hint.
- Active ownership remains isolated: P6 owns the remaining contextual guards;
  P7 owns the real cross-plugin provider broker fixture; P8 owns publication
  repository/activation integration and the watcher. Resume P8 with repository
  publication in the activation transaction, then LISTEN plus poll/reconnect,
  heartbeat, apply/failed acknowledgements, and two-node convergence tests.
- P5 can resume only after the product freeze for cumulative additive database
  grants, per-runtime lease roles, short-lived Host-signed actor delegation, and
  the provider-neutral entitlement minimum. The recommended choice is all four
  ADR defaults; P5 does not block independent P6-P8 work.

## P6 Identity Target And P8 Approval Authority Checkpoint

- Weighted V3 remains **48%** and P6 remains **61% (11 of 18 rows)**. Guard
  batches are not credited as the inherited/custom/raw row until that complete
  high-risk boundary closes.
- `4850bc999` authorizes five target-dependent identity-admin routes: user
  update, client-IP clearing, permission overrides, role replacement, and
  session revocation. Contextual Core Guard production coverage is now **111 of
  123**; three self-resource routes, four executable bootstrap flows, two
  entity-meta value routes, two attachment reads, and comment creation remain.
- These five low-frequency authorized requests deliberately perform one narrow
  PostgreSQL target lookup. A process-local target-role cache was rejected
  because another API node could grant or revoke `super_admin` without
  invalidating it. Unauthorized requests perform no Store I/O. Two isolated
  PostgreSQL pools proved a grant and revoke are observed by the next guard.
- P6 verification passed focused HTTP twenty times, focused race five times,
  full HTTP/Identity/bootstrap tests, `go vet ./...`, and `go build ./...`.
- `d513aea77` independently extends the theme publication ledger with the exact
  prior Core-replacement approval actor. Database constraints reject approval
  without an actor/source tuple and reject an actor without approval; historical
  rows protect Down. P8 activation/compensation code remains uncommitted until
  all active-theme mutation paths and real PostgreSQL atomicity tests pass.
- Resume P6 with the three identity self-resource routes. P7 owns exact schema-
  validated Plugin B -> Host broker -> Plugin A evidence. P8 next commits the
  activation/compensation repository slice before starting the watcher.
