# 2026-07-22 Social login provider plan handoff

Superseded by `../2026-07-27-github-social-login-plan-handoff.md`.

## Changed

- Added the ready task book
  `plans/2026-07-22-social-login-provider-plugins.md`.
- Prioritized social login in the knowledge index and Identity module.
- Defined milestone delivery for Core completion, unified admin/user UI,
  shared SDK, and GitHub/Google/Discord/Telegram plugins.

## Decisions

- Auth providers coexist through Identity Registry; they are not a singleton
  Provider Slot.
- Plugins verify external subjects. Core owns accounts, external links,
  activation, callback state, risk/session policies, sessions, and audit.
- Plugin enable does not expose a login button; Host activation is explicit,
  exact-artifact bound, revisioned, and audited.
- No automatic email linking, no external first-user bootstrap, and no unlink
  of the final usable login method.
- Core password login and Safe Mode recovery remain available.

## Next

- Give the task book to Grok and implement M0 first.
- Review each milestone independently; do not accept a single unsplit patch.
- Require exact test output and evidence for callback replay, account linking,
  lifecycle invalidation, and secret redaction before advancing milestones.

## Open Questions

- None before M0. M0 must verify current official provider protocols and may
  propose an ADR only if that evidence conflicts with the frozen boundaries.
