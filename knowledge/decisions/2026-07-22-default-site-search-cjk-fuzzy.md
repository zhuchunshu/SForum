# 2026-07-22 Default Site Search: CJK N-gram And Trigram Fuzzy Matching

## Status

Accepted. Extends `2026-07-21-search-framework-site-default.md`; the protected
site-search provider and optional external-provider boundary do not change.

## Context

PostgreSQL's built-in `simple` text-search configuration is dependable for
Latin words and requires no deployment-specific tokenizer, but a continuous
Chinese sentence becomes one lexeme. It therefore cannot recall a short Chinese
word inside a title or body. Exact CJK bigrams improve substring recall but do
not cover one-character queries, existing body text, or misspellings.

The alternatives considered were:

- `zhparser` / `pg_jieba`: good dictionary segmentation, but require custom
  PostgreSQL images, dictionary lifecycle work, and a new native dependency.
- PGroonga: broad multilingual capability, but materially expands database
  packaging and operational complexity.
- Meilisearch: mature typo tolerance and language tooling, already supported as
  an optional plugin, but intentionally not required by the default stack.
- PostgreSQL `pg_trgm` plus Host-generated CJK n-grams: available in the
  official PostgreSQL image, indexable with GIN, and keeps the default provider
  self-contained.

## Decision

1. Keep `simple` FTS and its weighted generated `tsv` column for ordinary
   lexical search.
2. Generate Unicode Han unigrams and adjacent bigrams in Go for title, excerpt,
   and body. Store them in weighted `cjk_tsv` (`A` / `B` / `C`) and index it
   with GIN. Unigrams support one-character searches; bigrams preserve useful
   adjacency and reduce false positives for longer queries.
3. Enable PostgreSQL's trusted `pg_trgm` contrib extension. Add a generated
   `fuzzy_text` column containing title plus excerpt and a
   `GIN (fuzzy_text gin_trgm_ops)` index. Use indexed literal-contains and
   `word_similarity` operators for prefixes, infixes, and small misspellings.
   The migrator's administrator connection installs the database-level
   extension into Host-owned `sforum_host_extensions` before switching to the
   constrained Core owner. Search SQL schema-qualifies the trigram function,
   operator, and opclass; Core never owns the extension or its schema.
4. Keep trigram matching off the full body to bound index size. Chinese body
   recall uses the more compact `cjk_tsv`; optional external providers may
   offer richer body typo tolerance.
5. Keep PostgreSQL's default word-similarity threshold (`0.6`) to avoid broad
   short-query false positives. Do not mutate database/session-global
   thresholds from public requests.
6. Sort by relevance first, then pinned state, activity time, and topic ID.
   The final ID tie-breaker makes page ordering deterministic.
7. Existing installations must run the normal search reindex once after the
   migration so old documents receive body CJK n-grams. The generated trigram
   column is backfilled by PostgreSQL during migration, so title/excerpt fuzzy
   search is available immediately.

## Consequences

- The default Docker stack supports Chinese terms, one-character Han queries,
  Chinese body search after reindex, English lexical search, literal infix
  matching, and conservative typo tolerance without a separate service.
- `search.provider`, public ACL checks, reindex jobs, admin selection/reset, and
  permission `search.manage` remain unchanged.
- `pg_trgm` becomes a schema dependency for Core migrations. The official
  `postgres:17-alpine` image used by SForum includes it; custom PostgreSQL
  deployments must provide the standard contrib extension.
- Meilisearch remains the recommended optional provider when operators need
  dictionary-aware segmentation, synonyms, or more aggressive typo tolerance.
