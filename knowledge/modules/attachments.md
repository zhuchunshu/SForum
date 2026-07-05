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
- The service generates a random public id and object key from the configured
  path template, streams the object to the provider, and computes SHA-256 during
  upload.
- If metadata insert fails after object upload, the service best-effort deletes
  the remote object.
- Admin delete is soft delete. Orphan cleanup only physically deletes rows with
  `status = deleted`, `reference_count = 0`, and `deleted_at` older than
  `attachment.cleanup_orphan_after_days`.
- `attachments.cleanup_orphans` is defined as a River maintenance job contract;
  worker wiring can be enabled when the worker process begins registering
  module jobs.

## API

User-facing:

- `POST /api/v1/attachments`
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
  permission denial, invalid extension, and cleanup retention cutoff;
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
