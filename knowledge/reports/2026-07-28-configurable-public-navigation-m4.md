Milestone: M4 - Backup, history, restore, and import UI
Status: completed

Changed:
- Added one-location/all-location recommended-default previews, explicit
  destructive confirmation, snapshot history/detail/restore, JSON export, and
  two-phase merge/replace import inside the existing Navigation tab.
- Added bounded client file/schema/shape validation, structured change and
  inert-extension warnings, preview-token/revision fencing, persistent errors,
  and themed 10-second success Toasts.
- Added snapshot actor attribution while preserving nullable attribution for
  historical rows, and corrected the OpenAPI backup/preview response shapes.

Contracts / migrations:
- Added additive migration `202607280073_site_navigation_snapshot_actor.sql`:
  nullable `actor_user_id` with `ON DELETE SET NULL`.
- Snapshot APIs now expose optional actor identity and structured
  `changeEntries`/`warningEntries`; legacy preview fields remain compatible.

Permissions / security / compatibility:
- `settings.site.manage` remains authoritative for every recovery endpoint;
  client controls do not replace API authorization.
- Import stays server-previewed and actor/revision fenced; extension references
  remain inert, and exports contain no database ids, actors, or secrets.
- Default/replace/restore operations require explicit confirmation and do not
  delete plugin declarations or secrets.

Verification:
- `cd apps/web && bun test` -> PASS (702 pass, 0 fail, 4689 expectations)
- `cd apps/web && bun run typecheck` -> PASS
- `cd apps/web && bun run build` -> PASS (existing dependency/chunk warnings)
- `cd apps/api && go test ./app/Models/SiteChrome/... ./app/Http/Controllers/SiteChrome/...` -> PASS
- isolated PostgreSQL migration/command tests -> PASS (2 tests, 16.393s)
- `ruby scripts/validate-openapi-refs.rb` -> PASS (2457 refs / 54 files)
- `node tests/validate-architecture-boundaries.mjs` -> PASS (1443 files; 166 above review threshold)
- `git diff --check` -> PASS
- Authenticated Browser QA -> PASS: desktop and `390x844`, no horizontal
  overflow or console errors, recovery dialog `358x412`, revision 4 -> 5
  default restore and 5 -> 6 snapshot restore, actor `用户 #1`, history detail,
  themed Toasts, and original topbar restored. User confirmed JSON export and
  import both work.

Knowledge base:
- plan ledger: M4 completed; M5 is next
- module notes: `knowledge/modules/frontend.md`, `knowledge/modules/options.md`
- hot handoff: `knowledge/sessions/2026-07-28-public-navigation-platform-handoff.md`

Remaining risks:
- M5 must make all topbar/mobile/Core fallback render paths consume the same
  SSR payload while preserving Host-owned utility controls and fail-closed
  behavior.

Next:
M5 - Topbar, mobile, and Core fallback runtime wiring.
