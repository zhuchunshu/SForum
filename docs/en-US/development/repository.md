# Repository map

[← Development](./README.md)

```text
SForum/
├── AGENTS.md
├── apps/api/                 # Go API + worker + CLI
├── apps/web/                 # Nuxt
├── contracts/                # OpenAPI + Protobuf
├── extensions/               # builtin / optional / dev / fixtures
├── docs/                     # handbooks + extension reference
├── knowledge/                # decisions, modules, handoffs
├── scripts/                  # dev / test helpers
├── tests/                    # repo validation scripts
├── deploy/                   # Caddy example, backup helpers
├── compose.yaml
├── compose.dev.yaml
├── compose.prod.yaml
└── deploy.sh
```

## `apps/api` (Laravel-style layout)

| Path | Role |
| --- | --- |
| `cmd/api` · `cmd/worker` · `cmd/migrate` · `cmd/sforum` | Binaries (`sforum` CLI: [docs](./cli.md)) |
| `bootstrap/` | Runtime assembly |
| `app/Http/` | Controllers & routes |
| `app/Models/` | Domain services |
| `app/Providers/` | Provider wiring |
| `app/Support/` | Jobs, Search, Cache, Extensions… |
| `database/` | Goose + sqlc |
| `sdk/plugin` | Public plugin SDK |

## `apps/web`

| Path | Role |
| --- | --- |
| `app/pages` | Routes (incl. admin) |
| `app/components` | SF library |
| `app/composables` · `layouts` · `middleware` · `plugins` | App shell |

## Related

- Extension tech: `docs/extensions/`  
- Memory: `knowledge/index.md`  
- Product/architecture: sibling docs in this locale  
