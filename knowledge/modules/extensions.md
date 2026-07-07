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

The extension foundation is implemented with plugin/theme lifecycle separation
and plugin runtime v1.

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
  - `GET /api/v1/admin/extensions/event-definitions`
  - `GET /api/v1/admin/extensions/event-deliveries`
  - `ALL /api/v1/extensions/:extensionId/*` proxies declared enabled plugin
    routes after host-side route matching and access checks.
- The admin UI has an independent "Extensions" sidebar folder registered
  through the low-code admin module registry and protected by
  `extension.manage`. Its first submenu set is Overview, Plugins, Themes,
  Settings, and Event Log. Enabled plugins and the active theme can inject
  manifest-declared core-container admin pages under the fixed
  `/extensions/{id}/pages/*` admin namespace; installed extensions also have a
  "Manage" entry from plugin/theme list rows.
- Theme rows show `enabled` as "current theme" rather than "enabled".
  Uploaded Nuxt Layer themes can now be activated through the self-hosted theme
  runtime. Activation queues a River job, builds an isolated Nuxt/Nitro artifact,
  health-checks a preview server, writes the active release file, and lets the
  web runtime restart onto the selected artifact. Failed builds keep the
  previous active theme running.
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
  Slots. `topic.before_create` is the first synchronous filter event; lifecycle,
  user registration, topic/comment creation, and attachment upload are observe
  events with delivery tracking.
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
- Theme switching is zero-downtime (blue-green). The web supervisor
  (`apps/web/scripts/theme-proxy.mjs`, shared by production `runtime.mjs` and
  local `dev-theme-runtime.mjs`) runs a built-in reverse proxy that owns the
  external port (PORT, default 3000). Child processes listen on a separate
  address: production uses a per-release unix socket via
  `NITRO_UNIX_SOCKET` (Nitro `node-server` does not honor `PORT=0`); local dev
  uses `PORT=0` and the supervisor parses the listening port from nuxt dev
  stdout. On `current.json` change the supervisor spawns the new child, waits
  for its health check to pass, then atomically swaps the proxy upstream and
  drains the old child, so traffic is never interrupted. A candidate that fails
  health check leaves the old child serving.
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

The manifest file is `sforum.extension.json`.

Required identity fields: `id`, `name`, `description`, `url`, `author`,
`version`, `type`, and `sforumVersion`.

Capability fields: `permissions`, `settings`, `migrations`, `backend`,
`frontend`, `adminPages`, `routes`, `hooks`, `events`, `jobs`, and `providers`.

The v2 admin declaration is an `admin` object. Existing top-level
`adminPages` should be compatibility-mapped during migration.

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
`routes`, `hooks`, `events`, `jobs`, `providers`, `migrations`, or
`permissions`. Invalid theme manifests use the existing 422 envelope with reason
`extension.manifest_invalid`.

## Developer Console

`apps/api/cmd/sforum` provides a Laravel-artisan-style developer console.

- `./scripts/sforum.sh make:plugin`
- `./scripts/sforum.sh make:theme`

Both commands support interactive Huh forms and `--no-interaction` flag-driven
generation. Default output is `extensions/dev/{plugins,themes}/{id}`; `--builtin`
targets `extensions/builtin/{plugins,themes}/{id}`.

## Next Steps

- Make plugins truly usable with a real `mail.provider` plugin slice: manifest
  risk review, subprocess startup, health checks, route proxying, settings,
  logs, event delivery visibility, disable cleanup, and failed-enable rollback.
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
