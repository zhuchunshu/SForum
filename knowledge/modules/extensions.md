# Extensions Module

## Accepted V3 Target (P0-P5 Complete)

The accepted target, including the canonical 99-row comparison and detailed
architecture mind map, is documented in
`../decisions/2026-07-13-trusted-plugin-theme-platform-v3.md`; its phased task
book is `../plans/2026-07-13-trusted-plugin-theme-platform-v3.md`. The remainder
of this module note describes the current implementation unless a section is
explicitly labeled as target behavior.

- Delegated extension managers may upload, statically inspect, and store inert
  packages. Static install never executes code; executable `install.plan` and
  `install` hooks are deferred to first enable after exact-digest, actor-bound
  `super_admin` confirmation.
- Route Registry v1 includes add, alias, redirect, rewrite, before, after,
  filter, wrap, replace, global middleware, uploads, streams, SSE, and
  WebSocket on any declared public/admin/API path and HTTP method. Trusted
  custom guards and raw request authority are allowed after separate high-risk
  confirmation.
- Plugins may own PostgreSQL schemas/roles, execute real migrations and
  transactions, use stable core views/commands, or explicitly request raw core
  database access.
- Host API v2 targets HashiCorp go-plugin gRPC plus Protobuf while v1 remains a
  temporary compatibility adapter.
- Hooks, components, templates, content, services, providers, jobs, commands,
  cache, assets, packages, and plugin-defined extension points become versioned
  registries. Lifecycle `uninstall.plan` and `uninstall` hooks lead business and
  external cleanup; Host cleanup remains fallback.
- Admin Surface, Query, Identity/Permission/Auth/Profile, Media Pipeline, and
  Navigation/Region registries close the high-frequency WordPress-class gaps.
  Transactional Host Commands cover normal cross-module atomic writes without
  making raw core database access the default.
- Every core module publishes an Extension Surface Matrix; Host/Frontend API LTS,
  compatibility test farm, signed marketplace index, and five independent
  reference plugins are release gates rather than ecosystem follow-ups.
- Safe mode, CLI disable, immutable snapshot rollback, audit/tracing, and
  multi-node convergence ship before the platform is considered complete.

P1 now enforces one actor-bound, one-use, maximum-five-minute challenge over
the complete exact artifact and canonical impact document. Durable grants cover
package, backend, admin frontend, migrations, declarations, authority,
Host/Frontend contracts, and dependencies. Delegated managers retain inert
upload and static preview, while only an active `super_admin` may authorize
first execution. Host-owned Safe Mode, PostgreSQL-only recovery commands,
startup-attempt containment, and trust audit events are operational.

P2 now implements explicit Manifest V3 versioning, all sharded Registry and
platform declaration families, exact `packageFiles` SHA-256 validation,
required/optional/conflict/provides graph resolution, embedded Draft 2020-12
JSON Schema, modular OpenAPI, V3 CLI scaffolds/digest refresh, fourteen
authoritative fixtures, and canonical `sforum.trust-impact@2` disclosure. The
generated author catalog is `docs/extensions/catalogs/manifest-v3.md`. These
contracts describe later runtime phases; V3 validation alone does not activate
an unimplemented Registry family.

P3 now runs exact Manifest-selected protocol v2 over go-plugin gRPC/AutoMTLS
without silent v1 downgrade. Each process receives a runtime-scoped Host broker
whose calls bind token, exact artifact/grant/epoch/instance identity, authority,
deadline, locale, and trace. Plugin-supplied Actor values are rejected and
cleared until Host-attested delegation exists. The Go SDK exposes generated Host clients;
typed own-settings Query/stream, Permission, safe Identity, declared Job
enqueue, and namespaced Audit adapters are operational. Protocol v1 remains
available, explicitly deprecated, and isolated on its legacy loopback gateway.

Service discovery now uses immutable, revisioned snapshots with deterministic
priority ordering, strict/exact SemVer resolution (including build metadata),
Host-authoritative caller grants, typed schema validation, unary and bounded
stream relay, instance-bound removal, and crash/restart reaping. Unsupported
`before`/`after`/`wrap` service composition fails closed until the P7 chain is
implemented. The Go SDK publishes immutable service descriptors and dispatches
typed handlers. `sforum.content-policy` is the first V2 built-in reference and
retains a real `protocol_v1` build-tag entry plus rollback Manifest.

P4 now has a Host-owned pure lifecycle state machine with the authoritative ten
states, six operations, eleven actions, recommended safety gates, and a single
failed/cancelled recovery path. An additive PostgreSQL operation/step ledger
stores exact artifact and authority snapshots, idempotency fingerprints, stable
step attempts, checkpoints, monotonic progress, typed errors, actor/audit
snapshots, and preserve/export/remove intent. Its repository serializes each
extension, uses revision/state CAS, and resumes the same logical operation
after a process restart. Coordinator execution claims and heartbeats every
plugin action, Host gate, and forced skip with exact revision fencing and
bounded detached terminal persistence.

Static package upgrades are inert candidates: `staged_version_id` is
separate from `active_version_id`; upload does not stop the active process,
change enabled state, clear providers, revoke the active grant, execute hooks,
or write migration execution history. Trust preview/challenge binds the staged
artifact while runtime trust still binds the active artifact. An instance-bound
admission gate closes routes, services, jobs, and schedule admission before
waiting for inflight work. First trusted enable, staged promotion, retained
runtime hooks, three uninstall data modes, Host finalization, and retry/skip/
forced recovery UI are production-wired and covered by durable recovery tests.

Versioned River plugin jobs now execute only against the live exact artifact,
trust grant, job contract, and payload schema that authorized their enqueue.
The worker cancels legacy, revoked, removed, disabled, or incompatible rows;
Manager and the V2 client repeat the check against the running and startup-
frozen Manifest to close upgrade races. The deterministic upgrade policy
selects execute, drain, declared payload migration, or cancel. Lifecycle-driven
River row enumeration/migration is part of the production drain coordinator.
Manifest V3 additionally freezes bounded `maxAttempts`, fixed/exponential retry
policy, and per-artifact job concurrency in each row. Plugin schedules publish
through River's dynamic periodic bundle; the periodic callback creates only a
Host trigger marker, and its worker must acquire exact schedule admission before
inserting the real versioned job. Disable/upgrade removes the periodic entry and
closes trigger admission through the same lifecycle boundary.

P6 now publishes exact plugin OpenAPI contracts and Host-derived route policy
metadata from the same lifecycle-fenced snapshot used for schema validation.
Unsafe operations use the existing Host client-IP limiter. A required standard
`Idempotency-Key` is available only to bounded unsafe HTTP routes and is enforced
through a 24-hour CAS-fenced replay ledger with canonical request fingerprints,
current-guard reauthorization, bounded 2xx responses, exact artifact/route/
credential scope, and fail-closed storage behavior. `extension.view` consumers
can read the immutable aggregate and generated-client metadata through separate
admin endpoints; streaming modes reject required replay both statically and at
runtime.

