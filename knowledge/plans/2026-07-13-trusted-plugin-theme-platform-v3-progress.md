# Trusted Plugin And Theme Platform V3 Progress Ledger

Date: 2026-07-14
Overall progress: **25%**
Active phase: **P4 - Lifecycle V2, Dependency Graph, And Authoritative Hooks (20%, 3 of 15 rows)**

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
| P4 Lifecycle/dependencies | 7% | 20% | 1% |
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
