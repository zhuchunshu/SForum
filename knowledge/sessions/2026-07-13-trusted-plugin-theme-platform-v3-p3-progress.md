# 2026-07-13 Trusted Plugin And Theme Platform V3 P3 Progress

## Status

- Overall V3: **19%**.
- P0-P2: **100%**.
- P3: **46%**, active (6 of 13 task/test rows complete).
- Branch: `main`; last implementation commit: `756c33738`.
- Working tree was clean before this documentation checkpoint.

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
- Added generated Host clients to the Go SDK; request contexts preserve actor,
  locale, and trace but overwrite runtime-owned identity and authority.
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

1. Retain validated service descriptors returned by the plugin handshake.
2. Implement the versioned Service Discovery registry over the Host broker,
   including deterministic conflict ordering and stale-provider removal.
3. Add unary invocation, bidirectional streaming, crash/restart, stale token,
   and provider disappearance tests before moving to the reference plugin.

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
