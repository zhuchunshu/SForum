Milestone: M3 - Admin editor and accessible ordering
Status: completed

Changed:
- Replaced the legacy single-row tab with one revisioned draft for topbar,
  sidebar, mobile, and footer.
- Added source/state badges, typed safe links, shared `SFIconPicker`, transfer,
  dirty/revision handling, and one explicit batch save.
- Added FormKit drag plus accessible top/up/down/bottom controls that write the
  same draft model.

Contracts / migrations:
- Consumes M2 admin read and revisioned batch apply; no new persistence,
  transport contract, or migration.

Permissions / security / compatibility:
- `settings.site.manage` remains server-authoritative for all reads/writes.
- Only `operator.*` links are editable; Core/extension ownership and legacy
  topbar compatibility are unchanged.

Verification:
- `cd apps/web && bun test` -> PASS (698 pass, 0 fail, 4674 expectations)
- `cd apps/web && bun run typecheck` -> PASS
- `cd apps/web && bun run build` -> PASS (existing dependency warnings only)
- `node tests/validate-architecture-boundaries.mjs` -> PASS (1438 files)
- `git diff --check` -> PASS
- Authenticated Browser QA -> PASS: desktop plus `390x844`, shared geometry,
  location transition, validation, no horizontal overflow, save Toast and
  revision 1 -> 2 -> 3, plus user-confirmed real drag reorder.

Knowledge base:
- plan ledger: M3 completed; M4 is next
- module notes: `knowledge/modules/frontend.md`, `knowledge/modules/options.md`
- hot handoff: `knowledge/sessions/2026-07-28-public-navigation-platform-handoff.md`

Remaining risks:
- M4 must surface destructive preview fences and recovery commands without
  duplicating the document/editor state.

Next:
M4 - Backup, history, restore, and import UI.
