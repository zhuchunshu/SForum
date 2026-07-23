# 2026-07-24 Session Handoff

## Changed

- Rebuilt `/settings/profile` and `/settings/security` left/right sidebars to
  match home + notifications public three-column chrome.
- Shared shell CSS: `apps/web/app/assets/css/sforum-settings.css` (1100 hide
  right, 980 hide left; desktop main-column independent scroll).
- Left: `SFHomeNavigation` (route mode, compose/site nav/guidelines, no category
  list) + `SFSettingsAccountNav` in `#after-navigation`.
- Profile right: `SFProfileSettingsPreview` restyled as rail-section preview +
  visibility scope (no card island).
- Security right: live device count + current device / token summary from
  existing sessions/tokens APIs.
- Shared mobile drawer keys `forum-mobile-menu-open` /
  `forum-mobile-info-open`; panel-right + menu controls on narrow widths.
- Theme templates (default + nocturne) for profile/security use
  `sf-theme-shell--fullwidth-3col`; hybrid/theme CSS includes
  `.sforum-settings*`.
- Tests: `profileSettingsCanvas.test.ts` chrome constraints;
  `securitySettingsChrome.test.ts` added. Typecheck + focused tests pass.

## Decisions

- Prefer reusing `SFHomeNavigation` and notifications rail patterns over a
  settings-only island chrome.
- Extract only `SFSettingsAccountNav`; do not invent a full SettingsShell yet.
- Security shares the same three-column shell so profile↔security navigation
  does not jump layout.
- Right-rail summary for security uses only existing client data (no new API).

## Next

- Visual QA in browser against `/`, `/notifications`, `/settings/profile`,
  `/settings/security` at desktop + ≤1100 + ≤980 widths.
- Optional: restyle security main-column cards to public token surfaces later
  (out of chrome scope).

## Open Questions

- None blocking.
