# Decision: Compose Web-Only Loopback Ports

## Status

Accepted

## Context

The development and production Compose stacks should avoid exposing internal
services on host ports. Only the web entry point should be reachable from the
host, and that port should bind to loopback instead of all interfaces.

## Decision

Only the `web` service publishes a host port in both development and
production:

- Host binding: `127.0.0.1:${WEB_PORT}:3000`.
- Browser-facing API base URL: `/api/v1`.
- Internal API target from Nuxt: `http://api:8080/api/v1`.

The API, worker, PostgreSQL, Redis, Meilisearch, and Mailpit services do not
publish host ports. They communicate through Docker Compose service DNS names.

Production TLS or public-domain routing should be handled by a host-level
reverse proxy or another explicitly managed boundary that forwards to the web
loopback port.

## Consequences

- Local and production defaults expose less attack surface.
- The browser sees a single same-origin web entry point.
- Nuxt owns only a thin API proxy path; Fiber remains the owner of API behavior
  and forum domain logic.
- Operators who need direct database, cache, search, or mailpit access should
  use `docker compose exec`, temporary port forwarding, or a separate
  intentionally configured operations path.

## Update 2026-07-05

Development now runs the frontend and API as host processes. To support that
loop, `compose.dev.yaml` publishes PostgreSQL, Redis, Meilisearch, and Mailpit
to loopback-only host ports. This is a development-only exception; production
Compose still publishes only the web entry point.

## Update 2026-07-16: V3 P6 WebSocket Ingress Exception

Trusted Route Registry WebSocket handlers must receive the original HTTP
Upgrade, but Nitro's HTTP `sendProxy` path does not bridge that transport.
Production therefore has one narrow exception to the web-only rule:

- `api` publishes `127.0.0.1:${API_PORT}:8080` for a host reverse proxy;
- ordinary HTTP and `/api/v1/*` still enter through Nuxt on `WEB_PORT`;
- only WebSocket Upgrade requests bypass Nuxt and enter Fiber directly;
- the edge preserves Host, Origin, cookies, authorization, and Upgrade headers;
- unknown WebSocket paths fail closed in Fiber and do not fall back to Nuxt;
- PostgreSQL, Redis, Meilisearch, workers, and other internal services remain
  unpublished in production.

This exception is loopback-only and does not create a second browser origin.
Operators must configure the API trusted-proxy policy for the effective
Caddy/Docker peer when forwarded client IP is authoritative.
