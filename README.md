# SForum

SForum is a forum project in the foundation stage.

The repository contains project documentation, collaboration rules, a lightweight knowledge base, and the first runnable application scaffold.

## Repository Map

- `AGENTS.md` - working rules for AI agents and contributors.
- `docs/` - product, architecture, and planning documents.
- `knowledge/` - project memory for decisions, module notes, and session handoffs.
- `apps/` - application source code for `web` and `api`.
- `contracts/` - API contracts such as OpenAPI.
- `compose*.yaml` - planned Docker Compose files for development and production.
- `deploy.sh` - bilingual production deployment entry point.
- `tests/` - future tests.
- `assets/` - future static or design assets.

## Development

Start the local dependency services first:

```sh
./scripts/dev.sh
```

The script starts PostgreSQL, Redis, and Mailpit with Docker (Meilisearch is optional via compose profile `search`)
Compose, waits for healthy services, and runs database migrations by default.
It does not start the frontend or API.

Run the frontend and API locally in separate terminals:

```sh
cd apps/web && bun run dev
./scripts/api-dev.sh
```

`bun run dev` starts Nuxt directly. Public themes activate synchronously through
the Page Registry (L0 CSS + L1 templates), without rebuilding Nuxt or restarting
Nitro. Extension settings use the host Schema renderer by default; explicitly
trusted prebuilt components load from immutable digest URLs without a host build.
`bun run preview` serves the fixed `.output` build for local production checks.

In development, the API process embeds the background worker, so queued jobs run
when `air` starts the API. To mimic the
production split, disable `EMBED_WORKER_IN_API` and run the worker manually:

```sh
./scripts/worker-dev.sh
```

After Dockerfile or dependency changes, rebuild the migration image explicitly:

```sh
./scripts/dev.sh --build
```

Frontend build and typecheck scripts use separate Nuxt temporary directories so
`bun run build` and `bun run typecheck` do not churn the dev server's `.nuxt`
state or trigger noisy reloads.

Useful endpoints:

- Web: `http://127.0.0.1:3000`
- API liveness: `http://127.0.0.1:3000/api/v1/health`
- API readiness: `http://127.0.0.1:3000/api/v1/ready` (PG required; Redis/Meili degraded-ready)
- Web health: `http://127.0.0.1:3000/health`
- Meilisearch health (optional profile `search`): `http://127.0.0.1:17700/health`
- Mailpit UI: `http://127.0.0.1:18025`

Development dependency services publish loopback-only host ports so locally
started `air` and Nuxt can connect to them. Production Compose keeps internal
services private and publishes only the web entry point.

Production web starts the Nuxt output directly. Public theme activate/switch is
runtime-only (Page Registry + skin CSS). Extension settings use host-rendered
Schema/Actions or explicitly trusted prebuilt components loaded by immutable
digest; operators never build extension frontend code. Do not use
`bun run preview` as the production web server.

## Start Here

1. Read `AGENTS.md`.
2. Read `knowledge/index.md`.
3. Check the latest file under `knowledge/sessions/` when resuming work.
4. Update the knowledge base after making meaningful product, technical, or process decisions.
