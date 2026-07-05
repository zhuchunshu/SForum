# 2026-07-06 Forum Backend Foundation

## Changed

- Added forum backend schema for `categories`, `topics`, `comments`, shared
  content `posts`, and `post_revisions`.
- Added `app/Models/Forum` with Markdown/HTML rendering, HTML sanitization,
  topic creation, tree comment creation, comment editing/deletion policy
  checks, and public comment tree/flat read models.
- Added forum HTTP routes under `/api/v1/categories`, `/api/v1/topics`, and
  `/api/v1/comments`.
- Registered the forum provider in API bootstrap.
- Added modular OpenAPI forum paths and schemas.

## Decisions

- Database `posts` is the shared content table; frontend-visible posts are
  `topics`.
- Comments are stored as full trees with parent/root/path/depth fields.
- Intended UI direction is desktop reading-flow comments plus mobile flat
  reply-context comments.
- Markdown and HTML are accepted content source formats; JSON remains reserved
  but rejected for publishing.

## Next

- Implement Nuxt pages/composables that consume real topic/comment APIs.
- Add category administration and topic moderation endpoints.
- Wire forum writes into Meilisearch indexing once the search module becomes
  executable.

## Open Questions

- Exact topic edit/delete/lock/pin admin surface.
- Whether tags and votes/reactions are in the first public forum milestone.
