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

## Template helpers

Theme templates run in a restricted `html/template` environment. Only these
functions are available; anything else fails template inspection.

| Helper | Output | Notes |
| --- | --- | --- |
| `{{ siteName }}` | string | Site display name |
| `{{ asset "id" }}` | string | Declared package asset URL |
| `{{ route "id" }}` | string | Host route for a stable page/route id |
| `{{ i18n .Locale "key" }}` | string | Host translation for the active locale |
| `{{ safeHTML … }}` | markup | Only accepts a Host-reviewed `SafeHTML` value; cannot mint islands |
| `{{ icon "i-lucide-inbox" }}` | markup | Host icon placeholder, see below |

Standard `and`/`or`/`not`, comparison, `len`, and `index` are also allowed.

### Host icons in templates

```html
<h2>{{ icon "i-lucide-inbox" }} Inbox</h2>
```

The helper emits a placeholder element, not SVG:

```html
<span class="sf-host-icon" data-sf-icon="i-lucide-inbox" aria-hidden="true"></span>
```

The Host client runtime fills it from the same `/_sforum/icons` surface the
plugin SDK uses, so themes never bundle an icon set and never hand-write SVG
(inline `<svg>` in a template is rejected by inspection anyway).

- Names use `i-(lucide|tabler)-<kebab-name>`, the same grammar as contribution
  `icon` fields and the plugin SDK.
- An unsupported name emits an HTML comment
  (`<!-- sforum.icon: unsupported name "…" -->`) instead of failing the render,
  so a typo is discoverable in the source without turning a page into a 500.
- Icons are decorative and `aria-hidden`. With JavaScript disabled the
  placeholder stays empty; content, accessibility, and crawler output are
  unaffected.

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

### What activation checks, and what it does not

Activation runs three preflights on the exact artifact, in this order:

1. **Page Registry package preflight** (`PreflightThemePackage`) — manifest,
   `theme.json`, L0 CSS, every `theme.json.pages[].template`, the optional
   `shell` template, and add-path signature conflicts. The two states
   "fresh install" and "replacing the current active theme" are both checked, so
   a theme may reuse the same paths as the theme it replaces.
2. **Component Registry transition preflight** — the post-switch component graph
   is built and compiled without publishing it. Any `components[]` declaration
   is compiled here, which is why every component `propsSchema`/`resultSchema`
   must be a **package-local schema with its own exact `packageFiles[]`
   declaration**.
3. **Asset Registry transition preflight** — the exact asset publication graph
   after the switch, validated with the same rules as the eventual CAS.

Two consequences authors hit early:

- **Page coverage is not a requirement.** A theme may replace one page or all
  core pages; pages it does not declare keep rendering through the Host core
  fallback. There is no "34 templates or nothing" rule.
- **A theme's `components[]` cannot point at a Host core page schema.** A
  component with `ssrTemplate` must repeat the template's `viewModelSchema`
  verbatim, and for a core page that is a Host-owned reference such as
  `sforum.page.home@1`, which no theme package can declare. The theme L1 surface
  is therefore `theme.json.pages[].template` + manifest `templates[]` +
  `packageFiles[]`; declare a component only for a package-local L2 component
  with its own schema. `make:theme` generates no `components[]` for this reason.

`extension validate` and `extension test` load the package through the same
manifest validation the Host uses, so an unactivatable declaration shape is
reported at CLI time instead of only at activation time.
