# 2026-07-05 Icon Picker Handoff

## Changed

- Added reusable `SFIconPicker` for admin/user setting forms that need an icon
  field.
- Added `@iconify-json/tabler` as a frontend runtime dependency and configured
  Nuxt Icon to serve local `lucide` and `tabler` collections explicitly.
- Added an Icons section to the dev-only `/components` preview page.
- Tightened component preview mobile widths so the icon picker and existing
  preview cards do not create horizontal page overflow.

## Decisions

- Icon values are stored as plain Nuxt Icon names, such as `i-tabler-settings`
  or `i-lucide-settings-2`.
- The picker includes a curated preset grid plus a custom input for any valid
  Iconify/Nuxt Icon name.

## Next

- Reuse `SFIconPicker` when category, navigation, badge, or profile settings
  gain icon fields.

## Open Questions

- Which backend option keys should eventually store site/admin navigation icon
  choices.
