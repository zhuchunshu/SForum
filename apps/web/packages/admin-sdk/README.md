# @sforum/admin-sdk

Typed bridge contracts for trusted SForum admin plugin page bodies.

```ts
import type { AdminPageBridgeV1 } from '@sforum/admin-sdk'
import { ADMIN_MICRO_FRONTEND_API_VERSION } from '@sforum/admin-sdk'
```

Bundle this package into the plugin's final ESM output. The Host supplies the
bridge at mount time and continues to own routing, permissions, navigation,
appearance, and failure isolation.

See `docs/extensions/trusted-admin-components.md` in the SForum repository.
