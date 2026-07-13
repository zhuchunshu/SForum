# Trusted Plugin And Theme Platform V3 Progress Ledger

Date: 2026-07-13
Overall progress: **18%**
Active phase: **P3 - Host API V2 And Generated SDKs (31%)**

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
| P3 Host API v2 | 8% | 31% | 2% |
| P4 Lifecycle/dependencies | 7% | 0% | 0% |
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

- Last implementation commit: `01156b709 feat(admin): show protocol
  deprecation telemetry`.
- P3 commits so far: `b4d50005f`, `bda361626`, `1b9923372`, `ff4661103`,
  `7afa0c174`, `063f9897a`, `7320324e6`, `ef3ac6288`, `1bd1988c2`, and
  `01156b709`.
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
- Verification passed: `go test ./...`, `go build ./...`,
  `./scripts/proto.sh check`, 1,607 OpenAPI refs across 40 files, Nuxt
  typecheck, and all 277 web validation tests.
- Rendered admin-page browser QA remains pending because both available local
  browser sessions were unauthenticated; the attempted route rendered the
  theme 404/dev overlay rather than the protected admin page.
- Working tree was clean at `01156b709` before this documentation checkpoint.
- Next command after committing this checkpoint: implement Host API v2 over
  `go-plugin.GRPCBroker`, register host-owned generated services with exact
  runtime-token/identity validation, and start with typed query, permission,
  settings, jobs, and audit compatibility adapters while retaining the v1
  loopback Host API until P13.
