# 2026-07-06 Go Run Startup Optimization

## Changed

- Added `./scripts/api-dev.sh` as the recommended host-run API entry. It loads
  the repository `.env`, checks the configured API port, and reports occupied
  ports without stopping user processes.
- Added `./scripts/sforum.sh` for the developer console. It loads `.env`,
  builds `apps/api/tmp/sforum`, and executes the cached binary instead of
  recommending repeated `go run ./cmd/sforum`.
- Moved extension manifest types and validation into the lightweight
  `app/Support/ExtensionManifest` package, while keeping compatibility aliases
  in `app/Models/Extensions`.
- Updated README, `scripts/dev.sh`, and extension module notes to point at the
  new development entries.

## Decisions

- Keep production/API capabilities intact. This pass improves daily developer
  startup behavior and CLI dependency shape without removing storage adapters,
  startup migrations, River, or plugin runtime from the API binary.
- Treat naked `go run` as a diagnostic tool, not the recommended development
  workflow.

## Measurements

- Before changes, `cmd/api` had 586 dependencies, with cold-cache build around
  49.11 seconds and hot-cache build around 1.25 seconds.
- Before changes, `cmd/sforum` had 270 dependencies, with cold-cache build
  around 24.90 seconds and hot-cache `go run ./cmd/sforum --help` around
  0.54 seconds.
- After the manifest split, `cmd/sforum` has 183 dependencies and cold-cache
  build measured around 17.09 seconds.
- After hot-cache warmup, `go build ./cmd/sforum` measured around 0.44 seconds
  and `go build ./cmd/api` measured around 0.72 seconds.

## Next

- If API cold builds remain painful, plan a separate local-development build
  profile or adapter-splitting pass for optional remote storage, startup
  migrations, and plugin runtime dependencies.

## Open Questions

- Whether `cmd/sforum` should eventually be installed into a project-local
  `bin/` directory and reused until source files change.
