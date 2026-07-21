# 2026-07-06 Session Handoff

## Changed

- Split extension lifecycle semantics: plugins use enable/disable, themes use
  verify/activate.
- Added protected built-in default theme package at
  `extensions/builtin/themes/sforum-default`.
- Moved public homepage, default/auth layouts, auth pages, navbar, footer, and
  public theme CSS into the default theme Nuxt Layer.
- Updated admin extension UI so themes show current/verify/restore-default
  states and uploaded theme activation is blocked in v1.
- Updated OpenAPI refs, frontend tests, homepage validation, and runtime config
  for `BUILTIN_EXTENSION_ROOT`.

## Decisions

- Only `sforum.default-theme` is actually applied in v1.
- Uploaded themes can be installed and verified but return
  `extension.theme_runtime_unavailable` when activated.
- `appearance.theme` remains the stored option key, but UI/docs should call it
  an appearance preset / 配色预设.

## Next

- Implement real plugin runtime restart/supervision before enabling the Restart
  action.
- Implement Nuxt rebuild, health-check, and rollback before uploaded themes can
  be activated.

## Open Questions

- None for this boundary; uploaded theme runtime design is future work.
