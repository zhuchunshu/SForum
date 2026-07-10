# 2026-07-11 Trusted Admin And Jobs Runtime Handoff

## Changed

- Completed immutable Web Release build, supervisor acknowledgement, API
  coordination, trusted Admin SDK rendering, trust/release UI, and stale-tab
  monitoring.
- Unified trusted plugin and theme lifecycle operations through Web Releases.
- Added the River Jobs admin API/page with view/manage permissions and three
  production trusted component slots.
- Added canonical Web Release deployment variables while retaining the legacy
  `theme_releases` volume and fallbacks.
- Added initial canonical release bootstrapping and daily retention cleanup for
  active/rollback/five-success artifacts, seven-day failed artifacts, and
  thirty-day build logs.
- Added a real trusted Jobs-slot fixture and local registry integration gate.
  The gate covers frozen install, disabled scripts, host-peer deduplication,
  clean Nuxt typecheck/build, Node preview, and cross-language artifact digest.
- Recovered a partially applied moderation migration and added a forward
  migration for the missing `web_releases.composition_snapshot` column.

## Decisions

- Trusted Vue code remains build-time, client-only, digest-approved code.
- River remains authoritative for job transitions; SForum supplies policy,
  presentation, aggregation, and explicit extension points.

## Verification

- Related Go race tests and full API builds passed.
- OpenAPI references, Nuxt typecheck, and full Nuxt production build passed.
- The complete trusted fixture Web Release build and Node preview passed.
- API migrations applied through `202607110013`; health returned HTTP 200.
- Browser verified admin routing and the Jobs workbench with a super-admin
  session before that browser session expired after backend restart.

## Next

- Expand Jobs history beyond the bounded newest-100 window only when operator
  demand justifies cursor UI.
- Add deployment metrics/export only after stable operator semantics are known.
