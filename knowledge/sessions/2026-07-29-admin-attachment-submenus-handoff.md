# 2026-07-29 Admin Attachment Submenus Handoff

## Changed

- Replaced the standalone Attachment Settings sidebar page item with a folder
  containing Basic Configuration and Attachment Management children.
- Added independent `/attachments/settings` and `/attachments/manager` pages,
  each with its own admin page registration, permission, toolbar, and refresh
  state.
- Kept `/attachments` as a permission-aware compatibility redirect that also
  honors old `?tab=manager` bookmarks.
- Mapped the stable attachment Admin Surface placement to the new management
  page so existing extension contributions remain available.

## Decisions

- Treat settings and governance as distinct routes while preserving the old
  entry URL and stable extension placement as compatibility boundaries.

## Next

- The operator will manually verify both submenu links and permission-specific
  visibility.

## Open Questions

- None.
