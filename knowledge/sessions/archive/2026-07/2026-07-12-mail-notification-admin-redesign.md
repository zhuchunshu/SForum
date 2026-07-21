# 2026-07-12 Mail And Notification Admin Redesign

## Changed

- Added generic extension setting presentation metadata and select rendering.
- Made the built-in SMTP settings grouped, bilingual, recommendation-led, and
  secret-preserving.
- Added global reply/mention/moderation channel policy and transactional fanout.
- Added Core policy, custom test-email, self-test notification, and delivery
  management APIs and UI.
- Added a visible System sidebar entry for `/settings/mail`.

## Decisions

- Core never branches on SMTP plugin IDs, field names, encryption modes, or
  ports. Provider-specific setup guidance lives in the plugin manifest.
- Test notifications always target the current administrator and never create
  an email projection.
- Test email accepts one custom plain email address and reports queueing, not
  final delivery.

## Next

- Complete browser QA for desktop/mobile and verify delivery behavior against
  the running development environment.

## Open Questions

- Per-user channel preferences, digests, unsubscribe links, and additional
  notification channels remain intentionally out of scope.
