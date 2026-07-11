# 2026-07-12 Session Handoff · Public taxonomy list pages

## Changed

- Implemented public list pages in the default theme layer:
  - `/tags` — T01 weight tag cloud (`pages/tags/index.vue`)
  - `/categories` — C04 equal-size tile grid by group (`pages/categories/index.vue`)
- Added `sforum-taxonomy.css` and registered it in the theme layer `nuxt.config`.
- Helpers in `forumTaxonomy.ts`: index paths, `tagCloudSizeBucket` (log scale),
  `tagHotThreshold`, `isCreatedWithinDays`.
- Navbar Categories/Tags now route to list pages; Tags hidden when public tag
  pages are disabled.
- Host `routeRules` cover `/tags`, `/categories` (and `/en/...`) with SWR.
- i18n: `taxonomy.*` keys in zh-CN / en-US.
- Tests: `publicTaxonomyPages.test.ts`, expanded `forumTaxonomy.test.ts`,
  navbar contract update.

## Decisions

- No new API endpoints this slice. Weekly rising uses popular-by-`topicCount`
  fallback (no weekly delta field). “本周” chip = tags created in last 7 days.
- Pending-review stat tile omitted (public listTags has no pending).
- Detail pages `/tags/:slug` and `/c/:slug` unchanged; list links reuse them.
- Explore hub (08 series) still out of scope.

## Next

- Optional: breadcrumb intermediate “全部标签/分类” on detail pages.
- Optional: backend weekly tag activity / delta for true “本周上升”.
- Optional: enforce `tagPublicPages` on `GET /api/v1/tags` (API hardening).

## Open Questions

- Whether `/explore` hub is still desired as a third entry.
