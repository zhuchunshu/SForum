# 2026-07-04 Session Handoff

## Changed

- Updated development and production Compose files so only `web` publishes a
  host port, bound as `127.0.0.1:${WEB_PORT}:3000`.
- Removed host port publishing for API, PostgreSQL, Redis, Meilisearch, and
  Mailpit.
- Added a Nuxt `/api/v1/*` proxy route to reach the Fiber API internally at
  `api:8080`.
- Updated development and production environment examples for same-origin API
  access.
- Updated `scripts/dev.sh` and `deploy.sh` to enforce or report the new
  web-only entry point.

## Decisions

- Browser-facing API calls should use `/api/v1`; Nuxt proxies those requests to
  the API over the Compose network.
- Public TLS or domain routing should sit outside this Compose stack and target
  the loopback web port.

## Next

- Verify the stack after future API routes are added, especially proxy behavior
  for non-GET requests and auth cookies.

## Open Questions

- Should operations docs include a sanctioned temporary tunnel/port-forward flow
  for database, search, or Mailpit inspection?
