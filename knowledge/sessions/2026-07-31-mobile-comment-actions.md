# 2026-07-31 Mobile Comment Actions

## Changed

- Mobile comment headers now place the public floor number and an accessible
  overflow trigger together at the right, with publication metadata on a
  dedicated second row.
- At `640px` and below, reply and permalink remain in the bottom action strip;
  edit, delete, report, and extension-provided actions move into the overflow
  menu. Desktop retains the complete inline action strip.
- Added localized overflow labels and a focused presentation regression test.

## Decisions

- The menu consumes the same permission-filtered action array and dispatches
  through the existing handler. API authorization, confirmations, and
  extension action contracts are unchanged.
- The Host component/style owns the responsive behavior, so the active
  immutable Page Registry theme continues mounting the same Host island.

## Verification

- `bun test tests/forum/commentMobileActions.test.ts tests/forum/forumTopicPresentation.test.ts tests/forum/defaultThemeTopicPage.test.ts tests/forum/defaultThemeTopicTypography.test.ts`: 15 passed.
- `bun run typecheck`: passed.
- `node tests/validate-architecture-boundaries.mjs`: passed.
- Browser QA on `/t/87` with `data-provider="sforum.default-theme"` and
  `data-template="1"`: `402x905` shows only reply/permalink inline and opens
  the report menu item; `1280x720` hides the mobile trigger and shows all three
  available inline actions. Both viewports have zero horizontal overflow and
  no console errors or warnings.

## Next

- None.

## Open Questions

- None.
