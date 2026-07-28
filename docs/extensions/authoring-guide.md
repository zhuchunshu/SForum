# Plugin authoring guide

> **Handbooks (bilingual):**  
> [中文 · 扩展运营侧](../zh-CN/usage/extensions.md) ·
> [English · extensions (ops)](../en-US/usage/extensions.md) ·
> [中文开发](../zh-CN/development/README.md) ·
> [English development](../en-US/development/README.md) ·
> [Docs hub](../README.md)

This guide is the hand-written **technical** entry point for building SForum
plugins (Wave F4.2). Generated host catalogs live under
[catalogs/](./catalogs/README.md) — always regenerate those from code after
changing events, capabilities, contribution points, provider slots, or core
schedules.

## What you should depend on

| Surface | Use for |
| --- | --- |
| Manifest V3 `sforum.extension.json` (+ optional `includes`) | Versioned Registry/platform declarations and exact package-file identity |
| Go SDK `github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2` | New typed backend server, generated Host clients, exact runtime binding, health/readiness |
| Host API v2 wire `sforum.host/v2` | Typed queries/commands and versioned platform services over the runtime-scoped Host broker |
| Go SDK + Host API v1 | Compatibility for existing SMTP, storage, and content-policy plugins until P13 |
| go-plugin v2 gRPC / v1 net-rpc | Exact Manifest-selected process protocol; no silent downgrade |
| Published catalogs | Which events/points/slots/capabilities exist |

**Do not** import `app/Models/*` or other host business packages from a
third-party plugin. Built-in SMTP still uses internal packages for historical
reasons; new plugins should follow the SDK path shown by the Host API fixture.

Forum revision payloads are deliberately not a plugin query or mutation surface.
Plugins may observe the safe revision metadata on `topic.updated` and
`comment.updated` (`revisionNo`, operation, changed fields, and restore origin),
but never raw historical source, staff reasons, IPs, attachment-provider data,
or a bypass for Core's history, restore, redaction, or CAS policy.

### Identity / external auth providers

Reference package: `extensions/builtin/plugins/sforum-auth-github` (Manifest V3
`identity.runtime@1`).

| Plugin may | Host alone owns |
| --- | --- |
| Build authorize URL from Host `state` / PKCE challenge / absolute callback | OAuth `state`, PKCE verifier, callback transaction, absolute callback URL |
| Exchange code with bounded timeouts; return raw `providerSubject` in typed complete | Subject HMAC digest, users, links, roles, risk/session, browser sessions |
| Discard access tokens after identity proof | Public activation CAS, rate limits, audit, public error mapping |

**Closed surfaces (do not replace):** Core `GET /api/v1/auth/providers/{id}/callback`,
AuthSession issue/renew/destroy, callback state store, registration tickets.
Route Registry replacement of the callback is forbidden for integrity reasons.

External auth providers must ship complete operator setup guidance in their
Settings Document: where to create provider credentials, which public site and
callback URLs to enter, how secrets are stored, and the order for save, probe,
and Host activation. Use callout `linkLabel` + `linkUrl` for the official
application or documentation page; the Host validates HTTP(S) links and renders
them without vendor-specific Core branches.

Operator docs: [zh-CN](../zh-CN/usage/github-login.md) ·
[en-US](../en-US/usage/github-login.md).

## Where to put your package

Repository layout lives under [`extensions/`](../../extensions/README.md).
**Only `extensions/builtin/` is boot-scanned into the admin extension list.**

| Goal | Directory | Boot scan | Notes |
| --- | --- | --- | --- |
| Protected built-in (ship in product, auto list) | `extensions/builtin/plugins/<dir>/` | **Yes** (`SyncBuiltins`) | Themes: `…/themes/`. Backend build is **not** automatic for new ids — see below. |
| Local experiment (default scaffold) | `extensions/dev/plugins/<id>/` | No | Gitignored; never appears in admin by itself. |
| Ship in git, operator installs | `extensions/optional/plugins/<dir>/` | No | See [`extensions/optional/README.md`](../../extensions/optional/README.md). |
| Independent source collection | `<external-root>/{plugins,themes}/<dir>/` | Yes, when configured | Set `EXTERNAL_EXTENSION_ROOTS`; discovery remains inert. |
| CI / contract lock | `extensions/fixtures/…` | No | Test inputs only. |
| Production site | Admin upload → `EXTENSION_ROOT` | No | Operator trust + enable path. |

### Auto scan vs auto build

- **Built-in register:** API/worker boot runs `SyncBuiltins` over
  `BUILTIN_EXTENSION_ROOT` (default `extensions/builtin`, or
  `storage/builtin-dev` when using `./scripts/api-dev.sh`). Every
  `plugins/*` and `themes/*` child directory with a valid package is
  written into the extension store and shows up under admin Themes/Plugins.
