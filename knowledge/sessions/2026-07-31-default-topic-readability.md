# 2026-07-31 Default Topic Readability Handoff

## Changed

- Added a semantic 12/14/14/16px typography scale to the default theme.
- Increased topic and comment bodies to 16px and raised meaningful metadata,
  composer guidance, comment actions, and right-rail labels to readable sizes.
- Changed publication and comment metadata from muted to secondary text color.
- Added a focused default-theme typography contract test and refreshed the
  immutable source-package digest.
- Kept the mobile discussion heading aligned like desktop: reply count and
  title stay on the left, the latest-reply link stays on the right, and the
  section lead-in plus heading height no longer consume a large blank block.
- Applied the row contract to Core fallback and both default-theme skin assets,
  with a focused mobile geometry regression assertion.

## Verification

- `bun test tests/forum/defaultThemeTopicTypography.test.ts tests/forum/defaultThemeTopicPage.test.ts`
  passed (10 tests).
- Default-theme `extension validate` and `extension test` passed with no
  errors or warnings.
- Rebuilt and reactivated the immutable built-in theme artifact. The active
  skin digest is `33296d3dd6a4dbb3feb385e1cf195191e70092f598c03392292c0254dd37d9a6`.
- Chrome desktop QA on `/t/1` confirmed `data-provider="sforum.default-theme"`
  and `data-template="1"`. Computed sizes are 14px for publication metrics,
  16px for topic/comment bodies, 14px for reply controls, and 12px for the
  reply identity hint. No console errors or warnings were present.

## Open Questions

- The mobile geometry source package is staged as a new immutable candidate;
  final local runtime reactivation and 402px Browser evidence await explicit
  confirmation in the open admin dialog.
