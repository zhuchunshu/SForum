# Extensions Module

## Purpose

Owns installable plugins and theme templates for SForum. The first foundation
aims for a WordPress-like operator experience while keeping execution behind
SForum-controlled extension points.

## Current Status

The extension foundation is implemented.

- `extension.manage` is the new permission for uploading, enabling, disabling,
  and inspecting extensions. It is seeded for `super_admin`.
- Extension packages are uploaded as ZIP archives through the admin API and
  must include a root `sforum.extension.json` manifest.
- Backend model code validates manifest identity, type, version, compatibility
  field, backend entry paths, frontend layer paths, migrations, and unsafe ZIP
  paths before writing files.
- Installed packages are stored under `EXTENSION_ROOT`, not in the public
  attachment system.
- Database tables: `extensions`, `extension_versions`, `extension_settings`,
  and `extension_events`.
- Admin API routes:
  - `GET /api/v1/admin/extensions`
  - `POST /api/v1/admin/extensions`
  - `POST /api/v1/admin/extensions/:id/enable`
  - `POST /api/v1/admin/extensions/:id/disable`
  - `GET /api/v1/admin/extensions/:id/events`
  - `ALL /api/v1/extensions/:extensionId/*` currently returns
    `extension.route_unavailable` until runtime proxying is implemented.
- The admin UI has an independent "Extensions" sidebar folder registered
  through the low-code admin module registry and protected by
  `extension.manage`. Its first submenu set is Overview, Plugins, Themes,
  Settings, and Event Log.

## Boundaries

- Extension settings stay in `extension_settings`; do not put extension-owned
  configuration into `web_options`.
- Extension archive files stay under `EXTENSION_ROOT`; do not expose them
  through public attachment URLs.
- The first runtime foundation performs local preflight checks for backend
  entries and Nuxt layer paths. It does not yet supervise long-running plugin
  child processes or proxy plugin HTTP routes.
- Theme packages can declare a Nuxt Layer path. Enabling a theme runs the
  theme builder boundary before the database status changes. The current
  default builder verifies the layer directory; a later deployment/build
  supervisor should replace it to run the actual Nuxt rebuild and health check.
- Backend plugin packages can declare a backend entry and RPC protocol. The
  current default preflight verifies the entry exists; a later RPC supervisor
  should replace it to run a HashiCorp go-plugin compatible handshake.

## Permissions

- `extension.manage`: install and manage plugins and theme templates. Granted
  to `super_admin` by migration and seed catalog.

Frontend visibility mirrors this permission for navigation only. API policy
checks remain authoritative.

## Manifest

The manifest file is `sforum.extension.json`.

Important fields: `id`, `name`, `version`, `type`, `sforumVersion`,
`permissions`, `settings`, `migrations`, `backend`, `frontend`, `adminPages`,
`routes`, `hooks`, and `jobs`.

## Next Steps

- Add a real plugin runtime supervisor that starts child processes, performs
  RPC handshakes, and proxies `/api/v1/extensions/:extensionId/*`.
- Add a real theme activation worker that writes active Nuxt layer state,
  triggers web rebuild, runs a health check, and rolls back on failure.
- Implement extension settings CRUD from manifest-declared settings.
- Add upgrade, rollback, and uninstall operations.
- Add signature/trust metadata if SForum later ships an extension marketplace.
