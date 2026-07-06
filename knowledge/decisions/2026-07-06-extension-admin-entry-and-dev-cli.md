# Extension Admin Entry And Developer CLI

## Status

Accepted.

## Context

SForum needs WordPress-like plugin and theme visibility in the admin area, but
the project is Go API + Nuxt SSR rather than an in-process PHP runtime. The
extension system already separates plugin enable/disable from theme activation,
and uploaded themes still cannot be applied until a rebuild, health-check, and
rollback runtime exists.

## Decision

- Every extension manifest must include operator-facing metadata:
  `description`, `url`, and `author`.
- Plugins and themes may declare `adminPages` and `settings`.
- Admin pages are rendered by SForum's core container under
  `/extensions/{id}/pages/*`; v1 supports `about` and `settings` views.
- Enabled plugins and the active theme can inject sidebar navigation. Installed
  but inactive extensions are still reachable from extension list "Manage"
  actions.
- Themes may declare settings and admin pages, but still cannot declare backend
  runtime, plugin routes, hooks, events, jobs, providers, migrations, or
  permissions in v1.
- Extension-owned settings stay in `extension_settings`, are resolved over
  manifest defaults, and can be reset to defaults.
- The Laravel-artisan-style developer console lives in Go under
  `apps/api/cmd/sforum` so it can reuse backend manifest validation. The first
  commands are `make:plugin` and `make:theme`, backed by Cobra and Huh.

## Consequences

- SForum gets visible in-admin extension entry points without loading arbitrary
  third-party Nuxt components.
- Extension authors can scaffold packages consistently under
  `extensions/dev/{plugins,themes}/{id}` or `extensions/builtin` via `--builtin`.
- Future richer extension UIs can build on this namespace without allowing
  extension packages to shadow core admin routes.
