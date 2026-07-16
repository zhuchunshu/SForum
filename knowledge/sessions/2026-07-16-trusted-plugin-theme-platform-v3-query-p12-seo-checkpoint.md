# 2026-07-16 Trusted Plugin And Theme Platform V3 Query/P12/SEO Checkpoint

## Status

- Weighted progress remains **62.6%**: P6 13/18, P7 14/22, P8 18/18, and
  P9 4/16. P10-P13 remain uncredited.
- The long-running goal remains active on `main`. Do not report completion until
  all 99 authoritative rows, 14 final boundaries, five reference plugin classes,
  and final verification gates pass.

## Changed

- `0815d2bce`, `7a8401e47`, `4127c1c4b`: immutable legacy and recovery plugin
  runtime desired full-set publication with real PostgreSQL/race evidence.
- `1b8a8064e`: exact Route runtime admission for selected providers,
  contributions, conflicts, and Core fallback.
- `776b9e089`, `a873e3a59`, `81e8f732d`, `f83d10b6b`: query delegation wire,
  Query Registry Host outlet, SDK helpers, and reflected-token rejection.
- `e5df1fcf8`, `f1dfd7efc`, `1c6dcd10b`: SEO Manifest, trust impact,
  lifecycle/startup/Safe Mode publication, and OpenAPI. `1c6dcd10b` contains both
  lifecycle and OpenAPI files because a delegate committed the shared index.

## Verification

- Query HostAPI, QueryRegistry, and SDK normal/race plus vet passed.
- P12 Models/CLI normal/race and isolated real PostgreSQL tests passed.
- Routes normal/race, Localization, and vet passed.
- SEO Manifest, Models, SEORegistry, Extensions normal/race/vet passed.
- OpenAPI validation passed all 1,900 references.

## Decisions

- No progress credit is awarded for these partial rows. Query lacks production
  bootstrap; P12 lacks complete node application ownership; SEO lacks provider
  transport and SSR consumers.
- Do not invent Query plugin handler, result-filter, join, or relation semantics
  absent the frozen Manifest contract.
- Delegates may edit and test only. The primary agent exclusively owns the Git
  index and commits after the shared-index incident above.
- External Grok/Codex CLI delegation stays disabled under managed private-repo
  disclosure policy. Do not bypass it.

## Next

1. Review and land Query production bootstrap with live identity RBAC and exact
   Manager runtime admission.
2. Wire plugin runtime coordinator and theme watcher into real API/worker
   ownership, then prove restart/two-node convergence.
3. Build the SEO provider-to-Host-final-policy-to-SSR vertical and its
   JavaScript-disabled failure matrix.
4. Recalculate progress only when an authoritative row closes completely.

## Preserve

- Never stage `apps/api/app/Models/PageViewModels/source_test.go`.
- Never stage
  `extensions/builtin/plugins/sforum-content-policy/sforum.extension.json`.
