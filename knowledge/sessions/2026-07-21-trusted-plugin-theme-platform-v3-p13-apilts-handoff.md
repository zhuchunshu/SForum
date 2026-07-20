# 2026-07-21 Session Handoff — V3 P13 APILTS Production Wiring

## Changed

- `apilts.Process()` process-local registry + `ProtocolV1ContractID`.
- `Registry.ShimCalls` / `CanRemoveWithZeroShim` deletion gate helper.
- `ProtocolStarterConfig.ShimTelemetry` optional inject; V1 start + RPC call
  record `sforum.protocol.v1`; V2 does not.
- API + worker bootstrap inject `apilts.Process()`.
- CLI: `sforum extension api-lts` / `--json`.
- Docs: `docs/extensions/v3/p13-migration-and-lts.md` production wiring section.

## Verified green

- `go test` APILTS, CompatFarm, Extensions (shim tests), cmd/sforum
- `go build ./bootstrap ./cmd/sforum`
- `go run ./cmd/sforum extension api-lts` smoke

## Decisions / accepted boundaries

- Deletion checklist items remain **open** until LTS window + zero shim:
  core Nuxt presentation move-out, request-time loader removal, v1 path removal.
- `SFPageOutlet` fail-closed emergency fallback is **never** fully removed.
- SMTP + storage-fs remain production V1; content-policy is V2.

## Next

1. Keep observing process APILTS counters in live API/worker after V1 traffic.
2. Do not close P13 deletion rows until `CanRemoveWithZeroShim` is true for a
   full published window (and all other checklist items remain true).
3. Optional later: admin HTTP inspector for live process snapshot (would bump
   route catalog past 244 — not required for residual honesty).

## Unowned dirty WIP (do not stage)

- route-inspector web/OpenAPI, content-policy manifest, PageViewModels, go.mod,
  host-api-v2.md, websocket revoke test, ADR noise.

## Rollback

- Revert `9bf9d93fb` … `e10eeae15` for this APILTS wiring chain.
