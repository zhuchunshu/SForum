# Mail, Notifications, and SMTP Plugin Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add durable in-app notifications and asynchronous email delivery while moving every SMTP delivery concern out of core into the protected `sforum.smtp` plugin.

**Architecture:** Core owns notification records, mail delivery records, River jobs, provider selection, content rendering, APIs, and policy checks. The extension runtime exposes a typed `mail.provider` RPC; the built-in SMTP subprocess is the only SMTP implementation. API and standalone/embedded workers independently reconcile enabled plugins so queued mail works in production.

**Tech Stack:** Go 1.25, Fiber v3, pgx/v5, PostgreSQL, River, HashiCorp go-plugin/net-rpc, Nuxt 4, Vue 3, Nuxt UI 4, Bun, modular OpenAPI.

---

## File Map

- `apps/api/database/migrations/202607110016_notifications_mail_deliveries.sql`: notification, mail delivery, provider selection, and legacy-adoption schema.
- `apps/api/app/Models/Notifications/{types,store,postgres_store,service}.go`: core inbox and delivery domain.
- `apps/api/app/Jobs/Notifications/deliver_mail.go`: River mail worker with ID-only args.
- `apps/api/app/Support/Mail/{types,renderer}.go`: provider-neutral message and localized rendering only; delete core provider implementations.
- `apps/api/app/Support/Extensions/{providers,protocol,manager}.go`: first-class selection plus typed mail RPC.
- `extensions/builtin/plugins/sforum-smtp/`: manifest, Go subprocess, SMTP transport, settings route, and tests.
- `apps/api/app/Http/Controllers/Notifications/controller.go`: current-user inbox API.
- `apps/api/app/Http/Controllers/Mail/controller.go`: provider selection, delivery inspection, and queued test mail.
- `apps/api/app/Providers/{notifications,mail}.go`: route wiring and authoritative permission dependencies.
- `apps/api/bootstrap/{app,worker}.go`: API dispatcher and worker-owned extension runtime.
- `apps/web/app/{composables/useNotifications.ts,pages/notifications.vue,pages/admin/settings/mail.vue}`: inbox and provider-neutral admin UI.
- `extensions/builtin/themes/sforum-default/layer/app/components/SFNavbar.vue`: unread entry and badge.
- `contracts/openapi/{paths/notifications.yaml,paths/mail.yaml,schemas/notifications.yaml,schemas/mail.yaml,openapi.yaml}`: public contract.
- `knowledge/{index.md,modules/mail.md,modules/notifications.md,modules/extensions.md,sessions/2026-07-11-mail-notifications-smtp-plugin.md}`: project memory.

### Task 1: Add Persistent Notification and Delivery Records

**Files:**
- Create: `apps/api/database/migrations/202607110016_notifications_mail_deliveries.sql`
- Create: `apps/api/app/Models/Notifications/types.go`
- Create: `apps/api/app/Models/Notifications/store.go`
- Create: `apps/api/app/Models/Notifications/postgres_store.go`
- Create: `apps/api/app/Models/Notifications/postgres_store_test.go`

- [ ] **Step 1: Write failing PostgreSQL store tests**

Test `CreateBundleTx` with a real test transaction: insert one inbox row and one queued delivery with idempotency key `comment:42:mention:7`; assert a second insert returns the existing IDs, unread count is one, and transaction rollback leaves both tables empty.

```go
bundle, err := store.CreateBundleTx(ctx, tx, notifications.CreateBundleInput{
    Notification: notifications.CreateInput{RecipientUserID: 7, Type: notifications.TypeMention, TargetType: "comment", TargetID: 42, DedupeKey: "comment:42:mention:7"},
    Delivery: notifications.CreateDeliveryInput{Recipient: "member@example.com", TemplateKey: "forum.mention", IdempotencyKey: "comment:42:mention:7"},
})
require.NoError(t, err)
require.Equal(t, notifications.DeliveryQueued, bundle.Delivery.Status)
```

- [ ] **Step 2: Run the focused test and verify RED**

