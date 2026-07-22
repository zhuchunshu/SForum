# Forum Module

## Purpose

Owns category groups/categories, tags, topics, tree comments, shared content,
accepted revisions, lifecycle states, public read models, and forum policy.

## Current Status

- Core taxonomy, topic/comment creation and lifecycle, public/admin UI, runtime
  settings, moderation integration, search projection, and million-scale read
  path M0-M7 are implemented.
- Content revisions V1 is **active**: M4 restore, historical attachment safety,
  and super-admin redaction are complete; M5 admin content management is next.
- PostgreSQL site search is the protected default. Meilisearch is optional and
  must not be described as the required/default forum read path.

Active revision sources:

- Plan: `../plans/2026-07-22-forum-content-revisions-v1.md`
- M0 contract matrix:
  `../plans/2026-07-22-forum-content-revisions-v1-m0-contract-tests.md`
- Decision: `../decisions/2026-07-22-forum-content-revisions-ledger.md`
- Handoff: `../sessions/2026-07-22-forum-content-revisions-v1-m4-handoff.md`

## Domain Model

- Category groups contain ordered categories. Current category visibility is
  `public` or `hidden`; role/category ACL is deferred.
- Topics own title, slug, category, author, state, counters, activity, tags,
  and a shared `content_id`.
- Comments form arbitrary-depth trees through parent/root/path metadata and own
  a shared `content_id`.
- `posts` stores raw content, sanitized HTML, plain text, format/editor/render
  versions, hash, and current accepted revision.
- `post_revisions` is an immutable accepted-version ledger. Derived HTML,
  excerpt, and plain text are recomputed rather than duplicated as historical
  authority.
- `topic_revision_snapshots` freezes topic metadata needed to reconstruct an
  accepted revision without turning the hot topic row into history storage.
- Topic states are `active`, `locked`, `hidden`, and `deleted`; tags are
  `active`, `pending`, or `disabled`.

## Content Revisions V1

Migration `202607220052` added `posts.current_revision`, numbered nullable
ledger metadata, concurrent `(post_id, revision_no)` indexes, and topic
revision snapshots. New topic/comment creation writes accepted revision 1 in
the same transaction.

Mixed legacy reads expose effective `currentRevision >= 1`; edited state is
effective revision `> 1`, so the initial accepted row does not mark all content
as edited. Backfill command:

```text
sforum revisions backfill --batch=N [--loop]
```

It claims `current_revision=0` batches with `FOR UPDATE SKIP LOCKED`, preserves
legacy row order, inserts the current snapshot, and is idempotent.

M2 added `topic.revision.view_any` and `post.revision.view_any`, permission
migration `202607220053`, moderator template/frontend labels, and read-only
history/admin content surfaces. Topic/comment revision list/detail are
service-gated by the matching history permission: lists return summary headers
only, while detail returns raw source only after authorization and renders a
safe historical preview on demand. Admin topic/comment content list/detail
require `admin.access` plus matching edit-any or history-view permission and do
not add admin mutation routes.

M3 makes PATCH versioned: `expectedRevision` is required in the public API,
the service rejects already-stale requests before synchronous filters, and the
PostgreSQL write transaction locks the resource/post and repeats CAS before any
write. Effective edits append exactly one accepted final snapshot and increment
`posts.current_revision`; semantic no-ops leave timestamps, ledger, audit,
cache, and search untouched. Cross-author edits require a trimmed reason of at
most 500 runes and append generic audit in the same transaction. The public
editors submit their loaded token. `comment.before_update` / `comment.updated`
now complement enriched safe `topic.updated` revision metadata.

V1 boundaries:

- M2 complete: view permissions, revision list/detail read models, admin
  content list/detail, tests, and OpenAPI.
- M3 complete: mandatory `expectedRevision`, two-stage CAS, final accepted edit
  snapshots, no-op detection, reason/audit rules, and comment update hooks/events.
- M4 complete: canonical topic/comment restore append new `restore` revisions,
  re-run filters/rendering/moderation/cache/search, validate current taxonomy and
  historical attachment ownership/availability, and redaction tombstones with
  transaction-bound audit. Legacy incomplete snapshots restore content only.
- M5 next owns the admin management UI. Diff UI, public history, and force
  overwrite remain out of scope.
- Use npm package `diff` 9.0.0 (BSD-3-Clause) for the diff UI; do not install
  npm `jsdiff`.
- Collaboration, CRDT, drafts, notifications, retention controls, and public
  revision browsing remain outside V1.
- Revision query/mutation, raw history content, reason text, IPs, and attachment
  provider data remain closed to plugins in V1.

## Content And Rendering Rules

- Backend accepts Markdown and HTML source formats. JSON publishing remains
  rejected until a native structured-editor contract is accepted.
- Markdown renders with `goldmark` plus GFM extensions; display HTML is
  sanitized with `bluemonday`.
- `RenderVersion` is `goldmark-bluemonday-v2`; existing rows keep earlier HTML
  until a later edit or explicit migration.
- Content/excerpt limits are Unicode-rune based. Excerpts derive at read time
  from `plain_text` using `forum.reading.excerpt_rune_limit`.
- Topic/comment writes may provide `content.attachmentIds`. Explicit arrays
  replace references transactionally, omission preserves them on edit, and an
  empty array clears them.
