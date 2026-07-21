# 2026-07-08 Admin Shell Footer

## Changed

- Added `SFAdminFooter` for the admin shell main content area.
- The footer splits content into left and right areas: current-year copyright
  on the left through `admin.shell.footerCopyright`, and a short official
  product summary on the right through `admin.shell.footerProductSummary`.
- Wired the footer into `apps/web/app/layouts/admin.vue` after the page slot,
  with `mt-auto` so it sits at the bottom on sparse admin pages and follows the
  content on longer pages.
- Added admin framework validation assertions to keep the admin shell from
  reusing the public/theme `SFFooter`.

## Verification

- `bun tests/validate-admin-framework.ts`
- `cd apps/web && bun run typecheck`
- Browser check against `http://127.0.0.1:3000/control-panel` reached the
  running local web server but redirected to `/login`, so authenticated admin
  visual verification was blocked by the current browser session state.

## Next

- Re-check visually in an authenticated admin browser session when available.
