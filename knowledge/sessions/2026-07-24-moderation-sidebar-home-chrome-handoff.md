# 2026-07-24 Session Handoff

## Changed

- Rewrote `/moderation` left/right sidebars to use home public chrome tokens
  instead of a custom notifications-style rail.
- Shell: `sforum-home` + `sforum-home__layout--with-right` +
  `sforum-home__sidebar` / `__main` / `__right`.
- Left after-nav: new `ModerationWorkbenchNav` using
  `sf-home-navigation__label|link|count` (same as settings account nav).
- Right rails: `ModerationQueueRail` + rewritten `ModerationDecisionRail` use
  `sf-home-right-rail` cards.
- Host CSS: desktop `sforum-home__layout` now applies
  `--sf-public-edge-inset` so host-chrome pages (moderation is
  Replaceable:false) leave the viewport edge like the theme shell.
- Tests + typecheck pass.

## Decisions

- Do not invent moderation-only sidebar chrome; reuse home layout/nav/right-rail
  classes. Keep workbench-only styles in `sforum-moderation.css` for main-column
  content and decision controls.
- edge-inset lives on host `sforum-home__layout` so Replaceable:false pages
  without `.sf-theme--default` still align.

## Next

- Visual QA at desktop / ≤1100 / ≤980 for queue + review modes.
- Optional: notifications left rail could later migrate to
  `sf-home-navigation__link` for the same token language.

## Open Questions

- None blocking.
