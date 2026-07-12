# Development And Deployment

Status: foundation scaffold started on 2026-07-04. This document records the
target workflow and the current first implementation slice.

## Goals

- One command starts the full local development environment.
- Local development includes PostgreSQL, Redis, Meilisearch, API, and Nuxt web
  app by default, with the worker available through an explicit profile.
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
- Starts PostgreSQL, Redis, Meilisearch, and Mailpit with Docker Compose.
- Stops old Compose-managed `web`, `api`, and `worker` containers before
  starting dependencies, so local processes can own their ports.
- Waits for dependency services to be running or healthy.
- Runs Goose database migrations automatically after PostgreSQL is healthy.
- API and worker processes also run embedded Goose migrations during startup
  when `MIGRATE_ON_STARTUP=true`, so direct process starts stay schema-safe.
- Prints local dependency URLs and the follow-up local frontend/API commands.

Recommended default Compose command inside `scripts/dev.sh`:

```sh
docker compose -f compose.yaml -f compose.dev.yaml up --remove-orphans --wait postgres redis meilisearch mailpit
```

Rebuild the migration image explicitly after Dockerfile, dependency, or
toolchain changes:

```sh
./scripts/dev.sh --build
```

Skip only the dependency-start one-shot migration when deliberately testing
dependency startup. API and worker starts still follow `MIGRATE_ON_STARTUP`:

```sh
./scripts/dev.sh --no-migrate
```

Start the frontend and API locally after dependencies are up:

```sh
cd apps/web && bun run dev
cd apps/api && air
```

### Development Services

- `web`: local Nuxt dev server with Vite HMR, started manually.
- `api`: local Fiber API with Go hot reload, started manually with Air.
- `worker`: local background worker with Go hot reload when job testing needs
  it, started manually with the worker Air config.
- `postgres`: PostgreSQL with a named development volume.
- `redis`: Redis with a named development volume or ephemeral storage.
- `meilisearch`: Meilisearch with a named development volume.
- `mailpit`: optional local SMTP/web inbox for plugin-backed mail testing; the
  core app should not grow direct mail provider logic.
- `minio`: optional local S3-compatible object storage once uploads exist.

### Hot Reload

- Nuxt uses its built-in Vite HMR from `apps/web`.
- Go services use local `air` from `apps/api`.
- Air loads the repository root `.env` through `env_files`, so local API and
  worker processes use the same development configuration.
- `bun run dev` passes `--dotenv ../../.env`, so Nuxt also reads the repository
  root `.env` when started from `apps/web`.
- The web app may start before the API in development. SSR reads
  startup site options with a short timeout and falls back to local defaults so
  the site can open while the API is still compiling.
- Web generated output directories such as `.output`, `.nitro`, coverage, and
  test reports are ignored by Nuxt/Vite watchers.
- `bun run build` and `bun run typecheck` use sibling Nuxt temporary build
  directories (`.nuxt-build` and `.nuxt-typecheck`) so they do not churn the
  dev server's `.nuxt` state.
- After `bun run build`, `bun run preview` starts the generated Nitro server
  directly from `.output/server/index.mjs` with `HOST=0.0.0.0` and
  `--env-file=../../.env`, so local preview uses the same internal API target
  as development. The installed `nuxi preview` command does not expose a host
  flag in this project version, so `nuxt preview --host 0.0.0.0` misreads
  `0.0.0.0` as a root directory.
- Nuxt UI's automatic remote font provider module is disabled until the product
  intentionally chooses web fonts, avoiding build-time network retries.
- Compose Watch is not part of the default development loop now that frontend
  and backend processes run locally.

### Development Ports

Development publishes dependency services to loopback so local frontend and API
processes can connect without joining the Compose network:

- Web: `http://127.0.0.1:3000`
- API via web: `http://127.0.0.1:3000/api/v1`
- API direct: `http://127.0.0.1:8080/api/v1`
- PostgreSQL: `127.0.0.1:15432`
- Redis: `127.0.0.1:16379`
- Meilisearch: `http://127.0.0.1:17700`
- Mailpit SMTP: `127.0.0.1:11025`
- Mailpit UI: `http://127.0.0.1:18025`

The one-shot migration container still uses Compose DNS internally:

- PostgreSQL: `postgres:5432`
- Redis: `redis:6379`
- Meilisearch: `meilisearch:7700`
- Mailpit: `mailpit:1025` and `mailpit:8025`

`WEB_PORT`, `POSTGRES_PORT`, `REDIS_PORT`, `MEILI_PORT`,
`MAILPIT_SMTP_PORT`, and `MAILPIT_UI_PORT` are configurable in `.env`.
Production Compose keeps only the web entry point published.

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
6. Run migrations through the one-shot `migrate` Compose service.
7. Start infrastructure and app services; API and worker startup run the same
   embedded migrations again and should normally no-op.
8. Confirm service status.
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
- `APP_URL` (first-run fallback for runtime `site.url`)
- `MIGRATE_ON_STARTUP=true`
- `WEB_PORT`
- `NUXT_PUBLIC_API_BASE_URL=/api/v1`
- `NUXT_API_INTERNAL_BASE_URL=http://api:8080/api/v1`
- `APP_LOCALE=zh-CN` (first-run fallback for runtime default locale)
- `SUPPORTED_LOCALES=zh-CN,en-US` (first-run fallback for enabled locales)
- `POSTGRES_DB`
- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `REDIS_PASSWORD`
- `MEILI_MASTER_KEY`
- `SESSION_HASH_SECRET` (session signing secret; must be a high-entropy random value in production)
- `CSRF_TRUSTED_ORIGINS` (comma-separated public site origins trusted by the
  CSRF middleware, e.g. `https://forum.example.com`; supports
  `https://*.example.com` wildcard subdomains. When the API runs behind the
  Nuxt reverse proxy the API sees an internal Host while the browser sends the
  public site as `Origin` — the public origin **must** be listed here or every
  unsafe request is rejected. Defaults to the origin derived from `APP_URL`.)
