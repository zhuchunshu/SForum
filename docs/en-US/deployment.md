# Deployment

[← English docs home](./README.md)

## Quick install

SForum supports Linux `amd64` and `arm64`; Windows is not supported. The server
needs:

- Docker Engine and Docker Compose `2.24.4` or newer;
- `curl`, `tar`, and the standard `install` command;
- access to GitHub and `ghcr.io`;
- free loopback ports `3000` and `18080`, or alternative ports selected in the
  wizard.

Git is not required. The commands below download and verify the **latest stable
release** bootstrap. It then refreshes the complete matching deploy toolkit:

```sh
(
  set -eu
  mkdir -p sforum
  cd sforum
  bootstrap_dir="$(mktemp -d .sforum-bootstrap.XXXXXX)"
  trap 'rm -rf "$bootstrap_dir"' EXIT HUP INT TERM
  curl -fsSLo "$bootstrap_dir/sforum-bootstrap.sh" \
    https://github.com/zhuchunshu/SForum/releases/latest/download/sforum-bootstrap.sh
  curl -fsSLo "$bootstrap_dir/SHA256SUMS" \
    https://github.com/zhuchunshu/SForum/releases/latest/download/SHA256SUMS
  (
    cd "$bootstrap_dir"
    awk '$2 == "sforum-bootstrap.sh" { print }' SHA256SUMS > sforum-bootstrap.sha256
    test "$(wc -l < sforum-bootstrap.sha256 | tr -d '[:space:]')" = 1
    if command -v sha256sum >/dev/null 2>&1; then
      sha256sum -c sforum-bootstrap.sha256
    else
      shasum -a 256 -c sforum-bootstrap.sha256
    fi
    if command -v gh >/dev/null 2>&1; then
      gh attestation verify sforum-bootstrap.sh --repo zhuchunshu/SForum
    fi
  )
  install -m 0755 "$bootstrap_dir/sforum-bootstrap.sh" ./sforum-bootstrap.sh
  rm -rf "$bootstrap_dir"
  trap - EXIT HUP INT TERM
  ./sforum-bootstrap.sh install
)
```

That URL always points at the latest stable Release bootstrap. It resolves an
immutable target tag, verifies that tag's `sforum-deploy.tar.gz`, and installs
only the matching Compose files, entrypoints, production example, and required
`deploy/` helpers.

### Verify the download (required)

Every Release publishes `SHA256SUMS` (covering every asset) and GitHub build
provenance attestations.

- **Never pipe a remote response directly into a shell**: save the bootstrap
  before downloading its checksum file.
- Verify exactly one `sforum-bootstrap.sh` entry. The bootstrap applies the same
  exact-entry rule to the full deploy bundle.
- The commands run in a `set -eu` subshell, so a failed download, missing or
  duplicate checksum entry, checksum failure, or provenance failure aborts
  before extraction.
- `gh attestation verify` is optional and checks build provenance (requires the
  GitHub CLI, authenticated).

### Channel semantics (stable / prerelease)

- **The default channel is `stable`**: bootstrap install and upgrade resolve
  only the newest **stable** Release.
- **Prereleases are never selected implicitly**: pass `--channel prerelease`,
  or pin an immutable version such as `--version v3.0.0-alpha.N`.
- Whatever the choice, the scripts resolve to a concrete `vX.Y.Z` tag before
  pulling images and run the matching GHCR images; production Compose never
  runs a floating `latest` image tag.
- If no stable Release exists yet, `latest` fails closed with a hint to use
  `--channel prerelease` or an explicit version.

For repeatable deployments, pin an explicit immutable version (replace
`$SFORUM_VERSION` with a real tag, for example the prerelease
`v3.0.0-alpha.13`):

```sh
./sforum-bootstrap.sh install --version $SFORUM_VERSION
./sforum-bootstrap.sh upgrade --version $SFORUM_VERSION
```

## Target shape

- Compose: `web`, `api` with its always-embedded Worker, PostgreSQL, Redis
- Public exposure: **loopback-only** web (and optional API WebSocket ingress); TLS on the host reverse proxy
- Same-origin: browsers hit one domain; Nuxt proxies ordinary `/api/v1/*` HTTP; WebSocket Upgrade may go to the API loopback port

