# Trusted Admin Plugin Runtime Design

## Status

Approved design direction for the first of two dependent projects.

This document defines the trusted frontend plugin runtime. The River job
monitoring and management module is a separate follow-up project and will be
the first production consumer of the component slot runtime.

## Context

SForum plugins currently extend the host through manifests, subprocess RPC,
provider slots, events, typed contributions, namespaced routes, settings, and
host-rendered admin pages. The web app does not load plugin-owned Vue, HTML, or
JavaScript. Existing admin pages support only host-rendered `about` and
`settings` views, and the existing `forum.topic.actions` contribution point is
a host-rendered action descriptor.

The job monitoring project needs enabled plugins to contribute arbitrary Vue
components directly to core-owned job table columns, row actions, and job
detail sections. Descriptor-only actions are intentionally insufficient for
this requirement.

Arbitrary Vue code in the admin origin is not a sandboxed capability. It can
access the DOM, page state, and APIs available to the current administrator.
SForum will therefore treat this code as fully trusted frontend code, require
an explicit super administrator grant, and include it in an auditable Nuxt
release built from static imports.

## Goals

- Let plugins export arbitrary client-side Vue components for core-declared
  admin component slots.
- Keep the insertion locations, slot context, ordering, and layout constraints
  owned by core modules.
- Bind trust to the exact extension version and package content digest.
- Rebuild the complete Nuxt/Nitro artifact when the active theme or trusted
  frontend plugin set changes.
- Preserve the currently active site when dependency installation, build,
  verification, activation, or runtime startup fails.
- Reuse the existing River worker and theme release infrastructure instead of
  introducing a second competing artifact switcher.
- Keep existing plugins and themes backward compatible.

## Non-Goals

- Implement the job monitoring API or admin page in this project.
- Add public forum, authentication, layout, navigation, or error-page slots.
- Let plugins override arbitrary core pages, routes, layouts, components,
  Nuxt configuration, Vite configuration, or Nitro behavior.
- Load remote components at runtime through ESM or Module Federation.
- Run plugin components during server-side rendering.
- Execute npm lifecycle scripts or support dependencies that require them.
- Add extension signatures, a publisher trust chain, or a marketplace.
- Support multi-node API or web rollout. V1 keeps the existing single-node
  self-hosted process topology with separate API, worker, and web processes.
- Turn the existing backend subprocess boundary into an operating-system
  security sandbox.

## Library And Framework Survey

The runtime should use existing mature project facilities:

- Nuxt and Vite already support static imports, code splitting, client-only
  plugins, and generated modules. A generated registry is sufficient; a
  separate component loader dependency is unnecessary.
- Bun is already the web package manager and build runner. It supports frozen
  lockfile installs and disabling package lifecycle scripts.
- River already runs long-lived extension build work. The existing `theme`
  queue remains the serialized web build queue for compatibility.
- PostgreSQL is the authoritative store for extension state, trust grants,
  release history, and global build locking.
- The existing ThemeRuntime builder, preview health check, atomic current file,
  and rollback conventions should be generalized into WebReleaseRuntime.

Rejected alternatives:

- A Nuxt Layer or Module per plugin would allow plugins to shadow core files
  and mutate build configuration, exceeding the accepted slot boundary.
- Runtime ESM or Module Federation would avoid rebuilds but introduce SSR,
  dependency sharing, CSP, cache invalidation, and recovery complexity without
  reducing same-origin trust.
- Sandboxed iframes would provide a stronger isolation boundary but would not
  satisfy direct Vue component insertion into core tables.

## Trust Model

An uploaded plugin that declares an admin frontend is untrusted by default.
Only an active `super_admin` may grant or revoke frontend trust.

A grant is bound to:

- Extension ID.
- Extension version.
- Canonical package content digest.
- Admin frontend API version.
- Declared trusted component contribution points and IDs.
- Granting user and timestamp.

The package digest covers the complete immutable installed extension version,
including its normalized manifest, backend files, frontend files, local
assets, `package.json`, and `bun.lock`. Files are hashed as sorted normalized
relative paths plus file mode and bytes; directory metadata and modification
times are excluded. Uploaded package trees may not contain symbolic links. Any
package file change invalidates the grant. Protected built-in plugins are
trusted by source policy and do not need an interactive grant.

