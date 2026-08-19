# Developer CLI (`sforum`)

[← Development guide](./README.md)

`sforum` is SForum's developer console (in the spirit of Laravel Artisan),
implemented in `apps/api/cmd/sforum`. It covers scaffolding, extension
validation, exact digests, packaging, contract tests, out-of-band recovery,
account/data maintenance, and fake-data seeding.

Run commands from `apps/api`:

```sh
cd apps/api
go run ./cmd/sforum --help
go run ./cmd/sforum --version
```

## Command map

| Command | Purpose |
| --- | --- |
| `version` | Print build information (version, commit, build time) |
| `make:plugin` | Scaffold a plugin |
| `make:theme` | Scaffold a theme |
| `seed:forum` | Bulk-write fake forum data (small profile) |
| `seed:perf` | Million-scale read-path seed (alias of `seed:forum --profile=perf-1m`) |
| `users:reset-password` | Interactively reset a user's password (alias `user:reset-password`) |
| `revisions backfill` | Backfill the forum content revision ledger in batches |
| `extension build` | Build author frontend assets, refresh digests, and run all package gates |
| `extension validate` | Validate an extension package (includes/template preflight) |
| `extension digest` | Inspect or refresh Manifest V3 `packageFiles` digests |
| `extension test` | Host contract checks (capabilities, events, entrypoints) |
| `extension package` | Build a zip + SBOM stub |
| `extension docs generate` | Generate host docs from the Go catalogs |
| `extension list` | Out-of-band extension recovery state (no plugin code) |
| `extension disable` / `disable-all` | Out-of-band disable of third-party extensions |
| `extension quarantine` | Out-of-band quarantine of an exact built-in/system artifact |
| `extension command list/run` | List / run trusted plugin commands |
| `extension api-lts` | Print Host/Frontend API LTS and shim telemetry |
| `extension system-tier list/upsert/disable` | Out-of-band system tier management (no package code) |
| `dev:cleanup-orphan-plugins` | Stop reparented extension backend plugin processes (safe for live `sforum-api` children) |

> This table is the authoritative command list; adding or removing commands must
> update this table and `docs/zh-CN/development/cli.md`
> (`tests/validate-docs.mjs` verifies coverage).

---

## Build identity: `version`

```sh
go run ./cmd/sforum version
go run ./cmd/sforum --version
```

Prints the unified build identity: SForum version, Git commit, and build time
(the same build arguments injected into the release images).

---

## Scaffolding: `make:plugin` / `make:theme`

Interactive (asks for ID, name, backend stub, etc.):

```sh
go run ./cmd/sforum make:plugin
go run ./cmd/sforum make:theme
```

Non-interactive examples:

```sh
# Local experiment (default → extensions/dev/, gitignored, not in admin list)
go run ./cmd/sforum make:plugin \
  --id acme.demo \
  --name "Acme Demo" \
  --description "…" \
  --backend \
  --no-interaction

# Protected built-in package (→ extensions/builtin/, scanned by SyncBuiltins)
go run ./cmd/sforum make:plugin \
  --id sforum.foo \
  --name "Foo" \
  --description "…" \
  --backend \
  --builtin \
  --no-interaction

# Custom output directory
go run ./cmd/sforum make:plugin \
  --id acme.demo --name "Acme Demo" --description "…" \
  --backend --no-interaction --out /tmp/acme.demo
```

### Common flags

| Flag | Applies to | Notes |
| --- | --- | --- |
| `--id` | both | Stable extension ID, e.g. `acme.demo` |
| `--name` / `--description` | both | Display name and summary |
| `--url` / `--author-*` | both | Website and author info |
| `--out` | both | Output directory; omitted follows dev/builtin rules |
| `--builtin` | both | Write under `extensions/builtin/` instead of `dev/` |
| `--no-interaction` | both | Disable interactive prompts |
| `--backend` | plugin | Generate a `backend/plugin` stub and README |
| `--complex` | plugin | Multi-file manifest (includes + langs + settings shards) |
| `--prebuilt-settings` | both | Author-prebuilt Admin settings component + Schema fallback |
| `--vue-admin-page` | plugin | Vue/Vite admin page workspace using the Plugin UI SDK |
| `--provider-slot` | plugin | Declare a provider slot + `provider_probe` (requires `--backend`) |

