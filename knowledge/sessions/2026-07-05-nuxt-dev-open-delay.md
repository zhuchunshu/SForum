# 2026-07-05 Nuxt Dev Open Delay

## Changed

- Moved frontend `build` and `typecheck` build directories from nested
  `.nuxt/build` and `.nuxt/typecheck` to sibling `.nuxt-build` and
  `.nuxt-typecheck`.
- Added the new sibling directories to Nuxt top-level ignores, Vite watch
  ignores, and `.gitignore`.
- Added a `scripts/dev.sh` warning for local `HTTP_PORT` and
  `NUXT_API_INTERNAL_BASE_URL` mismatches.
- Fixed the current local ignored `.env` so `NUXT_API_INTERNAL_BASE_URL` points
  at the running local API on `127.0.0.1:8081` instead of the unrelated service
  occupying `127.0.0.1:8080`.
- Opened Docker Desktop, started development dependencies with
  `./scripts/dev.sh`, and ran migrations so registration-related API calls can
  use PostgreSQL again.

## Findings

- The running Nuxt dev server returned the Nuxt 503 loading screen:
  `.nuxt/dist directory has been removed. Restarting Nuxt...`
- `apps/web/.nuxt/dist` existed but was empty, so the active dev server could
  not serve the real app until restarted/regenerated.
- The API was healthy at `http://127.0.0.1:8081/api/v1/health`, while
  `http://127.0.0.1:8080/api/v1/health` returned a gunicorn 404 from another
  local service.
- Before Docker dependencies were started, DB-backed API routes such as
  `/api/v1/auth/registration-status` and `/api/v1/web-options` returned 500
  even though the frontend page could render with fallbacks.

## Next

- Keep `NUXT_API_INTERNAL_BASE_URL` aligned with `HTTP_PORT` when changing the
  local API port.

## Verification

- `bun run typecheck` in `apps/web`.
- `bash -n scripts/dev.sh`.
- `./scripts/dev.sh --print-command`.
- `./scripts/dev.sh`.
- `curl -i http://127.0.0.1:3000/register` returned 200 and rendered the
  registration page.
- `curl -i http://127.0.0.1:3000/api/v1/auth/registration-status` returned
  200 with `nextUserIsInitialSuperAdmin: true`.
- `curl -i http://127.0.0.1:3000/api/v1/web-options` returned 200 with
  `site.name`.

## Open Questions

- None.
