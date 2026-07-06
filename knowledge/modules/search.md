# Search Module

## Purpose

Owns full-text forum search and synchronization between PostgreSQL and
Meilisearch.

## Current Status

Planned. No Meilisearch indexing code has been added.

Forum taxonomy fields are now available from the core forum read models:
category ID/slug/name, category group context, and active topic tag summaries.
These fields should be part of future public search documents, but full index
write/rebuild behavior remains follow-up work.

## Planned Approach

- PostgreSQL is authoritative.
- Meilisearch stores rebuildable public search documents.
- Topic and post writes should create durable indexing events.
- A worker processes indexing events and can rebuild an index from PostgreSQL.
- Private, deleted, draft, and moderation-only content must not be indexed.
- Index documents should include locale/content-language fields once content
  language is captured.

## Candidate Documents

- `topics`: topic title, slug, category ID/slug/name, category group context,
  active tags, author summary, visibility, latest activity, reply count.
- `posts`: post body excerpt, topic/category references, author summary,
  created/updated timestamps, visibility.

## Open Questions

- Whether MVP search covers topics only or topics plus posts.
- How private categories and role-scoped search should behave.
- Whether content-language filtering is part of MVP.
- Exact ranking settings, typo tolerance, synonyms, and stop words.

## Next Steps

- Decide whether search ships in Milestone 1 or Milestone 2.
- Define Meilisearch index settings before writing indexing code.