### After scaffolding

1. Implement the backend and compile the executable to the Manifest's
   `backend.entry` (usually `backend/plugin`).
2. When `frontend/admin/package.json` exists, run `extension build`; it builds the frontend, refreshes digests, and runs every package gate.
3. Without an author frontend, run `extension digest --write`, `extension validate`, and `extension test` directly.
4. Run `extension package` when distributing.

Third-party plugins use the public SDK (`apps/api/sdk/plugin`) and must **not**
import host business packages such as `app/Models/*`. Full authoring rules:
[plugin authoring guide](../../extensions/authoring-guide.md).

---

## Package helpers: `extension …`

### One-command author build — `build`

```sh
go run ./cmd/sforum extension build <package-root>
go run ./cmd/sforum extension build --allow-scaffold <package-root>
go run ./cmd/sforum extension build --skip-install <package-root>
```

When `frontend/admin/package.json` exists, the command runs `bun install` and
`bun run build`. A `bun.lock` or `bun.lockb` automatically enables
`--frozen-lockfile`. It then refreshes exact digests, performs full Manifest
and template validation, and runs Host contract tests. Packages without an
author frontend skip Bun but still run all three package gates.

- `--skip-install` skips `bun install` when dependencies already exist; it does not skip the build.
- `--allow-scaffold` permits only a missing backend binary during contract tests; other errors still fail.

This is an author-invoked local command and executes the plugin's own
`package.json` scripts. Upload, install, enable, and production runtime never
call it. Production continues to load only exact-manifest `.mjs` / `.css`
artifacts.

### Validate — `validate`

```sh
go run ./cmd/sforum extension validate <package-root>
go run ./cmd/sforum extension validate <package-root> --json   # merged Manifest
```

Loads the package, resolves `includes`, validates Manifest V3, and preflights
page templates for explicit V3 packages.

Page Registry themes follow the fail-closed three-way identity rule: every
`theme.json.pages[].template` path must match exactly one Manifest V3
`templates[]` declaration and one `kind: "template"` `packageFiles[]` entry;
paths and SHA-256 digests must agree. `theme.json` only maps pages and cannot
replace exact-artifact declarations. Missing or stale digests fail validation
and activation with `extension.build_failed`.

### Exact digests — `digest`

Manifest V3 binds executables, frontends, and migrations with `packageFiles`
SHA-256 digests. **Refresh after touching package files**:

```sh
go run ./cmd/sforum extension digest <package-root>           # check only
go run ./cmd/sforum extension digest --write <package-root>   # write back + validate
```

When adding theme templates, declare their identity, contract, ViewModel, and
`packageFiles[]` entries first, then run `digest --write`. The command refreshes
`packageFiles[]` and declared inline template digests but never infers template
identity or file membership from `theme.json`.

### Contract tests — `test`

```sh
go run ./cmd/sforum extension test <package-root>
go run ./cmd/sforum extension test --allow-scaffold <package-root>  # scaffold stage may lack the backend binary
go run ./cmd/sforum extension test --skip-backend-binary <package-root>
go run ./cmd/sforum extension test --json <package-root>
```

Checks capabilities, events, contribution points, providers, jobs, and backend
entrypoints against the host catalog.
`--allow-scaffold` is an alias of `--skip-backend-binary`.

After modifying a built-in theme, run from `apps/api`:

```sh
go run ./cmd/sforum extension digest --write ../../extensions/builtin/themes/<dir>
go run ./cmd/sforum extension validate ../../extensions/builtin/themes/<dir>
go run ./cmd/sforum extension test ../../extensions/builtin/themes/<dir>
```

Then run `./scripts/build-builtin-plugins.sh` at the repo root, restart the API
so `SyncBuiltins` stages the new digests, and activate through the admin UI.
Never edit `storage/builtin-dev/` or `storage/extensions/**` directly.

### Package — `package`

Builds a zip of the extension root plus an SBOM stub.

