# Buildless Extension Settings UI

Date: 2026-07-13
Status: Implemented (P0–P6)

V3 preservation note: the active
`2026-07-13-trusted-plugin-theme-platform-v3.md` keeps Settings Document Schema/
Actions, prebuilt admin micro-frontends, exact-digest trust, and mandatory
Schema fallback. V3 adds separately trusted public L2 and Component/Admin
Surface registries; it does not restore operator builds or runtime SFC compile.

## Context

SForum already stores plugin and theme settings in `extension_settings` and
can render a generic host-owned form from manifest setting definitions. It also
supports trusted Vue admin components through `frontend.admin`, but those
components currently enter a static registry and require a full Web Release:
typecheck the host, build Nuxt/Nitro, verify the artifact, and switch the web
runtime.

That pipeline is appropriate for a host release, but it is too expensive for
ordinary extension settings UI. Most theme and plugin settings only need
fields, tabs, groups, help text, recommended values, secret preservation, and
host-mediated actions such as “test connection”. Rebuilding the whole site for
those surfaces couples extension configuration to the host deployment artifact.

The product direction is therefore:

> Build only when authoring code requires it. Installing, enabling,
> configuring, or switching an extension must not rebuild SForum by default.

## Decision

### 1. One settings system, three rendering modes

Plugins and themes share one settings document and one host page:

1. **Schema UI** — host-rendered fields, tabs, groups, columns, callouts,
   recommended defaults, and secret-preservation guidance. No author JS and no
   web build.
2. **Schema + Actions** — Schema UI plus host-rendered buttons that invoke
   declared, permission-checked extension actions. No author UI JS and no web
   build.
3. **Component UI** — an optional author-prebuilt admin micro-frontend for
   complex workflows. The operator does not build SForum; the host dynamically
   loads an immutable package asset after explicit administrator trust.

Schema UI is the default for themes and ordinary plugins. Component UI is a
normal advanced capability, not the default representation of settings.

### 2. Backward-compatible settings document

Existing array-shaped `settings.json` files remain valid and normalize to a
versioned document with default Schema UI:

```json
{
  "schemaVersion": 1,
  "ui": {
    "mode": "schema",
    "layout": "form"
  },
  "fields": []
}
```

The first UI contract supports:

- `layout: form | tabs`
- ordered tabs referencing field groups
- groups and optional columns
- localized callouts/help text
- existing field types (`text`, `string`, `number`, `boolean`, `select`,
  `secret`, `textarea`), defaults, recommended values, options, placeholders,
  localization, secret semantics, and optional control `width`
  (`default` | `full`)
- safe fallback to the ordinary linear form when optional UI presentation is
  absent or cannot be rendered

Presentation does not change storage semantics. Field keys remain the
authority for `extension_settings`.

### 3. Settings actions are declared, not arbitrary URLs

Schema pages may declare operations such as probe, test mail, refresh status,
or regenerate metadata. The browser calls a host-owned endpoint. The host
checks the actor, action declaration, lifecycle state, input field allowlist,
timeout, and audit policy before invoking an existing provider probe, plugin
RPC method, or plugin-owned route adapter.

Manifests cannot attach an arbitrary remote URL or arbitrary browser script to
an action. API policy checks remain authoritative.

### 4. Configuration is separate from runtime enablement

Installed or disabled extensions may read and save their declared settings
without starting extension code. This enables the beginner-friendly sequence:

```text
install → configure → optional safe probe → review capabilities → enable
```

Actions that require extension code are unavailable while disabled unless the
specific executor supports a restricted, short-lived probe runtime. A disabled
extension never registers normal routes, events, jobs, providers, or schedules
merely because its settings page is open.

### 5. Component UI is author-prebuilt and operator-buildless

The long-term Component UI contract is a self-contained admin micro-frontend
artifact shipped inside the extension package. It uses a versioned mount API
rather than raw Vue SFC runtime compilation or host dependency resolution.

Conceptual package shape:

```text
frontend/admin/dist/
├── settings.mjs
└── settings.css
```

The v1 micro-frontend module exports a small contract such as `apiVersion`,
`mount(target, bridge)`, and optional `unmount()`. Authors may use Vue or
another frontend library, but must bundle their own implementation dependencies
for this artifact. The host supplies a narrow bridge for settings values,
save/reset, namespaced API calls, navigation, localized messages, appearance
tokens, and Toasts.

