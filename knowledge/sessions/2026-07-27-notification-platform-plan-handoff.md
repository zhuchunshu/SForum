# 2026-07-28 Notification Platform V2 Final Handoff

## Changed

- Completed M0-M7. The task book is archived at
  `../plans/archive/2026-07/2026-07-27-notification-platform-v2.md` and the
  detailed evidence is in
  `../reports/2026-07-28-notification-platform-v2-final.md`.
- Corrected transactional reply, mention, approval, and rejection fanout;
  added the persistent Notification Registry, layered policy/preferences,
  dedicated admin/user settings, Host API v2 plugin emission, durable-revision
  SSE, generic channel delivery, and the protected Web Push reference plugin.
- Registered `/settings/notifications` as the replaceable
  `forum.settings.notifications` Page Registry page with a Host-owned settings
  island in both built-in themes.
- The final full repository gate passed with `.env` exported and outside the
  process-list sandbox. No files were staged or committed.
- Fixed a production comment-create failure in the legacy policy projection:
  PostgreSQL boolean values now use `BOOL_OR` instead of unsupported
  `MAX(boolean)`, with a real PostgreSQL regression test.

## Decisions

- Keep `mail_deliveries`, `mail.deliver`, and existing mail APIs. Generic
  external channels use `notification_channel_deliveries` plus attempt rows.
- `notification_channel_deliveries.notification_id` is nullable so an enabled
  external channel does not fabricate an inbox row when `in_app=false`.
- Web Push uses `github.com/SherClockHolmes/webpush-go@v1.4.0`; SSE uses Fiber
  `SendStreamWriter` with PostgreSQL NOTIFY as a wake hint only.
- Core owns the minimal `/_sforum/notifications/` service worker and click
  safety. Provider plugins own VAPID and transport behavior, never worker code.
- Hidden, deleted, unknown, or unauthorized targets fail closed and clear
  actor, payload, identifiers, and route data at read time.

## Next

- No implementation milestone remains. Start a fresh conversation only for an
  independent adversarial review using the prompt in the final report.
- Keep digests, scheduled summaries, unsubscribe-link semantics, broadcast,
  marketing, and additional vendor channels deferred unless a new task book is
  approved.

## Open Questions

- None for V2 closure. Independent review may reopen a finding only with a
  concrete production call path, failing test, security proof, or contract
  mismatch.
