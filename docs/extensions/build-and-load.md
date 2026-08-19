# Build, digest, and load your extension

> **Handbooks (bilingual):**  
> [中文 · 构建与加载](../zh-CN/extensions/build-and-load.md) ·
> [English · Build, digest, and load](../en-US/extensions/build-and-load.md)

This page walks through the complete author loop for a plugin with an
executable backend (and optionally frontend assets): scaffolding, wiring the Go
module, building the binary and frontend artifacts, refreshing exact digests,
packaging, and loading the package into a running SForum instance.

It complements [authoring-guide.md](./authoring-guide.md) (contracts and
manifest semantics) and [routes.md](./routes.md) (declared HTTP routes).

## The shape of a built package

```text
my-plugin/
├── sforum.extension.json      # Manifest V3
├── README.md
├── backend/
│   ├── go.mod                 # your Go module (see below)
│   ├── go.sum
│   ├── *.go                   # Protocol V2 server (pluginv2.Serve)
│   └── plugin                 # the built executable (backend.entry)
├── frontend/
│   └── admin/
│       ├── src/               # optional Vue authoring source
│       └── dist/              # immutable ESM/CSS loaded in production
└── schemas/
    └── *.json                 # packageFiles kind "schema" (route docs, etc.)
```

`make:plugin` scaffolds the manifest, README, a placeholder `backend/plugin`
stub, and (with `--backend`) a `backend/README.md` with the minimal SDK program.
Add `--vue-admin-page` for a real Vue page workspace plus immediately valid
placeholder output. The scaffold does **not** generate a Go module — you create
`go.mod` yourself.

```bash
cd apps/api
go run ./cmd/sforum make:plugin \
  --id acme.notes --name "Acme Notes" --description "…" \
  --url https://example.com/acme-notes --author-name Acme \
  --backend --no-interaction --out /tmp/acme.notes

# Beginner-friendly Vue dashboard inside the Host admin shell
go run ./cmd/sforum make:plugin \
  --id acme.dashboard --name "Acme Dashboard" --description "…" \
  --url https://example.com/acme-dashboard --author-name Acme \
  --vue-admin-page --no-interaction --out /tmp/acme.dashboard
```

For local experiments the scaffold default lands under `extensions/dev/`
(gitignored, never auto-registered). Use `--out` for a disposable path, or
`--builtin` for a protected built-in package under `extensions/builtin/`.

## 1. Wire the backend Go module

The backend imports the public SDK, which lives in the host module
`github.com/zhuchunshu/sforum/apps/api`. Because that module is not published
as a versioned dependency, every in-repo package uses a `replace` directive to
a local checkout. The toolchain version is anchored by the host `go.mod`
(currently Go 1.26.6).

`backend/go.mod` (in-repo package):

```go
module github.com/zhuchunshu/sforum/extensions/builtin/plugins/sforum-smtp/backend

go 1.26.6

require github.com/zhuchunshu/sforum/apps/api v0.0.0

// Point at the repository checkout containing apps/api.
replace github.com/zhuchunshu/sforum/apps/api => ../../../../../apps/api
```

- Inside the repo, the relative `replace` path must reach the repo root
  `apps/api` from your `backend/` directory (count the `..` levels).
- Outside the repo, clone SForum anywhere and point `replace` at
  `<your-checkout>/apps/api` (absolute or relative path). The host module is
  large, so `go mod tidy`/`go build` download many transitive modules — in a
  mainland-China network, set the local proxy first (see `AGENTS.md`).
- `module` path is your choice for out-of-repo plugins; keep it stable.

Minimal `main.go`:

```go
package main

import pluginv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2"

func main() {
	pluginv2.Serve(pluginv2.NewServer())
}
```

Add route, hook, provider, query, job, or notification handlers to the server
type as needed (see [routes.md](./routes.md) and
[authoring-guide.md](./authoring-guide.md)).

```bash
cd <package-root>/backend
go mod tidy
go build -trimpath -buildvcs=false -ldflags="-s -w" -o plugin .
```

