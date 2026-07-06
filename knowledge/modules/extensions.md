# Extensions Module

## Purpose

Owns installable plugins and themes for SForum. Plugins are multi-enable
runtime extensions. Themes are Nuxt Layer packages with exactly one active
applied theme.

## Current Status

The extension foundation is implemented with plugin/theme lifecycle separation
and plugin runtime v1.

- `extension.manage` is the permission for uploading, verifying, enabling
  plugins, activating the protected default theme, and inspecting extensions.
  It is seeded for `super_admin`.
- Extension packages are uploaded as ZIP archives through the admin API and
  must include a root `sforum.extension.json` manifest.
- Backend model code validates manifest identity, type, version, compatibility
  field, backend entry paths, frontend layer paths, migrations, and unsafe ZIP
  paths before writing files.
- Installed packages are stored under `EXTENSION_ROOT`, not in the public
  attachment system.
- Database tables: `extensions`, `extension_versions`, `extension_settings`,
  and `extension_events`.
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
  - `ALL /api/v1/extensions/:extensionId/*` proxies declared enabled plugin
    routes after host-side route matching and access checks.
- The admin UI has an independent "Extensions" sidebar folder registered
  through the low-code admin module registry and protected by
  `extension.manage`. Its first submenu set is Overview, Plugins, Themes,
  Settings, and Event Log.
- Theme rows show `enabled` as "current theme" rather than "enabled".
  Uploaded themes can be verified but cannot be activated in v1; attempts
  return `extension.theme_runtime_unavailable`.

## Boundaries

- Extension settings stay in `extension_settings`; do not put extension-owned
  configuration into `web_options`.
- Extension archive files stay under `EXTENSION_ROOT`; do not expose them
  through public attachment URLs.
- Plugin runtime v1 uses HashiCorp go-plugin subprocess handshakes, starts
  enabled plugin backends on API startup, proxies declared plugin routes under
  `/api/v1/extensions/:extensionId/*`, emits lifecycle hooks, and exposes a
  provider slot registry with built-in defaults.
- Theme packages can declare a Nuxt Layer path, but v1 statically applies only
  `extensions/builtin/themes/sforum-default/layer` from the web Nuxt config.
  Uploaded theme activation must wait for a Nuxt rebuild, health-check, and
  rollback runtime.
- Keep plugin `Enable/Disable` separate from theme `Activate`. Do not call
  plugin runtime hooks when activating a theme.
- Backend plugin packages can declare a backend entry and RPC protocol. The
  first supported protocol is `hashicorp-go-plugin` protocol version 1.

## Permissions

- `extension.manage`: install and manage plugins and themes. Granted to
  `super_admin` by migration and seed catalog. No separate `theme.manage`
  permission exists yet.

Frontend visibility mirrors this permission for navigation only. API policy
checks remain authoritative.

## Manifest

The manifest file is `sforum.extension.json`.

Important fields: `id`, `name`, `version`, `type`, `sforumVersion`,
`permissions`, `settings`, `migrations`, `backend`, `frontend`, `adminPages`,
`routes`, `hooks`, `jobs`, and `providers`.

For `type: theme`, v1 accepts only Nuxt Layer packages. The manifest must
declare a safe, non-empty `frontend.layer` path and must not declare plugin or
admin capabilities: `backend`, `routes`, `hooks`, `jobs`, `providers`,
`migrations`, `adminPages`, `settings`, or `permissions`. Invalid theme
manifests use the existing 422 envelope with reason `extension.manifest_invalid`.

## Next Steps

- Add a real theme activation worker that writes active Nuxt layer state,
  triggers web rebuild, runs a health check, and rolls back on failure. Only
  then should uploaded themes be activatable.
- Implement extension settings CRUD from manifest-declared settings.
- Add upgrade, rollback, and uninstall operations.
- Add signature/trust metadata if SForum later ships an extension marketplace.