```sh
# Default: nearly all root files go into the zip
go run ./cmd/sforum extension package <package-root>

# Release package: skip common source/dev files
go run ./cmd/sforum extension package <package-root> --exclude-source

# Custom output path
go run ./cmd/sforum extension package <package-root> \
  --exclude-source \
  -o /tmp/acme.demo.sforum.zip
```

| Behavior | Notes |
| --- | --- |
| Included by default | Nearly all files under the package root |
| Always skipped | `.git/`, `node_modules/`, `vendor/`, existing `*.sforum.zip` |
| `--exclude-source` additionally skips | `*.go`, `go.mod`/`go.sum`, `*.vue`/`*.ts`/`*.tsx`, Sass, source maps, `package.json`/`tsconfig` configs, `testdata/` / `__tests__/`, etc. |
| Usually kept for release | `sforum.extension.json`, manifest shards, `backend/plugin`, prebuilt `.mjs`/`.css`, `README.md` |
| Default output | `<package-root>/<dirname>.sforum.zip` + adjacent `.sbom.json` |

Package validation runs first. Example output:

```text
package	…/acme.demo.sforum.zip
digest	…
sbom	…/acme.demo.sforum.zip.sbom.json
files	12
skipped	8	(source/dev files)   # only with --exclude-source and skips
```

**Important:**

- Runtime installation does **not** need sources, but the default `package`
  does **not** strip them; add `--exclude-source` for distribution, or pack from
  a clean release directory.
- `--exclude-source` is a heuristic filter, not "only the `packageFiles` list".
- Uploaded zips are lazily validated and stored; first enable of executable
  logic requires an exact-artifact trust confirmation. Operator guidance:
  [extensions & themes](../usage/extensions.md).

Recommended release loop:

```sh
# 1. Compile the backend to backend/plugin
# 2. Build frontend assets (if present) + refresh digests + run package gates
go run ./cmd/sforum extension build <package-root>
# 3. Build the release zip
go run ./cmd/sforum extension package <package-root> --exclude-source -o /tmp/my-plugin.sforum.zip
```

### Host docs — `docs generate`

After changing host surfaces (events, capabilities, contribution points,
provider slots, schedules, etc.):

```sh
go run ./cmd/sforum extension docs generate
go run ./cmd/sforum extension docs generate --check   # CI: fail on drift
```

Writes to `docs/extensions/catalogs/` by default (`--out` overrides).

### Plugin commands — `command`

```sh
go run ./cmd/sforum extension command list
go run ./cmd/sforum extension command run <command-id>
go run ./cmd/sforum extension command run <command-id> --input '{"key":"value"}'
go run ./cmd/sforum extension command run <command-id> --input-file /tmp/input.json
```

Requires a usable `DATABASE_URL` (or `--database-url`). Add `--safe-mode` to
enforce Safe Mode on top of `SFORUM_SAFE_MODE`. `list --json` emits
machine-readable output.

### Out-of-band recovery

Without starting the SForum main process or executing plugin code:

```sh
go run ./cmd/sforum extension list
go run ./cmd/sforum extension list --json
go run ./cmd/sforum extension disable <extension-id>
go run ./cmd/sforum extension disable-all
go run ./cmd/sforum extension quarantine <extension-id> \
  --expect-version <exact-version> \
  --expect-digest <64-hex-digest>
```

`quarantine` only isolates an **exact** built-in/system artifact:
`--expect-version` and `--expect-digest` must match the active version to avoid
mis-quarantines.

### API LTS status

```sh
go run ./cmd/sforum extension api-lts
go run ./cmd/sforum extension api-lts --json
```

### System tier — `system-tier`

Manage SystemTier membership (load order and recovery semantics) without
loading package code:

```sh
go run ./cmd/sforum extension system-tier list
go run ./cmd/sforum extension system-tier upsert <extension-id> \
  --role infra --priority 100 --enabled true
go run ./cmd/sforum extension system-tier disable <extension-id>
```

`upsert --role` accepts `auth|cache|storage|infra`; lower `--priority` loads
first. All subcommands accept `--database-url` and `--json`.

---

## Seed data: `seed:forum` and `seed:perf`