- **External discovery:** API/worker boot scans each comma-separated
  `EXTERNAL_EXTENSION_ROOTS` collection. Valid packages are copied into
  `EXTENSION_ROOT` as immutable uploaded snapshots. New packages stay inert;
  changes become staged candidates and never inherit trust or auto-promote.
- **Auto build (dev only):** `scripts/build-builtin-plugins.sh` (invoked by
  `api-dev` / `worker-dev`) stages builtins into `storage/builtin-dev` and
  `go build`s **only the plugin ids hard-coded in that script**. A new
  builtin package is scanned after restart once it is present under the
  active builtin root, but you must either extend the script or build +
  `extension digest --write` yourself before enable.

Scaffold defaults:

```bash
cd apps/api
# Local scratch (gitignored) — does NOT auto-register
go run ./cmd/sforum make:plugin --id acme.demo --name "Acme Demo" \
  --description "…" --backend --no-interaction
# → ../../extensions/dev/plugins/acme.demo

# Protected built-in — SyncBuiltins will pick it up after restart
go run ./cmd/sforum make:plugin --id sforum.foo --name "Foo" \
  --description "…" --backend --builtin --no-interaction
# → ../../extensions/builtin/plugins/sforum.foo
```

Full CLI reference (make, digest, validate, test, package `--exclude-source`,
seed, recovery):  
[中文 · 开发者 CLI](../zh-CN/development/cli.md) ·
[English · Developer CLI](../en-US/development/cli.md).

Full map: [`extensions/README.md`](../../extensions/README.md).

## Quick start

```bash
# Scaffold (from apps/api). Prefer --out for a disposable path, or omit --out
# to land under extensions/dev (gitignored). Use --builtin for SyncBuiltins.
cd apps/api
go run ./cmd/sforum make:plugin \
  --id acme.demo \
  --name "Acme Demo" \
  --description "Example plugin" \
  --url https://example.com/acme-demo \
  --author-name Acme \
  --backend \
  --no-interaction \
  --out /tmp/acme.demo

# Contract checks (catalogs, capabilities, backend entry)
go run ./cmd/sforum extension test --allow-scaffold /tmp/acme.demo

# After changing packaged files, refresh exact digests and revalidate
go run ./cmd/sforum extension digest --write /tmp/acme.demo

# After you change host catalogs, refresh docs
go run ./cmd/sforum extension docs generate
go run ./cmd/sforum extension docs generate --check
```

Scaffolds use versioned Schema UI by default. Optional buildless variants:

```bash
# Provider + host-rendered Probe action (known provider slot; backend required)
go run ./cmd/sforum make:plugin ... --backend --provider-slot mail.provider

# Framework-neutral author-prebuilt component with Schema fallback
go run ./cmd/sforum make:plugin ... --prebuilt-settings
go run ./cmd/sforum make:theme ... --prebuilt-settings
```

Minimal backend using the public SDK:

```go
package main

import pluginsdk "github.com/zhuchunshu/sforum/apps/api/sdk/plugin"

type myPlugin struct{ pluginsdk.Noop }

func (myPlugin) InvokeHook(req pluginsdk.HookRequest) (pluginsdk.HookResponse, error) {
	// filter/validate: stay cheap; observe: never block the host write path.
	return pluginsdk.HookResponse{OK: true}, nil
}

func main() { pluginsdk.Serve(myPlugin{}) }
```

Build the executable to the path declared in `backend.entry` (usually
`backend/plugin`) before enabling the plugin in admin.

The scaffold and current built-in references still demonstrate protocol v1
while their v2 migrations are completed. New typed runtime work should follow
[Host API v2 and non-Go runtimes](./host-api-v2.md), including the generated Go
SDK, AutoMTLS broker, exact trust binding, and non-Go interoperability rules.

## Manifest V3 package contract

All new CLI scaffolds emit explicit `manifestVersion: 3`. The complete generated
root-field catalog is [manifest-v3.md](./catalogs/manifest-v3.md); the embedded
Draft 2020-12 schema and Go semantic validator are both authoritative.

Version handling is fail-closed:

- omitted `manifestVersion` remains the legacy V1 contract;
- explicit V2 remains accepted for compatibility;
- V3-only declarations require explicit version 3 and never upgrade a V1/V2
  package implicitly;
- future versions are rejected until the host implements them.

Large manifests may shard supported families through `includes`. A family must
be declared either at the root or through its include, never both. Include paths
are package-relative JSON files/directories and are resolved during inert
preflight; loading them never executes package code.

Every V3 package declares exact SHA-256 identities in `packageFiles`. Supported
kinds are `executable`, `frontend`, `locale`, `schema`, `migration`, `template`,
`asset`, and `openapi`. Backend, guard, migration, template, asset, OpenAPI, and
L2 references must resolve to the matching declared file kind and digest.

### Notification types and Host emission

Executable plugins may declare inert `notificationTypes`. Each type id must be
under the plugin id namespace, bind `contractVersion: <type>@1`, and reference
one exact `packageFiles` JSON Schema through `payloadSchema`. A plugin type
cannot be `required`; only Core may own non-configurable required notices.

