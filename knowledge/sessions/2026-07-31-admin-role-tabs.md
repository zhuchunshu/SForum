# 2026-07-31 Admin Role Tabs

## Changed

- `/control-panel/roles` now separates the user-group list and extension
  permission reviews into `groups` and `approvals` fixed tabs.
- The route shell owns the query-synced tab selection and active toolbar. Each
  workflow lives in a focused identity-domain tab component.
- Permission reviews load only after their tab is first opened. An applied
  extension permission still refreshes Host role data and merges the exact
  grant into an existing dirty role draft without overwriting unrelated edits.

## Decisions

- Reused `SFAdminFixedTabNav` and the existing admin settings geometry instead
  of introducing another tab implementation.
- Kept the backend, OpenAPI, and permission contracts unchanged; this is a
  frontend responsibility and presentation split only.

## Next

- Manually verify both tabs, their toolbar states, `?tab=` refresh behavior,
  approval-to-group synchronization, and narrow-screen overflow.

## Open Questions

- None.
