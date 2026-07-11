# Notifications Module

## Scope

The first release provides durable in-app notifications plus optional email
projection for replies, mentions, and pre-publication moderation approval or
rejection.

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

Digests, unsubscribe links, and per-channel user preferences remain out of
scope for this release.