At runtime use Host API v2 `notifications.emit@1` through
`(*pluginv2.Host).EmitNotification`. The request contains one declared type and
payload version, a structured payload, at most 50 explicit active user ids, a
declared target descriptor, and a visible-ASCII idempotency key. The Host
validates the exact artifact/grant/epoch/instance, schema digest and value,
target, recipient policy, 16 KiB payload bound, count, rolling rate, and
deadline in its transaction. It writes recipient-owned projections and
redacted audit evidence; it never accepts a plugin actor, session, cookie,
arbitrary URL, required type, raw notification-table write, or bulk broadcast.

See the generated [notification contract](./catalogs/notifications.md) and
`extensions/fixtures/plugins/sforum-notification-reference`.

```bash
cd apps/api
go run ./cmd/sforum extension digest --write <package-root>
go run ./cmd/sforum extension validate <package-root>
go run ./cmd/sforum extension test <package-root>
```

`digest --write` updates inline `packageFiles` after local file changes and then
revalidates the whole package. Keep `packageFiles` inline when using this helper;
included package-file shards remain valid but must be refreshed by the package
authoring pipeline.

Dependency entries use exactly one of `id` or `capability`:

| Kind | Meaning |
| --- | --- |
| `required` | A matching extension/capability and semantic version must exist before activation |
| `optional` | Orders activation when present, but absence is allowed |
| `conflict` | A matching installed version blocks the graph |
| `provides` | Publishes one capability at an exact semantic version |

The host rejects missing required versions, ambiguous capability providers,
cycles, duplicate extension identities, and ambiguous replacement providers.
Activation order is deterministic.

Static install is always inert. Uploaded packages with backend code, migrations,
custom/raw guards, raw/kernel database authority, executable lifecycle hooks,
admin code, or public L2 files require an actor-bound `super_admin` confirmation
for the exact artifact before first execution. The impact document binds the
Manifest contract, declarations, real executable bytes, authorities,
dependencies, and Host/Frontend contracts. Any relevant change invalidates the
grant. Capability declarations describe and shape trusted code; they are not a
sandbox.

### Public L2 honesty

Public L2 (package-local prebuilt ESM/CSS mounted by `SFExtensionWidget`) is
**fully trusted browser code**, not a sandbox. When a contribution mounts, the
Host bridge sets `trust: fully_trusted_browser_code` and the public widget shows
an operator-facing honesty note: the component can use the current browser
session and page authority for its namespaced API. Enable `SFORUM_V3_PUBLIC_L2`
only for extensions you trust. SSR/L1 slot content remains the fallback when L2
fails, is quarantined, or is disabled.

Themes remain presentation-only. They may declare templates, assets,
components, navigation/regions, settings, package files, and other presentation
contracts, but cannot own plugin business/runtime declarations. A plugin's
versioned business data contract does not change when a theme overrides its
presentation.

System error pages are a narrower theme-only surface. A theme may replace
`system.forbidden`, `system.not_found`, `system.rate_limited`, and
`system.server_error` with L0/L1 presentation, but the Host owns the HTTP
status, public copy, cache/robots policy, and home/back/retry actions. Plugins
cannot replace `system.*` pages, and system error templates cannot mount public
L2 widgets.

Manifest V3 currently defines and validates the full target contract. Runtime
implementations land phase by phase (Host API v2, lifecycle/database, Route
Registry, platform registries, themes, and L2); a valid declaration is not proof
that a later runtime family is active in the current host release.

## Buildless settings UI: choose the smallest capable mode

Plugins and themes share one versioned Settings Document. Existing array-form
`settings` remains compatible, but new packages should emit `schemaVersion: 1`.

| Need | Manifest mode | Author JS | Operator host build |
| --- | --- | --- | --- |
| Fields, tabs, groups, columns, callouts | Schema | None | None |
| Provider connection test or other catalogued operation | Schema + Actions | None | None |
| Truly complex interaction | Prebuilt component + Schema fallback | Fully trusted `.mjs` | None |

### Schema field types and width

Host Schema UI supports these `fields[].type` values:

| Type | Control |
| --- | --- |
| `text` / `string` | Single-line input |
| `number` | Numeric input |
| `boolean` | Checkbox |
| `select` | Select when `options` is present (options also force select for other free-text types) |
| `secret` | Password input; empty save preserves the stored secret |
| `textarea` | Multi-line autoresizing textarea |

Optional `fields[].width`:

| Value | Behavior |
| --- | --- |
| `default` (or omitted) | Control is capped (`max-w-xl`) — good for short values |
| `full` | Control fills the available column width — prefer for long copy and `textarea` |

Example:

```json
{
  "key": "home.notice.zh-CN",
  "label": { "zh-CN": "首页提示条", "en-US": "Homepage notice" },
  "type": "textarea",
  "width": "full",
  "default": "",
  "groupId": "home-copy"
}
```

