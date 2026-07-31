# Deployment

[← English docs home](./README.md)

## Quick install

SForum supports Linux `amd64` and `arm64`; Windows is not supported. The server
needs:

- Docker Engine and Docker Compose `2.24.4` or newer;
- `curl` and `tar`;
- access to GitHub and `ghcr.io`;
- free loopback ports `3000` and `18080`, or alternative ports selected in the
  wizard.

Git is not required. Download the stable deployment bundle and start the
interactive installer:

```sh
mkdir -p sforum
curl -fsSL https://github.com/zhuchunshu/SForum/archive/refs/tags/v3.0.0.tar.gz \
  | tar -xz --strip-components=1 -C sforum
cd sforum
./deploy.sh
```

Every prompt has a recommended value. Press Enter when unsure: the installer
uses PostgreSQL and Redis managed by Docker Compose, generates the required
secrets, pulls version-matched GitHub container images, runs migrations, and
waits for the API and Web services to become healthy. The default local address
is `http://127.0.0.1:3000`. Configure the HTTPS reverse proxy described below
before exposing a public site.

Without a version, `deploy.sh` resolves GitHub's latest stable Release to a
concrete tag before deployment; it never runs a floating `latest` image. For a
repeatable deployment, use `./deploy.sh --version v3.0.0`.

## Target shape

- Compose: `web`, `api`, `worker`, PostgreSQL, Redis  
- Public exposure: **loopback-only** web (and optional API WebSocket ingress); TLS on the host reverse proxy  
- Same-origin: browsers hit one domain; Nuxt proxies ordinary `/api/v1/*` HTTP; WebSocket Upgrade may go to the API loopback port  

## Configuration

Run `./deploy.sh` for the recommended path. On first install it explains each
choice and generates the PostgreSQL, Redis, session, verification, identity
HMAC, option-encryption, and Marketplace verifier keys. Press Enter for every
question to get a runnable local configuration. Secrets are never printed and
`.env.production` is written with mode `0600`.

The beginner wizard deliberately uses PostgreSQL and Redis managed by Compose;
no separate database or connection string is required. External services are
an advanced deployment and are not exposed by this version of the wizard.

The default `APP_URL` is for local verification. Public deployments must set a
real HTTPS URL and configure host-level Caddy or Nginx. The generated
`deployment-local-untrusted` Marketplace key keeps unknown indexes locked; use
the official public key and key ID before enabling the official Marketplace.
Never put its private key in the repository or containers.

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

- the `sforum` management CLI for Linux and macOS on amd64 and arm64; Windows
  is not a supported SForum platform;
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

Every release supports `linux/amd64` and `linux/arm64`. The simplest
interactive installation accepts Enter for every prompt:

```sh
./deploy.sh
```

Pin a version explicitly, or accept all recommended defaults non-interactively:

```sh
./deploy.sh --version v3.0.0 --lang en
./deploy.sh --version v3.0.0 --lang zh
./deploy.sh --version v3.0.0 --lang en --yes --action deploy
```

This combines `compose.yaml`, `compose.prod.yaml`, and `compose.release.yaml`.
It pulls all four version-matched GHCR images before changing the database,
then starts the managed PostgreSQL and Redis services. A fresh database skips
backup; an existing install is backed up before the old app stops and the
target migrator runs. Startup always uses `--no-build`. Docker Compose 2.24.4
or newer is required. The script records success only after API readiness, the
Web root, and all five long-running services pass verification.

Before touching the database, the script rejects placeholder secrets and port
conflicts, verifies the three Go image build identities, and acquires a
deployment lock. Migration and startup use `--pull never` after the explicit
pull, so stopping the old app introduces no later registry dependency. A
migration or health failure writes `status=recovery_required`, the attempted
and previous versions, and the backup path to `.deployrc`; it is never reported
as a successful deployment.

Equivalent non-interactive commands:

```sh
export SFORUM_VERSION=v3.0.0
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

`deploy.sh` intentionally uses published images only, so a beginner cannot
accidentally start a source build on the server. Use the development guide and
`scripts/dev.sh` for source development and custom builds.

## Zero-downtime updates

Use `upgrade.sh` to update an existing Compose installation. Enter a release at
the interactive prompt, pass it as a positional argument, or use `--version`.
Pressing Enter selects `latest`:

```sh
./upgrade.sh
./upgrade.sh v3.0.0
./upgrade.sh --version v3.0.0
./upgrade.sh --yes                       # unattended: newest release, no prompts
```

An existing installation does not need a fresh clone. To refresh the stable
updater before entering the interactive update flow:

```sh
cd /path/to/sforum
curl -fsSLo upgrade.sh \
  https://raw.githubusercontent.com/zhuchunshu/SForum/v3.0.0/upgrade.sh
chmod 0755 upgrade.sh
./upgrade.sh
```

Use the `upgrade.sh` shipped with `v3.0.0-alpha.13` or newer. The copy bundled
with `v3.0.0-alpha.12` cannot start the database compatibility check correctly.
If the current installation directory came from that release, fetch the fixed,
version-pinned script before updating:

```sh
curl -fsSLo upgrade.sh \
  https://raw.githubusercontent.com/zhuchunshu/SForum/v3.0.0-alpha.13/upgrade.sh
chmod 0755 upgrade.sh
./upgrade.sh v3.0.0-alpha.13
```

Here, `latest` is not a floating container image tag. The script queries the
GitHub Release list, selects the newest published Release (including
prereleases), and resolves it to a concrete `vX.Y.Z` tag. Before changing the
installation it prints the current resolved version and target resolved version
and asks whether to use that target. Only an explicit `--yes` skips version
input and confirmation. A successful update persists the resolved tag in
`.deployrc`, never `latest`.

The first run against the legacy direct-port topology asks for confirmation to
perform a one-time blue/green ingress conversion. That conversion stops the old
services before starting the stable Caddy ingress and therefore has a short
maintenance window. After conversion, releases without database migrations are
started and checked in the standby API/Web slot before Caddy switches traffic
atomically and the old slot stops, keeping HTTP service continuous. Existing
WebSocket connections may need to reconnect during the switch.

Two Workers are never allowed to consume jobs concurrently. The updater
gracefully stops the old Worker before starting the new one, so queue consumption
pauses briefly, while durable River jobs are not lost. Before updating, the
script checks both SForum Core and River migrations. If either has a pending
migration, the zero-downtime update is refused; use
`./deploy.sh --version <release>` and accept its backup, migration, and
maintenance window instead.

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
`GET /api/v1/admin/overview/resources` and account for CPU, RSS, and available
PSS for the API, an independent Worker, and backend plugin processes owned
directly by them. Production Linux images read `/proc` directly and do not
depend on BusyBox `ps`. Production Compose shares each Worker's PID namespace
with its API, allowing the API to discover and correctly attribute Worker and
plugin processes without a host PID namespace or Docker socket. Requests share
one process-table sample for up to 5 seconds and display a rolling 60-second
median. Systems without PSS support do not fabricate an "effective" value.
Plugin details are ordered from highest to lowest RSS and exclude unrelated
services and orphaned processes.

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