The 2026-07-18 response-cancellation hardening preserves and persists the last
valid response after a caller disconnects during response-stage processing.
Final Schema validation, committed-failure audit, and required-replay completion
run on a bounded context detached from that caller; runtime-owned failures still
remain inspectable. P6 is complete at **18/18** after production review.
Plugin terminal responses and response modifiers cannot author `Link`; Core and
Host canonical policy retain that metadata authority. Alias, rewrite, and
301/308 redirect output now carries one structured Host-owned canonical path,
from which the Fiber response writer generates the canonical `Link`. Stream
lifetime/lease authority, opaque binary transport, bounded backpressure, and
durable payload-free failure evidence pass the joined production gates.

## Purpose

Owns installable plugins and themes for SForum. Plugins are multi-enable
runtime extensions. Themes are Page Registry + L0/L1 runtime packages with
exactly one active theme; public activation does not rebuild Nuxt.

SForum core should stay framework-focused. Product verticals that vary by
deployment or vendor, including payment gateways, outbound mail delivery,
notification channels, analytics, and external integrations, should be
implemented as plugins by default. Core must expose the contracts that make
plugins practical: explicit events, filters, provider slots, permission gates,
admin selection/reset flows, typed payloads, SDK helpers, and protected built-in
plugins when useful. For areas such as payments, core may define canonical
intents, transactions, refunds, webhook idempotency, and entitlement interfaces
while provider-specific behavior remains in plugins.

## Current Status

### Buildless extension settings UI (P0–P6 implemented 2026-07-13)

Plugins and themes use one versioned settings document and one
host renderer. Schema UI (fields/tabs/groups/callouts) and Schema + Actions are
host-rendered and require no extension frontend build. Complex settings UI remains available
through an author-prebuilt admin micro-frontend that the host dynamically loads
after explicit digest-bound administrator trust; operators do not rebuild
SForum. The trusted Vue/static-registry release path has been removed.

- Decision: `decisions/2026-07-13-buildless-extension-settings-ui.md`
- Task book: `plans/2026-07-13-buildless-extension-settings-ui.md`
- Handoff: `sessions/2026-07-13-buildless-extension-settings-ui-plan.md`
- Settings arrays normalize to schema/form; canonical JSON keeps old arrays as
  arrays and new documents as objects.
- `SFExtensionSettingsRenderer` owns form/tabs/groups/columns/callouts and
  linear presentation fallback for plugins and themes.
- Schema field types: `text`/`string`, `number`, `boolean`, `select` (via
  `options`), `secret`, `textarea`. Optional `width: default|full` controls
  whether inputs stay capped (`max-w-xl`) or fill the column.
- Settings Actions are host-rendered allowlisted descriptors. `provider_probe`
  uses a restricted short-lived plugin process with no Host API token and no
  route/event/job/schedule/provider registration; SMTP and filesystem storage
  are reference migrations.
- `adminFrontendDigest` covers the prebuilt component contract and entry/CSS
  bytes, but V3 authorizes it only through the complete exact-artifact grant.
  A legacy frontend-only grant cannot bypass package/backend/migration or
  declaration trust changes.
- Admin Micro-frontend API v1 loads package-local `.mjs`/`.css` through an
  authenticated immutable digest endpoint after one-use actor-bound
  confirmation. Import/API/mount/CSS/cleanup/quarantine failure falls back to
  Schema UI.
- Important: do not restore a dev-only theme settings SFC that differs from
  production. The default theme should gain tabs/groups through the same Schema
  UI renderer used by uploaded themes.

### Runtime Page Registry & simple themes (accepted direction)

Public themes use **runtime Page Registry + L0/L1** (activate without Nuxt rebuild).
L2 widgets are **disabled** until integrity/trust. See remediation handoffs
2026-07-13 (security) and **round-2 lifecycle** (routes, access, loader SSR,
synchronous lifecycle, contract_version).

- **ADR:** `decisions/2026-07-13-runtime-page-registry-themes.md`
- **Plan (P0–P5, commit/rollback rules):**
  `plans/2026-07-13-runtime-page-registry-themes.md`
- **Round-2 handoff:**
  `sessions/2026-07-13-runtime-page-registry-round2-remediation.md`
- **P0 inventory (page ids, reserved paths, Layer touchpoints, flags):**
  `docs/extensions/page-catalog.md` (Go catalog SOT lands in P1)
- Target: L0 skin + L1 runtime templates + L2 author-prebuilt widgets;
  themes and plugins **add/replace view pages** via Page Registry; operators
  do not rebuild SForum for normal theme activation.
- **Lifecycle closed (round-2):**
  - Deterministic route signatures + match order (static > param > catch-all)
  - Access enum fail-closed (`public|login|guest|moderation|permission`)
  - Host SSR `LoaderGateway` (loopback only; no Cookie/Auth forward)
  - Extension enable/disable registers/clears Page Registry entries immediately
  - `page_provider_bindings.contract_version` + approve/resolve re-check
  - Atomic theme contribution replace for same add paths
- **Built-in themes:**
  - `sforum.default-theme` — protected default warm public shell
  - `sforum.nocturne-theme` — second builtin **runtime-only** package
    (`extensions/builtin/themes/sforum-nocturne/`); navy + cyan L0 skin and
    home L1 hero around `<sf-home-page>`
  - Dev reference: `sforum.signal-garden` under `extensions/dev/themes/`
    (and fixtures), not builtin
- Nuxt Layer activation is removed; themes ship `theme.json`, templates, and
  static assets only.
- Narrows “no core route override” to **no core API / security route override**;
  view-page replace is an intentional new capability.

### Extension surface density (next framework track)

After F1–F4.3 platform hardening, gaps are (1) sparse filters / public UI /
entity meta and (2) **service providers not fully plugin-configurable** beyond
mail. Implementation checklist:

- Plan: `plans/2026-07-12-extension-surface-density.md` (waves **E1–E8**)
- **E1.1–E1.4 done** (core exit met; optional E1.5 skipped)
- **E2 complete** (public contribution density exit met):
  - E2.1: `forum.topic.sidebar` / `forum.topic.badges` → topic detail
    `extensionSidebar` / `extensionBadges`
  - E2.2: `forum.comment.actions` → `CommentList.extensionActions` (list-level;
    `requiresAuth` UX only; body `{ topicId, commentId }`)
  - E2.3: `forum.nav.items` (`navItem`) → `GET /site/nav-items`
    `{ items, extensionItems }`; core/operator first, contributions second;
    no `/admin` / `/api`
  - E2.4: `forum.topic.list.badges` (`topicBadge`) →
    `TopicList.extensionListBadges` list-level; default theme row pills
