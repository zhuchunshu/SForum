# 2026-07-21 Session Handoff — V3 P13 forum.home presentation ownership

## Changed

- `apps/web/app/components/SFHomePage.vue` — full home interactive UI island.
- `apps/web/app/pages/index.vue` — thin SEO + `SFPageOutlet` + fail-closed
  `<SFHomePage />` slot.
- `SFThemeTemplate` maps `forum.component.home_page` → `SFHomePage` (not
  `HostPageIsland`).
- Default + Nocturne `templates/home.html` mark `data-theme-owned="presentation"`.
- Tests: thin route shell, island contracts, theme shells, mapping.

## Commits

- `5b26b80fe` feat(web): extract forum.home body into SFHomePage island
- `e7fb57ed7` feat(themes): mark default and nocturne home shells theme-owned
- `65181c354` test(web): cover forum.home presentation ownership migration

## Verified green

- `bun test tests/defaultThemeHomepage.test.ts tests/pageOutlet.test.ts`
  `tests/accountSecurityPage.test.ts tests/p9JoinedVisualMatrix.test.ts`

## Decisions

- Home body island is Host-owned component; themes own L1 shell structure.
- Fail-closed outlet fallback retained forever.
- Do not delete `LoadTemplate` / Protocol V1 until APILTS RemoveAfter + zero shim.

## Next

1. Repeat thin-route + body-island pattern for topic, auth, taxonomy, profile,
   settings, legal pages (additive; keep emergency slots).
2. Optionally move public CSS source-of-truth to theme packages while Host keeps
   emergency CSS for fail-closed.
3. Only after full page parity + LTS checklist: remove request-time loader and
   v1 paths.

## Unowned dirty WIP (do not stage)

- route-inspector web/OpenAPI, content-policy, PageViewModels, go.mod,
  host-api-v2.md, websocket revoke, ADR noise.
