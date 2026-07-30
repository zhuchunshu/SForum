# 2026-07-31 Forum Code Highlighting

## Changed

- Replaced the old token-only presentation with the selected Demo 02
  paper-line code block in topic, comment, moderation, and editor-preview
  content.
- Added language names and marks, sticky line numbers, internal scrolling,
  localized copy controls and Toast feedback, light/dark tokens, and an
  expanded `highlight.js` language catalog without adding a dependency.
- Fixed `/t/59`: an undeclared block now renders as `纯文本 / TXT` instead of
  being guessed as CSS, and selected-theme prose CSS can no longer add a nested
  rounded border to the enhanced `pre`.

## Decisions

- Unspecified code is plain text. Automatic detection is intentionally not
  used because short logs and procedural text are frequently misclassified.
- Unknown explicit language labels remain visible but do not select an
  unrelated grammar.

## Verification

- `bun test tests/forum/codeHighlight.test.ts tests/forum/defaultThemeTopicPage.test.ts`
  passed: 12 tests.
- `bun run typecheck` passed.
- `bun run build` passed.
- `/t/59` passed rendered desktop, `390x844` mobile, and dark-mode checks with
  no console errors, no page overflow, borderless inner `pre`, and working copy
  state plus success Toast.
- Architecture validation still reports only unrelated dirty-worktree ratchet
  changes in Identity service size and the admin users page baseline.

## Next

- No code-highlighting follow-up is required. Keep the focused test when
  extending language aliases or selected-theme prose rules.

## Open Questions

- None.
