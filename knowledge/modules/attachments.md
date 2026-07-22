# Attachments Module

## Purpose

Owns user-uploaded files, storage provider configuration, attachment metadata,
admin governance, and orphan cleanup.

## Current Status

Attachment system foundation is implemented.

- Backend module `apps/api/app/Models/Attachments` owns upload rules,
  metadata, settings DTO conversion, admin list/detail/status/delete actions,
  and orphan cleanup.
- HTTP controller `apps/api/app/Http/Controllers/Attachments` exposes user and
  admin APIs under `/api/v1`.
- Storage provider adapters live under `apps/api/app/Support/Storage`.
- Migration `202607050004_attachments.sql` creates `attachments`,
  `attachment_references`, and the attachment permissions.
- Admin UI `apps/web/app/pages/admin/attachments.vue` is registered as the
  standalone top-level "Attachment settings" page with "Basic Configuration"
  and "Attachment Management" tabs. The settings tab now leads with a
  beginner-friendly recommended local-upload configuration and provides a
  one-click restore-and-save action for the recommended defaults.
- OpenAPI contract includes upload, metadata, content, admin settings, provider
  test, list, detail, update, soft delete, and cleanup endpoints.

## Permissions

- `attachment.upload`: upload attachments. Granted to the default `member`
  role and to `super_admin`.
- `attachment.manage`: list, inspect, disable, restore, soft-delete, and clean
  up attachments. Granted to `super_admin`.
- `attachment.settings.manage`: manage storage provider and upload settings.
  Granted to `super_admin`.

Backend policy checks are authoritative. Frontend visibility only mirrors these
permissions for navigation and tab usability.

## Data Model

- `attachments` stores public id, owner, provider, object key, original file
  name, MIME type, extension, size, SHA-256, optional image dimensions,
  visibility, status, reference count, timestamps, and soft-delete timestamp.
- `attachment_references` records business references such as post, topic,
  avatar, or future resource types.
- `reference_count > 0` blocks physical deletion. Referenced attachments can be
  disabled/hidden, but not physically removed by cleanup.

## Provider Slot (F3.5 → E6)

Host contract slot: `attachment.storage.provider` (`Support/Storage.ProviderSlot`).

**Decision (E6.0 accepted):**
`knowledge/decisions/2026-07-12-attachment-storage-plugin-provider.md`

| Stage | Status |
| --- | --- |
| E6.0 contract + selection encoding | **Done** — `plugin:<extensionId>` helpers in `Support/Storage` |
| E6.1 resolver / candidates / restore | **Done** — settings `candidates[]`, options accept `plugin:`, disable fallback to `local` |
| E6.2 chunked storage RPC | **Done** — PluginProtocol Storage* + `PluginStorageAdapter` + Manager gate/timeout |
| E6.3 admin select/test/settings polish | **Done** — plugin panel (no core secret forms), Probe `reason`, toast detail |
| E6.4 reference storage plugin | **Done** — builtin `sforum.storage-fs` (filesystem; S3-shaped RPC, no cloud SDK) |

**Runtime today (L4–L6 for storage slot):** concrete drivers remain **in core**
under `Support/Storage`. Operators select via `attachment.provider` (core id or
`plugin:<extensionId>`). Admin settings return `providerSlot`, `drivers[]`, and
`candidates[]`. Plugin path uses chunked RPC; Probe returns `reason` + message.
Reference: enable `sforum.storage-fs`, select `plugin:sforum.storage-fs`, set
root path, test connection, upload. Disable plugin → selection falls back to
`local`. Multi-backend migration and browser presigned upload remain out of
scope for E6.

## Runtime Configuration

Attachment product configuration is stored in `web_options`, including the
local provider filesystem root.

- `attachment.local.root` defines the local provider filesystem boundary. It
  defaults to `storage/app/attachments`; relative paths resolve from the API
  process working directory.
- The recommended beginner configuration is local storage, uploads enabled,
  20 MB per file, common image/PDF/TXT/ZIP allow-lists, public visibility, and
  30-day orphan retention.
- Admins with `attachment.settings.manage` can configure the local root, object
  path templates, and public URL prefixes. The API rejects empty paths,
  traversal segments, control characters, and angle brackets.
- Secret options are masked in admin responses. Blank secret updates keep the
  existing value.
- Public web options expose only upload knobs needed by the frontend:
  `attachment.upload.enabled`, `attachment.max_file_size_mb`,
  `attachment.allowed_extensions`, and `attachment.allowed_mime_types`.

Important runtime options:

- `attachment.provider`
- `attachment.upload.enabled`
- `attachment.path_template`
- `attachment.public_base_url`
- `attachment.max_file_size_mb`
- `attachment.allowed_extensions`
- `attachment.allowed_mime_types`
- `attachment.default_visibility`
- `attachment.cleanup_orphan_after_days`
- `attachment.local.root`

## Storage Providers

SForum owns a small `StorageAdapter` interface:

- `Put`
- `Open`
- `Delete`
- `Stat`
- `Exists`
- `PublicURL`
- `SignedURL`
- `Probe`

Supported providers in the first implementation:

- `local`: local filesystem under `attachment.local.root`.
- `aliyun_oss`: Aliyun OSS through `github.com/aliyun/aliyun-oss-go-sdk/oss`.
- `tencent_cos`: Tencent Cloud COS through
  `github.com/tencentyun/cos-go-sdk-v5`.
