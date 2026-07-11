# 2026-07-12 Mail, Notifications, and SMTP Plugin Handoff

## Changed

- Added durable notification inbox and mail-delivery tables.
- Added reply, mention, and moderation-result transactional fanout.
- Added notification self-service and mail administration APIs plus OpenAPI.
- Moved every SMTP transport concern into protected plugin `sforum.smtp`.
- Queued password-reset and admin test mail through River.
- Added notification page, unread Navbar badge, provider-neutral mail admin,
  and plugin-owned SMTP settings.
- Added legacy SMTP adoption and extension secret masking/preservation.

## Decisions

- Missing providers skip email but never block in-app notifications.
- API and worker processes own independent plugin runtimes.
- Plugin settings are injected only into the selected subprocess environment.

## Next

- Add notification channel preferences, digests, and unsubscribe semantics only
  as a separate product slice.

## Open Questions

- None for the first release.
