# Runtime Themes (L0 / L1 / L2)

SForum public themes **do not rebuild Nuxt** and **do not restart Nitro**.

## Package layout

```text
my-theme/
  sforum.extension.json   # type: theme; frontend.layer optional/legacy only
  theme.json              # pages[] + skin
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
    }
  ],
  "skin": {
    "css": ["assets/theme.css"],
    "tokens": "assets/tokens.css"
  }
}
```

## Activation

Admin **Activate theme** → API `ActivateTheme` (sync):

1. DB marks theme active
2. Page Registry registers contributions / binds replaces
3. Host injects L0 CSS from `/api/v1/site/active-theme/skin`
4. **No** Web Release, **no** `extension.theme_activate` job, **no** Nitro switch

## L1 templates

- Allowlisted HTML + `{{var}}` + `<sf-*>` islands
- Rejected: `<script>`, iframes, inline handlers, `javascript:` URLs
- Host islands: `sf-home-page`, `sf-navbar`, `sf-footer`, `sf-extension-widget`

## Plugin page data

Optional `data.source=plugin` + `data.route` loads through
`/api/v1/extensions/:id/...` (permissioned proxy).

## L2 widgets

Use `<SFExtensionWidget extension-id="..." entry="dist/widget.js" />` or
`<sf-extension-widget>` in L1 HTML. Author builds once; site does not rebuild.

## Web Release

Only for **trusted admin plugin frontends**, not public themes.

## Scaffold

```sh
cd apps/api && go run ./cmd/sforum make:theme example.theme
```

See also `docs/extensions/page-catalog.md`.
