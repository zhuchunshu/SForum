# 2026-07-27 Notification Platform V2 Plan Handoff

## Changed

- Added the ready task book
  `plans/2026-07-27-notification-platform-v2.md`.
- Added the accepted architecture decision
  `decisions/2026-07-27-notification-platform-v2.md`.
- Promoted notification correctness, admin/user preferences, plugin emission,
  realtime refresh, and external channel providers from scattered later work
  into one active delivery program.
- Added the required multi-conversation execution protocol: one milestone or
  declared slice per fresh conversation, mandatory knowledge updates, a fixed
  small-report format, and a copy-ready prompt for the next conversation.

## Decisions

- Core owns recipient policy, required notices, inbox state, idempotency,
  outbox, safe presentation, and channel selection.
- Top-level replies notify the topic author; nested replies notify the direct
  parent author. Pending approval performs eligible reply/mention fanout exactly
  once.
- Existing Core wire types remain compatible. Plugin types are versioned,
  namespaced, inert declarations and default disabled pending admin review.
- Admin policy is a hard limit plus recommended defaults. User preferences use
  inherit/enabled/disabled.
- Plugins emit only through an exact-artifact Host API v2 command and never
  write Core notification tables.
- Realtime uses durable recipient revisions; PostgreSQL NOTIFY only wakes; SSE
  carries refresh signals rather than private payloads.
- The first `notification.channel` reference is Web Push, gated by M0 library
  and service-worker security review. Core owns the minimal worker; the plugin
  owns VAPID/protocol behavior.

## Next

- Start a fresh implementation conversation with the M0 prompt embedded in the
  task book. M0 audits production paths, freezes additive schemas and
  compatibility, completes the Web Push/SSE library survey, and proves the
  Host-owned service-worker boundary without changing production behavior.

## Open Questions

- M0 must choose the maintained Web Push library after checking current
  maintenance, license, protocol support, and cancellation behavior.
- M0 must freeze whether generic channel delivery uses a new common table or an
  outbox envelope over channel-owned tables while preserving
  `mail_deliveries`.
