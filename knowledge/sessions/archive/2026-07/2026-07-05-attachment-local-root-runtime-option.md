# 2026-07-05 Attachment Local Root Runtime Option Handoff

## Changed

- Moved the local attachment storage root from process env config to the
  admin-managed `attachment.local.root` runtime option.
- Updated backend settings DTOs, local storage adapter probing, admin UI,
  Compose mount paths, OpenAPI, and env examples.

## Decisions

- Default local root is `storage/app/attachments`.
- Relative roots resolve from the API process working directory.
- Local provider tests create the directory and verify it is writable.

## Next

- Keep deployment volume mounts aligned with the saved runtime option.

## Open Questions

- None.
