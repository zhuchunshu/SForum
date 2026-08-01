# 2026-08-01 Deleted Comment Placeholder

## Changed

- Shared `SFComment` now checks the comment lifecycle status and renders a
  localized deletion notice block for `deleted` comments.
- Added a themed message-card treatment with an icon, title, description, and
  Chinese/English locale entries.
- Added a focused presentation contract test.

## Decisions

- Keep the deleted comment row and its metadata/replies visible when the API
  returns the row; replace only the removed body with the notice block.

## Verification

- `bun test tests/forum/forumCommentPresentation.test.ts`
- `bun run typecheck`
- Locale JSON parse validation

## Next

- None.

## Open Questions

- None.
