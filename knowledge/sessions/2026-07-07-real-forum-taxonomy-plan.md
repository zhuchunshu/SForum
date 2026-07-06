# 2026-07-07 Real Forum Taxonomy Plan Handoff

## Changed

- Added the formal implementation plan at
  `docs/superpowers/plans/2026-07-07-real-forum-categories-tags.md`.
- No feature implementation has started in this handoff.

## Decisions

- SForum must be treated as an open-source forum framework, not a single
  hard-coded forum deployment.
- Core forum defaults must be safe and ready to use, but configurable and
  resettable from admin UI.
- The first taxonomy implementation should cover two-level category groups,
  public/hidden categories, controlled-flexible tags, public filters, admin
  management, OpenAPI, permissions, events, and knowledge-base updates.
- Category access v1 remains public/hidden only.
- Tags default to controlled mode. Operators can switch to review or open mode
  through admin settings.

## Next

- Read `docs/superpowers/plans/2026-07-07-real-forum-categories-tags.md`.
- Start at Task 1 and implement task-by-task with tests.
- Keep unrelated theme runtime worktree changes intact.

## Open Questions

- None for Phase 1 implementation.
