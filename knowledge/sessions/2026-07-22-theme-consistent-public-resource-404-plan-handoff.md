# 2026-07-22 Theme-Consistent Public Resource 404 Plan Handoff

## Changed

- Added focused task book:
  `../plans/2026-07-22-theme-consistent-public-resource-404.md`.
- Scope is deliberately limited to public resource-not-found 404s: topic,
  category, tag, profile, and unknown routes.
- Recorded live evidence that missing topic 60 returns API 404 but renders
  `provider=core`, Host chrome, HTTP 200, and SWR cache, while a healthy topic
  uses `sforum.default-theme` L1.
- No production code, contract, theme package, or dependency was changed.

## Decisions

- Ordinary business 404 must keep the healthy selected theme's real chrome.
- Default theme 404 keeps its navbar and desktop left navigation; the right
  information rail may be intentionally absent.
- Semantic 404 is non-retryable and must not become `transport_unavailable`.
- Resource 404 enters `system.not_found`; it does not render partial resource
  data or redirect home.
- Host owns HTTP status, safe copy/actions, privacy, SEO, and cache policy;
  theme owns L1 layout/presentation.
- Core remains mandatory but emergency-only for actual theme/runtime failure.
- No new dependency, operator setting, permission, or public L2 execution.

## Next

1. In a fresh implementation task, read the focused task book and start G0.
2. Actively run and close current-head regression M7, including its full gate,
   knowledge completion handoff, and explicit release of overlapping files.
3. Then implement M0-M6 in order: contract freeze -> classification/retry ->
   real Nuxt 404 -> themed body/chrome -> resource/security matrix -> browser
   proof -> full gate.
4. On completion, update the broader system-error book to consume this 404
   slice without claiming 403/429/5xx are done.

## Open Questions

- Whether regression M7 reveals an out-of-scope external blocker; ordinary
  in-scope regression failures are owned by G0 and should be fixed there.
- The exact reviewed Component Catalog ID for the dedicated 404 body island;
  M0 must freeze it before code changes.
