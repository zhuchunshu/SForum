# 2026-07-03 Architecture Planning

## Changed

- Replaced the architecture placeholder with a proposed forum architecture.
- Added a stack decision record for Nuxt/Fiber/PostgreSQL/Redis/Meilisearch.
- Added library research notes for frontend rendering, backend framework,
  PostgreSQL access, migrations, sessions, search, and user content rendering.
- Added planned module notes for frontend, backend, forum, and search.
- Updated roadmap and project index to reflect the proposed architecture.
- Added a development/deployment workflow covering Docker Compose,
  one-command local hot reload, and bilingual interactive production deploys.
- Added multilingual product requirements with Simplified Chinese as the
  default locale.

## Decisions

- Proposed a two-app monorepo: `apps/web` for Nuxt and `apps/api` for Fiber.
- Proposed same-origin routing with Nuxt at `/` and Fiber at `/api/v1/*`.
- Proposed `pgx + sqlc + goose` for PostgreSQL access and migrations.
- Proposed Redis-backed browser sessions instead of JWT-first browser auth.
- Proposed Meilisearch as derived search state rebuilt from PostgreSQL.
- Proposed Docker Compose for both local and production orchestration.
- Proposed `scripts/dev.sh` for local startup and `deploy.sh` for bilingual
  production operations.
- Proposed Nuxt i18n for frontend localization, `zh-CN` as default, and
  `en-US` as the first secondary locale.

## Next

- Confirm MVP scope and whether search ships in the first executable milestone.
- Confirm deployment target and reverse proxy strategy.
- Confirm backup destination and retention policy.
- Confirm registration policy, email provider, and upload/storage plan.
- Confirm whether English translations are mandatory for MVP launch.
- After confirmation, scaffold `apps/web`, `apps/api`, local services, and
  initial migrations.

## Open Questions

- Should signup be open, invite-only, or admin-created at first?
- Are tags, votes/reactions, and notifications part of MVP?
- Should topics use linear posts only, or support nested/threaded replies?
- Should production target a single VPS/container host first?
- Which locale should be added after Simplified Chinese and English, if any?
