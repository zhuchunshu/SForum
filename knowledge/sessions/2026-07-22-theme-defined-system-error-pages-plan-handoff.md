# 2026-07-22 Theme-Defined System Error Pages Plan Handoff

## Changed

- Added the ready task book
  `knowledge/plans/2026-07-22-theme-defined-system-error-pages.md`.
- Defined selected-theme L0/L1 coverage for 403, 404, 429, and server-error
  browser pages with complete Host emergency fallback.
- Registered the plan in the knowledge index, frontend/extensions modules, and
  plan ledger.

## Decisions

- System errors are virtual Page Registry surfaces with stable IDs; they are not
  routable public paths.
- Themes own L1 structure while Host owns the original status, safe localized
  content, actions, SEO, caching, authorization, and fallback.
- System error pages are theme-only replacement surfaces and may not execute L2
  or accept plugin replacements.
- 5xx theming is best-effort. Resolver/runtime failure must immediately render a
  complete non-recursive Host emergency page with the same status.
- No new dependency, admin setting, or permission is needed.

## Next

- Wait for or obtain an explicit handoff from the overlapping current-HEAD
  regression work before editing shared Page Registry/error files.
- Start M0 read-only: audit actual error producers, selected-theme SSR assets,
  resolver timing, Component Catalog generation, and current worktree ownership.
- Write the ADR and freeze exact contracts/component IDs before M1 code.
- Implement M1-M6 in order and report exact gate/browser evidence per milestone.

## Open Questions

- M0 must measure the final system-error resolve timeout under the frozen
  maximum of one second.
- M0 must confirm which 502/504 cases reach Nuxt as full-page errors.
