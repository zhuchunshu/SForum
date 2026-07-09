# 2026-07-10 Homepage A Direction Restored

## Changed

- Reworked the default-theme homepage to directly match the accepted A
  Linux.do-style compact direction.
- `extensions/builtin/themes/sforum-default/layer/app/pages/index.vue` now
  uses `definePageMeta({ layout: false })`, renders an internal A-style
  topbar, keeps the 238px left rail plus dense topic table, removes the
  oversized homepage H1/content toolbar, and renders `SFFooter` manually.
- `extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css`
  now owns A-style light and dark surface tokens, a 1520px shell, compact
  topbar/search/button treatment, mobile topbar wrapping, and the corrected
  mobile one-column homepage grid.
- Added `home.filter.categories` and `home.filter.tags` translations.
- Updated `apps/web/tests/defaultThemeHomepage.test.ts` so future changes must
  preserve the A-style homepage shell instead of the previous global-navbar
  compromise.

## Decisions

- The homepage should no longer render inside the public default layout, because
  the global `SFNavbar` made the accepted A direction look like a partial
  hybrid. Other public pages still use the default layout with `SFNavbar` and
  `SFFooter`.
- Dark mode remains supported as an A-structure dark mapping; it should not add
  back the old oversized H1 or blue-black dashboard-like homepage hierarchy.

## Verification

- `cd apps/web && bun test tests/defaultThemeHomepage.test.ts tests/defaultThemeTopicPage.test.ts`
- `cd apps/web && bun run typecheck`
- Browser visual QA on `http://127.0.0.1:3000/`:
  - Desktop 1280x720: A topbar, 238px left rail, dense table, no global navbar,
    no oversized H1, no console warnings/errors.
  - Mobile 390x844: left rail hidden, topbar wraps, main grid remains full
    width after fixing the one-column breakpoint.

## Next

- If the topbar needs production auth/language/theme controls, add them in the
  A visual language instead of reintroducing the old global navbar.

## Open Questions

- Whether the homepage A topbar should expose login/register and language/theme
  controls immediately, or keep those routes reachable from other public pages
  until a compact A-compatible control set is designed.