### Reference A — theme tabs/groups, no JavaScript

This is the default for new themes. Upload, configure, activate through Page
Registry, and save values without building Nuxt:

```json
{
  "id": "acme.reading-theme",
  "name": "Acme Reading Theme",
  "description": "Readable runtime theme.",
  "url": "https://example.com/theme",
  "author": { "name": "Acme" },
  "version": "1.0.0",
  "type": "theme",
  "sforumVersion": ">=0.1.0",
  "admin": {
    "entry": "/settings",
    "pages": [{ "path": "/settings", "label": "Theme settings", "view": "settings" }]
  },
  "settings": {
    "schemaVersion": 1,
    "ui": {
      "mode": "schema",
      "layout": "tabs",
      "tabs": [
        { "id": "home", "label": { "zh-CN": "首页", "en-US": "Home" }, "groups": ["hero", "layout"] }
      ],
      "groups": [
        { "id": "hero", "label": { "zh-CN": "主视觉", "en-US": "Hero" }, "columns": 2 },
        { "id": "layout", "label": { "zh-CN": "布局", "en-US": "Layout" } }
      ],
      "callouts": [{
        "id": "instant",
        "tone": "info",
        "title": { "zh-CN": "保存后立即生效", "en-US": "Applies immediately after save" },
        "tab": "home"
      }]
    },
    "fields": [
      { "key": "hero.title", "label": { "zh-CN": "标题", "en-US": "Title" }, "type": "text", "default": "Welcome", "groupId": "hero", "column": 1 },
      { "key": "hero.compact", "label": { "zh-CN": "紧凑模式", "en-US": "Compact" }, "type": "boolean", "default": "false", "groupId": "hero", "column": 2 },
      { "key": "hero.blurb", "label": { "zh-CN": "简介", "en-US": "Blurb" }, "type": "textarea", "width": "full", "default": "", "groupId": "hero" }
    ]
  }
}
```

See `extensions/builtin/themes/sforum-default/manifest/settings.json` and
`extensions/fixtures/themes/sforum-schema-theme/`.

### Reference B — provider Schema + Probe action

SMTP is the production reference. Operators can install → configure while
disabled → test with the restricted probe runtime → enable. The probe cannot
register normal routes, events, jobs, schedules, or providers and never sends
test mail:

```json
{
  "backend": { "entry": "backend/plugin", "rpc": "hashicorp-go-plugin", "protocolVersion": 1 },
  "providers": [{ "slot": "mail.provider", "label": "SMTP", "timeoutMs": 15000 }],
  "settings": {
    "schemaVersion": 1,
    "ui": {
      "mode": "schema",
      "layout": "tabs",
      "tabs": [{ "id": "connection", "label": "Connection", "groups": ["server", "credentials"] }],
      "groups": [
        { "id": "server", "label": "Server", "columns": 2 },
        { "id": "credentials", "label": "Credentials", "columns": 2 }
      ]
    },
    "fields": [
      { "key": "host", "label": "Host", "type": "text", "default": "127.0.0.1", "groupId": "server" },
      { "key": "port", "label": "Port", "type": "number", "default": "1025", "groupId": "server" },
      { "key": "username", "label": "Username", "type": "text", "default": "", "groupId": "credentials" },
      { "key": "password", "label": "Password", "type": "secret", "default": "", "groupId": "credentials" }
    ],
    "actions": [{
      "id": "probe",
      "kind": "provider_probe",
      "label": { "zh-CN": "测试连接", "en-US": "Test connection" },
      "placement": "footer",
      "useDraftValues": true,
      "fields": ["host", "port", "username", "password"]
    }]
  }
}
```

Implement SDK `ProviderProbe`. The host owns permission checks, declared-field
and secret preserve/replace semantics, 15-second timeout, response bounds, and
credential-free audit. Storage providers reuse `StorageProbe` through the same
Action catalog.

### Reference C — trusted prebuilt complex component

Use this only when Schema + Actions cannot express the interaction. The author
ships final `.mjs`/`.css` bytes; SForum never compiles uploaded Vue SFCs and
never accepts remote script URLs. Schema fields remain mandatory fallback:

```json
{
  "settings": {
    "schemaVersion": 1,
    "ui": {
      "mode": "component",
      "layout": "form",
      "component": {
        "id": "settings",
        "apiVersion": 1,
        "entry": "frontend/admin/dist/settings.mjs",
        "css": "frontend/admin/dist/settings.css"
      }
    },
    "fields": [{ "key": "message", "label": "Message", "type": "text", "default": "Hello" }]
  }
}
```

```js
export const apiVersion = 1
export function mount(target, bridge) {
  const input = document.createElement('input')
  input.value = bridge.settings.values().message || ''
  const onInput = () => bridge.settings.updateValue('message', input.value)
  input.addEventListener('input', onInput)
  target.append(input)
  return () => {
    input.removeEventListener('input', onInput)
    input.remove()
  }
}
```

