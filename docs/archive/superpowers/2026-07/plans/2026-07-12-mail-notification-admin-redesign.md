# Mail And Notification Admin Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a discoverable Core mail/notification center and a beginner-friendly, manifest-driven SMTP plugin settings page while keeping all SMTP behavior outside Core.

**Architecture:** Extend the generic extension setting contract with provider-neutral UI metadata. Store global event/channel policy in Core Options and inject its resolved view into the notification Outbox; expose policy and self-test operations through permission-protected Core controllers. The SMTP plugin supplies all SMTP-specific choices and guidance through its manifest.

**Tech Stack:** Go 1.25, Fiber v3, PostgreSQL/pgx, River, Nuxt 4, Vue 3, Nuxt UI 4, TypeScript, Bun, modular OpenAPI.

---

## File Map

- `apps/api/app/Support/ExtensionManifest/manifest.go`: generic setting presentation schema and validation only.
- `apps/api/app/Models/Options/notification_options.go`: strongly typed recommended notification policy and persistence normalization.
- `apps/api/app/Models/Notifications/fanout.go`: channel-aware transactional projection.
- `apps/api/app/Http/Controllers/Mail/controller.go`: provider-neutral policy administration beside existing mail state.
- `apps/api/app/Http/Controllers/Notifications/controller.go`: current-admin in-app test notification.
- `apps/web/app/pages/admin/settings/mail.vue`: Core four-tab management center; split small local components only if the page approaches 500 lines.
- `apps/web/app/pages/admin/extensions/[extensionId]/pages/[...pagePath].vue`: generic metadata-driven settings controls.
- `extensions/builtin/plugins/sforum-smtp/sforum.extension.json`: SMTP-specific labels, select options, defaults, and guidance.
- `contracts/openapi/{paths,schemas}/{mail,notifications,extensions}.yaml`: changed API and manifest contracts.
- `knowledge/modules/{mail,notifications,extensions}.md` and `knowledge/index.md`: durable architecture/status notes.

The worktree already contains unrelated/uncommitted extension-localization and admin-navigation changes. Before every commit, stage explicit paths or hunks and verify `git diff --cached --name-only`; never reset or overwrite those changes.

### Task 1: Generic Extension Setting Presentation Contract

**Files:**
- Modify: `apps/api/app/Support/ExtensionManifest/manifest.go`
- Modify: `apps/api/app/Support/ExtensionManifest/manifest_test.go`
- Modify: `contracts/openapi/schemas/extensions.yaml`

- [ ] **Step 1: Write failing manifest tests**

Add tests proving a setting accepts `placeholder`, `recommendedValue`, `group`, and ordered `options`, and rejects blank/duplicate option values or a recommended value outside an options list.

```go
setting := ManifestSetting{
    Key: "encryption", Label: "Encryption", Type: "select",
    RecommendedValue: "starttls", Group: "server",
    Options: []ManifestSettingOption{{Value: "starttls", Label: "STARTTLS"}},
}
```

- [ ] **Step 2: Verify the tests fail**

Run: `cd apps/api && go test ./app/Support/ExtensionManifest -run 'TestManifestSettingPresentation' -count=1`

Expected: FAIL because the new fields/types do not exist.

- [ ] **Step 3: Implement the minimal generic schema**

Add provider-neutral fields and normalization/validation:

```go
type ManifestSettingOption struct { Value string `json:"value"`; Label string `json:"label"`; Description string `json:"description,omitempty"` }
type ManifestSetting struct {
    Key string `json:"key"`; Label string `json:"label"`; Description string `json:"description,omitempty"`
    Type string `json:"type"`; Default string `json:"default,omitempty"`
    Placeholder string `json:"placeholder,omitempty"`; RecommendedValue string `json:"recommendedValue,omitempty"`
    Group string `json:"group,omitempty"`; Options []ManifestSettingOption `json:"options,omitempty"`
}
```

