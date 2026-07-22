# 2026-07-23 Current HEAD Regression Completion Handoff

## Changed

- Closed `knowledge/plans/2026-07-22-current-head-regression-remediation.md`
  M7/G0 and released shared Page Registry/error-flow files to the focused
  public-resource 404 task.
- Repaired stale PostgreSQL identity fixture migration lists so
  extension-owned permission localization exists before identity registry
  integration tests run.
- Updated the HostAPI command-domain Postgres harness to apply embedded Goose
  migrations, preserving `NO TRANSACTION` migrations such as concurrent index
  creation.
- Made bootstrap lifecycle/reference-package E2E tests establish the production
  plugin runtime genesis boundary before manual lifecycle/runtime operations.
- Restored trusted admin settings components to the reviewed digest-bound API
  asset endpoint and refreshed V3 route catalog metadata plus reviewed catalog
  validation counts.

## Decisions

- G0 did not broaden into public 404 implementation. Ordinary 404 theme
  behavior remains owned by
  `knowledge/plans/2026-07-22-theme-consistent-public-resource-404.md`.
- Production plugin runtime genesis is required before legacy enable/disable
  publication. Tests that manually construct production services must seed the
  same immutable authority instead of relying on package order.
- Lifecycle V2 E2E fixtures that hand-build an enabled runtime start from an
  empty startup genesis, then create the enabled fixture, so historical
  publication FKs do not block preserve cleanup.
- Completed current-head plan stays in `knowledge/plans/` until the 404 handoff
  consumes the G0 dependency; then it can move to archive.

## Verification

- `git diff --check`: passed.
- Focused bootstrap pair on fresh migrated PostgreSQL:
  `go test -v ./bootstrap -run 'TestProductionLifecycleStackUninstallsPreservedDataThroughRealRuntimeAndPostgres|TestReferenceSEOFormalZipUploadTrustEnableRestartDisableUpgradeUninstall' -count=1`: passed.
- Full `./scripts/test.sh` was run once against fresh PostgreSQL
  `sforum_codex_m7_final_20260723`; it passed Go, CompatFarm, protobuf/SDK docs,
  OpenAPI refs, staged extension contracts, production trust, WebSocket proxy,
  Nuxt typecheck, web unit subsets, and product validators through theme
  runtime. It then exposed stale trusted-admin and V3 catalog validators.
- Rerun after targeted fixes:
  - `node tests/validate-trusted-admin-runtime.js`: passed.
  - `cd apps/web && bun run typecheck`: passed.
  - `cd apps/web && bun test tests/extensionSettingsOwnership.test.ts`: passed.
  - `node tests/validate-theme-activation.js`: passed.
  - `node tests/validate-dev-worker-script.js`: passed.
  - `node tests/validate-signal-garden-theme.js`: passed.
  - `node tests/validate-sf-components.js`: passed.
  - `node tests/validate-page-registry-runtime.js`: offline contracts passed;
    live HTTP smoke skipped by script because `PAGE_REGISTRY_API` was unset.
  - `node tests/validate-v3-p0-catalogs.mjs`: passed with 265 routes and 153 UI
    surfaces.
- Earlier G0 verification also passed:
  - `go test -race ./app/Support/Search ./bootstrap`
  - `cd apps/web && bun test`
  - `cd apps/web && bun run build`
  - `ruby scripts/validate-openapi-refs.rb`

## Browser Smoke

- The user's Nuxt dev server was listening on port 3000.
- `curl -i -sS http://127.0.0.1:3000/` returned HTTP 200 with active-theme HTML,
  including `data-sforum-theme="ocean_blue"` and default-theme digest CSS.
- Direct API health probes on ports 9000 and 9002 were not running, so
  authenticated advanced-reply, adjacent search page, and forced theme-failure
  browser flows were not completed in this G0 commit.

## Next

- Start M0 of
  `knowledge/plans/2026-07-22-theme-consistent-public-resource-404.md`.
- Keep the focused 404 implementation limited to public resource-not-found
  behavior; do not expand to 403/429/5xx.

## Open Questions

- None for G0. The remaining browser flows require a running API and suitable
  authenticated session.
