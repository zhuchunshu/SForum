# 2026-08-28 Built-in Shortcodes M10B Handoff

## Changed

- Reconciled all four Go/TypeScript parser conformance findings and added/used
  shared fixture cases for int64 boundaries, blank protected bodies, lazy
  continuation closings, and list-to-heading canonical newlines.
- Fixed TypeScript protected Markdown canonical export so list-to-heading
  output matches Go/Goldmark byte-for-byte.
- Re-ran focused parser, protected-cache, secret-marker, Host policy,
  lifecycle, subprocess, extension, catalog, OpenAPI, architecture, and
  desktop/mobile SSR checks.
- Added the final-gate report at
  `knowledge/reports/2026-08-28-builtin-shortcodes-m10b-final-gate.md`.

## Evidence

- Shared Go/TypeScript shortcode conformance: PASS.
- Focused Go/Web security and lifecycle suites: PASS.
- Source digest/validate/test, V3 catalog, release baseline, architecture, and
  OpenAPI gates: PASS.
- Real API/Nuxt SSR: `private, no-store`; no raw source/protected marker in
  public JSON, HTML, or Nuxt payload. Playwright 390x844: no horizontal
  overflow.

## Blocking

- SyncBuiltins stages `sforum-shortcodes@1.3.0`, but the running PostgreSQL/API
  active artifact and both runtime catalogs remain `1.2.0` at digest
  `70d43c0e...`. Normal super_admin trust/activation with a real actor session
  is still required; no direct database promotion was used.
- Full Go/Web gates and `scripts/test.sh` retain the environment failures listed
  in the M10B report. The task book must remain active; do not claim release
  completion.

## Next

1. Use a real super_admin browser/API session to activate the staged exact
   `1.3.0` artifact through the normal trust/activation flow.
2. Restart API, verify content/editor catalogs and package/L2/backend digests,
   then exercise disable, Safe Mode, crash, timeout, rollback, uninstall, and
   restart fallbacks against the active artifact.
3. Re-run `./scripts/test.sh`, full Browser actor/leakage matrix, and the final
   unique-marker scan. Only then update the plan to complete and archive it.
