# 2026-07-30 Session Handoff

## Changed

- Added nullable per-user accent and daytime-background preferences. A missing
  row means live inheritance from operator settings.
- Added authenticated `PUT /auth/appearance` and `DELETE /auth/appearance`,
  shared appearance validation, OpenAPI contracts, persistence, and allowed /
  denied HTTP coverage.
- Added `/settings/appearance`, its account-sidebar entry, immediate in-memory
  preview, save-only persistence, reset/discard behavior, bilingual copy, and
  success feedback.
- Registered `forum.settings.appearance` across Page Registry, ViewModels,
  Host islands, and both built-in theme source packages.
- Unified normal and system-error document appearance resolution. Authenticated
  403/404/429/5xx pages now honor saved user accent/background overrides, while
  Core error fallbacks restore public options and session state before render.

## Decisions

- Appearance precedence is admin draft, user draft, saved user override, then
  operator default. Restoring site settings deletes the user override rather
  than snapshotting current operator values.

## Verification

- Focused frontend suite: 102 passed. Focused Appearance, Options, Identity,
  Identity controller, Pages, ThemeCompiler, and PageViewModels Go packages
  passed, including real HTTP session tests.
- `bun run build` completed successfully; only existing dependency/chunk and
  icon JSON import-attribute warnings were emitted.
- OpenAPI reference validation passed with 2,549 references. Both built-in
  theme source packages pass Manifest V3 validation with 32 templates.
- Browser interaction confirmed inheritance, immediate preview, unsaved reload
  rollback, saved reload persistence, and restoration to site inheritance;
  operator also completed visual verification without findings.
- Full frontend typecheck remains blocked only by unrelated dirty-worktree
  errors. The active browser runtime still reported Core fallback for this new
  page, so no immutable active-theme runtime claim is made here.
- Appearance writes now live in a focused `AppearancePreferenceService` rather
  than growing the legacy Identity service. Architecture validation still
  reports unrelated concurrent growth in Identity external-auth, Extensions,
  and ExtensionManifest files.
- Error-page follow-up: 50 focused frontend tests passed and the production
  build completed. Authenticated Browser QA compared a normal page with a hard
  refreshed 404 at desktop and 390x844: both resolved `violet`, `cloud_blue`,
  `#7c3aed`, and `#f1f6fb`; the 404 reported `sforum.default-theme`,
  `data-template="1"`, no overflow, no framework overlay, and no console
  warnings. The QA account was restored to site inheritance afterward.

## Next

- Rebuild/stage and activate a built-in theme artifact containing
  `forum.settings.appearance`, then verify `/pages/resolve` reports the active
  theme provider and `data-template="1"`.

## Open Questions

- None.