Run: `cd apps/api && go test ./app/Models/Notifications -run TestPostgresStore -count=1`

Expected: FAIL because the migration, package, and `CreateBundleTx` do not exist.

- [ ] **Step 3: Add the schema and minimal store**

Create `notifications` and `mail_deliveries` with the fields and indexes from the design. Use unique `dedupe_key` and `idempotency_key`. Add `mail_provider_selection(slot PRIMARY KEY, extension_id, updated_at)` and a `runtime_migrations(key PRIMARY KEY, completed_at)` marker table for legacy adoption. Implement:

```go
type Store interface {
    CreateBundleTx(context.Context, pgx.Tx, CreateBundleInput) (Bundle, error)
    List(context.Context, ListInput) (Page, error)
    UnreadCount(context.Context, int64) (int64, error)
    MarkRead(context.Context, int64, int64) error
    MarkAllRead(context.Context, int64) (int64, error)
    GetDelivery(context.Context, int64) (MailDelivery, error)
    UpdateDelivery(context.Context, DeliveryUpdate) error
}
```

Return the pre-existing logical row on unique-key conflicts rather than creating duplicates.

- [ ] **Step 4: Run store and migration tests**

Run: `cd apps/api && go test ./app/Models/Notifications ./database/migrator -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/database/migrations/202607110016_notifications_mail_deliveries.sql apps/api/app/Models/Notifications
git commit -m "feat(api): add notification and mail delivery records"
```

### Task 2: Promote `mail.provider` to a Typed Extension Contract

**Files:**
- Modify: `apps/api/app/Support/ExtensionManifest/manifest.go`
- Modify: `apps/api/app/Support/ExtensionManifest/manifest_test.go`
- Modify: `apps/api/app/Support/Extensions/providers.go`
- Modify: `apps/api/app/Support/Extensions/providers_test.go`
- Modify: `apps/api/app/Support/Extensions/protocol.go`
- Modify: `apps/api/app/Support/Extensions/protocol_test.go`
- Modify: `apps/api/app/Support/Extensions/manager.go`
- Modify: `apps/api/app/Models/Extensions/store.go`
- Modify: `apps/api/app/Models/Extensions/postgres_store.go`

- [ ] **Step 1: Write failing manifest, registry, and RPC tests**

Assert that `mail.provider` is accepted, selection rejects a disabled/non-declaring extension, disabling the selected extension clears the row, and net/rpc forwards this request/result exactly:

```go
type MailProviderRequest struct {
    DeliveryID, CorrelationID string
    FromAddress, FromName string
    To []string
    Subject, TextBody, HTMLBody string
}
type MailProviderResponse struct {
    OK bool
    Classification string // temporary | permanent
    Reason string
    Message string
}
```

- [ ] **Step 2: Run and verify RED**

Run: `cd apps/api && go test ./app/Support/ExtensionManifest ./app/Support/Extensions -run 'Mail|Provider' -count=1`

Expected: FAIL because the slot, persistent selection, and `SendMail` protocol method are absent.

- [ ] **Step 3: Implement the smallest typed contract**

Add `mail.provider` to `knownProviderSlot`. Replace the in-memory-only mail selection path with a store-backed `MailProviderRegistry` that resolves only enabled plugins declaring the slot. Extend `PluginProtocol`, `netRPCClient`, and `netRPCServer` with `SendMail(MailProviderRequest)`. Add `Manager.SendMail(ctx, extensionID, request)` and return `extension.runtime_unavailable` when the subprocess is absent.

- [ ] **Step 4: Run extension tests**

Run: `cd apps/api && go test ./app/Support/ExtensionManifest ./app/Support/Extensions ./app/Models/Extensions -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/app/Support/ExtensionManifest apps/api/app/Support/Extensions apps/api/app/Models/Extensions
git commit -m "feat(api): add mail provider plugin contract"
```

### Task 3: Build the Protected SMTP Plugin

