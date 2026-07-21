# Mail, Notifications, and Built-in SMTP Plugin Design

## Status

Approved in conversation on 2026-07-11. This document is the implementation
boundary for the first complete mail and in-app notification slice.

## Goals

- Keep core provider-neutral: core defines mail contracts, scheduling,
  selection, and delivery records, but contains no email delivery provider.
- Provide SMTP delivery exclusively through a protected built-in plugin.
- Add a durable, user-owned in-app notification inbox.
- Cover password reset, admin test mail, replies, mentions, and moderation
  approval/rejection in the first release.
- Make forum writes independent from external email availability.

## Non-goals

- Digests, unsubscribe links, per-notification channel preferences, and
  user-configurable delivery schedules.
- Vendor-specific email APIs such as SES, Resend, or SendGrid.
- Plugin-defined notification audiences or arbitrary core route overrides.
- Rendering trusted plugin HTML inside notification content.

## Architectural Boundary

### Core ownership

Core owns:

- The `mail.provider` provider slot and its manifest/RPC contract.
- Provider discovery, administrator selection, health visibility, timeout
  enforcement, error classification, and disable fallback.
- A provider-neutral `MailMessage` request and result contract.
- River mail jobs and durable delivery records.
- Notification types, recipient resolution, inbox persistence, read state,
  permissions, APIs, and user interface.
- Forum-event orchestration for replies, mentions, and moderation results.
- Localized, transport-neutral mail content rendering.

Core must not contain an SMTP client, SMTP authentication, TLS negotiation,
provider configuration validation, log-delivery provider, or no-op provider
implementation. When no usable provider is selected, the core dispatcher marks
the delivery as skipped with `provider_unavailable`; it does not invoke a
provider.

### Plugin ownership

The protected built-in `sforum.smtp` plugin lives under
`extensions/builtin/plugins/` and owns all SMTP service behavior:

- SMTP host, port, encryption, credentials, and sender configuration.
- Configuration validation and connection/send diagnostics.
- SMTP/TLS/STARTTLS/authentication and RFC message transport.
- Mapping transport failures into the provider RPC error categories.

SMTP settings, including secrets, are stored in `extension_settings`. They are
not registered as core `web_options`, included in River arguments, or written
to delivery records and logs.

## Provider Contract and Lifecycle

`mail.provider` becomes a first-class known provider slot. A plugin declaration
identifies the slot and display label; the enabled plugin runtime exposes a
typed mail-send RPC. The request contains normalized recipients, subject,
plain-text and optional HTML bodies, sender metadata, correlation ID, and
delivery ID. It contains no user session or authorization credential.

Only an enabled, healthy plugin declaring `mail.provider` may be selected.
Selecting a provider requires `settings.manage`. Plugin enable/disable remains
under `extension.manage`. Disabling the selected plugin atomically clears the
selection. The framework then reports mail as unconfigured while in-app
notifications remain operational.

RPC results distinguish:

- Success: mark the delivery `sent`.
- Temporary transport failure: return an error to River for bounded retry.
- Permanent configuration or authentication failure: mark `failed` without
  retrying.
- Timeout/runtime loss: retry within the River attempt limit, then mark
  `failed`.
- Missing, disabled, or unselected provider: mark `skipped` with
  `provider_unavailable` and do not retry.

Stable reason values are stored and exposed; raw credentials, full RPC
payloads, and sensitive server responses are not.

## Persistence

### Notifications

`notifications` stores one recipient-owned inbox item per notification:

- ID and recipient user ID.
- Stable type: reply, mention, moderation approved, or moderation rejected.
- Optional actor user ID.
- Target resource type and ID.
- Versioned JSON payload containing only display/link inputs.
- Creation time and nullable `read_at`.

The table does not store pre-rendered HTML. Frontends localize stable types and
payloads. Indexes support recipient plus creation time and recipient unread
queries. A deterministic uniqueness key prevents duplicate fanout for the same
event, type, and recipient.

### Mail deliveries

`mail_deliveries` stores the durable delivery fact:

- ID, recipient, template key, and versioned template data.
- Correlation and idempotency keys.
- Status: `queued`, `sending`, `sent`, `failed`, or `skipped`.
- Selected extension ID when dispatch begins.
- Attempt count, stable reason, sanitized error summary, and timestamps.

Job arguments contain only the delivery ID. Workers re-read the current user,
delivery, content, and provider selection before sending. A unique idempotency
key prevents duplicate logical deliveries even when a River job is retried.

## Event and Delivery Flow

Reply, mention, and moderation result handling writes each recipient's
notification, creates its `mail_deliveries` row, and inserts the related River
mail job in the same PostgreSQL transaction as the authoritative domain change.
The implementation uses River's transactional insert API rather than a
separate queue or custom outbox. Rollback removes the notification, delivery,
and queued side effect together.

Recipient rules for the first release are explicit:

