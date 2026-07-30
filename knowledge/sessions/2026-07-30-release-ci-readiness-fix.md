# 2026-07-30 Release CI Readiness Fix

## Changed

- Repaired the GitHub Actions release gate against the current worktree:
  architecture responsibilities were moved to focused owners, stale static
  fixtures and exact catalog counts were synchronized, missing localization
  keys were added, and Nuxt type failures were corrected.
- `LocalePreferenceService` now owns current-user locale persistence;
  attachment storage provider discovery is owned directly by
  `AttachmentStorageProviderCatalog`; extension settings and Manifest settings
  declarations live with their lifecycle/document owners. Reduced architecture
  baselines were retained.
- The moderation validator now follows the real topic page -> composer drawer
  -> comment submission ownership chain. `SFEditorToolbar` has stable P9
  identity `core.component.shared.sfeditor_toolbar`, and generated V3 catalogs
  now contain 327 routes and 267 UI surfaces.
- Protected `sforum.auth-github` advanced to `1.0.2` with its release baseline.
  No commit, push, tag, or GitHub Release was created.

## Verification

- Passed actionlint, release/exact-CI/asset validators, architecture boundaries,
  built-in plugin release validation, OpenAPI refs, protocol/Host API V2 docs,
  compatibility farm, Go build, focused Go catalog/Page Registry tests, Nuxt
  typecheck and production build, all 792 Bun tests, and all product validators.
- Browser QA passed on `/topics/new`: the desktop editor toolbar rendered and
  toggled bold state accessibly; `390x844` had no page-level horizontal
  overflow, and the toolbar scrolled within its own region. Console errors and
  warnings were empty.
- The Codex sandbox denies `/bin/ps` for
  `TestDevCleanupOrphanPluginsDryRunExecutesRealPath`. All other `cmd/sforum`
  tests pass when that one environment-specific test is skipped; Actions is not
  subject to this local sandbox restriction.
- The temporary compatibility database `sforum_ci_fix_20260730` was dropped.

## Next

- Commit and push the reviewed worktree, require the main CI to pass, then
  manually release `v3.0.0-alpha.3`. Do not reuse `v3.0.0-alpha.2`: that tag
  already identifies the failed SHA and release identities are immutable.

## Open Questions

- None for the CI repair.
