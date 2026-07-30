# 2026-07-31 Mobile Topic Editor Visibility

## Changed

- Topic create/edit now apply the existing `360px` desktop and `330px` mobile
  canvas minimums to `.sf-editor__body` as well as content, preview, and loading
  surfaces.
- Strengthened the composer main-column selector so its `112px` desktop and
  `118px` mobile bottom reservation outranks the later shared home padding.
  Before this correction the computed mobile padding was only `28px`, leaving
  the editor status row behind the fixed action dock at maximum scroll.
- Added a focused regression assertion for the shared composer CSS contract.

## Decisions

- Kept the fix local to `SFTopicComposerPage.css`; the shared `SFEditor`,
  comment drawer, and admin editor sizing remain unchanged.

## Verification

- `bun test tests/forum/defaultThemeTopicComposer.test.ts tests/forum/topicEditPage.test.ts`:
  15 passed, 0 failed.
- `git diff --check`: passed.
- Chrome at `402x905`: `/topics/new` and `/topics/87/edit` both compute `118px`
  main-column bottom padding. At maximum scroll the editor footer ends at
  `785.6px`, the fixed dock starts at `817px`, and the `31.4px` gap keeps the
  complete footer visible. Edit write/preview switching passed.
- Chrome at `1280x720`: the main column keeps `112px` bottom padding and the
  editor footer remains above the desktop dock.
- The current development pages still emit unrelated existing i18n missing-key
  and `SFNotificationPreview` resolution warnings; no relevant console errors
  were observed.

## Next

- None.

## Open Questions

- None.