The trust request includes the digest shown to the administrator. The service
recalculates it immediately before insert and rejects a mismatch, preventing a
package change between review and grant.

Every build recalculates the digest from the copied immutable package before
dependency installation and compares it with the grant. The builder also
records a resolved dependency snapshot digest derived from the frozen lockfile,
package integrity data, Bun version, and host peer versions.

This digest proves which installed package and dependency inputs were approved;
it is not a browser sandbox or a closed-world proof. Fully trusted code may
still perform network behavior allowed by host policy. The grant screen states
that limitation.

Users with `extension.manage` may enable, disable, retry, or roll back a plugin
whose exact package digest still has a valid grant. They cannot grant or revoke
trust.

Revocation is two-phase. A request immediately marks the grant as
`revocation_pending`, excludes it from every new composition and rollback
target, and queues a release without its components. `revoked_at` becomes
final after the safe release activates. A failed safe build leaves the request
pending with a blocking warning and a retry action; it does not silently clear
the revocation request.

Trust means the plugin may execute arbitrary browser code with the authority of
the current administrator. Backend APIs remain authoritative and continue to
enforce their own actor, action, and resource policies, but trusted frontend
code can invoke any API the current administrator can invoke.

## Architecture

### Manifest And Trust Service

The extension manifest parser validates the frontend declaration, safe package
paths, API compatibility, required package files, and trusted component
contributions. The trust service calculates canonical package digests and owns
trust grants and revocation.

### Admin SDK And Slot Catalog

The Admin SDK exports:

- The `AdminSlotProps<Point>` type mapping for Vue component props.
- `useSForumAdminHost` for the supported runtime capability object.
- Slot context and slot option mappings.
- The current Admin SDK API version.

Core modules own a typed slot catalog. A slot definition specifies its stable
ID, owning module, context type, options type, multiplicity, ordering, layout
constraints, and error fallback behavior. Plugins cannot create slots.

The infrastructure project implements the generic catalog, registry, and slot
renderer. Production `admin.jobs.*` entries and their job DTO types are owned
by the follow-up jobs project so this project does not freeze an incomplete job
contract. End-to-end tests use a test-only slot catalog entry that is excluded
from production builds.

### Web Release Planner

The planner computes one immutable desired composition containing:

- The active theme layer.
- Every enabled trusted admin frontend plugin.
- Each plugin version, package digest, component map, lockfile digest,
  contribution points, and component IDs.
- The SForum web source revision, root web lockfile digest, Admin SDK version,
  Bun version, and build-contract version.

The planner produces a deterministic composition hash. A request matching an
active or already queued composition is idempotent.

### Web Release Builder

The builder runs only in the worker process. It creates an isolated workspace,
copies the selected immutable extension package versions, recalculates their
digests, installs dependencies with frozen lockfiles and lifecycle scripts
disabled, generates the static component registry, runs Nuxt typecheck and
build, starts a preview server, and verifies release health. It then records an
artifact digest and moves the release to `ready`; it never starts plugin
backends, changes effective extension state, or writes the current pointer.

### Generated Component Registry

The generator builds two artifacts. An SSR-safe metadata registry comes only
from validated manifest contributions and contains no plugin code. A client
registry maps manifest component IDs to safe package-relative Vue module paths
and imports them only from a Nuxt `.client.ts` plugin. Server-side rendering
uses metadata to emit stable headers, layout, and placeholders without
evaluating plugin modules.

Contribution order is deterministic:

1. Contribution `order`.
2. Extension ID.
3. Contribution ID.

Unknown slots, duplicate contribution IDs, incompatible SDK versions, missing
component mappings, unused component mappings, invalid module paths, and
module-graph escapes fail the build before activation.

### Host Slot Renderer

`SFAdminExtensionSlot` receives a core-owned slot ID and typed context. During
SSR, it uses host-validated manifest metadata for stable layout and
placeholders. In the browser, it loads matching client registry components and
renders each contribution inside an independent host error boundary. The host
interprets slot options such as column label, order, and width; the plugin
component owns its internal Vue UI.

### Web Release Runtime