**Files:**
- Create: `extensions/builtin/plugins/sforum-smtp/sforum.extension.json`
- Create: `extensions/builtin/plugins/sforum-smtp/backend/go.mod`
- Create: `extensions/builtin/plugins/sforum-smtp/backend/main.go`
- Create: `extensions/builtin/plugins/sforum-smtp/backend/plugin.go`
- Create: `extensions/builtin/plugins/sforum-smtp/backend/smtp_transport.go`
- Create: `extensions/builtin/plugins/sforum-smtp/backend/smtp_transport_test.go`
- Create: `extensions/builtin/plugins/sforum-smtp/backend/settings.go`
- Create: `extensions/builtin/plugins/sforum-smtp/backend/settings_test.go`
- Modify: `apps/api/Dockerfile`
- Modify: `compose.yaml`

- [ ] **Step 1: Write failing local SMTP transport tests**

Use a loopback test SMTP listener to assert plain SMTP, STARTTLS, implicit TLS, AUTH PLAIN, UTF-8 subject encoding, HTML/text multipart output, missing host as permanent failure, and connection refusal as temporary failure. Never access the public network.

- [ ] **Step 2: Run and verify RED**

Run: `cd extensions/builtin/plugins/sforum-smtp/backend && go test ./... -count=1`

Expected: FAIL because the plugin package is absent.

- [ ] **Step 3: Implement the plugin**

Declare ID `sforum.smtp`, `type: plugin`, backend RPC v1, `mail.provider`, manifest settings for sender/host/port/encryption/username/password, and host-checked settings/test routes. Implement `SendMail` entirely in this plugin using `net/smtp`, `crypto/tls`, `mime/multipart`, and `mime/quotedprintable`. The core import is limited to the shared provider protocol package; no SMTP source is added under core.

- [ ] **Step 4: Build the executable into the built-in package and run tests**

Run: `cd extensions/builtin/plugins/sforum-smtp/backend && go test ./... -count=1 && go build -o ../backend/plugin .`

Expected: PASS and an executable at the manifest entry path. Update the API image build so protected built-in plugin executables are built for the target platform rather than committing the binary.

- [ ] **Step 5: Commit**

```bash
git add extensions/builtin/plugins/sforum-smtp apps/api/Dockerfile compose.yaml
git commit -m "feat(plugin): add protected smtp mail provider"
```

### Task 4: Replace Core Mail Providers with Delivery Dispatch

**Files:**
- Delete: `apps/api/app/Support/Mail/smtp.go`
- Delete: `apps/api/app/Support/Mail/dev_log.go`
- Delete: `apps/api/app/Support/Mail/noop.go`
- Replace: `apps/api/app/Support/Mail/service.go`
- Modify: `apps/api/app/Support/Mail/types.go`
- Modify: `apps/api/app/Support/Mail/service_test.go`
- Create: `apps/api/app/Support/Mail/renderer.go`
- Create: `apps/api/app/Support/Mail/renderer_test.go`
- Create: `apps/api/app/Jobs/Notifications/deliver_mail.go`
- Create: `apps/api/app/Jobs/Notifications/deliver_mail_test.go`

- [ ] **Step 1: Write failing dispatcher tests**

Cover selected healthy plugin success, no selection producing `skipped/provider_unavailable`, temporary response returning an error for River retry, permanent response persisting `failed`, and retry idempotency avoiding a second send after `sent`.

```go
err := worker.Work(ctx, &river.Job[DeliverMailArgs]{Args: DeliverMailArgs{DeliveryID: 41}})
require.NoError(t, err)
require.Equal(t, notifications.DeliverySkipped, store.delivery.Status)
require.Equal(t, "provider_unavailable", store.delivery.Reason)
```

- [ ] **Step 2: Run and verify RED**

Run: `cd apps/api && go test ./app/Support/Mail ./app/Jobs/Notifications -count=1`

Expected: FAIL because `DeliverMailArgs` and provider-neutral service do not exist.

- [ ] **Step 3: Implement renderer and River worker**