- Hidden/deleted/moderation-only content stays out of public SSR, sitemap, and
  search indexes.

## Public URLs And Reads

| Resource | URL |
| --- | --- |
| Categories | `/categories`, `/c/:categorySlug` |
| Tags | `/tags`, `/tags/:tagSlug` |
| Topics | `/t/<path>` |

`seo.topic_url_mode` controls topic paths:

| Mode | Shape | Lookup |
| --- | --- | --- |
| `id_slug` | `/t/123/hello-world` | topic ID |
| `id` | `/t/123` | topic ID |
| `slug` | `/t/hello-world` | globally unique slug |

The catch-all topic route recognizes old shapes and redirects to the canonical
path. Only a 404 advances to the next lookup candidate; network/API failures
are not swallowed.

Public reads expose only active/locked topics. Locked topics remain readable
but reject new comments. Viewer-aware deleted-comment tombstones never expose
body fields or deleted-parent reply excerpts.

## Pagination, Cache, And Scale

The completed task book is archived at
`../plans/archive/2026-07/2026-07-21-million-scale-read-path.md`; durable results
live in `../reports/` and
`../decisions/2026-07-21-read-replica-and-api-horizontal-scale.md`.

- Public topic and flat-comment lists accept shallow page pagination and opaque
  keyset `after`; cursor wins over page. Responses return `hasMore` and
  `nextCursor` where applicable.
- Topic keysets include pin state as the first ordering dimension.
- Tree comments page root comments and cap descendants per root using
  `forum.comments.tree_descendants_per_root` (default 50). Truncated roots set
  `hasMoreChildren` and load more through the replies endpoint.
- `CachedStore` caches taxonomy, topic detail, topic lists, and eligible public
  comment lists through Redis-backed generation keys.
- Topic-list invalidation uses global/category/tag scoped generations; writes
  bump only affected scopes plus global.
- Topic detail caches support ID and slug lookup with reverse-map invalidation.
  There is deliberately no composite topic-page cache.
- Anonymous `/t/**` may use Nuxt SWR; session-bearing or fail-closed responses
  must be `no-store`.
- Horizontal default remains one PostgreSQL primary plus shared Redis. Read
  replicas are deferred to measured thresholds; no `DATABASE_READ_URL` runtime
  path exists yet.

## Policy And Runtime Settings

Forum options cover default category, tag creation/public pages, tag limits,
pagination, content limits, cooldown/daily caps, edit windows, comment depth,
tree descendant cap, excerpt length, guest read mode, list sort/hot window,
author close/delete rules, edit marks, duplicate-title policy, soft-delete
visibility, and mention limits.

- Recommended defaults are configurable and resettable in the multi-tab forum
  settings UI.
- `forum.guest.read=login_required` makes public taxonomy/topic/search/comment
  endpoints return 401 to anonymous users.
- New-user trust options may tighten cooldowns/daily caps and outbound-link
  policy during the configured trust window.
- Tag modes are `controlled`, `review`, and `open`.
- Public topic/comment page sizes default to 20, accept 1-100, and remain
  server-authoritative when callers omit `perPage`.

## Authorization

- Topic create: login plus `topic.create`.
- Comment create: login plus `post.create`.
- Own/any topic edit: `topic.edit_own` / `topic.edit_any`.
- Own/any topic delete: `topic.delete_own` / `topic.delete_any`.
- Topic lock/unlock: `topic.lock`; pin/unpin: `topic.pin`.
- Own/any comment edit: `post.edit_own` / `post.edit_any`.
- Own/any comment delete: `post.delete_own` / `post.delete_any`.
- Category administration: `category.manage`; tag administration:
  `tag.manage`.
- Forum settings writes are restricted to the permission owning the changed
  setting family.
- API policy checks are authoritative. Frontend visibility is only UX.

## Extension Boundary

- Core owns canonical forum tables, policy, revision authority, core handlers,
  and default read models.
- Plugins use declared hooks/events, contributions, providers, queries,
  navigation/regions, and other versioned registries.
- Trusted route/guard/query replacements are possible only through V3 exact-
  artifact grants and own the declared authorization contract. Ordinary
  integrations must not request raw core database access or whole-route power.
- Search indexing, notification fanout, analytics, and provider integrations
  project from stable Core contracts.
- Regenerate the Forum Extension Surface Matrix when revision events/catalogs
  change.

## Important Paths

| Path | Responsibility |
| --- | --- |
| `apps/api/app/Models/Forum` | Domain services, stores, policy, cache decorator |
| `apps/api/app/Http/Controllers/Forum` | Public/admin HTTP handlers |
| `apps/api/database/queries/forum.sql` | sqlc forum queries |
| `contracts/openapi/paths/forum.yaml` | Forum HTTP contract |
| `contracts/openapi/schemas/forum.yaml` | Forum schemas |
| `apps/web/app/components/forum` | Reusable forum UI |
| `apps/web/app/pages/t/[...path].vue` | Topic route host |

## Next Steps

1. Continue content revisions V1 from M4 without changing the accepted ledger,
   CAS, authorization, or redaction boundaries.
2. Keep OpenAPI, allowed/denied policy tests, module status, and Extension
   Surface Matrix synchronized at each milestone.
3. Treat role-scoped category ACL, reactions/bookmarks, and structured-editor
   storage as separate future tracks rather than revision scope.