- F4.4 entity meta → **E3**; F4.5 feature flags → **E4** (already implemented)
- **E5 complete:** workflow reference plugin `sforum.content-policy`
  (`extensions/builtin/plugins/sforum-content-policy/`) — filters on
  topic/comment before_create(+topic update), settings, topic badge/sidebar,
  public SDK backend; authoring guide + `docs/extensions/scenario-map.md`
- **E6.0 complete:** storage plugin-provider decision
  (`decisions/2026-07-12-attachment-storage-plugin-provider.md`) + selection
  encoding helpers (`plugin:<extensionId>` in `Support/Storage`).
- **E6.1 complete:** attachment settings candidates + options accept
  `plugin:`; disable plugin clears selection to `local`; Put/Open plugin path
  fail-closed until E6.2 RPC.
- **North star next:** storage **E6.2–E6.4** (RPC + reference plugin) and
  search (**E7**); other slots in **E8**. Today only `mail.provider` is
  end-to-end RPC; storage plugin selection is wired without transport yet.
- Non-goals remain: arbitrary hooks, **core API / security route** override,
  untrusted raw HTML/script injection (L1 templates are sandboxed host
  composition; see runtime page registry ADR)

The extension foundation is implemented with plugin/theme lifecycle separation
and plugin runtime v1.

### Event / filter hardening (F1.3)

- Host event catalog (`app/Support/Events`) documents `timeoutMs` and
  `failurePolicy` on every definition. Sync filters default to
  `fail_closed` + 2000ms; observe events use `fail_open` + 5000ms.
- Sync filter/validate invokes apply a host `context.WithTimeout` and force
  `extension.hook_timeout` when the plugin ignores cancellation.
- Sync deliveries are recorded in `extension_event_deliveries` (same log as
  observe) so slow (`extension.hook_slow`, ≥500ms) and failed hooks are
  visible under the admin Event Log.
- **Rule for plugin authors:** never block a filter with heavy I/O, mail, or
  indexing — enqueue a River job instead.

### Audit (F1.4)

- Extension enable / disable / install / theme activate also append to host
  `audit_events` (in addition to `extension_events`).
- Settings updates append `settings.update` with changed option **names** only
  (secrets never logged).
- Permission/role grant paths already wrote `audit_events` in identity.
- Daily schedule `audit.cleanup_events` deletes rows older than 90 days
  (recommended retention; runtime option can follow later).

### Capability grants + Host API (F2.1 / F2.2)

- Catalog: `app/Support/Capabilities` (`host.api`, `settings.own`,
  `permissions.check`, `jobs.enqueue`, `audit.append`, `net.outbound`,
  `users.read`) with risk tiers and admin copy.
- Manifest field `capabilities: string[]` (plugins only; validated against
  catalog). Host implies minimal grants from backend/jobs/settings/
  `mail.provider` when omitted.
- List/detail returns `capabilityGrants` for enable-time review. First enable
  requires `POST .../enable` body `{ "confirmCapabilities": true }`.
- Host API v1 (`sforum.host/v1`): `app/Support/HostAPI` loopback gateway with
  per-extension bearer token injected as `SFORUM_HOST_API_*` env. Methods:
  Ping, CheckPermission, GetSettings, EnqueueOwnJob, AppendAudit, GetUserSafe.
- Plugin jobs enqueue as River kind `extension.plugin_job`. Client stub:
  `hostapi.Client` / `ClientFromEnv`. Decision:
  `decisions/2026-07-12-host-api-v1-capabilities.md`.
- **F4.1 public Go plugin SDK:** `apps/api/sdk/plugin` is the supported
  authoring surface (`Serve`, `Noop`, `HostFromEnv` / typed Host helpers,
  read-only catalogs for events/capabilities/contribution points/provider
  slots/core schedules, and `LoadAndTest` contract reports). CLI:
  `sforum extension test [path]` (deeper than `validate`; checks catalogs,
  capabilities, backend entry). Fixtures under
  `extensions/fixtures/plugins/` (hostapi / events / schedules) are exercised
  by `go test ./sdk/plugin`. Handoff:
  `sessions/2026-07-12-f4-1-sdk-contract-tests.md`.
- **F4.2 catalog → documentation:** same SDK catalogs render Markdown under
  `docs/extensions/catalogs/` via `sforum extension docs generate` (and
  `--check` for drift). `go test ./sdk/plugin` fails if committed docs lag
  code. Hand-written authoring guide:
  `docs/extensions/authoring-guide.md` (SMTP + Host API fixture). Handoff:
  `sessions/2026-07-12-f4-2-catalog-docs.md`.
- **F4.3 contribution point expansion:** host catalog adds
  `forum.composer.toolbar`, `forum.profile.tabs`, `admin.dashboard.widgets`,
  `system.health.checks` with host-owned payload types only. Runtime:
  `GET /composer/toolbar`, public profile `extensionTabs`, admin overview
  `extensionWidgets`, `/ready` merges health descriptors without plugin RPC.
  Handoff: `sessions/2026-07-12-f4-3-contribution-points.md`.
- **F4.4 entity meta:** `entity_field_definitions` + `entity_meta_values` (user/
  topic, host-owned EAV). Permission `entity_meta.manage`. Event
  `entity_meta.updated`. Admin `/entity-meta`. Decision:
  `decisions/2026-07-12-entity-meta-and-feature-flags.md`.
- **F4.5 feature flags:** `features.*` options (≠ RBAC). Manifest
  `requiresFeatures` gates enable. Admin `/settings/features` + restore
  defaults. Public web-options only safe flags.
- **F2.3 resilience:** per-extension concurrency semaphore (default 4),
  consecutive-failure circuit (default 5 / 30s open), hook + mail deadlines
  (protocol uses goroutine+select because net/rpc lacks context). Observe /
  fail_open skips when circuit is open (`extension.circuit_open_skipped`);
  fail_closed returns `extension.circuit_open`. Runtime `state=degraded` with
  `circuitOpen`, `consecutiveFailures`, `lastFailureReason` on admin cards.
- **F2.4 lifecycle:** same-id ZIP upload upgrades (drain runtime, status →
  `installed`, trust revoke on digest change, re-enable required). Uninstall via
  `DELETE /admin/extensions/{id}` after disable; builtin/system blocked; package
  dir removed unless `retainPackage`. Migration ledger
  `extension_migration_ledger` records `manifest.migrations` paths+checksums
  without executing SQL (v1). Disable drains subprocess before status change.
  See `sessions/2026-07-12-f2-4-extension-lifecycle.md`.

