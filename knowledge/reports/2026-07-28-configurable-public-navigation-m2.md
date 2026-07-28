Milestone: M2 - Revisioned commands, defaults, snapshots, and backup
Status: completed

Changed:
- Added the SiteChrome command boundary for revisioned document CAS, defaults,
  snapshots, export/import previews, and restore.
- Added permission-protected admin endpoints and full denied-route coverage.
- Added bounded opaque actor/revision-fenced preview tokens and inert extension
  backup references.

Contracts / migrations:
- Uses the M1 additive navigation schema; no legacy table rewrite.
- Added stable SiteChrome audit actions and a focused Options transaction
  public-surface revision collaborator.

Permissions / security / compatibility:
- `settings.site.manage` is authoritative for each M2 read/mutation endpoint.
- Only `operator.*` definitions are directly editable; Core is canonicalized,
  and extension references stay inert until registry publication.
- `/site/nav-items` and `site_nav_items` remain untouched API-LTS surfaces.

Verification:
- `cd apps/api && go test ./app/Models/Options/... ./app/Models/SiteChrome/... ./app/Http/Controllers/SiteChrome/... ./bootstrap/...` -> PASS
- `DATABASE_URL=<local isolated schema> go test ./database/migrator -run 'TestSiteNavigation(V1MigrationPreservesLegacyTopbarRows|CommandsAreAtomicAndRetainSnapshots)' -count=1` -> PASS (migration preservation, concurrent CAS, audit/store rollback, retention, restore)
- `ruby scripts/validate-openapi-refs.rb` -> PASS (2450 refs / 54 files)
- `node tests/validate-architecture-boundaries.mjs` -> PASS (1436 files)
- `git diff --check` -> PASS

Knowledge base:
- plan ledger: M2 completed
- module notes: `knowledge/modules/options.md`
- hot handoff: `knowledge/sessions/2026-07-28-public-navigation-platform-handoff.md`

Remaining risks:
- M3 must present conflict recovery and inert extension state without creating
  a second navigation model in the browser.

Next:
M3 - Admin editor and accessible ordering.
