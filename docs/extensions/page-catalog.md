# Page Registry Catalog

Status: implemented
Date: 2026-07-13

The Go catalog in `apps/api/app/Support/Pages` is authoritative. Themes and
plugins may contribute public view pages through `theme.json`; they cannot
override API, authentication authority, admin, or internal runtime routes.

## Runtime options

| Option | Default | Purpose |
| --- | --- | --- |
| `pages.registry_enabled` | `true` | Resolve stable page ids and added paths |
| `themes.runtime_l0_enabled` | `true` | Active theme skin/assets |
| `themes.runtime_l1_enabled` | `true` | Runtime templates and add/replace |

There is no Layer activation option or frontend release runtime.

## Core page ids

| Page id | Default path | Access |
| --- | --- | --- |
| `forum.home` | `/` | public |
| `forum.category.index` | `/categories` | public |
| `forum.category.show` | `/c/:categorySlug` | public |
| `forum.tag.index` | `/tags` | public/feature-gated |
| `forum.tag.show` | `/tags/:tagSlug` | public/feature-gated |
| `forum.topic.show` | `/t/:path(.*)` | public |
| `forum.topic.create` | `/topics/new` | login |
| `forum.topic.reply` | `/topics/reply` | login |
| `forum.profile.show` | `/u/:username` | public/feature-gated |
| `forum.settings.profile` | `/settings/profile` | login |
| `forum.settings.security` | `/settings/security` | login |
| `auth.login` | `/login` | guest |
| `auth.register` | `/register` | guest/feature-gated |
| `auth.forgot_password` | `/forgot-password` | public |
| `auth.reset_password` | `/reset-password` | public |
| `site.terms` | `/terms` | public |
| `site.privacy` | `/privacy` | public |
| `site.guidelines` | `/guidelines` | public |
| `system.forbidden` | virtual | public |
| `system.not_found` | virtual | public |
| `system.rate_limited` | virtual | public |
| `system.server_error` | virtual | public |

Virtual system error pages have no routable public path. The Host selects them
only after it has already normalized a browser error status: 403, 404, 429, or
500/502/503/504. They keep the original HTTP status, `Cache-Control: no-store`,
and `noindex,nofollow`; API JSON error envelopes and 401 login redirects are
unchanged.

Admin pages, moderation workbenches, notifications, and component previews are
host-owned and outside public Page Registry replacement. System error pages are
theme-replaceable presentation surfaces only: plugins cannot replace them, and
their templates cannot declare public L2 widgets.

## Reserved paths

Added paths are rejected when they collide with the configured admin prefix,
`/admin`, `/api`, `/_nuxt`, `/__nuxt`, `/__sforum`, `/health`, attachment URLs,
or authentication/provider callback paths. Checks apply after stripping the
optional locale prefix.

## Lifecycle

- Plugin enable registers its approved page package; disable/uninstall clears it.
- Theme activation atomically replaces old and new theme contributions.
- Plugin replacement of a core page requires explicit `super_admin` approval.
- Plugin replacement of `system.*` error pages is rejected; the selected theme
  owns only L0/L1 presentation for those virtual surfaces.
- Access policy is fail-closed (`public`, `login`, `guest`, `moderation`, or
  permission key).
- SSR data comes through the host Loader Gateway; extension templates receive
  allowlisted data and never raw session cookies.
- `page_provider_bindings.contract_version` prevents stale approval reuse.

Public theme activation is independent from Settings Component trust. Both
development and production run the same host Nuxt application; no activation
path builds or restarts it.
