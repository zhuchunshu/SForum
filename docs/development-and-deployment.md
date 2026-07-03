# Development And Deployment

Status: foundation scaffold started on 2026-07-04. This document records the
target workflow and the current first implementation slice.

## Goals

- One command starts the full local development environment.
- Local development includes PostgreSQL, Redis, Meilisearch, API, worker, and
  Nuxt web app.
- Frontend and backend code changes reload automatically.
- Production deployment uses Docker Compose and a high-interaction `deploy.sh`
  script.
- `deploy.sh` supports both English and Chinese prompts.
- Application runtime defaults to Simplified Chinese (`zh-CN`) and declares
  supported product locales explicitly.
- Production operations should be understandable for a small team without
  Kubernetes.

## Target Files

```text
.
|-- deploy.sh
|-- compose.yaml
|-- compose.dev.yaml
|-- compose.prod.yaml
|-- .env.example
|-- .env.production.example
|-- deploy/
|   |-- caddy/
|   |   `-- Caddyfile
|   |-- scripts/
|   |   |-- backup-postgres.sh
|   |   |-- restore-postgres.sh
|   |   `-- wait-for-health.sh
|   `-- volumes/
|       `-- README.md
`-- scripts/
    |-- dev.sh
    |-- dev-down.sh
    |-- logs.sh
    `-- test.sh
```

The first foundation scaffold creates these files. They should continue to
evolve as the app gains real migrations, releases, and backups.

## Local Development

Primary command:

```sh
./scripts/dev.sh
```

Expected behavior:

- Verifies Docker and Docker Compose are available.
- Creates `.env` from `.env.example` when missing.
- Ensures default local locale settings are `APP_LOCALE=zh-CN` and
  `SUPPORTED_LOCALES=zh-CN,en-US`.
- Starts all required services with Docker Compose.
- Runs database migrations automatically or prompts when destructive changes
  are possible.
- Streams combined logs by default.
- Prints local URLs, service ports, and useful follow-up commands.

Recommended Compose command inside `scripts/dev.sh`:

```sh
docker compose -f compose.yaml -f compose.dev.yaml up --build --watch
```

If Compose Watch is unavailable, fall back to:

```sh
docker compose -f compose.yaml -f compose.dev.yaml up --build
```

### Development Services

- `web`: Nuxt dev server with Vite HMR.
- `api`: Fiber API with Go hot reload.
- `worker`: background worker with Go hot reload.
- `postgres`: PostgreSQL with a named development volume.
- `redis`: Redis with a named development volume or ephemeral storage.
- `meilisearch`: Meilisearch with a named development volume.
- `mailpit`: optional local SMTP/web inbox for email testing.
- `minio`: optional local S3-compatible object storage once uploads exist.

### Hot Reload

- Nuxt uses its built-in Vite HMR.
- Go services should use `air` in development containers.
- Compose Watch should sync source files into containers and rebuild when
  dependency files change.

Suggested watch rules:

- Sync `apps/web/app`, `apps/web/server`, `apps/web/public`, and
  `apps/web/nuxt.config.ts` into the `web` container.
- Rebuild `web` when `apps/web/package.json`, `bun.lock`, or Nuxt config
  dependency settings change.
- Sync `apps/api` Go source into `api` and `worker` containers.
- Rebuild Go containers when `apps/api/go.mod` or `apps/api/go.sum` changes.

### Development Ports

Default local ports:

- Web: `http://localhost:3000`
- API: `http://localhost:18080`
- PostgreSQL: `localhost:15432`
- Redis: `localhost:16379`
- Meilisearch: `http://localhost:17700`
- Mailpit: `http://localhost:18025`
- MinIO: `http://localhost:9001`

Ports should be configurable in `.env`.

## Production Deployment

Primary command:

```sh
./deploy.sh
```

Production should use Docker Compose with pinned image tags or locally built
release images.

Recommended Compose command for deploy:

```sh
docker compose -f compose.yaml -f compose.prod.yaml up -d --build
```

### Production Services

- `proxy`: Caddy or another reverse proxy with automatic TLS.
- `web`: Nuxt production server.
- `api`: Fiber production API.
- `worker`: background worker.
- `postgres`: PostgreSQL with a persistent named volume or host-mounted data
  directory.
