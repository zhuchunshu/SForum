# Trusted Admin Settings Components

SForum settings have three presentation levels:

| Need | Contract | Author JavaScript | Operator build |
| --- | --- | --- | --- |
| Ordinary fields and layout | Schema UI | No | No |
| Provider probes and host operations | Schema + Settings Actions | No UI code | No |
| Complex interactive workflow | Prebuilt Admin Micro-frontend API v1 | Fully trusted prebuilt module | No |

Use Schema UI unless a workflow genuinely needs custom client behavior. There
is no runtime Vue SFC compilation, static admin registry, remote script URL, or
extension-triggered Nuxt build.

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
GET /api/v1/admin/extensions/{id}/frontend/assets/{digest}/entry
GET /api/v1/admin/extensions/{id}/frontend/assets/{digest}/style
```

The authenticated same-origin responses use immutable private caching, exact
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
