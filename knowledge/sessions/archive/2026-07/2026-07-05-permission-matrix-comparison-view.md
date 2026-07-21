# 2026-07-05 Permission Matrix Comparison View

## Changed

- Updated the admin permission matrix so it no longer renders every user group
  as a visible column by default.
- Added a comparison-scope control with user-group search, explicit group
  selection capped at five groups, and a differences-only audit mode.
- Added localized Chinese and English copy for the comparison controls and
  empty states.
- Extended `tests/validate-identity-ui.js` so future changes keep the matrix
  from regressing to an unbounded horizontal table.

## Decisions

- Keep the Roles page as the primary editor for a single user group's
  permissions.
- Treat the Permissions page as an audit/comparison surface. Admins can compare
  a small number of groups directly, then use differences-only mode for review.
- Keep the API unchanged because this is a frontend presentation and
  usability improvement.

## Next

- If administrators later need full exports, add a dedicated export/action flow
  instead of making the default table show every group.

## Open Questions

- Should the comparison scope eventually support saved presets for common
  operator/moderator/admin group comparisons?
