# 2026-08-19 Mobile Drawer Viewport Fix

## Changed

- Removed the stale topbar offset from the default theme's mobile drawer and
  backdrop, restoring the approved full-viewport `top: 0` / `inset: 0`
  contract already used by the Core fallback.
- Added a focused regression that checks the Core and default-theme CSS owners
  together.
- Rebuilt the development built-ins and reactivated the exact default-theme
  artifact through the normal super-admin flow. The active package digest is
  `0020c49a61597d770c5082044bc2c84332c664f909b90c67be0124caf8db0683`.

## Root Cause

- The 2026-08-01 global mobile style change updated only Core
  `sforum-theme.css`. The default runtime theme retained its older
  `var(--sf-public-topbar-height)` declarations and overrode Core after the
  immutable skin loaded.

## Verification

- Full Web suite: 891 passed, 0 failed.
- Nuxt typecheck passed.
- Architecture boundary validation passed.
- Default-theme `extension validate` and `extension test` passed.
- Runtime `/site/active-theme/skin` and `/pages/resolve` both report the new
  digest, selected `sforum.default-theme`, and non-fallback `forum.home`.
- Authenticated Chrome at `402x849` reports both drawers at `top: 0` and the
  backdrop covering the complete viewport. Left drawer bounds are
  `0,0,320,849`; right drawer bounds are `82,0,402,849`. There is no horizontal
  overflow and no browser warning or error.

## Next

- None.

## Open Questions

- None.