Keep only `Message`, renderer inputs, and localized templates in `Support/Mail`. Implement `DeliverMailArgs.Kind() == "mail.deliver"` with queue `mail`, ID-only payload, delivery status transitions, current provider resolution, and typed RPC invocation. Sanitize stored errors and never include settings or bodies in River args.

- [ ] **Step 4: Prove no provider implementation remains in core**

Run: `cd apps/api && go test ./app/Support/Mail ./app/Jobs/Notifications -count=1 && ! rg -n 'net/smtp|smtp\.NewClient|PlainAuth' app/Support/Mail`

Expected: tests PASS and `rg` finds no SMTP implementation in core.

- [ ] **Step 5: Commit**

```bash
git add apps/api/app/Support/Mail apps/api/app/Jobs/Notifications
git commit -m "refactor(api): dispatch mail exclusively through plugins"
```

### Task 5: Wire Plugin Runtime into API and Standalone Workers

**Files:**
- Modify: `apps/api/bootstrap/app.go`
- Modify: `apps/api/bootstrap/worker.go`
- Modify: `apps/api/bootstrap/worker_test.go`
- Modify: `apps/api/app/Providers/extensions.go`

- [ ] **Step 1: Write failing runtime assembly tests**

Inject a fake starter and assert both API assembly and `newWorkerWithPool` sync built-ins, reconcile enabled `sforum.smtp`, register `mail.deliver`, and close plugin subprocesses during shutdown.

- [ ] **Step 2: Run and verify RED**

Run: `cd apps/api && go test ./bootstrap -run 'Mail|ExtensionRuntime' -count=1`

Expected: FAIL because the worker has no extension runtime or mail worker.

- [ ] **Step 3: Implement worker-owned runtime assembly**

Create extension store/service/runtime inside `newWorkerWithPool`, call `SyncBuiltins`, run legacy adoption, reconcile enabled plugins, construct the notification store/provider registry, and register the mail worker. Add runtime closure to `Worker.Close`. In API assembly, replace synchronous mail service injection with a notification mail dispatcher that creates delivery rows and River jobs.

- [ ] **Step 4: Run bootstrap tests**

Run: `cd apps/api && go test ./bootstrap ./app/Providers -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/bootstrap apps/api/app/Providers/extensions.go
git commit -m "feat(api): run mail plugins in api and worker processes"
```

### Task 6: Migrate Legacy SMTP Settings Safely

**Files:**
- Create: `apps/api/app/Models/Notifications/legacy_mail.go`
- Create: `apps/api/app/Models/Notifications/legacy_mail_test.go`
- Modify: `apps/api/app/Models/Options/types.go`
- Modify: `apps/api/app/Models/Options/service.go`
- Modify: `apps/api/app/Models/Options/postgres_store.go`
- Modify: `apps/api/database/migrations/202607110016_notifications_mail_deliveries.sql`

- [ ] **Step 1: Write failing adoption tests**

Test legacy SMTP values and password copy into `sforum.smtp` settings, SMTP selection only after copy succeeds, `dev_log/noop/unknown` becoming unconfigured, repeated adoption preserving newer plugin values, and the completion marker containing no secret.

- [ ] **Step 2: Run and verify RED**

Run: `cd apps/api && go test ./app/Models/Notifications -run TestAdoptLegacyMail -count=1`

Expected: FAIL because `AdoptLegacyMail` does not exist.

- [ ] **Step 3: Implement idempotent post-sync adoption**

Execute after `SyncBuiltins`. Lock the marker row transactionally; if absent, copy only missing plugin setting keys, select/enable SMTP only for legacy `smtp`, then record `mail_provider_plugin_v1`. Remove SMTP fields and provider normalization from the core options catalog after all runtime reads use the new registry.

- [ ] **Step 4: Run options and migration tests**

Run: `cd apps/api && go test ./app/Models/Notifications ./app/Models/Options ./database/migrator -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/app/Models/Notifications apps/api/app/Models/Options apps/api/database/migrations/202607110016_notifications_mail_deliveries.sql
git commit -m "feat(api): migrate legacy smtp settings to plugin"
```

