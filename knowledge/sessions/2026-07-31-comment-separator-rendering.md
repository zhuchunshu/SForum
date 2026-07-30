# 2026-07-31 Comment Separator Rendering Fix

## Changed

- Removed the bottom border from rows in the default flat comment stream. The
  line matched the reported faux-scrollbar position exactly and could appear
  fully or partially while the browser repainted hovered comment content.
- Added a focused CSS contract test that prevents the flat stream from
  reintroducing a bottom border.
- Removed the earlier speculative document clipping, main-column clipping, and
  code-block scrollbar overrides after confirming they did not address the
  reported element.

## Decisions

- Keep row spacing as the flat-stream separator. Tree-mode branch borders are a
  separate hierarchy affordance and remain unchanged.
- Preserve native rich-content code-block horizontal scrolling.

## Verification

- `bun test tests/forum/commentStreamVisualContract.test.ts tests/forum/defaultThemeTopicPage.test.ts`
- `bun run typecheck`
- Browser QA on `/t/59#comment-197` at `1920x1024` and `390x844`: the affected
  row computes a zero-width bottom border after hot reload, pointer movement
  over the code block does not reveal a separator, the mobile document remains
  free of horizontal overflow, and console warnings/errors remain empty.

## Next

- None.

## Open Questions

- None.
