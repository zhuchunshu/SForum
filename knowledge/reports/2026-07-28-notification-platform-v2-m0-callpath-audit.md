# Notification Platform V2 M0 Call-Path Audit

Date: 2026-07-28

Scope: current production notification, mail-admin, route-guard, manifest,
lifecycle-publication, and Host API v2 call paths. This is evidence only; it
does not alter a runtime contract or production behavior. Findings reflect the
working tree at audit time, which already contained unrelated/uncommitted
notification and forum changes.

## 1. Current production notification path

1. API assembly creates one `notifications.Outbox`, gives it the resolved
   global policy reader, and injects adapters into Forum and Moderation:
   `apps/api/bootstrap/api_assembly_domains.go:82-91`.
2. Forum writes call the injected transaction-scoped adapter after a comment
   (`apps/api/app/Models/Forum/postgres_store_ops.go:202`) or topic
   (`apps/api/app/Models/Forum/postgres_store_topic_mutations.go:104`);
   moderation does the same during its workbench transaction
   (`apps/api/app/Models/Moderation/workbench_store.go:335`). The adapters only
   translate domain inputs to `Outbox` events
   (`apps/api/bootstrap/notification_adapter.go:12-25`).
3. `NotifyCommentTx`, `NotifyTopicTx`, and `NotifyModerationTx` resolve active
   recipients, derive stable event/recipient dedupe keys, and select the two
   current projections from the global policy
   (`apps/api/app/Models/Notifications/fanout.go:42-72`, `105-171`).
4. The projection helper writes the inbox row, the mail row, or both in the
   caller's transaction (`apps/api/app/Models/Notifications/outbox.go:109-139`).
   `notifications.dedupe_key` and `mail_deliveries.idempotency_key` are each
   conflict keys (`apps/api/app/Models/Notifications/postgres_store.go:40-68`,
   `75-88`). If a mail projection exists, the same transaction enqueues a
   unique River `mail.deliver` job (`outbox.go:61-68`, `95-105`).
5. The worker loads the persisted mail delivery, treats terminal rows as no-op,
   resolves the selected mail plugin, writes attempt status, and invokes only
   `SendMail`; temporary failures return an error for River retry
   (`apps/api/app/Jobs/Notifications/deliver_mail.go:41-89`). Worker assembly
   deliberately reuses the production plugin runtime
   (`apps/api/bootstrap/worker.go:617-623`).

Conclusion: M1/M2 must preserve the transaction boundary, two existing
idempotency keys, River mail job semantics, `mail_deliveries`, and the
`mail.deliver` API. There is no generic channel projection or channel worker
yet.

## 2. Current public and administrator API/UI path

- Inbox routes are registered directly by the Notifications provider:
  `apps/api/app/Http/Controllers/Notifications/controller.go:31-38` and
  `apps/api/app/Providers/notifications.go:15-18`. Each public operation takes
  the recipient only from the authenticated session; list and unread queries
  pass that ID into the store, while read mutations constrain both notification
  and recipient (`controller.go:60-120`; `postgres_store.go:91-134`).
- Existing OpenAPI freezes the present REST names and response shapes at
  `contracts/openapi/paths/notifications.yaml:1-43` and
  `contracts/openapi/schemas/notifications.yaml:1-41`. The sole administrator
  notification test creates an inbox record for the actor and no mail
  projection (`controller.go:40-58`).
- The frontend's `useNotifications` calls only REST list/unread/read endpoints
  (`apps/web/app/composables/notifications/useNotifications.ts:12-25`), and
  SSR loads list plus unread count (`apps/web/app/components/notifications/SFNotificationsPage.vue:45-65`).
  No EventSource, SSE endpoint, service worker, Push API, or Web Push code is
  present in this production path.
- Mail/notification administration remains a `settings.mail.manage` surface.
  The controller owns provider selection, mail delivery history, the global
  dual-channel policy, and reset (`apps/api/app/Http/Controllers/Mail/controller.go:26-148`);
  the policy has exactly reply/mention/moderation x in-app/email fields and
  defaults all enabled (`apps/api/app/Models/Options/notification_options.go:5-60`).
  The page is a four-tab mail shell and its notification tab calls these legacy
  routes (`apps/web/app/pages/admin/settings/mail.vue:13-58`,
  `apps/web/app/components/admin/settings/mail/tabs/SFAdminMailNotificationsTab.vue:13-32`).
- `mail_deliveries` listing intentionally removes template data and idempotency
  keys before the admin response (`postgres_store.go:158-177`), but the current
  admin UI does render the email recipient (`SFAdminMailDeliveriesTab.vue:20-26`).

Conclusion: new generic-channel administration must be a separate contract and
redacted health surface. Do not rename, widen, or route a generic delivery
through `/admin/mail/*`; retain this page and API as the email compatibility
facade.

