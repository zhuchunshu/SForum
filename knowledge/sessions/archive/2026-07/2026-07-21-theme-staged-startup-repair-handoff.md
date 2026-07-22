# 2026-07-21 Session Handoff

## Changed

- Fixed API boot failure when the active builtin theme package still references
  retired host islands (`sf-my-home-page` after `/my` removal).
- Root cause: `SyncBuiltins` only **stages** a new digest for an enabled theme;
  the active package and `theme_runtime_publications` kept the old digest, so
  page preflight and the theme runtime watcher both failed, and fail-closed
  fallback hit the same broken default package.
- Startup repair now promotes a healthy staged builtin theme via
  `ActivateThemeExact` when active-package preflight fails, and uses the staged
  artifact for default-theme fallback compilation without switching themes.

## Decisions

- Prefer staged-builtin promotion over re-allowlisting removed host islands.
- Keep explicit admin activation for normal theme upgrades; auto-promote only
  when the active package can no longer pass host page preflight.

## Next

- None required for boot. Optional: admin UI surface for “staged builtin theme
  ready to activate” so operators can promote without waiting for a failure.

## Open Questions

- None.
