# 2026-07-04 Laravel-Style API Directory Migration

## Changed

- Migrated the Go API module path from
  `github.com/inkedus/sforum/apps/api` to
  `github.com/zhuchunshu/sforum/apps/api`.
- Migrated backend code from `apps/api/internal/*` to a Laravel-style project
  shape:
  - `bootstrap` for API runtime assembly.
  - `config` for typed configuration.
  - `app/Http` for the Fiber HTTP kernel.
  - `app/Http/Controllers/Identity` for identity controllers and routes.
  - `app/Providers` for provider wiring.
  - `app/Models/Identity` for identity domain types, services, policies, and
    stores.
  - `app/Support/*` for infrastructure wrappers.
  - `database/*` for migrations, handwritten SQL, and generated `sqlc` code.
- Updated `sqlc.yaml`, migration command paths, API imports, architecture docs,
  and knowledge module notes for the new layout.

## Decisions

- Keep the implementation Go-native and explicit even though directory names
  now intentionally resemble Laravel.
- Use `Controller` naming in `app/Http/Controllers/*` instead of the previous
  generic `Handler` naming.
- Do not recreate Laravel's dynamic container; providers remain explicit Go
  constructors collected by bootstrap.

## Verification

- `go test ./...` passed in `apps/api`.

## Next

- Future backend code should use the Laravel-style directories and the
  `github.com/zhuchunshu/sforum/apps/api` module path.
- When jobs are implemented, use `app/Support/Jobs` for the shared River
  wrapper and `app/Jobs/*` for application job handlers.

## Open Questions

- Whether future domain packages should stay under `app/Models/<Domain>` or
  split into more specific Laravel-style directories such as `app/Services`,
  `app/Policies`, and `app/Repositories` once a domain grows.
