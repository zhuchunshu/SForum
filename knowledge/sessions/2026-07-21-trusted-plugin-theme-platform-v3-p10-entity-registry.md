# 2026-07-21 Session Handoff — P10 Entity/Taxonomy/Field Registry (10/15)

**Branch:** `main` only.
**Overall:** display **80.0%**. **P10:** **10/15**.

## Commits

- `96384e489` feat(entity): add Entity/Taxonomy/Field Schema Registry
- `3203c6620` feat(manifest): declare Entity/Taxonomy/Field Schema surfaces
- `e6e0c1170` feat(entity): wire Entity Registry into lifecycle plan @11

## Closed

- Immutable `Support/EntityRegistry` kinds: `entity` / `taxonomy` / `field`
- Package-local cross-refs, storage-key prefix, Safe Mode, CAS replace
- Host-derived permission allow/deny, index plans, import/export plans,
  deletion policies (soft/hard/retain)
- Manifest.entities + JSON Schema + includes sharding
- Lifecycle plan `sforum.lifecycle.registry-plan@11` / family `entity.v1`
- Production bootstrap process-local Entity Registry

## Verification

```bash
cd apps/api && go test ./app/Support/EntityRegistry/ -count=1
cd apps/api && go test ./app/Support/ExtensionManifest/ -count=1
cd apps/api && go test ./app/Support/Extensions/ -count=1 -run 'LifecycleEntity'
cd apps/api && go test ./bootstrap/ -count=1 -run 'ProductionLifecycle|Lifecycle'
cd apps/api && go build ./...
```

## Next

1. Plugin-extend-plugin content/entity types via versioned dependencies + hooks
2. Attachment read/write policy + rich-content XSS product binding
3. Reference blocks (vote/product-card/embed/workflow form)
4. Reference media plugin + attack surface matrix

## Dirty WIP — DO NOT STAGE

route-inspector web, content-policy, PageViewModels, go.mod, host-api-v2,
websocket revoke test, ADR noise.

## Rollback

Revert `e6e0c1170` → `3203c6620` → `96384e489`. Compatible digests max includes
@11; no migration; Safe Mode core-only unchanged for other registries.
