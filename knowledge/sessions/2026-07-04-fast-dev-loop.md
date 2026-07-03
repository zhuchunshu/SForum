# 2026-07-04 Fast Dev Loop

## Changed

- Made `scripts/dev.sh` default to `docker compose up` without forced rebuilds
  or automatic Compose Watch.
- Added explicit `--build`/`--rebuild` and `--watch` flags to `scripts/dev.sh`.
- Avoided rewriting `.env` when `NUXT_PUBLIC_API_BASE_URL` already has the
  intended value.
- Lowered API and worker Air debounce from 1000ms to 200ms.
- Added a persistent Go build cache volume for development API and worker
  containers.
- Added Nuxt/Vite watcher ignores for frontend generated output directories.
- Made `bun run build` and `bun run typecheck` use separate Nuxt temporary
  directories.
- Expanded optional Compose Watch ignore rules for frontend and backend
  generated output.
- Disabled Nuxt UI's automatic remote font provider integration to avoid
  build-time network retries while using local/system fonts.
- Replaced empty optional argument arrays in `scripts/dev.sh` with scalar flags
  and a non-empty Compose argument array so macOS Bash with `set -u` does not
  fail on empty array expansion.
- Added `./scripts/dev.sh --print-command` for lightweight command resolution
  checks without starting the long-running Compose stack.
- Updated README, development docs, the knowledge index, and the Compose
  development decision record.

## Decisions

- Default local development should favor bind mounts plus Nuxt/Vite HMR and
  Air, because source trees are already mounted into the containers.
- Forced image rebuilds and Compose Watch should be explicit opt-ins instead of
  part of every startup.
- One-off frontend commands should not write into the dev server's active Nuxt
  temp state when an isolated build directory is enough.
- Nuxt UI remote font integration should stay off until the product explicitly
  chooses hosted web fonts.
- Avoid expanding potentially empty arrays in shell scripts that run under
  macOS's older Bash with `set -u`.

## Next

- Run the stack and compare cold start, warm start, frontend HMR, and Go Air
  rebuild times on a normal local machine.
- If frontend reload remains slow on Docker Desktop, consider a documented
  local-native web mode that keeps only infrastructure in Compose.

## Open Questions

- Should dependency changes trigger a scripted node_modules refresh for the web
  named volume, or should developers continue using `./scripts/dev.sh --build`
  plus volume recreation when dependencies change?
