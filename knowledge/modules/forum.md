# Forum Module

## Purpose

Owns the core discussion model: categories, user-facing topics/posts, tree
comments, shared content records, revisions, topic states, slugs, and public
read models.

## Current Status

Backend foundation implemented on 2026-07-06.

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
- Go domain logic lives under `apps/api/app/Models/Forum`; HTTP routes live
  under `apps/api/app/Http/Controllers/Forum`.

## Domain Shape

- Category: groups topics and defines visibility defaults.
- Topic: user-facing post/thread with title, slug, author, category, state,
  counters, and latest activity.
- Comment: arbitrary-depth tree reply under a topic.
- Post: shared content record used by topics and comments. It is not the
  frontend "帖子" concept.
- Post revision: audit history for edited shared content.
- Topic state: active, locked, hidden, or deleted.

## SEO URL Shape

- Category: `/c/:categorySlug`
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
- `GET /api/v1/topics`
- `POST /api/v1/topics`
- `GET /api/v1/topics/{topicID}`
- `GET /api/v1/topics/{topicID}/comments?view=tree|flat`
- `POST /api/v1/topics/{topicID}/comments`
- `GET /api/v1/comments/{commentID}/replies`
- `PATCH /api/v1/comments/{commentID}`
- `DELETE /api/v1/comments/{commentID}`

## Permission Boundaries

- Create topic: login required plus `topic.create`.
- Create comment: login required plus existing `post.create`.
- Edit comment: author with `post.edit_own`, or any user with
  `post.edit_any`.
- Delete comment: author with `post.delete_own`, or any user with
  `post.delete_any`.
- Future topic lock/pin/hide/delete endpoints should reuse the existing
  `topic.lock`, `topic.pin`, `topic.edit_any`, and `topic.delete_any`
  permissions.

## Comment Display Decision

The backend stores full tree comments. The intended public UI should render
desktop comments with the A-style reading-flow/connection-line layout and
mobile comments with the D-style flat list plus "replying to" context labels.

## Open Questions

- Whether tags are in MVP or deferred.
- Edit grace period and revision visibility rules.
- Whether votes/reactions exist in MVP.
- When to add topic editing, deletion, locking, hiding, and pinning endpoints.
- Whether category management ships as a backend-only API first or with an
  admin UI.
- How to reconcile the accepted future Tiptap/native-JSON decision with the v1
  shared `posts` table when the rich editor becomes a backend write path.

## Next Steps

- Implement Nuxt consumers for the backend topic and comment APIs.
- Add topic moderation/admin endpoints when the moderation UI starts.
- Wire topic/comment writes into the future Meilisearch indexer.