- `extension.view`, `extension.plugin.manage`, and `extension.theme.manage`
  separate inspection, plugin lifecycle/configuration, and theme activation.
  They are seeded for `super_admin`; `extension.manage` is only a parent alias.
- Extension packages are uploaded as ZIP archives through the admin API and
  must include a root `sforum.extension.json` manifest.
- Backend model code validates manifest identity, required description, URL,
  author metadata, type, version, compatibility field, backend entry paths,
  theme/component asset paths, admin page declarations, migrations, and unsafe ZIP
  paths before writing files.
- Installed packages are stored under `EXTENSION_ROOT`, not in the public
  attachment system.
- Database tables include `extensions`, `extension_versions`,
  `extension_settings`, `extension_events`, `extension_event_deliveries`, and
  `extension_frontend_trust_grants`.
- Extension rows include source metadata: `source` (`builtin` or `uploaded`),
  `is_system`, and `is_deletable`.
- Startup sync reads `BUILTIN_EXTENSION_ROOT`, registers the protected built-in
  `sforum.default-theme`, and repairs unsafe theme state so the built-in default
  theme is active when no theme or an uploaded theme is active in v1. It also
  prunes stale `source=builtin` rows whose manifest directories no longer exist
  under the current built-in extension tree.
- Production and development Compose builds now use the repository root as the
  API/web build context so images can copy `extensions/builtin`. API and worker
  containers set `BUILTIN_EXTENSION_ROOT=/app/extensions/builtin`, store uploaded
  archives under the persistent `extension_packages` volume mounted at
  `/var/lib/sforum/extensions`, and keep attachment uploads separate.
- Extension verify, plugin enable, and theme activation require the active
  package path and installed manifest to exist. Theme verify checks `theme.json`,
  templates, and assets; plugins use backend preflight.
- Admin API routes:
  - `GET /api/v1/admin/extensions`
  - `POST /api/v1/admin/extensions`
  - `POST /api/v1/admin/extensions/:id/enable` for plugins
  - `POST /api/v1/admin/extensions/:id/disable` for plugins
  - `POST /api/v1/admin/extensions/:id/verify` for plugin preflight or buildless
    theme package verification
  - `POST /api/v1/admin/extensions/:id/activate` synchronously activates themes
  - `GET /api/v1/admin/extensions/:id/events`
  - `GET /api/v1/admin/extensions/navigation`
  - `GET /api/v1/admin/extensions/:id/settings`
  - `PUT /api/v1/admin/extensions/:id/settings`
  - `POST /api/v1/admin/extensions/:id/settings/reset`
  - `GET /api/v1/admin/extensions/contribution-points`
  - `GET /api/v1/admin/extensions/contributions`
  - `GET /api/v1/admin/extensions/event-definitions`
  - `GET /api/v1/admin/extensions/event-deliveries`
  - `ALL /api/v1/extensions/:extensionId/*` proxies declared enabled plugin
    routes after host-side route matching and access checks.
- The admin UI has an independent "Extensions" sidebar folder registered
  through the low-code admin module registry and protected by
  `extension.manage`. Submenus: Overview, Plugins, Themes, Settings, Event
  Log, Extension Points, and Page Registry. Enabled plugins and the active
  theme can inject manifest-declared core-container admin pages under the
  fixed `/extensions/{id}/pages/*` admin namespace; installed extensions also
  have a "Manage" entry from plugin/theme list rows.
- **App Store** is a separate top-level sidebar folder (`admin.nav.extensionStore`)
  with Themes (`/extensions/store/themes`) and Plugins
  (`/extensions/store/plugins`). Legacy `/extensions/store` redirects to the
  plugins shelf. Both shelves are framework shells only (sticky
  search/sort/category chips + card grid + coming-soon banner). They do not
  call a remote catalog or install APIs yet; operators still manage packages
  via Plugins/Themes ZIP upload.
- Theme rows show `enabled` as "current theme" rather than "enabled". Activation
  synchronously replaces Page Registry bindings and active skin assets without
  a worker, Nuxt build, or Web restart.
- Overview, Plugins, and Themes list rows show the extension manifest
  `description` under the name (up to two lines) so operators can scan purpose
  without opening the detail panel. The overview detail panel also shows the
  full description.
- Manifest `langs` is optional localized display overrides. Top-level
  `name`/`description`/`url`/`author` remain the default English copy. When
  `langs` is absent, hosts use the top-level fields as-is and no translation
  work is required. `langs.zh` (or `zh-CN`) may override the same display
  fields; UI locale `zh-CN` also matches short code `zh`. Built-in SMTP plugin
  and default theme ship Chinese overrides. Admin list/detail rows recompute
  display copy from the current Nuxt i18n locale, so switching language updates
  names and descriptions immediately. Built-in package changes require an API
  restart (`SyncBuiltins` on boot) before the stored active version reflects a
  new `langs` block.
- Extension pages do not poll build progress; lifecycle mutations return their
  final extension state synchronously.
- The admin Event Log page uses a non-shrinking page wrapper inside the admin
  scroll panel, so long event-definition and audit sections expand normally
  instead of being clipped by the dashboard flex layout. Event definitions,
  delivery attempts, and lifecycle audit rows are paginated independently, with
  long event names, IDs, and error messages wrapping instead of being truncated.
- Extension Platform v2 direction is accepted. The target is a complete
  operator loop: upload, manifest inspection, permission/risk review, enable or
  activate, configure, observe logs/errors/event deliveries, disable or
  roll back, and restore safe defaults. SForum should feel WordPress-like to
  site operators while keeping plugins and themes behind explicit Go API + Nuxt
  SSR contracts.
- Extension admin manifest v2 is implemented for management entry and sidebar
  behavior: new manifests may declare `admin.entry` and `admin.pages[]`,
  legacy `adminPages` remains compatible, `Manage` resolves inside the admin
  shell, and sidebar injection requires explicit `menu: true`.
- Typed extension contributions are implemented as an Itf-inspired but
  host-owned registry. Plugins may declare `contributions[]` to known points;
  core validates payloads, resolves effective enabled-plugin contributions,
  exposes read-only admin inspection, and the first runtime consumer is
  `forum.topic.actions`.
- Trusted admin settings components use Admin Micro-frontend API v1 only.
  Authors ship package-local prebuilt `.mjs`/`.css`; the host loads them from an
  immutable digest endpoint after exact `super_admin` approval. Jobs and other
  admin modules do not expose executable extension slots.

## Boundaries

- Extension settings stay in `extension_settings`; do not put extension-owned
  configuration into `web_options`. Settings are resolved from
  manifest-declared defaults plus stored values and can be reset to manifest
  defaults.
