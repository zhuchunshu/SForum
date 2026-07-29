# 2026-07-30 Multi-Instance S3 Storage Handoff

## Changed

- Added Host-owned attachment storage instances, revisioned settings,
  SecretStore references, management APIs, OpenAPI contracts, and admin UI.
- Added `instance:<uuid>` routing to upload, read, cleanup, public URL, and
  signed URL flows; historical attachments retain their original instance.
- Extended Protocol V2 and the Go plugin SDK with instance-aware object calls,
  draft probes, configuration, removal, and isolated per-instance backends.
- Added protected `sforum.storage-s3` using AWS SDK v2 for AWS S3, MinIO,
  Cloudflare R2, and compatible services.
- Prevented the multi-instance S3 plugin itself from appearing as a writable
  attachment provider. Only configured `instance:<uuid>` selections are now
  accepted by both Admin and API validation.
- Added a plugin-list **Configure storage** action that opens Attachment
  Configuration with the S3 instance editor ready, and moved provider-specific
  public URL and credential editing fully into that instance Schema flow.
- New built-in storage-provider plugins now start installed instead of enabled.
  Existing enabled statuses and the filesystem reference plugin remain intact
  so historical `plugin:sforum.storage-fs` attachments stay readable.
- Fixed the create-instance locale path: Attachment Settings now resolves the
  multi-instance provider name, field labels, descriptions, placeholders, and
  options using the request locale instead of the English fallback Schema.
- Extended the generic provider-field contract with `required`; S3 `1.0.1`
  declares Bucket as required and provides clearer AWS, R2, MinIO, credential,
  prefix, and secret-retention guidance.
- The instance editor now marks required/optional fields, hides noisy empty
  defaults, explains that draft probing needs no save, and keeps the current
  probe result visible until any field changes. Draft probing still performs a
  real temporary object write/read/delete cycle.
- Removed protected FTP/SFTP plugins and rewired local/Docker build manifests.
- Fixed the local built-in staging sync so excluded backend binaries from
  removed plugins cannot leave manifest-less directories that break API boot.
- Made removed built-in synchronization publication-aware: FTP/SFTP are
  removed from the latest runtime desired set and hidden from the catalog while
  immutable historical version identities remain retained.
- Patch-bumped existing protected executable plugins because the shared plugin
  SDK contract changed, then refreshed the release baseline.

## Decisions

- Core owns only generic instance authority, routing, secrets, and selection;
  S3-compatible vendor behavior stays in the plugin.
- Switching affects new writes only. Instances referenced by historical
  attachments cannot be deleted.
- Activation is probe-gated and failed probes never change the writer.

## Verification

- Focused Go tests passed for bootstrap, attachments, extension catalog,
  manifest schema, runtime adapter, storage selection, and plugin SDK.
- S3 backend compiled; `sforum extension digest --write` and
  `sforum extension test` passed with zero errors and warnings.
- OpenAPI refs, built-in release contracts, Vue SFC parsing, locale JSON, and
  focused Bun attachment settings tests passed.
- Architecture validation passes after lowering the Options baseline and
  extracting page reconciliation, Protocol V2 provider invocation, and Page
  Registry provider management into focused owners. All four affected
  large-file ratchets were lowered to their new actual sizes.
- The full repository gate now passes release, architecture, built-in release,
  and the S3-related Go suites. The two new storage-instance API error codes
  have server-side Chinese and English messages and Localization tests pass.
  The aggregate Go stage remains blocked by the pre-existing shared development
  database's active V2 `sforum.default-theme` row; the formal SEO ZIP chain is
  documented as passing on a clean migrated database in the Manifest V3
  handoff, and its fail-closed rejection of this stale row is expected.
- The staging cleanup and published-built-in prune regression tests, migration,
  and API startup were not run for the follow-up fixes at the user's request;
  manual verification remains.
- The provider-form follow-up ran JSON parsing, scoped diff checks, built-in
  source-baseline regeneration, and OpenAPI reference validation only. Browser
  QA and test suites were intentionally left to manual verification.

## Next

- Apply migration `202607300001_attachment_storage_instances.sql` in the normal
  deployment flow, enable `sforum.storage-s3`, and create/probe an instance.
- Existing installations may disable `sforum.storage-fs` after confirming no
  historical attachments still use `plugin:sforum.storage-fs`.
- Runtime operator QA against real AWS S3, MinIO, or R2 credentials remains.

## Open Questions

- Cross-instance copy/migration and optional direct browser uploads remain
  separate future work.
