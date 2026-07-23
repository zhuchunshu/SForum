# 2026-07-24 Session Handoff

## Changed

- Rebuilt `/moderation` left/right sidebars to match home + notifications public
  three-column chrome.
- **Layout shell now reuses `sforum-home__layout` / `__sidebar` / `__main` /
  `__right` + `sf-home-right-rail`** (same as tags/composer/profile) so
  padding/gap/scroll come from `sforum-home.css`, not a parallel grid.
- Left: `SFHomeNavigation` + workbench sources/type filters in
  `#after-navigation`.
- Right: home right-rail card language; decision rail uses the same shell.
- **Counts bugfix:** history tab badge used `processedToday` (today KPI)
  while history list is all-time. API `QueueCounts` now includes
  `historyTotal`; frontend `sourceCountFor` binds badges to pending/open/
  history totals and falls back to active list `total`.
- i18n, unit tests, OpenAPI, Go store updated.

## Decisions

- Prefer reusing `sforum-home__*` shell classes over copying layout CSS.
- `processedToday` remains a KPI only (right-rail stats on history tab).
- Shared mobile drawer keys with other public pages.

## Next

- Hard-refresh browser after API reload so `historyTotal` is present.
- Visual QA against `/` and `/notifications` sidebars.

## Open Questions

- None blocking.
