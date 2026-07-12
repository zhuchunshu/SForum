# Nocturne Harbor

Built-in SForum public theme with a cool indigo-harbor look: deep navy
surfaces, cyan accent, and soft card chrome.

- ID: `sforum.nocturne-theme`
- Type: `theme`
- Format: **runtime L0/L1** (`theme.json` + CSS + HTML templates)
- No Nuxt Layer, no Web Release, no site rebuild on activate

## Package layout

```text
sforum-nocturne/
  sforum.extension.json
  theme.json
  assets/theme.css
  assets/tokens.css
  templates/home.html
```

## Style direction

| Token family | Role |
| --- | --- |
| Navy / ink | Page background and primary text |
| Cyan | Accent (buttons, links, focus rings) |
| Soft card white / slate | Surfaces and borders |

Compared with:

- **sforum.default-theme** — warm orange left-rail forum shell
- **sforum.signal-garden** (dev) — bright green community garden

## Activation

1. API syncs builtins (`SyncBuiltins`) from `extensions/builtin/themes/`.
2. Admin **Activate theme** → sync preflight + L0 CSS inject.
3. Replacing core `forum.home` still requires **super_admin** page approve
   when using Page Registry replace bindings.
4. Missing pages fall back to host core pages.

See `docs/extensions/runtime-themes.md` and
`docs/extensions/page-catalog.md`.
