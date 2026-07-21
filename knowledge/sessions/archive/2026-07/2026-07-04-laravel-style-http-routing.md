# 2026-07-04 Laravel-Style HTTP Routing

## Changed

- Added a proposed backend HTTP composition decision.
- Updated the backend module note with the target bootstrap/provider/routes
  structure.
- Updated the architecture document with Laravel-inspired Go/Fiber composition
  guidance.
- Added research notes linking the Laravel routing, provider, lifecycle, and
  middleware concepts to SForum's Go backend.
- Updated the knowledge index navigation and current project state.
- Implemented the first routing refactor slice: `internal/http` now accepts an
  ordered route-provider list, identity has provider/routes files, and
  `cmd/api` delegates application assembly to `internal/bootstrap`.
- Added route-provider, identity provider, and bootstrap tests.

## Decisions

- Route registration now follows the first Laravel-inspired but Go-explicit
  architecture slice.
- `cmd/api/main.go` should become process-only.
- `internal/bootstrap` should own application assembly.
- `internal/http` should own Fiber setup, global/API middleware, system routes,
  error handling, and route-provider interfaces.
- Domain modules should own provider files and route files.
- Avoid package-level route auto-registration, `init` functions, filesystem
  scanning, and a full DI container for now.

## Next

- Add future modules through explicit providers in `internal/bootstrap`.
- Keep module route declarations in focused `routes.go` files.
- Keep OpenAPI paths synchronized with route files as endpoints are added.

## Open Questions

- Whether worker/job provider registration should share the same provider list
  or use a separate worker bootstrap remains open.
