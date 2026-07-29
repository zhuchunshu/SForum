# 2026-07-29 Admin Attachment Submenus Handoff

## Changed

- Replaced the standalone Attachment Settings sidebar page item with a folder
  containing Basic Configuration and Attachment Management children.
- Added independent `/attachments/settings` and `/attachments/manager` pages,
  each with its own admin page registration, permission, toolbar, and refresh
  state.
- Renamed the settings child to Attachment Configuration and split its content
  into query-synchronized Basic Configuration and Compression Configuration
  tabs. The compression tab reports the current no-processor state without
  inventing persistence fields.
- Kept `/attachments` as a permission-aware compatibility redirect that also
  honors old `?tab=manager` bookmarks.
- Added server-backed button pagination to Attachment Management, including
  list range/total feedback, first-page filter resets, and last-page recovery
  after cleanup removes the current page.
- Mapped the stable attachment Admin Surface placement to the new management
  page so existing extension contributions remain available.
- Completed the Basic Configuration field guidance: all common and Core
  provider inputs now have persistent help text, size/retention controls show
  units, FTP toggles explain their behavior, and secret fields explain that a
  blank value preserves the saved credential. Extracted the provider-specific
  fields into a focused component to keep the tab shell cohesive.

## Decisions

- Treat settings and governance as distinct routes while preserving the old
  entry URL and stable extension placement as compatibility boundaries.

## Next

- The operator will manually verify both submenu links, permission-specific
  visibility, the Basic Configuration help text and units, and Attachment
  Management paging with more than 20 matching files.

## Open Questions

- None.
