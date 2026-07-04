# 2026-07-05 Dev Startup Speed

## Changed

- Added a persistent Go module cache volume for development API, worker, and
  migration containers.
- Moved Air temporary binaries for API and worker into Docker named volumes.
- Made the development worker opt-in with `./scripts/dev.sh --worker` while the
  worker runtime is still idle.
- Removed the development web service's startup dependency on the API by moving
  the production `web -> api` dependency into `compose.prod.yaml`.
- Added a short startup timeout for the global web-options fetch so Nuxt SSR can
  render with fallback site options while the API is compiling or reloading.
- Ignored Go `_test.go` edits in Air so test-only changes do not restart API or
  worker services.

## Decisions

- Keep migrations automatic before API startup.
- Keep the worker available, but do not pay its hot-reload cost by default
  until real job handlers are wired.
- Prefer Docker named volumes for Go module/build caches and hot-reload
  binaries instead of repeatedly downloading modules or writing binaries through
  host bind mounts.

## Next

- After the next clean restart, compare default `./scripts/dev.sh` startup with
  `./scripts/dev.sh --worker`.
- When concrete job handlers are added, revisit whether the worker should return
  to the default development stack.

## Verification

- `bash -n scripts/dev.sh`
- `./scripts/dev.sh --print-command`
- `./scripts/dev.sh --worker --print-command`
- `docker compose -f compose.yaml -f compose.dev.yaml config --services`
- `COMPOSE_PROFILES=worker docker compose -f compose.yaml -f compose.dev.yaml config --services`
- `docker compose -f compose.yaml -f compose.dev.yaml up --remove-orphans --no-build -d`
- `docker compose -f compose.yaml -f compose.dev.yaml exec -T web bun run typecheck`
- `curl -sS --max-time 5 http://127.0.0.1:3000/health`
- `curl -sS --max-time 5 http://127.0.0.1:3000/api/v1/health`

Observed API restart after the new cache volumes rebuilt in about two seconds
without repeated `go: downloading ...` output. The default running service list
no longer includes `worker`.

## Open Questions

- Should dependency file changes trigger an automatic dev image rebuild prompt
  instead of relying on `./scripts/dev.sh --build`?
