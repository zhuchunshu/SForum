# Forum Module

## Purpose

Owns the core discussion model: categories, topics, posts, post revisions,
topic states, slugs, and public read models.

## Current Status

Planned. No application code has been added.

## Initial Domain Shape

- Category: groups topics and defines visibility/moderation defaults.
- Topic: discussion thread with title, slug, author, category, state, and
  latest activity.
- Post: message inside a topic.
- Post revision: audit history for edited posts.
- Topic state: open, locked, pinned, archived, hidden, or deleted.

## SEO URL Shape

- Category: `/c/:categorySlug`
- Topic: `/t/:topicID/:topicSlug`

The topic ID gives stable lookup. The slug is for readability and should
redirect to the canonical slug if changed.

## Content Rules

- Store source Markdown.
- Render and sanitize HTML for display.
- Keep edit history for posts after the grace period selected by product rules.
- Hide deleted or moderation-only content from public SSR pages, sitemap, and
  Meilisearch indexes.
- Category labels, moderation labels, and system-authored forum text must be
  localizable, defaulting to Simplified Chinese.
- User-authored posts and topics are stored as written and are not translated by
  default.

## Open Questions

- Whether tags are in MVP or deferred.
- Whether nested replies are allowed or posts remain linear inside a topic.
- Edit grace period and revision visibility rules.
- Whether votes/reactions exist in MVP.

## Next Steps

- Confirm MVP forum scope.
- Draft initial PostgreSQL schema once scope is agreed.
- Define read models needed by Nuxt category and topic pages.