## Configuration

Run `./sforum-bootstrap.sh install` for the recommended path. After refreshing
the verified toolkit, it explains each choice and generates the PostgreSQL,
Redis, session, verification, identity
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

## HTTPS reverse proxy

SForum listens on loopback ports only; TLS is terminated by a host reverse
proxy. See `deploy/caddy/Caddyfile` in the deploy bundle:

- Proxy the Web target `http://127.0.0.1:${WEB_PORT:-3000}` (same-origin HTTP
  API is forwarded by Nuxt);
- optionally route `/api/v1/` WebSocket upgrades directly to the API target
  `http://127.0.0.1:${API_PORT:-18080}`;
- set `TRUST_PROXY` and a precise `TRUSTED_PROXIES` list to trust forwarded
  client IPs — never blanket-trust the internet.

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
- Linux amd64 and arm64 backend bundles containing API, migrator, CLI,
  and the exact protected built-ins extracted from the scanned candidate image;
- fixed-name **`sforum-bootstrap.sh`**, **`sforum-deploy.tar.gz`**, and legacy
  compatibility **`upgrade.sh`** assets;
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
- `ghcr.io/zhuchunshu/sforum-migrate`
- `ghcr.io/zhuchunshu/sforum-web`

Every release supports `linux/amd64` and `linux/arm64`. The simplest
interactive installation accepts Enter for every prompt:

```sh
./sforum-bootstrap.sh install
```

Pin a version explicitly, or accept all recommended defaults non-interactively
(replace `$SFORUM_VERSION` with the real tag):

```sh
./sforum-bootstrap.sh install --version $SFORUM_VERSION --lang en
./sforum-bootstrap.sh install --version $SFORUM_VERSION --lang zh
./sforum-bootstrap.sh install --version $SFORUM_VERSION --lang en --yes
```

Non-interactive prerelease channel (resolves the newest published Release,
including prereleases):

```sh
./sforum-bootstrap.sh install --channel prerelease --lang en --yes
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

Equivalent non-interactive commands (`<VERSION>` must be the resolved concrete
tag, for example `v3.0.0-alpha.13`):

```sh
SFORUM_VERSION=<VERSION>
export SFORUM_VERSION
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

Use the bootstrap to update an existing Compose installation. Every run first
refreshes itself and the target Release's complete toolkit, then hands off to
`upgrade.sh`. The default selects the **latest stable release**:

```sh
./sforum-bootstrap.sh upgrade
./sforum-bootstrap.sh upgrade --version $SFORUM_VERSION
./sforum-bootstrap.sh upgrade --yes
./sforum-bootstrap.sh upgrade --channel prerelease
./sforum-bootstrap.sh upgrade --channel prerelease --yes
```

Prereleases are never selected implicitly: pass `--channel prerelease` or an
explicit immutable tag such as `v3.0.0-alpha.N`. Either way the script resolves
to a concrete tag and runs the matching images; `.deployrc` persists the
resolved tag, never `latest`.

An existing installation does not need a fresh clone. To adopt the bootstrap
once, download it into the existing installation from the **latest stable
Release asset** (never floating `main` content), verify it, then promote it:

```sh
(
  set -eu
  cd /path/to/sforum
  bootstrap_dir="$(mktemp -d .sforum-bootstrap.XXXXXX)"
  trap 'rm -rf "$bootstrap_dir"' EXIT HUP INT TERM
  curl -fsSLo "$bootstrap_dir/sforum-bootstrap.sh" \
    https://github.com/zhuchunshu/SForum/releases/latest/download/sforum-bootstrap.sh
  curl -fsSLo "$bootstrap_dir/SHA256SUMS" \
    https://github.com/zhuchunshu/SForum/releases/latest/download/SHA256SUMS
  (
    cd "$bootstrap_dir"
    awk '$2 == "sforum-bootstrap.sh" { print }' SHA256SUMS > sforum-bootstrap.sha256
    test "$(wc -l < sforum-bootstrap.sha256 | tr -d '[:space:]')" = 1
    if command -v sha256sum >/dev/null 2>&1; then
      sha256sum -c sforum-bootstrap.sha256
    else
      shasum -a 256 -c sforum-bootstrap.sha256
    fi
    if command -v gh >/dev/null 2>&1; then
      gh attestation verify sforum-bootstrap.sh --repo zhuchunshu/SForum
    fi
  )
  install -m 0755 "$bootstrap_dir/sforum-bootstrap.sh" ./sforum-bootstrap.sh
  rm -rf "$bootstrap_dir"
  trap - EXIT HUP INT TERM
  ./sforum-bootstrap.sh upgrade
)
```

