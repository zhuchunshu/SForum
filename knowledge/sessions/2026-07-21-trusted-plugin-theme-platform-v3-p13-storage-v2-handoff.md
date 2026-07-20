# 2026-07-21 Session Handoff — V3 P13 Storage Protocol V2

## Changed

- Host `protocolV2Client` implements all `Storage*` methods over known-slot
  `ProviderCall` (`attachment.storage.provider`) with base64 chunk payloads
  (`b310e68eb`).
- Default `sforum.storage-fs` migrates to Protocol V2 (`f3eba05cc`):
  - `plugin_v2.go` ProviderCall ops: probe, put_begin, put_chunk, open,
    get_chunk, close, delete, stat, exists, public_url, signed_url
  - V1 rollback: `sforum.extension.v1.json` + `-tags protocol_v1`
- Integration tests (`12b2feb62`):
  - `TestProtocolV2StorageBuiltinRoundTrip`
  - V1 matrix row retargeted to storage-fs rollback fixture

## Verified green

- `go test` storage-fs backend unit tests
- `go test ./app/Support/Extensions/ -run TestProtocolV2StorageBuiltinRoundTrip`
- `go test ./app/Support/Extensions/ -run TestProtocolV1BuiltInCompatibilityPackages`
- V1 rollback binary builds with `-tags protocol_v1`

## Decisions / accepted boundaries

- Storage V2 uses ProviderCall + base64 (not TransferFile) to mirror SMTP and
  keep `PluginStorageAdapter` / `storage.Adapter` unchanged.
- LTS deletion checklist items remain **open** (core Nuxt presentation,
  request-time loader, v1 paths) until zero-shim telemetry window.
- `SFPageOutlet` fail-closed emergency fallback is never fully removed.

## Next

1. Do not close P13 deletion rows until `CanRemoveWithZeroShim` is true for a
   full published LTS window.
2. Optional: re-run full `go test ./...` / `./scripts/test.sh` when machine free.
3. Unowned dirty WIP must not be staged (route-inspector, content-policy, etc.).

## Rollback

- Revert `12b2feb62` then `f3eba05cc` then `b310e68eb` for this storage chain.
