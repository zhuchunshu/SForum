# Extension Plugin Runtime Design

## Status

Approved design direction, pending implementation plan.

## Context

SForum already has extension ZIP upload, manifest validation, extension tables,
admin lifecycle pages, and plugin/theme status separation. The current runtime
is intentionally shallow: plugin enable verifies local files and records state,
but it does not start plugin processes, proxy plugin routes, run hooks, or let
plugins participate in core provider decisions.

The next step is to make plugins actually useful without letting them take over
core API contracts. The design keeps a WordPress-like operator workflow
("upload, inspect, enable") while using an out-of-process runtime that matches
the Go API and Nuxt SSR architecture.

## Goals

- Enabled plugins can expose declared HTTP routes under their own namespace:
  `/api/v1/extensions/{extensionId}/*`.
- Enabled plugins can register typed hooks that core modules explicitly emit.
- Enabled plugins can implement controlled provider slots that core modules
  opt into, such as search provider, storage provider, human verification
  provider, or future risk scoring provider.
- Plugin failures do not crash the API process.
- API policy checks remain authoritative in the host API.
- Default behavior stays beginner-friendly: built-in providers remain active
  until an operator explicitly selects a plugin provider, and reset restores
  built-in behavior.

## Non-Goals

- Do not allow plugins to arbitrarily replace existing core routes such as
  `/api/v1/auth/login` or `/api/v1/admin/users`.
- Do not load third-party Go code in-process with `plugin.Open`.
- Do not implement uploaded theme activation in this plugin runtime. Uploaded
  themes still need a separate Nuxt rebuild, health-check, and rollback worker.
- Do not expose extension package files through public attachment URLs.

## Library Survey

- Use HashiCorp `go-plugin` for the plugin RPC boundary. It is designed around
  local subprocess plugins, RPC/gRPC handshakes, protocol versioning, and plugin
  crash isolation. It also matches the existing manifest value
  `backend.rpc = "hashicorp-go-plugin"`.
- Use the existing Fiber/fasthttp stack for request handling. The v1 route
  gateway uses `fasthttp.HostClient` because Fiber v3 runs on fasthttp and the
  dependency is already present in the API module. Avoid adding a large proxy
  package for this narrow localhost proxy boundary.
- Keep Nuxt theme activation aligned with Nuxt Layers and `extends`, but leave
  it outside this plugin runtime scope.

References:

- https://github.com/hashicorp/go-plugin
- https://pkg.go.dev/net/http/httputil
- https://nuxt.com/docs/4.x/getting-started/layers

## Architecture

Add a plugin runtime under `apps/api/app/Support/Extensions`. It owns process
supervision, route dispatch, hook invocation, provider registration, health
state, and runtime events. The existing extension service remains responsible
for install, manifest validation, database state, and permission checks for
admin lifecycle actions.

The runtime has four boundaries:

- `RuntimeSupervisor`: starts, stops, restarts, and health-checks enabled plugin
  subprocesses.
- `RouteGateway`: matches host extension routes, enforces host-side access, and
  proxies requests to the plugin's local HTTP listener.
- `HookBus`: invokes enabled plugins for known hook points with typed payloads.
- `ProviderRegistry`: exposes core-owned provider slots and resolves the active
  implementation for each slot.

The API provider wires the controller to a runtime instance. On API startup, the
runtime starts all enabled plugin extensions whose manifests declare a backend
entry. On shutdown, it stops plugin subprocesses gracefully.

## Manifest Additions

Keep existing manifest fields and extend them in a backward-compatible way:

```json
{
  "backend": {
    "entry": "backend/plugin",
    "rpc": "hashicorp-go-plugin",
    "protocolVersion": 1
  },
  "routes": [
    {
      "path": "/hello",
      "methods": ["GET"],
      "access": "public",
      "timeoutMs": 3000
    },
    {
      "path": "/admin/reindex",
      "methods": ["POST"],
      "access": "permission",
      "permission": "extension.demo.manage",
      "timeoutMs": 10000
    }
  ],
  "hooks": [
    { "name": "extension.enabled" },
    { "name": "user.registered" }
  ],
  "providers": [
    {
      "slot": "search.provider",
      "label": "Demo Search",
      "timeoutMs": 3000
    }
  ]
}
```

Validation rules:

- `backend.entry` must remain inside the installed extension package and must be
  an executable file when the plugin is enabled.
- `backend.rpc` accepts only `hashicorp-go-plugin` for this release.
- `backend.protocolVersion` defaults to the current host plugin protocol.
- Route paths must start with `/`, must not contain `..`, must not be empty
  after normalization, and must not include a scheme or host.
- Route methods must be explicit HTTP methods.
- Route `access` defaults to `login`. Supported values are `public`, `login`,
  and `permission`.
- `public` routes are allowed only for `GET`, `HEAD`, and `OPTIONS`.
- `permission` routes must declare a permission key listed in the extension
  manifest permissions.
- Hook names must be known host hook points.
- Provider slots must be known host provider slots.

## Plugin Routes

Plugins define routes only inside their namespace:

```text
/api/v1/extensions/{extensionId}/{pluginRoutePath}
```

The host API remains the authoritative policy layer. Before proxying a request,
the host verifies:

- The extension exists, is a plugin, and has status `enabled`.
- The plugin process is running and healthy.
- The requested path and method match a declared route.
- The route access rule passes for the current actor.

Policy mapping:

- Actor: anonymous visitor or loaded SForum user.
- Action: `extension.route.invoke`.
- Resource: `{extensionId}:{method}:{path}`.

Failure mapping:

- `401 auth.required` when login is required and there is no session.
- `403 permission.denied` when a permission route fails.
- `404 extension.route_not_found` when no declared plugin route matches.
- `405 extension.route_method_not_allowed` when the path exists but the method
  is not declared.
