# Decision: Docker Compose Development And Deployment

## Status

Proposed

## Context

SForum should be easy to start locally and easy to deploy on a production host.
The requested direction is:

- Production uses Docker Compose for one-command deployment.
- Production deployment is controlled by a high-interaction `deploy.sh`.
- `deploy.sh` supports English and Simplified Chinese.
- Development starts all services with one command.
- Development supports hot reload for frontend and backend code.

## Decision

Use Docker Compose as the default development and production orchestration
layer:

- `compose.yaml`: shared service definitions.
- `compose.dev.yaml`: development overrides, source mounts/watch rules, dev
  commands, exposed service ports, and optional local-only tools.
- `compose.prod.yaml`: production overrides, reverse proxy, persistent volumes,
  production commands, and health checks.

Use these command entry points:

- `./scripts/dev.sh` for local development.
- `./deploy.sh` for production installation and deployment.

Development hot reload:

- Nuxt uses Vite HMR.
- Go API and worker use `air`.
- Docker Compose Watch syncs source files and rebuilds containers when
  dependency files change, with a normal `docker compose up --build` fallback.

Production deployment:

- Use Docker Compose on a single host by default.
- Use Caddy as the preferred reverse proxy because automatic TLS keeps the
  default deployment simpler.
- Route `/` to Nuxt and `/api/v1/*` to Fiber on the same origin.
- `deploy.sh` provides bilingual menus for setup, deploy/update, migrations,
  backup, restore, status, logs, restart, stop, and rollback.

## Consequences

- The project remains deployable without Kubernetes or platform-specific
  tooling.
- Development and production share service definitions while still allowing
  environment-specific behavior.
- A bilingual deploy script adds implementation work but makes operations easier
  for both Chinese and English users.
- Rollback can be image-level early on; database migration rollback must remain
  explicit and conservative.
- Compose Watch availability varies by Docker Compose version, so the dev script
  needs a fallback path.

## Follow-up

- Create `compose.yaml`, `compose.dev.yaml`, and `compose.prod.yaml` when app
  scaffolding begins.
- Implement `scripts/dev.sh` after web/api service commands exist.
- Implement `deploy.sh` with language persistence and safety prompts before the
  first production deployment.
- Decide production backup destination and retention policy.

## Update 2026-07-04

Port exposure was narrowed by
`2026-07-04-compose-web-only-loopback-ports.md`: both development and production
Compose stacks should publish only the `web` service on
`127.0.0.1:${WEB_PORT}`. API, database, cache, search, and support services
stay internal to the Compose network.

## Update 2026-07-04 Fast Dev Loop

`scripts/dev.sh` should default to `docker compose up` without forced `--build`
or automatic Compose Watch. The development stack already bind-mounts source
trees, so Nuxt/Vite HMR and Air receive code changes directly. Developers can
opt into `./scripts/dev.sh --build` after Dockerfile, dependency, or toolchain
changes, and `./scripts/dev.sh --watch` when deliberately testing Compose Watch
rules. Frontend build/typecheck commands should use separate Nuxt temporary
directories, and generated output directories should be ignored by development
watchers so one-off commands do not trigger repeated reloads.

## Update 2026-07-05 Dev Startup Speed

During early development, `cmd/worker` is an idle runtime without concrete job
handlers, so the default development script should not start it. Use
`./scripts/dev.sh --worker` when testing jobs. Development Compose persists Go
module cache, Go build cache, and Air temporary binaries in named volumes to
avoid repeated dependency downloads and slow host-bind writes. In development,
the web service may start before the API and SSR should use short startup
timeouts with local defaults for non-critical site options.
