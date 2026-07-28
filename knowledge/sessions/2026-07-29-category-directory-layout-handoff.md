# 2026-07-29 Category Directory Layout Handoff

## Changed

- Removed the duplicate desktop center-column sticky offset that placed the
  category heading roughly one navbar height below its intended top padding.
- Replaced the left-rail category-group list with a compact center-toolbar
  selector that preserves the existing `?group=<id>` URL contract.
- The selector uses a non-empty internal value for "all groups" because Reka
  UI reserves the empty string for clearing a select; choosing it still removes
  the `group` query parameter.
- Added category filters matching the tag-directory navigation: all, hot,
  created within seven days, and A-Z. Desktop sidebar, center toolbar, and the
  mobile left drawer share one filter state.
- Reduced desktop and mobile right rails to the directory overview and five
  most active categories; removed group distribution and instructional tips.

## Decisions

- Keep the left rail for compose, global navigation, and four compact directory
  filter modes. The potentially unbounded group selection remains beside the
  directory search instead of returning to the rail.
- Fix the shared three-column geometry rather than compensating with a
  category-heading margin.

## Next

- Rebuild, stage, and activate the default-theme immutable artifact, then
  manually verify `/categories` on desktop and mobile.

## Open Questions

- None in source. Browser, build, typecheck, and automated tests were not run
  by explicit request.
