# 2026-07-06 Extension Stale Builtin Cleanup

## Changed

- Added startup pruning for `source=builtin` extension rows that are no longer
  present under `BUILTIN_EXTENSION_ROOT`.
- Added installed package and `sforum.extension.json` existence checks before
  extension verification and plugin enable operations.
- Added plugin runtime manager preflight for declared backend entry files.

## Decisions

- Stale built-in rows are system-owned synchronization artifacts, so startup
  sync may remove them when at least one current built-in manifest is found.
- Uploaded extensions remain untouched by built-in pruning.

## Next

- Add first-class uninstall/rollback endpoints before exposing manual extension
  deletion in the admin UI.

## Open Questions

- None.