Theme layers and trusted admin plugins are inputs to the same Nuxt/Nitro
artifact. ThemeRuntime therefore becomes a focused WebReleaseRuntime rather
than adding a parallel plugin artifact switcher.

The runtime owns build input assembly, preview verification, release
coordination, process switching, and rollback. Existing theme release records
remain as extension-domain history, while `web_releases` becomes the canonical
durable planning and history record for complete web artifacts. Supervisor
acknowledgement remains authoritative for actual web traffic.

### API-Owned Activation Coordinator

The API process already owns HashiCorp plugin subprocesses and in-memory route
targets. A `WebReleaseCoordinator` inside the API is therefore the only process
allowed to start or stop plugin backends, change effective extension state, or
request a web switch. The worker stops at `ready`; the Node web supervisor owns
only child web processes and proxy targets.

The coordinator holds a PostgreSQL advisory lock, reconciles ready and
activating releases, stages the required plugin runtime, writes the desired
release pointer, waits for the supervisor's durable acknowledgement, and then
finalizes or compensates database state.

## Manifest Contract

The existing `frontend.layer` field remains supported for themes. Plugins may
add an admin component map and trusted component contributions. This example
shows the follow-up jobs consumer after the jobs module has registered its
production slot contracts:

```json
{
  "frontend": {
    "admin": {
      "root": "frontend/admin",
      "apiVersion": 1,
      "components": {
        "latency-column": "components/JobLatencyCell.vue"
      },
      "locales": {
        "zh-CN": "locales/zh-CN.json",
        "en-US": "locales/en-US.json"
      }
    }
  },
  "contributions": [{
    "point": "admin.jobs.table.columns",
    "id": "latency-column",
    "order": 100,
    "label": {
      "zh-CN": "\u5ef6\u8fdf",
      "en-US": "Latency"
    },
    "payload": {
      "component": "latency-column",
      "width": 120
    }
  }]
}
```

Validation rules:

- Only plugin manifests may declare `frontend.admin` in this release.
- `root` is a safe package-relative directory.
- `root/package.json` and `root/bun.lock` are required.
- `apiVersion` must be a positive host-supported version.
- Component IDs match `^[a-z0-9][a-z0-9._-]{0,80}$` and are unique within the
  plugin. Component module paths are relative to `root`, remain inside it after
  normalization, and end in `.vue`, `.ts`, `.tsx`, `.js`, or `.jsx`.
- Admin frontends provide `zh-CN` and `en-US` locale JSON files. Additional
  locale keys are allowed when their paths remain inside `root`. Locale files
  contain nested objects with string leaves and are namespaced by extension ID.
- Every trusted component contribution point must exist in the installed core
  slot catalog and accept its manifest payload.
- A trusted component contribution must declare a non-empty component ID in
  its payload, and that ID must exist in `frontend.admin.components`.
- Every component map entry must be referenced by exactly one trusted component
  contribution in V1.
- Frontend source trees may not contain symbolic links.
- Dependency specifications using `workspace:`, `file:`, or `link:` are
  rejected. Lockfile resolution must pin all allowed dependencies.
- Vue, Nuxt, Nuxt UI, Vue Router, and the SForum Admin SDK are host-provided
  peers. They may not appear as private runtime dependencies with competing
  copies, and declared peer ranges must match the host release.
- The declaration is rejected if the extension archive does not contain every
  required path.
- The Vite module graph for a component may contain its admin frontend root,
  its isolated locked dependencies, and host-provided peers only. Local imports
  that escape the admin root are rejected.

The existing typed contribution registry remains the source of safe SSR
metadata, admin inspection, order, and point validation. Trusted component
points differ only in that an active Web Release may attach a client component
mapping after full-trust approval.

The point shown above is reserved for the jobs project. It becomes an accepted
production catalog entry only when that project defines its complete typed
context and host consumer.

## SDK Contract

The SDK package import specifier is `@sforum/admin-sdk`. The package uses
semantic versions, while manifest `apiVersion` represents its breaking contract
generation. Additive minor releases remain compatible; a breaking change
increments `apiVersion`, and the host rejects unsupported generations before
build.

A slot-owning core module augments `AdminSlotContextMap` and
`AdminSlotOptionsMap`. A component uses the point-specific prop type:

