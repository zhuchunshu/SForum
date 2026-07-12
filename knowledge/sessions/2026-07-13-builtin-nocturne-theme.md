# 2026-07-13 Session Handoff — Builtin Nocturne Harbor theme

## Changed

- Added second **built-in runtime theme** package:
  `extensions/builtin/themes/sforum-nocturne/`
- Extension id: `sforum.nocturne-theme` (display: **Nocturne Harbor** / 夜港主题)
- Format: **L0/L1 only** (`theme.json` + `assets/*` + `templates/home.html`)
  — no Nuxt Layer, no admin frontend, no Web Release coupling
- Visual direction: deep navy / indigo paper, **cyan accent**, soft card
  chrome (distinct from default warm-orange shell and Signal Garden green)

## Decisions

- Second builtin theme is a **skin + home L1 chrome** package, not a full
  Layer fork of `sforum-default`. Host pages keep core components; L0 CSS
  recolors public tokens (`--sf-*` / `--sf-public-*`).
- Keep Signal Garden under `extensions/dev/` (and fixtures) as the green
  reference; Nocturne is the second **builtin** for operator theme switch
  without upload.

## Verify

```bash
cd apps/api && go run ./cmd/sforum extension validate ../../extensions/builtin/themes/sforum-nocturne
# Template + CSS preflight (ValidateTemplate / ValidateCSS) OK
```

## Next (operator)

1. Restart or re-sync API so `SyncBuiltins` picks up `sforum-nocturne`.
2. Admin → Themes → activate **Nocturne Harbor**.
3. If Page Registry requires it, super_admin approve `forum.home` replace.
4. With `bun run dev` + `air`, activation is **runtime-only** (no Nuxt rebuild).

## Open Questions

- Whether default theme’s leftover `layer/` tree should eventually shrink to
  L0/L1-only like Nocturne (optional cleanup; not required for this package).
