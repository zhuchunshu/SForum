# Extensions Module

## Purpose

Owns installable plugins and themes for SForum. Plugins are multi-enable
runtime extensions. Themes are Nuxt Layer packages with exactly one active
applied theme.

SForum core should stay framework-focused. Product verticals that vary by
deployment or vendor, including payment gateways, outbound mail delivery,
notification channels, analytics, and external integrations, should be
implemented as plugins by default. Core may add explicit events, filters,
provider slots, permission gates, admin selection/reset flows, SDK helpers, and
protected built-in plugins when those make extension development practical.

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
  `extension_events`, and `extension_event_deliveries`.
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
- Extension verify and plugin enable operations require the active package path
  and installed `sforum.extension.json` to still exist before changing runtime
  state. Theme verify reports missing packages, missing installed manifests, and
  missing Nuxt Layer directories as `extension.build_failed`; plugins keep using
  `extension.preflight_failed` for preflight failures.
- Admin API routes:
  - `GET /api/v1/admin/extensions`
  - `POST /api/v1/admin/extensions`
  - `POST /api/v1/admin/extensions/:id/enable` for plugins
  - `POST /api/v1/admin/extensions/:id/disable` for plugins
  - `POST /api/v1/admin/extensions/:id/verify` for plugin preflight or theme
    Nuxt Layer verification without applying it
  - `POST /api/v1/admin/extensions/:id/activate` for themes
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
  Uploaded themes can be verified but cannot be activated in v1; attempts
  return `extension.theme_runtime_unavailable`.

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
- Theme packages can declare a Nuxt Layer path, but v1 statically applies only
  `extensions/builtin/themes/sforum-default/layer` from the web Nuxt config.
  Uploaded theme activation must wait for a Nuxt rebuild, health-check, and
  rollback runtime.
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

For `type: theme`, v1 accepts only Nuxt Layer packages. The manifest must
declare a safe, non-empty `frontend.layer` path. Themes may declare `settings`
and `adminPages` for core-container admin pages, but must not declare backend
runtime or plugin execution capabilities: `backend`, `routes`, `hooks`,
`events`, `jobs`, `providers`, `migrations`, or `permissions`. Invalid theme
manifests use the existing 422 envelope with reason `extension.manifest_invalid`.

## Developer Console

`apps/api/cmd/sforum` provides a Laravel-artisan-style developer console.

- `go run ./cmd/sforum make:plugin`
- `go run ./cmd/sforum make:theme`

Both commands support interactive Huh forms and `--no-interaction` flag-driven
generation. Default output is `extensions/dev/{plugins,themes}/{id}`; `--builtin`
targets `extensions/builtin/{plugins,themes}/{id}`.

## Next Steps

- Add a real theme activation worker that writes active Nuxt layer state,
  triggers web rebuild, runs a health check, and rolls back on failure. Only
  then should uploaded themes be activatable.
- Add plugin author documentation for provider-slot based systems such as mail,
  notifications, and payments before building those verticals.
- Add upgrade, rollback, and uninstall operations.
- Add signature/trust metadata if SForum later ships an extension marketplace.
