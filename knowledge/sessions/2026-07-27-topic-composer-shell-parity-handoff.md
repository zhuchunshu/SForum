# 2026-07-27 Topic Composer Shell Parity Handoff

## Changed

- `forum.topic.edit` and `forum.topic.create` now use the same builtin-theme
  `fullwidth-3col` shell contract.
- `SFHostPublicChrome` now preserves the same public topbar tracks, 24 px
  viewport inset, 230 px left rail, 270 px right rail, column padding, and
  center-column scrolling when Page Registry falls back to Core.
- Theme completeness tests require create/edit templates in both builtin
  themes to declare the shared shell.
- Added the missing `sf-topic-editor` template-validator allowlist entry; it
  already existed in the production runtime binding map.
- CSS validation now rejects the legacy `behavior:` property by declaration
  boundary without falsely rejecting `overscroll-behavior:`.
- Rebuilt and activated default-theme digest
  `5d4387b97222dab25531247e84cb21d65f541dc537c6a1e776d6440ea82778f2`.

## Decisions

- Core fallback may change the page provider, but it must not introduce a
  visibly different public shell.
- Repository theme source is not runtime evidence. Completion requires
  staging, exact activation, provider/digest inspection, and rendered browser
  checks.

## Verification

- `go test ./app/Support/Pages`
- Focused Bun tests: 64 passed, 0 failed, 841 assertions.
- Browser: create/edit both resolve `sforum.default-theme`, template `1`,
  shared `fullwidth-3col`, matching desktop rails/padding/scrolling, no overlay,
  and no console errors.
- Edit preview/write interaction passed.

## Next

- Keep create/edit paired in future theme and shell changes.
- Run the exact-artifact activation checklist for every builtin-theme edit.

## Open Questions

- None for this fix.