An active super administrator confirms a one-use actor/version/API/component/
digest-bound challenge. Import, contract, mount, CSS, cleanup, or quarantine
failure automatically returns to Schema UI. See
`extensions/fixtures/plugins/sforum-prebuilt-settings/` and
[trusted-admin-components.md](./trusted-admin-components.md).

## Reference 1 — built-in SMTP (`sforum.smtp`)

Path: `extensions/builtin/plugins/sforum-smtp/`

Use this package when you need a **provider-slot** plugin with settings and
outbound network:

| Area | What SMTP does |
| --- | --- |
| Manifest | `providers: [{ slot: "mail.provider", ... }]`, explicit `capabilities` |
| Multi-file layout | Thin root + `includes` for langs, versioned settings, and admin |
| Backend | Implements `SendMail` over go-plugin; no public HTTP `RouteTarget` |
| Settings | Host injects `SFORUM_SETTING_*` env vars into the child process |
| Admin UI | Host-rendered tabs/groups/callouts + `provider_probe`; no frontend package |

Key manifest ideas (simplified):

```json
{
  "id": "sforum.smtp",
  "type": "plugin",
  "capabilities": ["host.api", "settings.own", "net.outbound"],
  "backend": {
    "entry": "backend/plugin",
    "rpc": "hashicorp-go-plugin",
    "protocolVersion": 1
  },
  "providers": [{ "slot": "mail.provider", "label": "SMTP", "timeoutMs": 15000 }]
}
```

Contract-test it without requiring a rebuilt binary:

```bash
cd apps/api
go run ./cmd/sforum extension test --skip-backend-binary \
  ../../extensions/builtin/plugins/sforum-smtp
```

Read the full package README and `manifest/` files for settings groups/actions
and Chinese/English identity langs.

## Reference 2 — content policy workflow (`sforum.content-policy`)

Path: `extensions/builtin/plugins/sforum-content-policy/`

Use this package when you need a **non-provider workflow** plugin: sync
filters, settings, and public UI contributions (Wave E5). SMTP remains the
**mail provider** reference; this package is the **filter + contribution**
reference.

| Area | What content policy does |
| --- | --- |
| Manifest | Explicit `capabilities`, three filter `events`, multi-file `includes` |
| Backend | Public SDK `Serve` + `Noop`; implements `InvokeHook` only |
| Filters | `topic.before_create`, `topic.before_update`, `comment.before_create` |
| Settings | Keywords, mode (`reject` / `tag`), force tag, scan toggles, optional public badge |
| Contributions | Optional `forum.topic.badges` (`show_topic_badge`, default off) + `forum.topic.sidebar` → `/guidelines` |
| Providers | None (not a slot plugin) |

Operator path (fresh contributor):

1. Admin → Extensions → Plugins → enable **SForum Content Policy** (confirm
   capabilities: `host.api`, `settings.own`, `audit.append`).
2. Manage settings → add keywords (one per line; `#` comments allowed).
3. Keep mode **Reject publish** (recommended default).
4. Publish a topic/reply containing a keyword → API `422` with reason
   `content_policy.keyword_blocked`.
5. Topic detail shows a guidelines sidebar card. The title badge stays hidden
   until **Show badge under topic title** is enabled in plugin settings.

Author rules demonstrated by this package:

- Filters stay **cheap** (substring match on env-injected settings only).
- No Host API / network / River jobs inside `InvokeHook`.
- `mode=tag` only patches `tagSlugs` on topics; comments still reject.
- Force-tag requires the slug to already exist under host tag policy.

```bash
cd apps/api
go run ./cmd/sforum extension test --skip-backend-binary \
  ../../extensions/builtin/plugins/sforum-content-policy

# Build binary (also run by scripts/build-builtin-plugins.sh)
(cd ../../extensions/builtin/plugins/sforum-content-policy/backend && \
  go test ./... && go build -o plugin .)
```

See package `README.md` for settings env names and force-tag notes.

## Reference 3 — filesystem storage (`sforum.storage-fs`)

Path: `extensions/builtin/plugins/sforum-storage-fs/`

Use this package when you need a **storage provider-slot** plugin (Wave E6).
SMTP remains the **mail** provider reference; this package is the
**attachment.storage.provider** reference (no cloud credentials required).

| Area | What filesystem storage does |
| --- | --- |
| Manifest | `providers: [{ slot: "attachment.storage.provider", ... }]`, `settings.own` |
| Backend | Public SDK `Serve` + `Noop`; implements chunked `Storage*` RPCs |
| Settings | Absolute `root_path`, optional `public_base_url` (env `SFORUM_SETTING_*`) |
| Operator loop | Enable → Attachment settings select `plugin:sforum.storage-fs` → configure → Test connection → upload |

Key RPC methods (host streams ~1 MiB chunks by default):

