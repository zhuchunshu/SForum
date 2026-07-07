# Forum Module

## Purpose

Owns the core discussion model: categories, user-facing topics/posts, tree
comments, shared content records, revisions, topic states, slugs, and public
read models.

## Current Status

Backend foundation implemented on 2026-07-06. Real taxonomy slice implemented
on 2026-07-07.

- `categories` owns public forum sections. The first seed category is
  `general` / `综合讨论`.
- `topics` owns user-facing posts/threads: title, slug, category, author,
  state, counters, and a `content_id`.
- `comments` owns tree-shaped replies under topics: parent/root references,
  stable path keys, depth, reply counters, and state.
- `posts` is the shared content table for both topics and comments. It stores
  raw content, sanitized HTML, plain text, excerpt, source format, editor type,
  editor version, render version, and content hash.
- `post_revisions` stores previous shared-content snapshots when comments are
  edited.
- Taxonomy now uses two levels: `category_groups` contain ordered
  `categories`. v1 category access is only `public` or `hidden`; role-scoped
  category access is deferred.
- Core tags live in `tags` with `active`, `pending`, and `disabled` statuses.
  Topic/tag joins live in `topic_tags`, and topic summaries/details expose
  active tag summaries.
- Runtime forum settings live in `web_options`: default category slug, tag
  creation mode, public tag pages, and max tags per topic. Recommended defaults
  are configurable and resettable.
- Tag creation modes are `controlled`, `review`, and `open`. Controlled mode
  only allows approved tags, review mode creates pending tags, and open mode
  creates active tags directly.
- Public Nuxt pages now consume real forum data for the homepage, category
  pages, and tag pages. Tag pages can be disabled through public runtime
  options.
- Admin UI includes category group/category management, tag management, and
  forum settings under the low-code admin module registry.
- Go domain logic lives under `apps/api/app/Models/Forum`; HTTP routes live
  under `apps/api/app/Http/Controllers/Forum`.

## Domain Shape

- Category: groups topics and defines visibility defaults.
- Category group: groups categories for navigation and operator management.
- Topic: user-facing post/thread with title, slug, author, category, state,
  counters, and latest activity.
- Tag: reusable topic label controlled by core settings and admin review.
- Comment: arbitrary-depth tree reply under a topic.
- Post: shared content record used by topics and comments. It is not the
  frontend "帖子" concept.
- Post revision: audit history for edited shared content.
- Topic state: active, locked, hidden, or deleted.

## SEO URL Shape

- Category: `/c/:categorySlug`
- Tag: `/tags/:tagSlug`
- Topic: `/t/:topicID/:topicSlug`

The topic ID gives stable lookup. The slug is for readability and should
redirect to the canonical slug if changed.

## Content Rules

- v1 stores accepted content in the shared `posts` table as raw content,
  sanitized HTML, extracted plain text, and excerpt.
- Markdown and HTML source formats are accepted by the backend in v1.
- `json` is reserved in the schema for future structured editors, but the API
  rejects JSON publishing until a Tiptap/native-JSON acceptance contract exists.
- Render Markdown with `goldmark`; sanitize display HTML with `bluemonday`.
- Client-generated HTML remains untrusted. The API owns final rendering and
  sanitization before storage.
- Keep edit history through `post_revisions` for comment edits. Topic editing
  endpoints are deferred.
- Hide deleted or moderation-only content from public SSR pages, sitemap, and
  Meilisearch indexes.
- Category labels, moderation labels, and system-authored forum text must be
  localizable, defaulting to Simplified Chinese.
- User-authored topics and comments are stored as written and are not
  translated by default.

## API Surface

- `GET /api/v1/categories`
- `GET /api/v1/category-groups`
- `GET /api/v1/tags`
- `GET /api/v1/topics`
- `POST /api/v1/topics`
- `GET /api/v1/topics/{topicID}`
- `GET /api/v1/topics/{topicID}/comments?view=tree|flat`
- `POST /api/v1/topics/{topicID}/comments`
- `GET /api/v1/comments/{commentID}/replies`
- `PATCH /api/v1/comments/{commentID}`
- `DELETE /api/v1/comments/{commentID}`
- `GET /api/v1/admin/forum/category-groups`
- `POST /api/v1/admin/forum/category-groups`
- `PATCH /api/v1/admin/forum/category-groups/{groupID}`
- `GET /api/v1/admin/forum/categories`
- `POST /api/v1/admin/forum/categories`
- `PATCH /api/v1/admin/forum/categories/{categoryID}`
- `GET /api/v1/admin/forum/tags`
- `POST /api/v1/admin/forum/tags`
- `PATCH /api/v1/admin/forum/tags/{tagID}`
- `GET /api/v1/admin/forum/settings`
- `PUT /api/v1/admin/forum/settings`
- `POST /api/v1/admin/forum/settings/reset`

