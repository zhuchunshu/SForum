# 2026-07-28 Notification Email Opt-In Handoff

## Changed

- Added an explicit per-type email control to the admin notification policy and
  made recommended restoration default to in-app on, email off.
- Routed Core reply, mention, and moderation projections through the
  transaction-scoped layered resolver, so personal email preferences now affect
  real delivery and cannot bypass a site-disabled channel.
- Personal notification settings show a chooser only when the Host policy
  allows it; site-disabled email renders a managed-state badge.
- Added migration `202607280071_notification_email_opt_in_defaults.sql` for
  untouched defaults while preserving existing operator choices.
- Fixed shared account-settings mobile navigation overflow at `390x844`.

## Decisions

- Site `enabled=false` is a hard gate; user preference rows may remain stored
  but never become effective until the site enables the channel again.
- Default migrations must not overwrite a saved legacy option or an edited V2
  policy. Restore-default is the explicit path to the new recommendation.

## Next

- No implementation follow-up is required.

## Open Questions

- None.
