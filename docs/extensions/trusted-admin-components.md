# Trusted Admin Components

SForum has two full-trust admin component paths. Both run on the admin origin
with the current administrator's browser authority; neither is a sandbox.
Backend permissions, extension route namespaces, settings allowlists, and
secret handling remain authoritative.

| Path | New packages | Operator build | Runtime contract |
| --- | --- | --- | --- |
| Admin Micro-frontend API v1 | Recommended for complex settings | None | Author-prebuilt `.mjs`, digest URL, `mount(target, bridge)` |
| Legacy trusted Vue Web Release | Compatibility only | Host Nuxt/Nitro Web Release | `frontend.admin` SFC registry + trusted contribution |

Use Schema UI or Schema + Settings Actions whenever possible. They need no
author JavaScript and are the mandatory fallback for a prebuilt settings
component.

## Admin Micro-frontend API v1 (recommended)

Declare a versioned settings document with `ui.mode=component`, a package-local
entry under `frontend/admin/dist/`, optional CSS, and ordinary Schema fields:

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
    "fields": [
      { "key": "message", "label": "Message", "type": "text", "default": "Hello" }
    ]
  }
}
```

The entry is framework-neutral:

```js
export const apiVersion = 1

export function mount(target, bridge) {
  const button = document.createElement('button')
  button.textContent = 'Save'
  const onClick = () => bridge.settings.save()
  button.addEventListener('click', onClick)
  target.append(button)
  return () => {
    button.removeEventListener('click', onClick)
    button.remove()
  }
}
```

`AdminMicroFrontendBridgeV1` from `@sforum/admin-sdk` provides:

- extension id/version, locale, appearance tokens;
- declared setting items, current draft reads, draft updates, save, and reset;
- namespaced extension API requests (backend authorization still applies);
- theme-aware Toasts, host translation, and admin navigation;
- cleanup through the function returned by `mount`.

The host never passes Cookie or Authorization values as bridge data. Stored
secrets remain masked; only a secret draft actively typed by the administrator
can be present in settings values.

### Trust and asset rules

- Assets must come from the installed extension package. Remote URLs and
  runtime Vue SFC compilation are rejected.
- Entry must be `.mjs`; optional style must be `.css`; both stay under
  `frontend/admin/dist/`, are regular files, size-bounded, and path-contained.
- The authenticated asset endpoint accepts only `entry` or `style`, uses the
  immutable `adminFrontendDigest` URL, `nosniff`, private immutable caching,
  ETag, and `Cross-Origin-Resource-Policy: same-origin`.
- Uploaded components require a one-use, five-minute confirmation bound to
  actor, extension id, version, API version, component id, and digest. The
  durable technical boundary is the stored digest grant.
- Protected builtin components use source trust but the same API, loader,
  cleanup, failure boundary, and Schema fallback.
- Changed bytes or version invalidate the usable grant. Missing trust, failed
  import, API mismatch, mount/cleanup failure, or three failures in one browser
  session falls back to Schema UI.
- “Restore declarative Schema UI” never removes settings, secrets, the package,
  backend state, or public theme activation. Uploaded component trust can be
  revoked without creating a Web Release.

Reference fixture:
`extensions/fixtures/plugins/sforum-prebuilt-settings/`.

## Legacy trusted Vue Web Release (deprecated for new settings pages)

Legacy slots such as Jobs components still use `frontend.admin` and a static
registry. Declare API version `1`, a safe root, exact component/locale maps,
`package.json`, and a frozen `bun.lock`. Each component is referenced by one
trusted contribution.

```json
{
  "frontend": {
    "admin": {
      "root": "frontend/admin",
      "apiVersion": 1,
      "components": { "latency": "components/LatencyCell.vue" },
      "locales": { "zh-CN": "locales/zh-CN.json", "en-US": "locales/en-US.json" }
    }
  },
  "contributions": [{
    "point": "admin.jobs.table.columns",
    "id": "latency",
    "label": { "zh-CN": "延迟", "en-US": "Latency" },
    "payload": { "component": "latency", "width": 120 }
  }]
}
```

Trust and Web Release composition use the dedicated `adminFrontendDigest`, so
backend/settings/public-theme changes do not rebuild the host. An unchanged
active/ready composition is reused. The release typecheck policy is explicitly
`off`, `report` (recommended), or `block`; CI and `./scripts/test.sh` always
retain mandatory typecheck.

Author packages must not contain `node_modules` under `frontend/admin`.
Compatible Vue/Nuxt/Nuxt UI/Vue Router/admin-sdk peers are supplied by the host
in dev and linked only inside the isolated Web Release workspace. Lifecycle
scripts and imports escaping the admin root are rejected.

Legacy build failures keep the active release. Inspect Admin → Extensions →
Web releases. The old `admin.extension.settings.page` path remains compatible,
but new settings pages should use Schema, Actions, or Micro-frontend API v1.
