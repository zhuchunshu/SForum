# 2026-07-23 Extension Permission Localization Handoff

## Changed

- Extension permission labels/descriptions now accept `LocalizedText` maps.
- Identity publication binds locale maps; permission APIs resolve them by the
  request locale and return `label` plus `description`.
- Admin permission screens use extension-provided labels after built-in Core
  i18n, and the admin-surface reference fixture declares zh-CN/en-US copy.
- Migration `202607231001` adds permission presentation metadata.

## Decisions

- Extension-specific permission translations stay out of Core locale files.
- Permission ownership and grants remain unchanged; localization is display
  metadata protected by the existing extension catalog owner check.

## Next

- Run the full repository gate after unrelated current worktree typecheck
  failures are resolved.

## Open Questions

- None for this change.
