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
| `cd apps/api && go run ./cmd/sforum` | Developer CLI |

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
2. Scaffold with `go run ./cmd/sforum make:plugin …` or use `extensions/dev/`  
3. Manifest V3 + trust for executable enable  
4. Regenerate catalogs after host surface changes  
5. Do not import host business packages from third-party plugins  

## Seed data

```sh
cd apps/api && go run ./cmd/sforum seed:forum
```

Append-only; no domain events; needs `DATABASE_URL`.

## Next

- [Testing](./testing.md)  
- [Repository map](./repository.md)  
