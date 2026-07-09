# 2026-07-10 Admin Taxonomy Icon Color Handoff

## Changed

- Added optional icon and icon color fields to forum categories and tags.
- Exposed those fields through existing admin taxonomy create/update/list endpoints.
- Added admin category/tag form controls using `SFIconPicker` plus hex color input.
- Added admin list previews for category and tag icons.

## Decisions

- The first release only previews taxonomy visuals in admin pages.
- Category groups, public theme pages, topic summaries, tag summaries, and search documents do not consume these fields yet.
- Icon values remain plain Nuxt Icon names, and colors remain six-digit hex strings.

## Next

- Decide later whether public category/tag pages should display these visual fields.
- If public display is enabled, update theme-layer pages and public topic/tag summaries intentionally.

## Open Questions

- None for the admin-only release.
