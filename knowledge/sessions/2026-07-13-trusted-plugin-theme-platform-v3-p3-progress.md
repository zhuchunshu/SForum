# 2026-07-13 Trusted Plugin And Theme Platform V3 P3 Progress

## Status

- Overall V3: **17%**.
- P0-P2: **100%**.
- P3: **15%**, active (2 of 13 task/test rows complete).
- Branch: `main`; last implementation commit: `7afa0c174`.
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

## Commits

- `b4d50005f docs(extensions): choose Host API V2 toolchain`
- `bda361626 build(proto): add deterministic generation toolchain`
- `1b9923372 fix(proto): run pinned generators from module root`
- `ff4661103 feat(proto): add Host API V2 contracts`
- `7afa0c174 test(proto): enforce Host API V2 contract drift`

## Verification

- `./scripts/proto.sh lint` passed.
- `./scripts/proto.sh check` passed after generated files were committed.
- `go test ./sdk/plugin/v2/...` passed.
- `go build ./...` passed.
- Existing v1 runtime code and fixtures were unchanged in this slice.

## Next

1. Add a gRPC `plugin.GRPCPlugin` implementation beside the existing
   `netRPCPlugin` without changing v1 wire types or `pluginsdk.Serve`.
2. Select `VersionedPlugins` and `AllowedProtocols` exactly from the trusted
   Manifest protocol version; v2 must not silently downgrade to v1.
3. Implement typed handshake/health/readiness first, including exact identity,
   runtime token, limits, protocol mismatch, cancellation, and stale-token
   tests, then add the host broker.

## Compression Rule

Monitor context usage continuously. Before an expected context compression,
update the durable progress ledger and this handoff with percentages, exact
commits, dirty files, verification, and the next command; then commit every
coherent buildable slice before continuing.

## Open Questions

- None. The accepted V3 ADR and Host API V2 toolchain decision define the
  current product boundary.
