# 2026-07-30 Shared Public Page Headings

## Changed

- Added `SFPublicPageHeader` as the shared heading, subtitle, metadata, and
  action-slot primitive for Core public list and workbench shells.
- Centralized page and section title typography in `sforum-theme.css`.
- Migrated home/search, category index/detail, tag index/detail, notification
  index/detail, account settings, and moderation queue headers.
- Removed the corresponding page-owned heading typography declarations and
  added a focused shared-header regression test.

## Decisions

- Keep two semantic levels: compact page headings use 22px/700 (20px mobile),
  while home/search feed section headings retain 20px/600.
- Content titles and distinct shells such as topics, profiles, authentication,
  legal documents, composers, and error pages are not list-page headings and
  remain outside this component.

## Verification

- Focused Bun suite: 56 passed, 0 failed, 513 assertions.
- Nuxt typecheck reached generated types but remains blocked by unrelated
  existing errors in attachment admin, personalization navigation, user
  language, search SEO, and admin surface files.
- `defaultThemeHomepage.test.ts` retains one unrelated stale exact-class
  assertion for the current `sforum-home__main sforum-content-column` markup;
  its other 14 tests pass.
- Scoped diff whitespace validation passes. Browser QA is intentionally left
  to the operator.

## Next

- Operator manually verifies the migrated routes at desktop and mobile widths.

## Open Questions

- None.