```ts
import type { AdminSlotProps } from '@sforum/admin-sdk'

defineProps<AdminSlotProps<'admin.jobs.table.columns'>>()
```

The supported runtime export is `useSForumAdminHost()`, which returns:

- `extensionId` for the component owner.
- Current `locale` and a `t` translator restricted to the owning extension's
  embedded locale namespace.
- `navigate` for SForum admin routes.
- `toast` with project dismissal and theme behavior.
- `extensionRequest` for the owning plugin's declared extension routes through
  the existing host API and CSRF pipeline.

The SDK never exposes session cookies, server-only state, River clients, raw
database access, permission bypasses, or arbitrary host service lookup.
Components may use public Vue, Vue Router, Nuxt client, and Nuxt UI APIs, but
imports from SForum app-internal aliases such as `~/`, private `#build` output,
or core source paths are unsupported and rejected by module-graph validation.

Manifest contributions are the install-time capability and SSR layout
declaration. Their component IDs map directly to manifest module paths, so
missing, unused, or escaping mappings are validated without executing plugin
code in Node or a preview browser.

Plugin components may use Vue, Nuxt client composables, Nuxt UI, the Admin SDK,
local source, local assets, and locked npm dependencies. They cannot export
Nuxt modules or build hooks through this contract.

The generated build aliases and deduplicates host-provided peers so every
component uses the same Vue, router, Nuxt UI, and Admin SDK instances as core.

## Dependency Installation

Each plugin admin frontend is installed from its own package root:

```text
bun install --frozen-lockfile --ignore-scripts
```

The install step may use the deployment's configured package registry and
proxy environment. The builder forwards only the proxy and registry variables
needed for installation. The Nuxt build step receives a separate strict
allowlist of public build variables and never inherits database URLs, session
secrets, option encryption keys, signing material, or unrelated process
environment.

Install and build output is bounded and scrubbed against known sensitive
values before persistence. A dependency that requires a lifecycle script or
native post-install build is unsupported in this release and fails with an
actionable compatibility message.

Plugins cannot supply a Nuxt config, Vite plugin, Nitro plugin, server route,
server middleware, or package-manager install hook.

Plugin manifests cannot relax the host Content Security Policy or add browser
network origins. External integrations use declared, policy-checked extension
API routes so browser credentials and raw session cookies remain host-owned.

The isolated workspace prevents accidental dependency and output
contamination; it is not an operating-system security sandbox. The package has
already received a full-trust grant. Deployments may add container isolation,
but the core contract does not claim to make malicious trusted code safe.

## Persistence Model

### `extension_frontend_trust_grants`

Stores extension ID, extension version, package digest, API version, declared
component contribution points and IDs, granting user, grant time, revocation
request time and actor, final revocation time, and revoking user. Only one
non-revoked grant may exist for an exact extension version and digest.

### `web_releases`

Stores release ID, trigger kind, trigger extension ID, desired generation,
composition hash, active theme ID, theme version, theme layer path, theme
package digest, status, activation checkpoint, artifact path, artifact digest,
server entry, build log, public error reason, previous release ID, requesting
actor, activation actor, and timestamps.

The release-level activation checkpoint records global coordinator progress
such as pointer write and supervisor acknowledgement. Per-extension effect
checkpoints record plugin runtime preparation and effective status changes.

Supported states are:

- `queued`
- `resolving`
- `installing`
- `building`
- `verifying`
- `ready`
- `activating`
- `active`
- `inactive`
- `failed`
- `superseded`
- `rolled_back`

Only one release is active. Failed, superseded, inactive, and rolled-back
states are immutable. An active release moves to `inactive` after a normal
replacement or `rolled_back` when compensation replaces a failed activation.

Retry and rollback requests always create a new queued release with a new ID.
They never mutate a failed, superseded, or rolled-back release back into a
running state.

### `web_release_extensions`

Stores the immutable plugin snapshot for a release: extension ID, version,
package digest, frontend root, component map, API version, trusted component
metadata, locale map digest, lockfile digest, resolved dependency snapshot
digest, and deterministic order metadata.

### `web_release_extension_effects`

Stores backend lifecycle effects that must coordinate with a release:
extension ID, previous effective status, target effective status, and the
latest durable activation checkpoint. Most releases have zero or one effect;
the structure also supports a future coordinated rollback without inferring
backend state from frontend component membership.

