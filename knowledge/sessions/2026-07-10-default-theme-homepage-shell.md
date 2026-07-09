# 2026-07-10 Default Theme Homepage Shell

## Changed

- Kept the original global `SFNavbar` in the default layout and removed the
  homepage-specific topbar contract from the redesign tests.
- Adjusted the default-theme homepage shell to stay broad while keeping
  moderate page gutters, matching the accepted A direction without a
  zero-margin edge-to-edge body.
- Reworked homepage accent styling to use global SForum appearance variables
  instead of fixed teal/blue values for active states, notices, badges, category
  dots, and scrollbars.

## Decisions

- The homepage should not render a second topbar; global public navigation
  remains owned by `extensions/builtin/themes/sforum-default/layer/app/layouts/default.vue`.
- The homepage may keep its A-style workbench border and left rail, but page
  gutters should stay present on both desktop and mobile.

## Verification

- `cd apps/web && bun test tests/defaultThemeHomepage.test.ts tests/defaultThemeTopicPage.test.ts`

## Next

- Browser visual QA is still needed later to compare the rendered homepage
  against the A demo after the user is ready to resume verification.
