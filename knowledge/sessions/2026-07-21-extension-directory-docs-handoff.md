# 2026-07-21 Session Handoff

## Changed

- Added `extensions/README.md` — directory map, SyncBuiltins vs dev build vs
  optional/fixtures/upload.
- Added `extensions/dev/README.md` — scaffold-only, not auto-registered.
- Documented “Where to put your package” in
  `docs/extensions/authoring-guide.md`.
- Cross-links: `extensions/optional/README.md`, `extensions/fixtures/README.md`,
  `knowledge/modules/extensions.md`, `AGENTS.md` stack map.

## Decisions

- Author source of truth for layout: `extensions/README.md` + authoring-guide
  section; knowledge module points there rather than duplicating a second table.

## Next

- None required for docs. When adding a new protected backend builtin, update
  `scripts/build-builtin-plugins.sh` hard-coded list and keep the README list
  in sync if that inventory is still listed explicitly.

## Open Questions

- None.