- `ftp`: FTP through `github.com/jlaffaye/ftp` v0.2.0, chosen to keep project
  Go compatibility at 1.25.7.
- `sftp`: SSH/SFTP through `github.com/pkg/sftp` and `golang.org/x/crypto/ssh`.

The first version uses server-mediated multipart upload only. Browser direct
upload and presigned upload credentials are intentionally deferred.

## Upload And Cleanup

- Upload requires a logged-in active actor with `attachment.upload`.
- The API enforces configured max size, allowed extensions, and allowed MIME
  types.
- Avatar uploads reuse the attachment storage pipeline, but apply avatar
  runtime rules (`avatar.max_size_kb`, `avatar.max_dimension`,
  `avatar.allow_gif`, `avatar.compress_enabled`, `avatar.target_dimension`,
  and `avatar.compress_quality`) instead of the general attachment allow-list.
  Avatar attachments are always stored with public visibility.
- The service generates a random public id and object key from the configured
  path template, streams the object to the provider, and computes SHA-256 during
  upload.
- If metadata insert fails after object upload, the service best-effort deletes
  the remote object.
- Admin delete is soft delete. Orphan cleanup only physically deletes rows with
  `status = deleted`, `reference_count = 0`, and `deleted_at` older than
  `attachment.cleanup_orphan_after_days`.
- Profile avatar replacement writes `attachment_references` with resource
  `user` and context `avatar`, decrementing the old avatar reference and
  incrementing the new one so referenced avatars are not orphan-cleaned.
- Forum topic/comment content now submits explicit `content.attachmentIds`.
  The forum write transaction validates active public attachments owned by the
  editor, replaces references, and updates `reference_count`; topic/comment
  deletion clears the corresponding references.
- Forum content revisions V1 has completed M1 schema/backfill groundwork.
  Revision rows now have `attachment_ids BIGINT[]` snapshots for new creates and
  backfilled current rows. They store IDs only, never bytes, URLs, credentials,
  object keys, checksums, or provider internals. Restore remains M4 and must
  only rebind IDs from the selected immutable revision that are still active and
  valid for the original resource author/current visibility policy; unavailable
  IDs fail closed with `forum.revision_attachment_unavailable` and no partial
  write.
- Public attachment reads resolve real topic/comment/post status, category
  visibility, author, and reviewer permission. Pending is author/reviewer-only;
  hidden/deleted or hidden-category media is reviewer-only. Forum and
  unreferenced media always use the API proxy, while avatar/SEO/site assets may
  use permanent provider URLs.
- Logo/favicon/apple-touch-icon options atomically maintain `site` references;
  migration `202607130001_site_attachment_references.sql` backfills valid
  historical option values.
- `attachments.cleanup_orphans` is defined as a River maintenance job contract;
  worker wiring can be enabled when the worker process begins registering
  module jobs.

## API

User-facing:

- `POST /api/v1/attachments`
- `POST /api/v1/profile/avatar` for avatar-specific image upload and profile
  attachment.
- `GET /api/v1/attachments/:publicId`
- `GET /api/v1/attachments/:publicId/content`

Admin:

- `GET /api/v1/admin/attachment-settings`
- `PUT /api/v1/admin/attachment-settings`
- `POST /api/v1/admin/attachment-settings/test`
- `GET /api/v1/admin/attachments`
- `GET /api/v1/admin/attachments/:id`
- `PATCH /api/v1/admin/attachments/:id`
- `DELETE /api/v1/admin/attachments/:id`
- `POST /api/v1/admin/attachments/cleanup`

Public attachment DTOs omit storage-only fields such as object key and SHA-256.
Admin DTOs include object key, checksum, visibility, deleted timestamp, and
references for governance.

## Tests

Current focused coverage includes:

- attachment option permission checks, normalization, invalid path templates,
  MIME/extension validation, secret mask, and blank-secret retention;
- local adapter put/open/stat/delete and unsafe key rejection;
- upload metadata creation, SHA-256 calculation, remote rollback on DB failure,
  permission denial, invalid extension, avatar JPEG compression/public
  visibility, GIF rejection when disabled, and cleanup retention cutoff;
- existing backend HTTP/options/identity tests;
- frontend typecheck and Bun tests.

## Next Steps

- Wire `attachments.cleanup_orphans` into `bootstrap.NewWorker` once module job
  registration has a dependency assembly pattern.
- Add attachment reference writes from post/topic/avatar features when those
  modules land.
- Add optional image thumbnail/transcoding and virus scanning as separate
  background capabilities.
- Consider WebDAV/rclone and direct browser upload only after the first
  provider set is stable.

## SEO Assets (2026-07-11)

- `POST /api/v1/admin/seo/assets` requires `seo.manage`, accepts image uploads,
  reuses the configured attachment storage provider, and always creates public
  attachments.
- SEO contexts use `attachment_references` with resource type `seo`, resource
  id `0`, and context such as `seo/home-og-image`. Replacement atomically
  decrements the prior attachment reference count and increments the new one.
- A unique `(resource_type, resource_id, context)` index prevents concurrent
  replacements from leaving multiple active references for one SEO field.
- Restoring SEO recommended defaults clears form URLs but does not delete
  uploaded attachments; the UI states this explicitly.
