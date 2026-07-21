# 2026-07-13 Session Handoff — E6.2 storage RPC + PluginStorageAdapter

## Changed

### Protocol + runtime

- `Support/Extensions`: `PluginProtocol` Storage* methods (PutBegin/PutChunk,
  Open/GetChunk, Close, Delete, Stat, Exists, PublicURL/SignedURL, Probe)
- Payload types in `storage_protocol.go`; `ProtocolNoop` safe defaults
- `ProtocolStarter` net/rpc client/server + ctx-bounded `callStorage`
- `Manager` implements `StorageRuntime` with F2.3 gate + `DefaultStorageTimeout`
  (120s); circuit open / timeout exported errors
- Close sessions skip circuit accounting (best-effort cleanup)

### Host adapter + wiring

- `PluginStorageAdapter` in Extensions package (avoids Storage↔Extensions cycle)
- Default chunk 1 MiB; multi-chunk Put/Open; empty object Final empty chunk
- `Attachments.Service.WithStoragePluginRuntime`; `adapterForSettings` builds
  plugin adapter when `plugin:<id>` available
- Upload/Open map plugin RPC failures → `ErrStorageUnavailable`
- Bootstrap API + worker inject shared extension runtime into attachment service
- SDK `plugin.Noop` embeds `ProtocolNoop`; SMTP embeds same for new interface

### Tests

- Memory runtime round-trip + empty object + denied put
- Protocol helper StorageProbe not-implemented
- Attachments: fail-closed without runtime; adapter with stub runtime

## Decisions

- Keep Adapter interface in Support/Storage; adapter implementation lives in
  Extensions to break import cycle with Models/Extensions storage catalog
- No reference plugin in this slice (E6.4); Noop rejects storage by default
- Multi-backend migration still out of scope

## Next

1. **E6.3** — admin test-connection copy/toast polish if Probe UX incomplete
2. **E6.4** — S3-compatible or temp-dir reference plugin + authoring guide
3. Optional: fixture plugin for full go-plugin storage handshake CI

## Open Questions

- Whether admin Probe already surfaces plugin StorageProbe messages clearly
  enough for E6.3, or needs dedicated reason mapping
