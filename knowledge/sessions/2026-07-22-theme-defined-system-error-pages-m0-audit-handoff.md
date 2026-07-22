# 2026-07-22 Theme-Defined System Error Pages — M0 Audit Handoff

## Changed

- Completed **read-only M0** for
  `plans/2026-07-22-theme-defined-system-error-pages.md`.
- Recorded baseline flow, producers, timing/SSR evidence, frozen contract map,
  and overlapping file ownership in the plan.
- **No production code, OpenAPI, theme package, or ADR file was modified.**

## Decisions

- Dependency gate remains closed: current-HEAD regression M0–M6 are evidenced
  complete, but **M7 is still open** and there is **no explicit handoff** of
  shared Page Registry / error files.
- Therefore M1–M6 of the error-pages book must not start. Only audit notes may
  land in the plan/knowledge until the gate opens.
- Framework-native stack is sufficient; no new library.
- Proposed production system-error resolve budget: **≤1s, single attempt**.

## Next

1. Wait for regression plan M7 green **or** an explicit written handoff of the
   overlapping files listed in the theme-error plan M0 section.
2. On unblock: write ADR for D1–D7, finish M0 exit, then implement M1→M6 in
   order (catalog/OpenAPI → Host islands → runtime restrictions → built-in
   themes → browser matrix → full gate).
3. Preserve regression fixes: selected-theme `system.not_found`, fail-closed
   Core fallback, no `forceDefaultTheme` reintroduction.

## Open Questions

- When will regression M7 close / who owns the handoff?
- Whether 502/504 appear as full Nuxt pages in real deploys.
- Final Component Catalog registration path for error details/actions islands.
