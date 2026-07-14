# Trusted Plugin And Theme Platform V3 Progress Ledger

Date: 2026-07-14
Overall progress: **27%**
Active phase: **P4 - Lifecycle V2, Dependency Graph, And Authoritative Hooks (47%, 7 of 15 rows)**

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
| P4 Lifecycle/dependencies | 7% | 47% | 3% |
| P5 Database/commands | 8% | 0% | 0% |
| P6 Routes/middleware | 10% | 0% | 0% |
| P7 Workflow/admin/query/identity | 10% | 0% | 0% |
| P8 Theme compiler/runtime | 8% | 0% | 0% |
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
