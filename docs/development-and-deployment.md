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
- Only the `web` service publishes a host port, and it binds to
  `127.0.0.1:${WEB_PORT}`. Other services communicate only on the Docker
  Compose network.
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

`deploy/caddy/Caddyfile` is an optional host-level reverse proxy example. It is
not used as a default Docker Compose service because the Compose stack should
publish only `web`.

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
- Starts all required services with Docker Compose, reusing existing
  development images by default.
- Runs database migrations automatically or prompts when destructive changes
  are possible.
- Streams combined logs by default.
- Prints local web URLs, internal service names, and useful follow-up commands.

Recommended default Compose command inside `scripts/dev.sh`:

```sh
docker compose -f compose.yaml -f compose.dev.yaml up
```

Rebuild development images explicitly after Dockerfile, dependency, or toolchain
changes:

```sh
docker compose -f compose.yaml -f compose.dev.yaml up --build
```

Enable Compose Watch only when deliberately testing watch rules:

```sh
./scripts/dev.sh --watch
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
- Source bind mounts feed code changes into containers by default.
- Web generated output directories such as `.output`, `.nitro`, coverage, and
  test reports are ignored by Nuxt/Vite watchers and optional Compose Watch.
- `bun run build` and `bun run typecheck` use separate Nuxt temporary build
  directories so they do not churn the dev server's `.nuxt` state.
- Nuxt UI's automatic remote font provider module is disabled until the product
  intentionally chooses web fonts, avoiding build-time network retries.
- Compose Watch is optional and should not be enabled by default while the same
  source trees are bind-mounted.

Suggested watch rules:

- Sync `apps/web/app`, `apps/web/server`, `apps/web/public`, and
  `apps/web/nuxt.config.ts` into the `web` container.
- Ignore frontend generated output such as `.nuxt`, `.output`, `.nitro`,
  `.vite`, `.cache`, `dist`, `coverage`, `playwright-report`, and
  `test-results`.
- Rebuild `web` when `apps/web/package.json`, `bun.lock`, or Nuxt config
  dependency settings change.
- Sync `apps/api` Go source into `api` and `worker` containers.
- Ignore backend generated output such as `tmp`, `bin`, coverage files, and Go
  test binaries.
- Rebuild Go containers when `apps/api/go.mod` or `apps/api/go.sum` changes.

### Development Ports

Only `web` publishes a host port, bound to loopback:

- Web: `http://127.0.0.1:3000`
- API via web: `http://127.0.0.1:3000/api/v1`

Internal services use Compose DNS names and are not reachable directly from the
host by default:

- API: `api:8080`
- PostgreSQL: `postgres:5432`
- Redis: `redis:6379`
- Meilisearch: `meilisearch:7700`
- Mailpit: `mailpit:1025` and `mailpit:8025`

`WEB_PORT` should be configurable in `.env`. Internal ports should stay stable
unless the service image or application runtime changes.

## Production Deployment

Primary command:

```sh
./deploy.sh
```

Production should use Docker Compose with pinned image tags or locally built
release images.

The Compose stack should publish only the `web` service to
`127.0.0.1:${WEB_PORT}`. If the site needs public TLS or a public domain, run a
host-level reverse proxy outside this Compose stack and point it at that
loopback web port.

Recommended Compose command for deploy:

```sh
docker compose -f compose.yaml -f compose.prod.yaml up -d --build
```

### Production Services

- `web`: Nuxt production server, the only service with a host port. It proxies
  same-origin `/api/v1/*` traffic to the API over the Compose network.
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
- `/api/v1/*` is received by `web` and proxied internally to `api:8080`.
- Health endpoints remain accessible to the proxy and deploy script.

The Docker Compose stack should not publish a separate proxy service. A public
reverse proxy, if used, should live on the host or in another explicitly managed
network boundary and forward to `127.0.0.1:${WEB_PORT}`.

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
- `WEB_PORT` is free or already owned by this deployment on `127.0.0.1`.
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
- `WEB_PORT`
- `NUXT_PUBLIC_API_BASE_URL=/api/v1`
- `NUXT_API_INTERNAL_BASE_URL=http://api:8080/api/v1`
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

## Jobs And Worker Runtime

The `worker` service is the durable background job runtime. It should consume
River-backed PostgreSQL queues through `apps/api/app/Support/Jobs`.

Initial named queues are `critical`, `default`, `search`, `mail`,
`notifications`, and `maintenance`. Production deployments may run more than one
worker container, but queue concurrency and PostgreSQL pool sizing must be
configured deliberately before scaling out.

Redis should not be treated as the first durable queue store. It remains the
session, cache, and rate-limit backing service unless a later design introduces
a clearly non-critical fast-lane queue.

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
