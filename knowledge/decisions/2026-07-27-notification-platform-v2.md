# 2026-07-27 Notification Platform V2

## Status

Accepted for implementation through
`../plans/archive/2026-07/2026-07-27-notification-platform-v2.md`.

## Context

SForum V1 has a durable recipient-owned inbox, reply/mention/moderation rows,
email projection through River, a selected plugin-owned mail transport, an
inbox page, unread count, and global reply/mention/moderation channel options.

That baseline is useful but not a complete forum notification framework:

- top-level comments do not notify the topic author;
- only comment creation parses mentions;
- comments approved after pre-publication review do not fan out reply/mention
  notices;
- moderation comment targets cannot always produce a reliable link;
- notification types and frontend presentation are hard-coded;
- global notification settings are coupled to Mail;
- users have no preference surface;
- plugins have no bounded way to declare or create inbox notices;
- `notification.channel` remains a roadmap string rather than a proven
  provider contract;
- unread state has no recipient-safe realtime refresh path.

The project is plugin-first, but recipient authority, account/security notices,
privacy, idempotency, and the canonical inbox cannot be delegated to arbitrary
plugin code or raw database access.

## Decision

### Core Owns Notification Semantics

Core remains the authority for:

- recipient eligibility and ownership;
- Core reply, mention, moderation, account, and system types;
- type/category/channel/state vocabulary;
- global policy and user-preference resolution;
- required-notice classification;
- idempotency, outbox composition, retries, and delivery audit;
- read/unread state and safe target presentation;
- plugin type admission, quotas, and audit;
- realtime recipient revisions;
- provider selection and lifecycle fallback.

Plugins do not write Core inbox, preference, or delivery tables.

### Correct Direct Interaction Semantics

- An active top-level comment notifies the topic author.
- An active nested comment notifies the direct parent comment author.
- Active topic/comment creation notifies mentioned active users.
- Self-notifications are skipped.
- Reply and mention remain separate types when both apply.
- Pending content approval performs the same reply/mention fanout exactly once
  in the approval transaction.
- Content edits do not produce new mention notifications in V2.

### Use A Versioned Notification Registry

Core and plugins declare versioned notification descriptors containing bounded
type identity, category, payload schema, localization, icon, target, and channel
metadata.

- Existing Core wire type values remain compatible.
- Plugin ids are extension-namespaced.
- Plugin declarations are inert package data, activated only for the exact
  trusted/enabled artifact.
- Host retains validated inert descriptor snapshots so historical notifications
  have safe fallback presentation after disable, uninstall, or upgrade.
- Only Core may declare a required type.

### Layer Global Policy And User Preferences

Effective delivery is resolved per recipient, type, and channel:

1. the type and channel must be available;
2. the recipient must be eligible;
3. a Host-required channel is delivered;
4. otherwise an explicit allowed user override wins;
5. otherwise the site recommended default applies.

User preferences are three-state: `inherit`, `enabled`, or `disabled`.
Administrators own hard limits, recommendations, and whether a setting is user
configurable. Defaults are not copied into every user row.

Notification policy gets a dedicated
`settings.notifications.manage` permission and admin surface. The existing
Mail policy routes remain a compatibility projection during API LTS; Mail
continues to own mail providers and mail-delivery history.

### Expose A Bounded Plugin Emit Command

Host API v2 gains a versioned notification-emission command. It is bound to the
calling extension, artifact, grant, epoch, instance, locale, deadline, and
trace.

Host validates:

- active type ownership and version;
- payload schema and size;
- explicit bounded recipients and active-user status;
- safe target descriptor;
- idempotency;
- rate and count limits.

Bulk broadcast is a separate high-risk power and is not part of V2.

### Keep Presentation Inert And Permission-Safe

Notification payloads remain structured data, never plugin-rendered HTML.
Plugin localization and icons are inert validated descriptors. Targets refer
to declared routes/entities rather than arbitrary URLs.

Read APIs re-check target visibility. A denied, hidden, deleted, or unknown
target returns a generic unavailable presentation with no title, excerpt,
review note, actor detail, or route leakage.

### Use Durable Recipient Revisions For Realtime Refresh

Each recipient has a monotonic durable notification revision. Inbox writes and
read-state changes increment it transactionally.

PostgreSQL `NOTIFY` is a wake hint only. An authenticated Core SSE endpoint
carries revision/refresh signals without private notification payloads. Connect
and reconnect compare the client cursor with durable state and use the normal
REST APIs for data.

The endpoint is recipient-bound, no-store, cancellable, bounded, and not
replaceable by plugins.

### Add Independent Notification Channel Providers

`notification.channel` is a versioned family of independently selected channel
providers. One provider does not replace all channels.