## 3. Route catalog and guard ownership

- The production HTTP app installs the Notifications and Mail providers along
  with the immutable lifecycle route plan/dispatcher
  (`apps/api/bootstrap/api_assembly_http.go:28-40`).
- The core route catalog assigns the existing recipient routes to
  `core.guard.notifications.recipient`
  (`apps/api/app/Support/Routes/core_catalog_gen.go:219-222`). The guard is
  registered in the production registry
  (`apps/api/app/Http/core_guard_authorizer.go:155-191`) and accepts only the
  four known route IDs before requiring an authenticated actor
  (`core_guard_authorizer.go:392-404`). The store then enforces recipient ID in
  SQL. A new inbox/SSE route must be added to this closed guard module (or a
  distinct closed Core guard), not treated as a manifest-declared route.
- Mail routes use catalog `permission_any` with both the fine-grained and
  legacy parent keys (`core_catalog_gen.go:119-126`), matching the controller's
  direct `settings.mail.manage` check (`Mail/controller.go:36-46`). Existing
  admin notification test is catalogued under `settings.manage`
  (`core_catalog_gen.go:133`), while its controller uses the finer mail
  permission (`Notifications/controller.go:40-47`); M1 must not rely on the
  catalog alone when changing that endpoint.

## 4. Extension manifest, lifecycle, and Host API v2 boundary

- A plugin manifest can declare routes and providers, including stable contract
  IDs, schemas, fallback, timeout, and handler
  (`apps/api/app/Support/ExtensionManifest/manifest.go:68-128`, `199-274`).
  This is a declaration mechanism, not table-write authority.
- During exact-runtime lifecycle publication, Core validates and replaces the
  provider-slot snapshot together with other registries
  (`apps/api/app/Support/Extensions/lifecycle_registry_publication.go:485-517`).
  The Host broker requires an attested extension/version/artifact/instance,
  exact slot contract and compiled schemas, then invokes through the versioned
  registry (`apps/api/app/Support/Extensions/provider_slot_hostapi.go:50-99`).
  `Gateway.RegisterProtocolV2` freezes bound services before broker registration
  (`apps/api/app/Support/HostAPI/gateway.go:50-79`).
- Current `mail.provider` is an older dedicated selection facade, not the
  versioned generic broker. It only accepts an enabled plugin manifest declaring
  that slot and stores a selected extension ID
  (`apps/api/app/Support/Extensions/mail_providers.go:10-63`). It has no
  channel-delivery lifecycle or generic provider state.
- Host API v2 currently exposes query/command, database, permission, identity,
  jobs, audit, service discovery and optional cache/secrets/files/HTTP services
  (`apps/api/app/Support/HostAPI/v2.go:44-82`). A repository-wide source scan of
  `apps/api/app/Support/HostAPI` and `apps/api/sdk/plugin/v2` found no
  notification host service or notification write command. Therefore an
  executable Web Push plugin cannot legitimately create inbox rows, user
  preferences, deliveries, or subscriptions through current Host API v2.

Conclusion: the V2 plan's `notification.channel` must be a new, bounded,
Host-owned contract, published only after exact-artifact lifecycle admission.
The existing generic provider broker is a viable invocation transport after
that contract exists, but it does not itself grant notification authority.
Web Push provider code must receive only Core-created, bounded delivery
material; it must not obtain raw recipient/session/database authority.

## 5. M0 compatibility decisions supported by the audit

1. Keep `mail_deliveries`, `mail.deliver`, `MailProviderRegistry`, and all
   `/admin/mail/*` routes as the email compatibility path. A new common
   `notification_channel_deliveries` table is safer than generalizing the
   mail table because the latter's primary data model, renderer, worker and
   public admin shape are explicitly email/template/recipient based.
2. Keep the inbox API response shape and its recipient-bound Core guard
   unchanged. Add additive routes/contracts for preferences, durable revisions,
   SSE, subscriptions, generic delivery health, and test operations.
3. Keep SSE Core-owned and recipient-bound. The current UI has only REST
   refresh; no existing extension route should be reused as a notification
   stream.
4. Keep the Web Push plugin at the exact-artifact provider boundary. A Host API
   addition is required before it can request bounded delivery; it cannot be
   implemented by granting direct Core-table access or by reusing `mail.provider`.

## Commands and limits

Executed read-only commands included `git status --short`, `rg` source/call
site searches, and `nl -ba` reads of every path cited above. The initial status
reported pre-existing changes in Forum/Notifications/bootstrap and unrelated
knowledge files; none were staged, reverted, or edited by this audit. No test,
build, migration, server, network request, or production mutation was run:
this report audits source state only and does not establish runtime behavior.