The executable must land at the path declared in `backend.entry`
(`backend/plugin` by convention). `extension test` fails while the binary is
missing (use `--skip-backend-binary` / `--allow-scaffold` during scaffolding).

## 2. Build frontend assets (only when needed)

The Host never compiles uploaded source. Anything the runtime mounts — admin
prebuilt components, public L2 modules, editor L2 nodes — must be **final,
self-contained ESM/CSS bytes** committed into the package at the declared
paths:

- `--prebuilt-settings` scaffolds a working `frontend/admin/dist/settings.mjs`
  + `.css` pair with a required Schema fallback.
- `--vue-admin-page` scaffolds `AdminDashboard.vue`, Vite configuration,
  `@sforum/admin-sdk`, and `@sforum/plugin-ui`. Its page inherits the Host
  sidebar, topbar, tabs, heading, route guard, and permission enforcement.
- Fixture L2 modules (e.g. `sforum-custom-content/frontend/editor/vote.mjs`)
  are hand-written single-file ES modules.
- You may use any bundler (esbuild, rolldown, vite, …) to produce the final
  single-file `.mjs`; the package ships the output, not an executable build
  environment. A hand-written module needs no `package.json`; the Vue scaffold
  includes one only for the author's local build.

Vue scaffold build loop:

```bash
cd <package-root>/frontend/admin
bun install
bun run build
cd ../../..
sforum extension digest --write .
sforum extension validate .
sforum extension test --allow-scaffold .
```

The initial placeholder `dist` files make the new package valid before Bun is
installed. `bun run build` replaces the dashboard output and preserves a
sibling prebuilt settings component. `extension package --exclude-source`
removes `.vue`, `.ts`, Vite config, package metadata, and locks while retaining
the final `.mjs`/`.css` files. Production never runs these build scripts.

Declare each artifact in `packageFiles` with `kind: "frontend"` (or `asset` /
`schema` / `template` as appropriate) plus the exact path. All digests are
refreshed by `extension digest --write`.

## 3. Refresh exact digests

Every packaged file referenced by the manifest — backend executable, frontend
assets, schemas, templates — is bound by SHA-256 in `packageFiles`. After any
file change:

```bash
cd apps/api
go run ./cmd/sforum extension digest --write <package-root>
```

This rewrites inline digests (including inline template digests) and
revalidates the whole package. For Page Registry themes, add the matching
`templates[]` declaration and `packageFiles[]` entry **before** running it;
digest refresh never infers template identity or membership from `theme.json`.

## 4. Validate and contract-test

```bash
go run ./cmd/sforum extension validate <package-root>            # schema + preflight
go run ./cmd/sforum extension test <package-root>                # host contract check
go run ./cmd/sforum extension test --json <package-root>
```

While the backend binary is still missing, `extension test --allow-scaffold`
(same as `--skip-backend-binary`) keeps contract checks green.

## 5. Package for distribution

```bash
go run ./cmd/sforum extension package <package-root> --exclude-source \
  -o /tmp/my-plugin.sforum.zip
```

`--exclude-source` drops Go/JS/TS sources, manifests-in-progress, `testdata`,
and other non-runtime files; the zip keeps the executable, prebuilt frontend,
schemas, and README. Output is `<name>.sforum.zip` + a sibling `.sbom.json`.
Default (no flag) packages everything except `.git`/`node_modules`/`vendor`
and existing zips.

## 6. Load into a running instance

| Way | How | Admin list |
| --- | --- | --- |
| External source collection | Put the package at `<root>/plugins/<id>/` (or `<root>/themes/<id>/` for themes) and set `EXTERNAL_EXTENSION_ROOTS=<root>` in `.env`; restart the API | Snapshot appears automatically; still inert until you trust + enable |
| Upload | Admin → Extensions → Plugins → install the zip | Installed; trust + enable |
| Built-in (in-repo packages) | `make:plugin --builtin`, then restart the API so `SyncBuiltins` registers it | Automatic |
| `extensions/dev/` | Never scanned — use one of the above | — |

