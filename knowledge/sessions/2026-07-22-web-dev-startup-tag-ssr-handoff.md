# 2026-07-22 Web Dev Startup And Tag SSR Handoff

## Changed

- Tag detail AsyncData now has stable empty shapes, concurrent reads, safe
  translation calls, and no template-side `.length` access.
- Page-level theme Host islands use Nuxt lazy components; navbar/footer remain
  eager because they are shared public chrome.
- Nuxt DevTools is opt-in via `NUXT_DEVTOOLS=true`.
- Development disables payload extraction to avoid stale HMR
  `/_payload.json` 404/HTML responses and hydration ownership mismatches.
- Guest middleware no longer reaches `useRoute()` through the auth return
  helper; it uses the middleware `to` route.

## Decisions

- Production payload extraction and public SWR behavior remain enabled.
- Dev startup favors a stable default; operators can explicitly enable
  DevTools when diagnosing a frontend issue.

## Next

- No required follow-up for this bug.

## Open Questions

- None.
