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