### `web_release_events`

Stores immutable release transitions and operator actions: release ID, actor
when present, previous state, next state, stable reason, cleaned message, and
timestamp. Release detail responses use this table for the build and activation
timeline.

## Artifact And Process Topology

V1 supports one API runtime owner, one web supervisor, and one or more build
workers on one deployment node. API, worker, and web services mount the same
read-write `WEB_RELEASE_ROOT`. API and worker services also mount the immutable
extension package root; the web service sees built artifacts only. Multi-node
artifact distribution and coordinated proxy switching are outside V1.

The release root contains three durable signals:

- `current.json`: the desired release, written atomically by the API
  coordinator. It contains release ID, composition hash, artifact path,
  artifact digest, server entry, active theme identity, activation request
  time, and reload mode (`prompt` or `force`).
- `active.json`: the release actually receiving web traffic, written atomically
  by the web supervisor after candidate health and stabilization checks.
- `failures/{releaseId}.json`: a candidate switch failure acknowledgement from
  the web supervisor. The old proxy target remains active.

Every artifact has an immutable release manifest and digest. The supervisor
checks both before starting a candidate. `current.json` expresses intent;
`active.json` is authoritative for the web artifact currently serving traffic.
The database records desired state, history, and reconciliation checkpoints,
but is not allowed to contradict a valid `active.json` acknowledgement.

## Lifecycle And Concurrency

Build and activation are separate durable phases:

1. Enabling or disabling a trusted frontend plugin, switching a theme,
   granting trust to an already enabled plugin, revoking an active frontend,
   restoring defaults, or requesting a rebuild computes a new desired
   generation.
2. The service creates a queued `web_release` and enqueues an
   `extension.web_release_build` River job.
3. The job uses the existing `theme` queue for backward compatibility and also
   acquires a PostgreSQL build advisory lock. The queue setting alone is
   insufficient because multiple worker instances may each run one queue
   worker.
4. The worker recalculates package digests, resolves and installs dependencies,
   records the resolved dependency snapshot, generates registries, runs
   typecheck and build, and verifies a preview.
5. Before and after expensive build work, the worker compares the desired
   generation. A stale release becomes `superseded` and never reaches
   activation.
6. A verified artifact moves to `ready`. The worker persists the artifact
   manifest and digest and stops.
7. The API coordinator acquires a separate activation advisory lock, rechecks
   generation and trust, applies the operation-specific runtime preparation,
   and advances durable activation checkpoints.
8. The API writes `current.json`. The web supervisor verifies the artifact,
   starts a candidate child, checks it, switches the proxy only after a stable
   readiness window, and then writes `active.json`. A failed candidate writes a
   failure acknowledgement and leaves the old child serving.
9. The coordinator observes the acknowledgement and commits the release as
   active, moves the former active release to inactive, finalizes pending trust
   revocations, and records transition events. Failure acknowledgement triggers
   idempotent compensation.

Operation-specific ordering preserves usable authority:

- Enable: the API coordinator starts and health-checks the plugin backend,
  records `runtime_prepared`, commits the backend effective status, records
  `effective_committed`, and only then requests the web switch. Old UI does not
  expose the new component during the short preparation window. A failed switch
  restores the previous status and stops the staged runtime.
- Disable: the coordinator switches to the artifact without the component and
  waits for `active.json` before committing the disabled backend status and
  stopping its runtime. Old browser tabs then receive the existing explicit
  disabled response.
- Theme switch or frontend-only grant/revoke: the coordinator requests the web
  switch first and commits theme or trust finalization only after
  acknowledgement.

`extensions.status` describes effective backend lifecycle state. Pending UI
states such as `enabling`, `disabling`, `grant_activating`, and
`revocation_pending` derive from release and effect checkpoints. The active web
composition derives from the supervisor acknowledgement, not from an early
extension status update.

A rollback target is eligible only if every included plugin digest still has
valid trust and no pending revocation. Rollback creates a new release; it never
points directly at an unchecked historical artifact.

## Crash Recovery And Reconciliation

Every phase is idempotent by release ID and checkpoint. Partial build
directories belong to one release and can be replaced on River retry. The API
coordinator runs reconciliation at startup and periodically while active:

