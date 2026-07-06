# 2026-07-06 Extension Platform v2 Direction

## Status

Accepted.

## Context

SForum already has an extension foundation: ZIP uploads, manifest validation,
plugin enable/disable lifecycle, HashiCorp go-plugin subprocess startup,
health/preflight checks, declared plugin route proxying, event delivery
records, extension settings, and core-container admin pages.

The next risk is product drift. A WordPress-like operator experience is useful,
but copying WordPress' PHP include execution model would conflict with SForum's
Go API plus Nuxt SSR architecture and with the plugin-first core boundary.

Uploaded themes are also intentionally not activatable yet. Nuxt currently
statically extends the protected built-in `sforum.default-theme` layer, so
theme activation requires a build, health-check, preview, switch, and rollback
pipeline rather than runtime inclusion.

## Decision

- Define Extension Platform v2 as a complete product loop, not a single API:
  upload, inspect, review permissions and risks, enable or activate, configure,
  observe logs/errors/deliveries, disable or roll back, and restore safe
  defaults.
- Keep the operator experience WordPress-like while keeping the runtime modern
  and controlled. Plugins extend SForum only through manifests, subprocess RPC,
  provider slots, events/filters, plugin route namespaces, settings, and
  host-owned admin pages.
- Do not allow arbitrary core route override, monkey-patching, raw session
  cookie authority, or bypassing API policy checks.
- Treat the new manifest `admin` shape as the target direction while preserving
  compatibility with top-level `adminPages` during migration:
  `admin.entry` selects the `Manage` destination and `admin.pages[]` declares
  pages with `path`, `label`, `view`, `menu`, `icon`, and `order`.
- Make sidebar injection explicit. Extension admin pages must default to
  `menu: false`; only pages marked `menu: true` may appear in admin navigation.
- Keep `Manage` inside the admin shell. It may resolve to `admin.entry`, a
  settings page, the first declared admin page, or a generated system detail
  page, but it must not jump directly to an external URL.
- Let disabled plugins keep basic management pages available while runtime
  capabilities, routes, subprocesses, and runtime menu entries are off. Let
  inactive themes expose settings/management pages without taking over public
  UI or injecting sidebar entries by default.
- Build admin page capability in layers: host-generated pages first, manifest
  content pages second, and extension-owned frontend assets only later under
  CSP, permission, route, and resource validation controls.
- Use a real `mail.provider` plugin as the first full vertical slice for v2.
  It should exercise provider slots, secrets, settings, no-op fallback, error
  reporting, provider selection UI, SDK examples, and documentation.
- Promote Provider Slots to core extension contracts, starting with
  `mail.provider`, `notification.channel`, `payment.provider`,
  `search.provider`, `attachment.storage.provider`,
  `editor.sanitizer.provider`, and `auth.risk.provider`.
- Keep vendor behavior in plugins. Core may own provider-neutral lifecycle and
  shared records when needed, such as payment intents, transactions, webhook
  idempotency, entitlements, mail message contracts, or notification
  preferences.
- Implement uploaded theme activation only through a build pipeline:
  validate, build in a temporary location, health-check, preview, administrator
  confirmation, atomic switch, and rollback.
- Complete extension lifecycle later with upgrade, rollback, uninstall,
  migrations, dependency checks, data retention policy, version compatibility,
  signatures, trusted sources, marketplace metadata, SDK, local debugging,
  packaging, and example plugins.

## Consequences

- v2 work should start with plugin usability and a mail provider slice before
  attempting marketplace or arbitrary frontend extension UIs.
- Existing `adminPages` behavior should be treated as v1 compatibility; new
  work should converge on `admin.entry` and explicit `admin.pages[].menu`.
- Theme activation remains blocked until a deployment-style Nuxt build and
  rollback runtime exists.
- Product verticals such as SMTP, Stripe, Alipay, Feishu, Meilisearch
  replacement, storage providers, and risk scoring should be implemented as
  plugins behind host-owned contracts.
- Future implementation plans should update OpenAPI, manifest validation, admin
  UI, permission notes, and tests together when changing extension behavior.
