# Developer CLI (`sforum`)

[← Development](./README.md)

`sforum` is SForum’s developer console (Artisan-style tooling) at
`apps/api/cmd/sforum`. Use it to scaffold packages, validate manifests, refresh
exact digests, build release zips, run host contract checks, recover extensions
out of band, and seed forum data.

Run commands from `apps/api` by default:

```sh
cd apps/api
go run ./cmd/sforum --help
```

## Command map

| Command | Purpose |
| --- | --- |
| `make:plugin` | Scaffold a plugin package |
| `make:theme` | Scaffold a theme package |
| `seed:forum` | Append fake forum data |
| `extension validate` | Validate a package (includes + template preflight) |
| `extension digest` | Inspect or refresh Manifest V3 `packageFiles` digests |
| `extension test` | Host contract checks (capabilities, events, entry, …) |
| `extension package` | Build zip + SBOM stub |
| `extension docs generate` | Generate host catalog docs from Go catalogs |
| `extension command list/run` | List / run trusted plugin commands |
| `extension list` | Recovery inventory without starting plugin code |
| `extension disable` / `disable-all` | Out-of-band disable third-party extensions |
| `extension api-lts` | Print Host/Frontend API LTS and shim telemetry |

---

## Scaffolding: `make:plugin` / `make:theme`

Interactive prompts:

```sh
go run ./cmd/sforum make:plugin
go run ./cmd/sforum make:theme
```

Non-interactive examples:

```sh
# Local experiment (default → extensions/dev/, gitignored, not admin-listed)
go run ./cmd/sforum make:plugin \
  --id acme.demo \
  --name "Acme Demo" \
  --description "Example plugin" \
  --backend \
  --no-interaction

# Protected built-in (→ extensions/builtin/, SyncBuiltins picks it up)
go run ./cmd/sforum make:plugin \
  --id sforum.foo \
  --name "Foo" \
  --description "…" \
  --backend \
  --builtin \
  --no-interaction

# Explicit output path
go run ./cmd/sforum make:plugin \
  --id acme.demo --name "Acme Demo" --description "…" \
  --backend --no-interaction --out /tmp/acme.demo
```

### Common flags

| Flag | Applies to | Meaning |
| --- | --- | --- |
| `--id` | both | Stable extension id, e.g. `acme.demo` |
| `--name` / `--description` | both | Display name and short blurb |
| `--url` / `--author-*` | both | Site and author metadata |
| `--out` | both | Output directory; omit to use dev/builtin rules |
| `--builtin` | both | Write under `extensions/builtin/` instead of `dev/` |
| `--no-interaction` | both | Disable interactive prompts |
| `--backend` | plugin | Stub `backend/plugin` + README |
| `--complex` | plugin | Multi-file manifest (includes + langs + settings shards) |
| `--prebuilt-settings` | both | Author-prebuilt admin settings component + Schema fallback |
| `--provider-slot` | plugin | Declare provider slot + `provider_probe` (requires `--backend`) |

### After scaffolding

1. Implement the backend and build the executable to the Manifest `backend.entry` path (usually `backend/plugin`).  
2. Refresh exact digests with `extension digest --write`.  
3. Run `extension validate` / `extension test`.  
4. Ship with `extension package` when you need a zip.

Third-party plugins should use the public SDK (`apps/api/sdk/plugin`) and **must not** import host business packages such as `app/Models/*`. Full authoring rules: [Plugin authoring guide](../../extensions/authoring-guide.md).

---

## Package helpers: `extension …`

### Validate — `validate`

```sh
go run ./cmd/sforum extension validate <package-root>
go run ./cmd/sforum extension validate <package-root> --json   # merged Manifest JSON
```

Loads the package, resolves `includes`, validates Manifest V3, and preflights page templates for explicit V3 packages.

### Exact digests — `digest`

Manifest V3 binds executables, frontends, migrations, and more via SHA-256 in `packageFiles`. **Refresh after any packaged file change:**

```sh
go run ./cmd/sforum extension digest <package-root>           # inspect
go run ./cmd/sforum extension digest --write <package-root>   # rewrite root manifest + revalidate
```

### Contract tests — `test`

```sh
go run ./cmd/sforum extension test <package-root>
go run ./cmd/sforum extension test --allow-scaffold <package-root>  # scaffold: backend binary optional
go run ./cmd/sforum extension test --skip-backend-binary <package-root>
go run ./cmd/sforum extension test --json <package-root>
```

Checks capabilities, events, contribution points, providers, jobs, backend entry, and related host catalogs.  
`--allow-scaffold` is an alias of `--skip-backend-binary`.

### Package — `package`

Build a zip of the extension root plus an SBOM stub.

