# 2026-07-06 Extension Platform v2 Planning

## Changed

- Added `docs/extension-platform-v2.md` as the product and technical roadmap
  for the next extension-platform iteration.
- Added an accepted decision record for Extension Platform v2 direction.
- Updated the extension module note with the v2 target loop, admin manifest
  direction, Provider Slot priorities, and staged next steps.
- Added knowledge index and docs index pointers for future sessions.

## Decisions

- SForum should provide WordPress-like extension management for operators, but
  should not copy WordPress' PHP include runtime model.
- Extension admin sidebar entries must be opt-in through manifest metadata.
- `Manage` should resolve to an in-admin route selected by manifest entry,
  settings page, first declared page, or generated detail fallback.
- `mail.provider` is the recommended first full v2 vertical slice.
- Uploaded theme activation must wait for a build, health-check, preview,
  atomic switch, and rollback pipeline.

## Next

- Implement manifest `admin` compatibility and validation while preserving
  existing `adminPages`.
- Update admin extension navigation and list-row Manage routing to follow the
  v2 resolution rules.
- Design and implement the `mail.provider` host contract, default no-op
  provider, provider selection/reset UI, and first example plugin.
- Add plugin logs and richer enable-failure rollback visibility.

## Open Questions

- Whether the first mail provider should be a protected built-in SMTP plugin,
  an uploaded example plugin, or both.
- Which extension-owned frontend page layer is acceptable after host-rendered
  pages and safe manifest content are exhausted.