1. Acquire the activation advisory lock.
2. Read the database release/effect checkpoints plus `current.json`,
   `active.json`, and any candidate failure acknowledgement.
3. Wait for the web supervisor to publish `active.json` at cold start; do not
   guess the serving artifact from database status alone.
4. If `active.json` names a verified activating release, finalize it and mark
   the previous release inactive even when the prior API process crashed before
   writing those transitions.
5. If an activating release has not written `current.json`, resume from its
   last runtime/effective checkpoint. If `current.json` exists but the
   supervisor reports failure, restore the previous desired pointer and run
   compensation exactly once.
6. Reconcile the API-owned plugin RuntimeManager to effective extension status;
   API restart therefore recreates all enabled subprocesses and route targets.
7. If an acknowledgement references a missing, mismatched, or unverified
   artifact, refuse it, retain the last valid active release, and surface a
   blocking operational error.

These rules cover crashes before pointer write, after pointer write, after web
switch, and before final database transition. The supervisor owns actual web
traffic, the API owns plugin runtime state, and PostgreSQL owns durable desired
state and history.

## Permissions And API Policy

Read operations require `extension.manage` and normal admin access. Existing
enable, disable, retry, and eligible rollback operations require
`extension.manage`.

Grant and revoke operations require an active super administrator, enforced in
the API service even if a future role happens to contain a similarly named
permission. Frontend button visibility is only a usability aid.

Every unsafe endpoint uses the existing session and CSRF pipeline. Allowed and
denied paths receive explicit tests.

The recommended frontend default is protected built-in components only, with
no uploaded plugin frontend grants. A super-administrator-only restore action
marks all uploaded frontend grants for revocation and builds that recommended
composition. Final revocation occurs after activation. It preserves installed
packages, extension settings, credentials, and backend plugin enabled states;
the confirmation explains those preserved values.

Trust grants, revocations, and extension lifecycle requests are written to the
existing extension event stream. Trust grant rows retain structured actor and
digest data. Every release state change, including restore defaults,
activation, failure, supersede, retry, and rollback, is written to
`web_release_events` with its structured release linkage.

## Admin Experience

The Extensions area gains a Trusted Frontend panel for plugins and a Web
Releases page.

The plugin panel shows:

- No declaration, trust required, trusted, grant invalidated, queued,
  resolving, installing, building, verifying, ready, activating, active,
  inactive, failed, superseded, and rolled-back states as applicable.
- The package digest, frontend root, component map, Admin SDK version, declared
  contribution points and component IDs, dependency summary, and current
  release.
- Grant or revoke actions for super administrators.
- Enable, disable, retry, and eligible rollback actions for extension managers.
- A restore recommended frontend defaults action for super administrators,
  with explicit notice that plugin settings, credentials, and backend runtimes
  are preserved.

The grant confirmation presents the plugin author, source, version, digest,
frontend root, component module paths, contribution points and component IDs,
direct and transitive dependency summary, disabled lifecycle scripts, and an
explicit same-origin code execution warning.

The Web Releases page lists release composition, trigger, status, duration,
previous release, and cleaned build logs. It uses server-side pagination.

Enable and rebuild endpoints return HTTP 202 with the release summary. The UI
polls only while a release is in a non-final build state, prevents overlapping
refreshes, and stops polling on active, inactive, failed, superseded, or
rolled-back state or component unmount. Queued, successful, retried, and
rolled-back user actions use theme-aware Toasts with a ten-second duration.
Blocking trust guidance and build failures remain inline. Error Toasts do not
auto-dismiss.

## Runtime Error Handling

Generated plugin component modules are client-only. SSR renders stable host
placeholders and does not import or evaluate plugin modules.

Each contribution is wrapped in a host boundary. A component load, render,
setup, or captured descendant error replaces only that contribution with a
fallback containing the extension ID, contribution ID, retry action, and link
to plugin management.

Failures are counted per web release, extension, contribution, and browser
session. After three failed render attempts, the contribution is quarantined
for that session until the user explicitly retries or a new web release loads.

An in-realm boundary cannot stop a deliberate infinite loop or memory
exhaustion. The grant screen states this limitation.

