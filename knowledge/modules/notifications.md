# Notifications Module

## Scope

The first release provides durable in-app notifications plus optional email
projection for replies, mentions, and pre-publication moderation approval or
rejection.

## Notification Platform V2

Notification Platform V2 is **completed**:

- Task book:
  `../plans/archive/2026-07/2026-07-27-notification-platform-v2.md`
- Decision:
  `../decisions/2026-07-27-notification-platform-v2.md`
- Handoff:
  `../sessions/2026-07-27-notification-platform-plan-handoff.md`

M0-M7 are implemented and the final repository gate is green. The completed
task book is archived under `../plans/archive/2026-07/`; exact command results,
runtime proof, and explicitly skipped checks are recorded in
`../reports/2026-07-28-notification-platform-v2-final.md`.
V2 now provides correct create/approval fanout, one versioned registry and
policy resolver, dedicated admin and own-user settings, bounded plugin emission,
server-side inbox filters, durable-revision SSE, and a protected built-in Web
Push provider.

Production hotfix (2026-07-28): the legacy Core policy projection must aggregate
PostgreSQL boolean columns with `BOOL_OR`/`BOOL_AND`, never `MAX(boolean)`.
`MAX(boolean)` caused comment creation to roll back while reading notification
policy. A real PostgreSQL regression test now covers this compatibility query.

### Plugin Extension Surface

| Surface | V2 contract |
| --- | --- |
| Type declaration | Manifest V3 `notificationTypes`; inert, extension-namespaced, exact schema file/digest, never plugin-required |
| Emission | Host API v2 `notifications.emit@1`; actorless exact artifact/grant/epoch/instance admission |
| SDK | `pluginv2.Host.NotificationEmitRequest` / `EmitNotification` |
| Recipient authority | Explicit active users only, maximum 50; Host policy and preferences remain authoritative |
| Payload/target | 16 KiB structured payload validated against the exact Draft 2020-12 schema; declared target equality only |
| Idempotency/rate | Host Command receipt transaction plus 60 committed requests per rolling minute |
| Audit | Accepted and rejected metadata is redacted; payload values and recipient ids are never audit fields |
| Closed surfaces | Raw notification/preference/delivery table writes, required plugin types, session/actor forgery, arbitrary URLs, and bulk broadcast |

The executable reference fixture is
`extensions/fixtures/plugins/sforum-notification-reference`. Disabled,
uninstalled, stale-artifact, or cross-namespace callers cannot emit; historical
rows retain versioned structured payload and use Host fallback presentation.

## Persistence and Delivery

- `notifications` remains recipient-owned and adds category/payload version plus
  bounded target metadata without changing legacy ids, types, read state, or
  dedupe keys.
- `notification_type_descriptors`, global policies, user preferences, recipient
  revisions, and exact-artifact lifecycle state are additive and CAS/revisioned.
- `mail_deliveries` is the durable email projection and River job authority.
- `notification_channel_deliveries` is the generic external ledger. Its
  `notification_id` is deliberately nullable because `in_app=false` with
  `web_push=true` must not fabricate an inbox row; the row carries a bounded
  payload/target envelope and per-subscription attempt ledger.
- Comment writes create reply/mention inbox rows, deliveries, and jobs inside
  the existing comment transaction. Moderation decisions do the same before
  decision commit.
- Markdown mentions are discovered from goldmark AST text nodes; fenced and
  inline code are ignored. Duplicate usernames and self-notifications are
  filtered.
- Reply and mention remain separate notification types when both apply.
- One resolver combines active type/channel, Core-required state, admin hard
  limits/recommendations, user `inherit/enabled/disabled`, recipient eligibility,
  and provider availability. Existing Mail policy routes are compatibility
  projections over this authority.

## API and UI

Authenticated self-service routes:

- `GET /api/v1/notifications`
- `GET /api/v1/notifications/unread-count`
- `PATCH /api/v1/notifications/:id/read`
- `POST /api/v1/notifications/read-all`
- `GET /api/v1/notifications/stream`
- `GET|PUT /api/v1/notification-preferences`
- `POST /api/v1/notification-preferences/restore`
- `GET|POST|DELETE /api/v1/web-push/*` (current-user subscriptions only)
- `POST /api/v1/admin/notifications/test` (`settings.manage`, recipient fixed
  to the current actor, type `admin_test`, no email projection)

`settings.notifications.manage` protects the dedicated admin policy, channel
selection/reset/test, and redacted delivery-health APIs. Compatibility
inheritance from `settings.manage` remains; `settings.mail.manage` does not
silently grant notification authority.

The default theme adds `/notifications` and a Navbar unread badge. API queries
always bind the current session user ID; another user's notification appears as
not found.

The inbox uses server-authoritative category/type/unread filters and cursor
pagination. `useNotifications` shares one EventSource, coalesces refresh signals,
and falls back to normal REST/manual refresh. SSE contains revision signals only;
durable PostgreSQL rows and recipient revision remain truth.

The notification page uses the canonical `topic.create` permission helper for
the shared forum navigation. Below the desktop right-rail breakpoint, the same
unread summary/current-detail component is shown in the right mobile drawer;
do not hide the drawer instance with the desktop rail media rule.

Forum targets are re-authorized at read time. Hidden/deleted/non-public topics,
inactive comments, unknown targets, and resolver errors fail closed by clearing
actor, payload, target id/type/path and returning `targetAvailable=false`.

Admin policy lives at `/control-panel/settings/notifications`; personal
preferences and Web Push live at `/settings/notifications`. The selected Web
Push provider is `sforum.web-push`; VAPID secrets remain in SecretStore and API
responses expose only `secretSet` plus the public key. The Host worker is fixed
at `/_sforum/notifications/sw.js` with scope `/_sforum/notifications/`, imports
no plugin code, and accepts only bounded same-origin click paths.

Digests, scheduled summaries, unsubscribe-link semantics, broadcast/marketing,
and additional vendor channels remain deferred.
