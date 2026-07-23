# Runtime Themes

Installable themes are buildless Page Registry packages. Activating a theme
does not compile Nuxt, start a worker job, restart Nitro, or change the admin
component lifecycle.

## Package layout

```text
my-theme/
├── sforum.extension.json
├── theme.json
├── assets/
│   └── theme.css
└── templates/
    └── home.html
```

The manifest uses `type: "theme"` and may declare Settings Document/admin pages.
It must not declare plugin backend, routes, hooks, events, jobs, providers,
contributions, migrations, permissions, capabilities, or frontend Layer fields.

`theme.json` owns runtime presentation:

- L0 skin CSS and package assets;
- L1 add/replace page contributions using stable Page Registry ids;
- host template islands and allowlisted data bindings.

Themes cannot replace core API, authentication authority, admin routes, or
security endpoints. The host owns route matching, access checks, loader data,
sanitization, and template islands.

## System error pages

Themes may provide L1 templates for these Host-selected virtual surfaces:

| Page id | Contract | Status family |
| --- | --- | --- |
| `system.forbidden` | `sforum.page.forbidden@1` | 403 |
| `system.not_found` | `sforum.page.not_found@1` | 404 |
| `system.rate_limited` | `sforum.page.rate_limited@1` | 429 |
| `system.server_error` | `sforum.page.server_error@1` | 500, 502, 503, 504 |

These pages do not have public paths. The Host selects the page id after the
error status is known, preserves the original status, applies no-store and
noindex/nofollow policy, and supplies localized public copy plus home/back/retry
actions through reviewed islands.

Allowed system-error islands are `sf-navbar`, `sf-footer`,
`sf-error-details`, `sf-error-actions`, `sf-error-recovery`,
`sf-error-sidebar`, and `sf-error-rail`. Public L2 widgets such as
`sf-extension-widget` are rejected for system error templates. Plugins cannot
replace `system.*` pages.

## Activation

`POST /api/v1/admin/extensions/{id}/activate` verifies the installed package,
replaces the old theme's Page Registry bindings atomically, switches active skin
assets, persists the active theme, and returns the resulting `Extension`.

The default theme follows the same runtime contract. Failed validation leaves
the previous active theme and registry bindings intact. Public settings are read
from `GET /api/v1/site/active-theme/settings`; secrets are never returned.

## Settings

Theme settings use the same Settings Document and
`SFExtensionSettingsRenderer` as plugins. Tabs, groups, columns, callouts,
recommended defaults, and secret semantics work in development and production
without a theme-specific Vue page. Complex admin settings may use the same
prebuilt digest-trusted component contract, but public Page Registry activation
and admin component trust remain separate lifecycles.

## Validation

```bash
cd apps/api
go run ./cmd/sforum extension validate /path/to/my-theme
go run ./cmd/sforum extension test /path/to/my-theme
```

Use `make:theme` for a Schema settings scaffold. The generated package contains
no Nuxt Layer or runtime build instructions.
