# Knowledge Index

This is the entry point for project memory.

## Current Project State

- Repository initialized on 2026-07-03.
- Basic documentation and knowledge-base skeleton created.
- First application scaffold has been added under `apps/web` and `apps/api`.
- Forum architecture stack has been proposed and foundation scaffolding has
  started.
- Proposed stack: Nuxt 4/Vue 3/Nuxt UI/Bun frontend; Go Fiber v3,
  PostgreSQL, Redis, and Meilisearch backend.
- Development/deployment workflow has been proposed: Docker Compose for local
  and production orchestration, `scripts/dev.sh` for one-command hot-reload
  development, and bilingual `deploy.sh` for production operations.
- Docker Compose development and production now publish only the `web` service
  to `127.0.0.1:${WEB_PORT}`. API, PostgreSQL, Redis, Meilisearch, and support
  services stay on the Compose network, with `/api/v1/*` proxied through Nuxt.
- Product internationalization is required from the first implementation.
  Default locale is Simplified Chinese (`zh-CN`); first secondary locale is
  English (`en-US`).

## Navigation

- `decisions/` - decision records for architecture, product, and process choices.
- `modules/` - notes for each feature area or system module.
- `sessions/` - short handoffs from previous work sessions.
- `glossary.md` - shared terms and domain language.
- `research.md` - library and ecosystem research notes.
- `../docs/architecture.md` - proposed technical architecture and directory
  layout.
- `../docs/development-and-deployment.md` - proposed local development,
  hot-reload, Docker Compose, and production deployment workflow.
- `../apps/web` - Nuxt web scaffold with default `zh-CN` localization.
- `../apps/api` - Go Fiber API and worker scaffold.

## How To Use This In A New Session

1. Read `AGENTS.md`.
2. Read this file.
3. Open the latest handoff in `sessions/`.
4. Open related module notes.
5. Continue work and update these notes before stopping.

## Open Questions

- What is the first usable MVP scope?
- Which forum features are required versus later enhancements?
- What deployment target should the architecture optimize for?
- Should Meilisearch ship in the first executable milestone or immediately
  after core forum reads/writes?
- What production backup destination and retention policy should be used?
- Should English translations be mandatory for MVP launch or allowed to lag
  during internal development?