- Extension archive files stay under `EXTENSION_ROOT`; do not expose them
  through public attachment URLs.
- Plugin runtime v1 uses HashiCorp go-plugin subprocess handshakes, starts
  enabled plugin backends on API startup, proxies declared plugin routes under
  `/api/v1/extensions/:extensionId/*`, emits lifecycle hooks, and exposes a
  provider slot registry with built-in defaults.
- Plugin event and extension-point v1 keeps extension points explicit. Plugins
  may declare first-class `events`; legacy `hooks` remain a compatibility alias.
  Core route overriding and arbitrary monkey-patching are not allowed.
  Replacement behavior must go through core-owned filter events or Provider
  Slots. Sync write-path hooks today: `topic.before_create`,
  `topic.before_update` (E1.2), `comment.before_create` (E1.1),
  `user.before_register` validate (E1.3), `attachment.before_upload` validate
  (E1.4). Lifecycle and post-commit observes remain:
  `user.registered` / `topic.*` / `comment.created` / `attachment.uploaded`.

  This is the current v1 boundary only. V3 replaces it with exact-artifact
  trusted Route/Hook/Service registries; see the accepted target at the top of
  this note.
- Declarative contributions are separate from events, filters, provider slots,
  and routes. Contributions are ordered descriptors that a host-owned consumer
  interprets; they do not execute code, override routes, render raw HTML, or
  bypass API policy checks. The first point, `forum.topic.actions`, maps
  enabled plugin descriptors to topic-page buttons that call declared extension
  routes through the normal route proxy.
- The trusted-component direction does not turn contributions into executable
  UI. Descriptor points remain host-rendered. Prebuilt executable code is
  confined to Settings Document component mode and exact digest trust. Plugins
  still cannot create points, override core routes, execute components during
  SSR, or inject code into public theme UI.
- Event delivery attempts are recorded separately from lifecycle audit logs in
  `extension_event_deliveries`. The runtime has River job args and worker
  plumbing for durable async delivery, and falls back to inline delivery when no
  dispatcher is configured.
- Theme packages use runtime `theme.json` (L0 skin + L1 page contributions).
  `ActivateTheme` is **synchronous**: DB active theme + Page Registry register;
  no Nuxt build, no theme `current.json`, no Nitro restart. Public pages live
  on the host; themes may replace/add views via registry (core API / security
  routes remain non-overridable). Plugin `replace` of core pages requires
  super_admin approval; theme activate auto-binds that theme's replaces.
- There is no runtime extension frontend build, dependency installation, host
  peer resolution, release coordinator, or admin registry. `bun run dev` is
  plain Nuxt; production starts the Nuxt output directly.
- Keep plugin `Enable/Disable` separate from theme `Activate`. Do not call
  plugin runtime hooks when activating a theme.
- Backend plugin packages can declare a backend entry and RPC protocol. The
  first supported protocol is `hashicorp-go-plugin` protocol version 1.
- Payment, mail, notification, analytics, and integration packages must not
  override core routes or smuggle vendor-specific behavior into core modules.
  They should use declared plugin routes, explicit host events, and provider
  slots owned by the module whose behavior they extend.

## Permissions

- Fine-grained keys are `extension.view`, `extension.plugin.manage`, and
  `extension.theme.manage`; the legacy parent `extension.manage` expands only
  to those three. API policy remains authoritative.

Frontend visibility mirrors this permission for navigation only. API policy
checks remain authoritative.

## Manifest

The manifest file is `sforum.extension.json`. It is always the **only package
entrypoint** (ZIP root, builtin tree, verify/enable, CLI).

Required identity fields: `id`, `name`, `description`, `url`, `author`,
`version`, `type`, and `sforumVersion`.

Capability fields: `permissions`, `settings`, `migrations`, `backend`,
`frontend`, `adminPages`, `routes`, `hooks`, `events`, `jobs`, `providers`, and
`contributions`. Optional identity localization: root `langs` (or includes;
see below).

The v2 admin declaration is an `admin` object. Existing top-level
`adminPages` should be compatibility-mapped during migration.

### Multi-file authoring (`includes`)

Complex packages may keep a thin root file and move bulky blocks into partials
via optional `includes`. Host `LoadPackage` / ZIP install merges includes into
one `Manifest`, then runs existing `Normalize` / `Validate`. Downstream code
still sees a single merged model (stored as canonical merged JSON).

Reference packages:

- Provider (mail): `extensions/builtin/plugins/sforum-smtp/`
- Workflow (filters + contributions): `extensions/builtin/plugins/sforum-content-policy/`

SMTP reference package uses
`manifest/langs/`, `manifest/settings.json`, `manifest/contributions.json`,
`manifest/admin.json`, and `manifest/frontend.json`.

Decision: `knowledge/decisions/2026-07-12-extension-manifest-split.md`  
Plan: `docs/superpowers/plans/2026-07-12-extension-manifest-split.md`

**Root file should hold:** identity defaults, high-risk runtime boundary
(`backend`, `providers`, `permissions`, `migrations` paths), and optional
`includes`.

**Include keys:** `langs`, `settings`, `contributions`, `admin`, `events`,
`routes`, `jobs`, migrations, permissions, hooks, providers, and admin pages.

**No dual source:** the same block must not appear both in the root file and
under `includes` (fail with `extension.manifest_invalid`).

Simple plugins/themes may omit `includes` and keep today's single-file layout.

#### Identity `langs` (directory-per-locale preferred)

Two declarative i18n layers stay separate; prebuilt components receive the
active locale through the bridge:

1. Identity — root defaults + `langs` / `includes.langs` → list and install review
2. Settings / contribution labels — `LocalizedText` in settings/contributions

`includes.langs` supports:

| Shape | Example |
|-------|---------|
| Directory (recommended) | `"langs": "manifest/langs"` → `manifest/langs/zh-CN.json` bodies are `ManifestLocale`; filename is the locale key |
| Explicit list | `"langs": ["manifest/langs/zh-CN.json", "..."]` |
| Single map file | `"langs": "manifest/langs.json"` → today's `map[string]ManifestLocale` |

Directory rules: only `*.json`; empty directory invalid; locale fallback after
merge remains exact → language prefix → root defaults (`LocalizedDisplay`).

Example complex layout:

```text
my-plugin/
  sforum.extension.json
  manifest/
    langs/zh-CN.json
    langs/en-US.json
    settings.json
    contributions.json
  frontend/admin/locales/   # Vue UI only
```

```json
{
  "admin": {
    "entry": "/settings",
    "pages": [
      {
        "path": "/settings",
        "label": "设置",
        "view": "settings",
        "menu": false
      },
      {
        "path": "/dashboard",
        "label": "控制台",
        "view": "content",
        "menu": true,
        "icon": "i-lucide-layout-dashboard",
        "order": 100
      }
    ]
  }
}
```

