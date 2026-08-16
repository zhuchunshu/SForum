# Development

[← English docs home](../README.md)

For contributors and integrators.

## Chapters

| Doc | Topic |
| --- | --- |
| [Environment setup](./setup.md) | Toolchain, deps, first run |
| [Daily workflow](./workflow.md) | Hot reload, scripts, OpenAPI, extensions |
| [Developer CLI](./cli.md) | `sforum`: scaffold, validate, digest, package, seed, recovery |
| [API usage](./api.md) | Authentication, CSRF, PATs, response envelope |
| [Testing & gates](./testing.md) | Unit tests, repo gate, contracts |
| [Repository map](./repository.md) | Directory roles |

## Stack snapshot

| Layer | Tech |
| --- | --- |
| Web | Nuxt 4 · Vue 3 · Nuxt UI 4 · Bun · Tailwind · i18n (`zh-CN` default) |
| API | Go Fiber v3 · PostgreSQL · Redis · River · Goose · sqlc |
| Contracts | Modular OpenAPI · Protobuf Host API v2 |
| Extensions | Manifest V3 · exact-artifact trust · Page Registry themes |

## Extension authoring entries

- [Plugin routes (declared HTTP routes)](../../extensions/routes.md) · [Build, digest, and load](../../extensions/build-and-load.md) · [Plugin authoring guide](../../extensions/authoring-guide.md)

## Collaboration (summary)

Full rules: root `AGENTS.md`.

- Plugin-first core  
- Model permissions early  
- Keep OpenAPI in sync  
- No emoji as UI icons; use Tabler / Nuxt Icon  
- Read `knowledge/index.md` before sensitive work  

## Quick commands

```sh
./scripts/dev.sh
./scripts/api-dev.sh
cd apps/web && bun run dev
./scripts/test.sh
```