- Email continues through the existing `mail.provider`.
- Core owns policy, projection, idempotency, delivery state, and retries.
- Channel plugins own transport/vendor behavior.
- Provider disable, uninstall, trust revoke, artifact change, and Safe Mode
  fail closed and remain inspectable.

The first reference is a protected built-in Web Push plugin, subject to an M0
library and service-worker security gate. Core owns a minimal standardized
service worker and authenticated subscription association; it never imports
plugin code. The plugin owns VAPID and Web Push protocol behavior.

## Alternatives Rejected

### Let Plugins Insert Notifications Directly

Rejected because raw writes bypass recipient policy, schema validation,
idempotency, quotas, audits, lifecycle fencing, and safe historical rendering.

### Treat `comment.created` As The Notification API

Rejected because observe events do not provide a bounded recipient/type
contract, do not own user preferences, and cannot safely create canonical inbox
state.

### Keep Notification Policy Under Mail

Rejected because in-app, Web Push, and future channels are not mail settings.
Mail remains one transport/provider domain.

### Store Boolean Preferences

Rejected because booleans cannot distinguish an explicit user choice from
inheritance. They make later operator default changes overwrite or strand user
intent.

### Send Full Notifications Through SSE

Rejected because it duplicates the REST contract, increases privacy exposure,
and makes ephemeral transport appear authoritative.

### Let A Web Push Plugin Serve Arbitrary Origin-Wide Worker Code

Rejected because a root-scope service worker can observe or control unrelated
site traffic. The Host-owned worker remains minimal and declarative.

### Implement SMS/IM Providers In Core

Rejected because vendor transport belongs in plugins and would broaden this
program without proving a better framework contract.

## Consequences

### Positive

- Direct interaction notifications become reliable and testable.
- Operators and users have clear, layered control.
- Plugin types and channels gain stable contracts without raw Core authority.
- Historical notices survive plugin lifecycle changes.
- Realtime refresh remains multi-node and recoverable.
- New providers can reuse the mail-style select/configure/test/reset workflow.

### Costs

- Additive migrations, policy compatibility, descriptor lifecycle, and
  recipient revisions increase implementation scope.
- Approval-time fanout must load stored content and recipients carefully inside
  the decision transaction.
- Web Push requires browser permission/subscription UX and a security-reviewed
  service-worker boundary.
- Plugin declarations, Host commands, SDK, docs, OpenAPI, and generated
  extension catalogs must evolve together.

## Compatibility

- Existing notification ids, type values, payloads, dedupe keys, read state,
  and list/read endpoints remain valid.
- Existing reply/mention/moderation channel options are mapped into the new
  policy authority.
- Existing `/admin/mail/policy` routes remain compatibility projections during
  API LTS.
- Existing `mail_deliveries` history and Mail admin APIs are preserved.
- Unknown historical types always have a generic safe fallback.

## Security Notes

- User ids for inbox/preferences/SSE always come from the authenticated session.
- Admin mutation requires `settings.notifications.manage` or the documented
  parent compatibility permission.
- Required types are Core-only and non-disableable.
- Plugin emission is exact-artifact and grant bound, rate limited, and audited.
- Private payloads, hidden targets, subscription keys, provider secrets, and
  session evidence do not enter public responses, SSE data, logs, or audits.
- Notification delivery must not become an account-enumeration oracle.

## Follow-up

M0 evidence selected `github.com/SherClockHolmes/webpush-go@v1.4.0` and Fiber's
native SSE writer. External channel deliveries use the additive
`notification_channel_deliveries` ledger; `mail_deliveries`, `mail.deliver`,
and Mail APIs remain the email authority. The Host worker boundary is viable
only under `/_sforum/notifications/`; the survey found that `/_sforum` had not
previously been reserved from Page Registry claims, so M6 must preserve the new
Host reservation and regression proof. See
`../reports/2026-07-28-notification-platform-v2-m0-library-service-worker-survey.md`.

M6 implementation evidence changed one frozen storage detail. A configurable
external channel can be enabled while `in_app` is disabled, so no canonical
inbox row exists for that projection. Consequently,
`notification_channel_deliveries.notification_id` is nullable and the delivery
stores its bounded payload and target envelope directly. The ledger still binds
recipient, type, channel, selected provider/artifact, idempotency, status, and
attempt history. This preserves the rule that external delivery must never
create an inbox row merely to satisfy a foreign key.

Implementation uses `github.com/SherClockHolmes/webpush-go@v1.4.0`, Fiber's
`SendStreamWriter`, a process-wide reconnecting PostgreSQL listener, and the
Host-owned `/_sforum/notifications/` worker scope. The exact built-in Web Push
artifact was staged, enabled, configured through SecretStore, selected, and
invoked through River; see the final report linked from the task book.
