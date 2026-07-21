# 2026-07-07 Theme Activation Timeout Fix

## Changed

- Fixed uploaded theme activations getting stuck in `building` when River cancels
  the job context: release status updates now use a short non-canceled context
  so failed builds can still persist `failed` and build logs.
- Disabled River's default 1 minute timeout for `extension.theme_activate`; the
  theme runtime's own build and preview timeouts remain authoritative.
- Captured preview server stdout/stderr into `ThemeRuntime` build logs.
- Let `apps/web` preserve an externally supplied `NUXT_BUILD_DIR` so theme
  releases use their own build directory.

## Verification

- `go test ./app/Jobs/Extensions ./app/Support/ThemeRuntime ./app/Support/Jobs ./bootstrap ./config -count=1`
- `node tests/validate-theme-runtime.js`
- `node tests/validate-theme-activation-progress.js`
- `./scripts/test.sh`
- Local smoke activation completed: release 14 for `sforum.signal-garden`
  became active after a 1m36s River job.
