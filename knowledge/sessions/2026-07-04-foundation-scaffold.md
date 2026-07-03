# 2026-07-04 Foundation Scaffold

## Changed

- Added `apps/api` with a Go Fiber v3 API entry, worker entry, config loader,
  localization helpers, JSON error shape, and `/api/v1/health`.
- Added `apps/web` with a Nuxt scaffold, Nuxt i18n configuration, default
  Simplified Chinese locale catalog, English catalog, homepage, and `/health`.
- Added Dockerfiles for API and web services.
- Added Docker Compose files for shared, development, and production service
  definitions.
- Added one-command development scripts under `scripts/`.
- Added bilingual interactive `deploy.sh` and PostgreSQL backup/restore helper
  scripts.
- Added `.env.example`, `.env.production.example`, and initial OpenAPI health
  contract.

## Decisions

- Keep the first code slice focused on health checks, localization, and
  operable development/deployment structure.
- Do not add forum domain tables or auth flows until the foundation builds and
  runs cleanly.

## Next

- Resolve Go and Bun dependencies and generate lock files.
- Verify `go test ./...`, Nuxt typecheck/build, and Docker Compose config.
- Add real database connectivity and migrations after the foundation is green.

## Open Questions

- Which production host should be the first Docker Compose target?
- Should English translations be required for every MVP feature before launch?