The bootstrap preserves `.env.production`, `.deployrc`, `deploy/runtime/`, and
all data volumes. It backs up managed tooling under `.sforum/tooling-backups/`.
For a pinned version, including a prerelease, replace both rolling URLs with:

```sh
https://github.com/zhuchunshu/SForum/releases/download/<TAG>/sforum-bootstrap.sh
https://github.com/zhuchunshu/SForum/releases/download/<TAG>/SHA256SUMS
```

> Historical compatibility note: the `upgrade.sh` shipped with
> `v3.0.0-alpha.12` and earlier cannot start the database compatibility check
> correctly, and its `latest` semantics included prereleases. Use the script
> shipped with `v3.0.0-alpha.13` or newer; installation directories from that
> era must fetch the fixed, version-pinned script first. That immutable tag is
> historical fact and will not be rewritten to main/latest.

```sh
(
  set -eu
  curl -fsSLo upgrade.sh \
    https://raw.githubusercontent.com/zhuchunshu/SForum/v3.0.0-alpha.13/upgrade.sh
  printf '%s  %s\n' ae186e13ca9551014e21ce7f77a7335413791268d97fb0b72f6be9820dedfe13 upgrade.sh > upgrade.sh.sha256
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum -c upgrade.sh.sha256
  else
    shasum -a 256 -c upgrade.sh.sha256
  fi
  chmod 0755 upgrade.sh
  ./upgrade.sh v3.0.0-alpha.13
)
```

Here, `latest` is not a floating container image tag. On the stable channel the
script queries GitHub `/releases/latest` (stable releases only); on the
prerelease channel it queries the Release list and takes the newest published
Release, including prereleases. It resolves the choice to a concrete `vX.Y.Z`
tag. Before changing the installation it prints the current resolved version
and target resolved version and asks whether to use that target. Only an
explicit `--yes` skips version input and confirmation.

The first run against the legacy direct-port topology asks for confirmation to
perform a one-time blue/green ingress conversion. That conversion stops the old
services before starting the stable Caddy ingress and therefore has a short
maintenance window. After conversion, releases with an unchanged database or
only explicitly declared backward-compatible Core migrations keep the active
slot serving. The updater backs up and applies bounded online migrations before
starting and checking the standby API/Web slot. Caddy then switches traffic
atomically before the old slot stops. Existing WebSocket connections may need
to reconnect during the switch.

Every blue/green API slot embeds its Worker. During standby health checks, the
candidate and active APIs may briefly consume from the same queue; River's
database locking owns job claims and retries. After the switch, the updater
stops the old API and never starts or retains a standalone Worker. Before
updating, the script checks both SForum Core and River migrations. Online execution requires
a target migrator with the capability label, an audited `-- +sforum OnlineSafe`
declaration on every pending Core SQL migration, transactional lock and
statement timeouts, and an exact River migration set. A failed online migration
leaves the old slot serving and never switches traffic. Undeclared Core, any
River, and older migrator images are refused; use
`./deploy.sh --version <release>` and accept its backup, migration, and
maintenance window instead. That path recognizes an existing blue/green edge,
backs up first, then stops all old slots before migrating and starting the
direct target services.

## Health checks, logs, and failure recovery

### Health checks

| Check | Address | Use |
| --- | --- | --- |
| Web health | `http://127.0.0.1:${WEB_PORT:-3000}/health` | Container liveness and startup |
| API liveness | `http://127.0.0.1:${API_PORT:-18080}/api/v1/health` | Process liveness |
| API ready | `http://127.0.0.1:${API_PORT:-18080}/api/v1/ready` | PostgreSQL dependency ready (Redis/Meili degrade-ready) |

