# Trusted Admin Components

SForum plugins may render arbitrary client-side Vue components only inside
core-owned trusted admin slots. This is a full-trust capability: code runs on
the admin origin with the current administrator's browser authority. Backend
permissions remain authoritative, but they do not sandbox frontend code.

## Jobs Slots

The Jobs monitor owns the first production slots:

- `admin.jobs.table.columns` renders one cell for each job row. Its localized
  contribution label becomes the column heading; `payload.width` is optional.
- `admin.jobs.row.actions` renders beside the core view/retry/cancel controls.
- `admin.jobs.detail.sections` renders after the core job arguments and status
  inside the detail modal.

All three receive `AdminSlotProps<Point>` from `@sforum/admin-sdk`. The context
contains a read-only `job`; options come from the manifest payload after the
host removes `component`. Use `useSForumAdminHost()` for localized messages,
theme-aware Toasts, admin navigation, and namespaced extension API requests.

## Manifest

Declare `frontend.admin` with API version `1`, a safe root, an exact component
map, and both `zh-CN` and `en-US` locale JSON files. Every component must be
referenced by exactly one trusted contribution. The admin frontend root must
contain `package.json` and a frozen `bun.lock`.

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
    "order": 100,
    "label": { "zh-CN": "延迟", "en-US": "Latency" },
    "payload": { "component": "latency", "width": 120 }
  }]
}
```

Uploaded packages are untrusted until an active `super_admin` grants trust for
the exact extension ID, version, package digest, API version, points, and
component IDs. Any package change invalidates the grant. Revocation builds and
activates a safe release before finalizing removal.

Dependencies must be pinned by Bun. Lifecycle scripts are disabled. Private
copies of Vue, Nuxt, Nuxt UI, Vue Router, and `@sforum/admin-sdk` are forbidden;
declare compatible host peer ranges instead.

**Author packages must not contain `node_modules` under `frontend/admin`.**
Host peers are injected only by the host:

- **Dev / Nuxt**: absolute Vite/Nuxt aliases (`apps/web/build/admin-host-peers.mjs`)
  so SFCs under `extensions/**` resolve bare imports without a local install.
- **Production Web Release**: the builder copies the admin frontend into an
  isolated workspace, then links host peers there after package inspection.
  Source trees and uploaded ZIPs stay free of host `node_modules`.

Optional local `bun run dev:compose` may prune leftover peer-only
`node_modules` under builtin admin roots; it must never write peers back into
extension source. Imports may not escape the admin frontend root or its
workspace-isolated dependencies.

Build failures keep the currently active site. Inspect diagnostics under Admin
> Extensions > Web releases. A component failure is isolated by the host; the
third failure quarantines that contribution for the current browser session.
Use Admin > Extensions > Restore recommended defaults to rebuild a built-in-only
admin frontend; plugin packages, settings, credentials, and backend state are
preserved.