`admin.entry` selects the list-row `Manage` destination. Manage resolution
should prefer `admin.entry`, then a declared `/settings` page, then the first
declared page, then a generated system detail page. The resolved destination
must stay inside the admin shell and must not be an external URL.

Sidebar entries are opt-in. `admin.pages[].menu` defaults to `false`; only
`menu: true` pages should appear in sidebar navigation. Disabled plugins should
not contribute runtime menu entries. Inactive themes may keep management pages
available but must not take over public UI or inject sidebar entries by
default.

For `type: theme`, v1 accepts only buildless runtime packages with `theme.json`
and optional `assets/`/`templates/`. Themes may declare `settings` and admin
pages, but must
not declare backend runtime or plugin execution capabilities: `backend`,
`routes`, `hooks`, `events`, `jobs`, `providers`, `contributions`,
`migrations`, or `permissions`. Invalid theme manifests use the existing 422
envelope with reason `extension.manifest_invalid`.

## Developer Console

`apps/api/cmd/sforum` provides a Laravel-artisan-style developer console.

- `./scripts/sforum.sh make:plugin`
- `./scripts/sforum.sh make:theme`
- `cd apps/api && go run ./cmd/sforum extension validate <package-dir>`
  — loads via `LoadPackage` (resolves `includes`), prints summary; `--json`
  prints the merged manifest.
- `cd apps/api && go run ./cmd/sforum extension digest --write <package-dir>`
  — refreshes inline Manifest V3 `packageFiles` digests and revalidates.
- `cd apps/api && go run ./cmd/sforum extension test <package-dir>`
  — runs package/schema, host catalog, and backend contract checks.
- `cd apps/api && go run ./cmd/sforum extension docs generate --check`
  — verifies generated host and Manifest V3 author catalogs.

Both make commands support interactive Huh forms and `--no-interaction`
flag-driven generation. Default output is `extensions/dev/{plugins,themes}/{id}`;
`--builtin` targets `extensions/builtin/{plugins,themes}/{id}`.

`make:plugin --complex` scaffolds a multi-file package:

- thin `sforum.extension.json` with `includes`
- `manifest/langs/{zh-CN,en-US}.json`
- `manifest/settings.json` versioned Settings Document
- `manifest/admin.json`

All new scaffolds default to Schema/tabs. `--provider-slot <known.slot>` adds a
host-rendered `provider_probe` action and requires `--backend`.
`--prebuilt-settings` adds an Admin Micro-frontend API v1 `.mjs`/`.css`
template while retaining Schema fallback fields.

## Current settings/admin behavior

- Generic manifest settings support `placeholder`, `recommendedValue`,
  ordered `options`, and `group`. Presentation fields (`label`, `description`,
  `placeholder`, `group`, option labels) accept either a plain string or a
  locale map (`LocalizedText`). Settings GET/PUT/reset resolve copy from the
  request `Accept-Language` and return plain strings only.
- Host dynamic settings page uses `SFExtensionSettingsRenderer`; installed or
  disabled plugins and inactive themes can be configured without starting
  extension code. Enabled backend plugins still restart with rollback after
  persistence. Secrets stay encrypted/masked and blank drafts preserve them.
- Executable settings-page/header/footer contributions are not supported.
- Default theme is Schema-only (multi-tab groups/columns/callouts); its stale
  admin SFC/package/locale files were removed. Public values still resolve via
  `GET /site/active-theme/settings`.
- `mail.provider` is now implemented end-to-end. The protected `sforum.smtp`
  plugin is the first real provider vertical; core contains no SMTP provider
  code. SMTP now owns multi-locale Schema settings plus a structured Probe
  action and no custom settings SFC.

- Multi-file extension manifests are implemented: `LoadPackage`, SMTP reference
  package, `make:plugin --complex`, `extension validate`, and settings/
  contributions directory shards. See
  `decisions/2026-07-12-extension-manifest-split.md`.
- Promote Provider Slots into first-class contracts, starting with
  `mail.provider`, `notification.channel`, `payment.provider`,
  `search.provider`, `attachment.storage.provider`,
  `editor.sanitizer.provider`, and `auth.risk.provider`.
- Add preview/approval UI and stronger uninstall/rollback controls for runtime
  theme packages without introducing host builds.
- Add plugin author documentation for provider-slot based systems such as mail,
  notifications, and payments before building those verticals.
- Add upgrade, rollback, and uninstall operations.
- Add plugin migration execution, dependency checks, SForum version
  compatibility checks, signature/trust metadata, marketplace metadata, local
  debugging, packaging, SDK docs, and example plugins.

## V3 P5 database platform (complete)

- The PostgreSQL Database Registry owns deterministic schema/role names,
  credential rotation/revocation, exact `search_path`, eight-connection budgets,
  five-second statement timeouts, and fifteen-second idle-transaction timeouts.
- Plugin migrations use exact package declarations, Goose parsing without its
  global history, checksums, advisory locks, a separate durable ledger, and a
  public read-only preflight. Two independent processes prove once-only execution.
- Host-owned `sforum_core_v1` stable views expose only safe identity, public
  forum, public entity-meta, and public attachment metadata fields. PUBLIC has
  no privileges and every view is security-barrier and non-updatable.
- Protocol V2 Host Query production-binds four immutable stable-view queries
  with server-attested exact identity, live trust checks, allowlisted shapes,
  bounded cursor pagination, and read-only PostgreSQL transactions.
- Transactional Host Commands now have server-attested exact scope, a durable
  receipt ledger, transaction-scoped idempotency locking, audit insertion, and
  replay in the same PostgreSQL transaction as domain writes. Six immutable
  production definitions cover identity, topic visibility, entity metadata,
  moderation, provider-neutral entitlement, and attachment workflows. API,
  embedded worker, and standalone worker bind the catalog before plugin brokers.
- Exact trust review prominently shows database authority, core compatibility,
  backup/retention, migration digests, and transaction policy before execution.
- The core migrator blocks before Goose when an enabled exact-trusted
  `raw_core`/`kernel` declaration excludes the target SForum release.
- Host Query emits bounded redacted traces; direct-role SQL tracing remains an
  operator-owned PostgreSQL boundary because plugin-controlled
  `application_name` is not trusted attribution.
- Protocol V2 DatabaseService now has an exact artifact-bound immutable catalog
  core, typed parameters/results, bounded own-schema transactions, durable
  idempotent replay, audit, and real PostgreSQL isolation/revocation proof. It
  is registered fail-closed but not production-bound until exact manifest
  operation declarations are loaded before broker registration.
