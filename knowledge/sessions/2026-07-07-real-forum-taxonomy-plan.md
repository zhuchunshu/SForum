# 2026-07-07 Real Forum Taxonomy Implementation Handoff

## Changed

- Implemented the real forum taxonomy plan through Task 8 in
  `docs/superpowers/plans/2026-07-07-real-forum-categories-tags.md`.
- Added schema, permissions, runtime options, backend domain/settings
  resolution, Postgres store behavior, public/admin controllers, OpenAPI
  contracts, Nuxt public taxonomy pages, admin forum management pages, i18n,
  and knowledge-base updates.
- Public Nuxt pages now read real category groups, tags, and topics for the
  homepage, `/c/:categorySlug`, and `/tags/:tagSlug`.
- Admin UI now exposes `/forum/categories`, `/forum/tags`, and
  `/forum/settings` through the low-code admin module registry.
- Topic creation now passes normalized tag slugs and creation mode into the
  store so Postgres resolves/creates tags inside the same transaction as the
  topic and post insert.

## Decisions

- SForum must be treated as an open-source forum framework, not a single
  hard-coded forum deployment.
- Core forum defaults must be safe and ready to use, but configurable and
  resettable from admin UI.
- Category access v1 remains public/hidden only.
- Tags default to controlled mode. Operators can switch to review or open mode
  through admin settings.
- Public tag pages can be disabled through `forum.tags.public_pages`.
- Plugins extend taxonomy behavior only through explicit events/settings/future
  provider slots, not by overriding core routes or raw table semantics.

## Verification

- `go test -count=1 ./app/Models/Options ./database/migrations ./app/Models/Identity`
- `go test -count=1 ./app/Models/Forum ./app/Support/Extensions`
- `go test -count=1 ./app/Http/Controllers/Forum ./app/Models/Forum`
- `ruby scripts/validate-openapi-refs.rb`
- `bun test apps/web/tests/adminForum.test.ts apps/web/tests/forumTaxonomy.test.ts`
- `cd apps/web && bun run typecheck`

## Next

- Run Task 9 final verification: OpenAPI validation, full API tests, frontend
  typecheck, and `./scripts/test.sh`.
- Keep unrelated pre-existing worktree changes intact.

## Open Questions

- When to add tag merge history, taxonomy moderation queues, sitemap/search
  indexing, and role-scoped category permissions.
