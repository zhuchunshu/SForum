# Roadmap

## Milestone 0: Project Setup

- Initialize Git.
- Create documentation skeleton.
- Create knowledge-base workflow.

## Milestone 1: Foundation

- Confirm proposed architecture and MVP scope.
- Scaffold `apps/web` and `apps/api`.
- Add local development services for PostgreSQL, Redis, and Meilisearch.
- Add `compose.yaml`, `compose.dev.yaml`, and `scripts/dev.sh` so one command
  starts all local services with hot reload.
- Add backend config, logging, health checks, migrations, and database
  connectivity.
- Add frontend Nuxt shell, routing conventions, and SEO metadata conventions.
- Add Nuxt i18n configuration with `zh-CN` as default and `en-US` as the first
  secondary locale.
- Add `compose.prod.yaml`, `.env.production.example`, and an initial bilingual
  interactive `deploy.sh`.
- Define the first OpenAPI contract skeleton.
- Create initial schema migrations for users, roles, permissions, categories,
  topics, posts, and post revisions.
- Add identity foundation: open registration, first-user `super_admin`
  bootstrapping, Redis-backed sessions, default `member` assignment, and
  initial RBAC policy helpers.

## Milestone 2: Core Forum

- User profiles and account settings.
- Categories and topics.
- Posts and replies.
- Basic moderation.
- Localized UI, validation display, plugin-backed notifications/mail where
  present, and localized seed/admin labels for shipped features.
- SEO-friendly category and topic pages.

## Milestone 3: Search And Operations

- Meilisearch indexing and rebuild workflow.
- Search UI.
- Rate limits and abuse controls.
- Plugin-backed notification and mail providers if product scope requires them.
- Payment and monetization integrations only after a core payment framework plus
  plugin/provider-slot design is accepted.
- Production backup, restore, deploy status, logs, and rollback workflows in
  `deploy.sh`.
