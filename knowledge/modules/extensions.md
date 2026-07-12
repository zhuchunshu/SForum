# Extensions Module

## Purpose

Owns installable plugins and themes for SForum. Plugins are multi-enable
runtime extensions. Themes are Nuxt Layer packages with exactly one active
applied theme.

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
  encoding helpers (`plugin:<extensionId>` in `Support/Storage`). Runtime
  still L1 until E6.1+.
- **North star next:** storage **E6.1–E6.4** and search (**E7**) reach
  mail-like L4–L6; other slots in **E8**. Today only `mail.provider` is
  end-to-end; storage/search drivers remain mostly in core.
- Non-goals remain: arbitrary hooks, core route override, public raw HTML

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

- `extension.manage` is the permission for uploading, verifying, enabling
  plugins, activating the protected default theme, and inspecting extensions.
  It is seeded for `super_admin`.
- Extension packages are uploaded as ZIP archives through the admin API and
  must include a root `sforum.extension.json` manifest.
- Backend model code validates manifest identity, required description, URL,
  author metadata, type, version, compatibility field, backend entry paths,
  frontend layer paths, admin page declarations, migrations, and unsafe ZIP
  paths before writing files.
- Installed packages are stored under `EXTENSION_ROOT`, not in the public
  attachment system.
- Database tables: `extensions`, `extension_versions`, `extension_settings`,
  `extension_events`, `extension_event_deliveries`, and
  `extension_theme_releases`.
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
- Extension verify, plugin enable, and uploaded theme activation operations
  require the active package path and installed `sforum.extension.json` to still
  exist before changing runtime state. Theme verify and queued activation report
  missing packages, missing installed manifests, and missing Nuxt Layer
  directories as `extension.build_failed`; plugins keep using
  `extension.preflight_failed` for preflight failures.
- Admin API routes:
  - `GET /api/v1/admin/extensions`
  - `POST /api/v1/admin/extensions`
  - `POST /api/v1/admin/extensions/:id/enable` for plugins
  - `POST /api/v1/admin/extensions/:id/disable` for plugins
  - `POST /api/v1/admin/extensions/:id/verify` for plugin preflight or theme
    Nuxt Layer verification
  - `POST /api/v1/admin/extensions/:id/activate` for themes. Uploaded themes
    return `202 Accepted` with a queued `themeRelease`; restoring the built-in
    default theme remains immediate.
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
  Log, Extension Points, and Web Releases. Enabled plugins and the active
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
- Theme rows show `enabled` as "current theme" rather than "enabled".
  Uploaded Nuxt Layer themes can now be activated through the self-hosted theme
  runtime. Activation queues a River job, builds an isolated Nuxt/Nitro artifact,
  health-checks a preview server, writes the active release file, and lets the
  web runtime restart onto the selected artifact. Failed builds keep the
  previous active theme running.
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
- The admin extension overview now mirrors the Themes page for uploaded theme
  activation progress: queued/building/switching rows show status text,
  percent progress, helper copy, and short polling while a release is active.
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
- Trusted admin plugin components are implemented. The runtime treats arbitrary Vue components as fully trusted,
  client-only code; keeps SSR-safe slot metadata in manifest contributions;
  binds `super_admin` approval to the package digest; generates a static Nuxt
  registry; and unifies theme and plugin inputs under WebReleaseRuntime. The
  worker builds immutable artifacts, the API owns activation and plugin
  runtime state, and the web supervisor acknowledges the actual proxy target.
  The job monitoring module is the first production component-slot consumer,
  owning table-column, row-action, and detail-section slots. See
  `decisions/2026-07-10-trusted-admin-plugin-runtime.md` and
  `docs/superpowers/specs/2026-07-10-trusted-admin-plugin-runtime-design.md`.

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
- The accepted trusted-component direction does not turn all contributions
  into executable UI. Descriptor points remain host-rendered. A core module
  must explicitly declare a trusted admin component point with typed manifest
  metadata and context; only a digest-approved Web Release may attach its
  client component mapping. Plugins still cannot create points, override core
  routes, execute components during SSR, or inject into public theme UI.
- Event delivery attempts are recorded separately from lifecycle audit logs in
  `extension_event_deliveries`. The runtime has River job args and worker
  plumbing for durable async delivery, and falls back to inline delivery when no
  dispatcher is configured.
- Theme packages can declare a Nuxt Layer path. Uploaded theme activation is a
  deployment-like pipeline: manifest validation, queued release row, temporary
  Nuxt build, preview health check, atomic `current.json` switch, and web
  supervisor restart onto the new Nitro server. The selected uploaded layer is
  applied before the protected default theme layer, so themes may be incremental
  overlays that omit public pages, layouts, components, or assets and inherit
  them from `sforum.default-theme`. Multi-node rollout, signed marketplace
  trust, arbitrary theme dependency installation, and administrator preview
  approval are still future work.
- Production theme switching is zero-downtime (blue-green). The web supervisor
  uses `apps/web/scripts/theme-proxy.mjs` to own the external port (`PORT`,
  default 3000), starts each Nitro candidate on a per-release unix socket via
  `NITRO_UNIX_SOCKET`, waits for health, atomically swaps the proxy upstream,
  and then drains the old child. A candidate that fails health checking leaves
  the old Nitro server available.
