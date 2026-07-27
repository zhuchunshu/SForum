# 2026-07-27 Tri-State Color Mode Plan Handoff

## Changed

- Added the M0-M5 task book for reliable Automatic/Light/Dark preferences.
- Added exact Grok new-conversation prompts, a per-milestone small-report
  protocol, a rolling ledger, and final independent Codex review prompt.
- Recorded the confirmed `localhost` versus `127.0.0.1` origin split.

## Decisions

- Automatic (`system`) is the recommended default; Light and Dark are explicit.
- Stored preference and resolved Light/Dark are separate contracts.
- V1 reuses Nuxt Color Mode and browser-local storage; account sync is deferred.
- Cookie persistence is closed until SSR/shared-cache isolation is designed.
- Host owns preference; themes consume classes/tokens and plugins see resolved
  Light/Dark only.
- One safe canonical local origin fixes the confirmed development loss.

## Verification

- Planning-only change; no application code was modified.
- Current browser diagnosis proved same-origin refresh/navigation persistence
  and independent state between `localhost:3000` and `127.0.0.1:3000`.
- No relevant browser console warnings/errors were observed.

## Next

- Start only M0 with the exact prompt in
  `knowledge/plans/2026-07-27-tristate-color-mode-reliability.md`.
- M0 must freeze the shared API, native library choice, cache behavior, safe
  origin mechanism, and test matrix before implementation.

## Open Questions

- M0 must choose the exact framework-native canonical-origin mechanism and
  excluded request paths from current production wiring.
- M0 must confirm whether current Nuxt Color Mode startup has any same-origin
  persistence risk beyond the proven origin split.
