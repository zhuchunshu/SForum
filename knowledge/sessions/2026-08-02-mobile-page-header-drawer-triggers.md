# 2026-08-02 Mobile Page Header Drawer Triggers

## Changed

- Removed duplicate mobile right-drawer buttons from notification index and
  detail content headers.
- Removed duplicate mobile left/right drawer buttons from the shared account
  settings header and moderation queue/review headers.
- Kept the page-owned right drawers and shared `forum-mobile-info-open` state;
  the public navbar remains the mobile drawer entry point.
- Replaced that entry point with the authenticated user's `SFAvatar` on mobile
  and hid the separate authenticated avatar dropdown trigger at narrow widths.
- Extracted `usePublicUserMenu` so the desktop dropdown and mobile drawer share
  profile, settings, permission-aware moderation, email-verification resend,
  and logout actions. The mobile drawer prepends the account section before
  each page's existing right-rail information.
- Centralized the 12 page-owned right-drawer headers. Authenticated drawers use
  “个人中心” while their inner rail keeps the contextual heading such as
  “主题信息”; guests retain the contextual drawer heading.
- Made the mobile account actions an accessible collapsible section controlled
  by the avatar/name identity row. It starts collapsed each time the drawer is
  mounted and exposes its state through `aria-expanded`.
- Removed the guest synthetic avatar and guest right-rail trigger from the
  mobile navbar. Guests now receive one login/registration action; sites with
  closed registration show login only.
- Added focused source-contract regressions for notification, account settings,
  moderation, and the shared mobile avatar/menu composition.

## Verification

- Focused mobile navigation test after the guest follow-up: 8 passed, 0 failed.
- Full Web suite after the heading and collapse follow-ups: 864 passed, 0
  failed.
- Nuxt typecheck: passed.
- Architecture boundary validation: passed (1,596 production files scanned).
- Authenticated Chrome QA at `402x905` passed on `/notifications`,
  `/settings/profile`, and `/moderation`: no duplicate header trigger, no
  horizontal overflow, no framework overlay, clean console, and selected-theme
  `data-provider="sforum.default-theme"` / `data-template="1"` evidence.
- Authenticated Chrome QA at `390x844` passed on `/t/88`: exactly one avatar
  trigger opens the page-owned right drawer; the drawer renders the full user
  menu followed by topic information, closes cleanly, has no horizontal
  overflow, and reports no console warnings/errors. At `1280x800`, the mobile
  trigger is hidden and the desktop avatar dropdown remains visible.
- Notification filter interaction changed the active filter to unread and kept
  the duplicate-trigger count at zero.
- Guest Browser QA at `390x844` passed on `/`: the active default-theme template
  rendered no synthetic avatar or right-rail trigger, the single login action
  navigated to `/login`, and neither page had horizontal overflow or relevant
  console output. The local site has registration closed, so the combined open-
  registration label is covered by the focused source contract.
- Full Web suite after the guest follow-up: 879 passed, 0 failed. Nuxt
  typecheck and architecture boundary validation passed.
- The post-collapse authenticated Chrome page and session remained readable,
  but automated pointer dispatch timed out while the user's DevTools debugging
  session owned the page. The new collapsed/expanded interaction therefore has
  source-contract, typecheck, and full-suite evidence; its final rendered click
  loop remains a manual follow-up.

## Open Questions

- Recheck one authenticated mobile drawer interaction after the active DevTools
  debugging session releases pointer control.
