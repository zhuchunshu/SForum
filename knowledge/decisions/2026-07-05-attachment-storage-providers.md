# Decision: Attachment Storage Providers

## Status

Accepted

## Context

SForum needs an attachment system that supports local upload, cloud object
storage, legacy FTP environments, and remote servers without turning the core
application into a storage SDK. The project guidance prefers mature
third-party libraries and clear module boundaries.

## Decision

Use a small SForum-owned `StorageAdapter` interface and provider-specific thin
adapters.

First provider set:

- Local filesystem under `ATTACHMENT_LOCAL_ROOT`.
- Aliyun OSS via `github.com/aliyun/aliyun-oss-go-sdk/oss`.
- Tencent Cloud COS via `github.com/tencentyun/cos-go-sdk-v5`.
- FTP via `github.com/jlaffaye/ftp` v0.2.0.
- SFTP via `github.com/pkg/sftp` and `golang.org/x/crypto/ssh`.

FTP is pinned to v0.2.0 because v0.2.1 requires Go 1.26, while the project
currently targets Go 1.25.7. The v0.2.0 API still supports the adapter methods
SForum needs, including context dialing, EPSV control, explicit TLS, file size,
retrieve, store, and delete.

The first release keeps uploads server-mediated. It does not expose cloud
direct-upload credentials or browser presigned upload flows.

## Consequences

- Domain code depends on `StorageAdapter`, not provider SDKs.
- Provider credentials stay in masked `web_options` secret fields.
- Local filesystem writes stay bounded by `ATTACHMENT_LOCAL_ROOT`; admins only
  configure object path templates and public URL prefixes.
- FTP remains available for older deployments, but SFTP is the preferred
  "remote server" provider.
- Go CDK `blob`, WebDAV, rclone, direct upload, image processing, and virus
  scanning remain future extension points.
