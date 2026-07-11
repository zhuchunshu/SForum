# 2026-07-12 Public Taxonomy Pages UI

## Decision

Public list pages for taxonomy will follow two selected static demos:

| Route (intent) | Demo | Direction |
|----------------|------|-----------|
| `/tags` | `tmp/demo/grok/taxonomy-demo/01-tags-weight-cloud` (**T01**) | Classic weight tag cloud: font size maps to topic count, filter chips (全部/热门/本周/A–Z), summary stats, weekly rising tags |
| `/categories` | `tmp/demo/grok/taxonomy-demo/09-cats-tile-grid` (**C04**) | Equal-size tile grid by category group: color bar, icon, description, topic/reply counts, enter CTA |

Related but separate: explore hub demos remain under
`tmp/demo/grok/tags-demo` (08 series) and are **not** the primary direction for
standalone full-list pages.

## Why

- T01 is the familiar forum `/tags` mental model without forcing a dense ops table.
- C04 is scannable and consistent for a moderate category count, and maps cleanly
  to existing `ForumCategory` / `ForumCategoryGroup` fields (icon, iconColor,
  description, topicCount, commentCount).

## Implementation notes (when landing in Nuxt)

- Prefer the default theme V32 warm-orange shell (sticky topbar; optional left
  nav alignment) with appearance accent tokens.
- Public pages should respect forum settings such as `tagPublicPages` and
  category visibility; API policy remains authoritative.
- Tag cloud sizing should derive from real `topicCount` (bucket or log scale),
  not hardcoded demo sizes.
- Category tiles should group by `ForumCategoryGroup` and hide non-public
  categories for anonymous/public views.
- Chinese/Unicode tag slugs are already supported; do not assume ASCII-only
  labels in the cloud.

## Non-goals (this decision)

- Does not select the 08 explore-hub layout as the primary `/tags` or
  `/categories` page.
- Does not implement routes in this decision; demos only until product work
  starts.

## References

- Tags demo: `tmp/demo/grok/taxonomy-demo/01-tags-weight-cloud/index.html`
- Categories demo: `tmp/demo/grok/taxonomy-demo/09-cats-tile-grid/index.html`
- Index: `tmp/demo/grok/taxonomy-demo/index.html`
- Taxonomy types: `apps/web/app/utils/forumTaxonomy.ts`
