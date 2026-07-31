# Deployment

[← English docs home](./README.md)

## Target shape

- Compose: `web`, `api`, `worker`, PostgreSQL, Redis  
- Public exposure: **loopback-only** web (and optional API WebSocket ingress); TLS on the host reverse proxy  
- Same-origin: browsers hit one domain; Nuxt proxies ordinary `/api/v1/*` HTTP; WebSocket Upgrade may go to the API loopback port  

## Configuration

```sh
cp .env.production.example .env.production
```

`deploy.sh` and the database helpers tighten `.env.production` to mode `0600`
so other users on the host cannot read production secrets.

Change at least:

- `POSTGRES_PASSWORD` / `DATABASE_URL`  
- `REDIS_PASSWORD`  
- `APP_URL` / `APP_DOMAIN`  
- session/CSRF-related secrets per example comments  
- `MARKETPLACE_ED25519_PUBLIC_KEY_HEX` (the 32-byte, 64-hex-character public key for the signed Marketplace index) and its `MARKETPLACE_ED25519_KEY_ID`. Production/staging API startup fails before readiness when the key is missing; never store the private key in the repository or container.

## Maintainer releases

Maintainers create a version from a clean `main` worktree synchronized with
`origin/main`:

```sh
./scripts/release.sh
```

The helper returns immediately after pushing the version tag while GitHub
Actions continues the release. Use `--wait` only when the current terminal must
track the result; `--no-wait` is the default. Interactive releases accept a
one-line operator-written highlight; pressing Enter uses generated notes only.
Use `./scripts/release.sh 2.8.0 --notes-file /tmp/release-notes.md` for multi-line
Markdown. Manual highlights are prepended to GitHub's complete generated notes.
On GitHub, Release waits for and
reuses the exact commit's existing `main` push CI result. Image build, scan, and
promotion begin only after that run succeeds, without repeating the repository
gate for the tag. After image scan and Compose smoke pass, the GitHub Release
also publishes:

- the `sforum` management CLI for Linux, macOS, and Windows on amd64 and arm64;
- Linux amd64 and arm64 backend bundles containing API, worker, migrator, CLI,
  and the exact protected built-ins extracted from the scanned candidate image;
- `SHA256SUMS` for every archive plus GitHub build provenance attestations.

The Linux backend bundle does not contain the Nuxt web runtime, PostgreSQL, or
Redis, so it is not a complete site installer. The four version-matched Docker
images below remain the recommended production path. Verify a downloaded asset
with `gh attestation verify <file> --repo zhuchunshu/SForum` and its entry in
`SHA256SUMS`.

## Deploy entrypoint

### Published images (recommended)

Stable releases publish these images to GitHub Container Registry:

- `ghcr.io/zhuchunshu/sforum-api`
- `ghcr.io/zhuchunshu/sforum-worker`
- `ghcr.io/zhuchunshu/sforum-migrate`
- `ghcr.io/zhuchunshu/sforum-web`

Every release supports `linux/amd64` and `linux/arm64`. Pin a complete version
in production instead of deploying `latest`:

```sh
./deploy.sh --version v2.8.0
./deploy.sh --version v2.8.0 --lang en
./deploy.sh --version v2.8.0 --lang zh
```

This combines `compose.yaml`, `compose.prod.yaml`, and `compose.release.yaml`.
It pulls the selected version before backup, migration, and startup, so the
API, worker, migration command, and web app use one release. Docker Compose
2.24.4 or newer is required for the `!reset` override. After startup,
`deploy.sh` waits for both API `/api/v1/ready` and the Web root; a timeout fails
the deployment instead of printing a completion URL.

Equivalent non-interactive commands:

```sh
export SFORUM_VERSION=v2.8.0
docker compose --env-file .env.production \
  -f compose.yaml -f compose.prod.yaml -f compose.release.yaml pull
docker compose --env-file .env.production \
  -f compose.yaml -f compose.prod.yaml -f compose.release.yaml run --rm -T migrate
docker compose --env-file .env.production \
  -f compose.yaml -f compose.prod.yaml -f compose.release.yaml up -d --no-build
```

