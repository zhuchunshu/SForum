# 2026-07-05 Session Handoff

## Changed

- Implemented the attachment system foundation across backend, frontend,
  migrations, config, Compose, OpenAPI, tests, and knowledge docs.
- Added backend `attachments` module, storage adapters, admin/user routes,
  attachment settings stored in `web_options`, and a River maintenance job
  contract for orphan cleanup.
- Added standalone admin top-level page "Attachment settings" with Basic
  Configuration and Attachment Management tabs.
- Added permissions `attachment.upload`, `attachment.manage`, and
  `attachment.settings.manage`.

## Decisions

- Use a small internal `StorageAdapter` interface with thin provider adapters.
- Treat "remote server" as SFTP in the first release, while keeping FTP as a
  compatibility provider.
- Keep local filesystem root in `ATTACHMENT_LOCAL_ROOT`; admin settings manage
  object paths and public URL prefixes only.
- Use server-mediated multipart upload for v1; direct browser upload is
  deferred.

## Verification

- `go test ./...` passed in `apps/api`.
- `bun run typecheck` passed in `apps/web`.
- `bun test` passed in `apps/web`.
- `contracts/openapi.yaml` parses as YAML.
- `apps/web/i18n/locales/zh-CN.json` and `en-US.json` parse as JSON.

## Next

- Wire `attachments.cleanup_orphans` into the worker runtime once module worker
  dependencies are assembled there.
- Add reference writes when post/topic/avatar upload usage lands.
- Add optional thumbnail/transcoding and virus scanning as separate future
  jobs.

## Open Questions

- Whether physical cleanup should also handle uploaded objects whose metadata
  insert failed and provider rollback failed, or whether that waits for a
  dedicated remote-object reconciliation job.