### Task 7: Add Transactional Forum Notification Fanout

**Files:**
- Create: `apps/api/app/Models/Notifications/service.go`
- Create: `apps/api/app/Models/Notifications/service_test.go`
- Modify: `apps/api/app/Models/Forum/store.go`
- Modify: `apps/api/app/Models/Forum/postgres_store.go`
- Modify: `apps/api/app/Models/Forum/service.go`
- Modify: `apps/api/app/Models/Forum/service_test.go`
- Modify: `apps/api/app/Models/Moderation/service.go`
- Modify: `apps/api/app/Models/Moderation/service_test.go`

- [ ] **Step 1: Write failing recipient and transaction tests**

Test reply author, unique mentions, approval/rejection author, self-filtering, distinct reply+mention records, and rollback when River transactional insertion fails. Use parsed rich-content mention identities already available in forum content; do not regex rendered HTML.

- [ ] **Step 2: Run and verify RED**

Run: `cd apps/api && go test ./app/Models/Notifications ./app/Models/Forum ./app/Models/Moderation -run 'Notification|Mention|Decision' -count=1`

Expected: FAIL because notification fanout is not wired.

- [ ] **Step 3: Implement the fanout boundary**

Add a narrow `Notifier` interface to Forum and Moderation services. Ensure PostgreSQL domain writes expose/use the same `pgx.Tx`; call `CreateBundleTx` and River `InsertTx` before commit. Generate deterministic keys such as `comment:{id}:reply:{recipient}` and `moderation:{decisionID}:{status}:{recipient}`.

- [ ] **Step 4: Run domain tests**

Run: `cd apps/api && go test ./app/Models/Notifications ./app/Models/Forum ./app/Models/Moderation -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/app/Models/Notifications apps/api/app/Models/Forum apps/api/app/Models/Moderation
git commit -m "feat(api): create forum notifications transactionally"
```

### Task 8: Queue Password Reset and Test Mail

**Files:**
- Modify: `apps/api/app/Models/Identity/password_reset.go`
- Modify: `apps/api/app/Models/Identity/password_reset_test.go`
- Modify: `apps/api/app/Http/Controllers/Identity/controller.go`
- Create: `apps/api/app/Http/Controllers/Mail/controller.go`
- Create: `apps/api/app/Http/Controllers/Mail/controller_test.go`
- Create: `apps/api/app/Providers/mail.go`

- [ ] **Step 1: Write failing async and enumeration tests**

Assert password-reset token plus delivery plus River job commit together; nonexistent address, provider unavailable, and enqueue success return the same public accepted envelope. Assert test mail requires `settings.manage` and returns HTTP 202 with delivery ID.

- [ ] **Step 2: Run and verify RED**

Run: `cd apps/api && go test ./app/Models/Identity ./app/Http/Controllers/Identity ./app/Http/Controllers/Mail -run 'PasswordReset|TestMail' -count=1`

Expected: FAIL because reset still sends synchronously and the mail controller is absent.

- [ ] **Step 3: Implement queued delivery**

Replace the current `mail.Send` dependency with a `QueueTemplateTx` interface. Keep confirm-token behavior unchanged. Add provider list/select/reset, bounded delivery list/detail, and test-mail endpoints to the new controller with ownership-independent `settings.manage` checks.

- [ ] **Step 4: Run identity and controller tests**

Run: `cd apps/api && go test ./app/Models/Identity ./app/Http/Controllers/Identity ./app/Http/Controllers/Mail ./app/Providers -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/app/Models/Identity apps/api/app/Http/Controllers/Identity apps/api/app/Http/Controllers/Mail apps/api/app/Providers/mail.go
git commit -m "feat(api): queue password reset and test mail"
```

### Task 9: Expose the Current User Notification API

**Files:**
- Create: `apps/api/app/Http/Controllers/Notifications/controller.go`
- Create: `apps/api/app/Http/Controllers/Notifications/controller_test.go`
- Create: `apps/api/app/Providers/notifications.go`
- Modify: `apps/api/bootstrap/app.go`

