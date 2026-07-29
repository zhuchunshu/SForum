# Multi-Instance S3-Compatible Attachment Storage

Date: 2026-07-30

## Context

One plugin must support several AWS S3, MinIO, Cloudflare R2, or compatible
accounts at the same time. Operators need one-click switching for new uploads,
while existing attachments must remain readable from the service that received
them. Provider credentials cannot live in attachment rows or ordinary options.

## Decision

- Core adds a generic named-instance contract for providers declaring
  `multiInstance: true`; vendor-specific configuration and object operations
  remain in plugins.
- `attachment.provider=instance:<uuid>` identifies the exact instance used for
  a write. Attachments retain that value permanently; changing the current
  writer affects only new attachments.
- Core owns instance metadata, configuration revision checks, permissions,
  probe/activation APIs, deletion protection, and routing. A focused adapter
  keeps these additions out of the legacy extension facades.
- Secret fields are stored in Host SecretStore under revision-specific opaque
  references. API responses return empty values plus `secretSet`; failed
  optimistic updates cannot replace credentials used by the active revision.
- Activation requires a real provider probe. A failed probe leaves the current
  writer unchanged. Core `local` remains the one-click recommended fallback.
- An instance cannot be deleted while active or while any attachment records
  it. No implicit fallback reads, dual writes, or cross-instance copy occur.
- The protected `sforum.storage-s3` plugin uses AWS SDK for Go v2 and supports
  custom endpoints, region, bucket, credentials/default chain, path style,
  object prefix, public URLs, signed URLs, and write/read/stat/delete probing.
- Protected FTP and SFTP built-ins are removed. Protocol-oriented storage can
  return later as optional third-party plugins if operator demand justifies it.

## Consequences

Core changes are required, but only for reusable provider-instance semantics;
AWS, R2, and MinIO logic does not enter Core. Historical reads depend on the
owning plugin remaining enabled and its instance credentials remaining valid.
Data migration between instances is a separate explicit workflow.
