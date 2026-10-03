# 2026-08-22 Built-in Shortcodes M0 Handoff

## Changed

- Completed M0 only. No product runtime, OpenAPI, database, extension package,
  or frontend behavior changed.
- Accepted the structured shortcode/public-source secrecy ADR:
  `../decisions/2026-08-22-structured-shortcodes-public-source-secrecy.md`.
- Recorded the threat model, complete content producer/consumer inventory,
  library survey, current leak, and exact M1 entry points:
  `../reports/2026-08-22-builtin-shortcodes-m0-threat-inventory.md`.
- Updated the task book, forum/extensions modules, plan index, and knowledge
  index. M1 remains not started and awaits explicit user confirmation.

## Decisions

- Host nodes are `sforumShortcodeRef` (block atom) and
  `sforumShortcodeBlock` (block container), with exact `id`,
  `contractVersion`, and typed `arguments` attrs.
- Canonical V1 syntax is `[friend-links]`; `[friend_links]` is literal.
  `topic-tag` is the sole legacy alias and imports as `category`.
- Admitted standalone shortcodes in `sourceFormat=markdown` API input convert
  the complete accepted document to canonical `editor-document` storage.
- Reference shortcodes are standalone blocks. Limits are 32 nodes, nesting
  four, 16 scalar args, 64-byte keys, and 512-code-point strings.
- Public source and raw-source hashes are removed without a compatibility leak.
  Dedicated topic/comment edit-source reads use mutation-equivalent policy;
  admin/revision source authority stays unchanged.
- Stored public plain/search/excerpt/SEO placeholders use the site default
  locale. A locale change requires re-render and reindex.
- Failed references may link to a Core URL only after a current Host public-
  visibility recheck. Protected fallback always fails closed.
- Protected descendants do not fan out mentions or outbound projections, but
  Host moderation, sanitizer, attachment, and author trust checks still inspect
  them.

## Next

Wait for explicit user confirmation, then execute M1 only.

Primary M1 backend/contract entries:

- `apps/api/app/Models/Forum/types.go`
- `apps/api/app/Models/Forum/service.go`
- `apps/api/app/Models/Forum/service_ops.go`
- `apps/api/app/Models/Forum/postgres_store_ops.go`
- `apps/api/app/Http/Controllers/Forum/routes.go`
- `apps/api/app/Http/Controllers/Forum/controller.go`
- `contracts/openapi/schemas/forum.yaml`
- `contracts/openapi/paths/forum.yaml`

Primary M1 frontend entries:

- `apps/web/app/utils/forum/forumTaxonomy.ts`
- `apps/web/app/composables/forum/useForumApi.ts`
- `apps/web/app/components/forum/SFTopicEditPage.vue`
- `apps/web/app/composables/forum/useTopicCommentComposerDrawer.ts`

M1 must prove public topic ID/slug, comment tree/flat/replies, create/update,
Page Registry, and Nuxt SSR/hydration omit source and secret markers while
author/edit-any edit-source reads work and denied/hidden/deleted paths remain
closed. Do not start M2.

## Verification

- `node tests/validate-docs.mjs` passed.
- `node tests/validate-architecture-boundaries.mjs` passed (1,612 production
  files scanned; existing 172 files above the 500-line review threshold).
- `git diff --check`, relative M0 knowledge-link checks, frozen-name checks, and
  M1-not-started consistency checks passed.
- No Go/Web/OpenAPI runtime suite was required because M0 changed knowledge
  files only.

## Open Questions

- None for M0. Later milestones must not reinterpret the frozen ADR without a
  replacement decision and user confirmation.