- [ ] **Step 1: Write failing API authorization tests**

Cover login-required list/count/read-all, cursor/page bounds, marking only the current user's item, 404 for another user's ID, and stable envelopes.

- [ ] **Step 2: Run and verify RED**

Run: `cd apps/api && go test ./app/Http/Controllers/Notifications -count=1`

Expected: FAIL because the controller does not exist.

- [ ] **Step 3: Implement and register routes**

Register `GET /notifications`, `GET /notifications/unread-count`, `PATCH /notifications/:id/read`, and `POST /notifications/read-all`. Resolve actor exclusively through the session manager and pass actor ID into every store operation.

- [ ] **Step 4: Run API tests**

Run: `cd apps/api && go test ./app/Http/Controllers/Notifications ./app/Providers ./bootstrap -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/app/Http/Controllers/Notifications apps/api/app/Providers/notifications.go apps/api/bootstrap/app.go
git commit -m "feat(api): expose user notification inbox"
```

### Task 10: Update the Modular OpenAPI Contract

**Files:**
- Create: `contracts/openapi/paths/notifications.yaml`
- Modify: `contracts/openapi/paths/mail.yaml`
- Create: `contracts/openapi/schemas/notifications.yaml`
- Modify: `contracts/openapi/schemas/mail.yaml`
- Modify: `contracts/openapi.yaml`

- [ ] **Step 1: Add failing contract validation fixtures**

Add path references for the four inbox endpoints and provider/delivery/test-mail endpoints before creating their schema targets.

- [ ] **Step 2: Run and verify RED**

Run: `ruby scripts/validate-openapi-refs.rb`

Expected: FAIL with missing notification/mail schema references.

- [ ] **Step 3: Define exact schemas and security**

Document HTTP 202 queued responses, paginated inbox/delivery shapes, provider health/selection, stable reasons, session security, `settings.manage`, and 401/403/404/422 responses. Remove `smtp` as a core enum and represent providers by extension ID.

- [ ] **Step 4: Validate refs**

Run: `ruby scripts/validate-openapi-refs.rb`

Expected: `OpenAPI references are valid.`

- [ ] **Step 5: Commit**

```bash
git add contracts/openapi.yaml contracts/openapi/paths/mail.yaml contracts/openapi/paths/notifications.yaml contracts/openapi/schemas/mail.yaml contracts/openapi/schemas/notifications.yaml
git commit -m "docs(api): define notification and mail provider contracts"
```

### Task 11: Build the Inbox and Unread Navbar State

**Files:**
- Create: `apps/web/app/composables/useNotifications.ts`
- Create: `apps/web/app/pages/notifications.vue`
- Create: `apps/web/tests/notificationsPage.test.ts`
- Modify: `extensions/builtin/themes/sforum-default/layer/app/components/SFNavbar.vue`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`

- [ ] **Step 1: Write failing frontend source/behavior tests**

Assert SSR-safe initial inbox fetch, authenticated unread-count fetch, badge hidden at zero, reply/mention/moderation labels and links, single/read-all actions, and no user ID in API requests.

- [ ] **Step 2: Run and verify RED**

Run: `cd apps/web && bun test tests/notificationsPage.test.ts`

Expected: FAIL because the page/composable are absent.

- [ ] **Step 3: Implement the user experience**

Use existing `$fetch`/auth composable conventions. Add a bell icon (`i-lucide-bell`) in `SFNavbar`, a stable-size badge, paginated list, empty/loading/error states, and appearance-aware success Toasts for read actions. Localize payload-based text; never render payload HTML.

- [ ] **Step 4: Run frontend tests and typecheck**

Run: `cd apps/web && bun test tests/notificationsPage.test.ts tests/defaultThemeNavbar.test.ts && bun run typecheck`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/app/composables/useNotifications.ts apps/web/app/pages/notifications.vue apps/web/tests/notificationsPage.test.ts apps/web/i18n/locales extensions/builtin/themes/sforum-default/layer/app/components/SFNavbar.vue
git commit -m "feat(web): add notification inbox and unread badge"
```

