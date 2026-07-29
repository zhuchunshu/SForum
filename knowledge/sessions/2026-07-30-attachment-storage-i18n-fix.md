# 2026-07-30 Attachment Storage I18n Fix

## Changed

- Moved storage-instance UI messages from the incorrect `admin.home` namespace
  to `admin.attachments`, fixing raw translation keys on Attachment Configuration.
- Storage probe APIs now return localized administrator-facing messages while
  preserving stable `reason` codes and stored raw diagnostics.
- Storage-instance deletion now returns `200 { data: { deleted: true } }`
  instead of `204 No Content`, matching the shared Nuxt API envelope contract.
- The local-switch command is hidden when local storage is already active;
  switching a writer immediately synchronizes the main form's selection and
  then refreshes both settings and the instance list.
- Added frontend and Go regression coverage plus OpenAPI field descriptions.

## Decisions

- Machine-readable probe reasons are not rendered in the UI; client Toasts use
  the localized API message.

## Next

- Browser verification needs an authenticated administrator session; the
  available Browser session redirects to login.

## Open Questions

- None.
