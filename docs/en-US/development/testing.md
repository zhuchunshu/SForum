# Testing & quality gates

[← Development](./README.md)

## Common commands

```sh
cd apps/api && go test ./...
cd apps/web && bun run typecheck
node tests/validate-architecture-boundaries.mjs
ruby scripts/validate-openapi-refs.rb
./scripts/test.sh
```

`./scripts/test.sh` is the full gate (Go tests, OpenAPI refs, Nuxt typecheck, `tests/validate-*`). Use before large merges.

## File-size and architecture preflight

- Before changing a handwritten production file above 500 lines, check its
  current size and decide whether the new responsibility should be extracted.
- An unbaselined file must not exceed 1000 lines, and a legacy file must not
  exceed its recorded cap. Do not raise a baseline only to make CI pass; use
  the decision-record process in `AGENTS.md` for a genuinely unavoidable
  exception.
- After adding, moving, or materially growing production files, run
  `node tests/validate-architecture-boundaries.mjs` before committing or
  pushing, then run the slower broad gate.
- Focused tests do not replace the architecture preflight or the full
  `cd apps/web && bun test` required for shared frontend contracts.

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
