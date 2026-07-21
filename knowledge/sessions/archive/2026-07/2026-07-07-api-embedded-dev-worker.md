# 2026-07-07 API Embedded Development Worker

## Changed

- Added `EMBED_WORKER_IN_API`, defaulting to `true` for `APP_ENV=development`
  and `false` for production.
- `bootstrap.NewAPI` now starts an embedded River worker when the flag is
  enabled, using the API PostgreSQL pool and the same theme activation worker
  registration as `cmd/worker`.
- Updated development docs and `.env` examples so local `air` API startup also
  consumes background jobs such as uploaded theme activation.

## Decisions

- Keep production API and worker processes separate by default because theme
  builds can consume CPU and memory, and background work should not compete with
  HTTP request handling in production.
- Keep `scripts/worker-dev.sh` as an optional split-process tool for debugging
  production-like behavior.

## Verification

- `cd apps/api && go test ./config ./bootstrap`
- `node tests/validate-dev-worker-script.js`
- `./scripts/test.sh`