- `redis`: Redis with persistence settings appropriate for sessions/cache.
- `meilisearch`: Meilisearch with persistent data.

Optional later services:

- `backup`: scheduled backup sidecar or host cron wrapper.
- `mailpit`: disabled in production.
- `minio`: production only if self-hosted uploads are required; otherwise use
  hosted S3/R2-compatible storage.

### Reverse Proxy

Use same-origin routing in production:

- `/` routes to `web`.
- `/api/v1/*` routes to `api`.
- Health endpoints remain accessible to the proxy and deploy script.

Caddy is the preferred default because it keeps TLS automation simple for a
single-server Docker Compose deployment.

## `deploy.sh` Interaction Design

`deploy.sh` should be a real operations console, not only a wrapper around
`docker compose up`.

### Language Selection

On first run:

```text
Choose language / 选择语言:
1) English
2) 简体中文
```

Persist the choice in a deploy config file such as `.deployrc`. Also allow:

```sh
./deploy.sh --lang en
./deploy.sh --lang zh
```

All menus, warnings, confirmations, and success/failure messages should use the
selected language.

### Main Menu

Required menu actions:

- Install or first-time setup.
- Deploy or update.
- Run migrations.
- Create PostgreSQL backup.
- Restore PostgreSQL backup.
- View service status.
- View logs.
- Restart services.
- Stop services.
- Roll back to previous image tag or Compose release when available.
- Renew/reload proxy certificates when the proxy supports it.
- Exit.

### Preflight Checks

Before deploy or update:

- Docker is installed and running.
- Docker Compose plugin is available.
- Required ports are free or already owned by this deployment.
- `.env.production` exists and required secrets are set.
- Domain DNS appears to point to the host when TLS is enabled.
- Available disk space is above a configured threshold.
- PostgreSQL backup is created before migrations by default.

### Safety Rules

- Never overwrite `.env.production` without confirmation.
- Never restore a database backup without a typed confirmation.
- Never run destructive cleanup without listing affected volumes/images first.
- Always print the exact Compose project name and environment file in use.
- Keep backups outside containers under a documented host path.

### Deployment Flow

Default update flow:

1. Select language.
2. Load `.env.production`.
3. Run preflight checks.
4. Pull or build images.
5. Create PostgreSQL backup.
6. Start infrastructure services.
7. Run migrations.
8. Start app services.
9. Run health checks.
10. Print URLs, service status, and rollback hint.

### Rollback

Rollback should be supported once image tagging exists:

- Keep the previously deployed image tag in `.deployrc` or a release metadata
  file.
- Re-run Compose with the previous tag.
- Do not automatically roll back database migrations unless a specific
  reversible migration plan exists.

## Environment Files

Use separate files:

- `.env`: local development.
- `.env.example`: documented local defaults.
- `.env.production`: production secrets and host settings.
- `.env.production.example`: required production keys without secrets.

Important production variables:

- `APP_ENV=production`
- `APP_URL`
- `API_BASE_URL`
- `APP_LOCALE=zh-CN`
- `SUPPORTED_LOCALES=zh-CN,en-US`
- `POSTGRES_DB`
- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `REDIS_PASSWORD`
- `MEILI_MASTER_KEY`
- `SESSION_SECRET`
- `CSRF_SECRET`
- `SMTP_*`
- `S3_*` once uploads exist.

## Health Checks

Each app service should expose a health endpoint:

- API: `/api/v1/health`
- Web: `/health`
- Worker: internal command or heartbeat endpoint if exposed only on the Docker
  network.

Compose health checks should gate dependent services where practical.

## Backup Strategy

Minimum viable production backup:

- `deploy.sh` can create an on-demand PostgreSQL dump.
- Backups include timestamp, git commit or image tag, and database name.
- The deploy script creates a backup before migrations.
- Documentation clearly states where backups are stored.

Later:

- Add scheduled backups.
- Push encrypted backup archives to S3/R2-compatible storage.
- Add restore drills to release process.

## Sources Checked

- Docker Compose Watch: https://docs.docker.com/compose/how-tos/file-watch/
- Docker Compose profiles: https://docs.docker.com/compose/how-tos/profiles/
- Docker Compose production guide: https://docs.docker.com/compose/how-tos/production/
- Air live reload for Go: https://github.com/air-verse/air