### Task 12: Make Mail Admin Provider-neutral and Add SMTP Plugin Settings

**Files:**
- Modify: `apps/web/app/pages/admin/settings/mail.vue`
- Create: `apps/web/tests/adminMailProviders.test.ts`
- Modify: `apps/web/app/config/adminModules.ts`
- Create: `extensions/builtin/plugins/sforum-smtp/frontend/components/SmtpSettings.vue`
- Modify: `extensions/builtin/plugins/sforum-smtp/sforum.extension.json`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`

- [ ] **Step 1: Write failing admin UI tests**

Assert the core page contains no SMTP host/password fields, lists plugin providers, supports select/reset/test and recent delivery states, explains unconfigured degradation, and states reset preserves plugin secrets. Assert SMTP fields exist only in the plugin management component.

- [ ] **Step 2: Run and verify RED**

Run: `cd apps/web && bun test tests/adminMailProviders.test.ts`

Expected: FAIL because SMTP fields are still core-owned.

- [ ] **Step 3: Implement provider-neutral and plugin-owned views**

Refactor `/admin/settings/mail` to call the new core APIs. Declare a digest-approved trusted admin page for `sforum.smtp` settings, with host-authorized update/test routes, explicit secret clearing, field-level permanent errors, and success Toasts. Keep the core mail page under `settings.manage`; plugin lifecycle navigation remains under `extension.manage`.

- [ ] **Step 4: Run tests and typecheck**

Run: `cd apps/web && bun test tests/adminMailProviders.test.ts tests/adminExtensions.test.ts && bun run typecheck`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/app/pages/admin/settings/mail.vue apps/web/tests/adminMailProviders.test.ts apps/web/app/config/adminModules.ts apps/web/i18n/locales extensions/builtin/plugins/sforum-smtp
git commit -m "feat(web): manage mail providers and smtp plugin"
```

### Task 13: Update Knowledge and Run the Full Gate

**Files:**
- Modify: `knowledge/index.md`
- Modify: `knowledge/modules/mail.md`
- Create: `knowledge/modules/notifications.md`
- Modify: `knowledge/modules/extensions.md`
- Create: `knowledge/sessions/2026-07-11-mail-notifications-smtp-plugin.md`

- [ ] **Step 1: Update project memory**

Record the provider-only core boundary, SMTP plugin ID/settings ownership, notification types, queues, permissions, migration behavior, endpoints, and remaining non-goals. Remove statements claiming SMTP/dev-log/noop are core providers.

- [ ] **Step 2: Run focused backend verification**

Run: `cd apps/api && gofmt -w app/Models/Notifications app/Jobs/Notifications app/Http/Controllers/Notifications app/Http/Controllers/Mail app/Support/Mail app/Support/Extensions bootstrap && go test ./...`

Expected: PASS with no race, panic, or package failure.

- [ ] **Step 3: Run contract and frontend verification**

Run: `ruby scripts/validate-openapi-refs.rb && cd apps/web && bun test && bun run typecheck`

Expected: OpenAPI valid; Bun tests and Nuxt typecheck PASS.

- [ ] **Step 4: Run the repository test gate**

Run: `./scripts/test.sh`

Expected: all Go, OpenAPI, Nuxt, and repo validation steps PASS. Do not stop or replace the user's port-3000 process.

- [ ] **Step 5: Commit documentation and any verification-only fixes**

```bash
git add knowledge
git commit -m "docs: record mail and notification architecture"
```

- [ ] **Step 6: Review final diff for boundary violations**

Run: `git status --short && rg -n 'net/smtp|smtp\.NewClient|PlainAuth' apps/api/app apps/api/bootstrap`

Expected: only intended files are changed and the SMTP search has no core result. SMTP transport matches exist only below `extensions/builtin/plugins/sforum-smtp`.
