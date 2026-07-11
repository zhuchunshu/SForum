# 2026-07-12 Session Handoff · Taxonomy page demos

## Changed

- Kept explore hub series under `tmp/demo/grok/tags-demo` (08 + 08B–08H).
- Added full-list demos under `tmp/demo/grok/taxonomy-demo` (T01–T05 tags, C01–C05 categories).
- User selected **T01** (tag weight cloud) + **C04** (category tile grid) as the
  public `/tags` and `/categories` UI direction.

## Decisions

- See `knowledge/decisions/2026-07-12-public-taxonomy-pages-ui.md`.

## Next

- When implementing: Nuxt routes for public tags/categories pages, wire to
  existing taxonomy API, match T01/C04 layout in default theme, i18n, and
  visibility/`tagPublicPages` checks.

## Open Questions

- Whether `/explore` (08 series) is still needed as a third entry, or only
  nav links to `/tags` + `/categories`.
