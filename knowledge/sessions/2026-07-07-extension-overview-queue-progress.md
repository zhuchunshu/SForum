# 2026-07-07 Extension Overview Queue Progress

## Changed

- Added uploaded theme activation progress display to the admin Extensions
  overview list.
- Reused the existing theme release helper state, progress labels, and short
  polling behavior already used by the Themes page.
- Updated the theme activation progress validation script so it covers both the
  Themes page and the Extensions overview.

## Decisions

- Keep the overview behavior UI-only and contract-compatible. No new API shape
  was needed because `GET /api/v1/admin/extensions` already returns
  `themeRelease`.

## Verification

- `node tests/validate-theme-activation-progress.js`
- `bun test tests/adminExtensions.test.ts`
- `bun run typecheck`
- `./scripts/test.sh`

## Open Questions

- Rendered browser validation was blocked by the local API returning the
  existing admin auth 503 recovery page instead of the Extensions overview.
