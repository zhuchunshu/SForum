# Testing & quality gates

[← Development](./README.md)

## Common commands

```sh
cd apps/api && go test ./...
cd apps/web && bun run typecheck
ruby scripts/validate-openapi-refs.rb
./scripts/test.sh
```

`./scripts/test.sh` is the full gate (Go tests, OpenAPI refs, Nuxt typecheck, `tests/validate-*`). Use before large merges.

## Contract / extension checks

| Area | Notes |
| --- | --- |
| OpenAPI | Modular paths/schemas; validate refs after edits |
| Host catalogs | `extension docs generate` vs `docs/extensions/catalogs` |
| V3 catalogs | Identity/matrix drift rejected in CI |
| Host API v2 docs | `tests/validate-host-api-v2-docs.mjs` locks docs to constants |

## Testing expectations

- Cover allow **and** deny paths for privileged routes  
- Extension trust / Safe Mode boundaries need regression  
- Do not land exploit payloads (see `AGENTS.md`)  
- Browser scripts under `tests/` may need running dev servers  

## Build / preview

```sh
cd apps/web && bun run build
cd apps/web && bun run preview
```

Preview is for local production checks—not the production process manager. See [Deployment](../deployment.md).
