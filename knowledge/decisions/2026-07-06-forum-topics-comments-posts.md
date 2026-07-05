# Decision: Forum Topics, Tree Comments, And Shared Posts Content

## Status

Accepted

## Context

SForum needs its first real forum backend model. Product language should call
user-facing threads "帖子 / topics" and replies "comments", while content
storage must support multiple future editors without duplicating body fields
across business tables.

An earlier editor decision accepted Tiptap and triple content storage as the
future rich editor direction. The backend v1 needs a simpler executable path
now, while leaving room for structured editor input later.

## Decision

Use three core tables for the v1 forum backend:

- `topics` stores the user-facing post/thread shell.
- `comments` stores tree-shaped replies under topics.
- `posts` stores shared content for both topics and comments, including raw
  content, sanitized HTML, plain text, excerpt, source format, editor metadata,
  render version, and content hash.

Use `post_revisions` for shared-content edit snapshots.

Use `goldmark` plus `bluemonday` for the first renderer/sanitizer path.
Markdown and HTML are accepted in v1; JSON content is reserved in the schema
but rejected by the API until a structured editor contract is designed.

## Consequences

- The frontend "帖子" concept maps to `topics`, not database `posts`.
- Tree comments can power nested desktop discussions, mobile flat reply views,
  moderation path lookup, and future notification context without another
  migration.
- The shared content table gives future Markdown, rich-text, and structured
  editors one rendering boundary.
- Topic editing and moderation endpoints remain separate follow-up work.

## Follow-up

- Implement Nuxt topic/comment consumers.
- Add category management and topic moderation endpoints.
- Enqueue search indexing jobs from topic/comment writes once the indexer is
  implemented.
- Revisit `posts` content columns when Tiptap/native JSON becomes an accepted
  backend write format.
