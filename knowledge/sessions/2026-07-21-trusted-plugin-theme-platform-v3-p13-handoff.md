# 2026-07-21 Session Handoff — V3 P13 Residual

## Changed

- Fixed lifecycle publication fence tests (nine compatible digests; bare
  disabled policy on first CAS attempt).
- Authorized P9 extension inspector read routes (`extension.view` /
  `extension.manage`) and catalog partition tests.
- Hardened Host API v2 docs validator against gofmt spacing drift.
- Registered P9 inspector pages in admin framework validator + sidebar order.
- Bumped P0 route inventory gate to 244 routes.
- Added five-class ESM surface family union gate.
- Recorded final-gates evidence map; credited command gates.

## Verified green

- `go test ./...`
- `go build ./...` (API binary built for live probe)
- `bun run build` (Nuxt)
- `bun run typecheck` (via `scripts/test.sh`)
- `ruby scripts/validate-openapi-refs.rb` (via `scripts/test.sh`)
- `./scripts/test.sh`
- SEO JS-disabled product unit; RuntimeRollout; sforum recovery; bootstrap
  lifecycle/cleanup packages

## Decisions / accepted boundaries

- Do **not** delete Protocol V1 shims, core Nuxt public pages, request-time
  template loader, or emergency Page Outlet fallback until LTS checklist is
  fully true (`docs/extensions/v3/p13-migration-and-lts.md`).
- Media Manifest cannot declare scan/CDN stages (Host-runtime only).
- Live API cold-start in this workspace failed on missing storage root and
  unbuilt `sforum.smtp` backend entry — recorded, not papered over.

## Next

1. Operator env: create storage root, build builtin plugin backends (or safe
   mode), re-run API+worker live matrix and multi-route Baiduspider/desktop
   browser evidence.
2. Only after APILTS telemetry is zero for a full LTS window: execute migration
   deletion rows (still not in this session).
3. When residual browser/live rows close, set P13 to 100% and program complete.

## Open Questions

- Whether operators accept historical P8 Playwright + unit SEO JS-disabled as
  sufficient browser credit without a fresh mobile viewport pass.
- Whether live cold-start should ship a dev bootstrap that auto-creates
  storage root and skips unbuilt V1 plugin binaries under safe policy.

## Unowned dirty WIP (do not stage into V3)

- route-inspector web/OpenAPI, content-policy manifest, PageViewModels,
  go.mod, host-api-v2.md, websocket revoke test, ADR noise.

## Rollback

- Revert `afe221d89` … `359b1d375` for this session's P13 final-gate chain.
