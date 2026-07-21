# Extensions layout

Repository-owned packages live under `extensions/`. **Only built-in packages
are scanned into the admin extension list on API/worker boot.** Everything else
is scaffold, optional ship-with-repo content, test fixtures, or operator
upload storage.

Author-facing detail: [`docs/extensions/authoring-guide.md`](../docs/extensions/authoring-guide.md)
(section **Where to put your package**).

## Directory map

| Path | Git | Boot scan (`SyncBuiltins`) | Dev auto-build | How it reaches admin list |
| --- | --- | --- | --- | --- |
| [`builtin/`](./builtin/) | Tracked | **Yes** — `BUILTIN_EXTENSION_ROOT` | Partial — see below | Automatic on API/worker start |
| [`dev/`](./dev/) | **Ignored** (`.gitignore`) | No | No | Never, unless you move/copy it |
| [`optional/`](./optional/) | Tracked | No | No | Operator install (upload or copy into `EXTENSION_ROOT`) |
| [`fixtures/`](./fixtures/) | Tracked | No | No | Tests / CLI only — not product install |
| Runtime `EXTENSION_ROOT` (default `storage/extensions`) | Local data | No (upload path) | No | Admin upload / install flow |

Layout under each source tree:

```text
extensions/
  builtin/
    plugins/<package-dir>/   # protected built-ins (mail, storage, site search, …)
    themes/<package-dir>/    # default + bundled themes
  dev/
    plugins/<package-dir>/   # make:plugin default output
    themes/<package-dir>/    # make:theme default output
  optional/
    plugins/<package-dir>/   # ship-with-repo, not SyncBuiltins
  fixtures/
    plugins/…  themes/…      # CI and contract locks
```

## What “auto scan / build / register” actually means

### Auto register (admin extension list)

On API and worker boot, Host runs `SyncBuiltins`:

1. Read `BUILTIN_EXTENSION_ROOT` (default relative: `extensions/builtin`;
   Compose: `/app/extensions/builtin`).
2. Walk `plugins/*` and `themes/*` subdirectories.
3. Load each package via `LoadPackage` (`sforum.extension.json` + `includes`).
4. Snapshot into the extension store so admin **Themes / Plugins** lists show them.

**Only packages under that builtin root are registered this way.** Putting a
package in `dev/`, `optional/`, or `fixtures/` does nothing at boot.

### Auto build (local dev)

`./scripts/api-dev.sh` and `./scripts/worker-dev.sh` call
[`scripts/build-builtin-plugins.sh`](../scripts/build-builtin-plugins.sh):

1. rsync/copy `extensions/builtin/` → `storage/builtin-dev/` (gitignored).
2. `go build` **only the plugins hard-coded in that script** (today:
   `sforum-smtp`, `sforum-content-policy`, `sforum-storage-fs`,
   `sforum-search-site`).
3. Refresh Manifest V3 digests **in staging only** (source tree digests stay
   stable for git).
4. Export `BUILTIN_EXTENSION_ROOT=…/storage/builtin-dev` for Air.

Implications:

- A new directory under `extensions/builtin/plugins/` is **scanned** after
  restart (if it is present in the tree pointed at by
  `BUILTIN_EXTENSION_ROOT`).
- Its backend binary is **not** built unless you add it to
  `build-builtin-plugins.sh` or build + digest by hand.
- Themes under `builtin/themes/` are copied into staging; they are buildless
  runtime packages (no go-plugin binary).

### Manual path for third-party / optional plugins

1. Develop anywhere (often `extensions/dev/…` or an external folder).
2. Package and install via admin upload, **or** place an exact package under
   `EXTENSION_ROOT` using the product install flow.
3. Trust / enable in admin (executable enable still requires the normal
   super_admin trust path).

See also [`optional/README.md`](./optional/README.md).

## Scaffold defaults

```bash
# From repo root / apps/api
go run ./cmd/sforum make:plugin --id acme.demo ...   # → extensions/dev/plugins/acme.demo
go run ./cmd/sforum make:theme  --id acme.skin ...   # → extensions/dev/themes/acme.skin

go run ./cmd/sforum make:plugin --id sforum.foo ... --builtin
# → extensions/builtin/plugins/sforum.foo

go run ./cmd/sforum make:plugin --id acme.demo ... --out /path/to/package
```

`extensions/dev` is local-only (gitignored). Use it for experiments; promote to
`builtin` only when the package should ship as a protected built-in, or package
for upload when it is operator-installed software.

## Recommended choices

| Goal | Put the package here |
| --- | --- |
| Protected core vertical that must appear after every boot | `extensions/builtin/plugins/<dir>/` (+ extend `build-builtin-plugins.sh` if it has a Go backend) |
| Bundled default / alternate theme | `extensions/builtin/themes/<dir>/` |
| Local experiment, not committed | `extensions/dev/{plugins,themes}/` (default scaffold) |
| Ship in git but operator must install | `extensions/optional/plugins/<dir>/` |
| Lock a contract for CI | `extensions/fixtures/{plugins,themes}/` |
| Production site install | Admin upload → `EXTENSION_ROOT` |

## Related docs

- [Plugin authoring guide](../docs/extensions/authoring-guide.md)
- [Optional extensions](./optional/README.md)
- [Fixtures](./fixtures/README.md)
- [Dev scaffolds](./dev/README.md)
- Knowledge module: `knowledge/modules/extensions.md`
