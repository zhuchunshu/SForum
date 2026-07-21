# 2026-07-13 Trusted Plugin And Theme Platform V3 P3 Progress

## Status

- Overall V3: **21%**.
- P0-P2: **100%**.
- P3: **69%**, active (9 of 13 task/test rows complete).
- Branch: `main`; last implementation commit: `876836f7a`.
- Dirty files are limited to the in-flight transactional Host Command slice;
  two-plugin and V1 package agents are writing isolated test files.

## 2026-07-14 Checkpoint

- Added immutable Service Registry snapshots and Host broker discovery,
  resolution, exact invocation, typed streaming, conflict inspection, and
  instance-bound unregister.
- Added SDK service definitions/dispatch, handshake freeze, idle timeout, and
  Host-authoritative caller permission semantics.
- Added per-extension lifecycle serialization plus real crash/restart/current-
  generation cleanup tests.
- Rejected and cleared unattested plugin Actor values; locale and trace remain
  preserved while runtime identity and authority are rebound.
- Bound V2 hooks to complete Manifest event contracts and fail-closed result/
  patch schemas.
- Migrated `sforum.content-policy` to V2 with strict typed hooks and service,
  while retaining a real V1 build-tag rollback package.
- Added reproducible Linux package/digest gates and a dedicated generated SDK/
  documentation CI drift workflow.

## Changed

- Recorded the Protobuf/gRPC/Buf library and compatibility decision.
- Added an isolated pinned Buf and official Go generator tool module.
- Added versioned `sforum.protocol.v2`, `sforum.plugin.v2`, and
  `sforum.host.v2` Protobuf packages.
- Generated the Go SDK for 18 services and 147 message/enum declarations.
- Defined typed request context, exact runtime identity, disclosed authority,
  bounded streaming frames, service discovery, and transactional Host Command
  plan/execute contracts.
- Added descriptor tests plus repository lint and generated-code drift checks.
- Exposed exact runtime grant, artifact, epoch, token, and authority identity
  for v2 envelopes.
- Added a protocol-v2 plugin SDK server with bounded messages, deadlines,
  concurrency control, health/readiness, and exact identity validation.
- Added exact Manifest-driven gRPC/net-rpc negotiation with AutoMTLS and no
  silent downgrade, covered by real subprocess compatibility tests.
- Added runtime and admin UI telemetry for protocol version, transport,
  deprecation, starts, calls, and last call.
- Added a runtime-scoped, AutoMTLS `GRPCBroker` Host channel with exact token,
  artifact, trust grant, epoch, instance, authority, and deadline validation.
- Added generated Host clients to the Go SDK; request contexts preserve locale
  and trace but clear unattested Actor and overwrite runtime identity/authority.
- Added typed own-settings Query/stream, Permission, safe Identity, declared
  Job enqueue, and namespaced Audit compatibility services without exposing
  the legacy loopback endpoint to protocol v2.
- Added real subprocess rejection tests for stale identity, forged authority,
  expired deadline, cancellation, oversized messages, concurrency saturation,
  and restart broker rebinding.

## Commits

- `b4d50005f docs(extensions): choose Host API V2 toolchain`
- `bda361626 build(proto): add deterministic generation toolchain`
- `1b9923372 fix(proto): run pinned generators from module root`
- `ff4661103 feat(proto): add Host API V2 contracts`
- `7afa0c174 test(proto): enforce Host API V2 contract drift`
- `063f9897a feat(extensions): expose exact runtime trust identity`
- `7320324e6 feat(sdk): add protocol v2 plugin server`
- `ef3ac6288 feat(extensions): negotiate gRPC protocol v2`
- `1bd1988c2 feat(extensions): expose protocol deprecation telemetry`
- `01156b709 feat(admin): show protocol deprecation telemetry`
- `d5eef0127 feat(extensions): add runtime-bound Host API broker`
- `297f6b92f feat(hostapi): adapt core services to protocol v2`
- `7dcac6eab fix(sdk): preserve Host API server streams`
- `fc1fef8f4 test(hostapi): cover protocol v2 broker callbacks`
- `756c33738 test(extensions): cover protocol v2 concurrency gate`

## Verification

- `./scripts/proto.sh check` passed.
- `go test ./...` passed.
- `go build ./...` passed.
- OpenAPI validation passed with 1,607 refs across 40 files.
- Nuxt typecheck and all 277 web validation tests passed.
- Real subprocess v2/v2 handshake, AutoMTLS, health, readiness, and hook tests
  passed, as did exact trust and both no-downgrade mismatch directions.
- Real Host broker callbacks and server streaming passed. Invalid identity,
  authority, deadline, cancellation, oversized message, and restart cases
  returned their required gRPC codes; focused `-race` verification passed.
- Rendered admin-page browser QA is still pending: both available local browser
  sessions were unauthenticated and rendered the theme 404/dev overlay.
- Existing v1 runtime remains operational and is explicitly deprecated.

## Next

1. Complete real consumer-to-provider List/Resolve/Invoke/Stream subprocess
   coverage, including authority, conflicts, stale channels, and disappearance.
2. Close SMTP/storage/content-policy V1 package compatibility and transactional
   Host Command rollback gates.
3. Implement remaining route/file/progress/job streaming and run the full P3
   repository exit gate before starting P4.

## Compression Rule

Monitor context usage continuously. Before an expected context compression,
update the durable progress ledger and this handoff with percentages, exact
commits, dirty files, verification, and the next command; then commit every
coherent buildable slice before continuing.

Every user-visible progress update must include both the displayed overall V3
percentage and the active phase percentage.

## Open Questions

- None. The accepted V3 ADR and Host API V2 toolchain decision define the
  current product boundary.
