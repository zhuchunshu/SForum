# Extensions Module

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
- `adminFrontendDigest` covers only the prebuilt component contract and
  entry/CSS bytes. Trust uses it independently from package/backend/public-theme/
  settings changes.
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
