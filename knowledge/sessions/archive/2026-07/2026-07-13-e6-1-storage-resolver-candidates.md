# 2026-07-13 Session Handoff — E6.1 storage resolver + candidates

## Changed

### Host selection + candidates

- `Support/Storage`: `Candidate`, `CoreCandidates`, `PluginCandidate`, `MergeCandidates`
- `Models/Options`: `normalizeAttachmentProvider` accepts `plugin:<extensionId>`
  (syntax + charset); plugin selection skips core cloud-secret validation
- `Models/Attachments`:
  - `StorageProviderCatalog` + `WithStorageProviderCatalog`
  - Settings/UpdateSettings return `candidates[]` (core + enabled plugins)
  - `ensureProviderSelectable` on save; `adapterForSettings` parses selection
  - Plugin path **fail-closed** (`ErrStorageUnavailable`) until E6.2 RPC
  - `ClearStorageProviderSelectionIfMatch` → write `local`
- `Models/Extensions`: `ListStorageProviderCandidates` /
  `IsStorageProviderAvailable`; drain calls storage clearer on disable
- Bootstrap: shared `attachmentService` with catalog + selection clearer
- OpenAPI: `AttachmentProvider` free string; `AttachmentStorageCandidate`;
  settings fields `providerSlot` / `drivers` / `candidates`
- Admin UI: provider select from `candidates`; plugin hint + settings deep-link

## Decisions

- No second selection table; keep `attachment.provider`
- Plugin transport still absent: selecting a plugin is allowed for wiring tests,
  but upload/open/probe fail closed with clear unavailability
- Disable any plugin clears storage selection only when value is
  `plugin:<thatId>`

## Next

1. **E6.2** PluginProtocol storage RPCs (chunked Put/Open) + SDK Noop
2. Host `PluginStorageAdapter` implementing `storage.Adapter`
3. E6.3 polish test-connection for plugins; E6.4 S3 reference plugin

## Open Questions

- Whether Probe should return a dedicated “RPC not implemented” reason vs generic
  storage_unavailable (v1 generic)
- Health field on candidates (runtime degraded) deferred to E6.2/E6.3
