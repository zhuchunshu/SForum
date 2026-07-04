# Decision: Local Dev Dependencies And Host Processes

## Status

Accepted

## Context

The previous development script started the full Compose stack, including the
Nuxt web service and Go API. During active frontend and backend work, the
preferred loop is to run `bun run dev` and `air` directly on the host while
keeping shared services reproducible through Docker Compose.

## Decision

`./scripts/dev.sh` starts only local development dependencies:

- PostgreSQL
- Redis
- Meilisearch
- Mailpit

The script stops old Compose-managed `web`, `api`, and `worker` containers
before starting dependencies, waits for the dependency services, and runs Goose
migrations through the existing one-shot `migrate` service by default.

Frontend and backend hot reload run as host processes:

- `cd apps/web && bun run dev`
- `cd apps/api && air`

Nuxt and Air both load the repository root `.env` so developers do not need to
manually source environment variables in each terminal.

## Consequences

- The local feedback loop uses native Nuxt/Vite and Air behavior directly.
- Development dependency services publish loopback-only host ports.
- Production Compose remains stricter: only the web entry point is published,
  and internal services stay private on the Compose network.
- `--watch` and `--worker` are no longer `scripts/dev.sh` options. Worker tests
  should start the worker locally when needed.