| RPC | Role |
| --- | --- |
| `StoragePutBegin` / `StoragePutChunk` | Write object by key (Final commits) |
| `StorageOpen` / `StorageGetChunk` | Read object bytes |
| `StorageClose` | Abort put or release read session |
| `StorageDelete` / `StorageStat` / `StorageExists` | Lifecycle helpers |
| `StoragePublicURL` / `StorageSignedURL` | Optional URL strings (ACL still host-owned) |
| `StorageProbe` | Admin “Test connection” |

Minimal author shape:

```go
type storePlugin struct{ pluginsdk.Noop }

func (storePlugin) StorageProbe(pluginsdk.StorageProbeRequest) (pluginsdk.StorageProbeResponse, error) {
	return pluginsdk.StorageProbeResponse{OK: true}, nil
}
// … implement PutBegin/PutChunk/Open/GetChunk/Close/Delete …

func main() { pluginsdk.Serve(storePlugin{}) }
```

```bash
cd apps/api
go run ./cmd/sforum extension test --skip-backend-binary \
  ../../extensions/builtin/plugins/sforum-storage-fs

(cd ../../extensions/builtin/plugins/sforum-storage-fs/backend && \
  go test ./... && go build -o plugin .)
```

Rules demonstrated by this package:

- Object **keys** are host-generated; never invent alternate namespaces.
- Secrets/settings live in `extension_settings` (not `attachment.*` core options).
- Fail closed on bad config / I/O; host maps failures to
  `attachment.storage_unavailable` for uploads.
- Prefer a dedicated root directory; do not share core `local` root.
- For S3/MinIO/R2, keep the **same RPC surface** and put the vendor SDK only
  inside your plugin (core stays free of new cloud SDKs).

See package `README.md` for the full operator path.

## Reference 4 — Host API fixture (`sforum.contract.hostapi`)

Path: `extensions/fixtures/plugins/sforum-contract-hostapi/`

Use this package when you need **Host API + events** without a product vertical:

| Area | What the fixture does |
| --- | --- |
| Capabilities | Explicit `host.api`, `settings.own`, `jobs.enqueue` |
| Events | Declares `topic.created` (observe) and `topic.before_create` (filter) |
| Jobs | Declares `sforum.contract.hostapi.demo` so `EnqueueOwnJob` can be authorized |
| Backend | SDK `Serve` + `HostFromEnv` / `Ping` inside `Health` when env is present |

```go
// Health pings the host when SFORUM_HOST_API_* is injected.
host, _ := pluginsdk.HostFromEnv()
resp, _ := pluginsdk.Ping(ctx, host)
```

CI builds this binary and starts it through the real protocol starter so
handshake + Host API + hooks stay green (`go test ./sdk/plugin`).

Related fixtures:

- `sforum-contract-events` — events + `forum.topic.actions` contribution only
- `sforum-contract-schedules` — documents that schedules stay host-owned

## Scenario map (which mechanism?)

| I want to… | Use |
| --- | --- |
| Run code after a topic is created | observe `topic.created` (+ optional Host API job) |
| Change or reject a new topic | filter `topic.before_create` |
| Change or reject a topic edit | filter `topic.before_update` |
| Change or reject a new reply | filter `comment.before_create` |
| Reject registration | validate `user.before_register` |
| Reject an upload by metadata | validate `attachment.before_upload` |
| Add a topic detail button | contribution `forum.topic.actions` |
| Add topic badges / sidebar / list pills | `forum.topic.badges` / `sidebar` / `list.badges` |
| Add public nav entries | contribution `forum.nav.items` |
| Swap outbound mail transport | provider `mail.provider` (see `sforum.smtp`) |
| Swap attachment storage | provider `attachment.storage.provider` (see `sforum.storage-fs`) |
| Swap full-text search | provider `search.provider` (Wave E7) |
| Store per-topic/plugin structured data | entity meta (F4.4 / E3) |
| Own HTTP API under the host proxy | manifest `routes` + backend `RouteTarget` |
| Call host from the plugin process | Host API + declared `capabilities` |
| End-to-end **workflow** sample | enable `sforum.content-policy` (this section) |
| End-to-end **mail provider** sample | enable `sforum.smtp` + select in Mail settings |
| End-to-end **storage provider** sample | enable `sforum.storage-fs` + select in Attachment settings |

## Manifest checklist

1. **Contract** — explicit `manifestVersion: 3` for new packages; stable `id`,
   semantic `version`, `sforumVersion`, URL, author, and description.
2. **Type** — `plugin` vs presentation-only `theme`; never place business or
   runtime declarations in a theme.
3. **Stable identities** — every V3 declaration uses a namespaced stable id and
   explicit `contractVersion`; replacement targets are declared, not inferred.
   System error page targets are theme-only and L0/L1-only.
4. **Package files** — every referenced executable/frontend/migration/template/
   asset/OpenAPI file has the correct kind and exact SHA-256 digest.
5. **Routes and guards** — explicit action, methods, mode, fallback, schemas,
   contract version, and core/custom guard. Raw request ownership is high-risk.