Keep the contract semantic-free: no SMTP keys, modes, or ports in validation.

- [ ] **Step 4: Update OpenAPI and verify**

Document every new property in `ExtensionManifestSetting`, then run:

`ruby scripts/validate-openapi-refs.rb`

Expected: `OpenAPI refs valid` (or the repository's equivalent success message).

- [ ] **Step 5: Run focused tests and commit only this contract slice**

Run: `cd apps/api && go test ./app/Support/ExtensionManifest -count=1`

Commit: `feat(extensions): add setting presentation metadata`

### Task 2: Notification Policy Options

**Files:**
- Create: `apps/api/app/Models/Options/notification_options.go`
- Create: `apps/api/app/Models/Options/notification_options_test.go`
- Modify: the existing option-name/default/validation registry files discovered beside `apps/api/app/Models/Options/service.go`

- [ ] **Step 1: Write failing default, update, and restore tests**

Cover missing values resolving to all enabled, one channel disabled independently, invalid booleans rejected, and restore returning all enabled.

```go
want := NotificationPolicy{
    Reply: ChannelPolicy{InAppEnabled: true, EmailEnabled: true},
    Mention: ChannelPolicy{InAppEnabled: true, EmailEnabled: true},
    Moderation: ChannelPolicy{InAppEnabled: true, EmailEnabled: true},
}
```

- [ ] **Step 2: Verify focused failure**

Run: `cd apps/api && go test ./app/Models/Options -run NotificationPolicy -count=1`

Expected: FAIL because `NotificationPolicy` and option names do not exist.

- [ ] **Step 3: Implement typed policy and canonical defaults**

Define `ChannelPolicy`, `NotificationPolicy`, `NotificationPolicy(ctx)`, update inputs, and restore inputs using the existing Options service conventions. Use six boolean option names and one shared `true` default; moderation approval/rejection share one family.

- [ ] **Step 4: Run Options tests**

Run: `cd apps/api && go test ./app/Models/Options -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

Commit: `feat(options): add notification channel policy`

### Task 3: Apply Policy To Transactional Fanout

**Files:**
- Modify: `apps/api/app/Models/Notifications/outbox.go`
- Modify: `apps/api/app/Models/Notifications/fanout.go`
- Create: `apps/api/app/Models/Notifications/fanout_policy_test.go`
- Modify: `apps/api/bootstrap/app.go`
- Modify: `apps/api/bootstrap/worker.go` only if its Outbox construction needs the same dependency

- [ ] **Step 1: Write failing channel-combination tests**

For reply, mention, and moderation, prove: both enabled creates bundle; in-app only creates notification; email only creates delivery; both disabled creates neither; the surrounding transaction succeeds in all cases.

- [ ] **Step 2: Verify focused failure**

Run: `cd apps/api && go test ./app/Models/Notifications -run Policy -count=1`

Expected: FAIL because Outbox always creates both projections.

- [ ] **Step 3: Add a narrow policy reader dependency**

```go
type PolicyReader interface { NotificationPolicy(context.Context) (options.NotificationPolicy, error) }
```

Update fanout to call `CreateNotificationTx` and/or `CreateDeliveryTx` according to the event family. Reuse existing store transaction methods; do not duplicate SQL. Decide policy before writes, and return policy-load errors so configuration failures cannot silently invent behavior.

- [ ] **Step 4: Wire Options into API Outbox construction and run tests**

Run: `cd apps/api && go test ./app/Models/Notifications ./bootstrap -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

Commit: `feat(notifications): honor global channel policy`

### Task 4: Core Policy And Test APIs

**Files:**
- Modify: `apps/api/app/Http/Controllers/Mail/controller.go`
- Create or modify: `apps/api/app/Http/Controllers/Mail/controller_test.go`
- Modify: `apps/api/app/Http/Controllers/Notifications/controller.go`
- Create or modify: `apps/api/app/Http/Controllers/Notifications/controller_test.go`
- Modify: `apps/api/bootstrap/app.go`
- Modify: `contracts/openapi/paths/mail.yaml`
- Modify: `contracts/openapi/schemas/mail.yaml`
- Modify: `contracts/openapi/paths/notifications.yaml`
- Modify: `contracts/openapi/schemas/notifications.yaml`
- Modify: `contracts/openapi.yaml`

- [ ] **Step 1: Write failing authorization and behavior tests**

Cover 401, 403, successful policy read/update/restore, invalid policy payload, self-owned `admin_test` notification, no mail delivery from that test, and malformed custom test-email recipient.

- [ ] **Step 2: Verify controller tests fail**

Run: `cd apps/api && go test ./app/Http/Controllers/Mail ./app/Http/Controllers/Notifications -count=1`

Expected: FAIL for missing routes and missing server-side email validation.

- [ ] **Step 3: Implement Core endpoints**

Add provider-neutral routes under `/admin/mail/policy` plus restore, and `POST /admin/notifications/test`. Load the actor for `settings.manage`; the test notification derives `RecipientUserID` from that actor and accepts no target ID. Validate test email using `net/mail.ParseAddress` plus exact-address normalization so display-name forms are not accepted accidentally.

- [ ] **Step 4: Update modular OpenAPI and validate refs**

Run: `ruby scripts/validate-openapi-refs.rb`

Expected: success.

- [ ] **Step 5: Run focused API tests and commit**

Run: `cd apps/api && go test ./app/Http/Controllers/Mail ./app/Http/Controllers/Notifications -count=1`

Commit: `feat(api): manage and test notification channels`

### Task 5: Generic Dynamic Settings UX And SMTP Declaration

**Files:**
- Modify: `apps/web/app/utils/adminExtensions.ts`
- Modify: `apps/web/app/pages/admin/extensions/[extensionId]/pages/[...pagePath].vue`
- Modify: `apps/web/tests/adminExtensions.test.ts`
- Modify: `extensions/builtin/plugins/sforum-smtp/sforum.extension.json`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`

- [ ] **Step 1: Write failing frontend tests**

Test type normalization, ordered groups, select options, recommended-value reset that preserves secret fields, and the SMTP manifest containing `starttls`, `tls`, and `none` choices with 587 as the recommended port.

- [ ] **Step 2: Verify failure**

Run: `cd apps/web && bun test tests/adminExtensions.test.ts`

Expected: FAIL for missing presentation metadata behavior.

- [ ] **Step 3: Implement generic controls**

Render enumerated settings with `USelect`, secret values with `UInput type="password"`, booleans with a switch/checkbox, and numbers with numeric input. Render manifest descriptions and placeholders. Group by generic `group` metadata without naming SMTP groups in Core. Reset non-secret form values to `recommendedValue ?? default`; omit secrets from reset payload so configured credentials survive.

- [ ] **Step 4: Enrich only the SMTP manifest with SMTP advice**

Declare server/authentication/sender groups; encryption choices; host, port, sender examples; recommended STARTTLS/587 values; application-password guidance; and bilingual manifest-localized labels supported by the existing localization work. Keep all SMTP-specific strings in this plugin declaration.

- [ ] **Step 5: Run tests and commit scoped hunks**

Run: `cd apps/web && bun test tests/adminExtensions.test.ts`

Commit: `feat(smtp): improve provider settings experience`

### Task 6: Core Mail And Notification Admin Center

**Files:**
- Modify: `apps/web/app/config/adminModules.ts`
- Modify: `apps/web/app/pages/admin/settings/mail.vue`
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`
- Create: `apps/web/tests/adminMailNotifications.test.ts`
- Modify: `tests/validate-admin-framework.ts`

- [ ] **Step 1: Write failing page/registry tests**

Assert `/settings/mail` is a visible System child labeled Mail and Notifications, retains `settings.manage`, exposes four tabs, custom-recipient validation, current-user email prefill, policy restore, and self-test actions.

- [ ] **Step 2: Verify failure**

Run: `cd apps/web && bun test tests/adminMailNotifications.test.ts`

Expected: FAIL because the entry is absent and the current page is single-view.

- [ ] **Step 3: Implement the four-tab Core page**

Use existing Nuxt UI controls (`UTabs`, `USelect`, `UInput`, `USwitch`, `UButton`, `UTable` where appropriate), `UDashboardToolbar`, and `useAdminPage('/settings/mail')`. Keep status sections unframed, avoid nested cards, localize statuses/reasons, show persistent inline errors, and use 10-second theme-aware success Toasts.

The mail test field starts with `auth.user.email` when available but remains editable and transient. The notification test always calls the self-test route and links to `/notifications` after success.

- [ ] **Step 4: Run frontend tests and typecheck**

Run: `cd apps/web && bun test tests/adminMailNotifications.test.ts tests/adminExtensions.test.ts`

Run: `cd apps/web && bun run typecheck`

Expected: PASS.

- [ ] **Step 5: Commit only this UI slice**

Commit: `feat(web): add mail and notification admin center`

### Task 7: Knowledge, Contract, And Repository Validation

**Files:**
- Modify: `knowledge/modules/mail.md`
- Modify: `knowledge/modules/notifications.md`
- Modify: `knowledge/modules/extensions.md`
- Modify: `knowledge/index.md`
- Create: `knowledge/sessions/2026-07-12-mail-notification-admin-redesign.md`
- Modify: relevant `tests/validate-*.js|.ts` files only when they assert changed contracts

- [ ] **Step 1: Update durable documentation**

Record the generic setting metadata boundary, Core global channel policy, self-test semantics, visible admin entry, custom test recipient, default/restore behavior, and the rule that SMTP-specific advice stays in `sforum.smtp`.

- [ ] **Step 2: Run backend and contract gates**

Run: `cd apps/api && go test ./...`

Run: `ruby scripts/validate-openapi-refs.rb`

Expected: PASS.

- [ ] **Step 3: Run frontend gates**

Run: `cd apps/web && bun run typecheck`

Run: `cd apps/web && bun test`

Expected: PASS.

- [ ] **Step 4: Run full repository gate**

Run: `./scripts/test.sh`

Expected: exit 0 with Go tests, OpenAPI validation, Nuxt typecheck, and repository validators passing.

- [ ] **Step 5: Commit docs and validation adjustments**

Commit: `docs: record mail and notification administration`

### Task 8: Browser QA And Final Review

**Files:**
- Modify only files required by defects found during QA.

- [ ] **Step 1: Inspect existing dev processes without stopping port 3000**

Run: `lsof -nP -iTCP:3000 -sTCP:LISTEN` and the equivalent API-port check from `.env`/running processes.

Expected: reuse the user's frontend; start only missing API/dependencies with approved project scripts.

- [ ] **Step 2: Verify desktop and mobile flows in a real browser**

Check the System sidebar entry, all four Core tabs, provider navigation, custom email validation, queued feedback, self-test inbox result, policy toggles/reset, SMTP select/options/help text, dark mode, 1440px desktop, and 390px mobile. Confirm no overlap, blank content, nested cards, or console errors.

- [ ] **Step 3: Fix discovered defects test-first and rerun focused checks**

For each defect, add or tighten the closest automated test, observe failure, implement the smallest fix, and rerun that test plus typecheck.

- [ ] **Step 4: Run final full gate and inspect repository state**

Run: `./scripts/test.sh`

Run: `git status --short && git log -8 --oneline`

Expected: full gate exits 0; only pre-existing unrelated user changes remain unstaged.

- [ ] **Step 5: Commit QA fixes if any**

Commit: `fix(web): polish mail and notification administration`
