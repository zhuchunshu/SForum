# Runtime Themes (L0 / L1 / L2)

SForum public themes **do not rebuild Nuxt** and **do not restart Nitro**.

## Package layout

```text
my-theme/
  sforum.extension.json   # type: theme; no frontend.admin for ordinary themes
  theme.json              # pages[] + skin (unified page contract for themes & plugins)
  assets/theme.css        # L0 skin
  assets/tokens.css       # optional tokens
  templates/home.html     # L1 HTML + <sf-*> host islands
```

### theme.json

```json
{
  "pages": [
    {
      "id": "my.theme.home",
      "action": "replace",
      "target": "forum.home",
      "template": "templates/home.html",
      "contract": "sforum.page.home@1"
    },
    {
      "id": "my.plugin.docs",
      "action": "add",
      "path": "/demo-docs/:slug",
      "template": "templates/docs.html",
      "access": "public"
    }
  ],
  "skin": {
    "css": ["assets/theme.css"],
    "tokens": "assets/tokens.css"
  }
}
```

Plugins that add/replace **view pages** use the same `theme.json` `pages[]`
contract (not a separate plugin-only page format).

## Activation

Admin **Activate theme** → API `ActivateTheme` (sync, atomic):

1. **Preflight** theme.json, templates (bluemonday), CSS, routes, contributions
2. DB transaction marks theme active
3. Page Registry registers **candidates only** (no silent replace approval)
4. Host injects L0 CSS from `/api/v1/site/active-theme/skin` (digest query `?v=`)
5. **No** Web Release, **no** `extension.theme_activate` job, **no** Nitro switch

Any preflight/registry failure keeps the previous theme fully usable.

### Core page replace approval

- Theme/plugin activation only **registers candidates**.
- Replacing a core page requires **super_admin** `POST /admin/pages/:pageId/approve`
  bound to extension id, version, package digest, contribution id, contract.
- `extension.theme.manage` alone cannot approve core replaces.
- Restore core: `POST /admin/pages/:pageId/restore-core` (super_admin).
- All approve/restore actions write `audit_events`.

Activation impact preview: `GET /admin/pages/activate-preview/:extensionId`.

## L1 templates

Security boundary is **bluemonday allowlist** (not regex):

- Allowed layout tags + registered host islands only
- Forbidden: script, iframe, object, embed, style, SVG, MathML, form, base, meta,
  inline handlers, javascript/data URL schemes
- Size / depth / attribute limits enforced
- Host islands: `sf-home-page`, `sf-navbar`, `sf-footer`, `sf-home-navigation`
- **Not allowed:** `sf-extension-widget` (L2 disabled)

User content never renders as raw template HTML; only host SF islands load data.

Constrained pages (login, register, password, settings, topic create, …) always
execute mutations via **core Vue components** even if a replace candidate exists.

## Plugin page data (loader)

Optional `data.source=plugin` + `data.route` is loaded **server-side** during
resolve (not browser → arbitrary URL):

- Relative plugin route only (no absolute URL)
- Host loopback RouteGateway; minimal actor id header; **no session cookie forward**
- Timeout, max body size, JSON content-type, sensitive-key rejection
- Failure → host error state / core fallback

## L2 widgets — **disabled**

`SFExtensionWidget` does **not** dynamic-import remote or package JS.

L2 remains **unimplemented** until package-digest trust, integrity, CSP, and
admin grant lifecycle are complete. Do not ship half-executable widget loaders.

## Theme assets

`GET /api/v1/site/theme-assets/:extensionId/*`

- Only the **active** theme package
- Optional `?v=<packageDigest>` for immutable cache
- No SVG/JS/HTML; nosniff + CSP; symlink escape rejected

## Dynamic add routes

Manifest `action: add` paths are resolved by API
`GET /pages/resolve-path?path=…` and rendered by Nuxt catch-all
`pages/[...sfRegistryPage].vue` (real paths, not forced under `/x/*`).
Reserved prefixes and core path collisions are rejected.
Access `login` / moderation returns 401/403 from the API.

## Web Release

Only for legacy **trusted Vue admin plugin frontends**. Author-prebuilt Admin
Micro-frontend API v1 settings components load by exact digest and do not enter
Web Release composition.

Ordinary themes must **not** declare `frontend.admin` / `ThemeSettingsPage.vue`.
Theme settings use the host **schema-driven** extension settings page from
`manifest/settings.json`.

## Scaffold

```sh
cd apps/api && go run ./cmd/sforum make:theme example.theme
```

See also `docs/extensions/page-catalog.md` and fixture
`extensions/fixtures/plugins/page-registry-demo`.