6. **Database and lifecycle** — authority tier, compatibility range where raw,
   backup/retention, plan/execute, progress, and checkpoint contracts.
7. **Permissions** — definitions use `assignmentPolicy: host`; recommended role
   mappings are advisory and extension code never self-grants. `label` and
   `description` accept either a default string or an extension-owned locale map
   such as `{ "zh-CN": "管理扩展", "en-US": "Manage extension" }`.
8. **Dependencies** — required/optional/conflict/provides semantics resolve to
   one deterministic, acyclic graph.
9. **Host catalogs** — capabilities, events, contributions, providers, and jobs
   use the published generated catalogs.
10. **Notifications** — namespaced inert types bind an exact schema file;
    runtime emission uses `notifications.emit@1`, never raw tables or broadcast.
11. **Operator defaults** — settings include safe defaults and reset-friendly
    recommended values so first-time operators are not blocked.

## Host API methods (v1)

Version id: `sforum.host/v1`. Env injected by the host:

- `SFORUM_HOST_API_URL`
- `SFORUM_HOST_API_TOKEN`
- `SFORUM_EXTENSION_ID`
- `SFORUM_HOST_API_VERSION`

| Method | Capability | Purpose |
| --- | --- | --- |
| `Ping` | `host.api` | Connectivity probe |
| `GetSettings` | `settings.own` | Read this extension's settings |
| `CheckPermission` | `permissions.check` | Ask if a user holds a permission |
| `EnqueueOwnJob` | `jobs.enqueue` | Enqueue a **declared** job kind |
| `AppendAudit` | `audit.append` | Append namespaced audit events |
| `GetUserSafe` | `users.read` | Read non-secret user fields |

SDK helpers: `pluginsdk.Ping`, `GetSettings`, `CheckPermission`, `EnqueueOwnJob`,
`AppendAudit`, `GetUserSafe`.

## Event rules of thumb

- **filter / validate** — default `fail_closed`, short timeout (see catalog). Never send mail, call remote APIs, or reindex inside a filter; enqueue a job.
- **observe** — async; failures should not break the write path.
- Prefer Host API or River jobs for anything that can take more than a few hundred milliseconds.

### Which filter for which scenario

| Scenario | Event | Kind | Can patch | Notes |
| --- | --- | --- | --- | --- |
| Block or rewrite a new topic (title/tags/category/body) | `topic.before_create` | filter | `categorySlug`, `tagSlugs`, `title`, `content` | Runs after permission check; host re-validates after patch |
| Block or rewrite a topic edit (title/tags/category/body) | `topic.before_update` | filter | `categorySlug`, `tagSlugs`, `title`, `content` | After edit permission + author edit window; payload only includes fields present in the request; plugins may still patch missing fields (e.g. force tags). Host re-validates after patch |
| Block or rewrite a new reply body | `comment.before_create` | filter | `content` only | After auth + topic active check; before content limits / render / commit. `parentId` is payload-only (not patchable in v1) |
| Block registration (e.g. disposable email) | `user.before_register` | validate | — (reject-only) | After host field/password policy parse, before password hash / user row. Payload: `username`, `email`, `locale` only — **never password**. Prefer stable `reason` codes; minimize PII in plugin logs |
| Block upload the core allowlist would accept | `attachment.before_upload` | validate | — (reject-only) | After host MIME sniff / size / extension policy, before storage `Put`. Payload: `actorUserId`, `contentType`, `sizeBytes`, `filename` only — **never raw file bytes** |
| Side effects after commit | `user.registered`, `topic.created`, `topic.updated`, `comment.created`, `attachment.uploaded`, … | observe | — | Async; do not use for rejection |

Reject with a stable `reason` code (mapped to the API error envelope as 422). Do not put passwords, raw file bytes, or stack traces in filter/validate payloads or messages.

## Schedules

Do not start `time.Ticker` / cron loops in the plugin process. Core schedules
are listed in [schedules.md](./catalogs/schedules.md). Plugin-owned periodics
will go through the host registry in a later wave.

## Admin / trusted components

Prefer Schema or Settings Actions. For complex settings, ship the prebuilt
Admin Micro-frontend API v1 contract. It is **full trust** on the admin origin
after exact digest approval. Runtime Vue SFC compilation, executable admin
slots, remote scripts, and extension-triggered host builds are unsupported. Read
[trusted-admin-components.md](./trusted-admin-components.md).

## Validation and packaging commands

```bash
# Parse + merge includes, print summary
go run ./cmd/sforum extension validate <package-root>

# Refresh inline Manifest V3 packageFiles digests and revalidate
go run ./cmd/sforum extension digest --write <package-root>

# Host contract report (exit 1 on errors)
go run ./cmd/sforum extension test <package-root>
go run ./cmd/sforum extension test --json <package-root>

# Release zip (omit common source/dev files)
go run ./cmd/sforum extension package <package-root> --exclude-source

# Catalog documentation drift
go run ./cmd/sforum extension docs generate --check
```

