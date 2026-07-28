# M1 - Backend Persistence And Public Resolver

Milestone: M1 - Backend persistence and public resolver
Status: completed

## Changed

- Added focused SiteChrome document catalog, reader, resolver, typed link
  validation, capability seam, and public resolved-navigation controller.
- Routed HTTP and Page Registry through one production `SiteChrome.Service` so
  both observe the same exact-artifact Navigation Registry runtime.

## Contracts / Migrations

- Added additive migration `202607280072_site_navigation_v1.sql` with state,
  definitions, placements, snapshot table, indexes, and legacy topbar migration.
- Kept `site_nav_items` and `/site/nav-items` intact as API-LTS surfaces.

## Permissions / Security / Compatibility

- Public resolution filters authenticated and permission visibility from the
  server-side actor, rejects unsafe/reserved targets and forged source ownership,
  omits unavailable extension references, and honors Registry Safe Mode.
- No personalized output is cached as anonymous output; M2 owns mutation-time
  public-surface revision bumps and cache invalidation.

## Verification

- PASS: `cd apps/api && go test ./app/Models/SiteChrome/... ./app/Support/NavigationRegistry/... ./app/Http/Controllers/... ./bootstrap/... ./database/migrator/...`
- PASS: isolated `TestSiteNavigationV1MigrationPreservesLegacyTopbarRows` using local `DATABASE_URL`.
- PASS: `ruby scripts/validate-openapi-refs.rb` (2450 refs across 54 files).
- PASS: `node tests/validate-architecture-boundaries.mjs` (1434 production files scanned).
- PASS: `git diff --check`.

## Knowledge Base

- Plan ledger/checklist, Options module note, index, plans index, and single hot handoff updated.

## Remaining Risks

- Theme-declared location metadata and rendered location wiring are M6 work.
- M2 must make document revision, audit, snapshot, and public-surface revision atomic.

## Next

M2 - Revisioned commands, defaults, snapshots, and backup.
