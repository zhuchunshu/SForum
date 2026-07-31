# SForum

Maintainable, plugin-first open-source forum framework.

Core is the host (identity, forum primitives, permissions, extension runtime, contracts). Deployment-specific behavior—mail transport, optional search engines, storage vendors, and similar—lives in extensions.

## Documentation

| Language | Start here |
| --- | --- |
| **简体中文** | [`docs/zh-CN/README.md`](./docs/zh-CN/README.md) |
| **English** | [`docs/en-US/README.md`](./docs/en-US/README.md) |
| Hub | [`docs/README.md`](./docs/README.md) |

Quick paths:

- [快速开始](./docs/zh-CN/getting-started.md) / [Getting started](./docs/en-US/getting-started.md)
- [使用说明](./docs/zh-CN/usage/README.md) / [Usage](./docs/en-US/usage/README.md)
- [开发指南](./docs/zh-CN/development/README.md) / [Development](./docs/en-US/development/README.md)
- [生产部署](./docs/zh-CN/deployment.md) / [Deployment](./docs/en-US/deployment.md)
- Extension technical reference: [`docs/extensions/`](./docs/extensions/)

## Repository map

| Path | Role |
| --- | --- |
| `apps/web` | Nuxt 4 frontend |
| `apps/api` | Go Fiber API, worker, CLI |
| `contracts/` | OpenAPI + Protobuf |
| `extensions/` | Built-in / optional / dev packages |
| `docs/` | Bilingual handbooks + extension reference |
| `knowledge/` | Decisions, module notes, session handoffs |
| `scripts/` | Dev and test helpers |
| `deploy.sh` + `compose*.yaml` | Production and dependency orchestration |

## Local development

```sh
./scripts/dev.sh                 # PostgreSQL, Redis, Mailpit + migrations
./scripts/api-dev.sh             # API (embeds worker in dev)
cd apps/web && bun run dev       # Nuxt on :3000
```

Background jobs: development defaults to `EMBED_WORKER_IN_API=true`, so the API process consumes the River queue. For **production split** process development (`EMBED_WORKER_IN_API=false`), run a standalone worker:

```sh
./scripts/worker-dev.sh          # optional; only when the API does not embed the worker
```

Useful URLs:

- Web: http://127.0.0.1:3000  
- API health: http://127.0.0.1:3000/api/v1/health  
- API ready: http://127.0.0.1:3000/api/v1/ready  
- Mailpit: http://127.0.0.1:18025  

Meilisearch is **optional** (`docker compose --profile search up -d meilisearch`). Default search is built-in site PostgreSQL FTS.

Full steps: [docs/zh-CN/getting-started.md](./docs/zh-CN/getting-started.md) or [docs/en-US/getting-started.md](./docs/en-US/getting-started.md).

## Production

```sh
mkdir -p sforum
curl -fsSL https://github.com/zhuchunshu/SForum/archive/refs/tags/v3.0.0.tar.gz \
  | tar -xz --strip-components=1 -C sforum
cd sforum
./deploy.sh                              # interactive install; Enter uses latest stable
./upgrade.sh                             # update; Enter selects the newest GitHub Release
./upgrade.sh --version v3.0.0            # update to a specific release
```

`upgrade.sh` accepts a positional version or `--version`; its default `latest`
includes prereleases. It resolves that choice to a concrete tag, prints the
current and target versions, and asks for confirmation (`--yes` skips prompts).
The first blue/green ingress conversion has a short maintenance window. Later
migration-free releases keep API/Web HTTP traffic available; WebSockets may
reconnect and Worker consumption pauses briefly without losing durable jobs.
Targets with pending SForum Core or River migrations must use `deploy.sh` and
its maintenance-window migration path.

Details: [docs/zh-CN/deployment.md](./docs/zh-CN/deployment.md) / [docs/en-US/deployment.md](./docs/en-US/deployment.md).

## Community

- Report vulnerabilities privately according to [`SECURITY.md`](./SECURITY.md).
- SForum is available under the [`MIT License`](./LICENSE).

## Contributing / agents

1. Read [`AGENTS.md`](./AGENTS.md)  
2. Read [`docs/`](./docs/README.md) for usage and development  
3. Read [`knowledge/index.md`](./knowledge/index.md) for current project memory  
4. Keep OpenAPI, tests, and knowledge notes updated with code changes  
