# 2026-07-23 Profile Settings Canvas Handoff

## Changed

- Reworked `/settings/profile` into the Option B-inspired canvas while keeping
  the existing Page Registry body island and default-theme ownership boundary.
- Kept the real profile and avatar APIs, draft/baseline dirty tracking, field
  validation, success/error alerts, Toast feedback, localized zh-CN/en-US copy,
  desktop three-column layout, and mobile settings drawer.
- Added the frontend `attachment.upload` permission constant so avatar upload
  controls respect both runtime avatar options and the user's upload grant.
- Split the large scoped style block into a focused CSS asset and extracted the
  shared live preview so desktop, inline mobile, and the default-theme right
  drawer render the same draft data.
- Connected the page to the navbar's shared left/right drawer state and changed
  success Toasts to the active appearance primary color.
- Added a focused Bun test covering Page Registry ownership, real API fields,
  avatar permission/API wiring, preview dirty state, responsive drawer
  primitives, and i18n completeness.

## Decisions

- The right preview intentionally mirrors the default public profile fields and
  does not fabricate signature, follower, privacy, or other product data that
  the current profile API does not own.
- The save flow resets the draft baseline only after a successful `PUT
  /profile`; avatar upload/removal updates profile/avatar state without marking
  text fields dirty.

## Next

- User review and merge from branch `codex/profile-settings-canvas`; no Git
  commit was created in this session.
- Optional before merge: rerun broader repository gates if other branches move
  underneath this worktree.

## Open Questions

- None for the current page scope.
