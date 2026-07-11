# 2026-07-12 — Site chrome catalogs as dedicated tables

## Context

Settings Wave 2 needs header nav items, friend links, and homepage
announcement banners. These are ordered multi-row catalogs, not scalar
runtime policy.

## Decision

Store them in PostgreSQL tables under the `SiteChrome` module:

- `site_nav_items`
- `site_friend_links`
- `site_announcements`

Brand assets (logo/favicon) and legal page Markdown bodies remain in
`web_options` as public options.

Admin mutations require `settings.site.manage` (parent `settings.manage`
still expands). Public list endpoints expose only enabled rows;
announcements also filter by optional `starts_at` / `ends_at` window.

## Alternatives considered

1. **JSON in `web_options`** — simple but poor for reorder/CRUD, validation,
   and concurrent admin edits.
2. **First-class CMS pages table for legal only** — deferred; Markdown options
   are enough for stubs.

## Consequences

- New migration `202607120003_site_chrome.sql` seeds default nav (Home /
  Categories / Tags / Search).
- API surface: `/api/v1/site/*` public + `/api/v1/admin/site/*` admin CRUD.
- Frontend admin UI and theme wiring consume these endpoints in later Wave 2
  commits.
