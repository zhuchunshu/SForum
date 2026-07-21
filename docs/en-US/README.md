# SForum Documentation (English)

[简体中文](../zh-CN/README.md) · [Docs hub](../README.md)

SForum is a maintainable, plugin-first open-source forum framework: core is the host and contracts; verticals and vendor behavior live in extensions.

## Read by role

| You are… | Start here |
| --- | --- |
| Running it for the first time | [Getting started](./getting-started.md) |
| Site operator | [Usage](./usage/README.md) |
| Contributor / integrator | [Development](./development/README.md) |
| Shipping to production | [Deployment](./deployment.md) |
| Plugin / theme author | [Extensions (ops)](./usage/extensions.md) → [Authoring guide](../extensions/authoring-guide.md) |
| Product / architecture | [Product](./product.md) · [Architecture](./architecture.md) · [Roadmap](./roadmap.md) |

## Contents

### Usage

- [Usage overview](./usage/README.md)
- [First registration & super admin](./usage/first-login.md)
- [Admin control panel](./usage/admin.md)
- [Forum day-to-day](./usage/forum.md)
- [Search](./usage/search.md)
- [Extensions & themes (operators)](./usage/extensions.md)

### Development

- [Development overview](./development/README.md)
- [Environment setup](./development/setup.md)
- [Daily workflow](./development/workflow.md)
- [Developer CLI](./development/cli.md)
- [Testing & gates](./development/testing.md)
- [Repository map](./development/repository.md)

### Other

- [Deployment](./deployment.md)
- [Architecture](./architecture.md)
- [Product](./product.md)
- [Roadmap](./roadmap.md)

## Technical reference (path-stable)

These paths are referenced by CI and generators—do not move casually:

- [Plugin authoring guide](../extensions/authoring-guide.md)
- [Host API v2](../extensions/host-api-v2.md)
- [Runtime themes](../extensions/runtime-themes.md)
- [Generated catalogs](../extensions/catalogs/)
- [V3 platform docs](../extensions/v3/)

## Docs vs knowledge

| Location | Purpose |
| --- | --- |
| `docs/zh-CN` / `docs/en-US` | Human handbooks for ops and development |
| `docs/extensions` | Extension contracts and generated catalogs |
| `knowledge/` | Decisions, module status, session handoffs |
