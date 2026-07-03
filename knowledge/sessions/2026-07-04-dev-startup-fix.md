# 2026-07-04 Dev Startup Fix

## Changed

- Verified that `golang:1.26-alpine` and `oven/bun:1.3-alpine` exist on Docker
  Hub; the original `auth.docker.io` EOF was a transient Docker Hub connectivity
  failure rather than a missing image tag.
- Updated the web Dockerfile to copy `bun.lock` and install with
  `bun install --frozen-lockfile` so dependency resolution is deterministic and
  cacheable.
- Refreshed `apps/web/bun.lock` with Bun 1.3.14 after frozen Docker install
  exposed an integrity/download retry case.
- Tightened API and worker Air configs with interrupt shutdown, cleanup, and a
  vendor directory exclusion while preserving the generated SQL exclusion.
- Added Compose rebuild triggers for `go.sum`, `bun.lock`, and
  `nuxt.config.ts`.

## Decisions

- Keep using the existing Docker Compose development path; no base image tag
  change was needed.
- Treat Docker Hub EOF and one-off Bun tarball integrity failures as network or
  cache transients unless they reproduce consistently.

## Next

- Consider pinning direct Nuxt/Bun package versions in `package.json` instead of
  keeping `latest` ranges.
- Consider whether `scripts/dev.sh` should enable Compose Watch by default while
  bind mounts are also present, because Compose currently warns that watched
  paths are already bind-mounted.
- Add `APP_URL`/i18n base URL wiring before SEO tags become part of acceptance
  checks.

## Open Questions

- Should local development prefer bind mounts only, Compose Watch only, or keep
  both paths available?
