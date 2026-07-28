# 2026-07-28 Notification Detail Type Navigation Handoff

## Changed

- Extracted the notification type rail into shared `SFNotificationTypeNav`.
- Added the same type menu and loaded-scope counts to notification detail on
  desktop and in the mobile left drawer.
- Detail highlights its notification type; selecting a menu item returns to
  the inbox with the shared filter state applied.

## Decisions

- Counts keep the inbox's existing first-loaded-page semantics rather than
  claiming an API-wide type total that the current contract does not expose.
- List and detail own their data loading, while the shared component owns only
  type labels, icons, count presentation, and selection events.

## Evidence

- Notification frontend suite: 28 passed.
- Nuxt typecheck, architecture boundary validation, and `git diff --check`
  passed.
- Browser QA was attempted through the required Browser path. The in-app
  browser had no authenticated session; the authenticated Chrome page was
  found, but Chrome control timed out after reload before DOM/screenshot and
  interaction evidence could be collected.

## Next

- Repeat desktop and 390x844 Browser verification when Chrome control is
  responsive: list row -> detail -> type menu -> filtered inbox.

## Open Questions

- None.