- The V3 boundary is now frozen as additive grants. The legacy single
  `database.authority` enum expands cumulatively for compatibility; new
  manifests declare an exact grant set. Direct credentials use per-runtime
  lease roles so rolling source and target runtimes coexist until exact drain.
- Exact `raw_core` runtime leases apply physical PostgreSQL ACLs for disclosed
  Core DML while rejecting DDL, ownership, role inheritance, River internals,
  arbitrary function execution, PUBLIC escape, and foreign-owned objects.
  Durable CAS fences, heartbeat/drain/reaper cleanup, and kernel-owner
  reconciliation prevent stale source authority from surviving an upgrade.
- Actor-scoped Host Commands use short-lived Host-signed delegation from a core
  route/admin invocation. The provider-neutral entitlement minimum contains
  subject, resource/capability, lifecycle state, source, validity, idempotency,
  and audit without billing semantics.
- Provider-neutral entitlement persistence now lives in the dedicated
  `Models/Entitlements` package. Grant, revoke, expire, and effective checks
  share exact request fingerprints, transaction-scoped advisory locking,
  append-only lifecycle evidence, and same-transaction `audit_events` writes.
  Every mutation also has a `pgx.Tx` entry point so a Host Command can compose
  it with other domain writes without opening raw Core database authority.
- P5 closed at 17/17 after real PostgreSQL tests covered migration once,
  source/target lease overlap, raw-core allowed/denied operations, compatibility
  blocking, all six Host Command domains, policy/idempotency/audit/receipt/
  storage rollback, and entitlement concurrent replay/revision/revoke behavior.

## V3 P7 provider management checkpoint

- Versioned Provider Slots use durable exact-artifact choices in
  `extension_provider_slot_selections`; both contract owner and candidate bind
  active immutable extension-version rows and package digests through revision
  CAS. Selection events are append-only audit evidence.
- Runtime selection is a preferred ordering, not implicit replacement power.
  `next` tries the selected candidate then the remaining deterministic order;
  `closed` tries only the selected exact candidate. A stale closed choice fails
  before plugin execution.
- `GET /api/v1/admin/extensions/provider-slots` exposes default/selected/stale,
  runtime availability, priority ties, exact identities, fallback, timeout, and
  schemas. Super admins may select, reset, or run the side-effect-free
  `ProviderProbe` RPC; viewers may inspect selection events.
- API and worker bind the same PostgreSQL selection store. Disable/uninstall
  invalidates choices for either the contract owner or provider candidate before
  route/mail/storage provider cleanup.
- Admin UI: `/control-panel/extensions/provider-slots`. It preserves provider
  settings and secrets when restoring recommended priority defaults.

## V3 P6 streamed transport and P7 command checkpoint

- Route Protocol V2 supports bounded bidirectional streams with a 1 MiB chunk
  ceiling, an authenticated unary preflight, and one exact Manager admission
  lease held until stream termination. Fiber bridges HTTP streamed responses,
  multipart uploads, SSE, and WebSocket traffic; disconnect, disable, Safe Mode,
  upgrade drain, timeout, and cancellation terminate the stream.
- Real subprocess coverage crosses Fiber, immutable Route Registry, Dispatcher,
  Manager admission, gRPC, and the plugin SDK. It verifies intact multipart
  bytes, SSE media type/events, opaque generic binary chunks, WebSocket
  subprotocol/echo, bounded TCP slow-consumer backpressure, and admission release
  after client disconnect and ForceDrain. The production recorder persists four
  stable incident classes while normal, caller, Host writer, and drain paths
  remain incident-free. Composed non-buffered before/after/filter/wrap chains
  remain fail closed until their product semantics are frozen.
- Production Caddy sends only real WebSocket Upgrade requests directly to the
  loopback Host API ingress because Nitro's HTTP proxy does not bridge Upgrade.
  It preserves Host/Origin/session authority, excludes the Host-owned
  `vite-hmr` subprotocol, and leaves ordinary HTTP on Nuxt. Unknown and Safe
  Mode WebSocket paths fail closed in Fiber without a Nuxt fallback.
- Plugin CLI commands are Manifest-declared and published in an immutable,
  revisioned exact-artifact Registry. Namespace/conflict checks happen before
  selection; execution rechecks live trust, Safe Mode, command contract,
  artifact identity, and runtime admission before Protocol V2 invocation.
- `sforum plugin <extension-id> <command>` is the out-of-band runner. It boots
  only the selected trusted plugin runtime, never the API or Nuxt application,
  records Host-owned audit evidence, and cannot be used to bypass Safe Mode.
- Focused command, registry, transport, CLI, and race gates pass. Full nested
  builtin-plugin module gates still need Goldmark and go-redis `go.sum` entries;
  they are not considered green until those sums are repaired and tests rerun.

## V3 P7 Admin Surface And Query checkpoint

- P7 is complete at 22/22. The immutable Admin Surface Registry publishes
  declarations for all twelve V3 kinds to exact active runtime instances,
  restores/removes them through lifecycle snapshots, and invokes typed Protocol
  V2 handlers under exact admission with one frozen validator for both input and
  output.
- `GET /api/v1/admin/admin-surfaces` requires `admin.access`, filters each
  declaration by its Host-owned permission, removes modifiers whose targets are
  hidden, and redacts artifact, runtime, handler, and permission internals.
  `POST /api/v1/admin/admin-surfaces/:surfaceId/invoke` repeats authorization,
  requires a durable exact-artifact audit attempt, and returns stable localized
  errors without recording typed input or output documents. The terminal
  success/failure append is best-effort; only the pre-call attempt is guaranteed.
- The invocation boundary fences both publication races: an audit for an old
  contract cannot execute the replacement instance, and an admitted old call
  continues to validate against its original schema after a concurrent upgrade.
- Production consumption freezes kind-specific placement plus distinct input and
  output schemas, then renders concrete navigation, dashboard, list column,
  filter, row/bulk action, form, notice, editor/detail, importer, and exporter
  consumers with actor/idempotency propagation for mutations. An independent
  uploaded reference plugin exercises the complete family through exact trust,
  Protocol V2, restart restoration, permission denial, desktop, and mobile flows.
- The P7 execution-policy matrix is also closed. Named Manifest, Registry,
  Manager, real Protocol V2, and HostAPI gates join priority, exact hook/provider
  deadlines, fail-open/fail-closed isolation, version mismatch, optional and
  required dependency disable, package cycles, provider fallback, and exact
  staged-upgrade ownership.
- Host-owned role mapping is now closed by `TestP7HostOwnedRoleMappingJoined`.
  A real lifecycle Registry publication creates only a durable pending
  recommendation; restart restoration, denied operator/bearer requests, explicit
  cookie approval, additive PostgreSQL mapping/grant/audit evidence, and replay
  all pass without replacing unrelated role permissions. Identity/Auth/Profile
  surfaces, trusted automation authority, and the membership joined
  no-implicit-grant gate are closed.