Full flag tables and release loop:  
[中文 · 开发者 CLI](../zh-CN/development/cli.md) ·
[English · Developer CLI](../en-US/development/cli.md).

## Contribution points (F4.3 / E2)

Declare only host-owned points (see [contribution-points.md](./catalogs/contribution-points.md)).
Payloads are JSON descriptors — **never executable code**.

| Point | Payload type | Runtime consumer |
| --- | --- | --- |
| `forum.topic.actions` | `extensionRoute` | Topic detail `extensionActions` |
| `forum.topic.sidebar` | `topicSidebarCard` (`extensionRoute` \| `hostLink`) | Topic detail `extensionSidebar` (default theme side cards) |
| `forum.topic.badges` | `topicBadge` (`tone` + optional host `href`) | Topic detail `extensionBadges` (under title) |
| `forum.comment.actions` | `extensionRoute` (+ optional `requiresAuth`) | Comment list `extensionActions` (row menus) |
| `forum.nav.items` | `navItem` (`hostLink` \| public `extensionRoute` GET) | `GET /site/nav-items` → `extensionItems` (default theme navbar) |
| `forum.topic.list.badges` | `topicBadge` | Topic list `extensionListBadges` (list-row pills) |
| `forum.composer.toolbar` | `extensionRoute` | `GET /composer/toolbar` + composer UI |
| `forum.profile.tabs` | `profileSection` (`extensionRoute` \| `hostLink`) | Public profile `extensionTabs` |
| `admin.dashboard.widgets` | `dashboardLink` (`adminLink` + admin route) | Admin overview `extensionWidgets` |
| `system.health.checks` | `healthDescriptor` (`static` \| `extensionRuntime`) | Merged into `GET /ready` without plugin RPC |

### Topic secondary surfaces (E2.1)

- **Sidebar** title/icon come from contribution `label` / `icon` (approved
  `i-lucide-*` / `i-tabler-*` only). Payload chooses target only.
- **Badges** use host `tone` enum: `neutral|info|success|warning|danger`.
  Optional `href` is a site-relative path (not `/api`, not external).
- Themes must render empty-safe: omit UI blocks when arrays are empty.
- No public executable extension injection; actions still go through the extension route
  proxy and host policy.

### Comment row actions (E2.2)

- Same payload spirit as `forum.topic.actions`: `extensionRoute` with
  `POST|PUT|PATCH|DELETE` only (no GET).
- Optional `requiresAuth: true` is a **UX hint** — the default theme hides the
  control for guests; the extension route proxy still enforces login/policy.
- Host returns descriptors once on `CommentList.extensionActions` (not per
  comment row). Clients attach them to each row menu and POST
  `{ topicId, commentId }` to the proxy path.

### Public navigation (E2.3)

- Point: `forum.nav.items`, payload type `navItem`.
- **Merge order (documented):** core / operator-configured `items` first;
  contribution `extensionItems` second (by contribution `order`).
- `hostLink`: site-relative path only — **not** `/api`, **not** `/admin`.
- `extensionRoute`: **GET only** (public page open via extension route proxy).
- Response: `GET /site/nav-items` returns
  `{ items: SiteNavItem[], extensionItems?: SiteExtensionNavItem[] }`.

### Topic list badges (E2.4)

- Same `topicBadge` payload as detail badges (`tone` + optional host `href`).
- Host returns descriptors once on `TopicList.extensionListBadges` (not per
  topic row). Themes attach the same set to each list row; no per-row plugin
  RPC and no custom components.

## Entity meta / custom fields (F4.4)

Host-owned EAV fields on `user` and `topic` — **no** per-plugin `ALTER` on core
tables.

- Admin defines fields (`entity_meta.manage`): key, type
  (`string|text|number|boolean`), visibility (`public|owner|admin`).
- Values: `GET/PUT /api/v1/entity-meta/{entityType}/{entityId}` (visibility
  filtered).
- Observe event: `entity_meta.updated` after successful value writes.

## Feature flags vs permissions (F4.5)

| Concern | Mechanism |
| --- | --- |
| Who may act | RBAC permission keys |
| Whether a product surface is on | `features.*` runtime options |

Declare product prerequisites on plugins only:

```json
"requiresFeatures": ["features.search"]
```

Enable fails with `extension.features_required` if any listed flag is disabled.
Themes must not declare `requiresFeatures`. Public `GET /web-options` exposes
only safe public flags; restore defaults via
`POST /admin/features/restore-defaults`.

## Related docs

- [Host catalogs (generated)](./catalogs/README.md)
- [Manifest V3 schema catalog (generated)](./catalogs/manifest-v3.md)
- [Extension platform v2](../extension-platform-v2.md)
- [Trusted admin components](./trusted-admin-components.md)
- Fixtures README: `extensions/fixtures/README.md`
