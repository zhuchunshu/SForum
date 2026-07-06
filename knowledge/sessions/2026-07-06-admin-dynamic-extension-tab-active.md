# 2026-07-06 Admin Dynamic Extension Tab Active

## Changed

- Fixed admin multi-tab active synchronization for dynamic extension admin
  routes such as `/extensions/sforum.default-theme/pages/about`.
- Added route-backed placeholder tabs for dynamic extension pages so direct
  opens and switching back to an existing dynamic tab mark the correct tab
  active immediately.
- Updated the Extensions sidebar folder active/open rule so
  `/extensions/{id}/pages/*` keeps the extension navigation group highlighted.
- Extended the admin framework validation script to cover dynamic extension
  route recognition and sidebar open behavior.

## Decisions

- Dynamic extension admin pages remain custom tabs rather than static registry
  pages. The dynamic page component still owns the final label/icon from
  manifest data, while the layout owns route-level active synchronization.

## Next

- When adding richer extension page views, keep them under the existing
  `/extensions/{id}/pages/*` namespace so this route-backed tab behavior
  continues to apply.

## Open Questions

- None.