- `TRUST_PROXY` (whether to trust `X-Forwarded-For` / CDN client-IP headers when
  the TCP peer is a trusted proxy. Default: `true` outside production, `false`
  in production — production must set this explicitly.)
- `TRUSTED_PROXIES` (comma-separated proxy IPs or CIDRs that may forward client
  IPs, e.g. `10.0.0.0/8,172.16.0.0/12`. Required in production when
  `TRUST_PROXY=true`. Never set `0.0.0.0/0`.)
- `TRUST_PROXY_PRIVATE` / `TRUST_PROXY_LOOPBACK` (trust RFC1918 private ranges
  and loopback as proxies. Default `true` outside production for Docker/Nuxt;
  default `false` in production.)
- `PROXY_HEADER` (header Fiber uses for `c.IP()`, default `X-Forwarded-For`.
  Business code uses `clientip.FromCtx`, which also reads `CF-Connecting-IP`,
  `True-Client-IP`, and `X-Real-IP`.)
- Mail provider variables or extension settings only when a mail plugin is
  installed. Avoid adding core `SMTP_*` settings for vendor-specific delivery.
- `S3_*` once uploads exist.

> Note: CSRF protection uses a double-submit cookie (`csrf_`) plus an
> `X-Csrf-Token` header, backed by the shared Redis storage. The frontend reads
> the cookie and echoes it on unsafe requests via `useApiClient`. The reverse
> proxy should forward `X-Forwarded-Proto` so HTTPS Origin/Referer fallback
> works correctly. It must also preserve or append `X-Forwarded-For` (and, for
> Cloudflare, pass through `CF-Connecting-IP`) so login/session and
> post/comment IP audit fields record the real client address.

## Health Checks

Distinguish **liveness** from **readiness** for the API process:

| Endpoint | Purpose | Failure means |
| --- | --- | --- |
| `GET /api/v1/health` | Liveness — process is up | Restart the container/process |
| `GET /api/v1/ready` | Readiness — safe to take traffic | Keep process, stop routing traffic |

### API readiness policy (F1)

- **PostgreSQL** is required. Failure → HTTP `503` and `data.ready=false`.
- **Redis** and **Meilisearch** are optional for readiness. Failure is reported
  per component; overall response stays HTTP `200` with `data.status=degraded`
  and `data.ready=true` so forum traffic is not blocked when search/cache is
  briefly unavailable.
- Probe timeout is short (~2s). Prefer `ready` for load-balancer / Compose
  `service_healthy` gates that should wait for the database.

Other surfaces:

- Web: `/health` (Nuxt/Nitro process)
- Worker: no public HTTP probe. The worker (or API when
  `EMBED_WORKER_IN_API=true`) publishes a Redis heartbeat key
  `sforum:worker:heartbeat` (TTL 45s). Admin overview shows stale/unknown when
  the key is missing or older than 45s.

Compose health checks should gate dependent services where practical. Example
API probes:

```yaml
# liveness
test: ["CMD", "curl", "-fsS", "http://127.0.0.1:8080/api/v1/health"]
# readiness (prefer this before attaching traffic)
test: ["CMD", "curl", "-fsS", "http://127.0.0.1:8080/api/v1/ready"]
```

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

### Queue Configuration

Worker queue concurrency is configured by environment variables:

- `WORKER_DATABASE_MAX_CONNS`: PostgreSQL pool size for the worker process.
- `WORKER_SHUTDOWN_TIMEOUT`: graceful worker stop timeout, such as `30s`.
- `JOB_QUEUE_CRITICAL_WORKERS`: workers for small consistency-critical jobs.
- `JOB_QUEUE_DEFAULT_WORKERS`: workers for ordinary background jobs.
- `JOB_QUEUE_SEARCH_WORKERS`: workers for Meilisearch indexing and rebuilds.
- `JOB_QUEUE_MAIL_WORKERS`: workers reserved for plugin-backed outbound mail
  delivery.
- `JOB_QUEUE_NOTIFICATIONS_WORKERS`: workers reserved for plugin-backed
  notification fanout.
- `JOB_QUEUE_MAINTENANCE_WORKERS`: workers for cleanup and scheduled
  maintenance jobs.

### Application Migrations

SForum schema migrations use Goose SQL files under `apps/api/database/migrations`.
The files are embedded into the API, worker, and `sforum-migrate` binaries, so
runtime containers do not need a separate migrations directory.

By default, `MIGRATE_ON_STARTUP=true` makes API and worker processes run
PostgreSQL migrations once during startup before module services open their
normal connection pools. The migrator uses Goose's PostgreSQL table lock so
parallel API/worker starts serialize safely. Set `MIGRATE_ON_STARTUP=false`
only for special maintenance cases where an operator is running migrations
separately and wants startup to skip the check.

`deploy.sh` still runs the one-shot `migrate` service after backup and before
updating app services so migration failures surface before traffic reaches a
new release. Startup migration then acts as an idempotent safety net.

### River Migrations

River owns its internal job tables and runs its own migration line. Before a
worker can consume durable jobs in a new database, run:

```sh
cd apps/api
go install github.com/riverqueue/river/cmd/river@latest
river migrate-up --database-url "$DATABASE_URL" --line main
```

Use `river migrate-list --database-url "$DATABASE_URL" --line main` to inspect
which River migrations have been applied. Do not use `river migrate-down` in
production unless an operator has already accepted that it may remove River job
tables and queued jobs.

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