```sh
# Default: almost every file under the root
go run ./cmd/sforum extension package <package-root>

# Release: omit common source / dev files
go run ./cmd/sforum extension package <package-root> --exclude-source

# Explicit output path
go run ./cmd/sforum extension package <package-root> \
  --exclude-source \
  -o /tmp/acme.demo.sforum.zip
```

| Behavior | Detail |
| --- | --- |
| Default include | Nearly all files under the package root |
| Always skipped | `.git/`, `node_modules/`, `vendor/`, existing `*.sforum.zip` |
| Extra with `--exclude-source` | `*.go`, `go.mod`/`go.sum`, `*.vue`/`*.ts`/`*.tsx`, Sass, source maps, `package.json`/`tsconfig` and similar, `testdata/` / `__tests__/` … |
| Typically kept for release | `sforum.extension.json`, manifest shards, `backend/plugin`, prebuilt `.mjs`/`.css`, `README.md` |
| Default output | `<package-root>/<dirname>.sforum.zip` + sidecar `.sbom.json` |

Validation runs before packing. Example output:

```text
package	…/acme.demo.sforum.zip
digest	…
sbom	…/acme.demo.sforum.zip.sbom.json
files	12
skipped	8	(source/dev files)   # only with --exclude-source when files were skipped
```

**Important:**

- Operators do **not** need source code at install time, but default `package` **does not strip sources**. Use `--exclude-source` for distribution, or stage a clean release directory first.  
- `--exclude-source` is heuristic filtering, not “only files listed in `packageFiles`”.  
- Upload is inert (validate + store only). First executable enable requires exact-artifact trust confirmation. Operator view: [Extensions & themes](../usage/extensions.md).

Recommended release loop:

```sh
# 1. Build backend into backend/plugin
# 2. Refresh digests
go run ./cmd/sforum extension digest --write <package-root>
# 3. Validate + contract checks
go run ./cmd/sforum extension validate <package-root>
go run ./cmd/sforum extension test <package-root>
# 4. Release zip
go run ./cmd/sforum extension package <package-root> --exclude-source -o /tmp/my-plugin.sforum.zip
```

### Host docs — `docs generate`

After host surface changes (events, capabilities, contribution points, provider slots, schedules, …):

```sh
go run ./cmd/sforum extension docs generate
go run ./cmd/sforum extension docs generate --check   # CI: fail on drift vs committed docs
```

Default output: `docs/extensions/catalogs/` (override with `--out`).

### Plugin commands — `command`

```sh
go run ./cmd/sforum extension command list
go run ./cmd/sforum extension command run <command-id>
```

Requires a usable `DATABASE_URL` (or `--database-url`). Optional `--safe-mode`.

### Out-of-band recovery

Without starting the main SForum process or plugin code:

```sh
go run ./cmd/sforum extension list
go run ./cmd/sforum extension disable <extension-id>
go run ./cmd/sforum extension disable-all
```

### API LTS status

```sh
go run ./cmd/sforum extension api-lts
go run ./cmd/sforum extension api-lts --json
```

---

## Seed data: `seed:forum`

```sh
# config.Load does not read .env — export vars first
set -a; . ../../.env; set +a   # adjust path if needed when cwd is apps/api

go run ./cmd/sforum seed:forum
go run ./cmd/sforum seed:forum --count=100 --users=20 --comments-max=3
go run ./cmd/sforum seed:forum --dry-run
go run ./cmd/sforum seed:forum --database-url 'postgres://…'
```

| Trait | Detail |
| --- | --- |
| Writes | Append-only; safe to re-run |
| Events | Does not fire domain events |
| Environment | **Dev/test only** — never against production |
| Dependency | `DATABASE_URL` or `--database-url` |

---

## Package layout (quick map)

| Directory | Role | Boot-scanned into admin |
| --- | --- | --- |
| `extensions/dev/` | Local experiments (scaffold default) | No |
| `extensions/builtin/` | Protected product built-ins | Yes (`SyncBuiltins`) |
| `extensions/optional/` | In-repo optional packages (install required) | No |
| `extensions/fixtures/` | CI / contract fixtures | No |
| Runtime `EXTENSION_ROOT` | Operator upload storage | No |

Full map: [extensions/README.md](../../../extensions/README.md).  
Mechanism and trust: [Authoring guide](../../extensions/authoring-guide.md), [operator extensions](../usage/extensions.md).

---

## Related docs

- [Daily workflow](./workflow.md)  
- [Environment setup](./setup.md)  
- [Testing & gates](./testing.md)  
- [Plugin authoring guide](../../extensions/authoring-guide.md)  
- [Host API v2](../../extensions/host-api-v2.md)  
