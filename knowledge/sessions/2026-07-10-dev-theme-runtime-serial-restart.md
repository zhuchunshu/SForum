# 2026-07-10 Dev Theme Runtime Serial Restart

## Changed

- Replaced local blue-green Nuxt dev switching with one serial process
  lifecycle.
- Added explicit selection state, process-group waiting, latest-change
  convergence, identity-safe exits, crash recovery, and last-known-good
  rollback.
- Hardened cleanup so macOS `EPERM` existence probes remain in the bounded wait,
  unexpected leader exits clean descendants before recovery, and cleanup
  failures cannot launch a rollback or external restart in parallel.
- Clear only the shared Nuxt build directory's persisted Nitro route responses
  before each local child launch so SWR HTML from the previous theme cannot
  survive a serial restart; Vite and dependency caches remain intact.
- Kept production Nitro blue-green switching and the `current.json` contract
  unchanged.

## Root Cause

- Nuxt 4.4.8 rejected the parallel candidate with
  `Another Nuxt dev server is already running`; bypassing the lock would still
  share generated files, caches, and HMR ports.
- On macOS, `kill(-pgid, 0)` can briefly return `EPERM` after `SIGTERM` succeeds
  and before the child is reaped. Treating that probe as fatal aborted the first
  real serial-switch smoke test.
- Reusing one Nuxt build directory also reused `cache/nitro/routes` SWR output.
  Generated routes correctly selected Signal Garden, but the server returned a
  byte-identical cached default homepage until that route cache was invalidated.

## Decisions

- Local theme activation may have a short Nuxt restart outage.
- Do not set `NUXT_IGNORE_LOCK` and do not create parallel local build slots.

## Verification

- `cd apps/web && bun test tests/devThemeLifecycle.test.ts tests/devRuntimeStartup.test.ts tests/themeProxy.test.ts`
- `node tests/validate-theme-runtime.js`
- `cd apps/web && bun run typecheck`
- Isolated default -> Signal Garden -> default smoke on port 4317 returned HTTP
  200 for all three states and changed markers from `sforum-home-page` to
  `signal-garden-home` / `sg-hero`, then back to `sforum-home-page`, without the
  Nuxt lock error or `EPERM`.
- `./scripts/test.sh`
