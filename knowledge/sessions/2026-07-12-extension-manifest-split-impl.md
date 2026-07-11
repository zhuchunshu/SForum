# 2026-07-12 Session Handoff

## Changed

- Implemented multi-file extension manifests end-to-end (Phases 1–4 plan):
  - `LoadPackage` / `LoadPackageFS` / includes merge
    (`apps/api/app/Support/ExtensionManifest/load.go`)
  - Builtin sync + ZIP install + snapshot load paths wired
  - `sforum.smtp` migrated to `manifest/*` layout
  - `make:plugin --complex` scaffold
  - `sforum extension validate [path] [--json]`
  - Settings/contributions directory shards with unique-key checks

## Commits (main, for rollback)

1. `feat(extensions): load multi-file manifests via includes merge`
2. `feat(extensions): migrate sforum.smtp to multi-file manifest`
3. (this session) CLI + complex scaffold + docs

## Decisions

- Snapshot stores merged canonical entry JSON; partial files remain on disk.
- Identity langs prefer `manifest/langs/{locale}.json`.
- Dual-source root vs includes is rejected.
- Settings i18n-schema split deferred.

## Next

- Restart API so builtin sync re-snapshots SMTP multi-file package if needed.
- Optional: pack/marketplace helpers beyond `extension validate --json`.

## Open Questions

- Whether stored DB manifest JSON should ever retain raw includes (current:
  merged only).
