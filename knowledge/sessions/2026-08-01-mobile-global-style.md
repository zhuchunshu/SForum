# 2026-08-01 Mobile Global Style Handoff

## Changed

- Implemented the approved D soft-panel mobile direction in the shared public
  navbar, default-theme home feed, and global mobile drawer baseline.
- Mobile topbar controls are limited to appearance, sidebar, and avatar;
  notification preview, language, and compose controls stay desktop-only.
- Added fixed mobile navigation in the order home, post, notifications. The
  post item uses the existing topic-create route for permitted users and sends
  guests to login; the notification item uses the shared unread count.
- Mobile topic rows wrap titles naturally and place avatar, author, time, and
  category/reply metadata below the title. The API-backed category and at most
  two tag links sit in the same horizontal metadata row as the author ID and
  wrap only when space is insufficient. Desktop topic rows retain their
  existing table geometry, with titles no longer forced into one-line ellipsis.
- Reduced only the inline mobile topic-row author avatar from 30px to 24px;
  shared list and desktop avatar sizing is unchanged.
- Mobile drawer backdrops now cover the full viewport and drawers start at the
  viewport top, above the topbar and search region.
- Mobile public layout shells no longer inherit the desktop viewport minimum
  height, removing the large blank block between short content lists and the
  footer while retaining page-specific bottom space for the fixed nav.

## Decisions

- The selected D variant is the mobile product direction: restrained white
  panels, 14px radius, and light elevation without continuous row separators.
- Responsive changes remain scoped to narrow media queries; desktop geometry
  is an explicit compatibility surface.

## Verification

- Focused frontend tests: 70 passed, 0 failed.
- Navbar and notification regression tests after the compact-height fix:
  36 passed, 0 failed.
- Nuxt typecheck: passed.
- In-app Browser on the user's port-3000 server confirmed desktop regression:
  1280px viewport, 58px topbar, three-column grid, hidden mobile nav, and
  square topic rows with separators.
- Prior 402x905 mobile Browser evidence confirmed no horizontal overflow,
  wrapped topic titles and visible metadata, the three-item bottom nav, and a
  full-viewport drawer backdrop above the topbar.
- The 24px avatar follow-up received source/diff verification only; final
  rendered confirmation is intentionally left for operator manual QA.

## Next

- Keep the user's port-3000 dev server and existing demo tabs running.
- Revisit only if the operator requests a different mobile visual direction or
  a separate breakpoint contract.

## Open Questions

- None for the approved D mobile direction.