This avoids making runtime loading depend on Nuxt/Vue peer resolution and
keeps the artifact author-build-once/operator-build-never.

### 6. Administrator confirmation is simple, while trust remains digest-bound

An uploaded extension that contains Component UI may be installed and enabled
without trusting that UI; the host falls back to Schema UI. To load Component
UI, an authorized administrator explicitly confirms the risk. The UI may use a
verification code, re-authentication, or an equivalent intentional-action
confirmation.

The confirmation is usability friction, not the technical trust boundary.
Trust is stored against at least:

- extension id
- version
- admin frontend API version
- admin frontend content digest

Changing the component artifact invalidates the old grant and returns the
settings page to Schema UI until reconfirmed. Component assets must come from
the installed package; arbitrary remote component URLs are not supported.

Component UI runs with the administrator browser’s authority. SForum does not
claim that a CAPTCHA or confirmation dialog sandboxes malicious code.

### 7. Runtime frontend releases are removed

SForum has not shipped, so there is no installed-version compatibility burden.
The trusted Vue `frontend.admin`, static registry, Web Release, runtime Nuxt
Layer, release supervisor, and extension frontend build paths are removed
rather than retained as a migration track.

- Schema and Settings Actions are always host-rendered.
- Component UI is always author-prebuilt and loaded by immutable digest.
- Plugin enable/disable and theme activation complete synchronously.
- Public theme activation remains independent Page Registry + L0/L1.
- The API/worker images contain no Bun, Web source tree, or frontend dependencies.
- Host typecheck/build happens only when building SForum itself.

## Security boundaries

- Schema UI executes no extension-authored JavaScript.
- Settings and actions accept only manifest-declared fields and operations.
- Secrets remain encrypted, masked, and preserve-on-empty under the API.
- Component assets are immutable, same-package, digest-addressed resources.
- Component trust is explicit and invalidated by artifact changes.
- API permissions remain authoritative regardless of frontend mode.
- Component load/render failure falls back to Schema UI and must not break the
  rest of admin.
- Existing error isolation/quarantine behavior should be reused.

## Consequences

### Positive

- Most plugin and theme settings become install-and-use with zero operator
  build.
- Theme settings can be rich without allowing uploaded theme JavaScript by
  default.
- Provider plugins can offer probes and operational controls without custom
  frontend code.
- Complex extensions remain highly extensible through trusted prebuilt UI.
- Host upgrades and extension settings changes become separate deployment
  concerns.

### Costs

- A versioned settings presentation contract and renderer must be maintained.
- Settings action execution needs a catalog, lifecycle rules, and tests.
- The micro-frontend bridge becomes a public SDK contract.
- Authors of complex Component UI must produce portable browser bundles before
  packaging the extension.

## Implemented contract notes

- Settings Document schema version is `1`. Legacy arrays preserve array-shaped
  canonical JSON; explicit documents preserve object-shaped canonical JSON.
- The first Settings Action catalog kind is `provider_probe`. Disabled-plugin
  probes use a short-lived protocol starter without Host API token or normal
  runtime registration. SMTP performs connect/TLS/auth only and sends no mail;
  storage reuses `StorageProbe`.
- `adminFrontendDigest` fingerprints only the prebuilt component contract and
  entry/CSS bytes. Package identity remains stored and verified separately.
- CI/full test typecheck remains mandatory for the SForum host.
- Admin Micro-frontend API v1 requires package-local
  `frontend/admin/dist/*.mjs` and optional `.css`. Assets are regular,
  path-contained, size-bounded, authenticated, same-origin, immutable, and
  available only through `entry`/`style` aliases.
- Uploaded trust confirmation is one-use and five minutes, bound to actor,
  extension id, version, API version, component id, and digest. The durable
  grant binds the same extension identity and digest; confirmation remains
  intentional-action friction, not sandboxing.
- The bridge exposes settings draft/save/reset, namespaced API, Toast, locale/
  translation, navigation, and appearance tokens. It exposes no Cookie or
  Authorization value. Mount/CSS/import/API mismatch/cleanup/quarantine failure
  falls back to Schema UI.

## Explicit non-goals

- Runtime compilation of uploaded Vue SFC files
- Arbitrary remote admin scripts or manifest URLs
- Treating a verification code as code sandboxing
- Rebinding public theme activation to a Nuxt build
- Removing the generic form fallback
- Replacing backend permission checks with frontend trust