```sh
# config.Load does not read .env; export the variables first
set -a; . ../../.env; set +a   # adjust the path from apps/api as needed

go run ./cmd/sforum seed:forum
go run ./cmd/sforum seed:forum --count=100 --users=20 --comments-max=3
go run ./cmd/sforum seed:forum --profile=perf-1m --dry-run
go run ./cmd/sforum seed:forum --database-url 'postgres://…'
```

| Feature | Notes |
| --- | --- |
| Writes | Append-only; repeatable |
| Events | Does not trigger domain events |
| Environment | **development/test only**, never against production |
| Dependency | `DATABASE_URL` or `--database-url` |

Key `seed:forum` flags (see `--help` for the full list):

| Flag | Default | Notes |
| --- | --- | --- |
| `--profile` | `small` | `small` or `perf-1m` |
| `--count` | 1000 | Topic count (perf-1m default 1,000,000) |
| `--users` | 50 | Fake users (perf-1m default 200) |
| `--comments-max` | 5 | Max comments per regular topic (perf-1m default 0) |
| `--categories` | 0 | perf-1m category count (default 20) |
| `--hot-comments` | 0 | perf-1m hot-thread comments (default 50000) |
| `--hot-slug` | empty | perf-1m hot-thread slug (default `perf-hot-thread`) |
| `--category-slug` | empty | small-mode category slug (default `general`) |
| `--batch` | 20 | Log/batch size (perf-1m default 5000) |
| `--dry-run` | false | Print the plan only |
| `--confirm-perf-db` | false | Required for non-dry-run perf-1m writes |
| `--database-url` | env | Override `DATABASE_URL` |

`seed:perf` is an alias of `seed:forum --profile=perf-1m` with the same perf
parameters (`--count` 1,000,000, `--users` 200, `--categories` 20,
`--hot-comments` 50000, `--batch` 5000, …) and also requires
`--confirm-perf-db` to write.

---

## Account maintenance: `users:reset-password`

Interactively resets any user's password (reads only
`DATABASE_URL`/`--database-url`; no other app config is loaded):

```sh
go run ./cmd/sforum users:reset-password
go run ./cmd/sforum users:reset-password --database-url 'postgres://…'
# Alias:
go run ./cmd/sforum user:reset-password
```

---

## Content revisions: `revisions backfill`

Backfills the forum content revision ledger in bounded batches:

```sh
go run ./cmd/sforum revisions backfill
go run ./cmd/sforum revisions backfill --batch 1000 --loop   # until pending=0
go run ./cmd/sforum revisions backfill --database-url 'postgres://…'
```

`--batch` is the posts processed per batch (default 100); `--loop` keeps going
until nothing is pending.

---

## Dev process cleanup: `dev:cleanup-orphan-plugins`

Stops reparented/orphaned SForum extension backend plugin processes (safe for
live `sforum-api` children; PID-allowlist filtered):

```sh
go run ./cmd/sforum dev:cleanup-orphan-plugins
go run ./cmd/sforum dev:cleanup-orphan-plugins --dry-run   # list PIDs, no signals
```

---

## Package layout (quick map)

| Directory | Use | Boot-scanned into admin |
| --- | --- | --- |
| `extensions/dev/` | Local experiments (scaffold default) | No |
| `extensions/builtin/` | Shipped, protected built-ins | Yes (`SyncBuiltins`) |
| `extensions/optional/` | In-repo optional, operator-installed | No |
| `extensions/fixtures/` | CI / contract fixtures | No |
| Runtime `EXTENSION_ROOT` | Operator-uploaded installs | No |
| `EXTERNAL_EXTENSION_ROOTS` | Independent source trees | Yes (lazy snapshot, never auto-enabled) |

Full map: [extensions/README.md](../../../extensions/README.md).
Mechanics and trust model: [plugin authoring guide](../../extensions/authoring-guide.md), [operator extensions guide](../usage/extensions.md).

---

## Related docs

- [API usage](./api.md)
- [Daily workflow](./workflow.md)
- [Environment setup](./setup.md)
- [Testing & gates](./testing.md)
- [Plugin authoring guide](../../extensions/authoring-guide.md)
- [Host API v2](../../extensions/host-api-v2.md)
