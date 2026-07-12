# 2026-07-13 Session Handoff — E6.3 admin polish + E6.4 storage-fs

## Changed

### E6.3 Admin + Probe

- Attachment settings: plugin selection no longer falls through to SFTP forms
- Dedicated plugin panel: settings deep-link, secrets note, test connection
- Probe API returns optional `reason`; toast shows message · reason
- i18n zh-CN / en-US copy updated for post-RPC reality
- OpenAPI `AttachmentProbe.reason`

### E6.4 Reference plugin `sforum.storage-fs`

- Package: `extensions/builtin/plugins/sforum-storage-fs/`
- Slot `attachment.storage.provider`; settings `root_path`, `public_base_url`
- Chunked Storage* RPCs on local filesystem; path escape rejected
- Built by `scripts/build-builtin-plugins.sh`
- Authoring guide Reference 3 + scenario-map rows
- Unit tests + `sforum extension test` clean

## Decisions

- Ship filesystem reference first (CI-friendly); S3/MinIO remains same RPC
  surface for third parties without a core PR
- Core `local` stays zero-config default; plugin is a second filesystem backend

## Next

1. Optional E6.5: document core-driver retention / future OSS extraction
2. Optional: S3-compatible product plugin when MinIO CI is available
3. E7 search provider slot

## Open Questions

- Whether builtin should default `root_path` under data dir for one-click demo
  (v1 requires operator-set absolute path)