The admin shell compares its embedded release ID with a lightweight current
release endpoint. Ordinary changes request a reload. Trust revocation and
security rollback force a reload after the safe release becomes active. Code
already executing in an old tab cannot be recalled before that reload; this is
part of the fully trusted code warning.

Nuxt serves `GET /__sforum/admin-release` with `Cache-Control: no-store`. Its
non-sensitive response contains the current release ID and reload mode. This is
a web runtime endpoint rather than a Go API contract.

## API Contract

Add these admin endpoints to the modular extensions OpenAPI contract:

- `GET /api/v1/admin/extensions/{extensionId}/frontend`
- `POST /api/v1/admin/extensions/{extensionId}/frontend/trust`
- `DELETE /api/v1/admin/extensions/{extensionId}/frontend/trust`
- `GET /api/v1/admin/web-releases`
- `GET /api/v1/admin/web-releases/{releaseId}`
- `POST /api/v1/admin/web-releases/{releaseId}/retry`
- `POST /api/v1/admin/web-releases/{releaseId}/rollback`
- `POST /api/v1/admin/web-releases/restore-defaults`

Existing plugin enable, plugin disable, and theme activation endpoints remain
the operator entry points. When an operation changes the web composition they
return HTTP 202 with a Web Release summary instead of reporting completion
before activation.

Trust lifecycle responses are explicit:

- Granting trust to a disabled plugin stores the grant and returns HTTP 200; a
  later enable request creates the release.
- Granting trust to an already enabled plugin creates the desired composition
  immediately and returns HTTP 202 with its release.
- Revoking a frontend that is absent from the active release finalizes
  immediately with HTTP 200.
- Revoking a frontend present in the active release records pending revocation,
  creates the safe composition, and returns HTTP 202. Final revocation follows
  supervisor acknowledgement.
- Restore defaults returns HTTP 200 when already at the recommended composition
  and HTTP 202 when it queues a release.

Stable error reasons include trust required, trust invalidated, unsupported
Admin SDK version, invalid frontend package, dependency install failure, build
failure, verification failure, stale release, ineligible rollback, and release
activation failure. Backend-localized messages follow the existing API
envelope.

## Migration And Compatibility

The manifest addition is optional. Plugins without `frontend.admin` behave
exactly as they do today. Themes continue to use `frontend.layer`.

Deployment migration proceeds in this order:

1. Add trust and web release tables, manifest parsing, and API response fields.
2. Introduce `WEB_RELEASE_ROOT`, default it to the existing theme release root,
   and mount the same read-write volume into API, worker, and web services.
3. Add WebReleaseRuntime, supervisor acknowledgement files, and the API-owned
   coordinator while retaining reads of the legacy theme current pointer.
4. Create an initial Web Release for the currently active theme and empty
   trusted plugin set.
5. Update the development supervisor and production runtime to prefer the new
   Web Release pointer and fall back to the legacy theme pointer during the
   compatibility period.
6. Route uploaded theme activation through the Web Release builder and API
   activation coordinator.
7. Remove legacy pointer writes only after existing theme activation and
   rollback regression tests pass against the unified runtime.

Immutable installed extension versions and previous web artifacts remain
available while referenced by an active or rollback-eligible release. V1 keeps
the active artifact, its eligible rollback target, and the five newest
successful artifacts. Unreferenced failed or superseded artifacts are removed
after seven days. Release metadata and transition events remain; cleaned build
logs are retained for thirty days. These are fixed recommended V1 defaults
rather than operator settings.

## Testing Strategy

Implementation follows test-driven development.

Backend unit and integration coverage includes:

- Manifest normalization and validation for safe roots, component maps, API
  versions, bilingual locale maps, lockfiles, dependency protocols, and known
  slots.
- Complete package digest stability, copied-package revalidation, resolved
  dependency snapshot integrity, and automatic grant invalidation after any
  package file change.
- Local module graph containment inside the admin frontend root and rejection
  of unsupported host-internal imports.
- Pending revocation exclusion from new compositions and final revocation only
  after safe release activation.
- Super-administrator-only grant and revoke behavior.
- Super-administrator-only restore defaults behavior, including preserved
  settings, credentials, installed packages, and backend enabled states.