- Query Registry has Host Protocol V2 `InvokeQuery`/`FilterQueryResult`,
  composite Core+Registry Schema validation, and a real-subprocess reference
  plugin fixture (`sforum-query-reference` /
  `TestReferenceQueryPluginJoinedGates`) covering pagination, filter, login,
  cost, Schema, provider failure, disable, and Safe Mode. Production Redis
  caching uses distinct API/embedded/standalone clients, transactional River
  invalidation, owner-scoped semantic tags, final permission rechecks, complete
  semantic cache-key isolation, and stale-write fences. The authoritative
  normal/race matrix joins PostgreSQL rollback, Redis process restart,
  ForceDrain, lifecycle upgrade, worker close ownership, and exact test
  discovery; both P7 Query rows are closed.
- Third-party Query execution now requires `ContextualExecutionAdmission`.
  The older release-only `ExecutionAdmission` remains source-compatible but is
  deliberately rejected at runtime because it cannot deliver Manager
  `ForceDrain` into an in-flight provider/filter transport. Exact admission also
  compares the Manager-frozen database `VersionID`; a wrong id is rejected
  before cache load even when extension version, digest, and instance match.
  Manager publishes `forced=true` only after every retained lease context has
  received the ForceDrain cause. Caller-owned custom cancellation causes retain
  both the standard cancellation class and their original domain cause.

## V3 P8 theme runtime checkpoint

- P8 is 16/18. The immutable compiler/runtime covers all 23 catalog identities,
  four-level exact fallback, typed HTML/island/SEO output, install-time template
  safety, public/admin skin isolation, exact visible activation, and multi-node
  durable convergence.
- All-catalog tests prove no theme filesystem opens after compilation and no
  provider binding Store reads after the one startup restore. Theme rendering
  and provider selection use immutable snapshots on the request path.
- The Page Registry demo add page renders its compiled L1 template and a
  defaulted Host navigation island in SSR, hydration, Baiduspider, and a real
  JavaScript-disabled browser.
- Commit `3e771b149` proves an exact theme switch survives real API and Nitro
  restarts, a concurrent exact-activation race produces one CAS winner and one
  stale-preview loser, and the winner survives a second restart. Startup builtin
  synchronization no longer resets a valid selected uploaded theme.
- The Page ViewModel row remains open: 23 typed schemas exist, but production
  construction still populates mostly base/form fields rather than each page's
  business data. Plugin business-contract preservation also remains open.

## V3 P12 runtime ownership checkpoint

- `04b159441`, `873e48248`, and `d46fd3597` close the first P12 production
  row for desired/active plugin and theme revisions, per-node acknowledgements,
  and startup reconciliation.
- An upgraded database with an active theme but an empty publication ledger now
  creates one exact genesis under the same transaction advisory lock as theme
  activation. Concurrent producers, activation races, approval preservation,
  invalid state, and ambiguous commit recovery have real PostgreSQL evidence.
- Theme heartbeat is independent from apply and has its own deadline. Ownership
  uncertainty cancels in-flight apply, acknowledgement lease loss is terminal,
  and initial readiness waits until publications committed during apply are also
  applied and acknowledged.
- Terminal Theme failure permanently closes process-local theme mutation
  admission, serializes with any in-flight activation, restores the protected
  default, and only then reaches the API failure channel. API shutdown is bounded
  and drains HTTP before shared Redis/PostgreSQL resources close.
- `fea430020` closes a publication safety gap without claiming the full second
  P12 row. Install, upgrade, and rollback now rebuild the canonical exact
  migration plan and lock its durable `target_ready` proof in the same
  transaction before publishing a target runtime revision. Missing, failed,
  malformed, or drifted proofs leave the source revision and publication marker
  untouched; enable and deactivate operations remain outside this migration gate.
- Real PostgreSQL evidence covers eight concurrent migration reconcilers, one SQL
  execution, proof-row lock overlap with eight publication committers, atomic
  marker/revision binding, failed migration, plan drift, and replay through a new
  connection pool. Full install/rollback multi-node acknowledgement and every
  runtime-publication producer still need end-to-end evidence before P12 task 2
  can close.
- This does not close P12 staged/canary rollout, migration-once rolling
  activation, multi-node upgrade/rollback tests, compatibility, marketplace,
  observability, privacy, or developer-workflow rows.
- **Builtin plugin SaveBuiltin desired-set (narrow):** plugin `SaveBuiltin`
  acquires `pluginRuntimeDesiredSetLock` before extension row locks. With no
  publication it only advances active and leaves genesis to
  `EnsureInitialPluginRuntimePublication`. With an existing full-set it never
  reprojects all enabled plugins; it preserves unrelated members and never
  resurrects a missing member (trust-revocation / disable). Upgrade source is
  the latest immutable publication member (exact `extension_versions` row it
  names), not mutable `active_version_id`, so a prior unsafe SyncBuiltins that
  already set active=B while publication still holds A still appends actorless
  A→B upgrade. Member already equal to target is idempotent; only a newly
  inserted executable builtin enable-adds. Themes and EnsureInitial one-shot
  genesis are unchanged.

## V3 P9 stable component identity checkpoint

- P9 is 1/16. Commit `a805cbe01` adds the neutral, standard-library-only
  `Support/ComponentCatalog` leaf and generates 119 active Core UI targets from
  the reviewed identity map. Every target freezes its stable ID, contract,
  page/component kind, source/route, and explicit public/admin owners; catalog
  and lookup results detach owner slices from caller mutation.
- `ExtensionManifest` depends only on the neutral catalog leaf, preserving the
  future dependency direction where the Component Registry may import both.
  Manifest V3 component targets require an exact `targetId` plus
  `targetContractVersion` pair. Core contracts must match the active catalog;
  cross-plugin contracts are syntax-checked now and exact-resolved during later
  Registry publication. Themes may target only Core surfaces with public
  ownership, and non-component reserved `core.*` targets are rejected.
- UI identities have explicit `active`/`retired` state. Removal retains the row
  and appends its exact ID/contract to the immutable-path tombstone ledger.
  Catalog validation compares the ledger with full reachable Git history and a
  generated reservation artifact, rejecting deletion plus regeneration and
  active reuse of a retired ID or contract.
- The stable-ID row passed generator/catalog drift, collision, owner/source,
  tombstone transition, target-contract, theme-owner, focused Go/race,
  downstream extension, vet/build, diff, and 1,842-reference OpenAPI gates.
  Component actions, provider priority/conflicts, runtime publication,
  templates/assets/L2, inspectors, and visual/failure exits remain open P9 work.
