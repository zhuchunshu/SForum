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

After dependencies are available, start the local stack with:

```sh
./scripts/dev.sh
```

Useful endpoints:

- Web: `http://127.0.0.1:3000`
- API health via web: `http://127.0.0.1:3000/api/v1/health`
- Web health: `http://127.0.0.1:3000/health`

Only the `web` service publishes a host port. API, PostgreSQL, Redis,
Meilisearch, and other support services communicate on the Docker Compose
network.

## Start Here

1. Read `AGENTS.md`.
2. Read `knowledge/index.md`.
3. Check the latest file under `knowledge/sessions/` when resuming work.
4. Update the knowledge base after making meaningful product, technical, or process decisions.
