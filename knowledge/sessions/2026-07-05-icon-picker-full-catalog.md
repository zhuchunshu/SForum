# 2026-07-05 Icon Picker Full Catalog Handoff

## Changed

- Reworked `SFIconPicker` so Tabler and Nuxt/Lucide choices come from the full
  local Iconify collections instead of a small hard-coded preset list.
- Added `/api/icon-collections/:collection` in the Nuxt server layer. It returns
  icon names in clamped pages, supports search, and maps `nuxt` to the local
  Lucide collection while preserving saved `i-lucide-*` values.
- The picker now loads 60 names at a time, auto-loads the next page when the
  option grid scrolls near the bottom, and registers only the visible icons with
  Iconify via Nuxt Icon's local `/api/_nuxt_icon/:collection` endpoint before
  rendering.
- Added focused Bun tests for catalog paging, search, normalization, dedupe, and
  scroll pagination helpers.

## Decisions

- Keep saved values as plain Nuxt Icon strings (`i-tabler-*` / `i-lucide-*`).
- Do not bundle all 6000+ Tabler SVG bodies into the initial client payload.
  Load the full name catalog in pages and prime only visible SVG data.
- This Nuxt server route is frontend infrastructure only; it does not expose or
  mutate protected forum resources and needs no new permission key.

## Verification

- `bun test tests/iconCatalog.test.ts`
- `bun test`
- `bun run typecheck`
- Browser QA with headless Google Chrome on `/components`: initial Tabler page
  showed 60/6194 icons with non-empty CSS masks, grid scroll loaded 120/6194,
  searching `database` selected `i-tabler-database`, and mobile viewport width
  had no horizontal overflow.

## Next

- Reuse `SFIconPicker` for category/navigation/badge/profile settings without
  adding per-feature icon lists.

## Open Questions

- None for the picker itself. Future settings screens still need their own
  product decisions for which option keys store chosen icon names.
