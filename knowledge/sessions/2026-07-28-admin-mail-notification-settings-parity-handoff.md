# 2026-07-28 Admin Mail And Notification Settings Parity Handoff

## Changed

- Aligned `/control-panel/settings/mail` and
  `/control-panel/settings/notifications` with the established Site Settings
  page geometry: 20px page titles, standard refresh toolbars, fixed tabs, and
  one settings panel per active tab.
- Reworked Mail Overview from a separate recommendation band plus horizontal
  status/step cards into one panel with an inline recommendation, status rows,
  and a vertical setup flow.
- Split Notification Type Policy and External Channels into separate fixed
  tabs. The route shell owns query state and active-tab refresh; each domain
  component owns its existing API and permission-aware behavior.
- Added responsive icon-only toolbar actions below the small breakpoint with
  accessible names and titles. Desktop labels remain visible.
- Added Chinese and English tab/panel copy and updated focused source-contract
  tests for the new ownership boundary.
- Codified the Site Settings geometry as the repository-wide admin-settings UI
  contract in `AGENTS.md`, added a shell regression covering Site, Mail, and
  Notification settings, and wired that regression into `./scripts/test.sh`.

## Decisions

- Reused the existing Site Settings visual contract and
  `SFAdminFixedTabNav`; no new shared UI abstraction or dependency was added.
- Kept `settings.notifications.manage`, provider selection, test delivery,
  restore-default, secret-preservation, and save semantics unchanged.
- Admin settings UI work is not complete without desktop and `390x844` Browser
  comparison against `/control-panel/settings`; source and unit checks are only
  the structural guard.

## Next

- No follow-up implementation is required for this UI parity work.

## Open Questions

- None.
