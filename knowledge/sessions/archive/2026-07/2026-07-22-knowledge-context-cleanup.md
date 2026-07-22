# 2026-07-22 Knowledge Context Cleanup

## Changed

- Reduced `knowledge/index.md` to active workstreams and current state.
- Kept seven hot handoffs; moved completed and superseded handoffs to the July
  archive.
- Moved completed/superseded task books to `plans/archive/2026-07/` and moved
  generated V3 traceability to `docs/extensions/v3/catalogs/`.
- Rewrote extension, forum, and frontend module notes around current behavior,
  boundaries, important paths, and real next work.
- Compacted the V3 parent plan and progress ledger; detailed history remains in
  Git and archived handoffs.
- Moved stale maturity, legacy-gap, and early-research documents to
  `knowledge/archive/`.

## Decisions

- Default context excludes session/plan archives.
- `index.md` stays below 150 lines and is not a changelog.
- Ordinary hot handoffs stay below 80 lines and one per active workstream.
- Generated catalogs live under `docs/`, not active plans.

## Next

- Apply the same living-note cleanup when another module exceeds roughly 500
  lines or contains completed work in `Next Steps`.
- Archive a hot handoff in the same change that closes or supersedes its track.

## Open Questions

- None for this cleanup.
