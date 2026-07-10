# 2026-07-10 Dev Theme Runtime Serial Restart

## Changed

- Replaced local blue-green Nuxt dev switching with one serial process
  lifecycle.
- Added explicit selection state, process-group waiting, latest-change
  convergence, identity-safe exits, crash recovery, and last-known-good
  rollback.
- Kept production Nitro blue-green switching and the `current.json` contract
  unchanged.

## Root Cause

- Nuxt 4.4.8 rejected the parallel candidate with
  `Another Nuxt dev server is already running`; bypassing the lock would still
  share generated files, caches, and HMR ports.

## Decisions

- Local theme activation may have a short Nuxt restart outage.
- Do not set `NUXT_IGNORE_LOCK` and do not create parallel local build slots.

## Verification

- `cd apps/web && bun test tests/devThemeLifecycle.test.ts tests/devRuntimeStartup.test.ts tests/themeProxy.test.ts`
- `node tests/validate-theme-runtime.js`
- `cd apps/web && bun run typecheck`
- Isolated default -> uploaded -> default smoke on an alternate port.
- `./scripts/test.sh`
