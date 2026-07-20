# 2026-07-21 Session Handoff — V3 P13 presentation chrome ownership

## Changed

- Added `SFHostPublicChrome` for Page Registry core/fallback public chrome.
- Nuxt `layouts/default.vue` is a pass-through (no Host navbar/footer).
- `SFPageOutlet` wraps non-auth fail-closed pages with host chrome; theme success
  path does not double-wrap.
- Default + Nocturne non-auth L1 templates mount `sf-navbar` and `sf-footer`.
- Completeness gate requires chrome islands + `data-theme-owned=presentation`.
- Task book: presentation migration + Program DoD theme ownership rows checked.

## Commits

- `762119312` feat(web): add SFHostPublicChrome for fail-closed public chrome
- `c195871bf` feat(web): thin Nuxt default layout; fail-closed uses host chrome
- `3789225a1` feat(themes): mount navbar and footer islands on public L1 shells
- `6b49b59f1` test(presentation): cover theme chrome islands and host fail-closed shell

## Tests

- bun: defaultThemeHomepage, presentationOwnershipRemaining, pageOutlet,
  authRouteRendering, p9JoinedVisualMatrix — pass
- go: TestBuiltinThemesCoverAllReplaceablePages — pass

## Residual

- LTS-blocked: request-time template loader, Protocol V1 paths, compatibility
  path removal (APILTS RemoveAfter + zero-shim required).
- Host island CSS for interactive body islands remains Host-owned by design.

## Next

1. Do not delete LoadTemplate / Protocol V1 / fail-closed SFPageOutlet.
2. Leave LTS rows open until telemetry checklist.
3. Do not stage unowned dirty WIP.
