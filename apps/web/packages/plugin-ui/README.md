# @sforum/plugin-ui

Versioned Vue components for trusted SForum plugin page bodies.

```ts
import {
  SPluginButton,
  SPluginIcon,
  SPluginPage,
  SPluginSection
} from '@sforum/plugin-ui'
import '@sforum/plugin-ui/style.css'
```

Mount a page through the SDK adapter so the Host icon resolver is injected for
you:

```ts
import { ADMIN_MICRO_FRONTEND_API_VERSION } from '@sforum/admin-sdk'
import { definePluginPage } from '@sforum/plugin-ui'
import '@sforum/plugin-ui/style.css'
import AdminDashboard from './AdminDashboard.vue'

export const { apiVersion, mount } = definePluginPage(
  AdminDashboard,
  ADMIN_MICRO_FRONTEND_API_VERSION
)
```

## Host icons

`splugin-button`, `SPluginSection`, `SPluginAlert`, `SPluginEmptyState`, and
`SPluginTable` columns accept an `icon` prop using the Host icon name grammar
(`i-lucide-*` / `i-tabler-*`); `SPluginIcon` is the standalone primitive. The
SVG markup always comes from the Host, so plugins never bundle an icon set and
never inline their own SVG.

Icons are decorative by default. Pass `label` to `SPluginIcon` when the icon
carries meaning. An unknown name, a Host without an icon resolver, or a failed
request all degrade to an empty placeholder plus a single console warning; they
never throw and never block the page from mounting.

The package deliberately has no Nuxt, Nuxt UI, router, or private SForum
component dependency. Bundle it into the plugin's final ESM/CSS output before
installation; SForum does not compile package source at runtime.

See `docs/extensions/trusted-admin-components.md` and scaffold a working page
with `sforum make:plugin --vue-admin-page`.
