# 2026-07-05 Attachment Local Root Runtime Option

## Status

Accepted.

## Context

The attachment foundation originally treated the local storage root as process
environment configuration. Attachment settings now need a single admin-managed
runtime surface for local storage behavior while keeping backend policy checks
authoritative.

## Decision

Store the local attachment provider root in `web_options` as
`attachment.local.root`, guarded by `attachment.settings.manage`.

- Default value: `storage/app/attachments`.
- Relative paths resolve from the API process working directory.
- The options service rejects empty roots, path traversal segments, control
  characters, and angle brackets.
- The local storage probe creates and removes a temporary file so provider tests
  verify the root is writable, not only creatable.
- Docker Compose mounts the uploads volume at `/app/storage/app/attachments`
  to match the default runtime option.

## Consequences

- `ATTACHMENT_LOCAL_ROOT` is no longer read by API process config or documented
  in env examples.
- Operators can move local attachment storage through the admin settings API/UI
  after preparing the target path and filesystem permissions.
- Deployments should keep volume mounts aligned with the configured option.
