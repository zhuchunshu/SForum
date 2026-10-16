# Migrating From SForum-old To V3 Contracts

This page maps the SForum-old authoring patterns (Controller, Middleware, Blade,
Session, interface hooks, plugin helpers) onto the Manifest V3 contracts that
exist today, lists the operations that are **deliberately closed**, and gives
copy-paste quick starts that work in a clean temporary directory.

Every mapping below is a contract that exists in code. Where SForum-old had a
pattern that V3 does not implement, the row says so and names the reason instead
of pointing at a substitute that would not work.

Related: [Authoring guide](./authoring-guide.md), [Routes](./routes.md),
[Runtime themes](./runtime-themes.md), [Extension README](../../extensions/README.md).

## Migration table

| SForum-old pattern | V3 counterpart | What changes for the author |
| --- | --- | --- |
| Controller class + method | `routes[]` entry (`action: add\|replace`, `path`, `methods`, `access`, `permission`, `guard`, `handler`, `requestSchema`, `responseSchema`, `timeoutMs`) | The handler is a declared backend entry invoked over Protocol V2 with a typed request document; it never receives a framework request object. `replace` requires exact-artifact trust and a frozen target route. |
| Route middleware (before/after) | A route entry with `action: "before"` / `"after"`, `guard: "core.guard.inherit"`, and explicit `mutableRequestFields` / `mutableResponseFields` | Middleware is a **declared patch of an existing route**, not a global pipeline. Only the fields you list can be changed; the Host validates and applies them. Before/after runs with the target route's authorization, so it cannot widen access. |
| Blade view / template with PHP | Page Registry `theme.json.pages[]` + Manifest V3 `templates[]` + `packageFiles[]` (three-way identity) and the Host island allowlist | Templates are inert HTML with allowlisted Host islands. There is no template code execution, no arbitrary PHP, and no server-side template includes. Each `theme.json.pages[].template` path needs exactly one matching `templates[]` declaration and one `packageFiles[]` entry of kind `template`, with both digests agreeing. |
| Session state written by the plugin | `Host API v2` `PluginSessionService` (`get`/`set`/`delete`) with the SDK helper `Host.SessionState(ctx)` | Host-owned, plugin-namespaced **session** state for small bounded workflow values. The Host mints a single-invocation session delegation on the invocation you receive; you echo it back and never see a session id. State is ephemeral: tolerate a miss, keep values ≤ 1 MiB, and expect the effective TTL to be tightened to the Core session's remaining lifetime. Core session **mutation** remains a closed operation. |
| `Itf` interface hook | `hooks[]` (`id`, `contractVersion`, `name`, `kind`, `targetId`, `handler`, `inputSchema`) plus typed `events[]`, `queries[]` and `providers[]` | Hooks are declared by name and contract version against a Host-owned target; they are not arbitrary interface implementations discovered by reflection. Payloads are typed documents validated against the declared schema. |
| Plugin helper library (framework facades) | SDK `sdk/plugin/v2` (`pluginv2.NewTypedDocument`, typed documents, settings schema, admin surfaces, guards, schedules, commands) | Helpers build typed documents and declare contracts; they never hand you raw request, session, or database handles. Everything privileged is an explicit declaration plus an exact-artifact trust grant. |

## Deliberately closed operations

These are closed by design. Do not look for a workaround; each one is closed for
a security, integrity, or ownership reason.

- **Raw Fiber access.** Extensions never receive a framework context, router, or
  handler chain. The only transport is Protocol V2 with typed documents, and the
  request header projection drops `x-sforum-*`, hop-by-hop and credential headers
  unless the route declares the explicitly high-risk `raw_request` authority.
  **Breaking change (2026-10-16):** that authority must now come from a guard your
  package **declares itself** (`guards[]` with `kind: raw_request`, plus a matching
  `kind: executable` `packageFiles` entry). The Host-owned
  `core.guard.raw_request` id was removed because it granted raw authority with zero
  guard code; a package that still declares it is **refused at validation**, and an
  already-installed package that carries it fails closed on the next
  enable/restore. Replace the guard reference with your own declared guard id.
- **Core session mutation.** A plugin cannot read, mint, or rewrite Core session
  state, and it never receives a session identifier. What it can do is keep its
  own bounded state under the Host's session scope: `PluginSessionService` +
  `Host.SessionState(ctx)`, scoped to the invocation the Host initiated. A call
  without a Host session delegation is refused, not silently scoped. (See the
  migration table.)