- Allowed and denied `extension.manage` lifecycle operations.
- Deterministic composition hashes and idempotent duplicate requests.
- Separate global build and API activation advisory locks.
- API coordinator ownership of plugin runtime preparation and effective state.
- Every valid release state transition and rejection of invalid transitions.
- Normal active-to-inactive replacement and compensation-only rolled-back
  transitions.
- Immutable release transition event history.
- Superseding stale desired generations.
- Enable, disable, theme switch, pointer failure, and post-switch rollback
  compensation.
- Crash injection and restart reconciliation at every activation checkpoint:
  before pointer write, after pointer write, after supervisor switch, and before
  final database transition.
- `current.json`, `active.json`, failure acknowledgement, and artifact digest
  mismatch handling.
- Trust and revoke HTTP 200/202 behavior for disabled, enabled, inactive, and
  active plugin frontends.
- Rollback rejection after trust revocation.
- Retry and rollback creation of new immutable release records.
- Sanitized install and build environments, bounded logs, and sensitive value
  redaction.
- Source path rejection and rejection of every frontend source symlink.
- OpenAPI reference validation.

Frontend coverage includes:

- Registry ordering and exclusion of disabled, unknown, or untrusted entries.
- SSR metadata generation from manifests without plugin module evaluation.
- Component map and contribution validation diagnostics.
- The exact `@sforum/admin-sdk` export surface and rejection of unsupported
  private host imports.
- Client-only loading with no plugin execution during SSR.
- SSR slot metadata and placeholders that prevent hydration-time table shape
  changes.
- Typed slot context and host option handling.
- Independent contribution error boundaries and three-failure quarantine.
- Release change reload behavior.
- Prompt and forced reload behavior from `/__sforum/admin-release`.
- Trust UI permission visibility, risk content, progress polling, cleanup, and
  bilingual text.
- Theme-aware success Toasts and persistent error feedback.

Build integration coverage uses a fixture plugin and an ephemeral local package
registry to verify frozen dependency installation without public network
access, disabled lifecycle scripts, host peer deduplication, registry generation,
Nuxt typecheck and build, preview health, shared-volume artifact visibility,
supervisor acknowledgement, API reconciliation, and rollback.

Browser verification covers desktop and mobile viewports in light and dark
mode. A test-only slot consumer renders a real fixture Vue component so the
runtime is verified before the jobs project adds production slots.

Required regression gates include the existing theme activation tests, default
theme restore tests, plugin backend lifecycle tests, dynamic extension admin
page tests, Nuxt typecheck and build, OpenAPI reference validation, and
`./scripts/test.sh`.

## Acceptance Criteria

- An uploaded plugin admin frontend cannot enter a build without a matching
  super administrator grant for its exact digest.
- Enabling a trusted plugin returns a queued release while the current site
  remains available.
- A successful build activates one complete artifact containing the active
  theme and exact trusted plugin set.
- Install, build, preview, switch, and backend runtime failures preserve or
  restore the previous complete release.
- A changed package digest invalidates the grant before build.
- Every build revalidates the complete copied package and resolved dependency
  snapshot before producing a ready artifact.
- The worker never starts plugin backends or switches the serving pointer; the
  API coordinator owns activation and the web supervisor owns proxy traffic.
- Crash recovery reconciles database checkpoints with the supervisor's actual
  active acknowledgement without exposing an unverified artifact.
- Plugin modules never execute during SSR.
- Manifest metadata reserves stable SSR slot layout before client components
  load.
- One plugin component failure cannot remove the core page or other plugin
  contributions.
- Revoked plugin code cannot be restored through release rollback.
- One action restores the recommended built-in-only admin frontend while
  preserving plugin settings, credentials, packages, and backend state.
- Existing plugins without admin frontend code and existing theme workflows
  remain functional.
- The follow-up jobs project can register typed production slots without
  changing the loader, trust, build, or release architecture.

## Follow-Up Project

After this runtime is implemented and verified, a separate design and plan will
add the River-backed admin job monitoring module. That project owns job DTOs,
`jobs.view` and `jobs.manage`, worker heartbeat data, job and queue operations,
and the first production component slots:

- `admin.jobs.table.columns`
- `admin.jobs.row.actions`
- `admin.jobs.detail.sections`