## Permission Boundaries

- Create topic: login required plus `topic.create`.
- Create comment: login required plus existing `post.create`.
- Manage category groups and categories: `category.manage`.
- Manage tags and tag policy settings: `tag.manage`.
- Manage forum settings: `category.manage` or `tag.manage`, with writes
  limited to the permission that owns the changed values.
- Edit comment: author with `post.edit_own`, or any user with
  `post.edit_any`.
- Delete comment: author with `post.delete_own`, or any user with
  `post.delete_any`.
- Future topic lock/pin/hide/delete endpoints should reuse the existing
  `topic.lock`, `topic.pin`, `topic.edit_any`, and `topic.delete_any`
  permissions.

## Plugin Boundary

- Core owns category groups, categories, tag semantics, topic-tag joins, public
  routes, admin routes, runtime settings, and permission checks.
- Plugins may react through explicit forum events and future provider slots, but
  must not override core forum routes, mutate core taxonomy tables directly, or
  bypass API policy checks.
- Full search indexing, notification fanout, external analytics, and provider
  integrations remain plugin/provider work unless core needs a stable contract.

## Comment Display Decision

The backend stores full tree comments. The intended public UI should render
desktop comments with the A-style reading-flow/connection-line layout and
mobile comments with the D-style flat list plus "replying to" context labels.

## Open Questions

- Edit grace period and revision visibility rules.
- Whether votes/reactions exist in MVP.
- When to add topic editing, deletion, locking, hiding, and pinning endpoints.
- How to reconcile the accepted future Tiptap/native-JSON decision with the v1
  shared `posts` table when the rich editor becomes a backend write path.
- When to add tag merge history and taxonomy moderation workflow.
- When to add role-scoped category permissions.

## Next Steps

- Implement topic detail/comment Nuxt consumers for the backend topic and
  comment APIs.
- Add topic moderation/admin endpoints when the moderation UI starts.
- Wire topic/comment writes into the future Meilisearch indexer.

## Topic Lifecycle (Core Forum V1)

Topic lifecycle is implemented in `apps/api/app/Models/Forum` and exposed via
the public API:

- `PATCH /api/v1/topics/{topicID}` updates title/category/tags/content. Title
  changes regenerate the slug. Content changes preserve the triple-storage
  rule: prior content is copied to `post_revisions` before the `posts` row is
  overwritten.
- `DELETE /api/v1/topics/{topicID}` soft-deletes (status `deleted`, sets
  `deleted_at`, decrements category `topic_count`).
- `POST /api/v1/topics/{topicID}/{hide|restore|lock|unlock|pin|unpin}` apply
  status/pin transitions. `restore` clears `deleted_at`/`locked_at`.

Permission model reuses existing keys: own edit needs `post.edit_own`,
any edit needs `topic.edit_any`, own delete needs `post.delete_own`,
any delete/hide/restore needs `topic.delete_any`, lock/unlock needs
`topic.lock`, pin/unpin needs `topic.pin`.

Public reads are limited to `active` and `locked` topics; hidden/deleted
topics return 404 on public detail and are excluded from public lists. Locked
topics remain readable but reject new comments with `forum.topic_closed`.

Events: `topic.updated`, `topic.deleted`, `topic.hidden`, `topic.restored`,
`topic.locked`, `topic.unlocked`, `topic.pinned`, `topic.unpinned` are emitted
as observe events (see `app/Support/Events/catalog.go`).

`GetTopicForAction` loads a topic summary without public visibility filtering
for permission checks. `ScanTopicSummary`/`RowScanner` are exported so the
Profile model reuses the same SELECT column layout for recent-topic lists.

Frontend: `/t/:topicID/:topicSlug` renders topic detail with canonical slug
redirect (301 SSR / replace client), sanitized HTML, comment tree/flat views,
reply editor, and permission-aware action buttons. `/topics/new` and
`/t/:topicID/:topicSlug/edit` provide composer and edit flows using
`SFEditor` (submits markdown with `sourceFormat=markdown`,
`editorType=tiptap`).
