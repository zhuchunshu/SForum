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

Realtime badge hotfix (2026-07-28): the public Navbar starts notification
realtime from an immediate auth-state watcher instead of relying on child/parent
mount ordering. An asynchronous EventSource failure now triggers an immediate
REST reconciliation; a terminally closed source reconnects after one second.
Visible subscribed pages also reconcile every 30 seconds and whenever the tab
becomes visible, so API restarts, proxy interruptions, and missed wake signals
cannot leave the bell badge frozen until a full page reload.

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

Core email delivery is opt-in by default: recommended restoration keeps
`in_app` enabled and sets `email` disabled for reply, mention, and moderation
types. Migration `202607280071_notification_email_opt_in_defaults.sql` applies
that transition only to untouched V2 rows without a saved legacy choice, so an
operator's explicit existing email policy is not overwritten. Core fanout now
resolves all three channels through the same transaction-scoped layered policy;
a saved user `enabled` override cannot bypass a site-disabled email channel.

## API and UI

Authenticated self-service routes:

- `GET /api/v1/notifications`
- `GET /api/v1/notifications/:id`
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

The inbox and preview use sibling routes: `/notifications` for the list and
`/notifications/:notificationId` for the independent detail surface. Nuxt owns
them through `pages/notifications/index.vue` and
`pages/notifications/[notificationId].vue`; a flat parent page must not shadow
the nested route. Clicking a row opens the detail without leaving the
notification surface. Only the explicit target action navigates to the current
authorized topic or comment.

`GET /api/v1/notifications/:id` binds the recipient to the current actor and
returns 404 for missing or foreign rows. Forum previews are resolved and
re-authorized at read time, so the response contains the current visible topic
title, reply excerpt, author, and parent/original context rather than a stored
body snapshot. The Page Registry identity is `forum.notification.show` with
contract `sforum.page.notification_show@1`; its theme ViewModel supplies only
the shell while the Host detail island retains recipient authorization and
notification data ownership.

The inbox uses server-authoritative category/type/unread filters and cursor
pagination. `useNotifications` shares one EventSource, coalesces refresh signals,
and reconciles through REST on connect/error, tab visibility, and a 30-second
visible-page fallback interval. SSE contains revision signals only; durable
PostgreSQL rows and recipient revision remain truth.

Notification SSE connections use bounded leases and controlled reconnection.
The process-wide PostgreSQL `LISTEN` revision hub owns a cancellable listener
lifecycle. API bootstrap failure and normal shutdown stop and await that hub
before closing the shared pgx pool, so an acquired listener connection cannot
hide the original startup error by blocking `pgxpool.Close()`.
The Nuxt API route owns a dedicated Node stream proxy that destroys both the
upstream request and response when the downstream browser disconnects. The API
also expires every stream after one minute so abandoned proxy connections
cannot retain one of the four per-recipient slots indefinitely. The client
closes `EventSource` after any error, reconnects with exponential backoff capped
at 30 seconds, and disposes the source during Vite HMR; native unbounded
`EventSource` retry is not relied on. REST reconciliation remains available
during every disconnect.

The inbox and independent detail page share `SFNotificationTypeNav` in the
desktop left rail and mobile navigation drawer. Both surfaces use the same
loaded-scope count semantics and fixed type/icon catalog. On a detail page the
current notification type is highlighted; selecting any type restores the
existing inbox filter state and returns to `/notifications`.

The notification page uses the canonical `topic.create` permission helper for
the shared forum navigation. Below the desktop right-rail breakpoint, the same
unread summary/current-detail component is shown in the right mobile drawer;
do not hide the drawer instance with the desktop rail media rule.

Notification list sources are type-aware. Core `reply` and `mention` rows carry
a current actor summary and use the actor's configured `AvatarView`; moderation
results, admin tests, actorless plugin notifications, and unknown types use a
bounded Tabler icon fallback instead of fabricated notification-name initials.
The actor summary is populated in the recipient-owned list/detail store only
for user-authored Core types and is cleared together with actor/target/payload
when target re-authorization fails.

Forum targets are re-authorized at read time. Hidden/deleted/non-public topics,
inactive comments, unknown targets, and resolver errors fail closed by clearing
actor, payload, target id/type/path and returning `targetAvailable=false`.

Admin policy and external-channel management live as tabs under the unified
`/control-panel/settings/mail` surface; the old
`/control-panel/settings/notifications` URL redirects to the corresponding
tab. Personal preferences and Web Push live at `/settings/notifications`. The selected Web
Push provider is `sforum.web-push`; VAPID secrets remain in SecretStore and API
responses expose only `secretSet` plus the public key. The Host worker is fixed
at `/_sforum/notifications/sw.js` with scope `/_sforum/notifications/`, imports
no plugin code, and accepts only bounded same-origin click paths.

The unified admin route uses the shared settings geometry. Type Policy and
External Channels remain independent fixed Core tabs and focused panels beside
the mail tabs. The route shell owns tab/query state and refreshes only the
active tab, while API policy and `settings.notifications.manage` remain
authoritative and do not grant mail access.

The admin policy row uses a dedicated email-notification control. Disabling a
channel also clears its recommended state. Personal settings render a chooser
only for active, site-enabled, available, user-configurable channels; a
site-disabled email row renders a managed-state badge instead, even when a
historical user override remains stored. The shared settings mobile navigation
constrains its category select to the content width to prevent horizontal
overflow at `390x844`.

Digests, scheduled summaries, unsubscribe-link semantics, broadcast/marketing,
and additional vendor channels remain deferred.
