# M0 - Contract And Production-Wiring Freeze

Status: completed

## Changed

- Added the v1 SiteChrome navigation contract and modular OpenAPI endpoints.
- Mapped the real production path and separated it from compatibility paths.

## Production Wiring

`extension manifest -> lifecycle NavigationRegistry publication ->
pageSiteChromeService -> CorePageViewModels -> Page Registry template/Host
islands` is the production SSR path. `GET /site/nav-items` is currently a
legacy controller projection from `SiteChromeProvider` plus the direct
`forum.nav.items` provider; it is not the V3-composed authority.

## Frozen Semantics

- Locations: topbar, sidebar, mobile, footer; Core owns placement/document and
  themes report only rendering capability.
- `site_nav_items` remains the migration source and LTS topbar projection.
- Definitions, placements, overrides, sources, snapshots, and revisions are
  additive M1/M2 tables; plugin references remain external stable identities.
- Unsafe mutations use `settings.site.manage`, CAS revision, audit, prior
  snapshot, and public-surface revision bump after commit.
- Import is validate/preview then fenced merge or replace. Limits: 200
  definitions, 800 placements, 512 KiB backup, 20 snapshots.

## Drag Survey

Selected for M3 pending installation: `@formkit/drag-and-drop@0.6.1`, MIT,
Vue export, npm provenance, 909179-byte unpacked package. It is only mounted
after client hydration. Its public package metadata does not claim complete
keyboard sorting; button and keyboard-safe move controls are mandatory.

## Verification

- PASS `ruby scripts/validate-openapi-refs.rb`: 2450 refs, 54 files.
- PASS `go test ./app/Models/SiteChrome/... ./app/Support/NavigationRegistry/...`.
- PASS `node tests/validate-architecture-boundaries.mjs`.
- PASS `bun test`: 696 pass, 0 fail. The test baseline now follows moved
  identity components, the generated admin catalog, and the page-aware topic
  path contract; moderation's KPI rail no longer adds an extra horizontal
  inset against its stated shared-home geometry.

## Unblock

M1 is next. Add no persistence beyond the approved additive model and keep the
legacy topbar table/API as a compatibility projection.