All four GHCR packages must be public and linked to this repository. Before a
GitHub Release is created, the workflow pulls every versioned image with an
empty Docker credential directory; any anonymous pull failure blocks the
release. On the first publication the workflow may stop at this gate: make all
four packages Public, then rerun the failed jobs. Publication still uses the
repository `GITHUB_TOKEN`; no long-lived registry credential is required.

### Source builds

Development versions and source customizations can keep using the original
entrypoint:

```sh
./deploy.sh
./deploy.sh --lang en
./deploy.sh --lang zh
```

Language choice persists in `.deployrc`. Compose equivalent:

```sh
docker compose --env-file .env.production -f compose.yaml -f compose.prod.yaml up -d --build
```

## Ports (production examples)

| Use | Default |
| --- | --- |
| Web (loopback) | `127.0.0.1:${WEB_PORT:-3000}` |
| API WS ingress (loopback) | `127.0.0.1:${API_PORT:-18080}` |

See `deploy/caddy/Caddyfile`. For client IPs, set `TRUST_PROXY` and a precise `TRUSTED_PROXIES` list—never blanket-trust the internet.

## Process roles

| Service | Role |
| --- | --- |
| `web` | Nuxt production output; same-origin HTTP API proxy |
| `api` | Fiber API + extension runtime; optional WS ingress |
| `worker` | River consumer (split in production) |
| `postgres` / `redis` | Durable state / sessions & cache |

Do not treat `EMBED_WORKER_IN_API` as a production default.

## Backup and restore

The PostgreSQL helpers under `deploy/scripts/` write backups to a temporary
file and publish a mode-`0600` `.sql` file only after `pg_dump` succeeds. A
failed dump leaves no partial backup that could be mistaken for a valid one.

A restore requires `SFORUM_CONFIRM_RESTORE=RESTORE`. The helper stops the API
and Worker services that were running, restores into a separate temporary
database with `ON_ERROR_STOP=1` and one transaction, validates application
tables, and only then atomically swaps database names. Any SQL error returns a
failure without publishing partially restored data. Only the application
services that were running before the restore are started again.

## Runtime Memory And Diagnostics

The `/control-panel` resource cards read
`GET /api/v1/admin/overview/resources` and account for the API, an independent
Worker, and backend plugin processes owned directly by those processes. Requests
share one process-table sample for up to 5 seconds and display a rolling
60-second median. Linux can also expose complete process-family PSS; systems such
as macOS do not fabricate an "effective" PSS value. Plugin details are ordered
from highest to lowest RSS and exclude unrelated services and orphaned processes.

Development embeds the Worker in the API by default. The API row is labeled as
including the Worker, while the Worker row reports embedded slots and running
jobs instead of inventing a standalone Worker MiB value. With
`EMBED_WORKER_IN_API=false`, a standalone Worker is measured separately.

Go pprof is an explicit opt-in, loopback-only diagnostic surface and is disabled
by default:

```sh
# Temporarily enable API diagnostics (an embedded Worker is included)
PPROF_ENABLED=true PPROF_ADDR=127.0.0.1:6060

# Enable a separate profile only for a standalone Worker
WORKER_PPROF_ENABLED=true WORKER_PPROF_ADDR=127.0.0.1:6061
```

When enabled, use `http://127.0.0.1:6060/debug/pprof/` (or `6061` for the
standalone Worker). Never publish these ports or proxy them publicly. Remove the
flags and restart after profiling. `GOMEMLIMIT` can provide a Go runtime soft
heap target, for example `GOMEMLIMIT=512MiB`; it is not a hard RSS limit for
plugins or the container, which need their own resource limits.

## After go-live

1. Create the first super admin on empty DBs  
2. Site name, HTTPS URL, SMTP  
3. Attachment storage  
4. Search: site search is enough; Meili only if needed  
5. Extension trust policy and Safe Mode drill  
6. Health: `/health`, `/api/v1/health`, `/api/v1/ready`  

## Related

- [Getting started](./getting-started.md)  
- Archived long draft: `docs/archive/legacy-root/development-and-deployment.md`  
