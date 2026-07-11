# 2026-07-12 Session Handoff

## Changed

- Accepted design for splitting complex extension manifests:
  - Decision: `knowledge/decisions/2026-07-12-extension-manifest-split.md`
  - Implementation plan:
    `docs/superpowers/plans/2026-07-12-extension-manifest-split.md`
- Updated `knowledge/modules/extensions.md` Manifest section with planned
  multi-file authoring, `includes`, and directory-per-locale `langs`.
- Indexed the decision from `knowledge/index.md`.

## Decisions

- `sforum.extension.json` stays the only entrypoint.
- Host merges includes into one `Manifest`, then validates (OpenAPI-style).
- Root keeps identity defaults + high-risk runtime boundary + `includes`.
- Identity `langs` preferred complex form: `manifest/langs/{locale}.json`.
- Also support langs file list and single map file.
- Three i18n layers stay separate: identity langs, settings LocalizedText,
  frontend admin Vue locales.
- No dual-source for the same block (root vs includes).
- Simple single-file packages remain fully supported.
- Implementation not started; Phase 0 is documentation only.

## Next

1. Phase 1: implement `LoadPackage` + include merge + tests; wire all load
   sites (builtin sync, ZIP install, verify, CLI).
2. Phase 2: migrate `sforum.smtp` to thin root + `manifest/langs/` +
   settings/contributions partials.
3. Phase 3: scaffold + `extension validate` + author docs polish.

## Open Questions

- Whether API-stored raw JSON should keep the thin root or only the merged
  view (plan default: merged capability view for consumers; files on disk stay
  multi-file).
- Empty root arrays vs omitted keys for dual-source detection (plan default:
  empty/omitted = not defined).
