# 2026-07-30 Comment User Preview Handoff

## Changed

- Added `SFCommentUserPreview` and changed eligible comment avatar/name clicks
  to open the compact A preview before navigating.
- The card reads `GET /profiles/:username`, caches successful results, and
  retains the existing identity plus public-profile link on read failure.
- Outside pointer input and Escape close the card; Escape restores trigger
  focus. The absolute card is anchored to every comment node, including nested
  replies, so it scrolls away with the comment.
- Added Chinese/English copy and focused presentation tests.

## Decisions

- Reused the public profile API and canonical `/u/:username` route. Follow and
  message actions remain absent because Core has no such contracts.
- Kept the existing comment row styling. The card is 340px wide when space
  allows and contracts within the mobile viewport.
- Did not use a continuously repositioned popover primitive because the
  accepted behavior requires the card to leave the viewport with its comment.

## Verification

- `bun test tests/forum/forumCommentPresentation.test.ts`: 16 passed.
- `bun run build`: completed; only the existing Nuxt Icon JSON import-attribute
  warnings were emitted.
- Browser component QA at 1440x900 and 390x844: open, outside close, Escape and
  focus restoration, exact scroll delta, card-internal profile navigation, and
  no horizontal overflow all passed.

## Next

- Once the concurrently changing API/extensions build starts cleanly, repeat
  the same interaction matrix on a real selected-theme topic detail route.

## Open Questions

- None for the implemented interaction contract.