`deploy.sh` and `upgrade.sh` run these checks before recording success;
`SFORUM_DEPLOY_HEALTH_TIMEOUT_SECONDS` / `SFORUM_UPGRADE_HEALTH_TIMEOUT_SECONDS`
tune the wait.

### Logs

```sh
./deploy.sh --action logs          # follow all service logs
./deploy.sh --action status        # service status
docker compose --env-file .env.production logs -f api web
```

### Failure recovery

- **Failed deployment**: `.deployrc` records `status=recovery_required`, the
  attempted and previous versions, and the backup path. Inspect
  `./deploy.sh --action logs`, fix the cause, and retry.
- **Database rollback**: restores require an explicit
  `SFORUM_CONFIRM_RESTORE=RESTORE` confirmation (see below).
- **Version rollback**: after confirming migration compatibility, redeploy an
  earlier immutable version with `./deploy.sh --version <older-tag>`.
- **Safe Mode / out-of-band recovery**: if an extension blocks startup, use
  `sforum extension disable` / `disable-all` / `quarantine` out of band; see
  the [developer CLI](./development/cli.md) and
  [extensions & themes](./usage/extensions.md).

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
| `api` | Fiber API + extension runtime + embedded River Worker; optional WS ingress |
| `postgres` / `redis` | Durable state / sessions & cache |

The API always owns the Worker in development and production; there is no
environment switch for Worker ownership. HTTP and background work share the
PostgreSQL pool, Redis clients, SettingsLifecycle, SecretStore, and extension
runtime. Updates remove legacy `worker`, `worker-blue`, and `worker-green`
containers by their Compose labels.

Blue/green slot APIs also embed their Workers. River safely coordinates the
brief period with two candidate consumers, and stopping the old API after the
traffic switch also stops its Worker.

Embedded HTTP and background work share `DATABASE_MAX_CONNS`. If a sustained
queue needs its own pool or begins affecting request latency, tune that limit
with PostgreSQL observation or lower the affected queue concurrency instead of
raising River concurrency without a bound.

## Backup and restore

The PostgreSQL helpers under `deploy/scripts/` write backups to a temporary
file and publish a mode-`0600` `.sql` file only after `pg_dump` succeeds. A
failed dump leaves no partial backup that could be mistaken for a valid one.

```sh
./deploy.sh --action backup
```

A restore requires `SFORUM_CONFIRM_RESTORE=RESTORE`. The helper stops the API
and Worker services that were running, restores into a separate temporary
database with `ON_ERROR_STOP=1` and one transaction, validates application
tables, and only then atomically swaps database names. Any SQL error returns a
failure without publishing partially restored data. Only the application
services that were running before the restore are started again.

```sh
SFORUM_CONFIRM_RESTORE=RESTORE ./deploy.sh --action restore
```

Set retention and off-site backup per site policy (a product-level "backup
strategy" question remains open).

## Runtime Memory And Diagnostics

The `/control-panel` resource cards read
`GET /api/v1/admin/overview/resources` and account for CPU, RSS, and available
PSS for the API and its directly owned backend plugin processes. Production
Linux images read `/proc` directly and do not depend on BusyBox `ps` or a host
PID namespace or Docker socket. Requests share
one process-table sample for up to 5 seconds and display a rolling 60-second
median. Systems without PSS support do not fabricate an "effective" value.
Plugin details are ordered from highest to lowest RSS and exclude unrelated
services and orphaned processes.

The API row is labeled as including the Worker, while the Worker row reports
embedded slots and running jobs instead of inventing a standalone Worker MiB
value.

Go pprof is an explicit opt-in, loopback-only diagnostic surface and is disabled
by default:

```sh
# Temporarily enable API diagnostics (the Worker is included)
PPROF_ENABLED=true PPROF_ADDR=127.0.0.1:6060
```

When enabled, use `http://127.0.0.1:6060/debug/pprof/` for API and Worker
profiles. Never publish these ports or proxy them publicly. Remove the
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
- [Operator usage](./usage/README.md)
- Archived long draft: `docs/archive/legacy-root/development-and-deployment.md`
