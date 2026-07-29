# 2026-07-30 Daytime Background Admin Follow Handoff

## Changed

- Kept `appearance.light_background` in Core Personalization and expanded the
  accepted list from four to 12 localized presets.
- Connected the admin shell, Nuxt UI surfaces, and common Core admin white/slate
  utilities to the selected daytime palette.
- Replaced the preset cards' hidden native radio inputs with visible
  `button[role=radio]` controls. This prevents focus-driven viewport collapse in
  fixed admin shells while retaining keyboard focus and checked-state semantics.
- Added an admin-only in-memory appearance preview. Accent and daytime
  background choices now update the active admin shell immediately with a
  reduced-motion-aware color transition, but do not update `web_options` until
  the operator saves. Deactivating the Appearance surface clears the preview
  and restores persisted values; public routes never consume the preview.
- Kept every new runtime selector under `html:not(.dark)` so explicit and
  system-resolved dark mode retain the existing dark tokens.
- Focused Web tests pass, including the new preview ownership contract. Options and HTTP Go tests,
  OpenAPI reference validation, i18n JSON parsing, and `git diff --check` pass.
- Operator verification confirms the preset focus/viewport repair and the
  immediate unsaved admin preview both work as intended.

## Decisions

- Daytime background remains a Core-owned personalization setting. It does not
  move into the default theme or another built-in theme's settings.
- `pure_white` remains the recommended default and preserves existing installs.

## Next

- No remaining work for this appearance change.

## Open Questions

- None for implementation. Full typecheck and architecture gates remain blocked
  by pre-existing attachment, navigation, language, search, extension, identity,
  and manifest worktree changes outside this feature.
