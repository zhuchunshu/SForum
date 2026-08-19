# Trusted Admin Components

SForum settings have three presentation levels:

| Need | Contract | Author JavaScript | Operator build |
| --- | --- | --- | --- |
| Ordinary fields and layout | Schema UI | No | No |
| Provider probes and host operations | Schema + Settings Actions | No UI code | No |
| Complex interactive workflow | Prebuilt Admin Micro-frontend API v1 | Fully trusted prebuilt module | No |

Use Schema UI unless a workflow genuinely needs custom client behavior. There
is no runtime Vue SFC compilation, static admin registry, remote script URL, or
extension-triggered Nuxt build.

Ordinary plugin-owned admin pages use the same prebuilt artifact boundary. The
Host owns the admin layout, sidebar, topbar, tabs, route middleware, and page
heading; the component mounts only inside the page body.

## Package contract

Declare component mode in the versioned Settings Document and keep `fields` as
the required Schema fallback:

```json
{
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
  "fields": [
    { "key": "message", "label": "Message", "type": "text", "default": "Hello" }
  ]
}
```

The files must already be built when the package is created. Bundle framework
dependencies into the module; operators never install extension frontend
dependencies. Entry/CSS paths are package-relative, path-contained, size
bounded, and restricted to `frontend/admin/dist/*.mjs|*.css`.

### Plugin-owned admin page

Declare `view: component` on an `admin.pages[]` item. A plugin may declare
multiple pages, but every component id must be unique inside the package:

```json
{
  "admin": {
    "entry": "/dashboard",
    "pages": [
      {
        "path": "/dashboard",
        "label": "Plugin dashboard",
        "icon": "i-lucide-layout-dashboard",
        "view": "component",
        "menu": true,
        "permission": "acme.dashboard.view",
        "component": {
          "id": "dashboard",
          "apiVersion": 1,
          "entry": "frontend/admin/dist/dashboard.mjs",
          "css": "frontend/admin/dist/dashboard.css"
        }
      }
    ]
  }
}
```

The entry exports the same `apiVersion` and `mount(target, bridge)` shape. Its
page bridge supplies `page`, namespaced `request`, `toast`, `t`, `navigate`,
locale, and appearance. It deliberately does not expose settings draft methods.
Plugin authors may compile Vue SFC source to this module; production SForum
loads only the prebuilt output.

## Vue authoring with Plugin UI SDK v1

For a first-class page, start with the supported scaffold instead of writing
the mount adapter or CSS by hand:

```bash
go run ./cmd/sforum make:plugin ... --vue-admin-page
sforum extension build --allow-scaffold <package-root>
```

The scaffold consumes the public `@sforum/admin-sdk@1` and
`@sforum/plugin-ui@1` npm packages. Existing Vue workspaces can install the
same contracts with `bun add @sforum/admin-sdk@^1 @sforum/plugin-ui@^1`.

### Registry release ownership (maintainers)

The tag-driven Release workflow publishes both packages through npm Trusted
Publishing with provenance. It is idempotent only when an existing version has
the exact same tarball integrity; changed content under an existing version
fails the release before image promotion.

An npm package must exist before its Trusted Publisher can be configured. The
one-time namespace bootstrap was completed on 2026-08-20: the `sforum` owner
used interactive 2FA to publish the reviewed tarballs:

```bash
node scripts/ci/pack-web-sdks.mjs /tmp/sforum-web-sdks
npm publish /tmp/sforum-web-sdks/sforum-admin-sdk-1.0.0.tgz --access public
npm publish /tmp/sforum-web-sdks/sforum-plugin-ui-1.0.0.tgz --access public
```

Both package settings are now bound to the GitHub Actions Trusted Publisher
organization `zhuchunshu`, repository `SForum`, workflow filename
`release.yml`, no environment, allowed action `npm publish`. Do not repeat the
bootstrap or add a long-lived npm token to the repository. Later releases use
`.github/workflows/release.yml` exclusively. The interactive `1.0.0` bootstrap
versions have no provenance; verify provenance on the next SDK version that the
OIDC Release job actually publishes.

`@sforum/plugin-ui@1` currently provides:

- layout: `SPluginPage`, `SPluginSection`;
- forms: `SPluginButton`, `SPluginField`, `SPluginInput`, `SPluginSelect`;
- feedback: `SPluginAlert`, `SPluginEmptyState`;
- data: `SPluginTable`.

The generated Vite build bundles Vue and these components into the plugin's
own ESM/CSS. The SDK reads stable SForum appearance variables with standalone
fallbacks, so authors normally write no CSS. It does not import Nuxt UI, Nuxt
composables, Host route modules, or private `SF*` components. Those remain
implementation details rather than a plugin ABI.

The generated `admin.ts` exports the required API version and mount/cleanup
adapter. `AdminDashboard.vue` receives the typed `AdminPageBridgeV1`, uses SDK
components like ordinary Vue components, and can call namespaced APIs, Host
Toasts, translation, and navigation through that bridge.

## Module API

```js
export const apiVersion = 1

export async function mount(target, bridge) {
  const button = document.createElement('button')
  button.textContent = bridge.t('save')
  button.onclick = () => bridge.settings.save()
  target.append(button)

  return () => button.remove()
}
```

The v1 bridge supplies:

- extension id/version, locale, and appearance tokens;
- declared settings metadata, current draft values, update/save/reset;
- namespaced extension API requests;
- host Toasts, translation, and admin navigation.

It does not expose Cookie or Authorization values. The extension API remains
permission checked by the backend; bridge access is not an authorization grant.

## Trust and loading

An uploaded component is not loaded until an active `super_admin` completes the
one-use confirmation. The challenge is bound to actor, extension id, version,
API version, component id, and digest and expires after five minutes.

The durable technical boundary is the stored digest grant, not the confirmation
code. It binds extension id, version, API version, component id,
`adminFrontendDigest`, and package identity. Changed entry/CSS bytes invalidate
old trust. Assets are served only from the installed package through:

```text
GET /_sforum/private-assets/extensions/{id}/{digest}/entry
GET /_sforum/private-assets/extensions/{id}/{digest}/style
GET /_sforum/private-assets/extensions/{id}/{digest}/{componentId}/entry
GET /_sforum/private-assets/extensions/{id}/{digest}/{componentId}/style
```

The Host resource namespace streams to the authenticated Go asset handler; the
legacy `/api/v1/admin/extensions/{id}/frontend/assets/{digest}/{asset}` path is
an internal/compatibility upstream, not the browser-facing URL. Responses use
immutable private caching, exact
digest verification, MIME checks, `nosniff`, and
`Cross-Origin-Resource-Policy: same-origin`.

## Failure behavior

Missing, revoked, or invalidated trust never blocks settings. Import failure,
API mismatch, CSS failure, invalid cleanup, mount failure, or repeated
browser-session failures disposes the component and renders Schema UI. The
administrator may explicitly return to Schema UI; doing so does not delete
settings, secrets, backend state, or the package.

Component code is fully trusted after approval and is not sandboxed. Package
provenance, explicit approval, immutable digest identity, backend permissions,
namespaced APIs, error isolation, quarantine, and Schema fallback are the
controls.

Source convenience does not change the production trust boundary: uploaded
`.vue` files are never compiled, and package-local Vite/Nuxt configuration is
never executed by SForum. Release zips should be made with `--exclude-source`.

For ordinary admin pages, the same failures preserve the Host admin shell and
show a retryable page-local error. A page component is not served while its
extension is disabled or when the actor lacks its declared permission.