- **Arbitrary Nuxt Layer or frontend code injection.** Admin frontend assets are
  immutable digest paths served from the package; themes contribute templates and
  islands, not a layer that can override Host components.
- **Pre-plugin middleware.** Host middleware runs before any extension code, and
  there is no hook that lets an extension run before it. Route `before` entries
  patch a route that the Host has already authorized.
- **Actor minting.** Extensions cannot mint identity. The actor projection on
  every request is Host-derived (`actorId`, `authenticated`, permission keys);
  a plugin may only narrow it, never widen or fabricate it.
- **Client-IP or proxy classification claims.** Proxy trust belongs to the Host
  configuration; forwarded headers reach the plugin only as the client's own
  claim, with no Host vouching header.

## Reference packages

Focused reference packages live in `extensions/fixtures/plugins/`. The layout
contract has a dedicated one:

- `sforum-page-layout-reference` — the same package declares a public page three
  ways (`layout: "bare"`, `layout: "site"`, and no `layout` at all) so the
  difference is provable in one place: bare means the author owns the chrome, site
  means the Host supplies exactly one navbar and footer, and an omitted layout is
  the platform default `site`. Its templates also satisfy the three-way identity
  contract (each `theme.json` template path has exactly one `templates[]`
  declaration and one `packageFiles[]` entry of kind `template`, digests agreeing).

These fixtures are CI fixtures: their backends (when they have one) declare
`replace … => ../../../../../apps/api` and are compiled by the Go test harness, so
they are not meant to be copied out of the repository. For a clean-directory
package, use the quick starts below.

## Quick starts that pass in a clean temporary directory

These run outside the repository tree. `make:plugin` needs only the CLI; no
dependency install and no network access is required for the backend-only path.

### Plugin package: scaffold, refresh, validate, test, build

```bash
mkdir -p /tmp/sforum-quickstart
cd <repository>/apps/api

go run ./cmd/sforum make:plugin --no-interaction \
  --id acme.quickstart --name "Quickstart" \
  --description "clean temp dir proof" --url https://example.com/qs \
  --author-name Acme --backend --out /tmp/sforum-quickstart/acme.quickstart

go run ./cmd/sforum extension digest --write /tmp/sforum-quickstart/acme.quickstart
go run ./cmd/sforum extension validate /tmp/sforum-quickstart/acme.quickstart
go run ./cmd/sforum extension test /tmp/sforum-quickstart/acme.quickstart
go run ./cmd/sforum extension build --allow-scaffold --skip-install /tmp/sforum-quickstart/acme.quickstart
```

Verified transcript (2026-10-16, clean `/tmp` directory outside the repo):

```text
MAKE_OK
digest --write OK
validate OK
test OK
build OK
  errors:  0  warnings: 1  checks: 6
```

The single warning is `backend.scaffold_stub`: the generated backend entry is a
shell stub until you replace it with a real go-plugin binary. `--allow-scaffold`
exists so the author command chain stays green while you are still writing it.

### Theme package: scaffold and activate

```bash
go run ./cmd/sforum make:theme --no-interaction \
  --id acme.quicktheme --name "Quick Theme" \
  --description "theme quickstart" --url https://example.com/qt \
  --author-name Acme --out /tmp/sforum-quickstart/acme.quicktheme

go run ./cmd/sforum extension digest --write /tmp/sforum-quickstart/acme.quicktheme
go run ./cmd/sforum extension validate /tmp/sforum-quickstart/acme.quicktheme
go run ./cmd/sforum extension test /tmp/sforum-quickstart/acme.quicktheme
```

A one-page scaffold validates and tests cleanly, but it **cannot replace a theme
that owns the full Core page set**: the activation preflight rejects a package
whose declared pages do not cover the pages the outgoing theme provides. That
refusal is intentional (a partial theme would silently drop pages), and it is the
same check the CLI runs. Add the pages you intend to own, or install the theme as
a new package instead of replacing an active one.

### Development edit loop

```bash
cd <repository>/apps/api
APP_ENV=development go run ./cmd/sforum extension dev ../../extensions/dev/acme.quickstart --skip-install
```

One command rebuilds on every source edit, refreshes digests, runs validation and
the contract tests, and prints the new package digest plus the next step. It does
**not** activate anything: the exact artifact still has to be confirmed by an
operator in the admin extensions flow. See
[the CLI handbook](../zh-CN/development/cli.md) for the flags.