`extensions/dev/` is scratch space only. For daily local iteration the
recommended flow is **external source collection**:

```sh
# .env
EXTERNAL_EXTENSION_ROOTS=/abs/path/to/sforum-plugins
```

```text
/abs/path/to/sforum-plugins/
  plugins/acme.notes/          # your package source
```

- API/worker boot scans each root and copies valid packages into
  `EXTENSION_ROOT` as immutable snapshots. Changes become **staged
  candidates**; they never auto-promote or inherit trust.
- Restart the API after package changes so the scan picks up the new snapshot.

### Enable and trust

1. Admin → Extensions → Plugins → open your plugin.
2. **Enable** triggers the trust flow for executable content (backend binary,
   migrations, guards, L2 frontend, declared route registry contributions).
   A `super_admin` confirms the exact artifact (version + digest). The impact
   document lists declared authorities, dependencies, and the Host/Frontend
   contract versions.
3. After enable, declared routes, hooks, and settings become live. Smoke-test
   routes with `curl http://127.0.0.1:8080/<path>` or through the web origin.

### Iteration loop

```text
edit source → build backend/frontend → extension digest --write
→ restart API → admin re-enable (trust again if artifact changed)
```

Any relevant package change invalidates the existing trust grant (executable
artifacts and, per F2.4, frontend trust on digest change). Plan for a trust
confirmation each iteration that changes packaged bytes. If a plugin keeps the
API from booting, recover out-of-band:

```bash
go run ./cmd/sforum extension disable <extension-id>
go run ./cmd/sforum extension disable-all
go run ./cmd/sforum extension list --json
```

## Built-in packages (in-repo)

`extensions/builtin/` is boot-scanned by `SyncBuiltins`. `api-dev.sh` /
`worker-dev.sh` stage it into `storage/builtin-dev` and run
`scripts/build-builtin-plugins.sh`, which builds **only the ids hard-coded in
the script** and refreshes digests in the staging tree (never rewriting
tracked manifests).

A new built-in package is registered by `SyncBuiltins` after restart once it
exists under the active builtin root, but its backend is **not** auto-built
until you add its id to `scripts/build-builtin-plugins.sh` (or build +
`extension digest --write` yourself). Build a builtin backend by hand:

```bash
(cd extensions/builtin/plugins/<id>/backend && go test ./... && go build -o plugin .)
```

Do not edit `storage/builtin-dev/` or `storage/extensions/**` directly; treat
them as generated state.

## Troubleshooting

| Symptom | Cause / fix |
| --- | --- |
| `extension test` fails on backend | Binary missing at `backend.entry`; build it or use `--skip-backend-binary` during scaffolding |
| Enable returns `extension.build_failed` | A declared file digest or template identity is stale; run `extension digest --write` and restart |
| Route returns 404 / probe reports `miss` | Path not claimed by any enabled plugin route; check admin route inspector and conflicts |
| Unsafe route returns CSRF 403 | Browser call missing `X-Csrf-Token` (double-submit against `csrf_` cookie); use the app's `useApiClient` or a Bearer PAT |
| Route times out | Handler exceeded the route `timeoutMs` (default 3 s); stream large work or raise the declaration |
| External root changes not visible | Snapshot is inert and scanned at boot; restart the API and re-enable |
| Trust prompt every iteration | Expected — packaged byte changes invalidate the grant |
| API won't boot after a bad plugin | Out-of-band CLI: `extension disable-all` / `quarantine`, then re-enable selectively |

## Related

- [routes.md](./routes.md) — declared HTTP routes
- [authoring-guide.md](./authoring-guide.md) — contracts, settings, references
- [Manifest V3 catalog](./catalogs/manifest-v3.md)
- [`extensions/README.md`](../../extensions/README.md) — package directory map
- CLI reference: [中文 · 开发者 CLI](../zh-CN/development/cli.md) ·
  [English · Developer CLI](../en-US/development/cli.md)
