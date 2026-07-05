# 2026-07-05 Session Handoff

## Changed

- Added independent `seo.manage` permission and a migration that grants it to
  the `super_admin` role.
- Registered typed SEO runtime options under `seo.*`, including meta/social,
  verification, robots, sitemap, and structured-data controls.
- Rebuilt the admin SEO page as a multi-tab runtime SEO center.
- Added `useSForumSeo()` and connected the homepage to runtime SEO metadata,
  canonical URLs, robots meta, social tags, verification tags, and minimal
  JSON-LD.
- Added dynamic sitemap URL generation plus robots.txt and X-Robots-Tag
  runtime guards.

## Decisions

- `seo.*` admin reads and writes require `seo.manage`; non-SEO settings remain
  guarded by `settings.manage`.
- Local and preview URLs remain noindex even when indexing is enabled.
- Forum-content sitemap and DiscussionForumPosting output are prepared but wait
  for real public forum read models.

## Next

- When category/topic/profile pages exist, pass canonical path, timestamps,
  author/display data, and public visibility into `useSForumSeo()`.
- Re-test `nuxt-schema-org` in a browser runtime before replacing the manual
  JSON-LD fallback.

## Open Questions

- Which forum entities should be included in the first real content sitemap:
  categories only, topics only, or categories plus public topic detail pages?