- `503 extension.runtime_unavailable` when the plugin process is missing,
  unhealthy, or times out.

The proxy strips hop-by-hop headers and sets trusted host-side context headers:

- `X-SForum-Request-ID`
- `X-SForum-Extension-ID`
- `X-SForum-Actor-ID` when logged in
- `X-SForum-Locale`

Plugins must not receive raw session cookies as an authority signal. If a plugin
needs richer user data later, it should use a host RPC capability that enforces
host permissions.

## Hooks

Hooks are typed host extension points, not arbitrary monkey patches. Core code
must explicitly emit a hook before plugins can participate.

Hook categories:

- `observe`: asynchronous notification. Errors are recorded but do not block
  the original operation.
- `validate`: synchronous preflight. A plugin can reject the action with a
  stable reason, and the host maps that reason to the core operation's error
  response.
- `mutate`: synchronous transformation. The hook point defines exactly which
  fields may be changed and validates the returned patch.

The hook point, not the plugin, controls timeout, ordering, failure policy, and
whether sync blocking is allowed. Default timeout is 2 seconds for sync hooks
and 5 seconds for async delivery attempts. Async hooks should be queued through
the existing River job foundation once the event needs durability.

Initial hook points implemented with the runtime:

- `extension.enabled`: observe, emitted after a plugin is enabled and started.
- `extension.disabled`: observe, emitted after a plugin is disabled and stopped.

Reserved hook point names that should be added by the owning module when the
module has a stable service boundary:

- `user.registered`: observe, emitted after a new user is committed.
- `topic.before_create`: validate, emitted before a topic is committed.
- `topic.created`: observe, emitted after a topic is committed.
- `attachment.uploaded`: observe, emitted after attachment metadata commits.

## Provider Slots

Provider replacement is allowed only through named slots owned by core modules.
This gives SForum plugin power without letting plugins shadow arbitrary API
routes.

Provider slot rules:

- Each provider slot defines its interface, timeout, fallback behavior, and
  security policy.
- Built-in providers remain the default.
- Selecting a plugin provider is an explicit admin action on the owning module
  settings page.
- Each settings page must offer one-click restore to the recommended built-in
  default.
- If a selected plugin provider becomes unavailable, the slot follows its
  declared fallback policy. Search can fall back to built-in behavior; security
  slots such as human verification must fail closed unless their module defines
  a safer default.

Candidate slots:

- `search.provider`
- `attachment.storage.provider`
- `human_verification.provider`
- `auth.risk.provider`
- `editor.sanitizer.provider`

The active provider selection is core configuration for the owning module, not
extension-owned settings. Extension-specific settings remain in
`extension_settings`.

## Admin Experience

The plugin admin page shows runtime health in addition to install status:

- `stopped`
- `starting`
- `running`
- `failed`

Rows show route count, hook count, provider count, last runtime error, and a
restart action once the supervisor exists. Non-error success/status toasts keep
the existing 10-second auto-dismiss behavior; errors stay visible until
dismissed or resolved.

## Theme Boundary

Uploaded themes remain installable and verifiable but not activatable as part
of this plugin runtime. The theme activation worker will be a separate design:
write active layer state, run Nuxt build, start preview or health-check target,
switch the serving artifact, and rollback on failure.

## Testing Strategy

Use TDD for implementation.

Required backend tests:

- Manifest validation accepts route/hook/provider declarations and rejects
  unsafe paths, unknown methods, unknown hooks, unknown slots, and public unsafe
  write routes.
- Enabling a plugin starts the runtime after preflight and records runtime
  failures without marking unrelated extensions.
- Disabling a plugin stops its runtime process.
- Startup reconciles enabled plugins and attempts to start them.
- Route gateway enforces login, permission, method, and declared-path checks
  before proxying.
- Route gateway maps plugin-down and timeout cases to
  `extension.runtime_unavailable`.
- Hook bus invokes only enabled healthy plugins that declared the hook.
- Hook bus respects observe, validate, and mutate failure policies.
- Provider registry returns built-in defaults, applies an explicit plugin
  provider selection, and restores built-in defaults.
- OpenAPI ref validation passes after contract changes.

Required frontend tests:

- Plugin admin helpers render runtime status and counts.
- Plugin restart button is enabled only when runtime status exists.
- Provider settings UI keeps built-in defaults obvious and supports restore.

## Implementation Phases

1. Extend manifest structs, OpenAPI schemas, validation, and admin display
   helpers for routes, hooks, providers, and runtime status.
2. Add an in-process fake runtime and `RouteGateway` so route policy and error
   behavior can be tested before subprocess work.
3. Add `RuntimeSupervisor` with HashiCorp go-plugin handshake, start/stop,
   health state, startup reconciliation, and shutdown cleanup.
4. Replace `extension.route_unavailable` with declared route dispatch and
   proxying to healthy plugin processes.
5. Add `HookBus` and initial lifecycle hooks.
6. Add provider slot registry with built-in defaults and one explicit slot
   integration.
7. Update admin UI, OpenAPI, knowledge base, and session handoff.

## Acceptance Criteria

- An uploaded plugin with a valid backend entry and declared `GET /hello` route
  can be enabled and invoked at `/api/v1/extensions/{id}/hello`.
- Undeclared plugin paths do not proxy.
- Plugin routes cannot override or shadow core SForum API routes.
- A disabled plugin no longer receives route traffic or hooks.
- At least one observe hook is emitted and delivered to an enabled plugin.
- At least one provider slot can keep the built-in default, select a plugin
  provider, and restore the built-in default.
- Tests prove allowed and denied access paths for plugin route invocation.
- The knowledge base states that uploaded themes remain separate from plugin
  runtime activation.
