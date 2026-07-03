# 2026-07-03 Architecture Planning

## Changed

- Replaced the architecture placeholder with a proposed forum architecture.
- Added a stack decision record for Nuxt/Fiber/PostgreSQL/Redis/Meilisearch.
- Added library research notes for frontend rendering, backend framework,
  PostgreSQL access, migrations, sessions, search, and user content rendering.
- Added planned module notes for frontend, backend, forum, and search.
- Updated roadmap and project index to reflect the proposed architecture.

## Decisions

- Proposed a two-app monorepo: `apps/web` for Nuxt and `apps/api` for Fiber.
- Proposed same-origin routing with Nuxt at `/` and Fiber at `/api/v1/*`.
- Proposed `pgx + sqlc + goose` for PostgreSQL access and migrations.
- Proposed Redis-backed browser sessions instead of JWT-first browser auth.
- Proposed Meilisearch as derived search state rebuilt from PostgreSQL.

## Next

- Confirm MVP scope and whether search ships in the first executable milestone.
- Confirm deployment target and reverse proxy strategy.
- Confirm registration policy, email provider, and upload/storage plan.
- After confirmation, scaffold `apps/web`, `apps/api`, local services, and
  initial migrations.

## Open Questions

- Should signup be open, invite-only, or admin-created at first?
- Are tags, votes/reactions, and notifications part of MVP?
- Should topics use linear posts only, or support nested/threaded replies?
