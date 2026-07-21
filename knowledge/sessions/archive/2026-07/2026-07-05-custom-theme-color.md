# 2026-07-05 Session Handoff

## Changed

- Extended `appearance.theme` so admins can keep a preset or save a controlled
  custom value such as `custom:#4f46e5`.
- Added a custom color card to the admin personalization page with a native
  color picker, HEX input, live swatches, and existing save/reset behavior.
- Added frontend helpers that normalize custom theme values and derive CSS
  variables for accent, hover, soft, focus, contrast, dark-mode accent, and the
  primary color scale.
- Updated the root app to write custom theme CSS variables on `<html>`.
- Bridged Nuxt UI's generated `--ui-color-primary-*` and `--ui-primary` tokens
  to SForum's runtime theme variables so admin sidebar highlights and
  `color="primary"` controls follow the personalization setting.
- Updated backend option validation and tests so `custom:#rrggbb` is accepted
  and invalid custom color strings are rejected.

## Decisions

- Keep using `appearance.theme` as the single public runtime option instead of
  adding a second custom-color option.
- Store custom colors as `custom:#rrggbb`; do not accept arbitrary CSS color
  strings.
- Treat Nuxt UI's app-config `primary: green` as a generated default and bridge
  the CSS tokens directly for runtime theme switching.

## Next

- If richer brand customization is needed later, add explicit secondary/surface
  options with backend validation rather than allowing free-form CSS.

## Open Questions

- None.
