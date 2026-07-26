# 2026-07-26 Session Handoff — Settings Shell Refactor

## Changed

- Added `apps/web/app/components/SFSettingsShell.vue`: shared three-column
  chrome for `/settings/*` account pages. Owns the left `SFHomeNavigation` +
  `SFSettingsAccountNav`, the category-group `useAsyncData` fetch (key unified
  to `settings-categories`, shared cache across settings pages), the page head
  (title/description + drawer toggle buttons), the right rail aside, both
  mobile drawers + backdrop (`forum-mobile-menu-open` /
  `forum-mobile-info-open`), `SFContentColumnFooter`, and the
  `sforum-settings.css` import.
- Shell slots: `default` (main column), `#rail` (rendered once, reused by the
  desktop aside and the mobile right drawer — removes the previous in-file
  duplication of rail markup), `#head-actions` (extra head buttons, e.g.
  security's revoke-others).
- Shell props: `active`, `titleId`, `title`, `description`, `railLabel`,
  `railOpenLabel`, `publicProfilePath?`, `showRail?` (profile hides the rail
  until profile data resolves). Page-specific root `class` and
  `data-sforum-island-body` fall through via attrs onto the shell's `<main>`.
- `SFProfileSettingsPage` and `SFSecuritySettingsPage` rewritten to use the
  shell; only form/session/token business logic remains (~160/~175 lines less
  respectively). Behavior, markup classes, and i18n keys unchanged.
- Updated `tests/profileSettingsCanvas.test.ts` and
  `tests/securitySettingsChrome.test.ts`: chrome markup assertions now target
  `SFSettingsShell.vue`; page assertions check `<SFSettingsShell` usage,
  `active` prop, and page-specific class. 9/9 pass.
- Verified: `bun run typecheck` passes; both routes SSR without error on the
  running dev server (302 to login unauthenticated, as expected).

## Decisions

- Supersedes the 2026-07-24 decision "extract only `SFSettingsAccountNav`; do
  not invent a full SettingsShell yet": with two pages duplicating the full
  chrome (and rail markup duplicated twice within each), the user requested the
  extraction; `SFSettingsShell` is now the canonical wrapper for account
  settings pages.
- Category `useAsyncData` keys merged from per-page keys into one shared
  `settings-categories` key for cross-page cache reuse.

## Next

- Authenticated browser QA of both pages (desktop + mobile drawer open/close)
  to confirm no visual regression.
- Future settings pages (e.g. notifications preferences): add a link in
  `SFSettingsAccountNav`, wrap in `SFSettingsShell`.

## Open Questions

- None blocking.