- Local `dev-theme-runtime.mjs` consumes the same `current.json` signal and
  reuses the proxy's HTTP/WebSocket forwarding and health checks, but
  intentionally owns one `nuxt dev` process. A selection change clears the
  proxy target, stops and waits for the old process group, then starts the
  latest layer on `PORT=0`. This creates a brief development-only outage;
  parallel Nuxt dev instances would share the build lock, generated output,
  cache, and HMR resources and are therefore unsupported.
- `theme-releases/current.json` is the single runtime theme selection signal and
  is consumed by both production `runtime.mjs` and local `dev-theme-runtime.mjs`.
  Uploaded activation writes `mode: "uploaded"` with absolute `server`
  (built Nitro entry for production) and `layerPath` (Nuxt Layer source for
  local dev). Restoring the built-in default theme (synchronous API path) writes
  `mode: "default"` with no `server`/`layerPath`, so both runtimes fall back to
  the default `.output` / default theme layer. Legacy `{ "server": "..." }`
  files remain compatible.
- Keep plugin `Enable/Disable` separate from theme `Activate`. Do not call
  plugin runtime hooks when activating a theme.
- Backend plugin packages can declare a backend entry and RPC protocol. The
  first supported protocol is `hashicorp-go-plugin` protocol version 1.
- Payment, mail, notification, analytics, and integration packages must not
  override core routes or smuggle vendor-specific behavior into core modules.
  They should use declared plugin routes, explicit host events, and provider
  slots owned by the module whose behavior they extend.

## Permissions

- `extension.manage`: install and manage plugins and themes. Granted to
  `super_admin` by migration and seed catalog. No separate `theme.manage`
  permission exists yet.

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

**First include keys:** `langs`, `settings`, `contributions`, `admin`,
`frontend`, `events`, `routes`, `jobs`.

**No dual source:** the same block must not appear both in the root file and
under `includes` (fail with `extension.manifest_invalid`).

Simple plugins/themes may omit `includes` and keep today's single-file layout.

#### Identity `langs` (directory-per-locale preferred)

Three i18n layers stay separate:

1. Identity — root defaults + `langs` / `includes.langs` → list and install review
2. Settings / contribution labels — `LocalizedText` in settings/contributions
3. Frontend admin UI — `frontend.admin.locales` / Vue locale JSON

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

For `type: theme`, v1 accepts only Nuxt Layer packages. The manifest must
declare a safe, non-empty `frontend.layer` path. The layer directory must exist,
but it may contain only the files the theme wants to override; missing files
fall back to the protected default theme during Nuxt layer resolution. Themes
may declare `settings` and `adminPages` for core-container admin pages, but must
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
- `manifest/settings/*.json` shards (merged by filename)
- `manifest/admin.json`

## Next Steps

- Generic manifest settings support `placeholder`, `recommendedValue`,
  ordered `options`, and `group`. Presentation fields (`label`, `description`,
  `placeholder`, `group`, option labels) accept either a plain string or a
  locale map (`LocalizedText`). Settings GET/PUT/reset resolve copy from the
  request `Accept-Language` and return plain strings only.
- Host dynamic settings page (`view: settings`) is generic chrome: recommended
  defaults banner, form controls, and `SFAdminFormFooter`. Plugins may replace
  the whole form via trusted contribution `admin.extension.settings.page`, or
  inject `admin.extension.settings.header` / `footer`. Slot components are
  filtered to the current extension id.
- `mail.provider` is now implemented end-to-end. The protected `sforum.smtp`
  plugin is the first real provider vertical; core contains no SMTP provider
  code. Extension secret settings are masked/preserved and enabled plugins
  restart after settings changes. SMTP owns multi-locale settings and a custom
  settings page component under `frontend/admin`.

- Multi-file extension manifests are implemented: `LoadPackage`, SMTP reference
  package, `make:plugin --complex`, `extension validate`, and settings/
  contributions directory shards. See
  `decisions/2026-07-12-extension-manifest-split.md`.
- Implement the trusted admin plugin runtime specification before starting the
  River job monitoring module that consumes its first production slots.
- Generalize the current theme artifact builder and supervisor contract into a
  unified Web Release Runtime without regressing existing theme activation.

- Promote Provider Slots into first-class contracts, starting with
  `mail.provider`, `notification.channel`, `payment.provider`,
  `search.provider`, `attachment.storage.provider`,
  `editor.sanitizer.provider`, and `auth.risk.provider`.
- Add preview/approval UI, richer build logs, uninstall cleanup, and explicit
  rollback controls for theme releases.
- Add plugin author documentation for provider-slot based systems such as mail,
  notifications, and payments before building those verticals.
- Add upgrade, rollback, and uninstall operations.
- Add plugin migration execution, dependency checks, SForum version
  compatibility checks, signature/trust metadata, marketplace metadata, local
  debugging, packaging, SDK docs, and example plugins.