- A reply notifies the author of the replied-to content.
- A mention notifies each uniquely resolved mentioned user.
- Moderation approval/rejection notifies the affected content author.
- Actors never notify themselves.
- Duplicate mentions of one user in one content revision produce one item.
- One event that qualifies through more than one rule may produce distinct
  notification types, because reply and mention convey different intent.

The in-app record is authoritative and immediately available after commit.
Email is a secondary asynchronous projection. Email failure never rolls back a
forum write or removes an inbox notification.

Password reset persists the one-time token and its delivery before inserting
the mail job transactionally. The public request endpoint always returns the
same accepted response regardless of account existence, provider state, or
eventual delivery result. Admin test mail creates an independent delivery and
returns an accepted/queued response.

## API and Authorization

Core adds authenticated user endpoints to:

- Page through the current user's notifications.
- Read the current user's unread count.
- Mark one current-user notification as read.
- Mark all current-user notifications as read.

No endpoint accepts an arbitrary recipient user ID. Login state and ownership
checks are authoritative in the API.

Administrative mail endpoints allow operators with `settings.manage` to:

- List usable mail providers and current selection/health.
- Select a provider or restore the unconfigured recommended default.
- Queue a test mail.
- Inspect a bounded recent delivery list and sanitized status detail.

SMTP plugin configuration routes require `settings.manage` at the host policy
boundary. Managing the plugin lifecycle requires `extension.manage`. Plugin
RPC and routes cannot infer authority from raw cookies or bypass host checks.

All routes and schemas are added to the modular OpenAPI contract, including
security, error responses, stable reasons, pagination, and accepted responses.

## User and Operator Experience

The default theme adds a notification entry with a stable unread badge and a
paginated inbox. Items link to the relevant topic/comment or content-review
detail. Reply, mention, approval, and rejection labels are localized in
`zh-CN` and `en-US`.

The core mail page shows provider selection, health, recent deliveries, test
mail, and the distinction between mail and in-app delivery. Its one-click reset
clears provider selection but preserves plugin-owned secrets and says so.

The `sforum.smtp` management page owns SMTP fields and sender identity. Empty
secret updates preserve the stored password; an explicit secret-clear action
is required to remove it. Disabling the selected plugin explains that email
will stop while in-app notifications continue.

Successful user actions use appearance-aware Toasts and auto-dismiss within
10 seconds. Blocking configuration and send errors remain next to the relevant
control until dismissed or resolved.

## Compatibility Migration

The repository currently stores SMTP implementation and settings in core. The
new implementation removes that provider code from `app/Support/Mail` and
stops registering SMTP fields in core options.

After built-in extension synchronization makes `sforum.smtp` available, a
one-time idempotent application migration copies legacy sender and
`mail.smtp.*` values into the plugin's `extension_settings`. If legacy
`mail.provider` is `smtp`, the migration enables and selects `sforum.smtp` only
after settings migration succeeds. Legacy `dev_log`, `noop`, blank, and unknown
values become an unconfigured selection.

The migration records completion without logging values. It preserves an
existing SMTP password as a secret. Re-running it does not overwrite newer
plugin settings. After successful adoption, deprecated core option definitions
and UI fields are removed; legacy rows may be deleted by migration once no
runtime reader depends on them.

Existing password-reset request/confirm and admin test-mail route paths remain
compatible. The send request response changes to accepted/queued semantics and
is reflected in OpenAPI and frontend behavior.

## Testing Strategy

Implementation follows red-green-refactor. Required coverage includes:

- Core unit tests for provider selection, disable fallback, error
  classification, notification deduplication, self-notification filtering,
  ownership, unread counts, and delivery transitions.
- PostgreSQL integration tests proving notification/delivery/River inserts
  commit and roll back together and idempotency keys prevent duplicates.
- Migration tests for SMTP selection, secret preservation, non-SMTP fallback,
  repeat execution, and preservation of newer plugin values.
- Plugin contract tests for configuration validation, SMTP, TLS, STARTTLS,
  authentication, message assembly, and error classification against a local
  test SMTP server without public network access.
- API tests for allowed and denied paths, inbox isolation, pagination, unread
  behavior, accepted mail requests, and password-reset enumeration safety.
- Frontend tests for inbox rendering, unread badge, provider selection, SMTP
  settings, disable degradation, bilingual content, and feedback behavior.

Final verification runs `go test ./...`, Nuxt typecheck, OpenAPI reference
validation, focused frontend tests, and `./scripts/test.sh`.

## Acceptance Criteria

- No SMTP, log-mail, or no-op mail provider implementation remains in core.
- SMTP delivery works only through the enabled and selected built-in plugin.
- Missing or disabled providers produce observable skipped deliveries without
  affecting in-app notifications or forum writes.
- Reply, mention, and moderation result notifications are durable,
  recipient-isolated, deduplicated, and asynchronously projected to email.
- Password reset remains enumeration-safe and uses queued delivery.
- Existing SMTP settings migrate once without exposing or losing secrets.
- Permission-denied paths, OpenAPI, bilingual UI, and full repository tests are
  covered and passing.
