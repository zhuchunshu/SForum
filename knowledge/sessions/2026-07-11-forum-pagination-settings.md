# 2026-07-11 Session Handoff

## Changed

- Added topic and comment pagination runtime options with recommended defaults
  of 20 and a 1-100 range.
- Applied omitted-`perPage` defaults in forum topic/comment services and the
  Meilisearch query service.
- Added permission-aware controls to the forum settings page and included the
  values in one-click reset behavior.
- Removed fixed page-size request values from the built-in homepage, category,
  tag, search, and comment flows.
- Updated the modular OpenAPI contract and focused backend/frontend tests.

## Decisions

- API-side default resolution is authoritative.
- Explicit client page sizes remain supported and capped at 100.
- Pagination settings use existing `settings.manage` authorization.

## Next

- Monitor representative deployments before changing the recommended value of
  20 or the existing deep-page limit of 200.

## Open Questions

- None for this release.
