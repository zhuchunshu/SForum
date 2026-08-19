# @sforum/plugin-ui

Versioned Vue components for trusted SForum plugin page bodies.

```ts
import {
  SPluginButton,
  SPluginPage,
  SPluginSection
} from '@sforum/plugin-ui'
import '@sforum/plugin-ui/style.css'
```

The package deliberately has no Nuxt, Nuxt UI, router, or private SForum
component dependency. Bundle it into the plugin's final ESM/CSS output before
installation; SForum does not compile package source at runtime.

See `docs/extensions/trusted-admin-components.md` and scaffold a working page
with `sforum make:plugin --vue-admin-page`.
