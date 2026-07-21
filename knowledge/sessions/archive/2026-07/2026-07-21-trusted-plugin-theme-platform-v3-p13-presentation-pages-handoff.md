# 2026-07-21 Session Handoff — V3 P13 presentation page ownership

## Changed

- Extracted auth login/register/recovery forms into Host body islands
  (`SFLoginFormPage`, `SFRegisterFormPage`, `SFRecoveryRequestPage`,
  `SFRecoveryConfirmPage`); route shells keep guest/public meta + SFPageOutlet.
- Mapped identity auth components in `SFThemeTemplate` to those islands.
- Marked default/nocturne auth + not-found L1 shells
  `data-theme-owned="presentation"`.
- Completeness gate requires theme-owned marker on every replaceable L1 template.
- Web tests cover auth guest-on-shell / form-on-island, not-found ownership, and
  moderation.review non-replaceable boundary.

## Commits

- `db29579bc` feat(web): extract auth form pages into Host body islands
- `ceb2ccc04` feat(themes): mark auth L1 shells theme-owned
- `c9e17dd7f` test(web): cover auth form presentation ownership
- `e514099b0` feat(themes): mark not-found L1 shells theme-owned
- `01ccf51d5` test(pages): require theme-owned marker on replaceable L1 shells
- `2e1029921` test(web): cover not-found ownership and moderation non-replaceable

## Tests

- `bun test` presentationOwnershipRemaining + authRouteRendering + pageOutlet: pass
- `go test ./app/Support/Pages/ -run TestBuiltinThemesCoverAllReplaceablePages`: pass

## Status

- Replaceable public **page** presentation ownership: complete.
- Residual for P13 presentation row: Host `layouts/default.vue` chrome and
  Host island CSS under `apps/web/app/assets`.
- Residual for program close: LTS-blocked loader/v1/compatibility deletions.

## Next

1. Decide whether to migrate Nuxt public layout chrome into theme L1
   (`sf-navbar` / `sf-footer` placement) or accept Host chrome as residual.
2. Keep LTS deletion rows open until RemoveAfter + zero-shim telemetry.
3. Do not stage unowned dirty WIP listed in progress ledger.

## Open Questions

- None that block page-level ownership; layout/CSS migration is optional product
  depth vs honesty residual (document vs implement).
