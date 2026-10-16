# @sforum/admin-sdk

Typed bridge contracts for trusted SForum admin plugin page bodies.

```ts
import type { AdminPageBridgeV1 } from '@sforum/admin-sdk'
import { ADMIN_MICRO_FRONTEND_API_VERSION } from '@sforum/admin-sdk'
```

Bundle this package into the plugin's final ESM output. The Host supplies the
bridge at mount time and continues to own routing, permissions, navigation,
appearance, and failure isolation.

## Host icons

`AdminPageBridgeV1.icons` resolves Host icon names to sanitized inline SVG:

```ts
const icons = await bridge.icons?.resolve(['i-lucide-inbox'])
element.innerHTML = icons?.['i-lucide-inbox'] ?? ''
```

The member is optional: a Host that predates `1.1.0` omits it, and a plugin must
degrade instead of failing. Names must match
`i-(lucide|tabler)-<kebab-name>`; the Host never returns an icon for a name it
does not know. `@sforum/plugin-ui` wraps all of this in `SPluginIcon`, so most
plugins never call the resolver directly.

See `docs/extensions/trusted-admin-components.md` in the SForum repository.
