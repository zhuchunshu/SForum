# Knowledge Index

This is the entry point for project memory.

## Current Project State

- Repository initialized on 2026-07-03.
- Basic documentation and knowledge-base skeleton created.
- No application code has been added.
- Forum architecture stack has been proposed, but not yet scaffolded.
- Proposed stack: Nuxt 4/Vue 3/Nuxt UI/Bun frontend; Go Fiber v3,
  PostgreSQL, Redis, and Meilisearch backend.

## Navigation

- `decisions/` - decision records for architecture, product, and process choices.
- `modules/` - notes for each feature area or system module.
- `sessions/` - short handoffs from previous work sessions.
- `glossary.md` - shared terms and domain language.
- `research.md` - library and ecosystem research notes.
- `../docs/architecture.md` - proposed technical architecture and directory
  layout.

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
