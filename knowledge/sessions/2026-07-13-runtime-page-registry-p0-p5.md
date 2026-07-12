# 2026-07-13 Session Handoff — Runtime Page Registry P0–P5

## Changed

- Page Registry (catalog, Postgres bindings, resolve/approve/restore APIs, OpenAPI).
- Host owns public pages/components/layouts/CSS; `SFPageOutlet` + `SFThemeTemplate` L1.
- L0 skin via `/site/active-theme/skin` + theme-assets; no Nuxt rebuild.
- Theme activate is sync runtime-only; deleted `extension.theme_activate` job + ThemeRuntime package + production `runtime.mjs`.
- Default theme + Signal Garden runtime packages (`theme.json` L0/L1); Signal Garden mirrored under `tests/fixtures/themes/` (dev path gitignored).
- Admin Pages UI; L2 `SFExtensionWidget`; plugin data loader via extension route proxy fields.
- Web Release retained for trusted admin plugin frontends only.
- Docs: `docs/extensions/runtime-themes.md`, page-catalog, knowledge modules.

## Decisions

- Follow ADR `knowledge/decisions/2026-07-13-runtime-page-registry-themes.md`.
- Flags default: registry/L0/L1 on, layer activation off.
- P5 deletions immediate (no extra deprecation window).

## Next

- Optional: richer island registry / more `<sf-*>` mappings.
- Optional: remove remaining legacy layer trees under theme packages when operators no longer need them.
- Optional: drop unused `theme-proxy` / `dev-theme-runtime` once admin compose no longer needs them.

## Open Questions

- Whether to force-delete `dev-theme-runtime.mjs` entirely vs keep as optional `dev:compose` for admin HMR.
