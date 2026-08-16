# Daily workflow

[← Development](./README.md)

## Loop

1. Keep `./scripts/dev.sh` dependencies up  
2. `./scripts/api-dev.sh` for Go hot reload  
3. `cd apps/web && bun run dev` for frontend HMR  
4. Scoped tests: `cd apps/api && go test ./path/...`  
5. Contract edits → `ruby scripts/validate-openapi-refs.rb`  
6. Finish: update `knowledge/modules/<area>.md` and a hot handoff if needed  

## Scripts

| Script | Purpose |
| --- | --- |
| `./scripts/dev.sh` | Deps + migrate |
| `./scripts/dev-down.sh` | Stop deps |
| `./scripts/api-dev.sh` | API Air + builtin staging |
| `./scripts/worker-dev.sh` | Standalone worker |
| `./scripts/test.sh` | Full repo gate |
| `./scripts/build-builtin-plugins.sh` | Build built-in plugins |
| `ruby scripts/validate-openapi-refs.rb` | OpenAPI `$ref`s |
| `cd apps/api && go run ./cmd/sforum` | Developer CLI (see [Developer CLI](./cli.md)) |

## API changes

- HTTP: `apps/api/app/Http/`  
- Domain: `apps/api/app/Models/`  
- Migrations: Goose under `database/migrations/`  
- Queries: sqlc  
- OpenAPI: modular files under `contracts/openapi/`—not a giant root file  

## Frontend changes

- Pages under `apps/web/app/pages` (admin under `pages/admin`)  
- `SF*` components, Tabler/Lucide icons only  
- Keep `zh-CN` and `en-US` strings in sync  
- Themes activate via Page Registry without assuming a Nuxt rebuild  

## Extension changes

1. Read [authoring guide](../../extensions/authoring-guide.md)  
2. Scaffold, digest, package: [Developer CLI](./cli.md) (`make:plugin`, `digest`, `package --exclude-source`, …)  
3. Prefer `extensions/dev/` or `go run ./cmd/sforum make:plugin …`  
4. Manifest V3 + trust for executable enable  
5. Regenerate catalogs after host surface changes  
6. Do not import host business packages from third-party plugins  

## Seed data

```sh
cd apps/api && go run ./cmd/sforum seed:forum
```

Append-only; no domain events; needs `DATABASE_URL`. Full flags: [Developer CLI · seed:forum](./cli.md#seed-data-seedforum-and-seedperf).

## Next

- [Developer CLI](./cli.md)  
- [Testing](./testing.md)  
- [Repository map](./repository.md)  
