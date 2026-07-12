# 2026-07-12 Session Handoff — E6.0 storage provider decision

## Changed

### Decision

- `knowledge/decisions/2026-07-12-attachment-storage-plugin-provider.md`
  - Slot stays `attachment.storage.provider`; target L4–L6
  - Business code keeps `storage.Adapter`; host wraps plugin RPC
  - Selection: keep `attachment.provider`; core ids or `plugin:<extensionId>`
  - Core keeps `local` (+ existing drivers until optional E6.5)
  - Chunked RPC sketch (1 MiB), fail-closed upload, no multi-backend migration
  - URL/ACL host-owned; secrets in extension_settings for plugins
  - Reference plugin: S3-compatible (MinIO); AWS SDK only inside plugin

### Host boundary (minimal code)

- `apps/api/app/Support/Storage/selection.go` (+ tests)
- `slot.go` comments + `IsKnownDriver` rejects `plugin:` as core driver
- Catalog note in `sdk/plugin/docsgen.go` (plugin-implementable wording)

### Knowledge

- Plan E6.0 checked; modules attachments/extensions; index + this handoff

## Decisions

- Single option key `attachment.provider` (no second selection table in v1)
- Mandatory `plugin:` prefix for extension selection
- Disable selected plugin → prefer auto-fallback to `local` (E6.1 wires)

## Next

1. **E6.1** — ResolveSelection into Attachments service / adapter factory;
   candidates API; restore defaults; options validation accepts
   `plugin:<id>` when extension enabled + slot declared
2. **E6.2** — PluginProtocol storage RPCs + SDK Noop
3. No E6.4 reference binary until RPC exists

## Open Questions

- Exact chunk RPC message shape (fixed 1 MiB vs negotiated)
- Whether Stat/Exists are required on protocol day one or optional
