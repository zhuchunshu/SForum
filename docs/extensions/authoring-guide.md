# Plugin authoring guide

This guide is the hand-written entry point for building SForum plugins (Wave
F4.2). Generated host catalogs live under
[catalogs/](./catalogs/README.md) — always regenerate those from code after
changing events, capabilities, contribution points, provider slots, or core
schedules.

## What you should depend on

| Surface | Use for |
| --- | --- |
| Manifest `sforum.extension.json` (+ optional `includes`) | Identity, settings, routes, events, jobs, providers, contributions, capabilities |
| Go SDK `github.com/zhuchunshu/sforum/apps/api/sdk/plugin` | Backend `Serve`, Host API client, read-only catalogs, contract test helpers |
| Host API `sforum.host/v1` | Permission checks, own settings, enqueue declared jobs, audit, safe user reads |
| go-plugin RPC (`Health`, `RouteTarget`, `InvokeHook`, `SendMail`) | Process protocol the host starts and health-checks |
| Published catalogs | Which events/points/slots/capabilities exist |

**Do not** import `app/Models/*` or other host business packages from a
third-party plugin. Built-in SMTP still uses internal packages for historical
reasons; new plugins should follow the SDK path shown by the Host API fixture.

## Quick start

```bash
# Scaffold (from repo root)
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

# After you change host catalogs, refresh docs
go run ./cmd/sforum extension docs generate
go run ./cmd/sforum extension docs generate --check
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

## Reference 1 — built-in SMTP (`sforum.smtp`)

Path: `extensions/builtin/plugins/sforum-smtp/`

Use this package when you need a **provider-slot** plugin with settings and
outbound network:

| Area | What SMTP does |
| --- | --- |
| Manifest | `providers: [{ slot: "mail.provider", ... }]`, explicit `capabilities` |
| Multi-file layout | Thin root + `includes` for langs, settings, admin, frontend, contributions |
| Backend | Implements `SendMail` over go-plugin; no public HTTP `RouteTarget` |
| Settings | Host injects `SFORUM_SETTING_*` env vars into the child process |
| Admin UI | Trusted admin components under `frontend/admin` for settings chrome |

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

Read the full package README and `manifest/` shards for settings grouping and
Chinese/English identity langs.

## Reference 2 — Host API fixture (`sforum.contract.hostapi`)

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

## Manifest checklist

1. **Identity** — stable `id`, `version`, `sforumVersion`, URL, author, description.
2. **Type** — `plugin` vs `theme` (themes must not declare capabilities, routes, events, backend jobs).
3. **Capabilities** — only keys from [capabilities.md](./catalogs/capabilities.md); prefer explicit keys for operator review.
4. **Events / hooks** — only names from [events.md](./catalogs/events.md); match host `kind`.
5. **Contributions** — only points from [contribution-points.md](./catalogs/contribution-points.md); host-owned payloads.
6. **Providers** — only published [slots](./catalogs/provider-slots.md).
7. **Jobs** — declare every kind you will enqueue via Host API.
8. **Backend** — `hashicorp-go-plugin`, entry path present after build.
9. **Settings** — defaults and recommended values so first-time operators are not blocked.

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
| Side effects after commit | `user.registered`, `topic.created`, `topic.updated`, `comment.created`, … | observe | — | Async; do not use for rejection |

Reject with a stable `reason` code (mapped to the API error envelope as 422). Do not put passwords, raw file bytes, or stack traces in filter/validate payloads or messages.

## Schedules

Do not start `time.Ticker` / cron loops in the plugin process. Core schedules
are listed in [schedules.md](./catalogs/schedules.md). Plugin-owned periodics
will go through the host registry in a later wave.

## Admin / trusted components

If you ship Vue into admin slots, read
[trusted-admin-components.md](./trusted-admin-components.md). That path is
**full trust** on the admin origin; backend permissions remain authoritative
but do not sandbox the browser code.

## Validation commands

```bash
# Parse + merge includes, print summary
go run ./cmd/sforum extension validate <package-root>

# Host contract report (exit 1 on errors)
go run ./cmd/sforum extension test <package-root>
go run ./cmd/sforum extension test --json <package-root>

# Catalog documentation drift
go run ./cmd/sforum extension docs generate --check
```

## Contribution points (F4.3)

Declare only host-owned points (see [contribution-points.md](./catalogs/contribution-points.md)).
Payloads are JSON descriptors — **never executable code**.

| Point | Payload type | Runtime consumer |
| --- | --- | --- |
| `forum.topic.actions` | `extensionRoute` | Topic detail `extensionActions` |
| `forum.composer.toolbar` | `extensionRoute` | `GET /composer/toolbar` + composer UI |
| `forum.profile.tabs` | `profileSection` (`extensionRoute` \| `hostLink`) | Public profile `extensionTabs` |
| `admin.dashboard.widgets` | `dashboardLink` (`adminLink` + admin route) | Admin overview `extensionWidgets` |
| `system.health.checks` | `healthDescriptor` (`static` \| `extensionRuntime`) | Merged into `GET /ready` without plugin RPC |

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
- [Extension platform v2](../extension-platform-v2.md)
- [Trusted admin components](./trusted-admin-components.md)
- Fixtures README: `extensions/fixtures/README.md`
