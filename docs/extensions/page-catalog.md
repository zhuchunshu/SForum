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
| `forum.topic.edit` | `/topics/:topicId/edit` | login |
| `forum.profile.show` | `/u/:username` | public/feature-gated |
| `forum.settings.profile` | `/settings/profile` | login |
| `forum.settings.security` | `/settings/security` | login |
| `forum.settings.notifications` | `/settings/notifications` | login |
| `forum.notifications` | `/notifications` | login |
| `forum.notification.show` | `/notifications/:notificationId` | login |
| `moderation.review` | `/moderation` | moderation |
| `auth.login` | `/login` | guest |
| `auth.register` | `/register` | guest/feature-gated |
| `auth.forgot_password` | `/forgot-password` | public |
| `auth.reset_password` | `/reset-password` | public |
| `site.terms` | `/terms` | public |
| `site.privacy` | `/privacy` | public |
| `site.guidelines` | `/guidelines` | public |
| `site.outbound` | `/outbound` | public (host-owned link interstitial, not replaceable) |
| `system.forbidden` | virtual | public |
| `system.not_found` | virtual | public |
| `system.rate_limited` | virtual | public |
| `system.server_error` | virtual | public |
| `dev.components` | `/components` | public (dev gallery, not replaceable) |

`apps/api/app/Support/Pages/catalog_coverage_test.go` enforces that this
catalog and `apps/web/app/pages/` stay in sync in both directions (admin pages,
`[...sfRegistryPage].vue`, and `x/[...path].vue` are exempt catch-all hosts).

Virtual system error pages have no routable public path. The Host selects them
only after it has already normalized a browser error status: 403, 404, 429, or
500/502/503/504. They keep the original HTTP status, `Cache-Control: no-store`,
and `noindex,nofollow`; API JSON error envelopes and 401 login redirects are
unchanged.

Admin pages, moderation workbenches, and component previews are
host-owned and outside public Page Registry replacement. System error pages are
theme-replaceable presentation surfaces only: plugins cannot replace them, and
their templates cannot declare public L2 widgets.

## Standard page regions (`forum.page.regions`)

`apps/api/app/Support/RegionCatalog` is the single hand-edited source for page
regions. `pageRegionMatrix` whitelists which regions each public page exposes:

| Region | Meaning |
| --- | --- |
| `content_before` | Top of the page's main column |
| `content_after` | Bottom of the page's main column |
| `sidebar` | Right rail (pages that have one) |
| `sidebar_top` / `sidebar_bottom` | Above / below `sidebar` in the right rail |
| `list_before` / `list_after` | Above / below the page's primary list |
| `detail_before` / `detail_after` | Inside a detail body, above / below the main content |
| `comments_before` / `comments_after` | Above / below the comment list |
| `profile_header_after` | Directly under the profile header |

A region is only declared for a page that actually renders it. Every
page×region pair must have a real `<SFRegionOutlet>` in the page's host
component, and `apps/web/tests/framework/pageRegions.test.ts` fails when the
matrix and the render sites disagree in either direction. Two consequences
worth knowing: `forum.topic.reply` is intentionally absent (its component is a
redirect placeholder with no slots), and `hero`/`composer_after` were rejected
because they would have sat next to `content_before`/`content_after` without
adding a distinct slot.

`apps/web/app/config/pageRegionCatalog.gen.ts` is generated from that Go file
by `scripts/v3-catalog/generate.mjs` — do not hand-edit the web whitelist.

Plugins place content through the `forum.page.regions` contribution point
(payload type `regionPlacement`):

- `hostLink` — host-rendered link card, host-relative paths only.
- `extensionRoute` — action card executed via the `/extensions/{id}` proxy.
- `l2Widget` — reference to a public L2 component declared in the same
  manifest (`action: add`, no permission gate). The placement grants no
  execution rights: mounting still requires the public L2 runtime's descriptor
  admission (super_admin trust grant + SRI + per-page CSP), and untrusted
  widgets fail closed while descriptor cards keep rendering.

The frontend fetches `GET /site/page-regions?page=<id>` during SSR from
`SFPageOutletResolver`, shares the payload via `usePageRegions`, and renders
`SFRegionOutlet` instances placed in the native page components (which also
serve theme L1 templates as body islands, so both render paths get regions).
An empty region renders no DOM at all. CSP is aggregated exactly once per
response — by `SFThemeTemplate` on the template path, by the resolver
otherwise. `enabledBySetting` gating, ordering, and safe mode follow the shared
EffectiveContributions pipeline. Reference fixture:
`extensions/fixtures/plugins/sforum-region-demo`.

## Adding a core page

1. Add the `PageDefinition` to `coreCatalog` in
   `apps/api/app/Support/Pages/catalog.go` (id, path, access, contract,
   `CoreComponent: "pages/<path>"`).
2. Create the Nuxt shell `apps/web/app/pages/<path>.vue` wrapping the native
   component: `<SFPageOutlet page="<id>"><SFXxxPage /></SFPageOutlet>`.
3. Regenerate the component/route catalogs:
   `node scripts/v3-catalog/generate.mjs` (never edit `*_gen.go` by hand;
   CI uses `--check`).
4. If themes may replace the page, register the body island in
   `SFThemeTemplate.vue` (`islandComponents` + `legacyIslandBindings`) and add
   the island mapping expected by `RequiredThemeBodyIslandTag`.
5. If the page should expose standard regions, add the page×region entries to
   `pageRegionMatrix` in `apps/api/app/Support/RegionCatalog/catalog.go`, run
   `node scripts/v3-catalog/generate.mjs` (the web whitelist is generated from
   that Go file), and place a `<SFRegionOutlet>` for every declared region in
   the native component. `apps/web/tests/framework/pageRegions.test.ts` fails
   until the matrix and the render sites agree, so a region cannot be declared
   without a slot that actually renders it.
6. `catalog_coverage_test.go` will fail until catalog and page shell agree.

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
