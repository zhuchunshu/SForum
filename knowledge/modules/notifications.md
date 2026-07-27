# Notifications Module

## Scope

The first release provides durable in-app notifications plus optional email
projection for replies, mentions, and pre-publication moderation approval or
rejection.

## Active Program

Notification Platform V2 is **ready**, not implemented:

- Task book:
  `../plans/2026-07-27-notification-platform-v2.md`
- Decision:
  `../decisions/2026-07-27-notification-platform-v2.md`
- Handoff:
  `../sessions/2026-07-27-notification-platform-plan-handoff.md`

V2 closes the current recipient gaps, separates type/category/channel/state,
adds dedicated admin policy and own-user preferences, exposes bounded plugin
type/emission contracts, adds durable-revision SSE refresh, and proves
`notification.channel` with a protected built-in Web Push provider.

Known V1 gaps that must not be described as complete:

- top-level comments do not notify the topic author;
- only comment creation parses mentions;
- pending comments approved later do not fan out reply/mention notices;
- comment moderation notices do not always have a reliable topic link;
- types/presentation are hard-coded and plugins cannot create canonical inbox
  rows through a stable Host API;
- notification policy is coupled to Mail and users have no preferences;
- no recipient realtime refresh or external notification-channel provider
  contract exists.

## Persistence and Delivery

- `notifications` is recipient-owned and stores stable type, actor, target,
  versionable JSON payload, dedupe key, `read_at`, and timestamps.
- `mail_deliveries` is the durable email projection and River job authority.
- Comment writes create reply/mention inbox rows, deliveries, and jobs inside
  the existing comment transaction. Moderation decisions do the same before
  decision commit.
- Markdown mentions are discovered from goldmark AST text nodes; fenced and
  inline code are ignored. Duplicate usernames and self-notifications are
  filtered.
- Reply and mention remain separate notification types when both apply.
- Core resolves a global reply/mention/moderation policy with independent
  `inAppEnabled` and `emailEnabled` channels. Missing values and the restore
  action resolve to all channels enabled. Disabled channels skip only their
  projection inside the existing transaction.

## API and UI

Authenticated self-service routes:

- `GET /api/v1/notifications`
- `GET /api/v1/notifications/unread-count`
- `PATCH /api/v1/notifications/:id/read`
- `POST /api/v1/notifications/read-all`
- `POST /api/v1/admin/notifications/test` (`settings.manage`, recipient fixed
  to the current actor, type `admin_test`, no email projection)

The default theme adds `/notifications` and a Navbar unread badge. API queries
always bind the current session user ID; another user's notification appears as
not found.

The default theme notifications page was redesigned on 2026-07-23 as a
continuous, API-ordered message stream based on
`tmp/demos/notification-inbox-directions-20260723/01-continuous-stream.html`.
The page keeps filtering honest: type and unread filters apply only to the
currently loaded client page because `GET /api/v1/notifications` exposes only
`limit` and `beforeId`. The right rail uses
`GET /api/v1/notifications/unread-count` as the authoritative global unread
source; list-derived type counts are labeled as loaded-list summaries.

The notification page uses the canonical `topic.create` permission helper for
the shared forum navigation. Below the desktop right-rail breakpoint, the same
unread summary/current-detail component is shown in the right mobile drawer;
do not hide the drawer instance with the desktop rail media rule.

Target links are emitted only when the payload contains a reliable `topicId`
and optional `commentId`, or when the API target is a topic. Other notification
targets render as unavailable instead of inventing routes.

Digests, unsubscribe links, and per-channel user preferences remain out of
scope for this release.
